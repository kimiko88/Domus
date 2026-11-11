// Package sqlite provides SQLite persistence for household finance, expenses, and budgets.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/domain/finance"
	"github.com/google/uuid"
)

// FinanceRepository implements finance.Repository using SQLite.
type FinanceRepository struct {
	db *sql.DB
}

// NewFinanceRepository constructs a new FinanceRepository.
func NewFinanceRepository(db *sql.DB) *FinanceRepository {
	return &FinanceRepository{db: db}
}

// Create inserts a new expense into SQLite.
func (r *FinanceRepository) Create(ctx context.Context, e *finance.Expense) error {
	if e.ID == "" {
		e.ID = "exp_" + uuid.New().String()
	}
	now := time.Now().UTC()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now
	if e.ExpenseDate.IsZero() {
		e.ExpenseDate = now
	}

	var expiryStr *string
	if e.ContractExpiryDate != nil {
		s := e.ContractExpiryDate.UTC().Format(time.RFC3339)
		expiryStr = &s
	}

	isFixedInt := 0
	if e.IsFixed {
		isFixedInt = 1
	}

	query := `
		INSERT INTO expenses (
			id, title, amount, currency, category, subcategory,
			expense_date, is_fixed, payment_method, payer_user_id,
			notes, recurring_frequency, contract_expiry_date,
			energy_kwh, gas_smc, food_waste_value, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		e.ID, e.Title, e.Amount, e.Currency, string(e.Category), string(e.Subcategory),
		e.ExpenseDate.UTC().Format(time.RFC3339), isFixedInt, e.PaymentMethod, e.PayerUserID,
		e.Notes, string(e.RecurringFrequency), expiryStr,
		e.EnergyKWh, e.GasSmc, e.FoodWasteValue,
		e.CreatedAt.UTC().Format(time.RFC3339), e.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("finance_repo: failed to insert expense: %w", err)
	}
	return nil
}

// GetByID retrieves a single expense by ID.
func (r *FinanceRepository) GetByID(ctx context.Context, id string) (*finance.Expense, error) {
	query := `
		SELECT id, title, amount, currency, category, subcategory,
		       expense_date, is_fixed, payment_method, payer_user_id,
		       notes, recurring_frequency, contract_expiry_date,
		       energy_kwh, gas_smc, food_waste_value, created_at, updated_at
		FROM expenses
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanExpense(row)
}

