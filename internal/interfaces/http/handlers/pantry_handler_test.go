package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	appFinance "github.com/domus-hub/domus/internal/application/finance"
	appPantry "github.com/domus-hub/domus/internal/application/pantry"
	appShop "github.com/domus-hub/domus/internal/application/shopping"
	domainPantry "github.com/domus-hub/domus/internal/domain/pantry"
	domainShop "github.com/domus-hub/domus/internal/domain/shopping"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testShoppingAdapter struct {
	svc *appShop.Service
}

func (a *testShoppingAdapter) AddItem(ctx context.Context, title string, quantity string, category string, addedBy string) error {
	_, err := a.svc.AddItem(ctx, title, quantity, domainShop.Category(category), addedBy)
	return err
}

func setupPantryHandlerTest(t *testing.T) (*chi.Mux, *appPantry.Service, *appFinance.Service) {
	t.Helper()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path:    ":memory:",
		WALMode: false,
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = sqlite.Migrate(ctx, db)
	require.NoError(t, err)

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	pantryRepo := sqlite.NewPantryRepository(db)
	financeRepo := sqlite.NewFinanceRepository(db)
	shoppingRepo := sqlite.NewShoppingRepository(db)

	finSvc := appFinance.NewService(financeRepo, bus)
	shopSvc := appShop.NewService(shoppingRepo, bus)
	shopAdapter := &testShoppingAdapter{svc: shopSvc}

	pantrySvc := appPantry.NewService(pantryRepo, bus, finSvc, shopAdapter, nil)
	pantryHandler := handlers.NewPantryHandler(pantrySvc)

	r := chi.NewRouter()
	pantryHandler.RegisterRoutes(r)

	return r, pantrySvc, finSvc
}

