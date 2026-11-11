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
	domainPantry "github.com/domus-hub/domus/internal/domain/pantry"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPantryAndMealPlanner_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "pantry_e2e.db")

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

	// STEP 1: Scan and Parse Supermarket Receipt via /api/v1/receipts/parse
	receiptRaw := `
ESSELUNGA S.P.A.
VIA PISACANE 12 - MILANO
DATA 18/10/2026
PASTA PENNE RIGATE 500G 1.30
RISO CARNAROLI 1KG 2.80
LATTE PARZ SCREM 1L 1.50
TOTALE COMPLESSIVO EURO 5.60
`
	parsePayload, _ := json.Marshal(map[string]string{"raw_text": receiptRaw})
	resp, err := client.Post(ts.URL+"/api/v1/receipts/parse", "application/json", bytes.NewReader(parsePayload))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed domainPantry.Receipt
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "Esselunga", parsed.StoreName)
	assert.Equal(t, "2026-10-18", parsed.Date)
	assert.Equal(t, 5.60, parsed.TotalAmount)
	assert.Len(t, parsed.Items, 3)

	// STEP 2: Import Receipt (Updating Pantry + Automatically Recording in Home Finance)
	importPayload, _ := json.Marshal(map[string]any{
		"store_name":          parsed.StoreName,
		"date":                parsed.Date,
		"items":               parsed.Items,
		"total_amount":        parsed.TotalAmount,
		"auto_add_to_finance": true,
		"auto_update_pantry":  true,
	})
	resp, err = client.Post(ts.URL+"/api/v1/receipts/import", "application/json", bytes.NewReader(importPayload))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var importResult domainPantry.ReceiptImportResult
	err = json.NewDecoder(resp.Body).Decode(&importResult)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, 3, importResult.ItemsAdded)
	assert.NotNil(t, importResult.ExpenseID)

	// Verify expense recorded in Home Finance
	resp, err = client.Get(ts.URL + "/api/v1/finance/expenses")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var expenses []*domainFinance.Expense
	err = json.NewDecoder(resp.Body).Decode(&expenses)
	resp.Body.Close()
	require.NoError(t, err)
	require.Len(t, expenses, 1)
	assert.Equal(t, 5.60, expenses[0].Amount)
	assert.Equal(t, domainFinance.CategoryFood, expenses[0].Category)
	assert.Equal(t, "Spesa Esselunga", expenses[0].Title)

	// STEP 3: Verify Pantry Inventory items exist
	resp, err = client.Get(ts.URL + "/api/v1/pantry/items")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var pantryList []domainPantry.PantryItem
	err = json.NewDecoder(resp.Body).Decode(&pantryList)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Len(t, pantryList, 3)

	var pastaItem *domainPantry.PantryItem
	for i := range pantryList {
		if pantryList[i].Category == domainPantry.CategoryPantry && pantryList[i].Quantity == 500 {
			pastaItem = &pantryList[i]
			break
		}
	}
	require.NotNil(t, pastaItem, "pasta item should be in pantry with 500g")

	// STEP 4: Create a Recipe that uses Pasta (Portions Base = 2, takes 200g pasta)
	recipeBody, _ := json.Marshal(map[string]any{
		"title":             "Pasta al Pomodoro e Basilico",
		"category":          "first_course",
		"portions_base":     2,
		"prep_time_minutes": 5,
		"cook_time_minutes": 10,
		"instructions":      "Bollire la pasta e saltare con il pomodoro.",
		"ingredients": []map[string]any{
			{
				"pantry_item_id": pastaItem.ID,
				"name":           pastaItem.Name,
				"quantity":       200,
				"unit":           "g",
			},
		},
	})
	resp, err = client.Post(ts.URL+"/api/v1/recipes", "application/json", bytes.NewReader(recipeBody))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdRecipe domainPantry.Recipe
	err = json.NewDecoder(resp.Body).Decode(&createdRecipe)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "Pasta al Pomodoro e Basilico", createdRecipe.Title)

	// STEP 5: Check Cookable Recipes (500g in pantry >= 200g required -> Cookable!)
	resp, err = client.Get(ts.URL + "/api/v1/recipes/cookable")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var cookable []map[string]any
	err = json.NewDecoder(resp.Body).Decode(&cookable)
	resp.Body.Close()
	require.NoError(t, err)
	require.Len(t, cookable, 1)
	assert.True(t, cookable[0]["is_cookable"].(bool))

	// STEP 6: Plan a Meal for 4 Persons (requires 200g * (4/2) = 400g pasta!)
	mealBody, _ := json.Marshal(map[string]any{
		"plan_date":     "2026-10-18",
		"meal_type":     "lunch",
		"recipe_id":     createdRecipe.ID,
		"persons_count": 4,
	})
	resp, err = client.Post(ts.URL+"/api/v1/meals", "application/json", bytes.NewReader(mealBody))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var plannedMeal domainPantry.MealPlan
	err = json.NewDecoder(resp.Body).Decode(&plannedMeal)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, domainPantry.MealStatusPlanned, plannedMeal.Status)

	// STEP 7: Consume Meal Plan -> Proportional Ingredient Deduction
	// Pasta started at 500g, 400g deducted -> 100g remaining
	resp, err = client.Post(fmt.Sprintf("%s/api/v1/meals/%s/consume", ts.URL, plannedMeal.ID), "application/json", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var consumedMeal domainPantry.MealPlan
	err = json.NewDecoder(resp.Body).Decode(&consumedMeal)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, domainPantry.MealStatusConsumed, consumedMeal.Status)
	assert.NotNil(t, consumedMeal.ConsumedAt)

	// Verify updated pantry quantity
	resp, err = client.Get(fmt.Sprintf("%s/api/v1/pantry/items/%s", ts.URL, pastaItem.ID))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var updatedPasta domainPantry.PantryItem
	err = json.NewDecoder(resp.Body).Decode(&updatedPasta)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, 100.0, updatedPasta.Quantity)
}
