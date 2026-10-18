// Package medicine defines domain models and contracts for household medicine inventory,
// expiration tracking, period after opening (PAO), cold-chain requirements, and GS1 Datamatrix parsing.
package medicine

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidMedicineName = errors.New("medicine: name cannot be empty and must be under 150 characters")
	ErrInvalidQuantity     = errors.New("medicine: quantity cannot be negative")
	ErrInvalidThreshold    = errors.New("medicine: threshold cannot be negative")
	ErrInvalidExpiration   = errors.New("medicine: expiration date must follow YYYY-MM-DD or YYYY-MM format")
	ErrMedicineNotFound    = errors.New("medicine: item not found")
	ErrInvalidPaoDays      = errors.New("medicine: pao_days cannot be negative")
)

// Location specifies where the medicine is stored in the household.
type Location string

const (
	LocationBathroomCabinet Location = "armadietto_bagno"
	LocationBedsideTable    Location = "cassetto_notte"
	LocationTravelBag       Location = "borsa_viaggio"
	LocationFridge          Location = "frigo" // Per farmaci termolabili (2-8°C)
	LocationOther           Location = "altro"
)

// Unit defines the measurement or dosage unit for the medicine.
type Unit string

const (
	UnitTablets Unit = "compresse"
	UnitSachets Unit = "bustine"
	UnitDrops   Unit = "gocce"
	UnitMl      Unit = "ml"
	UnitPieces  Unit = "pz"
	UnitPack    Unit = "confezione"
)

// Medicine represents a pharmaceutical product or medical supply in the household.
type Medicine struct {
	ID                      string    `json:"id"`
	Name                    string    `json:"name"`
	ActiveIngredient        string    `json:"active_ingredient,omitempty"` // Es. "Paracetamolo", "Ibuprofene"
	Dosage                  string    `json:"dosage,omitempty"`            // Es. "1000 mg", "500 mg"
	Form                    string    `json:"form,omitempty"`              // Es. "compresse", "sciroppo", "gocce"
	AIC                     string    `json:"aic,omitempty"`               // Codice AIC a 9 cifre (Italia)
	Barcode                 string    `json:"barcode,omitempty"`           // EAN o Datamatrix raw
	LotNumber               string    `json:"lot_number,omitempty"`        // Numero di lotto
	ExpirationDate          string    `json:"expiration_date"`             // YYYY-MM-DD impressa sulla confezione
	PaoDays                 int       `json:"pao_days"`                    // Period After Opening (giorni di validità dall'apertura, 0 = non applicabile)
	OpenedAt                *string   `json:"opened_at,omitempty"`         // YYYY-MM-DD data in cui è stato aperto
	EffectiveExpirationDate string    `json:"effective_expiration_date"`   // Data di scadenza effettiva: min(ExpirationDate, OpenedAt + PaoDays)
	Location                Location  `json:"location"`                    // Ubicazione
	RequiresColdChain       bool      `json:"requires_cold_chain"`         // True se termolabile (conservare a 2-8°C)
	Quantity                float64   `json:"quantity"`                    // Quantità residua
	Unit                    Unit      `json:"unit"`                        // Unità di misura
	MinThreshold            float64   `json:"min_threshold"`               // Scorta minima per avviso ricetta/riordino
	PrescriptionRequired    bool      `json:"prescription_required"`       // True se richiede ricetta medica
	Notes                   string    `json:"notes,omitempty"`             // Posologia, istruzioni
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// CalculateEffectiveExpiration computes the actual expiration date considering original expiration and PAO.
func (m *Medicine) CalculateEffectiveExpiration() string {
	exp := NormalizeDate(m.ExpirationDate)
	if m.OpenedAt == nil || *m.OpenedAt == "" || m.PaoDays <= 0 {
		return exp
	}

	opened, err := time.Parse("2006-01-02", NormalizeDate(*m.OpenedAt))
	if err != nil {
		return exp
	}

	paoDate := opened.AddDate(0, 0, m.PaoDays)
	paoDateStr := paoDate.Format("2006-01-02")

	// Se la scadenza della confezione è precedente a quella PAO, vale la data confezione
	if exp != "" {
		expDate, errExp := time.Parse("2006-01-02", exp)
		if errExp == nil && expDate.Before(paoDate) {
			return exp
		}
	}

	return paoDateStr
}

// Validate validates fields before persistence.
func (m *Medicine) Validate() error {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" || len(m.Name) > 150 {
		return ErrInvalidMedicineName
	}
	if m.Quantity < 0 {
		return ErrInvalidQuantity
	}
	if m.MinThreshold < 0 {
		return ErrInvalidThreshold
	}
	if m.PaoDays < 0 {
		return ErrInvalidPaoDays
	}

	m.ExpirationDate = NormalizeDate(m.ExpirationDate)
	if m.ExpirationDate == "" {
		return ErrInvalidExpiration
	}
	if _, err := time.Parse("2006-01-02", m.ExpirationDate); err != nil {
		return ErrInvalidExpiration
	}

	if m.Location == "" {
		m.Location = LocationBathroomCabinet
	}
	if m.RequiresColdChain && m.Location != LocationFridge {
		// Se richiede frigo e location era generica, default a frigo
		if m.Location == LocationBathroomCabinet || m.Location == LocationOther {
			m.Location = LocationFridge
		}
	}
	if m.Unit == "" {
		m.Unit = UnitTablets
	}

	m.EffectiveExpirationDate = m.CalculateEffectiveExpiration()
	return nil
}

