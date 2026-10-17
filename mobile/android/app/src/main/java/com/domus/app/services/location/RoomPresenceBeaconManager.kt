package com.domus.app.services.location

import android.annotation.SuppressLint
import android.bluetooth.BluetoothAdapter
import android.bluetooth.le.BluetoothLeScanner
import android.bluetooth.le.ScanCallback
import android.bluetooth.le.ScanResult
import android.bluetooth.le.ScanSettings
import android.content.Context
import android.util.Log
import com.domus.app.domain.model.DomusRoomBeacon
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlin.math.pow

/**
 * Gestore della micro-localizzazione indoor stanza per stanza su Android.
 * Utilizza Bluetooth LE (Beacon / BLE Proxies) con soglia RSSI per identificare la presenza fisica dell'utente nella stanza.
 */
class RoomPresenceBeaconManager(
    private val context: Context? = null,
    val configuredBeacons: List<DomusRoomBeacon> = defaultBeacons
) {
    companion object {
        private const val TAG = "DomusBeaconManager"

        val defaultBeacons = listOf(
            DomusRoomBeacon(id = "b1", roomId = "living_room", roomName = "Salotto", major = 1, minor = 1, rssiThreshold = -65),
            DomusRoomBeacon(id = "b2", roomId = "kitchen", roomName = "Cucina", major = 1, minor = 2, rssiThreshold = -65),
            DomusRoomBeacon(id = "b3", roomId = "bedroom", roomName = "Camera da Letto", major = 1, minor = 3, rssiThreshold = -68)
        )

        val shared by lazy { RoomPresenceBeaconManager() }
    }

    private val _activeRoomId = MutableStateFlow<String?>(null)
    val activeRoomId: StateFlow<String?> = _activeRoomId.asStateFlow()

    private val _activeRoomName = MutableStateFlow<String?>(null)
    val activeRoomName: StateFlow<String?> = _activeRoomName.asStateFlow()

    private val _currentDistanceMeters = MutableStateFlow<Double?>(null)
    val currentDistanceMeters: StateFlow<Double?> = _currentDistanceMeters.asStateFlow()

    private val _isScanning = MutableStateFlow(false)
    val isScanning: StateFlow<Boolean> = _isScanning.asStateFlow()

    var onRoomChanged: ((String?) -> Unit)? = null

    private var bleScanner: BluetoothLeScanner? = null
    private var scanCallback: ScanCallback? = null

    @SuppressLint("MissingPermission")
    fun startMonitoring() {
        _isScanning.value = true
        try {
            val bluetoothAdapter = BluetoothAdapter.getDefaultAdapter()
            if (bluetoothAdapter != null && bluetoothAdapter.isEnabled) {
                bleScanner = bluetoothAdapter.bluetoothLeScanner
                scanCallback = object : ScanCallback() {
                    override fun onScanResult(callbackType: Int, result: ScanResult?) {
                        result?.let { handleScanResult(it) }
                    }

                    override fun onBatchScanResults(results: MutableList<ScanResult>?) {
                        results?.forEach { handleScanResult(it) }
                    }

                    override fun onScanFailed(errorCode: Int) {
                        Log.w(TAG, "BLE scan failed with error code: $errorCode")
                    }
                }

                val settings = ScanSettings.Builder()
                    .setScanMode(ScanSettings.SCAN_MODE_LOW_LATENCY)
                    .build()

                bleScanner?.startScan(null, settings, scanCallback)
                Log.i(TAG, "BLE beacon indoor tracking started")
            } else {
                Log.i(TAG, "Bluetooth not enabled, falling back to simulated beacon monitoring")
            }
        } catch (e: Exception) {
            Log.w(TAG, "Failed to start BLE scanning (permissions missing or hardware issue)", e)
        }
    }

    @SuppressLint("MissingPermission")
    fun stopMonitoring() {
        _isScanning.value = false
        try {
            scanCallback?.let { bleScanner?.stopScan(it) }
        } catch (e: Exception) {
            Log.w(TAG, "Error stopping BLE scan", e)
        }
        bleScanner = null
        scanCallback = null
        _activeRoomId.value = null
        _activeRoomName.value = null
        _currentDistanceMeters.value = null
    }

    fun simulateRoomProximity(roomId: String?, roomName: String? = null, distanceMeters: Double? = 0.8) {
        val matched = configuredBeacons.firstOrNull { it.roomId == roomId }
        val finalRoomName = roomName ?: matched?.roomName ?: (if (roomId == "living_room") "Salotto" else roomId?.replaceFirstChar { it.uppercase() })
        _activeRoomId.value = roomId
        _activeRoomName.value = finalRoomName
        _currentDistanceMeters.value = distanceMeters
        onRoomChanged?.invoke(roomId)
    }

    fun evaluateRssiSample(major: Int, minor: Int, rssi: Int) {
        val beacon = configuredBeacons.firstOrNull { it.major == major && it.minor == minor } ?: return

        if (rssi >= beacon.rssiThreshold) {
            val dist = calculateDistance(rssi, -59)
            _activeRoomId.value = beacon.roomId
            _activeRoomName.value = beacon.roomName
            _currentDistanceMeters.value = dist
            onRoomChanged?.invoke(beacon.roomId)
        } else if (_activeRoomId.value == beacon.roomId && rssi < (beacon.rssiThreshold - 8)) {
            _activeRoomId.value = null
            _activeRoomName.value = null
            _currentDistanceMeters.value = null
            onRoomChanged?.invoke(null)
        }
    }

    private fun handleScanResult(result: ScanResult) {
        val scanRecord = result.scanRecord?.bytes ?: return
        val (major, minor) = parseIBeaconMajorMinor(scanRecord) ?: return
        evaluateRssiSample(major, minor, result.rssi)
    }

    private fun parseIBeaconMajorMinor(scanRecord: ByteArray): Pair<Int, Int>? {
        var startByte = 2
        var patternFound = false
        while (startByte <= 5 && startByte < scanRecord.size - 24) {
            if (scanRecord[startByte].toInt() and 0xFF == 0x02 && scanRecord[startByte + 1].toInt() and 0xFF == 0x15) {
                patternFound = true
                break
            }
            startByte++
        }
        if (!patternFound) return null

        val major = ((scanRecord[startByte + 18].toInt() and 0xFF) shl 8) or (scanRecord[startByte + 19].toInt() and 0xFF)
        val minor = ((scanRecord[startByte + 20].toInt() and 0xFF) shl 8) or (scanRecord[startByte + 21].toInt() and 0xFF)
        return Pair(major, minor)
    }

    private fun calculateDistance(rssi: Int, txPower: Int): Double {
        if (rssi == 0) return -1.0
        val ratio = rssi * 1.0 / txPower
        return if (ratio < 1.0) {
            ratio.pow(10.0)
        } else {
            0.89976 * ratio.pow(7.7095) + 0.111
        }
    }
}
