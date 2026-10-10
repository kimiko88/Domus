package dashboard_test

import (
	"context"
	"database/sql"
	"testing"

	appDash "github.com/domus-hub/domus/internal/application/dashboard"
	domainDash "github.com/domus-hub/domus/internal/domain/dashboard"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}

	createTable := `
	CREATE TABLE custom_dashboards (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		icon TEXT NOT NULL DEFAULT 'layout-dashboard',
		is_default BOOLEAN NOT NULL DEFAULT FALSE,
		sort_order INTEGER NOT NULL DEFAULT 0,
		layout_json TEXT NOT NULL DEFAULT '[]',
		kiosk_config_json TEXT NOT NULL DEFAULT '{"enabled": false, "screensaver_timeout_sec": 120, "wake_on_presence": true}',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(createTable); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	return db, func() { db.Close() }
}

func TestDashboardService_CRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewDashboardRepository(db)
	svc := appDash.NewService(repo)

	ctx := context.Background()

	// 1. Create Dashboard
	dash := &domainDash.Dashboard{
		ID:        "tablet_living_room",
		Title:     "Tablet Salotto",
		Icon:      "tablet",
		SortOrder: 1,
		LayoutJSON: `[
			{"id": "w1", "type": "thermostat", "title": "Clima Salotto", "entity_key": "climate.living_room", "col_span": 2, "row_span": 2},
			{"id": "w2", "type": "sankey", "title": "Flusso Solare", "col_span": 4, "row_span": 2}
		]`,
	}
	dash.SetKioskConfig(domainDash.KioskConfig{
		Enabled:               true,
		ScreensaverTimeoutSec: 60,
		WakeOnPresence:        true,
		ShowClock:             true,
	})

	if err := svc.Create(ctx, dash); err != nil {
		t.Fatalf("failed to create dashboard: %v", err)
	}

	// 2. Get By ID
	fetched, err := svc.GetByID(ctx, "tablet_living_room")
	if err != nil {
		t.Fatalf("failed to get dashboard: %v", err)
	}
	if fetched.Title != "Tablet Salotto" {
		t.Errorf("expected title 'Tablet Salotto', got %s", fetched.Title)
	}
	kCfg := fetched.GetKioskConfig()
	if !kCfg.Enabled || kCfg.ScreensaverTimeoutSec != 60 {
		t.Errorf("unexpected kiosk config: %+v", kCfg)
	}

	// 3. Set Default
	if err := svc.SetDefault(ctx, "tablet_living_room"); err != nil {
		t.Fatalf("failed to set default: %v", err)
	}
	fetched, _ = svc.GetByID(ctx, "tablet_living_room")
	if !fetched.IsDefault {
		t.Errorf("expected is_default=true")
	}

	// 4. List
	all, err := svc.List(ctx)
	if err != nil || len(all) != 1 {
		t.Fatalf("expected 1 dashboard in list, got %d", len(all))
	}

	// 5. Delete
	if err := svc.Delete(ctx, "tablet_living_room"); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}
	_, err = svc.GetByID(ctx, "tablet_living_room")
	if err != domainDash.ErrDashboardNotFound {
		t.Errorf("expected ErrDashboardNotFound, got %v", err)
	}
}
