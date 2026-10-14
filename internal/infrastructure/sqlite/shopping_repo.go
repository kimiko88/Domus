package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	domainShop "github.com/domus-hub/domus/internal/domain/shopping"
	"github.com/google/uuid"
)

// ShoppingRepository implements shopping.Repository for SQLite.
type ShoppingRepository struct {
	db *sql.DB
}

// NewShoppingRepository creates a new SQLite-backed ShoppingRepository.
func NewShoppingRepository(db *sql.DB) *ShoppingRepository {
	return &ShoppingRepository{db: db}
}

// SaveItem inserts or updates a shopping list item.
func (r *ShoppingRepository) SaveItem(ctx context.Context, item *domainShop.Item) error {
	if item.ID == "" {
		item.ID = "shop_" + uuid.NewString()[:8]
	}
	now := time.Now().UTC()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}

	var completedAtStr *string
	if item.CompletedAt != nil {
		s := item.CompletedAt.Format(time.RFC3339)
		completedAtStr = &s
	}

	var completedInt int
	if item.Completed {
		completedInt = 1
	}

	query := `
		INSERT INTO shopping_items (id, name, quantity, category, completed, added_by, completed_by, created_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			quantity = excluded.quantity,
			category = excluded.category,
			completed = excluded.completed,
			added_by = excluded.added_by,
			completed_by = excluded.completed_by,
			completed_at = excluded.completed_at
	`
	_, err := r.db.ExecContext(
		ctx, query,
		item.ID,
		item.Name,
		item.Quantity,
		string(item.Category),
		completedInt,
		item.AddedBy,
		item.CompletedBy,
		item.CreatedAt.Format(time.RFC3339),
		completedAtStr,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to save shopping item: %w", err)
	}
	return nil
}

// GetItem retrieves a shopping item by ID.
func (r *ShoppingRepository) GetItem(ctx context.Context, id string) (*domainShop.Item, error) {
	query := `
		SELECT id, name, quantity, category, completed, added_by, completed_by, created_at, completed_at
		FROM shopping_items
		WHERE id = ?
	`
	var (
		item           domainShop.Item
		categoryStr    string
		completedInt   int
		createdAtStr   string
		completedAtStr sql.NullString
		completedBy    sql.NullString
		quantity       sql.NullString
	)

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&item.ID,
		&item.Name,
		&quantity,
		&categoryStr,
		&completedInt,
		&item.AddedBy,
		&completedBy,
		&createdAtStr,
		&completedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("shopping item not found: %s", id)
		}
		return nil, fmt.Errorf("sqlite: failed to get shopping item: %w", err)
	}

	item.Category = domainShop.Category(categoryStr)
	item.Completed = completedInt == 1
	item.Quantity = quantity.String
	if completedBy.Valid {
		item.CompletedBy = &completedBy.String
	}
	if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		item.CreatedAt = t
	}
	if completedAtStr.Valid {
		if t, err := time.Parse(time.RFC3339, completedAtStr.String); err == nil {
			item.CompletedAt = &t
		}
	}

	return &item, nil
}

// ListItems returns all shopping list items.
func (r *ShoppingRepository) ListItems(ctx context.Context) ([]*domainShop.Item, error) {
	query := `
		SELECT id, name, quantity, category, completed, added_by, completed_by, created_at, completed_at
		FROM shopping_items
		ORDER BY completed ASC, created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list shopping items: %w", err)
	}
	defer rows.Close()

	var items []*domainShop.Item
	for rows.Next() {
		var (
			item           domainShop.Item
			categoryStr    string
			completedInt   int
			createdAtStr   string
			completedAtStr sql.NullString
			completedBy    sql.NullString
			quantity       sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&quantity,
			&categoryStr,
			&completedInt,
			&item.AddedBy,
			&completedBy,
			&createdAtStr,
			&completedAtStr,
		); err != nil {
			return nil, fmt.Errorf("sqlite: scan error in shopping list: %w", err)
		}
		item.Category = domainShop.Category(categoryStr)
		item.Completed = completedInt == 1
		item.Quantity = quantity.String
		if completedBy.Valid {
			item.CompletedBy = &completedBy.String
		}
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			item.CreatedAt = t
		}
		if completedAtStr.Valid {
			if t, err := time.Parse(time.RFC3339, completedAtStr.String); err == nil {
				item.CompletedAt = &t
			}
		}
		items = append(items, &item)
	}

	if items == nil {
		items = []*domainShop.Item{}
	}
	return items, nil
}

// DeleteItem removes an item by ID.
func (r *ShoppingRepository) DeleteItem(ctx context.Context, id string) error {
	query := `DELETE FROM shopping_items WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete shopping item: %w", err)
	}
	return nil
}

// ClearCompleted removes all completed items and returns the number of deleted records.
func (r *ShoppingRepository) ClearCompleted(ctx context.Context) (int, error) {
	query := `DELETE FROM shopping_items WHERE completed = 1`
	res, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("sqlite: failed to clear completed shopping items: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
