package com.domus.app.presentation.screens.kiosk

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.*
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.delay
import java.text.SimpleDateFormat
import java.util.*

/**
 * Screensaver notturno OLED per Tablet a parete — Fase 5 Android.
 * Mostra orologio minimale ad altissimo contrasto, meteo e consumi riducendo al minimo la luminosità.
 * Si risveglia con un tocco ovunque sullo schermo.
 */
@Composable
fun KioskScreensaverView(
    onWake: () -> Unit,
    modifier: Modifier = Modifier
) {
    var currentTime by remember { mutableStateOf("12:00") }
    var currentDate by remember { mutableStateOf("Lunedì, 11 Ottobre") }

    LaunchedEffect(Unit) {
        val timeFormat = SimpleDateFormat("HH:mm", Locale.ITALY)
        val dateFormat = SimpleDateFormat("EEEE, d MMMM", Locale.ITALY)
        while (true) {
            val now = Date()
            currentTime = timeFormat.format(now)
            currentDate = dateFormat.format(now).replaceFirstChar { it.uppercase() }
            delay(1000)
        }
    }

    Box(
        modifier = modifier
            .fillMaxSize()
            .background(Color.Black)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null
            ) {
                onWake()
            },
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center
        ) {
            Spacer(modifier = Modifier.weight(1f))

            // Minimal Clock
            Text(
                text = currentTime,
                fontSize = 84.sp,
                fontWeight = FontWeight.Thin,
                color = Color.White.copy(alpha = 0.88f),
                letterSpacing = (-2).sp
            )

            Spacer(modifier = Modifier.height(8.dp))

            // Date
            Text(
                text = currentDate,
                fontSize = 20.sp,
                fontWeight = FontWeight.Normal,
                color = Color.White.copy(alpha = 0.55f)
            )

            Spacer(modifier = Modifier.height(18.dp))

            // House summary badges
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                Text(
                    text = "🌡️ 21.5°C Interna",
                    fontSize = 14.sp,
                    color = Color.White.copy(alpha = 0.6f)
                )
                Text(
                    text = "•",
                    fontSize = 14.sp,
                    color = Color.White.copy(alpha = 0.4f)
                )
                Text(
                    text = "🌙 Tutto Spento",
                    fontSize = 14.sp,
                    color = Color.White.copy(alpha = 0.6f)
                )
            }

            Spacer(modifier = Modifier.weight(1f))

            // Wake up prompt
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(6.dp),
                modifier = Modifier.padding(bottom = 32.dp)
            ) {
                Text("👆", fontSize = 12.sp)
                Text(
                    text = "Tocca lo schermo per risvegliare Domus",
                    fontSize = 12.sp,
                    color = Color.White.copy(alpha = 0.35f)
                )
            }
        }
    }
}
