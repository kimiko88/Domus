package ocpp_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/plugins/ocpp"
)

func TestOCPP_ParseRawCallAndFormatResult(t *testing.T) {
	// Sample OCPP 1.6J BootNotification Call
	rawCall := `[2, "msg-001", "BootNotification", {"chargePointVendor": "Wallbox", "chargePointModel": "Pulsar Plus"}]`

	msgType, id, action, payload, err := ocpp.ParseRawCall([]byte(rawCall))
	if err != nil {
		t.Fatalf("ParseRawCall failed: %v", err)
	}

	if msgType != ocpp.MessageTypeCall {
		t.Errorf("expected MessageTypeCall (2), got %d", msgType)
	}
	if id != "msg-001" {
		t.Errorf("expected id 'msg-001', got %q", id)
	}
	if action != ocpp.ActionBootNotification {
		t.Errorf("expected action 'BootNotification', got %q", action)
	}

	var req ocpp.BootNotificationReq
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("failed to decode BootNotification payload: %v", err)
	}
	if req.ChargePointVendor != "Wallbox" || req.ChargePointModel != "Pulsar Plus" {
		t.Errorf("unexpected payload content: %+v", req)
	}

	// Format Response CallResult
	respPayload := ocpp.BootNotificationConf{
		Status:      "Accepted",
		CurrentTime: time.Now().UTC(),
		Interval:    60,
	}
	resBytes, err := ocpp.FormatCallResult(id, respPayload)
	if err != nil {
		t.Fatalf("FormatCallResult failed: %v", err)
	}

	var rawRes []json.RawMessage
	if err := json.Unmarshal(resBytes, &rawRes); err != nil || len(rawRes) != 3 {
		t.Fatalf("invalid CallResult format: %s", string(resBytes))
	}
}

func TestController_SolarSurplusModulationAndGridProtection(t *testing.T) {
	cfg := ocpp.ControllerConfig{
		MaxContractWatts: 3300.0, // 3.3 kW home contract
		GridVoltage:      230.0,
		MinCurrentAmps:   6.0,
		MaxCurrentAmps:   32.0,
		DefaultMode:      ocpp.ModeSolarOnly,
	}

	var sentAmps float64
	ctrl := ocpp.NewController(nil, cfg).WithProfileSender(func(ctx context.Context, cpID string, amps float64) error {
		sentAmps = amps
		return nil
	})

	ctx := context.Background()
	now := time.Now()

	// 1. Solar Only with low surplus (800W < 6A * 230V = 1380W) -> Expect 0A (paused)
	res, err := ctrl.CalculateOptimalCurrent(ctx, "cp-garage", 800.0, 500.0, now)
	if err != nil {
		t.Fatalf("CalculateOptimalCurrent failed: %v", err)
	}
	if res.TargetAmps != 0.0 {
		t.Errorf("expected 0A target in SolarOnly mode when surplus < 1.4kW, got %.1fA", res.TargetAmps)
	}
	if sentAmps != 0.0 {
		t.Errorf("expected profile sender called with 0A, got %.1fA", sentAmps)
	}

	// 2. Solar Only with 2500W surplus (2500 / 230 = ~10.8A) -> Expect 10A modulated
	res, err = ctrl.CalculateOptimalCurrent(ctx, "cp-garage", 2500.0, 400.0, now)
	if err != nil {
		t.Fatalf("CalculateOptimalCurrent failed: %v", err)
	}
	if res.TargetAmps != 10.0 {
		t.Errorf("expected 10A target for 2.5kW solar surplus, got %.1fA", res.TargetAmps)
	}

	// 3. Switch to SolarOptimized mode (guaranteed 6A base + solar surplus)
	ctrl.SetMode(ocpp.ModeSolarOptimized)
	// With 1500W surplus (~6.5A extra) -> 6A base + 6A extra = 12A
	res, err = ctrl.CalculateOptimalCurrent(ctx, "cp-garage", 1500.0, 300.0, now)
	if err != nil {
		t.Fatalf("CalculateOptimalCurrent failed: %v", err)
	}
	if res.TargetAmps != 12.0 {
		t.Errorf("expected 12A target in SolarOptimized with 1.5kW surplus, got %.1fA", res.TargetAmps)
	}

	// 4. Grid Overload Protection: Oven + Kettle turned on (House Load = 2800W out of 3300W contract)
	// Remaining grid headroom is 500W / 230V = ~2.1A -> Wallbox must be throttled down to prevent blackout!
	res, err = ctrl.CalculateOptimalCurrent(ctx, "cp-garage", 0.0, 2800.0, now)
	if err != nil {
		t.Fatalf("CalculateOptimalCurrent failed: %v", err)
	}
	if res.TargetAmps > 6.0 {
		t.Errorf("expected current to be throttled to safe minimum (6A), got %.1fA", res.TargetAmps)
	}

	// 5. Update telemetry and check GetStatus
	ctrl.UpdateWallboxTelemetry("cp-garage", ocpp.StatusCharging, 2300.0, 10.0, 75.0, 15400.0, now)
	wb, lastRes, ok := ctrl.GetStatus("cp-garage")
	if !ok {
		t.Fatalf("expected GetStatus to return true")
	}
	if wb.VehicleSoC != 75.0 {
		t.Errorf("expected vehicle SoC 75%%, got %.1f%%", wb.VehicleSoC)
	}
	if wb.Status != ocpp.StatusCharging {
		t.Errorf("expected StatusCharging, got %s", wb.Status)
	}
	if lastRes.ChargePointID != "cp-garage" {
		t.Errorf("unexpected last result: %+v", lastRes)
	}
}
