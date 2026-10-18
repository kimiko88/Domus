package sqlite_test

import (
	"context"
	"testing"

	domain "github.com/domus-hub/domus/internal/domain/medicine"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
)

func TestMedicineRepository_CRUDAndFiltering(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path:    ":memory:",
		WALMode: false,
	})
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer db.Close()

	if err := sqlite.Migrate(ctx, db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	repo := sqlite.NewMedicineRepository(db)

	// 1. Create medicine
	item := &domain.Medicine{
		ID:                      "med-test-1",
		Name:                    "Tachipirina 1000 mg",
		ActiveIngredient:        "Paracetamolo",
		Dosage:                  "1000 mg",
		Form:                    "compresse",
		AIC:                     "034234035",
		Barcode:                 "08003423403501",
		LotNumber:               "LOT123",
		ExpirationDate:          "2027-10-31",
		EffectiveExpirationDate: "2027-10-31",
		Location:                domain.LocationBathroomCabinet,
		Quantity:                16,
		Unit:                    domain.UnitTablets,
		MinThreshold:            4,
	}

	if err := repo.Create(ctx, item); err != nil {
		t.Fatalf("failed to create medicine: %v", err)
	}

	// 2. GetByID
	fetched, err := repo.GetByID(ctx, "med-test-1")
	if err != nil {
		t.Fatalf("failed to get medicine by id: %v", err)
	}
	if fetched.Name != item.Name || fetched.AIC != item.AIC {
		t.Errorf("expected %s and AIC %s, got %s and %s", item.Name, item.AIC, fetched.Name, fetched.AIC)
	}

	// 3. Update
	opened := "2026-10-18"
	fetched.OpenedAt = &opened
	fetched.Quantity = 14
	if err := repo.Update(ctx, fetched); err != nil {
		t.Fatalf("failed to update medicine: %v", err)
	}

	updated, err := repo.GetByID(ctx, "med-test-1")
	if err != nil {
		t.Fatalf("failed to get updated: %v", err)
	}
	if updated.Quantity != 14 || updated.OpenedAt == nil || *updated.OpenedAt != opened {
		t.Errorf("update was not saved properly: %+v", updated)
	}

	// 4. List with filter
	list, err := repo.List(ctx, domain.Filter{Location: domain.LocationBathroomCabinet})
	if err != nil {
		t.Fatalf("failed to list medicines: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 medicine in bathroom cabinet, got %d", len(list))
	}

	// 5. Delete
	if err := repo.Delete(ctx, "med-test-1"); err != nil {
		t.Fatalf("failed to delete medicine: %v", err)
	}

	_, err = repo.GetByID(ctx, "med-test-1")
	if err == nil {
		t.Fatalf("expected error getting deleted medicine, got nil")
	}
}
