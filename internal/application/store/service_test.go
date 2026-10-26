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

func TestStoreService_CommunityWorkflow(t *testing.T) {
	repo := store.NewMemoryRepository()
	ctx := context.Background()
	svc := store.NewService(repo)

	// 1. Submit community package without repository URL should fail
	_, err := svc.SubmitCommunityPackage(ctx, store.SubmitPackageRequest{
		PackageID: "domus.community.missing-repo",
		Name:      "Missing Repo Plugin",
		License:   "MIT",
	})
	require.Error(t, err)

	// 2. Submit community package without license should fail
	_, err = svc.SubmitCommunityPackage(ctx, store.SubmitPackageRequest{
		PackageID:     "domus.community.missing-license",
		Name:          "Missing License Plugin",
		RepositoryURL: "https://github.com/community/plugin",
	})
	require.Error(t, err)

	// 3. Valid community package submission
	pkg, err := svc.SubmitCommunityPackage(ctx, store.SubmitPackageRequest{
		PackageID:     "domus.community.wasi-lora",
		Name:          "LoRa Gateway WASI",
		Type:          store.TypeWasmPlugin,
		Version:       "1.0.0",
		Author:        "LoRa Fans",
		Description:   "Community LoRa plugin",
		RepositoryURL: "https://github.com/community/wasi-lora",
		License:       "GPL-3.0",
		Capabilities:  []string{"wasi:serial-port", "events:write"},
	})
	require.NoError(t, err)
	assert.Equal(t, store.TierCommunity, pkg.Tier)
	assert.Equal(t, store.ReviewStatusPending, pkg.ReviewStatus)
	assert.Equal(t, "https://github.com/community/wasi-lora", pkg.RepositoryURL)
	assert.Equal(t, "GPL-3.0", pkg.License)

	// 4. Filter by status: pending
	pending, err := svc.ListPackagesWithOptions(ctx, "", "", string(store.ReviewStatusPending))
	require.NoError(t, err)
	assert.Len(t, pending, 1)

	// 5. Approve package
	approved, err := svc.ApprovePackage(ctx, pkg.ID)
	require.NoError(t, err)
	assert.Equal(t, store.ReviewStatusApproved, approved.ReviewStatus)

	// 6. Reject package
	rejected, err := svc.RejectPackage(ctx, pkg.ID, "Security concern")
	require.NoError(t, err)
	assert.Equal(t, store.ReviewStatusRejected, rejected.ReviewStatus)
}
