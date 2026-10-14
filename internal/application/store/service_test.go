package store_test

import (
	"context"
	"testing"

	"github.com/domus-hub/domus/internal/application/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreService_InstallAndList(t *testing.T) {
	repo := store.NewMemoryRepository()
	ctx := context.Background()

	testPkg := &store.Package{
		ID:        "pkg_test_1",
		PackageID: "blueprints.lighting.test",
		Name:      "Test Blueprint",
		Type:      store.TypeBlueprint,
		Version:   "1.0.0",
		Author:    "Community",
	}
	require.NoError(t, repo.SavePackage(ctx, testPkg))

	svc := store.NewService(repo)

	pkgs, err := svc.ListPackages(ctx, "")
	require.NoError(t, err)
	assert.NotEmpty(t, pkgs)

	// Filter by blueprint
	bpPkgs, err := svc.ListPackages(ctx, string(store.TypeBlueprint))
	require.NoError(t, err)
	assert.NotEmpty(t, bpPkgs)
	assert.Equal(t, store.TypeBlueprint, bpPkgs[0].Type)

	// Install package
	installed, err := svc.InstallPackage(ctx, bpPkgs[0].ID)
	require.NoError(t, err)
	assert.True(t, installed.Installed)
	assert.NotNil(t, installed.InstalledAt)

	// Uninstall
	uninstalled, err := svc.UninstallPackage(ctx, bpPkgs[0].ID)
	require.NoError(t, err)
	assert.False(t, uninstalled.Installed)
}
