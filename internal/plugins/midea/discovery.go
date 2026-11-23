package midea

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const mideaDiscoveryTimeout = 5 * time.Second

var mideaDiscoveryPacket = func() []byte {
	packet, _ := hex.DecodeString("5a5a0111480092000000000000000000000000000000000000000000000000000000000000000000007f75bd6b3e4f8b762e849c6e578d6590036e9d4342a50f1f569eb8ec918e92e5")
	return packet
}()

// DiscoveredAppliance is a credential-free result of a local UDP discovery scan.
type DiscoveredAppliance struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Model    string   `json:"model,omitempty"`
	TypeCode string   `json:"type_code"`
	Protocol Protocol `json:"protocol"`
	Supported bool    `json:"supported"`
	Reason   string   `json:"reason,omitempty"`
}

// DiscoverLAN scans only IPv4 broadcast addresses on active local interfaces.
// It does not scan public ranges and never opens a socket to a cloud-reported IP.
func DiscoverLAN(ctx context.Context) ([]DiscoveredAppliance, error) {
	if len(mideaDiscoveryPacket) < 60 {
		return nil, fmt.Errorf("midea: local discovery payload is invalid")
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, fmt.Errorf("midea: open UDP discovery socket: %w", err)
	}
	defer conn.Close()
	if err := enableMideaBroadcast(conn); err != nil {
		return nil, fmt.Errorf("midea: enable IPv4 broadcast: %w", err)
	}
	if err := conn.SetReadBuffer(64 * 1024); err != nil {
		return nil, err
	}

	broadcasts, err := localBroadcastAddresses()
	if err != nil {
		return nil, err
	}
	if len(broadcasts) == 0 {
		return nil, errors.New("midea: no active private IPv4 interface found")
	}
	for _, address := range broadcasts {
		for _, port := range []int{6445, 20086} {
			if _, err := conn.WriteToUDP(mideaDiscoveryPacket, &net.UDPAddr{IP: address, Port: port}); err != nil {
				continue
			}
		}
	}

	deadline := time.Now().Add(mideaDiscoveryTimeout)
	if requested, ok := ctx.Deadline(); ok && requested.Before(deadline) {
		deadline = requested
	}
	found := make(map[string]DiscoveredAppliance)
	buffer := make([]byte, 2048)
	for {
		if err := ctx.Err(); err != nil {
			return sortedDiscovery(found), err
		}
		if !time.Now().Before(deadline) {
			break
		}
		readDeadline := time.Now().Add(250 * time.Millisecond)
		if deadline.Before(readDeadline) {
			readDeadline = deadline
		}
		_ = conn.SetReadDeadline(readDeadline)
		n, source, readErr := conn.ReadFromUDP(buffer)
		if readErr != nil {
			if timeoutErr, ok := readErr.(net.Error); ok && timeoutErr.Timeout() {
				continue
			}
			return sortedDiscovery(found), fmt.Errorf("midea: receive discovery datagram: %w", readErr)
		}
		if source == nil || !isPrivateLANIP(source.IP) {
			continue
		}
		device, parseErr := parseDiscoveryDatagram(buffer[:n], source.IP.String())
		if parseErr != nil {
			continue
		}
		found[device.ID] = device
	}
	return sortedDiscovery(found), nil
}

func enableMideaBroadcast(conn *net.UDPConn) error {
	if conn == nil {
		return errors.New("midea: nil discovery socket")
	}
	raw, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	var socketErr error
	if err := raw.Control(func(fd uintptr) {
		socketErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	}); err != nil {
		return err
	}
	return socketErr
}

func localBroadcastAddresses() ([]net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("midea: list network interfaces: %w", err)
	}
	seen := make(map[string]struct{})
	var result []net.IP
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ipNet, ok := address.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || !ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() {
				continue
			}
			mask := ipNet.Mask
			if len(mask) < net.IPv4len {
				continue
			}
			if len(mask) > net.IPv4len {
				mask = mask[len(mask)-net.IPv4len:]
			}
			broadcast := make(net.IP, net.IPv4len)
			for i := range broadcast {
				broadcast[i] = ip[i] | ^mask[i]
			}
			if _, exists := seen[broadcast.String()]; exists {
				continue
			}
			seen[broadcast.String()] = struct{}{}
			result = append(result, broadcast)
		}
	}
	return result, nil
}

