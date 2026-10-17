package pantry

import (
	"strings"
	"time"
)

// RecipeCategory represents culinary courses.
type RecipeCategory string

const (
	RecipeFirstCourse RecipeCategory = "first_course" // Primi piatti (pasta, riso, zuppe)
	RecipeMainCourse  RecipeCategory = "main_course"  // Secondi (carne, pesce, uova)
	RecipeSideDish    RecipeCategory = "side_dish"    // Contorni
	RecipeSingleDish  RecipeCategory = "single_dish"  // Piatti unici
	RecipeBreakfast   RecipeCategory = "breakfast"    // Colazioni
	RecipeDessert     RecipeCategory = "dessert"      // Dolci
	RecipeSnack       RecipeCategory = "snack"        // Spuntini / Merende
)

// RecipeIngredient links an ingredient to a recipe, optionally mapping to a pantry item.
type RecipeIngredient struct {
	ID           string  `json:"id"`
	RecipeID     string  `json:"recipe_id"`
	PantryItemID *string `json:"pantry_item_id,omitempty"`
	Name         string  `json:"name"`
	Quantity     float64 `json:"quantity"` // Quantity for PortionsBase
	Unit         Unit    `json:"unit"`
	IsOptional   bool    `json:"is_optional"`
}

// Recipe defines a culinary preparation with base portions and ingredients.
type Recipe struct {
	ID              string             `json:"id"`
	Title           string             `json:"title"`
	Description     string             `json:"description,omitempty"`
	Category        RecipeCategory     `json:"category"`
	PortionsBase    int                `json:"portions_base"` // Default portion size (e.g. 2 or 4)
	PrepTimeMinutes int                `json:"prep_time_minutes"`
	CookTimeMinutes int                `json:"cook_time_minutes"`
	Instructions    string             `json:"instructions"`
	ImageURL        *string            `json:"image_url,omitempty"`
	Ingredients     []RecipeIngredient `json:"ingredients,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// Validate checks recipe integrity.
func (r *Recipe) Validate() error {
	trimmed := strings.TrimSpace(r.Title)
	if trimmed == "" {
		return ErrInvalidRecipeTitle
	}
	r.Title = trimmed
	if r.PortionsBase < 1 {
		return ErrInvalidPortions
	}
	if r.Category == "" {
		r.Category = RecipeMainCourse
	}
	return nil
}

// ScaledIngredient represents the exact dose needed for a specific number of diners.
type ScaledIngredient struct {
	Name         string  `json:"name"`
	PantryItemID *string `json:"pantry_item_id,omitempty"`
	Quantity     float64 `json:"quantity"`
	Unit         Unit    `json:"unit"`
	IsOptional   bool    `json:"is_optional"`
}

// ScaleIngredients calculates the exact ingredient quantities proportional to the given number of diners.
// Formula: ScaledQuantity = (BaseQuantity / PortionsBase) * personsCount
func (r *Recipe) ScaleIngredients(personsCount int) []ScaledIngredient {
	if personsCount <= 0 {
		personsCount = 1
	}
	base := float64(r.PortionsBase)
	if base <= 0 {
		base = 1.0
	}
	multiplier := float64(personsCount) / base

	scaled := make([]ScaledIngredient, len(r.Ingredients))
	for i, ing := range r.Ingredients {
		scaled[i] = ScaledIngredient{
			Name:         ing.Name,
			PantryItemID: ing.PantryItemID,
			Quantity:     ing.Quantity * multiplier,
			Unit:         ing.Unit,
			IsOptional:   ing.IsOptional,
		}
	}
	return scaled
}
