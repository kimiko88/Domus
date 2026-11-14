package tuya_local

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// DiscoveredDevice represents a Tuya / Smart Life device broadcast on the local network.
type DiscoveredDevice struct {
	IP         string `json:"ip"`
	DeviceID   string `json:"gwId"`
	Active     int    `json:"active"`
	Version    string `json:"version"`
	ProductKey string `json:"productKey"`
}

var tuyaDiscoveryKey = md5.Sum([]byte("yGAdlopoPVldABfn"))

const (
	tuyaDiscoveryInterval = 6 * time.Second
	tuya6699HeaderSize    = 18
	tuya6699FooterSize    = 4
	tuyaGCMNonceSize      = 12
	tuyaGCMTagSize        = 16
)

// DiscoveryListener listens for Tuya discovery packets and actively solicits
// replies from newer v3.5 devices.
type DiscoveryListener struct {
	ports       []int
	onDevice    func(DiscoveredDevice)
	stopCh      chan struct{}
	wg          sync.WaitGroup
	lifecycleMu sync.Mutex
	mu          sync.RWMutex
	devices     map[string]DiscoveredDevice
	started     bool
	sequence    uint32
}

// NewDiscoveryListener creates a listener for Tuya UDP broadcasts.
func NewDiscoveryListener(onDevice func(DiscoveredDevice)) *DiscoveryListener {
	return &DiscoveryListener{
		ports:    []int{6666, 6667, 7000},
		onDevice: onDevice,
		devices:  make(map[string]DiscoveredDevice),
	}
}

// Start opens UDP listeners on the configured ports. It is safe to call repeatedly.
func (l *DiscoveryListener) Start(ctx context.Context) error {
	l.lifecycleMu.Lock()
	defer l.lifecycleMu.Unlock()

	l.mu.Lock()
	if l.started {
		l.mu.Unlock()
		return nil
	}
	l.stopCh = make(chan struct{})
	stopCh := l.stopCh
	l.started = true
	l.mu.Unlock()

	bound := 0
	bound7000 := false
	for _, port := range l.ports {
		addr := net.UDPAddr{Port: port, IP: net.IPv4zero}
		conn, err := net.ListenUDP("udp4", &addr)
		if err != nil {
			slog.Warn("tuya_local: could not bind UDP discovery port", "port", port, "error", err)
			continue
		}
		bound++
		if port == 7000 {
			bound7000 = true
		}
		l.wg.Add(1)
		go l.listenLoop(conn, stopCh)
	}
	if bound == 0 {
		l.mu.Lock()
		l.started = false
		close(stopCh)
		l.mu.Unlock()
		return errors.New("tuya_local: no discovery UDP port could be bound")
	}

	if bound7000 {
		l.wg.Add(1)
		go l.solicitationLoop(stopCh)
	} else {
		slog.Warn("tuya_local: UDP 7000 is unavailable; v3.5 active discovery replies cannot be received")
	}

	go func() {
		select {
		case <-ctx.Done():
			l.Stop()
		case <-stopCh:
		}
	}()
	return nil
}

// Stop closes the UDP listeners. It is safe to call repeatedly and after a failed Start.
func (l *DiscoveryListener) Stop() {
	l.lifecycleMu.Lock()
	defer l.lifecycleMu.Unlock()

	l.mu.Lock()
	if !l.started {
		l.mu.Unlock()
		return
	}
	l.started = false
	close(l.stopCh)
	l.mu.Unlock()
	l.wg.Wait()
}

func validDeviceID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') &&
			!(r >= '0' && r <= '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func (l *DiscoveryListener) listenLoop(conn *net.UDPConn, stopCh <-chan struct{}) {
	defer l.wg.Done()
	defer conn.Close()

	buf := make([]byte, 4096)
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		n, source, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}
		if n <= 0 || source == nil || source.IP == nil {
			continue
		}
		if !(source.IP.IsPrivate() || source.IP.IsLoopback() || source.IP.IsLinkLocalUnicast()) {
			continue
		}

		dev, err := decodeTuyaDiscoveryPacket(buf[:n])
		if err != nil || !validDeviceID(dev.DeviceID) {
			continue
		}

		// Never trust the address embedded in an unauthenticated broadcast packet.
		// The UDP peer address is the only source of the device's LAN IP.
		dev.IP = source.IP.String()
		l.handleDiscovered(dev)
	}
}

