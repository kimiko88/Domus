package com.domus.app.presentation.screens.automation

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import com.domus.app.domain.model.AutomationAction
import com.domus.app.domain.model.AutomationCondition
import com.domus.app.domain.model.AutomationRule
import com.domus.app.domain.model.AutomationTrigger
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.IndigoPrimary
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.viewmodel.DashboardViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AutomationBuilderScreen(
    viewModel: DashboardViewModel,
    onDismiss: () -> Unit
) {
    val automations by viewModel.automations.collectAsState()
    var showCreateDialog by remember { mutableStateOf(false) }
    var testFeedbackMessage by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(Unit) {
        viewModel.loadAutomations()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text("Regole & Automazioni", fontWeight = FontWeight.Bold, fontSize = 20.sp)
                        Text("Se accade X con condizione Y allora fai Z", fontSize = 12.sp, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f))
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onDismiss) {
                        Text("✕", fontSize = 18.sp, fontWeight = FontWeight.Bold)
                    }
                },
                actions = {
                    Button(
                        onClick = { showCreateDialog = true },
                        colors = ButtonDefaults.buttonColors(containerColor = IndigoPrimary),
                        shape = RoundedCornerShape(12.dp)
                    ) {
                        Text("+ Nuova", fontWeight = FontWeight.Bold, fontSize = 13.sp)
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
                .padding(horizontal = 16.dp, vertical = 8.dp)
        ) {
            testFeedbackMessage?.let { msg ->
                Surface(
                    shape = RoundedCornerShape(12.dp),
                    color = EmeraldSuccess.copy(alpha = 0.15f),
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(bottom = 12.dp)
                ) {
                    Row(
                        modifier = Modifier.padding(12.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text("✅", fontSize = 16.sp)
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(msg, color = EmeraldSuccess, fontWeight = FontWeight.SemiBold, fontSize = 13.sp)
                    }
                }
            }

            if (automations.isEmpty()) {
                Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text("🪄", fontSize = 48.sp)
                        Spacer(modifier = Modifier.height(12.dp))
                        Text("Nessuna automazione trovata", fontWeight = FontWeight.Bold, fontSize = 16.sp)
                        Text("Crea la tua prima regola automatica per gestire la casa.", fontSize = 13.sp, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f))
                        Spacer(modifier = Modifier.height(16.dp))
                        Button(onClick = { showCreateDialog = true }) {
                            Text("Crea Automazione")
                        }
                    }
                }
            } else {
                LazyColumn(
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.fillMaxSize()
                ) {
                    items(automations, key = { it.id }) { rule ->
                        AutomationRuleCard(
                            rule = rule,
                            onToggleEnabled = { enabled ->
                                viewModel.updateAutomation(rule.copy(enabled = enabled))
                            },
                            onTriggerTest = {
                                viewModel.triggerAutomation(rule.id)
                                testFeedbackMessage = "Test inviato con successo per: ${rule.name}"
                            },
                            onDelete = {
                                viewModel.deleteAutomation(rule.id)
                            }
                        )
                    }
                }
            }
        }
    }

    if (showCreateDialog) {
        CreateAutomationDialog(
            onDismiss = { showCreateDialog = false },
            onCreate = { newRule ->
                viewModel.createAutomation(newRule)
                showCreateDialog = false
            }
        )
    }
}

