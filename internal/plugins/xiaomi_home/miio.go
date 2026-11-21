// Package xiaomi_home provides a constrained local miIO transport for Xiaomi IP devices.
// It does not automate Xiaomi account login and never asks for an account password.
package xiaomi_home

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"sort"
	"sync"
	"strings"
	"time"
)

const (
	miIOPort       = 54321
	miIOMagic      = 0x2131
	packetHeader   = 32
	maxPacketBytes = 4096
)

var methodPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.]{0,63}$`)

type Client struct {
	timeout time.Duration
}

type DeviceInfo struct {
	IP       string `json:"ip"`
	DeviceID uint32 `json:"device_id"`
	Stamp    uint32 `json:"device_timestamp"`
	Model    string `json:"model,omitempty"`
	Firmware string `json:"firmware,omitempty"`
	MAC      string `json:"mac,omitempty"`
	Hardware string `json:"hardware,omitempty"`
}

type miIOResponse struct {
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error,omitempty"`
}

// NewClient creates a bounded-timeout UDP client. Commands are sent only to
// private LAN IP literals; hostnames and public/loopback destinations are rejected.
func NewClient(timeout time.Duration) *Client {
	if timeout < 500*time.Millisecond || timeout > 10*time.Second {
		timeout = 2 * time.Second
	}
	return &Client{timeout: timeout}
}

// Handshake sends the 32-byte miIO hello packet to one private LAN IP.
func (c *Client) Handshake(ctx context.Context, target string) (DeviceInfo, error) {
	ip, err := validateTarget(target)
	if err != nil {
		return DeviceInfo{}, err
	}
	conn, addr, err := c.dial(ctx, ip)
	if err != nil {
		return DeviceInfo{}, err
	}
	defer conn.Close()
	if _, err := conn.Write(helloPacket()); err != nil {
		return DeviceInfo{}, fmt.Errorf("xiaomi: send miIO handshake: %w", err)
	}
	packet, err := c.readPacket(conn)
	if err != nil {
		return DeviceInfo{}, err
	}
	deviceID, stamp, err := parseHello(packet)
	if err != nil {
		return DeviceInfo{}, err
	}
	return DeviceInfo{IP: addr.IP.String(), DeviceID: deviceID, Stamp: stamp}, nil
}


// DiscoverLAN finds miIO devices on private IPv4 networks attached to this host.
// It uses the protocol's unauthenticated hello handshake; it never retrieves or
// guesses device tokens. Networks wider than /24 are narrowed to the /24 that
// contains the host interface address, and at most 512 addresses are probed.
func (c *Client) DiscoverLAN(ctx context.Context) ([]DeviceInfo, error) {
	targets, err := localIPv4Targets()
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return []DeviceInfo{}, nil
	}

	timeout := c.timeout
	if timeout <= 0 || timeout > 750*time.Millisecond {
		timeout = 750 * time.Millisecond
	}
	scanner := NewClient(timeout)
	sem := make(chan struct{}, 32)
	results := make(chan DeviceInfo, len(targets))
	var wg sync.WaitGroup

	for _, target := range targets {
		select {
		case <-ctx.Done():
		case sem <- struct{}{}:
		}
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()
			probeCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			info, probeErr := scanner.Handshake(probeCtx, ip)
			if probeErr == nil {
				results <- info
			}
		}(target)
	}
	wg.Wait()
	close(results)

	found := make([]DeviceInfo, 0, len(results))
	seen := make(map[string]struct{}, len(results))
	for info := range results {
		if _, ok := seen[info.IP]; ok {
			continue
		}
		seen[info.IP] = struct{}{}
		found = append(found, info)
	}
	sort.Slice(found, func(i, j int) bool { return found[i].IP < found[j].IP })
	return found, nil
}

