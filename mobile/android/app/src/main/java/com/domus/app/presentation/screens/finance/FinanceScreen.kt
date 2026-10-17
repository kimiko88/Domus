package com.domus.app.presentation.screens.finance

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
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.domain.model.ExpenseDto
import com.domus.app.domain.model.FinancialSummaryDto
import com.domus.app.presentation.theme.*
import com.domus.app.presentation.viewmodel.DashboardViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FinanceScreen(
    viewModel: DashboardViewModel? = null,
    onDismiss: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    var selectedPeriod by remember { mutableStateOf("month") }
    var showAddDialog by remember { mutableStateOf(false) }

    LaunchedEffect(selectedPeriod) {
        viewModel?.loadFinance(selectedPeriod)
    }

    val summary by viewModel?.financeSummary?.collectAsState() ?: remember { mutableStateOf(null) }
    val expenses by viewModel?.financeExpenses?.collectAsState() ?: remember { mutableStateOf(emptyList()) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text(
                            text = "Bilancio & Spese Casa",
                            style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                            color = Color.White
                        )
                        Text(
                            text = "Controllo uscite e risparmio intelligente",
                            style = MaterialTheme.typography.bodySmall,
                            color = Slate400
                        )
                    }
                },
                actions = {
                    TextButton(
                        onClick = onDismiss,
                        modifier = Modifier.defaultMinSize(minWidth = 48.dp, minHeight = 48.dp)
                    ) {
                        Text("Chiudi", color = CyanAccent)
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = Slate900)
            )
        },
        floatingActionButton = {
            FloatingActionButton(
                onClick = { showAddDialog = true },
                containerColor = CyanAccent,
                contentColor = Slate900,
                modifier = Modifier.semantics {
                    contentDescription = "Registra Nuova Spesa"
                    role = Role.Button
                }
            ) {
                Text("+", fontSize = 24.sp, fontWeight = FontWeight.Bold)
            }
        },
        containerColor = Slate900,
        modifier = modifier
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Period selector
            item {
                PeriodSelector(
                    selectedPeriod = selectedPeriod,
                    onSelect = { selectedPeriod = it }
                )
            }

            // Top KPI Cards
            item {
                FinanceKpisRow(summary = summary)
            }

            // Smart Savings Insights (Chicche Domus)
            if (summary?.insights?.isNotEmpty() == true) {
                item {
                    Text(
                        text = "💡 Suggerimenti Intelligenti di Risparmio",
                        style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.Bold),
                        color = Color.White
                    )
                }
                items(summary?.insights ?: emptyList()) { insight ->
                    Card(
                        shape = RoundedCornerShape(12.dp),
                        colors = CardDefaults.cardColors(containerColor = Slate800),
                        border = BorderStroke(
                            1.dp,
                            when (insight.severity) {
                                "alert" -> RoseDanger.copy(alpha = 0.5f)
                                "warning" -> AmberWarning.copy(alpha = 0.5f)
                                else -> CyanAccent.copy(alpha = 0.3f)
                            }
                        ),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Column(modifier = Modifier.padding(14.dp)) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(
                                    text = insight.title,
                                    fontWeight = FontWeight.SemiBold,
                                    fontSize = 14.sp,
                                    color = Color.White,
                                    modifier = Modifier.weight(1f)
                                )
                                insight.potential_saving?.let { saving ->
                                    if (saving > 0) {
                                        Surface(
                                            color = EmeraldSuccess.copy(alpha = 0.2f),
                                            shape = RoundedCornerShape(6.dp)
                                        ) {
                                            Text(
                                                text = "+€${"%.2f".format(saving)}",
                                                color = EmeraldSuccess,
                                                fontSize = 12.sp,
                                                fontWeight = FontWeight.Bold,
                                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                                            )
                                        }
                                    }
                                }
                            }
                            Spacer(modifier = Modifier.height(6.dp))
                            Text(
                                text = insight.message,
                                fontSize = 12.sp,
                                color = Slate400,
                                lineHeight = 16.sp
                            )
                        }
                    }
                }
            }

            // Category breakdown
            if (summary?.categories?.isNotEmpty() == true) {
                item {
                    Text(
                        text = "📊 Ripartizione Categorie",
                        style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.Bold),
                        color = Color.White
                    )
                }
                items(summary?.categories ?: emptyList()) { cat ->
                    Card(
                        shape = RoundedCornerShape(10.dp),
                        colors = CardDefaults.cardColors(containerColor = Slate800),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(12.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Column {
                                Text(cat.name, color = Color.White, fontSize = 14.sp, fontWeight = FontWeight.Medium)
                                Text("${cat.count} spese registrate", color = Slate400, fontSize = 11.sp)
                            }
                            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                Text("€${"%.2f".format(cat.total_amount)}", color = Color.White, fontWeight = FontWeight.Bold)
                                Surface(
                                    color = Slate700,
                                    shape = RoundedCornerShape(4.dp)
                                ) {
                                    Text(
                                        "${cat.percentage.toInt()}%",
                                        color = CyanAccent,
                                        fontSize = 11.sp,
                                        fontWeight = FontWeight.SemiBold,
                                        modifier = Modifier.padding(horizontal = 4.dp, vertical = 2.dp)
                                    )
                                }
                            }
                        }
                    }
                }
            }

            // Recent Expenses List
            item {
                Text(
                    text = "📝 Ultime Transazioni",
                    style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.Bold),
                    color = Color.White
                )
            }
            if (expenses.isEmpty()) {
                item {
                    Text("Nessuna spesa recente registrata.", color = Slate400, fontSize = 13.sp)
                }
            } else {
                items(expenses) { exp ->
                    ExpenseRowCard(expense = exp)
                }
            }

            item {
                Spacer(modifier = Modifier.height(72.dp))
            }
        }
    }

    if (showAddDialog) {
        AddExpenseDialog(
            onDismiss = { showAddDialog = false },
            onConfirm = { title, amount, category, isFixed, kwh ->
                viewModel?.addExpense(
                    title = title,
                    amount = amount,
                    category = category,
                    isFixed = isFixed,
                    kwh = kwh
                )
                showAddDialog = false
            }
        )
    }
}

