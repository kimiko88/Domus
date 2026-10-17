// Package finance defines the domain model, categories, financial summaries, and savings insights for Home Finance.
package finance

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidTitle       = errors.New("finance: title cannot be empty and must be under 150 characters")
	ErrInvalidAmount      = errors.New("finance: amount must be strictly greater than zero")
	ErrExpenseNotFound    = errors.New("finance: expense not found")
	ErrInvalidCategory    = errors.New("finance: invalid expense category")
	ErrInvalidDate        = errors.New("finance: expense date is invalid or in the future")
	ErrInvalidBudgetLimit = errors.New("finance: budget limit must be greater than zero")
)

// Category identifies standard household expense domains.
type Category string

const (
	CategoryFood            Category = "food"             // Spesa supermercato, panetteria, macelleria
	CategoryUtilities       Category = "utilities"        // Elettricità, gas, acqua, rifiuti (TARI), fibra/telefono
	CategoryTransport       Category = "transport"        // Benzina, diesel, ricarica EV, telepass, manutenzione, bollo
	CategoryHomeMaintenance Category = "home_maintenance" // Materiali, ferramenta, elettrodomestici, arredamento
	CategoryInsurance       Category = "insurance"        // Auto, casa, sanitaria, vita
	CategoryOther           Category = "other"            // Salute, farmacia, istruzione, svago
)

// Subcategory provides granular classification within each macro category.
type Subcategory string

const (
	// Utilities
	SubcategoryElectricity   Subcategory = "electricity"
	SubcategoryGas           Subcategory = "gas"
	SubcategoryWater         Subcategory = "water"
	SubcategoryWaste         Subcategory = "waste"
	SubcategoryInternetPhone Subcategory = "internet_phone"

	// Transport
	SubcategoryFuelPetrol       Subcategory = "fuel_petrol"
	SubcategoryFuelDiesel       Subcategory = "fuel_diesel"
	SubcategoryEVCharge         Subcategory = "ev_charge"
	SubcategoryToll             Subcategory = "toll"
	SubcategoryVehicleRepair    Subcategory = "vehicle_repair"
	SubcategoryVehicleTax       Subcategory = "vehicle_tax"

	// Insurance
	SubcategoryCarInsurance    Subcategory = "car_insurance"
	SubcategoryHomeInsurance   Subcategory = "home_insurance"
	SubcategoryHealthInsurance Subcategory = "health_insurance"
	SubcategoryLifeInsurance   Subcategory = "life_insurance"

	// Food
	SubcategoryGroceries Subcategory = "groceries"
	SubcategoryBakery    Subcategory = "bakery"
	SubcategoryButcher   Subcategory = "butcher"
	SubcategoryDiningOut Subcategory = "dining_out"
)

// RecurringFrequency specifies recurrence of contracts or bills.
type RecurringFrequency string

const (
	FrequencyNone       RecurringFrequency = "none"
	FrequencyMonthly    RecurringFrequency = "monthly"
	FrequencyBimonthly  RecurringFrequency = "bimonthly"
	FrequencyQuarterly  RecurringFrequency = "quarterly"
	FrequencySemiannual RecurringFrequency = "semiannual"
	FrequencyAnnual     RecurringFrequency = "annual"
)

