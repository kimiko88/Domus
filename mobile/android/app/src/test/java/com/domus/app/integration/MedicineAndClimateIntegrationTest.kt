package com.domus.app.integration

import com.domus.app.domain.model.*
import kotlinx.serialization.json.Json
import org.junit.Assert.*
import org.junit.Test

class MedicineAndClimateIntegrationTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testMedicineLifecycleAndExpirationSimulation() {
        // 1. Inserimento farmaco con PAO
        var medicine = MedicineDto(
            id = "med-sim-1",
            name = "Gocce Oculari Decongestionanti",
            active_ingredient = "Tetrizolina",
            expiration_date = "2027-06-30",
            pao_days = 30,
            opened_at = null,
            effective_expiration_date = "2027-06-30",
            location = "armadietto_bagno",
            quantity = 15.0,
            unit = "ml",
            min_threshold = 3.0
        )
        assertFalse("Il farmaco appena acquistato non deve risultare aperto", medicine.isOpened)
        assertFalse("Il farmaco con 15ml non deve risultare a scorta bassa", medicine.isLowStock)

        // 2. Apertura flacone: si attiva il PAO di 30 giorni
        medicine = medicine.copy(
            opened_at = "2026-10-19",
            effective_expiration_date = "2026-11-18"
        )
        assertTrue("Dopo l'apertura il farmaco deve risultare aperto", medicine.isOpened)
        assertEquals("La data effettiva deve essere ridotta alla scadenza del PAO", "2026-11-18", medicine.effective_expiration_date)

        // 3. Consumo dosi fino a soglia minima
        medicine = medicine.copy(quantity = medicine.quantity - 13.0) // Rimangono 2ml (sotto soglia 3.0)
        assertTrue("Con quantità 2ml e soglia 3ml deve risultare a scorta bassa", medicine.isLowStock)

        // 4. Generazione alert corrispondente
        val alert = MedicineAlertDto(
            medicine_id = medicine.id,
            medicine_name = medicine.name,
            type = "low_stock",
            severity = "warning",
            days_remaining = 30,
            message = "Scorta bassa per ${medicine.name}: rimasti ${medicine.quantity.toInt()} ${medicine.unit}",
            location = medicine.location
        )
        assertFalse("L'alert di scorta bassa è di severità warning, non critical", alert.isCritical)
        assertEquals("low_stock", alert.type)
    }

    @Test
    fun testPredictiveClimateEvaluationLifecycle() {
        // 1. Configurazione zona con esposizione Sud e riscaldamento a pavimento
        val zone = PredictiveZoneConfigDto(
            id = "zone-living-floor",
            name = "Zona Soggiorno Pavimento",
            trv_entity_key = "climate.radiant_living",
            temp_sensor_key = "sensor.living_ambient_temp",
            target_temperature = 20.5,
            orientation = "south",
            emitter_type = "floor_radiant",
            thermal_inertia_hours = 3.5,
            preheat_enabled = true,
            solar_cutoff_enabled = true
        )
        assertEquals(3.5, zone.thermal_inertia_hours, 0.001)

        // 2. Simulazione rilevamento fronte freddo barometrico
        val coldFrontEval = PredictiveEvaluationDto(
            id = "eval-cf",
            zone_id = zone.id,
            zone_name = zone.name,
            current_temp = 19.8,
            target_temp = 20.5,
            current_pressure = 997.5,
            pressure_trend_hpa_hr = -2.8,
            cold_front_detected = true,
            calculated_setpoint = 21.5, // +1°C boost preventivo
            mode = "cold_front_boost",
            reason = "Rilevato calo barometrico -2.8 hPa/h. Boost applicato per contrastare inerzia di 3.5h",
            evaluated_at = "2026-10-19T06:00:00Z"
        )
        assertEquals(PredictiveMode.COLD_FRONT_BOOST, coldFrontEval.parsedMode)
        assertTrue(coldFrontEval.cold_front_detected)
        assertTrue(coldFrontEval.calculated_setpoint > zone.target_temperature)

        // 3. Simulazione irraggiamento solare pomeridiano (Cutoff Solare)
        val solarEval = PredictiveEvaluationDto(
            id = "eval-solar",
            zone_id = zone.id,
            zone_name = zone.name,
            current_temp = 20.4,
            target_temp = 20.5,
            current_pressure = 1008.0,
            pressure_trend_hpa_hr = 0.1,
            estimated_solar_gain_kw = 1.45,
            cold_front_detected = false,
            calculated_setpoint = 19.0, // Setpoint abbassato per evitare sovratemperatura da radiazione solare
            mode = "solar_cutoff",
            reason = "Guadagno solare stimato +1.45 kW su facciata Sud: setpoint ridotto a 19.0°C",
            evaluated_at = "2026-10-19T13:30:00Z"
        )
        assertEquals(PredictiveMode.SOLAR_CUTOFF, solarEval.parsedMode)
        assertTrue(solarEval.calculated_setpoint < zone.target_temperature)

        // 4. Simulazione stato dashboard aggregato
        val systemStatus = SystemPredictiveStatusDto(
            total_zones = 1,
            enabled_zones = 1,
            cold_front_active = false,
            solar_cutoff_active = true,
            zones = listOf(
                ZoneDashboardStatusDto(config = zone, latest_evaluation = solarEval)
            ),
            evaluated_at = "2026-10-19T13:30:00Z"
        )
        assertTrue(systemStatus.solar_cutoff_active)
        assertFalse(systemStatus.cold_front_active)
        assertEquals(PredictiveMode.SOLAR_CUTOFF, systemStatus.zones.first().latest_evaluation?.parsedMode)
    }
}
