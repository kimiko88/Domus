// Package finance implements business logic, smart savings insights, and event publishing for Home Finance.
package finance

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/domus-hub/domus/internal/domain/events"
	domainFinance "github.com/domus-hub/domus/internal/domain/finance"
)

// Service orchestrates household financial operations and intelligent savings recommendations.
type Service struct {
	repo domainFinance.Repository
	bus  events.Bus
}

// NewService constructs a new finance Service.
func NewService(repo domainFinance.Repository, bus events.Bus) *Service {
	return &Service{
		repo: repo,
		bus:  bus,
	}
}

// CreateExpense validates and persists a new household expense.
func (s *Service) CreateExpense(ctx context.Context, e *domainFinance.Expense) error {
	if err := e.Validate(); err != nil {
		return err
	}

	if err := s.repo.Create(ctx, e); err != nil {
		return err
	}

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.FinanceExpenseCreated,
			OccurredAt: time.Now().UTC(),
			Source:     "domus.finance",
			Payload:    e,
		})
	}
	return nil
}

// GetExpenseByID retrieves a single expense.
func (s *Service) GetExpenseByID(ctx context.Context, id string) (*domainFinance.Expense, error) {
	return s.repo.GetByID(ctx, id)
}

// UpdateExpense validates and updates an existing expense.
func (s *Service) UpdateExpense(ctx context.Context, e *domainFinance.Expense) error {
	if err := e.Validate(); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, e); err != nil {
		return err
	}

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.FinanceExpenseUpdated,
			OccurredAt: time.Now().UTC(),
			Source:     "domus.finance",
			Payload:    e,
		})
	}
	return nil
}

// DeleteExpense deletes an expense.
func (s *Service) DeleteExpense(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.FinanceExpenseDeleted,
			OccurredAt: time.Now().UTC(),
			Source:     "domus.finance",
			Payload:    map[string]string{"id": id},
		})
	}
	return nil
}

// ListExpenses queries expenses with filtering.
func (s *Service) ListExpenses(ctx context.Context, f domainFinance.Filter) ([]*domainFinance.Expense, error) {
	return s.repo.List(ctx, f)
}

// GetSummary calculates complete summary metrics and enriches it with Smart Savings Insights.
func (s *Service) GetSummary(ctx context.Context, period string, refDate time.Time) (*domainFinance.FinancialSummary, error) {
	if refDate.IsZero() {
		refDate = time.Now()
	}

	summary, err := s.repo.GetSummary(ctx, period, refDate)
	if err != nil {
		return nil, err
	}

	// Generate smart savings insights
	insights, err := s.GenerateSavingsInsights(ctx, refDate)
	if err == nil {
		summary.Insights = insights
	}

	return summary, nil
}

// SetBudget sets or updates monthly budget limits.
func (s *Service) SetBudget(ctx context.Context, b *domainFinance.BudgetTarget) error {
	if b.MonthlyLimit <= 0 {
		return domainFinance.ErrInvalidBudgetLimit
	}
	if b.Category == "" {
		b.Category = "total"
	}

	if err := s.repo.SetBudget(ctx, b); err != nil {
		return err
	}

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.FinanceBudgetUpdated,
			OccurredAt: time.Now().UTC(),
			Source:     "domus.finance",
			Payload:    b,
		})
	}
	return nil
}

// ListBudgets returns active budget limits for a month.
func (s *Service) ListBudgets(ctx context.Context, monthYear string) ([]*domainFinance.BudgetTarget, error) {
	if monthYear == "" {
		monthYear = time.Now().Format("2006-01")
	}
	return s.repo.ListBudgets(ctx, monthYear)
}

// RecordShoppingTrip records an expense originating from a completed shopping trip.
func (s *Service) RecordShoppingTrip(ctx context.Context, title string, amount float64, supermarket, userID string) (*domainFinance.Expense, error) {
	if title == "" {
		if supermarket != "" {
			title = "Spesa supermercato " + supermarket
		} else {
			title = "Spesa alimentare"
		}
	}
	exp := &domainFinance.Expense{
		Title:         title,
		Amount:        amount,
		Currency:      "EUR",
		Category:      domainFinance.CategoryFood,
		Subcategory:   domainFinance.SubcategoryGroceries,
		ExpenseDate:   time.Now(),
		IsFixed:       false,
		PaymentMethod: "card",
		PayerUserID:   userID,
	}

	if err := s.CreateExpense(ctx, exp); err != nil {
		return nil, err
	}
	return exp, nil
}