// Expense represents an individual financial transaction or invoice.
type Expense struct {
	ID                  string             `json:"id"`
	Title               string             `json:"title"`
	Amount              float64            `json:"amount"`
	Currency            string             `json:"currency"`
	Category            Category           `json:"category"`
	Subcategory         Subcategory        `json:"subcategory,omitempty"`
	ExpenseDate         time.Time          `json:"expense_date"`
	IsFixed             bool               `json:"is_fixed"`
	PaymentMethod       string             `json:"payment_method,omitempty"` // card, cash, bank_transfer, direct_debit
	PayerUserID         string             `json:"payer_user_id,omitempty"`
	Notes               string             `json:"notes,omitempty"`
	RecurringFrequency  RecurringFrequency `json:"recurring_frequency"`
	ContractExpiryDate  *time.Time         `json:"contract_expiry_date,omitempty"`
	EnergyKWh           float64            `json:"energy_kwh,omitempty"`        // kWh in bolletta elettrica
	GasSmc              float64            `json:"gas_smc,omitempty"`           // Smc in bolletta gas
	FoodWasteValue      float64            `json:"food_waste_value,omitempty"`  // Valore economico del cibo scaduto/sprecato
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

// Validate checks business rules for an expense.
func (e *Expense) Validate() error {
	e.Title = strings.TrimSpace(e.Title)
	if e.Title == "" || len(e.Title) > 150 {
		return ErrInvalidTitle
	}
	if e.Amount <= 0 {
		return ErrInvalidAmount
	}
	if !isValidCategory(e.Category) {
		return ErrInvalidCategory
	}
	if e.Currency == "" {
		e.Currency = "EUR"
	}
	if e.RecurringFrequency == "" {
		e.RecurringFrequency = FrequencyNone
	}
	return nil
}

func isValidCategory(c Category) bool {
	switch c {
	case CategoryFood, CategoryUtilities, CategoryTransport,
		CategoryHomeMaintenance, CategoryInsurance, CategoryOther:
		return true
	default:
		return false
	}
}

// BudgetTarget sets monthly limit per category or overall.
type BudgetTarget struct {
	ID           string    `json:"id"`
	Category     string    `json:"category"` // category or "total"
	MonthlyLimit float64   `json:"monthly_limit"`
	MonthYear    string    `json:"month_year"` // YYYY-MM or "global"
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CategoryBreakdown gives aggregated amount and percentage per category.
type CategoryBreakdown struct {
	Category    Category `json:"category"`
	Name        string   `json:"name"`
	TotalAmount float64  `json:"total_amount"`
	Percentage  float64  `json:"percentage"`
	Count       int      `json:"count"`
}

// MonthlyTrend tracks spending over 12 months.
type MonthlyTrend struct {
	YearMonth      string  `json:"year_month"` // e.g. "2026-01"
	MonthName      string  `json:"month_name"` // e.g. "Gen 2026"
	TotalAmount    float64 `json:"total_amount"`
	FixedAmount    float64 `json:"fixed_amount"`
	VariableAmount float64 `json:"variable_amount"`
}

// FixedVsVariable separates non-compressible costs from variable ones.
type FixedVsVariable struct {
	FixedAmount        float64 `json:"fixed_amount"`
	FixedPercentage    float64 `json:"fixed_percentage"`
	VariableAmount     float64 `json:"variable_amount"`
	VariablePercentage float64 `json:"variable_percentage"`
}

// SavingsInsight provides automated smart recommendations.
type SavingsInsight struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"` // "energy_tariff", "food_waste", "contract_expiry", "spending_spike"
	Title           string  `json:"title"`
	Message         string  `json:"message"`
	PotentialSaving float64 `json:"potential_saving,omitempty"`
	Severity        string  `json:"severity"` // "info", "warning", "alert"
	ActionURL       string  `json:"action_url,omitempty"`
}

// FinancialSummary encapsulates all KPIs, breakdowns and trends for a given time window.
type FinancialSummary struct {
	Period              string              `json:"period"` // "day", "week", "month", "year"
	StartDate           string              `json:"start_date"`
	EndDate             string              `json:"end_date"`
	TotalSpent          float64             `json:"total_spent"`
	PreviousPeriodTotal float64             `json:"previous_period_total"`
	PercentageChange    float64             `json:"percentage_change"` // e.g. +12.5 or -8.3
	DailyAverage        float64             `json:"daily_average"`
	MonthlyBudget       float64             `json:"monthly_budget"`
	RemainingBudget     float64             `json:"remaining_budget"`
	Categories          []CategoryBreakdown `json:"categories"`
	MonthlyTrends       []MonthlyTrend      `json:"monthly_trends"`
	FixedVsVariable     FixedVsVariable     `json:"fixed_vs_variable"`
	Insights            []SavingsInsight    `json:"insights"`
}

// Filter provides querying parameters.
type Filter struct {
	Category    *Category
	Subcategory *Subcategory
	StartDate   *time.Time
	EndDate     *time.Time
	IsFixed     *bool
	PayerUserID *string
	Limit       int
	Offset      int
}

// Repository defines storage contracts for household expenses and budgets.
type Repository interface {
	Create(ctx context.Context, expense *Expense) error
	GetByID(ctx context.Context, id string) (*Expense, error)
	Update(ctx context.Context, expense *Expense) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter Filter) ([]*Expense, error)
	GetSummary(ctx context.Context, period string, refDate time.Time) (*FinancialSummary, error)
	SetBudget(ctx context.Context, budget *BudgetTarget) error
	GetBudget(ctx context.Context, category, monthYear string) (*BudgetTarget, error)
	ListBudgets(ctx context.Context, monthYear string) ([]*BudgetTarget, error)
}
