// Package handlers contains the HTTP handlers for pantry inventory, recipes, meal planning, and receipt OCR import.
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"context"
	"strings"

	appPantry "github.com/domus-hub/domus/internal/application/pantry"
	domainPantry "github.com/domus-hub/domus/internal/domain/pantry"
	"github.com/go-chi/chi/v5"
)

type PantryHandler struct {
	svc *appPantry.Service
}

func NewPantryHandler(svc *appPantry.Service) *PantryHandler {
	return &PantryHandler{svc: svc}
}

// RegisterRoutes mounts all pantry, recipe, meal planning, and receipt endpoints.
func (h *PantryHandler) RegisterRoutes(r chi.Router) {
	// 1. Pantry Inventory
	r.Route("/pantry", func(pr chi.Router) {
		pr.Get("/items", h.ListItems)
		pr.Post("/items", h.CreateItem)
		pr.Get("/items/{id}", h.GetItem)
		pr.Put("/items/{id}", h.UpdateItem)
		pr.Delete("/items/{id}", h.DeleteItem)
		pr.Post("/items/{id}/adjust", h.AdjustQuantity)
		pr.Get("/expiring", h.ListExpiring)
		pr.Get("/low-stock", h.ListLowStock)
	})

	// 2. Recipes & Cookable Finder
	r.Route("/recipes", func(rr chi.Router) {
		rr.Get("/", h.ListRecipes)
		rr.Post("/", h.CreateRecipe)
		rr.Get("/cookable", h.FindCookableRecipes)
		rr.Get("/{id}", h.GetRecipe)
		rr.Put("/{id}", h.UpdateRecipe)
		rr.Delete("/{id}", h.DeleteRecipe)
	})

	// 3. Meal Calendar
	r.Route("/meals", func(mr chi.Router) {
		mr.Get("/", h.ListMealPlans)
		mr.Post("/", h.CreateMealPlan)
		mr.Get("/{id}", h.GetMealPlan)
		mr.Put("/{id}", h.UpdateMealPlan)
		mr.Delete("/{id}", h.DeleteMealPlan)
		mr.Post("/{id}/consume", h.ConsumeMealPlan)
	})

	// 4. Receipt OCR & Smart Import
	r.Route("/receipts", func(rc chi.Router) {
		rc.Post("/parse", h.ParseReceipt)
		rc.Post("/parse-pdf", h.ParseReceiptPDF)
		rc.Post("/import", h.ImportReceipt)
	})
}

// --- PANTRY ITEMS ---

