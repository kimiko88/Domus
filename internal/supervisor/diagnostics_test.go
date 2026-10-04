package supervisor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadSoCTemperature(t *testing.T) {
	tempDir := t.TempDir()
	thermalFile := filepath.Join(tempDir, "temp")

	collector := NewDiagnosticsCollector(DefaultConfig(), nil, nil, nil)
	collector.thermalZonePath = thermalFile

	// 1. Missing file -> defaults safely to 0.0 without crash
	temp := collector.readSoCTemp()
	if temp != 0.0 {
		t.Fatalf("expected 0.0 for missing thermal file, got %v", temp)
	}

	// 2. Valid millidegree reading (43500 -> 43.5 C)
	if err := os.WriteFile(thermalFile, []byte("43500\n"), 0644); err != nil {
		t.Fatalf("failed to write thermal file: %v", err)
	}
	temp = collector.readSoCTemp()
	if temp != 43.5 {
		t.Fatalf("expected 43.5 C, got %v", temp)
	}

	// 3. Negative or invalid string
	if err := os.WriteFile(thermalFile, []byte("invalid\n"), 0644); err != nil {
		t.Fatalf("failed to write thermal file: %v", err)
	}
	temp = collector.readSoCTemp()
	if temp != 0.0 {
		t.Fatalf("expected 0.0 for invalid thermal content, got %v", temp)
	}
}

func TestReadMemInfo(t *testing.T) {
	tempDir := t.TempDir()
	meminfoFile := filepath.Join(tempDir, "meminfo")

	collector := NewDiagnosticsCollector(DefaultConfig(), nil, nil, nil)
	collector.meminfoPath = meminfoFile

	sampleMeminfo := `MemTotal:        3884032 kB
MemFree:         1204000 kB
MemAvailable:    2500000 kB
Buffers:          150000 kB
Cached:          1200000 kB
SwapTotal:             0 kB
SwapFree:              0 kB
`
	if err := os.WriteFile(meminfoFile, []byte(sampleMeminfo), 0644); err != nil {
		t.Fatalf("failed to write meminfo: %v", err)
	}

	mem := collector.readMemInfo()
	expectedTotal := uint64(3884032 * 1024)
	expectedAvailable := uint64(2500000 * 1024)
	if mem.TotalBytes != expectedTotal {
		t.Fatalf("expected TotalBytes=%d, got %d", expectedTotal, mem.TotalBytes)
	}
	if mem.AvailableBytes != expectedAvailable {
		t.Fatalf("expected AvailableBytes=%d, got %d", expectedAvailable, mem.AvailableBytes)
	}
	if mem.UsedBytes != (expectedTotal - expectedAvailable) {
		t.Fatalf("unexpected used bytes: %d", mem.UsedBytes)
	}
	if mem.UsedPercent <= 0 || mem.UsedPercent >= 100 {
		t.Fatalf("expected realistic used percent between 0 and 100, got %f", mem.UsedPercent)
	}
}

func TestReadCPULoad(t *testing.T) {
	tempDir := t.TempDir()
	loadavgFile := filepath.Join(tempDir, "loadavg")

	collector := NewDiagnosticsCollector(DefaultConfig(), nil, nil, nil)
	collector.loadavgPath = loadavgFile

	if err := os.WriteFile(loadavgFile, []byte("0.45 0.32 0.18 2/184 10294\n"), 0644); err != nil {
		t.Fatalf("failed to write loadavg: %v", err)
	}

	load1, load5, load15 := collector.readCPULoad()
	if load1 != 0.45 || load5 != 0.32 || load15 != 0.18 {
		t.Fatalf("unexpected load values: %v, %v, %v", load1, load5, load15)
	}
}

func TestCollectDiagnostics(t *testing.T) {
	tempDir := t.TempDir()
	collector := NewDiagnosticsCollector(DefaultConfig(), nil, nil, nil)
	collector.dataDirPath = tempDir

	ctx := context.Background()
	diag, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("failed to collect diagnostics: %v", err)
	}

	if diag.Hostname == "" {
		t.Fatal("expected non-empty hostname")
	}
	if diag.Architecture == "" {
		t.Fatal("expected non-empty architecture")
	}
	if diag.CPUCores <= 0 {
		t.Fatalf("expected at least 1 CPU core, got %d", diag.CPUCores)
	}
	if diag.Timestamp.IsZero() {
		t.Fatal("expected non-zero timestamp")
	}
}

func TestSanitizedDiagnosticsExport(t *testing.T) {
	collector := NewDiagnosticsCollector(DefaultConfig(), nil, nil, nil)

	ctx := context.Background()
	exportData, err := collector.Export(ctx)
	if err != nil {
		t.Fatalf("failed to export diagnostics: %v", err)
	}

	// Marshal to JSON and inspect sanitization
	bytes, err := json.Marshal(exportData)
	if err != nil {
		t.Fatalf("failed to marshal export to JSON: %v", err)
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(bytes, &rawMap); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if _, ok := rawMap["diagnostics"]; !ok {
		t.Fatal("export missing 'diagnostics' root key")
	}
	if _, ok := rawMap["exported_at"]; !ok {
		t.Fatal("export missing 'exported_at' root key")
	}

	// Verify sanitization helper
	inputEnv := map[string]string{
		"DOMUS_PORT":         "8080",
		"DOMUS_SECRET_KEY":   "my-super-secret-password-12345",
		"DATABASE_PASSWORD":  "sqlite_secret",
		"MQTT_AUTH_TOKEN":    "token-abc-xyz",
		"PUBLIC_DEVICE_NAME": "Living Room Light",
	}
	sanitized := sanitizeEnv(inputEnv)

	if sanitized["DOMUS_PORT"] != "8080" {
		t.Errorf("expected DOMUS_PORT preserved, got %v", sanitized["DOMUS_PORT"])
	}
	if sanitized["PUBLIC_DEVICE_NAME"] != "Living Room Light" {
		t.Errorf("expected PUBLIC_DEVICE_NAME preserved, got %v", sanitized["PUBLIC_DEVICE_NAME"])
	}
	if sanitized["DOMUS_SECRET_KEY"] != "[REDACTED]" {
		t.Errorf("expected DOMUS_SECRET_KEY to be redacted, got %v", sanitized["DOMUS_SECRET_KEY"])
	}
	if sanitized["DATABASE_PASSWORD"] != "[REDACTED]" {
		t.Errorf("expected DATABASE_PASSWORD to be redacted, got %v", sanitized["DATABASE_PASSWORD"])
	}
	if sanitized["MQTT_AUTH_TOKEN"] != "[REDACTED]" {
		t.Errorf("expected MQTT_AUTH_TOKEN to be redacted, got %v", sanitized["MQTT_AUTH_TOKEN"])
	}
}
