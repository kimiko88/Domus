package com.domus.app.presentation.screens.climate

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import com.domus.app.domain.model.*
import com.domus.app.presentation.theme.*
import com.domus.app.presentation.viewmodel.DashboardViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PredictiveClimateScreen(
    viewModel: DashboardViewModel,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier
) {
    LaunchedEffect(Unit) {
        viewModel.loadPredictiveStatus()
    }

    val status by viewModel.predictiveStatus.collectAsState()
    val isLoading by viewModel.isPredictiveLoading.collectAsState()
    var evaluatingZoneId by remember { mutableStateOf<String?>(null) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        text = "Clima Predittivo & Inerzia",
                        fontWeight = FontWeight.Bold,
                        color = Color.White
                    )
                },
                navigationIcon = {
                    IconButton(
                        onClick = onDismiss,
                        modifier = Modifier.size(48.dp)
                    ) {
                        Text("✕", fontSize = 20.sp, color = Color.White)
                    }
                },
                actions = {
                    IconButton(
                        onClick = { viewModel.evaluateAllPredictiveZones() },
                        modifier = Modifier.size(48.dp)
                    ) {
                        Text("🔄", fontSize = 18.sp)
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = DomusSurfaceDark)
            )
        },
        containerColor = DomusBackgroundDark,
        modifier = modifier
    ) { innerPadding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Banner Eventi Meteo Attivi
            status?.let { s ->
                if (s.cold_front_active) {
                    item {
                        WeatherAlertBanner(
                            iconText = "❄️",
                            title = "Ondata di Freddo Imminente",
                            description = "Pre-riscaldamento predittivo attivato per anticipare la perdita termica dovuta al calo repentino della temperatura esterna.",
                            color = Color(0xFF06B6D4)
                        )
                    }
                }
                if (s.solar_cutoff_active) {
                    item {
                        WeatherAlertBanner(
                            iconText = "☀️",
                            title = "Spegnimento Anticipato per Irraggiamento Solare",
                            description = "Riscaldamento modulato verso il basso per sfruttare il guadagno termico passivo solare stimato sulle vetrate.",
                            color = Color(0xFFF59E0B)
                        )
                    }
                }

                item {
                    PredictiveMetricsRow(status = s)
                }
            }

            // Intestazione Sezione Zone
            item {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "Zone Clima Monitorate",
                        fontWeight = FontWeight.Bold,
                        color = Color.White,
                        fontSize = 18.sp
                    )
                    TextButton(onClick = { viewModel.evaluateAllPredictiveZones() }) {
                        Text("Ricalcola Tutte", color = AccentBlue, fontSize = 13.sp, fontWeight = FontWeight.Bold)
                    }
                }
            }

            // Lista Zone
            status?.zones?.let { zones ->
                if (zones.isEmpty()) {
                    item { EmptyZonesPlaceholder() }
                } else {
                    items(zones, key = { it.config.id }) { zoneItem ->
                        PredictiveZoneCard(
                            zone = zoneItem,
                            isEvaluating = evaluatingZoneId == zoneItem.config.id,
                            onEvaluate = {
                                evaluatingZoneId = zoneItem.config.id
                                viewModel.evaluatePredictiveZone(zoneItem.config.id) {
                                    evaluatingZoneId = null
                                }
                            }
                        )
                    }
                }
            } ?: item {
                if (isLoading) {
                    Box(modifier = Modifier.fillMaxWidth().padding(48.dp), contentAlignment = Alignment.Center) {
                        CircularProgressIndicator(color = AccentBlue)
                    }
                }
            }

            item { Spacer(modifier = Modifier.height(24.dp)) }
        }
    }
}

