package sqlite_test

import (
	"context"
	"testing"
	"time"

	domain "github.com/domus-hub/domus/internal/domain/pantry"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
)

func setupPantryTestDB(t *testing.T) *sqlite.PantryRepository {
	t.Helper()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path:    ":memory:",
		WALMode: false,
	})
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}

	ctx := context.Background()
	if err := sqlite.Migrate(ctx, db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return sqlite.NewPantryRepository(db)
}

func TestPantryItemRepositoryCRUD(t *testing.T) {
	repo := setupPantryTestDB(t)
	ctx := context.Background()

	exp := "2026-10-25"
	item := &domain.PantryItem{
		Name:           "Pasta Spaghetti 500g",
		Category:       domain.CategoryPantry,
		Quantity:       3.0,
		Unit:           domain.UnitPiece,
		MinThreshold:   1.0,
		ExpirationDate: &exp,
		Location:       domain.LocationPantry,
		Cost:           1.19,
	}

	// 1. Create
	if err := repo.CreatePantryItem(ctx, item); err != nil {
		t.Fatalf("failed to create pantry item: %v", err)
	}

	// 2. Get by ID
	fetched, err := repo.GetPantryItem(ctx, item.ID)
	if err != nil {
		t.Fatalf("failed to get pantry item: %v", err)
	}
	if fetched.Name != item.Name || fetched.Quantity != 3.0 {
		t.Errorf("mismatched fetched item: %+v", fetched)
	}

	// 3. Get by Name
	byName, err := repo.GetPantryItemByName(ctx, "pasta spaghetti 500G")
	if err != nil || byName == nil {
		t.Fatalf("failed to get by case-insensitive name: %v", err)
	}

	// 4. Adjust Quantity
	updated, err := repo.AdjustPantryQuantity(ctx, item.ID, -2.5)
	if err != nil {
		t.Fatalf("failed to adjust quantity: %v", err)
	}
	if updated.Quantity != 0.5 {
		t.Errorf("expected 0.5 after deduction, got %.1f", updated.Quantity)
	}

	// 5. Low stock list (0.5 <= 1.0 threshold)
	lowStock, err := repo.ListLowStockPantryItems(ctx)
	if err != nil {
		t.Fatalf("failed to list low stock: %v", err)
	}
	if len(lowStock) != 1 {
		t.Errorf("expected 1 low stock item, got %d", len(lowStock))
	}

	// 6. Delete
	if err := repo.DeletePantryItem(ctx, item.ID); err != nil {
		t.Fatalf("failed to delete item: %v", err)
	}
	_, err = repo.GetPantryItem(ctx, item.ID)
	if err == nil {
		t.Errorf("expected ErrItemNotFound after delete")
	}
}

func TestRecipeAndMealPlanRepository(t *testing.T) {
	repo := setupPantryTestDB(t)
	ctx := context.Background()

	// 1. Create Recipe with ingredients
	recipe := &domain.Recipe{
		Title:        "Risotto allo Zafferano",
		Category:     domain.RecipeFirstCourse,
		PortionsBase: 4,
		Instructions: "Tostare il riso, aggiungere brodo bollente e zafferano.",
		Ingredients: []domain.RecipeIngredient{
			{Name: "Riso Carnaroli", Quantity: 320, Unit: domain.UnitGram},
			{Name: "Zafferano", Quantity: 1, Unit: domain.UnitPiece},
			{Name: "Burro", Quantity: 40, Unit: domain.UnitGram},
			{Name: "Parmigiano Reggiano", Quantity: 60, Unit: domain.UnitGram},
		},
	}

	if err := repo.CreateRecipe(ctx, recipe); err != nil {
		t.Fatalf("failed to create recipe: %v", err)
	}

	fetchedRecipe, err := repo.GetRecipe(ctx, recipe.ID)
	if err != nil {
		t.Fatalf("failed to get recipe: %v", err)
	}
	if len(fetchedRecipe.Ingredients) != 4 {
		t.Fatalf("expected 4 ingredients loaded, got %d", len(fetchedRecipe.Ingredients))
	}

	// 2. Create Meal Plan
	plan := &domain.MealPlan{
		PlanDate:     "2026-10-19",
		MealType:     domain.MealDinner,
		RecipeID:     &recipe.ID,
		PersonsCount: 6,
		Status:       domain.MealStatusPlanned,
	}

	if err := repo.CreateMealPlan(ctx, plan); err != nil {
		t.Fatalf("failed to create meal plan: %v", err)
	}

	plans, err := repo.ListMealPlans(ctx, "2026-10-18", "2026-10-20")
	if err != nil {
		t.Fatalf("failed to list meal plans: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 meal plan, got %d", len(plans))
	}
	if plans[0].Recipe == nil || plans[0].Recipe.Title != "Risotto allo Zafferano" {
		t.Errorf("expected recipe joined into meal plan")
	}

	// 3. Mark consumed
	now := time.Now().UTC()
	if err := repo.MarkMealPlanConsumed(ctx, plan.ID, now); err != nil {
		t.Fatalf("failed to mark consumed: %v", err)
	}

	consumedPlan, err := repo.GetMealPlan(ctx, plan.ID)
	if err != nil {
		t.Fatalf("failed to get plan: %v", err)
	}
	if consumedPlan.Status != domain.MealStatusConsumed || consumedPlan.ConsumedAt == nil {
		t.Errorf("expected plan status consumed with timestamp")
	}
}
