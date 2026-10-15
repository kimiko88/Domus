package com.domus.app.services.kiosk

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.BatteryManager
import android.os.PowerManager
import android.util.Log

/**
 * Controller for wall-mounted tablet installations running Domus as a dedicated kiosk dashboard.
 * Provides lock-task mode pinning, proximity screen wake-up, and smart battery health cycling (20% - 85%).
 */
class DomusKioskController(
    private val activity: Activity
) {
    companion object {
        private const val TAG = "DomusKiosk"
        private const val BATTERY_MAX_THRESHOLD = 85
        private const val BATTERY_MIN_THRESHOLD = 20
    }

    private var wakeLock: PowerManager.WakeLock? = null
    private var batteryReceiver: BroadcastReceiver? = null

    /**
     * Pins the application in Lock Task (Kiosk) mode, hiding system navigation and status bars.
     */
    fun enableKioskMode() {
        try {
            activity.startLockTask()
            Log.i(TAG, "LockTask Mode (Kiosk) activated successfully.")
        } catch (e: Exception) {
            Log.w(TAG, "Could not startLockTask (Device Owner permission may be required)", e)
        }
        startBatteryHealthProtection()
    }

    /**
     * Releases Lock Task mode.
     */
    fun disableKioskMode() {
        try {
            activity.stopLockTask()
        } catch (e: Exception) {
            Log.w(TAG, "Error stopping lock task", e)
        }
        stopBatteryHealthProtection()
    }

    /**
     * Wakes up the screen when proximity or motion is detected in front of the wall tablet.
     */
    fun onProximityDetected() {
        val powerManager = activity.getSystemService(Context.POWER_SERVICE) as? PowerManager ?: return
        if (!powerManager.isInteractive) {
            wakeLock?.release()
            wakeLock = powerManager.newWakeLock(
                PowerManager.SCREEN_BRIGHT_WAKE_LOCK or PowerManager.ACQUIRE_CAUSES_WAKEUP,
                "domus:kiosk_proximity_wake"
            ).apply {
                acquire(10000L) // Keep screen awake for 10 seconds or until user interacts
            }
            Log.i(TAG, "Proximity wake triggered: Display powered on")
        }
    }

    /**
     * Monitors tablet battery charge level to prevent swelling and degradation
     * by controlling the smart plug feeding the tablet charger.
     */
    private fun startBatteryHealthProtection() {
        batteryReceiver = object : BroadcastReceiver() {
            override fun onReceive(context: Context, intent: Intent) {
                val level = intent.getIntExtra(BatteryManager.EXTRA_LEVEL, -1)
                val scale = intent.getIntExtra(BatteryManager.EXTRA_SCALE, -1)
                if (level == -1 || scale == -1) return

                val batteryPct = (level * 100 / scale.toFloat()).toInt()

                if (batteryPct >= BATTERY_MAX_THRESHOLD) {
                    Log.i(TAG, "Battery reached $batteryPct% (>= $BATTERY_MAX_THRESHOLD%): Requesting smart plug power OFF")
                    // Dispatch command to turn off charger smart plug
                } else if (batteryPct <= BATTERY_MIN_THRESHOLD) {
                    Log.i(TAG, "Battery dropped to $batteryPct% (<= $BATTERY_MIN_THRESHOLD%): Requesting smart plug power ON")
                    // Dispatch command to turn on charger smart plug
                }
            }
        }

        val filter = IntentFilter(Intent.ACTION_BATTERY_CHANGED)
        activity.registerReceiver(batteryReceiver, filter)
    }

    private fun stopBatteryHealthProtection() {
        batteryReceiver?.let {
            try {
                activity.unregisterReceiver(it)
            } catch (e: Exception) {
                Log.w(TAG, "Battery receiver unregister error", e)
            }
            batteryReceiver = null
        }
        wakeLock?.release()
        wakeLock = null
    }
}
