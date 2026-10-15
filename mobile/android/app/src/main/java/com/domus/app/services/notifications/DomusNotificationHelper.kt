package com.domus.app.services.notifications

import android.annotation.SuppressLint
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.media.AudioAttributes
import android.net.Uri
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import com.domus.app.MainActivity

object DomusNotificationHelper {

    const val CHANNEL_CRITICAL_ALARM = "domus_critical_alarm_channel"
    const val CHANNEL_HOME_NOTICES = "domus_home_notices_channel"

    const val NOTIFICATION_ID_CRITICAL = 9001
    const val NOTIFICATION_ID_NOTICE = 9002

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

            notificationManager.createNotificationChannels(listOf(criticalChannel, noticesChannel))
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
}
