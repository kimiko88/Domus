// Package automation provides ready-to-use blueprints and automation templates.
package automation

import (
	"fmt"
	"time"

	domauto "github.com/domus-hub/domus/internal/domain/automation"
	"github.com/google/uuid"
)

// BlueprintParameter describes an input entity required to instantiate a blueprint.
type BlueprintParameter struct {
	Key         string `json:"key"`         // e.g. "window_sensor"
	Name        string `json:"name"`        // e.g. "Sensore Finestra"
	Description string `json:"description"` // e.g. "Sensore magnetico apertura/chiusura"
	Domain      string `json:"domain"`      // e.g. "binary_sensor"
}

// Blueprint represents a pre-packaged automation template.
type Blueprint struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Category    string               `json:"category"` // "comfort", "energy", "security", "lighting"
	Parameters  []BlueprintParameter `json:"parameters"`
	Factory     func(params map[string]string) (*domauto.Automation, error) `json:"-"`
}

// Catalog holds the collection of built-in automation blueprints.
var Catalog = []Blueprint{
	{
		ID:          "window_climate_saver",
		Title:       "Finestra Aperta con Clima Acceso",
		Description: "Spegne automaticamente il riscaldamento/clima se una finestra o porta rimane aperta per oltre 60 secondi.",
		Category:    "energy",
		Parameters: []BlueprintParameter{
			{Key: "window_sensor", Name: "Sensore Finestra", Description: "Contatto magnetico finestra/porta", Domain: "binary_sensor"},
			{Key: "climate_entity", Name: "Climatizzatore / Valvola", Description: "Termostato o clima da spegnere", Domain: "climate"},
		},
		Factory: func(params map[string]string) (*domauto.Automation, error) {
			win := params["window_sensor"]
			clima := params["climate_entity"]
			if win == "" || clima == "" {
				return nil, fmt.Errorf("parametri mancanti: window_sensor e climate_entity sono obbligatori")
			}
			return &domauto.Automation{
				ID:          uuid.NewString(),
				Name:        "Spegni Clima su Finestra Aperta",
				Description: "Generato da Blueprint",
				Enabled:     true,
				Triggers: []domauto.Trigger{
					{Platform: "state", EntityKey: win, To: "on", ForSeconds: 60},
				},
				Conditions: []domauto.Condition{
					{Condition: "state", EntityKey: clima, State: "on"},
				},
				Actions: []domauto.Action{
					{Type: "service", Service: "turn_off", TargetEntityKey: clima},
				},
				ExecutionMode: "single",
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			}, nil
		},
	},
	{
		ID:          "motion_night_light",
		Title:       "Luce Notturna Guidata con Movimento",
		Description: "Accende una luce di cortesia attenuata al 20% quando viene rilevato movimento tra le 23:00 e le 06:00, e la spegne dopo 2 minuti.",
		Category:    "lighting",
		Parameters: []BlueprintParameter{
			{Key: "motion_sensor", Name: "Sensore Movimento", Description: "Sensore PIR o radar mmWave", Domain: "binary_sensor"},
			{Key: "light_entity", Name: "Luce Notturna", Description: "Luce dimmerabile corridoio o bagno", Domain: "light"},
		},
		Factory: func(params map[string]string) (*domauto.Automation, error) {
			motion := params["motion_sensor"]
			light := params["light_entity"]
			if motion == "" || light == "" {
				return nil, fmt.Errorf("parametri mancanti: motion_sensor e light_entity sono obbligatori")
			}
			return &domauto.Automation{
				ID:          uuid.NewString(),
				Name:        "Luce Notturna su Movimento",
				Description: "Generato da Blueprint",
				Enabled:     true,
				Triggers: []domauto.Trigger{
					{Platform: "state", EntityKey: motion, To: "on"},
				},
				Conditions: []domauto.Condition{
					{Condition: "time", After: "23:00:00", Before: "06:00:00"},
				},
				Actions: []domauto.Action{
					{Type: "service", Service: "turn_on", TargetEntityKey: light, Params: map[string]any{"brightness": 50}},
					{Type: "delay", DelayMs: 120000},
					{Type: "service", Service: "turn_off", TargetEntityKey: light},
				},
				ExecutionMode: "restart",
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			}, nil
		},
	},
	{
		ID:          "laundry_cycle_finished",
		Title:       "Notifica Lavatrice Terminata",
		Description: "Invia una notifica quando il consumo della presa scende sotto i 3 Watt per oltre 2 minuti consecutivi.",
		Category:    "comfort",
		Parameters: []BlueprintParameter{
			{Key: "plug_power_sensor", Name: "Sensore Potenza Presa", Description: "Presa con misurazione consumi W", Domain: "sensor"},
		},
		Factory: func(params map[string]string) (*domauto.Automation, error) {
			plug := params["plug_power_sensor"]
			if plug == "" {
				return nil, fmt.Errorf("parametro mancante: plug_power_sensor è obbligatorio")
			}
			threeW := 3.0
			return &domauto.Automation{
				ID:          uuid.NewString(),
				Name:        "Avviso Lavatrice Conclusa",
				Description: "Generato da Blueprint",
				Enabled:     true,
				Triggers: []domauto.Trigger{
					{Platform: "numeric_state", EntityKey: plug, Below: &threeW, ForSeconds: 120},
				},
				Conditions: []domauto.Condition{},
				Actions: []domauto.Action{
					{
						Type:      "event",
						EventType: "notification.created",
						EventPayload: map[string]any{
							"title":   "🧺 Ciclo Bucato Completato",
							"message": "La lavatrice ha terminato il ciclo di lavaggio.",
							"type":    "info",
						},
					},
				},
				ExecutionMode: "single",
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			}, nil
		},
	},
}

// GetBlueprint finds a blueprint by its identifier.
func GetBlueprint(id string) (*Blueprint, bool) {
	for _, b := range Catalog {
		if b.ID == id {
			return &b, true
		}
	}
	return nil, false
}
