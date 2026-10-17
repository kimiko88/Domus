// Package pantry provides application services for household food inventory,
// recipe management, automated meal planning consumption, and smart receipt OCR parsing.
package pantry

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	domain "github.com/domus-hub/domus/internal/domain/pantry"
)

// ReceiptParser analyzes raw receipt text (from local OCR or photo upload)
// and extracts structured items, quantities, categories, and totals.
type ReceiptParser struct{}

// NewReceiptParser instantiates the smart receipt parser.
func NewReceiptParser() *ReceiptParser {
	return &ReceiptParser{}
}

var (
	storePatterns = []struct {
		pattern *regexp.Regexp
		name    string
	}{
		{regexp.MustCompile(`(?i)\besselunga\b`), "Esselunga"},
		{regexp.MustCompile(`(?i)\bcoop\b`), "Coop"},
		{regexp.MustCompile(`(?i)\bconad\b`), "Conad"},
		{regexp.MustCompile(`(?i)\blidl\b`), "Lidl"},
		{regexp.MustCompile(`(?i)\bcarrefour\b`), "Carrefour"},
		{regexp.MustCompile(`(?i)\beurospin\b`), "Eurospin"},
		{regexp.MustCompile(`(?i)\biper\b`), "Iper"},
		{regexp.MustCompile(`(?i)\bpam\b`), "Pam"},
		{regexp.MustCompile(`(?i)\bdespar\b`), "Despar"},
		{regexp.MustCompile(`(?i)\btigros\b`), "Tigros"},
		{regexp.MustCompile(`(?i)\baldi\b`), "Aldi"},
		{regexp.MustCompile(`(?i)\bpenny\b`), "Penny Market"},
		{regexp.MustCompile(`(?i)\bmd\s+spa\b`), "MD"},
	}

	dateRegex = regexp.MustCompile(`\b(\d{1,2})[/.-](\d{1,2})[/.-](\d{2,4})\b`)

	// Matches price patterns like 1,29, 12.50, 4.99€, etc.
	priceRegex = regexp.MustCompile(`(?i)(\d+[.,]\d{2})\s*(?:€|eur)?$`)

	// Quantity patterns
	weightGramRegex = regexp.MustCompile(`(?i)\b(\d+)\s*(?:g|gr)\b`)
	weightKgRegex   = regexp.MustCompile(`(?i)(?:\b(\d+(?:[.,]\d+)?)\s*(?:kg|chili)\b|\b(?:kg|chili)\s*(\d+(?:[.,]\d+)?)\b)`)
	volLiterRegex   = regexp.MustCompile(`(?i)(?:\b(\d+(?:[.,]\d+)?)\s*(?:l|lt|litri)\b|\b(?:l|lt|litri)\s*(\d+(?:[.,]\d+)?)\b)`)
	volMlRegex      = regexp.MustCompile(`(?i)\b(\d+)\s*(?:ml)\b`)
	packMultiRegex  = regexp.MustCompile(`(?i)\b(\d+)\s*x\s*(\d+(?:[.,]\d+)?)\s*(l|kg|g|ml)\b`)
	pieceCountRegex = regexp.MustCompile(`(?i)(?:\b(\d+)\s*(?:pz|pezzi|conf)\b|\b(?:pz|pezzi|conf)\s*(\d+)\b)`)

	totalLineRegex = regexp.MustCompile(`(?i)(?:totale\s*complessivo|totale\s*euro|totale|importo\s*pagato|euro|tot)\s*[:=]?\s*(\d+[.,]\d{2})`)

	skipLineRegex = regexp.MustCompile(`(?i)(?:scontrino|fiscale|documento\s*commerciale|p\.?\s*iva|partita\s*iva|cassa|operatore|subtotale|resto|contanti|carta|bancomat|pos|grazie|arrivederci|via\b|tel\b|codice|punti|sconto)`)
)

