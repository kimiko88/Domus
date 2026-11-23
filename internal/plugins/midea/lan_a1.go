package midea

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	a1DeviceType       = 0xA1
	a1MessageQuery     = 0x03
	a1MessageSet       = 0x02
	a1MessageAppliance = 0xA0
	a1BodyQuery        = 0x41
	a1BodySet          = 0x48

	mideaHeaderLength = 40
	mideaDigestLength = 16
	mideaReadTimeout  = 5 * time.Second
	mideaWriteTimeout = 2 * time.Second
)

var (
	mideaLANKey, _ = hex.DecodeString("6a92ef406bad2f0359baad994171ea6d")
	mideaLANSalt, _ = hex.DecodeString("78686469776a6e6368656b6434643531326368646a783564386534633339344432443753")
	ErrMideaAuth = errors.New("midea: V3 local authentication failed")
	ErrMideaFrame = errors.New("midea: invalid or unauthenticated LAN frame")
)

// A1LANConfig contains the private connection material for one A1 dehumidifier.
// TokenHex and KeyHex are used only by V3; callers must encrypt this structure
// before storing it. They are deliberately not part of Device or any API view.
type A1LANConfig struct {
	Device           Device
	ApplianceID      uint64
	MessageVersion   uint8
	TokenHex         string
	KeyHex           string
}

type A1LANDriver struct {
	protocol Protocol
	mu       sync.RWMutex
	devices  map[string]A1LANConfig
	dial     func(context.Context, string, string) (net.Conn, error)
}

func NewA1LANDriver(protocol Protocol) (*A1LANDriver, error) {
	if protocol != ProtocolMSmartV2 && protocol != ProtocolMSmartV3 {
		return nil, fmt.Errorf("midea: A1 LAN driver only supports M-Smart V2/V3, got %q", protocol)
	}
	return &A1LANDriver{
		protocol: protocol,
		devices:  make(map[string]A1LANConfig),
		dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return (&net.Dialer{Timeout: mideaReadTimeout, KeepAlive: 15 * time.Second}).DialContext(ctx, network, address)
		},
	}, nil
}

func (d *A1LANDriver) Protocol() Protocol { return d.protocol }

// Register adds an A1 device and its optional V3 credentials.
func (d *A1LANDriver) Register(config A1LANConfig) error {
	config.Device.Protocol = d.protocol
	if config.Device.TypeCode == "" {
		config.Device.TypeCode = "A1"
	}
	if !strings.EqualFold(config.Device.TypeCode, "A1") {
		return fmt.Errorf("%w: this driver implements A1 dehumidifiers only", ErrNoDriver)
	}
	config.Device.Kind = KindDehumidifier
	if config.Device.Port == 0 {
		config.Device.Port = 6444
	}
	if err := config.Device.Validate(); err != nil {
		return err
	}
	if config.ApplianceID == 0 {
		id, err := strconv.ParseUint(config.Device.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: A1 device id must be the numeric Midea appliance id", ErrInvalidDevice)
		}
		config.ApplianceID = id
	}
	if d.protocol == ProtocolMSmartV3 {
		token, tokenErr := hex.DecodeString(config.TokenHex)
		key, keyErr := hex.DecodeString(config.KeyHex)
		if tokenErr != nil || len(token) != 64 || keyErr != nil || len(key) != 32 {
			return fmt.Errorf("%w: V3 requires a 64-byte token and 32-byte key in hexadecimal form", ErrInvalidDevice)
		}
	} else {
		config.TokenHex, config.KeyHex = "", ""
	}
	if len(config.Device.Modes) == 0 {
		config.Device.Modes = []string{"manual", "continuous", "auto", "clothes_dry", "shoes_dry", "silent"}
	}
	if len(config.Device.FanSpeeds) == 0 {
		config.Device.FanSpeeds = []string{"lowest", "low", "medium", "high", "auto", "off"}
	}
	if config.Device.MinHumidity == 0 {
		config.Device.MinHumidity = 35
	}
	if config.Device.MaxHumidity == 0 {
		config.Device.MaxHumidity = 100
	}

	d.mu.Lock()
	d.devices[config.Device.ID] = cloneA1Config(config)
	d.mu.Unlock()
	return nil
}

