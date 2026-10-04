// Package handlers provides the audit log HTTP endpoint.
package handlers

import (
	"net/http"
	"strconv"

	"github.com/domus-hub/domus/internal/domain/audit"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
)

// AuditHandler serves audit log entries to authenticated administrators.
type AuditHandler struct {
	repo audit.Repository
}

// NewAuditHandler constructs an AuditHandler.
func NewAuditHandler(repo audit.Repository) *AuditHandler {
	return &AuditHandler{repo: repo}
}

// ServeHTTP handles GET /api/v1/audit
// Query params:
//
//	limit int  (default 50, max 500)
func (h *AuditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Must be authenticated
	if domusmw.GetUser(r.Context()) == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 500 {
				n = 500
			}
			limit = n
		}
	}

	entries, err := h.repo.List(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve audit log")
		return
	}

	if entries == nil {
		entries = []*audit.Entry{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
		"total":   len(entries),
	})
}