// localIPv4Targets returns bounded host targets from private LAN interfaces only.
// It deliberately excludes loopback, link-local, public and non-IPv4 interfaces.
func localIPv4Targets() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("xiaomi: enumerate LAN interfaces: %w", err)
	}
	seen := make(map[string]struct{})
	var targets []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagPointToPoint != 0 ||
			strings.HasPrefix(iface.Name, "wg") || strings.HasPrefix(iface.Name, "tailscale") ||
			strings.HasPrefix(iface.Name, "docker") || strings.HasPrefix(iface.Name, "br-") ||
			strings.HasPrefix(iface.Name, "veth") || strings.HasPrefix(iface.Name, "tun") ||
			strings.HasPrefix(iface.Name, "tap") {
			continue
		}
		addresses, addrErr := iface.Addrs()
		if addrErr != nil {
			continue
		}
		for _, address := range addresses {
			ip, network, parseErr := net.ParseCIDR(address.String())
			if parseErr != nil {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil || !ip4.IsPrivate() {
				continue
			}
			for _, target := range subnetTargets(ip4, network) {
				if _, exists := seen[target]; exists {
					continue
				}
				seen[target] = struct{}{}
				targets = append(targets, target)
				if len(targets) >= 512 {
					break
				}
			}
			if len(targets) >= 512 {
				break
			}
		}
		if len(targets) >= 512 {
			break
		}
	}
	sort.Strings(targets)
	return targets, nil
}

// subnetTargets expands a local IPv4 subnet into usable hosts. Large private
// networks are narrowed to the /24 containing the interface address.
func subnetTargets(ip net.IP, network *net.IPNet) []string {
	ip4 := ip.To4()
	if ip4 == nil || network == nil {
		return nil
	}
	prefix, bits := network.Mask.Size()
	if bits != 32 || prefix >= 31 {
		return nil
	}
	if prefix < 24 {
		prefix = 24
	}
	mask := net.CIDRMask(prefix, 32)
	baseIP := ip4.Mask(mask)
	hostCount := uint32(1) << uint(32-prefix)
	base := binary.BigEndian.Uint32(baseIP)
	local := binary.BigEndian.Uint32(ip4)
	targets := make([]string, 0, hostCount)
	for offset := uint32(1); offset < hostCount-1; offset++ {
		value := base + offset
		if value == local {
			continue
		}
		var raw [4]byte
		binary.BigEndian.PutUint32(raw[:], value)
		candidate := net.IPv4(raw[0], raw[1], raw[2], raw[3])
		if candidate.IsPrivate() {
			targets = append(targets, candidate.String())
		}
	}
	return targets
}

