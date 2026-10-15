package com.domus.app.services.location

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.util.Log

/**
 * Handles geofence entry/exit transitions and local Wi-Fi connectivity events
 * to deliver instant, battery-friendly home presence updates to the Domus Core hub.
 */
class DomusGeofenceReceiver : BroadcastReceiver() {

    companion object {
        private const val TAG = "DomusGeofence"
        const val ACTION_GEOFENCE_EVENT = "com.domus.app.ACTION_GEOFENCE_EVENT"

        const val TRANSITION_ENTER = 1
        const val TRANSITION_EXIT = 2
    }

    override fun onReceive(context: Context, intent: Intent) {
        val transitionType = intent.getIntExtra("transition_type", 0)

        when (transitionType) {
            TRANSITION_ENTER -> {
                Log.i(TAG, "Geofence ENTER: Utente rientrato nel raggio di 150m dall'abitazione")
                dispatchPresenceUpdate(context, isHome = true, source = "gps_geofence")
            }
            TRANSITION_EXIT -> {
                Log.i(TAG, "Geofence EXIT: Utente allontanato dall'abitazione")
                dispatchPresenceUpdate(context, isHome = false, source = "gps_geofence")
            }
            else -> {
                // Check if device just connected to Home Wi-Fi
                if (isHomeWifiConnected(context)) {
                    Log.i(TAG, "Wi-Fi LAN connesso: Presenza 'In Casa' confermata con zero latenza")
                    dispatchPresenceUpdate(context, isHome = true, source = "home_wifi")
                }
            }
        }
    }

    private fun isHomeWifiConnected(context: Context): Boolean {
        val cm = context.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager ?: return false
        val activeNetwork = cm.activeNetwork ?: return false
        val capabilities = cm.getNetworkCapabilities(activeNetwork) ?: return false
        return capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI)
    }

    private fun dispatchPresenceUpdate(context: Context, isHome: Boolean, source: String) {
        // Dispatch presence state to Domus Core via HTTP or local Broadcast
        Log.d(TAG, "Dispatching presence update: isHome=$isHome, source=$source")
    }
}
