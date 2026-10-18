package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/climate"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
)

func TestPredictiveClimateRepository_CRUDAndEvaluations(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path:    ":memory:",
		WALMode: false,
	})
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer db.Close()

	if err := sqlite.Migrate(ctx, db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	repo := sqlite.NewPredictiveClimateRepository(db)

	// 1. Create Zone
	zone := &climate.PredictiveZoneConfig{
		ID:                   "pzone_test_1",
		Name:                 "Salone Principale",
		AreaID:               "area_living",
		TRVEntityKey:         "climate.valvola_salone",
		TempSensorKey:        "sensor.salone_temperatura",
		PressureSensorKey:    "sensor.salone_pressione",
		HumiditySensorKey:    "sensor.salone_umidita",
		WindowSensorKey:      "binary_sensor.finestra_salone",
		TargetTemperature:    21.5,
		Orientation:          climate.OrientationSouth,
		WindowAreaSqm:        6.0,
		EmitterType:          climate.EmitterFloorRadiant,
		ThermalInertiaHours:  4.0,
		PreheatEnabled:       true,
		SolarCutoffEnabled:   true,
		PressureBoostEnabled: true,
		Enabled:              true,
	}

	if err := repo.Save(ctx, zone); err != nil {
		t.Fatalf("failed to save zone: %v", err)
	}

	// 2. GetByID
	fetched, err := repo.GetByID(ctx, "pzone_test_1")
	if err != nil {
		t.Fatalf("failed to get zone: %v", err)
	}
	if fetched.Name != "Salone Principale" || fetched.TargetTemperature != 21.5 {
		t.Errorf("unexpected zone data: %+v", fetched)
	}
	if fetched.Orientation != climate.OrientationSouth || fetched.EmitterType != climate.EmitterFloorRadiant {
		t.Errorf("unexpected orientation or emitter: %+v", fetched)
	}

	// 3. List
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("failed to list zones: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 zone, got %d", len(list))
	}

	// 4. Save and Retrieve Evaluation
	eval := &climate.PredictiveEvaluation{
		ID:                      "peval_test_1",
		ZoneID:                  "pzone_test_1",
		ZoneName:                "Salone Principale",
		CurrentTemp:             20.0,
		TargetTemp:              21.5,
		CurrentPressure:         1008.5,
		PressureTrendHpaHr:      -1.8,
		Humidity:                55.0,
		ApparentTemp:            20.2,
		OutdoorTempForecast:     8.5,
		DirectRadiationForecast: 450.0,
		EstimatedSolarGainKw:    1.2,
		ColdFrontDetected:       true,
		WindowOpen:              false,
		CalculatedSetpoint:      22.5,
		Mode:                    "cold_front_boost",
		Reason:                  "Anticipo termico fronte freddo",
		EvaluatedAt:             time.Now().UTC(),
	}

	if err := repo.SaveEvaluation(ctx, eval); err != nil {
		t.Fatalf("failed to save evaluation: %v", err)
	}

	latest, err := repo.GetLatestEvaluations(ctx)
	if err != nil {
		t.Fatalf("failed to get latest evaluations: %v", err)
	}
	if len(latest) != 1 {
		t.Errorf("expected 1 latest evaluation, got %d", len(latest))
	} else {
		if latest[0].ZoneID != "pzone_test_1" || latest[0].Mode != "cold_front_boost" {
			t.Errorf("unexpected latest eval: %+v", latest[0])
		}
	}

	history, err := repo.GetZoneEvaluations(ctx, "pzone_test_1", 10)
	if err != nil {
		t.Fatalf("failed to get zone evaluations: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 history item, got %d", len(history))
	}

	// 5. Save and Get Thermal Profile
	profile := &climate.ThermalProfile{
		ZoneID:           "pzone_test_1",
		HeatRatePerHr:    1.45,
		CoolRatePerHr:    0.28,
		OutdoorLossCoeff: 0.025,
		SolarBoostCoeff:  0.35,
		SamplesCount:     5,
		LastTrainedAt:    time.Now().UTC(),
	}

	if err := repo.SaveProfile(ctx, profile); err != nil {
		t.Fatalf("failed to save thermal profile: %v", err)
	}

	gotProfile, err := repo.GetProfile(ctx, "pzone_test_1")
	if err != nil {
		t.Fatalf("failed to get thermal profile: %v", err)
	}
	if gotProfile.ZoneID != "pzone_test_1" || gotProfile.HeatRatePerHr != 1.45 || gotProfile.SamplesCount != 5 {
		t.Errorf("unexpected profile: %+v", gotProfile)
	}

	// Update Profile (Upsert)
	gotProfile.HeatRatePerHr = 1.60
	gotProfile.SamplesCount = 6
	if err := repo.SaveProfile(ctx, gotProfile); err != nil {
		t.Fatalf("failed to update thermal profile: %v", err)
	}

	updatedProfile, err := repo.GetProfile(ctx, "pzone_test_1")
	if err != nil {
		t.Fatalf("failed to get updated profile: %v", err)
	}
	if updatedProfile.HeatRatePerHr != 1.60 || updatedProfile.SamplesCount != 6 {
		t.Errorf("expected updated rate 1.60, got %f", updatedProfile.HeatRatePerHr)
	}

	// List Profiles
	profiles, err := repo.ListProfiles(ctx)
	if err != nil {
		t.Fatalf("failed to list profiles: %v", err)
	}
	if len(profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(profiles))
	}

	// 6. Delete Zone (and verify cascade delete of profile)
	if err := repo.Delete(ctx, "pzone_test_1"); err != nil {
		t.Fatalf("failed to delete zone: %v", err)
	}

	_, err = repo.GetByID(ctx, "pzone_test_1")
	if err != climate.ErrPredictiveZoneNotFound {
		t.Errorf("expected ErrPredictiveZoneNotFound, got %v", err)
	}

	_, err = repo.GetProfile(ctx, "pzone_test_1")
	if err != climate.ErrThermalProfileNotFound {
		t.Errorf("expected ErrThermalProfileNotFound after zone delete, got %v", err)
	}
}
