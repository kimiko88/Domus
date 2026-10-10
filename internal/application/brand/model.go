package brand

import "time"

// BrandIdentifier represents supported smart home hardware manufacturers.
type BrandIdentifier string

const (
	BrandShelly    BrandIdentifier = "shelly"
	BrandHue       BrandIdentifier = "philips_hue"
	BrandSonoff    BrandIdentifier = "sonoff"
	BrandTasmota   BrandIdentifier = "tasmota"
	BrandWLED      BrandIdentifier = "wled"
	BrandTuyaLocal BrandIdentifier = "tuya_local"
	BrandMatter    BrandIdentifier = "matter"
	BrandHomeKit   BrandIdentifier = "homekit"
	BrandZigbee    BrandIdentifier = "zigbee"
	BrandZWave     BrandIdentifier = "zwave"
)

// BrandCatalogEntry describes a brand's characteristics, communication protocols, and capabilities.
type BrandCatalogEntry struct {
	ID              BrandIdentifier `json:"id"`
	Name            string          `json:"name"`
	Category        string          `json:"category"`
	Protocol        string          `json:"protocol"`
	ZeroCloud       bool            `json:"zero_cloud"`
	DefaultPort     int             `json:"default_port"`
	Description     string          `json:"description"`
	SupportedModels []string        `json:"supported_models"`
	PairingGuide    string          `json:"pairing_guide"`
	Icon            string          `json:"icon"`
}

// SuggestedEntity represents an entity that will be automatically created upon pairing.
type SuggestedEntity struct {
	Domain            string `json:"domain"`
	Name              string `json:"name"`
	KeySuffix         string `json:"key_suffix"`
	UnitOfMeasurement string `json:"unit_of_measurement,omitempty"`
	Writable          bool   `json:"writable"`
	Icon              string `json:"icon,omitempty"`
}

// DiscoveredDevice represents an identified smart home device on the local network.
type DiscoveredDevice struct {
	ID                string            `json:"id"`
	Brand             BrandIdentifier   `json:"brand"`
	Name              string            `json:"name"`
	IP                string            `json:"ip"`
	Port              int               `json:"port"`
	MAC               string            `json:"mac,omitempty"`
	Model             string            `json:"model"`
	Firmware          string            `json:"firmware,omitempty"`
	Status            string            `json:"status"` // "discovered", "paired", "unreachable"
	Capabilities      []string          `json:"capabilities"`
	SuggestedEntities []SuggestedEntity `json:"suggested_entities"`
	DiscoveredAt      time.Time         `json:"discovered_at"`
}

// PairRequest represents the input parameters to pair a brand device.
type PairRequest struct {
	Brand     BrandIdentifier `json:"brand"`
	IP        string          `json:"ip"`
	Port      int             `json:"port,omitempty"`
	Name      string          `json:"name"`
	AreaID    string          `json:"area_id,omitempty"`
	AuthToken string          `json:"auth_token,omitempty"`
}

// PairedDeviceRuntime holds runtime state for interacting with a paired device.
type PairedDeviceRuntime struct {
	DeviceID   string          `json:"device_id"`
	Brand      BrandIdentifier `json:"brand"`
	IP         string          `json:"ip"`
	Port       int             `json:"port"`
	Model      string          `json:"model"`
	AuthToken  string          `json:"auth_token,omitempty"`
	EntityKeys []string        `json:"entity_keys"`
}