// ParseText analyzes receipt plain text and extracts a structured Receipt.
func (p *ReceiptParser) ParseText(text string) (*domain.Receipt, error) {
	lines := strings.Split(text, "\n")
	receipt := &domain.Receipt{
		StoreName: "Supermercato",
		Date:      time.Now().Format("2006-01-02"),
		Items:     make([]domain.ReceiptItem, 0),
		RawText:   text,
	}

	// 1. Detect store name from top lines
	for i := 0; i < len(lines) && i < 8; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		foundStore := false
		for _, sp := range storePatterns {
			if sp.pattern.MatchString(line) {
				receipt.StoreName = sp.name
				foundStore = true
				break
			}
		}
		if foundStore {
			break
		}
	}

	// 2. Detect Date
	for _, line := range lines {
		if m := dateRegex.FindStringSubmatch(line); len(m) == 4 {
			day, _ := strconv.Atoi(m[1])
			month, _ := strconv.Atoi(m[2])
			year, _ := strconv.Atoi(m[3])
			if year < 100 {
				year += 2000
			}
			if day >= 1 && day <= 31 && month >= 1 && month <= 12 && year >= 2000 {
				receipt.Date = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
				break
			}
		}
	}

	// 3. Scan lines for products and total
	var totalFromExplicitLine float64
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		// Check for Total line
		if tm := totalLineRegex.FindStringSubmatch(line); len(tm) == 2 {
			priceStr := strings.ReplaceAll(tm[1], ",", ".")
			if val, err := strconv.ParseFloat(priceStr, 64); err == nil && val > totalFromExplicitLine {
				totalFromExplicitLine = val
			}
			continue
		}

		// Skip non-item headers/footers
		if skipLineRegex.MatchString(line) {
			continue
		}

		// Look for price at end of line
		pm := priceRegex.FindStringSubmatch(line)
		if len(pm) < 2 {
			continue
		}

		priceStr := strings.ReplaceAll(pm[1], ",", ".")
		priceVal, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || priceVal <= 0 || priceVal > 500 {
			continue
		}

		// Extract item name by removing price portion
		namePart := strings.TrimSpace(line[:len(line)-len(pm[0])])
		namePart = cleanItemName(namePart)
		if len(namePart) < 2 {
			continue
		}

		// Detect quantity and unit
		qty, unit := extractQuantityAndUnit(namePart)
		cat := inferCategory(namePart)

		receipt.Items = append(receipt.Items, domain.ReceiptItem{
			RawLine:    line,
			Name:       formatItemTitle(namePart),
			Quantity:   qty,
			Unit:       unit,
			UnitPrice:  priceVal / qty,
			TotalPrice: priceVal,
			Category:   cat,
		})
	}

	// 4. Calculate or confirm total amount
	if totalFromExplicitLine > 0 {
		receipt.TotalAmount = totalFromExplicitLine
	} else {
		var sum float64
		for _, item := range receipt.Items {
			sum += item.TotalPrice
		}
		receipt.TotalAmount = math.Round(sum*100) / 100
	}

	// 5. Confidence score
	score := 0.4
	if receipt.StoreName != "Supermercato" {
		score += 0.2
	}
	if len(receipt.Items) > 0 {
		score += 0.2
	}
	if totalFromExplicitLine > 0 {
		score += 0.2
	}
	receipt.Confidence = math.Min(1.0, score)

	return receipt, nil
}

func cleanItemName(raw string) string {
	res := regexp.MustCompile(`[#*]`).ReplaceAllString(raw, "")
	res = regexp.MustCompile(`\s{2,}`).ReplaceAllString(res, " ")
	return strings.TrimSpace(res)
}

func formatItemTitle(name string) string {
	words := strings.Fields(name)
	for i, w := range words {
		lower := strings.ToLower(w)
		if len(lower) > 0 {
			words[i] = strings.ToUpper(lower[:1]) + lower[1:]
		}
	}
	return strings.Join(words, " ")
}

