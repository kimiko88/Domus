package medicine_test

import (
	"context"
	"sync"
	"testing"
	"time"

	appMed "github.com/domus-hub/domus/internal/application/medicine"
	domainMed "github.com/domus-hub/domus/internal/domain/medicine"
)

type mockMedicineRepo struct {
	mu    sync.RWMutex
	items map[string]*domainMed.Medicine
}

func newMockRepo() *mockMedicineRepo {
	return &mockMedicineRepo{
		items: make(map[string]*domainMed.Medicine),
	}
}

func (m *mockMedicineRepo) Create(ctx context.Context, item *domainMed.Medicine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if item.ID == "" {
		item.ID = "med_mock"
	}
	m.items[item.ID] = item
	return nil
}

func (m *mockMedicineRepo) GetByID(ctx context.Context, id string) (*domainMed.Medicine, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.items[id]
	if !ok {
		return nil, domainMed.ErrMedicineNotFound
	}
	return item, nil
}

func (m *mockMedicineRepo) Update(ctx context.Context, item *domainMed.Medicine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[item.ID]; !ok {
		return domainMed.ErrMedicineNotFound
	}
	m.items[item.ID] = item
	return nil
}

func (m *mockMedicineRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return domainMed.ErrMedicineNotFound
	}
	delete(m.items, id)
	return nil
}

func (m *mockMedicineRepo) List(ctx context.Context, filter domainMed.Filter) ([]*domainMed.Medicine, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*domainMed.Medicine
	for _, it := range m.items {
		list = append(list, it)
	}
	return list, nil
}

func TestService_CreateWithAIFAAutoEnrichment(t *testing.T) {
	repo := newMockRepo()
	svc := appMed.NewService(repo, nil)

	// Inserimento rapido fornendo solo il codice AIC letto da barcode
	req := appMed.CreateMedicineRequest{
		AIC:            "034234035", // Tachipirina 1000 mg
		ExpirationDate: "2027-10-31",
	}

	med, err := svc.CreateMedicine(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error creating medicine: %v", err)
	}

	if med.Name != "Tachipirina 1000 mg compresse" {
		t.Errorf("expected auto-enriched name, got: %s", med.Name)
	}
	if med.ActiveIngredient != "Paracetamolo" {
		t.Errorf("expected Paracetamolo, got: %s", med.ActiveIngredient)
	}
	if med.Quantity != 16 {
		t.Errorf("expected default quantity 16, got: %v", med.Quantity)
	}
}

func TestService_OpenMedicineRecalculatesPAO(t *testing.T) {
	repo := newMockRepo()
	svc := appMed.NewService(repo, nil)

	req := appMed.CreateMedicineRequest{
		Name:           "Tobral Collirio 5ml",
		ExpirationDate: "2027-12-31",
		PaoDays:        28, // Dura 28 giorni dopo l'apertura
		Quantity:       1,
		Unit:           domainMed.UnitMl,
	}

	med, err := svc.CreateMedicine(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	// Apriamo il collirio oggi
	openDate := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	updated, err := svc.OpenMedicine(context.Background(), med.ID, &openDate)
	if err != nil {
		t.Fatalf("failed to open medicine: %v", err)
	}

	// 2026-10-18 + 28 giorni = 2026-11-15
	expectedExp := "2026-11-15"
	if updated.EffectiveExpirationDate != expectedExp {
		t.Errorf("expected effective expiration %s after opening, got: %s", expectedExp, updated.EffectiveExpirationDate)
	}
}

func TestService_ConsumeDose(t *testing.T) {
	repo := newMockRepo()
	svc := appMed.NewService(repo, nil)

	req := appMed.CreateMedicineRequest{
		Name:           "Brufen 400 mg",
		ExpirationDate: "2028-05-30",
		Quantity:       30,
		Unit:           domainMed.UnitTablets,
	}

	med, err := svc.CreateMedicine(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	// Consumiamo 2 compresse
	consumed, err := svc.ConsumeDose(context.Background(), med.ID, 2)
	if err != nil {
		t.Fatalf("failed to consume: %v", err)
	}
	if consumed.Quantity != 28 {
		t.Errorf("expected 28 remaining tablets, got: %v", consumed.Quantity)
	}
}

func TestService_ParseCodeAndLookup(t *testing.T) {
	svc := appMed.NewService(newMockRepo(), nil)

	// Test decodifica codice con AIC, scadenza e lotto
	parsed := svc.ParseCode("(01)08003423403501(17)261130(10)LOTABC99")
	if parsed.AIC != "034234035" {
		t.Errorf("expected AIC 034234035, got %s", parsed.AIC)
	}
	if parsed.ExpirationDate != "2026-11-30" {
		t.Errorf("expected ExpirationDate 2026-11-30, got %s", parsed.ExpirationDate)
	}
	if parsed.LotNumber != "LOTABC99" {
		t.Errorf("expected Lot LOTABC99, got %s", parsed.LotNumber)
	}

	// Test lookup AIFA
	results := svc.LookupAIFA("augmentin", "")
	if len(results) == 0 {
		t.Fatalf("expected to find Augmentin in AIFA catalog")
	}
}
