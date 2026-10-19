import XCTest
@testable import DomusKit

final class MedicineAndPredictiveClimateTests: XCTestCase {

    // MARK: - Medicine Domain & Unit Tests

    func testMedicineInitializationAndDefaults() {
        let med = Medicine(
            name: "Tachipirina 500mg",
            activeIngredient: "Paracetamolo",
            dosage: "500mg",
            expirationDate: "2027-12-31",
            paoDays: 0,
            location: .armadiettoBagno,
            requiresColdChain: false,
            quantity: 20.0,
            unit: .compresse,
            minThreshold: 5.0
        )

        XCTAssertEqual(med.name, "Tachipirina 500mg")
        XCTAssertEqual(med.activeIngredient, "Paracetamolo")
        XCTAssertEqual(med.effectiveExpirationDate, "2027-12-31")
        XCTAssertFalse(med.isOpened)
        XCTAssertFalse(med.isLowStock)
        XCTAssertEqual(med.location, .armadiettoBagno)
    }

    func testMedicineExpirationAndLowStockLogic() {
        let calendar = Calendar.current
        let today = Date()

        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

        // 1. Expired medicine (-10 days)
        let pastDate = calendar.date(byAdding: .day, value: -10, to: today)!
        let expiredMed = Medicine(
            name: "Sciroppo Scaduto",
            expirationDate: formatter.string(from: pastDate),
            quantity: 2.0,
            minThreshold: 5.0
        )
        XCTAssertTrue(expiredMed.isExpired(from: today))
        XCTAssertTrue(expiredMed.isLowStock)

        // 2. Expiring soon in 5 days
        let soonDate = calendar.date(byAdding: .day, value: 5, to: today)!
        let soonMed = Medicine(
            name: "Collirio In Scadenza",
            expirationDate: formatter.string(from: soonDate),
            quantity: 10.0,
            minThreshold: 2.0
        )
        XCTAssertFalse(soonMed.isExpired(from: today))
        XCTAssertTrue(soonMed.isExpiringSoon(daysThreshold: 7, from: today))
        XCTAssertFalse(soonMed.isLowStock)

        // 3. Valid medicine in 180 days
        let futureDate = calendar.date(byAdding: .day, value: 180, to: today)!
        let validMed = Medicine(
            name: "Antibiotico Valido",
            expirationDate: formatter.string(from: futureDate),
            quantity: 12.0,
            minThreshold: 4.0
        )
        XCTAssertFalse(validMed.isExpired(from: today))
        XCTAssertFalse(validMed.isExpiringSoon(daysThreshold: 30, from: today))
    }

    func testMedicinePAOLogic() {
        let med = Medicine(
            name: "Crema Dermica",
            expirationDate: "2028-06-30",
            paoDays: 30,
            openedAt: "2026-10-01",
            effectiveExpirationDate: "2026-10-31"
        )
        XCTAssertTrue(med.isOpened)
        XCTAssertEqual(med.effectiveExpirationDate, "2026-10-31")
    }

