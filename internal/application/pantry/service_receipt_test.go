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
	err     error
	calls   int
	expense *domainFinance.Expense
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
	f.expense = &domainFinance.Expense{ID: "receipt-expense-test", Title: title, Amount: amount, Category: category, Subcategory: subcategory, ExpenseDate: date, IsFixed: isFixed, Notes: notes}
	copy := *f.expense
	return &copy, nil
}

func (f *receiptFinanceRecorder) GetExpenseByID(ctx context.Context, id string) (*domainFinance.Expense, error) {
	if f.expense == nil || f.expense.ID != id { return nil, errors.New("expense not found") }
	copy := *f.expense
	return &copy, nil
}

func (f *receiptFinanceRecorder) UpdateExpense(ctx context.Context, expense *domainFinance.Expense) error {
	if f.expense == nil || expense.ID != f.expense.ID { return errors.New("expense not found") }
	copy := *expense
	f.expense = &copy
	return nil
}

func (f *receiptFinanceRecorder) DeleteExpense(ctx context.Context, id string) error {
	if f.expense == nil || f.expense.ID != id { return errors.New("expense not found") }
	f.expense = nil
	return nil
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

func TestImportReceiptAlwaysStoresPurchaseHistory(t *testing.T) {
	service, repo := newReceiptImportService(t, nil)
	result, err := service.ImportReceipt(context.Background(), appPantry.ImportReceiptRequest{
		StoreName: "Market Rossi", Date: "2026-11-26", TotalAmount: 3.25, Confidence: 0.8,
		AutoAddToFinance: false, AutoUpdatePantry: false,
		Items: []domainPantry.ReceiptItem{{Name: "Pasta", Quantity: 1, Unit: domainPantry.UnitPiece, UnitPrice: 3.25, TotalPrice: 3.25, Category: domainPantry.CategoryPantry}},
	})
	require.NoError(t, err)
	require.NotNil(t, result.ReceiptID)
	receipt, err := repo.GetReceipt(context.Background(), *result.ReceiptID)
	require.NoError(t, err)
	require.Equal(t, "Market Rossi", receipt.StoreName)
	require.Equal(t, 0.8, receipt.Confidence)
	require.False(t, receipt.AutoAddToFinance)
	require.False(t, receipt.AutoUpdatePantry)
	require.Len(t, receipt.Items, 1)
}

func TestUpdateReceiptUpdatesLinkedFinanceButDoesNotReplayPantryStock(t *testing.T) {
	finance := &receiptFinanceRecorder{}
	service, repo := newReceiptImportService(t, finance)
	ctx := context.Background()
	result, err := service.ImportReceipt(ctx, appPantry.ImportReceiptRequest{
		StoreName: "Esselunga", Date: "2026-11-26", TotalAmount: 10,
		AutoAddToFinance: true, AutoUpdatePantry: true,
		Items: []domainPantry.ReceiptItem{{Name: "Pasta", Quantity: 500, Unit: domainPantry.UnitGram, UnitPrice: 0.02, TotalPrice: 10, Category: domainPantry.CategoryPantry}},
	})
	require.NoError(t, err)
	require.NotNil(t, result.ReceiptID)
	stockBefore, err := repo.GetPantryItemByName(ctx, "Pasta")
	require.NoError(t, err)
	require.Equal(t, 500.0, stockBefore.Quantity)
	receipt, err := service.UpdateReceipt(ctx, *result.ReceiptID, appPantry.UpdateReceiptRequest{
		DocumentType: "receipt", StoreName: "Coop", Date: "2026-11-27", TotalAmount: 12,
		Items: []domainPantry.ReceiptItem{{Name: "Pasta integrale", Quantity: 250, Unit: domainPantry.UnitGram, UnitPrice: 0.048, TotalPrice: 12, Category: domainPantry.CategoryPantry}},
	})
	require.NoError(t, err)
	require.Equal(t, "Coop", receipt.StoreName)
	require.Equal(t, 12.0, receipt.TotalAmount)
	require.Equal(t, "Pasta integrale", receipt.Items[0].Name)
	require.NotNil(t, finance.expense)
	require.Equal(t, "Spesa Coop", finance.expense.Title)
	require.Equal(t, 12.0, finance.expense.Amount)
	require.Equal(t, time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC), finance.expense.ExpenseDate)
	stockAfter, err := repo.GetPantryItemByName(ctx, "Pasta")
	require.NoError(t, err)
	require.Equal(t, 500.0, stockAfter.Quantity, "editing historical receipt must not change current inventory")
}

func TestUpdateReceiptRejectsInvalidStoreAndDate(t *testing.T) {
	service, repo := newReceiptImportService(t, nil)
	result, err := service.ImportReceipt(context.Background(), appPantry.ImportReceiptRequest{StoreName: "Coop", Date: "2026-11-26", TotalAmount: 1})
	require.NoError(t, err)
	require.NotNil(t, result.ReceiptID)
	_, err = service.UpdateReceipt(context.Background(), *result.ReceiptID, appPantry.UpdateReceiptRequest{StoreName: "  ", Date: "not-a-date", TotalAmount: 1})
	require.ErrorIs(t, err, appPantry.ErrInvalidReceiptImport)
	stored, err := repo.GetReceipt(context.Background(), *result.ReceiptID)
	require.NoError(t, err)
	require.Equal(t, "Coop", stored.StoreName, "invalid edit must leave history unchanged")
}