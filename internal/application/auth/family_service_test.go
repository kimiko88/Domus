package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFamilyService_Lifecycle(t *testing.T) {
	svc, repos := setupAuthService(t)
	svc.WithInvites(repos.Invites)
	ctx := context.Background()

	// 1. Initial Admin setup (family creator)
	admin, err := svc.SetupInitialAdmin(ctx, "family_creator", "SuperSecret123")
	require.NoError(t, err)
	assert.Equal(t, user.RoleAdmin, admin.Role)

	// 2. Creator creates an invitation for an Operator member (e.g. spouse)
	inviteOperator, err := svc.CreateFamilyInvite(ctx, admin.ID, user.RoleOperator, 48*time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, inviteOperator.Token)
	assert.Equal(t, user.RoleOperator, inviteOperator.Role)
	assert.Equal(t, admin.ID, inviteOperator.CreatedByUserID)
	assert.False(t, inviteOperator.IsUsed())
	assert.False(t, inviteOperator.IsExpired())

	// 3. Creator creates an invitation for a Viewer member (e.g. child / guest)
	inviteViewer, err := svc.CreateFamilyInvite(ctx, admin.ID, user.RoleViewer, 24*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, user.RoleViewer, inviteViewer.Role)

	// 4. List pending invites
	invites, err := svc.ListFamilyInvites(ctx)
	require.NoError(t, err)
	assert.Len(t, invites, 2)

	// 5. Get invite by token (public inspection before registration)
	fetchedInv, err := svc.GetInviteByToken(ctx, inviteOperator.Token)
	require.NoError(t, err)
	assert.Equal(t, user.RoleOperator, fetchedInv.Role)

	// 6. Register invited operator member
	operatorUser, opSession, err := svc.RegisterWithInvite(
		ctx,
		inviteOperator.Token,
		"spouse_operator",
		"SecurePassword456",
		"spouse@home.local",
		"192.168.1.10",
		"Mozilla/5.0",
	)
	require.NoError(t, err)
	assert.NotNil(t, operatorUser)
	assert.NotNil(t, opSession)
	assert.Equal(t, user.RoleOperator, operatorUser.Role)
	assert.Equal(t, "spouse_operator", operatorUser.Username)
	assert.Equal(t, operatorUser.ID, opSession.UserID)

	// Verify token cannot be reused
	_, _, err = svc.RegisterWithInvite(
		ctx,
		inviteOperator.Token,
		"another_person",
		"Password123",
		"",
		"192.168.1.11",
		"Mobile",
	)
	assert.ErrorIs(t, err, user.ErrInviteUsed)

	// 7. Register invited viewer member
	viewerUser, _, err := svc.RegisterWithInvite(
		ctx,
		inviteViewer.Token,
		"child_viewer",
		"KidPassword123",
		"",
		"192.168.1.12",
		"Tablet",
	)
	require.NoError(t, err)
	assert.Equal(t, user.RoleViewer, viewerUser.Role)

	// 8. List family members
	members, err := svc.ListFamilyMembers(ctx)
	require.NoError(t, err)
	assert.Len(t, members, 3)

	// 9. Update family member role: promote viewer to operator
	err = svc.UpdateFamilyMemberRole(ctx, admin.ID, viewerUser.ID, user.RoleOperator)
	require.NoError(t, err)

	updatedViewer, err := repos.Users.GetByID(ctx, viewerUser.ID)
	require.NoError(t, err)
	assert.Equal(t, user.RoleOperator, updatedViewer.Role)

	// 10. Attempting to demote the sole admin must fail
	err = svc.UpdateFamilyMemberRole(ctx, admin.ID, admin.ID, user.RoleViewer)
	assert.ErrorIs(t, err, auth.ErrCannotDemoteLastAdmin)

	// 11. Remove member
	err = svc.RemoveFamilyMember(ctx, admin.ID, viewerUser.ID)
	require.NoError(t, err)

	membersAfterRemove, err := svc.ListFamilyMembers(ctx)
	require.NoError(t, err)
	assert.Len(t, membersAfterRemove, 2)

	// Cannot remove oneself
	err = svc.RemoveFamilyMember(ctx, admin.ID, admin.ID)
	assert.ErrorIs(t, err, auth.ErrCannotRemoveSelf)

	// 12. Revoke an invitation
	extraInvite, err := svc.CreateFamilyInvite(ctx, admin.ID, user.RoleViewer, 1*time.Hour)
	require.NoError(t, err)
	err = svc.RevokeFamilyInvite(ctx, extraInvite.ID)
	require.NoError(t, err)

	_, err = svc.GetInviteByToken(ctx, extraInvite.Token)
	assert.ErrorIs(t, err, user.ErrInviteNotFound)
}
