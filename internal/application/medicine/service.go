// Package medicine provides application services and orchestrators for medicine cabinet management,
// PAO calculation, stock tracking, and smart expiration alarms.
package medicine

import (
	"context"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	domain "github.com/domus-hub/domus/internal/domain/medicine"
)

// Service coordinates medicine cabinet business use cases.
type Service struct {
	repo domain.Repository
	bus  events.Bus
}

// NewService creates a new medicine application service.
func NewService(repo domain.Repository, bus events.Bus) *Service {
	return &Service{
		repo: repo,
		bus:  bus,
	}
}

// CreateMedicineRequest payload for registering a medicine.
type CreateMedicineRequest struct {
	Name                 string          `json:"name"`
	ActiveIngredient     string          `json:"active_ingredient,omitempty"`
	Dosage               string          `json:"dosage,omitempty"`
	Form                 string          `json:"form,omitempty"`
	AIC                  string          `json:"aic,omitempty"`
	Barcode              string          `json:"barcode,omitempty"`
	LotNumber            string          `json:"lot_number,omitempty"`
	ExpirationDate       string          `json:"expiration_date"`
	PaoDays              int             `json:"pao_days"`
	OpenedAt             *string         `json:"opened_at,omitempty"`
	Location             domain.Location `json:"location"`
	RequiresColdChain    bool            `json:"requires_cold_chain"`
	Quantity             float64         `json:"quantity"`
	Unit                 domain.Unit     `json:"unit"`
	MinThreshold         float64         `json:"min_threshold"`
	PrescriptionRequired bool            `json:"prescription_required"`
	Notes                string          `json:"notes,omitempty"`
}

// UpdateMedicineRequest payload for updating an existing item.
type UpdateMedicineRequest struct {
	Name                 string          `json:"name"`
	ActiveIngredient     string          `json:"active_ingredient,omitempty"`
	Dosage               string          `json:"dosage,omitempty"`
	Form                 string          `json:"form,omitempty"`
	AIC                  string          `json:"aic,omitempty"`
	Barcode              string          `json:"barcode,omitempty"`
	LotNumber            string          `json:"lot_number,omitempty"`
	ExpirationDate       string          `json:"expiration_date"`
	PaoDays              int             `json:"pao_days"`
	OpenedAt             *string         `json:"opened_at,omitempty"`
	Location             domain.Location `json:"location"`
	RequiresColdChain    bool            `json:"requires_cold_chain"`
	Quantity             float64         `json:"quantity"`
	Unit                 domain.Unit     `json:"unit"`
	MinThreshold         float64         `json:"min_threshold"`
	PrescriptionRequired bool            `json:"prescription_required"`
	Notes                string          `json:"notes,omitempty"`
}

func (s *Service) CreateMedicine(ctx context.Context, req CreateMedicineRequest) (*domain.Medicine, error) {
	// Se fornito AIC o Barcode, ma mancanti campi descrittivi, arricchimento automatico tramite catalogo AIFA
	if req.AIC != "" && req.ActiveIngredient == "" {
		if rec, found := domain.LookupByAIC(req.AIC); found {
			if req.Name == "" {
				req.Name = rec.Name
			}
			req.ActiveIngredient = rec.ActiveIngredient
			if req.Dosage == "" {
				req.Dosage = rec.Dosage
			}
			if req.Form == "" {
				req.Form = rec.Form
			}
			if req.Unit == "" {
				req.Unit = rec.Unit
			}
			if req.Quantity <= 0 && rec.DefaultQuantity > 0 {
				req.Quantity = rec.DefaultQuantity
			}
			if rec.RequiresColdChain {
				req.RequiresColdChain = true
				if req.Location == "" || req.Location == domain.LocationBathroomCabinet {
					req.Location = domain.LocationFridge
				}
			}
			if rec.DefaultPaoDays > 0 && req.PaoDays == 0 {
				req.PaoDays = rec.DefaultPaoDays
			}
			req.PrescriptionRequired = rec.PrescriptionReq
		}
	}

	med := &domain.Medicine{
		Name:                 req.Name,
		ActiveIngredient:     req.ActiveIngredient,
		Dosage:               req.Dosage,
		Form:                 req.Form,
		AIC:                  req.AIC,
		Barcode:              req.Barcode,
		LotNumber:            req.LotNumber,
		ExpirationDate:       req.ExpirationDate,
		PaoDays:              req.PaoDays,
		OpenedAt:             req.OpenedAt,
		Location:             req.Location,
		RequiresColdChain:    req.RequiresColdChain,
		Quantity:             req.Quantity,
		Unit:                 req.Unit,
		MinThreshold:         req.MinThreshold,
		PrescriptionRequired: req.PrescriptionRequired,
		Notes:                req.Notes,
	}

	if err := med.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, med); err != nil {
		return nil, fmt.Errorf("medicine service: failed to create: %w", err)
	}

	return med, nil
}

