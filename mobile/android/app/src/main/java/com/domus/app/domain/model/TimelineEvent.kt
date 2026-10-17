package com.domus.app.domain.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import java.text.SimpleDateFormat
import java.util.*

enum class TimelineCategory(val title: String, val icon: String) {
    ALL("Tutti", "📋"),
    SECURITY("Sicurezza", "🛡️"),
    ACCESS("Accessi", "🚪"),
    CLIMATE_AND_LIGHTS("Luci & Clima", "💡"),
    ENERGY("Energia", "⚡")
}

@Serializable
data class TimelineEvent(
    val id: String = UUID.randomUUID().toString(),
    @SerialName("entity_id") val entityId: String,
    @SerialName("entity_name") val entityName: String,
    val domain: String,
    val state: String,
    val message: String,
    @SerialName("context_type") val contextType: String = "user", // "user", "automation", "system"
    @SerialName("context_name") val contextName: String? = null,
    val timestamp: Long = System.currentTimeMillis()
) {
    val category: TimelineCategory
        get() = when (domain) {
            "alarm_control_panel", "camera", "binary_sensor" -> TimelineCategory.SECURITY
            "lock", "cover" -> TimelineCategory.ACCESS
            "light", "climate" -> TimelineCategory.CLIMATE_AND_LIGHTS
            "sensor" -> if (entityId.contains("power") || entityId.contains("energy") || entityId.contains("potenza")) {
                TimelineCategory.ENERGY
            } else {
                TimelineCategory.SECURITY
            }
            else -> TimelineCategory.SECURITY
        }

    val formattedTime: String
        get() {
            val sdf = SimpleDateFormat("HH:mm", Locale.getDefault())
            return sdf.format(Date(timestamp))
        }

    companion object {
        val sampleTimeline = listOf(
            TimelineEvent(
                entityId = "lock.garage_door",
                entityName = "Porta Garage",
                domain = "lock",
                state = "unlocked",
                message = "Porta garage aperta",
                contextType = "user",
                contextName = "Marco",
                timestamp = System.currentTimeMillis() - 22 * 60 * 1000
            ),
            TimelineEvent(
                entityId = "binary_sensor.garden_motion",
                entityName = "Sensore Giardino",
                domain = "binary_sensor",
                state = "on",
                message = "Rilevato movimento in giardino",
                contextType = "system",
                contextName = "Radar PIR",
                timestamp = System.currentTimeMillis() - 64 * 60 * 1000
            ),
            TimelineEvent(
                entityId = "sensor.home_power",
                entityName = "Consumo Generale",
                domain = "sensor",
                state = "3820 W",
                message = "Superata soglia potenza contrattuale: consumati 3.8 kW",
                contextType = "system",
                contextName = "Energy Monitor",
                timestamp = System.currentTimeMillis() - 2 * 60 * 60 * 1000
            ),
            TimelineEvent(
                entityId = "scene.dinner",
                entityName = "Scenario Cena",
                domain = "scene",
                state = "active",
                message = "Attivato scenario Cena e luci calde",
                contextType = "automation",
                contextName = "Regola Oraria",
                timestamp = System.currentTimeMillis() - 3 * 60 * 60 * 1000
            ),
            TimelineEvent(
                entityId = "alarm_control_panel.home",
                entityName = "Allarme Domus",
                domain = "alarm_control_panel",
                state = "armed_away",
                message = "Allarme perimetrale inserito",
                contextType = "user",
                contextName = "App Android",
                timestamp = System.currentTimeMillis() - 5 * 60 * 60 * 1000
            )
        )
    }
}
