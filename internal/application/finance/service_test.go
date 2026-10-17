package finance_test

import (
	"context"
	"testing"
	"time"

	appFinance "github.com/domus-hub/domus/internal/application/finance"
	domainFinance "github.com/domus-hub/domus/internal/domain/finance"
)

type mockRepo struct {
	expenses map[string]*domainFinance.Expense
	budgets  map[string]*domainFinance.BudgetTarget
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		expenses: make(map[string]*domainFinance.Expense),
		budgets:  make(map[string]*domainFinance.BudgetTarget),
	}
}

func (m *mockRepo) Create(ctx context.Context, e *domainFinance.Expense) error {
	m.expenses[e.ID] = e
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*domainFinance.Expense, error) {
	e, ok := m.expenses[id]
	if !ok {
		return nil, domainFinance.ErrExpenseNotFound
	}
	return e, nil
}

func (m *mockRepo) Update(ctx context.Context, e *domainFinance.Expense) error {
	if _, ok := m.expenses[e.ID]; !ok {
		return domainFinance.ErrExpenseNotFound
	}
	m.expenses[e.ID] = e
	return nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.expenses[id]; !ok {
		return domainFinance.ErrExpenseNotFound
	}
	delete(m.expenses, id)
	return nil
}

func (m *mockRepo) List(ctx context.Context, f domainFinance.Filter) ([]*domainFinance.Expense, error) {
	var result []*domainFinance.Expense
	for _, e := range m.expenses {
		result = append(result, e)
	}
	return result, nil
}

func (m *mockRepo) GetSummary(ctx context.Context, period string, refDate time.Time) (*domainFinance.FinancialSummary, error) {
	total := 0.0
	for _, e := range m.expenses {
		total += e.Amount
	}
	return &domainFinance.FinancialSummary{
		Period:     period,
		TotalSpent: total,
	}, nil
}

func (m *mockRepo) SetBudget(ctx context.Context, b *domainFinance.BudgetTarget) error {
	m.budgets[b.Category] = b
	return nil
}

func (m *mockRepo) GetBudget(ctx context.Context, category, monthYear string) (*domainFinance.BudgetTarget, error) {
	b, ok := m.budgets[category]
	if !ok {
		return nil, nil
	}
	return b, nil
}

func (m *mockRepo) ListBudgets(ctx context.Context, monthYear string) ([]*domainFinance.BudgetTarget, error) {
	var res []*domainFinance.BudgetTarget
	for _, b := range m.budgets {
		res = append(res, b)
	}
	return res, nil
}

func TestFinanceService_CreateAndValidate(t *testing.T) {
	repo := newMockRepo()
	svc := appFinance.NewService(repo, nil)

	ctx := context.Background()

	// Invalid expense: negative amount
	badExp := &domainFinance.Expense{
		ID:          "exp_bad",
		Title:       "Test",
		Amount:      -10,
		Category:    domainFinance.CategoryFood,
		ExpenseDate: time.Now(),
	}
	if err := svc.CreateExpense(ctx, badExp); err != domainFinance.ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}

	// Valid expense
	validExp := &domainFinance.Expense{
		ID:          "exp_good",
		Title:       "Spesa Supermercato",
		Amount:      65.40,
		Category:    domainFinance.CategoryFood,
		Subcategory: domainFinance.SubcategoryGroceries,
		ExpenseDate: time.Now(),
	}
	if err := svc.CreateExpense(ctx, validExp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fetched, err := svc.GetExpenseByID(ctx, "exp_good")
	if err != nil {
		t.Fatalf("failed to fetch expense: %v", err)
	}
	if fetched.Amount != 65.40 {
		t.Errorf("expected 65.40, got %f", fetched.Amount)
	}
}

func TestFinanceService_SavingsInsights(t *testing.T) {
	repo := newMockRepo()
	svc := appFinance.NewService(repo, nil)
	ctx := context.Background()
	now := time.Now()

	// 1. High energy bill: 200€ for 400 kWh = 0.50 €/kWh (> 0.32 €/kWh threshold)
	_ = repo.Create(ctx, &domainFinance.Expense{
		ID:          "exp_energy",
		Title:       "Bolletta Enel",
		Amount:      200.0,
		Category:    domainFinance.CategoryUtilities,
		Subcategory: domainFinance.SubcategoryElectricity,
		EnergyKWh:   400.0,
		ExpenseDate: now.AddDate(0, 0, -5),
	})

	// 2. Food waste: 25.50€
	_ = repo.Create(ctx, &domainFinance.Expense{
		ID:             "exp_food_waste",
		Title:          "Verdure e yogurt scaduti",
		Amount:         25.50,
		Category:       domainFinance.CategoryFood,
		FoodWasteValue: 25.50,
		ExpenseDate:    now.AddDate(0, 0, -2),
	})

	// 3. Expiring contract: Car insurance in 15 days
	expiryDate := now.AddDate(0, 0, 15)
	_ = repo.Create(ctx, &domainFinance.Expense{
		ID:                 "exp_insurance",
		Title:              "Assicurazione Auto Genertel",
		Amount:             420.0,
		Category:           domainFinance.CategoryInsurance,
		ContractExpiryDate: &expiryDate,
		ExpenseDate:        now.AddDate(0, -11, 0),
	})

	insights, err := svc.GenerateSavingsInsights(ctx, now)
	if err != nil {
		t.Fatalf("failed to generate insights: %v", err)
	}

	if len(insights) < 3 {
		t.Fatalf("expected at least 3 insights, got %d", len(insights))
	}

	hasEnergy := false
	hasFoodWaste := false
	hasExpiry := false

	for _, ins := range insights {
		switch ins.Type {
		case "energy_tariff":
			hasEnergy = true
		case "food_waste":
			hasFoodWaste = true
		case "contract_expiry":
			hasExpiry = true
		}
	}

	if !hasEnergy {
		t.Errorf("missing energy_tariff insight")
	}
	if !hasFoodWaste {
		t.Errorf("missing food_waste insight")
	}
	if !hasExpiry {
		t.Errorf("missing contract_expiry insight")
	}
}
