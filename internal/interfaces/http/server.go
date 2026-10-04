// Package http provides the HTTP server, routing, and middlewares for Domus.
package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	appauto "github.com/domus-hub/domus/internal/application/automation"
	appscene "github.com/domus-hub/domus/internal/application/scene"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	wsinfra "github.com/domus-hub/domus/internal/infrastructure/websocket"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	wstransport "github.com/domus-hub/domus/internal/interfaces/websocket"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// ServerOption configures additional features on the HTTP server.
type ServerOption func(*Server)

// WithAuth attaches authentication service, rate limiter, and auth routes.
func WithAuth(authService *auth.Service, limiter *auth.RateLimiter) ServerOption {
	return func(s *Server) {
		s.authService = authService
		s.rateLimiter = limiter
	}
}

// WithRepositories attaches domain repositories and enables hardware/spatial routes.
func WithRepositories(repos *sqlite.Repositories, bus events.Bus) ServerOption {
	return func(s *Server) {
		s.repos = repos
		s.bus = bus
	}
}

// WithAutomation attaches an automation engine and scene service.
func WithAutomation(engine *appauto.Engine, sceneService *appscene.Service) ServerOption {
	return func(s *Server) {
		s.autoEngine = engine
		s.sceneService = sceneService
	}
}

// WithWebSocket attaches the WebSocket real-time broadcast hub.
func WithWebSocket(hub *wsinfra.Hub) ServerOption {
	return func(s *Server) {
		s.wsHub = hub
	}
}

// Server encapsulates the HTTP router and net/http server lifecycle.
type Server struct {
	cfg          config.ServerConfig
	router       chi.Router
	httpServer   *http.Server
	listener     net.Listener
	authService  *auth.Service
	rateLimiter  *auth.RateLimiter
	repos        *sqlite.Repositories
	bus          events.Bus
	wsHub        *wsinfra.Hub
	autoEngine   *appauto.Engine
	sceneService *appscene.Service
}

// NewServer builds and configures the HTTP server and router.
func NewServer(cfg config.ServerConfig, version string, startedAt time.Time, pinger handlers.Pinger, opts ...ServerOption) *Server {
	srv := &Server{
		cfg: cfg,
	}

	for _, opt := range opts {
		opt(srv)
	}

	r := chi.NewRouter()

	// Global standard middlewares
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(domusmw.RequestLogger())
	r.Use(chimw.Recoverer)

	// If auth service is present, extract sessions globally
	if srv.authService != nil {
		r.Use(domusmw.Authenticate(srv.authService))
	}

	healthHandler := handlers.NewHealthHandler(version, startedAt, pinger)

	// Public health routes
	r.Method(http.MethodGet, "/health", healthHandler)

	// WebSocket real-time route
	if srv.wsHub != nil {
		wsH := wstransport.NewHandler(srv.wsHub, srv.authService)
		r.Method(http.MethodGet, "/ws", wsH)
	}

	// API v1 namespace
	r.Route("/api/v1", func(api chi.Router) {
		api.Route("/system", func(sys chi.Router) {
			sys.Method(http.MethodGet, "/health", healthHandler)
		})

		// Auth routes
		if srv.authService != nil {
			authH := handlers.NewAuthHandler(srv.authService, cfg.CookieSecure)
			api.Route("/auth", func(authRouter chi.Router) {
				authRouter.Post("/setup", authH.Setup)
				authRouter.Get("/setup", authH.SetupStatus)
				authRouter.Get("/setup-status", authH.SetupStatus)

				// Rate-limited login
				if srv.rateLimiter != nil {
					authRouter.With(domusmw.RateLimit(srv.rateLimiter)).Post("/login", authH.Login)
				} else {
					authRouter.Post("/login", authH.Login)
				}

				// Protected auth routes
				authRouter.With(domusmw.RequireAuth()).Post("/logout", authH.Logout)
				authRouter.With(domusmw.RequireAuth()).Get("/me", authH.Me)
			})
		}

		// Domain repository routes (Areas, Devices, Entities)
		if srv.repos != nil {
			areaH := handlers.NewAreaHandler(srv.repos.Areas)
			deviceH := handlers.NewDeviceHandler(srv.repos.Devices)
			entityH := handlers.NewEntityHandler(srv.repos.Entities, srv.bus)

			api.Route("/areas", func(areaRouter chi.Router) {
				areaRouter.Get("/", areaH.List)
				areaRouter.Post("/", areaH.Create)
				areaRouter.Get("/{id}", areaH.GetByID)
			})

			api.Route("/devices", func(devRouter chi.Router) {
				devRouter.Get("/", deviceH.List)
				devRouter.Get("/{id}", deviceH.GetByID)
			})

			api.Route("/entities", func(entRouter chi.Router) {
				entRouter.Get("/", entityH.List)
				entRouter.Get("/{id}", entityH.GetByID)
				entRouter.Post("/{id}/command", entityH.Command)
			})

			autoH := handlers.NewAutomationHandler(srv.repos.Automations, srv.autoEngine)
			api.Route("/automations", func(autoRouter chi.Router) {
				autoRouter.Get("/", autoH.List)
				autoRouter.Post("/", autoH.Create)
				autoRouter.Get("/{id}", autoH.Get)
				autoRouter.Put("/{id}", autoH.Update)
				autoRouter.Delete("/{id}", autoH.Delete)
				autoRouter.Post("/{id}/trigger", autoH.Trigger)
				autoRouter.Get("/{id}/runs", autoH.ListRuns)
			})

			sceneH := handlers.NewSceneHandler(srv.repos.Scenes, srv.sceneService)
			api.Route("/scenes", func(sceneRouter chi.Router) {
				sceneRouter.Get("/", sceneH.List)
				sceneRouter.Post("/", sceneH.Create)
				sceneRouter.Get("/{id}", sceneH.Get)
				sceneRouter.Put("/{id}", sceneH.Update)
				sceneRouter.Delete("/{id}", sceneH.Delete)
				sceneRouter.Post("/{id}/apply", sceneH.Apply)
			})
		}
	})

	srv.router = r
	srv.httpServer = &http.Server{
		Addr:         cfg.Addr(),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return srv
}

// Router returns the underlying Chi router for testing or route inspection.
func (s *Server) Router() chi.Router {
	return s.router
}

// Start begins listening and serving HTTP traffic. It is blocking until closed.
func (s *Server) Start() error {
	addr := s.cfg.Addr()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("http server failed to listen on %s: %w", addr, err)
	}
	s.listener = ln

	if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server error: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the HTTP server without interrupting active connections.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
