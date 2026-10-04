package supervisor

import (
	"context"
	"encoding/json"
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
