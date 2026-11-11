// Package e2e_test provides end-to-end tests for the Domus Smart Home platform.
package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	appMed "github.com/domus-hub/domus/internal/application/medicine"
	domainMed "github.com/domus-hub/domus/internal/domain/medicine"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMedicineCabinet_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "medicine_e2e.db")

	cfg := config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 0,
		},
		Database: config.DatabaseConfig{
			Path:          dbPath,
			WALMode:       true,
			BusyTimeoutMs: 5000,
			MaxOpenConns:  1,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	db, err := sqlite.Open(cfg.Database)
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = sqlite.Migrate(ctx, db)
	require.NoError(t, err)

	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 256,
		Workers:    2,
	})
	defer bus.Close()

	repos := sqlite.NewRepositories(db)
	authOption, sessionID := configureE2EAuth(t, repos)
	srv := serverpkg.NewServer(cfg.Server, "1.0.0-test", time.Now(), db,
		serverpkg.WithRepositories(repos, bus),
		authOption,
	)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := newAuthenticatedE2EClient(ts, sessionID)

	// STEP 1: Parse GS1 Datamatrix from box camera scan
	codePayload := map[string]string{
		"code": "(01)08003423403501(17)280430(10)LOT789",
	}
	bodyBytes, _ := json.Marshal(codePayload)
	resp, err := client.Post(ts.URL+"/api/v1/medicines/parse-code", "application/json", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parseRes domainMed.DatamatrixParseResult
	err = json.NewDecoder(resp.Body).Decode(&parseRes)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "034234035", parseRes.AIC)
	assert.Equal(t, "2028-04-30", parseRes.ExpirationDate)
	assert.Equal(t, "LOT789", parseRes.LotNumber)

	// STEP 2: Quick-create medicine with AIFA auto-enrichment
	createReq := appMed.CreateMedicineRequest{
		AIC:            parseRes.AIC,
		ExpirationDate: parseRes.ExpirationDate,
		LotNumber:      parseRes.LotNumber,
		Location:       domainMed.LocationBathroomCabinet,
	}
	createBody, _ := json.Marshal(createReq)
	resp, err = client.Post(ts.URL+"/api/v1/medicines", "application/json", bytes.NewReader(createBody))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var tachi domainMed.Medicine
	err = json.NewDecoder(resp.Body).Decode(&tachi)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "Tachipirina 1000 mg compresse", tachi.Name)
	assert.Equal(t, "Paracetamolo", tachi.ActiveIngredient)
	assert.Equal(t, 16.0, tachi.Quantity)

	// STEP 3: Register cold-chain eye drops (Tobral) with PAO of 28 days
	dropsReq := appMed.CreateMedicineRequest{
		Name:              "Tobral Collirio 5ml",
		ActiveIngredient:  "Tobramicina",
		Dosage:            "0.3%",
		ExpirationDate:    "2028-10-31",
		PaoDays:           28,
		Location:          domainMed.LocationFridge,
		RequiresColdChain: true,
		Quantity:          1,
		Unit:              domainMed.UnitMl,
	}
	dropsBody, _ := json.Marshal(dropsReq)
	resp, err = client.Post(ts.URL+"/api/v1/medicines", "application/json", bytes.NewReader(dropsBody))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var tobral domainMed.Medicine
	err = json.NewDecoder(resp.Body).Decode(&tobral)
	resp.Body.Close()
	require.NoError(t, err)
	assert.True(t, tobral.RequiresColdChain)
	assert.Equal(t, "2028-10-31", tobral.EffectiveExpirationDate)

	// STEP 4: Open Tobral bottle today -> PAO updates effective expiration date to today + 28 days
	openPayload := map[string]string{
		"opened_at": "2026-10-18",
	}
	openBody, _ := json.Marshal(openPayload)
	resp, err = client.Post(ts.URL+"/api/v1/medicines/"+tobral.ID+"/open", "application/json", bytes.NewReader(openBody))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var openedTobral domainMed.Medicine
	err = json.NewDecoder(resp.Body).Decode(&openedTobral)
	resp.Body.Close()
	require.NoError(t, err)
	// 2026-10-18 + 28 days = 2026-11-15
	assert.Equal(t, "2026-11-15", openedTobral.EffectiveExpirationDate)

	// STEP 5: Consume 1 dose of Tachipirina
	consumePayload := map[string]float64{"amount": 1}
	consumeBody, _ := json.Marshal(consumePayload)
	resp, err = client.Post(ts.URL+"/api/v1/medicines/"+tachi.ID+"/consume", "application/json", bytes.NewReader(consumeBody))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var afterDose domainMed.Medicine
	err = json.NewDecoder(resp.Body).Decode(&afterDose)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, 15.0, afterDose.Quantity)

	// STEP 6: Check alerts with fridge temperature anomaly (9.2°C > 8.0°C)
	resp, err = client.Get(ts.URL + "/api/v1/medicines/alerts?fridge_temp=9.2")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var alerts []domainMed.MedicineAlert
	err = json.NewDecoder(resp.Body).Decode(&alerts)
	resp.Body.Close()
	require.NoError(t, err)

	hasColdChainAlert := false
	for _, a := range alerts {
		if a.Type == domainMed.AlertColdChainBreached {
			hasColdChainAlert = true
		}
	}
	assert.True(t, hasColdChainAlert, "expected cold chain breach alert when fridge temp exceeds 8C")
}
