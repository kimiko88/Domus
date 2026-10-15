package com.domus.app.presentation.screens.alarm

import androidx.compose.animation.animateColorAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.domain.model.Entity
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.RoseError
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AlarmBottomSheet(
    entity: Entity,
    onArmHome: (String) -> Unit,
    onArmAway: (String) -> Unit,
    onDisarm: (String, String) -> Unit,
    onDismiss: () -> Unit
) {
    var pinCode by remember { mutableStateOf("") }
    val isArmed = entity.isArmed

    val stateColor by animateColorAsState(
        targetValue = when {
            entity.state == "triggered" -> RoseError
            isArmed -> Color(0xFF3B82F6) // Blue
            else -> EmeraldSuccess
        },
        label = "alarm_state_color"
    )

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        containerColor = MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(topStart = 24.dp, topEnd = 24.dp)
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 24.dp, vertical = 16.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Stato Attuale Allarme
            Surface(
                shape = CircleShape,
                color = stateColor.copy(alpha = 0.15f),
                modifier = Modifier.size(72.dp)
            ) {
                Box(contentAlignment = Alignment.Center) {
                    Text(
                        text = if (isArmed) "🛡️" else "🔓",
                        fontSize = 32.sp
                    )
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            Text(
                text = entity.name,
                fontWeight = FontWeight.Bold,
                fontSize = 18.sp
            )

            Text(
                text = entity.state.uppercase().replace("_", " "),
                color = stateColor,
                fontWeight = FontWeight.Black,
                fontSize = 13.sp,
                letterSpacing = 1.sp
            )

            Spacer(modifier = Modifier.height(20.dp))

            // Display PIN digitato (pallini mascherati)
            Row(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                modifier = Modifier.padding(vertical = 8.dp)
            ) {
                repeat(4) { idx ->
                    val isFilled = idx < pinCode.length
                    Box(
                        modifier = Modifier
                            .size(16.dp)
                            .background(
                                color = if (isFilled) stateColor else Slate700.copy(alpha = 0.5f),
                                shape = CircleShape
                            )
                    )
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            // Tastierino Numerico PIN 3x4
            val keys = listOf(
                listOf("1", "2", "3"),
                listOf("4", "5", "6"),
                listOf("7", "8", "9"),
                listOf("C", "0", "⌫")
            )

            Column(
                verticalArrangement = Arrangement.spacedBy(10.dp),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                for (row in keys) {
                    Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                        for (key in row) {
                            ElevatedButton(
                                onClick = {
                                    when (key) {
                                        "C" -> pinCode = ""
                                        "⌫" -> if (pinCode.isNotEmpty()) pinCode = pinCode.dropLast(1)
                                        else -> if (pinCode.length < 4) pinCode += key
                                    }
                                },
                                shape = CircleShape,
                                modifier = Modifier.size(60.dp),
                                colors = ButtonDefaults.elevatedButtonColors(
                                    containerColor = MaterialTheme.colorScheme.surfaceVariant
                                )
                            ) {
                                Text(
                                    text = key,
                                    fontSize = 18.sp,
                                    fontWeight = FontWeight.Bold,
                                    textAlign = TextAlign.Center
                                )
                            }
                        }
                    }
                }
            }

            Spacer(modifier = Modifier.height(20.dp))

            // Pulsanti Azioni Rapide
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                if (isArmed) {
                    Button(
                        onClick = {
                            onDisarm(entity.id, pinCode)
                            pinCode = ""
                            onDismiss()
                        },
                        colors = ButtonDefaults.buttonColors(containerColor = EmeraldSuccess),
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.weight(1f)
                    ) {
                        Text("Disarma", fontWeight = FontWeight.Bold)
                    }
                } else {
                    Button(
                        onClick = {
                            onArmHome(entity.id)
                            onDismiss()
                        },
                        colors = ButtonDefaults.buttonColors(containerColor = Color(0xFFF59E0B)),
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.weight(1f)
                    ) {
                        Text("Arma Parziale", fontWeight = FontWeight.Bold)
                    }

                    Button(
                        onClick = {
                            onArmAway(entity.id)
                            onDismiss()
                        },
                        colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF3B82F6)),
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.weight(1f)
                    ) {
                        Text("Arma Totale", fontWeight = FontWeight.Bold)
                    }
                }
            }

            Spacer(modifier = Modifier.height(16.dp))
        }
    }
}
