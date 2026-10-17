package com.domus.app.services.kiosk

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.BatteryManager
import android.os.PowerManager
import android.util.Log
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Controller for wall-mounted tablet installations running Domus as a dedicated kiosk dashboard.
 * Provides lock-task mode pinning, proximity screen wake-up, night OLED screensaver,
 * and smart battery health cycling (20% - 80%).
 */
class DomusKioskController(
    private val activity: Activity? = null
) {
    companion object {
        private const val TAG = "DomusKiosk"
        private const val BATTERY_MAX_THRESHOLD = 80
        private const val BATTERY_MIN_THRESHOLD = 20

        val shared by lazy { DomusKioskController(null) }
    }

    private val _isKioskModeEnabled = MutableStateFlow(false)
    val isKioskModeEnabled: StateFlow<Boolean> = _isKioskModeEnabled.asStateFlow()

    private val _isScreensaverActive = MutableStateFlow(false)
    val isScreensaverActive: StateFlow<Boolean> = _isScreensaverActive.asStateFlow()

    private val _isBatteryProtectionEnabled = MutableStateFlow(true)
    val isBatteryProtectionEnabled: StateFlow<Boolean> = _isBatteryProtectionEnabled.asStateFlow()

    private val _smartPlugEntityId = MutableStateFlow("switch.tablet_charger")
    val smartPlugEntityId: StateFlow<String> = _smartPlugEntityId.asStateFlow()

    private val _batteryPercentage = MutableStateFlow(78)
    val batteryPercentage: StateFlow<Int> = _batteryPercentage.asStateFlow()

    private val _isCharging = MutableStateFlow(false)
    val isCharging: StateFlow<Boolean> = _isCharging.asStateFlow()

    var onBatteryProtectionAction: ((entityId: String, turnOn: Boolean) -> Unit)? = null

    private var wakeLock: PowerManager.WakeLock? = null
    private var batteryReceiver: BroadcastReceiver? = null

    fun setKioskMode(enabled: Boolean) {
        _isKioskModeEnabled.value = enabled
        if (enabled) {
            enableKioskMode()
        } else {
            disableKioskMode()
        }
    }

    fun enterScreensaver() {
        _isScreensaverActive.value = true
    }

    fun wakeFromScreensaver() {
        _isScreensaverActive.value = false
    }

    fun setBatteryProtection(enabled: Boolean) {
        _isBatteryProtectionEnabled.value = enabled
    }

    fun setSmartPlugId(id: String) {
        _smartPlugEntityId.value = id
    }

    /**
     * Pins the application in Lock Task (Kiosk) mode.
     */
    fun enableKioskMode() {
        activity?.let {
            try {
                it.startLockTask()
                Log.i(TAG, "LockTask Mode (Kiosk) activated successfully.")
            } catch (e: Exception) {
                Log.w(TAG, "Could not startLockTask (Device Owner permission may be required)", e)
            }
        }
        startBatteryHealthProtection()
    }

    /**
     * Releases Lock Task mode.
     */
    fun disableKioskMode() {
        activity?.let {
            try {
                it.stopLockTask()
            } catch (e: Exception) {
                Log.w(TAG, "Error stopping lock task", e)
            }
        }
        stopBatteryHealthProtection()
    }

    /**
     * Wakes up the screen when proximity or motion is detected.
     */
    @Suppress("DEPRECATION")
    fun onProximityDetected() {
        val powerManager = activity?.getSystemService(Context.POWER_SERVICE) as? PowerManager ?: return
        if (!powerManager.isInteractive) {
            wakeLock?.release()
            wakeLock = powerManager.newWakeLock(
                PowerManager.SCREEN_BRIGHT_WAKE_LOCK or PowerManager.ACQUIRE_CAUSES_WAKEUP,
                "domus:kiosk_proximity_wake"
            ).apply {
                acquire(10000L)
            }
            wakeFromScreensaver()
            Log.i(TAG, "Proximity wake triggered: Display powered on")
        }
    }

    private fun startBatteryHealthProtection() {
        if (activity == null) return
        batteryReceiver = object : BroadcastReceiver() {
            override fun onReceive(context: Context, intent: Intent) {
                val level = intent.getIntExtra(BatteryManager.EXTRA_LEVEL, -1)
                val scale = intent.getIntExtra(BatteryManager.EXTRA_SCALE, -1)
                val status = intent.getIntExtra(BatteryManager.EXTRA_STATUS, -1)
                val isPlugged = status == BatteryManager.BATTERY_STATUS_CHARGING || status == BatteryManager.BATTERY_STATUS_FULL
                _isCharging.value = isPlugged

                if (level == -1 || scale == -1) return
                val batteryPct = (level * 100 / scale.toFloat()).toInt()
                _batteryPercentage.value = batteryPct

                if (!_isBatteryProtectionEnabled.value) return

                val plugId = _smartPlugEntityId.value
                if (batteryPct >= BATTERY_MAX_THRESHOLD && isPlugged) {
                    Log.i(TAG, "Battery reached $batteryPct% (>= $BATTERY_MAX_THRESHOLD%): Requesting smart plug power OFF")
                    onBatteryProtectionAction?.invoke(plugId, false)
                } else if (batteryPct <= BATTERY_MIN_THRESHOLD && !isPlugged) {
                    Log.i(TAG, "Battery dropped to $batteryPct% (<= $BATTERY_MIN_THRESHOLD%): Requesting smart plug power ON")
                    onBatteryProtectionAction?.invoke(plugId, true)
                }
            }
        }

        val filter = IntentFilter(Intent.ACTION_BATTERY_CHANGED)
        try {
            activity.registerReceiver(batteryReceiver, filter)
        } catch (e: Exception) {
            Log.w(TAG, "Error registering battery receiver", e)
        }
    }

    private fun stopBatteryHealthProtection() {
        batteryReceiver?.let {
            try {
                activity?.unregisterReceiver(it)
            } catch (e: Exception) {
                Log.w(TAG, "Battery receiver unregister error", e)
            }
            batteryReceiver = null
        }
        wakeLock?.release()
        wakeLock = null
    }
}
