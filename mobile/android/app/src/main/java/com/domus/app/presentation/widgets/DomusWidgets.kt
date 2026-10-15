package com.domus.app.presentation.widgets

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.presentation.theme.AmberWarning
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.RoseDanger
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.theme.Slate900

/**
 * 1. Domus Quick Controls Widget (4x2):
 * Displays 4 favorite home entities with instant one-tap background toggles.
 */
@Composable
fun DomusQuickControlsWidget(
    onToggleEntity: (String) -> Unit = {}
) {
    Card(
        shape = RoundedCornerShape(22.dp),
        colors = CardDefaults.cardColors(containerColor = Slate900),
        border = androidx.compose.foundation.BorderStroke(1.dp, Slate700),
        modifier = Modifier
            .fillMaxWidth()
            .height(130.dp)
            .padding(8.dp)
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(12.dp)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("Domus", fontWeight = FontWeight.Black, fontSize = 13.sp, color = Color.White)
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Controls", fontSize = 11.sp, color = CyanAccent)
                }
                Text("LAN Attiva • Sub-ms", fontSize = 10.sp, color = EmeraldSuccess)
            }

            Spacer(modifier = Modifier.height(8.dp))

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween
            ) {
                WidgetItemButton(icon = "💡", label = "Salotto", isOn = true) { onToggleEntity("light.salotto") }
                WidgetItemButton(icon = "💡", label = "Cucina", isOn = false) { onToggleEntity("light.cucina") }
                WidgetItemButton(icon = "🔌", label = "Presa TV", isOn = true) { onToggleEntity("switch.tv") }
                WidgetItemButton(icon = "🚪", label = "Cancello", isOn = false) { onToggleEntity("cover.gate") }
            }
        }
    }
}

/**
 * 2. Domus Alarm Widget (2x2):
 * Displays live alarm status with quick Arm/Disarm action.
 */
@Composable
fun DomusAlarmWidget(
    isArmed: Boolean = true,
    onToggleAlarm: () -> Unit = {}
) {
    Card(
        shape = RoundedCornerShape(22.dp),
        colors = CardDefaults.cardColors(containerColor = Slate900),
        border = androidx.compose.foundation.BorderStroke(1.dp, if (isArmed) RoseDanger.copy(alpha = 0.5f) else Slate700),
        modifier = Modifier
            .size(150.dp)
            .padding(8.dp)
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(14.dp),
            verticalArrangement = Arrangement.SpaceBetween
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = if (isArmed) "🛡️" else "🔓",
                    fontSize = 24.sp
                )
                Box(
                    modifier = Modifier
                        .size(8.dp)
                        .background(if (isArmed) RoseDanger else EmeraldSuccess, CircleShape)
                )
            }

            Column {
                Text(
                    text = if (isArmed) "Allarme Armato" else "Disarmato",
                    fontWeight = FontWeight.Bold,
                    fontSize = 13.sp,
                    color = Color.White
                )
                Text(
                    text = if (isArmed) "Perimetrale Notte" else "Nessun Sensore",
                    fontSize = 10.sp,
                    color = Color.White.copy(alpha = 0.5f)
                )
            }

            Button(
                onClick = onToggleAlarm,
                shape = RoundedCornerShape(8.dp),
                colors = ButtonDefaults.buttonColors(
                    containerColor = if (isArmed) RoseDanger else CyanAccent
                ),
                contentPadding = PaddingValues(0.dp),
                modifier = Modifier
                    .fillMaxWidth()
                    .height(28.dp)
            ) {
                Text(
                    text = if (isArmed) "Disarma" else "Arma",
                    fontSize = 11.sp,
                    fontWeight = FontWeight.Bold,
                    color = if (isArmed) Color.White else Color.Black
                )
            }
        }
    }
}

@Composable
private fun RowScope.WidgetItemButton(
    icon: String,
    label: String,
    isOn: Boolean,
    onClick: () -> Unit
) {
    Surface(
        shape = RoundedCornerShape(12.dp),
        color = if (isOn) CyanAccent.copy(alpha = 0.15f) else Slate800,
        border = androidx.compose.foundation.BorderStroke(1.dp, if (isOn) CyanAccent.copy(alpha = 0.5f) else Slate700),
        modifier = Modifier
            .weight(1f)
            .padding(horizontal = 3.dp)
            .clickable { onClick() }
    ) {
        Column(
            modifier = Modifier.padding(vertical = 8.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Text(icon, fontSize = 16.sp)
            Spacer(modifier = Modifier.height(2.dp))
            Text(
                text = label,
                fontSize = 10.sp,
                color = if (isOn) CyanAccent else Color.White.copy(alpha = 0.7f),
                fontWeight = if (isOn) FontWeight.Bold else FontWeight.Normal,
                maxLines = 1
            )
        }
    }
}
