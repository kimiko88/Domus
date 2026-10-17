package com.domus.app.presentation.screens.socket

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.domain.model.Entity
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SocketControlSheet(
    entity: Entity,
    onToggle: (String) -> Unit,
    onDismiss: () -> Unit
) {
    val isOn = entity.isOn
    val powerWatts = if (isOn) (entity.powerWatts ?: 65.0) else 0.0
    val energyKWh = entity.energyKWh ?: 1.42

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        containerColor = MaterialTheme.colorScheme.surface,
        dragHandle = { BottomSheetDefaults.DragHandle() }
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 24.dp)
                .padding(bottom = 32.dp)
                .verticalScroll(rememberScrollState()),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Icon Glow
            Box(
                modifier = Modifier
                    .size(90.dp)
                    .background(
                        color = if (isOn) EmeraldSuccess.copy(alpha = 0.15f) else Color.Gray.copy(alpha = 0.1f),
                        shape = CircleShape
                    ),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = "🔌",
                    fontSize = 40.sp
                )
            }

            Spacer(modifier = Modifier.height(16.dp))

            Text(
                text = entity.name,
                fontWeight = FontWeight.Bold,
                fontSize = 22.sp,
                color = MaterialTheme.colorScheme.onSurface
            )

            Text(
                text = if (isOn) "Presa Attiva • In erogazione" else "Presa Spenta • Carico disconnesso",
                fontSize = 14.sp,
                color = if (isOn) EmeraldSuccess else MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
            )

            Spacer(modifier = Modifier.height(24.dp))

            // Power button toggle
            Button(
                onClick = { onToggle(entity.id) },
                colors = ButtonDefaults.buttonColors(
                    containerColor = if (isOn) Color(0xFFEF4444).copy(alpha = 0.15f) else EmeraldSuccess.copy(alpha = 0.15f),
                    contentColor = if (isOn) Color(0xFFEF4444) else EmeraldSuccess
                ),
                shape = RoundedCornerShape(16.dp),
                modifier = Modifier
                    .fillMaxWidth()
                    .height(52.dp)
            ) {
                Text(
                    text = if (isOn) "Disattiva Presa" else "Attiva Presa",
                    fontWeight = FontWeight.Bold,
                    fontSize = 16.sp
                )
            }

            Spacer(modifier = Modifier.height(24.dp))

            // Metrics Row
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                MetricCard(
                    title = "Potenza Istantanea",
                    value = String.format("%.1f", powerWatts),
                    unit = "W",
                    icon = "⚡",
                    accentColor = Color(0xFFFBBF24),
                    modifier = Modifier.weight(1f)
                )

                MetricCard(
                    title = "Energia Oggi",
                    value = String.format("%.2f", energyKWh),
                    unit = "kWh",
                    icon = "🍃",
                    accentColor = EmeraldSuccess,
                    modifier = Modifier.weight(1f)
                )
            }

            Spacer(modifier = Modifier.height(24.dp))

            // Hourly Consumption Preview
            Text(
                text = "Consumo Ultime 6 Ore",
                fontWeight = FontWeight.SemiBold,
                fontSize = 15.sp,
                color = MaterialTheme.colorScheme.onSurface,
                modifier = Modifier.fillMaxWidth()
            )

            Spacer(modifier = Modifier.height(12.dp))

            HourlyPowerBars(baseWatts = powerWatts)
        }
    }
}

@Composable
fun MetricCard(
    title: String,
    value: String,
    unit: String,
    icon: String,
    accentColor: Color,
    modifier: Modifier = Modifier
) {
    Card(
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = Slate800),
        modifier = modifier
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween,
                modifier = Modifier.fillMaxWidth()
            ) {
                Text(text = icon, fontSize = 18.sp)
                Text(text = title, fontSize = 12.sp, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f))
            }
            Spacer(modifier = Modifier.height(8.dp))
            Row(verticalAlignment = Alignment.Bottom) {
                Text(
                    text = value,
                    fontWeight = FontWeight.Black,
                    fontSize = 24.sp,
                    color = accentColor
                )
                Spacer(modifier = Modifier.width(4.dp))
                Text(
                    text = unit,
                    fontSize = 13.sp,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f),
                    modifier = Modifier.padding(bottom = 3.dp)
                )
            }
        }
    }
}

@Composable
fun HourlyPowerBars(baseWatts: Double) {
    val hours = listOf("02:00", "06:00", "10:00", "14:00", "18:00", "Ora")
    val multipliers = listOf(0.2f, 0.1f, 0.8f, 0.95f, 1.1f, 1.0f)

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(120.dp)
            .background(Slate800, RoundedCornerShape(16.dp))
            .padding(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.Bottom
    ) {
        hours.zip(multipliers).forEach { (hour, mult) ->
            val w = baseWatts * mult
            val heightPercent = if (baseWatts > 0) (w / (baseWatts * 1.3)).coerceIn(0.1, 1.0) else 0.1

            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Bottom,
                modifier = Modifier.fillMaxHeight()
            ) {
                Box(
                    modifier = Modifier
                        .width(18.dp)
                        .weight(1f, fill = false)
                        .fillMaxHeight(heightPercent.toFloat())
                        .background(
                            color = if (hour == "Ora") EmeraldSuccess else CyanAccent.copy(alpha = 0.6f),
                            shape = RoundedCornerShape(topStart = 6.dp, topEnd = 6.dp)
                        )
                )
                Spacer(modifier = Modifier.height(6.dp))
                Text(text = hour, fontSize = 10.sp, color = Color.White.copy(alpha = 0.6f))
            }
        }
    }
}
