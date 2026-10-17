// Package pantry defines the domain models, validation, and contracts for
// household food inventory, recipes, meal planning, and receipt scanning.
package pantry

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidPantryName   = errors.New("pantry: item name cannot be empty and must be under 150 characters")
	ErrInvalidQuantity     = errors.New("pantry: quantity cannot be negative")
	ErrInvalidThreshold    = errors.New("pantry: threshold cannot be negative")
	ErrItemNotFound        = errors.New("pantry: item not found")
	ErrInvalidRecipeTitle  = errors.New("pantry: recipe title cannot be empty")
	ErrInvalidPortions     = errors.New("pantry: portions base must be strictly positive (>= 1)")
	ErrRecipeNotFound      = errors.New("pantry: recipe not found")
	ErrInvalidMealDate     = errors.New("pantry: meal plan date must follow YYYY-MM-DD")
	ErrMealPlanNotFound    = errors.New("pantry: meal plan not found")
	ErrInvalidPersonsCount = errors.New("pantry: persons count must be at least 1")
)

// Category classifies food and pantry goods into semantic categories.
type Category string

const (
	CategoryProduce   Category = "produce"    // Frutta & Verdura
	CategoryDairy     Category = "dairy"      // Latte, Formaggi, Latticini, Uova
	CategoryMeatFish  Category = "meat_fish"  // Carne, Pesce, Salumi
	CategoryBakery    Category = "bakery"     // Pane, Pasticceria, Prodotti da forno
	CategoryPantry    Category = "pantry"     // Pasta, Riso, Farina, Scatolame, Legumi
	CategoryFrozen    Category = "frozen"     // Surgelati e Gelati
	CategoryDrinks    Category = "drinks"     // Acqua, Succhi, Vino, Bevande
	CategoryCondiment Category = "condiments" // Olio, Aceto, Spezie, Salse
	CategorySnack     Category = "snacks"     // Biscotti, Snack, Cioccolato
	CategoryHousehold Category = "household"  // Detersivi, Carta, Cura della casa
	CategoryOther     Category = "other"      // Altro
)

// ValidCategories contains all supported pantry item categories.
var ValidCategories = []Category{
	CategoryProduce,
	CategoryDairy,
	CategoryMeatFish,
	CategoryBakery,
	CategoryPantry,
	CategoryFrozen,
	CategoryDrinks,
	CategoryCondiment,
	CategorySnack,
	CategoryHousehold,
	CategoryOther,
}

// Unit represents the physical measurement unit of the item.
type Unit string

const (
	UnitPiece    Unit = "pz"
	UnitGram     Unit = "g"
	UnitKilogram Unit = "kg"
	UnitMillilit Unit = "ml"
	UnitLiter    Unit = "l"
	UnitPack     Unit = "pack"
)

// Location specifies where the product is kept in the household.
type Location string

const (
	LocationPantry  Location = "dispensa"
	LocationFridge  Location = "frigo"
	LocationFreezer Location = "freezer"
	LocationCellar  Location = "cantina"
)

// PantryItem represents an ingredient or product stored in the household.
type PantryItem struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Category       Category  `json:"category"`
	Quantity       float64   `json:"quantity"`
	Unit           Unit      `json:"unit"`
	MinThreshold   float64   `json:"min_threshold"`
	ExpirationDate *string   `json:"expiration_date,omitempty"` // YYYY-MM-DD
	Location       Location  `json:"location"`
	Barcode        *string   `json:"barcode,omitempty"`
	Cost           float64   `json:"cost"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Validate checks entity integrity before persisting.
func (p *PantryItem) Validate() error {
	trimmed := strings.TrimSpace(p.Name)
	if trimmed == "" || len(trimmed) > 150 {
		return ErrInvalidPantryName
	}
	p.Name = trimmed
	if p.Quantity < 0 {
		return ErrInvalidQuantity
	}
	if p.MinThreshold < 0 {
		return ErrInvalidThreshold
	}
	if p.Category == "" {
		p.Category = CategoryOther
	}
	if p.Unit == "" {
		p.Unit = UnitPiece
	}
	if p.Location == "" {
		p.Location = LocationPantry
	}
	return nil
}

// IsLowStock returns true if current quantity is less than or equal to the minimum threshold.
func (p *PantryItem) IsLowStock() bool {
	return p.Quantity <= p.MinThreshold
}

// IsExpired checks if the item expiration date is strictly in the past.
func (p *PantryItem) IsExpired(now time.Time) bool {
	if p.ExpirationDate == nil || *p.ExpirationDate == "" {
		return false
	}
	t, err := time.Parse("2006-01-02", *p.ExpirationDate)
	if err != nil {
		return false
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return t.Before(today)
}

// IsExpiringSoon checks if the item expires within the given number of days.
func (p *PantryItem) IsExpiringSoon(now time.Time, withinDays int) bool {
	if p.ExpirationDate == nil || *p.ExpirationDate == "" {
		return false
	}
	t, err := time.Parse("2006-01-02", *p.ExpirationDate)
	if err != nil {
		return false
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	deadline := today.AddDate(0, 0, withinDays)
	return !t.Before(today) && (t.Before(deadline) || t.Equal(deadline))
}
