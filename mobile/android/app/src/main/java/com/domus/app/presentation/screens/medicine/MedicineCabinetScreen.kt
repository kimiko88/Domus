package com.domus.app.presentation.screens.medicine

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
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
fun MedicineCabinetScreen(
    viewModel: DashboardViewModel,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier
) {
    LaunchedEffect(Unit) {
        viewModel.loadMedicines()
        viewModel.loadMedicineAlerts()
    }

    val medicines by viewModel.medicines.collectAsState()
    val alerts by viewModel.medicineAlerts.collectAsState()
    val isLoading by viewModel.isMedicineLoading.collectAsState()

    var searchQuery by remember { mutableStateOf("") }
    var selectedFilter by remember { mutableStateOf("all") }
    var selectedLocation by remember { mutableStateOf<String?>(null) }
    var showAddDialog by remember { mutableStateOf(false) }

    val filteredMedicines = remember(medicines, searchQuery, selectedFilter, selectedLocation) {
        medicines.filter { med ->
            val queryMatch = searchQuery.isBlank() ||
                    med.name.contains(searchQuery, ignoreCase = true) ||
                    (med.active_ingredient?.contains(searchQuery, ignoreCase = true) == true) ||
                    (med.aic?.contains(searchQuery, ignoreCase = true) == true)

            val locationMatch = selectedLocation == null || med.location.equals(selectedLocation, ignoreCase = true)

            val statusMatch = when (selectedFilter) {
                "expiring" -> true // Handled by alert/effective expiration
                "low_stock" -> med.isLowStock
                "cold_chain" -> med.requires_cold_chain || med.location.equals("frigo", ignoreCase = true)
                else -> true
            }
            queryMatch && locationMatch && statusMatch
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        text = stringResource(R.string.medicine_cabinet_title),
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
                        onClick = { showAddDialog = true },
                        modifier = Modifier.size(48.dp)
                    ) {
                        Text("＋", fontSize = 22.sp, color = AccentBlue)
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
            // Banner Avvisi Critici
            if (alerts.isNotEmpty()) {
                item {
                    MedicineAlertsBanner(alerts = alerts)
                }
            }

            // Ricerca e Filtri
            item {
                MedicineSearchAndFilters(
                    searchQuery = searchQuery,
                    onSearchQueryChange = { searchQuery = it },
                    selectedFilter = selectedFilter,
                    onFilterChange = { selectedFilter = it },
                    selectedLocation = selectedLocation,
                    onLocationChange = { selectedLocation = it }
                )
            }

            // Elenco Farmaci
            if (isLoading && medicines.isEmpty()) {
                item {
                    Box(modifier = Modifier.fillMaxWidth().padding(48.dp), contentAlignment = Alignment.Center) {
                        CircularProgressIndicator(color = AccentBlue)
                    }
                }
            } else if (filteredMedicines.isEmpty()) {
                item {
                    MedicineEmptyState()
                }
            } else {
                items(filteredMedicines, key = { it.id }) { med ->
                    MedicineCard(
                        medicine = med,
                        onConsumeDose = { viewModel.consumeMedicine(med.id, 1.0) },
                        onOpenPackage = { viewModel.openMedicine(med.id) }
                    )
                }
            }

            item { Spacer(modifier = Modifier.height(24.dp)) }
        }
    }

    if (showAddDialog) {
        AddMedicineDialog(
            onDismiss = { showAddDialog = false },
            onSave = { req ->
                viewModel.createMedicine(req) { success ->
                    if (success) showAddDialog = false
                }
            }
        )
    }
}

@Composable
private fun MedicineAlertsBanner(alerts: List<MedicineAlertDto>) {
    Card(
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = Color(0xFF331414)),
        border = BorderStroke(1.dp, Color(0xFFEF4444).copy(alpha = 0.4f)),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(
                    imageVector = Icons.Default.Warning,
                    contentDescription = null,
                    tint = Color(0xFFEF4444)
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text(
                    text = stringResource(R.string.medicine_alerts_banner_title),
                    fontWeight = FontWeight.Bold,
                    color = Color.White,
                    fontSize = 16.sp
                )
                Spacer(modifier = Modifier.weight(1f))
                Surface(
                    shape = CircleShape,
                    color = Color(0xFFEF4444).copy(alpha = 0.2f)
                ) {
                    Text(
                        text = "${alerts.size}",
                        color = Color(0xFFEF4444),
                        fontWeight = FontWeight.Bold,
                        fontSize = 12.sp,
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp)
                    )
                }
            }

            alerts.take(3).forEach { alert ->
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Box(
                        modifier = Modifier
                            .size(6.dp)
                            .background(if (alert.isCritical) Color(0xFFEF4444) else Color(0xFFF59E0B), CircleShape)
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(
                        text = alert.message,
                        color = Color.White.copy(alpha = 0.8f),
                        fontSize = 13.sp
                    )
                }
            }
        }
    }
}

