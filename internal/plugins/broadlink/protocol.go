// Package broadlink implements the local UDP protocol used by BroadLink RM remotes.
package broadlink

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	udpPort       = 80
	commandRemote = 0x6a
	commandAuth   = 0x65
)

var (
	initialKey = [16]byte{0x09, 0x76, 0x28, 0x34, 0x3f, 0xe9, 0x9e, 0x23, 0x76, 0x5c, 0x15, 0x13, 0xac, 0xcf, 0x8b, 0x02}
	initialIV  = [16]byte{0x56, 0x2e, 0x17, 0x99, 0x6d, 0x09, 0x3d, 0x28, 0xdd, 0xb3, 0xba, 0x69, 0x5a, 0x2e, 0x6f, 0x58}
)

var ErrUnsupportedDevice = errors.New("broadlink: unsupported RM device model")

type DiscoveredDevice struct {
	MAC         string `json:"mac"`
	IP          string `json:"ip"`
	Port        int    `json:"port"`
	DeviceType  uint16 `json:"device_type"`
	Model       string `json:"model"`
	SupportsIR  bool   `json:"supports_ir"`
	SupportsRF  bool   `json:"supports_rf"`
}

// DeviceModel returns the canonical model and RF capability for supported RM remotes.
// It intentionally excludes plugs, sensors and other BroadLink product families.
func DeviceModel(deviceType uint16) (string, bool, bool) {
	switch deviceType {
	case 0x2737:
		return "RM Mini 3", true, false
	case 0x278f:
		return "RM Mini", true, false
	case 0x5f36, 0x6507, 0x6508, 0x27b7, 0x27c2, 0x27c7, 0x27cc, 0x27cd, 0x27d0, 0x27d1, 0x27d3, 0x27dc, 0x27de:
		return "RM Mini 3", true, false
	case 0x51da, 0x520c, 0x5216, 0x521c, 0x610e, 0x62bc, 0x648d, 0x653a:
		return "RM4 Mini", true, false
	case 0x520d, 0x5211, 0x6070, 0x610f, 0x62be, 0x6539:
		return "RM4C Mini", true, false
	case 0x5209, 0x5212:
		return "RM4 TV Mate", true, false
	case 0x2712, 0x272a, 0x273d, 0x277c, 0x2783, 0x2787, 0x278b, 0x2797, 0x279d, 0x27a1, 0x27a6, 0x27a9, 0x27c3:
		return "RM Pro", true, true
	case 0x520b, 0x5213, 0x6026, 0x61a2, 0x649b, 0x653c:
		return "RM4 Pro", true, true
	case 0x5218, 0x6184:
		return "RM4C Pro", true, true
	default:
		return "", false, false
	}
}

func usesModernRemoteFrame(deviceType uint16) bool {
	model, _, _ := DeviceModel(deviceType)
	if model == "RM4 Mini" || model == "RM4C Mini" || model == "RM4 TV Mate" || model == "RM4 Pro" || model == "RM4C Pro" {
		return true
	}
	switch deviceType {
	case 0x5f36, 0x6507, 0x6508, 0x27b7, 0x27c2, 0x27c7, 0x27cc, 0x27cd, 0x27d0, 0x27d1, 0x27d3, 0x27dc, 0x27de:
		return true
	default:
		return false
	}
}

func discoveryPacket(localIP net.IP, localPort int, now time.Time) []byte {
	packet := make([]byte, 0x30)
	_, offset := now.Zone()
	binary.LittleEndian.PutUint32(packet[0x08:0x0c], uint32(int32(offset)))
	binary.LittleEndian.PutUint16(packet[0x0c:0x0e], uint16(now.Year()))
	packet[0x0e] = byte(now.Second())
	packet[0x0f] = byte(now.Minute())
	packet[0x10] = byte(now.Hour())
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	packet[0x11] = byte(weekday)
	packet[0x12] = byte(now.Day())
	packet[0x13] = byte(now.Month())
	if ip := localIP.To4(); ip != nil {
		copy(packet[0x18:0x1c], ip)
	}
	binary.LittleEndian.PutUint16(packet[0x1c:0x1e], uint16(localPort))
	packet[0x26] = 0x06
	binary.LittleEndian.PutUint16(packet[0x20:0x22], checksum(packet))
	return packet
}

func parseDiscoveryResponse(packet []byte, source *net.UDPAddr) (DiscoveredDevice, bool) {
	if len(packet) < 0x40 || source == nil || source.IP == nil {
		return DiscoveredDevice{}, false
	}
	deviceType := binary.LittleEndian.Uint16(packet[0x34:0x36])
	model, ir, rf := DeviceModel(deviceType)
	if model == "" || !source.IP.IsPrivate() {
		return DiscoveredDevice{}, false
	}
	mac := make([]byte, 6)
	for i := range mac {
		mac[i] = packet[0x3f-i]
	}
	return DiscoveredDevice{
		MAC: strings.ToUpper(hex.EncodeToString(mac[0:1]) + ":" + hex.EncodeToString(mac[1:2]) + ":" + hex.EncodeToString(mac[2:3]) + ":" + hex.EncodeToString(mac[3:4]) + ":" + hex.EncodeToString(mac[4:5]) + ":" + hex.EncodeToString(mac[5:6])),
		IP: source.IP.String(), Port: source.Port, DeviceType: deviceType,
		Model: model, SupportsIR: ir, SupportsRF: rf,
	}, true
}

func parseMAC(raw string) ([6]byte, error) {
	var mac [6]byte
	clean := strings.NewReplacer(":", "", "-", "").Replace(strings.TrimSpace(raw))
	if len(clean) != 12 {
		return mac, fmt.Errorf("broadlink: invalid MAC address")
	}
	decoded, err := hex.DecodeString(clean)
	if err != nil {
		return mac, fmt.Errorf("broadlink: invalid MAC address: %w", err)
	}
	copy(mac[:], decoded)
	return mac, nil
}

// checksum is the BroadLink 16-bit additive checksum (seed 0xbeaf).
func checksum(data []byte) uint16 {
	sum := uint32(0xbeaf)
	for _, b := range data {
		sum += uint32(b)
	}
	return uint16(sum & 0xffff)
}

func encryptPayload(plain []byte, key [16]byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	if len(plain) == 0 {
		return nil, nil
	}
	padded := make([]byte, (len(plain)+aes.BlockSize-1)/aes.BlockSize*aes.BlockSize)
	copy(padded, plain)
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, initialIV[:]).CryptBlocks(out, padded)
	return out, nil
}

func decryptPayload(encrypted []byte, key [16]byte) ([]byte, error) {
	if len(encrypted) == 0 {
		return nil, nil
	}
	if len(encrypted)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("broadlink: encrypted payload has invalid length")
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(encrypted))
	cipher.NewCBCDecrypter(block, initialIV[:]).CryptBlocks(out, encrypted)
	return out, nil
}
