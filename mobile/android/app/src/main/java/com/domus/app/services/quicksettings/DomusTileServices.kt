package com.domus.app.services.quicksettings

import android.content.Intent
import android.os.Build
import android.service.quicksettings.Tile
import android.service.quicksettings.TileService
import androidx.annotation.RequiresApi
import com.domus.app.MainActivity

/**
 * Quick Settings actions open the authenticated Domus UI.
 *
 * These entry points deliberately do not call the service API directly:
 * the app owns authentication, authorization and any biometric confirmation
 * required for physical actions or scenes.
 */
@RequiresApi(Build.VERSION_CODES.N)
private fun TileService.openDomus() {
    val intent = Intent(this, MainActivity::class.java).apply {
        flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
    }
    @Suppress("DEPRECATION")
    startActivityAndCollapse(intent)
}

@RequiresApi(Build.VERSION_CODES.N)
class DomusGateTileService : TileService() {
    override fun onStartListening() {
        super.onStartListening()
        qsTile?.let { tile ->
            tile.label = "Cancello Domus"
            tile.subtitle = "Apri Domus per gestire"
            tile.state = Tile.STATE_INACTIVE
            tile.updateTile()
        }
    }

    override fun onClick() {
        super.onClick()
        openDomus()
    }
}

@RequiresApi(Build.VERSION_CODES.N)
class DomusAlarmTileService : TileService() {
    override fun onStartListening() {
        super.onStartListening()
        qsTile?.let { tile ->
            tile.label = "Allarme Domus"
            tile.subtitle = "Apri Domus per gestire"
            tile.state = Tile.STATE_INACTIVE
            tile.updateTile()
        }
    }

    override fun onClick() {
        super.onClick()
        openDomus()
    }
}

@RequiresApi(Build.VERSION_CODES.N)
class DomusLightTileService : TileService() {
    override fun onStartListening() {
        super.onStartListening()
        qsTile?.let { tile ->
            tile.label = "Luci Domus"
            tile.subtitle = "Apri Domus per gestire"
            tile.state = Tile.STATE_INACTIVE
            tile.updateTile()
        }
    }

    override fun onClick() {
        super.onClick()
        openDomus()
    }
}

@RequiresApi(Build.VERSION_CODES.N)
class DomusSceneTileService : TileService() {
    override fun onStartListening() {
        super.onStartListening()
        qsTile?.let { tile ->
            tile.label = "Scenario Domus"
            tile.subtitle = "Apri Domus per gestire"
            tile.state = Tile.STATE_INACTIVE
            tile.updateTile()
        }
    }

    override fun onClick() {
        super.onClick()
        openDomus()
    }
}