// NormalizeDate converts YYYY-MM into the last day of the month YYYY-MM-DD, or trims YYYY-MM-DD.
func NormalizeDate(d string) string {
	d = strings.TrimSpace(d)
	if len(d) == 7 && strings.Contains(d, "-") {
		// YYYY-MM -> last day of month
		parts := strings.Split(d, "-")
		if len(parts) == 2 {
			y, errY := strconv.Atoi(parts[0])
			m, errM := strconv.Atoi(parts[1])
			if errY == nil && errM == nil && m >= 1 && m <= 12 {
				firstOfNext := time.Date(y, time.Month(m+1), 1, 0, 0, 0, 0, time.UTC)
				lastDay := firstOfNext.AddDate(0, 0, -1)
				return lastDay.Format("2006-01-02")
			}
		}
	}
	if len(d) >= 10 {
		return d[:10]
	}
	return d
}

// DatamatrixParseResult contains decoded fields from a GS1 Datamatrix or Barcode.
type DatamatrixParseResult struct {
	AIC            string `json:"aic,omitempty"`
	GTIN           string `json:"gtin,omitempty"`
	ExpirationDate string `json:"expiration_date,omitempty"` // YYYY-MM-DD
	LotNumber      string `json:"lot_number,omitempty"`
	SerialNumber   string `json:"serial_number,omitempty"`
	RawCode        string `json:"raw_code"`
}

// ParseDatamatrix extracts AIC, expiration date (YYMMDD), and lot number from GS1 formatted codes.
func ParseDatamatrix(raw string) *DatamatrixParseResult {
	clean := strings.TrimSpace(raw)
	res := &DatamatrixParseResult{RawCode: clean}

	// 1. Caso semplice: solo codice AIC a 9 cifre (es. "034234035")
	if matched, _ := regexp.MatchString(`^\d{9}$`, clean); matched {
		res.AIC = clean
		return res
	}

	// 2. Parser GS1 con parentesi tipo (01)08012345678901(17)260531(10)LOT1234
	if strings.Contains(clean, "(") {
		r01 := regexp.MustCompile(`\(01\)(\d{14})`)
		r17 := regexp.MustCompile(`\(17\)(\d{6})`)
		r10 := regexp.MustCompile(`\(10\)([^()]+)`)
		r21 := regexp.MustCompile(`\(21\)([^()]+)`)

		if m := r01.FindStringSubmatch(clean); len(m) > 1 {
			res.GTIN = m[1]
			res.AIC = extractAICFromGTIN(m[1])
		}
		if m := r17.FindStringSubmatch(clean); len(m) > 1 {
			res.ExpirationDate = parseGS1Date(m[1])
		}
		if m := r10.FindStringSubmatch(clean); len(m) > 1 {
			res.LotNumber = strings.TrimSpace(m[1])
		}
		if m := r21.FindStringSubmatch(clean); len(m) > 1 {
			res.SerialNumber = strings.TrimSpace(m[1])
		}
		return res
	}

	// 3. Parser GS1 continuo (FNC1 o stringa diretta senza parentesi):
	// Es: "01080123456789011726053110LOT1234"
	// Riconosce 01 (14 cifre) seguito da 17 (6 cifre) e 10 (lotto)
	rStream := regexp.MustCompile(`01(\d{14})17(\d{6})(?:10([A-Za-z0-9]+))?`)
	if m := rStream.FindStringSubmatch(clean); len(m) > 2 {
		res.GTIN = m[1]
		res.AIC = extractAICFromGTIN(m[1])
		res.ExpirationDate = parseGS1Date(m[2])
		if len(m) > 3 && m[3] != "" {
			res.LotNumber = m[3]
		}
		return res
	}

	// 4. Fallback: cerca data YYMMDD se preceduta da 17
	r17Simple := regexp.MustCompile(`(?:^|[^0-9])17(\d{6})`)
	if m := r17Simple.FindStringSubmatch(clean); len(m) > 1 {
		res.ExpirationDate = parseGS1Date(m[1])
	}

	// Fallback AIC
	rAIC := regexp.MustCompile(`\b(\d{9})\b`)
	if m := rAIC.FindStringSubmatch(clean); len(m) > 1 {
		res.AIC = m[1]
	}

	return res
}

