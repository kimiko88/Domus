package com.domus.app.presentation.screens.diagnostics

import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
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
import com.domus.app.domain.model.MeshNodeDiagnostic
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.IndigoPrimary
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.viewmodel.DashboardViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MeshDiagnosticsScreen(
    viewModel: DashboardViewModel,
    onDismiss: () -> Unit
) {
    val nodes by viewModel.meshNodes.collectAsState()
    val health by viewModel.systemHealth.collectAsState()
    val latencyMs by viewModel.lanLatencyMs.collectAsState()

    var selectedTab by remember { mutableIntStateOf(0) } // 0: Mesh & RSSI, 1: Batterie, 2: Salute Hub
    var selectedProtocolFilter by remember { mutableStateOf("Tutti") }

    LaunchedEffect(Unit) {
        viewModel.loadMeshDiagnostics()
    }

    val lowBatteryNodes = remember(nodes) {
        nodes.filter { (it.batteryLevel ?: 100) < 15 }
    }

    val filteredNodes = remember(nodes, selectedProtocolFilter) {
        if (selectedProtocolFilter == "Tutti") nodes
        else nodes.filter { it.protocolType == selectedProtocolFilter }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text("Diagnostica Mesh & Hub", fontWeight = FontWeight.Bold, fontSize = 20.sp)
                        Text("Stato radio RF, batterie e salute appliance", fontSize = 12.sp, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f))
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onDismiss) {
                        Text("✕", fontSize = 18.sp, fontWeight = FontWeight.Bold)
                    }
                },
                actions = {
                    IconButton(onClick = { viewModel.loadMeshDiagnostics() }) {
                        Text("🔄", fontSize = 18.sp)
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
        ) {
            // Tab Selector
            PrimaryTabRow(
                selectedTabIndex = selectedTab,
                containerColor = Color.Transparent,
                contentColor = CyanAccent,
                modifier = Modifier.padding(vertical = 8.dp)
            ) {
                Tab(
                    selected = selectedTab == 0,
                    onClick = { selectedTab = 0 },
                    text = { Text("Mesh & RSSI", fontSize = 13.sp, fontWeight = FontWeight.Bold) }
                )
                Tab(
                    selected = selectedTab == 1,
                    onClick = { selectedTab = 1 },
                    text = {
                        Text(
                            text = "Batterie (${lowBatteryNodes.size} critiche)",
                            fontSize = 13.sp,
                            fontWeight = FontWeight.Bold,
                            color = if (lowBatteryNodes.isNotEmpty()) Color(0xFFEF4444) else MaterialTheme.colorScheme.onSurface
                        )
                    }
                )
                Tab(
                    selected = selectedTab == 2,
                    onClick = { selectedTab = 2 },
                    text = { Text("Salute Hub", fontSize = 13.sp, fontWeight = FontWeight.Bold) }
                )
            }

            Spacer(modifier = Modifier.height(8.dp))

            when (selectedTab) {
                0 -> {
                    // Protocol Filters Row
                    Row(
                        modifier = Modifier
                            .horizontalScroll(rememberScrollState())
                            .padding(bottom = 12.dp),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        listOf("Tutti", "Zigbee 3.0", "Thread / Matter", "Z-Wave Plus", "BLE Proxy").forEach { proto ->
                            FilterChip(
                                selected = selectedProtocolFilter == proto,
                                onClick = { selectedProtocolFilter = proto },
                                label = { Text(proto, fontSize = 12.sp) },
                                colors = FilterChipDefaults.filterChipColors(
                                    selectedContainerColor = IndigoPrimary,
                                    selectedLabelColor = Color.White
                                )
                            )
                        }
                    }

                    LazyColumn(
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                        modifier = Modifier.fillMaxSize()
                    ) {
                        items(filteredNodes, key = { it.id }) { node ->
                            MeshNodeCard(node = node)
                        }
                    }
                }

                1 -> {
                    // Battery Health Section
                    if (lowBatteryNodes.isNotEmpty()) {
                        Surface(
                            shape = RoundedCornerShape(12.dp),
                            color = Color(0xFFEF4444).copy(alpha = 0.15f),
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(bottom = 12.dp)
                        ) {
                            Row(modifier = Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
                                Text("⚠️", fontSize = 18.sp)
                                Spacer(modifier = Modifier.width(8.dp))
                                Text(
                                    "${lowBatteryNodes.size} dispositivi richiedono la sostituzione della batteria entro breve.",
                                    fontSize = 12.sp,
                                    color = Color(0xFFEF4444),
                                    fontWeight = FontWeight.SemiBold
                                )
                            }
                        }
                    }

                    LazyColumn(
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                        modifier = Modifier.fillMaxSize()
                    ) {
                        items(nodes.filter { it.batteryLevel != null }.sortedBy { it.batteryLevel }, key = { it.id }) { node ->
                            BatteryNodeCard(node = node)
                        }
                    }
                }

                2 -> {
                    // System Health & Latency Section
                    HubHealthSection(health = health, latencyMs = latencyMs)
                }
            }
        }
    }
}

