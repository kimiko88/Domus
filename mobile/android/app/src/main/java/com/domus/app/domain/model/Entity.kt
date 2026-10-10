package com.domus.app.domain.model

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

@Serializable
data class Entity(
    val id: String,
    val name: String,
    val domain: String,
    val state: String,
    val attributes: Map<String, JsonElement> = emptyMap(),
    val areaId: String? = null,
    val lastChanged: String = "",
    val lastUpdated: String = ""
) {
    val isLight: Boolean get() = domain == "light"
    val isSwitch: Boolean get() = domain == "switch"
    val isSensor: Boolean get() = domain == "sensor"
    val isClimate: Boolean get() = domain == "climate"
    val isCamera: Boolean get() = domain == "camera"
    val isAlarm: Boolean get() = domain == "alarm_control_panel"

    val isOn: Boolean
        get() = state.lowercase() in listOf("on", "open", "armed_home", "armed_away")

    val brightness: Int?
        get() = attributes["brightness"]?.toString()?.toIntOrNull()

    val currentTemperature: Double?
        get() = attributes["current_temperature"]?.toString()?.toDoubleOrNull()
}