func extractAICFromGTIN(gtin string) string {
	// In Italia il GTIN farmaceutico GS1 è tipicamente 080 + 9 cifre AIC + check digit (14 cifre)
	// oppure 0080 + 9 cifre AIC + check digit (14 cifre) o 80 + 9 cifre AIC + check digit (13 cifre)
	if len(gtin) == 14 {
		if strings.HasPrefix(gtin, "080") {
			return gtin[3:12]
		}
		if strings.HasPrefix(gtin, "0080") {
			return gtin[4:13]
		}
	}
	if len(gtin) == 13 && strings.HasPrefix(gtin, "80") {
		return gtin[2:11]
	}
	return gtin
}

func parseGS1Date(yymmdd string) string {
	if len(yymmdd) != 6 {
		return ""
	}
	yy, errY := strconv.Atoi(yymmdd[0:2])
	mm, errM := strconv.Atoi(yymmdd[2:4])
	dd, errD := strconv.Atoi(yymmdd[4:6])
	if errY != nil || errM != nil || errD != nil || mm < 1 || mm > 12 {
		return ""
	}

	// Anno 20YY
	fullYear := 2000 + yy

	// Se dd == 00, significa ultimo giorno del mese
	if dd == 0 {
		firstOfNext := time.Date(fullYear, time.Month(mm+1), 1, 0, 0, 0, 0, time.UTC)
		lastDay := firstOfNext.AddDate(0, 0, -1)
		return lastDay.Format("2006-01-02")
	}

	return fmt.Sprintf("%04d-%02d-%02d", fullYear, mm, dd)
}

// AlertType categorizes notifications about medicines.
type AlertType string

const (
	AlertExpired           AlertType = "expired"
	AlertExpiringSoon7D    AlertType = "expiring_soon_7d"
	AlertExpiringSoon30D   AlertType = "expiring_soon_30d"
	AlertLowStock          AlertType = "low_stock"
	AlertColdChainBreached AlertType = "cold_chain_warning"
)

// MedicineAlert represents an actionable notification for an item.
type MedicineAlert struct {
	MedicineID    string    `json:"medicine_id"`
	MedicineName  string    `json:"medicine_name"`
	Type          AlertType `json:"type"`
	Severity      string    `json:"severity"` // "critical", "warning", "info"
	DaysRemaining int       `json:"days_remaining"`
	Message       string    `json:"message"`
	Location      Location  `json:"location"`
}

// ComputeAlerts evaluates medicines against current date and fridge temperature.
func ComputeAlerts(items []*Medicine, now time.Time, fridgeTemp *float64) []MedicineAlert {
	var alerts []MedicineAlert
	nowZero := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	for _, m := range items {
		// Scadenza
		effExp := m.CalculateEffectiveExpiration()
		if expDate, err := time.Parse("2006-01-02", effExp); err == nil {
			days := int(expDate.Sub(nowZero).Hours() / 24)
			if days < 0 {
				alerts = append(alerts, MedicineAlert{
					MedicineID:    m.ID,
					MedicineName:  m.Name,
					Type:          AlertExpired,
					Severity:      "critical",
					DaysRemaining: days,
					Message:       fmt.Sprintf("%s è scaduto il %s (%d giorni fa)", m.Name, effExp, -days),
					Location:      m.Location,
				})
			} else if days <= 7 {
				alerts = append(alerts, MedicineAlert{
					MedicineID:    m.ID,
					MedicineName:  m.Name,
					Type:          AlertExpiringSoon7D,
					Severity:      "critical",
					DaysRemaining: days,
					Message:       fmt.Sprintf("%s scade tra %d giorni (%s)", m.Name, days, effExp),
					Location:      m.Location,
				})
			} else if days <= 30 {
				alerts = append(alerts, MedicineAlert{
					MedicineID:    m.ID,
					MedicineName:  m.Name,
					Type:          AlertExpiringSoon30D,
					Severity:      "warning",
					DaysRemaining: days,
					Message:       fmt.Sprintf("%s scade tra %d giorni (%s)", m.Name, days, effExp),
					Location:      m.Location,
				})
			}
		}

		// Scorte basse
		if m.MinThreshold > 0 && m.Quantity <= m.MinThreshold {
			alerts = append(alerts, MedicineAlert{
				MedicineID:    m.ID,
				MedicineName:  m.Name,
				Type:          AlertLowStock,
				Severity:      "warning",
				DaysRemaining: 0,
				Message:       fmt.Sprintf("Scorta in esaurimento per %s: rimasti %.1f %s (soglia minima: %.1f)", m.Name, m.Quantity, m.Unit, m.MinThreshold),
				Location:      m.Location,
			})
		}

		// Controllo catena del freddo (termolabili in frigo 2-8°C)
		if m.RequiresColdChain && fridgeTemp != nil && *fridgeTemp > 8.0 {
			alerts = append(alerts, MedicineAlert{
				MedicineID:    m.ID,
				MedicineName:  m.Name,
				Type:          AlertColdChainBreached,
				Severity:      "critical",
				DaysRemaining: 0,
				Message:       fmt.Sprintf("Allarme catena del freddo: %s richiede 2-8°C ma la temperatura del frigo è %.1f°C", m.Name, *fridgeTemp),
				Location:      m.Location,
			})
		}
	}

	return alerts
}
