package medicine_test

import (
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/medicine"
)

func TestMedicine_ValidationAndPAO(t *testing.T) {
	opened := "2026-10-01"
	med := &medicine.Medicine{
		ID:             "med-1",
		Name:           "Tobral Collirio 5 ml",
		ExpirationDate: "2027-06-30",
		PaoDays:        28, // 28 giorni dopo apertura
		OpenedAt:       &opened,
		Location:       medicine.LocationFridge,
		Quantity:       1,
		Unit:           medicine.UnitMl,
		MinThreshold:   0,
	}

	if err := med.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	// 2026-10-01 + 28 days = 2026-10-29
	expectedEff := "2026-10-29"
	if med.EffectiveExpirationDate != expectedEff {
		t.Errorf("expected effective expiration %s, got %s", expectedEff, med.EffectiveExpirationDate)
	}
}

func TestMedicine_PAODoesNotExceedBoxExpiration(t *testing.T) {
	opened := "2026-10-01"
	med := &medicine.Medicine{
		ID:             "med-2",
		Name:           "Sciroppo",
		ExpirationDate: "2026-10-15", // Scadenza scatola molto vicina!
		PaoDays:        90,           // PAO darebbe gennaio 2027, ma la scatola scade prima!
		OpenedAt:       &opened,
		Quantity:       1,
		Unit:           medicine.UnitMl,
	}

	if err := med.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Deve vincere la data di scadenza della confezione
	if med.EffectiveExpirationDate != "2026-10-15" {
		t.Errorf("expected effective expiration to be 2026-10-15, got %s", med.EffectiveExpirationDate)
	}
}

func TestParseDatamatrix_VariousFormats(t *testing.T) {
	// 1. Plain AIC
	res1 := medicine.ParseDatamatrix("034234035")
	if res1.AIC != "034234035" {
		t.Errorf("expected AIC 034234035, got %s", res1.AIC)
	}

	// 2. GS1 with parentheses: (01)08003423403501(17)270430(10)LOT9988
	res2 := medicine.ParseDatamatrix("(01)08003423403501(17)270430(10)LOT9988")
	if res2.AIC != "034234035" {
		t.Errorf("expected AIC 034234035, got %s", res2.AIC)
	}
	if res2.ExpirationDate != "2027-04-30" {
		t.Errorf("expected ExpirationDate 2027-04-30, got %s", res2.ExpirationDate)
	}
	if res2.LotNumber != "LOT9988" {
		t.Errorf("expected Lot LOT9988, got %s", res2.LotNumber)
	}

	// 3. GS1 continuous stream: 01080034234035011726053110BATCH777
	res3 := medicine.ParseDatamatrix("01080034234035011726053110BATCH777")
	if res3.AIC != "034234035" {
		t.Errorf("expected AIC 034234035, got %s", res3.AIC)
	}
	if res3.ExpirationDate != "2026-05-31" {
		t.Errorf("expected ExpirationDate 2026-05-31, got %s", res3.ExpirationDate)
	}
	if res3.LotNumber != "BATCH777" {
		t.Errorf("expected Lot BATCH777, got %s", res3.LotNumber)
	}

	// 4. GS1 with 00 for day (meaning end of month): (17)270200 -> 2027-02-28
	res4 := medicine.ParseDatamatrix("(17)270200")
	if res4.ExpirationDate != "2027-02-28" {
		t.Errorf("expected 2027-02-28 for day 00, got %s", res4.ExpirationDate)
	}
}

func TestAIFACatalog_LookupAndSearch(t *testing.T) {
	// Exact AIC lookup
	rec, found := medicine.LookupByAIC("034234035")
	if !found {
		t.Fatalf("expected to find Tachipirina 1000 mg by AIC 034234035")
	}
	if rec.ActiveIngredient != "Paracetamolo" {
		t.Errorf("expected Paracetamolo, got %s", rec.ActiveIngredient)
	}

	// Search by name
	results := medicine.SearchAIFA("brufen")
	if len(results) == 0 {
		t.Fatalf("expected to find Brufen in search results")
	}
	if results[0].ActiveIngredient != "Ibuprofene" {
		t.Errorf("expected Ibuprofene, got %s", results[0].ActiveIngredient)
	}

	// Search cold chain items
	tobral := medicine.SearchAIFA("tobral")
	if len(tobral) == 0 || !tobral[0].RequiresColdChain {
		t.Errorf("expected Tobral to require cold chain")
	}
}

func TestComputeAlerts(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

	opened := "2026-10-10"
	items := []*medicine.Medicine{
		{
			ID:                      "m-expired",
			Name:                    "Sciroppo Scaduto",
			ExpirationDate:          "2026-09-01",
			EffectiveExpirationDate: "2026-09-01",
			Quantity:                5,
			Location:                medicine.LocationBathroomCabinet,
		},
		{
			ID:                      "m-7d",
			Name:                    "Collirio in Scadenza 7d",
			ExpirationDate:          "2027-10-01",
			PaoDays:                 15,
			OpenedAt:                &opened, // Opened 2026-10-10 + 15d = 2026-10-25 (7 days from now)
			EffectiveExpirationDate: "2026-10-25",
			Quantity:                1,
			Location:                medicine.LocationFridge,
		},
		{
			ID:                      "m-lowstock",
			Name:                    "Tachipirina Scorta Bassa",
			ExpirationDate:          "2028-10-01",
			EffectiveExpirationDate: "2028-10-01",
			Quantity:                2,
			MinThreshold:            4,
			Location:                medicine.LocationBathroomCabinet,
		},
		{
			ID:                      "m-coldchain",
			Name:                    "Insulina Humalog",
			ExpirationDate:          "2028-10-01",
			EffectiveExpirationDate: "2028-10-01",
			RequiresColdChain:       true,
			Location:                medicine.LocationFridge,
			Quantity:                5,
		},
	}

	// Test con frigo a 9.5°C (supera gli 8.0°C!)
	fridgeTemp := 9.5
	alerts := medicine.ComputeAlerts(items, now, &fridgeTemp)

	hasExpired := false
	has7d := false
	hasLowStock := false
	hasColdChain := false

	for _, a := range alerts {
		if a.Type == medicine.AlertExpired {
			hasExpired = true
		}
		if a.Type == medicine.AlertExpiringSoon7D {
			has7d = true
		}
		if a.Type == medicine.AlertLowStock {
			hasLowStock = true
		}
		if a.Type == medicine.AlertColdChainBreached {
			hasColdChain = true
		}
	}

	if !hasExpired {
		t.Errorf("expected expired alert")
	}
	if !has7d {
		t.Errorf("expected 7d expiring alert")
	}
	if !hasLowStock {
		t.Errorf("expected low stock alert")
	}
	if !hasColdChain {
		t.Errorf("expected cold chain alert when fridge temp is 9.5C")
	}
}