@Composable
private fun WeatherAlertBanner(
    iconText: String,
    title: String,
    description: String,
    color: Color
) {
    Card(
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = color.copy(alpha = 0.12f)),
        border = BorderStroke(1.dp, color.copy(alpha = 0.35f)),
        modifier = Modifier.fillMaxWidth()
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(iconText, fontSize = 24.sp)
            Spacer(modifier = Modifier.width(12.dp))
            Column {
                Text(text = title, color = Color.White, fontWeight = FontWeight.Bold, fontSize = 15.sp)
                Spacer(modifier = Modifier.height(2.dp))
                Text(text = description, color = Color.White.copy(alpha = 0.8f), fontSize = 12.sp)
            }
        }
    }
}

@Composable
private fun PredictiveMetricsRow(status: SystemPredictiveStatusDto) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(10.dp)
    ) {
        MetricTile(
            title = "Zone Attive",
            value = "${status.enabled_zones}/${status.total_zones}",
            color = AccentBlue,
            modifier = Modifier.weight(1f)
        )
        MetricTile(
            title = "Ondata Freddo",
            value = if (status.cold_front_active) "Attivo" else "Nominale",
            color = if (status.cold_front_active) Color(0xFF06B6D4) else Color(0xFF10B981),
            modifier = Modifier.weight(1f)
        )
        MetricTile(
            title = "Taglio Solare",
            value = if (status.solar_cutoff_active) "Attivo" else "Nominale",
            color = if (status.solar_cutoff_active) Color(0xFFF59E0B) else Color(0xFF10B981),
            modifier = Modifier.weight(1f)
        )
    }
}

@Composable
private fun MetricTile(title: String, value: String, color: Color, modifier: Modifier = Modifier) {
    Card(
        shape = RoundedCornerShape(14.dp),
        colors = CardDefaults.cardColors(containerColor = DomusSurfaceDark),
        border = BorderStroke(1.dp, Color.White.copy(alpha = 0.08f)),
        modifier = modifier
    ) {
        Column(
            modifier = Modifier.padding(12.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Text(text = title, color = Color.Gray, fontSize = 11.sp)
            Spacer(modifier = Modifier.height(4.dp))
            Text(text = value, color = color, fontWeight = FontWeight.Bold, fontSize = 15.sp)
        }
    }
}

@Composable
private fun PredictiveZoneCard(
    zone: ZoneDashboardStatusDto,
    isEvaluating: Boolean,
    onEvaluate: () -> Unit
) {
    val eval: PredictiveEvaluationDto? = zone.latest_evaluation
    val mode: PredictiveMode = eval?.parsedMode ?: PredictiveMode.COMFORT

    Card(
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = DomusSurfaceDark),
        border = BorderStroke(1.dp, Color.White.copy(alpha = 0.08f)),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // Header Zona: Nome e Modalità Raccomandata
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column {
                    Text(
                        text = zone.config.name,
                        fontWeight = FontWeight.Bold,
                        color = Color.White,
                        fontSize = 17.sp
                    )
                    Text(
                        text = zone.config.area_id ?: "Casa",
                        color = Color.Gray,
                        fontSize = 12.sp
                    )
                }

                Surface(
                    shape = CircleShape,
                    color = modeColor(mode).copy(alpha = 0.18f),
                    border = BorderStroke(1.dp, modeColor(mode).copy(alpha = 0.5f))
                ) {
                    Text(
                        text = modeDisplayName(mode),
                        color = modeColor(mode),
                        fontWeight = FontWeight.Bold,
                        fontSize = 11.sp,
                        modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp)
                    )
                }
            }

            HorizontalDivider(color = Color.White.copy(alpha = 0.08f))

            // Griglia Temperature: Attuale, Target, Calcolato
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                TemperatureColumn(
                    label = "Attuale",
                    value = eval?.current_temp?.let { String.format("%.1f°C", it) } ?: "--",
                    color = Color.White
                )

                Text("→", fontSize = 16.sp, color = Color.Gray, modifier = Modifier.align(Alignment.CenterVertically))

                TemperatureColumn(
                    label = "Target",
                    value = String.format("%.1f°C", zone.config.target_temperature),
                    color = AccentBlue
                )

                Text("→", fontSize = 16.sp, color = Color.Gray, modifier = Modifier.align(Alignment.CenterVertically))

                TemperatureColumn(
                    label = "Calcolato",
                    value = eval?.calculated_setpoint?.let { String.format("%.1f°C", it) } ?: "--",
                    color = if (eval != null && eval.calculated_setpoint != zone.config.target_temperature) Color(0xFFF59E0B) else Color(0xFF10B981)
                )
            }

            // Sensori Ambientali e Ragione Algoritmo
            if (eval != null) {
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                    Text(
                        text = String.format("%.1f hPa", eval.current_pressure),
                        color = Color.Gray,
                        fontSize = 12.sp
                    )
                    Text(
                        text = "${eval.humidity.toInt()}% UR",
                        color = Color.Gray,
                        fontSize = 12.sp
                    )
                    if (eval.estimated_solar_gain_kw > 0.0) {
                        Text(
                            text = String.format("+%.2f kW Sole", eval.estimated_solar_gain_kw),
                            color = Color(0xFFF59E0B),
                            fontWeight = FontWeight.SemiBold,
                            fontSize = 12.sp
                        )
                    }
                    if (eval.window_open) {
                        Text(
                            text = "Finestra Aperta",
                            color = Color(0xFFEF4444),
                            fontWeight = FontWeight.Bold,
                            fontSize = 12.sp
                        )
                    }
                }

                if (eval.reason.isNotBlank()) {
                    Text(
                        text = eval.reason,
                        color = Color.White.copy(alpha = 0.7f),
                        fontSize = 12.sp,
                        fontStyle = androidx.compose.ui.text.font.FontStyle.Italic
                    )
                }
            }

            // Bottone Valuta Ora
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) {
                Button(
                    onClick = onEvaluate,
                    enabled = !isEvaluating,
                    shape = CircleShape,
                    colors = ButtonDefaults.buttonColors(containerColor = DomusSurfaceDark),
                    border = BorderStroke(1.dp, Color.White.copy(alpha = 0.2f)),
                    modifier = Modifier.height(36.dp)
                ) {
                    if (isEvaluating) {
                        CircularProgressIndicator(color = AccentBlue, modifier = Modifier.size(16.dp))
                    } else {
                        Text("Valuta Ora", fontSize = 12.sp, fontWeight = FontWeight.Bold)
                    }
                }
            }
        }
    }
}

