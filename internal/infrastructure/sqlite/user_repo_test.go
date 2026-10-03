// Package sqlite_test tests the SQLite repositories for users, sessions, and audit logs (TDD red-to-green phase).
package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/audit"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sqlite.Repositories {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_repo.db")

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

	return sqlite.NewRepositories(db)
}

func TestUserRepository(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	// Initial count must be 0
	count, err := repos.Users.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	now := time.Now().UTC().Truncate(time.Second)
	u := &user.User{
		ID:           uuid.NewString(),
		Username:     "admin",
		Email:        "admin@domus.local",
		PasswordHash: "$argon2id$v=19$...",
		Role:         user.RoleAdmin,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Create user
	err = repos.Users.Create(ctx, u)
	require.NoError(t, err)

	// Count must be 1
	count, err = repos.Users.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Get by ID
	fetched, err := repos.Users.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, fetched.ID)
	assert.Equal(t, u.Username, fetched.Username)
	assert.Equal(t, u.Email, fetched.Email)
	assert.Equal(t, u.Role, fetched.Role)
	assert.True(t, fetched.Enabled)

	// Get by Username
	byName, err := repos.Users.GetByUsername(ctx, "admin")
	require.NoError(t, err)
	assert.Equal(t, u.ID, byName.ID)

	// Non-existent user
	_, err = repos.Users.GetByID(ctx, "non-existent")
	assert.ErrorIs(t, err, user.ErrUserNotFound)

	_, err = repos.Users.GetByUsername(ctx, "nobody")
	assert.ErrorIs(t, err, user.ErrUserNotFound)

	// Duplicate username must fail
	duplicateUser := &user.User{
		ID:           uuid.NewString(),
		Username:     "admin",
		PasswordHash: "hash2",
		Role:         user.RoleViewer,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	err = repos.Users.Create(ctx, duplicateUser)
	assert.Error(t, err, "Duplicate username must violate unique constraint")
}

func TestSessionRepository(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	// Create user for FK
	u := &user.User{
		ID:           uuid.NewString(),
		Username:     "operator1",
		PasswordHash: "hash",
		Role:         user.RoleOperator,
		Enabled:      true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	require.NoError(t, repos.Users.Create(ctx, u))

	sess := &user.Session{
		ID:        uuid.NewString(),
		UserID:    u.ID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second),
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		IPAddress: "192.168.1.50",
		UserAgent: "Mozilla/5.0",
	}

	// Create session
	require.NoError(t, repos.Sessions.Create(ctx, sess))

	// Get session
	fetched, err := repos.Sessions.GetByID(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, fetched.ID)
	assert.Equal(t, sess.UserID, fetched.UserID)
	assert.Equal(t, sess.IPAddress, fetched.IPAddress)

	// Delete session
	require.NoError(t, repos.Sessions.Delete(ctx, sess.ID))

	// Should be not found now
	_, err = repos.Sessions.GetByID(ctx, sess.ID)
	assert.ErrorIs(t, err, user.ErrSessionNotFound)

	// Test DeleteExpired
	expiredSess := &user.Session{
		ID:        uuid.NewString(),
		UserID:    u.ID,
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
	}
	require.NoError(t, repos.Sessions.Create(ctx, expiredSess))

	err = repos.Sessions.DeleteExpired(ctx, time.Now().UTC())
	require.NoError(t, err)

	_, err = repos.Sessions.GetByID(ctx, expiredSess.ID)
	assert.ErrorIs(t, err, user.ErrSessionNotFound)
}

func TestAuditRepository(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	entry1 := &audit.Entry{
		ActorType:  audit.ActorLocalUser,
		ActorID:    "usr-1",
		Action:     "auth.login.success",
		TargetType: "session",
		TargetID:   "sess-1",
		Payload:    `{"ip":"127.0.0.1"}`,
		CreatedAt:  time.Now().UTC().Add(-1 * time.Minute),
	}
	entry2 := &audit.Entry{
		ActorType:  audit.ActorSystem,
		Action:     "auth.setup.admin_created",
		TargetType: "user",
		TargetID:   "usr-1",
		Payload:    `{}`,
		CreatedAt:  time.Now().UTC(),
	}

	require.NoError(t, repos.Audit.Record(ctx, entry1))
	require.NoError(t, repos.Audit.Record(ctx, entry2))

	entries, err := repos.Audit.List(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	// Must be ordered by created_at DESC
	assert.Equal(t, "auth.setup.admin_created", entries[0].Action)
	assert.Equal(t, "auth.login.success", entries[1].Action)
}