@Composable
fun AutomationRuleCard(
    rule: AutomationRule,
    onToggleEnabled: (Boolean) -> Unit,
    onTriggerTest: () -> Unit,
    onDelete: () -> Unit
) {
    Card(
        shape = RoundedCornerShape(18.dp),
        colors = CardDefaults.cardColors(containerColor = Slate800),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Box(
                        modifier = Modifier
                            .size(36.dp)
                            .background(if (rule.enabled) CyanAccent.copy(alpha = 0.2f) else Color.Gray.copy(alpha = 0.2f), CircleShape),
                        contentAlignment = Alignment.Center
                    ) {
                        Text(if (rule.enabled) "🪄" else "⏸️", fontSize = 18.sp)
                    }
                    Spacer(modifier = Modifier.width(10.dp))
                    Column {
                        Text(
                            text = rule.name,
                            fontWeight = FontWeight.Bold,
                            fontSize = 15.sp,
                            color = Color.White
                        )
                        rule.description?.let {
                            Text(it, fontSize = 12.sp, color = Color.White.copy(alpha = 0.6f), maxLines = 1)
                        }
                    }
                }

                Switch(
                    checked = rule.enabled,
                    onCheckedChange = onToggleEnabled,
                    colors = SwitchDefaults.colors(
                        checkedThumbColor = Color.White,
                        checkedTrackColor = CyanAccent
                    )
                )
            }

            Spacer(modifier = Modifier.height(14.dp))

            // Triggers section
            Text("TRIGGER", fontSize = 10.sp, fontWeight = FontWeight.Bold, color = CyanAccent)
            rule.triggers.forEach { trigger ->
                Text("• ${trigger.displayIcon} ${trigger.displayTitle}", fontSize = 12.sp, color = Color.White.copy(alpha = 0.85f))
            }

            // Conditions section
            if (rule.conditions.isNotEmpty()) {
                Spacer(modifier = Modifier.height(6.dp))
                Text("CONDIZIONI", fontSize = 10.sp, fontWeight = FontWeight.Bold, color = Color(0xFFFBBF24))
                rule.conditions.forEach { cond ->
                    Text("• ${cond.displayIcon} ${cond.displayTitle}", fontSize = 12.sp, color = Color.White.copy(alpha = 0.85f))
                }
            }

            // Actions section
            if (rule.actions.isNotEmpty()) {
                Spacer(modifier = Modifier.height(6.dp))
                Text("AZIONI", fontSize = 10.sp, fontWeight = FontWeight.Bold, color = EmeraldSuccess)
                rule.actions.forEach { act ->
                    Text("• ${act.displayIcon} ${act.displayTitle}", fontSize = 12.sp, color = Color.White.copy(alpha = 0.85f))
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            // Action Buttons (Test / Delete)
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.End,
                verticalAlignment = Alignment.CenterVertically
            ) {
                OutlinedButton(
                    onClick = onTriggerTest,
                    shape = RoundedCornerShape(10.dp),
                    contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp)
                ) {
                    Text("▶ Test", fontSize = 12.sp)
                }

                Spacer(modifier = Modifier.width(8.dp))

                IconButton(onClick = onDelete) {
                    Text("🗑️", fontSize = 14.sp)
                }
            }
        }
    }
}

@Composable
fun CreateAutomationDialog(
    onDismiss: () -> Unit,
    onCreate: (AutomationRule) -> Unit
) {
    var name by remember { mutableStateOf("") }
    var description by remember { mutableStateOf("") }
    var triggerType by remember { mutableStateOf("time") } // time, state, geofence
    var triggerTime by remember { mutableStateOf("07:30") }
    var requireAlarmDisarmed by remember { mutableStateOf(false) }
    var actionService by remember { mutableStateOf("turn_on") }
    var actionTarget by remember { mutableStateOf("light.living_room") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Nuova Regola Automatica", fontWeight = FontWeight.Bold) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    label = { Text("Nome Regola") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth()
                )

                OutlinedTextField(
                    value = description,
                    onValueChange = { description = it },
                    label = { Text("Descrizione") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth()
                )

                Text("Innesco (Trigger):", fontWeight = FontWeight.SemiBold, fontSize = 13.sp)
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    listOf("time" to "Orario", "state" to "Sensore", "geofence" to "Posizione").forEach { (type, label) ->
                        FilterChip(
                            selected = triggerType == type,
                            onClick = { triggerType = type },
                            label = { Text(label, fontSize = 12.sp) }
                        )
                    }
                }

                if (triggerType == "time") {
                    OutlinedTextField(
                        value = triggerTime,
                        onValueChange = { triggerTime = it },
                        label = { Text("Orario (HH:mm)") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth()
                    )
                }

                Text("Azione:", fontWeight = FontWeight.SemiBold, fontSize = 13.sp)
                OutlinedTextField(
                    value = actionTarget,
                    onValueChange = { actionTarget = it },
                    label = { Text("Entità Target (es. light.living_room)") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth()
                )
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    val triggers = when (triggerType) {
                        "time" -> listOf(AutomationTrigger(platform = "time", at = "$triggerTime:00"))
                        "geofence" -> listOf(AutomationTrigger(platform = "geofence", eventType = "leave_home"))
                        else -> listOf(AutomationTrigger(platform = "state", entityKey = "binary_sensor.motion", to = "on"))
                    }
                    val conditions = if (requireAlarmDisarmed) {
                        listOf(AutomationCondition(condition = "state", entityKey = "alarm_control_panel.home", state = "disarmed"))
                    } else emptyList()
                    val actions = listOf(
                        AutomationAction(type = "service", service = actionService, targetEntityKey = actionTarget)
                    )

                    onCreate(
                        AutomationRule(
                            name = name.ifBlank { "Nuova Automazione" },
                            description = description.ifBlank { null },
                            enabled = true,
                            triggers = triggers,
                            conditions = conditions,
                            actions = actions
                        )
                    )
                },
                enabled = name.isNotBlank()
            ) {
                Text("Salva Regola")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Annulla")
            }
        }
    )
}
