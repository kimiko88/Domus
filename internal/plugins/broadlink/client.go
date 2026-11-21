package broadlink

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

var ErrDeviceLocked = errors.New("broadlink: device is locked; disable device lock in the BroadLink app")

type Client struct {
	ip         string
	deviceType uint16
	mac        [6]byte
	deviceID   uint32
	key        [16]byte
	counter    uint16
	timeout    time.Duration
	mu         sync.Mutex
}

func NewClient(ip string, deviceType uint16, mac string, deviceID uint32, key []byte) (*Client, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil || !parsedIP.IsPrivate() {
		return nil, fmt.Errorf("broadlink: target must be a private LAN IP")
	}
	model, _, _ := DeviceModel(deviceType)
	if model == "" {
		return nil, ErrUnsupportedDevice
	}
	parsedMAC, err := parseMAC(mac)
	if err != nil {
		return nil, err
	}
	client := &Client{ip: parsedIP.String(), deviceType: deviceType, mac: parsedMAC, deviceID: deviceID, timeout: 2500 * time.Millisecond}
	if len(key) == 16 {
		copy(client.key[:], key)
	}
	return client, nil
}

func (c *Client) Auth(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	payload := make([]byte, 0x50)
	for i := 0x04; i < 0x14; i++ {
		payload[i] = 0x31
	}
	payload[0x1e] = 0x01
	payload[0x2d] = 0x01
	copy(payload[0x30:], []byte("Domus"))
	plain, err := c.exchange(ctx, commandAuth, payload, initialKey, 0)
	if err != nil {
		return err
	}
	if len(plain) < 20 {
		return fmt.Errorf("broadlink: authentication response is too short")
	}
	c.deviceID = binary.LittleEndian.Uint32(plain[:4])
	copy(c.key[:], plain[4:20])
	return nil
}

func (c *Client) sendSubcommand(ctx context.Context, subcommand uint32, data []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var payload []byte
	modern := usesModernRemoteFrame(c.deviceType)
	if modern {
		payload = make([]byte, 6+len(data))
		binary.LittleEndian.PutUint16(payload[0:2], uint16(4+len(data)))
		binary.LittleEndian.PutUint32(payload[2:6], subcommand)
		copy(payload[6:], data)
	} else {
		payload = make([]byte, 4+len(data))
		binary.LittleEndian.PutUint32(payload[:4], subcommand)
		copy(payload[4:], data)
	}
	plain, err := c.exchange(ctx, commandRemote, payload, c.key, c.deviceID)
	if err != nil {
		return nil, err
	}
	if modern {
		if len(plain) < 2 {
			return nil, fmt.Errorf("broadlink: malformed RM4 response")
		}
		length := int(binary.LittleEndian.Uint16(plain[:2]))
		end := length + 2
		if end > len(plain) || end < 6 {
			return nil, fmt.Errorf("broadlink: malformed RM4 response length")
		}
		return append([]byte(nil), plain[6:end]...), nil
	}
	if len(plain) < 4 {
		return nil, fmt.Errorf("broadlink: malformed RM response")
	}
	return append([]byte(nil), plain[4:]...), nil
}

func (c *Client) exchange(ctx context.Context, command uint16, payload []byte, key [16]byte, deviceID uint32) ([]byte, error) {
	encrypted, err := encryptPayload(payload, key)
	if err != nil {
		return nil, err
	}
	c.counter++
	packet := make([]byte, 0x38+len(encrypted))
	copy(packet[:8], []byte{0x5a, 0xa5, 0xaa, 0x55, 0x5a, 0xa5, 0xaa, 0x55})
	binary.LittleEndian.PutUint16(packet[0x24:0x26], c.deviceType)
	binary.LittleEndian.PutUint16(packet[0x26:0x28], command)
	binary.LittleEndian.PutUint16(packet[0x28:0x2a], c.counter)
	// BroadLink's wire header stores the MAC in reverse byte order.
	for i := 0; i < 6; i++ {
		packet[0x2a+i] = c.mac[5-i]
	}
	binary.LittleEndian.PutUint32(packet[0x30:0x34], deviceID)
	if len(payload) > 0 {
		binary.LittleEndian.PutUint16(packet[0x34:0x36], checksum(payload))
		copy(packet[0x38:], encrypted)
	}
	binary.LittleEndian.PutUint16(packet[0x20:0x22], checksum(packet))

	remote := &net.UDPAddr{IP: net.ParseIP(c.ip), Port: udpPort}
	conn, err := (&net.Dialer{}).DialContext(ctx, "udp4", remote.String())
	if err != nil {
		return nil, fmt.Errorf("broadlink: connect to %s: %w", remote, err)
	}
	defer conn.Close()
	deadline := time.Now().Add(c.timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)
	if _, err := conn.Write(packet); err != nil {
		return nil, fmt.Errorf("broadlink: send packet: %w", err)
	}
	response := make([]byte, 2048)
	n, err := conn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("broadlink: no response from %s: %w", remote, err)
	}
	response = response[:n]
	if len(response) < 0x38 {
		return nil, fmt.Errorf("broadlink: malformed response packet")
	}
	if code := binary.LittleEndian.Uint16(response[0x22:0x24]); code != 0 {
		if code == 0xffff || code == 0xfffe {
			return nil, ErrDeviceLocked
		}
		return nil, fmt.Errorf("broadlink: device returned error 0x%04x", code)
	}
	if len(response) == 0x38 {
		return nil, nil
	}
	return decryptPayload(response[0x38:], key)
}

func (c *Client) SendData(ctx context.Context, data []byte) error {
	if len(data) == 0 || len(data) > 4096 {
		return fmt.Errorf("broadlink: command code must contain between 1 and 4096 bytes")
	}
	_, err := c.sendSubcommand(ctx, 0x02, data)
	return err
}

func (c *Client) StartLearning(ctx context.Context, medium string) error {
	if medium == "rf" {
		if _, _, rf := DeviceModel(c.deviceType); !rf {
			return fmt.Errorf("broadlink: RF learning is available only on RM Pro models")
		}
		if _, err := c.sendSubcommand(ctx, 0x19, nil); err != nil {
			return err
		}
		frequency, err := c.sendSubcommand(ctx, 0x1a, nil)
		if err != nil {
			return err
		}
		var frequencyPayload []byte
		if len(frequency) >= 5 && frequency[0] != 0 {
			frequencyPayload = make([]byte, 4)
			copy(frequencyPayload, frequency[1:5])
		}
		_, err = c.sendSubcommand(ctx, 0x1b, frequencyPayload)
		return err
	}
	if medium != "" && medium != "ir" {
		return fmt.Errorf("broadlink: learning medium must be ir or rf")
	}
	_, err := c.sendSubcommand(ctx, 0x03, nil)
	return err
}

func (c *Client) ReadLearning(ctx context.Context) ([]byte, error) {
	return c.sendSubcommand(ctx, 0x04, nil)
}

func (c *Client) DeviceInfo(ctx context.Context) ([]byte, error) {
	return c.sendSubcommand(ctx, 0x01, nil)
}