func TestPantryHandler_ItemsCRUDAndAdjust(t *testing.T) {
	r, _, _ := setupPantryHandlerTest(t)

	// 1. Initially empty
	req := httptest.NewRequest(http.MethodGet, "/pantry/items", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var items []domainPantry.PantryItem
	err := json.Unmarshal(rr.Body.Bytes(), &items)
	require.NoError(t, err)
	assert.Empty(t, items)

	// 2. Create item
	expDate := "2026-10-25"
	createBody := appPantry.CreatePantryItemRequest{
		Name:           "Pasta Rigatoni 500g",
		Category:       domainPantry.CategoryPantry,
		Quantity:       5.0,
		Unit:           domainPantry.UnitPiece,
		MinThreshold:   2.0,
		ExpirationDate: &expDate,
		Location:       domainPantry.LocationPantry,
		Cost:           1.39,
	}
	raw, _ := json.Marshal(createBody)
	req = httptest.NewRequest(http.MethodPost, "/pantry/items", bytes.NewReader(raw))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var created domainPantry.PantryItem
	err = json.Unmarshal(rr.Body.Bytes(), &created)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Pasta Rigatoni 500g", created.Name)
	assert.Equal(t, 5.0, created.Quantity)

	// 3. Get item by ID
	req = httptest.NewRequest(http.MethodGet, "/pantry/items/"+created.ID, nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 4. Adjust quantity (-4.0 -> remaining 1.0, which is below min_threshold 2.0)
	adjBody := map[string]float64{"delta": -4.0}
	rawAdj, _ := json.Marshal(adjBody)
	req = httptest.NewRequest(http.MethodPost, "/pantry/items/"+created.ID+"/adjust", bytes.NewReader(rawAdj))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var adjusted domainPantry.PantryItem
	err = json.Unmarshal(rr.Body.Bytes(), &adjusted)
	require.NoError(t, err)
	assert.Equal(t, 1.0, adjusted.Quantity)

	// 5. Verify low stock endpoint finds it
	req = httptest.NewRequest(http.MethodGet, "/pantry/low-stock", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var lowStockList []domainPantry.PantryItem
	err = json.Unmarshal(rr.Body.Bytes(), &lowStockList)
	require.NoError(t, err)
	assert.Len(t, lowStockList, 1)
	assert.Equal(t, created.ID, lowStockList[0].ID)

	// 6. Delete item
	req = httptest.NewRequest(http.MethodDelete, "/pantry/items/"+created.ID, nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify 404 after deletion
	req = httptest.NewRequest(http.MethodGet, "/pantry/items/"+created.ID, nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestPantryHandler_RecipesAndCookable(t *testing.T) {
	r, pantrySvc, _ := setupPantryHandlerTest(t)
	ctx := context.Background()

	// Populate pantry with 400g pasta and 100g parmigiano
	_, err := pantrySvc.CreatePantryItem(ctx, appPantry.CreatePantryItemRequest{
		Name:         "Pasta Mezze Maniche",
		Category:     domainPantry.CategoryPantry,
		Quantity:     400,
		Unit:         domainPantry.UnitGram,
		MinThreshold: 100,
		Location:     domainPantry.LocationPantry,
	})
	require.NoError(t, err)

	_, err = pantrySvc.CreatePantryItem(ctx, appPantry.CreatePantryItemRequest{
		Name:         "Pecorino Romano",
		Category:     domainPantry.CategoryDairy,
		Quantity:     150,
		Unit:         domainPantry.UnitGram,
		MinThreshold: 50,
		Location:     domainPantry.LocationFridge,
	})
	require.NoError(t, err)

	// Create Recipe: Cacio e Pepe
	recipeReq := appPantry.CreateRecipeRequest{
		Title:           "Pasta Cacio e Pepe",
		Category:        domainPantry.RecipeFirstCourse,
		PortionsBase:    2,
		PrepTimeMinutes: 5,
		CookTimeMinutes: 12,
		Instructions:    "Cuocere la pasta al dente, mantecare con pecorino e pepe nero.",
		Ingredients: []appPantry.RecipeIngredientRequest{
			{Name: "Pasta Mezze Maniche", Quantity: 200, Unit: domainPantry.UnitGram},
			{Name: "Pecorino Romano", Quantity: 80, Unit: domainPantry.UnitGram},
			{Name: "Pepe Nero", Quantity: 5, Unit: domainPantry.UnitGram, IsOptional: true},
		},
	}
	raw, _ := json.Marshal(recipeReq)
	req := httptest.NewRequest(http.MethodPost, "/recipes", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var createdRecipe domainPantry.Recipe
	err = json.Unmarshal(rr.Body.Bytes(), &createdRecipe)
	require.NoError(t, err)
	assert.NotEmpty(t, createdRecipe.ID)
	assert.Equal(t, "Pasta Cacio e Pepe", createdRecipe.Title)

	// Check /recipes/cookable
	req = httptest.NewRequest(http.MethodGet, "/recipes/cookable", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var cookableList []appPantry.CookableRecipeResult
	err = json.Unmarshal(rr.Body.Bytes(), &cookableList)
	require.NoError(t, err)
	assert.Len(t, cookableList, 1)
	assert.True(t, cookableList[0].IsCookable)
	assert.Equal(t, 100.0, cookableList[0].AvailablePercent)
}

func TestPantryHandler_MealPlanningAndConsumptionScaling(t *testing.T) {
	r, pantrySvc, _ := setupPantryHandlerTest(t)
	ctx := context.Background()

	// 1. Add pantry items
	pastaItem, err := pantrySvc.CreatePantryItem(ctx, appPantry.CreatePantryItemRequest{
		Name:         "Spaghetti",
		Category:     domainPantry.CategoryPantry,
		Quantity:     1000, // 1000g
		Unit:         domainPantry.UnitGram,
		MinThreshold: 200,
		Location:     domainPantry.LocationPantry,
	})
	require.NoError(t, err)

	// 2. Create recipe with portions_base = 2 (takes 200g spaghetti for 2 persons)
	recipe, err := pantrySvc.CreateRecipe(ctx, appPantry.CreateRecipeRequest{
		Title:        "Spaghetti al Pomodoro",
		Category:     domainPantry.RecipeFirstCourse,
		PortionsBase: 2,
		Ingredients: []appPantry.RecipeIngredientRequest{
			{
				PantryItemID: &pastaItem.ID,
				Name:         "Spaghetti",
				Quantity:     200,
				Unit:         domainPantry.UnitGram,
			},
		},
	})
	require.NoError(t, err)

	// 3. Plan meal for 4 persons (scaling factor = 4 / 2 = 2x -> requires 400g spaghetti!)
	mealReq := appPantry.CreateMealPlanRequest{
		PlanDate:     "2026-10-20",
		MealType:     domainPantry.MealLunch,
		RecipeID:     &recipe.ID,
		PersonsCount: 4,
	}
	raw, _ := json.Marshal(mealReq)
	req := httptest.NewRequest(http.MethodPost, "/meals", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var plannedMeal domainPantry.MealPlan
	err = json.Unmarshal(rr.Body.Bytes(), &plannedMeal)
	require.NoError(t, err)
	assert.Equal(t, domainPantry.MealStatusPlanned, plannedMeal.Status)

	// 4. Consume meal -> deducts 400g from 1000g => 600g remaining
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/meals/%s/consume", plannedMeal.ID), nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var consumedMeal domainPantry.MealPlan
	err = json.Unmarshal(rr.Body.Bytes(), &consumedMeal)
	require.NoError(t, err)
	assert.Equal(t, domainPantry.MealStatusConsumed, consumedMeal.Status)
	assert.NotNil(t, consumedMeal.ConsumedAt)

	// Check updated pantry stock
	updatedItem, err := pantrySvc.GetPantryItem(ctx, pastaItem.ID)
	require.NoError(t, err)
	assert.Equal(t, 600.0, updatedItem.Quantity)
}

func TestPantryHandler_ReceiptOCRAndAutoFinanceImport(t *testing.T) {
	r, _, _ := setupPantryHandlerTest(t)

	// 1. Raw receipt test text
	receiptText := `
ESSELUNGA S.P.A.
DATA 17/10/2026
LATTE INTERO 1L 1.59
PASTA PENNE 500G 1.25
TOTALE COMPLESSIVO EURO 2.84
`
	parseReq := map[string]string{"raw_text": receiptText}
	raw, _ := json.Marshal(parseReq)
	req := httptest.NewRequest(http.MethodPost, "/receipts/parse", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var parsed domainPantry.Receipt
	err := json.Unmarshal(rr.Body.Bytes(), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "Esselunga", parsed.StoreName)
	assert.Equal(t, "2026-10-17", parsed.Date)
	assert.Equal(t, 2.84, parsed.TotalAmount)
	assert.Len(t, parsed.Items, 2)

	// 2. Import receipt into Pantry and Home Finance
	importReq := appPantry.ImportReceiptRequest{
		StoreName:        parsed.StoreName,
		Date:             parsed.Date,
		Items:            parsed.Items,
		TotalAmount:      parsed.TotalAmount,
		AutoAddToFinance: true,
		AutoUpdatePantry: true,
	}
	rawImport, _ := json.Marshal(importReq)
	req = httptest.NewRequest(http.MethodPost, "/receipts/import", bytes.NewReader(rawImport))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var importRes domainPantry.ReceiptImportResult
	err = json.Unmarshal(rr.Body.Bytes(), &importRes)
	require.NoError(t, err)
	assert.Equal(t, 2, importRes.ItemsAdded)
	assert.NotNil(t, importRes.ExpenseID)
	assert.Equal(t, 2.84, importRes.ExpenseAmount)

	// 3. Verify items now exist in pantry
	req = httptest.NewRequest(http.MethodGet, "/pantry/items", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var pantryItems []domainPantry.PantryItem
	err = json.Unmarshal(rr.Body.Bytes(), &pantryItems)
	require.NoError(t, err)
	assert.Len(t, pantryItems, 2)
}