@Composable
fun PeriodSelector(selectedPeriod: String, onSelect: (String) -> Unit) {
    val periods = listOf(
        "day" to "Giorno",
        "week" to "Settimana",
        "month" to "Mese",
        "year" to "Anno"
    )
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState()),
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        periods.forEach { (key, label) ->
            val isSelected = selectedPeriod == key
            Surface(
                color = if (isSelected) CyanAccent else Slate800,
                shape = RoundedCornerShape(8.dp),
                border = BorderStroke(1.dp, if (isSelected) CyanAccent else Slate700),
                modifier = Modifier
                    .defaultMinSize(minHeight = 44.dp)
                    .clickable { onSelect(key) }
                    .semantics {
                        contentDescription = "Periodo $label"
                        role = Role.Button
                    }
            ) {
                Box(
                    modifier = Modifier.padding(horizontal = 14.dp, vertical = 10.dp),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = label,
                        color = if (isSelected) Slate900 else Color.White,
                        fontWeight = if (isSelected) FontWeight.Bold else FontWeight.Medium,
                        fontSize = 13.sp
                    )
                }
            }
        }
    }
}

@Composable
fun FinanceKpisRow(summary: FinancialSummaryDto?) {
    val total = summary?.total_spent ?: 0.0
    val budget = summary?.monthly_budget ?: 0.0
    val remaining = summary?.remaining_budget ?: 0.0
    val isOver = budget > 0 && total > budget

    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            // Totale Speso
            Card(
                shape = RoundedCornerShape(12.dp),
                colors = CardDefaults.cardColors(containerColor = Slate800),
                modifier = Modifier.weight(1f)
            ) {
                Column(modifier = Modifier.padding(14.dp)) {
                    Text("Totale Speso", color = Slate400, fontSize = 12.sp)
                    Spacer(modifier = Modifier.height(4.dp))
                    Text("€${"%.2f".format(total)}", color = Color.White, fontSize = 20.sp, fontWeight = FontWeight.Bold)
                    summary?.percentage_change?.let { pct ->
                        Text(
                            text = if (pct <= 0) "↓ ${"%.1f".format(Math.abs(pct))}% vs prec" else "↑ ${"%.1f".format(pct)}% vs prec",
                            color = if (pct <= 0) EmeraldSuccess else RoseDanger,
                            fontSize = 11.sp,
                            fontWeight = FontWeight.SemiBold
                        )
                    }
                }
            }

            // Budget Mensile
            Card(
                shape = RoundedCornerShape(12.dp),
                colors = CardDefaults.cardColors(containerColor = Slate800),
                border = if (isOver) BorderStroke(1.dp, RoseDanger) else null,
                modifier = Modifier.weight(1f)
            ) {
                Column(modifier = Modifier.padding(14.dp)) {
                    Text("Budget Residuo", color = Slate400, fontSize = 12.sp)
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        text = "€${"%.2f".format(remaining)}",
                        color = if (isOver) RoseDanger else Color.White,
                        fontSize = 20.sp,
                        fontWeight = FontWeight.Bold
                    )
                    Text("Max: €${"%.0f".format(budget)}", color = Slate400, fontSize = 11.sp)
                }
            }
        }

        // Fisse vs Variabili
        summary?.fixed_vs_variable?.let { fv ->
            Card(
                shape = RoundedCornerShape(10.dp),
                colors = CardDefaults.cardColors(containerColor = Slate800),
                modifier = Modifier.fillMaxWidth()
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(10.dp),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    Text("Incomprimibili: €${"%.2f".format(fv.fixed_amount)} (${fv.fixed_percentage.toInt()}%)", color = AmberWarning, fontSize = 11.sp)
                    Text("Ottimizzabili: €${"%.2f".format(fv.variable_amount)} (${fv.variable_percentage.toInt()}%)", color = EmeraldSuccess, fontSize = 11.sp)
                }
            }
        }
    }
}

