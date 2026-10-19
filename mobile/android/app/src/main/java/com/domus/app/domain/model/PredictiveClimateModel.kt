package com.domus.app.domain.model

import kotlinx.serialization.Serializable

@Serializable
enum class RoomOrientation(val value: String) {
    NORTH("north"),
    SOUTH("south"),
    EAST("east"),
    WEST("west");

    companion object {
        fun fromValue(v: String?): RoomOrientation {
            return entries.firstOrNull { it.value.equals(v, ignoreCase = true) } ?: SOUTH
        }
    }
}

@Serializable
enum class EmitterType(val value: String) {
    RADIATOR("radiator"),
    FLOOR_RADIANT("floor_radiant"),
    FANCOIL("fancoil");

    companion object {
        fun fromValue(v: String?): EmitterType {
            return entries.firstOrNull { it.value.equals(v, ignoreCase = true) } ?: RADIATOR
        }
    }
}

@Serializable
enum class PredictiveMode(val value: String) {
    COMFORT("comfort"),
    COLD_FRONT_BOOST("cold_front_boost"),
    SOLAR_CUTOFF("solar_cutoff"),
    PREHEAT("preheat"),
    ECO("eco"),
    FROST_PROTECTION("frost_protection");

    companion object {
        fun fromValue(v: String?): PredictiveMode {
            return entries.firstOrNull { it.value.equals(v, ignoreCase = true) } ?: COMFORT
        }
    }
}

@Serializable
data class PredictiveZoneConfigDto(
    val id: String = "",
    val name: String = "",
    val area_id: String? = null,
    val trv_entity_key: String = "",
    val temp_sensor_key: String = "",
    val pressure_sensor_key: String? = null,
    val humidity_sensor_key: String? = null,
    val window_sensor_key: String? = null,
    val target_temperature: Double = 20.0,
    val orientation: String = "south",
    val window_area_sqm: Double = 2.0,
    val emitter_type: String = "radiator",
    val thermal_inertia_hours: Double = 1.5,
    val preheat_enabled: Boolean = true,
    val solar_cutoff_enabled: Boolean = true,
    val pressure_boost_enabled: Boolean = true,
    val enabled: Boolean = true,
    val created_at: String? = null,
    val updated_at: String? = null
)

@Serializable
data class CreatePredictiveZoneRequestDto(
    val name: String,
    val area_id: String? = null,
    val trv_entity_key: String,
    val temp_sensor_key: String,
    val pressure_sensor_key: String? = null,
    val humidity_sensor_key: String? = null,
    val window_sensor_key: String? = null,
    val target_temperature: Double = 20.0,
    val orientation: String = "south",
    val window_area_sqm: Double = 2.0,
    val emitter_type: String = "radiator",
    val thermal_inertia_hours: Double = 1.5,
    val preheat_enabled: Boolean = true,
    val solar_cutoff_enabled: Boolean = true,
    val pressure_boost_enabled: Boolean = true,
    val enabled: Boolean = true
)

@Serializable
data class PredictiveEvaluationDto(
    val id: String = "",
    val zone_id: String = "",
    val zone_name: String = "",
    val current_temp: Double = 20.0,
    val target_temp: Double = 20.0,
    val current_pressure: Double = 1013.25,
    val pressure_trend_hpa_hr: Double = 0.0,
    val humidity: Double = 50.0,
    val apparent_temp: Double = 20.0,
    val outdoor_temp_forecast: Double = 10.0,
    val direct_radiation_forecast: Double = 0.0,
    val estimated_solar_gain_kw: Double = 0.0,
    val cold_front_detected: Boolean = false,
    val window_open: Boolean = false,
    val calculated_setpoint: Double = 20.0,
    val mode: String = "comfort",
    val reason: String = "",
    val evaluated_at: String = ""
) {
    val parsedMode: PredictiveMode
        get() = PredictiveMode.fromValue(mode)
}

@Serializable
data class ZoneDashboardStatusDto(
    val config: PredictiveZoneConfigDto,
    val latest_evaluation: PredictiveEvaluationDto? = null
)

@Serializable
data class SystemPredictiveStatusDto(
    val total_zones: Int = 0,
    val enabled_zones: Int = 0,
    val cold_front_active: Boolean = false,
    val solar_cutoff_active: Boolean = false,
    val zones: List<ZoneDashboardStatusDto> = emptyList(),
    val evaluated_at: String = ""
)

@Serializable
data class PredictiveZonesResponseDto(
    val zones: List<PredictiveZoneConfigDto> = emptyList(),
    val total: Int = 0
)

@Serializable
data class ThermalProfileDto(
    val zone_id: String = "",
    val heat_rate_per_hr: Double = 0.0,
    val cool_rate_per_hr: Double = 0.0,
    val outdoor_loss_coeff: Double = 0.0,
    val solar_boost_coeff: Double = 0.0,
    val samples_count: Int = 0,
    val last_trained_at: String = ""
)

@Serializable
data class PreheatRecommendationDto(
    val zone_id: String = "",
    val target_time: String = "",
    val target_temp: Double = 20.0,
    val current_temp: Double = 20.0,
    val outdoor_temp: Double = 10.0,
    val preheat_minutes: Int = 0,
    val start_time: String = "",
    val effective_heat_rate: Double = 0.0,
    val reason: String = ""
)

@Serializable
data class CoastDownRecommendationDto(
    val zone_id: String = "",
    val departure_time: String = "",
    val comfort_limit_temp: Double = 19.5,
    val coast_down_minutes: Int = 0,
    val stop_time: String = "",
    val energy_saved_kwh: Double = 0.0,
    val reason: String = ""
)
