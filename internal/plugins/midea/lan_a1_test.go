package midea

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"testing"
)

func a1FixturePayload() []byte {
	payload := make([]byte, 20)
	payload[0] = 0x01 // power
	payload[1] = 0x03 // auto
	payload[2] = 0x3C // medium fan
	payload[6] = 45  // target humidity
	payload[7] = 0x80
	payload[8] = 0x10 // pump enabled
	payload[9] = 60   // tank level
	payload[14] = 50  // tank threshold
	payload[15] = 55  // current humidity
	payload[16] = 94  // 22 C
	payload[18] = 0x20 // swing
	return payload
}

func TestBuildA1QueryFrame(t *testing.T) {
	frame, err := buildA1QueryFrame(3)
	if err != nil {
		t.Fatalf("build query: %v", err)
	}
	if frame[0] != 0xAA || frame[2] != a1DeviceType || frame[8] != 3 || frame[9] != a1MessageQuery {
		t.Fatalf("unexpected AA header: %x", frame[:10])
	}
	if frame[10] != a1BodyQuery || frame[11] != 0x81 || frame[13] != 0xFF {
		t.Fatalf("unexpected query body: %x", frame[10:])
	}
	if int(frame[1]) != len(frame)-1 {
		t.Fatalf("header length = %d, wire payload length = %d", frame[1], len(frame)-1)
	}
	var checksum byte
	for _, b := range frame[1:] { checksum += b }
	if checksum != 0 {
		t.Fatalf("A1 message checksum = 0x%02x, want 0", checksum)
	}
	if frame[len(frame)-2] != mideaCRC8(frame[10:len(frame)-2]) {
		t.Fatal("A1 body CRC8 mismatch")
	}
}

func TestBuildA1SetPacketEncodesDehumidifierControls(t *testing.T) {
	frame, err := buildA1SetPacket(12345, 3, map[string]any{
		"power": true,
		"mode": "continuous",
		"mode_code": 3,
		"fan_speed": "high",
		"fan_speed_code": 60,
		"target_humidity": 48,
		"child_lock": true,
		"pump_enable": true,
		"swing": true,
		"water_level_set": 75,
	}, true)
	if err != nil {
		t.Fatalf("build A1 set: %v", err)
	}
	body := frame[10 : len(frame)-1]
	if frame[9] != a1MessageSet || body[0] != a1BodySet {
		t.Fatalf("unexpected set message header/body: %x", frame[:12])
	}
	if body[1] != 0x03 || body[2] != 2 || body[3] != 80 || body[7] != 48 {
		t.Fatalf("unexpected power/mode/fan/humidity fields: %x", body)
	}
	if body[8]&0x80 == 0 || body[9]&0x90 != 0x90 || body[10] != 0x08 || body[13] != 75 {
		t.Fatalf("unexpected flags/swing/water-level fields: %x", body)
	}
}

func TestParseA1DehumidifierStatus(t *testing.T) {
	frame, err := buildA1DataFrame(3, a1MessageQuery, a1BodyQuery, a1FixturePayload())
	if err != nil { t.Fatal(err) }
	state, err := parseA1Response(frame)
	if err != nil { t.Fatalf("parse response: %v", err) }
	expected := map[string]any{
		"power": true,
		"mode": "auto",
		"fan_speed": "medium",
		"target_humidity": 45,
		"current_humidity": 55,
		"current_temperature": 22.0,
		"water_tank_level": 60,
		"water_level_set": 50,
		"tank_full": true,
		"child_lock": true,
		"pump_enable": true,
		"swing": true,
	}
	for key, want := range expected {
		if got := state[key]; got != want {
			t.Errorf("%s = %#v, want %#v", key, got, want)
		}
	}
}

func TestParseA1RejectsBadChecksum(t *testing.T) {
	frame, err := buildA1DataFrame(3, a1MessageQuery, a1BodyQuery, a1FixturePayload())
	if err != nil { t.Fatal(err) }
	frame[len(frame)-1] ^= 0x01
	if _, err := parseA1Response(frame); !errors.Is(err, ErrMideaFrame) {
		t.Fatalf("expected frame checksum rejection, got %v", err)
	}
}

func TestA1HumidityAndModeValidation(t *testing.T) {
	device := dehumidifierFixture()
	device.MinHumidity = 35
	device.MaxHumidity = 100
	if err := ValidateCommand(device, Command{Values: map[string]any{"target_humidity": 35}}); err != nil {
		t.Fatalf("minimum target should be accepted: %v", err)
	}
	if err := ValidateCommand(device, Command{Values: map[string]any{"target_humidity": 34}}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("expected low humidity target rejection, got %v", err)
	}
	if err := ValidateCommand(device, Command{Values: map[string]any{"mode": "shoes_dry"}}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("unsupported mode must not be exposed by a profile missing modes: %v", err)
	}
	device.Modes = []string{"manual", "continuous", "auto", "clothes_dry", "shoes_dry", "silent"}
	if err := ValidateCommand(device, Command{Values: map[string]any{"mode": "shoes_dry"}}); err != nil {
		t.Fatalf("supported dehumidifier mode rejected: %v", err)
	}
}