func (d *A1LANDriver) Remove(id string) {
	d.mu.Lock()
	delete(d.devices, id)
	d.mu.Unlock()
}

func (d *A1LANDriver) Discover(ctx context.Context) ([]Device, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]Device, 0, len(d.devices))
	for _, config := range d.devices {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		result = append(result, cloneDevice(config.Device))
	}
	return result, nil
}

func (d *A1LANDriver) Read(ctx context.Context, device Device) (map[string]any, error) {
	config, err := d.config(device.ID)
	if err != nil {
		return nil, err
	}
	session, err := d.openSession(ctx, config)
	if err != nil {
		return nil, err
	}
	defer session.close()

	version, err := session.queryMessageVersion(ctx, config)
	if err != nil {
		return nil, err
	}
	return session.queryA1(ctx, config, version)
}

func (d *A1LANDriver) Execute(ctx context.Context, device Device, command Command) error {
	config, err := d.config(device.ID)
	if err != nil {
		return err
	}
	session, err := d.openSession(ctx, config)
	if err != nil {
		return err
	}
	defer session.close()

	version, err := session.queryMessageVersion(ctx, config)
	if err != nil {
		return fmt.Errorf("midea: A1 appliance version query: %w", err)
	}
	current, err := session.queryA1(ctx, config, version)
	if err != nil {
		return fmt.Errorf("midea: A1 state query before write: %w", err)
	}
	values := make(map[string]any, len(current)+len(command.Values))
	for key, value := range current {
		values[key] = value
	}
	for key, value := range command.Values {
		values[key] = value
	}
	filterReset := command.Action == "reset_filter"
	packet, err := buildA1SetPacket(config.ApplianceID, version, values, filterReset)
	if err != nil {
		return err
	}
	// The physical command is transmitted once. A missing acknowledgement is
	// not grounds for retransmission; Adapter.Command performs a fresh read-back.
	if err := session.writePacket(ctx, packet); err != nil {
		return fmt.Errorf("midea: A1 write delivery is uncertain: %w", err)
	}
	_ = session.readOptionalAck()
	return nil
}

func (d *A1LANDriver) config(id string) (A1LANConfig, error) {
	d.mu.RLock()
	config, ok := d.devices[id]
	d.mu.RUnlock()
	if !ok {
		return A1LANConfig{}, ErrDeviceNotFound
	}
	return cloneA1Config(config), nil
}

func cloneA1Config(config A1LANConfig) A1LANConfig {
	config.Device = cloneDevice(config.Device)
	return config
}

type a1Session struct {
	conn       net.Conn
	protocol   Protocol
	security   *mideaV3Security
	applianceID uint64
}

