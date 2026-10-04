package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/domus-hub/domus/internal/supervisor"
)

var (
	// Version is set during build via -ldflags
	Version   = "1.0.0"
	BuildTime = "unknown"
)

func main() {
	cfg := supervisor.DefaultConfig()
	cfg.Version = Version

	var (
		socketPath    = flag.String("socket", cfg.SupervisorSocket, "Path to supervisor Unix domain socket")
		ctrSocket     = flag.String("containerd", cfg.ContainerdSocket, "Path to containerd Unix domain socket")
		coreImage     = flag.String("core-image", cfg.CoreImage, "OCI container image for Domus Core")
		dataDir       = flag.String("data-dir", cfg.DataDir, "Root persistent data directory (/data)")
		coreHealthURL = flag.String("health-url", cfg.CoreHealthURL, "HTTP health check endpoint for Domus Core")
		showVersion   = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("Domus Supervisor v%s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	cfg.SupervisorSocket = *socketPath
	cfg.ContainerdSocket = *ctrSocket
	cfg.CoreImage = *coreImage
	cfg.DataDir = *dataDir
	cfg.CoreHealthURL = *coreHealthURL

	log.Printf("[Supervisor] Starting Domus Supervisor v%s...", Version)
	log.Printf("[Supervisor] Containerd socket: %s", cfg.ContainerdSocket)
	log.Printf("[Supervisor] API Unix socket:   %s", cfg.SupervisorSocket)
	log.Printf("[Supervisor] Data directory:    %s", cfg.DataDir)

	// Ensure parent socket directory exists
	if err := os.MkdirAll(filepath.Dir(cfg.SupervisorSocket), 0755); err != nil {
		log.Fatalf("[Supervisor] Failed to create socket directory: %v", err)
	}

	// Initialize runtime, rauc controller, and health checker
	runtime := supervisor.NewContainerdRuntime(cfg.ContainerdSocket)
	rauc := supervisor.NewCLIRaucController(cfg.RaucAgentPath)
	healthChecker := supervisor.NewHTTPHealthChecker(cfg.HealthCheckTimeout)

	engine := supervisor.NewEngine(cfg, runtime, rauc, healthChecker)
	apiServer := supervisor.NewAPIServer(engine)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start supervisor engine
	if err := engine.Start(ctx); err != nil {
		log.Printf("[Supervisor] Warning: Engine start returned error: %v", err)
	}

	// Start Unix Domain Socket HTTP API server in background
	go func() {
		log.Printf("[Supervisor] Serving Unix Domain Socket API on %s", cfg.SupervisorSocket)
		if err := apiServer.ServeUnix(ctx, cfg.SupervisorSocket); err != nil {
			log.Printf("[Supervisor] Unix server stopped: %v", err)
		}
	}()

	sig := <-sigChan
	log.Printf("[Supervisor] Received signal %v, initiating shutdown...", sig)
	cancel()

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	_ = engine.Stop(stopCtx)

	log.Printf("[Supervisor] Domus Supervisor stopped.")
}
