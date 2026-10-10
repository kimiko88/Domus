package brand

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Scanner handles LAN probes to identify smart home devices across various protocols.
type Scanner struct {
	client *http.Client
}

// NewScanner creates a new scanner instance with a low probe timeout.
func NewScanner(client *http.Client) *Scanner {
	if client == nil {
		client = &http.Client{
			Timeout: 2 * time.Second,
		}
	}
	return &Scanner{client: client}
}

// ProbeIP interrogates a specific IP address against known brand signatures.
func (s *Scanner) ProbeIP(ctx context.Context, ip string, brandHint BrandIdentifier) (*DiscoveredDevice, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil, fmt.Errorf("scanner: ip cannot be empty")
	}

	// 1. If brandHint is provided, test it first
	switch brandHint {
	case BrandShelly:
		if dev, err := s.probeShelly(ctx, ip); err == nil && dev != nil {
			return dev, nil
		}
	case BrandWLED:
		if dev, err := s.probeWLED(ctx, ip); err == nil && dev != nil {
			return dev, nil
		}
	case BrandTasmota:
		if dev, err := s.probeTasmota(ctx, ip); err == nil && dev != nil {
			return dev, nil
		}
	case BrandSonoff:
		if dev, err := s.probeSonoff(ctx, ip); err == nil && dev != nil {
			return dev, nil
		}
	case BrandHue:
		if dev, err := s.probeHue(ctx, ip); err == nil && dev != nil {
			return dev, nil
		}
	}

	// 2. Sequential multi-brand probe order (fastest signatures first)
	if dev, err := s.probeShelly(ctx, ip); err == nil && dev != nil {
		return dev, nil
	}
	if dev, err := s.probeWLED(ctx, ip); err == nil && dev != nil {
		return dev, nil
	}
	if dev, err := s.probeTasmota(ctx, ip); err == nil && dev != nil {
		return dev, nil
	}
	if dev, err := s.probeSonoff(ctx, ip); err == nil && dev != nil {
		return dev, nil
	}
	if dev, err := s.probeHue(ctx, ip); err == nil && dev != nil {
		return dev, nil
	}

	return nil, fmt.Errorf("scanner: no supported smart device recognized at %s", ip)
}

// ScanTargets probes a list of target IPs concurrently.
func (s *Scanner) ScanTargets(ctx context.Context, ips []string) []*DiscoveredDevice {
	var results []*DiscoveredDevice
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, 16) // Max 16 concurrent probes

	for _, ip := range ips {
		wg.Add(1)
		go func(targetIP string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			dev, err := s.ProbeIP(ctx, targetIP, "")
			if err == nil && dev != nil {
				mu.Lock()
				results = append(results, dev)
				mu.Unlock()
			}
		}(ip)
	}

	wg.Wait()
	return results
}

