// Package config — TDD RED: audit handler tests.
package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/audit"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuditRepo is an in-memory audit repository for tests.
type mockAuditRepo struct {
	entries []*audit.Entry
}

func (m *mockAuditRepo) Record(_ context.Context, e *audit.Entry) error {
	e.ID = int64(len(m.entries) + 1)
	m.entries = append(m.entries, e)
	return nil
}

func (m *mockAuditRepo) List(_ context.Context, limit int) ([]*audit.Entry, error) {
	if limit <= 0 || limit > len(m.entries) {
		return m.entries, nil
	}
	return m.entries[:limit], nil
}

func (m *mockAuditRepo) PurgeOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

func newAdminContext(r *http.Request) *http.Request {
	admin := &user.User{ID: "u1", Username: "admin", Role: user.RoleAdmin, Enabled: true}
	return r.WithContext(domusmw.WithUser(r.Context(), admin))
}

func TestAuditHandler_List_RequiresAuth(t *testing.T) {
	repo := &mockAuditRepo{}
	h := handlers.NewAuditHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuditHandler_List_ReturnsEntries(t *testing.T) {
	repo := &mockAuditRepo{
		entries: []*audit.Entry{
			{
				ID:        1,
				ActorType: audit.ActorLocalUser,
				ActorID:   "u1",
				Action:    "auth.login.success",
				CreatedAt: time.Now().UTC(),
			},
		},
	}
	h := handlers.NewAuditHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?limit=10", nil)
	req = newAdminContext(req)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp struct {
		Entries []*audit.Entry `json:"entries"`
		Total   int            `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Len(t, resp.Entries, 1)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, "auth.login.success", resp.Entries[0].Action)
}

func TestAuditHandler_List_LimitQueryParam(t *testing.T) {
	repo := &mockAuditRepo{}
	for i := 0; i < 20; i++ {
		repo.entries = append(repo.entries, &audit.Entry{
			ID: int64(i + 1), ActorType: audit.ActorSystem, Action: "test",
		})
	}
	h := handlers.NewAuditHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?limit=5", nil)
	req = newAdminContext(req)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Entries []*audit.Entry `json:"entries"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Len(t, resp.Entries, 5)
}
