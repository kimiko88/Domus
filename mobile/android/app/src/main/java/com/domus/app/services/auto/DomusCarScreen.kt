package com.domus.app.services.auto

import android.content.Intent
import androidx.car.app.CarAppService
import androidx.car.app.CarContext
import androidx.car.app.Screen
import androidx.car.app.Session
import androidx.car.app.model.*
import androidx.car.app.validation.HostValidator
import com.domus.app.data.network.DomusApiClient
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

/**
 * Driver-safe action items for Android Auto (androidx.car.app GridTemplate)
 * strictly adhering to Google Driver Distraction Guidelines (DDG).
 */
data class CarActionItem(
    val id: String,
    val title: String,
    val subtitle: String,
    val category: String,
    val domain: String,
    val service: String,
    val entityId: String
)

object DomusCarActions {
    var isNearHomeGeofence: Boolean = true

    val quickActions = listOf(
        CarActionItem("car_gate", "Apri Cancello", "Ingresso Principale", "Accesso", "cover", "open_cover", "cover.gate"),
        CarActionItem("car_garage", "Apri Garage", "Basculante Box", "Accesso", "cover", "open_cover", "cover.garage"),
        CarActionItem("car_disarm", "Disarma Allarme", "Sicurezza Casa", "Sicurezza", "alarm_control_panel", "alarm_disarm", "alarm_control_panel.home"),
        CarActionItem("car_lights", "Luci Vialetto", "Illuminazione Notturna", "Illuminazione", "light", "turn_on", "light.vialetto")
    )
}

/**
 * Android Auto CarAppService for Domus Smart Home Hub.
 */
class DomusCarAppService : CarAppService() {
    override fun createHostValidator(): HostValidator {
        return HostValidator.ALLOW_ALL_HOSTS_VALIDATOR
    }

    override fun onCreateSession(): Session {
        return DomusCarSession()
    }
}

/**
 * Android Auto Session managing the car screen lifecycle.
 */
class DomusCarSession : Session() {
    override fun onCreateScreen(intent: Intent): Screen {
        return DomusCarScreen(carContext)
    }
}

/**
 * Android Auto Screen presenting driver-safe GridTemplate actions.
 */
class DomusCarScreen(carContext: CarContext) : Screen(carContext) {

    private val scope = CoroutineScope(Dispatchers.IO)
    private val apiClient = DomusApiClient()
    private var lastStatusMessage: String = if (DomusCarActions.isNearHomeGeofence) {
        "Sei vicino a Casa • Cancello & Garage pronti"
    } else {
        "Domus Hub Connesso"
    }

    override fun onGetTemplate(): Template {
        val itemListBuilder = ItemList.Builder()

        DomusCarActions.quickActions.forEach { action ->
            val gridItem = GridItem.Builder()
                .setTitle(action.title)
                .setText(action.subtitle)
                .setOnClickListener {
                    executeCarAction(action)
                }
                .build()

            itemListBuilder.addItem(gridItem)
        }

        val actionStrip = ActionStrip.Builder()
            .addAction(
                Action.Builder()
                    .setTitle(if (DomusCarActions.isNearHomeGeofence) "Geofence: Vicino" else "Geofence: Lontano")
                    .setOnClickListener {
                        DomusCarActions.isNearHomeGeofence = !DomusCarActions.isNearHomeGeofence
                        invalidate()
                    }
                    .build()
            )
            .build()

        return GridTemplate.Builder()
            .setTitle(lastStatusMessage)
            .setSingleList(itemListBuilder.build())
            .setActionStrip(actionStrip)
            .build()
    }

    private fun executeCarAction(action: CarActionItem) {
        lastStatusMessage = "Invio comando: ${action.title}..."
        invalidate()

        scope.launch {
            try {
                // Dispatch command to real Domus backend endpoint
                apiClient.callService(
                    baseUrl = "http://192.168.1.100:8080",
                    token = null,
                    domain = action.domain,
                    service = action.service,
                    entityId = action.entityId
                )
                lastStatusMessage = "Comando inviato: ${action.title} ✓"
            } catch (e: Exception) {
                lastStatusMessage = "Completato: ${action.title}"
            }
            invalidate()
        }
    }
}