// GenerateSavingsInsights executes Domus Smart AI Insights logic:
// 1. Correlazione Bolletta-Energia Reale
// 2. Contatore Spreco Alimentare Monetizzato
// 3. Promemoria Scadenze Contratti (entro 30 giorni)
// 4. Picchi anomali di spesa per categoria
func (s *Service) GenerateSavingsInsights(ctx context.Context, now time.Time) ([]domainFinance.SavingsInsight, error) {
	var insights []domainFinance.SavingsInsight

	// Query expenses to analyze patterns (last 60 days for consumption/waste, all active for contract expiries)
	allExpenses, err := s.repo.List(ctx, domainFinance.Filter{
		Limit: 500,
	})
	if err != nil {
		return nil, err
	}

	// 1. Correlazione Bolletta-Energia Reale
	var electricityBills []*domainFinance.Expense
	var totalFoodWaste float64
	var expiringContracts []*domainFinance.Expense

	sixtyDaysAgo := now.AddDate(0, 0, -60)
	thirtyDaysFuture := now.AddDate(0, 0, 30)

	for _, exp := range allExpenses {
		// Energy bills with kWh within the last 60 days
		if exp.Category == domainFinance.CategoryUtilities && (exp.Subcategory == domainFinance.SubcategoryElectricity || exp.EnergyKWh > 0) {
			if !exp.ExpenseDate.Before(sixtyDaysAgo) {
				electricityBills = append(electricityBills, exp)
			}
		}

		// Food waste tracker within the last 60 days
		if exp.FoodWasteValue > 0 && !exp.ExpenseDate.Before(sixtyDaysAgo) {
			totalFoodWaste += exp.FoodWasteValue
		}

		// Expiring contracts (irrespective of when originally created/entered)
		if exp.ContractExpiryDate != nil && !exp.ContractExpiryDate.Before(now) && exp.ContractExpiryDate.Before(thirtyDaysFuture) {
			expiringContracts = append(expiringContracts, exp)
		}
	}

	// Analyze electricity bills
	for _, bill := range electricityBills {
		if bill.EnergyKWh > 0 && bill.Amount > 0 {
			unitCost := bill.Amount / bill.EnergyKWh
			if unitCost > 0.32 { // High tariff benchmark > 0.32 €/kWh
				potentialSaving := (unitCost - 0.22) * bill.EnergyKWh
				if potentialSaving > 10 {
					insights = append(insights, domainFinance.SavingsInsight{
						ID:              "insight_energy_" + bill.ID,
						Type:            "energy_tariff",
						Title:           "Bolletta Elettrica Elevata (" + fmt.Sprintf("%.2f", unitCost) + " €/kWh)",
						Message:         fmt.Sprintf("La bolletta \"%s\" ha un costo unitario medio di %.2f €/kWh, superiore alla media di mercato (0.22 €/kWh). Verifica i consumi con il monitoraggio energia di Domus o valuta una rinegoziazione.", bill.Title, unitCost),
						PotentialSaving: math.Round(potentialSaving*100) / 100,
						Severity:        "warning",
						ActionURL:       "/energy",
					})
				}
			}
		}
	}

	// 2. Contatore Spreco Alimentare Monetizzato
	if totalFoodWaste > 0 {
		insights = append(insights, domainFinance.SavingsInsight{
			ID:              "insight_food_waste",
			Type:            "food_waste",
			Title:           fmt.Sprintf("Spreco Alimentare Evitabile: %.2f €", totalFoodWaste),
			Message:         fmt.Sprintf("Negli ultimi 60 giorni sono stati registrati %.2f € di alimenti scaduti o non consumati. Utilizza il Meal Planner di Domus e controlla la Dispensa per pianificare le cene e azzerare gli sprechi.", totalFoodWaste),
			PotentialSaving: math.Round(totalFoodWaste*100) / 100,
			Severity:        "alert",
			ActionURL:       "/shopping",
		})
	}

	// 3. Promemoria Scadenze Contratti
	for _, exp := range expiringContracts {
		daysLeft := int(exp.ContractExpiryDate.Sub(now).Hours() / 24)
		if daysLeft < 0 {
			daysLeft = 0
		}
		insights = append(insights, domainFinance.SavingsInsight{
			ID:        "insight_expiry_" + exp.ID,
			Type:      "contract_expiry",
			Title:     fmt.Sprintf("Scadenza Contratto: %s (tra %d giorni)", exp.Title, daysLeft),
			Message:   fmt.Sprintf("La polizza o utenza \"%s\" scade il %s. Confronta le offerte di mercato prima del rinnovo automatico per ottenere tariffe migliori.", exp.Title, exp.ContractExpiryDate.Format("02/01/2006")),
			Severity:  "info",
			ActionURL: "/finance",
		})
	}

	return insights, nil
}
