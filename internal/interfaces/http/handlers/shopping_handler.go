package handlers

import (
	"encoding/json"
	"net/http"

	appShop "github.com/domus-hub/domus/internal/application/shopping"
	domainShop "github.com/domus-hub/domus/internal/domain/shopping"
	"github.com/go-chi/chi/v5"
)

type ShoppingHandler struct {
	svc *appShop.Service
}

func NewShoppingHandler(svc *appShop.Service) *ShoppingHandler {
	return &ShoppingHandler{svc: svc}
}

func (h *ShoppingHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListItems(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if items == nil {
		items = []*domainShop.Item{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *ShoppingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string              `json:"name"`
		Quantity    string              `json:"quantity"`
		Category    domainShop.Category `json:"category"`
		Supermarket string              `json:"supermarket"`
		AddedBy     string              `json:"added_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name cannot be empty")
		return
	}

	it, err := h.svc.AddItemWithStore(r.Context(), req.Name, req.Quantity, req.Category, req.Supermarket, req.AddedBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, it)
}

func (h *ShoppingHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		CompletedBy string `json:"completed_by"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	it, err := h.svc.ToggleItem(r.Context(), id, req.CompletedBy)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, it)
}

func (h *ShoppingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteItem(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (h *ShoppingHandler) ClearCompleted(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.ClearCompleted(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"cleared": count})
}
