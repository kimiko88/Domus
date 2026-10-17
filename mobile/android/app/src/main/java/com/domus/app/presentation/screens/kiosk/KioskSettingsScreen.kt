package com.domus.app.presentation.screens.kiosk

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.viewmodel.DashboardViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun KioskSettingsScreen(
    viewModel: DashboardViewModel,
    onDismiss: () -> Unit
) {
    val isKioskEnabled by viewModel.isKioskModeEnabled.collectAsState()
    val isBatteryProtectionEnabled by viewModel.isBatteryProtectionEnabled.collectAsState()
    val smartPlugId by viewModel.kioskSmartPlugId.collectAsState()
    var plugInput by remember(smartPlugId) { mutableStateOf(smartPlugId) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text("Modalità Kiosk", fontWeight = FontWeight.Bold, fontSize = 20.sp)
                        Text("Impostazioni per tablet a parete always-on", fontSize = 12.sp, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f))
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
            Spacer(modifier = Modifier.height(4.dp))

            // SECTION 1: KIOSK DISPLAY PINNING
            Card(
                shape = RoundedCornerShape(18.dp),
                colors = CardDefaults.cardColors(containerColor = Slate800),
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("📟", fontSize = 22.sp)
                        Spacer(modifier = Modifier.width(10.dp))
                        Text("Schermo Sempre Attivo a Parete", fontWeight = FontWeight.Bold, fontSize = 15.sp, color = Color.White)
                    }

                    Spacer(modifier = Modifier.height(14.dp))

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text("Modalità Kiosk (Lock-Task)", fontSize = 14.sp, color = Color.White)
                        Switch(
                            checked = isKioskEnabled,
                            onCheckedChange = { viewModel.setKioskMode(it) },
                            colors = SwitchDefaults.colors(checkedThumbColor = Color.White, checkedTrackColor = CyanAccent)
                        )
                    }

                    Spacer(modifier = Modifier.height(14.dp))

                    Button(
                        onClick = {
                            viewModel.enterScreensaver()
                            onDismiss()
                        },
                        colors = ButtonDefaults.buttonColors(containerColor = CyanAccent.copy(alpha = 0.2f), contentColor = CyanAccent),
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text("🌙 Avvia Screensaver Notturno Adesso", fontWeight = FontWeight.Bold, fontSize = 13.sp)
                    }
                }
            }

            // SECTION 2: BATTERY HEALTH CYCLING 80% / 20%
            Card(
                shape = RoundedCornerShape(18.dp),
                colors = CardDefaults.cardColors(containerColor = Slate800),
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("🔋", fontSize = 22.sp)
                        Spacer(modifier = Modifier.width(10.dp))
                        Text("Protezione Ciclo Batteria", fontWeight = FontWeight.Bold, fontSize = 15.sp, color = Color.White)
                    }

                    Spacer(modifier = Modifier.height(14.dp))

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text("Ciclo 80% / 20%", fontSize = 14.sp, color = Color.White)
                        Switch(
                            checked = isBatteryProtectionEnabled,
                            onCheckedChange = { viewModel.setBatteryProtection(it) },
                            colors = SwitchDefaults.colors(checkedThumbColor = Color.White, checkedTrackColor = EmeraldSuccess)
                        )
                    }

                    Spacer(modifier = Modifier.height(12.dp))

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text("Livello Batteria Attuale:", fontSize = 13.sp, color = Color.White.copy(alpha = 0.7f))
                        Text("78%", fontWeight = FontWeight.Bold, fontSize = 13.sp, color = EmeraldSuccess)
                    }

                    Spacer(modifier = Modifier.height(6.dp))

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text("Stato Carica:", fontSize = 13.sp, color = Color.White.copy(alpha = 0.7f))
                        Text("In Carica ⚡️", fontWeight = FontWeight.Bold, fontSize = 13.sp, color = CyanAccent)
                    }

                    Spacer(modifier = Modifier.height(14.dp))

                    OutlinedTextField(
                        value = plugInput,
                        onValueChange = {
                            plugInput = it
                            viewModel.setKioskSmartPlugId(it)
                        },
                        label = { Text("ID Presa Smart Ricarica") },
                        singleLine = true,
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth()
                    )

                    Spacer(modifier = Modifier.height(8.dp))

                    Text(
                        "Quando la carica raggiunge l'80%, l'app invia il comando di spegnimento alla presa smart per preservare la batteria al litio; la riaccende al 20%.",
                        fontSize = 11.sp,
                        color = Color.White.copy(alpha = 0.6f)
                    )
                }
            }

            Spacer(modifier = Modifier.height(24.dp))
        }
    }
}
