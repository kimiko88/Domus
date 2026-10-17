// Package handlers implements HTTP REST endpoints for the Domus Home Finance module.
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	appFinance "github.com/domus-hub/domus/internal/application/finance"
	domainFinance "github.com/domus-hub/domus/internal/domain/finance"
	"github.com/go-chi/chi/v5"
)

// FinanceHandler handles HTTP requests for expenses, summaries, budgets, and savings insights.
type FinanceHandler struct {
	svc *appFinance.Service
}

// NewFinanceHandler creates a new FinanceHandler instance.
func NewFinanceHandler(svc *appFinance.Service) *FinanceHandler {
	return &FinanceHandler{svc: svc}
}

// List returns a list of expenses matching query filters.
func (h *FinanceHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var filter domainFinance.Filter

	if cat := q.Get("category"); cat != "" {
		c := domainFinance.Category(cat)
		filter.Category = &c
	}
	if subcat := q.Get("subcategory"); subcat != "" {
		sc := domainFinance.Subcategory(subcat)
		filter.Subcategory = &sc
	}
	if from := q.Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.StartDate = &t
		} else if t, err := time.Parse("2006-01-02", from); err == nil {
			filter.StartDate = &t
		}
	}
	if to := q.Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.EndDate = &t
		} else if t, err := time.Parse("2006-01-02", to); err == nil {
			filter.EndDate = &t
		}
	}
	if isFixed := q.Get("is_fixed"); isFixed != "" {
		b := isFixed == "true" || isFixed == "1"
		filter.IsFixed = &b
	}
	if payer := q.Get("payer_user_id"); payer != "" {
		filter.PayerUserID = &payer
	}
	if lim := q.Get("limit"); lim != "" {
		if l, err := strconv.Atoi(lim); err == nil && l > 0 {
			filter.Limit = l
		}
	}
	if off := q.Get("offset"); off != "" {
		if o, err := strconv.Atoi(off); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	expenses, err := h.svc.ListExpenses(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if expenses == nil {
		expenses = []*domainFinance.Expense{}
	}
	writeJSON(w, http.StatusOK, expenses)
}

// Create handles creation of a new expense.
func (h *FinanceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var exp domainFinance.Expense
	if err := json.NewDecoder(r.Body).Decode(&exp); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if err := h.svc.CreateExpense(r.Context(), &exp); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, exp)
}

// GetByID returns a single expense by ID.
func (h *FinanceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	exp, err := h.svc.GetExpenseByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Expense not found")
		return
	}
	writeJSON(w, http.StatusOK, exp)
}

// Update updates an existing expense.
func (h *FinanceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var exp domainFinance.Expense
	if err := json.NewDecoder(r.Body).Decode(&exp); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	exp.ID = id

	if err := h.svc.UpdateExpense(r.Context(), &exp); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, exp)
}

// Delete removes an expense.
func (h *FinanceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteExpense(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Expense not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

// GetSummary returns KPIs, category breakdown, 12-month trends, and smart savings insights.
func (h *FinanceHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	period := q.Get("period")
	if period == "" {
		period = "month"
	}

	refDate := time.Now()
	if d := q.Get("date"); d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			refDate = t
		} else if t, err := time.Parse(time.RFC3339, d); err == nil {
			refDate = t
		}
	}

	summary, err := h.svc.GetSummary(r.Context(), period, refDate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// SetBudget sets or updates monthly budget targets.
func (h *FinanceHandler) SetBudget(w http.ResponseWriter, r *http.Request) {
	var b domainFinance.BudgetTarget
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if err := h.svc.SetBudget(r.Context(), &b); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, b)
}

// ListBudgets returns the list of budgets for a given month.
func (h *FinanceHandler) ListBudgets(w http.ResponseWriter, r *http.Request) {
	monthYear := r.URL.Query().Get("month_year")
	budgets, err := h.svc.ListBudgets(r.Context(), monthYear)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if budgets == nil {
		budgets = []*domainFinance.BudgetTarget{}
	}
	writeJSON(w, http.StatusOK, budgets)
}

// GetInsights returns automated smart savings tips.
func (h *FinanceHandler) GetInsights(w http.ResponseWriter, r *http.Request) {
	insights, err := h.svc.GenerateSavingsInsights(r.Context(), time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if insights == nil {
		insights = []domainFinance.SavingsInsight{}
	}
	writeJSON(w, http.StatusOK, insights)
}
