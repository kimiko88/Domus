package com.domus.app.presentation.screens.climate

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.R
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
                        text = stringResource(R.string.predictive_climate_title),
                        fontWeight = FontWeight.Bold,
                        color = Color.White
                    )
                },
                navigationIcon = {
                    IconButton(
                        onClick = onDismiss,
                        modifier = Modifier.size(48.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Close,
                            contentDescription = stringResource(R.string.common_close),
                            tint = Color.White
                        )
                    }
                },
                actions = {
                    IconButton(
                        onClick = { viewModel.evaluateAllPredictiveZones() },
                        modifier = Modifier.size(48.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Refresh,
                            contentDescription = stringResource(R.string.climate_evaluate_all),
                            tint = AccentBlue
                        )
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
                            icon = Icons.Default.Warning,
                            title = stringResource(R.string.climate_cold_front_alert_title),
                            description = stringResource(R.string.climate_cold_front_alert_desc),
                            color = Color(0xFF06B6D4)
                        )
                    }
                }

                if (s.solar_cutoff_active) {
                    item {
                        WeatherAlertBanner(
                            icon = Icons.Default.Star,
                            title = stringResource(R.string.climate_solar_cutoff_alert_title),
                            description = stringResource(R.string.climate_solar_cutoff_alert_desc),
                            color = Color(0xFFF59E0B)
                        )
                    }
                }

                // Header Metriche
                item {
                    PredictiveMetricsRow(status = s)
                }

                // Elenco Zone Termiche
                item {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text(
                            text = stringResource(R.string.climate_zones_header),
                            fontWeight = FontWeight.Bold,
                            color = Color.White,
                            fontSize = 18.sp
                        )

                        TextButton(
                            onClick = { viewModel.evaluateAllPredictiveZones() }
                        ) {
                            Text(stringResource(R.string.climate_evaluate_all), color = AccentBlue)
                        }
                    }
                }

                if (s.zones.isEmpty()) {
                    item {
                        EmptyZonesPlaceholder()
                    }
                } else {
                    items(s.zones, key = { it.config.id }) { zoneItem ->
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
    icon: androidx.compose.ui.graphics.vector.ImageVector,
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
            Icon(imageVector = icon, contentDescription = null, tint = color, modifier = Modifier.size(28.dp))
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
            title = stringResource(R.string.climate_metric_zones),
            value = "${status.enabled_zones}/${status.total_zones}",
            color = AccentBlue,
            modifier = Modifier.weight(1f)
        )
        MetricTile(
            title = stringResource(R.string.climate_metric_cold_front),
            value = if (status.cold_front_active) stringResource(R.string.common_active) else stringResource(R.string.common_nominal),
            color = if (status.cold_front_active) Color(0xFF06B6D4) else Color(0xFF10B981),
            modifier = Modifier.weight(1f)
        )
        MetricTile(
            title = stringResource(R.string.climate_metric_solar_cutoff),
            value = if (status.solar_cutoff_active) stringResource(R.string.common_active) else stringResource(R.string.common_nominal),
            color = if (status.solar_cutoff_active) Color(0xFFF59E0B) else Color(0xFF10B981),
            modifier = Modifier.weight(1f)
        )
    }
}

@Composable
private fun MetricTile(
    title: String,
    value: String,
    color: Color,
    modifier: Modifier = Modifier
) {
    Card(
        shape = RoundedCornerShape(14.dp),
        colors = CardDefaults.cardColors(containerColor = DomusSurfaceDark),
        border = BorderStroke(1.dp, Color.White.copy(alpha = 0.08f)),
        modifier = modifier
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Text(text = value, color = color, fontWeight = FontWeight.Bold, fontSize = 16.sp)
            Spacer(modifier = Modifier.height(2.dp))
            Text(text = title, color = Color.Gray, fontSize = 11.sp, maxLines = 1)
        }
    }
}