func TestMideaPacketEncryptDecryptRoundTrip(t *testing.T) {
	packet, err := buildMideaPacket(0x123456789ABC, []byte{0xAA, 0x01, 0x02, 0x03})
	if err != nil { t.Fatal(err) }
	if len(packet) < mideaHeaderLength+mideaDigestLength || packet[0] != 0x5A || packet[1] != 0x5A {
		t.Fatalf("invalid Midea packet header/length: %x", packet[:min(8, len(packet))])
	}
	decoded, err := decodeMideaPacket(packet)
	if err != nil { t.Fatalf("decode packet: %v", err) }
	if !bytes.Equal(decoded, []byte{0xAA, 0x01, 0x02, 0x03}) {
		t.Fatalf("decoded packet = %x", decoded)
	}
	packet[len(packet)-1] ^= 0x01
	if _, err := decodeMideaPacket(packet); !errors.Is(err, ErrMideaFrame) {
		t.Fatalf("expected packet authentication failure, got %v", err)
	}
}

func TestMideaV3HandshakeAndEncryptedEnvelopeRoundTrip(t *testing.T) {
	client := newMideaV3Security()
	server := newMideaV3Security()
	token, _ := hex.DecodeString(strings.Repeat("11", 64))
	handshake, err := client.encode(token, 0)
	if err != nil { t.Fatal(err) }
	decodedToken, err := server.decode(handshake)
	if err != nil || !bytes.Equal(decodedToken, token) {
		t.Fatalf("handshake round-trip failed: err=%v", err)
	}

	client.tcpKey = bytes.Repeat([]byte{0x21}, 32)
	server.tcpKey = append([]byte(nil), client.tcpKey...)
	payload := bytes.Repeat([]byte{0x7A}, 72)
	frame, err := client.encode(payload, 6)
	if err != nil { t.Fatal(err) }
	decoded, err := server.decode(frame)
	if err != nil { t.Fatalf("decode encrypted envelope: %v", err) }
	if !bytes.Equal(decoded, payload) {
		t.Fatal("encrypted envelope payload mismatch")
	}
}

func TestReadMideaPacketRejectsLengthBeforeAllocation(t *testing.T) {
	header := make([]byte, mideaHeaderLength)
	header[0], header[1] = 0x5A, 0x5A
	header[4], header[5] = 0xFF, 0xFF
	_, err := readMideaPacket(bytes.NewReader(header))
	if !errors.Is(err, ErrMideaFrame) {
		t.Fatalf("expected oversize frame rejection, got %v", err)
	}
}

func TestA1LANDriverRequiresV3CredentialsAndPrivateLAN(t *testing.T) {
	driver, err := NewA1LANDriver(ProtocolMSmartV3)
	if err != nil { t.Fatal(err) }
	config := A1LANConfig{
		Device: Device{ID: "12345", Name: "Basement dehumidifier", TypeCode: "A1",
			Kind: KindDehumidifier, Protocol: ProtocolMSmartV3, Host: "192.168.1.20", Port: 6444},
		ApplianceID: 12345,
	}
	if err := driver.Register(config); !errors.Is(err, ErrInvalidDevice) {
		t.Fatalf("missing V3 token/key should be rejected, got %v", err)
	}
	config.TokenHex = strings.Repeat("11", 64)
	config.KeyHex = strings.Repeat("22", 32)
	config.Device.Host = "203.0.113.10"
	if err := driver.Register(config); !errors.Is(err, ErrInvalidDevice) {
		t.Fatalf("public LAN address should be rejected, got %v", err)
	}
	config.Device.Host = "192.168.1.20"
	if err := driver.Register(config); err != nil {
		t.Fatalf("valid V3 device config rejected: %v", err)
	}
	devices, err := driver.Discover(context.Background())
	if err != nil || len(devices) != 1 {
		t.Fatalf("configured device listing failed: count=%d err=%v", len(devices), err)
	}
}

func TestA1TemperatureConversionIsStable(t *testing.T) {
	for raw, want := range map[byte]float64{0: -25, 50: 0, 92: 21, 94: 22, 100: 25} {
		got := (float64(raw) - 50) / 2
		if math.Abs(got-want) > 0.00001 {
			t.Fatalf("raw %d => %v, want %v", raw, got, want)
		}
	}
}
