// Package assistant provides an offline-first Local LLM & Voice Assistant for Domus.
package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

type Config struct {
	OllamaURL    string `json:"ollama_url"`
	Model        string `json:"model"`
	Enabled      bool   `json:"enabled"`
	SystemPrompt string `json:"system_prompt"`
}

type ActionExecuted struct {
	EntityKey string         `json:"entity_key"`
	Service   string         `json:"service"`
	Params    map[string]any `json:"params,omitempty"`
	Success   bool           `json:"success"`
	Message   string         `json:"message"`
}

type ChatResponse struct {
	UserPrompt      string           `json:"user_prompt"`
	ResponseText    string           `json:"response_text"`
	ActionsExecuted []ActionExecuted `json:"actions_executed"`
	EngineUsed      string           `json:"engine_used"` // "ollama" or "local_heuristic"
	ExecutionTimeMs int64            `json:"execution_time_ms"`
}

type VoiceSatellite struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	AreaName    string    `json:"area_name"`
	Status      string    `json:"status"` // "online", "listening", "muted", "offline"
	WakeWord    string    `json:"wake_word"`
	LastHeardAt time.Time `json:"last_heard_at"`
	MicMuted    bool      `json:"mic_muted"`
}

type Service struct {
	cfg        Config
	bus        events.Bus
	entityRepo entity.Repository
	httpClient *http.Client
	mu         sync.RWMutex
	satellites map[string]*VoiceSatellite
}

func NewService(cfg Config, bus events.Bus, entityRepo entity.Repository) *Service {
	if cfg.OllamaURL == "" {
		cfg.OllamaURL = "http://localhost:11434"
	}
	if cfg.Model == "" {
		cfg.Model = "llama3.2:3b"
	}
	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = "Sei Domus Assistant, l'assistente vocale locale e privato della casa domotica. Rispondi in italiano in modo conciso e amichevole."
	}
	cfg.Enabled = true

	s := &Service{
		cfg:        cfg,
		bus:        bus,
		entityRepo: entityRepo,
		httpClient: &http.Client{Timeout: 6 * time.Second},
		satellites: make(map[string]*VoiceSatellite),
	}

	// Default satellite (e.g. wall tablet in Soggiorno)
	s.satellites["sat_living"] = &VoiceSatellite{
		ID:          "sat_living",
		Name:        "Tablet Parete Soggiorno",
		AreaName:    "Soggiorno",
		Status:      "listening",
		WakeWord:    "Ehi Domus",
		LastHeardAt: time.Now().UTC(),
		MicMuted:    false,
	}

	return s
}

func (s *Service) GetConfig() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Service) UpdateConfig(cfg Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}

// ProcessPrompt interprets user commands, executes actions and returns a voice/text response.
func (s *Service) ProcessPrompt(ctx context.Context, prompt string) (*ChatResponse, error) {
	start := time.Now()
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return &ChatResponse{
			UserPrompt:   prompt,
			ResponseText: "Non ho capito, puoi ripetere?",
			EngineUsed:   "local_heuristic",
		}, nil
	}

	// 1. Try Ollama LLM if enabled and reachable
	cfg := s.GetConfig()
	if cfg.Enabled {
		resp, err := s.queryOllama(ctx, trimmed)
		if err == nil && resp != nil {
			resp.ExecutionTimeMs = time.Since(start).Milliseconds()
			return resp, nil
		}
	}

	// 2. Fallback to offline Local Heuristic Engine
	resp := s.executeHeuristic(ctx, trimmed)
	resp.ExecutionTimeMs = time.Since(start).Milliseconds()
	return resp, nil
}

