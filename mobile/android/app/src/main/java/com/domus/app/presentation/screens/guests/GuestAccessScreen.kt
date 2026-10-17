package com.domus.app.presentation.screens.guests

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.widget.Toast
import androidx.compose.foundation.background
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.domain.model.DomusUserAccount
import com.domus.app.domain.model.FamilyInviteDto
import com.domus.app.domain.model.UserAccessRole
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.IndigoPrimary
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.viewmodel.DashboardViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun GuestAccessScreen(
    viewModel: DashboardViewModel,
    onDismiss: () -> Unit
) {
    val accounts by viewModel.familyAccounts.collectAsState()
    val invites by viewModel.familyInvites.collectAsState()
    val context = LocalContext.current

    var showInviteDialog by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        viewModel.loadFamilyAndInvites()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text("Accesso Ospiti & Famiglia", fontWeight = FontWeight.Bold, fontSize = 20.sp)
                        Text("Profili, ruoli RBAC e link di accesso temporanei", fontSize = 12.sp, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f))
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onDismiss) {
                        Text("✕", fontSize = 18.sp, fontWeight = FontWeight.Bold)
                    }
                },
                actions = {
                    Button(
                        onClick = { showInviteDialog = true },
                        colors = ButtonDefaults.buttonColors(containerColor = IndigoPrimary),
                        shape = RoundedCornerShape(12.dp)
                    ) {
                        Text("+ Invita", fontWeight = FontWeight.Bold, fontSize = 13.sp)
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.surface)
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize()
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            item {
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = "MEMBRI DELLA CASA (${accounts.size})",
                    fontSize = 11.sp,
                    fontWeight = FontWeight.Bold,
                    color = CyanAccent
                )
            }

            items(accounts, key = { it.id }) { account ->
                UserAccountCard(account = account)
            }

            if (invites.isNotEmpty()) {
                item {
                    Spacer(modifier = Modifier.height(12.dp))
                    Text(
                        text = "INVITI ATTIVI & QR CODE (${invites.size})",
                        fontSize = 11.sp,
                        fontWeight = FontWeight.Bold,
                        color = Color(0xFFFBBF24)
                    )
                }

                items(invites, key = { it.id }) { invite ->
                    InviteCard(
                        invite = invite,
                        onCopyLink = { link ->
                            val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                            clipboard.setPrimaryClip(ClipData.newPlainText("Domus Invite", link))
                            Toast.makeText(context, "Link di invito copiato negli appunti!", Toast.LENGTH_SHORT).show()
                        },
                        onRevoke = {
                            viewModel.revokeFamilyInvite(invite.id)
                        }
                    )
                }
            }

            item {
                Spacer(modifier = Modifier.height(24.dp))
            }
        }
    }

    if (showInviteDialog) {
        CreateInviteDialog(
            onDismiss = { showInviteDialog = false },
            onCreate = { role, hours ->
                viewModel.createFamilyInvite(role, hours)
                showInviteDialog = false
            }
        )
    }
}

@Composable
fun UserAccountCard(account: DomusUserAccount) {
    Card(
        shape = RoundedCornerShape(16.dp),
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
                            .size(40.dp)
                            .background(IndigoPrimary.copy(alpha = 0.2f), CircleShape),
                        contentAlignment = Alignment.Center
                    ) {
                        Text(account.role.icon, fontSize = 20.sp)
                    }

                    Spacer(modifier = Modifier.width(12.dp))

                    Column {
                        Text(
                            text = account.name,
                            fontWeight = FontWeight.Bold,
                            fontSize = 15.sp,
                            color = Color.White
                        )
                        Text(
                            text = account.role.displayName,
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold,
                            color = CyanAccent
                        )
                    }
                }

                Surface(
                    shape = RoundedCornerShape(8.dp),
                    color = Slate700
                ) {
                    Text(
                        text = if (account.canDisarmAlarm) "Allarme: Abilitato" else "Allarme: Bloccato",
                        fontSize = 11.sp,
                        fontWeight = FontWeight.Bold,
                        color = if (account.canDisarmAlarm) EmeraldSuccess else Color(0xFFEF4444),
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                    )
                }
            }

            Spacer(modifier = Modifier.height(10.dp))

            Text(
                text = "Stanze consentite: ${account.allowedRooms.joinToString(", ")}",
                fontSize = 12.sp,
                color = Color.White.copy(alpha = 0.7f)
            )

            account.expiresAtMillis?.let {
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = "Scadenza accesso temporaneo: 24h",
                    fontSize = 11.sp,
                    color = Color(0xFFFBBF24)
                )
            }
        }
    }
}

@Composable
fun InviteCard(
    invite: FamilyInviteDto,
    onCopyLink: (String) -> Unit,
    onRevoke: () -> Unit
) {
    val link = invite.inviteUrl ?: "http://192.168.1.100:8080/invite/${invite.token}"

    Card(
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = Slate800),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("🎟️", fontSize = 24.sp)
                    Spacer(modifier = Modifier.width(10.dp))
                    Column {
                        Text("Invito: Ruolo ${invite.role.replaceFirstChar { it.uppercase() }}", fontWeight = FontWeight.Bold, fontSize = 14.sp, color = Color.White)
                        Text("Token: ${invite.token.take(8)}...", fontSize = 11.sp, color = Color.White.copy(alpha = 0.6f))
                    }
                }

                IconButton(onClick = onRevoke) {
                    Text("🗑️", fontSize = 14.sp)
                }
            }

            Spacer(modifier = Modifier.height(10.dp))

            Button(
                onClick = { onCopyLink(link) },
                colors = ButtonDefaults.buttonColors(containerColor = CyanAccent.copy(alpha = 0.2f), contentColor = CyanAccent),
                shape = RoundedCornerShape(10.dp),
                modifier = Modifier.fillMaxWidth()
            ) {
                Text("📋 Copia Link di Accesso / QR", fontSize = 13.sp, fontWeight = FontWeight.Bold)
            }
        }
    }
}

@Composable
fun CreateInviteDialog(
    onDismiss: () -> Unit,
    onCreate: (String, Int) -> Unit
) {
    var selectedRole by remember { mutableStateOf("guest") }
    var selectedHours by remember { mutableIntStateOf(24) }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Nuovo Link di Accesso Ospite", fontWeight = FontWeight.Bold) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text("Seleziona Profilo Permessi:", fontSize = 13.sp, fontWeight = FontWeight.SemiBold)
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    listOf("guest" to "Ospite", "family" to "Famiglia", "child" to "Bambino").forEach { (role, label) ->
                        FilterChip(
                            selected = selectedRole == role,
                            onClick = { selectedRole = role },
                            label = { Text(label, fontSize = 12.sp) }
                        )
                    }
                }

                Text("Validità Scadenza:", fontSize = 13.sp, fontWeight = FontWeight.SemiBold)
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    listOf(24 to "24 Ore", 48 to "48 Ore", 168 to "7 Giorni").forEach { (hours, label) ->
                        FilterChip(
                            selected = selectedHours == hours,
                            onClick = { selectedHours = hours },
                            label = { Text(label, fontSize = 12.sp) }
                        )
                    }
                }
            }
        },
        confirmButton = {
            Button(onClick = { onCreate(selectedRole, selectedHours) }) {
                Text("Genera Invito")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Annulla")
            }
        }
    )
}
