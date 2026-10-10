package com.domus.app.presentation.screens

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
import com.domus.app.presentation.components.ConnectionStatusBanner
import com.domus.app.presentation.components.EntityCard
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
                    .padding(vertical = 8.dp)
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
                            onToggle = { viewModel.toggleEntity(it) }
                        )
                    }
                }
            }
        }
    }
}
