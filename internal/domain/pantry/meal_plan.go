package pantry

import (
	"strings"
	"time"
)

// MealType represents the daily slot of the meal.
type MealType string

const (
	MealBreakfast MealType = "breakfast"
	MealLunch     MealType = "lunch"
	MealDinner    MealType = "dinner"
	MealSnack     MealType = "snack"
	MealDessert   MealType = "dessert"
)

// MealStatus represents the operational state of the planned meal.
type MealStatus string

const (
	MealStatusPlanned  MealStatus = "planned"
	MealStatusConsumed MealStatus = "consumed"
	MealStatusSkipped  MealStatus = "skipped"
)

// MealPlan represents an entry in the household dining schedule.
type MealPlan struct {
	ID           string     `json:"id"`
	PlanDate     string     `json:"plan_date"` // YYYY-MM-DD
	MealType     MealType   `json:"meal_type"`
	RecipeID     *string    `json:"recipe_id,omitempty"`
	Recipe       *Recipe    `json:"recipe,omitempty"`
	CustomTitle  *string    `json:"custom_title,omitempty"`
	PersonsCount int        `json:"persons_count"`
	Notes        *string    `json:"notes,omitempty"`
	Status       MealStatus `json:"status"`
	ConsumedAt   *time.Time `json:"consumed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Validate checks meal plan integrity.
func (m *MealPlan) Validate() error {
	m.PlanDate = strings.TrimSpace(m.PlanDate)
	if _, err := time.Parse("2006-01-02", m.PlanDate); err != nil {
		return ErrInvalidMealDate
	}
	if m.PersonsCount < 1 {
		return ErrInvalidPersonsCount
	}
	if m.MealType == "" {
		m.MealType = MealDinner
	}
	if m.Status == "" {
		m.Status = MealStatusPlanned
	}
	return nil
}

// DisplayTitle returns the recipe title if present or custom title fallback.
func (m *MealPlan) DisplayTitle() string {
	if m.Recipe != nil && m.Recipe.Title != "" {
		return m.Recipe.Title
	}
	if m.CustomTitle != nil && *m.CustomTitle != "" {
		return *m.CustomTitle
	}
	return string(m.MealType)
}
