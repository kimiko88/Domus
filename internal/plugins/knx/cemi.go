package knx

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// KNXnet/IP Service Type Identifiers
const (
	ServiceSearchReq           uint16 = 0x0201
	ServiceSearchResp          uint16 = 0x0202
	ServiceConnectReq          uint16 = 0x0209
	ServiceConnectResp         uint16 = 0x020A
	ServiceConnectionStateReq  uint16 = 0x020B
	ServiceConnectionStateResp uint16 = 0x020C
	ServiceDisconnectReq       uint16 = 0x0209
	ServiceDisconnectResp      uint16 = 0x020A
	ServiceTunnellingReq       uint16 = 0x0420
	ServiceTunnellingAck       uint16 = 0x0421
)

// cEMI Message Codes
const (
	CEMIMsgDataReq byte = 0x11 // L_Data.req (outbound from Domus to KNX bus)
	CEMIMsgDataCon byte = 0x2E // L_Data.con (confirmation)
	CEMIMsgDataInd byte = 0x29 // L_Data.ind (inbound from KNX bus to Domus)
)

// KNXHeader represents the 6-byte KNXnet/IP packet header.
type KNXHeader struct {
	HeaderLength    uint8  // Always 0x06
	ProtocolVersion uint8  // Always 0x10
	ServiceType     uint16 // e.g. 0x0420 TUNNELLING_REQ
	TotalLength     uint16 // Total packet length including header
}

// Encode serializes the header into 6 bytes.
func (h KNXHeader) Encode() []byte {
	b := make([]byte, 6)
	b[0] = 0x06
	b[1] = 0x10
	binary.BigEndian.PutUint16(b[2:4], h.ServiceType)
	binary.BigEndian.PutUint16(b[4:6], h.TotalLength)
	return b
}

// DecodeKNXHeader parses the 6-byte KNXnet/IP header.
func DecodeKNXHeader(b []byte) (KNXHeader, error) {
	if len(b) < 6 {
		return KNXHeader{}, errors.New("knx: packet too short for header")
	}
	if b[0] != 0x06 || b[1] != 0x10 {
		return KNXHeader{}, fmt.Errorf("knx: unsupported header or version 0x%02X 0x%02X", b[0], b[1])
	}
	return KNXHeader{
		HeaderLength:    b[0],
		ProtocolVersion: b[1],
		ServiceType:     binary.BigEndian.Uint16(b[2:4]),
		TotalLength:     binary.BigEndian.Uint16(b[4:6]),
	}, nil
}

// CEMIFrame represents a Common External Message Interface data frame.
type CEMIFrame struct {
	MessageCode byte
	Source      uint16
	Destination uint16 // Group Address or Individual Address
	IsGroup     bool
	APCI        uint16 // Application Layer Control (e.g. 0x0080 for write, 0x0040 for response, 0x0000 for read)
	Data        []byte
}

// BuildCEMIFrame encodes a CEMIFrame into raw bytes for L_Data.req.
func BuildCEMIFrame(src, dst uint16, apci uint16, data []byte) []byte {
	// Standard cEMI L_Data.req:
	// Byte 0: Message Code (0x11)
	// Byte 1: AddInfo Length (0x00)
	// Byte 2: Control Field 1 (0xBC - standard frame, repeat flag, normal priority)
	// Byte 3: Control Field 2 (0xE0 - group address destination, hop count 6)
	// Bytes 4-5: Source Address
	// Bytes 6-7: Destination Address
	// Byte 8: Data Length (number of bytes following APCI + 1)
	// Bytes 9-10: APCI + data (or APCI followed by data)

	var frame []byte
	ctrl2 := byte(0xE0) // Group address
	ctrl1 := byte(0xBC)

	if len(data) == 0 {
		// Small data embedded in APCI (e.g. 1-bit boolean or read)
		frame = make([]byte, 11)
		frame[0] = CEMIMsgDataReq
		frame[1] = 0x00
		frame[2] = ctrl1
		frame[3] = ctrl2
		binary.BigEndian.PutUint16(frame[4:6], src)
		binary.BigEndian.PutUint16(frame[6:8], dst)
		frame[8] = 0x01 // 1 byte for APCI
		binary.BigEndian.PutUint16(frame[9:11], apci)
	} else if len(data) == 1 && (apci == 0x0080 || apci == 0x0040) && data[0] <= 0x3F {
		// 6-bit data optimization (e.g. DPT1 boolean): embed directly in lower 6 bits of APCI
		frame = make([]byte, 11)
		frame[0] = CEMIMsgDataReq
		frame[1] = 0x00
		frame[2] = ctrl1
		frame[3] = ctrl2
		binary.BigEndian.PutUint16(frame[4:6], src)
		binary.BigEndian.PutUint16(frame[6:8], dst)
		frame[8] = 0x01
		packedAPCI := apci | uint16(data[0]&0x3F)
		binary.BigEndian.PutUint16(frame[9:11], packedAPCI)
	} else {
		// Extended data (> 6 bits: DPT5 1 byte, DPT9 2 bytes, DPT14 4 bytes)
		// Length byte = 1 + len(data)
		dataLen := byte(1 + len(data))
		frame = make([]byte, 11+len(data))
		frame[0] = CEMIMsgDataReq
		frame[1] = 0x00
		frame[2] = ctrl1
		frame[3] = ctrl2
		binary.BigEndian.PutUint16(frame[4:6], src)
		binary.BigEndian.PutUint16(frame[6:8], dst)
		frame[8] = dataLen
		binary.BigEndian.PutUint16(frame[9:11], apci)
		copy(frame[11:], data)
	}

	return frame
}

