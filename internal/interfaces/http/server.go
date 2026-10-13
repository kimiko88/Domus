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
	appchore "github.com/domus-hub/domus/internal/application/chore"
	appdash "github.com/domus-hub/domus/internal/application/dashboard"
	"github.com/domus-hub/domus/internal/application/energy"
	apphelper "github.com/domus-hub/domus/internal/application/helper"
	"github.com/domus-hub/domus/internal/application/presence"
	appscene "github.com/domus-hub/domus/internal/application/scene"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	appalarm "github.com/domus-hub/domus/internal/application/alarm"
	appassistant "github.com/domus-hub/domus/internal/application/assistant"
	appbrand "github.com/domus-hub/domus/internal/application/brand"
	appcal "github.com/domus-hub/domus/internal/application/calendar"
	appcamera "github.com/domus-hub/domus/internal/application/camera"
	appmedia "github.com/domus-hub/domus/internal/application/media"
	appmesh "github.com/domus-hub/domus/internal/application/mesh"
	appnotif "github.com/domus-hub/domus/internal/application/notification"
	appota "github.com/domus-hub/domus/internal/application/ota"
	appshop "github.com/domus-hub/domus/internal/application/shopping"
	appstore "github.com/domus-hub/domus/internal/application/store"
	apptag "github.com/domus-hub/domus/internal/application/tag"
	appweather "github.com/domus-hub/domus/internal/application/weather"
	appwebhook "github.com/domus-hub/domus/internal/application/webhook"
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

