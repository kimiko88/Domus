package matter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatter_ManualCodeParserFailsClosedUntilControllerExists(t *testing.T) {
	_, err := ParseManualCode("34970112332")
	require.ErrorIs(t, err, ErrMatterStackUnavailable)
	_, err = ParseManualCode("12345")
	require.Error(t, err)
	_, err = ParseManualCode("1234567890a")
	require.Error(t, err)
}

func TestMatter_QRCodeParserFailsClosedUntilRealDecoderExists(t *testing.T) {
	_, err := ParseQRCode("MT:Y.K9042C00KA0648G00")
	require.ErrorIs(t, err, ErrMatterStackUnavailable)
	_, err = ParseQRCode("INVALID:QR")
	require.Error(t, err)
}

func TestMatter_TLVEncoding(t *testing.T) {
	require.Equal(t, []byte{byte(TLVTypeBooleanTrue)}, EncodeTLVBool(true))
	require.Equal(t, []byte{byte(TLVTypeBooleanFalse)}, EncodeTLVBool(false))
	require.Len(t, EncodeTLVUInt32(0x12345678), 5)
}

func TestMatterAdapterFailsClosedWithoutController(t *testing.T) {
	adapter := NewAdapter("matter", nil)
	ctx := context.Background()
	require.ErrorIs(t, adapter.Start(ctx), ErrMatterStackUnavailable)
	require.ErrorIs(t, adapter.HealthCheck(ctx), ErrMatterStackUnavailable)
	node, err := adapter.CommissionDevice(ctx, "MT:Y.K9042C00KA0648G00", "Luce")
	require.ErrorIs(t, err, ErrMatterStackUnavailable)
	require.Nil(t, node)
	require.Empty(t, adapter.ListNodes())
	require.NoError(t, adapter.Stop(ctx))
}
