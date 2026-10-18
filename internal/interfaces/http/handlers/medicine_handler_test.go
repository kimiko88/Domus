package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appMed "github.com/domus-hub/domus/internal/application/medicine"
	domainMed "github.com/domus-hub/domus/internal/domain/medicine"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMedicineHandlerTest(t *testing.T) (*chi.Mux, *appMed.Service) {
	t.Helper()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path:    ":memory:",
		WALMode: false,
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = sqlite.Migrate(ctx, db)
	require.NoError(t, err)

	repo := sqlite.NewMedicineRepository(db)
	svc := appMed.NewService(repo, nil)
	handler := handlers.NewMedicineHandler(svc)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)
	return r, svc
}

func TestMedicineHandler_Flow(t *testing.T) {
	router, _ := setupMedicineHandlerTest(t)

	// 1. Parse Datamatrix Code
	parseReqBody := map[string]string{
		"code": "(01)08003423403501(17)270630(10)LOTABC12",
	}
	body, _ := json.Marshal(parseReqBody)
	req := httptest.NewRequest(http.MethodPost, "/medicines/parse-code", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var parseRes domainMed.DatamatrixParseResult
	err := json.NewDecoder(rec.Body).Decode(&parseRes)
	require.NoError(t, err)
	assert.Equal(t, "034234035", parseRes.AIC)
	assert.Equal(t, "2027-06-30", parseRes.ExpirationDate)
	assert.Equal(t, "LOTABC12", parseRes.LotNumber)

	// 2. Lookup AIFA
	req = httptest.NewRequest(http.MethodGet, "/medicines/lookup?aic=034234035", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var aifaResults []domainMed.AIFARecord
	err = json.NewDecoder(rec.Body).Decode(&aifaResults)
	require.NoError(t, err)
	require.NotEmpty(t, aifaResults)
	assert.Equal(t, "Tachipirina 1000 mg compresse", aifaResults[0].Name)

	// 3. Create Medicine using Quick Add (AIC + Expiration Date)
	createReq := appMed.CreateMedicineRequest{
		AIC:            parseRes.AIC,
		ExpirationDate: parseRes.ExpirationDate,
		LotNumber:      parseRes.LotNumber,
		PaoDays:        0,
		Location:       domainMed.LocationBathroomCabinet,
	}
	body, _ = json.Marshal(createReq)
	req = httptest.NewRequest(http.MethodPost, "/medicines", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var created domainMed.Medicine
	err = json.NewDecoder(rec.Body).Decode(&created)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Tachipirina 1000 mg compresse", created.Name)
	assert.Equal(t, "Paracetamolo", created.ActiveIngredient)
	assert.Equal(t, 16.0, created.Quantity)

	// 4. List Medicines
	req = httptest.NewRequest(http.MethodGet, "/medicines?location=armadietto_bagno", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var list []domainMed.Medicine
	err = json.NewDecoder(rec.Body).Decode(&list)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// 5. Consume 2 tablets
	consumeBody, _ := json.Marshal(map[string]float64{"amount": 2})
	req = httptest.NewRequest(http.MethodPost, "/medicines/"+created.ID+"/consume", bytes.NewReader(consumeBody))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var afterConsume domainMed.Medicine
	err = json.NewDecoder(rec.Body).Decode(&afterConsume)
	require.NoError(t, err)
	assert.Equal(t, 14.0, afterConsume.Quantity)

	// 6. Test Open Bottle with PAO
	collirioReq := appMed.CreateMedicineRequest{
		Name:           "Tobral Collirio 5ml",
		ExpirationDate: "2027-12-31",
		PaoDays:        28,
		Quantity:       1,
		Unit:           domainMed.UnitMl,
		Location:       domainMed.LocationFridge,
	}
	body, _ = json.Marshal(collirioReq)
	req = httptest.NewRequest(http.MethodPost, "/medicines", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var collirio domainMed.Medicine
	_ = json.NewDecoder(rec.Body).Decode(&collirio)

	openBody, _ := json.Marshal(map[string]string{"opened_at": "2026-10-01"})
	req = httptest.NewRequest(http.MethodPost, "/medicines/"+collirio.ID+"/open", bytes.NewReader(openBody))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var openedCollirio domainMed.Medicine
	err = json.NewDecoder(rec.Body).Decode(&openedCollirio)
	require.NoError(t, err)
	// 2026-10-01 + 28d = 2026-10-29
	assert.Equal(t, "2026-10-29", openedCollirio.EffectiveExpirationDate)

	// 7. Get Alerts
	req = httptest.NewRequest(http.MethodGet, "/medicines/alerts?fridge_temp=9.0", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var alerts []domainMed.MedicineAlert
	err = json.NewDecoder(rec.Body).Decode(&alerts)
	require.NoError(t, err)
	// At least collirio in fridge with temp > 8.0 will have alert if requires cold chain or expiration
	assert.NotEmpty(t, alerts)

	// 8. Delete
	req = httptest.NewRequest(http.MethodDelete, "/medicines/"+created.ID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