@Composable
fun ExpenseRowCard(expense: ExpenseDto) {
    Card(
        shape = RoundedCornerShape(10.dp),
        colors = CardDefaults.cardColors(containerColor = Slate800),
        modifier = Modifier.fillMaxWidth()
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(expense.title, color = Color.White, fontSize = 14.sp, fontWeight = FontWeight.SemiBold)
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                    Text(expense.category, color = CyanAccent, fontSize = 11.sp)
                    if (expense.is_fixed) {
                        Surface(color = AmberWarning.copy(alpha = 0.2f), shape = RoundedCornerShape(4.dp)) {
                            Text("Fisso", color = AmberWarning, fontSize = 10.sp, modifier = Modifier.padding(horizontal = 4.dp))
                        }
                    }
                    expense.energy_kwh?.let { kwh ->
                        if (kwh > 0) {
                            Text("${kwh.toInt()} kWh", color = Slate400, fontSize = 11.sp)
                        }
                    }
                }
            }
            Text("€${"%.2f".format(expense.amount)}", color = Color.White, fontSize = 15.sp, fontWeight = FontWeight.Bold)
        }
    }
}

@Composable
fun AddExpenseDialog(
    onDismiss: () -> Unit,
    onConfirm: (title: String, amount: Double, category: String, isFixed: Boolean, kwh: Double?) -> Unit
) {
    var title by remember { mutableStateOf("") }
    var amountStr by remember { mutableStateOf("") }
    var category by remember { mutableStateOf("food") }
    var isFixed by remember { mutableStateOf(false) }
    var kwhStr by remember { mutableStateOf("") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Registra Nuova Spesa", color = Color.White) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                OutlinedTextField(
                    value = title,
                    onValueChange = { title = it },
                    label = { Text("Descrizione") },
                    singleLine = true,
                    colors = OutlinedTextFieldDefaults.colors(focusedTextColor = Color.White, unfocusedTextColor = Color.White),
                    modifier = Modifier.fillMaxWidth()
                )
                OutlinedTextField(
                    value = amountStr,
                    onValueChange = { amountStr = it },
                    label = { Text("Importo (€)") },
                    singleLine = true,
                    colors = OutlinedTextFieldDefaults.colors(focusedTextColor = Color.White, unfocusedTextColor = Color.White),
                    modifier = Modifier.fillMaxWidth()
                )
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text("Costo Fisso (incomprimibile)", color = Slate400, fontSize = 12.sp)
                    Switch(checked = isFixed, onCheckedChange = { isFixed = it })
                }
                if (category == "utilities") {
                    OutlinedTextField(
                        value = kwhStr,
                        onValueChange = { kwhStr = it },
                        label = { Text("Consumo kWh") },
                        singleLine = true,
                        colors = OutlinedTextFieldDefaults.colors(focusedTextColor = Color.White, unfocusedTextColor = Color.White),
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    val amt = amountStr.toDoubleOrNull() ?: 0.0
                    val kwh = kwhStr.toDoubleOrNull()
                    if (title.isNotBlank() && amt > 0) {
                        onConfirm(title, amt, category, isFixed, kwh)
                    }
                },
                colors = ButtonDefaults.buttonColors(containerColor = CyanAccent, contentColor = Slate900)
            ) {
                Text("Salva", fontWeight = FontWeight.Bold)
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Annulla", color = Slate400)
            }
        },
        containerColor = Slate800
    )
}
