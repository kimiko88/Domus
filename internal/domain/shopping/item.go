package shopping

import "time"

type Category string

const (
	CategoryGrocery  Category = "alimentari"
	CategoryProduce  Category = "frutta_verdura"
	CategoryBeverage Category = "bevande"
	CategoryPersonal Category = "igiene"
	CategoryHome     Category = "casa"
	CategoryOther    Category = "altro"
)

type Item struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Quantity    string     `json:"quantity,omitempty"`
	Category    Category   `json:"category"`
	Completed   bool       `json:"completed"`
	AddedBy     string     `json:"added_by"`
	CompletedBy *string    `json:"completed_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