@Composable
private fun MedicineSearchAndFilters(
    searchQuery: String,
    onSearchQueryChange: (String) -> Unit,
    selectedFilter: String,
    onFilterChange: (String) -> Unit,
    selectedLocation: String?,
    onLocationChange: (String?) -> Unit
) {
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        // Barra di Ricerca
        OutlinedTextField(
            value = searchQuery,
            onValueChange = onSearchQueryChange,
            placeholder = { Text(stringResource(R.string.medicine_search_placeholder), color = Color.Gray, fontSize = 14.sp) },
            leadingIcon = { Text("🔍", fontSize = 14.sp) },
            trailingIcon = {
                if (searchQuery.isNotEmpty()) {
                    IconButton(onClick = { onSearchQueryChange("") }) {
                        Text("✕", fontSize = 16.sp, color = Color.Gray)
                    }
                }
            },
            singleLine = true,
            shape = RoundedCornerShape(12.dp),
            colors = OutlinedTextFieldDefaults.colors(
                focusedContainerColor = DomusSurfaceDark,
                unfocusedContainerColor = DomusSurfaceDark,
                focusedBorderColor = AccentBlue,
                unfocusedBorderColor = Color.White.copy(alpha = 0.15f),
                focusedTextColor = Color.White,
                unfocusedTextColor = Color.White
            ),
            modifier = Modifier.fillMaxWidth()
        )

        // Filtri di stato
        Row(
            modifier = Modifier.horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            FilterChipItem(
                label = stringResource(R.string.common_all),
                selected = selectedFilter == "all",
                onClick = { onFilterChange("all") }
            )
            FilterChipItem(
                label = stringResource(R.string.medicine_filter_expiring),
                selected = selectedFilter == "expiring",
                onClick = { onFilterChange("expiring") }
            )
            FilterChipItem(
                label = stringResource(R.string.medicine_filter_low_stock),
                selected = selectedFilter == "low_stock",
                onClick = { onFilterChange("low_stock") }
            )
            FilterChipItem(
                label = stringResource(R.string.medicine_filter_cold_chain),
                selected = selectedFilter == "cold_chain",
                onClick = { onFilterChange("cold_chain") }
            )
        }

        // Filtri posizione
        Row(
            modifier = Modifier.horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            LocationChipItem(
                label = stringResource(R.string.medicine_all_locations),
                selected = selectedLocation == null,
                onClick = { onLocationChange(null) }
            )
            MedicineLocation.entries.forEach { loc ->
                val label = when (loc) {
                    MedicineLocation.ARMADIETTO_BAGNO -> stringResource(R.string.medicine_loc_bathroom)
                    MedicineLocation.CASSETTO_NOTTE -> stringResource(R.string.medicine_loc_nightstand)
                    MedicineLocation.BORSA_VIAGGIO -> stringResource(R.string.medicine_loc_travel_bag)
                    MedicineLocation.FRIGO -> stringResource(R.string.medicine_loc_fridge)
                    MedicineLocation.ALTRO -> stringResource(R.string.medicine_loc_other)
                }
                LocationChipItem(
                    label = label,
                    selected = selectedLocation.equals(loc.value, ignoreCase = true),
                    onClick = {
                        onLocationChange(if (selectedLocation == loc.value) null else loc.value)
                    }
                )
            }
        }
    }
}

