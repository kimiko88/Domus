package pantry

// ReceiptItem represents a single extracted item from a supermarket receipt.
type ReceiptItem struct {
	RawLine         string   `json:"raw_line"`
	Name            string   `json:"name"`
	Quantity        float64  `json:"quantity"`
	Unit            Unit     `json:"unit"`
	UnitPrice       float64  `json:"unit_price"`
	TotalPrice      float64  `json:"total_price"`
	Category        Category `json:"category"`
	MatchedPantryID *string  `json:"matched_pantry_id,omitempty"`
}

// Receipt represents the parsed structure of a supermarket receipt.
type Receipt struct {
	StoreName   string        `json:"store_name"`
	Date        string        `json:"date"` // YYYY-MM-DD
	Items       []ReceiptItem `json:"items"`
	TotalAmount float64       `json:"total_amount"`
	RawText     string        `json:"raw_text,omitempty"`
	Confidence  float64       `json:"confidence"` // 0.0 to 1.0 score
}

// ReceiptImportResult summarizes the outcome of confirming a receipt import.
type ReceiptImportResult struct {
	ItemsAdded    int     `json:"items_added"`
	ItemsUpdated  int     `json:"items_updated"`
	ExpenseID     *string `json:"expense_id,omitempty"`
	ExpenseAmount float64 `json:"expense_amount"`
}
