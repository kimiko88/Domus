package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	appMed "github.com/domus-hub/domus/internal/application/medicine"
	domainMed "github.com/domus-hub/domus/internal/domain/medicine"
	"github.com/go-chi/chi/v5"
)

type MedicineHandler struct {
	svc *appMed.Service
}

func NewMedicineHandler(svc *appMed.Service) *MedicineHandler {
	return &MedicineHandler{svc: svc}
}

func (h *MedicineHandler) RegisterRoutes(r chi.Router) {
	r.Route("/medicines", func(mr chi.Router) {
		mr.Get("/", h.List)
		mr.Post("/", h.Create)
		mr.Get("/alerts", h.GetAlerts)
		mr.Post("/parse-code", h.ParseCode)
		mr.Get("/lookup", h.LookupAIFA)

		mr.Route("/{id}", func(idR chi.Router) {
			idR.Get("/", h.Get)
			idR.Put("/", h.Update)
			idR.Delete("/", h.Delete)
			idR.Post("/open", h.Open)
			idR.Post("/consume", h.Consume)
		})
	})
}

func (h *MedicineHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := domainMed.Filter{
		Location: domainMed.Location(q.Get("location")),
		Search:   q.Get("q"),
	}

	if status := q.Get("status"); status != "" {
		switch status {
		case "expired":
			val := true
			filter.IsExpired = &val
		case "expiring_7d":
			days := 7
			filter.ExpiringDays = &days
		case "expiring_30d":
			days := 30
			filter.ExpiringDays = &days
		case "cold_chain":
			val := true
			filter.ColdChain = &val
		case "low_stock":
			val := true
			filter.LowStock = &val
		}
	}

	items, err := h.svc.ListMedicines(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *MedicineHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req appMed.CreateMedicineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	med, err := h.svc.CreateMedicine(r.Context(), req)
	if err != nil {
		if errors.Is(err, domainMed.ErrInvalidMedicineName) ||
			errors.Is(err, domainMed.ErrInvalidExpiration) ||
			errors.Is(err, domainMed.ErrInvalidQuantity) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, med)
}

func (h *MedicineHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	med, err := h.svc.GetMedicine(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainMed.ErrMedicineNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "medicine not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, med)
}

func (h *MedicineHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req appMed.UpdateMedicineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	med, err := h.svc.UpdateMedicine(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, domainMed.ErrMedicineNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "medicine not found")
			return
		}
		if errors.Is(err, domainMed.ErrInvalidMedicineName) ||
			errors.Is(err, domainMed.ErrInvalidExpiration) ||
			errors.Is(err, domainMed.ErrInvalidQuantity) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, med)
}

func (h *MedicineHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteMedicine(r.Context(), id); err != nil {
		if errors.Is(err, domainMed.ErrMedicineNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "medicine not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (h *MedicineHandler) Open(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		OpenedAt *string `json:"opened_at,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	var openedTime *time.Time
	if req.OpenedAt != nil && *req.OpenedAt != "" {
		if t, err := time.Parse("2006-01-02", *req.OpenedAt); err == nil {
			openedTime = &t
		}
	}

	med, err := h.svc.OpenMedicine(r.Context(), id, openedTime)
	if err != nil {
		if errors.Is(err, domainMed.ErrMedicineNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "medicine not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, med)
}

func (h *MedicineHandler) Consume(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Amount float64 `json:"amount"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Amount <= 0 {
		req.Amount = 1.0
	}

	med, err := h.svc.ConsumeDose(r.Context(), id, req.Amount)
	if err != nil {
		if errors.Is(err, domainMed.ErrMedicineNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "medicine not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, med)
}

func (h *MedicineHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	var fridgeTemp *float64
	if tempStr := r.URL.Query().Get("fridge_temp"); tempStr != "" {
		if t, err := strconv.ParseFloat(tempStr, 64); err == nil {
			fridgeTemp = &t
		}
	}

	alerts, err := h.svc.GetAlerts(r.Context(), fridgeTemp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, alerts)
}

func (h *MedicineHandler) ParseCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "code cannot be empty")
		return
	}

	result := h.svc.ParseCode(req.Code)
	writeJSON(w, http.StatusOK, result)
}

func (h *MedicineHandler) LookupAIFA(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	aic := r.URL.Query().Get("aic")

	results := h.svc.LookupAIFA(query, aic)
	writeJSON(w, http.StatusOK, results)
}
