package pantry

import "context"

// ReceiptHistoryRepository stores confirmed purchase records independently
// from live pantry quantities. Editing historic receipts never replays stock.
type ReceiptHistoryRepository interface {
	CreateReceipt(ctx context.Context, receipt *Receipt) error
	GetReceipt(ctx context.Context, id string) (*Receipt, error)
	UpdateReceipt(ctx context.Context, receipt *Receipt) error
	ListReceipts(ctx context.Context, limit int) ([]Receipt, error)
}