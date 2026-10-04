package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInviteRepository(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	// 1. Create a creator user first
	creator := &user.User{
		ID:           uuid.NewString(),
		Username:     "creator",
		PasswordHash: "hash123",
		Role:         user.RoleAdmin,
		Enabled:      true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	require.NoError(t, repos.Users.Create(ctx, creator))

	// 2. Create family invite
	inv := &user.FamilyInvite{
		ID:              uuid.NewString(),
		Token:           "secret_token_12345",
		Role:            user.RoleOperator,
		CreatedByUserID: creator.ID,
		ExpiresAt:       time.Now().UTC().Add(48 * time.Hour),
		CreatedAt:       time.Now().UTC(),
	}
	err := repos.Invites.Create(ctx, inv)
	require.NoError(t, err)

	// 3. GetByToken
	fetched, err := repos.Invites.GetByToken(ctx, inv.Token)
	require.NoError(t, err)
	assert.Equal(t, inv.ID, fetched.ID)
	assert.Equal(t, user.RoleOperator, fetched.Role)
	assert.Equal(t, creator.ID, fetched.CreatedByUserID)
	assert.False(t, fetched.IsUsed())
	assert.False(t, fetched.IsExpired())

	// 4. GetByID
	fetchedByID, err := repos.Invites.GetByID(ctx, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, inv.Token, fetchedByID.Token)

	// 5. ListActive
	activeList, err := repos.Invites.ListActive(ctx)
	require.NoError(t, err)
	assert.Len(t, activeList, 1)

	// 6. Create invited member user
	member := &user.User{
		ID:           uuid.NewString(),
		Username:     "family_member",
		PasswordHash: "hash456",
		Role:         user.RoleOperator,
		Enabled:      true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	require.NoError(t, repos.Users.Create(ctx, member))

	// 7. MarkUsed
	now := time.Now().UTC()
	err = repos.Invites.MarkUsed(ctx, inv.ID, member.ID, now)
	require.NoError(t, err)

	// Verify it's no longer in ListActive
	activeAfterUsed, err := repos.Invites.ListActive(ctx)
	require.NoError(t, err)
	assert.Len(t, activeAfterUsed, 0)

	// Verify MarkUsed again fails with ErrInviteUsed
	err = repos.Invites.MarkUsed(ctx, inv.ID, member.ID, now)
	assert.ErrorIs(t, err, user.ErrInviteUsed)

	// 8. Delete
	err = repos.Invites.Delete(ctx, inv.ID)
	require.NoError(t, err)
	_, err = repos.Invites.GetByID(ctx, inv.ID)
	assert.ErrorIs(t, err, user.ErrInviteNotFound)
}

func TestUserRepository_List_UpdateRole_Delete(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	u1 := &user.User{
		ID:           uuid.NewString(),
		Username:     "user1",
		PasswordHash: "hash1",
		Role:         user.RoleAdmin,
		Enabled:      true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	u2 := &user.User{
		ID:           uuid.NewString(),
		Username:     "user2",
		PasswordHash: "hash2",
		Role:         user.RoleViewer,
		Enabled:      true,
		CreatedAt:    time.Now().UTC().Add(time.Second),
		UpdatedAt:    time.Now().UTC().Add(time.Second),
	}
	require.NoError(t, repos.Users.Create(ctx, u1))
	require.NoError(t, repos.Users.Create(ctx, u2))

	// Test List
	list, err := repos.Users.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, "user1", list[0].Username)
	assert.Equal(t, "user2", list[1].Username)

	// Test UpdateRole
	err = repos.Users.UpdateRole(ctx, u2.ID, user.RoleOperator)
	require.NoError(t, err)

	updatedU2, err := repos.Users.GetByID(ctx, u2.ID)
	require.NoError(t, err)
	assert.Equal(t, user.RoleOperator, updatedU2.Role)

	// Test Delete
	err = repos.Users.Delete(ctx, u2.ID)
	require.NoError(t, err)

	_, err = repos.Users.GetByID(ctx, u2.ID)
	assert.ErrorIs(t, err, user.ErrUserNotFound)

	count, err := repos.Users.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
