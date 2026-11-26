package sqlite

import (
	"context"
	"testing"

	domain "github.com/domus-hub/domus/internal/domain/pantry"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/stretchr/testify/require"
)

func TestReceiptHistoryRepositoryCreateListAndUpdate(t *testing.T) {
	db, err := Open(config.DatabaseConfig{Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, Migrate(context.Background(), db))
	repo := NewPantryRepository(db)
	ctx := context.Background()
	receipt := &domain.Receipt{
		ID: "receipt-test", StoreName: "Market Rossi", Date: "2026-11-26", DocumentType: "receipt",
		Items: []domain.ReceiptItem{{Name: "Pasta", Quantity: 500, Unit: domain.UnitGram, UnitPrice: 0.002, TotalPrice: 1, Category: domain.CategoryPantry}},
		TotalAmount: 1, Confidence: 0.75, AutoAddToFinance: true, RawText: "FISCAL RECEIPT PRIVATE SOURCE TEXT",
	}
	require.NoError(t, repo.CreateReceipt(ctx, receipt))
	loaded, err := repo.GetReceipt(ctx, receipt.ID)
	require.NoError(t, err)
	require.Equal(t, "Market Rossi", loaded.StoreName)
	require.Equal(t, "2026-11-26", loaded.Date)
	require.Equal(t, 0.75, loaded.Confidence)
	require.True(t, loaded.AutoAddToFinance)
	require.Empty(t, loaded.RawText, "raw OCR text must not be persisted")
	require.Len(t, loaded.Items, 1)
	loaded.StoreName = "Coop"
	loaded.Date = "2026-11-27"
	loaded.TotalAmount = 2.5
	loaded.Items[0].Name = "Pasta integrale"
	require.NoError(t, repo.UpdateReceipt(ctx, loaded))
	updated, err := repo.GetReceipt(ctx, receipt.ID)
	require.NoError(t, err)
	require.Equal(t, "Coop", updated.StoreName)
	require.Equal(t, "2026-11-27", updated.Date)
	require.Equal(t, 2.5, updated.TotalAmount)
	require.Equal(t, "Pasta integrale", updated.Items[0].Name)
	require.True(t, !updated.UpdatedAt.Before(updated.CreatedAt))
	rows, err := repo.ListReceipts(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, receipt.ID, rows[0].ID)
	_, err = repo.GetReceipt(ctx, "missing")
	require.ErrorIs(t, err, domain.ErrReceiptNotFound)
}

func TestReceiptHistoryListClampsInvalidLimit(t *testing.T) {
	db, err := Open(config.DatabaseConfig{Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, Migrate(context.Background(), db))
	rows, err := NewPantryRepository(db).ListReceipts(context.Background(), 0)
	require.NoError(t, err)
	require.Empty(t, rows)
}