func (h *PantryHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	category := domainPantry.Category(r.URL.Query().Get("category"))
	location := domainPantry.Location(r.URL.Query().Get("location"))

	items, err := h.svc.ListPantryItems(r.Context(), category, location)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if items == nil {
		items = []domainPantry.PantryItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *PantryHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var req appPantry.CreatePantryItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	item, err := h.svc.CreatePantryItem(r.Context(), req)
	if err != nil {
		if errors.Is(err, domainPantry.ErrInvalidPantryName) || errors.Is(err, domainPantry.ErrInvalidQuantity) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *PantryHandler) GetItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.svc.GetPantryItem(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainPantry.ErrItemNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "pantry item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *PantryHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req appPantry.CreatePantryItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	item, err := h.svc.UpdatePantryItem(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, domainPantry.ErrItemNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "pantry item not found")
			return
		}
		if errors.Is(err, domainPantry.ErrInvalidPantryName) || errors.Is(err, domainPantry.ErrInvalidQuantity) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *PantryHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeletePantryItem(r.Context(), id); err != nil {
		if errors.Is(err, domainPantry.ErrItemNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "pantry item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (h *PantryHandler) AdjustQuantity(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Delta float64 `json:"delta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	item, err := h.svc.AdjustQuantity(r.Context(), id, req.Delta)
	if err != nil {
		if errors.Is(err, domainPantry.ErrItemNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "pantry item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *PantryHandler) ListExpiring(w http.ResponseWriter, r *http.Request) {
	days := 3
	if dStr := r.URL.Query().Get("days"); dStr != "" {
		if d, err := strconv.Atoi(dStr); err == nil && d > 0 {
			days = d
		}
	}

	items, err := h.svc.ListExpiringItems(r.Context(), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if items == nil {
		items = []domainPantry.PantryItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *PantryHandler) ListLowStock(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListLowStockItems(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if items == nil {
		items = []domainPantry.PantryItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

// --- RECIPES ---

func (h *PantryHandler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	category := domainPantry.RecipeCategory(r.URL.Query().Get("category"))
	recipes, err := h.svc.ListRecipes(r.Context(), category)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if recipes == nil {
		recipes = []domainPantry.Recipe{}
	}
	writeJSON(w, http.StatusOK, recipes)
}

func (h *PantryHandler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	var req appPantry.CreateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	recipe, err := h.svc.CreateRecipe(r.Context(), req)
	if err != nil {
		if errors.Is(err, domainPantry.ErrInvalidRecipeTitle) || errors.Is(err, domainPantry.ErrInvalidPortions) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, recipe)
}

func (h *PantryHandler) GetRecipe(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	recipe, err := h.svc.GetRecipe(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainPantry.ErrRecipeNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "recipe not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, recipe)
}

func (h *PantryHandler) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req appPantry.CreateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	recipe, err := h.svc.UpdateRecipe(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, domainPantry.ErrRecipeNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "recipe not found")
			return
		}
		if errors.Is(err, domainPantry.ErrInvalidRecipeTitle) || errors.Is(err, domainPantry.ErrInvalidPortions) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, recipe)
}

func (h *PantryHandler) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteRecipe(r.Context(), id); err != nil {
		if errors.Is(err, domainPantry.ErrRecipeNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "recipe not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (h *PantryHandler) FindCookableRecipes(w http.ResponseWriter, r *http.Request) {
	cookable, err := h.svc.FindCookableRecipes(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if cookable == nil {
		cookable = []appPantry.CookableRecipeResult{}
	}
	writeJSON(w, http.StatusOK, cookable)
}

// --- MEAL PLANS ---

func (h *PantryHandler) ListMealPlans(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	plans, err := h.svc.ListMealPlans(r.Context(), startDate, endDate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if plans == nil {
		plans = []domainPantry.MealPlan{}
	}
	writeJSON(w, http.StatusOK, plans)
}

func (h *PantryHandler) CreateMealPlan(w http.ResponseWriter, r *http.Request) {
	var req appPantry.CreateMealPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	plan, err := h.svc.CreateMealPlan(r.Context(), req)
	if err != nil {
		if errors.Is(err, domainPantry.ErrInvalidMealDate) || errors.Is(err, domainPantry.ErrInvalidPersonsCount) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, plan)
}

func (h *PantryHandler) GetMealPlan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	plan, err := h.svc.GetMealPlan(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainPantry.ErrMealPlanNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "meal plan not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (h *PantryHandler) UpdateMealPlan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req appPantry.CreateMealPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	plan, err := h.svc.UpdateMealPlan(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, domainPantry.ErrMealPlanNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "meal plan not found")
			return
		}
		if errors.Is(err, domainPantry.ErrInvalidMealDate) || errors.Is(err, domainPantry.ErrInvalidPersonsCount) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (h *PantryHandler) DeleteMealPlan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteMealPlan(r.Context(), id); err != nil {
		if errors.Is(err, domainPantry.ErrMealPlanNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "meal plan not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

// ConsumeMealPlan marks the meal as consumed and proportionally decreases pantry ingredient stocks.
func (h *PantryHandler) ConsumeMealPlan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	plan, err := h.svc.ConsumeMealPlan(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainPantry.ErrMealPlanNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "meal plan not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

// --- RECEIPT OCR & IMPORT ---

func (h *PantryHandler) ParseReceipt(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req struct {
		RawText string `json:"raw_text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	receipt, err := h.svc.ParseReceipt(r.Context(), req.RawText)
	if err != nil {
		writeError(w, http.StatusBadRequest, "PARSE_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

// ParseReceiptPDF extracts selectable text from a bounded PDF and reuses the receipt parser.
func (h *PantryHandler) ParseReceiptPDF(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxReceiptPDFBytes)
	pdf, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "PDF_TOO_LARGE", "PDF must be 10 MB or smaller")
		return
	}
	rawText, err := extractPDFText(r.Context(), pdf)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, context.DeadlineExceeded) { status = http.StatusRequestTimeout }
		if strings.Contains(err.Error(), "unavailable") { status = http.StatusServiceUnavailable }
		if errors.Is(err, errPDFTextTooLarge) { status = http.StatusRequestEntityTooLarge }
		writeError(w, status, "PDF_EXTRACTION_ERROR", err.Error())
		return
	}
	receipt, err := h.svc.ParseReceipt(r.Context(), rawText)
	if err != nil {
		writeError(w, http.StatusBadRequest, "PARSE_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

func (h *PantryHandler) ImportReceipt(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 256*1024)
	var req appPantry.ImportReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	result, err := h.svc.ImportReceipt(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