// probeShelly tests both Gen2/Gen3 RPC and Gen1 HTTP APIs.
func (s *Scanner) probeShelly(ctx context.Context, ip string) (*DiscoveredDevice, error) {
	// Try Gen2/Gen3 RPC first
	gen2URL := fmt.Sprintf("http://%s/rpc/Shelly.GetDeviceInfo", ip)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gen2URL, nil)
	if err == nil {
		resp, err := s.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var data struct {
				ID      string `json:"id"`
				Model   string `json:"model"`
				Mac     string `json:"mac"`
				Gen     int    `json:"gen"`
				Ver     string `json:"ver"`
				App     string `json:"app"`
				Profile string `json:"profile"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil && data.ID != "" {
				name := data.App
				if name == "" {
					name = data.Model
				}
				if name == "" {
					name = "Shelly Plus/Pro"
				}

				return &DiscoveredDevice{
					ID:           fmt.Sprintf("shelly_%s", strings.ToLower(data.ID)),
					Brand:        BrandShelly,
					Name:         fmt.Sprintf("Shelly %s (%s)", name, ip),
					IP:           ip,
					Port:         80,
					MAC:          data.Mac,
					Model:        data.Model,
					Firmware:     data.Ver,
					Status:       "discovered",
					Capabilities: []string{"switch", "power_meter", "voltage_meter", "temperature"},
					SuggestedEntities: []SuggestedEntity{
						{Domain: "switch", Name: "Interruttore Principale", KeySuffix: "relay_0", Writable: true, Icon: "toggle-right"},
						{Domain: "sensor", Name: "Potenza Attiva", KeySuffix: "power", UnitOfMeasurement: "W", Writable: false, Icon: "zap"},
						{Domain: "sensor", Name: "Tensione Rete", KeySuffix: "voltage", UnitOfMeasurement: "V", Writable: false, Icon: "gauge"},
						{Domain: "sensor", Name: "Energia Totale", KeySuffix: "energy", UnitOfMeasurement: "kWh", Writable: false, Icon: "battery-charging"},
						{Domain: "sensor", Name: "Temperatura Interna", KeySuffix: "temperature", UnitOfMeasurement: "°C", Writable: false, Icon: "thermometer"},
					},
					DiscoveredAt: time.Now().UTC(),
				}, nil
			}
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
	}

	// Fallback to Gen1 HTTP API
	gen1URL := fmt.Sprintf("http://%s/shelly", ip)
	req1, err := http.NewRequestWithContext(ctx, http.MethodGet, gen1URL, nil)
	if err == nil {
		resp, err := s.client.Do(req1)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var data struct {
				Type string `json:"type"`
				Mac  string `json:"mac"`
				Fw   string `json:"fw"`
				Auth bool   `json:"auth"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil && data.Type != "" {
				return &DiscoveredDevice{
					ID:           fmt.Sprintf("shelly_%s", strings.ToLower(data.Mac)),
					Brand:        BrandShelly,
					Name:         fmt.Sprintf("Shelly %s (%s)", data.Type, ip),
					IP:           ip,
					Port:         80,
					MAC:          data.Mac,
					Model:        data.Type,
					Firmware:     data.Fw,
					Status:       "discovered",
					Capabilities: []string{"switch", "power_meter"},
					SuggestedEntities: []SuggestedEntity{
						{Domain: "switch", Name: "Interruttore", KeySuffix: "relay_0", Writable: true, Icon: "toggle-right"},
						{Domain: "sensor", Name: "Potenza Attiva", KeySuffix: "power", UnitOfMeasurement: "W", Writable: false, Icon: "zap"},
					},
					DiscoveredAt: time.Now().UTC(),
				}, nil
			}
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
	}

	return nil, fmt.Errorf("not shelly")
}

// probeWLED tests WLED /json/info API.
func (s *Scanner) probeWLED(ctx context.Context, ip string) (*DiscoveredDevice, error) {
	url := fmt.Sprintf("http://%s/json/info", ip)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status not ok")
	}

	var data struct {
		Ver  string `json:"ver"`
		Name string `json:"name"`
		Mac  string `json:"mac"`
		Leds struct {
			Count int `json:"count"`
			Fps   int `json:"fps"`
		} `json:"leds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || data.Ver == "" {
		return nil, fmt.Errorf("invalid wled json")
	}

	name := data.Name
	if name == "" {
		name = "WLED Strip"
	}

	return &DiscoveredDevice{
		ID:           fmt.Sprintf("wled_%s", strings.ToLower(strings.ReplaceAll(data.Mac, ":", ""))),
		Brand:        BrandWLED,
		Name:         fmt.Sprintf("%s (%s)", name, ip),
		IP:           ip,
		Port:         80,
		MAC:          data.Mac,
		Model:        fmt.Sprintf("WLED Controller (%d LEDs)", data.Leds.Count),
		Firmware:     data.Ver,
		Status:       "discovered",
		Capabilities: []string{"light", "rgb", "effects", "dimmer"},
		SuggestedEntities: []SuggestedEntity{
			{Domain: "light", Name: "Striscia LED Master", KeySuffix: "light_strip", Writable: true, Icon: "sparkles"},
			{Domain: "sensor", Name: "Numero LED", KeySuffix: "led_count", Writable: false, Icon: "cpu"},
		},
		DiscoveredAt: time.Now().UTC(),
	}, nil
}

// probeTasmota tests /cm?cmnd=Status%200 API.
func (s *Scanner) probeTasmota(ctx context.Context, ip string) (*DiscoveredDevice, error) {
	url := fmt.Sprintf("http://%s/cm?cmnd=Status%%200", ip)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status not ok")
	}

	var data struct {
		Status struct {
			DeviceName   string   `json:"DeviceName"`
			FriendlyName []string `json:"FriendlyName"`
			Topic        string   `json:"Topic"`
		} `json:"Status"`
		StatusFWR struct {
			Version string `json:"Version"`
		} `json:"StatusFWR"`
		StatusNET struct {
			Mac string `json:"Mac"`
		} `json:"StatusNET"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || (data.Status.Topic == "" && data.StatusFWR.Version == "") {
		return nil, fmt.Errorf("invalid tasmota response")
	}

	name := data.Status.DeviceName
	if name == "" && len(data.Status.FriendlyName) > 0 {
		name = data.Status.FriendlyName[0]
	}
	if name == "" {
		name = "Dispositivo Tasmota"
	}

	return &DiscoveredDevice{
		ID:           fmt.Sprintf("tasmota_%s", strings.ToLower(strings.ReplaceAll(data.StatusNET.Mac, ":", ""))),
		Brand:        BrandTasmota,
		Name:         fmt.Sprintf("%s (%s)", name, ip),
		IP:           ip,
		Port:         80,
		MAC:          data.StatusNET.Mac,
		Model:        "Tasmota ESP",
		Firmware:     data.StatusFWR.Version,
		Status:       "discovered",
		Capabilities: []string{"switch", "power_meter"},
		SuggestedEntities: []SuggestedEntity{
			{Domain: "switch", Name: "Interruttore", KeySuffix: "switch", Writable: true, Icon: "toggle-right"},
			{Domain: "sensor", Name: "Potenza Attiva", KeySuffix: "power", UnitOfMeasurement: "W", Writable: false, Icon: "zap"},
		},
		DiscoveredAt: time.Now().UTC(),
	}, nil
}

