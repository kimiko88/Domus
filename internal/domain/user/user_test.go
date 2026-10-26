package user

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRole_Valid(t *testing.T) {
	assert.True(t, RoleAdmin.Valid())
	assert.True(t, RoleOperator.Valid())
	assert.True(t, RoleViewer.Valid())
	assert.True(t, RoleGuest.Valid())
	assert.True(t, RoleKid.Valid())
	assert.False(t, Role("superuser").Valid())
	assert.False(t, Role("").Valid())
}

func TestPermissions_KidGuest(t *testing.T) {
	kid := DefaultKidPermissions("kid_1")
	assert.Equal(t, RoleKid, kid.Role)
	assert.False(t, kid.CanManageSettings)
	assert.False(t, kid.CanManageAutomations)
	assert.False(t, kid.CanViewCameras)
	assert.True(t, kid.CanAccessView("/"))
	assert.True(t, kid.CanAccessView("/media"))
	assert.False(t, kid.CanAccessView("/settings"))

	guest := DefaultGuestPermissions("guest_1")
	assert.Equal(t, RoleGuest, guest.Role)
	assert.False(t, guest.CanManageSettings)
	assert.True(t, guest.CanAccessView("/"))
	assert.False(t, guest.CanAccessView("/cameras"))
}

func TestUser_Validate(t *testing.T) {
	t.Run("valid user", func(t *testing.T) {
		u := &User{
			ID:           "u-1",
			Username:     "alice",
			Email:        "alice@example.com",
			PasswordHash: "$argon2id$v=19$...",
			Role:         RoleAdmin,
			Enabled:      true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		require.NoError(t, u.Validate())
	})

	t.Run("short username", func(t *testing.T) {
		u := &User{
			Username:     "al",
			PasswordHash: "hash",
			Role:         RoleOperator,
		}
		assert.ErrorIs(t, u.Validate(), ErrInvalidUsername)
	})

	t.Run("long username", func(t *testing.T) {
		u := &User{
			Username:     strings.Repeat("a", 65),
			PasswordHash: "hash",
			Role:         RoleOperator,
		}
		assert.ErrorIs(t, u.Validate(), ErrInvalidUsername)
	})

	t.Run("invalid role", func(t *testing.T) {
		u := &User{
			Username:     "validname",
			PasswordHash: "hash",
			Role:         Role("unknown"),
		}
		assert.ErrorIs(t, u.Validate(), ErrInvalidRole)
	})

	t.Run("guest role is valid", func(t *testing.T) {
		u := &User{
			Username:     "guest_user",
			PasswordHash: "hash",
			Role:         RoleGuest,
			IsGuest:      true,
		}
		assert.NoError(t, u.Validate())
	})

	t.Run("missing password hash", func(t *testing.T) {
		u := &User{
			Username: "validname",
			Role:     RoleViewer,
		}
		err := u.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password hash cannot be empty")
	})

	t.Run("password hash omitted from json", func(t *testing.T) {
		u := &User{
			ID:           "u-2",
			Username:     "bob",
			PasswordHash: "secret_hash",
			Role:         RoleViewer,
		}
		data, err := json.Marshal(u)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "secret_hash")
	})
}

func TestSession_IsExpired(t *testing.T) {
	sActive := &Session{
		ID:        "sess-1",
		UserID:    "u-1",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	assert.False(t, sActive.IsExpired())

	sExpired := &Session{
		ID:        "sess-2",
		UserID:    "u-1",
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	assert.True(t, sExpired.IsExpired())
}

func TestFamilyInvite(t *testing.T) {
	t.Run("valid invite", func(t *testing.T) {
		inv := &FamilyInvite{
			ID:              "inv-1",
			Token:           "tok_abc123",
			Role:            RoleOperator,
			CreatedByUserID: "u-1",
			ExpiresAt:       time.Now().Add(24 * time.Hour),
		}
		require.NoError(t, inv.Validate())
		assert.False(t, inv.IsExpired())
		assert.False(t, inv.IsUsed())
	})

	t.Run("invalid details", func(t *testing.T) {
		inv1 := &FamilyInvite{Token: "tok"}
		assert.ErrorIs(t, inv1.Validate(), ErrInvalidInvite)

		inv2 := &FamilyInvite{ID: "inv-2"}
		assert.ErrorIs(t, inv2.Validate(), ErrInvalidInvite)

		inv3 := &FamilyInvite{ID: "inv-3", Token: "tok", Role: Role("bad")}
		assert.ErrorIs(t, inv3.Validate(), ErrInvalidRole)
	})

	t.Run("expired and used check", func(t *testing.T) {
		now := time.Now()
		userID := "user-2"
		inv := &FamilyInvite{
			ID:           "inv-4",
			Token:        "tok",
			Role:         RoleViewer,
			ExpiresAt:    now.Add(-10 * time.Minute),
			UsedAt:       &now,
			UsedByUserID: &userID,
		}
		assert.True(t, inv.IsExpired())
		assert.True(t, inv.IsUsed())
	})
}
