package com.domus.app.presentation.screens.adaptive

import android.content.res.Configuration
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.domain.model.Entity
import com.domus.app.presentation.components.EntityCard
import com.domus.app.presentation.screens.DashboardScreen
import com.domus.app.presentation.screens.alarm.AlarmBottomSheet
import com.domus.app.presentation.screens.assistant.AssistantChatScreen
import com.domus.app.presentation.screens.automation.AutomationBuilderScreen
import com.domus.app.presentation.screens.camera.CameraStreamScreen
import com.domus.app.presentation.screens.chores.ChoresScreen
import com.domus.app.presentation.screens.climate.ThermostatDial
import com.domus.app.presentation.screens.diagnostics.MeshDiagnosticsScreen
import com.domus.app.presentation.screens.energy.EnergyScreen
import com.domus.app.presentation.screens.finance.FinanceScreen
import com.domus.app.presentation.screens.floorplan.FloorplanCanvas
import com.domus.app.presentation.screens.guests.GuestAccessScreen
import com.domus.app.presentation.screens.kiosk.KioskScreensaverView
import com.domus.app.presentation.screens.kiosk.KioskSettingsScreen
import com.domus.app.presentation.screens.light.LightControlSheet
import com.domus.app.presentation.screens.notifications.NotificationHistoryScreen
import com.domus.app.presentation.screens.presence.PresenceAndNfcScreen
import com.domus.app.presentation.screens.shopping.ShoppingListScreen
import com.domus.app.presentation.screens.socket.SocketControlSheet
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.IndigoPrimary
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.viewmodel.DashboardViewModel

/**
 * Layout adattivo avanzato per Tablet Android, Foldable e Kiosk a parete (3 Colonne / Split View)
 * con fallback fluido a DashboardScreen su smartphone.
 * Allineato con AdaptiveMainView.swift di iOS.
 */
@Composable
fun AdaptiveMainLayout(
    viewModel: DashboardViewModel,
    modifier: Modifier = Modifier
) {
    val configuration = LocalConfiguration.current
    val isTablet = configuration.screenWidthDp >= 840 || (configuration.screenWidthDp >= 600 && configuration.orientation == Configuration.ORIENTATION_LANDSCAPE)
    val isScreensaverActive by viewModel.isScreensaverActive.collectAsState()

    Box(modifier = modifier.fillMaxSize()) {
        if (isTablet) {
            TabletSplitView(viewModel = viewModel)
        } else {
            DashboardScreen(viewModel = viewModel)
        }

        if (isScreensaverActive) {
            KioskScreensaverView(onWake = { viewModel.wakeFromScreensaver() })
        }
    }
}

