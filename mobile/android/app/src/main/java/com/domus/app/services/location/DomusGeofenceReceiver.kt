package com.domus.app.services.location

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.util.Log
import com.domus.app.data.network.DomusApiClient
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

/**
 * Handles geofence entry/exit transitions and local Wi-Fi connectivity events
 * to deliver instant, battery-friendly home presence updates to the Domus Core hub.
 * On geofence transitions, also calls `/api/v1/presence/commute` to update the
 * CommuteTracker so ETA-based automations (preheat, gate open) are triggered correctly.
 */
class DomusGeofenceReceiver : BroadcastReceiver() {

    companion object {
        private const val TAG = "DomusGeofence"
        const val ACTION_GEOFENCE_EVENT = "com.domus.app.ACTION_GEOFENCE_EVENT"

        const val TRANSITION_ENTER = 1
        const val TRANSITION_EXIT = 2

        // Saved SharedPreferences keys (written by DashboardViewModel on login)
        private const val PREFS_NAME = "domus_prefs"
        private const val KEY_BASE_URL = "base_url"
        private const val KEY_TOKEN = "auth_token"
        private const val KEY_USER_ID = "user_id"
    }

    // BroadcastReceiver lifecycle is very short; use a SupervisorJob scope
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    override fun onReceive(context: Context, intent: Intent) {
        val transitionType = intent.getIntExtra("transition_type", 0)
        val lat = intent.getDoubleExtra("latitude", 0.0)
        val lon = intent.getDoubleExtra("longitude", 0.0)

        when (transitionType) {
            TRANSITION_ENTER -> {
                Log.i(TAG, "Geofence ENTER: Utente rientrato nel raggio di 150m dall'abitazione")
                dispatchPresenceUpdate(context, isHome = true, source = "gps_geofence")
                dispatchCommuteUpdate(context, lat, lon, isArrived = true)
            }
            TRANSITION_EXIT -> {
                Log.i(TAG, "Geofence EXIT: Utente allontanato dall'abitazione")
                dispatchPresenceUpdate(context, isHome = false, source = "gps_geofence")
                if (lat != 0.0 || lon != 0.0) {
                    dispatchCommuteUpdate(context, lat, lon, isArrived = false)
                }
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
        Log.d(TAG, "Dispatching presence update: isHome=$isHome, source=$source")
        // Broadcast locally for in-process listeners (e.g. DashboardViewModel)
        val broadcastIntent = Intent("com.domus.app.ACTION_PRESENCE_CHANGED").apply {
            putExtra("is_home", isHome)
            putExtra("source", source)
        }
        context.sendBroadcast(broadcastIntent)
    }

    /**
     * Sends a commute GPS location update to the Domus Core hub so the
     * CommuteTracker can compute ETA and trigger preheat/lighting automations.
     */
    private fun dispatchCommuteUpdate(context: Context, lat: Double, lon: Double, isArrived: Boolean) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val baseUrl = prefs.getString(KEY_BASE_URL, null) ?: return
        val token = prefs.getString(KEY_TOKEN, null)
        val userId = prefs.getString(KEY_USER_ID, "me") ?: "me"

        scope.launch {
            try {
                val apiClient = DomusApiClient()
                val result = apiClient.postCommuteLocation(
                    baseUrl = baseUrl,
                    token = token,
                    userId = userId,
                    latitude = lat,
                    longitude = lon
                )
                if (result.isSuccess) {
                    Log.i(TAG, "Commute location aggiornata: lat=$lat lon=$lon arrived=$isArrived")
                } else {
                    Log.w(TAG, "Commute update fallita: ${result.exceptionOrNull()?.message}")
                }
            } catch (e: Exception) {
                Log.e(TAG, "Errore dispatch commute update", e)
            }
        }
    }
}
