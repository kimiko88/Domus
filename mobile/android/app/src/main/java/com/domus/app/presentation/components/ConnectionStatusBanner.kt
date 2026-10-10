package com.domus.app.presentation.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
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
    modifier: Modifier = Modifier
) {
    val (statusText, badgeColor, dotColor) = when (connection.mode) {
        TransportMode.LOCAL_LAN -> Triple(
            "LAN Diretta • ${connection.latencyMs}ms",
            EmeraldSuccess.copy(alpha = 0.15f),
            EmeraldSuccess
        )
        TransportMode.REMOTE_TLS -> Triple(
            "Remoto TLS • ${connection.latencyMs}ms",
            IndigoPrimary.copy(alpha = 0.15f),
            IndigoPrimary
        )
        TransportMode.CONNECTING -> Triple(
            "Ricerca Hub in corso...",
            AmberWarning.copy(alpha = 0.15f),
            AmberWarning
        )
        TransportMode.DISCONNECTED -> Triple(
            "Disconnesso",
            RoseError.copy(alpha = 0.15f),
            RoseError
        )
    }

    Surface(
        shape = RoundedCornerShape(20.dp),
        color = badgeColor,
        modifier = modifier
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp)
        ) {
            Box(
                modifier = Modifier
                    .size(8.dp)
                    .background(dotColor, CircleShape)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = statusText,
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.onSurface
            )
            if (!connection.isConnected && connection.mode != TransportMode.CONNECTING) {
                Spacer(modifier = Modifier.width(6.dp))
                TextButton(
                    onClick = onReconnectClick,
                    contentPadding = PaddingValues(0.dp),
                    modifier = Modifier.height(20.dp)
                ) {
                    Text("Riconnetti", fontSize = 11.sp, color = IndigoPrimary)
                }
            }
        }
    }
}
