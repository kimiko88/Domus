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
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	infrasup "github.com/domus-hub/domus/internal/infrastructure/supervisor"
	wsinfra "github.com/domus-hub/domus/internal/infrastructure/websocket"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/domus-hub/domus/internal/interfaces/http/swagger"
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

// WithMQTTStatus sets the MQTT broker connectivity state for metrics.
func WithMQTTStatus(connected bool) ServerOption {
	return func(s *Server) {
		s.mqttConnected = connected
	}
}

// WithCORS configures the allowed CORS origins (for dev frontends on different ports).
func WithCORS(origins []string) ServerOption {
	return func(s *Server) {
		s.allowedOrigins = origins
	}
}

// Server encapsulates the HTTP router and net/http server lifecycle.
type Server struct {
	cfg           config.ServerConfig
	router        chi.Router
	httpServer    *http.Server
	listener      net.Listener
	startedAt     time.Time
	authService   *auth.Service
	rateLimiter   *auth.RateLimiter
	repos         *sqlite.Repositories
	bus           events.Bus
	wsHub         *wsinfra.Hub
	autoEngine    *appauto.Engine
	sceneService  *appscene.Service
	mqttConnected bool
	allowedOrigins []string
}

// NewServer builds and configures the HTTP server and router.
func NewServer(cfg config.ServerConfig, version string, startedAt time.Time, pinger handlers.Pinger, opts ...ServerOption) *Server {
	srv := &Server{
		cfg:       cfg,
		startedAt: startedAt,
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

	// Security headers (CSP, X-Frame-Options, etc.)
	r.Use(domusmw.SecurityHeaders())

	// CORS (optional, for development frontends)
	if len(srv.allowedOrigins) > 0 {
		r.Use(domusmw.CORS(srv.allowedOrigins))
	}

	// If auth service is present, extract sessions globally
	if srv.authService != nil {
		r.Use(domusmw.Authenticate(srv.authService))
	}

	healthHandler := handlers.NewHealthHandler(version, startedAt, pinger)
	swaggerHandler := swagger.NewHandler()

	// Public health routes
	r.Method(http.MethodGet, "/health", healthHandler)

	// Embedded OpenAPI & Swagger UI documentation routes
	r.Get("/docs", swaggerHandler.ServeUI)
	r.Get("/swagger", swaggerHandler.ServeUI)
	r.Get("/api/openapi.yaml", swaggerHandler.ServeSpec)

	// WebSocket real-time route
	if srv.wsHub != nil {
		wsH := wstransport.NewHandler(srv.wsHub, srv.authService)
		r.Method(http.MethodGet, "/ws", wsH)
	}

	// API v1 namespace
	r.Route("/api/v1", func(api chi.Router) {
		api.Route("/system", func(sys chi.Router) {
			sys.Method(http.MethodGet, "/health", healthHandler)
			if srv.repos != nil {
				metricsH := handlers.NewSystemMetricsHandler(srv.startedAt, srv.repos, srv.wsHub, srv.mqttConnected)
				sys.Get("/metrics", metricsH.ServeHTTP)
			}

			// Domus OS Appliance Diagnostics & Maintenance
			supClient := infrasup.NewClient("/run/domus/supervisor.sock", "")
			osH := handlers.NewSystemOSHandler(supClient)
			sys.Get("/diagnostics", osH.GetDiagnostics)
			sys.Get("/diagnostics/export", osH.ExportDiagnostics)
			sys.Get("/maintenance/status", osH.GetMaintenanceStatus)
			sys.Post("/maintenance/enable", osH.EnableMaintenance)
			sys.Post("/maintenance/disable", osH.DisableMaintenance)
		})

		// Auth routes
		if srv.authService != nil {
			authH := handlers.NewAuthHandler(srv.authService, cfg.CookieSecure)
			familyH := handlers.NewFamilyHandler(srv.authService, cfg.CookieSecure)

			api.Route("/auth", func(authRouter chi.Router) {
				authRouter.Post("/setup", authH.Setup)
				authRouter.Get("/setup", authH.SetupStatus)
				authRouter.Get("/setup-status", authH.SetupStatus)

				// Invite lookup & registration
				authRouter.Get("/invite/{token}", familyH.ValidateInvite)
				authRouter.Post("/register", familyH.RegisterWithInvite)

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

			// Family management routes (authenticated)
			api.Route("/family", func(famRouter chi.Router) {
				famRouter.Use(domusmw.RequireAuth())
				// All authenticated members can view other family members
				famRouter.Get("/members", familyH.ListMembers)

				// Role modifications and member removal restricted to Admin
				famRouter.With(domusmw.RequireRole(user.RoleAdmin)).Put("/members/{id}/role", familyH.UpdateMemberRole)
				famRouter.With(domusmw.RequireRole(user.RoleAdmin)).Delete("/members/{id}", familyH.RemoveMember)

				// Invites management restricted to Admin
				famRouter.With(domusmw.RequireRole(user.RoleAdmin)).Get("/invites", familyH.ListInvites)
				famRouter.With(domusmw.RequireRole(user.RoleAdmin)).Post("/invites", familyH.CreateInvite)
				famRouter.With(domusmw.RequireRole(user.RoleAdmin)).Delete("/invites/{id}", familyH.RevokeInvite)
			})
		}

		// Domain repository routes (Areas, Devices, Entities)
		if srv.repos != nil {
			areaH := handlers.NewAreaHandler(srv.repos.Areas)
			deviceH := handlers.NewDeviceHandler(srv.repos.Devices)
			entityH := handlers.NewEntityHandler(srv.repos.Entities, srv.bus)

			api.Group(func(domainRouter chi.Router) {
				if srv.authService != nil {
					domainRouter.Use(domusmw.RequireAuth())
				}

				domainRouter.Route("/areas", func(areaRouter chi.Router) {
					areaRouter.Get("/", areaH.List)
					if srv.authService != nil {
						areaRouter.With(domusmw.RequireRole(user.RoleAdmin)).Post("/", areaH.Create)
					} else {
						areaRouter.Post("/", areaH.Create)
					}
					areaRouter.Get("/{id}", areaH.GetByID)
				})

				domainRouter.Route("/devices", func(devRouter chi.Router) {
					devRouter.Get("/", deviceH.List)
					devRouter.Get("/{id}", deviceH.GetByID)
				})

				domainRouter.Route("/entities", func(entRouter chi.Router) {
					entRouter.Get("/", entityH.List)
					entRouter.Get("/{id}", entityH.GetByID)
					if srv.authService != nil {
						entRouter.With(domusmw.RequireRole(user.RoleOperator, user.RoleAdmin)).Post("/{id}/command", entityH.Command)
					} else {
						entRouter.Post("/{id}/command", entityH.Command)
					}
					if srv.repos.History != nil {
						histH := handlers.NewHistoryHandler(srv.repos.History, srv.repos.Entities)
						entRouter.Get("/{id}/history", histH.GetEntityHistory)
					}
				})

				autoH := handlers.NewAutomationHandler(srv.repos.Automations, srv.autoEngine)
				domainRouter.Route("/automations", func(autoRouter chi.Router) {
					autoRouter.Get("/", autoH.List)
					if srv.authService != nil {
						autoRouter.With(domusmw.RequireRole(user.RoleAdmin)).Post("/", autoH.Create)
						autoRouter.Get("/{id}", autoH.Get)
						autoRouter.With(domusmw.RequireRole(user.RoleAdmin)).Put("/{id}", autoH.Update)
						autoRouter.With(domusmw.RequireRole(user.RoleAdmin)).Delete("/{id}", autoH.Delete)
						autoRouter.With(domusmw.RequireRole(user.RoleOperator, user.RoleAdmin)).Post("/{id}/trigger", autoH.Trigger)
					} else {
						autoRouter.Post("/", autoH.Create)
						autoRouter.Get("/{id}", autoH.Get)
						autoRouter.Put("/{id}", autoH.Update)
						autoRouter.Delete("/{id}", autoH.Delete)
						autoRouter.Post("/{id}/trigger", autoH.Trigger)
					}
					autoRouter.Get("/{id}/runs", autoH.ListRuns)
				})

				sceneH := handlers.NewSceneHandler(srv.repos.Scenes, srv.sceneService)
				domainRouter.Route("/scenes", func(sceneRouter chi.Router) {
					sceneRouter.Get("/", sceneH.List)
					if srv.authService != nil {
						sceneRouter.With(domusmw.RequireRole(user.RoleAdmin)).Post("/", sceneH.Create)
						sceneRouter.Get("/{id}", sceneH.Get)
						sceneRouter.With(domusmw.RequireRole(user.RoleAdmin)).Put("/{id}", sceneH.Update)
						sceneRouter.With(domusmw.RequireRole(user.RoleAdmin)).Delete("/{id}", sceneH.Delete)
						sceneRouter.With(domusmw.RequireRole(user.RoleOperator, user.RoleAdmin)).Post("/{id}/apply", sceneH.Apply)
					} else {
						sceneRouter.Post("/", sceneH.Create)
						sceneRouter.Get("/{id}", sceneH.Get)
						sceneRouter.Put("/{id}", sceneH.Update)
						sceneRouter.Delete("/{id}", sceneH.Delete)
						sceneRouter.Post("/{id}/apply", sceneH.Apply)
					}
				})

				// Audit log endpoint (admin-only via RBAC)
				auditH := handlers.NewAuditHandler(srv.repos.Audit)
				if srv.authService != nil {
					domainRouter.With(domusmw.RequireRole(user.RoleAdmin)).Get("/audit", auditH.ServeHTTP)
				} else {
					domainRouter.Get("/audit", auditH.ServeHTTP)
				}
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
