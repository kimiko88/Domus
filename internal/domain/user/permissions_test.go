package user_test

import (
	"testing"

	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPermissionsRestrictKidAndGuestResources(t *testing.T) {
	for _, role := range []user.Role{user.RoleKid, user.RoleGuest} {
		permissions := user.DefaultPermissions("member-1", role)
		require.NotNil(t, permissions)
		assert.False(t, permissions.CanControlDevices)
		assert.False(t, permissions.CanManageAutomations)
		assert.False(t, permissions.CanManageSettings)
		assert.False(t, permissions.CanViewCameras)
		assert.False(t, permissions.CanAccessArea("area-1"))
		assert.False(t, permissions.CanAccessEntity("entity-1"))
	}
}

func TestNilAndEmptyAllowListsHaveDifferentMeaning(t *testing.T) {
	all := &user.UserPermissions{Role: user.RoleViewer}
	assert.True(t, all.CanAccessArea("area-1"))
	assert.True(t, all.CanAccessEntity("entity-1"))

	none := &user.UserPermissions{
		Role: user.RoleViewer,
		AllowedAreaIDs: []string{},
		AllowedEntityIDs: []string{},
	}
	assert.False(t, none.CanAccessArea("area-1"))
	assert.False(t, none.CanAccessEntity("entity-1"))
}

func TestCanAccessEntityInAreaRequiresBothGrants(t *testing.T) {
	member := &user.User{
		ID: "member-1",
		Role: user.RoleOperator,
		Permissions: &user.UserPermissions{
			AllowedAreaIDs: []string{"kitchen"},
			AllowedEntityIDs: []string{"light-kitchen", "light-bedroom"},
		},
	}
	assert.True(t, member.CanAccessEntityInArea("light-kitchen", "kitchen"))
	assert.False(t, member.CanAccessEntityInArea("light-bedroom", "bedroom"))
	assert.False(t, member.CanAccessEntityInArea("light-kitchen", ""))
	assert.False(t, member.CanAccessEntityInArea("unknown", "kitchen"))
}

func TestAdminAlwaysHasFullPermissions(t *testing.T) {
	admin := &user.User{
		ID: "admin-1",
		Role: user.RoleAdmin,
		Permissions: &user.UserPermissions{
			AllowedAreaIDs: []string{},
			AllowedEntityIDs: []string{},
			CanControlDevices: false,
		},
	}
	permissions := admin.EffectivePermissions()
	require.NotNil(t, permissions)
	assert.True(t, permissions.CanControlDevices)
	assert.True(t, permissions.CanManageSettings)
	assert.True(t, permissions.CanViewCameras)
	assert.True(t, permissions.CanAccessArea("any-area"))
	assert.True(t, permissions.CanAccessEntity("any-entity"))
}

func TestViewerDefaultsAreReadOnlyAndHideCameras(t *testing.T) {
	permissions := user.DefaultPermissions("viewer-1", user.RoleViewer)
	require.NotNil(t, permissions)
	assert.False(t, permissions.CanControlDevices)
	assert.False(t, permissions.CanManageAutomations)
	assert.False(t, permissions.CanManageSettings)
	assert.False(t, permissions.CanViewCameras)
	assert.True(t, permissions.CanAccessArea("living-room"))
	assert.True(t, permissions.CanAccessView("/"))
	assert.False(t, permissions.CanAccessView("/family"))
	assert.False(t, permissions.CanAccessView("/automations"))
	assert.False(t, permissions.CanAccessDashboard("dashboard-1"))
}
