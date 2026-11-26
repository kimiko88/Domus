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
	"unicode"

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
		{regexp.MustCompile(`(?i)\bcarrefour\s+(?:market|express)\b`), "Carrefour Market"},
		{regexp.MustCompile(`(?i)\bpam\s+panorama\b`), "Pam Panorama"},
		{regexp.MustCompile(`(?i)\bipercoop\b`), "Ipercoop"},
		{regexp.MustCompile(`(?i)\bconad\s+(?:city|superstore|margherita)\b`), "Conad"},
		{regexp.MustCompile(`(?i)\beurospar\b`), "Eurospar"},
		{regexp.MustCompile(`(?i)\binterspar\b`), "Interspar"},
		{regexp.MustCompile(`(?i)\bin['’]?s\s+mercato\b`), "In's Mercato"},
		{regexp.MustCompile(`(?i)\biperal\b`), "Iperal"},
		{regexp.MustCompile(`(?i)\bsupermercati\s+tosano\b|\btosano\b`), "Tosano"},
		{regexp.MustCompile(`(?i)\bpenny(?:\s+market)?\b`), "Penny Market"},
		{regexp.MustCompile(`(?i)\bmaxi\s*di\b`), "Maxi Di"},
		{regexp.MustCompile(`(?i)\bsole\s*365\b`), "Sole 365"},
		{regexp.MustCompile(`(?i)\bnatura\s*si\b`), "NaturaSì"},
		{regexp.MustCompile(`(?i)\bdpi[uù]\b`), "Dpiù"},
		{regexp.MustCompile(`(?i)\bdec[oò]\b`), "Decò"},
		{regexp.MustCompile(`(?i)\bil\s+gigante\b`), "Il Gigante"},
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
		{regexp.MustCompile(`(?i)\bmd(?:\s+market|\s+spa)?\b`), "MD"},
		{regexp.MustCompile(`(?i)\bfamila\b`), "Famila"},
		{regexp.MustCompile(`(?i)\bsigma\b`), "Sigma"},
		{regexp.MustCompile(`(?i)\bcrai\b`), "CRAI"},
		{regexp.MustCompile(`(?i)\btodis\b`), "Todis"},
		{regexp.MustCompile(`(?i)\bbennet\b`), "Bennet"},
		{regexp.MustCompile(`(?i)\brossetto\b`), "Rossetto"},
		{regexp.MustCompile(`(?i)\bunes\b`), "Unes"},
		{regexp.MustCompile(`(?i)\bdok\b`), "Dok"},
		{regexp.MustCompile(`(?i)\bmigross\b`), "Migross"},
		{regexp.MustCompile(`(?i)\bprix\b`), "Prix"},
		{regexp.MustCompile(`(?i)\bekom\b`), "Ekom"},
		{regexp.MustCompile(`(?i)\bpewex\b`), "Pewex"},
		{regexp.MustCompile(`(?i)\ba\s*&\s*o\b`), "A&O"},
		{regexp.MustCompile(`(?i)\bard\s+discount\b`), "ARD Discount"},
	}
	storeLegalSuffixRegex = regexp.MustCompile(`(?i)\s+(?:s\.?\s*p\.?\s*a\.?|s\.?\s*r\.?\s*l\.?|s\.?\s*n\.?\s*c\.?)\s*$`)
	storeHeaderLabelRegex = regexp.MustCompile(`(?i)^\s*(?:ragione\s+sociale|denominazione|insegna|punto\s+vendita|negozio|esercente)\s*[:\-]\s*`)
	storeHeaderNoiseRegex = regexp.MustCompile(`(?i)(?:scontrino|ricevuta|fattura|invoice|fiscale|documento\s*commerciale|commerciale|pagamento|partita\s*iva|p\.?\s*iva|codice\s*fiscale|cassa|operatore|subtotale|totale|resto|contanti|bancomat|\bpos\b|grazie|arrivederci|\bvia\b|\bviale\b|\bpiazza\b|\bcorso\b|\bstrada\b|\bcap\b|\btel\b|telefono|www\.|https?://|\b(?:ora|data)\b)`)
	storeGenericHeaderRegex = regexp.MustCompile(`(?i)^(?:supermercato|market|negozio|alimentari|discount|minimarket|farmacia|ristorante|bar|panificio|macelleria|pescheria)$`)
	dateRegex = regexp.MustCompile(`\b(\d{1,2})[/.-](\d{1,2})[/.-](\d{2,4})\b`)

	// Matches price patterns like 1,29, 12.50, 4.99€, etc.
	// Italian thousands separators, ungrouped prices and US notation.
	receiptMoneyPattern = "(?:[0-9]{1,3}(?:[. ][0-9]{3})+|[0-9]+),[0-9]{2}|(?:[0-9]{1,3}(?:,[0-9]{3})+|[0-9]+)\\.[0-9]{2}|[0-9]+[.,][0-9]{2}"
	priceRegex = regexp.MustCompile("(?i)(" + receiptMoneyPattern + ")\\s*(?:€|eur)?$")
	receiptAmountRegex = regexp.MustCompile(receiptMoneyPattern)
	strongTotalLabelRegex = regexp.MustCompile("(?i)\\b(?:totale\\s+(?:complessivo|da\\s+pagare|documento|fattura)|importo\\s+(?:pagato|da\\s+pagare)|da\\s+pagare)\\b")
	totalLabelRegex = regexp.MustCompile("(?i)\\btotale\\b")
	currencyTotalLabelRegex = regexp.MustCompile("(?i)\\b(?:euro|tot)\\b")

	// Quantity patterns
	weightGramRegex = regexp.MustCompile(`(?i)\b(\d+)\s*(?:g|gr)\b`)
	weightKgRegex   = regexp.MustCompile(`(?i)(?:\b(\d+(?:[.,]\d+)?)\s*(?:kg|chili)\b|\b(?:kg|chili)\s*(\d+(?:[.,]\d+)?)\b)`)
	volLiterRegex   = regexp.MustCompile(`(?i)(?:\b(\d+(?:[.,]\d+)?)\s*(?:l|lt|litri)\b|\b(?:l|lt|litri)\s*(\d+(?:[.,]\d+)?)\b)`)
	volMlRegex      = regexp.MustCompile(`(?i)\b(\d+)\s*(?:ml)\b`)
	packMultiRegex  = regexp.MustCompile(`(?i)\b(\d+)\s*x\s*(\d+(?:[.,]\d+)?)\s*(l|kg|g|ml)\b`)
	pieceCountRegex = regexp.MustCompile(`(?i)(?:\b(\d+)\s*(?:pz|pezzi|conf)\b|\b(?:pz|pezzi|conf)\s*(\d+)\b)`)


	skipLineRegex = regexp.MustCompile("(?i)(?:scontrino|fiscale|documento\\s*commerciale|p\\.?\\s*iva|partita\\s*iva|\\bcassa\\b|operatore|subtotale|resto|contanti|carta\\s+(?:di\\s+credito|credito|pagamento)|bancomat|\\bpos\\b|grazie|arrivederci|via\\b|tel\\b|codice|punti|sconto)")
)