// probeSonoff tests Sonoff LAN /zeroconf/info on port 8081.
func (s *Scanner) probeSonoff(ctx context.Context, ip string) (*DiscoveredDevice, error) {
	url := fmt.Sprintf("http://%s:8081/zeroconf/info", ip)
	bodyPayload := []byte(`{"deviceid":"","data":{}}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status not ok")
	}

	var data struct {
		Seq   int `json:"seq"`
		Error int `json:"error"`
		Data  struct {
			Switch  string `json:"switch"`
			Startup string `json:"startup"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || data.Error != 0 {
		return nil, fmt.Errorf("invalid sonoff lan payload")
	}

	return &DiscoveredDevice{
		ID:           fmt.Sprintf("sonoff_%s", strings.ReplaceAll(ip, ".", "_")),
		Brand:        BrandSonoff,
		Name:         fmt.Sprintf("Sonoff LAN (%s)", ip),
		IP:           ip,
		Port:         8081,
		Model:        "Sonoff DIY Switch",
		Firmware:     "eWeLink LAN v3",
		Status:       "discovered",
		Capabilities: []string{"switch"},
		SuggestedEntities: []SuggestedEntity{
			{Domain: "switch", Name: "Interruttore Sonoff", KeySuffix: "switch", Writable: true, Icon: "toggle-right"},
		},
		DiscoveredAt: time.Now().UTC(),
	}, nil
}

// probeHue tests Philips Hue Bridge signatures.
func (s *Scanner) probeHue(ctx context.Context, ip string) (*DiscoveredDevice, error) {
	url := fmt.Sprintf("http://%s/api/config", ip)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status not ok")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Name       string `json:"name"`
		BridgeID   string `json:"bridgeid"`
		ModelID    string `json:"modelid"`
		APIVersion string `json:"apiversion"`
		SWVersion  string `json:"swversion"`
	}
	if err := json.Unmarshal(body, &data); err != nil || (data.BridgeID == "" && !strings.Contains(string(body), "Philips hue")) {
		return nil, fmt.Errorf("not hue bridge")
	}

	name := data.Name
	if name == "" {
		name = "Philips Hue Bridge"
	}

	return &DiscoveredDevice{
		ID:           fmt.Sprintf("hue_%s", strings.ToLower(data.BridgeID)),
		Brand:        BrandHue,
		Name:         fmt.Sprintf("%s (%s)", name, ip),
		IP:           ip,
		Port:         80,
		MAC:          data.BridgeID,
		Model:        "Hue Bridge v2",
		Firmware:     data.SWVersion,
		Status:       "discovered",
		Capabilities: []string{"light", "bridge", "rgb", "color_temp"},
		SuggestedEntities: []SuggestedEntity{
			{Domain: "light", Name: "Tutte le Luci Hue", KeySuffix: "all_lights", Writable: true, Icon: "sun"},
		},
		DiscoveredAt: time.Now().UTC(),
	}, nil
}
