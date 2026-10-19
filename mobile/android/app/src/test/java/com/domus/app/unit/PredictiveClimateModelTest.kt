package com.domus.app.unit

import com.domus.app.domain.model.*
import kotlinx.serialization.json.Json
import org.junit.Assert.*
import org.junit.Test

class PredictiveClimateModelTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testPredictiveZoneConfigSerialization() {
        val config = PredictiveZoneConfigDto(
            id = "zone-living",
            name = "Zona Soggiorno",
            trv_entity_key = "climate.trv_soggiorno",
            temp_sensor_key = "sensor.temp_soggiorno",
            target_temperature = 21.0,
            orientation = "south",
            emitter_type = "radiator",
            thermal_inertia_hours = 2.0,
            preheat_enabled = true,
            solar_cutoff_enabled = true,
            pressure_boost_enabled = true,
            enabled = true
        )

        val encoded = json.encodeToString(PredictiveZoneConfigDto.serializer(), config)
        assertTrue(encoded.contains("Zona Soggiorno"))
        assertTrue(encoded.contains("climate.trv_soggiorno"))

        val decoded = json.decodeFromString(PredictiveZoneConfigDto.serializer(), encoded)
        assertEquals(config.id, decoded.id)
        assertEquals(21.0, decoded.target_temperature, 0.001)
        assertEquals("south", decoded.orientation)
        assertEquals(2.0, decoded.thermal_inertia_hours, 0.001)
        assertTrue(decoded.enabled)
    }

    @Test
    fun testPredictiveEvaluationAndModes() {
        val eval = PredictiveEvaluationDto(
            id = "eval-1",
            zone_id = "zone-living",
            zone_name = "Zona Soggiorno",
            current_temp = 19.5,
            target_temp = 21.0,
            current_pressure = 998.0,
            pressure_trend_hpa_hr = -2.1,
            humidity = 52.0,
            apparent_temp = 19.0,
            outdoor_temp_forecast = 5.0,
            direct_radiation_forecast = 120.0,
            estimated_solar_gain_kw = 0.2,
            cold_front_detected = true,
            window_open = false,
            calculated_setpoint = 22.0,
            mode = "cold_front_boost",
            reason = "Calo barometrico -2.1 hPa/h: boost termico prima del fronte freddo",
            evaluated_at = "2026-10-19T14:00:00Z"
        )

        assertEquals(PredictiveMode.COLD_FRONT_BOOST, eval.parsedMode)
        assertTrue(eval.cold_front_detected)
        assertEquals(22.0, eval.calculated_setpoint, 0.001)

        val encoded = json.encodeToString(PredictiveEvaluationDto.serializer(), eval)
        val decoded = json.decodeFromString(PredictiveEvaluationDto.serializer(), encoded)
        assertEquals(PredictiveMode.COLD_FRONT_BOOST, decoded.parsedMode)
    }

    @Test
    fun testSystemPredictiveStatusDto() {
        val status = SystemPredictiveStatusDto(
            total_zones = 4,
            enabled_zones = 3,
            cold_front_active = true,
            solar_cutoff_active = false,
            zones = listOf(
                ZoneDashboardStatusDto(
                    config = PredictiveZoneConfigDto(id = "z1", name = "Cucina"),
                    latest_evaluation = PredictiveEvaluationDto(id = "e1", zone_id = "z1", mode = "comfort")
                )
            ),
            evaluated_at = "2026-10-19T14:15:00Z"
        )

        assertEquals(4, status.total_zones)
        assertEquals(3, status.enabled_zones)
        assertTrue(status.cold_front_active)
        assertFalse(status.solar_cutoff_active)
        assertEquals(1, status.zones.size)

        val encoded = json.encodeToString(SystemPredictiveStatusDto.serializer(), status)
        val decoded = json.decodeFromString(SystemPredictiveStatusDto.serializer(), encoded)
        assertEquals(status.total_zones, decoded.total_zones)
        assertEquals(status.zones[0].config.name, decoded.zones[0].config.name)
    }

    @Test
    fun testRecommendationsDto() {
        val preheat = PreheatRecommendationDto(
            zone_id = "z1",
            target_time = "18:00",
            target_temp = 21.0,
            current_temp = 19.0,
            outdoor_temp = 6.0,
            preheat_minutes = 45,
            start_time = "17:15",
            effective_heat_rate = 2.6,
            reason = "Pre-riscaldamento ottimale calcolato su inerzia di 2h"
        )
        assertEquals(45, preheat.preheat_minutes)
        assertEquals("17:15", preheat.start_time)

        val coastDown = CoastDownRecommendationDto(
            zone_id = "z1",
            departure_time = "23:00",
            comfort_limit_temp = 19.5,
            coast_down_minutes = 40,
            stop_time = "22:20",
            energy_saved_kwh = 0.65,
            reason = "Spegnimento anticipato senza perdita di comfort"
        )
        assertEquals(40, coastDown.coast_down_minutes)
        assertEquals(0.65, coastDown.energy_saved_kwh, 0.001)
    }

    @Test
    fun testEnumsFromValue() {
        assertEquals(RoomOrientation.SOUTH, RoomOrientation.fromValue("south"))
        assertEquals(RoomOrientation.NORTH, RoomOrientation.fromValue("north"))
        assertEquals(EmitterType.FLOOR_RADIANT, EmitterType.fromValue("floor_radiant"))
        assertEquals(EmitterType.RADIATOR, EmitterType.fromValue("radiator"))
        assertEquals(PredictiveMode.SOLAR_CUTOFF, PredictiveMode.fromValue("solar_cutoff"))
        assertEquals(PredictiveMode.COMFORT, PredictiveMode.fromValue("comfort"))
    }
}
