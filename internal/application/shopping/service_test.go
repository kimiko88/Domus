package shopping_test

import (
	"context"
	"testing"

	"github.com/domus-hub/domus/internal/application/shopping"
	domainShop "github.com/domus-hub/domus/internal/domain/shopping"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShoppingService_CrudAndToggle(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	repo := shopping.NewMemoryRepository()
	svc := shopping.NewService(repo, bus)
	ctx := context.Background()

	// Add item 1
	item1, err := svc.AddItem(ctx, "Latte Parzialmente Scremato", "2 bottiglie", domainShop.CategoryBeverage, "Marco")
	require.NoError(t, err)
	assert.NotEmpty(t, item1.ID)
	assert.False(t, item1.Completed)
	assert.Equal(t, "Marco", item1.AddedBy)

	// Add item 2
	item2, err := svc.AddItem(ctx, "Pomodori San Marzano", "1 kg", domainShop.CategoryProduce, "Chiara")
	require.NoError(t, err)
	assert.NotEmpty(t, item2.ID)

	// List items
	items, err := svc.ListItems(ctx)
	require.NoError(t, err)
	assert.Len(t, items, 2)

	// Toggle item 1 (completed by Chiara)
	toggled, err := svc.ToggleItem(ctx, item1.ID, "Chiara")
	require.NoError(t, err)
	assert.True(t, toggled.Completed)
	require.NotNil(t, toggled.CompletedBy)
	assert.Equal(t, "Chiara", *toggled.CompletedBy)
	assert.NotNil(t, toggled.CompletedAt)

	// Clear completed
	cleared, err := svc.ClearCompleted(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, cleared)

	// List remaining
	remaining, err := svc.ListItems(ctx)
	require.NoError(t, err)
	assert.Len(t, remaining, 1)
	assert.Equal(t, item2.ID, remaining[0].ID)
}
