package knx_test

import (
	"testing"

	"github.com/domus-hub/domus/internal/plugins/knx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKNX_GroupAddressFormatting(t *testing.T) {
	raw, err := knx.ParseGroupAddress("1/2/3")
	require.NoError(t, err)
	assert.Equal(t, uint16((1<<11)|(2<<8)|3), raw)
	assert.Equal(t, "1/2/3", knx.FormatGroupAddress(raw))

	// Out of bounds checks
	_, err = knx.ParseGroupAddress("32/0/0")
	assert.Error(t, err)

	_, err = knx.ParseGroupAddress("1/8/0")
	assert.Error(t, err)

	_, err = knx.ParseGroupAddress("1/0/256")
	assert.Error(t, err)

	_, err = knx.ParseGroupAddress("invalid")
	assert.Error(t, err)
}

func TestKNX_IndividualAddressFormatting(t *testing.T) {
	raw, err := knx.ParseIndividualAddress("1.1.250")
	require.NoError(t, err)
	assert.Equal(t, uint16((1<<12)|(1<<8)|250), raw)
	assert.Equal(t, "1.1.250", knx.FormatIndividualAddress(raw))
}

func TestKNX_DPT1(t *testing.T) {
	assert.Equal(t, byte(0x01), knx.EncodeDPT1(true))
	assert.Equal(t, byte(0x00), knx.EncodeDPT1(false))
	assert.True(t, knx.DecodeDPT1(0x01))
	assert.False(t, knx.DecodeDPT1(0x00))
}

func TestKNX_DPT5(t *testing.T) {
	// 0% -> 0
	assert.Equal(t, byte(0), knx.EncodeDPT5(0))
	assert.Equal(t, 0.0, knx.DecodeDPT5(0))

	// 100% -> 255
	assert.Equal(t, byte(255), knx.EncodeDPT5(100))
	assert.Equal(t, 100.0, knx.DecodeDPT5(255))

	// 50% -> 128
	b := knx.EncodeDPT5(50)
	assert.InDelta(t, 50.0, knx.DecodeDPT5(b), 0.5)
}

func TestKNX_DPT9(t *testing.T) {
	testTemps := []float64{21.5, 20.0, 0.0, -5.25}

	for _, temp := range testTemps {
		encoded, err := knx.EncodeDPT9(temp)
		require.NoError(t, err)
		require.Len(t, encoded, 2)

		decoded, err := knx.DecodeDPT9(encoded)
		require.NoError(t, err)
		assert.InDelta(t, temp, decoded, 0.05)
	}
}

func TestKNX_DPT14(t *testing.T) {
	power := 3450.75
	encoded := knx.EncodeDPT14(power)
	require.Len(t, encoded, 4)

	decoded, err := knx.DecodeDPT14(encoded)
	require.NoError(t, err)
	assert.InDelta(t, power, decoded, 0.1)
}
