package com.domus.app.services.quicksettings

import android.content.Intent
import android.graphics.drawable.Icon
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.service.quicksettings.Tile
import android.service.quicksettings.TileService
import androidx.annotation.RequiresApi
import com.domus.app.MainActivity
import com.domus.app.data.network.DomusApiClient
import com.domus.app.data.storage.SharedPreferencesTokenStorage
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

/**
 * 1. Quick Settings Tile: Apertura rapida cancello o portone d'ingresso.
 */
@RequiresApi(Build.VERSION_CODES.N)
class DomusGateTileService : TileService() {
    private val scope = CoroutineScope(Dispatchers.IO)

    override fun onStartListening() {
        super.onStartListening()
        val tile = qsTile ?: return
        tile.label = "Cancello Domus"
        tile.state = Tile.STATE_INACTIVE
        tile.updateTile()
    }

    override fun onClick() {
        super.onClick()
        val tile = qsTile ?: return

        // Instant visual feedback
        tile.state = Tile.STATE_ACTIVE
        tile.subtitle = "Apertura in corso..."
        tile.updateTile()

        scope.launch {
            val storage = SharedPreferencesTokenStorage(applicationContext)
            val baseUrl = storage.getLocalEndpoint() ?: "http://192.168.1.100:8080"
            val token = storage.getToken()
            DomusApiClient().callService(baseUrl, token, "cover", "open_cover", "cover.cancello")

            Handler(Looper.getMainLooper()).post {
                tile.state = Tile.STATE_INACTIVE
                tile.subtitle = "Aperto"
                tile.updateTile()
            }
        }
    }
}

/**
 * 2. Quick Settings Tile: Stato allarme e attivazione rapida.
 */
@RequiresApi(Build.VERSION_CODES.N)
class DomusAlarmTileService : TileService() {

    private var isArmed = false
    private val scope = CoroutineScope(Dispatchers.IO)

    override fun onStartListening() {
        super.onStartListening()
        updateTileState()
    }

    override fun onClick() {
        super.onClick()
        if (isLocked) {
            unlockAndRun { toggleAlarm() }
        } else {
            toggleAlarm()
        }
    }

    private fun toggleAlarm() {
        isArmed = !isArmed
        updateTileState()

        scope.launch {
            val storage = SharedPreferencesTokenStorage(applicationContext)
            val baseUrl = storage.getLocalEndpoint() ?: "http://192.168.1.100:8080"
            val token = storage.getToken()
            val service = if (isArmed) "alarm_arm_home" else "alarm_disarm"
            DomusApiClient().callService(baseUrl, token, "alarm_control_panel", service, "alarm.home")
        }

        if (!isArmed) {
            // If disarming, prompt for PIN or Biometrics by launching Domus app
            val intent = Intent(this, MainActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
                putExtra("action", "disarm_prompt")
            }
            @Suppress("DEPRECATION")
            startActivityAndCollapse(intent)
        }
    }

    private fun updateTileState() {
        val tile = qsTile ?: return
        tile.label = "Allarme Domus"
        tile.state = if (isArmed) Tile.STATE_ACTIVE else Tile.STATE_INACTIVE
        tile.subtitle = if (isArmed) "Armato (Totale)" else "Disarmato"
        tile.updateTile()
    }
}

/**
 * 3. Quick Settings Tile: Spegni tutte le luci in uscita.
 */
@RequiresApi(Build.VERSION_CODES.N)
class DomusLightTileService : TileService() {
    private val scope = CoroutineScope(Dispatchers.IO)

    override fun onStartListening() {
        super.onStartListening()
        val tile = qsTile ?: return
        tile.label = "Luci Domus"
        tile.subtitle = "Spegni Tutte"
        tile.state = Tile.STATE_INACTIVE
        tile.updateTile()
    }

    override fun onClick() {
        super.onClick()
        val tile = qsTile ?: return

        tile.state = Tile.STATE_ACTIVE
        tile.subtitle = "Invio comando..."
        tile.updateTile()

        scope.launch {
            val storage = SharedPreferencesTokenStorage(applicationContext)
            val baseUrl = storage.getLocalEndpoint() ?: "http://192.168.1.100:8080"
            val token = storage.getToken()
            DomusApiClient().callService(baseUrl, token, "light", "turn_off", "light.all")

            Handler(Looper.getMainLooper()).post {
                tile.state = Tile.STATE_INACTIVE
                tile.subtitle = "Luci Spente ✓"
                tile.updateTile()
            }
        }
    }
}

/**
 * 4. Quick Settings Tile: Attivazione Scenario ("Buonanotte" / "Rientro").
 */
@RequiresApi(Build.VERSION_CODES.N)
class DomusSceneTileService : TileService() {

    private var currentScene = "Buonanotte"
    private val scope = CoroutineScope(Dispatchers.IO)

    override fun onStartListening() {
        super.onStartListening()
        val tile = qsTile ?: return
        tile.label = "Scenario Domus"
        tile.subtitle = currentScene
        tile.state = Tile.STATE_INACTIVE
        tile.updateTile()
    }

    override fun onClick() {
        super.onClick()
        val tile = qsTile ?: return

        tile.state = Tile.STATE_ACTIVE
        tile.subtitle = "Attivazione $currentScene..."
        tile.updateTile()

        scope.launch {
            val storage = SharedPreferencesTokenStorage(applicationContext)
            val baseUrl = storage.getLocalEndpoint() ?: "http://192.168.1.100:8080"
            val token = storage.getToken()
            val sceneId = if (currentScene == "Buonanotte") "scene.buonanotte" else "scene.rientro"
            DomusApiClient().callService(baseUrl, token, "scene", "turn_on", sceneId)

            Handler(Looper.getMainLooper()).post {
                currentScene = if (currentScene == "Buonanotte") "Rientro Casa" else "Buonanotte"
                tile.state = Tile.STATE_INACTIVE
                tile.subtitle = currentScene
                tile.updateTile()
            }
        }
    }
}
