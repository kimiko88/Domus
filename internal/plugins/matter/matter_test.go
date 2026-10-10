package matter

import (
	"context"
	"fmt"
	"testing"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatter_ManualCodeParser(t *testing.T) {
	// 11-digit code test
	code := "34970112332"
	payload, err := ParseManualCode(code)
	require.NoError(t, err)
	assert.Equal(t, uint8(1), payload.Version)
	assert.NotZero(t, payload.SetupPasscode)

	// Invalid length
	_, err = ParseManualCode("12345")
	assert.Error(t, err)

	// Non-numeric
	_, err = ParseManualCode("1234567890a")
	assert.Error(t, err)
}

func TestMatter_QRCodeParser(t *testing.T) {
	qr := "MT:Y.K9042C00KA0648G00"
	payload, err := ParseQRCode(qr)
	require.NoError(t, err)
	assert.Equal(t, uint32(20202021), payload.SetupPasscode)
	assert.Equal(t, uint16(3840), payload.Discriminator)

	_, err = ParseQRCode("INVALID:QR")
	assert.Error(t, err)
}

func TestMatter_TLVEncoding(t *testing.T) {
	bTrue := EncodeTLVBool(true)
	assert.Equal(t, []byte{byte(TLVTypeBooleanTrue)}, bTrue)

	bFalse := EncodeTLVBool(false)
	assert.Equal(t, []byte{byte(TLVTypeBooleanFalse)}, bFalse)

	u32 := EncodeTLVUInt32(0x12345678)
	assert.Equal(t, byte(TLVTypeUnsignedInt4), u32[0])
	assert.Len(t, u32, 5)
}

func TestMatter_AdapterLifecycleAndCommissioning(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	defer bus.Close()

	adapter := NewAdapter("matter", bus)
	ctx := context.Background()
	require.NoError(t, adapter.Start(ctx))
	defer adapter.Stop(ctx)

	assert.Equal(t, "matter", adapter.ID())
	assert.Equal(t, "matter", adapter.Type())

	// Commission device via QR Code
	node, err := adapter.CommissionDevice(ctx, "MT:Y.K9042C00KA0648G00", "Luce Salotto Matter")
	require.NoError(t, err)
	assert.NotZero(t, node.NodeID)
	assert.Equal(t, "Luce Salotto Matter", node.Name)

	nodes := adapter.ListNodes()
	assert.Len(t, nodes, 1)

	entKey := fmt.Sprintf("light.matter_%d", node.NodeID)
	err = adapter.handleCommand(ctx, events.Event{
		ID:   "cmd-1",
		Type: events.CommandRequested,
		Payload: map[string]any{
			"entity_key": entKey,
			"command":    "turn_on",
			"params":     map[string]any{"brightness": float64(200)},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, uint8(200), node.Level)
	assert.True(t, node.OnOff)
}
