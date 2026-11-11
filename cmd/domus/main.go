// Package main is the primary entrypoint for the Domus local-first smart home hub.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	appauto "github.com/domus-hub/domus/internal/application/automation"
	appchore "github.com/domus-hub/domus/internal/application/chore"
	appenergy "github.com/domus-hub/domus/internal/application/energy"
	apphistory "github.com/domus-hub/domus/internal/application/history"
	apppresence "github.com/domus-hub/domus/internal/application/presence"
	"github.com/domus-hub/domus/internal/application/safety"
	appscene "github.com/domus-hub/domus/internal/application/scene"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	wsinfra "github.com/domus-hub/domus/internal/infrastructure/websocket"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
	"github.com/domus-hub/domus/internal/plugins"
	"github.com/google/uuid"
)

var (
	// Version is populated at build time via -ldflags.
	Version = "1.0.0-dev"
	// BuildTime is populated at build time via -ldflags.
	BuildTime = "unknown"
)

func main() {
	configPath := flag.String("config", "domus.toml", "Path to TOML configuration file")
	flag.Parse()

	// 1. Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "domus: failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// 2. Setup structured logging
	initLogger(cfg.Logging)

	slog.Info("starting domus hub",
		"version", Version,
		"build_time", BuildTime,
		"config_path", *configPath,
		"server_addr", cfg.Server.Addr(),
		"db_path", cfg.Database.Path,
	)

	startedAt := time.Now()

	// 3. Initialize SQLite dual database pool (writer & concurrent readers) & apply migrations
	pool, err := sqlite.OpenDualPool(cfg.Database)
	if err != nil {
		slog.Error("failed to open sqlite database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	db := pool.Write

	ctxMigrate, cancelMigrate := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelMigrate()

	if err := sqlite.Migrate(ctxMigrate, db); err != nil {
		slog.Error("failed to apply database migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database initialized and migrations up-to-date")

	// 4. Initialize Repositories and Application Services with Dual Pool
	repos := sqlite.NewRepositoriesWithPool(pool)

	// Warm up in-memory entity cache for sub-50-microsecond reads
	ctxWarmUp, cancelWarmUp := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelWarmUp()
	if err := repos.WarmUp(ctxWarmUp); err != nil {
		slog.Warn("failed to warm up entity cache", "error", err)
	} else {
		slog.Info("entity cache warmed up successfully")
	}

	// Initialize envelope encryption at rest for sensitive credentials
	masterKeyPath := filepath.Join(filepath.Dir(cfg.Database.Path), "master.key")
	if cfg.Database.Path == ":memory:" {
		masterKeyPath = filepath.Join(os.TempDir(), "domus_master.key")
	}
	masterKey, err := crypto.LoadOrCreateMasterKey(masterKeyPath)
	if err != nil {
		// Never continue with an ephemeral key: encrypted values would become
		// unreadable after restart and new writes would be encrypted irreversibly.
		slog.Error("failed to load or persist master key; refusing to start", "path", masterKeyPath, "error", err)
		os.Exit(1)
	}
	envelopeCipher, err := crypto.NewEnvelopeCipher(masterKey)
	if err != nil {
		slog.Error("failed to initialize envelope encryption; refusing to start", "error", err)
		os.Exit(1)
	}
	repos.WithCipher(envelopeCipher)
	slog.Info("envelope encryption at rest initialized", "key_path", masterKeyPath)

	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory:      cfg.Auth.Argon2Memory,
		Iterations:  cfg.Auth.Argon2Time,
		Parallelism: cfg.Auth.Argon2Threads,
		SaltLength:  16,
		KeyLength:   32,
	})
	authService := auth.NewService(
		repos.Users,
		repos.Sessions,
		repos.Audit,
		hasher,
		cfg.Auth.SessionTTL.Duration,
	).WithInvites(repos.Invites)
	loginLimiter := auth.NewRateLimiter(cfg.Auth.RateLimitLoginPerMin, time.Minute)

	// 5. Initialize Event Bus
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 1024,
		Workers:    4,
	})
	defer bus.Close()

	// 5.1 Initialize Plugin Supervision Tree & Circuit Breaker
	pluginSupervisor := plugins.NewSupervisor(bus)
	ctxSupervisor, cancelSupervisor := context.WithCancel(context.Background())
	defer cancelSupervisor()
	if err := pluginSupervisor.Start(ctxSupervisor); err != nil {
		slog.Warn("failed to start plugin supervisor", "error", err)
	}

	// 5.2 Initialize High-Priority Life-Safety Safe-Stop Pipeline (<50ms reaction)
	safetyService := safety.NewService(bus)
	defer safetyService.Stop()

	// 5.3 Initialize SQLite Storage Maintenance & Flash Wear Protection
	maintService := sqlite.NewMaintenanceService(pool, sqlite.MaintenanceConfig{
		CheckpointInterval: 4 * time.Hour,
		DBPath:             cfg.Database.Path,
	})
	ctxMaint, cancelMaint := context.WithCancel(context.Background())
	defer cancelMaint()
	maintService.Start(ctxMaint)
	defer maintService.Stop()

	// 6. Initialize WebSocket Hub
	wsHub := wsinfra.NewHub(bus)
	ctxHub, cancelHub := context.WithCancel(context.Background())
	defer cancelHub()
	go wsHub.Run(ctxHub)

	// 7. Initialize Batched State History Recorder & Retention Service
	historyRecorder := apphistory.NewRecorder(repos.History, bus, apphistory.RecorderConfig{
		BatchSize:     50,
		FlushInterval: 2 * time.Second,
	})
	if err := historyRecorder.Start(context.Background()); err != nil {
		slog.Error("failed to start history recorder", "error", err)
	}
	defer historyRecorder.Stop()

	retentionService := apphistory.NewRetentionService(
		repos.History,
		repos.Audit,
		repos.Automations,
		apphistory.RetentionConfig{
			StateHistoryDays:  cfg.Retention.StateHistoryDays,
			AuditLogDays:      cfg.Retention.AuditLogDays,
			AutomationRunDays: cfg.Retention.AutomationRunDays,
		},
	)
	retentionService.Start(context.Background(), 24*time.Hour)
	defer retentionService.Stop()

	// 8. Initialize Domestic Chores Reminder Scheduler
	choreScheduler := appchore.NewScheduler(repos.Chores, bus, 1*time.Minute)
	if err := choreScheduler.Start(context.Background()); err != nil {
		slog.Warn("failed to start chore scheduler", "error", err)
	}
	defer choreScheduler.Stop()

	// 9. Initialize Rule Automation Engine & Scenes
	autoEngine := appauto.NewEngine(repos.Automations, repos.Entities, bus)
	sceneService := appscene.NewService(repos.Scenes, bus)

	// 10. Initialize Energy Load Shedding & Multi-tier Presence
	energyCtrl := appenergy.NewController(appenergy.LoadSheddingConfig{
		MaxPowerWatts:    3300.0,
		OverloadDuration: 60,
		RecoveryWatts:    2500.0,
		RecoveryDelaySec: 120,
		PriorityLoads:    []string{"switch.ev_charger", "switch.water_heater", "switch.dehumidifier"},
		Enabled:          true,
	}, bus)

	homeGeofence := apppresence.HomeGeofence{
		Latitude:  cfg.Presence.HomeLatitude,
		Longitude: cfg.Presence.HomeLongitude,
		RadiusM:   cfg.Presence.GeofenceRadiusM,
		Enabled:   cfg.Presence.GeofenceEnabled || cfg.Presence.HomeLatitude != 0 || cfg.Presence.HomeLongitude != 0,
	}
	presenceSvc := apppresence.NewService(homeGeofence, bus)

	var sunEngine *appauto.SunEngine
	if homeGeofence.Enabled {
		sunEngine = appauto.NewSunEngine(homeGeofence.Latitude, homeGeofence.Longitude)
	} else {
		slog.Warn("home coordinates are not configured; GPS geofencing and solar position are disabled")
	}

	// 11. Telemetry Downsampler (Nightly compaction of telemetry > 30 days old into hourly buckets)
	downsampler := apphistory.NewDownsampler(db)
	ctxDownsample, cancelDownsample := context.WithCancel(context.Background())
	defer cancelDownsample()
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		time.Sleep(10 * time.Second)
		cutoff := time.Now().AddDate(0, 0, -30)
		if res, err := downsampler.DownsampleOlderThan(ctxDownsample, cutoff); err == nil && res.BucketsCreated > 0 {
			slog.Info("telemetry downsampling complete", "buckets", res.BucketsCreated, "purged", res.RawPointsPurged)
		}
		for {
			select {
			case <-ctxDownsample.Done():
				return
			case t := <-ticker.C:
				cutoff := t.AddDate(0, 0, -30)
				if res, err := downsampler.DownsampleOlderThan(ctxDownsample, cutoff); err == nil && res.BucketsCreated > 0 {
					slog.Info("nightly telemetry downsampling complete", "buckets", res.BucketsCreated, "purged", res.RawPointsPurged)
				}
			}
		}
	}()

	// 12. Initialize HTTP Server
	srv := serverpkg.NewServer(
		cfg.Server,
		Version,
		startedAt,
		db,
		serverpkg.WithAuth(authService, loginLimiter),
		serverpkg.WithRepositories(repos, bus),
		serverpkg.WithAutomation(autoEngine, sceneService),
		serverpkg.WithAdvancedFeatures(energyCtrl, presenceSvc, sunEngine),
		serverpkg.WithWebSocket(wsHub),
	)

	serverErrCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", cfg.Server.Addr())
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
	}()

	// 7. Emit SystemStarted domain event
	ctxEvt, cancelEvt := context.WithTimeout(context.Background(), 2*time.Second)
	_ = bus.Publish(ctxEvt, events.Event{
		ID:         uuid.NewString(),
		Type:       events.SystemStarted,
		OccurredAt: time.Now(),
		Source:     "domus.system",
	})
	cancelEvt()

	// 8. Wait for termination signals or fatal server error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("received shutdown signal", "signal", sig.String())
	case err := <-serverErrCh:
		slog.Error("fatal http server error", "error", err)
	}

	// 9. Graceful shutdown
	slog.Info("initiating graceful shutdown...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("error during http server shutdown", "error", err)
	}

	if err := pluginSupervisor.Stop(shutdownCtx); err != nil {
		slog.Error("error during plugin supervisor shutdown", "error", err)
	}

	slog.Info("domus hub stopped cleanly")
}

func initLogger(cfg config.LoggingConfig) {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}
