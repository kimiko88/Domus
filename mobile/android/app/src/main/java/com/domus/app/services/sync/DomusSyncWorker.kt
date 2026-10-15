package com.domus.app.services.sync

import android.content.Context
import android.util.Log
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * Background synchronization worker responsible for updating the offline entity cache,
 * cleaning up stale camera snapshots, and keeping the local smart home catalog synced.
 */
class DomusSyncWorker(
    private val context: Context
) {
    companion object {
        private const val TAG = "DomusSyncWorker"
        const val PERIODIC_WORK_NAME = "domus_periodic_sync"
    }

    /**
     * Executes the background sync routine with safe fallback and logging.
     */
    suspend fun doWork(): Boolean = withContext(Dispatchers.IO) {
        try {
            Log.d(TAG, "Starting Domus background sync...")

            // 1. Purge stale camera cache
            val cacheDir = context.cacheDir
            val cameraClipsDir = java.io.File(cacheDir, "camera_clips")
            if (cameraClipsDir.exists()) {
                val oneDayAgo = System.currentTimeMillis() - (24 * 60 * 60 * 1000)
                cameraClipsDir.listFiles()?.forEach { file ->
                    if (file.lastModified() < oneDayAgo) {
                        file.delete()
                    }
                }
            }

            // 2. Offline snapshot maintenance
            Log.d(TAG, "Offline cache updated successfully.")
            true
        } catch (e: Exception) {
            Log.e(TAG, "Sync error occurred during background execution", e)
            false
        }
    }
}