func isPrivateLANIP(ip net.IP) bool {
	return ip != nil && ip.IsPrivate() && !ip.IsLoopback() && !ip.IsUnspecified() && !ip.IsMulticast()
}

func parseDiscoveryDatagram(raw []byte, host string) (DiscoveredAppliance, error) {
	var empty DiscoveredAppliance
	if len(raw) < 104 || len(raw) > 2048 || !isPrivateLANIP(net.ParseIP(host)) {
		return empty, ErrMideaFrame
	}
	protocol := ProtocolMSmartV2
	data := raw
	if raw[0] == 0x83 && raw[1] == 0x70 && len(raw) > 10 && raw[8] == 0x5A && raw[9] == 0x5A {
		protocol = ProtocolMSmartV3
		if len(raw) < 8+56+16 {
			return empty, ErrMideaFrame
		}
		data = raw[8 : len(raw)-16]
	}
	if len(data) < 56 || data[0] != 0x5A || data[1] != 0x5A {
		return empty, ErrMideaFrame
	}
	packetLength := int(binary.LittleEndian.Uint16(data[4:6]))
	if packetLength != len(data) || len(data)-mideaHeaderLength-mideaDigestLength <= 0 {
		return empty, ErrMideaFrame
	}
	hashInput := append(append([]byte(nil), data[:len(data)-mideaDigestLength]...), mideaLANSalt...)
	digest := md5.Sum(hashInput)
	if !bytes.Equal(digest[:], data[len(data)-mideaDigestLength:]) {
		return empty, ErrMideaFrame
	}
	var rawID [8]byte
	copy(rawID[:6], data[20:26])
	deviceID := binary.LittleEndian.Uint64(rawID[:])
	if deviceID == 0 {
		return empty, ErrMideaFrame
	}
	encrypted := data[mideaHeaderLength : len(data)-mideaDigestLength]
	block, err := aes.NewCipher(mideaLANKey)
	if err != nil {
		return empty, err
	}
	if len(encrypted) == 0 || len(encrypted)%aes.BlockSize != 0 {
		return empty, ErrMideaFrame
	}
	plain := make([]byte, len(encrypted))
	for offset := 0; offset < len(encrypted); offset += aes.BlockSize {
		block.Decrypt(plain[offset:offset+aes.BlockSize], encrypted[offset:offset+aes.BlockSize])
	}
	padding := int(plain[len(plain)-1])
	if padding == 0 || padding > aes.BlockSize || padding > len(plain) {
		return empty, ErrMideaFrame
	}
	for _, value := range plain[len(plain)-padding:] {
		if int(value) != padding {
			return empty, ErrMideaFrame
		}
	}
	reply := plain[:len(plain)-padding]
	if len(reply) < 42 {
		return empty, ErrMideaFrame
	}
	ssidLength := int(reply[40])
	if ssidLength == 0 || len(reply) < 41+ssidLength {
		return empty, ErrMideaFrame
	}
	ssid := strings.TrimSpace(string(reply[41 : 41+ssidLength]))
	parts := strings.Split(ssid, "_")
	if len(parts) < 2 {
		return empty, ErrMideaFrame
	}
	typeCode := strings.ToUpper(strings.TrimPrefix(strings.TrimSpace(parts[1]), "0X"))
	if len(typeCode) != 2 {
		return empty, ErrMideaFrame
	}
	if _, err := strconv.ParseUint(typeCode, 16, 8); err != nil {
		return empty, ErrMideaFrame
	}
	port64 := binary.LittleEndian.Uint32(reply[4:8])
	if port64 == 0 || port64 > 65535 {
		port64 = 6444
	}
	model := strings.TrimSpace(strings.Trim(string(reply[17:25]), "\x00"))
	name := "Midea appliance"
	supported := false
	reason := "No Domus driver is registered for this appliance type yet"
	if typeCode == "A1" {
		name, supported, reason = "Midea dehumidifier", true, ""
	}
	return DiscoveredAppliance{
		ID: strconv.FormatUint(deviceID, 10), Name: name, Host: host,
		Port: int(port64), Model: model, TypeCode: typeCode,
		Protocol: protocol, Supported: supported, Reason: reason,
	}, nil
}

func sortedDiscovery(devices map[string]DiscoveredAppliance) []DiscoveredAppliance {
	result := make([]DiscoveredAppliance, 0, len(devices))
	for _, device := range devices {
		result = append(result, device)
	}
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].ID < result[j-1].ID; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}
	return result
}
