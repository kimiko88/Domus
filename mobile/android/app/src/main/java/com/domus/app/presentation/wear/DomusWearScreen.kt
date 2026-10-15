package com.domus.app.presentation.wear

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.IndigoPrimary
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.theme.Slate900

/**
 * Circular Wear OS smart watch interface for rapid home automation control.
 */
@Composable
fun DomusWearScreen(
    onToggleDevice: (String) -> Unit = {}
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.Black)
            .padding(horizontal = 14.dp, vertical = 20.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        // Wear Watch Header & Complication
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.Center
        ) {
            Text("⚡", fontSize = 12.sp)
            Spacer(modifier = Modifier.width(4.dp))
            Text(
                text = "1.8 kW • 21.0°C",
                color = CyanAccent,
                fontSize = 11.sp,
                fontWeight = FontWeight.Bold
            )
        }

        Spacer(modifier = Modifier.height(10.dp))

        // Quick Wear Scrollable Tiles
        LazyColumn(
            modifier = Modifier.fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(6.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            item {
                WearItemTile("💡 Salotto", "Accesa • 80%", true) { onToggleDevice("light.salotto") }
            }
            item {
                WearItemTile("🛡️ Allarme", "Armato Notte", true) { onToggleDevice("alarm.home") }
            }
            item {
                WearItemTile("🌙 Scenario", "Buonanotte", false) { onToggleDevice("scene.goodnight") }
            }
            item {
                WearItemTile("🚪 Cancello", "Tocca x aprire", false) { onToggleDevice("cover.gate") }
            }
        }
    }
}

@Composable
private fun WearItemTile(
    title: String,
    subtitle: String,
    isActive: Boolean,
    onClick: () -> Unit
) {
    Surface(
        shape = RoundedCornerShape(16.dp),
        color = if (isActive) CyanAccent.copy(alpha = 0.2f) else Slate800,
        border = androidx.compose.foundation.BorderStroke(1.dp, if (isActive) CyanAccent else Color.DarkGray),
        modifier = Modifier
            .fillMaxWidth(0.9f)
            .clickable { onClick() }
    ) {
        Column(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Text(
                text = title,
                fontSize = 12.sp,
                fontWeight = FontWeight.Bold,
                color = Color.White
            )
            Text(
                text = subtitle,
                fontSize = 9.sp,
                color = if (isActive) CyanAccent else Color.White.copy(alpha = 0.6f)
            )
        }
    }
}
