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
import com.domus.app.presentation.viewmodel.DashboardViewModel

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
    viewModel: DashboardViewModel? = null,
    onDismiss: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    LaunchedEffect(Unit) {
        viewModel?.loadEnergyFlow()
    }

    val serverFlow by viewModel?.energyFlow?.collectAsState() ?: remember { mutableStateOf(null) }

    // Energy nodes metrics
    val solarWatts = serverFlow?.solar_watts?.toInt() ?: 0
    val homeWatts = serverFlow?.house_consumption_watts?.toInt() ?: 0
    val batteryWatts = serverFlow?.battery_watts?.toInt() ?: 0
    val batteryPercentage = serverFlow?.battery_soc?.toInt() ?: 0
    val gridWatts = serverFlow?.grid_watts?.toInt() ?: 0 // Negative = Exporting to grid

    var contractLimitKw by remember { mutableFloatStateOf(4.5f) }
    val punTariffEur = 0.128 // €/kWh PUN attuale

    // EV Wallbox OCPP Smart Charging State
    var wallboxMode by remember { mutableStateOf("SOLAR_OPTIMIZED") }
    var wallboxPowerW by remember { mutableDoubleStateOf(0.0) }
    var wallboxAmps by remember { mutableDoubleStateOf(0.0) }
    var wallboxSoc by remember { mutableStateOf<Int?>(72) }
    var wallboxKwh by remember { mutableDoubleStateOf(14.8) }
    var wallboxStatus by remember { mutableStateOf("SuspendedEV") }

    val entities by viewModel?.filteredEntities?.collectAsState() ?: remember { mutableStateOf(emptyList()) }
    val consumers = remember(entities) {
        entities.filter { it.domain == "switch" || it.domain == "climate" || it.domain == "heater" }.take(6).mapIndexed { i, e ->
            val power = e.attributes["current_power_w"]?.toString()?.replace("\"", "")?.toDoubleOrNull()?.toInt()
                ?: e.attributes["power"]?.toString()?.replace("\"", "")?.toDoubleOrNull()?.toInt()
                ?: (if (e.isOn) 1200 else 0)
            HeavyConsumer(
                id = e.id,
                name = e.name,
                powerWatts = power,
                priority = i + 1,
                isEnabled = e.isOn
            )
        }
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

        LazyColumn(
            modifier = Modifier
                .fillMaxWidth()
                .weight(1f)
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
            contentPadding = PaddingValues(bottom = 16.dp)
        ) {
            // Energy Flow Diagram Box
            item {
                Card(
                    shape = RoundedCornerShape(20.dp),
                    colors = CardDefaults.cardColors(containerColor = Slate800),
                    border = BorderStroke(1.dp, Slate700),
                    modifier = Modifier.fillMaxWidth()
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
            }

            // EV Wallbox OCPP Smart Charging Card
            item {
                Card(
                    shape = RoundedCornerShape(16.dp),
                    colors = CardDefaults.cardColors(containerColor = Slate800),
                    border = BorderStroke(1.dp, Slate700),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Column(modifier = Modifier.padding(14.dp)) {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text("⚡", fontSize = 16.sp)
                                Spacer(modifier = Modifier.width(6.dp))
                                Column {
                                    Text(
                                        text = "Wallbox EV (OCPP 1.6J)",
                                        fontWeight = FontWeight.Bold,
                                        color = Color.White,
                                        fontSize = 14.sp
                                    )
                                    Text(
                                        text = "Ricarica Solare Dinamica",
                                        fontSize = 11.sp,
                                        color = Color.White.copy(alpha = 0.6f)
                                    )
                                }
                            }

                            val isCharging = wallboxStatus == "Charging"
                            val statusText = when (wallboxStatus) {
                                "Charging" -> "In Carica"
                                "SuspendedEV" -> "In Attesa Veicolo"
                                "Preparing" -> "Cavo Connesso"
                                else -> "Disponibile"
                            }
                            val statusColor = if (isCharging) EmeraldSuccess else CyanAccent

                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = statusColor.copy(alpha = 0.15f),
                                border = BorderStroke(1.dp, statusColor.copy(alpha = 0.5f))
                            ) {
                                Text(
                                    text = statusText,
                                    fontSize = 11.sp,
                                    fontWeight = FontWeight.Bold,
                                    color = statusColor,
                                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp)
                                )
                            }
                        }

                        Spacer(modifier = Modifier.height(12.dp))

                        // Telemetry Grid
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween
                        ) {
                            Column(
                                modifier = Modifier
                                    .weight(1f)
                                    .background(Slate900, RoundedCornerShape(8.dp))
                                    .padding(8.dp),
                                horizontalAlignment = Alignment.CenterHorizontally
                            ) {
                                Text("Potenza", fontSize = 10.sp, color = Color.White.copy(alpha = 0.6f))
                                Text(String.format("%.1f kW", wallboxPowerW / 1000.0), fontSize = 13.sp, fontWeight = FontWeight.Bold, color = AmberWarning)
                            }
                            Spacer(modifier = Modifier.width(6.dp))
                            Column(
                                modifier = Modifier
                                    .weight(1f)
                                    .background(Slate900, RoundedCornerShape(8.dp))
                                    .padding(8.dp),
                                horizontalAlignment = Alignment.CenterHorizontally
                            ) {
                                Text("Corrente", fontSize = 10.sp, color = Color.White.copy(alpha = 0.6f))
                                Text(String.format("%.0f A", wallboxAmps), fontSize = 13.sp, fontWeight = FontWeight.Bold, color = CyanAccent)
                            }
                            Spacer(modifier = Modifier.width(6.dp))
                            Column(
                                modifier = Modifier
                                    .weight(1f)
                                    .background(Slate900, RoundedCornerShape(8.dp))
                                    .padding(8.dp),
                                horizontalAlignment = Alignment.CenterHorizontally
                            ) {
                                Text("Batteria EV", fontSize = 10.sp, color = Color.White.copy(alpha = 0.6f))
                                Text("${wallboxSoc ?: "--"}%", fontSize = 13.sp, fontWeight = FontWeight.Bold, color = EmeraldSuccess)
                            }
                            Spacer(modifier = Modifier.width(6.dp))
                            Column(
                                modifier = Modifier
                                    .weight(1f)
                                    .background(Slate900, RoundedCornerShape(8.dp))
                                    .padding(8.dp),
                                horizontalAlignment = Alignment.CenterHorizontally
                            ) {
                                Text("Sessione", fontSize = 10.sp, color = Color.White.copy(alpha = 0.6f))
                                Text(String.format("%.1f kWh", wallboxKwh), fontSize = 13.sp, fontWeight = FontWeight.Bold, color = Color.White)
                            }
                        }

                        Spacer(modifier = Modifier.height(12.dp))

                        // Solar Charging Mode Selector
                        Text(
                            text = "MODALITÀ DI RICARICA",
                            fontSize = 10.sp,
                            fontWeight = FontWeight.Bold,
                            color = Color.White.copy(alpha = 0.6f)
                        )
                        Spacer(modifier = Modifier.height(6.dp))

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(6.dp)
                        ) {
                            val modes = listOf(
                                "SOLAR_ONLY" to "☀️ Solare",
                                "SOLAR_OPTIMIZED" to "⚡ Smart",
                                "FAST" to "🚀 Rapida",
                                "NIGHT" to "🌙 Notte"
                            )
                            modes.forEach { (modeKey, label) ->
                                val isSelected = wallboxMode == modeKey
                                Surface(
                                    shape = RoundedCornerShape(8.dp),
                                    color = if (isSelected) EmeraldSuccess else Slate700,
                                    modifier = Modifier
                                        .weight(1f)
                                        .clickable { wallboxMode = modeKey }
                                ) {
                                    Text(
                                        text = label,
                                        fontSize = 11.sp,
                                        fontWeight = if (isSelected) FontWeight.Bold else FontWeight.Normal,
                                        color = if (isSelected) Color.White else Color.White.copy(alpha = 0.8f),
                                        modifier = Modifier
                                            .padding(vertical = 8.dp)
                                            .wrapContentWidth(Alignment.CenterHorizontally)
                                    )
                                }
                            }
                        }
                    }
                }
            }

            // Peak Load Shedding Contract Slider
            item {
                Card(
                    shape = RoundedCornerShape(16.dp),
                    colors = CardDefaults.cardColors(containerColor = Slate800),
                    border = BorderStroke(1.dp, Slate700),
                    modifier = Modifier.fillMaxWidth()
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
            }

            // Prioritized Heavy Loads Section Title
            item {
                Text(
                    text = "CARICHI PRIORITARI INTELLIGENTI",
                    fontSize = 11.sp,
                    fontWeight = FontWeight.Bold,
                    color = CyanAccent,
                    modifier = Modifier.padding(top = 8.dp, bottom = 4.dp)
                )
            }
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
                                viewModel?.toggleEntity(consumer.id)
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