@Composable
private fun TemperatureColumn(label: String, value: String, color: Color) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Text(text = value, color = color, fontWeight = FontWeight.Bold, fontSize = 16.sp)
        Text(text = label, color = Color.Gray, fontSize = 11.sp)
    }
}

private fun modeColor(mode: PredictiveMode): Color {
    return when (mode) {
        PredictiveMode.COMFORT -> AccentBlue
        PredictiveMode.COLD_FRONT_BOOST -> Color(0xFF06B6D4)
        PredictiveMode.SOLAR_CUTOFF -> Color(0xFFF59E0B)
        PredictiveMode.PREHEAT -> Color(0xFFEF4444)
        PredictiveMode.ECO -> Color(0xFF10B981)
        PredictiveMode.FROST_PROTECTION -> Color(0xFF818CF8)
    }
}

private fun modeDisplayName(mode: PredictiveMode): String {
    return when (mode) {
        PredictiveMode.COMFORT -> "Comfort"
        PredictiveMode.COLD_FRONT_BOOST -> "Anticipo Freddo"
        PredictiveMode.SOLAR_CUTOFF -> "Taglio Solare"
        PredictiveMode.PREHEAT -> "Pre-riscaldamento"
        PredictiveMode.ECO -> "Eco"
        PredictiveMode.FROST_PROTECTION -> "Antigelo"
    }
}

@Composable
private fun EmptyZonesPlaceholder() {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 40.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Text(
            text = "Nessuna zona predittiva configurata",
            color = Color.White,
            fontWeight = FontWeight.Bold,
            fontSize = 16.sp
        )
        Text(
            text = "Configura le zone clima con parametri di inerzia termica dal pannello web",
            color = Color.Gray,
            fontSize = 13.sp
        )
    }
}