@Composable
private fun FilterChipItem(label: String, selected: Boolean, onClick: () -> Unit) {
    Surface(
        shape = CircleShape,
        color = if (selected) AccentBlue else DomusSurfaceDark,
        border = BorderStroke(1.dp, if (selected) AccentBlue else Color.White.copy(alpha = 0.15f)),
        modifier = Modifier
            .clickable(onClick = onClick)
            .semantics { role = Role.Button }
    ) {
        Text(
            text = label,
            color = if (selected) Color.White else Color.White.copy(alpha = 0.8f),
            fontWeight = if (selected) FontWeight.Bold else FontWeight.Normal,
            fontSize = 12.sp,
            modifier = Modifier.padding(horizontal = 14.dp, vertical = 8.dp)
        )
    }
}

@Composable
private fun LocationChipItem(label: String, selected: Boolean, onClick: () -> Unit) {
    Surface(
        shape = CircleShape,
        color = if (selected) Color.White.copy(alpha = 0.2f) else DomusSurfaceDark,
        border = BorderStroke(1.dp, Color.White.copy(alpha = 0.1f)),
        modifier = Modifier
            .clickable(onClick = onClick)
            .semantics { role = Role.Button }
    ) {
        Text(
            text = label,
            color = Color.White.copy(alpha = 0.85f),
            fontSize = 11.sp,
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp)
        )
    }
}

@Composable
private fun MedicineCard(
    medicine: MedicineDto,
    onConsumeDose: () -> Unit,
    onOpenPackage: () -> Unit
) {
    Card(
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = DomusSurfaceDark),
        border = BorderStroke(1.dp, Color.White.copy(alpha = 0.08f)),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            // Header: Nome e Badge Scadenza
            Row(verticalAlignment = Alignment.Top) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = medicine.name,
                        fontWeight = FontWeight.Bold,
                        color = Color.White,
                        fontSize = 16.sp
                    )
                    if (!medicine.active_ingredient.isNullOrBlank()) {
                        Text(
                            text = medicine.active_ingredient,
                            color = Color.Gray,
                            fontSize = 13.sp
                        )
                    }
                    if (!medicine.dosage.isNullOrBlank()) {
                        Text(
                            text = medicine.dosage,
                            color = AccentBlue,
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold
                        )
                    }
                }

                Column(horizontalAlignment = Alignment.End) {
                    Surface(
                        shape = CircleShape,
                        color = if (medicine.isLowStock) Color(0xFFEF4444).copy(alpha = 0.2f) else Color(0xFF10B981).copy(alpha = 0.2f)
                    ) {
                        Text(
                            text = if (medicine.isLowStock) stringResource(R.string.medicine_status_low_stock) else stringResource(R.string.medicine_status_valid),
                            color = if (medicine.isLowStock) Color(0xFFEF4444) else Color(0xFF10B981),
                            fontWeight = FontWeight.Bold,
                            fontSize = 11.sp,
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp)
                        )
                    }

                    if (medicine.requires_cold_chain) {
                        Spacer(modifier = Modifier.height(4.dp))
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text("❄️", fontSize = 12.sp)
                            Spacer(modifier = Modifier.width(2.dp))
                            Text("Frigo", color = Color(0xFF06B6D4), fontSize = 11.sp, fontWeight = FontWeight.Bold)
                        }
                    }
                }
            }

            HorizontalDivider(color = Color.White.copy(alpha = 0.08f))

            // Metadati: Scadenza, Posizione, PAO
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Column {
                    Text(stringResource(R.string.medicine_field_expiration), color = Color.Gray, fontSize = 11.sp)
                    Text(medicine.effective_expiration_date, color = Color.White, fontWeight = FontWeight.SemiBold, fontSize = 13.sp)
                }

                Column {
                    Text(stringResource(R.string.medicine_field_location), color = Color.Gray, fontSize = 11.sp)
                    Text(medicine.location, color = Color.White, fontWeight = FontWeight.SemiBold, fontSize = 13.sp)
                }

                if (medicine.pao_days > 0) {
                    Column {
                        Text("PAO", color = Color.Gray, fontSize = 11.sp)
                        Text(
                            text = if (medicine.isOpened) "Aperto (${medicine.pao_days}gg)" else "Sigillato",
                            color = if (medicine.isOpened) Color(0xFFF59E0B) else Color.Gray,
                            fontWeight = FontWeight.SemiBold,
                            fontSize = 13.sp
                        )
                    }
                }
            }

            // Barra Quantità & Azioni
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "${medicine.quantity.toInt()} ${medicine.unit}",
                    color = if (medicine.isLowStock) Color(0xFFEF4444) else Color.White,
                    fontWeight = FontWeight.Bold,
                    fontSize = 16.sp
                )

                Spacer(modifier = Modifier.weight(1f))

                if (medicine.pao_days > 0 && !medicine.isOpened) {
                    OutlinedButton(
                        onClick = onOpenPackage,
                        shape = CircleShape,
                        colors = ButtonDefaults.outlinedButtonColors(contentColor = Color(0xFFF59E0B)),
                        border = BorderStroke(1.dp, Color(0xFFF59E0B).copy(alpha = 0.5f)),
                        modifier = Modifier.height(36.dp)
                    ) {
                        Text(stringResource(R.string.medicine_action_open), fontSize = 12.sp, fontWeight = FontWeight.Bold)
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                }

                Button(
                    onClick = onConsumeDose,
                    shape = CircleShape,
                    colors = ButtonDefaults.buttonColors(containerColor = AccentBlue),
                    modifier = Modifier.height(36.dp)
                ) {
                    Text(stringResource(R.string.medicine_action_take_dose), fontSize = 12.sp, fontWeight = FontWeight.Bold)
                }
            }
        }
    }
}