func (l *DiscoveryListener) solicitationLoop(stopCh <-chan struct{}) {
	defer l.wg.Done()

	// Send immediately on startup, then periodically because many 3.5 devices
	// only respond after receiving an app-style REQ_DEVINFO broadcast.
	l.sendSolicitations()
	ticker := time.NewTicker(tuyaDiscoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			l.sendSolicitations()
		}
	}
}

func (l *DiscoveryListener) sendSolicitations() {
	interfaces, err := net.Interfaces()
	if err != nil {
		slog.Warn("tuya_local: could not enumerate network interfaces for v3.5 discovery", "error", err)
		return
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagBroadcast == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			slog.Debug("tuya_local: could not read interface addresses for discovery", "interface", iface.Name, "error", err)
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			localIP := ipNet.IP.To4()
			mask := ipNet.Mask
			if localIP == nil || len(mask) != net.IPv4len || !localIP.IsPrivate() {
				continue
			}
			broadcast := make(net.IP, net.IPv4len)
			for i := range broadcast {
				broadcast[i] = localIP[i] | ^mask[i]
			}
			if broadcast.Equal(localIP) || broadcast.IsUnspecified() {
				continue
			}
			if err := l.sendSolicitation(localIP, broadcast); err != nil {
				slog.Debug("tuya_local: v3.5 discovery broadcast failed",
					"interface", iface.Name, "local_ip", localIP.String(),
					"broadcast", broadcast.String(), "error", err)
			}
		}
	}
}

func (l *DiscoveryListener) sendSolicitation(localIP, broadcast net.IP) error {
	if localIP.To4() == nil || broadcast.To4() == nil {
		return errors.New("tuya_local: v3.5 discovery requires IPv4 addresses")
	}

	l.mu.Lock()
	l.sequence++
	seq := l.sequence
	l.mu.Unlock()

	packet, err := buildTuya35DiscoveryRequest(seq, localIP.String())
	if err != nil {
		return fmt.Errorf("build discovery request: %w", err)
	}

	listenConfig := net.ListenConfig{
		Control: func(network, address string, raw syscall.RawConn) error {
			var socketErr error
			if err := raw.Control(func(fd uintptr) {
				socketErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
			}); err != nil {
				return err
			}
			return socketErr
		},
	}
	packetConn, err := listenConfig.ListenPacket(context.Background(), "udp4", net.JoinHostPort(localIP.String(), "0"))
	if err != nil {
		return fmt.Errorf("bind discovery sender to %s: %w", localIP, err)
	}
	defer packetConn.Close()

	_, err = packetConn.WriteTo(packet, &net.UDPAddr{IP: broadcast, Port: 7000})
	if err != nil {
		return fmt.Errorf("send to %s:7000: %w", broadcast, err)
	}
	return nil
}