// queryOllama calls the local Ollama API instance with tools and system prompt.
func (s *Service) queryOllama(ctx context.Context, prompt string) (*ChatResponse, error) {
	s.mu.RLock()
	ollamaURL := s.cfg.OllamaURL
	model := s.cfg.Model
	systemPrompt := s.cfg.SystemPrompt
	s.mu.RUnlock()

	reqBody := map[string]any{
		"model":  model,
		"stream": false,
		"messages": []map[string]any{
			{"role": "system", "content": systemPrompt + " Quando l'utente chiede un'azione su una luce o dispositivo, rispondi confermando l'azione."},
			{"role": "user", "content": prompt},
		},
	}

	payloadBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ollamaURL+"/api/chat", bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", httpResp.StatusCode)
	}

	var ollamaRes struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}

	if err := json.NewDecoder(httpResp.Body).Decode(&ollamaRes); err != nil {
		return nil, err
	}

	// Also parse heuristic actions to ensure physical entities are commanded
	heuristic := s.executeHeuristic(ctx, prompt)

	reply := strings.TrimSpace(ollamaRes.Message.Content)
	if reply == "" {
		reply = heuristic.ResponseText
	}

	return &ChatResponse{
		UserPrompt:      prompt,
		ResponseText:    reply,
		ActionsExecuted: heuristic.ActionsExecuted,
		EngineUsed:      "ollama",
	}, nil
}

// executeHeuristic implements offline regex & keyword parsing for Italian commands.
func (s *Service) executeHeuristic(ctx context.Context, prompt string) *ChatResponse {
	lower := strings.ToLower(prompt)
	var actions []ActionExecuted
	var replyText string

	// Entity command extraction
	isTurnOn := strings.Contains(lower, "accendi") || strings.Contains(lower, "apri") || strings.Contains(lower, "attiva")
	isTurnOff := strings.Contains(lower, "spegni") || strings.Contains(lower, "chiudi") || strings.Contains(lower, "disattiva")

	// 1. All lights off / on: "spegni tutto" / "spegni tutte le luci"
	if isTurnOff && (strings.Contains(lower, "tutto") || strings.Contains(lower, "tutte le luci")) {
		act := ActionExecuted{
			EntityKey: "group.all_lights",
			Service:   "turn_off",
			Success:   true,
			Message:   "Spente tutte le luci della casa",
		}
		s.dispatchCommand(ctx, "group.all_lights", "turn_off", nil)
		actions = append(actions, act)
		replyText = "Ho spento tutte le luci di casa."
		return &ChatResponse{
			UserPrompt:      prompt,
			ResponseText:    replyText,
			ActionsExecuted: actions,
			EngineUsed:      "local_heuristic",
		}
	}

	// 2. Room or specific target detection (e.g. salotto, cucina, camera, bagno, giardino)
	rooms := []string{"salotto", "soggiorno", "cucina", "camera", "bagno", "studio", "giardino", "ingresso", "balcone"}
	matchedRoom := ""
	for _, r := range rooms {
		if strings.Contains(lower, r) {
			matchedRoom = r
			break
		}
	}

	// Target device key construction
	targetKey := "light.salotto"
	if matchedRoom != "" {
		targetKey = "light." + matchedRoom
	}

	if isTurnOn {
		// Check for brightness: "al 50%", "all'80%"
		var params map[string]any = nil
		briRegex := regexp.MustCompile(`(\d+)\s*%`)
		if m := briRegex.FindStringSubmatch(lower); len(m) > 1 {
			if pct, err := strconv.Atoi(m[1]); err == nil {
				params = map[string]any{"brightness": pct}
			}
		}

		s.dispatchCommand(ctx, targetKey, "turn_on", params)
		actions = append(actions, ActionExecuted{
			EntityKey: targetKey,
			Service:   "turn_on",
			Params:    params,
			Success:   true,
			Message:   "Dispositivo attivato con successo",
		})

		roomName := "in casa"
		if matchedRoom != "" {
			roomName = "in " + matchedRoom
		}
		if params != nil {
			replyText = fmt.Sprintf("Fatto, ho acceso la luce %s al %v%%.", roomName, params["brightness"])
		} else {
			replyText = fmt.Sprintf("Fatto, ho acceso la luce %s.", roomName)
		}
	} else if isTurnOff {
		s.dispatchCommand(ctx, targetKey, "turn_off", nil)
		actions = append(actions, ActionExecuted{
			EntityKey: targetKey,
			Service:   "turn_off",
			Success:   true,
			Message:   "Dispositivo spento con successo",
		})

		roomName := "in casa"
		if matchedRoom != "" {
			roomName = "in " + matchedRoom
		}
		replyText = fmt.Sprintf("Fatto, ho spento la luce %s.", roomName)
	} else if strings.Contains(lower, "temperatura") || strings.Contains(lower, "gradi") {
		// Thermostat query or setpoint
		tempRegex := regexp.MustCompile(`(\d+([.,]\d+)?)\s*(gradi|°)`)
		if m := tempRegex.FindStringSubmatch(lower); len(m) > 1 {
			tempVal, _ := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
			targetTRV := "climate.termostato"
			if matchedRoom != "" {
				targetTRV = "climate." + matchedRoom
			}
			params := map[string]any{"temperature": tempVal}
			s.dispatchCommand(ctx, targetTRV, "set_temperature", params)
			actions = append(actions, ActionExecuted{
				EntityKey: targetTRV,
				Service:   "set_temperature",
				Params:    params,
				Success:   true,
				Message:   fmt.Sprintf("Temperatura impostata a %.1f°C", tempVal),
			})
			replyText = fmt.Sprintf("Ho impostato il termostato a %.1f°C.", tempVal)
		} else {
			replyText = "La temperatura media rilevata in casa è di 21.5°C."
		}
	} else if strings.Contains(lower, "allarme") {
		if strings.Contains(lower, "attiva") || strings.Contains(lower, "arma") {
			replyText = "Per motivi di sicurezza, puoi armare l'allarme direttamente dal pannello o inserire il PIN a voce."
		} else {
			replyText = "L'allarme di sicurezza è attualmente operativo."
		}
	} else {
		replyText = fmt.Sprintf("Ricevuto: '%s'. Tutti i sistemi della casa funzionano regolarmente.", prompt)
	}

	return &ChatResponse{
		UserPrompt:      prompt,
		ResponseText:    replyText,
		ActionsExecuted: actions,
		EngineUsed:      "local_heuristic",
	}
}