// Info authenticates to a local device with its 16-byte (32 hex character)
// miIO token and retrieves its model and firmware identity.
func (c *Client) Info(ctx context.Context, target, token string) (DeviceInfo, error) {
	ip, err := validateTarget(target)
	if err != nil {
		return DeviceInfo{}, err
	}
	result, err := c.Call(ctx, target, token, "miIO.info", []any{})
	if err != nil {
		return DeviceInfo{}, err
	}
	var payload struct {
		Model    string `json:"model"`
		Firmware string `json:"fw_ver"`
		MAC      string `json:"mac"`
		Hardware string `json:"hw_ver"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return DeviceInfo{}, errors.New("xiaomi: device returned invalid miIO.info payload")
	}
	info := DeviceInfo{IP: ip.String(), Model: payload.Model, Firmware: payload.Firmware, MAC: payload.MAC, Hardware: payload.Hardware}
	if info.Model == "" {
		return DeviceInfo{}, errors.New("xiaomi: token accepted but device did not return a model")
	}
	return info, nil
}

// Call invokes a single miIO method on a locally paired device.
// The method is syntax-checked; callers must apply a per-device command allowlist
// before exposing write-capable methods to users.
func (c *Client) Call(ctx context.Context, target, tokenHex, method string, params any) (json.RawMessage, error) {
	ip, err := validateTarget(target)
	if err != nil {
		return nil, err
	}
	token, err := decodeToken(tokenHex)
	if err != nil {
		return nil, err
	}
	if !methodPattern.MatchString(method) {
		return nil, errors.New("xiaomi: invalid miIO method")
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil || len(paramsJSON) > 64*1024 {
		return nil, errors.New("xiaomi: invalid or oversized command parameters")
	}

	conn, addr, err := c.dial(ctx, ip)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if _, err := conn.Write(helloPacket()); err != nil {
		return nil, fmt.Errorf("xiaomi: send miIO handshake: %w", err)
	}
	hello, err := c.readPacket(conn)
	if err != nil {
		return nil, err
	}
	deviceID, deviceStamp, err := parseHello(hello)
	if err != nil {
		return nil, err
	}

	requestID := uint64(time.Now().UnixNano()%9999) + 1
	request := map[string]any{"id": requestID, "method": method, "params": params}
	plain, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("xiaomi: encode command: %w", err)
	}
	plain = append(plain, 0)
	encrypted, err := encryptPayload(plain, token)
	if err != nil {
		return nil, err
	}
	packet, err := buildPacket(deviceID, deviceStamp+1, encrypted, token)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write(packet); err != nil {
		return nil, fmt.Errorf("xiaomi: send command to %s: %w", addr.IP, err)
	}
	responsePacket, err := c.readPacket(conn)
	if err != nil {
		return nil, err
	}
	responseBody, err := parseResponse(responsePacket, token)
	if err != nil {
		return nil, err
	}
	var response miIOResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, errors.New("xiaomi: invalid miIO response")
	}
	if response.ID != requestID {
		return nil, errors.New("xiaomi: response ID does not match the request")
	}
	if len(response.Error) != 0 && string(response.Error) != "null" {
		return nil, errors.New("xiaomi: device rejected the command")
	}
	if len(response.Result) == 0 {
		return json.RawMessage("null"), nil
	}
	return response.Result, nil
}

func (c *Client) dial(ctx context.Context, ip net.IP) (*net.UDPConn, *net.UDPAddr, error) {
	addr := &net.UDPAddr{IP: ip, Port: miIOPort}
	dialer := net.Dialer{Timeout: c.timeout}
	raw, err := dialer.DialContext(ctx, "udp", addr.String())
	if err != nil {
		return nil, nil, fmt.Errorf("xiaomi: connect to local device: %w", err)
	}
	conn, ok := raw.(*net.UDPConn)
	if !ok {
		raw.Close()
		return nil, nil, errors.New("xiaomi: unexpected transport")
	}
	deadline := time.Now().Add(c.timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("xiaomi: set UDP deadline: %w", err)
	}
	return conn, addr, nil
}

func (c *Client) readPacket(conn *net.UDPConn) ([]byte, error) {
	buffer := make([]byte, maxPacketBytes)
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("xiaomi: device did not answer on UDP %d: %w", miIOPort, err)
	}
	packet := buffer[:n]
	if len(packet) < packetHeader || len(packet) > maxPacketBytes || binary.BigEndian.Uint16(packet[:2]) != miIOMagic {
		return nil, errors.New("xiaomi: invalid miIO packet header")
	}
	if int(binary.BigEndian.Uint16(packet[2:4])) != len(packet) {
		return nil, errors.New("xiaomi: invalid miIO packet length")
	}
	return append([]byte(nil), packet...), nil
}

func helloPacket() []byte {
	packet, _ := hex.DecodeString("21310020ffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	return packet
}

func parseHello(packet []byte) (uint32, uint32, error) {
	if len(packet) != packetHeader || binary.BigEndian.Uint16(packet[:2]) != miIOMagic ||
		binary.BigEndian.Uint16(packet[2:4]) != packetHeader {
		return 0, 0, errors.New("xiaomi: invalid miIO handshake response")
	}
	deviceID := binary.BigEndian.Uint32(packet[8:12])
	stamp := binary.BigEndian.Uint32(packet[12:16])
	if deviceID == 0 || deviceID == 0xffffffff || stamp == 0xffffffff {
		return 0, 0, errors.New("xiaomi: invalid device identity in handshake")
	}
	return deviceID, stamp, nil
}

func buildPacket(deviceID, stamp uint32, encrypted, token []byte) ([]byte, error) {
	length := packetHeader + len(encrypted)
	if length > maxPacketBytes || length > 0xffff {
		return nil, errors.New("xiaomi: command packet exceeds maximum size")
	}
	packet := make([]byte, length)
	binary.BigEndian.PutUint16(packet[0:2], miIOMagic)
	binary.BigEndian.PutUint16(packet[2:4], uint16(length))
	// Bytes 4..7 are reserved and remain zero.
	binary.BigEndian.PutUint32(packet[8:12], deviceID)
	binary.BigEndian.PutUint32(packet[12:16], stamp)
	copy(packet[packetHeader:], encrypted)
	digestInput := make([]byte, 0, 16+len(token)+len(encrypted))
	digestInput = append(digestInput, packet[:16]...)
	digestInput = append(digestInput, token...)
	digestInput = append(digestInput, encrypted...)
	checksum := md5.Sum(digestInput) // Required by the miIO wire protocol, not used as a password hash.
	copy(packet[16:32], checksum[:])
	return packet, nil
}

func parseResponse(packet, token []byte) ([]byte, error) {
	if len(packet) <= packetHeader {
		return nil, errors.New("xiaomi: device returned an empty command response")
	}
	encrypted := packet[packetHeader:]
	digestInput := make([]byte, 0, 16+len(token)+len(encrypted))
	digestInput = append(digestInput, packet[:16]...)
	digestInput = append(digestInput, token...)
	digestInput = append(digestInput, encrypted...)
	expected := md5.Sum(digestInput) // Required by the miIO wire protocol.
	if subtle.ConstantTimeCompare(expected[:], packet[16:32]) != 1 {
		return nil, errors.New("xiaomi: miIO response authentication failed; check device token")
	}
	plain, err := decryptPayload(encrypted, token)
	if err != nil {
		return nil, errors.New("xiaomi: could not decrypt miIO response; check device token")
	}
	plain = bytes.TrimRight(plain, "\x00")
	return plain, nil
}

func encryptPayload(plaintext, token []byte) ([]byte, error) {
	block, err := newTokenCipher(token)
	if err != nil {
		return nil, err
	}
	padding := aes.BlockSize - len(plaintext)%aes.BlockSize
	padded := append(append([]byte(nil), plaintext...), bytes.Repeat([]byte{byte(padding)}, padding)...)
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, tokenIV(token)).CryptBlocks(out, padded)
	return out, nil
}

func decryptPayload(ciphertext, token []byte) ([]byte, error) {
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("xiaomi: invalid encrypted payload length")
	}
	block, err := newTokenCipher(token)
	if err != nil {
		return nil, err
	}
	plain := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, tokenIV(token)).CryptBlocks(plain, ciphertext)
	padding := int(plain[len(plain)-1])
	if padding < 1 || padding > aes.BlockSize || padding > len(plain) {
		return nil, errors.New("xiaomi: invalid miIO padding")
	}
	var bad byte
	for _, b := range plain[len(plain)-padding:] {
		bad |= b ^ byte(padding)
	}
	if bad != 0 {
		return nil, errors.New("xiaomi: invalid miIO padding")
	}
	return plain[:len(plain)-padding], nil
}

func newTokenCipher(token []byte) (cipher.Block, error) {
	if len(token) != 16 {
		return nil, errors.New("xiaomi: device token must be 16 bytes")
	}
	key := md5.Sum(token)
	return aes.NewCipher(key[:])
}

func tokenIV(token []byte) []byte {
	key := md5.Sum(token)
	input := make([]byte, 0, len(key)+len(token))
	input = append(input, key[:]...)
	input = append(input, token...)
	iv := md5.Sum(input)
	return iv[:]
}

func decodeToken(tokenHex string) ([]byte, error) {
	tokenHex = strings.TrimSpace(tokenHex)
	if len(tokenHex) != 32 {
		return nil, errors.New("xiaomi: device token must contain exactly 32 hexadecimal characters")
	}
	token, err := hex.DecodeString(tokenHex)
	if err != nil || len(token) != 16 {
		return nil, errors.New("xiaomi: device token must be hexadecimal")
	}
	return token, nil
}

func validateTarget(target string) (net.IP, error) {
	target = strings.TrimSpace(target)
	if target == "" || strings.ContainsAny(target, "/?#@\\\r\n\t :") {
		return nil, errors.New("xiaomi: target must be a private LAN IP literal")
	}
	ip := net.ParseIP(target)
	if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() {
		return nil, errors.New("xiaomi: target must be a private LAN IP literal")
	}
	if !ip.IsPrivate() {
		return nil, errors.New("xiaomi: target must be a private LAN IP literal")
	}
	return ip, nil
}
