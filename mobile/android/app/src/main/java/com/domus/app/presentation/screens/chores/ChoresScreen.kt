package com.domus.app.presentation.screens.chores

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.animateColorAsState
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
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.presentation.theme.AmberWarning
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.IndigoPrimary
import com.domus.app.presentation.theme.RoseDanger
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.theme.Slate900
import com.domus.app.presentation.viewmodel.DashboardViewModel

enum class ChorePriority(val label: String, val color: Color) {
    URGENT("Urgente", RoseDanger),
    HIGH("Alta", AmberWarning),
    MEDIUM("Media", CyanAccent),
    LOW("Bassa", Slate700)
}

data class Chore(
    val id: String,
    val title: String,
    val room: String,
    val assignee: String,
    val points: Int,
    val priority: ChorePriority,
    val frequency: String,
    var isDone: Boolean = false
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChoresScreen(
    viewModel: DashboardViewModel? = null,
    onDismiss: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    LaunchedEffect(Unit) {
        viewModel?.loadChores()
    }

    val serverChores by viewModel?.chores?.collectAsState() ?: remember { mutableStateOf(emptyList()) }

    val chores = remember(serverChores) {
        serverChores.map { dto ->
            Chore(
                id = dto.id,
                title = dto.title,
                room = dto.area_id.ifBlank { "Casa" },
                assignee = dto.assigned_user_id.ifBlank { "Tutti" },
                points = when (dto.priority.lowercase()) {
                    "urgent" -> 50
                    "high" -> 30
                    "medium" -> 20
                    else -> 10
                },
                priority = when (dto.priority.lowercase()) {
                    "urgent" -> ChorePriority.URGENT
                    "high" -> ChorePriority.HIGH
                    "medium" -> ChorePriority.MEDIUM
                    else -> ChorePriority.LOW
                },
                frequency = dto.recurrence_rule.ifBlank { if (dto.recurrence_type != "none") dto.recurrence_type else "Una tantum" },
                isDone = dto.status == "completed"
            )
        }
    }

    var selectedFilter by remember { mutableStateOf("Tutti") }
    val dynamicFilters = remember(chores) {
        val list = mutableListOf("Tutti", "Da Fare", "Completati")
        val assignees = chores.map { it.assignee }.filter { it.isNotBlank() && it != "Tutti" }.distinct()
        list.addAll(assignees)
        val rooms = chores.map { it.room }.filter { it.isNotBlank() && it != "Casa" }.distinct()
        list.addAll(rooms)
        list
    }

    val totalPoints = chores.filter { it.isDone }.sumOf { it.points }
    val pendingCount = chores.count { !it.isDone }

    val userLeaderboard = remember(chores) {
        val completed = chores.filter { it.isDone && it.assignee.isNotBlank() && it.assignee != "Tutti" }
        completed.groupBy { it.assignee }
            .map { (user, userChores) -> user to userChores.sumOf { it.points } }
            .sortedByDescending { it.second }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(Slate900)
    ) {
        // Top Header
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            IconButton(onClick = onDismiss) {
                Text("✕", fontSize = 20.sp, color = Color.White)
            }

            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = "Faccende Domestiche",
                    fontWeight = FontWeight.Bold,
                    fontSize = 18.sp,
                    color = Color.White
                )
                Text(
                    text = "$pendingCount attività in attesa",
                    fontSize = 12.sp,
                    color = AmberWarning
                )
            }

            // Family Karma Points Badge
            Surface(
                shape = RoundedCornerShape(12.dp),
                color = AmberWarning.copy(alpha = 0.15f),
                border = BorderStroke(1.dp, AmberWarning)
            ) {
                Row(
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text("⭐", fontSize = 12.sp)
                    Spacer(modifier = Modifier.width(4.dp))
                    Text(
                        text = "$totalPoints pt",
                        color = AmberWarning,
                        fontWeight = FontWeight.Bold,
                        fontSize = 13.sp
                    )
                }
            }
        }

        // Leaderboard Karma Row
        Card(
            shape = RoundedCornerShape(16.dp),
            colors = CardDefaults.cardColors(containerColor = Slate800),
            border = BorderStroke(1.dp, Slate700),
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 6.dp)
        ) {
            if (userLeaderboard.isEmpty()) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(14.dp),
                    horizontalArrangement = Arrangement.Center,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "🏆 Nessun punto karma accumulato",
                        color = Color.White.copy(alpha = 0.6f),
                        fontSize = 13.sp
                    )
                }
            } else {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(14.dp),
                    horizontalArrangement = Arrangement.SpaceAround,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    userLeaderboard.take(3).forEachIndexed { index, (user, points) ->
                        if (index > 0) {
                            Box(modifier = Modifier.height(28.dp).width(1.dp).background(Slate700))
                        }
                        Column(horizontalAlignment = Alignment.CenterHorizontally) {
                            Text(
                                text = "👤 $user",
                                fontWeight = FontWeight.Bold,
                                color = Color.White,
                                fontSize = 14.sp
                            )
                            Text(
                                text = if (index == 0) "$points Karma 🏆" else "$points Karma",
                                color = if (index == 0) AmberWarning else CyanAccent,
                                fontWeight = FontWeight.SemiBold,
                                fontSize = 12.sp
                            )
                        }
                    }
                }
            }
        }

        // Filter Pills Row
        Row(
            modifier = Modifier
                .horizontalScroll(rememberScrollState())
                .padding(horizontal = 16.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            dynamicFilters.forEach { filter ->
                val isSelected = selectedFilter == filter
                Surface(
                    shape = RoundedCornerShape(20.dp),
                    color = if (isSelected) IndigoPrimary else Slate800,
                    border = BorderStroke(1.dp, if (isSelected) CyanAccent else Slate700),
                    modifier = Modifier.clickable { selectedFilter = filter }
                ) {
                    Text(
                        text = filter,
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = if (isSelected) Color.White else Color.White.copy(alpha = 0.6f),
                        modifier = Modifier.padding(horizontal = 14.dp, vertical = 6.dp)
                    )
                }
            }
        }

        // Chores LazyColumn or Empty State
        val filteredChores = chores.filter {
            when (selectedFilter) {
                "Tutti" -> true
                "Da Fare" -> !it.isDone
                "Completati" -> it.isDone
                else -> it.assignee == selectedFilter || it.room == selectedFilter
            }
        }

        if (filteredChores.isEmpty()) {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(32.dp),
                contentAlignment = Alignment.Center
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text("✓", fontSize = 48.sp, color = Color.White.copy(alpha = 0.3f))
                    Spacer(modifier = Modifier.height(12.dp))
                    Text(
                        text = "Nessuna faccenda presente",
                        fontSize = 16.sp,
                        fontWeight = FontWeight.Medium,
                        color = Color.White.copy(alpha = 0.6f)
                    )
                }
            }
        } else {
            LazyColumn(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(horizontal = 16.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
                contentPadding = PaddingValues(vertical = 10.dp)
            ) {
            items(filteredChores, key = { it.id }) { chore ->
                val isDone = chore.isDone
                val cardColor by animateColorAsState(
                    targetValue = if (isDone) Slate800.copy(alpha = 0.4f) else Slate800,
                    label = "chore_bg"
                )

                Card(
                    shape = RoundedCornerShape(16.dp),
                    colors = CardDefaults.cardColors(containerColor = cardColor),
                    border = BorderStroke(
                        1.dp,
                        if (isDone) Color.Transparent else chore.priority.color.copy(alpha = 0.4f)
                    ),
                    modifier = Modifier
                        .fillMaxWidth()
                        .semantics {
                            role = Role.Checkbox
                            contentDescription = "${chore.title}, stanza ${chore.room}, assegnato a ${chore.assignee}, ${if (isDone) "completato" else "da fare"}"
                        }
                        .clickable {
                            viewModel?.completeChore(chore.id)
                        }
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(14.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        // Checkbox
                        Surface(
                            shape = CircleShape,
                            color = if (isDone) EmeraldSuccess else Color.Transparent,
                            border = BorderStroke(
                                2.dp,
                                if (isDone) EmeraldSuccess else Color.White.copy(alpha = 0.4f)
                            ),
                            modifier = Modifier.size(26.dp)
                        ) {
                            if (isDone) {
                                Box(contentAlignment = Alignment.Center) {
                                    Text("✓", color = Color.White, fontSize = 14.sp, fontWeight = FontWeight.Bold)
                                }
                            }
                        }

                        Spacer(modifier = Modifier.width(12.dp))

                        Column(modifier = Modifier.weight(1f)) {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text(
                                    text = chore.title,
                                    fontSize = 15.sp,
                                    fontWeight = if (isDone) FontWeight.Normal else FontWeight.Bold,
                                    color = if (isDone) Color.White.copy(alpha = 0.4f) else Color.White,
                                    textDecoration = if (isDone) TextDecoration.LineThrough else TextDecoration.None
                                )
                                Spacer(modifier = Modifier.width(8.dp))
                                Surface(
                                    shape = RoundedCornerShape(6.dp),
                                    color = chore.priority.color.copy(alpha = 0.15f)
                                ) {
                                    Text(
                                        text = chore.priority.label,
                                        color = chore.priority.color,
                                        fontSize = 10.sp,
                                        fontWeight = FontWeight.Bold,
                                        modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                                    )
                                }
                            }

                            Spacer(modifier = Modifier.height(4.dp))

                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text(
                                    text = "📍 ${chore.room} • 👤 ${chore.assignee} • ⏱️ ${chore.frequency}",
                                    fontSize = 12.sp,
                                    color = Color.White.copy(alpha = 0.5f)
                                )
                            }
                        }

                        // Points Tag
                        Surface(
                            shape = RoundedCornerShape(10.dp),
                            color = IndigoPrimary.copy(alpha = 0.2f),
                            border = BorderStroke(1.dp, IndigoPrimary.copy(alpha = 0.4f))
                        ) {
                            Text(
                                text = "+${chore.points} pt",
                                fontSize = 11.sp,
                                fontWeight = FontWeight.Bold,
                                color = CyanAccent,
                                modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                            )
                        }
                    }
                }
            }
        }
    }
}
}
