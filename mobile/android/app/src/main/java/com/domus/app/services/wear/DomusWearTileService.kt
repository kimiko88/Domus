package com.domus.app.services.wear

import android.app.Service
import android.content.Intent
import android.os.IBinder
import com.domus.app.data.network.DomusApiClient
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * Rapid Scene definition for Wear OS Tiles.
 */
data class WearQuickScene(
    val id: String,
    val name: String,
    val iconEmoji: String,
    val entityId: String,
    val actionService: String = "turn_on"
)

/**
 * Watchface complication data model for Wear OS.
 */
data class WearComplicationData(
    val indoorTemperature: Double = 21.5,
    val activeLightsCount: Int = 3,
    val isAlarmArmed: Boolean = true,
    val alarmMode: String = "Armato Notte"
)

/**
 * Service managing Wear OS Tiles and Complications for Domus Hub.
 * Provides the 3 requested rapid scenes ("Esco", "Buonanotte", "Cinema"),
 * live alarm status and watchface complication metrics.
 */
class DomusWearTileService : Service() {

    companion object {
        val shared by lazy { DomusWearTileService() }

        val quickScenes = listOf(
            WearQuickScene("scene_leave", "Esco", "🚶", "scene.leave_home"),
            WearQuickScene("scene_night", "Buonanotte", "🌙", "scene.goodnight"),
            WearQuickScene("scene_cinema", "Cinema", "🎬", "scene.cinema")
        )
    }

    private val scope = CoroutineScope(Dispatchers.IO)
    private val apiClient = DomusApiClient()

    private val _complicationState = MutableStateFlow(WearComplicationData())
    val complicationState: StateFlow<WearComplicationData> = _complicationState.asStateFlow()

    private val _lastExecutedScene = MutableStateFlow<String?>(null)
    val lastExecutedScene: StateFlow<String?> = _lastExecutedScene.asStateFlow()

    override fun onBind(intent: Intent?): IBinder? {
        return null
    }

    /**
     * Executes a scene directly from the Wear OS Tile or Watchface.
     */
    fun triggerWearScene(scene: WearQuickScene, onComplete: ((Boolean) -> Unit)? = null) {
        scope.launch {
            try {
                apiClient.callService(
                    baseUrl = "http://192.168.1.100:8080",
                    token = null,
                    domain = "scene",
                    service = scene.actionService,
                    entityId = scene.entityId
                )
                _lastExecutedScene.value = scene.name
                onComplete?.invoke(true)
            } catch (e: Exception) {
                _lastExecutedScene.value = "${scene.name} (Offline)"
                onComplete?.invoke(false)
            }
        }
    }

    /**
     * Updates complication values from live server state.
     */
    fun updateComplication(temp: Double, lights: Int, alarmArmed: Boolean, mode: String) {
        _complicationState.value = WearComplicationData(
            indoorTemperature = temp,
            activeLightsCount = lights,
            isAlarmArmed = alarmArmed,
            alarmMode = mode
        )
    }

    /**
     * Formats textual complication string for smart watch dials.
     */
    fun getWatchFaceComplicationText(): String {
        val current = _complicationState.value
        return "${current.indoorTemperature}°C • ${current.activeLightsCount} Luci"
    }

    /**
     * Formats textual alarm state for watch tiles.
     */
    fun getAlarmTileText(): String {
        val current = _complicationState.value
        return if (current.isAlarmArmed) "🛡️ ${current.alarmMode}" else "🛡️ Disarmato"
    }
}