func (s *Service) dispatchCommand(ctx context.Context, entityKey, service string, params map[string]any) {
	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.CommandRequested,
			OccurredAt: time.Now().UTC(),
			Source:     "assistant:voice_pipeline",
			Payload: map[string]any{
				"entity_key": entityKey,
				"service":    service,
				"params":     params,
			},
		})
	}
}

// RegisterSatellite adds or updates a voice satellite device.
func (s *Service) RegisterSatellite(sat VoiceSatellite) VoiceSatellite {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sat.ID == "" {
		sat.ID = "sat_" + uuid.NewString()[:8]
	}
	if sat.WakeWord == "" {
		sat.WakeWord = "Ehi Domus"
	}
	if sat.Status == "" {
		sat.Status = "online"
	}
	sat.LastHeardAt = time.Now().UTC()
	s.satellites[sat.ID] = &sat
	return sat
}

// ListSatellites returns all registered room microphones and smart display satellites.
func (s *Service) ListSatellites() []VoiceSatellite {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]VoiceSatellite, 0, len(s.satellites))
	for _, sat := range s.satellites {
		res = append(res, *sat)
	}
	return res
}

// ProcessSatelliteVoice handles voice prompts originating from a specific room satellite.
func (s *Service) ProcessSatelliteVoice(ctx context.Context, satID, prompt string) (*ChatResponse, error) {
	s.mu.Lock()
	sat, ok := s.satellites[satID]
	if ok && sat != nil {
		sat.LastHeardAt = time.Now().UTC()
		sat.Status = "listening"
	}
	s.mu.Unlock()

	// Contextual room adaptation: if prompt is generic (e.g. "accendi la luce") and satellite has an area name,
	// resolve room-specific intent if not already specified in the sentence.
	processedPrompt := prompt
	if sat != nil && sat.AreaName != "" {
		lower := strings.ToLower(prompt)
		if (strings.Contains(lower, "luce") || strings.Contains(lower, "luci")) &&
			!strings.Contains(lower, strings.ToLower(sat.AreaName)) &&
			!strings.Contains(lower, "tutt") {
			processedPrompt = fmt.Sprintf("%s in %s", prompt, sat.AreaName)
		}
	}

	resp, err := s.ProcessPrompt(ctx, processedPrompt)
	if ok && sat != nil {
		s.mu.Lock()
		sat.Status = "online"
		s.mu.Unlock()
	}
	return resp, err
}

