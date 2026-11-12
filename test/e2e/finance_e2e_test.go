// Package e2e provides end-to-end integration tests for the full Domus application stack.
package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	domainFinance "github.com/domus-hub/domus/internal/domain/finance"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeFinance_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "finance_e2e.db")

	cfg := config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 0,
		},
		Database: config.DatabaseConfig{
			Path:          dbPath,
			WALMode:       true,
			BusyTimeoutMs: 5000,
			MaxOpenConns:  1,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	db, err := sqlite.Open(cfg.Database)
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = sqlite.Migrate(ctx, db)
	require.NoError(t, err)

	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 256,
		Workers:    2,
	})
	defer bus.Close()

	repos := sqlite.NewRepositories(db)
	authOption, sessionID := configureE2EAuth(t, repos)
	srv := serverpkg.NewServer(cfg.Server, "1.0.0-test", time.Now(), db,
		serverpkg.WithRepositories(repos, bus),
		authOption,
	)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := newAuthenticatedE2EClient(ts, sessionID)

	// 1. Set Monthly Budget for the current month so the test is time-independent.
	now := time.Now().UTC()
	budgetPayload, _ := json.Marshal(map[string]any{
		"category": "total", "monthly_limit": 1800.0, "month_year": now.Format("2006-01"),
	})
	resp, err := client.Post(fmt.Sprintf("%s/api/v1/finance/budgets", ts.URL), "application/json", bytes.NewReader(budgetPayload))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 2. Create Expenses
	expensesToCreate := []map[string]interface{}{
		{
			"id":           "exp_food_1",
			"title":        "Spesa Conad",
			"amount":       95.50,
			"category":     "food",
			"subcategory":  "groceries",
			"expense_date": now.Format(time.RFC3339),
			"is_fixed":     false,
		},
		{
			"id":           "exp_util_1",
			"title":        "Bolletta Luce Ottobre",
			"amount":       180.00,
			"category":     "utilities",
			"subcategory":  "electricity",
			"expense_date": now.Format(time.RFC3339),
			"is_fixed":     true,
			"energy_kwh":   360.0, // 180 / 360 = 0.50 €/kWh -> triggers high tariff insight!
		},
		{
			"id":           "exp_trans_1",
			"title":        "Pieno Diesel",
			"amount":       75.00,
			"category":     "transport",
			"subcategory":  "fuel_diesel",
			"expense_date": now.Format(time.RFC3339),
			"is_fixed":     false,
		},
		{
			"id":                   "exp_ins_1",
			"title":                "Assicurazione Casa",
			"amount":               240.00,
			"category":             "insurance",
			"subcategory":          "home_insurance",
			"expense_date":         now.AddDate(0, -11, 0).Format(time.RFC3339),
			"is_fixed":             true,
			"contract_expiry_date": now.AddDate(0, 0, 18).Format(time.RFC3339), // expires in 18 days!
		},
		{
			"id":               "exp_waste_1",
			"title":            "Cibo andato a male",
			"amount":           18.50,
			"category":         "food",
			"expense_date":     now.Format(time.RFC3339),
			"food_waste_value": 18.50, // triggers food waste insight!
		},
	}

	for _, exp := range expensesToCreate {
		b, _ := json.Marshal(exp)
		resp, err := client.Post(fmt.Sprintf("%s/api/v1/finance/expenses", ts.URL), "application/json", bytes.NewBuffer(b))
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()
	}

	// 3. List Expenses
	resp, err = client.Get(fmt.Sprintf("%s/api/v1/finance/expenses", ts.URL))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var expList []*domainFinance.Expense
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&expList))
	resp.Body.Close()
	assert.Len(t, expList, 5)

	// 4. Get Summary with KPIs, Category Breakdown, 12-Month Trends, and Insights
	resp, err = client.Get(fmt.Sprintf("%s/api/v1/finance/summary?period=month", ts.URL))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var summary domainFinance.FinancialSummary
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&summary))
	resp.Body.Close()

	// Current month expenses: 95.50 + 180.00 + 75.00 + 18.50 = 369.00 (exp_ins_1 was 11 months ago)
	assert.Equal(t, 369.00, summary.TotalSpent)
	assert.Equal(t, 1800.0, summary.MonthlyBudget)
	assert.Equal(t, 1431.00, summary.RemainingBudget)
	assert.NotEmpty(t, summary.Categories)
	assert.Len(t, summary.MonthlyTrends, 12)
	assert.Equal(t, 180.00, summary.FixedVsVariable.FixedAmount)
	assert.Equal(t, 189.00, summary.FixedVsVariable.VariableAmount)

	// Verify Insights
	require.NotEmpty(t, summary.Insights)
	hasEnergyInsight := false
	hasFoodWasteInsight := false
	hasExpiryInsight := false

	for _, ins := range summary.Insights {
		switch ins.Type {
		case "energy_tariff":
			hasEnergyInsight = true
		case "food_waste":
			hasFoodWasteInsight = true
		case "contract_expiry":
			hasExpiryInsight = true
		}
	}

	assert.True(t, hasEnergyInsight, "expected energy_tariff insight for high unit price")
	assert.True(t, hasFoodWasteInsight, "expected food_waste insight")
	assert.True(t, hasExpiryInsight, "expected contract_expiry insight for policy expiring in 18 days")

	// 5. Update Expense
	updateJSON := `{"title":"Spesa Conad Maxi","amount":100.0,"category":"food"}`
	req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/v1/finance/expenses/exp_food_1", ts.URL), bytes.NewBufferString(updateJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 6. Delete Expense
	delReq, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/finance/expenses/exp_trans_1", ts.URL), nil)
	resp, err = client.Do(delReq)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Verify it was deleted
	getReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/finance/expenses/exp_trans_1", ts.URL), nil)
	resp, err = client.Do(getReq)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}
