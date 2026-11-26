// Package sqlite implements receipt history persistence using the shared pantry database.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	domain "github.com/domus-hub/domus/internal/domain/pantry"
	"github.com/google/uuid"
)

// CreateReceipt stores only reviewed, structured purchase data. OCR source text is excluded.
func (r *PantryRepository) CreateReceipt(ctx context.Context, receipt *domain.Receipt) error {
	if receipt == nil {
		return fmt.Errorf("receipt history: nil receipt")
	}
	if receipt.ID == "" {
		receipt.ID = "receipt_" + uuid.NewString()
	}
	if receipt.DocumentType == "" {
		receipt.DocumentType = "receipt"
	}
	if receipt.Items == nil {
		receipt.Items = []domain.ReceiptItem{}
	}
	itemsJSON, err := json.Marshal(receipt.Items)
	if err != nil {
		return fmt.Errorf("receipt history: encode items: %w", err)
	}
	now := time.Now().UTC()
	if receipt.CreatedAt.IsZero() {
		receipt.CreatedAt = now
	}
	receipt.UpdatedAt = now
	_, err = r.db.ExecContext(ctx,
		"INSERT INTO receipt_history (id, document_type, store_name, purchase_date, items_json, total_amount, confidence, expense_id, auto_add_to_finance, auto_update_pantry, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		receipt.ID, receipt.DocumentType, receipt.StoreName, receipt.Date, string(itemsJSON),
		receipt.TotalAmount, receipt.Confidence, receipt.ExpenseID, boolInt(receipt.AutoAddToFinance), boolInt(receipt.AutoUpdatePantry),
		receipt.CreatedAt.UTC().Format(time.RFC3339), receipt.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("receipt history: insert: %w", err)
	}
	return nil
}

// GetReceipt returns a single saved purchase.
func (r *PantryRepository) GetReceipt(ctx context.Context, id string) (*domain.Receipt, error) {
	return scanReceipt(r.db.QueryRowContext(ctx, receiptSelect+" WHERE id = ?", id))
}

// UpdateReceipt modifies historical purchase details without altering pantry stock.
func (r *PantryRepository) UpdateReceipt(ctx context.Context, receipt *domain.Receipt) error {
	if receipt == nil || receipt.ID == "" {
		return domain.ErrReceiptNotFound
	}
	itemsJSON, err := json.Marshal(receipt.Items)
	if err != nil {
		return fmt.Errorf("receipt history: encode items: %w", err)
	}
	receipt.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(ctx,
		"UPDATE receipt_history SET document_type = ?, store_name = ?, purchase_date = ?, items_json = ?, total_amount = ?, confidence = ?, expense_id = ?, auto_add_to_finance = ?, auto_update_pantry = ?, updated_at = ? WHERE id = ?",
		receipt.DocumentType, receipt.StoreName, receipt.Date, string(itemsJSON), receipt.TotalAmount, receipt.Confidence,
		receipt.ExpenseID, boolInt(receipt.AutoAddToFinance), boolInt(receipt.AutoUpdatePantry), receipt.UpdatedAt.UTC().Format(time.RFC3339), receipt.ID,
	)
	if err != nil {
		return fmt.Errorf("receipt history: update: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrReceiptNotFound
	}
	return nil
}

// ListReceipts returns newest purchases first; the query limit is bounded.
func (r *PantryRepository) ListReceipts(ctx context.Context, limit int) ([]domain.Receipt, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, receiptSelect+" ORDER BY purchase_date DESC, created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.Receipt, 0)
	for rows.Next() {
		receipt, scanErr := scanReceipt(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *receipt)
	}
	return result, rows.Err()
}

const receiptSelect = "SELECT id, document_type, store_name, purchase_date, items_json, total_amount, confidence, expense_id, auto_add_to_finance, auto_update_pantry, created_at, updated_at FROM receipt_history"

func scanReceipt(s scannable) (*domain.Receipt, error) {
	var receipt domain.Receipt
	var itemsJSON, createdAt, updatedAt string
	var autoFinance, autoPantry int
	if err := s.Scan(&receipt.ID, &receipt.DocumentType, &receipt.StoreName, &receipt.Date, &itemsJSON, &receipt.TotalAmount, &receipt.Confidence, &receipt.ExpenseID, &autoFinance, &autoPantry, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrReceiptNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal([]byte(itemsJSON), &receipt.Items); err != nil {
		return nil, fmt.Errorf("receipt history: decode items for %s: %w", receipt.ID, err)
	}
	if receipt.Items == nil {
		receipt.Items = []domain.ReceiptItem{}
	}
	receipt.AutoAddToFinance = autoFinance != 0
	receipt.AutoUpdatePantry = autoPantry != 0
	receipt.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	receipt.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &receipt, nil
}