@Composable
fun TabletSplitView(viewModel: DashboardViewModel) {
    val entities by viewModel.filteredEntities.collectAsState()
    val allEntities by viewModel.allEntities.collectAsState()
    val areas by viewModel.areas.collectAsState()
    val selectedAreaId by viewModel.selectedAreaId.collectAsState()

    var showFloorplan by remember { mutableStateOf(false) }
    var selectedEntityForDetail by remember { mutableStateOf<Entity?>(null) }

    // Dialog sheets
    var showAssistant by remember { mutableStateOf(false) }
    var showAutomations by remember { mutableStateOf(false) }
    var showTimeline by remember { mutableStateOf(false) }
    var showEnergy by remember { mutableStateOf(false) }
    var showDiagnostics by remember { mutableStateOf(false) }
    var showGuests by remember { mutableStateOf(false) }
    var showShopping by remember { mutableStateOf(false) }
    var showChores by remember { mutableStateOf(false) }
    var showNotifications by remember { mutableStateOf(false) }
    var showPresenceNfc by remember { mutableStateOf(false) }
    var showKioskSettings by remember { mutableStateOf(false) }
    var showFinance by remember { mutableStateOf(false) }

    Row(modifier = Modifier.fillMaxSize()) {
        // COLONNA 1: SIDEBAR (Stanze, Moduli, Riepilogo)
        Surface(
            modifier = Modifier
                .width(280.dp)
                .fillMaxHeight(),
            color = MaterialTheme.colorScheme.surface,
            tonalElevation = 2.dp
        ) {
            LazyColumn(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                // Header
                item {
                    Text("Domus", fontWeight = FontWeight.Black, fontSize = 24.sp, color = MaterialTheme.colorScheme.onSurface)
                    Text("Appliance OS • Tablet Edition", fontSize = 11.sp, color = CyanAccent, fontWeight = FontWeight.Bold)
                    Spacer(modifier = Modifier.height(12.dp))
                }

                // Riepilogo Casa
                item {
                    Text("RIEPILOGO CASA", fontSize = 11.sp, fontWeight = FontWeight.Bold, color = Slate700)
                    Spacer(modifier = Modifier.height(6.dp))
                    val activeLightsCount = allEntities.count { it.isLight && it.isOn }
                    SidebarStatusRow(icon = "💡", title = "Luci Accese", value = "$activeLightsCount", color = Color(0xFFFBBF24))
                    val alarmState = allEntities.firstOrNull { it.isAlarm }?.state?.replaceFirstChar { it.uppercase() } ?: "Disarmato"
                    SidebarStatusRow(icon = "🛡️", title = "Stato Allarme", value = alarmState, color = EmeraldSuccess)
                }

                // Visualizzazione (Griglia / Pianta)
                item {
                    Text("VISUALIZZAZIONE", fontSize = 11.sp, fontWeight = FontWeight.Bold, color = Slate700)
                    Spacer(modifier = Modifier.height(6.dp))
                    SidebarButton(
                        icon = if (showFloorplan) "▦" else "🗺️",
                        title = if (showFloorplan) "Mostra Griglia Dispositivi" else "Mostra Pianta 2D",
                        onClick = { showFloorplan = !showFloorplan },
                        accentColor = CyanAccent
                    )
                }

                // Funzionalità Casa
                item {
                    Text("FUNZIONALITÀ CASA", fontSize = 11.sp, fontWeight = FontWeight.Bold, color = Slate700)
                    Spacer(modifier = Modifier.height(6.dp))
                    SidebarButton(icon = "🎙️", title = "Assistente AI", onClick = { showAssistant = true }, accentColor = CyanAccent)
                    SidebarButton(icon = "🪄", title = "Regole & Automazioni", onClick = { showAutomations = true }, accentColor = Color(0xFF8B5CF6))
                    SidebarButton(icon = "🕰️", title = "Timeline della Casa", onClick = { showTimeline = true }, accentColor = IndigoPrimary)
                    SidebarButton(icon = "⚡", title = "Cruscotto Energia", onClick = { showEnergy = true }, accentColor = Color(0xFFFBBF24))
                    SidebarButton(icon = "📡", title = "Diagnostica Mesh & Hub", onClick = { showDiagnostics = true }, accentColor = EmeraldSuccess)
                    SidebarButton(icon = "👥", title = "Accesso Ospiti & Membri", onClick = { showGuests = true }, accentColor = Color(0xFF6366F1))
                    SidebarButton(icon = "🛒", title = "Lista della Spesa", onClick = { showShopping = true }, accentColor = Color(0xFF10B981))
                    SidebarButton(icon = "💰", title = "Bilancio & Spese", onClick = { showFinance = true }, accentColor = Color(0xFF10B981))
                    SidebarButton(icon = "🧹", title = "Faccende Domestiche", onClick = { showChores = true }, accentColor = Color(0xFFF97316))
                    SidebarButton(icon = "🔔", title = "Centro Notifiche", onClick = { showNotifications = true }, accentColor = Color(0xFFEF4444))
                    SidebarButton(icon = "📍", title = "Presenza & NFC", onClick = { showPresenceNfc = true }, accentColor = CyanAccent)
                    SidebarButton(icon = "📟", title = "Modalità Kiosk", onClick = { showKioskSettings = true }, accentColor = Color.White)
                }

                // Stanze
                item {
                    Text("STANZE", fontSize = 11.sp, fontWeight = FontWeight.Bold, color = Slate700)
                    Spacer(modifier = Modifier.height(6.dp))
                    SidebarAreaItem("Tutte le Stanze", selected = selectedAreaId == null) {
                        viewModel.selectArea(null)
                    }
                    areas.forEach { area ->
                        SidebarAreaItem(area.name, selected = selectedAreaId == area.id) {
                            viewModel.selectArea(area.id)
                        }
                    }
                }
            }
        }

        VerticalDivider(modifier = Modifier.fillMaxHeight().width(1.dp), color = Slate700)

        // COLONNA 2: CONTENUTO PRINCIPALE (Griglia o Pianta 2D)
        Box(modifier = Modifier.weight(1f).fillMaxHeight().padding(16.dp)) {
            if (showFloorplan) {
                FloorplanCanvas(
                    entities = entities,
                    onEntityClick = { viewModel.toggleEntity(it) },
                    onDismiss = { showFloorplan = false }
                )
            } else {
                LazyVerticalGrid(
                    columns = GridCells.Adaptive(160.dp),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.fillMaxSize()
                ) {
                    items(entities, key = { it.id }) { entity ->
                        EntityCard(
                            entity = entity,
                            onToggle = { viewModel.toggleEntity(it) },
                            onClick = { selectedEntityForDetail = entity }
                        )
                    }
                }
            }
        }

        // COLONNA 3: INSPECTOR / DETTAGLIO RAPIDO
        VerticalDivider(modifier = Modifier.fillMaxHeight().width(1.dp), color = Slate700)

        Surface(
            modifier = Modifier.width(320.dp).fillMaxHeight(),
            color = MaterialTheme.colorScheme.surface
        ) {
            Box(modifier = Modifier.fillMaxSize().padding(16.dp), contentAlignment = Alignment.Center) {
                val selected = selectedEntityForDetail
                if (selected != null) {
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(16.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text(selected.name, fontWeight = FontWeight.Bold, fontSize = 20.sp)
                        Text("Stato: ${selected.state.uppercase()}", color = CyanAccent, fontWeight = FontWeight.SemiBold)

                        Button(
                            onClick = { viewModel.toggleEntity(selected.id) },
                            colors = ButtonDefaults.buttonColors(containerColor = IndigoPrimary),
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Text(if (selected.isOn) "Disattiva" else "Attiva")
                        }

                        OutlinedButton(
                            onClick = { selectedEntityForDetail = null },
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Text("Chiudi Pannello")
                        }
                    }
                } else {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text("🏠", fontSize = 48.sp)
                        Spacer(modifier = Modifier.height(10.dp))
                        Text("Pannello Dettagli", fontWeight = FontWeight.Bold, fontSize = 16.sp)
                        Text("Seleziona un dispositivo dalla griglia per visualizzare i controlli dedicati.", fontSize = 12.sp, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f), modifier = Modifier.padding(horizontal = 16.dp))
                    }
                }
            }
        }
    }

    // Modal Sheets for Dialogs
    if (showAssistant) {
        androidx.compose.ui.window.Dialog(onDismissRequest = { showAssistant = false }, properties = androidx.compose.ui.window.DialogProperties(usePlatformDefaultWidth = false)) {
            AssistantChatScreen(viewModel = viewModel, onDismiss = { showAssistant = false })
        }
    }

    if (showAutomations) {
        androidx.compose.ui.window.Dialog(onDismissRequest = { showAutomations = false }, properties = androidx.compose.ui.window.DialogProperties(usePlatformDefaultWidth = false)) {
            AutomationBuilderScreen(viewModel = viewModel, onDismiss = { showAutomations = false })
        }
    }

    if (showTimeline) {
        androidx.compose.ui.window.Dialog(onDismissRequest = { showTimeline = false }, properties = androidx.compose.ui.window.DialogProperties(usePlatformDefaultWidth = false)) {
            com.domus.app.presentation.screens.timeline.ActivityTimelineScreen(viewModel = viewModel, onDismiss = { showTimeline = false })
        }
    }

    if (showEnergy) {
        androidx.compose.ui.window.Dialog(onDismissRequest = { showEnergy = false }, properties = androidx.compose.ui.window.DialogProperties(usePlatformDefaultWidth = false)) {
            EnergyScreen(viewModel = viewModel, onDismiss = { showEnergy = false })
        }
    }

    if (showDiagnostics) {
        androidx.compose.ui.window.Dialog(onDismissRequest = { showDiagnostics = false }, properties = androidx.compose.ui.window.DialogProperties(usePlatformDefaultWidth = false)) {
            MeshDiagnosticsScreen(viewModel = viewModel, onDismiss = { showDiagnostics = false })
        }
    }

    if (showGuests) {
        androidx.compose.ui.window.Dialog(onDismissRequest = { showGuests = false }, properties = androidx.compose.ui.window.DialogProperties(usePlatformDefaultWidth = false)) {
            GuestAccessScreen(viewModel = viewModel, onDismiss = { showGuests = false })
        }
    }

    if (showShopping) {
        androidx.compose.ui.window.Dialog(onDismissRequest = { showShopping = false }, properties = androidx.compose.ui.window.DialogProperties(usePlatformDefaultWidth = false)) {
            ShoppingListScreen(viewModel = viewModel, onDismiss = { showShopping = false })
        }
    }

    if (showChores) {
        androidx.compose.ui.window.Dialog(onDismissRequest = { showChores = false }, properties = androidx.compose.ui.window.DialogProperties(usePlatformDefaultWidth = false)) {
            ChoresScreen(viewModel = viewModel, onDismiss = { showChores = false })
        }
    }

    if (showNotifications) {
        androidx.compose.ui.window.Dialog(onDismissRequest = { showNotifications = false }, properties = androidx.compose.ui.window.DialogProperties(usePlatformDefaultWidth = false)) {
            NotificationHistoryScreen(viewModel = viewModel, onDismiss = { showNotifications = false })
        }
    }

    if (showPresenceNfc) {
        androidx.compose.ui.window.Dialog(onDismissRequest = { showPresenceNfc = false }, properties = androidx.compose.ui.window.DialogProperties(usePlatformDefaultWidth = false)) {
            PresenceAndNfcScreen(viewModel = viewModel, onDismiss = { showPresenceNfc = false })
        }
    }

    if (showKioskSettings) {
        androidx.compose.ui.window.Dialog(onDismissRequest = { showKioskSettings = false }, properties = androidx.compose.ui.window.DialogProperties(usePlatformDefaultWidth = false)) {
            KioskSettingsScreen(viewModel = viewModel, onDismiss = { showKioskSettings = false })
        }
    }

    if (showFinance) {
        androidx.compose.ui.window.Dialog(onDismissRequest = { showFinance = false }, properties = androidx.compose.ui.window.DialogProperties(usePlatformDefaultWidth = false)) {
            FinanceScreen(viewModel = viewModel, onDismiss = { showFinance = false })
        }
    }
}

