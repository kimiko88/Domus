package com.domus.app.domain.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import java.util.UUID

@Serializable
data class AutomationTrigger(
    val id: String = UUID.randomUUID().toString(),
    val platform: String = "time", // "time", "state", "sun", "geofence", "event"
    @SerialName("entity_key") val entityKey: String? = null,
    val from: String? = null,
    val to: String? = null,
    val at: String? = null, // "HH:mm:ss"
    @SerialName("event_type") val eventType: String? = null,
    @SerialName("for_seconds") val forSeconds: Int? = null
) {
    val displayTitle: String
        get() = when (platform) {
            "time" -> "Orario: ${at ?: "07:30"}"
            "sun" -> "Alba / Tramonto"
            "state" -> "${entityKey ?: "Sensore"} diventa ${to ?: "attivo"}"
            "geofence" -> "Presenza: ${eventType ?: "Uscita/Ingresso"}"
            else -> "Evento: $platform"
        }

    val displayIcon: String
        get() = when (platform) {
            "time" -> "⏰"
            "sun" -> "☀️"
            "state" -> "🚶"
            "geofence" -> "📍"
            else -> "⚡"
        }
}

@Serializable
data class AutomationCondition(
    val id: String = UUID.randomUUID().toString(),
    val condition: String = "state", // "state", "time", "numeric_state"
    @SerialName("entity_key") val entityKey: String? = null,
    val state: String? = null,
    val after: String? = null,
    val before: String? = null
) {
    val displayTitle: String
        get() = when (condition) {
            "state" -> "${entityKey ?: "Entità"} è '${state ?: "on"}'"
            "time" -> when {
                after != null && before != null -> "Tra le $after e le $before"
                after != null -> "Dopo le $after"
                before != null -> "Prima delle $before"
                else -> "Fascia oraria"
            }
            else -> "Condizione: $condition"
        }

    val displayIcon: String
        get() = when (condition) {
            "state" -> "🛡️"
            "time" -> "⏳"
            else -> "⚙️"
        }
}

@Serializable
data class AutomationAction(
    val id: String = UUID.randomUUID().toString(),
    val type: String = "service", // "service", "delay", "scene"
    val service: String? = "turn_on", // "turn_on", "turn_off", "set_temperature", "notify"
    @SerialName("target_entity_key") val targetEntityKey: String? = null,
    val params: Map<String, String>? = null
) {
    val displayTitle: String
        get() {
            val target = targetEntityKey ?: "Dispositivo"
            return when (service) {
                "turn_on" -> "Accendi / Attiva $target"
                "turn_off" -> "Spegni / Disattiva $target"
                "set_temperature" -> {
                    val t = params?.get("temperature") ?: "20"
                    "Imposta clima a $t°C su $target"
                }
                "notify" -> "Invia notifica: \"${params?.get("message") ?: "Avviso Domus"}\""
                else -> "${service ?: "Azione"} -> $target"
            }
        }

    val displayIcon: String
        get() = when (service) {
            "turn_on" -> "🔌"
            "turn_off" -> "⭕"
            "set_temperature" -> "🌡️"
            "notify" -> "🔔"
            else -> "▶️"
        }
}

@Serializable
data class AutomationRule(
    val id: String = UUID.randomUUID().toString(),
    val name: String,
    val description: String? = null,
    val enabled: Boolean = true,
    val triggers: List<AutomationTrigger> = emptyList(),
    val conditions: List<AutomationCondition> = emptyList(),
    val actions: List<AutomationAction> = emptyList(),
    @SerialName("execution_mode") val executionMode: String = "single",
    @SerialName("last_run_at") val lastRunAt: String? = null,
    @SerialName("last_status") val lastStatus: String? = null
) {
    companion object {
        val sampleRules = listOf(
            AutomationRule(
                id = "auto.goodmorning",
                name = "Buongiorno e Riscaldamento",
                description = "Alle 07:00 nei giorni feriali accende il riscaldamento e prepara il caffè",
                enabled = true,
                triggers = listOf(AutomationTrigger(platform = "time", at = "07:00:00")),
                conditions = listOf(AutomationCondition(condition = "state", entityKey = "alarm_control_panel.home", state = "disarmed")),
                actions = listOf(
                    AutomationAction(type = "service", service = "set_temperature", targetEntityKey = "climate.thermostat", params = mapOf("temperature" to "21.0")),
                    AutomationAction(type = "service", service = "turn_on", targetEntityKey = "switch.coffee_machine")
                )
            ),
            AutomationRule(
                id = "auto.night_security",
                name = "Sicurezza Notturna Giardino",
                description = "Dopo le 23:00, se il radar perimetrale rileva movimento, accende i faretti",
                enabled = true,
                triggers = listOf(AutomationTrigger(platform = "state", entityKey = "binary_sensor.garden_motion", to = "on")),
                conditions = listOf(AutomationCondition(condition = "time", after = "23:00:00", before = "06:00:00")),
                actions = listOf(
                    AutomationAction(type = "service", service = "turn_on", targetEntityKey = "light.garden_floodlight"),
                    AutomationAction(type = "service", service = "notify", targetEntityKey = "notify.mobile_app", params = mapOf("message" to "Movimento rilevato in giardino di notte!"))
                )
            ),
            AutomationRule(
                id = "auto.leave_home",
                name = "Uscita da Casa",
                description = "Quando tutti i membri escono dal perimetro di casa, spegne tutte le luci e arma l'allarme",
                enabled = true,
                triggers = listOf(AutomationTrigger(platform = "geofence", eventType = "leave_home")),
                actions = listOf(
                    AutomationAction(type = "service", service = "turn_off", targetEntityKey = "light.all_lights"),
                    AutomationAction(type = "service", service = "turn_on", targetEntityKey = "alarm_control_panel.home", params = mapOf("arm_mode" to "armed_away"))
                )
            )
        )
    }
}
