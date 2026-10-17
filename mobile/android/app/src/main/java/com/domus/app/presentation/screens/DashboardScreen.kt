package com.domus.app.presentation.screens

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.domus.app.domain.model.Entity
import com.domus.app.presentation.components.ConnectionStatusBanner
import com.domus.app.presentation.components.EntityCard
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
import com.domus.app.presentation.screens.timeline.ActivityTimelineScreen
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.IndigoPrimary
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.viewmodel.DashboardViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    viewModel: DashboardViewModel,
    modifier: Modifier = Modifier
) {
    val connection by viewModel.connection.collectAsState()
    val entities by viewModel.filteredEntities.collectAsState()
    val areas by viewModel.areas.collectAsState()
    val selectedAreaId by viewModel.selectedAreaId.collectAsState()
    val searchQuery by viewModel.searchQuery.collectAsState()
    val isRefreshing by viewModel.isRefreshing.collectAsState()
    val authSession by viewModel.authSession.collectAsState()
    val isScreensaverActive by viewModel.isScreensaverActive.collectAsState()

    // Beacon detection
    val activeRoomName by viewModel.beaconManager.activeRoomName.collectAsState()
    val currentDistanceMeters by viewModel.beaconManager.currentDistanceMeters.collectAsState()
    val financeSummary by viewModel.financeSummary.collectAsState()

    // Floorplan toggle
    var isFloorplanMode by remember { mutableStateOf(false) }

    // Dialog sheets
    var showShoppingSheet by remember { mutableStateOf(false) }
    var showAssistantSheet by remember { mutableStateOf(false) }
    var showChoresSheet by remember { mutableStateOf(false) }
    var showEnergySheet by remember { mutableStateOf(false) }
    var showNotificationsSheet by remember { mutableStateOf(false) }
    var showAutomationBuilderSheet by remember { mutableStateOf(false) }
    var showActivityTimelineSheet by remember { mutableStateOf(false) }
    var showMeshDiagnosticsSheet by remember { mutableStateOf(false) }
    var showGuestAccessSheet by remember { mutableStateOf(false) }
    var showPresenceAndNfcSheet by remember { mutableStateOf(false) }
    var showKioskSettingsSheet by remember { mutableStateOf(false) }
    var showFinanceSheet by remember { mutableStateOf(false) }
    var showLoginSheet by remember { mutableStateOf(false) }

    // Selected Entity Detailed Inspectors
    var selectedLightEntity by remember { mutableStateOf<Entity?>(null) }
    var selectedClimateEntity by remember { mutableStateOf<Entity?>(null) }
    var selectedAlarmEntity by remember { mutableStateOf<Entity?>(null) }
    var selectedCameraEntity by remember { mutableStateOf<Entity?>(null) }
    var selectedSocketEntity by remember { mutableStateOf<Entity?>(null) }

    Box(modifier = modifier.fillMaxSize()) {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = {
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier.padding(end = 4.dp)
                        ) {
                            Text(
                                text = "Domus",
                                fontWeight = FontWeight.Black,
                                fontSize = 20.sp,
                                color = MaterialTheme.colorScheme.onBackground,
                                maxLines = 1,
                                softWrap = false
                            )
                            Spacer(modifier = Modifier.width(6.dp))
                            Text(
                                text = "Native",
                                fontSize = 11.sp,
                                fontWeight = FontWeight.Bold,
                                color = CyanAccent,
                                maxLines = 1,
                                softWrap = false,
                                modifier = Modifier
                                    .background(CyanAccent.copy(alpha = 0.15f), RoundedCornerShape(6.dp))
                                    .padding(horizontal = 5.dp, vertical = 2.dp)
                            )
                        }
                    },
                    actions = {
                        // Compact Connection Status Chip (tap to reconnect if disconnected)
                        ConnectionStatusBanner(
                            connection = connection,
                            compact = true,
                            onReconnectClick = { viewModel.reconnect() }
                        )

                        Spacer(modifier = Modifier.width(4.dp))

                        // Floorplan toggle
                        IconButton(
                            onClick = { isFloorplanMode = !isFloorplanMode },
                            modifier = Modifier
                                .size(38.dp)
                                .background(IndigoPrimary.copy(alpha = 0.15f), CircleShape)
                        ) {
                            Text(if (isFloorplanMode) "▦" else "🗺️", fontSize = 15.sp)
                        }

                        Spacer(modifier = Modifier.width(4.dp))

                        // Profile / Login button
                        IconButton(
                            onClick = { showLoginSheet = true },
                            modifier = Modifier
                                .size(38.dp)
                                .background(
                                    (if (authSession.isAuthenticated) EmeraldSuccess else Color(0xFFF97316)).copy(alpha = 0.15f),
                                    CircleShape
                                )
                        ) {
                            Text(if (authSession.isAuthenticated) "👤" else "🔑", fontSize = 15.sp)
                        }

                        Spacer(modifier = Modifier.width(2.dp))

                        // Overflow / More options menu
                        var showOverflowMenu by remember { mutableStateOf(false) }
                        Box {
                            IconButton(
                                onClick = { showOverflowMenu = true },
                                modifier = Modifier
                                    .size(38.dp)
                                    .background(Slate700.copy(alpha = 0.3f), CircleShape)
                            ) {
                                Text(
                                    "⋮",
                                    fontSize = 18.sp,
                                    fontWeight = FontWeight.Bold,
                                    color = MaterialTheme.colorScheme.onBackground
                                )
                            }

                            DropdownMenu(
                                expanded = showOverflowMenu,
                                onDismissRequest = { showOverflowMenu = false }
                            ) {
                                DropdownMenuItem(
                                    text = { Text("Presenza & Tag NFC") },
                                    leadingIcon = { Text("📍", fontSize = 16.sp) },
                                    onClick = {
                                        showOverflowMenu = false
                                        showPresenceAndNfcSheet = true
                                    }
                                )
                                DropdownMenuItem(
                                    text = { Text("Modalità Kiosk") },
                                    leadingIcon = { Text("📟", fontSize = 16.sp) },
                                    onClick = {
                                        showOverflowMenu = false
                                        showKioskSettingsSheet = true
                                    }
                                )
                                DropdownMenuItem(
                                    text = { Text("Riconnetti Hub") },
                                    leadingIcon = { Text("🔄", fontSize = 16.sp) },
                                    onClick = {
                                        showOverflowMenu = false
                                        viewModel.reconnect()
                                    }
                                )
                            }
                        }

                        Spacer(modifier = Modifier.width(6.dp))
                    },
                    colors = TopAppBarDefaults.topAppBarColors(
                        containerColor = MaterialTheme.colorScheme.background
                    )
                )
            },
            containerColor = MaterialTheme.colorScheme.background
        ) { innerPadding ->
            Column(
                modifier = Modifier
                    .padding(innerPadding)
                    .fillMaxSize()
                    .padding(horizontal = 16.dp)
            ) {
                // Disconnected Alert Banner (if offline, shows full actionable banner)
                AnimatedVisibility(
                    visible = !connection.isConnected,
                    enter = fadeIn(),
                    exit = fadeOut()
                ) {
                    ConnectionStatusBanner(
                        connection = connection,
                        compact = false,
                        onReconnectClick = { viewModel.reconnect() },
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(bottom = 6.dp)
                    )
                }

                // Room Presence Beacon Banner (Active indoor micro-location)
                activeRoomName?.let { roomName ->
                    Surface(
                        shape = RoundedCornerShape(16.dp),
                        color = CyanAccent.copy(alpha = 0.12f),
                        border = BorderStroke(1.dp, CyanAccent.copy(alpha = 0.35f)),
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 4.dp)
                            .semantics(mergeDescendants = true) {
                                contentDescription = "Stanza rilevata via beacon: $roomName"
                                role = Role.Button
                            }
                    ) {
                        Row(
                            modifier = Modifier.padding(horizontal = 12.dp, vertical = 7.dp),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.SpaceBetween
                        ) {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text("📍", fontSize = 14.sp)
                                Spacer(modifier = Modifier.width(6.dp))
                                Text(
                                    text = "Stanza Rilevata: $roomName",
                                    fontSize = 12.sp,
                                    fontWeight = FontWeight.Bold,
                                    color = CyanAccent
                                )
                            }
                            currentDistanceMeters?.let { dist ->
                                Text(
                                    text = String.format("%.1fm", dist),
                                    fontSize = 11.sp,
                                    fontWeight = FontWeight.SemiBold,
                                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                                )
                            }
                        }
                    }
                }

                // Quick Feature Horizontal Scroll (Assistant, Automations, Timeline, Energy, Mesh, Guests, Shopping, Chores, Notifications, Presence, Kiosk)
                Row(
                    modifier = Modifier
                        .horizontalScroll(rememberScrollState())
                        .padding(vertical = 6.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    QuickActionChip(icon = "🎙️", label = "Assistente", onClick = { showAssistantSheet = true })
                    QuickActionChip(icon = "🪄", label = "Automazioni", onClick = { showAutomationBuilderSheet = true })
                    QuickActionChip(icon = "📍", label = "Presenza", onClick = { showPresenceAndNfcSheet = true })
                    QuickActionChip(icon = "📟", label = "Kiosk", onClick = { showKioskSettingsSheet = true })
                    QuickActionChip(icon = "🕰️", label = "Timeline", onClick = { showActivityTimelineSheet = true })
                    QuickActionChip(icon = "⚡", label = "Energia", onClick = { showEnergySheet = true })
                    QuickActionChip(icon = "📡", label = "Rete & Mesh", onClick = { showMeshDiagnosticsSheet = true })
                    QuickActionChip(icon = "👥", label = "Ospiti", onClick = { showGuestAccessSheet = true })
                    QuickActionChip(icon = "🛒", label = "Spesa", onClick = { showShoppingSheet = true })
                    QuickActionChip(icon = "💰", label = "Spese", onClick = { showFinanceSheet = true })
                    QuickActionChip(icon = "🧹", label = "Faccende", onClick = { showChoresSheet = true })
                    QuickActionChip(icon = "🔔", label = "Notifiche", onClick = { showNotificationsSheet = true })
                }

                financeSummary?.let { summary ->
                    Surface(
                        shape = RoundedCornerShape(14.dp),
                        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.45f),
                        border = BorderStroke(1.dp, Slate700.copy(alpha = 0.35f)),
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 4.dp)
                            .clickable { showFinanceSheet = true }
                            .semantics(mergeDescendants = true) {
                                contentDescription = "Bilancio e Spese Casa: spesi %.2f euro".format(summary.total_spent)
                                role = Role.Button
                            }
                    ) {
                        Row(
                            modifier = Modifier.padding(horizontal = 14.dp, vertical = 8.dp),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.SpaceBetween
                        ) {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text("💰", fontSize = 16.sp)
                                Spacer(modifier = Modifier.width(8.dp))
                                Column {
                                    Text(
                                        text = "Spese del Mese",
                                        fontSize = 11.sp,
                                        fontWeight = FontWeight.SemiBold,
                                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                                    )
                                    Text(
                                        text = "€%.2f".format(summary.total_spent),
                                        fontSize = 15.sp,
                                        fontWeight = FontWeight.Bold,
                                        color = MaterialTheme.colorScheme.onSurface
                                    )
                                }
                            }
                            if (summary.monthly_budget > 0) {
                                Column(horizontalAlignment = Alignment.End) {
                                    Text(
                                        text = if (summary.remaining_budget >= 0) "Budget Residuo" else "Sforamento Budget",
                                        fontSize = 11.sp,
                                        fontWeight = FontWeight.SemiBold,
                                        color = if (summary.remaining_budget >= 0) EmeraldSuccess else Color(0xFFEF4444)
                                    )
                                    Text(
                                        text = "€%.2f".format(summary.remaining_budget),
                                        fontSize = 13.sp,
                                        fontWeight = FontWeight.Bold,
                                        color = if (summary.remaining_budget >= 0) EmeraldSuccess else Color(0xFFEF4444)
                                    )
                                }
                            }
                        }
                    }
                }

                if (isFloorplanMode) {
                    FloorplanCanvas(
                        entities = entities,
                        onEntityClick = { viewModel.toggleEntity(it) },
                        onDismiss = { isFloorplanMode = false },
                        modifier = Modifier.weight(1f)
                    )
                } else {
                    // Search Input
                    OutlinedTextField(
                        value = searchQuery,
                        onValueChange = { viewModel.setSearchQuery(it) },
                        placeholder = { Text("Cerca dispositivi o stanze...", fontSize = 14.sp) },
                        singleLine = true,
                        shape = RoundedCornerShape(12.dp),
                        colors = OutlinedTextFieldDefaults.colors(
                            focusedBorderColor = CyanAccent,
                            unfocusedBorderColor = Slate700.copy(alpha = 0.5f)
                        ),
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 4.dp)
                    )

                    // Area Filter Chips Horizontal Scroll
                    Row(
                        modifier = Modifier
                            .horizontalScroll(rememberScrollState())
                            .padding(vertical = 6.dp)
                    ) {
                        Surface(
                            shape = RoundedCornerShape(20.dp),
                            color = if (selectedAreaId == null) IndigoPrimary else MaterialTheme.colorScheme.surface,
                            modifier = Modifier
                                .clickable { viewModel.selectArea(null) }
                                .padding(end = 8.dp)
                        ) {
                            Text(
                                text = "Tutte le Stanze",
                                fontSize = 13.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = if (selectedAreaId == null) Color.White else MaterialTheme.colorScheme.onSurface,
                                modifier = Modifier.padding(horizontal = 14.dp, vertical = 8.dp)
                            )
                        }

                        areas.forEach { area ->
                            val isSelected = selectedAreaId == area.id
                            Surface(
                                shape = RoundedCornerShape(20.dp),
                                color = if (isSelected) IndigoPrimary else MaterialTheme.colorScheme.surface,
                                modifier = Modifier
                                    .clickable { viewModel.selectArea(area.id) }
                                    .padding(end = 8.dp)
                            ) {
                                Text(
                                    text = area.name,
                                    fontSize = 13.sp,
                                    fontWeight = FontWeight.SemiBold,
                                    color = if (isSelected) Color.White else MaterialTheme.colorScheme.onSurface,
                                    modifier = Modifier.padding(horizontal = 14.dp, vertical = 8.dp)
                                )
                            }
                        }
                    }

                    Spacer(modifier = Modifier.height(6.dp))

                    // Grid or Empty State
                    if (isRefreshing && entities.isEmpty()) {
                        Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                            CircularProgressIndicator(color = CyanAccent)
                        }
                    } else if (entities.isEmpty()) {
                        Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                                Text("📡 Nessun dispositivo trovato", fontWeight = FontWeight.Bold, fontSize = 16.sp)
                                Spacer(modifier = Modifier.height(6.dp))
                                Text(
                                    "Verifica che il server Domus sia avviato sulla rete locale",
                                    fontSize = 13.sp,
                                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                                )
                                Spacer(modifier = Modifier.height(16.dp))
                                Button(onClick = { viewModel.reconnect() }) {
                                    Text("Riprova Scansione LAN")
                                }
                            }
                        }
                    } else {
                        LazyVerticalGrid(
                            columns = GridCells.Fixed(2),
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                            verticalArrangement = Arrangement.spacedBy(12.dp),
                            modifier = Modifier.fillMaxSize()
                        ) {
                            items(entities, key = { it.id }) { entity ->
                                EntityCard(
                                    entity = entity,
                                    onToggle = { viewModel.toggleEntity(it) },
                                    onClick = {
                                        when {
                                            entity.isLight -> selectedLightEntity = entity
                                            entity.isClimate -> selectedClimateEntity = entity
                                            entity.isAlarm -> selectedAlarmEntity = entity
                                            entity.isCamera -> selectedCameraEntity = entity
                                            entity.isSwitch -> selectedSocketEntity = entity
                                            else -> viewModel.toggleEntity(entity.id)
                                        }
                                    }
                                )
                            }
                        }
                    }
                }
            }
        }

        // Ambient Screensaver Overlay
        AnimatedVisibility(
            visible = isScreensaverActive,
            enter = fadeIn(),
            exit = fadeOut()
        ) {
            KioskScreensaverView(onWake = { viewModel.wakeFromScreensaver() })
        }
    }

    // Modal Sheets and Fullscreen Dialogs for companion features
    if (showShoppingSheet) {
        Dialog(onDismissRequest = { showShoppingSheet = false }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            ShoppingListScreen(viewModel = viewModel, onDismiss = { showShoppingSheet = false })
        }
    }

    if (showAssistantSheet) {
        Dialog(onDismissRequest = { showAssistantSheet = false }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            AssistantChatScreen(viewModel = viewModel, onDismiss = { showAssistantSheet = false })
        }
    }

    if (showChoresSheet) {
        Dialog(onDismissRequest = { showChoresSheet = false }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            ChoresScreen(viewModel = viewModel, onDismiss = { showChoresSheet = false })
        }
    }

    if (showEnergySheet) {
        Dialog(onDismissRequest = { showEnergySheet = false }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            EnergyScreen(viewModel = viewModel, onDismiss = { showEnergySheet = false })
        }
    }

    if (showNotificationsSheet) {
        Dialog(onDismissRequest = { showNotificationsSheet = false }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            NotificationHistoryScreen(viewModel = viewModel, onDismiss = { showNotificationsSheet = false })
        }
    }

    if (showAutomationBuilderSheet) {
        Dialog(onDismissRequest = { showAutomationBuilderSheet = false }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            AutomationBuilderScreen(viewModel = viewModel, onDismiss = { showAutomationBuilderSheet = false })
        }
    }

    if (showActivityTimelineSheet) {
        Dialog(onDismissRequest = { showActivityTimelineSheet = false }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            ActivityTimelineScreen(viewModel = viewModel, onDismiss = { showActivityTimelineSheet = false })
        }
    }

    if (showMeshDiagnosticsSheet) {
        Dialog(onDismissRequest = { showMeshDiagnosticsSheet = false }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            MeshDiagnosticsScreen(viewModel = viewModel, onDismiss = { showMeshDiagnosticsSheet = false })
        }
    }

    if (showGuestAccessSheet) {
        Dialog(onDismissRequest = { showGuestAccessSheet = false }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            GuestAccessScreen(viewModel = viewModel, onDismiss = { showGuestAccessSheet = false })
        }
    }

    if (showPresenceAndNfcSheet) {
        Dialog(onDismissRequest = { showPresenceAndNfcSheet = false }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            PresenceAndNfcScreen(viewModel = viewModel, onDismiss = { showPresenceAndNfcSheet = false })
        }
    }

    if (showKioskSettingsSheet) {
        Dialog(onDismissRequest = { showKioskSettingsSheet = false }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            KioskSettingsScreen(viewModel = viewModel, onDismiss = { showKioskSettingsSheet = false })
        }
    }

    if (showFinanceSheet) {
        Dialog(onDismissRequest = { showFinanceSheet = false }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            FinanceScreen(viewModel = viewModel, onDismiss = { showFinanceSheet = false })
        }
    }

    if (showLoginSheet) {
        Dialog(onDismissRequest = { showLoginSheet = false }, properties = DialogProperties(usePlatformDefaultWidth = false)) {
            LoginScreen(
                onLogin = { url, user, pass ->
                    viewModel.login(url, user, pass) { success, _ ->
                        if (success) showLoginSheet = false
                    }
                },
                onDiscoverLan = { viewModel.reconnect() },
                discoveredHubUrl = connection.activeUrl,
                isLoading = isRefreshing,
                errorMessage = null
            )
        }
    }

    // Entity Specific Modal Inspectors
    selectedLightEntity?.let { light ->
        LightControlSheet(
            entity = light,
            onToggle = { viewModel.toggleEntity(light.id) },
            onSetBrightness = { id, brightness ->
                viewModel.callService("light", "turn_on", id, mapOf("brightness" to brightness))
            },
            onDismiss = { selectedLightEntity = null }
        )
    }

    selectedClimateEntity?.let { climate ->
        ThermostatDial(
            entity = climate,
            onSetTemperature = { id, temp ->
                viewModel.callService("climate", "set_temperature", id, mapOf("temperature" to temp))
            },
            onSetMode = { id, mode ->
                viewModel.callService("climate", "set_hvac_mode", id, mapOf("hvac_mode" to mode))
            },
            onDismiss = { selectedClimateEntity = null }
        )
    }

    selectedAlarmEntity?.let { alarm ->
        AlarmBottomSheet(
            entity = alarm,
            onArmHome = { viewModel.callService("alarm_control_panel", "alarm_arm_home", alarm.id) },
            onArmAway = { viewModel.callService("alarm_control_panel", "alarm_arm_away", alarm.id) },
            onDisarm = { id, code -> viewModel.callService("alarm_control_panel", "alarm_disarm", id, mapOf("code" to code)) },
            onDismiss = { selectedAlarmEntity = null }
        )
    }

    selectedCameraEntity?.let { camera ->
        Dialog(
            onDismissRequest = { selectedCameraEntity = null },
            properties = DialogProperties(usePlatformDefaultWidth = false)
        ) {
            CameraStreamScreen(
                cameraName = camera.name,
                onDismiss = { selectedCameraEntity = null }
            )
        }
    }

    selectedSocketEntity?.let { socket ->
        SocketControlSheet(
            entity = socket,
            onToggle = { viewModel.toggleEntity(socket.id) },
            onDismiss = { selectedSocketEntity = null }
        )
    }
}

@Composable
fun QuickActionChip(
    icon: String,
    label: String,
    onClick: () -> Unit
) {
    Surface(
        shape = RoundedCornerShape(16.dp),
        color = Slate800,
        border = BorderStroke(1.dp, Slate700),
        modifier = Modifier
            .clickable { onClick() }
            .semantics(mergeDescendants = true) {
                contentDescription = "$label, modulo rapido"
                role = Role.Button
            }
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 7.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(icon, fontSize = 14.sp)
            Spacer(modifier = Modifier.width(6.dp))
            Text(
                text = label,
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = Color.White
            )
        }
    }
}
