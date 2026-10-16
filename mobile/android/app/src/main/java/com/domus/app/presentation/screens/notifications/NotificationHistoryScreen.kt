package com.domus.app.presentation.screens.notifications

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
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
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

enum class NotificationCategory(val label: String, val icon: String, val color: Color) {
    ALL("Tutte", "🔔", CyanAccent),
    SECURITY("Sicurezza", "🛡️", RoseDanger),
    ENERGY("Energia", "⚡", AmberWarning),
    DEVICES("Dispositivi", "📱", CyanAccent),
    SYSTEM("Sistema", "⚙️", IndigoPrimary)
}

data class InAppNotification(
    val id: String,
    val title: String,
    val message: String,
    val category: NotificationCategory,
    val timeAgo: String,
    var isRead: Boolean = false,
    val actionLabel: String? = null
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun NotificationHistoryScreen(
    viewModel: DashboardViewModel? = null,
    onDismiss: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    LaunchedEffect(Unit) {
        viewModel?.loadNotifications()
    }

    val serverNotifs by viewModel?.notifications?.collectAsState() ?: remember { mutableStateOf(emptyList()) }

    var localReadIds by remember { mutableStateOf(setOf<String>()) }

    val notifications = remember(serverNotifs, localReadIds) {
        serverNotifs.map { dto ->
            val cat = when {
                dto.critical || dto.title.contains("allarme", true) || dto.title.contains("sicurezza", true) -> NotificationCategory.SECURITY
                dto.title.contains("energia", true) || dto.title.contains("consum", true) || dto.title.contains("kw", true) -> NotificationCategory.ENERGY
                dto.title.contains("aggiornamento", true) || dto.title.contains("sistema", true) -> NotificationCategory.SYSTEM
                else -> NotificationCategory.DEVICES
            }
            InAppNotification(
                id = dto.id,
                title = dto.title,
                message = dto.message,
                category = cat,
                timeAgo = "Recente",
                isRead = dto.resolved_at != null || localReadIds.contains(dto.id),
                actionLabel = dto.actions.firstOrNull()?.title
            )
        }
    }

    var selectedCategory by remember { mutableStateOf(NotificationCategory.ALL) }
    val unreadCount = notifications.count { !it.isRead }

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
                    text = "Centro Notifiche",
                    fontWeight = FontWeight.Bold,
                    fontSize = 18.sp,
                    color = Color.White
                )
                Text(
                    text = if (unreadCount > 0) "$unreadCount non lette" else "Tutto aggiornato",
                    fontSize = 12.sp,
                    color = if (unreadCount > 0) RoseDanger else EmeraldSuccess
                )
            }

            // Mark all read button
            IconButton(
                onClick = {
                    localReadIds = localReadIds + notifications.map { it.id }
                },
                enabled = unreadCount > 0
            ) {
                Text("✓✓", color = if (unreadCount > 0) CyanAccent else Color.White.copy(alpha = 0.3f), fontSize = 16.sp, fontWeight = FontWeight.Bold)
            }
        }

        // Category Filter Horizontal Scroll
        Row(
            modifier = Modifier
                .horizontalScroll(rememberScrollState())
                .padding(horizontal = 16.dp, vertical = 6.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            NotificationCategory.values().forEach { cat ->
                val isSelected = selectedCategory == cat
                Surface(
                    shape = RoundedCornerShape(20.dp),
                    color = if (isSelected) IndigoPrimary else Slate800,
                    border = BorderStroke(1.dp, if (isSelected) CyanAccent else Slate700),
                    modifier = Modifier.clickable { selectedCategory = cat }
                ) {
                    Row(
                        modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(cat.icon, fontSize = 12.sp)
                        Spacer(modifier = Modifier.width(6.dp))
                        Text(
                            text = cat.label,
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = if (isSelected) Color.White else Color.White.copy(alpha = 0.6f)
                        )
                    }
                }
            }
        }

        Spacer(modifier = Modifier.height(6.dp))

        // Notification List
        val filteredList = notifications.filter {
            selectedCategory == NotificationCategory.ALL || it.category == selectedCategory
        }

        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
            contentPadding = PaddingValues(vertical = 10.dp)
        ) {
            items(filteredList, key = { it.id }) { notif ->
                val cardColor by animateColorAsState(
                    targetValue = if (notif.isRead) Slate800.copy(alpha = 0.6f) else Slate800,
                    label = "notif_bg"
                )

                Card(
                    shape = RoundedCornerShape(16.dp),
                    colors = CardDefaults.cardColors(containerColor = cardColor),
                    border = BorderStroke(
                        1.dp,
                        if (!notif.isRead) notif.category.color.copy(alpha = 0.5f) else Slate700
                    ),
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable {
                            localReadIds = localReadIds + notif.id
                        }
                ) {
                    Column(modifier = Modifier.padding(14.dp)) {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.SpaceBetween
                        ) {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                if (!notif.isRead) {
                                    Box(
                                        modifier = Modifier
                                            .size(8.dp)
                                            .background(notif.category.color, CircleShape)
                                    )
                                    Spacer(modifier = Modifier.width(8.dp))
                                }
                                Text(
                                    text = notif.title,
                                    fontWeight = FontWeight.Bold,
                                    fontSize = 14.sp,
                                    color = Color.White
                                )
                            }
                            Text(
                                text = notif.timeAgo,
                                fontSize = 11.sp,
                                color = Color.White.copy(alpha = 0.4f)
                            )
                        }

                        Spacer(modifier = Modifier.height(4.dp))

                        Text(
                            text = notif.message,
                            fontSize = 13.sp,
                            color = Color.White.copy(alpha = 0.75f),
                            lineHeight = 18.sp
                        )

                        // Action button if present
                        notif.actionLabel?.let { action ->
                            Spacer(modifier = Modifier.height(10.dp))
                            Button(
                                onClick = {
                                    localReadIds = localReadIds + notif.id
                                },
                                shape = RoundedCornerShape(8.dp),
                                colors = ButtonDefaults.buttonColors(
                                    containerColor = notif.category.color.copy(alpha = 0.15f)
                                ),
                                contentPadding = PaddingValues(horizontal = 12.dp, vertical = 4.dp),
                                modifier = Modifier.height(32.dp)
                            ) {
                                Text(
                                    text = action,
                                    color = notif.category.color,
                                    fontSize = 12.sp,
                                    fontWeight = FontWeight.Bold
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
