package knx_test

import (
	"testing"

	"github.com/domus-hub/domus/internal/plugins/knx"
)

// FuzzDecodeKNXHeader tests that KNXnet/IP 6-byte header parsing never panics on corrupt bytes.
func FuzzDecodeKNXHeader(f *testing.F) {
	f.Add([]byte{0x06, 0x10, 0x04, 0x20, 0x00, 0x15}) // Standard valid TUNNELLING_REQ header
	f.Add([]byte{0x06, 0x10, 0x02, 0x01, 0x00, 0x06}) // SEARCH_REQ header
	f.Add([]byte{})
	f.Add([]byte{0x06, 0x10})
	f.Add([]byte{0x05, 0x10, 0x00, 0x00, 0x00, 0x00}) // Invalid header length
	f.Add([]byte{0x06, 0x20, 0x00, 0x00, 0x00, 0x00}) // Unsupported protocol version
	f.Add(make([]byte, 512))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = knx.DecodeKNXHeader(data)
	})
}

// FuzzParseCEMIFrame tests cEMI message parsing against arbitrary byte sequences.
func FuzzParseCEMIFrame(f *testing.F) {
	// Standard L_Data.ind 1-bit boolean frame (switch ON)
	f.Add([]byte{
		0x29, 0x00, 0xBC, 0xE0, 0x11, 0x01, 0x08, 0x01, 0x01, 0x00, 0x81,
	})
	// Extended cEMI frame with 2-byte DPT9 temperature payload
	f.Add([]byte{
		0x29, 0x00, 0xBC, 0xE0, 0x11, 0x02, 0x08, 0x02, 0x03, 0x00, 0x80, 0x0C, 0x1A,
	})
	// Truncated frames
	f.Add([]byte{})
	f.Add([]byte{0x29})
	f.Add([]byte{0x29, 0x00})
	f.Add([]byte{0x29, 0x05, 0x00, 0x00}) // addInfo claims 5 bytes but truncated
	f.Add([]byte{0x29, 0x00, 0xBC, 0xE0, 0x11, 0x01, 0x08, 0x01, 0xFF, 0x00, 0x80}) // dataLen 255 but no payload

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = knx.ParseCEMIFrame(data)
	})
}

// FuzzDecodeDPT tests KNX Data Point Type decoders against arbitrary binary data.
func FuzzDecodeDPT(f *testing.F) {
	f.Add([]byte{0x0C, 0x1A})             // Valid DPT9 (21.00 °C)
	f.Add([]byte{0x80, 0x00})             // Negative zero / min DPT9
	f.Add([]byte{0x7F, 0xFF})             // Max positive DPT9
	f.Add([]byte{0x44, 0x9A, 0x40, 0x00}) // Valid DPT14 (1234.0 W)
	f.Add([]byte{0x7F, 0x80, 0x00, 0x00}) // IEEE 754 +Infinity
	f.Add([]byte{0x7F, 0xC0, 0x00, 0x00}) // IEEE 754 NaN
	f.Add([]byte{})
	f.Add([]byte{0x01})
	f.Add([]byte{0x00, 0x01, 0x02})

	f.Fuzz(func(t *testing.T, data []byte) {
		// DPT9 (temperature)
		_, _ = knx.DecodeDPT9(data)

		// DPT14 (4-byte power/energy)
		_, _ = knx.DecodeDPT14(data)

		// DPT1 (1-bit boolean) and DPT5 (8-bit scaling)
		if len(data) > 0 {
			_ = knx.DecodeDPT1(data[0])
			_ = knx.DecodeDPT5(data[0])
		}
	})
}

// FuzzParseAddresses tests address string parsing (Group and Individual) against arbitrary string inputs.
func FuzzParseAddresses(f *testing.F) {
	f.Add("1/2/3")
	f.Add("0/0/0")
	f.Add("31/7/255")
	f.Add("1.1.250")
	f.Add("0.0.0")
	f.Add("15.15.255")
	f.Add("")
	f.Add("1/2")
	f.Add("1/2/3/4")
	f.Add("32/0/0")
	f.Add("0/8/0")
	f.Add("0/0/256")
	f.Add("invalid/string/address")
	f.Add("1.1")
	f.Add("16.0.0")
	f.Add("99999999999999999999999999999999999")

	f.Fuzz(func(t *testing.T, addrStr string) {
		_, _ = knx.ParseGroupAddress(addrStr)
		_, _ = knx.ParseIndividualAddress(addrStr)
	})
}
