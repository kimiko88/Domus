// Package tuya_local implements local-first LAN communication with Tuya and Smart Life devices.
package tuya_local

import (
	"bytes"
	"crypto/aes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
)

var (
	PrefixHeader = []byte{0x00, 0x00, 0x55, 0xAA}
	SuffixHeader = []byte{0x00, 0x00, 0xAA, 0x55}

	ErrInvalidPrefix   = errors.New("tuya: invalid prefix header")
	ErrInvalidSuffix   = errors.New("tuya: invalid suffix header")
	ErrPayloadTooShort = errors.New("tuya: packet payload is too short")
	ErrChecksumFailed  = errors.New("tuya: checksum validation failed")
)

// Tuya Command constants
const (
	CmdUDP        uint32 = 0x00
	CmdAPConfig   uint32 = 0x01
	CmdActive     uint32 = 0x02
	CmdBind       uint32 = 0x03
	CmdRenameGW   uint32 = 0x04
	CmdReset      uint32 = 0x05
	CmdControl    uint32 = 0x07 // Control command for v3.1 / v3.3
	CmdStatus     uint32 = 0x08 // Status update from device
	CmdHeartbeat  uint32 = 0x09 // Ping/keepalive
	CmdDpQuery    uint32 = 0x0A // Query DPS status
	CmdQueryHDL   uint32 = 0x0B
	CmdControlNew uint32 = 0x0D // Control command for v3.4 / v3.5
	CmdEnableWifi uint32 = 0x0E
	CmdDpQueryNew uint32 = 0x10
	CmdSceneExec  uint32 = 0x11
	CmdUpdateDPS  uint32 = 0x12
)

// Packet represents a decoded Tuya protocol frame.
type Packet struct {
	Sequence uint32
	Command  uint32
	Payload  []byte
	ReturnCode uint32
}

// EncodePacket constructs a binary Tuya frame for protocol v3.1 or v3.3.
func EncodePacket(seq, cmd uint32, payload []byte, localKey []byte, version string) ([]byte, error) {
	var body []byte
	var err error

	if len(payload) > 0 {
		if version == "3.3" || version == "3.1" {
			body, err = encryptAES_ECB(payload, localKey)
			if err != nil {
				return nil, fmt.Errorf("tuya: aes encrypt error: %w", err)
			}
			if version == "3.3" {
				// Protocol 3.3 includes header "3.3\0\0\0\0\0\0\0\0\0\0\0\0" (15 bytes) before ciphertext for some commands,
				// or raw ciphertext directly. Standard LAN control sends "3.3" + 12 zero bytes + encrypted data:
				prefix := append([]byte("3.3"), make([]byte, 12)...)
				body = append(prefix, body...)
			}
		} else {
			// Plain / unsupported fallback
			body = payload
		}
	}

	buf := new(bytes.Buffer)
	buf.Write(PrefixHeader)
	_ = binary.Write(buf, binary.BigEndian, seq)
	_ = binary.Write(buf, binary.BigEndian, cmd)

	// Payload length: len(body) + 4 bytes CRC + 4 bytes suffix
	payloadLen := uint32(len(body) + 8)
	_ = binary.Write(buf, binary.BigEndian, payloadLen)
	buf.Write(body)

	// Calculate IEEE CRC32 over prefix, seq, cmd, len, body
	crc := crc32.ChecksumIEEE(buf.Bytes())
	_ = binary.Write(buf, binary.BigEndian, crc)
	buf.Write(SuffixHeader)

	return buf.Bytes(), nil
}

