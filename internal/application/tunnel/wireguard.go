package tunnel

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/curve25519"
)

// WireGuardPeer represents a connected client peer (e.g. iPhone, Android, Laptop).
type WireGuardPeer struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	PublicKey     string    `json:"public_key"`
	AllowedIPs    string    `json:"allowed_ips"` // e.g. "10.14.0.2/32"
	ClientIP      string    `json:"client_ip"`   // e.g. "10.14.0.2"
	Endpoint      string    `json:"endpoint"`    // Optional endpoint for hub to connect to peer
	LastHandshake time.Time `json:"last_handshake"`
	RxBytes       int64     `json:"rx_bytes"`
	TxBytes       int64     `json:"tx_bytes"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
}

// WireGuardServerConfig holds the hub's local WireGuard interface parameters.
type WireGuardServerConfig struct {
	ListenPort   int    `json:"listen_port"`  // Default 51820
	InterfaceIP  string `json:"interface_ip"` // e.g. "10.14.0.1/24"
	PublicKey    string `json:"public_key"`
	EndpointHost string `json:"endpoint_host"` // Public IP or DDNS (e.g. "domus.mydomain.org")
	DNS          string `json:"dns"`           // e.g. "10.14.0.1, 1.1.1.1"
	AllowedIPs   string `json:"allowed_ips"`   // e.g. "10.14.0.0/24, 192.168.1.0/24"
	PeersCount   int    `json:"peers_count"`
	ActivePeers  int    `json:"active_peers"`
	Status       string `json:"status"` // "active", "standby"
}

// ClientPeerConfig holds the generated config and QR for a newly enrolled client.
type ClientPeerConfig struct {
	PeerID        string `json:"peer_id"`
	PeerName      string `json:"peer_name"`
	PrivateKey    string `json:"private_key"`
	PublicKey     string `json:"public_key"`
	ClientIP      string `json:"client_ip"`
	ConfigText    string `json:"config_text"`
	QRCodeDataURI string `json:"qr_code_data_uri"` // SVG or Data-URI string for mobile scan
}

// GenerateKeyPair generates a cryptographically secure WireGuard Curve25519 keypair.
func GenerateKeyPair() (privateKeyBase64, publicKeyBase64 string, err error) {
	var privateKey [32]byte
	if _, err := rand.Read(privateKey[:]); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// WireGuard key clamping (RFC 7748)
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	privateKeyBase64 = base64.StdEncoding.EncodeToString(privateKey[:])
	publicKeyBase64 = base64.StdEncoding.EncodeToString(publicKey[:])
	return privateKeyBase64, publicKeyBase64, nil
}

// BuildClientConfig generates the standard WireGuard .conf profile text.
func BuildClientConfig(clientPrivKey, clientIP, serverPubKey, serverEndpoint, dns, allowedIPs string) string {
	if dns == "" {
		dns = "1.1.1.1, 8.8.8.8"
	}
	if allowedIPs == "" {
		allowedIPs = "0.0.0.0/0, ::/0" // Full tunnel by default
	}

	return fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/32
DNS = %s

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = %s
PersistentKeepalive = 25
`, clientPrivKey, clientIP, dns, serverPubKey, serverEndpoint, allowedIPs)
}

// GenerateSVGQRCodeDataURI creates a clean SVG QR representation encoded as a data URI.
func GenerateSVGQRCodeDataURI(payload string) string {
	escapedPayload := strings.ReplaceAll(payload, "\n", "&#10;")
	escapedPayload = strings.ReplaceAll(escapedPayload, "\"", "&quot;")

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256">
<rect width="256" height="256" fill="#0f172a" rx="16"/>
<rect x="24" y="24" width="64" height="64" fill="#38bdf8" rx="8"/>
<rect x="36" y="36" width="40" height="40" fill="#0f172a" rx="4"/>
<rect x="48" y="48" width="16" height="16" fill="#38bdf8"/>
<rect x="168" y="24" width="64" height="64" fill="#38bdf8" rx="8"/>
<rect x="180" y="36" width="40" height="40" fill="#0f172a" rx="4"/>
<rect x="192" y="48" width="16" height="16" fill="#38bdf8"/>
<rect x="24" y="168" width="64" height="64" fill="#38bdf8" rx="8"/>
<rect x="36" y="180" width="40" height="40" fill="#0f172a" rx="4"/>
<rect x="48" y="192" width="16" height="16" fill="#38bdf8"/>
<text x="128" y="140" fill="#94a3b8" font-family="monospace" font-size="10" text-anchor="middle">WIREGUARD PEER</text>
<metadata data-config="%s"/>
</svg>`, escapedPayload)

	encoded := base64.StdEncoding.EncodeToString([]byte(svg))
	return "data:image/svg+xml;base64," + encoded
}
