package pantry

import (
	"errors"
	"time"
)

var ErrReceiptNotFound = errors.New("pantry: receipt not found")

// ReceiptItem represents a single product line extracted from a receipt.
type ReceiptItem struct {
	RawLine        string  `json:"raw_line"`
	Name           string  `json:"name"`
	Quantity       float64 `json:"quantity"`
	Unit           Unit    `json:"unit"`
	UnitPrice      float64 `json:"unit_price"`
	TotalPrice     float64 `json:"total_price"`
	Category       Category `json:"category"`
	MatchedPantryID *string `json:"matched_pantry_id,omitempty"`
}

// Receipt represents a parsed receipt and, once imported, a persistent purchase record.
// RawText is returned by OCR for review but is deliberately never persisted.
type Receipt struct {
	ID               string        `json:"id,omitempty"`
	StoreName        string        `json:"store_name"`
	Date             string        `json:"date"` // YYYY-MM-DD
	DocumentType     string        `json:"document_type,omitempty"`
	Items            []ReceiptItem `json:"items"`
	TotalAmount      float64       `json:"total_amount"`
	RawText          string        `json:"raw_text,omitempty"`
	Confidence       float64       `json:"confidence"`
	ExpenseID        *string       `json:"expense_id,omitempty"`
	AutoAddToFinance bool          `json:"auto_add_to_finance,omitempty"`
	AutoUpdatePantry bool          `json:"auto_update_pantry,omitempty"`
	CreatedAt        time.Time     `json:"created_at,omitempty"`
	UpdatedAt        time.Time     `json:"updated_at,omitempty"`
}

// ReceiptImportResult summarizes the outcome of confirming a receipt import.
type ReceiptImportResult struct {
	ReceiptID     *string `json:"receipt_id,omitempty"`
	ItemsAdded    int     `json:"items_added"`
	ItemsUpdated  int     `json:"items_updated"`
	ExpenseID     *string `json:"expense_id,omitempty"`
	ExpenseAmount float64 `json:"expense_amount"`
}