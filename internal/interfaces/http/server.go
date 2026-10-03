// Package http provides the HTTP server, routing, and middlewares for Domus.
package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
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

// Server encapsulates the HTTP router and net/http server lifecycle.
type Server struct {
	cfg         config.ServerConfig
	router      chi.Router
	httpServer  *http.Server
	listener    net.Listener
	authService *auth.Service
	rateLimiter *auth.RateLimiter
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

	// API v1 namespace
	r.Route("/api/v1", func(api chi.Router) {
		api.Route("/system", func(sys chi.Router) {
			sys.Method(http.MethodGet, "/health", healthHandler)
		})

		// Auth namespace
		if srv.authService != nil {
			authH := handlers.NewAuthHandler(srv.authService, cfg.CookieSecure)
			api.Route("/auth", func(authRouter chi.Router) {
				authRouter.Post("/setup", authH.Setup)
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
