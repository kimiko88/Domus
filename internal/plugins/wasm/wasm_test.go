package wasm_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/plugins/wasm"
)

// Minimal valid WASM v1 module with export section
func buildMinimalWasm(exportName string) []byte {
	// Magic: \x00asm, Version: \x01\x00\x00\x00
	buf := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}

	// Export section: ID = 7
	// Payload: 1 export (length of name, name, kind=0, index=0)
	expBytes := []byte{
		0x01, // 1 export
		byte(len(exportName)),
	}
	expBytes = append(expBytes, []byte(exportName)...)
	expBytes = append(expBytes, 0x00, 0x00) // kind=0 (func), idx=0

	// Section header: ID 7, length
	buf = append(buf, wasm.SecExport, byte(len(expBytes)))
	buf = append(buf, expBytes...)
	return buf
}

func TestWasmParserAndSandbox(t *testing.T) {
	// 1. Invalid magic
	_, err := wasm.ParseWasmBinary([]byte("invalid-header"))
	if err != wasm.ErrInvalidWasmMagic {
		t.Errorf("expected ErrInvalidWasmMagic, got %v", err)
	}

	// 2. Valid WASM with domus_handle_event
	wasmData := buildMinimalWasm("domus_handle_event")
	hdr, err := wasm.ParseWasmBinary(wasmData)
	if err != nil {
		t.Fatalf("ParseWasmBinary failed: %v", err)
	}
	if !hdr.HasEventHook {
		t.Errorf("expected HasEventHook to be true")
	}

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})

	// 3. Test Sandbox with capability permissions
	sandbox, err := wasm.NewSandboxRuntime("test_plugin", wasmData, []string{"events:read", "events:write"}, bus)
	if err != nil {
		t.Fatalf("NewSandboxRuntime failed: %v", err)
	}

	ctx := context.Background()
	err = sandbox.DispatchEvent(ctx, events.Event{
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now(),
	})
	if err != nil {
		t.Errorf("DispatchEvent failed: %v", err)
	}

	// 4. Test capability violation
	restrictedSandbox, _ := wasm.NewSandboxRuntime("no_net", wasmData, []string{}, bus)
	err = restrictedSandbox.DispatchEvent(ctx, events.Event{Type: events.EntityStateChanged})
	if err == nil {
		t.Errorf("expected capability error when reading events without permission")
	}
}
