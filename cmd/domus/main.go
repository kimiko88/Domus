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
	"syscall"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
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

	// 3. Initialize SQLite database & apply migrations
	db, err := sqlite.Open(cfg.Database)
	if err != nil {
		slog.Error("failed to open sqlite database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ctxMigrate, cancelMigrate := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelMigrate()

	if err := sqlite.Migrate(ctxMigrate, db); err != nil {
		slog.Error("failed to apply database migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database initialized and migrations up-to-date")

	// 4. Initialize Repositories and Application Services
	repos := sqlite.NewRepositories(db)
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
	)
	loginLimiter := auth.NewRateLimiter(cfg.Auth.RateLimitLoginPerMin, time.Minute)

	// 5. Initialize Event Bus
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 1024,
		Workers:    4,
	})
	defer bus.Close()

	// 6. Initialize HTTP Server
	srv := serverpkg.NewServer(
		cfg.Server,
		Version,
		startedAt,
		db,
		serverpkg.WithAuth(authService, loginLimiter),
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
