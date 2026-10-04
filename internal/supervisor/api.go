package supervisor

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// APIServer exposes the local Unix domain socket and loopback API for the supervisor.
type APIServer struct {
	engine *Engine
	mux    *http.ServeMux
}

// NewAPIServer creates and registers all supervisor management endpoints.
func NewAPIServer(engine *Engine) *APIServer {
	s := &APIServer{
		engine: engine,
		mux:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// Handler returns the HTTP handler for testing and serving.
func (s *APIServer) Handler() http.Handler {
	return s.mux
}

func (s *APIServer) registerRoutes() {
	s.mux.HandleFunc("GET /api/v1/supervisor/ping", s.handlePing)
	s.mux.HandleFunc("GET /api/v1/supervisor/status", s.handleStatus)
	s.mux.HandleFunc("GET /api/v1/supervisor/health", s.handleHealth)
	s.mux.HandleFunc("POST /api/v1/supervisor/core/restart", s.handleCoreRestart)
	s.mux.HandleFunc("POST /api/v1/supervisor/update/confirm", s.handleUpdateConfirm)
	s.mux.HandleFunc("POST /api/v1/supervisor/update/rollback", s.handleUpdateRollback)

	// App Management Endpoints
	s.mux.HandleFunc("GET /api/v1/apps", s.handleListApps)
	s.mux.HandleFunc("GET /api/v1/apps/{id}", s.handleGetApp)
	s.mux.HandleFunc("POST /api/v1/apps/install", s.handleInstallApp)
	s.mux.HandleFunc("POST /api/v1/apps/{id}/start", s.handleStartApp)
	s.mux.HandleFunc("POST /api/v1/apps/{id}/stop", s.handleStopApp)
	s.mux.HandleFunc("DELETE /api/v1/apps/{id}", s.handleUninstallApp)

	// System Diagnostics & Maintenance Endpoints
	s.mux.HandleFunc("GET /api/v1/system/diagnostics", s.handleDiagnostics)
	s.mux.HandleFunc("GET /api/v1/system/diagnostics/export", s.handleDiagnosticsExport)
	s.mux.HandleFunc("GET /api/v1/system/maintenance/status", s.handleMaintenanceStatus)
	s.mux.HandleFunc("POST /api/v1/system/maintenance/enable", s.handleMaintenanceEnable)
	s.mux.HandleFunc("POST /api/v1/system/maintenance/disable", s.handleMaintenanceDisable)
}

func (s *APIServer) handlePing(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "pong",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	st := s.engine.GetStatus()
	writeJSON(w, http.StatusOK, st)
}

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	st := s.engine.GetStatus()
	if st.CoreHealthy {
		writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	} else {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unhealthy"})
	}
}

func (s *APIServer) handleCoreRestart(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.RestartCore(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "core restart initiated"})
}

func (s *APIServer) handleUpdateConfirm(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.rauc.MarkGood(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "slot confirmed good"})
}

func (s *APIServer) handleUpdateRollback(w http.ResponseWriter, r *http.Request) {
	if err := s.engine.rauc.Rollback(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "rollback initiated"})
}

func (s *APIServer) handleListApps(w http.ResponseWriter, r *http.Request) {
	apps, err := s.engine.GetAppManager().ListApps(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, apps)
}

func (s *APIServer) handleGetApp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	manifest, status, err := s.engine.GetAppManager().GetApp(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"manifest": manifest,
		"status":   status,
	})
}

func (s *APIServer) handleInstallApp(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body with manifest is required"})
		return
	}
	manifest, err := s.engine.GetAppManager().Install(r.Context(), body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, manifest)
}

func (s *APIServer) handleStartApp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.engine.GetAppManager().Start(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "app started"})
}

func (s *APIServer) handleStopApp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.engine.GetAppManager().Stop(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "app stopped"})
}

func (s *APIServer) handleUninstallApp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	purge := r.URL.Query().Get("purge") == "true"
	if err := s.engine.GetAppManager().Uninstall(r.Context(), id, purge); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "app uninstalled"})
}

func (s *APIServer) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	diag, err := s.engine.GetDiagnosticsCollector().Collect(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, diag)
}

func (s *APIServer) handleDiagnosticsExport(w http.ResponseWriter, r *http.Request) {
	export, err := s.engine.GetDiagnosticsCollector().Export(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, export)
}

func (s *APIServer) handleMaintenanceStatus(w http.ResponseWriter, r *http.Request) {
	status := s.engine.GetMaintenanceManager().GetStatus()
	writeJSON(w, http.StatusOK, status)
}

func (s *APIServer) handleMaintenanceEnable(w http.ResponseWriter, r *http.Request) {
	var req EnableMaintenanceRequest
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	status, err := s.engine.GetMaintenanceManager().Enable(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *APIServer) handleMaintenanceDisable(w http.ResponseWriter, r *http.Request) {
	status, err := s.engine.GetMaintenanceManager().Disable(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// ServeUnix binds the HTTP server to a Unix Domain Socket with proper permissions.
func (s *APIServer) ServeUnix(ctx context.Context, socketPath string) error {
	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer listener.Close()
	defer os.Remove(socketPath)

	_ = os.Chmod(socketPath, 0660)

	server := &http.Server{
		Handler: s.mux,
	}

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	return server.Serve(listener)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
