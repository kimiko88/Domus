package pantry_test

import (
	"context"
	"errors"
	"testing"

	domainFinance "github.com/domus-hub/domus/internal/domain/finance"
	domainPantry "github.com/domus-hub/domus/internal/domain/pantry"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	appPantry "github.com/domus-hub/domus/internal/application/pantry"
	"github.com/stretchr/testify/require"
	"time"
)

type receiptFinanceRecorder struct {
	err   error
	calls int
}

func (f *receiptFinanceRecorder) RecordExpense(
	ctx context.Context,
	title string,
	amount float64,
	category domainFinance.Category,
	subcategory domainFinance.Subcategory,
	date time.Time,
	isFixed bool,
	notes string,
) (*domainFinance.Expense, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &domainFinance.Expense{ID: "receipt-expense-test"}, nil
}

func newReceiptImportService(t *testing.T, finance appPantry.FinanceRecorder) (*appPantry.Service, *sqlite.PantryRepository) {
	t.Helper()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.Migrate(context.Background(), db))

	repo := sqlite.NewPantryRepository(db)
	service := appPantry.NewService(repo, nil, finance, nil, nil)
	return service, repo
}

func TestImportReceiptPrevalidatesAllUnitsBeforeChangingPantry(t *testing.T) {
	service, repo := newReceiptImportService(t, nil)
	ctx := context.Background()

	_, err := service.CreatePantryItem(ctx, appPantry.CreatePantryItemRequest{
		Name: "Latte", Category: domainPantry.CategoryDairy,
		Quantity: 2, Unit: domainPantry.UnitPiece,
	})
	require.NoError(t, err)

	_, err = service.ImportReceipt(ctx, appPantry.ImportReceiptRequest{
		StoreName: "Coop", Date: "2026-11-26", TotalAmount: 3.50,
		AutoUpdatePantry: true,
		Items: []domainPantry.ReceiptItem{
			{Name: "Pasta", Quantity: 500, Unit: domainPantry.UnitGram, TotalPrice: 1.50, UnitPrice: 0.003, Category: domainPantry.CategoryPantry},
			{Name: "Latte", Quantity: 1, Unit: domainPantry.UnitLiter, TotalPrice: 2.00, UnitPrice: 2.00, Category: domainPantry.CategoryDairy},
		},
	})
	require.ErrorIs(t, err, appPantry.ErrReceiptInventoryConflict)

	_, err = repo.GetPantryItemByName(ctx, "Pasta")
	require.ErrorIs(t, err, domainPantry.ErrItemNotFound)
	latte, err := repo.GetPantryItemByName(ctx, "Latte")
	require.NoError(t, err)
	require.Equal(t, 2.0, latte.Quantity, "preflight conflict must not partially update existing inventory")
}

func TestImportReceiptRollsBackPantryWhenFinanceRecordingFails(t *testing.T) {
	finance := &receiptFinanceRecorder{err: errors.New("finance database unavailable")}
	service, repo := newReceiptImportService(t, finance)
	ctx := context.Background()

	_, err := service.CreatePantryItem(ctx, appPantry.CreatePantryItemRequest{
		Name: "Pasta", Category: domainPantry.CategoryPantry,
		Quantity: 500, Unit: domainPantry.UnitGram, Cost: 1.20,
	})
	require.NoError(t, err)

	_, err = service.ImportReceipt(ctx, appPantry.ImportReceiptRequest{
		StoreName: "Esselunga", Date: "2026-11-26", TotalAmount: 4.00,
		AutoAddToFinance: true, AutoUpdatePantry: true,
		Items: []domainPantry.ReceiptItem{
			{Name: "Pasta", Quantity: 500, Unit: domainPantry.UnitGram, TotalPrice: 1.40, UnitPrice: 0.0028, Category: domainPantry.CategoryPantry},
			{Name: "Riso", Quantity: 1, Unit: domainPantry.UnitKilogram, TotalPrice: 2.60, UnitPrice: 2.60, Category: domainPantry.CategoryPantry},
		},
	})
	require.Error(t, err)
	require.Equal(t, 1, finance.calls)

	pasta, err := repo.GetPantryItemByName(ctx, "Pasta")
	require.NoError(t, err)
	require.Equal(t, 500.0, pasta.Quantity, "existing stock must be restored after finance failure")
	require.InDelta(t, 1.20, pasta.Cost, 0.0001)

	_, err = repo.GetPantryItemByName(ctx, "Riso")
	require.ErrorIs(t, err, domainPantry.ErrItemNotFound, "newly created stock must be removed after finance failure")
}

func TestImportReceiptMergesDuplicateLinesBeforeWritingPantry(t *testing.T) {
	service, repo := newReceiptImportService(t, nil)
	ctx := context.Background()

	result, err := service.ImportReceipt(ctx, appPantry.ImportReceiptRequest{
		StoreName: "Conad", Date: "2026-11-26", TotalAmount: 2.00,
		AutoUpdatePantry: true,
		Items: []domainPantry.ReceiptItem{
			{Name: "Farina", Quantity: 500, Unit: domainPantry.UnitGram, TotalPrice: 1.00, UnitPrice: 0.002, Category: domainPantry.CategoryPantry},
			{Name: "farina", Quantity: 500, Unit: domainPantry.UnitGram, TotalPrice: 1.00, UnitPrice: 0.002, Category: domainPantry.CategoryPantry},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.ItemsAdded)
	require.Equal(t, 0, result.ItemsUpdated)

	item, err := repo.GetPantryItemByName(ctx, "Farina")
	require.NoError(t, err)
	require.Equal(t, 1000.0, item.Quantity)
	require.InDelta(t, 0.002, item.Cost, 0.0001)
}

func TestImportReceiptRejectsInvalidUnitsBeforeWriting(t *testing.T) {
	service, repo := newReceiptImportService(t, nil)
	_, err := service.ImportReceipt(context.Background(), appPantry.ImportReceiptRequest{
		StoreName: "Lidl", TotalAmount: 1.00, AutoUpdatePantry: true,
		Items: []domainPantry.ReceiptItem{
			{Name: "Pasta", Quantity: 1, Unit: domainPantry.Unit("cases"), TotalPrice: 1, Category: domainPantry.CategoryPantry},
		},
	})
	require.ErrorIs(t, err, appPantry.ErrInvalidReceiptImport)
	_, err = repo.GetPantryItemByName(context.Background(), "Pasta")
	require.ErrorIs(t, err, domainPantry.ErrItemNotFound)
}
