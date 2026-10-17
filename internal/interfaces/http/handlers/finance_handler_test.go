package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appFinance "github.com/domus-hub/domus/internal/application/finance"
	domainFinance "github.com/domus-hub/domus/internal/domain/finance"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memoryFinanceRepo struct {
	expenses map[string]*domainFinance.Expense
	budgets  map[string]*domainFinance.BudgetTarget
}

func newMemoryFinanceRepo() *memoryFinanceRepo {
	return &memoryFinanceRepo{
		expenses: make(map[string]*domainFinance.Expense),
		budgets:  make(map[string]*domainFinance.BudgetTarget),
	}
}

func (m *memoryFinanceRepo) Create(ctx context.Context, e *domainFinance.Expense) error {
	m.expenses[e.ID] = e
	return nil
}

func (m *memoryFinanceRepo) GetByID(ctx context.Context, id string) (*domainFinance.Expense, error) {
	e, ok := m.expenses[id]
	if !ok {
		return nil, domainFinance.ErrExpenseNotFound
	}
	return e, nil
}

func (m *memoryFinanceRepo) Update(ctx context.Context, e *domainFinance.Expense) error {
	if _, ok := m.expenses[e.ID]; !ok {
		return domainFinance.ErrExpenseNotFound
	}
	m.expenses[e.ID] = e
	return nil
}

func (m *memoryFinanceRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.expenses[id]; !ok {
		return domainFinance.ErrExpenseNotFound
	}
	delete(m.expenses, id)
	return nil
}

func (m *memoryFinanceRepo) List(ctx context.Context, f domainFinance.Filter) ([]*domainFinance.Expense, error) {
	var list []*domainFinance.Expense
	for _, e := range m.expenses {
		list = append(list, e)
	}
	return list, nil
}

func (m *memoryFinanceRepo) GetSummary(ctx context.Context, period string, refDate time.Time) (*domainFinance.FinancialSummary, error) {
	tot := 0.0
	for _, e := range m.expenses {
		tot += e.Amount
	}
	return &domainFinance.FinancialSummary{
		Period:     period,
		TotalSpent: tot,
		Categories: []domainFinance.CategoryBreakdown{
			{Category: domainFinance.CategoryFood, Name: "Cibo & Spesa", TotalAmount: tot, Percentage: 100, Count: len(m.expenses)},
		},
	}, nil
}

func (m *memoryFinanceRepo) SetBudget(ctx context.Context, b *domainFinance.BudgetTarget) error {
	m.budgets[b.Category] = b
	return nil
}

func (m *memoryFinanceRepo) GetBudget(ctx context.Context, category, monthYear string) (*domainFinance.BudgetTarget, error) {
	b, ok := m.budgets[category]
	if !ok {
		return nil, nil
	}
	return b, nil
}

func (m *memoryFinanceRepo) ListBudgets(ctx context.Context, monthYear string) ([]*domainFinance.BudgetTarget, error) {
	var list []*domainFinance.BudgetTarget
	for _, b := range m.budgets {
		list = append(list, b)
	}
	return list, nil
}

func TestFinanceHandler_Endpoints(t *testing.T) {
	repo := newMemoryFinanceRepo()
	svc := appFinance.NewService(repo, nil)
	h := handlers.NewFinanceHandler(svc)

	r := chi.NewRouter()
	r.Get("/api/v1/finance/expenses", h.List)
	r.Post("/api/v1/finance/expenses", h.Create)
	r.Get("/api/v1/finance/expenses/{id}", h.GetByID)
	r.Put("/api/v1/finance/expenses/{id}", h.Update)
	r.Delete("/api/v1/finance/expenses/{id}", h.Delete)
	r.Get("/api/v1/finance/summary", h.GetSummary)
	r.Post("/api/v1/finance/budgets", h.SetBudget)
	r.Get("/api/v1/finance/budgets", h.ListBudgets)
	r.Get("/api/v1/finance/insights", h.GetInsights)

	// 1. Create Expense
	payload := `{"id":"exp_h1","title":"Benzina Agip","amount":55.0,"category":"transport","subcategory":"fuel_petrol","expense_date":"2026-10-15T10:00:00Z"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/finance/expenses", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// 2. List Expenses
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/finance/expenses", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var list []*domainFinance.Expense
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	assert.Len(t, list, 1)
	assert.Equal(t, "Benzina Agip", list[0].Title)

	// 3. GetByID
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/finance/expenses/exp_h1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// 4. Update
	updatePayload := `{"title":"Benzina Q8","amount":60.0,"category":"transport","subcategory":"fuel_petrol","expense_date":"2026-10-15T10:00:00Z"}`
	req, _ = http.NewRequest(http.MethodPut, "/api/v1/finance/expenses/exp_h1", bytes.NewBufferString(updatePayload))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// 5. GetSummary
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/finance/summary?period=month", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var summary domainFinance.FinancialSummary
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &summary))
	assert.Equal(t, 60.0, summary.TotalSpent)

	// 6. SetBudget
	bgtPayload := `{"category":"total","monthly_limit":1200.0,"month_year":"2026-10"}`
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/finance/budgets", bytes.NewBufferString(bgtPayload))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// 7. ListBudgets
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/finance/budgets?month_year=2026-10", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// 8. Delete
	req, _ = http.NewRequest(http.MethodDelete, "/api/v1/finance/expenses/exp_h1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// 9. Verify Deleted
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/finance/expenses/exp_h1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}
