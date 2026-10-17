package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/ota"
	"github.com/domus-hub/domus/internal/application/store"
	"github.com/domus-hub/domus/internal/domain/calendar"
	"github.com/domus-hub/domus/internal/domain/camera"
	"github.com/domus-hub/domus/internal/domain/media"
	"github.com/domus-hub/domus/internal/domain/schedule"
	"github.com/domus-hub/domus/internal/domain/shopping"
	"github.com/domus-hub/domus/internal/domain/tag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShoppingRepository_SQLite(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	// 1. Create item
	item := &shopping.Item{
		ID:        "shop_item_1",
		Name:      "Latte Intero",
		Category:  shopping.CategoryGrocery,
		Quantity:  "2L",
		Completed: false,
		AddedBy:   "user_1",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
	err := repos.Shopping.SaveItem(ctx, item)
	require.NoError(t, err)

	// 2. GetItem
	got, err := repos.Shopping.GetItem(ctx, "shop_item_1")
	require.NoError(t, err)
	assert.Equal(t, "Latte Intero", got.Name)
	assert.Equal(t, "2L", got.Quantity)
	assert.False(t, got.Completed)

	// 3. ListItems
	items, err := repos.Shopping.ListItems(ctx)
	require.NoError(t, err)
	assert.Len(t, items, 1)

	// 4. Update item (mark completed)
	got.Completed = true
	now := time.Now().UTC().Truncate(time.Second)
	got.CompletedAt = &now
	user := "user_2"
	got.CompletedBy = &user
	err = repos.Shopping.SaveItem(ctx, got)
	require.NoError(t, err)

	updated, err := repos.Shopping.GetItem(ctx, "shop_item_1")
	require.NoError(t, err)
	assert.True(t, updated.Completed)
	assert.Equal(t, "user_2", *updated.CompletedBy)

	// 5. Clear completed
	cleared, err := repos.Shopping.ClearCompleted(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, cleared)

	itemsAfter, err := repos.Shopping.ListItems(ctx)
	require.NoError(t, err)
	assert.Empty(t, itemsAfter)

	// 6. DeleteItem
	item2 := &shopping.Item{
		ID:        "shop_item_2",
		Name:      "Mele",
		Category:  shopping.CategoryProduce,
		Quantity:  "1kg",
		Completed: false,
		AddedBy:   "user_1",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
	err = repos.Shopping.SaveItem(ctx, item2)
	require.NoError(t, err)

	err = repos.Shopping.DeleteItem(ctx, "shop_item_2")
	require.NoError(t, err)

	_, err = repos.Shopping.GetItem(ctx, "shop_item_2")
	assert.Error(t, err)
}

func TestCalendarRepository_SQLite(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	// 1. Schedules
	sched := &schedule.Schedule{
		ID:        "sched_1",
		Name:      "Riscaldamento Mattina",
		EntityKey: "climate.living_room",
		Grid: schedule.WeeklyGrid{
			"monday": []schedule.TimeSlot{
				{Start: "07:00", End: "09:00", Value: 21.0},
			},
		},
		Enabled:   true,
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
	}
	err := repos.Calendar.SaveSchedule(ctx, sched)
	require.NoError(t, err)

	gotSched, err := repos.Calendar.GetSchedule(ctx, "sched_1")
	require.NoError(t, err)
	assert.Equal(t, "Riscaldamento Mattina", gotSched.Name)
	assert.Equal(t, "climate.living_room", gotSched.EntityKey)

	scheds, err := repos.Calendar.ListSchedules(ctx)
	require.NoError(t, err)
	assert.Len(t, scheds, 1)

	gotSched.Enabled = false
	err = repos.Calendar.SaveSchedule(ctx, gotSched)
	require.NoError(t, err)

	updatedSched, err := repos.Calendar.GetSchedule(ctx, "sched_1")
	require.NoError(t, err)
	assert.False(t, updatedSched.Enabled)

	// 2. Events
	now := time.Now().UTC().Truncate(time.Second)
	evt := &calendar.Event{
		ID:          "cal_evt_1",
		Summary:     "Festa di Compleanno",
		Description: "Torta e regali in giardino",
		Location:    "Salotto",
		StartTime:   now,
		EndTime:     now.Add(2 * time.Hour),
		AllDay:      false,
		Source:      "local",
		CreatedAt:   now,
	}
	err = repos.Calendar.SaveCalendarEvent(ctx, evt)
	require.NoError(t, err)

	evts, err := repos.Calendar.ListCalendarEvents(ctx, now.Add(-time.Hour), now.Add(3*time.Hour))
	require.NoError(t, err)
	assert.Len(t, evts, 1)
	assert.Equal(t, "Festa di Compleanno", evts[0].Summary)

	err = repos.Calendar.DeleteCalendarEvent(ctx, "cal_evt_1")
	require.NoError(t, err)

	err = repos.Calendar.DeleteSchedule(ctx, "sched_1")
	require.NoError(t, err)
}

func TestTagRepository_SQLite(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	tg := &tag.Tag{
		ID:             "tag_nfc_001",
		TagID:          "04:A2:3B:5F:81",
		Name:           "NFC Portachiavi Auto",
		ActionType:     tag.ActionTypeAutomation,
		ActionTargetID: "auto_open_garage",
		LastScannedAt:  &now,
		CreatedAt:      now,
	}
	err := repos.Tags.SaveTag(ctx, tg)
	require.NoError(t, err)

	got, err := repos.Tags.GetTagByTagID(ctx, "04:A2:3B:5F:81")
	require.NoError(t, err)
	assert.Equal(t, "04:A2:3B:5F:81", got.TagID)
	assert.Equal(t, "NFC Portachiavi Auto", got.Name)
	assert.Equal(t, tag.ActionTypeAutomation, got.ActionType)

	list, err := repos.Tags.ListTags(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	got.Name = "Portachiavi Modificato"
	err = repos.Tags.SaveTag(ctx, got)
	require.NoError(t, err)

	updated, err := repos.Tags.GetTagByTagID(ctx, "04:A2:3B:5F:81")
	require.NoError(t, err)
	assert.Equal(t, "Portachiavi Modificato", updated.Name)

	err = repos.Tags.DeleteTag(ctx, "tag_nfc_001")
	require.NoError(t, err)

	listAfter, err := repos.Tags.ListTags(ctx)
	require.NoError(t, err)
	assert.Empty(t, listAfter)
}

func TestMediaRepository_SQLite(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	player := &media.MediaPlayer{
		ID:            "media_living_room",
		Name:          "Soundbar TV Salotto",
		Platform:      media.PlatformDLNA,
		State:         media.StatePlaying,
		Volume:        0.75,
		Muted:         false,
		CurrentTitle:  "Bohemian Rhapsody",
		CurrentArtist: "Queen",
		CreatedAt:     time.Now().UTC().Truncate(time.Second),
	}
	err := repos.Media.SavePlayer(ctx, player)
	require.NoError(t, err)

	got, err := repos.Media.GetPlayer(ctx, "media_living_room")
	require.NoError(t, err)
	assert.Equal(t, "Soundbar TV Salotto", got.Name)
	assert.Equal(t, 0.75, got.Volume)
	assert.Equal(t, "Bohemian Rhapsody", got.CurrentTitle)

	list, err := repos.Media.ListPlayers(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// Update existing player
	player.Volume = 0.80
	player.State = media.StatePaused
	err = repos.Media.SavePlayer(ctx, player)
	require.NoError(t, err)

	updated, err := repos.Media.GetPlayer(ctx, "media_living_room")
	require.NoError(t, err)
	assert.Equal(t, 0.80, updated.Volume)
	assert.Equal(t, media.StatePaused, updated.State)
}

func TestCameraRepository_SQLite(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	cam := &camera.Camera{
		ID:               "cam_entryway",
		Name:             "Ingresso Casa",
		AreaID:           nil,
		StreamURL:        "rtsp://192.168.1.100/ch0",
		SnapshotURL:      "/api/v1/cameras/cam_entryway/snapshot",
		StreamType:       camera.StreamTypeWebRTC,
		DetectionEnabled: true,
	}
	err := repos.Cameras.SaveCamera(ctx, cam)
	require.NoError(t, err)

	got, err := repos.Cameras.GetCamera(ctx, "cam_entryway")
	require.NoError(t, err)
	assert.Equal(t, "Ingresso Casa", got.Name)
	assert.True(t, got.DetectionEnabled)

	// Save AI detection event
	now := time.Now().UTC().Truncate(time.Second)
	evt := &camera.DetectionEvent{
		ID:           "evt_ai_001",
		CameraID:     cam.ID,
		Label:        "person",
		Score:        0.95,
		ThumbnailURL: "/snapshots/evt_001.jpg",
		DetectedAt:   now,
	}
	err = repos.Cameras.SaveEvent(ctx, evt)
	require.NoError(t, err)

	camEvents, err := repos.Cameras.ListEvents(ctx, cam.ID, 10)
	require.NoError(t, err)
	assert.Len(t, camEvents, 1)
	assert.Equal(t, "person", camEvents[0].Label)

	recentEvents, err := repos.Cameras.ListEvents(ctx, "", 10)
	require.NoError(t, err)
	assert.Len(t, recentEvents, 1)

	err = repos.Cameras.DeleteCamera(ctx, "cam_entryway")
	require.NoError(t, err)

	_, err = repos.Cameras.GetCamera(ctx, "cam_entryway")
	assert.Error(t, err)
}

func TestOTARepository_SQLite(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := &ota.DeviceOTA{
		ID:             "ota_esp32_relay",
		Name:           "ESP32 Giardino",
		DeviceKey:      "relay_garden",
		Platform:       "esphome",
		CurrentVersion: "1.0.0",
		LatestVersion:  "1.2.0",
		HasUpdate:      true,
		UpgradeState:   ota.StateIdle,
		Progress:       0,
		ReleaseNotes:   "Fix wifi reconnect loop",
		CheckedAt:      time.Now().UTC().Truncate(time.Second),
	}
	err := repos.OTA.SaveDevice(ctx, dev)
	require.NoError(t, err)

	got, err := repos.OTA.GetDevice(ctx, "ota_esp32_relay")
	require.NoError(t, err)
	assert.Equal(t, "ESP32 Giardino", got.Name)
	assert.Equal(t, "1.2.0", got.LatestVersion)
	assert.True(t, got.HasUpdate)

	list, err := repos.OTA.ListDevices(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// Update state during flashing
	dev.UpgradeState = ota.StateFlashing
	err = repos.OTA.SaveDevice(ctx, dev)
	require.NoError(t, err)

	updated, err := repos.OTA.GetDevice(ctx, "ota_esp32_relay")
	require.NoError(t, err)
	assert.Equal(t, ota.StateFlashing, updated.UpgradeState)
}

func TestStoreRepository_SQLite(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	pkg := &store.Package{
		ID:          "pkg_circadian",
		PackageID:   "community.lighting.circadian",
		Name:        "Circadian Rhythm Adaptive Lighting",
		Type:        store.TypeBlueprint,
		Version:     "2.1.0",
		Author:      "DomusCommunity",
		Description: "Adjusts light temperature dynamically based on sun angle",
		Installed:   false,
		DownloadURL: "https://plugins.domus.internal/circadian.tar.gz",
		Icon:        "sun",
	}
	err := repos.Store.SavePackage(ctx, pkg)
	require.NoError(t, err)

	got, err := repos.Store.GetPackage(ctx, "pkg_circadian")
	require.NoError(t, err)
	assert.Equal(t, "Circadian Rhythm Adaptive Lighting", got.Name)
	assert.Equal(t, store.TypeBlueprint, got.Type)
	assert.False(t, got.Installed)

	// List with filter
	blueprints, err := repos.Store.ListPackages(ctx, "blueprint")
	require.NoError(t, err)
	assert.Len(t, blueprints, 1)

	wasmPlugins, err := repos.Store.ListPackages(ctx, "wasm_plugin")
	require.NoError(t, err)
	assert.Empty(t, wasmPlugins)

	// Install package
	now := time.Now().UTC().Truncate(time.Second)
	pkg.Installed = true
	pkg.InstalledAt = &now
	err = repos.Store.SavePackage(ctx, pkg)
	require.NoError(t, err)

	installed, err := repos.Store.GetPackage(ctx, "pkg_circadian")
	require.NoError(t, err)
	assert.True(t, installed.Installed)
	assert.NotNil(t, installed.InstalledAt)

	// Delete
	err = repos.Store.DeletePackage(ctx, "pkg_circadian")
	require.NoError(t, err)

	all, err := repos.Store.ListPackages(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestStoreRepository_SeedDefaultPackages(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	// Initially empty
	pkgs, err := repos.Store.ListPackages(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, pkgs)

	// Seed
	err = repos.Store.SeedDefaultPackages(ctx)
	require.NoError(t, err)

	seeded, err := repos.Store.ListPackages(ctx, "")
	require.NoError(t, err)
	assert.Len(t, seeded, 5)

	// Idempotency check: seeding again should not duplicate
	err = repos.Store.SeedDefaultPackages(ctx)
	require.NoError(t, err)

	seededAgain, err := repos.Store.ListPackages(ctx, "")
	require.NoError(t, err)
	assert.Len(t, seededAgain, 5)
}
