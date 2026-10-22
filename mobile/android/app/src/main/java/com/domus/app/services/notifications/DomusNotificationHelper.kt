package com.domus.app.services.notifications

import android.annotation.SuppressLint
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.graphics.BitmapFactory
import android.media.AudioAttributes
import android.net.Uri
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import com.domus.app.MainActivity

object DomusNotificationHelper {

    const val CHANNEL_CRITICAL_ALARM = "domus_critical_alarm_channel"
    const val CHANNEL_HOME_NOTICES = "domus_home_notices_channel"
    /** Dedicated channel for AI vision events: face recognition and plate detection. */
    const val CHANNEL_AI_VISION = "domus_ai_vision_channel"

    const val NOTIFICATION_ID_CRITICAL = 9001
    const val NOTIFICATION_ID_NOTICE = 9002
    const val NOTIFICATION_ID_AI_VISION_BASE = 9100 // +sequential offset per event

    /**
     * Initializes specialized notification channels with strict priority policies
     * and audio attributes capable of overriding Do-Not-Disturb for critical alarms.
     */
    fun createNotificationChannels(context: Context) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val notificationManager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

            // 1. Critical Alarm Channel (AudioAttributes.USAGE_ALARM + FullScreenIntent)
            val alarmAudioAttributes = AudioAttributes.Builder()
                .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                .setUsage(AudioAttributes.USAGE_ALARM)
                .setFlags(AudioAttributes.FLAG_AUDIBILITY_ENFORCED)
                .build()

            val criticalChannel = NotificationChannel(
                CHANNEL_CRITICAL_ALARM,
                "Allarmi Critici Domus",
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = "Notifiche di emergenza, effrazione e allarmi di sicurezza prioritari"
                enableLights(true)
                lightColor = android.graphics.Color.RED
                enableVibration(true)
                vibrationPattern = longArrayOf(0, 500, 200, 500, 200, 1000)
                setSound(
                    Uri.parse("android.resource://" + context.packageName + "/raw/alarm_siren"),
                    alarmAudioAttributes
                )
                lockscreenVisibility = NotificationCompat.VISIBILITY_PUBLIC
            }

            // 2. Home Notices Channel (Standard priority with actionable buttons)
            val noticesChannel = NotificationChannel(
                CHANNEL_HOME_NOTICES,
                "Avvisi Domestici & Faccende",
                NotificationManager.IMPORTANCE_DEFAULT
            ).apply {
                description = "Promemoria carichi energetici, lista della spesa e faccende"
                enableLights(true)
                lightColor = android.graphics.Color.CYAN
                enableVibration(false)
            }

            // 3. AI Vision Channel — Face Recognition & Plate Detection snapshots
            val visionChannel = NotificationChannel(
                CHANNEL_AI_VISION,
                "Visione AI — Volti & Targhe",
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = "Notifiche ricche con istantanea per riconoscimento volti familiari e targhe autorizzate"
                enableLights(true)
                lightColor = android.graphics.Color.BLUE
                enableVibration(true)
                vibrationPattern = longArrayOf(0, 200, 100, 200)
                lockscreenVisibility = NotificationCompat.VISIBILITY_PRIVATE
            }