// ParseCEMIFrame parses raw bytes into a CEMIFrame structure.
func ParseCEMIFrame(b []byte) (*CEMIFrame, error) {
	if len(b) < 11 {
		return nil, errors.New("knx: cEMI frame too short")
	}

	msgCode := b[0]
	addInfoLen := int(b[1])
	offset := 2 + addInfoLen

	if len(b) < offset+9 {
		return nil, errors.New("knx: cEMI frame truncated after addInfo")
	}

	// ctrl1 := b[offset]
	ctrl2 := b[offset+1]
	isGroup := (ctrl2 & 0x80) != 0

	src := binary.BigEndian.Uint16(b[offset+2 : offset+4])
	dst := binary.BigEndian.Uint16(b[offset+4 : offset+6])
	dataLen := int(b[offset+6])

	rawAPCI := binary.BigEndian.Uint16(b[offset+7 : offset+9])
	apciType := rawAPCI & 0x03C0

	frame := &CEMIFrame{
		MessageCode: msgCode,
		Source:      src,
		Destination: dst,
		IsGroup:     isGroup,
		APCI:        apciType,
	}

	if dataLen <= 1 {
		// 6-bit inline data
		smallVal := byte(rawAPCI & 0x3F)
		frame.Data = []byte{smallVal}
	} else {
		// Extra data bytes start at offset+9
		extraBytes := dataLen - 1
		if len(b) < offset+9+extraBytes {
			return nil, errors.New("knx: truncated cEMI payload data")
		}
		frame.Data = make([]byte, extraBytes)
		copy(frame.Data, b[offset+9:offset+9+extraBytes])
	}

	return frame, nil
}

// BuildTunnellingRequest encapsulates a cEMI frame into a KNXnet/IP TUNNELLING_REQ packet.
func BuildTunnellingRequest(channelID byte, seqCounter byte, cemiFrame []byte) []byte {
	// Connection Header (4 bytes):
	// Byte 0: Structure Length (0x04)
	// Byte 1: Channel ID
	// Byte 2: Sequence Counter
	// Byte 3: Reserved (0x00)
	connHeader := []byte{0x04, channelID, seqCounter, 0x00}
	body := append(connHeader, cemiFrame...)

	hdr := KNXHeader{
		HeaderLength:    0x06,
		ProtocolVersion: 0x10,
		ServiceType:     ServiceTunnellingReq,
		TotalLength:     uint16(6 + len(body)),
	}

	return append(hdr.Encode(), body...)
}

// BuildTunnellingAck creates a TUNNELLING_ACK packet confirming receipt of a TUNNELLING_REQ.
func BuildTunnellingAck(channelID byte, seqCounter byte, status byte) []byte {
	connHeader := []byte{0x04, channelID, seqCounter, status}
	hdr := KNXHeader{
		HeaderLength:    0x06,
		ProtocolVersion: 0x10,
		ServiceType:     ServiceTunnellingAck,
		TotalLength:     uint16(6 + len(connHeader)),
	}
	return append(hdr.Encode(), connHeader...)
}
