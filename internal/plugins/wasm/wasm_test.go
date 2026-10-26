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

func buildMinimalWasiWasm(importMod, importField, exportName string) []byte {
	buf := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}

	impBytes := []byte{
		0x01, // 1 import
		byte(len(importMod)),
	}
	impBytes = append(impBytes, []byte(importMod)...)
	impBytes = append(impBytes, byte(len(importField)))
	impBytes = append(impBytes, []byte(importField)...)
	impBytes = append(impBytes, 0x00, 0x00) // kind=0 (func), idx=0

	buf = append(buf, wasm.SecImport, byte(len(impBytes)))
	buf = append(buf, impBytes...)

	if exportName != "" {
		expBytes := []byte{
			0x01,
			byte(len(exportName)),
		}
		expBytes = append(expBytes, []byte(exportName)...)
		expBytes = append(expBytes, 0x00, 0x00)

		buf = append(buf, wasm.SecExport, byte(len(expBytes)))
		buf = append(buf, expBytes...)
	}

	return buf
}

func TestWasiDetectionAndCapabilities(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})

	// 1. WASI module with sock_recv requires wasi:sockets
	wasiSockData := buildMinimalWasiWasm("wasi_snapshot_preview1", "sock_recv", "domus_handle_event")
	hdr, err := wasm.ParseWasmBinary(wasiSockData)
	if err != nil {
		t.Fatalf("ParseWasmBinary failed: %v", err)
	}
	if !hdr.IsWASI {
		t.Errorf("expected IsWASI to be true")
	}
	if len(hdr.Imports) != 1 || hdr.Imports[0].Field != "sock_recv" {
		t.Errorf("expected 1 import for sock_recv, got %+v", hdr.Imports)
	}

	// Should fail without wasi:sockets
	_, err = wasm.NewSandboxRuntime("wasi_sock_plugin", wasiSockData, []string{"events:read"}, bus)
	if err == nil {
		t.Errorf("expected error when wasi:sockets capability is missing")
	}

	// Should succeed with wasi:sockets
	sandbox, err := wasm.NewSandboxRuntime("wasi_sock_plugin", wasiSockData, []string{"events:read", "wasi:sockets"}, bus)
	if err != nil {
		t.Fatalf("NewSandboxRuntime failed with wasi:sockets granted: %v", err)
	}

	// Test Adapter lifecycle
	if sandbox.ID() != "wasi_sock_plugin" {
		t.Errorf("expected ID 'wasi_sock_plugin', got %s", sandbox.ID())
	}
	if sandbox.Type() != "wasm_plugin" {
		t.Errorf("expected Type 'wasm_plugin', got %s", sandbox.Type())
	}
	ctx := context.Background()
	if err := sandbox.HealthCheck(ctx); err != nil {
		t.Errorf("expected HealthCheck to pass, got %v", err)
	}
	if err := sandbox.Stop(ctx); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
	if err := sandbox.HealthCheck(ctx); err == nil {
		t.Errorf("expected HealthCheck to fail after Stop()")
	}
	if err := sandbox.Start(ctx); err != nil {
		t.Errorf("Start failed: %v", err)
	}
	if err := sandbox.HealthCheck(ctx); err != nil {
		t.Errorf("expected HealthCheck to pass after Start()")
	}
}
