// Package auth_test provides tests for the AuthService use cases.
package auth_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthService(t *testing.T) (*auth.Service, *sqlite.Repositories) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "auth_service.db")

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

	svc := auth.NewService(repos.Users, repos.Sessions, repos.Audit, hasher, 24*time.Hour)
	return svc, repos
}

func TestAuthService_SetupInitialAdmin(t *testing.T) {
	svc, _ := setupAuthService(t)
	ctx := context.Background()

	// Initial check: setup required
	req, err := svc.IsSetupRequired(ctx)
	require.NoError(t, err)
	assert.True(t, req)

	// Setup admin
	admin, err := svc.SetupInitialAdmin(ctx, "admin", "SecureAdminPass123!")
	require.NoError(t, err)
	assert.Equal(t, "admin", admin.Username)
	assert.Equal(t, user.RoleAdmin, admin.Role)

	// Setup no longer required
	req, err = svc.IsSetupRequired(ctx)
	require.NoError(t, err)
	assert.False(t, req)

	// Second setup attempt must fail with ErrSetupAlreadyCompleted
	_, err = svc.SetupInitialAdmin(ctx, "another_admin", "AnotherPass123!")
	assert.ErrorIs(t, err, auth.ErrSetupAlreadyCompleted)
}

func TestAuthService_LoginAndSessionValidation(t *testing.T) {
	svc, _ := setupAuthService(t)
	ctx := context.Background()

	// Setup admin
	password := "CorrectPass123!"
	_, err := svc.SetupInitialAdmin(ctx, "admin", password)
	require.NoError(t, err)

	// 1. Invalid password
	sess, u, err := svc.Login(ctx, "admin", "WrongPass!", "127.0.0.1", "curl/7.68")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
	assert.Nil(t, sess)
	assert.Nil(t, u)

	// 2. Non-existent username
	_, _, err = svc.Login(ctx, "nonexistent", password, "127.0.0.1", "curl/7.68")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)

	// 3. Successful login
	sess, u, err = svc.Login(ctx, "admin", password, "192.168.1.100", "Mozilla/5.0")
	require.NoError(t, err)
	require.NotNil(t, sess)
	require.NotNil(t, u)
	assert.Equal(t, "admin", u.Username)
	assert.NotEmpty(t, sess.ID)
	assert.Equal(t, "192.168.1.100", sess.IPAddress)

	// 4. Validate session
	valUser, valSess, err := svc.ValidateSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, valUser.ID)
	assert.Equal(t, sess.ID, valSess.ID)

	// 5. Logout
	err = svc.Logout(ctx, sess.ID)
	require.NoError(t, err)

	// 6. Validate session after logout must fail
	_, _, err = svc.ValidateSession(ctx, sess.ID)
	assert.ErrorIs(t, err, user.ErrSessionNotFound)
}

func TestAuthService_GuestAndPinLogin(t *testing.T) {
	svc, _ := setupAuthService(t)
	ctx := context.Background()

	// Setup admin first
	_, err := svc.SetupInitialAdmin(ctx, "admin", "SecureAdminPass1!")
	require.NoError(t, err)

	// 1. Create a guest user with PIN "7890", valid for 24h
	guest, err := svc.CreateGuest(ctx, "Babysitter", "7890", 24, true)
	require.NoError(t, err)
	assert.Equal(t, "Babysitter", guest.Username)
	assert.Equal(t, user.RoleGuest, guest.Role)
	assert.True(t, guest.IsGuest)
	assert.True(t, guest.KioskMode)
	assert.NotNil(t, guest.GuestExpiresAt)

	// 2. Kiosk login with valid PIN
	sess, loggedInUser, err := svc.LoginWithPin(ctx, "7890", "192.168.1.50", "TabletKiosk/1.0")
	require.NoError(t, err)
	assert.Equal(t, guest.ID, loggedInUser.ID)
	assert.NotEmpty(t, sess.ID)

	// 3. Kiosk login with invalid PIN
	_, _, err = svc.LoginWithPin(ctx, "0000", "192.168.1.50", "TabletKiosk/1.0")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)

	// 4. Validate guest session
	valUser, _, err := svc.ValidateSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, guest.ID, valUser.ID)

	// 5. Test expired guest
	past := time.Now().UTC().Add(-1 * time.Hour)
	guest.GuestExpiresAt = &past
	assert.True(t, guest.IsGuestExpired())
}