func extractQuantityAndUnit(name string) (float64, domain.Unit) {
	// Check pack multi (e.g. 6x1.5L)
	if m := packMultiRegex.FindStringSubmatch(name); len(m) == 4 {
		count, _ := strconv.ParseFloat(m[1], 64)
		vol, _ := strconv.ParseFloat(strings.ReplaceAll(m[2], ",", "."), 64)
		u := strings.ToLower(m[3])
		switch u {
		case "l":
			return count * vol, domain.UnitLiter
		case "kg":
			return count * vol, domain.UnitKilogram
		case "g":
			return count * vol, domain.UnitGram
		case "ml":
			return count * vol, domain.UnitMillilit
		}
	}

	// Check weight in kg
	if m := weightKgRegex.FindStringSubmatch(name); len(m) > 1 {
		strVal := m[1]
		if strVal == "" && len(m) > 2 {
			strVal = m[2]
		}
		val, _ := strconv.ParseFloat(strings.ReplaceAll(strVal, ",", "."), 64)
		if val > 0 {
			return val, domain.UnitKilogram
		}
	}

	// Check weight in g
	if m := weightGramRegex.FindStringSubmatch(name); len(m) == 2 {
		val, _ := strconv.ParseFloat(m[1], 64)
		if val > 0 {
			return val, domain.UnitGram
		}
	}

	// Check volume in l
	if m := volLiterRegex.FindStringSubmatch(name); len(m) > 1 {
		strVal := m[1]
		if strVal == "" && len(m) > 2 {
			strVal = m[2]
		}
		val, _ := strconv.ParseFloat(strings.ReplaceAll(strVal, ",", "."), 64)
		if val > 0 {
			return val, domain.UnitLiter
		}
	}

	// Check volume in ml
	if m := volMlRegex.FindStringSubmatch(name); len(m) == 2 {
		val, _ := strconv.ParseFloat(m[1], 64)
		if val > 0 {
			return val, domain.UnitMillilit
		}
	}

	// Check piece count
	if m := pieceCountRegex.FindStringSubmatch(name); len(m) > 1 {
		strVal := m[1]
		if strVal == "" && len(m) > 2 {
			strVal = m[2]
		}
		val, _ := strconv.ParseFloat(strVal, 64)
		if val > 0 {
			return val, domain.UnitPiece
		}
	}

	// Default fallback: 1 piece
	return 1.0, domain.UnitPiece
}

func inferCategory(name string) domain.Category {
	lower := strings.ToLower(name)

	switch {
	case containsAny(lower, "latte", "formaggio", "parmigiano", "grana", "mozzarella", "ricotta", "yogurt", "burro", "uova", "panna", "gorgonzola", "mascarpone", "caciotta"):
		return domain.CategoryDairy

	case containsAny(lower, "carne", "manzo", "pollo", "maiale", "pesce", "salmone", "orata", "prosciutto", "salame", "wurstel", "bistecca", "macinato", "tonno", "vitello", "tacchino", "merluzzo", "calamari", "gamberi"):
		return domain.CategoryMeatFish

	case containsAny(lower, "mela", "mele", "banana", "banane", "aranc", "limon", "insalata", "verdura", "zucchine", "carote", "patate", "cipolle", "pomodor", "fragole", "pere", "uva", "pesche", "spinaci", "finocchi"):
		return domain.CategoryProduce

	case containsAny(lower, "pane", "fette", "focaccia", "pizza", "crackers", "grissini", "piadina", "panino", "toast"):
		return domain.CategoryBakery

	case containsAny(lower, "pasta", "riso", "farina", "zucchero", "sale", "legumi", "fagioli", "ceci", "lenticchie", "passata", "pelati", "polenta", "couscous", "spaghetti", "penne", "fusilli", "rigatoni"):
		return domain.CategoryPantry

	case containsAny(lower, "surgelat", "gelat", "bastoncini", "congelat"):
		return domain.CategoryFrozen

	case containsAny(lower, "acqua", "vino", "birra", "succo", "cola", "aranciata", "the", "caffè", "bevanda", "aperitivo", "spumante"):
		return domain.CategoryDrinks

	case containsAny(lower, "olio", "aceto", "maionese", "ketchup", "spezie", "origano", "pepe", "senape", "dado", "brodo", "salsa"):
		return domain.CategoryCondiment

	case containsAny(lower, "biscotti", "merendin", "cioccolato", "snack", "patatine", "caramelle", "wafer", "nutella", "croissant", "barretta"):
		return domain.CategorySnack

	case containsAny(lower, "detersivo", "sapone", "carta", "scottex", "candeggina", "spugna", "dentifricio", "shampoo", "bagnoschiuma", "candela", "sacchetti"):
		return domain.CategoryHousehold

	default:
		return domain.CategoryOther
	}
}

func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