@Composable
private fun MedicineEmptyState() {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 48.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        Text("💊", fontSize = 42.sp)
        Text(
            text = stringResource(R.string.medicine_empty_state_title),
            color = Color.White,
            fontWeight = FontWeight.Bold,
            fontSize = 16.sp
        )
        Text(
            text = stringResource(R.string.medicine_empty_state_subtitle),
            color = Color.Gray,
            fontSize = 13.sp
        )
    }
}

@Composable
private fun AddMedicineDialog(
    onDismiss: () -> Unit,
    onSave: (CreateMedicineRequestDto) -> Unit
) {
    var name by remember { mutableStateOf("") }
    var activeIngredient by remember { mutableStateOf("") }
    var dosage by remember { mutableStateOf("") }
    var aic by remember { mutableStateOf("") }
    var expirationDate by remember { mutableStateOf("2027-12-31") }
    var quantity by remember { mutableStateOf("1") }
    var minThreshold by remember { mutableStateOf("1") }
    var paoDays by remember { mutableStateOf("0") }
    var location by remember { mutableStateOf("armadietto_bagno") }
    var unit by remember { mutableStateOf("compresse") }
    var requiresColdChain by remember { mutableStateOf(false) }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.medicine_add_title), color = Color.White, fontWeight = FontWeight.Bold) },
        text = {
            Column(
                modifier = Modifier.fillMaxWidth(),
                verticalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    label = { Text(stringResource(R.string.medicine_field_name)) },
                    modifier = Modifier.fillMaxWidth()
                )
                OutlinedTextField(
                    value = activeIngredient,
                    onValueChange = { activeIngredient = it },
                    label = { Text(stringResource(R.string.medicine_field_active)) },
                    modifier = Modifier.fillMaxWidth()
                )
                OutlinedTextField(
                    value = dosage,
                    onValueChange = { dosage = it },
                    label = { Text(stringResource(R.string.medicine_field_dosage)) },
                    modifier = Modifier.fillMaxWidth()
                )
                OutlinedTextField(
                    value = expirationDate,
                    onValueChange = { expirationDate = it },
                    label = { Text(stringResource(R.string.medicine_field_expiration) + " (YYYY-MM-DD)") },
                    modifier = Modifier.fillMaxWidth()
                )
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    val q = quantity.toDoubleOrNull() ?: 1.0
                    val minT = minThreshold.toDoubleOrNull() ?: 1.0
                    val pao = paoDays.toIntOrNull() ?: 0
                    onSave(
                        CreateMedicineRequestDto(
                            name = name,
                            active_ingredient = activeIngredient.ifBlank { null },
                            dosage = dosage.ifBlank { null },
                            aic = aic.ifBlank { null },
                            expiration_date = expirationDate,
                            pao_days = pao,
                            location = location,
                            requires_cold_chain = requiresColdChain,
                            quantity = q,
                            unit = unit,
                            min_threshold = minT
                        )
                    )
                },
                enabled = name.isNotBlank()
            ) {
                Text(stringResource(R.string.common_save))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(stringResource(R.string.common_cancel))
            }
        },
        containerColor = DomusSurfaceDark
    )
}
