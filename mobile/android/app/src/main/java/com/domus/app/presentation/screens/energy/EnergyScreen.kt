package com.domus.app.presentation.screens.energy

import androidx.compose.animation.core.*
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.presentation.theme.AmberWarning
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.IndigoPrimary
import com.domus.app.presentation.theme.RoseDanger
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.theme.Slate900

data class HeavyConsumer(
    val id: String,
    val name: String,
    val powerWatts: Int,
    val priority: Int,
    var isEnabled: Boolean = true
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EnergyScreen(
    onDismiss: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    // Energy nodes metrics
    var solarWatts by remember { mutableStateOf(3420) }
    var homeWatts by remember { mutableStateOf(2150) }
    var batteryWatts by remember { mutableStateOf(850) }
    var batteryPercentage by remember { mutableStateOf(84) }
    var gridWatts by remember { mutableStateOf(-420) } // Negative = Exporting to grid

    var contractLimitKw by remember { mutableStateOf(4.5f) }
    val punTariffEur = 0.128 // €/kWh PUN attuale

    val consumers = remember {
        mutableStateListOf(
            HeavyConsumer("1", "Pompa di Calore (HVAC)", 1250, 1, isEnabled = true),
            HeavyConsumer("2", "Colonnina Wallbox Auto", 2300, 4, isEnabled = false),
            HeavyConsumer("3", "Forno Elettrico Cucina", 1800, 2, isEnabled = false),
            HeavyConsumer("4", "Lavatrice con Asciugatrice", 900, 3, isEnabled = true)
        )
    }

    // Animated energy particles line effect
    val infiniteTransition = rememberInfiniteTransition(label = "energy_flow")
    val flowPhase by infiniteTransition.animateFloat(
        initialValue = 0f,
        targetValue = 40f,
        animationSpec = infiniteRepeatable(
            animation = tween(1200, easing = LinearEasing),
            repeatMode = RepeatMode.Restart
        ),
        label = "phase"
    )

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(Slate900)
    ) {
        // Top Header
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            IconButton(onClick = onDismiss) {
                Text("✕", fontSize = 20.sp, color = Color.White)
            }

            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = "Flussi di Energia",
                    fontWeight = FontWeight.Bold,
                    fontSize = 18.sp,
                    color = Color.White
                )
                Text(
                    text = "Autoconsumo: 78% • 100% Green",
                    fontSize = 12.sp,
                    color = EmeraldSuccess
                )
            }

            // Real-time PUN Price Badge
            Surface(
                shape = RoundedCornerShape(12.dp),
                color = CyanAccent.copy(alpha = 0.15f),
                border = BorderStroke(1.dp, CyanAccent)
            ) {
                Text(
                    text = "${punTariffEur} €/kWh",
                    color = CyanAccent,
                    fontWeight = FontWeight.Bold,
                    fontSize = 12.sp,
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                )
            }
        }

        // Energy Flow Diagram Box
        Card(
            shape = RoundedCornerShape(20.dp),
            colors = CardDefaults.cardColors(containerColor = Slate800),
            border = BorderStroke(1.dp, Slate700),
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 6.dp)
        ) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(210.dp)
                    .padding(16.dp)
            ) {
                // Background Flow Lines
                Canvas(modifier = Modifier.fillMaxSize()) {
                    val w = size.width
                    val h = size.height
                    val center = Offset(w / 2f, h / 2f)

                    val topNode = Offset(w / 2f, 24f)
                    val bottomNode = Offset(w / 2f, h - 24f)
                    val leftNode = Offset(36f, h / 2f)
                    val rightNode = Offset(w - 36f, h / 2f)

                    val dashEffect = PathEffect.dashPathEffect(floatArrayOf(12f, 8f), flowPhase)

                    // Solar to Center
                    drawLine(
                        color = AmberWarning,
                        start = topNode,
                        end = center,
                        strokeWidth = 3f,
                        pathEffect = dashEffect,
                        cap = StrokeCap.Round
                    )

                    // Center to Home
                    drawLine(
                        color = CyanAccent,
                        start = center,
                        end = bottomNode,
                        strokeWidth = 3f,
                        pathEffect = dashEffect,
                        cap = StrokeCap.Round
                    )

                    // Battery to Center
                    drawLine(
                        color = EmeraldSuccess,
                        start = leftNode,
                        end = center,
                        strokeWidth = 3f,
                        pathEffect = dashEffect,
                        cap = StrokeCap.Round
                    )

                    // Center to Grid
                    drawLine(
                        color = IndigoPrimary,
                        start = center,
                        end = rightNode,
                        strokeWidth = 3f,
                        pathEffect = dashEffect,
                        cap = StrokeCap.Round
                    )
                }

                // Solar Node (Top)
                EnergyNode(
                    icon = "☀️",
                    title = "Solare",
                    value = "${solarWatts} W",
                    color = AmberWarning,
                    modifier = Modifier.align(Alignment.TopCenter)
                )

                // Battery Node (Left)
                EnergyNode(
                    icon = "🔋",
                    title = "Batteria ($batteryPercentage%)",
                    value = "+${batteryWatts} W",
                    color = EmeraldSuccess,
                    modifier = Modifier.align(Alignment.CenterStart)
                )

                // Center Node Hub
                Surface(
                    shape = CircleShape,
                    color = Slate900,
                    border = BorderStroke(2.dp, CyanAccent),
                    modifier = Modifier
                        .size(46.dp)
                        .align(Alignment.Center)
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Text("⚡", fontSize = 18.sp)
                    }
                }

                // Grid Node (Right)
                EnergyNode(
                    icon = "🌐",
                    title = "Rete (Export)",
                    value = "${gridWatts} W",
                    color = IndigoPrimary,
                    modifier = Modifier.align(Alignment.CenterEnd)
                )

                // Home Load Node (Bottom)
                EnergyNode(
                    icon = "🏠",
                    title = "Carico Casa",
                    value = "${homeWatts} W",
                    color = CyanAccent,
                    modifier = Modifier.align(Alignment.BottomCenter)
                )
            }
        }

        // Peak Load Shedding Contract Slider
        Card(
            shape = RoundedCornerShape(16.dp),
            colors = CardDefaults.cardColors(containerColor = Slate800),
            border = BorderStroke(1.dp, Slate700),
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 6.dp)
        ) {
            Column(modifier = Modifier.padding(14.dp)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "Soglia Anti-Distacco (Load Shedding)",
                        fontWeight = FontWeight.Bold,
                        color = Color.White,
                        fontSize = 14.sp
                    )
                    Text(
                        text = String.format("%.1f kW", contractLimitKw),
                        color = CyanAccent,
                        fontWeight = FontWeight.Bold,
                        fontSize = 14.sp
                    )
                }

                Slider(
                    value = contractLimitKw,
                    onValueChange = { contractLimitKw = it },
                    valueRange = 2.0f..6.5f,
                    steps = 8,
                    colors = SliderDefaults.colors(
                        thumbColor = CyanAccent,
                        activeTrackColor = CyanAccent,
                        inactiveTrackColor = Slate700
                    )
                )

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    Text("2.0 kW (Base)", fontSize = 11.sp, color = Color.White.copy(alpha = 0.5f))
                    Text("4.5 kW (Standard)", fontSize = 11.sp, color = Color.White.copy(alpha = 0.5f))
                    Text("6.5 kW (Potenza Max)", fontSize = 11.sp, color = Color.White.copy(alpha = 0.5f))
                }
            }
        }

        // Prioritized Heavy Loads Table
        Text(
            text = "CARICHI PRIORITARI INTELLIGENTI",
            fontSize = 11.sp,
            fontWeight = FontWeight.Bold,
            color = CyanAccent,
            modifier = Modifier.padding(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 4.dp)
        )

        LazyColumn(
            modifier = Modifier
                .fillMaxWidth()
                .weight(1f)
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
            contentPadding = PaddingValues(bottom = 16.dp)
        ) {
            items(consumers, key = { it.id }) { consumer ->
                Card(
                    shape = RoundedCornerShape(12.dp),
                    colors = CardDefaults.cardColors(containerColor = Slate800),
                    border = BorderStroke(1.dp, Slate700),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(12.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Column {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text(
                                    text = consumer.name,
                                    fontWeight = FontWeight.Bold,
                                    color = Color.White,
                                    fontSize = 14.sp
                                )
                                Spacer(modifier = Modifier.width(6.dp))
                                Surface(
                                    shape = RoundedCornerShape(4.dp),
                                    color = Slate700
                                ) {
                                    Text(
                                        text = "P${consumer.priority}",
                                        fontSize = 10.sp,
                                        color = Color.White,
                                        modifier = Modifier.padding(horizontal = 4.dp, vertical = 2.dp)
                                    )
                                }
                            }
                            Text(
                                text = "Assorbimento: ${consumer.powerWatts} W",
                                fontSize = 12.sp,
                                color = if (consumer.isEnabled) CyanAccent else Color.White.copy(alpha = 0.4f)
                            )
                        }

                        Switch(
                            checked = consumer.isEnabled,
                            onCheckedChange = {
                                val idx = consumers.indexOfFirst { c -> c.id == consumer.id }
                                if (idx != -1) {
                                    consumers[idx] = consumer.copy(isEnabled = it)
                                }
                            },
                            colors = SwitchDefaults.colors(
                                checkedThumbColor = Color.White,
                                checkedTrackColor = CyanAccent
                            )
                        )
                    }
                }
            }
        }
    }
}

@Composable
fun EnergyNode(
    icon: String,
    title: String,
    value: String,
    color: Color,
    modifier: Modifier = Modifier
) {
    Surface(
        shape = RoundedCornerShape(12.dp),
        color = Slate900.copy(alpha = 0.9f),
        border = BorderStroke(1.dp, color.copy(alpha = 0.5f)),
        modifier = modifier
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(icon, fontSize = 14.sp)
            Spacer(modifier = Modifier.width(6.dp))
            Column {
                Text(title, fontSize = 10.sp, color = Color.White.copy(alpha = 0.6f))
                Text(value, fontSize = 12.sp, fontWeight = FontWeight.Bold, color = color)
            }
        }
    }
}