// ParseText analyzes receipt plain text and extracts a structured Receipt.
func (p *ReceiptParser) ParseText(text string) (*domain.Receipt, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("receipt text is empty")
	}
	if len(text) > 64*1024 {
		return nil, fmt.Errorf("receipt text exceeds the 64 KiB limit")
	}
	lines := strings.Split(text, "\n")
	receipt := &domain.Receipt{
		StoreName: "Supermercato",
		Date:      time.Now().Format("2006-01-02"),
		Items:     make([]domain.ReceiptItem, 0),
		RawText:   text,
	}

	// Known chains are matched first; independent merchants use a header heuristic.
	receipt.StoreName = detectReceiptStoreName(lines)

	// 2. Detect Date
	for _, line := range lines {
		if m := dateRegex.FindStringSubmatch(line); len(m) == 4 {
			day, _ := strconv.Atoi(m[1])
			month, _ := strconv.Atoi(m[2])
			year, _ := strconv.Atoi(m[3])
			if year < 100 {
				year += 2000
			}
			if year >= 2000 {
				candidate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
				if int(candidate.Day()) == day && int(candidate.Month()) == month && candidate.Year() == year {
					receipt.Date = candidate.Format("2006-01-02")
					break
				}
			}
		}
	}

	// 3. Scan lines for products and total
	var totalFromExplicitLine float64
	var totalPriority int
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		// Resolve an explicit total before parsing line items.
		if amount, priority, ok := parseReceiptTotal(line); ok {
			if priority >= totalPriority {
				totalFromExplicitLine = amount
				totalPriority = priority
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

		priceVal, err := parseReceiptMoney(pm[1])
		if err != nil || priceVal <= 0 || priceVal > 1_000_000 {
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

	// 5. Penalize a total that is inconsistent with extracted item lines.
	score := 0.4
	if receipt.StoreName != "Supermercato" { score += 0.2 }
	if len(receipt.Items) > 0 { score += 0.2 }
	if totalFromExplicitLine > 0 {
		if len(receipt.Items) == 0 {
			score += 0.05
		} else {
			var itemSum float64
			for _, item := range receipt.Items { itemSum += item.TotalPrice }
			tolerance := math.Max(0.10, receipt.TotalAmount*0.03)
			if math.Abs(itemSum-receipt.TotalAmount) <= tolerance { score += 0.2 } else { score += 0.05 }
		}
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
	for i, word := range words {
		runes := []rune(strings.ToLower(word))
		if len(runes) > 0 { runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]; words[i] = string(runes) }
	}
	return strings.Join(words, " ")
}

// detectReceiptStoreName recognizes major chains, then looks for a plausible
// first-line merchant header (including independent shops and pharmacies).
// detectReceiptStoreName checks known chains in the document header first,
// then accepts a plausible merchant line. It stops at the first item/total so
// a product name is never mistaken for a shop when the store header is absent.
func detectReceiptStoreName(lines []string) string {
	nonEmpty := 0
	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.Trim(rawLine, " -*_#|"))
		if line == "" {
			continue
		}
		nonEmpty++
		if nonEmpty > 10 {
			break
		}
		// A price at the end of a line marks the beginning of product rows.
		if priceRegex.MatchString(line) || strongTotalLabelRegex.MatchString(line) || totalLabelRegex.MatchString(line) {
			break
		}
		if storeHeaderNoiseRegex.MatchString(line) || dateRegex.MatchString(line) {
			continue
		}
		for _, candidate := range storePatterns {
			if candidate.pattern.MatchString(line) {
				return candidate.name
			}
		}
		candidate := strings.TrimSpace(storeHeaderLabelRegex.ReplaceAllString(line, ""))
		candidate = storeLegalSuffixRegex.ReplaceAllString(candidate, "")
		candidate = strings.TrimSpace(strings.Trim(candidate, ".,;:-"))
		runes := []rune(candidate)
		if len(runes) < 3 || len(runes) > 64 || storeGenericHeaderRegex.MatchString(candidate) {
			continue
		}
		letters, digits := 0, 0
		for _, char := range runes {
			if unicode.IsLetter(char) {
				letters++
			} else if unicode.IsDigit(char) {
				digits++
			}
		}
		if letters < 3 || digits > letters/2 || len(strings.Fields(candidate)) > 8 {
			continue
		}
		// Reject product-like headers with a quantity marker even if OCR
		// omitted the trailing price.
		if weightGramRegex.MatchString(candidate) || weightKgRegex.MatchString(candidate) ||
			volLiterRegex.MatchString(candidate) || volMlRegex.MatchString(candidate) ||
			packMultiRegex.MatchString(candidate) || pieceCountRegex.MatchString(candidate) {
			break
		}
		return candidate
	}
	return "Supermercato"
}
// parseReceiptMoney normalizes Italian and US decimal/thousands separators.
func parseReceiptMoney(raw string) (float64, error) {
	value := strings.TrimSpace(raw)
	value = strings.ReplaceAll(value, "€", "")
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "\u00a0", "")
	value = strings.ReplaceAll(value, "\u202f", "")
	lastComma, lastDot := strings.LastIndex(value, ","), strings.LastIndex(value, ".")
	switch {
	case lastComma >= 0 && lastDot >= 0:
		if lastComma > lastDot { value = strings.ReplaceAll(value, ".", ""); value = strings.Replace(value, ",", ".", 1) } else { value = strings.ReplaceAll(value, ",", "") }
	case lastComma >= 0:
		value = strings.Replace(value, ",", ".", 1)
	case strings.Count(value, ".") > 1:
		last := strings.LastIndex(value, "."); value = strings.ReplaceAll(value[:last], ".", "") + value[last:]
	}
	return strconv.ParseFloat(value, 64)
}

// parseReceiptTotal returns a labeled total and its precedence.
func parseReceiptTotal(line string) (float64, int, bool) {
	lower := strings.ToLower(line)
	if strings.Contains(lower, "subtotale") || strings.Contains(lower, "imponibile") { return 0, 0, false }
	priority := 0
	switch {
	case strongTotalLabelRegex.MatchString(line): priority = 3
	case totalLabelRegex.MatchString(line): priority = 2
	case currencyTotalLabelRegex.MatchString(line): priority = 1
	default: return 0, 0, false
	}
	matches := receiptAmountRegex.FindAllString(line, -1)
	if len(matches) == 0 { return 0, 0, false }
	amount, err := parseReceiptMoney(matches[len(matches)-1])
	if err != nil || amount < 0 || amount > 1_000_000 { return 0, 0, false }
	return amount, priority, true
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
