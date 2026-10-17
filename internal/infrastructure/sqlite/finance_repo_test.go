package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/finance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinanceRepository_SQLite(t *testing.T) {
	repos := setupTestDB(t)
	repo := repos.Finance
	require.NotNil(t, repo)

	ctx := context.Background()
	refDate := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

	// 1. Create expenses
	exp1 := &finance.Expense{
		ID:          "exp_test_1",
		Title:       "Spesa Supermercato",
		Amount:      120.50,
		Category:    finance.CategoryFood,
		Subcategory: finance.SubcategoryGroceries,
		ExpenseDate: refDate,
		IsFixed:     false,
	}
	require.NoError(t, repo.Create(ctx, exp1))

	exp2 := &finance.Expense{
		ID:          "exp_test_2",
		Title:       "Bolletta Luce",
		Amount:      85.00,
		Category:    finance.CategoryUtilities,
		Subcategory: finance.SubcategoryElectricity,
		ExpenseDate: refDate.AddDate(0, 0, -2),
		IsFixed:     true,
		EnergyKWh:   280.0,
	}
	require.NoError(t, repo.Create(ctx, exp2))

	exp3 := &finance.Expense{
		ID:          "exp_test_3",
		Title:       "Carburante Eni",
		Amount:      60.00,
		Category:    finance.CategoryTransport,
		Subcategory: finance.SubcategoryFuelDiesel,
		ExpenseDate: refDate.AddDate(0, 0, -5),
		IsFixed:     false,
	}
	require.NoError(t, repo.Create(ctx, exp3))

	// 2. GetByID
	got, err := repo.GetByID(ctx, "exp_test_1")
	require.NoError(t, err)
	assert.Equal(t, "Spesa Supermercato", got.Title)
	assert.Equal(t, 120.50, got.Amount)
	assert.Equal(t, finance.CategoryFood, got.Category)
	assert.False(t, got.IsFixed)

	// 3. Update
	got.Amount = 125.00
	require.NoError(t, repo.Update(ctx, got))
	updated, err := repo.GetByID(ctx, "exp_test_1")
	require.NoError(t, err)
	assert.Equal(t, 125.00, updated.Amount)

	// 4. List with Filter
	catFood := finance.CategoryFood
	foodExpenses, err := repo.List(ctx, finance.Filter{Category: &catFood})
	require.NoError(t, err)
	assert.Len(t, foodExpenses, 1)

	// 5. Budget Target
	bgt := &finance.BudgetTarget{
		ID:           "bgt_oct",
		Category:     "total",
		MonthlyLimit: 1500.0,
		MonthYear:    "2026-10",
	}
	require.NoError(t, repo.SetBudget(ctx, bgt))

	budgets, err := repo.ListBudgets(ctx, "2026-10")
	require.NoError(t, err)
	assert.NotEmpty(t, budgets)

	// 6. GetSummary
	summary, err := repo.GetSummary(ctx, "month", refDate)
	require.NoError(t, err)
	assert.Equal(t, "month", summary.Period)
	// Total: 125 + 85 + 60 = 270.00
	assert.Equal(t, 270.00, summary.TotalSpent)
	assert.Equal(t, 1500.0, summary.MonthlyBudget)
	assert.Equal(t, 1230.00, summary.RemainingBudget)
	assert.Len(t, summary.Categories, 3)
	assert.Equal(t, 85.00, summary.FixedVsVariable.FixedAmount)
	assert.Equal(t, 185.00, summary.FixedVsVariable.VariableAmount)
	assert.Len(t, summary.MonthlyTrends, 12)

	// 7. Delete
	require.NoError(t, repo.Delete(ctx, "exp_test_3"))
	_, err = repo.GetByID(ctx, "exp_test_3")
	assert.Equal(t, finance.ErrExpenseNotFound, err)
}