func (d *A1LANDriver) openSession(ctx context.Context, config A1LANConfig) (*a1Session, error) {
	address := net.JoinHostPort(config.Device.Host, strconv.Itoa(config.Device.Port))
	conn, err := d.dial(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("midea: connect to local appliance: %w", err)
	}
	session := &a1Session{conn: conn, protocol: d.protocol, applianceID: config.ApplianceID}
	if d.protocol == ProtocolMSmartV3 {
		token, _ := hex.DecodeString(config.TokenHex)
		key, _ := hex.DecodeString(config.KeyHex)
		session.security = newMideaV3Security()
		if err := session.authenticate(ctx, token, key); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	return session, nil
}

func (s *a1Session) close() {
	if s != nil && s.conn != nil {
		_ = s.conn.Close()
	}
}

func (s *a1Session) setDeadline(ctx context.Context, fallback time.Duration) error {
	deadline := time.Now().Add(fallback)
	if requested, ok := ctx.Deadline(); ok && requested.Before(deadline) {
		deadline = requested
	}
	return s.conn.SetDeadline(deadline)
}

func (s *a1Session) authenticate(ctx context.Context, token, key []byte) error {
	if len(token) != 64 || len(key) != 32 {
		return ErrMideaAuth
	}
	request, err := s.security.encode(token, 0)
	if err != nil {
		return err
	}
	if err := s.setDeadline(ctx, mideaReadTimeout); err != nil {
		return err
	}
	if _, err := s.conn.Write(request); err != nil {
		return fmt.Errorf("%w: write handshake: %v", ErrMideaAuth, err)
	}
	response, err := read8370Frame(s.conn)
	if err != nil {
		return fmt.Errorf("%w: read handshake: %v", ErrMideaAuth, err)
	}
	payload, err := s.security.decode(response)
	if err != nil || len(payload) != 64 {
		return ErrMideaAuth
	}
	ciphertext, signature := payload[:32], payload[32:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return ErrMideaAuth
	}
	plain := make([]byte, 32)
	cipher.NewCBCDecrypter(block, make([]byte, aes.BlockSize)).CryptBlocks(plain, ciphertext)
	digest := sha256.Sum256(plain)
	if !bytes.Equal(digest[:], signature) {
		return ErrMideaAuth
	}
	s.security.tcpKey = make([]byte, 32)
	for i := range plain {
		s.security.tcpKey[i] = plain[i] ^ key[i]
	}
	s.security.requestCount = 0
	return nil
}

func (s *a1Session) queryMessageVersion(ctx context.Context, config A1LANConfig) (uint8, error) {
	frame := buildA1ApplianceQuery(config.ApplianceID)
	response, err := s.exchange(ctx, frame, mideaReadTimeout)
	if err != nil {
		if config.MessageVersion != 0 {
			return config.MessageVersion, nil
		}
		return 0, fmt.Errorf("midea: appliance capability handshake failed: %w", err)
	}
	if len(response) < 11 || response[0] != 0xAA || response[9] != a1MessageAppliance {
		if config.MessageVersion != 0 {
			return config.MessageVersion, nil
		}
		return 0, fmt.Errorf("%w: unexpected appliance handshake response", ErrMideaFrame)
	}
	version := response[8]
	if version == 0 {
		return 0, fmt.Errorf("%w: appliance returned an empty message protocol version", ErrMideaFrame)
	}
	return version, nil
}

func (s *a1Session) queryA1(ctx context.Context, config A1LANConfig, version uint8) (map[string]any, error) {
	frame, err := buildA1QueryFrame(version)
	if err != nil {
		return nil, err
	}
	response, err := s.exchange(ctx, frame, mideaReadTimeout)
	if err != nil {
		return nil, err
	}
	return parseA1Response(response)
}

func (s *a1Session) writePacket(ctx context.Context, packet []byte) error {
	if err := s.setDeadline(ctx, mideaWriteTimeout); err != nil {
		return err
	}
	wire := packet
	if s.protocol == ProtocolMSmartV3 {
		var err error
		wire, err = s.security.encode(packet, 6)
		if err != nil {
			return err
		}
	}
	return writeAll(s.conn, wire)
}

func (s *a1Session) readOptionalAck() {
	_ = s.conn.SetReadDeadline(time.Now().Add(350 * time.Millisecond))
	if s.protocol == ProtocolMSmartV3 {
		_, _ = read8370Frame(s.conn)
		return
	}
	_, _ = readMideaPacket(s.conn)
}

func (s *a1Session) exchange(ctx context.Context, frame []byte, timeout time.Duration) ([]byte, error) {
	packet, err := buildMideaPacket(s.applianceID, frame)
	if err != nil {
		return nil, err
	}
	wire := packet
	if s.protocol == ProtocolMSmartV3 {
		wire, err = s.security.encode(packet, 6)
		if err != nil {
			return nil, err
		}
	}
	if err := s.setDeadline(ctx, timeout); err != nil {
		return nil, err
	}
	if err := writeAll(s.conn, wire); err != nil {
		return nil, err
	}
	var response []byte
	if s.protocol == ProtocolMSmartV3 {
		var wrapped []byte
		wrapped, err = read8370Frame(s.conn)
		if err == nil {
			response, err = s.security.decode(wrapped)
		}
	} else {
		response, err = readMideaPacket(s.conn)
	}
	if err != nil {
		return nil, err
	}
	return decodeMideaPacket(response)
}

func buildA1ApplianceQuery(applianceID uint64) []byte {
	body := make([]byte, 19)
	return buildA1Frame(applianceID, 0, a1MessageAppliance, body)
}

func buildA1QueryFrame(version uint8) ([]byte, error) {
	body := []byte{
		0x81, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	return buildA1DataFrame(version, a1MessageQuery, a1BodyQuery, body)
}

func buildA1SetPacket(applianceID uint64, version uint8, values map[string]any, filterReset bool) ([]byte, error) {
	body := make([]byte, 20)
	boolAt := func(key string) bool { value, _ := values[key].(bool); return value }
	intAt := func(key string, fallback int) (int, error) {
		if raw, exists := values[key]; exists {
			value, ok := numberAsInt(raw)
			if !ok {
				return 0, fmt.Errorf("midea: A1 %s must be an integer", key)
			}
			return value, nil
		}
		return fallback, nil
	}
	power := boolAt("power")
	promptTone := boolAt("prompt_tone")
	body[0] = 0x02
	if power {
		body[0] |= 0x01
	}
	if promptTone {
		body[0] |= 0x40
	}
	mode, err := intAt("mode_code", 1)
	if err != nil {
		return nil, err
	}
	if mode == 1 {
		if text, ok := values["mode"].(string); ok {
			mode, ok = a1ModeCode(text)
			if !ok {
				return nil, fmt.Errorf("%w: unsupported A1 mode %q", ErrInvalidCommand, text)
			}
		}
	}
	speed, err := intAt("fan_speed_code", 40)
	if err != nil {
		return nil, err
	}
	if text, ok := values["fan_speed"].(string); ok {
		var found bool
		speed, found = a1FanSpeedCode(text)
		if !found {
			return nil, fmt.Errorf("%w: unsupported A1 fan speed %q", ErrInvalidCommand, text)
		}
	}
	targetHumidity, err := intAt("target_humidity", 40)
	if err != nil {
		return nil, err
	}
	if targetHumidity < 35 || targetHumidity > 100 {
		return nil, fmt.Errorf("%w: A1 target humidity must be between 35 and 100%%", ErrInvalidCommand)
	}
	waterLevel, err := intAt("water_level_set", 50)
	if err != nil {
		return nil, err
	}
	if waterLevel != 25 && waterLevel != 50 && waterLevel != 75 && waterLevel != 100 {
		return nil, fmt.Errorf("%w: A1 water level must be 25, 50, 75 or 100", ErrInvalidCommand)
	}

	body[1], body[2], body[6] = byte(mode), byte(speed), byte(targetHumidity)
	if boolAt("child_lock") {
		body[7] |= 0x80
	}
	if boolAt("anion") {
		body[8] |= 0x40
	}
	if boolAt("pump") {
		body[8] |= 0x08
	}
	if boolAt("pump_enable") {
		body[8] |= 0x10
	}
	if filterReset {
		body[8] |= 0x80
	}
	if boolAt("swing") {
		body[9] = 0x08
	}
	body[12] = byte(waterLevel)
	return buildA1DataFrame(version, a1MessageSet, a1BodySet, body)
}

func buildA1DataFrame(version, messageType, bodyType byte, payload []byte) ([]byte, error) {
	if len(payload) == 0 || len(payload) > 64 {
		return nil, fmt.Errorf("%w: invalid A1 payload length", ErrMideaFrame)
	}
	body := make([]byte, 0, len(payload)+3)
	body = append(body, bodyType)
	body = append(body, payload...)
	body = append(body, 1)
	body = append(body, mideaCRC8(body))
	return buildA1HeaderFrame(version, messageType, body), nil
}

func buildA1HeaderFrame(version, messageType byte, body []byte) []byte {
	frame := make([]byte, 10, 10+len(body)+1)
	frame[0], frame[2], frame[8], frame[9] = 0xAA, a1DeviceType, version, messageType
	frame[1] = byte(len(frame) + len(body))
	frame = append(frame, body...)
	var sum byte
	for _, b := range frame[1:] {
		sum += b
	}
	return append(frame, -sum)
}

func parseA1Response(frame []byte) (map[string]any, error) {
	if len(frame) < 31 || frame[0] != 0xAA || frame[2] != a1DeviceType {
		return nil, fmt.Errorf("%w: A1 response header or length is invalid", ErrMideaFrame)
	}
	if frame[9] != a1MessageQuery && frame[9] != a1MessageSet && frame[9] != 0x04 && frame[9] != 0x05 {
		return nil, fmt.Errorf("%w: unexpected A1 message type 0x%02x", ErrMideaFrame, frame[9])
	}
	var checksum byte
	for _, b := range frame[1:] {
		checksum += b
	}
	if checksum != 0 {
		return nil, fmt.Errorf("%w: A1 message checksum mismatch", ErrMideaFrame)
	}
	body := frame[10 : len(frame)-1]
	if len(body) < 20 || body[0] != a1BodyQuery {
		return nil, fmt.Errorf("%w: A1 status body is missing or unsupported", ErrMideaFrame)
	}
	power := body[1]&0x01 != 0
	modeCode := body[2] & 0x0F
	speedCode := body[3] & 0x7F
	targetHumidity := int(body[7])
	if targetHumidity < 35 {
		targetHumidity = 35
	}
	tank := int(body[10] & 0x7F)
	waterLevel := int(body[15])
	if waterLevel != 25 && waterLevel != 50 && waterLevel != 75 && waterLevel != 100 {
		waterLevel = 50
	}
	currentHumidity := int(body[16])
	currentTemperature := (float64(body[17]) - 50) / 2
	mode, _ := a1ModeName(modeCode)
	fanSpeed, _ := a1FanSpeedName(speedCode)
	return map[string]any{
		"power": power,
		"mode": mode,
		"mode_code": int(modeCode),
		"fan_speed": fanSpeed,
		"fan_speed_code": int(speedCode),
		"target_humidity": targetHumidity,
		"current_humidity": currentHumidity,
		"current_temperature": currentTemperature,
		"water_tank_level": tank,
		"water_level_set": waterLevel,
		"tank_full": tank >= waterLevel && tank > 0,
		"child_lock": body[8]&0x80 != 0,
		"anion": body[9]&0x40 != 0,
		"pump": body[9]&0x08 != 0,
		"pump_enable": body[9]&0x10 != 0,
		"filter_cleaning_reminder": body[9]&0x80 != 0,
		"swing": body[19]&0x20 != 0,
	}, nil
}

var a1Modes = map[string]int{
	"manual": 1, "continuous": 2, "auto": 3, "clothes_dry": 4, "shoes_dry": 5, "silent": 10,
}
var a1FanSpeeds = map[string]int{
	"lowest": 1, "low": 40, "medium": 60, "high": 80, "auto": 102, "off": 127,
}

func a1ModeCode(value string) (int, bool) { code, ok := a1Modes[value]; return code, ok }
func a1ModeName(code int) (string, bool) {
	for name, candidate := range a1Modes {
		if candidate == code { return name, true }
	}
	if code == 11 { return "auto", true }
	return "", false
}
func a1FanSpeedCode(value string) (int, bool) { code, ok := a1FanSpeeds[value]; return code, ok }
func a1FanSpeedName(code int) (string, bool) {
	for name, candidate := range a1FanSpeeds {
		if candidate == code { return name, true }
	}
	return "", false
}

func numberAsInt(value any) (int, bool) {
	switch v := value.(type) {
	case int: return v, true
	case int32: return int(v), true
	case int64: return int(v), true
	case float64:
		if math.Trunc(v) == v && v >= math.MinInt32 && v <= math.MaxInt32 { return int(v), true }
	case float32:
		if math.Trunc(float64(v)) == float64(v) { return int(v), true }
	}
	return 0, false
}

func buildMideaPacket(deviceID uint64, message []byte) ([]byte, error) {
	if len(message) == 0 || len(mideaLANKey) != 16 || len(mideaLANSalt) == 0 {
		return nil, ErrMideaFrame
	}
	block, err := aes.NewCipher(mideaLANKey)
	if err != nil { return nil, err }
	padding := aes.BlockSize - len(message)%aes.BlockSize
	plain := append(append([]byte(nil), message...), bytes.Repeat([]byte{byte(padding)}, padding)...)
	encrypted := make([]byte, len(plain))
	for offset := 0; offset < len(plain); offset += aes.BlockSize {
		block.Encrypt(encrypted[offset:offset+aes.BlockSize], plain[offset:offset+aes.BlockSize])
	}
	packet := make([]byte, mideaHeaderLength, mideaHeaderLength+len(encrypted)+mideaDigestLength)
	packet[0], packet[1], packet[2], packet[3], packet[6] = 0x5A, 0x5A, 0x01, 0x11, 0x20
	binary.LittleEndian.PutUint32(packet[8:12], uint32(time.Now().UnixNano()))
	now := time.Now().UTC()
	pairs := []int{now.Year() % 100, int(now.Month()), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond() / 10_000_000, 0}
	for i := 0; i < len(pairs); i++ { packet[19-i] = byte(pairs[i]) }
	binary.LittleEndian.PutUint64(packet[20:28], deviceID)
	packet = append(packet, encrypted...)
	binary.LittleEndian.PutUint16(packet[4:6], uint16(len(packet)+mideaDigestLength))
	hashInput := append(append([]byte(nil), packet...), mideaLANSalt...)
	digest := md5.Sum(hashInput)
	packet = append(packet, digest[:]...)
	return packet, nil
}

func decodeMideaPacket(packet []byte) ([]byte, error) {
	if len(packet) < mideaHeaderLength+mideaDigestLength || packet[0] != 0x5A || packet[1] != 0x5A {
		return nil, ErrMideaFrame
	}
	expectedLength := int(binary.LittleEndian.Uint16(packet[4:6]))
	if expectedLength != len(packet) || (len(packet)-mideaHeaderLength-mideaDigestLength)%aes.BlockSize != 0 {
		return nil, ErrMideaFrame
	}
	hashInput := append(append([]byte(nil), packet[:len(packet)-mideaDigestLength]...), mideaLANSalt...)
	digest := md5.Sum(hashInput)
	if !bytes.Equal(digest[:], packet[len(packet)-mideaDigestLength:]) {
		return nil, fmt.Errorf("%w: Midea packet MD5 mismatch", ErrMideaFrame)
	}
	block, err := aes.NewCipher(mideaLANKey)
	if err != nil { return nil, err }
	ciphertext := packet[mideaHeaderLength:len(packet)-mideaDigestLength]
	plain := make([]byte, len(ciphertext))
	for offset := 0; offset < len(ciphertext); offset += aes.BlockSize {
		block.Decrypt(plain[offset:offset+aes.BlockSize], ciphertext[offset:offset+aes.BlockSize])
	}
	if len(plain) == 0 { return nil, ErrMideaFrame }
	padding := int(plain[len(plain)-1])
	if padding == 0 || padding > aes.BlockSize || padding > len(plain) {
		return nil, ErrMideaFrame
	}
	for _, value := range plain[len(plain)-padding:] {
		if int(value) != padding { return nil, ErrMideaFrame }
	}
	return plain[:len(plain)-padding], nil
}

func readMideaPacket(reader io.Reader) ([]byte, error) {
	header := make([]byte, mideaHeaderLength)
	if _, err := io.ReadFull(reader, header); err != nil { return nil, err }
	if header[0] != 0x5A || header[1] != 0x5A { return nil, ErrMideaFrame }
	length := int(binary.LittleEndian.Uint16(header[4:6]))
	if length < mideaHeaderLength+mideaDigestLength || length > 64*1024 {
		return nil, ErrMideaFrame
	}
	packet := make([]byte, length)
	copy(packet, header)
	if _, err := io.ReadFull(reader, packet[mideaHeaderLength:]); err != nil { return nil, err }
	return packet, nil
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := writer.Write(data)
		if err != nil { return err }
		if n == 0 { return io.ErrShortWrite }
		data = data[n:]
	}
	return nil
}

func mideaCRC8(data []byte) byte {
	var crc byte
	for _, value := range data {
		crc ^= value
		for bit := 0; bit < 8; bit++ {
			if crc&1 != 0 { crc = (crc >> 1) ^ 0x8C } else { crc >>= 1 }
		}
	}
	return crc
}

// mideaV3Security implements the documented-by-reference 0x8370 local
// handshake/envelope. It is stateless outside one TCP connection.
type mideaV3Security struct {
	tcpKey []byte
	requestCount uint16
}
func newMideaV3Security() *mideaV3Security { return &mideaV3Security{} }

func (s *mideaV3Security) encode(payload []byte, msgType byte) ([]byte, error) {
	padding := 0
	size := len(payload)
	data := append([]byte(nil), payload...)
	if msgType == 3 || msgType == 6 {
		if len(s.tcpKey) != 32 { return nil, ErrMideaAuth }
		padding = (aes.BlockSize - ((size + 2) % aes.BlockSize)) % aes.BlockSize
		size += padding + 32
		if padding > 0 {
			random := make([]byte, padding)
			if _, err := rand.Read(random); err != nil { return nil, err }
			data = append(data, random...)
		}
	}
	header := []byte{0x83, 0x70, byte(size >> 8), byte(size), 0x20, byte((padding << 4) | int(msgType))}
	plain := make([]byte, 2, len(data)+2)
	binary.BigEndian.PutUint16(plain, s.requestCount)
	plain = append(plain, data...)
	s.requestCount++
	if msgType == 3 || msgType == 6 {
		block, err := aes.NewCipher(s.tcpKey)
		if err != nil { return nil, err }
		encrypted := make([]byte, len(plain))
		cipher.NewCBCEncrypter(block, make([]byte, aes.BlockSize)).CryptBlocks(encrypted, plain)
		signatureInput := append(append([]byte(nil), header...), plain...)
		signature := sha256.Sum256(signatureInput)
		plain = append(encrypted, signature[:]...)
	}
	return append(header, plain...), nil
}

func (s *mideaV3Security) decode(frame []byte) ([]byte, error) {
	if len(frame) < 8 || frame[0] != 0x83 || frame[1] != 0x70 || frame[4] != 0x20 {
		return nil, ErrMideaFrame
	}
	size := int(binary.BigEndian.Uint16(frame[2:4]))
	if size+8 != len(frame) { return nil, ErrMideaFrame }
	padding := int(frame[5] >> 4)
	msgType := frame[5] & 0x0F
	data := append([]byte(nil), frame[6:]...)
	if msgType == 3 || msgType == 6 {
		if len(s.tcpKey) != 32 || len(data) < 34 || (len(data)-32)%aes.BlockSize != 0 {
			return nil, ErrMideaAuth
		}
		signature := data[len(data)-32:]
		ciphertext := data[:len(data)-32]
		block, err := aes.NewCipher(s.tcpKey)
		if err != nil { return nil, err }
		plain := make([]byte, len(ciphertext))
		cipher.NewCBCDecrypter(block, make([]byte, aes.BlockSize)).CryptBlocks(plain, ciphertext)
		signatureInput := append(append([]byte(nil), frame[:6]...), plain...)
		expected := sha256.Sum256(signatureInput)
		if !bytes.Equal(expected[:], signature) { return nil, ErrMideaAuth }
		if padding > 0 {
			if padding > len(plain) { return nil, ErrMideaFrame }
			plain = plain[:len(plain)-padding]
		}
		data = plain
	}
	if len(data) < 2 { return nil, ErrMideaFrame }
	return append([]byte(nil), data[2:]...), nil
}

func read8370Frame(reader io.Reader) ([]byte, error) {
	header := make([]byte, 6)
	if _, err := io.ReadFull(reader, header); err != nil { return nil, err }
	if header[0] != 0x83 || header[1] != 0x70 || header[4] != 0x20 {
		return nil, ErrMideaFrame
	}
	length := int(binary.BigEndian.Uint16(header[2:4]))
	if length > 64*1024 || length < 0 { return nil, ErrMideaFrame }
	frame := make([]byte, 8+length)
	copy(frame, header)
	if _, err := io.ReadFull(reader, frame[6:]); err != nil { return nil, err }
	return frame, nil
}

func a1SetValues(state map[string]any) map[string]any {
	values := make(map[string]any, len(state))
	for key, value := range state { values[key] = value }
	return values
}