@Composable
private fun PredictiveZoneCard(
    zone: ZoneDashboardStatusDto,
    isEvaluating: Boolean,
    onEvaluate: () -> Unit
) {
    val eval = zone.latest_evaluation
    val mode = eval?.parsedMode ?: PredictiveMode.COMFORT

    Card(
        shape = RoundedCornerShape(18.dp),
        colors = CardDefaults.cardColors(containerColor = DomusSurfaceDark),
        border = BorderStroke(1.dp, Color.White.copy(alpha = 0.08f)),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            // Header: Nome e Pill Modalità
            Row(verticalAlignment = Alignment.Top) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = zone.config.name,
                        fontWeight = FontWeight.Bold,
                        color = Color.White,
                        fontSize = 16.sp
                    )
                    Text(
                        text = "${zone.config.orientation.uppercase()} • ${zone.config.emitter_type} • ${zone.config.thermal_inertia_hours}h inerzia",
                        color = Color.Gray,
                        fontSize = 12.sp
                    )
                }

                Surface(
                    shape = CircleShape,
                    color = modeColor(mode).copy(alpha = 0.18f),
                    border = BorderStroke(1.dp, modeColor(mode).copy(alpha = 0.35f))
                ) {
                    Text(
                        text = mode.value.replace("_", " ").uppercase(),
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
                    label = stringResource(R.string.climate_temp_current),
                    value = eval?.let { String.format("%.1f°C", it.current_temp) } ?: "--",
                    color = Color.White
                )

                Icon(
                    imageVector = Icons.Default.ArrowForward,
                    contentDescription = null,
                    tint = Color.Gray,
                    modifier = Modifier.align(Alignment.CenterVertically)
                )

                TemperatureColumn(
                    label = stringResource(R.string.climate_temp_target),
                    value = String.format("%.1f°C", zone.config.target_temperature),
                    color = AccentBlue
                )

                Icon(
                    imageVector = Icons.Default.ArrowForward,
                    contentDescription = null,
                    tint = Color.Gray,
                    modifier = Modifier.align(Alignment.CenterVertically)
                )

                TemperatureColumn(
                    label = stringResource(R.string.climate_temp_calculated),
                    value = eval?.let { String.format("%.1f°C", it.calculated_setpoint) } ?: "--",
                    color = if (eval != null && eval.calculated_setpoint != zone.config.target_temperature) Color(0xFFF59E0B) else Color(0xFF10B981)
                )
            }

            // Sensori Ambientali e Ragione Algoritmo
            eval?.let { e ->
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                    Text(
                        text = String.format("%.1f hPa", e.current_pressure),
                        color = Color.Gray,
                        fontSize = 12.sp
                    )
                    Text(
                        text = "${e.humidity.toInt()}% UR",
                        color = Color.Gray,
                        fontSize = 12.sp
                    )
                    if (e.estimated_solar_gain_kw > 0) {
                        Text(
                            text = String.format("+%.2f kW Sole", e.estimated_solar_gain_kw),
                            color = Color(0xFFF59E0B),
                            fontWeight = FontWeight.SemiBold,
                            fontSize = 12.sp
                        )
                    }
                    if (e.window_open) {
                        Text(
                            text = stringResource(R.string.climate_window_open),
                            color = Color(0xFFEF4444),
                            fontWeight = FontWeight.Bold,
                            fontSize = 12.sp
                        )
                    }
                }

                if (e.reason.isNotBlank()) {
                    Text(
                        text = e.reason,
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
                        Text(stringResource(R.string.climate_action_evaluate), fontSize = 12.sp, fontWeight = FontWeight.Bold)
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
            text = stringResource(R.string.climate_empty_zones),
            color = Color.White,
            fontWeight = FontWeight.Bold,
            fontSize = 16.sp
        )
        Text(
            text = stringResource(R.string.climate_empty_zones_subtitle),
            color = Color.Gray,
            fontSize = 13.sp
        )
    }
}