func (s *Service) UpdateMedicine(ctx context.Context, id string, req UpdateMedicineRequest) (*domain.Medicine, error) {
	med, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	med.Name = req.Name
	med.ActiveIngredient = req.ActiveIngredient
	med.Dosage = req.Dosage
	med.Form = req.Form
	med.AIC = req.AIC
	med.Barcode = req.Barcode
	med.LotNumber = req.LotNumber
	med.ExpirationDate = req.ExpirationDate
	med.PaoDays = req.PaoDays
	med.OpenedAt = req.OpenedAt
	med.Location = req.Location
	med.RequiresColdChain = req.RequiresColdChain
	med.Quantity = req.Quantity
	med.Unit = req.Unit
	med.MinThreshold = req.MinThreshold
	med.PrescriptionRequired = req.PrescriptionRequired
	med.Notes = req.Notes

	if err := med.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, med); err != nil {
		return nil, err
	}
	return med, nil
}

func (s *Service) DeleteMedicine(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) GetMedicine(ctx context.Context, id string) (*domain.Medicine, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListMedicines(ctx context.Context, filter domain.Filter) ([]*domain.Medicine, error) {
	return s.repo.List(ctx, filter)
}

// OpenMedicine marks a medicine as opened on a specific date (or today), recalculating PAO expiration.
func (s *Service) OpenMedicine(ctx context.Context, id string, openedAt *time.Time) (*domain.Medicine, error) {
	med, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var dStr string
	if openedAt != nil {
		dStr = openedAt.Format("2006-01-02")
	} else {
		dStr = time.Now().UTC().Format("2006-01-02")
	}
	med.OpenedAt = &dStr

	if err := med.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, med); err != nil {
		return nil, err
	}
	return med, nil
}

// ConsumeDose decrements quantity of the medicine (e.g. 1 tablet, 5 ml).
func (s *Service) ConsumeDose(ctx context.Context, id string, amount float64) (*domain.Medicine, error) {
	if amount <= 0 {
		amount = 1.0
	}

	med, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	med.Quantity -= amount
	if med.Quantity < 0 {
		med.Quantity = 0
	}

	if err := med.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, med); err != nil {
		return nil, err
	}
	return med, nil
}

// ParseCode parses a raw scanned Datamatrix GS1 or barcode string.
func (s *Service) ParseCode(code string) *domain.DatamatrixParseResult {
	res := domain.ParseDatamatrix(code)

	// Se abbiamo trovato l'AIC, possiamo completare subito i dati dal catalogo AIFA
	if res.AIC != "" {
		if rec, found := domain.LookupByAIC(res.AIC); found {
			if res.GTIN == "" {
				res.GTIN = rec.AIC
			}
		}
	}
	return res
}

// LookupAIFA searches or retrieves metadata from the local AIFA catalog.
func (s *Service) LookupAIFA(query string, aic string) []domain.AIFARecord {
	if aic != "" {
		if rec, found := domain.LookupByAIC(aic); found {
			return []domain.AIFARecord{*rec}
		}
	}
	return domain.SearchAIFA(query)
}

// GetAlerts returns critical notifications for medicines: expired, expiring soon, low stock, or fridge cold chain issues.
func (s *Service) GetAlerts(ctx context.Context, fridgeTemp *float64) ([]domain.MedicineAlert, error) {
	all, err := s.repo.List(ctx, domain.Filter{})
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return domain.ComputeAlerts(all, now, fridgeTemp), nil
}
