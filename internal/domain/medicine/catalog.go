package medicine

import (
	"strings"
)

// AIFARecord contains metadata from the Italian Medicines Agency (AIFA) catalog.
type AIFARecord struct {
	AIC               string   `json:"aic"`
	Name              string   `json:"name"`
	ActiveIngredient  string   `json:"active_ingredient"`
	Dosage            string   `json:"dosage"`
	Form              string   `json:"form"`
	Unit              Unit     `json:"unit"`
	DefaultQuantity   float64  `json:"default_quantity"`
	RequiresColdChain bool     `json:"requires_cold_chain"`
	DefaultLocation   Location `json:"default_location"`
	DefaultPaoDays    int      `json:"default_pao_days"`
	PrescriptionReq   bool     `json:"prescription_required"`
}

// Built-in offline AIFA reference database covering widespread Italian household medicines.
var CommonAIFADatabase = []AIFARecord{
	{
		AIC:               "034234035",
		Name:              "Tachipirina 1000 mg compresse",
		ActiveIngredient:  "Paracetamolo",
		Dosage:            "1000 mg",
		Form:              "compresse",
		Unit:              UnitTablets,
		DefaultQuantity:   16,
		RequiresColdChain: false,
		DefaultLocation:   LocationBathroomCabinet,
		DefaultPaoDays:    0,
		PrescriptionReq:   false,
	},
	{
		AIC:               "034234023",
		Name:              "Tachipirina 500 mg compresse",
		ActiveIngredient:  "Paracetamolo",
		Dosage:            "500 mg",
		Form:              "compresse",
		Unit:              UnitTablets,
		DefaultQuantity:   30,
		RequiresColdChain: false,
		DefaultLocation:   LocationBathroomCabinet,
		DefaultPaoDays:    0,
		PrescriptionReq:   false,
	},
	{
		AIC:               "034234074",
		Name:              "Tachipirina Sciroppo 120 ml",
		ActiveIngredient:  "Paracetamolo",
		Dosage:            "120 mg / 5 ml",
		Form:              "sciroppo",
		Unit:              UnitMl,
		DefaultQuantity:   120,
		RequiresColdChain: false,
		DefaultLocation:   LocationBathroomCabinet,
		DefaultPaoDays:    90, // Scade 90 gg dopo apertura
		PrescriptionReq:   false,
	},
	{
		AIC:               "026772023",
		Name:              "Oki 80 mg granulato per soluzione orale",
		ActiveIngredient:  "Ketoprofene sale di lisina",
		Dosage:            "80 mg",
		Form:              "bustine",
		Unit:              UnitSachets,
		DefaultQuantity:   30,
		RequiresColdChain: false,
		DefaultLocation:   LocationBathroomCabinet,
		DefaultPaoDays:    0,
		PrescriptionReq:   true,
	},
	{
		AIC:               "042153018",
		Name:              "Oki Task 40 mg granulato orosolubile",
		ActiveIngredient:  "Ketoprofene sale di lisina",
		Dosage:            "40 mg",
		Form:              "bustine",
		Unit:              UnitSachets,
		DefaultQuantity:   10,
		RequiresColdChain: false,
		DefaultLocation:   LocationTravelBag,
		DefaultPaoDays:    0,
		PrescriptionReq:   false,
	},
	{
		AIC:               "022535035",
		Name:              "Brufen 400 mg compresse rivestite",
		ActiveIngredient:  "Ibuprofene",
		Dosage:            "400 mg",
		Form:              "compresse",
		Unit:              UnitTablets,
		DefaultQuantity:   30,
		RequiresColdChain: false,
		DefaultLocation:   LocationBathroomCabinet,
		DefaultPaoDays:    0,
		PrescriptionReq:   true,
	},
	{
		AIC:               "028266015",
		Name:              "Moment 200 mg compresse rivestite",
		ActiveIngredient:  "Ibuprofene",
		Dosage:            "200 mg",
		Form:              "compresse",
		Unit:              UnitTablets,
		DefaultQuantity:   12,
		RequiresColdChain: false,
		DefaultLocation:   LocationBathroomCabinet,
		DefaultPaoDays:    0,
		PrescriptionReq:   false,
	},
	{
		AIC:               "033224017",
		Name:              "Momentact 400 mg compresse",
		ActiveIngredient:  "Ibuprofene",
		Dosage:            "400 mg",
		Form:              "compresse",
		Unit:              UnitTablets,
		DefaultQuantity:   12,
		RequiresColdChain: false,
		DefaultLocation:   LocationBathroomCabinet,
		DefaultPaoDays:    0,
		PrescriptionReq:   false,
	},
	{
		AIC:               "026089073",
		Name:              "Augmentin 875 mg + 125 mg compresse",
		ActiveIngredient:  "Amoxicillina + Acido Clavulanico",
		Dosage:            "1 g",
		Form:              "compresse",
		Unit:              UnitTablets,
		DefaultQuantity:   12,
		RequiresColdChain: false,
		DefaultLocation:   LocationBathroomCabinet,
		DefaultPaoDays:    0,
		PrescriptionReq:   true,
	},
	{
		AIC:               "026089061",
		Name:              "Augmentin Sospensione Orale Ricostituita 70 ml",
		ActiveIngredient:  "Amoxicillina + Acido Clavulanico",
		Dosage:            "400 mg / 57 mg",
		Form:              "sospensione orale",
		Unit:              UnitMl,
		DefaultQuantity:   70,
		RequiresColdChain: true, // Una volta ricostituito va rigorosamente conservato in frigo 2-8°C
		DefaultLocation:   LocationFridge,
		DefaultPaoDays:    7, // Dura solo 7 giorni ricostituito in frigo!
		PrescriptionReq:   true,
	},
	{
		AIC:               "036494017",
		Name:              "Gaviscon Bruciore e Indigestione sospensione orale",
		ActiveIngredient:  "Sodio alginato + Calcio carbonato",
		Dosage:            "500 mg + 267 mg",
		Form:              "bustine",
		Unit:              UnitSachets,
		DefaultQuantity:   24,
		RequiresColdChain: false,
		DefaultLocation:   LocationBedsideTable,
		DefaultPaoDays:    0,
		PrescriptionReq:   false,
	},
	{
		AIC:               "019974011",
		Name:              "Bentelan 0,5 mg compresse effervescenti",
		ActiveIngredient:  "Betametasone",
		Dosage:            "0,5 mg",
		Form:              "compresse",
		Unit:              UnitTablets,
		DefaultQuantity:   10,
		RequiresColdChain: false,
		DefaultLocation:   LocationBathroomCabinet,
		DefaultPaoDays:    0,
		PrescriptionReq:   true,
	},
	{
		AIC:               "019974023",
		Name:              "Bentelan 1 mg compresse effervescenti",
		ActiveIngredient:  "Betametasone",
		Dosage:            "1 mg",
		Form:              "compresse",
		Unit:              UnitTablets,
		DefaultQuantity:   10,
		RequiresColdChain: false,
		DefaultLocation:   LocationBathroomCabinet,
		DefaultPaoDays:    0,
		PrescriptionReq:   true,
	},
	{
		AIC:               "028120016",
		Name:              "Voltaren Emulgel 1% gel 100g",
		ActiveIngredient:  "Diclofenac dietilammonio",
		Dosage:            "1%",
		Form:              "gel",
		Unit:              UnitPieces,
		DefaultQuantity:   1,
		RequiresColdChain: false,
		DefaultLocation:   LocationBathroomCabinet,
		DefaultPaoDays:    365,
		PrescriptionReq:   false,
	},
	{
		AIC:               "025647012",
		Name:              "Tobral Collirio 5 ml",
		ActiveIngredient:  "Tobramicina",
		Dosage:            "0.3%",
		Form:              "collirio",
		Unit:              UnitMl,
		DefaultQuantity:   5,
		RequiresColdChain: true, // Collirio termolabile
		DefaultLocation:   LocationFridge,
		DefaultPaoDays:    28, // Scade 28 giorni dopo l'apertura
		PrescriptionReq:   true,
	},
	{
		AIC:               "013046059",
		Name:              "Enterogermina 4 miliardi / 2 ml",
		ActiveIngredient:  "Spore di Bacillus clausii poliantibiotico-resistente",
		Dosage:            "4 miliardi",
		Form:              "flaconcini",
		Unit:              UnitPieces,
		DefaultQuantity:   10,
		RequiresColdChain: false,
		DefaultLocation:   LocationBathroomCabinet,
		DefaultPaoDays:    0,
		PrescriptionReq:   false,
	},
	{
		AIC:               "032906016",
		Name:              "Levotuss Sciroppo 200 ml",
		ActiveIngredient:  "Levodropropizina",
		Dosage:            "30 mg / 5 ml",
		Form:              "sciroppo",
		Unit:              UnitMl,
		DefaultQuantity:   200,
		RequiresColdChain: false,
		DefaultLocation:   LocationBathroomCabinet,
		DefaultPaoDays:    180,
		PrescriptionReq:   false,
	},
	{
		AIC:               "034873010",
		Name:              "Cardioaspirina 100 mg compresse gastroresistenti",
		ActiveIngredient:  "Acido acetilsalicilico",
		Dosage:            "100 mg",
		Form:              "compresse",
		Unit:              UnitTablets,
		DefaultQuantity:   30,
		RequiresColdChain: false,
		DefaultLocation:   LocationBedsideTable,
		DefaultPaoDays:    0,
		PrescriptionReq:   true,
	},
	{
		AIC:               "028430037",
		Name:              "Pantorc 20 mg compresse gastroresistenti",
		ActiveIngredient:  "Pantoprazolo",
		Dosage:            "20 mg",
		Form:              "compresse",
		Unit:              UnitTablets,
		DefaultQuantity:   28,
		RequiresColdChain: false,
		DefaultLocation:   LocationBedsideTable,
		DefaultPaoDays:    0,
		PrescriptionReq:   true,
	},
	{
		AIC:               "030800049",
		Name:              "Humalog 100 U/ml penne preriempite",
		ActiveIngredient:  "Insulina lispro",
		Dosage:            "100 U/ml",
		Form:              "penne",
		Unit:              UnitPieces,
		DefaultQuantity:   5,
		RequiresColdChain: true, // Conservazione tassativa in frigorifero (2-8°C)
		DefaultLocation:   LocationFridge,
		DefaultPaoDays:    28, // Una volta aperta la penna dura 28 giorni
		PrescriptionReq:   true,
	},
	{
		AIC:               "023103010",
		Name:              "Ventolin 100 mcg sospensione pressurizzata per inalazione",
		ActiveIngredient:  "Salbutamolo",
		Dosage:            "100 mcg",
		Form:              "inalatore",
		Unit:              UnitPieces,
		DefaultQuantity:   1,
		RequiresColdChain: false,
		DefaultLocation:   LocationTravelBag,
		DefaultPaoDays:    0,
		PrescriptionReq:   true,
	},
	{
		AIC:               "024402013",
		Name:              "Eutirox 100 mcg compresse",
		ActiveIngredient:  "Levotiroxina sodica",
		Dosage:            "100 mcg",
		Form:              "compresse",
		Unit:              UnitTablets,
		DefaultQuantity:   50,
		RequiresColdChain: false,
		DefaultLocation:   LocationBedsideTable,
		DefaultPaoDays:    0,
		PrescriptionReq:   true,
	},
}

// LookupByAIC finds an exact matching AIFA record by its 9-digit code.
func LookupByAIC(aic string) (*AIFARecord, bool) {
	aicClean := strings.TrimSpace(aic)
	for _, rec := range CommonAIFADatabase {
		if rec.AIC == aicClean {
			return &rec, true
		}
	}
	return nil, false
}

// SearchAIFA looks up medicines matching a search string by name or active ingredient.
func SearchAIFA(query string) []AIFARecord {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return CommonAIFADatabase
	}

	var results []AIFARecord
	for _, rec := range CommonAIFADatabase {
		if strings.Contains(strings.ToLower(rec.Name), q) ||
			strings.Contains(strings.ToLower(rec.ActiveIngredient), q) ||
			strings.Contains(rec.AIC, q) {
			results = append(results, rec)
		}
	}
	return results
}
