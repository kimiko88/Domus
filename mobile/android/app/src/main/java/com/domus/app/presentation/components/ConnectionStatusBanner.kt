package com.domus.app.presentation.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.domain.model.ServerConnection
import com.domus.app.domain.model.TransportMode
import com.domus.app.presentation.theme.AmberWarning
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.IndigoPrimary
import com.domus.app.presentation.theme.RoseError

@Composable
fun ConnectionStatusBanner(
    connection: ServerConnection,
    onReconnectClick: () -> Unit = {},
    compact: Boolean = false,
    modifier: Modifier = Modifier
) {
    val (statusText, badgeColor, dotColor) = when (connection.mode) {
        TransportMode.LOCAL_LAN -> Triple(
            if (compact) "LAN" else "LAN Diretta • ${connection.latencyMs}ms",
            EmeraldSuccess.copy(alpha = 0.15f),
            EmeraldSuccess
        )
        TransportMode.REMOTE_TLS -> Triple(
            if (compact) "TLS" else "Remoto TLS • ${connection.latencyMs}ms",
            IndigoPrimary.copy(alpha = 0.15f),
            IndigoPrimary
        )
        TransportMode.CONNECTING -> Triple(
            if (compact) "Ricerca..." else "Ricerca Hub in corso...",
            AmberWarning.copy(alpha = 0.15f),
            AmberWarning
        )
        TransportMode.DISCONNECTED -> Triple(
            if (compact) "Offline" else "Disconnesso",
            RoseError.copy(alpha = 0.15f),
            RoseError
        )
    }

    val isClickable = !connection.isConnected && compact
    val surfaceModifier = if (isClickable) {
        modifier
            .semantics { role = Role.Button }
            .clickable(onClick = onReconnectClick)
    } else {
        modifier
    }

    Surface(
        shape = RoundedCornerShape(if (compact) 12.dp else 20.dp),
        color = badgeColor,
        modifier = surfaceModifier
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(
                horizontal = if (compact) 8.dp else 12.dp,
                vertical = if (compact) 4.dp else 6.dp
            )
        ) {
            Box(
                modifier = Modifier
                    .size(if (compact) 6.dp else 8.dp)
                    .background(dotColor, CircleShape)
            )
            Spacer(modifier = Modifier.width(if (compact) 5.dp else 8.dp))
            Text(
                text = statusText,
                fontSize = if (compact) 11.sp else 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.onSurface,
                maxLines = 1,
                softWrap = false
            )
            if (!compact && !connection.isConnected && connection.mode != TransportMode.CONNECTING) {
                Spacer(modifier = Modifier.width(8.dp))
                TextButton(
                    onClick = onReconnectClick,
                    contentPadding = PaddingValues(0.dp),
                    modifier = Modifier.height(20.dp)
                ) {
                    Text("Riconnetti", fontSize = 11.sp, color = IndigoPrimary, fontWeight = FontWeight.Bold)
                }
            }
        }
    }
}
