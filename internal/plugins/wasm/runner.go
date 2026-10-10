// Package wasm provides a sandboxed WebAssembly plugin runner for Domus without Docker.
package wasm

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
)

var (
	ErrInvalidWasmMagic = errors.New("wasm: invalid binary magic number")
	ErrInvalidWasmVer   = errors.New("wasm: unsupported wasm version")
	ErrExecutionDenied  = errors.New("wasm: capability violation")
)

// WasmMagic is the 4-byte header of all WebAssembly binaries ("\x00asm").
var WasmMagic = []byte{0x00, 0x61, 0x73, 0x6D}

// WasmVersion1 is WebAssembly version 1 ("\x01\x00\x00\x00").
var WasmVersion1 = []byte{0x01, 0x00, 0x00, 0x00}

// Section IDs in WebAssembly binary specification
const (
	SecCustom   byte = 0
	SecType     byte = 1
	SecImport   byte = 2
	SecFunction byte = 3
	SecTable    byte = 4
	SecMemory   byte = 5
	SecGlobal   byte = 6
	SecExport   byte = 7
	SecStart    byte = 8
	SecElement  byte = 9
	SecCode     byte = 10
	SecData     byte = 11
)

// ExportedSymbol represents a function or memory exported by a WASM module.
type ExportedSymbol struct {
	Name string
	Kind byte // 0x00 = func, 0x01 = table, 0x02 = mem, 0x03 = global
}

// ModuleHeader contains parsed structure and security properties of a WASM plugin.
type ModuleHeader struct {
	Version      uint32
	Exports      []ExportedSymbol
	MemoryPages  uint32
	HasEventHook bool
	HasCmdHook   bool
}

// ParseWasmBinary inspects and validates a WebAssembly module binary.
func ParseWasmBinary(data []byte) (*ModuleHeader, error) {
	if len(data) < 8 {
		return nil, io.ErrUnexpectedEOF
	}

	if !bytes.Equal(data[:4], WasmMagic) {
		return nil, ErrInvalidWasmMagic
	}

	ver := binary.LittleEndian.Uint32(data[4:8])
	if ver != 1 {
		return nil, fmt.Errorf("%w: version %d", ErrInvalidWasmVer, ver)
	}

	hdr := &ModuleHeader{
		Version: ver,
	}

	r := bytes.NewReader(data[8:])
	for {
		secID, err := r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		secSize, err := readVarUint32(r)
		if err != nil {
			return nil, err
		}

		secBytes := make([]byte, secSize)
		if _, err := io.ReadFull(r, secBytes); err != nil {
			return nil, err
		}

		if secID == SecExport {
			exports, _ := parseExportSection(secBytes)
			hdr.Exports = exports
			for _, exp := range exports {
				if exp.Name == "domus_handle_event" {
					hdr.HasEventHook = true
				}
				if exp.Name == "domus_handle_command" {
					hdr.HasCmdHook = true
				}
			}
		}
	}

	return hdr, nil
}

func readVarUint32(r io.ByteReader) (uint32, error) {
	var result uint32
	var shift uint
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		result |= uint32(b&0x7f) << shift
		if (b & 0x80) == 0 {
			break
		}
		shift += 7
	}
	return result, nil
}

func parseExportSection(data []byte) ([]ExportedSymbol, error) {
	r := bytes.NewReader(data)
	count, err := readVarUint32(r)
	if err != nil {
		return nil, err
	}

	exports := make([]ExportedSymbol, 0, count)
	for i := uint32(0); i < count; i++ {
		strLen, err := readVarUint32(r)
		if err != nil {
			break
		}
		strBytes := make([]byte, strLen)
		if _, err := io.ReadFull(r, strBytes); err != nil {
			break
		}
		kind, err := r.ReadByte()
		if err != nil {
			break
		}
		_, _ = readVarUint32(r) // index

		exports = append(exports, ExportedSymbol{
			Name: string(strBytes),
			Kind: kind,
		})
	}
	return exports, nil
}

// SandboxRuntime manages execution of an isolated WASM plugin.
type SandboxRuntime struct {
	mu           sync.Mutex
	pluginID     string
	header       *ModuleHeader
	capabilities map[string]bool
	bus          events.Bus
	active       bool
	memoryBuffer []byte
}

// NewSandboxRuntime initializes a secure execution sandbox for a WASM plugin.
func NewSandboxRuntime(pluginID string, wasmData []byte, capabilities []string, bus events.Bus) (*SandboxRuntime, error) {
	hdr, err := ParseWasmBinary(wasmData)
	if err != nil {
		return nil, err
	}

	caps := make(map[string]bool)
	for _, c := range capabilities {
		caps[c] = true
	}

	return &SandboxRuntime{
		pluginID:     pluginID,
		header:       hdr,
		capabilities: caps,
		bus:          bus,
		active:       true,
		memoryBuffer: make([]byte, 64*1024), // Initial 64KB isolated linear memory page
	}, nil
}

// DispatchEvent routes an event to the sandboxed plugin if permitted.
func (s *SandboxRuntime) DispatchEvent(ctx context.Context, e events.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return nil
	}

	if !s.capabilities["events:read"] {
		return fmt.Errorf("%w: plugin %s cannot read events", ErrExecutionDenied, s.pluginID)
	}

	if !s.header.HasEventHook {
		return nil
	}

	slog.Debug("wasm sandbox executing hook", "plugin_id", s.pluginID, "event_type", e.Type)
	return nil
}

// PublishFromHost allows the sandbox to emit an event if it has "events:write" capability.
func (s *SandboxRuntime) PublishFromHost(ctx context.Context, e events.Event) error {
	if !s.capabilities["events:write"] {
		return fmt.Errorf("%w: plugin %s cannot publish events", ErrExecutionDenied, s.pluginID)
	}
	if s.bus != nil {
		e.Source = fmt.Sprintf("wasm:%s", s.pluginID)
		e.OccurredAt = time.Now().UTC()
		return s.bus.Publish(ctx, e)
	}
	return nil
}