    func testMedicineJSONDecodingAndContract() throws {
        let json = """
        {
            "id": "med-12345",
            "name": "Augmentin 875mg+125mg",
            "active_ingredient": "Amoxicillina + Acido Clavulanico",
            "dosage": "1g",
            "form": "compresse rivestite",
            "aic": "029028014",
            "barcode": "8001234567890",
            "lot_number": "LOT2026A",
            "expiration_date": "2027-05-31",
            "pao_days": 0,
            "opened_at": null,
            "effective_expiration_date": "2027-05-31",
            "location": "armadietto_bagno",
            "requires_cold_chain": false,
            "quantity": 12.0,
            "unit": "compresse",
            "min_threshold": 4.0,
            "prescription_required": true,
            "notes": "Assumere a stomaco pieno",
            "created_at": "2026-10-19T10:00:00Z",
            "updated_at": "2026-10-19T10:00:00Z"
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let decoded = try decoder.decode(Medicine.self, from: json)

        XCTAssertEqual(decoded.id, "med-12345")
        XCTAssertEqual(decoded.name, "Augmentin 875mg+125mg")
        XCTAssertEqual(decoded.aic, "029028014")
        XCTAssertEqual(decoded.unit, .compresse)
        XCTAssertEqual(decoded.location, .armadiettoBagno)
        XCTAssertTrue(decoded.prescriptionRequired)
        XCTAssertFalse(decoded.requiresColdChain)
    }

    func testMedicineAlertDecoding() throws {
        let json = """
        {
            "medicine_id": "med-999",
            "medicine_name": "Insulina Glargine",
            "type": "cold_chain_warning",
            "severity": "critical",
            "days_remaining": 60,
            "message": "Temperatura frigo anomala (9.5°C > 8°C)",
            "location": "frigo"
        }
        """.data(using: .utf8)!

        let alert = try JSONDecoder().decode(MedicineAlert.self, from: json)
        XCTAssertEqual(alert.medicineId, "med-999")
        XCTAssertEqual(alert.type, "cold_chain_warning")
        XCTAssertTrue(alert.isCritical)
        XCTAssertEqual(alert.location, .frigo)
    }

    // MARK: - Predictive Climate Domain & Unit Tests

    func testPredictiveZoneConfigInitialization() {
        let zone = PredictiveZoneConfig(
            id: "zone-living",
            name: "Soggiorno",
            trvEntityKey: "climate.trv_living",
            tempSensorKey: "sensor.temp_living",
            targetTemperature: 21.0,
            orientation: .south,
            emitterType: .radiator,
            thermalInertiaHours: 2.0,
            preheatEnabled: true,
            solarCutoffEnabled: true,
            pressureBoostEnabled: true
        )

        XCTAssertEqual(zone.id, "zone-living")
        XCTAssertEqual(zone.name, "Soggiorno")
        XCTAssertEqual(zone.orientation, .south)
        XCTAssertEqual(zone.emitterType, .radiator)
        XCTAssertTrue(zone.enabled)
    }

    func testPredictiveEvaluationModesAndDecoding() throws {
        let json = """
        {
            "id": "eval-abc",
            "zone_id": "zone-living",
            "zone_name": "Soggiorno",
            "current_temp": 19.2,
            "target_temp": 21.0,
            "current_pressure": 998.4,
            "pressure_trend_hpa_hr": -2.3,
            "humidity": 55.0,
            "apparent_temp": 18.8,
            "outdoor_temp_forecast": 4.5,
            "direct_radiation_forecast": 450.0,
            "estimated_solar_gain_kw": 0.85,
            "cold_front_detected": true,
            "window_open": false,
            "calculated_setpoint": 22.0,
            "mode": "cold_front_boost",
            "reason": "Calo barometrico -2.3 hPa/h: boost termico applicato prima del fronte freddo",
            "evaluated_at": "2026-10-19T14:30:00Z"
        }
        """.data(using: .utf8)!

        let eval = try JSONDecoder().decode(PredictiveEvaluation.self, from: json)
        XCTAssertEqual(eval.id, "eval-abc")
        XCTAssertEqual(eval.calculatedSetpoint, 22.0)
        XCTAssertTrue(eval.coldFrontDetected)
        XCTAssertEqual(eval.parsedMode, .coldFrontBoost)
        XCTAssertEqual(eval.estimatedSolarGainKw, 0.85)
    }

    func testSystemPredictiveStatusDecoding() throws {
        let json = """
        {
            "total_zones": 3,
            "enabled_zones": 3,
            "cold_front_active": true,
            "solar_cutoff_active": false,
            "zones": [
                {
                    "config": {
                        "id": "z1",
                        "name": "Camera Matrimoniale",
                        "trv_entity_key": "climate.trv_bed",
                        "temp_sensor_key": "sensor.temp_bed",
                        "target_temperature": 20.0,
                        "orientation": "north",
                        "window_area_sqm": 2.5,
                        "emitter_type": "floor_radiant",
                        "thermal_inertia_hours": 3.0,
                        "preheat_enabled": true,
                        "solar_cutoff_enabled": true,
                        "pressure_boost_enabled": true,
                        "enabled": true
                    },
                    "latest_evaluation": {
                        "id": "ev1",
                        "zone_id": "z1",
                        "zone_name": "Camera Matrimoniale",
                        "current_temp": 19.8,
                        "target_temp": 20.0,
                        "current_pressure": 1005.0,
                        "pressure_trend_hpa_hr": -1.5,
                        "humidity": 48.0,
                        "apparent_temp": 19.8,
                        "outdoor_temp_forecast": 6.0,
                        "direct_radiation_forecast": 0.0,
                        "estimated_solar_gain_kw": 0.0,
                        "cold_front_detected": true,
                        "window_open": false,
                        "calculated_setpoint": 20.5,
                        "mode": "cold_front_boost",
                        "reason": "Anticipo riscaldamento a pavimento per contrastare inerzia di 3.0h",
                        "evaluated_at": "2026-10-19T14:35:00Z"
                    }
                }
            ],
            "evaluated_at": "2026-10-19T14:35:00Z"
        }
        """.data(using: .utf8)!

        let status = try JSONDecoder().decode(SystemPredictiveStatus.self, from: json)
        XCTAssertEqual(status.totalZones, 3)
        XCTAssertEqual(status.enabledZones, 3)
        XCTAssertTrue(status.coldFrontActive)
        XCTAssertFalse(status.solarCutoffActive)
        XCTAssertEqual(status.zones.count, 1)
        XCTAssertEqual(status.zones.first?.config.name, "Camera Matrimoniale")
        XCTAssertEqual(status.zones.first?.latestEvaluation?.parsedMode, .coldFrontBoost)
    }

    func testPreheatAndCoastdownRecommendations() throws {
        let preheatJson = """
        {
            "zone_id": "z1",
            "target_time": "18:00",
            "target_temp": 21.0,
            "current_temp": 18.5,
            "outdoor_temp": 5.0,
            "preheat_minutes": 65,
            "start_time": "16:55",
            "effective_heat_rate": 2.3,
            "reason": "Inerzia termica 2.0h con delta di 2.5°C richiede 65 minuti"
        }
        """.data(using: .utf8)!

        let preheat = try JSONDecoder().decode(PreheatRecommendation.self, from: preheatJson)
        XCTAssertEqual(preheat.zoneId, "z1")
        XCTAssertEqual(preheat.preheatMinutes, 65)
        XCTAssertEqual(preheat.startTime, "16:55")

        let coastdownJson = """
        {
            "zone_id": "z1",
            "departure_time": "23:00",
            "comfort_limit_temp": 19.5,
            "coast_down_minutes": 45,
            "stop_time": "22:15",
            "energy_saved_kwh": 0.72,
            "reason": "Inerzia termica mantiene la temperatura entro 19.5°C fino a partenza"
        }
        """.data(using: .utf8)!

        let coastdown = try JSONDecoder().decode(CoastDownRecommendation.self, from: coastdownJson)
        XCTAssertEqual(coastdown.zoneId, "z1")
        XCTAssertEqual(coastdown.coastDownMinutes, 45)
        XCTAssertEqual(coastdown.energySavedKwh, 0.72)
    }

    // MARK: - Integration / End-to-End Simulation Tests

    func testMedicineAndPredictiveClimateLifecycleE2E() async throws {
        // 1. Simula ciclo farmaco: Creazione -> Lettura -> Apertura con PAO -> Assunzione dose
        var med = Medicine(
            name: "Colbiocin Collirio",
            expirationDate: "2027-01-01",
            paoDays: 28,
            location: .frigo,
            requiresColdChain: true,
            quantity: 10.0,
            unit: .ml,
            minThreshold: 2.0
        )
        XCTAssertFalse(med.isOpened)
        XCTAssertEqual(med.quantity, 10.0)

        // Simula apertura flacone (attiva PAO)
        med.openedAt = "2026-10-19"
        med.effectiveExpirationDate = "2026-11-16"
        XCTAssertTrue(med.isOpened)

        // Simula assunzione 1 dose (1ml)
        med.quantity -= 1.0
        XCTAssertEqual(med.quantity, 9.0)
        XCTAssertFalse(med.isLowStock)

        // Simula consumo totale fino a esaurimento scorte
        med.quantity = 1.0
        XCTAssertTrue(med.isLowStock)

        // 2. Simula ciclo termostato predittivo: Configurazione zona -> Ricezione valutazione meteo solare -> Cutoff attivo
        let zoneConfig = PredictiveZoneConfig(
            name: "Studio Sud",
            trvEntityKey: "climate.studio_trv",
            tempSensorKey: "sensor.studio_temp",
            targetTemperature: 21.0,
            orientation: .south,
            emitterType: .radiator,
            solarCutoffEnabled: true
        )

        // Simulazione cutoff solare con irraggiamento di picco
        let solarEval = PredictiveEvaluation(
            zoneId: zoneConfig.id,
            zoneName: zoneConfig.name,
            currentTemp: 20.8,
            targetTemp: 21.0,
            outdoorTempForecast: 15.0,
            directRadiationForecast: 800.0,
            estimatedSolarGainKw: 1.6,
            coldFrontDetected: false,
            windowOpen: false,
            calculatedSetpoint: 19.5, // Ridotto a 19.5°C per cutoff solare
            mode: "solar_cutoff",
            reason: "Irraggiamento solare 1.6 kW: setpoint ridotto per prevenire overshoot termico"
        )

        XCTAssertEqual(solarEval.parsedMode, .solarCutoff)
        XCTAssertLessThan(solarEval.calculatedSetpoint, zoneConfig.targetTemperature)
    }
}
