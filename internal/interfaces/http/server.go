// Package http provides the HTTP server, routing, and middlewares for Domus.
package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	appalarm "github.com/domus-hub/domus/internal/application/alarm"
	appanomaly "github.com/domus-hub/domus/internal/application/anomaly"
	appassistant "github.com/domus-hub/domus/internal/application/assistant"
	"github.com/domus-hub/domus/internal/application/auth"
	appauto "github.com/domus-hub/domus/internal/application/automation"
	appsafety "github.com/domus-hub/domus/internal/application/safety"
	appbrand "github.com/domus-hub/domus/internal/application/brand"
	appcal "github.com/domus-hub/domus/internal/application/calendar"
	appcamera "github.com/domus-hub/domus/internal/application/camera"
	appchore "github.com/domus-hub/domus/internal/application/chore"
	appclimate "github.com/domus-hub/domus/internal/application/climate"
	appcloudassistant "github.com/domus-hub/domus/internal/application/cloud_assistant"
	appdash "github.com/domus-hub/domus/internal/application/dashboard"
	"github.com/domus-hub/domus/internal/application/energy"
	appfinance "github.com/domus-hub/domus/internal/application/finance"
	apphelper "github.com/domus-hub/domus/internal/application/helper"
	apphist "github.com/domus-hub/domus/internal/application/history"
	applogbook "github.com/domus-hub/domus/internal/application/logbook"
	appmedia "github.com/domus-hub/domus/internal/application/media"
	appmed "github.com/domus-hub/domus/internal/application/medicine"
	appmesh "github.com/domus-hub/domus/internal/application/mesh"
	appnotif "github.com/domus-hub/domus/internal/application/notification"
	appota "github.com/domus-hub/domus/internal/application/ota"
	apppantry "github.com/domus-hub/domus/internal/application/pantry"
	"github.com/domus-hub/domus/internal/application/presence"
	appscene "github.com/domus-hub/domus/internal/application/scene"
	appschedule "github.com/domus-hub/domus/internal/application/schedule"
	appscript "github.com/domus-hub/domus/internal/application/script"
	appshop "github.com/domus-hub/domus/internal/application/shopping"
	appstore "github.com/domus-hub/domus/internal/application/store"
	apptag "github.com/domus-hub/domus/internal/application/tag"
	appte "github.com/domus-hub/domus/internal/application/template_entity"
	appweather "github.com/domus-hub/domus/internal/application/weather"
	appwebhook "github.com/domus-hub/domus/internal/application/webhook"
	appzone "github.com/domus-hub/domus/internal/application/zone"
	"github.com/domus-hub/domus/internal/domain/events"
	domainShop "github.com/domus-hub/domus/internal/domain/shopping"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/network"
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
	r.Use(domusmw.TraceContext())
	r.Use(domusmw.RequestLogger())
	r.Use(chimw.Recoverer)

	// Security headers (CSP, X-Frame-Options, etc.) and payload limits
	r.Use(domusmw.SecurityHeaders())
	r.Use(domusmw.MaxBodySize(4 * 1024 * 1024)) // 4 MB payload limit against inflation DoS

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
		if srv.repos != nil {
			wsH.WithBusAndRepo(srv.bus, srv.repos.Entities)
		} else {
			wsH.WithBusAndRepo(srv.bus, nil)
		}
		r.Method(http.MethodGet, "/ws", wsH)
	}

	// API v1 namespace
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(domusmw.Idempotency(nil))

		if srv.repos != nil && srv.repos.BannedIPs != nil {
			api.Use(domusmw.IPBanMiddleware(srv.repos.BannedIPs))
		}

		if srv.wsHub != nil {
			wsH := wstransport.NewHandler(srv.wsHub, srv.authService)
			if srv.repos != nil {
				wsH.WithBusAndRepo(srv.bus, srv.repos.Entities)
			} else {
				wsH.WithBusAndRepo(srv.bus, nil)
			}
			api.Method(http.MethodGet, "/ws", wsH)
		}

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
			if srv.repos != nil {
				authH.WithSecurity(srv.repos.TwoFactor, srv.repos.BannedIPs)
			}
			familyH := handlers.NewFamilyHandler(srv.authService, cfg.CookieSecure)

			api.Route("/auth", func(authRouter chi.Router) {
				authRouter.Post("/setup", authH.Setup)
				authRouter.Get("/setup", authH.SetupStatus)
				authRouter.Get("/setup-status", authH.SetupStatus)

				// Invite lookup & registration
				authRouter.Get("/invite/{token}", familyH.ValidateInvite)

				// Rate-limited auth endpoints
				if srv.rateLimiter != nil {
					authRouter.With(domusmw.RateLimit(srv.rateLimiter)).Post("/register", familyH.RegisterWithInvite)
					authRouter.With(domusmw.RateLimit(srv.rateLimiter)).Post("/login", authH.Login)
					authRouter.With(domusmw.RateLimit(srv.rateLimiter)).Post("/kiosk-login", authH.KioskLogin)
				} else {
					authRouter.Post("/register", familyH.RegisterWithInvite)
					authRouter.Post("/login", authH.Login)
					authRouter.Post("/kiosk-login", authH.KioskLogin)
				}

				// 2FA TOTP authentication
				authRouter.With(domusmw.RequireAuth()).Post("/2fa/setup", authH.Setup2FA)
				authRouter.With(domusmw.RequireAuth()).Post("/2fa/verify", authH.VerifyAndEnable2FA)
				authRouter.With(domusmw.RequireAuth()).Post("/2fa/disable", authH.Disable2FA)

				// Protected auth routes
				authRouter.With(domusmw.RequireAuth()).Post("/logout", authH.Logout)
				authRouter.With(domusmw.RequireAuth()).Get("/me", authH.Me)
			})

			// Security IP management routes (admin-only)
			api.Route("/security", func(secRouter chi.Router) {
				secRouter.Use(domusmw.RequireAuth(), domusmw.RequireRole(user.RoleAdmin))
				secRouter.Get("/banned-ips", authH.ListBannedIPs)
				secRouter.Post("/banned-ips/unban", authH.UnbanIP)
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
				if srv.repos.OutboundWebhooks != nil {
					outboundDispatcher := appwebhook.NewOutboundDispatcher(srv.repos.OutboundWebhooks, srv.bus, nil)
					outboundDispatcher.Start(context.Background())
					webhookH = webhookH.WithOutbound(outboundDispatcher)
				}
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
						var rollupSvc *apphist.RollupService
						if srv.repos.Statistics != nil {
							rollupSvc = apphist.NewRollupService(srv.repos.History, srv.repos.Statistics, srv.repos.Entities)
							go rollupSvc.Start(context.Background(), 1*time.Hour)
						}
						histH := handlers.NewHistoryHandler(srv.repos.History, srv.repos.Entities)
						if srv.repos.Statistics != nil {
							histH.WithStatistics(srv.repos.Statistics, rollupSvc)
						}
						entRouter.Get("/{id}/history", histH.GetEntityHistory)

						domainRouter.Route("/history", func(hRouter chi.Router) {
							hRouter.Get("/statistics", histH.GetStatistics)
							if srv.authService != nil {
								hRouter.With(domusmw.RequireRole(user.RoleAdmin)).Post("/statistics/rollup", histH.TriggerRollup)
							} else {
								hRouter.Post("/statistics/rollup", histH.TriggerRollup)
							}
						})
					}
				})

				// Mobile & ecosystem services endpoint (/api/v1/services/{domain}/{service})
				if srv.authService != nil {
					domainRouter.With(domusmw.RequireRole(user.RoleOperator, user.RoleAdmin)).Post("/services/{domain}/{service}", entityH.CallService)
				} else {
					domainRouter.Post("/services/{domain}/{service}", entityH.CallService)
				}

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
						chRouter.Get("/calendar.ics", choreH.ExportICal)
						chRouter.Post("/sync/google", choreH.SyncGoogleCalendar)
						chRouter.Get("/{id}", choreH.GetByID)
						chRouter.Put("/{id}", choreH.Update)
						chRouter.Delete("/{id}", choreH.Delete)
						chRouter.Post("/{id}/complete", choreH.Complete)
					})
				}

				// Advanced Features: Energy Load Shedding & Multi-tier Presence
				advH := handlers.NewAdvancedFeaturesHandler(srv.energyCtrl, srv.presenceSvc, srv.sunEngine).
					WithEntityRepository(srv.repos.Entities).
					WithWallboxRepository(srv.repos.Wallbox).
					WithVacationRepository(srv.repos.Vacation).
					WithCommuteRepository(srv.repos.Commute)
				domainRouter.Route("/energy", func(eRouter chi.Router) {
					eRouter.Get("/load-shedding", advH.GetLoadSheddingStatus)
					eRouter.Get("/flow", advH.GetEnergyFlow)
					eRouter.Get("/tariffs", advH.GetTariffs)
					eRouter.Get("/dynamic-tariffs", advH.GetDynamicTariffs)
					eRouter.Put("/flexible-loads", advH.UpdateFlexibleLoad)
					eRouter.Post("/evaluate-tariffs", advH.EvaluateTariffs)
					eRouter.Get("/bess/schedule", advH.GetBESSSchedule)
					eRouter.Post("/bess/optimize", advH.OptimizeBESS)
					eRouter.Put("/bess/config", advH.UpdateBESSConfig)
					eRouter.Get("/wallbox", advH.GetWallboxStatus)
					eRouter.Post("/wallbox/mode", advH.SetWallboxMode)
					eRouter.Post("/wallbox/modulate", advH.CalculateOptimalWallboxCurrent)
					if srv.authService != nil {
						eRouter.With(domusmw.RequireRole(user.RoleAdmin)).Put("/load-shedding", advH.UpdateLoadShedding)
					} else {
						eRouter.Put("/load-shedding", advH.UpdateLoadShedding)
					}
					if srv.repos.UtilityMeters != nil {
						umH := handlers.NewUtilityMeterHandler(srv.repos.UtilityMeters, srv.repos.Entities)
						eRouter.Get("/utility-meters", umH.List)
						eRouter.Post("/utility-meters", umH.Create)
						eRouter.Get("/utility-meters/{id}", umH.Get)
						eRouter.Put("/utility-meters/{id}", umH.Update)
						eRouter.Delete("/utility-meters/{id}", umH.Delete)
						eRouter.Get("/multi-utility-summary", umH.Summary)
					}
				})

				// Spatial Zones and Tracked Persons (Multi-Zone & Bayesian Presence)
				var zoneH *handlers.ZoneHandler
				if srv.repos.Zones != nil {
					zoneSvc := appzone.NewService(srv.repos.Zones, srv.bus)
					zoneH = handlers.NewZoneHandler(zoneSvc, srv.repos.Zones)
					domainRouter.Route("/zones", func(zRouter chi.Router) {
						zRouter.Get("/", zoneH.ListZones)
						zRouter.Post("/", zoneH.CreateZone)
						zRouter.Get("/{id}", zoneH.GetZone)
						zRouter.Put("/{id}", zoneH.UpdateZone)
						zRouter.Delete("/{id}", zoneH.DeleteZone)
					})
					domainRouter.Route("/persons", func(pRouter chi.Router) {
						pRouter.Get("/", zoneH.ListPersons)
						pRouter.Post("/", zoneH.CreatePerson)
						pRouter.Get("/{id}", zoneH.GetPerson)
						pRouter.Put("/{id}", zoneH.UpdatePerson)
						pRouter.Delete("/{id}", zoneH.DeletePerson)
						pRouter.Post("/{id}/location", zoneH.UpdatePersonLocation)
					})
				}

				// Template Entities (Expressions, Formula Evaluator & Virtual Sensors)
				if srv.repos.Templates != nil {
					teSvc := appte.NewService(srv.repos.Templates, srv.repos.Entities, srv.bus)
					_ = teSvc.Start(context.Background())
					teH := handlers.NewTemplateEntityHandler(teSvc, srv.repos.Templates)
					domainRouter.Route("/template-entities", func(teRouter chi.Router) {
						teRouter.Get("/", teH.List)
						teRouter.Post("/", teH.Create)
						teRouter.Post("/preview", teH.Preview)
						teRouter.Get("/{id}", teH.Get)
						teRouter.Put("/{id}", teH.Update)
						teRouter.Delete("/{id}", teH.Delete)
					})
				}

				domainRouter.Route("/presence", func(pRouter chi.Router) {
					pRouter.Post("/gps", advH.UpdateGPS)
					pRouter.Get("/status", advH.GetPresenceStatus)
					pRouter.Get("/vacation", advH.GetVacationMode)
					pRouter.Post("/vacation", advH.SetVacationMode)
					pRouter.Post("/vacation/actions", advH.RegisterVacationAction)
					pRouter.Post("/commute", advH.UpdateCommuteLocation)
					pRouter.Get("/commute/{userId}", advH.GetCommuteStatus)
					if zoneH != nil {
						pRouter.Get("/persons", zoneH.ListPersons)
					}
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
						dRouter.Post("/{id}/share", dashH.Share)
					})
					api.Get("/dashboards/shared/{token}", dashH.GetShared)
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
				var camRepo appcamera.Repository = appcamera.NewMemoryRepository()
				if srv.repos.Cameras != nil {
					camRepo = srv.repos.Cameras
				}
				camSvc := appcamera.NewService(camRepo, srv.bus)

				var tagRepo apptag.Repository = apptag.NewMemoryRepository()
				if srv.repos.Tags != nil {
					tagRepo = srv.repos.Tags
				}
				tagSvc := apptag.NewService(tagRepo, srv.bus)

				notifMgr := appnotif.NewManager(srv.bus)

				var calRepo appcal.Repository = appcal.NewMemoryRepository()
				if srv.repos.Calendar != nil {
					calRepo = srv.repos.Calendar
				}
				calSvc := appcal.NewService(calRepo, srv.bus)

				var mediaRepo appmedia.Repository = appmedia.NewMemoryRepository()
				if srv.repos.Media != nil {
					mediaRepo = srv.repos.Media
					_ = srv.repos.Media.SeedDefaultPlayers(context.Background())
				}
				mediaSvc := appmedia.NewService(mediaRepo, srv.bus)

				weatherCli := appweather.NewClient()
				meshSvc := appmesh.NewService(srv.repos.Devices)

				var otaRepo appota.Repository = appota.NewMemoryRepository()
				if srv.repos.OTA != nil {
					otaRepo = srv.repos.OTA
				}
				otaSvc := appota.NewService(otaRepo, srv.bus)

				var storeRepo appstore.Repository = appstore.NewMemoryRepository()
				if srv.repos.Store != nil {
					_ = srv.repos.Store.SeedDefaultPackages(context.Background())
					storeRepo = srv.repos.Store
				}
				storeSvc := appstore.NewService(storeRepo)

				ecoH := handlers.NewEcosystemHandler(camSvc, tagSvc, notifMgr, calSvc, mediaSvc, weatherCli, meshSvc, otaSvc, storeSvc).
					WithLocalNVR(appcamera.NewLocalNVR(camSvc, srv.bus).WithRepository(srv.repos.NVR)).
					WithBackupRepository(srv.repos.Backup)

				domainRouter.Route("/cameras", func(cRouter chi.Router) {
					cRouter.Get("/", ecoH.ListCameras)
					cRouter.Post("/", ecoH.CreateCamera)
					cRouter.Get("/nvr/status", ecoH.GetNVRStatus)
					cRouter.Get("/faces", ecoH.ListFamilyFaces)
					cRouter.Post("/faces", ecoH.RegisterFamilyFace)
					cRouter.Delete("/faces/{id}", ecoH.DeleteFamilyFace)
					cRouter.Get("/plates", ecoH.ListAuthorizedPlates)
					cRouter.Post("/plates", ecoH.RegisterAuthorizedPlate)
					cRouter.Delete("/plates/{plate}", ecoH.DeleteAuthorizedPlate)
					cRouter.Post("/{id}/webrtc", ecoH.NegotiateWebRTC)
					cRouter.Get("/{id}/zones", ecoH.GetCameraZones)
					cRouter.Post("/{id}/zones", ecoH.AddCameraZone)
					cRouter.Post("/{id}/detect", ecoH.ProcessAIDetection)
					cRouter.Get("/events", ecoH.ListCameraEvents)
					cRouter.Get("/events/{id}/clip", ecoH.GetEventClip)
				})

				domainRouter.Get("/nvr/status", ecoH.GetNVRStatus)

				domainRouter.Route("/tags", func(tRouter chi.Router) {
					tRouter.Get("/", ecoH.ListTags)
					tRouter.Post("/", ecoH.CreateTag)
					tRouter.Post("/{id}/scan", ecoH.ScanTag)
				})

				domainRouter.Route("/notifications/actionable", func(nRouter chi.Router) {
					nRouter.Get("/", ecoH.ListActionableNotifications)
					nRouter.Post("/", ecoH.CreateActionableNotification)
					nRouter.Post("/actions/{action_id}", ecoH.HandleNotificationAction)
				})
				domainRouter.Get("/notifications", ecoH.ListActionableNotifications)

				if srv.repos.PushTokens != nil {
					pushSvc := appnotif.NewPushService(srv.repos.PushTokens, nil, srv.bus, nil)
					pushH := handlers.NewPushHandler(pushSvc)
					domainRouter.Route("/notifications/push", func(pRouter chi.Router) {
						pRouter.Post("/tokens", pushH.RegisterToken)
						pRouter.Delete("/tokens", pushH.RevokeToken)
						pRouter.Get("/tokens", pushH.ListTokens)
						pRouter.Post("/send", pushH.Send)
					})
				}

				domainRouter.Route("/schedules", func(sRouter chi.Router) {
					sRouter.Get("/", ecoH.ListSchedules)
					sRouter.Post("/", ecoH.CreateSchedule)
					sRouter.Put("/{id}", ecoH.UpdateSchedule)
					sRouter.Delete("/{id}", ecoH.DeleteSchedule)
				})

				if srv.repos.WeeklySchedules != nil {
					weeklySvc := appschedule.NewMatrixService(srv.repos.WeeklySchedules, srv.bus, nil)
					weeklyH := handlers.NewWeeklyScheduleHandler(weeklySvc)
					domainRouter.Route("/schedules/weekly", func(wRouter chi.Router) {
						wRouter.Get("/", weeklyH.List)
						wRouter.Post("/", weeklyH.Create)
						wRouter.Get("/{id}", weeklyH.Get)
						wRouter.Put("/{id}", weeklyH.Update)
						wRouter.Delete("/{id}", weeklyH.Delete)
						wRouter.Post("/evaluate", weeklyH.Evaluate)
					})
				}

				if srv.repos.Scripts != nil {
					scriptSvc := appscript.NewService(srv.repos.Scripts, srv.bus)
					scriptH := handlers.NewScriptHandler(scriptSvc)
					domainRouter.Route("/scripts", func(scRouter chi.Router) {
						scRouter.Get("/", scriptH.List)
						scRouter.Post("/", scriptH.Create)
						scRouter.Get("/{id}", scriptH.Get)
						scRouter.Put("/{id}", scriptH.Update)
						scRouter.Delete("/{id}", scriptH.Delete)
						scRouter.Post("/{id}/execute", scriptH.Execute)
					})
				}

				if srv.repos.Logbook != nil {
					logbookSvc := applogbook.NewService(srv.repos.Logbook, srv.repos.Entities, srv.bus)
					_ = logbookSvc.Start(context.Background())
					logbookH := handlers.NewLogbookHandler(logbookSvc)
					domainRouter.Route("/logbook", func(lRouter chi.Router) {
						lRouter.Get("/", logbookH.Query)
					})
				}

				if srv.repos.ClimateCompensations != nil {
					climateSvc, err := appclimate.NewService(srv.repos.ClimateCompensations, srv.bus, nil)
					if err == nil {
						climateH := handlers.NewClimateCompensatorHandler(climateSvc)
						domainRouter.Route("/climate/compensations", func(cRouter chi.Router) {
							cRouter.Get("/", climateH.List)
							cRouter.Post("/", climateH.Create)
							cRouter.Get("/{id}", climateH.Get)
							cRouter.Put("/{id}", climateH.Update)
							cRouter.Delete("/{id}", climateH.Delete)
						})
					}
				}

				if srv.repos.PredictiveClimate != nil {
					predSvc := appclimate.NewPredictiveService(srv.repos.PredictiveClimate, srv.repos.Entities, weatherCli, srv.bus, nil)
					predH := handlers.NewPredictiveClimateHandler(predSvc)
					domainRouter.Route("/climate/predictive", func(pRouter chi.Router) {
						predH.RegisterRoutes(pRouter)
					})
				}

				domainRouter.Route("/calendar", func(calRouter chi.Router) {
					calRouter.Get("/events", ecoH.ListCalendarEvents)
					calRouter.Post("/import", ecoH.ImportICal)
				})

				domainRouter.Route("/media", func(mRouter chi.Router) {
					mRouter.Get("/players", ecoH.ListMediaPlayers)
					mRouter.Post("/players", ecoH.CreateMediaPlayer)
					mRouter.Post("/players/{id}/volume", ecoH.SetPlayerVolume)
					mRouter.Post("/players/{id}/state", ecoH.SetPlayerState)
					mRouter.Delete("/players/{id}", ecoH.DeleteMediaPlayer)
					mRouter.Post("/broadcast", ecoH.BroadcastAnnouncement)
				})

				domainRouter.Get("/nvr/status", ecoH.GetNVRStatus)

				domainRouter.Route("/weather", func(wRouter chi.Router) {
					wRouter.Get("/forecast", ecoH.GetWeatherForecast)
					wRouter.Get("/solar-forecast", ecoH.GetSolarForecast)
				})

				domainRouter.Route("/mesh", func(meshRouter chi.Router) {
					meshRouter.Get("/topology", ecoH.GetMeshTopology)
					meshRouter.Get("/thread/topology", ecoH.GetThreadTopology)
				})

				domainRouter.Route("/system/remote-access", func(raRouter chi.Router) {
					raRouter.Get("/wireguard/server", ecoH.GetWireGuardServer)
					raRouter.Get("/wireguard/peers", ecoH.ListWireGuardPeers)
					raRouter.Post("/wireguard/peers", ecoH.EnrollWireGuardPeer)
					raRouter.Delete("/wireguard/peers/{id}", ecoH.RemoveWireGuardPeer)
					raRouter.Get("/tailscale", ecoH.GetTailscaleStatus)
					raRouter.Post("/tailscale/connect", ecoH.ConnectTailscale)
				})

				domainRouter.Route("/system/backup/remote", func(bkRouter chi.Router) {
					bkRouter.Get("/config", ecoH.GetRemoteBackupConfig)
					bkRouter.Put("/config", ecoH.UpdateRemoteBackupConfig)
					bkRouter.Post("/config", ecoH.UpdateRemoteBackupConfig)
					bkRouter.Post("/sync", ecoH.TriggerRemoteBackupSync)
					bkRouter.Get("/records", ecoH.ListRemoteBackupRecords)
				})

				domainRouter.Route("/voice", func(vRouter chi.Router) {
					vRouter.Post("/process", ecoH.ProcessVoiceSpeech)
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
				var shopRepo appshop.Repository = appshop.NewMemoryRepository()
				if srv.repos.Shopping != nil {
					shopRepo = srv.repos.Shopping
				}
				shopSvc := appshop.NewService(shopRepo, srv.bus)
				shopH := handlers.NewShoppingHandler(shopSvc)
				domainRouter.Route("/shopping", func(shRouter chi.Router) {
					shRouter.Get("/", shopH.List)
					shRouter.Post("/", shopH.Create)
					shRouter.Post("/{id}/toggle", shopH.Toggle)
					shRouter.Delete("/{id}", shopH.Delete)
					shRouter.Post("/clear-completed", shopH.ClearCompleted)
				})

				// Household Finance, Expenses & Budgets
				if srv.repos.Finance != nil {
					financeSvc := appfinance.NewService(srv.repos.Finance, srv.bus)
					financeH := handlers.NewFinanceHandler(financeSvc)
					domainRouter.Route("/finance", func(finRouter chi.Router) {
						finRouter.Get("/expenses", financeH.List)
						finRouter.Post("/expenses", financeH.Create)
						finRouter.Get("/expenses/{id}", financeH.GetByID)
						finRouter.Put("/expenses/{id}", financeH.Update)
						finRouter.Delete("/expenses/{id}", financeH.Delete)
						finRouter.Get("/summary", financeH.GetSummary)
						finRouter.Get("/budgets", financeH.ListBudgets)
						finRouter.Post("/budgets", financeH.SetBudget)
						finRouter.Get("/insights", financeH.GetInsights)
					})
				}

				// Household Food Inventory, Recipes, Meal Planning & Receipt OCR
				if srv.repos.Pantry != nil {
					var finRecorder apppantry.FinanceRecorder
					if srv.repos.Finance != nil {
						finRecorder = appfinance.NewService(srv.repos.Finance, srv.bus)
					}
					var shopAdder apppantry.ShoppingListAdder
					if srv.repos.Shopping != nil {
						shopSvc := appshop.NewService(srv.repos.Shopping, srv.bus)
						shopAdder = &shoppingPantryAdapter{svc: shopSvc}
					}
					pantrySvc := apppantry.NewService(srv.repos.Pantry, srv.bus, finRecorder, shopAdder, nil)
					pantryH := handlers.NewPantryHandler(pantrySvc)
					pantryH.RegisterRoutes(domainRouter)
				}

				if srv.repos.Medicines != nil {
					medSvc := appmed.NewService(srv.repos.Medicines, srv.bus)
					medH := handlers.NewMedicineHandler(medSvc)
					medH.RegisterRoutes(domainRouter)
				}

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
							whRouter.Get("/outbound", webhookH.ListOutbound)
							whRouter.Post("/outbound", webhookH.CreateOutbound)
							whRouter.Delete("/outbound/{id}", webhookH.DeleteOutbound)
							whRouter.Post("/outbound/{id}/test", webhookH.TestOutbound)
						})
						domainRouter.Route("/webhooks/outbound", func(obRouter chi.Router) {
							obRouter.Get("/", webhookH.ListOutbound)
							obRouter.Post("/", webhookH.CreateOutbound)
							obRouter.Delete("/{id}", webhookH.DeleteOutbound)
							obRouter.Post("/{id}/test", webhookH.TestOutbound)
						})
					}
				}

				// Local LLM & Voice Assistant
				assistantSvc := appassistant.NewService(appassistant.Config{Enabled: true}, srv.bus, srv.repos.Entities)
				assistantH := handlers.NewAssistantHandler(assistantSvc)
				assistantH.RegisterRoutes(domainRouter)

				// Cloud Voice Assistant Bridge (Amazon Alexa & Google Home Smart Home Actions)
				cloudAssistantSvc := appcloudassistant.NewService(srv.repos.Entities, srv.bus, "", true)
				cloudAssistantH := handlers.NewCloudAssistantHandler(cloudAssistantSvc)
				cloudAssistantH.RegisterRoutes(api)

				// Audit log endpoint (admin-only via RBAC)
				auditH := handlers.NewAuditHandler(srv.repos.Audit)
				if srv.authService != nil {
					domainRouter.With(domusmw.RequireRole(user.RoleAdmin)).Get("/audit", auditH.ServeHTTP)
				} else {
					domainRouter.Get("/audit", auditH.ServeHTTP)
				}

				// Network device discovery, IoT fingerprinting & guardian quarantine
				networkScanner := network.NewScanner(srv.bus, network.Config{})
				networkGuardian, _ := network.NewGuardian(srv.bus, network.GuardianConfig{})
				networkH := handlers.NewNetworkHandler(networkScanner).WithGuardian(networkGuardian)
				domainRouter.Get("/network/devices", networkH.ListDiscovered)
				domainRouter.Post("/network/scan", networkH.TriggerScan)
				domainRouter.Post("/network/probe", networkH.ProbeHost)
				domainRouter.Get("/network/leaks", networkH.ListLeaks)
				domainRouter.Post("/network/quarantine", networkH.QuarantineDevice)
				domainRouter.Post("/network/unblock", networkH.UnblockDevice)

				// Household Predictive Anomaly Engine (Water Leaks, Appliance Health, Condensation Mold)
				anomalySvc := appanomaly.NewService(srv.bus).WithRepository(srv.repos.Anomaly)
				anomalyH := handlers.NewAnomalyHandler(anomalySvc)
				domainRouter.Route("/anomalies", func(anRouter chi.Router) {
					anomalyH.RegisterRoutes(anRouter)
				})

				// Deterministic Sub-50ms Emergency Safe-Stop Pipeline
				safetySvc := appsafety.NewService(srv.bus).WithRepository(srv.repos.Safety)
				safetyH := handlers.NewSafetyHandler(safetySvc)
				domainRouter.Route("/safety", func(safRouter chi.Router) {
					safetyH.RegisterRoutes(safRouter)
				})
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

type shoppingPantryAdapter struct {
	svc *appshop.Service
}

func (a *shoppingPantryAdapter) AddItem(ctx context.Context, title string, quantity string, category string, addedBy string) error {
	_, err := a.svc.AddItem(ctx, title, quantity, domainShop.Category(category), addedBy)
	return err
}