// DecodePacket unpacks and verifies a Tuya binary frame.
func DecodePacket(data []byte, localKey []byte) (*Packet, error) {
	if len(data) < 16 {
		return nil, ErrPayloadTooShort
	}

	if !bytes.Equal(data[:4], PrefixHeader) {
		return nil, ErrInvalidPrefix
	}

	if !bytes.Equal(data[len(data)-4:], SuffixHeader) {
		return nil, ErrInvalidSuffix
	}

	seq := binary.BigEndian.Uint32(data[4:8])
	cmd := binary.BigEndian.Uint32(data[8:12])
	payloadLen := binary.BigEndian.Uint32(data[12:16])

	expectedTotal := 16 + payloadLen
	if uint32(len(data)) < expectedTotal {
		return nil, ErrPayloadTooShort
	}

	// Suffix is last 4 bytes, CRC is 4 bytes before suffix
	bodyEnd := len(data) - 8
	body := data[16:bodyEnd]

	// Verify CRC
	expectedCRC := binary.BigEndian.Uint32(data[bodyEnd : bodyEnd+4])
	calculatedCRC := crc32.ChecksumIEEE(data[:bodyEnd])
	if expectedCRC != calculatedCRC {
		return nil, ErrChecksumFailed
	}

	// Decrypt payload if encrypted
	decrypted := body
	if len(body) > 0 {
		var toDecrypt []byte
		if bytes.HasPrefix(body, []byte("3.3")) && len(body) > 15 {
			toDecrypt = body[15:]
		} else {
			toDecrypt = body
		}

		// Try ECB decryption if localKey is provided
		if len(localKey) == 16 && len(toDecrypt)%16 == 0 {
			if dec, err := decryptAES_ECB(toDecrypt, localKey); err == nil {
				decrypted = dec
			}
		}
	}

	return &Packet{
		Sequence: seq,
		Command:  cmd,
		Payload:  decrypted,
	}, nil
}

// BuildControlPayload formats a DPS command into the expected JSON structure.
func BuildControlPayload(devID string, dps map[string]any) ([]byte, error) {
	payload := map[string]any{
		"devId": devID,
		"uid":   devID,
		"t":     fmt.Sprintf("%d", 0), // timestamp or 0
		"dps":   dps,
	}
	return json.Marshal(payload)
}

// encryptAES_ECB encrypts data using AES ECB with PKCS7 padding.
func encryptAES_ECB(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	data = pkcs7Pad(data, block.BlockSize())
	encrypted := make([]byte, len(data))
	for bs := 0; bs < len(data); bs += block.BlockSize() {
		block.Encrypt(encrypted[bs:bs+block.BlockSize()], data[bs:bs+block.BlockSize()])
	}
	return encrypted, nil
}

// decryptAES_ECB decrypts data using AES ECB and strips PKCS7 padding.
func decryptAES_ECB(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data)%block.BlockSize() != 0 {
		return nil, errors.New("tuya: ciphertext is not a multiple of block size")
	}
	decrypted := make([]byte, len(data))
	for bs := 0; bs < len(data); bs += block.BlockSize() {
		block.Decrypt(decrypted[bs:bs+block.BlockSize()], data[bs:bs+block.BlockSize()])
	}
	return pkcs7Unpad(decrypted)
}

func pkcs7Pad(src []byte, blockSize int) []byte {
	pad := blockSize - len(src)%blockSize
	return append(src, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

func pkcs7Unpad(src []byte) ([]byte, error) {
	length := len(src)
	if length == 0 {
		return nil, errors.New("tuya: empty payload unpadding")
	}
	unpadding := int(src[length-1])
	if unpadding > length || unpadding == 0 {
		return src, nil // unpadded raw fallback
	}
	for i := length - unpadding; i < length; i++ {
		if src[i] != byte(unpadding) {
			return src, nil // not valid padding, return as is
		}
	}
	return src[:length-unpadding], nil
}

// ComputeMD5Key generates a 16-byte MD5 digest for Tuya UDP discovery payload decryption.
func ComputeMD5Key(key []byte) []byte {
	hash := md5.Sum(key)
	return hash[:]
}

// ComputeHMAC_SHA256 returns HMAC-SHA256 hex string for protocol v3.4 signatures.
func ComputeHMAC_SHA256(data, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
