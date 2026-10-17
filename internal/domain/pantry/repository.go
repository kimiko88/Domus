package pantry

import (
	"context"
	"time"
)

// PantryRepository abstracts persistence for household pantry inventory items.
type PantryRepository interface {
	CreatePantryItem(ctx context.Context, item *PantryItem) error
	GetPantryItem(ctx context.Context, id string) (*PantryItem, error)
	GetPantryItemByName(ctx context.Context, name string) (*PantryItem, error)
	UpdatePantryItem(ctx context.Context, item *PantryItem) error
	DeletePantryItem(ctx context.Context, id string) error
	ListPantryItems(ctx context.Context, category Category, location Location) ([]PantryItem, error)
	ListExpiringPantryItems(ctx context.Context, days int) ([]PantryItem, error)
	ListLowStockPantryItems(ctx context.Context) ([]PantryItem, error)
	AdjustPantryQuantity(ctx context.Context, id string, delta float64) (*PantryItem, error)
}

// RecipeRepository abstracts persistence for culinary recipes and their ingredients.
type RecipeRepository interface {
	CreateRecipe(ctx context.Context, recipe *Recipe) error
	GetRecipe(ctx context.Context, id string) (*Recipe, error)
	UpdateRecipe(ctx context.Context, recipe *Recipe) error
	DeleteRecipe(ctx context.Context, id string) error
	ListRecipes(ctx context.Context, category RecipeCategory) ([]Recipe, error)
}

// MealPlanRepository abstracts persistence for the household dining calendar.
type MealPlanRepository interface {
	CreateMealPlan(ctx context.Context, plan *MealPlan) error
	GetMealPlan(ctx context.Context, id string) (*MealPlan, error)
	UpdateMealPlan(ctx context.Context, plan *MealPlan) error
	DeleteMealPlan(ctx context.Context, id string) error
	ListMealPlans(ctx context.Context, startDate, endDate string) ([]MealPlan, error)
	MarkMealPlanConsumed(ctx context.Context, id string, consumedAt time.Time) error
}

// Repository aggregates pantry, recipe, and meal plan persistence operations.
type Repository interface {
	PantryRepository
	RecipeRepository
	MealPlanRepository
}