            notificationManager.createNotificationChannels(listOf(criticalChannel, noticesChannel, visionChannel))
        }
    }

    /**
     * Dispatches a high-priority intrusion or critical safety notification.
     */
    @SuppressLint("MissingPermission")
    fun sendCriticalAlarmNotification(
        context: Context,
        title: String,
        message: String
    ) {
        val fullScreenIntent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra("route", "alarm")
        }
        val fullScreenPendingIntent = PendingIntent.getActivity(
            context,
            0,
            fullScreenIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notification = NotificationCompat.Builder(context, CHANNEL_CRITICAL_ALARM)
            .setSmallIcon(android.R.drawable.ic_lock_idle_alarm)
            .setContentTitle(title)
            .setContentText(message)
            .setPriority(NotificationCompat.PRIORITY_MAX)
            .setCategory(NotificationCompat.CATEGORY_ALARM)
            .setAutoCancel(false)
            .setOngoing(true)
            .setFullScreenIntent(fullScreenPendingIntent, true)
            .addAction(android.R.drawable.ic_menu_close_clear_cancel, "Disarma Subito", fullScreenPendingIntent)
            .build()

        NotificationManagerCompat.from(context).notify(NOTIFICATION_ID_CRITICAL, notification)
    }

    /**
     * Dispatches an interactive domestic notification with inline quick action buttons.
     */
    @SuppressLint("MissingPermission")
    fun sendHomeNoticeNotification(
        context: Context,
        title: String,
        message: String,
        actionTitle: String = "Spegni Luci"
    ) {
        val contentIntent = Intent(context, MainActivity::class.java)
        val pendingIntent = PendingIntent.getActivity(
            context,
            1,
            contentIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notification = NotificationCompat.Builder(context, CHANNEL_HOME_NOTICES)
            .setSmallIcon(android.R.drawable.ic_dialog_info)
            .setContentTitle(title)
            .setContentText(message)
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .addAction(android.R.drawable.ic_media_play, actionTitle, pendingIntent)
            .addAction(android.R.drawable.ic_menu_recent_history, "Posticipa 30m", pendingIntent)
            .build()

        NotificationManagerCompat.from(context).notify(NOTIFICATION_ID_NOTICE, notification)
    }

    /**
     * Dispatches a rich AI Vision notification for face recognition or plate detection events.
     *
     * Displays a BigPicture snapshot from the camera, the detected subject label,
     * and quick-action buttons to open the gate or dismiss the event as a false positive.
     *
     * @param eventType Either "face_recognized" or "plate_detected"
     * @param subjectLabel Human-readable label (e.g. "Marco Rossi" or "[IT] AB 123 CD")
     * @param cameraName Friendly name of the camera that generated the event
     * @param snapshotBytes Optional JPEG/PNG bytes for the BigPicture style; falls back to text-only
     * @param confidence Cosine similarity score (0.0–1.0) displayed in the sub-text
     * @param gateEntityId Optional entity_id to trigger gate opening via deep-link
     * @param notificationIndex Sequential index to differentiate concurrent events
     */
    @SuppressLint("MissingPermission")
    fun sendAIVisionNotification(
        context: Context,
        eventType: String,
        subjectLabel: String,
        cameraName: String = "Telecamera Esterna",
        snapshotBytes: ByteArray? = null,
        confidence: Float = 1.0f,
        gateEntityId: String? = null,
        notificationIndex: Int = 0
    ) {
        val isFace = eventType == "face_recognized"
        val icon = if (isFace) android.R.drawable.ic_menu_myplaces else android.R.drawable.ic_menu_directions
        val title = if (isFace) "Volto Riconosciuto — $subjectLabel" else "Targa Rilevata — $subjectLabel"
        val body = "Camera: $cameraName · Confidenza: ${(confidence * 100).toInt()}%"

        // Deep-link intent to camera detail screen
        val contentIntent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra("route", "cameras")
            putExtra("event_type", eventType)
            putExtra("subject_label", subjectLabel)
        }
        val contentPendingIntent = PendingIntent.getActivity(
            context, 200 + notificationIndex, contentIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        // Optional: Gate open action
        val gateIntent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra("route", "gate_open")
            putExtra("entity_id", gateEntityId ?: "")
        }
        val gatePendingIntent = PendingIntent.getActivity(
            context, 300 + notificationIndex, gateIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val builder = NotificationCompat.Builder(context, CHANNEL_AI_VISION)
            .setSmallIcon(icon)
            .setContentTitle(title)
            .setContentText(body)
            .setSubText(cameraName)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setCategory(NotificationCompat.CATEGORY_EVENT)
            .setAutoCancel(true)
            .setContentIntent(contentPendingIntent)
            .addAction(android.R.drawable.ic_menu_view, "Vedi Camera", contentPendingIntent)

        // Add gate action only when relevant (recognized face or authorized plate)
        if (gateEntityId != null) {
            builder.addAction(android.R.drawable.ic_menu_directions, "Apri Cancello", gatePendingIntent)
        }

        // Attach snapshot as BigPictureStyle if bytes are available
        if (snapshotBytes != null) {
            runCatching {
                val bmp = BitmapFactory.decodeByteArray(snapshotBytes, 0, snapshotBytes.size)
                val bigStyle = NotificationCompat.BigPictureStyle()
                    .bigPicture(bmp)
                    .setSummaryText(body)
                builder.setStyle(bigStyle)
                builder.setLargeIcon(bmp)
            }
        } else {
            builder.setStyle(
                NotificationCompat.BigTextStyle()
                    .bigText("$body\n\nTocca per aprire il feed della telecamera e verificare l'evento.")
            )
        }

        NotificationManagerCompat.from(context).notify(NOTIFICATION_ID_AI_VISION_BASE + notificationIndex, builder.build())
    }
}
