package com.domus.app.presentation.screens.presence

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.domain.model.DomusNfcTag
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.IndigoPrimary
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.viewmodel.DashboardViewModel
import com.domus.app.services.notifications.DomusNotificationHelper

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PresenceAndNfcScreen(
    viewModel: DashboardViewModel,
    onDismiss: () -> Unit
) {
    val context = LocalContext.current
    var radiusSlider by remember { mutableFloatStateOf(150f) }
    var isGeofenceActive by remember { mutableStateOf(true) }
    var isVacationModeEnabled by remember { mutableStateOf(false) }
    var vacationJitter by remember { mutableIntStateOf(15) }
    var commuteEtaMinutes by remember { mutableIntStateOf(18) }
    var commuteDistanceKm by remember { mutableDoubleStateOf(12.4) }
    var commutePhase by remember { mutableStateOf("In Avvicinamento") }
    val scope = rememberCoroutineScope()
    val nfcStatusMessage by viewModel.nfcManager.lastStatusMessage.collectAsState()
    val lastReadTag by viewModel.nfcManager.lastReadTag.collectAsState()
    var feedbackBanner by remember { mutableStateOf<String?>(null) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text("Presenza, Geofence & NFC", fontWeight = FontWeight.Bold, fontSize = 20.sp)
                        Text("Automazioni di prossimità a zero consumo", fontSize = 12.sp, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f))
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onDismiss) {
                        Text("✕", fontSize = 18.sp, fontWeight = FontWeight.Bold)
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.surface)
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize()
                .padding(horizontal = 16.dp)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            feedbackBanner?.let { banner ->
                Surface(
                    shape = RoundedCornerShape(12.dp),
                    color = EmeraldSuccess.copy(alpha = 0.15f),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Row(modifier = Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
                        Text("✅", fontSize = 16.sp)
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(banner, color = EmeraldSuccess, fontWeight = FontWeight.SemiBold, fontSize = 13.sp)
                    }
                }
            }

            // SECTION 1: PRESENCE & GEOFENCING
            Card(
                shape = RoundedCornerShape(18.dp),
                colors = CardDefaults.cardColors(containerColor = Slate800),
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text("📍", fontSize = 22.sp)
                            Spacer(modifier = Modifier.width(10.dp))
                            Text("Presenza Multi-Tier & Geofence", fontWeight = FontWeight.Bold, fontSize = 15.sp, color = Color.White)
                        }

                        Box(
                            modifier = Modifier
                                .size(12.dp)
                                .background(EmeraldSuccess, CircleShape)
                        )
                    }

                    Spacer(modifier = Modifier.height(14.dp))

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text("Monitoraggio Geofence", fontSize = 14.sp, color = Color.White)
                        Switch(
                            checked = isGeofenceActive,
                            onCheckedChange = { isGeofenceActive = it },
                            colors = SwitchDefaults.colors(checkedThumbColor = Color.White, checkedTrackColor = CyanAccent)
                        )
                    }

                    Spacer(modifier = Modifier.height(10.dp))

                    Text("Raggio Perimetro Casa: ${radiusSlider.toInt()} metri", fontSize = 13.sp, color = Color.White.copy(alpha = 0.8f))
                    Slider(
                        value = radiusSlider,
                        onValueChange = { radiusSlider = it },
                        valueRange = 50f..500f,
                        steps = 17,
                        colors = SliderDefaults.colors(thumbColor = CyanAccent, activeTrackColor = CyanAccent)
                    )

                    Spacer(modifier = Modifier.height(6.dp))

                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("📶", fontSize = 16.sp)
                        Spacer(modifier = Modifier.width(6.dp))
                        Text("Tier 1: Rilevamento Wi-Fi LAN attivo (zero batteria)", fontSize = 12.sp, color = CyanAccent)
                    }
                }
            }

            // SECTION 2: CRITICAL ALERTS & NOTIFICATIONS
            Card(
                shape = RoundedCornerShape(18.dp),
                colors = CardDefaults.cardColors(containerColor = Slate800),
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("🚨", fontSize = 22.sp)
                        Spacer(modifier = Modifier.width(10.dp))
                        Text("Allarmi Critici (Bypass Non Disturbare)", fontWeight = FontWeight.Bold, fontSize = 15.sp, color = Color.White)
                    }

                    Spacer(modifier = Modifier.height(10.dp))

                    Text(
                        "Canale AudioAttributes.USAGE_ALARM configurato per suonare a volume massimo in caso di allagamento, fuga gas o intrusione perimetrale.",
                        fontSize = 12.sp,
                        color = Color.White.copy(alpha = 0.7f)
                    )

                    Spacer(modifier = Modifier.height(12.dp))

                    Button(
                        onClick = {
                            DomusNotificationHelper.sendCriticalAlarmNotification(
                                context = context,
                                title = "Test Allarme Domus",
                                message = "Sirena di test inviata su canale critico con successo."
                            )
                            feedbackBanner = "Notifica critica di test inviata!"
                        },
                        colors = ButtonDefaults.buttonColors(containerColor = Color(0xFFEF4444).copy(alpha = 0.2f), contentColor = Color(0xFFEF4444)),
                        shape = RoundedCornerShape(10.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text("🔔 Invia Test Notifica Critica", fontWeight = FontWeight.Bold, fontSize = 13.sp)
                    }
                }
            }

            // SECTION 3: NFC TAG DISPATCHER & PROGRAMMING
            Card(
                shape = RoundedCornerShape(18.dp),
                colors = CardDefaults.cardColors(containerColor = Slate800),
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("🏷️", fontSize = 22.sp)
                        Spacer(modifier = Modifier.width(10.dp))
                        Text("Tag NFC & Automazioni Rapide", fontWeight = FontWeight.Bold, fontSize = 15.sp, color = Color.White)
                    }

                    Spacer(modifier = Modifier.height(10.dp))

                    Text(
                        "Scansiona un adesivo NFC per attivare uno scenario, aprire il cancello o spegnere tutte le luci prima di dormire.",
                        fontSize = 12.sp,
                        color = Color.White.copy(alpha = 0.7f)
                    )

                    if (nfcStatusMessage.isNotBlank()) {
                        Spacer(modifier = Modifier.height(8.dp))
                        Text(nfcStatusMessage, fontSize = 12.sp, color = CyanAccent, fontWeight = FontWeight.SemiBold)
                    }

                    lastReadTag?.let { tag ->
                        Spacer(modifier = Modifier.height(10.dp))
                        Surface(shape = RoundedCornerShape(10.dp), color = Slate700, modifier = Modifier.fillMaxWidth()) {
                            Column(modifier = Modifier.padding(10.dp)) {
                                Text("Ultimo Tag: ${tag.actionType.uppercase()}", fontWeight = FontWeight.Bold, fontSize = 12.sp, color = EmeraldSuccess)
                                Text("URI: ${tag.uri}", fontSize = 11.sp, color = Color.White)
                            }
                        }
                    }

                    Spacer(modifier = Modifier.height(14.dp))

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(10.dp)
                    ) {
                        Button(
                            onClick = {
                                viewModel.nfcManager.startReading()
                            },
                            colors = ButtonDefaults.buttonColors(containerColor = IndigoPrimary),
                            shape = RoundedCornerShape(12.dp),
                            modifier = Modifier.weight(1f)
                        ) {
                            Text("📱 Leggi NFC", fontSize = 13.sp, fontWeight = FontWeight.Bold)
                        }

                        Button(
                            onClick = {
                                viewModel.nfcManager.simulateTagScan("domus://scene/relax_evening")
                                feedbackBanner = "Simulata scansione Tag: Scenario Relax Serale attivato!"
                            },
                            colors = ButtonDefaults.buttonColors(containerColor = CyanAccent.copy(alpha = 0.2f), contentColor = CyanAccent),
                            shape = RoundedCornerShape(12.dp),
                            modifier = Modifier.weight(1f)
                        ) {
                            Text("✨ Simula Scan", fontSize = 13.sp, fontWeight = FontWeight.Bold)
                        }
                    }
                }
            }

            // SECTION 4: SMART VACATION PRESENCE SIMULATOR
            Card(
                shape = RoundedCornerShape(18.dp),
                colors = CardDefaults.cardColors(containerColor = Slate800),
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text("🏖️", fontSize = 22.sp)
                            Spacer(modifier = Modifier.width(10.dp))
                            Column {
                                Text("Simulatore Presenza Vacanza", fontWeight = FontWeight.Bold, fontSize = 15.sp, color = Color.White)
                                Text("Replay abitudini reali luci & tapparelle", fontSize = 11.sp, color = Color.White.copy(alpha = 0.6f))
                            }
                        }

                        Switch(
                            checked = isVacationModeEnabled,
                            onCheckedChange = { enabled ->
                                isVacationModeEnabled = enabled
                                feedbackBanner = if (enabled) "Modalità Vacanza attiva (±15 min jitter stocastico)" else "Modalità Vacanza disattivata"
                            },
                            colors = SwitchDefaults.colors(
                                checkedThumbColor = Color.White,
                                checkedTrackColor = EmeraldSuccess,
                                uncheckedThumbColor = Color.White.copy(alpha = 0.6f),
                                uncheckedTrackColor = Slate700
                            )
                        )
                    }

                    Spacer(modifier = Modifier.height(10.dp))

                    Surface(
                        shape = RoundedCornerShape(10.dp),
                        color = Slate700,
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Row(modifier = Modifier.padding(10.dp), verticalAlignment = Alignment.CenterVertically) {
                            Text("🎲", fontSize = 16.sp)
                            Spacer(modifier = Modifier.width(8.dp))
                            Text(
                                "Jitter casuale: ±$vacationJitter min per simulare presenza umana reale",
                                fontSize = 11.sp,
                                color = Color.White.copy(alpha = 0.8f)
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(10.dp))

                    Text("Routine Replay Configurate:", fontSize = 12.sp, fontWeight = FontWeight.SemiBold, color = CyanAccent)
                    Spacer(modifier = Modifier.height(4.dp))
                    Text("• 20:30 (±15m): Accensione Luce Salotto (light.living_room)", fontSize = 11.sp, color = Color.White.copy(alpha = 0.7f))
                    Text("• 22:15 (±15m): Chiusura Tapparelle Notte (cover.bedroom)", fontSize = 11.sp, color = Color.White.copy(alpha = 0.7f))
                    Text("• 23:00 (±15m): Spegnimento Luci Cucina (light.kitchen)", fontSize = 11.sp, color = Color.White.copy(alpha = 0.7f))
                }
            }

            // SECTION 5: COMMUTE TRACKER & ETA PRE-CLIMATE
            Card(
                shape = RoundedCornerShape(18.dp),
                colors = CardDefaults.cardColors(containerColor = Slate800),
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text("🚗", fontSize = 22.sp)
                            Spacer(modifier = Modifier.width(10.dp))
                            Column {
                                Text("Pendolarismo & ETA Casa", fontWeight = FontWeight.Bold, fontSize = 15.sp, color = Color.White)
                                Text("Fase: $commutePhase", fontSize = 11.sp, color = CyanAccent)
                            }
                        }

                        Surface(
                            shape = RoundedCornerShape(8.dp),
                            color = EmeraldSuccess.copy(alpha = 0.2f)
                        ) {
                            Text(
                                text = "$commuteEtaMinutes min",
                                color = EmeraldSuccess,
                                fontWeight = FontWeight.Bold,
                                fontSize = 14.sp,
                                modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(8.dp))

                    Text(
                        text = "Distanza stimata: $commuteDistanceKm km • Pre-climatizzazione automatica inserita",
                        fontSize = 11.sp,
                        color = Color.White.copy(alpha = 0.7f)
                    )
                }
            }

            Spacer(modifier = Modifier.height(20.dp))
        }
    }
}
