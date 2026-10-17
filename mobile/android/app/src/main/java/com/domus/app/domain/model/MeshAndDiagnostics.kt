package com.domus.app.domain.model

import androidx.compose.ui.graphics.Color
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class MeshNodeDto(
    val id: String,
    val name: String,
    @SerialName("protocol") val protocolType: String,
    val type: String = "end_device",
    val lqi: Int = 200,
    val battery: Int? = null,
    @SerialName("parent_id") val parentId: String? = null
)

@Serializable
data class MeshTopologyDto(
    @SerialName("protocol") val protocolType: String? = null,
    val nodes: List<MeshNodeDto> = emptyList()
)

@Serializable
data class SystemHealthDto(
    val status: String = "ok",
    val version: String? = "Domus Core 1.0",
    @SerialName("uptime_s") val uptimeSeconds: Long? = 0,
    val database: String? = "healthy"
)

data class MeshNodeDiagnostic(
    val id: String,
    val name: String,
    val room: String,
    val protocolType: String, // "Zigbee 3.0", "Thread / Matter", "Z-Wave Plus", "BLE Proxy"
    val lqi: Int, // 0-255
    val rssi: Int, // dBm (e.g. -58)
    val batteryLevel: Int? = null,
    val isRouter: Boolean = false
) {
    val signalQuality: String
        get() = when {
            rssi >= -60 -> "Eccellente"
            rssi >= -75 -> "Buono"
            rssi >= -85 -> "Accettabile"
            else -> "Critico"
        }

    val signalColor: Color
        get() = when {
            rssi >= -60 -> Color(0xFF10B981)
            rssi >= -75 -> Color(0xFF06B6D4)
            rssi >= -85 -> Color(0xFFF59E0B)
            else -> Color(0xFFEF4444)
        }

    companion object {
        val sampleNodes = listOf(
            MeshNodeDiagnostic(id = "zigbee.pir_corridor", name = "Sensore PIR Corridoio", room = "Corridoio", protocolType = "Zigbee 3.0", lqi = 235, rssi = -58, batteryLevel = 12, isRouter = false),
            MeshNodeDiagnostic(id = "zigbee.temp_bedroom", name = "Termo-Igrometro Letto", room = "Camera Padronale", protocolType = "Zigbee 3.0", lqi = 190, rssi = -68, batteryLevel = 14, isRouter = false),
            MeshNodeDiagnostic(id = "thread.door_sensor", name = "Sensore Finestra Studio", room = "Studio", protocolType = "Thread / Matter", lqi = 248, rssi = -52, batteryLevel = 88, isRouter = false),
            MeshNodeDiagnostic(id = "zwave.water_leak", name = "Allagamento Lavanderia", room = "Lavanderia", protocolType = "Z-Wave Plus", lqi = 180, rssi = -72, batteryLevel = 92, isRouter = false),
            MeshNodeDiagnostic(id = "ble.beacon_living", name = "BLE Proxy ESP32 Salotto", room = "Salotto", protocolType = "BLE Proxy", lqi = 255, rssi = -45, batteryLevel = null, isRouter = true),
            MeshNodeDiagnostic(id = "zigbee.bulb_kitchen", name = "Luce Cucina (Router)", room = "Cucina", protocolType = "Zigbee 3.0", lqi = 250, rssi = -48, batteryLevel = null, isRouter = true)
        )
    }
}
