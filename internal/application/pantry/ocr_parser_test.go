package pantry_test

import (
	"testing"

	appPantry "github.com/domus-hub/domus/internal/application/pantry"
	domainPantry "github.com/domus-hub/domus/internal/domain/pantry"
)

func TestReceiptParserEsselunga(t *testing.T) {
	rawReceipt := `
ESSELUNGA S.P.A.
VIA PISACANE 12 - MILANO
DATA 15/10/2026 ORA 18:42 SCONTRINO 042
P.IVA 01234567890

PASTA PENNE RIGATE 500G 1.29
LATTE PARZ SCREM 1L 1.49
MELE GOLDEN KG 1.500 2.99
MACINATO SCELTO MANZO 450G 4.90
ACQUA NATURALE 6X1.5L 2.10
BISCOTTI GOCCIOLE 500G 2.80

SUBTOTALE EURO 15.57
TOTALE COMPLESSIVO EURO 15.57
PAGAMENTO CARTA CREDITO 15.57
GRAZIE E ARRIVEDERCI
`

	parser := appPantry.NewReceiptParser()
	res, err := parser.ParseText(rawReceipt)
	if err != nil {
		t.Fatalf("unexpected error parsing receipt: %v", err)
	}

	if res.StoreName != "Esselunga" {
		t.Errorf("expected StoreName 'Esselunga', got '%s'", res.StoreName)
	}

	if res.Date != "2026-10-15" {
		t.Errorf("expected Date '2026-10-15', got '%s'", res.Date)
	}

	if res.TotalAmount != 15.57 {
		t.Errorf("expected TotalAmount 15.57, got %.2f", res.TotalAmount)
	}

	if len(res.Items) != 6 {
		t.Fatalf("expected 6 parsed items, got %d", len(res.Items))
	}

	// Verify Pasta
	item0 := res.Items[0]
	if item0.Category != domainPantry.CategoryPantry {
		t.Errorf("expected Pasta category 'pantry', got '%s'", item0.Category)
	}
	if item0.Quantity != 500 || item0.Unit != domainPantry.UnitGram {
		t.Errorf("expected 500g pasta, got %.1f %s", item0.Quantity, item0.Unit)
	}

	// Verify Latte
	item1 := res.Items[1]
	if item1.Category != domainPantry.CategoryDairy {
		t.Errorf("expected Latte category 'dairy', got '%s'", item1.Category)
	}
	if item1.Quantity != 1 || item1.Unit != domainPantry.UnitLiter {
		t.Errorf("expected 1l latte, got %.1f %s", item1.Quantity, item1.Unit)
	}

	// Verify Mele (kg)
	item2 := res.Items[2]
	if item2.Category != domainPantry.CategoryProduce {
		t.Errorf("expected Mele category 'produce', got '%s'", item2.Category)
	}
	if item2.Quantity != 1.5 || item2.Unit != domainPantry.UnitKilogram {
		t.Errorf("expected 1.5kg mele, got %.1f %s", item2.Quantity, item2.Unit)
	}

	// Verify Acqua (6x1.5L = 9L)
	item4 := res.Items[4]
	if item4.Category != domainPantry.CategoryDrinks {
		t.Errorf("expected Acqua category 'drinks', got '%s'", item4.Category)
	}
	if item4.Quantity != 9 || item4.Unit != domainPantry.UnitLiter {
		t.Errorf("expected 9l water (6x1.5), got %.1f %s", item4.Quantity, item4.Unit)
	}
}

func TestReceiptParserCoopFallbackTotal(t *testing.T) {
	rawReceipt := `
COOP LOMBARDIA
18/10/2026

PANE CASERECCIO 1.60
YOGURT GRECO 2PZ 2.20
PASSATA POMODORO 700G 0.99
`

	parser := appPantry.NewReceiptParser()
	res, err := parser.ParseText(rawReceipt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.StoreName != "Coop" {
		t.Errorf("expected Coop, got %s", res.StoreName)
	}
	if res.Date != "2026-10-18" {
		t.Errorf("expected 2026-10-18, got %s", res.Date)
	}

	// Calculated total = 1.60 + 2.20 + 0.99 = 4.79
	if res.TotalAmount != 4.79 {
		t.Errorf("expected sum total 4.79, got %.2f", res.TotalAmount)
	}

	if len(res.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(res.Items))
	}
}