// Update modifies an existing expense.
func (r *FinanceRepository) Update(ctx context.Context, e *finance.Expense) error {
	e.UpdatedAt = time.Now().UTC()
	var expiryStr *string
	if e.ContractExpiryDate != nil {
		s := e.ContractExpiryDate.UTC().Format(time.RFC3339)
		expiryStr = &s
	}

	isFixedInt := 0
	if e.IsFixed {
		isFixedInt = 1
	}

	query := `
		UPDATE expenses SET
			title = ?, amount = ?, currency = ?, category = ?, subcategory = ?,
			expense_date = ?, is_fixed = ?, payment_method = ?, payer_user_id = ?,
			notes = ?, recurring_frequency = ?, contract_expiry_date = ?,
			energy_kwh = ?, gas_smc = ?, food_waste_value = ?, updated_at = ?
		WHERE id = ?
	`

	res, err := r.db.ExecContext(ctx, query,
		e.Title, e.Amount, e.Currency, string(e.Category), string(e.Subcategory),
		e.ExpenseDate.UTC().Format(time.RFC3339), isFixedInt, e.PaymentMethod, e.PayerUserID,
		e.Notes, string(e.RecurringFrequency), expiryStr,
		e.EnergyKWh, e.GasSmc, e.FoodWasteValue, e.UpdatedAt.UTC().Format(time.RFC3339),
		e.ID,
	)
	if err != nil {
		return fmt.Errorf("finance_repo: failed to update expense: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return finance.ErrExpenseNotFound
	}
	return nil
}

// Delete removes an expense by ID.
func (r *FinanceRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM expenses WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("finance_repo: failed to delete expense: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return finance.ErrExpenseNotFound
	}
	return nil
}

// List queries expenses according to filter options.
func (r *FinanceRepository) List(ctx context.Context, f finance.Filter) ([]*finance.Expense, error) {
	var whereClauses []string
	var args []interface{}

	if f.Category != nil {
		whereClauses = append(whereClauses, "category = ?")
		args = append(args, string(*f.Category))
	}
	if f.Subcategory != nil {
		whereClauses = append(whereClauses, "subcategory = ?")
		args = append(args, string(*f.Subcategory))
	}
	if f.StartDate != nil {
		whereClauses = append(whereClauses, "expense_date >= ?")
		args = append(args, f.StartDate.UTC().Format(time.RFC3339))
	}
	if f.EndDate != nil {
		whereClauses = append(whereClauses, "expense_date <= ?")
		args = append(args, f.EndDate.UTC().Format(time.RFC3339))
	}
	if f.IsFixed != nil {
		whereClauses = append(whereClauses, "is_fixed = ?")
		if *f.IsFixed {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if f.PayerUserID != nil && *f.PayerUserID != "" {
		whereClauses = append(whereClauses, "payer_user_id = ?")
		args = append(args, *f.PayerUserID)
	}

	query := `
		SELECT id, title, amount, currency, category, subcategory,
		       expense_date, is_fixed, payment_method, payer_user_id,
		       notes, recurring_frequency, contract_expiry_date,
		       energy_kwh, gas_smc, food_waste_value, created_at, updated_at
		FROM expenses
	`
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	query += " ORDER BY expense_date DESC"

	if f.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", f.Limit)
		if f.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", f.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("finance_repo: failed to query expenses: %w", err)
	}
	defer rows.Close()

	var result []*finance.Expense
	for rows.Next() {
		exp, err := scanExpense(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, exp)
	}
	return result, rows.Err()
}

// GetSummary calculates aggregated financial metrics, category breakdown, 12-month trends, and budget stats.
func (r *FinanceRepository) GetSummary(ctx context.Context, period string, refDate time.Time) (*finance.FinancialSummary, error) {
	if refDate.IsZero() {
		refDate = time.Now()
	}

	start, end, prevStart, prevEnd, daysInPeriod := calculatePeriodBounds(period, refDate)

	startStr := start.UTC().Format(time.RFC3339)
	endStr := end.UTC().Format(time.RFC3339)
	prevStartStr := prevStart.UTC().Format(time.RFC3339)
	prevEndStr := prevEnd.UTC().Format(time.RFC3339)

	// 1. Total spent in current period
	var currentTotal float64
	err := r.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE expense_date >= ? AND expense_date <= ?", startStr, endStr).Scan(&currentTotal)
	if err != nil {
		return nil, fmt.Errorf("finance_repo: failed to calculate total: %w", err)
	}

	// 2. Total spent in previous period
	var prevTotal float64
	_ = r.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE expense_date >= ? AND expense_date <= ?", prevStartStr, prevEndStr).Scan(&prevTotal)

	// Percentage change
	var pctChange float64
	if prevTotal > 0 {
		pctChange = math.Round(((currentTotal-prevTotal)/prevTotal)*1000) / 10
	}

	// Daily average
	dailyAvg := 0.0
	if daysInPeriod > 0 {
		dailyAvg = math.Round((currentTotal/float64(daysInPeriod))*100) / 100
	}

	// 3. Category Breakdown
	catQuery := `
		SELECT category, COALESCE(SUM(amount), 0), COUNT(*)
		FROM expenses
		WHERE expense_date >= ? AND expense_date <= ?
		GROUP BY category
		ORDER BY 2 DESC
	`
	catRows, err := r.db.QueryContext(ctx, catQuery, startStr, endStr)
	if err != nil {
		return nil, fmt.Errorf("finance_repo: failed to query category breakdown: %w", err)
	}
	defer catRows.Close()

	var categories []finance.CategoryBreakdown
	for catRows.Next() {
		var catStr string
		var amt float64
		var count int
		if err := catRows.Scan(&catStr, &amt, &count); err != nil {
			return nil, err
		}
		pct := 0.0
		if currentTotal > 0 {
			pct = math.Round((amt/currentTotal)*1000) / 10
		}
		categories = append(categories, finance.CategoryBreakdown{
			Category:    finance.Category(catStr),
			Name:        getCategoryDisplayName(finance.Category(catStr)),
			TotalAmount: math.Round(amt*100) / 100,
			Percentage:  pct,
			Count:       count,
		})
	}
	if err := catRows.Err(); err != nil {
		return nil, fmt.Errorf("finance_repo: catRows error: %w", err)
	}

	// 4. Fixed vs Variable Breakdown
	var fixedAmount, varAmount float64
	fvRows, err := r.db.QueryContext(ctx, `
		SELECT is_fixed, COALESCE(SUM(amount), 0)
		FROM expenses
		WHERE expense_date >= ? AND expense_date <= ?
		GROUP BY is_fixed
	`, startStr, endStr)
	if err == nil {
		for fvRows.Next() {
			var isFixed int
			var amt float64
			if fvRows.Scan(&isFixed, &amt) == nil {
				if isFixed == 1 {
					fixedAmount = amt
				} else {
					varAmount = amt
				}
			}
		}
		if err := fvRows.Err(); err != nil {
			fvRows.Close()
			return nil, fmt.Errorf("finance_repo: fvRows error: %w", err)
		}
		fvRows.Close()
	}

	fixedPct := 0.0
	varPct := 0.0
	if currentTotal > 0 {
		fixedPct = math.Round((fixedAmount/currentTotal)*1000) / 10
		varPct = math.Round((varAmount/currentTotal)*1000) / 10
	}

	// 5. Monthly Trends (Last 12 calendar months)
	monthlyTrends := r.calculateLast12MonthsTrend(ctx, refDate)

	// 6. Monthly Budget Targets
	currMonthStr := refDate.Format("2006-01")
	monthlyBudget := 0.0
	var budgetVal float64
	err = r.db.QueryRowContext(ctx, "SELECT monthly_limit FROM budget_targets WHERE category = 'total' AND (month_year = ? OR month_year = 'global') ORDER BY CASE WHEN month_year = ? THEN 0 ELSE 1 END LIMIT 1", currMonthStr, currMonthStr).Scan(&budgetVal)
	if err == nil && budgetVal > 0 {
		monthlyBudget = budgetVal
	}

	remainingBudget := 0.0
	if monthlyBudget > 0 {
		remainingBudget = math.Round((monthlyBudget-currentTotal)*100) / 100
	}

	return &finance.FinancialSummary{
		Period:              period,
		StartDate:           start.Format("2006-01-02"),
		EndDate:             end.Format("2006-01-02"),
		TotalSpent:          math.Round(currentTotal*100) / 100,
		PreviousPeriodTotal: math.Round(prevTotal*100) / 100,
		PercentageChange:    pctChange,
		DailyAverage:        dailyAvg,
		MonthlyBudget:       monthlyBudget,
		RemainingBudget:     remainingBudget,
		Categories:          categories,
		MonthlyTrends:       monthlyTrends,
		FixedVsVariable: finance.FixedVsVariable{
			FixedAmount:        math.Round(fixedAmount*100) / 100,
			FixedPercentage:    fixedPct,
			VariableAmount:     math.Round(varAmount*100) / 100,
			VariablePercentage: varPct,
		},
		Insights: []finance.SavingsInsight{},
	}, nil
}

func (r *FinanceRepository) calculateLast12MonthsTrend(ctx context.Context, refDate time.Time) []finance.MonthlyTrend {
	var trends []finance.MonthlyTrend

	// 12 months starting from 11 months ago to current month
	year := refDate.Year()
	month := refDate.Month()

	for i := 11; i >= 0; i-- {
		mDate := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).AddDate(0, -i, 0)
		ym := mDate.Format("2006-01")
		mName := getItalianMonthName(mDate.Month()) + " " + fmt.Sprintf("%02d", mDate.Year()%100)

		mStart := time.Date(mDate.Year(), mDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		mEnd := mStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

		var total, fixed, variable float64
		query := `
			SELECT 
				COALESCE(SUM(amount), 0),
				COALESCE(SUM(CASE WHEN is_fixed = 1 THEN amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN is_fixed = 0 THEN amount ELSE 0 END), 0)
			FROM expenses
			WHERE expense_date >= ? AND expense_date <= ?
		`
		_ = r.db.QueryRowContext(ctx, query, mStart.Format(time.RFC3339), mEnd.Format(time.RFC3339)).Scan(&total, &fixed, &variable)

		trends = append(trends, finance.MonthlyTrend{
			YearMonth:      ym,
			MonthName:      mName,
			TotalAmount:    math.Round(total*100) / 100,
			FixedAmount:    math.Round(fixed*100) / 100,
			VariableAmount: math.Round(variable*100) / 100,
		})
	}
	return trends
}

// SetBudget inserts or updates a monthly budget target.
func (r *FinanceRepository) SetBudget(ctx context.Context, b *finance.BudgetTarget) error {
	if b.ID == "" {
		b.ID = "bgt_" + uuid.New().String()
	}
	if b.MonthYear == "" {
		b.MonthYear = "global"
	}
	now := time.Now().UTC()
	b.CreatedAt = now
	b.UpdatedAt = now

	query := `
		INSERT INTO budget_targets (id, category, monthly_limit, month_year, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(category, month_year) DO UPDATE SET
			monthly_limit = excluded.monthly_limit,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(ctx, query, b.ID, b.Category, b.MonthlyLimit, b.MonthYear, b.CreatedAt.Format(time.RFC3339), b.UpdatedAt.Format(time.RFC3339))
	return err
}

// GetBudget retrieves a budget target for a category and month.
func (r *FinanceRepository) GetBudget(ctx context.Context, category, monthYear string) (*finance.BudgetTarget, error) {
	query := `
		SELECT id, category, monthly_limit, month_year, created_at, updated_at
		FROM budget_targets
		WHERE category = ? AND (month_year = ? OR month_year = 'global')
		ORDER BY CASE WHEN month_year = ? THEN 0 ELSE 1 END LIMIT 1
	`
	var b finance.BudgetTarget
	var ca, ua string
	err := r.db.QueryRowContext(ctx, query, category, monthYear, monthYear).Scan(&b.ID, &b.Category, &b.MonthlyLimit, &b.MonthYear, &ca, &ua)
	if err != nil {
		return nil, err
	}
	b.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	b.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	return &b, nil
}

// ListBudgets returns all defined budget limits for a given month.
func (r *FinanceRepository) ListBudgets(ctx context.Context, monthYear string) ([]*finance.BudgetTarget, error) {
	query := `
		SELECT id, category, monthly_limit, month_year, created_at, updated_at
		FROM budget_targets
		WHERE month_year = ? OR month_year = 'global'
	`
	rows, err := r.db.QueryContext(ctx, query, monthYear)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*finance.BudgetTarget
	for rows.Next() {
		var b finance.BudgetTarget
		var ca, ua string
		if err := rows.Scan(&b.ID, &b.Category, &b.MonthlyLimit, &b.MonthYear, &ca, &ua); err != nil {
			return nil, err
		}
		b.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		b.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		result = append(result, &b)
	}
	return result, rows.Err()
}

func scanExpense(s scannable) (*finance.Expense, error) {
	var e finance.Expense
	var expDateStr, caStr, uaStr string
	var expiryStr sql.NullString
	var isFixedInt int
	var catStr, subcatStr, freqStr string

	err := s.Scan(
		&e.ID, &e.Title, &e.Amount, &e.Currency, &catStr, &subcatStr,
		&expDateStr, &isFixedInt, &e.PaymentMethod, &e.PayerUserID,
		&e.Notes, &freqStr, &expiryStr,
		&e.EnergyKWh, &e.GasSmc, &e.FoodWasteValue, &caStr, &uaStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, finance.ErrExpenseNotFound
		}
		return nil, err
	}

	e.Category = finance.Category(catStr)
	e.Subcategory = finance.Subcategory(subcatStr)
	e.RecurringFrequency = finance.RecurringFrequency(freqStr)
	e.IsFixed = isFixedInt == 1
	e.ExpenseDate, _ = time.Parse(time.RFC3339, expDateStr)
	e.CreatedAt, _ = time.Parse(time.RFC3339, caStr)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, uaStr)

	if expiryStr.Valid && expiryStr.String != "" {
		if t, err := time.Parse(time.RFC3339, expiryStr.String); err == nil {
			e.ContractExpiryDate = &t
		}
	}
	return &e, nil
}

func calculatePeriodBounds(period string, ref time.Time) (start, end, prevStart, prevEnd time.Time, days int) {
	loc := ref.Location()
	y, m, d := ref.Date()

	switch strings.ToLower(period) {
	case "day":
		start = time.Date(y, m, d, 0, 0, 0, 0, loc)
		end = start.Add(24 * time.Hour).Add(-time.Nanosecond)
		prevStart = start.AddDate(0, 0, -1)
		prevEnd = prevStart.Add(24 * time.Hour).Add(-time.Nanosecond)
		days = 1

	case "week":
		weekday := int(ref.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday -> 7
		}
		start = time.Date(y, m, d, 0, 0, 0, 0, loc).AddDate(0, 0, -(weekday - 1))
		end = start.AddDate(0, 0, 7).Add(-time.Nanosecond)
		prevStart = start.AddDate(0, 0, -7)
		prevEnd = prevStart.AddDate(0, 0, 7).Add(-time.Nanosecond)
		days = 7

	case "year":
		start = time.Date(y, 1, 1, 0, 0, 0, 0, loc)
		end = time.Date(y+1, 1, 1, 0, 0, 0, 0, loc).Add(-time.Nanosecond)
		prevStart = time.Date(y-1, 1, 1, 0, 0, 0, 0, loc)
		prevEnd = start.Add(-time.Nanosecond)
		days = 365

	case "month":
		fallthrough
	default:
		start = time.Date(y, m, 1, 0, 0, 0, 0, loc)
		end = start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		prevStart = start.AddDate(0, -1, 0)
		prevEnd = start.Add(-time.Nanosecond)
		days = int(end.Sub(start).Hours() / 24)
	}
	return
}

func getCategoryDisplayName(cat finance.Category) string {
	switch cat {
	case finance.CategoryFood:
		return "Cibo & Alimentari"
	case finance.CategoryUtilities:
		return "Bollette & Utenze"
	case finance.CategoryTransport:
		return "Carburante & Mobilità"
	case finance.CategoryHomeMaintenance:
		return "Materiali & Casa"
	case finance.CategoryInsurance:
		return "Assicurazioni"
	case finance.CategoryOther:
		return "Altro & Svago"
	default:
		return string(cat)
	}
}

func getItalianMonthName(m time.Month) string {
	switch m {
	case time.January:
		return "Gen"
	case time.February:
		return "Feb"
	case time.March:
		return "Mar"
	case time.April:
		return "Apr"
	case time.May:
		return "Mag"
	case time.June:
		return "Giu"
	case time.July:
		return "Lug"
	case time.August:
		return "Ago"
	case time.September:
		return "Set"
	case time.October:
		return "Ott"
	case time.November:
		return "Nov"
	case time.December:
		return "Dic"
	default:
		return ""
	}
}