@Composable
fun SidebarStatusRow(icon: String, title: String, value: String, color: Color) {
    Row(
        modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(icon, fontSize = 14.sp)
            Spacer(modifier = Modifier.width(8.dp))
            Text(title, fontSize = 13.sp, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.8f))
        }
        Text(value, fontSize = 13.sp, fontWeight = FontWeight.Bold, color = color)
    }
}

@Composable
fun SidebarButton(icon: String, title: String, onClick: () -> Unit, accentColor: Color) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onClick() }
            .padding(vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(icon, fontSize = 14.sp)
        Spacer(modifier = Modifier.width(8.dp))
        Text(title, fontSize = 13.sp, fontWeight = FontWeight.SemiBold, color = accentColor)
    }
}

@Composable
fun SidebarAreaItem(name: String, selected: Boolean, onClick: () -> Unit) {
    Surface(
        shape = RoundedCornerShape(10.dp),
        color = if (selected) IndigoPrimary else Color.Transparent,
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onClick() }
            .padding(vertical = 2.dp)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Text(name, fontSize = 13.sp, fontWeight = if (selected) FontWeight.Bold else FontWeight.Normal, color = if (selected) Color.White else MaterialTheme.colorScheme.onSurface)
            if (selected) Text("✓", fontSize = 12.sp, color = Color.White)
        }
    }
}