// WithAdvancedFeatures configures energy load shedding, multi-tier presence, and solar calculations.
func WithAdvancedFeatures(energyCtrl *energy.Controller, presenceSvc *presence.Service, sunEngine *appauto.SunEngine) ServerOption {
	return func(s *Server) {
		s.energyCtrl = energyCtrl
		s.presenceSvc = presenceSvc
		s.sunEngine = sunEngine
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
	cfg            config.ServerConfig
	router         chi.Router
	httpServer     *http.Server
	listener       net.Listener
	startedAt      time.Time
	authService    *auth.Service
	rateLimiter    *auth.RateLimiter
	repos          *sqlite.Repositories
	bus            events.Bus
	wsHub          *wsinfra.Hub
	autoEngine     *appauto.Engine
	sceneService   *appscene.Service
	mqttConnected  bool
	allowedOrigins []string
	energyCtrl     *energy.Controller
	presenceSvc    *presence.Service
	sunEngine      *appauto.SunEngine
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

			// Solar position & astronomical calculation
			advH := handlers.NewAdvancedFeaturesHandler(srv.energyCtrl, srv.presenceSvc, srv.sunEngine)
			sys.Get("/sun", advH.GetSunPosition)
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
				authRouter.Post("/kiosk-login", authH.KioskLogin)

				// Protected auth routes
				authRouter.With(domusmw.RequireAuth()).Post("/logout", authH.Logout)
				authRouter.With(domusmw.RequireAuth()).Get("/me", authH.Me)
			})

			// Family management routes (authenticated)
			api.Route("/family", func(famRouter chi.Router) {
				famRouter.Use(domusmw.RequireAuth())
				// All authenticated members can view other family members
				famRouter.Get("/members", familyH.ListMembers)

				// Guest profiles management
				famRouter.Get("/guests", familyH.ListGuests)
				famRouter.With(domusmw.RequireRole(user.RoleAdmin)).Post("/guests", familyH.CreateGuest)

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
			brandSvc := appbrand.NewService(srv.repos.Devices, srv.repos.Entities, srv.repos.Integrations, srv.bus, nil)
			integrationH := handlers.NewIntegrationHandler(srv.repos.Integrations, srv.bus).WithBrandService(brandSvc)

			var webhookH *handlers.WebhookHandler
			if srv.repos.DB != nil {
				webhookSvc := appwebhook.NewService(srv.repos.DB, srv.bus)
				webhookH = handlers.NewWebhookHandler(webhookSvc)
				api.HandleFunc("/webhooks/{webhook_id}", webhookH.HandleInbound)
			}

			api.Group(func(domainRouter chi.Router) {
				if srv.authService != nil {
					domainRouter.Use(domusmw.RequireAuth())
				}

				domainRouter.Route("/integrations", func(intRouter chi.Router) {
					intRouter.Get("/", integrationH.List)
					intRouter.Get("/catalog", integrationH.GetCatalog)
					intRouter.Get("/discovery/devices", integrationH.ListDiscovered)
					intRouter.Post("/discovery/scan", integrationH.Scan)
					intRouter.Post("/discovery/pair", integrationH.Pair)
					intRouter.Get("/{id}", integrationH.GetByID)
					intRouter.Post("/{id}/reconnect", integrationH.Reconnect)
				})

				domainRouter.Route("/areas", func(areaRouter chi.Router) {
					areaRouter.Get("/", areaH.List)
					if srv.authService != nil {
						areaRouter.With(domusmw.RequireRole(user.RoleAdmin)).Post("/", areaH.Create)
						areaRouter.With(domusmw.RequireRole(user.RoleAdmin)).Put("/{id}", areaH.Update)
						areaRouter.With(domusmw.RequireRole(user.RoleAdmin)).Delete("/{id}", areaH.Delete)
					} else {
						areaRouter.Post("/", areaH.Create)
						areaRouter.Put("/{id}", areaH.Update)
						areaRouter.Delete("/{id}", areaH.Delete)
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

				autoH := handlers.NewAutomationHandler(srv.repos.Automations, srv.autoEngine).WithEntities(srv.repos.Entities)
				domainRouter.Route("/automations", func(autoRouter chi.Router) {
					autoRouter.Get("/", autoH.List)
					autoRouter.Get("/blueprints", autoH.ListBlueprints)
					if srv.authService != nil {
						autoRouter.With(domusmw.RequireRole(user.RoleAdmin)).Post("/blueprints/{id}/instantiate", autoH.InstantiateBlueprint)
						autoRouter.With(domusmw.RequireRole(user.RoleAdmin)).Post("/", autoH.Create)
						autoRouter.Get("/{id}", autoH.Get)
						autoRouter.With(domusmw.RequireRole(user.RoleAdmin)).Put("/{id}", autoH.Update)
						autoRouter.With(domusmw.RequireRole(user.RoleAdmin)).Delete("/{id}", autoH.Delete)
						autoRouter.With(domusmw.RequireRole(user.RoleOperator, user.RoleAdmin)).Post("/{id}/trigger", autoH.Trigger)
						autoRouter.Post("/{id}/dry-run", autoH.DryRun)
					} else {
						autoRouter.Post("/blueprints/{id}/instantiate", autoH.InstantiateBlueprint)
						autoRouter.Post("/", autoH.Create)
						autoRouter.Get("/{id}", autoH.Get)
						autoRouter.Put("/{id}", autoH.Update)
						autoRouter.Delete("/{id}", autoH.Delete)
						autoRouter.Post("/{id}/trigger", autoH.Trigger)
						autoRouter.Post("/{id}/dry-run", autoH.DryRun)
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

				// Floors and multi-level floor plan markers
				if srv.repos.Floors != nil {
					floorH := handlers.NewFloorplanHandler(srv.repos.Floors)
					domainRouter.Route("/floors", func(flRouter chi.Router) {
						flRouter.Get("/", floorH.List)
						flRouter.Post("/", floorH.Create)
						flRouter.Get("/{id}", floorH.GetByID)
						flRouter.Put("/{id}", floorH.Update)
						flRouter.Delete("/{id}", floorH.Delete)
						flRouter.Get("/{id}/markers", floorH.ListMarkers)
						flRouter.Put("/{id}/markers", floorH.SaveMarker)
						flRouter.Delete("/{id}/markers/{markerId}", floorH.DeleteMarker)
					})
				}

				// Chores & Reminders
				if srv.repos.Chores != nil {
					choreService := appchore.NewService(srv.repos.Chores, srv.bus)
					choreH := handlers.NewChoreHandler(srv.repos.Chores, choreService)
					domainRouter.Route("/chores", func(chRouter chi.Router) {
						chRouter.Get("/", choreH.List)
						chRouter.Post("/", choreH.Create)
						chRouter.Get("/{id}", choreH.GetByID)
						chRouter.Put("/{id}", choreH.Update)
						chRouter.Delete("/{id}", choreH.Delete)
						chRouter.Post("/{id}/complete", choreH.Complete)
					})
				}

				// Advanced Features: Energy Load Shedding & Multi-tier Presence
				advH := handlers.NewAdvancedFeaturesHandler(srv.energyCtrl, srv.presenceSvc, srv.sunEngine)
				domainRouter.Route("/energy", func(eRouter chi.Router) {
					eRouter.Get("/load-shedding", advH.GetLoadSheddingStatus)
					eRouter.Get("/flow", advH.GetEnergyFlow)
					eRouter.Get("/tariffs", advH.GetTariffs)
					eRouter.Get("/dynamic-tariffs", advH.GetDynamicTariffs)
					eRouter.Put("/flexible-loads", advH.UpdateFlexibleLoad)
					eRouter.Post("/evaluate-tariffs", advH.EvaluateTariffs)
					if srv.authService != nil {
						eRouter.With(domusmw.RequireRole(user.RoleAdmin)).Put("/load-shedding", advH.UpdateLoadShedding)
					} else {
						eRouter.Put("/load-shedding", advH.UpdateLoadShedding)
					}
				})

				domainRouter.Route("/presence", func(pRouter chi.Router) {
					pRouter.Post("/gps", advH.UpdateGPS)
					pRouter.Get("/status", advH.GetPresenceStatus)
				})

				// Hardware Modern Standards & Commissioning
				domainRouter.Route("/hardware", func(hwRouter chi.Router) {
					hwRouter.Get("/homekit", advH.GetHomeKitPairing)
					hwRouter.Post("/matter/commission", advH.CommissionMatter)
				})

				// Custom Widget Dashboards & Kiosk Mode
				if srv.repos.Dashboards != nil {
					dashService := appdash.NewService(srv.repos.Dashboards)
					dashH := handlers.NewDashboardHandler(dashService)
					domainRouter.Route("/dashboards", func(dRouter chi.Router) {
						dRouter.Get("/", dashH.List)
						dRouter.Post("/", dashH.Create)
						dRouter.Get("/{id}", dashH.Get)
						dRouter.Put("/{id}", dashH.Update)
						dRouter.Delete("/{id}", dashH.Delete)
						dRouter.Post("/{id}/default", dashH.SetDefault)
					})
				}

				// User Virtual Helpers
				if srv.repos.Helpers != nil {
					helperService := apphelper.NewService(srv.repos.Helpers, srv.bus)
					helperH := handlers.NewHelperHandler(helperService)
					domainRouter.Route("/helpers", func(hRouter chi.Router) {
						hRouter.Get("/", helperH.List)
						hRouter.Post("/", helperH.Create)
						hRouter.Get("/{id}", helperH.Get)
						hRouter.Delete("/{id}", helperH.Delete)
						hRouter.Post("/{id}/action", helperH.Action)
					})
				}

				// Ecosystem routes: Cameras, Tags, Notifications, Schedules/Calendar, Media, Weather/Solar, Mesh, OTA, Store
				camSvc := appcamera.NewService(appcamera.NewMemoryRepository(), srv.bus)
				tagSvc := apptag.NewService(apptag.NewMemoryRepository(), srv.bus)
				notifMgr := appnotif.NewManager(srv.bus)
				calSvc := appcal.NewService(appcal.NewMemoryRepository(), srv.bus)
				mediaSvc := appmedia.NewService(appmedia.NewMemoryRepository(), srv.bus)
				weatherCli := appweather.NewClient()
				meshSvc := appmesh.NewService()
				otaSvc := appota.NewService(srv.bus)
				storeSvc := appstore.NewService()
				ecoH := handlers.NewEcosystemHandler(camSvc, tagSvc, notifMgr, calSvc, mediaSvc, weatherCli, meshSvc, otaSvc, storeSvc)

				domainRouter.Route("/cameras", func(cRouter chi.Router) {
					cRouter.Get("/", ecoH.ListCameras)
					cRouter.Post("/", ecoH.CreateCamera)
					cRouter.Get("/nvr/status", ecoH.GetNVRStatus)
					cRouter.Post("/{id}/webrtc", ecoH.NegotiateWebRTC)
					cRouter.Get("/{id}/zones", ecoH.GetCameraZones)
					cRouter.Post("/{id}/zones", ecoH.AddCameraZone)
					cRouter.Post("/{id}/detect", ecoH.ProcessAIDetection)
					cRouter.Get("/events", ecoH.ListCameraEvents)
					cRouter.Get("/events/{id}/clip", ecoH.GetEventClip)
				})

				domainRouter.Route("/tags", func(tRouter chi.Router) {
					tRouter.Get("/", ecoH.ListTags)
					tRouter.Post("/", ecoH.CreateTag)
					tRouter.Post("/{id}/scan", ecoH.ScanTag)
				})

				domainRouter.Route("/notifications/actionable", func(nRouter chi.Router) {
					nRouter.Post("/", ecoH.CreateActionableNotification)
					nRouter.Post("/actions/{action_id}", ecoH.HandleNotificationAction)
				})

				domainRouter.Route("/schedules", func(sRouter chi.Router) {
					sRouter.Get("/", ecoH.ListSchedules)
					sRouter.Post("/", ecoH.CreateSchedule)
				})

				domainRouter.Route("/calendar", func(calRouter chi.Router) {
					calRouter.Get("/events", ecoH.ListCalendarEvents)
					calRouter.Post("/import", ecoH.ImportICal)
				})

				domainRouter.Route("/media", func(mRouter chi.Router) {
					mRouter.Get("/players", ecoH.ListMediaPlayers)
					mRouter.Post("/players/{id}/volume", ecoH.SetPlayerVolume)
					mRouter.Post("/broadcast", ecoH.BroadcastAnnouncement)
				})

				domainRouter.Route("/weather", func(wRouter chi.Router) {
					wRouter.Get("/forecast", ecoH.GetWeatherForecast)
					wRouter.Get("/solar-forecast", ecoH.GetSolarForecast)
				})

				domainRouter.Route("/mesh", func(meshRouter chi.Router) {
					meshRouter.Get("/topology", ecoH.GetMeshTopology)
				})

				domainRouter.Route("/ota", func(otaRouter chi.Router) {
					otaRouter.Get("/devices", ecoH.ListOTADevices)
					otaRouter.Post("/devices/{id}/upgrade", ecoH.TriggerOTAUpgrade)
					otaRouter.Get("/flasher/catalog", ecoH.ListFlasherFirmwares)
					otaRouter.Get("/flasher/{id}/manifest", ecoH.GetFlasherManifest)
					otaRouter.Post("/flasher/provision", ecoH.GenerateFlasherProvision)
				})

				domainRouter.Route("/system/failover", func(foRouter chi.Router) {
					foRouter.Get("/status", ecoH.GetFailoverStatus)
					foRouter.Post("/heartbeat", ecoH.ReceiveHeartbeat)
					foRouter.Post("/promote", ecoH.PromoteNode)
				})

				domainRouter.Route("/store", func(stRouter chi.Router) {
					stRouter.Get("/packages", ecoH.ListStorePackages)
					stRouter.Post("/packages/{id}/install", ecoH.InstallStorePackage)
					stRouter.Post("/packages/{id}/uninstall", ecoH.UninstallStorePackage)
				})

				// Shared Shopping List routes
				shopSvc := appshop.NewService(appshop.NewMemoryRepository(), srv.bus)
				shopH := handlers.NewShoppingHandler(shopSvc)
				domainRouter.Route("/shopping", func(shRouter chi.Router) {
					shRouter.Get("/", shopH.List)
					shRouter.Post("/", shopH.Create)
					shRouter.Post("/{id}/toggle", shopH.Toggle)
					shRouter.Delete("/{id}", shopH.Delete)
					shRouter.Post("/clear-completed", shopH.ClearCompleted)
				})

				// Security Alarm Control Panel & Zigbee Direct Binding & Inbound Webhooks
				if srv.repos.DB != nil {
					alarmSvc := appalarm.NewService(srv.repos.DB, srv.bus, nil)
					alarmH := handlers.NewAlarmHandler(alarmSvc)
					alarmH.RegisterRoutes(domainRouter)

					bindingSvc := appmesh.NewBindingService(srv.repos.DB)
					bindingH := handlers.NewBindingHandler(bindingSvc)
					bindingH.RegisterRoutes(domainRouter)

					if webhookH != nil {
						domainRouter.Route("/webhook-configs", func(whRouter chi.Router) {
							whRouter.Get("/", webhookH.ListWebhooks)
							whRouter.Post("/", webhookH.CreateWebhook)
							whRouter.Delete("/{id}", webhookH.DeleteWebhook)
						})
					}
				}

				// Local LLM & Voice Assistant
				assistantSvc := appassistant.NewService(appassistant.Config{Enabled: true}, srv.bus, srv.repos.Entities)
				assistantH := handlers.NewAssistantHandler(assistantSvc)
				assistantH.RegisterRoutes(domainRouter)

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
		Addr:              cfg.Addr(),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
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
