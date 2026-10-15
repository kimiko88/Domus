package energy_test

import (
	"testing"

	domenergy "github.com/domus-hub/domus/internal/domain/energy"
)

func TestUtilityMeterValidation(t *testing.T) {
	validElectricity := domenergy.UtilityMeter{
		Name:           "Main Electric Meter",
		UtilityType:    "electricity",
		Unit:           "kWh",
		CostPerUnit:    0.28,
		SourceEntityID: "sensor.main_energy_kwh",
	}
	if err := validElectricity.Validate(); err != nil {
		t.Errorf("expected valid electricity meter, got: %v", err)
	}

	validGas := domenergy.UtilityMeter{
		Name:           "Home Gas Counter",
		UtilityType:    "gas",
		Unit:           "m³",
		CostPerUnit:    1.15,
		SourceEntityID: "sensor.gas_consumption",
	}
	if err := validGas.Validate(); err != nil {
		t.Errorf("expected valid gas meter, got: %v", err)
	}

	validWater := domenergy.UtilityMeter{
		Name:           "Water Flow Meter",
		UtilityType:    "water",
		Unit:           "L",
		CostPerUnit:    0.002,
		SourceEntityID: "sensor.water_total_liters",
	}
	if err := validWater.Validate(); err != nil {
		t.Errorf("expected valid water meter, got: %v", err)
	}

	invalidType := domenergy.UtilityMeter{
		Name:        "Unknown Utility",
		UtilityType: "hydrogen",
	}
	if err := invalidType.Validate(); err == nil {
		t.Errorf("expected validation error for unknown utility type")
	}
}