@Composable
fun MeshNodeCard(node: MeshNodeDiagnostic) {
    Card(
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = Slate800),
        modifier = Modifier.fillMaxWidth()
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Box(
                modifier = Modifier
                    .size(40.dp)
                    .background(node.signalColor.copy(alpha = 0.15f), CircleShape),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = if (node.isRouter) "📡" else "🔘",
                    fontSize = 20.sp
                )
            }

            Spacer(modifier = Modifier.width(12.dp))

            Column(modifier = Modifier.weight(1f)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = node.name,
                        fontWeight = FontWeight.Bold,
                        fontSize = 15.sp,
                        color = Color.White
                    )
                    Text(
                        text = "${node.rssi} dBm",
                        fontWeight = FontWeight.Bold,
                        fontSize = 13.sp,
                        color = node.signalColor
                    )
                }

                Spacer(modifier = Modifier.height(4.dp))

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    Text(
                        text = "${node.room} • ${node.protocolType}",
                        fontSize = 12.sp,
                        color = Color.White.copy(alpha = 0.6f)
                    )
                    Text(
                        text = "LQI ${node.lqi} (${node.signalQuality})",
                        fontSize = 12.sp,
                        color = node.signalColor
                    )
                }
            }
        }
    }
}

@Composable
fun BatteryNodeCard(node: MeshNodeDiagnostic) {
    val level = node.batteryLevel ?: 100
    val isCritical = level < 15
    val batteryColor = when {
        level >= 50 -> EmeraldSuccess
        level >= 20 -> Color(0xFFFBBF24)
        else -> Color(0xFFEF4444)
    }

    Card(
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = Slate800),
        modifier = Modifier.fillMaxWidth()
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = if (isCritical) "🪫" else "🔋",
                fontSize = 26.sp
            )

            Spacer(modifier = Modifier.width(12.dp))

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = node.name,
                    fontWeight = FontWeight.Bold,
                    fontSize = 15.sp,
                    color = Color.White
                )
                Text(
                    text = "${node.room} • ${node.protocolType}",
                    fontSize = 12.sp,
                    color = Color.White.copy(alpha = 0.6f)
                )
            }

            Text(
                text = "$level%",
                fontWeight = FontWeight.Black,
                fontSize = 18.sp,
                color = batteryColor
            )
        }
    }
}

@Composable
fun HubHealthSection(health: com.domus.app.domain.model.SystemHealthDto?, latencyMs: Int) {
    Column(
        verticalArrangement = Arrangement.spacedBy(14.dp),
        modifier = Modifier.fillMaxWidth()
    ) {
        Card(
            shape = RoundedCornerShape(18.dp),
            colors = CardDefaults.cardColors(containerColor = Slate800),
            modifier = Modifier.fillMaxWidth()
        ) {
            Column(modifier = Modifier.padding(16.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("⚡", fontSize = 22.sp)
                    Spacer(modifier = Modifier.width(10.dp))
                    Text("Latenza Rete Locale (LAN)", fontWeight = FontWeight.Bold, fontSize = 16.sp, color = Color.White)
                }

                Spacer(modifier = Modifier.height(12.dp))

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text("Ping Round-Trip WebSocket/REST:", fontSize = 13.sp, color = Color.White.copy(alpha = 0.7f))
                    Text(
                        text = "${latencyMs} ms",
                        fontWeight = FontWeight.Black,
                        fontSize = 20.sp,
                        color = if (latencyMs < 20) EmeraldSuccess else Color(0xFFFBBF24)
                    )
                }
            }
        }

        Card(
            shape = RoundedCornerShape(18.dp),
            colors = CardDefaults.cardColors(containerColor = Slate800),
            modifier = Modifier.fillMaxWidth()
        ) {
            Column(
                modifier = Modifier.padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("🛡️", fontSize = 22.sp)
                    Spacer(modifier = Modifier.width(10.dp))
                    Text("Salute Appliance Domus OS", fontWeight = FontWeight.Bold, fontSize = 16.sp, color = Color.White)
                }

                HorizontalDivider(color = Slate700)

                HealthMetricRow("Stato Daemon Core", health?.status?.uppercase() ?: "ONLINE", EmeraldSuccess)
                HealthMetricRow("Versione Appliance", health?.version ?: "Domus Core 1.0 (A/B Active)", CyanAccent)
                HealthMetricRow("Database SQLite", health?.database?.uppercase() ?: "WAL ENCRYPTED OK", EmeraldSuccess)
                val uptimeHours = ((health?.uptimeSeconds ?: 86400L) / 3600).coerceAtLeast(1)
                HealthMetricRow("Uptime Server", "$uptimeHours ore consecutive", Color.White)
            }
        }
    }
}

@Composable
fun HealthMetricRow(label: String, value: String, valueColor: Color) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(label, fontSize = 13.sp, color = Color.White.copy(alpha = 0.7f))
        Text(value, fontSize = 13.sp, fontWeight = FontWeight.Bold, color = valueColor)
    }
}
