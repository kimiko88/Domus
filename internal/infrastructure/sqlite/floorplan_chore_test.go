package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/chore"
	"github.com/domus-hub/domus/internal/domain/floorplan"
	"github.com/domus-hub/domus/internal/infrastructure/config"
)

func setupTestDB(t *testing.T) *Repositories {
	db, err := Open(config.DatabaseConfig{
		Path:          ":memory:",
		WALMode:       false,
		BusyTimeoutMs: 1000,
	})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	ctx := context.Background()
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	return NewRepositories(db)
}

func TestFloorplanRepository_CRUD(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	// 1. Create floor
	floor := &floorplan.Floor{
		ID:        "floor_ground",
		Name:      "Piano Terra",
		Level:     0,
		PlanSVG:   "<svg></svg>",
		SortOrder: 1,
	}
	if err := repos.Floors.CreateFloor(ctx, floor); err != nil {
		t.Fatalf("create floor failed: %v", err)
	}

	// 2. Get floor
	got, err := repos.Floors.GetFloor(ctx, "floor_ground")
	if err != nil {
		t.Fatalf("get floor failed: %v", err)
	}
	if got.Name != "Piano Terra" || got.Level != 0 {
		t.Errorf("unexpected floor data: %+v", got)
	}

	// 3. Save markers
	marker := &floorplan.Marker{
		ID:        "marker_1",
		FloorID:   floor.ID,
		EntityKey: "light.living_room",
		XPercent:  25.5,
		YPercent:  48.0,
		ShowBadge: true,
	}
	if err := repos.Floors.SaveMarker(ctx, marker); err != nil {
		t.Fatalf("save marker failed: %v", err)
	}

	markers, err := repos.Floors.ListMarkers(ctx, floor.ID)
	if err != nil {
		t.Fatalf("list markers failed: %v", err)
	}
	if len(markers) != 1 || markers[0].EntityKey != "light.living_room" {
		t.Errorf("unexpected markers list: %+v", markers)
	}

	// 4. Update marker position
	marker.XPercent = 30.0
	if err := repos.Floors.SaveMarker(ctx, marker); err != nil {
		t.Fatalf("update marker failed: %v", err)
	}
	markersUpdated, _ := repos.Floors.ListMarkers(ctx, floor.ID)
	if len(markersUpdated) != 1 || markersUpdated[0].XPercent != 30.0 {
		t.Errorf("expected updated x_percent 30.0, got %v", markersUpdated[0].XPercent)
	}
}

func TestChoreRepository_CRUD(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	dueDate := time.Now().UTC().Add(2 * time.Hour)
	task := &chore.Chore{
		ID:             "chore_robot",
		Title:          "Pulizia filtro robot",
		Description:    "Svuotare cassetto e pulire spazzole",
		Priority:       chore.PriorityHigh,
		DueDate:        &dueDate,
		RecurrenceType: chore.RecurrenceWeekly,
		Status:         chore.StatusPending,
	}

	if err := repos.Chores.Create(ctx, task); err != nil {
		t.Fatalf("create chore failed: %v", err)
	}

	got, err := repos.Chores.GetByID(ctx, "chore_robot")
	if err != nil {
		t.Fatalf("get chore failed: %v", err)
	}
	if got.Title != "Pulizia filtro robot" || got.Priority != chore.PriorityHigh {
		t.Errorf("unexpected chore: %+v", got)
	}

	// Filter by status
	list, err := repos.Chores.List(ctx, chore.Filter{Status: chore.StatusPending})
	if err != nil {
		t.Fatalf("list chores failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 pending chore, got %d", len(list))
	}

	// Check due before
	dueList, err := repos.Chores.GetPendingDueBefore(ctx, time.Now().UTC().Add(3*time.Hour))
	if err != nil {
		t.Fatalf("get pending due before failed: %v", err)
	}
	if len(dueList) != 1 {
		t.Errorf("expected 1 chore due before 3 hours, got %d", len(dueList))
	}
}
