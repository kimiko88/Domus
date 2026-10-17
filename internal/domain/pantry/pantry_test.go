package pantry_test

import (
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/pantry"
)

func TestPantryItemValidationAndAlerts(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	validExp := "2026-10-20"
	expiredDate := "2026-10-15"

	item := pantry.PantryItem{
		Name:           "Latte Parzialmente Scremato",
		Category:       pantry.CategoryDairy,
		Quantity:       1.0,
		Unit:           pantry.UnitLiter,
		MinThreshold:   2.0,
		ExpirationDate: &validExp,
		Location:       pantry.LocationFridge,
	}

	if err := item.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	// Should be detected as low stock because quantity (1.0) <= min_threshold (2.0)
	if !item.IsLowStock() {
		t.Errorf("expected item to be low stock")
	}

	// Should not be expired on 2026-10-18
	if item.IsExpired(now) {
		t.Errorf("item should not be expired yet")
	}

	// Should be expiring soon within 3 days (expires on 2026-10-20)
	if !item.IsExpiringSoon(now, 3) {
		t.Errorf("expected item to be expiring soon within 3 days")
	}

	// Test expired item
	item.ExpirationDate = &expiredDate
	if !item.IsExpired(now) {
		t.Errorf("expected item with date %s to be expired on %v", expiredDate, now)
	}
}

func TestRecipeIngredientScaling(t *testing.T) {
	recipe := pantry.Recipe{
		Title:        "Spaghetti alla Carbonara",
		Category:     pantry.RecipeFirstCourse,
		PortionsBase: 2, // Base recipe for 2 people
		Ingredients: []pantry.RecipeIngredient{
			{Name: "Spaghetti", Quantity: 200, Unit: pantry.UnitGram},
			{Name: "Guanciale", Quantity: 100, Unit: pantry.UnitGram},
			{Name: "Tuorli d'uovo", Quantity: 3, Unit: pantry.UnitPiece},
			{Name: "Pecorino Romano", Quantity: 50, Unit: pantry.UnitGram},
		},
	}

	if err := recipe.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	// Scale for 4 people (multiplier = 4 / 2 = 2.0)
	scaled4 := recipe.ScaleIngredients(4)
	if len(scaled4) != 4 {
		t.Fatalf("expected 4 ingredients, got %d", len(scaled4))
	}
	if scaled4[0].Quantity != 400 {
		t.Errorf("expected 400g spaghetti, got %.1f", scaled4[0].Quantity)
	}
	if scaled4[1].Quantity != 200 {
		t.Errorf("expected 200g guanciale, got %.1f", scaled4[1].Quantity)
	}
	if scaled4[2].Quantity != 6 {
		t.Errorf("expected 6 egg yolks, got %.1f", scaled4[2].Quantity)
	}
	if scaled4[3].Quantity != 100 {
		t.Errorf("expected 100g pecorino, got %.1f", scaled4[3].Quantity)
	}

	// Scale for 1 person (multiplier = 1 / 2 = 0.5)
	scaled1 := recipe.ScaleIngredients(1)
	if scaled1[0].Quantity != 100 {
		t.Errorf("expected 100g spaghetti for 1 person, got %.1f", scaled1[0].Quantity)
	}
}

func TestMealPlanValidation(t *testing.T) {
	validPlan := pantry.MealPlan{
		PlanDate:     "2026-10-18",
		MealType:     pantry.MealDinner,
		PersonsCount: 4,
	}
	if err := validPlan.Validate(); err != nil {
		t.Errorf("unexpected error on valid plan: %v", err)
	}

	invalidDatePlan := pantry.MealPlan{
		PlanDate:     "18-10-2026", // Wrong format
		MealType:     pantry.MealDinner,
		PersonsCount: 2,
	}
	if err := invalidDatePlan.Validate(); err == nil {
		t.Errorf("expected error for invalid date format")
	}

	zeroPersonsPlan := pantry.MealPlan{
		PlanDate:     "2026-10-18",
		MealType:     pantry.MealLunch,
		PersonsCount: 0,
	}
	if err := zeroPersonsPlan.Validate(); err == nil {
		t.Errorf("expected error for zero persons count")
	}
}