// buildTuya35DiscoveryRequest creates a 6699/GCM REQ_DEVINFO packet. The
// advertised address is the interface address bound by sendSolicitation.
func buildTuya35DiscoveryRequest(seq uint32, localIP string) ([]byte, error) {
	if net.ParseIP(localIP).To4() == nil {
		return nil, fmt.Errorf("invalid IPv4 address %q", localIP)
	}
	payload, err := json.Marshal(struct {
		From string `json:"from"`
		IP   string `json:"ip"`
	}{From: "app", IP: localIP})
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(tuyaDiscoveryKey[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, tuyaGCMNonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate discovery nonce: %w", err)
	}

	header := make([]byte, tuya6699HeaderSize)
	copy(header[:4], []byte{0x00, 0x00, 0x66, 0x99})
	// bytes 4:6 are the reserved field.
	binary.BigEndian.PutUint32(header[6:10], seq)
	binary.BigEndian.PutUint32(header[10:14], CmdReqDevInfo)
	binary.BigEndian.PutUint32(header[14:18], uint32(tuyaGCMNonceSize+len(payload)+tuyaGCMTagSize))

	sealed := aead.Seal(nil, nonce, payload, header[4:])
	packet := make([]byte, 0, len(header)+len(nonce)+len(sealed)+tuya6699FooterSize)
	packet = append(packet, header...)
	packet = append(packet, nonce...)
	packet = append(packet, sealed...)
	packet = append(packet, 0x00, 0x00, 0x99, 0x66)
	return packet, nil
}

func decodeTuyaDiscoveryPacket(data []byte) (DiscoveredDevice, error) {
	var zero DiscoveredDevice
	if len(data) == 0 {
		return zero, errors.New("tuya_local: empty discovery datagram")
	}

	var payload []byte
	switch {
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x00, 0x00, 0x55, 0xaa}):
		var err error
		payload, err = decodeTuya55AADiscovery(data)
		if err != nil {
			return zero, err
		}
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x00, 0x00, 0x66, 0x99}):
		var err error
		payload, err = decodeTuya6699Discovery(data)
		if err != nil {
			return zero, err
		}
	default:
		payload = data
	}

	if dev, ok := parseTuyaDiscoveryJSON(payload); ok {
		return dev, nil
	}

	// Some legacy devices send the AES-ECB encrypted JSON without a 55AA frame.
	decrypted, err := decryptAES_ECB(payload, tuyaDiscoveryKey[:])
	if err == nil {
		if dev, ok := parseTuyaDiscoveryJSON(decrypted); ok {
			return dev, nil
		}
	}

	// A few firmware variants include a clear 3.x header before encrypted JSON.
	if len(payload) > 15 && payload[0] == '3' && payload[1] == '.' {
		decrypted, err = decryptAES_ECB(payload[15:], tuyaDiscoveryKey[:])
		if err == nil {
			if dev, ok := parseTuyaDiscoveryJSON(decrypted); ok {
				return dev, nil
			}
		}
	}
	return zero, errors.New("tuya_local: unsupported or malformed discovery payload")
}

func decodeTuya55AADiscovery(data []byte) ([]byte, error) {
	if len(data) < 24 {
		return nil, errors.New("tuya_local: truncated 55AA discovery frame")
	}
	payloadLen := binary.BigEndian.Uint32(data[12:16])
	totalLen := uint64(16) + uint64(payloadLen)
	if payloadLen < 8 || totalLen > uint64(len(data)) {
		return nil, errors.New("tuya_local: invalid 55AA discovery frame length")
	}
	end := int(totalLen)
	if !bytes.Equal(data[end-4:end], []byte{0x00, 0x00, 0xaa, 0x55}) {
		return nil, errors.New("tuya_local: invalid 55AA discovery footer")
	}
	bodyEnd := end - 8 // Exclude CRC32 and footer.
	if bodyEnd < 16 {
		return nil, errors.New("tuya_local: missing 55AA discovery payload")
	}

	// Device-to-client 55AA frames carry a four-byte return code. Accept both
	// layouts because some older discovery firmware omits it.
	candidates := [][]byte{data[20:bodyEnd], data[16:bodyEnd]}
	for _, candidate := range candidates {
		if dev, ok := parseTuyaDiscoveryJSON(candidate); ok {
			_ = dev
			return candidate, nil
		}
		if normalized := stripTuyaVersionHeader(candidate); !bytes.Equal(normalized, candidate) {
			if dev, ok := parseTuyaDiscoveryJSON(normalized); ok {
				_ = dev
				return normalized, nil
			}
		}
		decrypted, err := decryptAES_ECB(candidate, tuyaDiscoveryKey[:])
		if err == nil {
			if normalized := stripTuyaVersionHeader(decrypted); len(normalized) > 0 {
				if dev, ok := parseTuyaDiscoveryJSON(normalized); ok {
					_ = dev
					return normalized, nil
				}
			}
		}
	}
	return nil, errors.New("tuya_local: cannot decode 55AA discovery payload")
}

