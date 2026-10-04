package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFamilyHandler(t *testing.T) (*handlers.FamilyHandler, *auth.Service, *user.User) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "family_handler_test.db")

	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, sqlite.Migrate(ctx, db))

	t.Cleanup(func() {
		_ = db.Close()
	})

	repos := sqlite.NewRepositories(db)
	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory:      32 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})

	svc := auth.NewService(repos.Users, repos.Sessions, repos.Audit, hasher, 24*time.Hour).
		WithInvites(repos.Invites)

	creator, err := svc.SetupInitialAdmin(ctx, "family_creator", "Password123")
	require.NoError(t, err)

	handler := handlers.NewFamilyHandler(svc, false)
	return handler, svc, creator
}

func TestFamilyHandler_InvitesAndRegistration(t *testing.T) {
	h, _, creator := setupFamilyHandler(t)

	// 1. Create Invite (POST /api/v1/family/invites)
	createReqBody := map[string]any{
		"role":             "operator",
		"expires_in_hours": 72,
	}
	bodyBytes, _ := json.Marshal(createReqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/family/invites", bytes.NewReader(bodyBytes))
	req = req.WithContext(domusmw.WithUser(req.Context(), creator))
	rr := httptest.NewRecorder()

	h.CreateInvite(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var inviteResp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &inviteResp))
	assert.Equal(t, "operator", inviteResp["role"])
	token, ok := inviteResp["token"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, token)

	// 2. List Invites (GET /api/v1/family/invites)
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/family/invites", nil)
	listReq = listReq.WithContext(domusmw.WithUser(listReq.Context(), creator))
	rrList := httptest.NewRecorder()

	h.ListInvites(rrList, listReq)
	assert.Equal(t, http.StatusOK, rrList.Code)

	var listResp map[string]any
	require.NoError(t, json.Unmarshal(rrList.Body.Bytes(), &listResp))
	invitesList, ok := listResp["invites"].([]any)
	require.True(t, ok)
	assert.Len(t, invitesList, 1)

	// 3. Validate Invite Token (GET /api/v1/auth/invite/{token})
	rCtx := chi.NewRouteContext()
	rCtx.URLParams.Add("token", token)
	valReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/invite/"+token, nil)
	valReq = valReq.WithContext(context.WithValue(valReq.Context(), chi.RouteCtxKey, rCtx))
	rrVal := httptest.NewRecorder()

	h.ValidateInvite(rrVal, valReq)
	assert.Equal(t, http.StatusOK, rrVal.Code)

	var valResp map[string]any
	require.NoError(t, json.Unmarshal(rrVal.Body.Bytes(), &valResp))
	assert.True(t, valResp["valid"].(bool))
	assert.Equal(t, "operator", valResp["role"])

	// 4. Register with Invite (POST /api/v1/auth/register)
	regReqBody := map[string]any{
		"token":    token,
		"username": "son_operator",
		"password": "SonPassword123",
		"email":    "son@home.local",
	}
	regBytes, _ := json.Marshal(regReqBody)
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBytes))
	rrReg := httptest.NewRecorder()

	h.RegisterWithInvite(rrReg, regReq)
	assert.Equal(t, http.StatusCreated, rrReg.Code)

	// Verify session cookie was set
	cookies := rrReg.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == domusmw.SessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)
	assert.NotEmpty(t, sessionCookie.Value)

	// Verify response body has operator role
	var regResp map[string]any
	require.NoError(t, json.Unmarshal(rrReg.Body.Bytes(), &regResp))
	userObj := regResp["user"].(map[string]any)
	assert.Equal(t, "son_operator", userObj["username"])
	assert.Equal(t, "operator", userObj["role"])

	// 5. List Family Members (GET /api/v1/family/members)
	membersReq := httptest.NewRequest(http.MethodGet, "/api/v1/family/members", nil)
	membersReq = membersReq.WithContext(domusmw.WithUser(membersReq.Context(), creator))
	rrMembers := httptest.NewRecorder()

	h.ListMembers(rrMembers, membersReq)
	assert.Equal(t, http.StatusOK, rrMembers.Code)

	var membersResp map[string]any
	require.NoError(t, json.Unmarshal(rrMembers.Body.Bytes(), &membersResp))
	membersList := membersResp["members"].([]any)
	assert.Len(t, membersList, 2)

	// 6. Update Role (PUT /api/v1/family/members/{id}/role)
	memberID := userObj["id"].(string)
	updateRoleBody, _ := json.Marshal(map[string]any{"role": "viewer"})
	rCtxUpdate := chi.NewRouteContext()
	rCtxUpdate.URLParams.Add("id", memberID)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/family/members/"+memberID+"/role", bytes.NewReader(updateRoleBody))
	updateReq = updateReq.WithContext(context.WithValue(updateReq.Context(), chi.RouteCtxKey, rCtxUpdate))
	updateReq = updateReq.WithContext(domusmw.WithUser(updateReq.Context(), creator))
	rrUpdate := httptest.NewRecorder()

	h.UpdateMemberRole(rrUpdate, updateReq)
	assert.Equal(t, http.StatusOK, rrUpdate.Code)

	// 7. Remove Member (DELETE /api/v1/family/members/{id})
	rCtxDelete := chi.NewRouteContext()
	rCtxDelete.URLParams.Add("id", memberID)
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/family/members/"+memberID, nil)
	delReq = delReq.WithContext(context.WithValue(delReq.Context(), chi.RouteCtxKey, rCtxDelete))
	delReq = delReq.WithContext(domusmw.WithUser(delReq.Context(), creator))
	rrDel := httptest.NewRecorder()

	h.RemoveMember(rrDel, delReq)
	assert.Equal(t, http.StatusNoContent, rrDel.Code)
}
