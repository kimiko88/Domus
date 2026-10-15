package com.domus.app.presentation.screens

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
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
import com.domus.app.presentation.screens.camera.CameraStreamScreen
import com.domus.app.presentation.screens.chores.ChoresScreen
import com.domus.app.presentation.screens.climate.ThermostatDial
import com.domus.app.presentation.screens.energy.EnergyScreen
import com.domus.app.presentation.screens.floorplan.FloorplanCanvas
import com.domus.app.presentation.screens.light.LightControlSheet
import com.domus.app.presentation.screens.notifications.NotificationHistoryScreen
import com.domus.app.presentation.screens.shopping.ShoppingListScreen
import com.domus.app.presentation.theme.CyanAccent
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

    // Sheet / Dialog visibility states
    var showShoppingSheet by remember { mutableStateOf(false) }
    var showAssistantSheet by remember { mutableStateOf(false) }
    var showChoresSheet by remember { mutableStateOf(false) }
    var showEnergySheet by remember { mutableStateOf(false) }
    var showNotificationsSheet by remember { mutableStateOf(false) }
    var showFloorplanSheet by remember { mutableStateOf(false) }

    // Selected Entity Detailed Inspectors
    var selectedLightEntity by remember { mutableStateOf<Entity?>(null) }
    var selectedClimateEntity by remember { mutableStateOf<Entity?>(null) }
    var selectedAlarmEntity by remember { mutableStateOf<Entity?>(null) }
    var selectedCameraEntity by remember { mutableStateOf<Entity?>(null) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                "Domus",
                                fontWeight = FontWeight.Black,
                                fontSize = 22.sp,
                                color = MaterialTheme.colorScheme.onBackground
                            )
                            Spacer(modifier = Modifier.width(6.dp))
                            Text(
                                "Native",
                                fontSize = 12.sp,
                                fontWeight = FontWeight.Bold,
                                color = CyanAccent,
                                modifier = Modifier
                                    .background(CyanAccent.copy(alpha = 0.15f), RoundedCornerShape(6.dp))
                                    .padding(horizontal = 6.dp, vertical = 2.dp)
                            )
                        }
                    }
                },
                actions = {
                    ConnectionStatusBanner(
                        connection = connection,
                        onReconnectClick = { viewModel.reconnect() }
                    )
                    Spacer(modifier = Modifier.width(12.dp))
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.background
                )
            )
        },
        containerColor = MaterialTheme.colorScheme.background,
        modifier = modifier
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .padding(innerPadding)
                .fillMaxSize()
                .padding(horizontal = 16.dp)
        ) {
            // Quick Feature Scroll Bar (Shopping, Assistant, Energy, Chores, Notifications, Floorplan)
            Row(
                modifier = Modifier
                    .horizontalScroll(rememberScrollState())
                    .padding(vertical = 4.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                QuickActionChip(icon = "🛒", label = "Spesa", onClick = { showShoppingSheet = true })
                QuickActionChip(icon = "🎙️", label = "Assistente", onClick = { showAssistantSheet = true })
                QuickActionChip(icon = "⚡", label = "Energia", onClick = { showEnergySheet = true })
                QuickActionChip(icon = "🧹", label = "Faccende", onClick = { showChoresSheet = true })
                QuickActionChip(icon = "🔔", label = "Notifiche", onClick = { showNotificationsSheet = true })
                QuickActionChip(icon = "🗺️", label = "Mappa", onClick = { showFloorplanSheet = true })
            }

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
                    .padding(vertical = 6.dp)
            )

            // Area Filter Chips Horizontal Scroll
            Row(
                modifier = Modifier
                    .horizontalScroll(rememberScrollState())
                    .padding(vertical = 6.dp)
            ) {
                // "Tutte" chip
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

            Spacer(modifier = Modifier.height(10.dp))

            // Loading / Empty / Grid State
            if (isRefreshing && entities.isEmpty()) {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
                    CircularProgressIndicator(color = CyanAccent)
                }
            } else if (entities.isEmpty()) {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
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
                                    else -> viewModel.toggleEntity(entity.id)
                                }
                            }
                        )
                    }
                }
            }
        }
    }

    // Modal Sheets and Fullscreen Dialogs for missing features
    if (showShoppingSheet) {
        Dialog(
            onDismissRequest = { showShoppingSheet = false },
            properties = DialogProperties(usePlatformDefaultWidth = false)
        ) {
            ShoppingListScreen(onDismiss = { showShoppingSheet = false })
        }
    }

    if (showAssistantSheet) {
        Dialog(
            onDismissRequest = { showAssistantSheet = false },
            properties = DialogProperties(usePlatformDefaultWidth = false)
        ) {
            AssistantChatScreen(onDismiss = { showAssistantSheet = false })
        }
    }

    if (showChoresSheet) {
        Dialog(
            onDismissRequest = { showChoresSheet = false },
            properties = DialogProperties(usePlatformDefaultWidth = false)
        ) {
            ChoresScreen(onDismiss = { showChoresSheet = false })
        }
    }

    if (showEnergySheet) {
        Dialog(
            onDismissRequest = { showEnergySheet = false },
            properties = DialogProperties(usePlatformDefaultWidth = false)
        ) {
            EnergyScreen(onDismiss = { showEnergySheet = false })
        }
    }

    if (showNotificationsSheet) {
        Dialog(
            onDismissRequest = { showNotificationsSheet = false },
            properties = DialogProperties(usePlatformDefaultWidth = false)
        ) {
            NotificationHistoryScreen(onDismiss = { showNotificationsSheet = false })
        }
    }

    if (showFloorplanSheet) {
        Dialog(
            onDismissRequest = { showFloorplanSheet = false },
            properties = DialogProperties(usePlatformDefaultWidth = false)
        ) {
            FloorplanCanvas(
                entities = entities,
                onEntityClick = { viewModel.toggleEntity(it) },
                onDismiss = { showFloorplanSheet = false }
            )
        }
    }

    // Entity Specific Modal Inspectors
    selectedLightEntity?.let { light ->
        ModalBottomSheet(
            onDismissRequest = { selectedLightEntity = null }
        ) {
            LightControlSheet(
                entity = light,
                onToggle = { viewModel.toggleEntity(light.id) },
                onBrightnessChange = { /* Send brightness */ },
                onDismiss = { selectedLightEntity = null }
            )
        }
    }

    selectedClimateEntity?.let { climate ->
        ModalBottomSheet(
            onDismissRequest = { selectedClimateEntity = null }
        ) {
            ThermostatDial(
                entity = climate,
                onSetTemperature = { /* Send target temp */ },
                onSetHvacMode = { /* Send mode */ },
                onDismiss = { selectedClimateEntity = null }
            )
        }
    }

    selectedAlarmEntity?.let { alarm ->
        ModalBottomSheet(
            onDismissRequest = { selectedAlarmEntity = null }
        ) {
            AlarmBottomSheet(
                entity = alarm,
                onArmHome = { viewModel.toggleEntity(alarm.id) },
                onArmAway = { viewModel.toggleEntity(alarm.id) },
                onDisarm = { _, _ -> viewModel.toggleEntity(alarm.id) },
                onDismiss = { selectedAlarmEntity = null }
            )
        }
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
        modifier = Modifier.clickable { onClick() }
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