func decodeTuya6699Discovery(data []byte) ([]byte, error) {
	if len(data) < tuya6699HeaderSize+tuyaGCMNonceSize+tuyaGCMTagSize+tuya6699FooterSize {
		return nil, errors.New("tuya_local: truncated 6699 discovery frame")
	}
	length := binary.BigEndian.Uint32(data[14:18])
	totalLen := uint64(tuya6699HeaderSize) + uint64(length) + tuya6699FooterSize
	if length < tuyaGCMNonceSize+tuyaGCMTagSize || totalLen != uint64(len(data)) {
		return nil, errors.New("tuya_local: invalid 6699 discovery frame length")
	}
	if !bytes.Equal(data[len(data)-tuya6699FooterSize:], []byte{0x00, 0x00, 0x99, 0x66}) {
		return nil, errors.New("tuya_local: invalid 6699 discovery footer")
	}

	block, err := aes.NewCipher(tuyaDiscoveryKey[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceStart := tuya6699HeaderSize
	nonceEnd := nonceStart + tuyaGCMNonceSize
	tagEnd := len(data) - tuya6699FooterSize
	nonce := data[nonceStart:nonceEnd]
	sealed := data[nonceEnd:tagEnd]
	plaintext, err := aead.Open(nil, nonce, sealed, data[4:tuya6699HeaderSize])
	if err != nil {
		return nil, fmt.Errorf("tuya_local: 6699 discovery authentication failed: %w", err)
	}

	// Discovery responses may include a four-byte return code; unsolicited
	// broadcasts commonly omit it. Strip it only when it exposes JSON or a 3.x header.
	if len(plaintext) >= 4 && plaintext[0] != '{' {
		candidate := plaintext[4:]
		if len(candidate) > 0 && candidate[0] == '{' {
			plaintext = candidate
		} else if len(candidate) >= 15 && candidate[0] == '3' && candidate[1] == '.' {
			plaintext = candidate
		}
	}
	return stripTuyaVersionHeader(plaintext), nil
}

func stripTuyaVersionHeader(payload []byte) []byte {
	if len(payload) >= 15 && payload[0] == '3' && payload[1] == '.' {
		return bytes.Trim(payload[15:], "\x00 \r\n\t")
	}
	return bytes.Trim(payload, "\x00 \r\n\t")
}

func parseTuyaDiscoveryJSON(payload []byte) (DiscoveredDevice, bool) {
	var zero DiscoveredDevice
	payload = bytes.Trim(payload, "\x00 \r\n\t")
	if len(payload) == 0 || payload[0] != '{' {
		return zero, false
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return zero, false
	}
	if nested, ok := fields["data"]; ok && len(nested) > 0 && nested[0] == '{' {
		var nestedFields map[string]json.RawMessage
		if json.Unmarshal(nested, &nestedFields) == nil {
			for key, value := range nestedFields {
				if _, exists := fields[key]; !exists {
					fields[key] = value
				}
			}
		}
	}

	dev := DiscoveredDevice{
		DeviceID:   rawJSONText(fields, "gwId", "devId", "dev_id", "id"),
		Version:    rawJSONText(fields, "version", "ver"),
		ProductKey: rawJSONText(fields, "productKey", "product_key"),
	}
	if active := rawJSONText(fields, "active"); active != "" {
		dev.Active, _ = strconv.Atoi(active)
	}
	if !validDeviceID(dev.DeviceID) {
		return zero, false
	}
	return dev, true
}

func rawJSONText(fields map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		raw, ok := fields[key]
		if !ok {
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) == nil {
			return strings.TrimSpace(value)
		}
		value = string(bytes.TrimSpace(raw))
		if len(value) > 0 && value[0] != '{' && value[0] != '[' && value != "null" {
			return strings.Trim(value, "\"")
		}
	}
	return ""
}

func (l *DiscoveryListener) handleDiscovered(dev DiscoveredDevice) {
	l.mu.Lock()
	l.devices[dev.DeviceID] = dev
	l.mu.Unlock()

	if l.onDevice != nil {
		l.onDevice(dev)
	}
}

// GetDiscovered returns all currently discovered devices.
func (l *DiscoveryListener) GetDiscovered() []DiscoveredDevice {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]DiscoveredDevice, 0, len(l.devices))
	for _, d := range l.devices {
		result = append(result, d)
	}
	return result
}

// GetDevice returns a discovered device by device ID if found on the local network.
func (l *DiscoveryListener) GetDevice(deviceID string) (DiscoveredDevice, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	d, ok := l.devices[deviceID]
	return d, ok
}
