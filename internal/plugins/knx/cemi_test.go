package knx_test

import (
	"testing"

	"github.com/domus-hub/domus/internal/plugins/knx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKNX_CEMIFrame_BuildAndParse(t *testing.T) {
	// 1. GroupValueWrite with 1-bit boolean data (e.g. switch ON)
	src := uint16(0x1101)
	dst := uint16(0x0901)
	cemi := knx.BuildCEMIFrame(src, dst, 0x0080, []byte{0x01})
	require.NotEmpty(t, cemi)

	parsed, err := knx.ParseCEMIFrame(cemi)
	require.NoError(t, err)
	assert.Equal(t, src, parsed.Source)
	assert.Equal(t, dst, parsed.Destination)
	assert.True(t, parsed.IsGroup)
	require.NotEmpty(t, parsed.Data)
	assert.Equal(t, byte(0x01), parsed.Data[0])

	// 2. GroupValueWrite with 2-byte temperature data
	tempData, _ := knx.EncodeDPT9(22.5)
	cemiTemp := knx.BuildCEMIFrame(src, dst, 0x0080, tempData)
	parsedTemp, err := knx.ParseCEMIFrame(cemiTemp)
	require.NoError(t, err)
	assert.Equal(t, dst, parsedTemp.Destination)
	require.Len(t, parsedTemp.Data, 2)
	decodedTemp, err := knx.DecodeDPT9(parsedTemp.Data)
	require.NoError(t, err)
	assert.InDelta(t, 22.5, decodedTemp, 0.1)
}

func TestKNX_TunnellingPackets(t *testing.T) {
	cemi := []byte{0x11, 0x00, 0xBC, 0xE0, 0x00, 0x00, 0x08, 0x01, 0x01, 0x00, 0x81}
	req := knx.BuildTunnellingRequest(2, 5, cemi)
	require.True(t, len(req) >= 10)

	hdr, err := knx.DecodeKNXHeader(req[:6])
	require.NoError(t, err)
	assert.Equal(t, knx.ServiceTunnellingReq, hdr.ServiceType)
	assert.Equal(t, uint16(len(req)), hdr.TotalLength)
	assert.Equal(t, byte(2), req[7]) // channel ID
	assert.Equal(t, byte(5), req[8]) // seq counter

	ack := knx.BuildTunnellingAck(2, 5, 0x00)
	ackHdr, err := knx.DecodeKNXHeader(ack[:6])
	require.NoError(t, err)
	assert.Equal(t, knx.ServiceTunnellingAck, ackHdr.ServiceType)
}
