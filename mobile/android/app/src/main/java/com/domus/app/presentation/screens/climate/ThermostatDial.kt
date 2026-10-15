package com.domus.app.presentation.screens.climate

import androidx.compose.animation.animateColorAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
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
import com.domus.app.domain.model.Entity
import com.domus.app.presentation.theme.AmberWarning
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ThermostatSheet(
    entity: Entity,
    onSetTemperature: (String, Double) -> Unit,
    onSetMode: (String, String) -> Unit,
    onDismiss: () -> Unit
) {
    var targetTemp by remember { mutableStateOf(entity.targetTemperature ?: 20.0) }
    var selectedMode by remember { mutableStateOf(entity.state.lowercase()) }

    val currentTemp = entity.currentTemperature ?: 21.2

    val tempColor by animateColorAsState(
        targetValue = when {
            targetTemp < 19.0 -> CyanAccent
            targetTemp <= 21.5 -> EmeraldSuccess
            else -> AmberWarning
        },
        label = "temp_color"
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
            Text(
                text = entity.name,
                fontWeight = FontWeight.Bold,
                fontSize = 18.sp
            )

            Text(
                text = "Attualmente: ${String.format("%.1f", currentTemp)}°C",
                fontSize = 13.sp,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.65f)
            )

            Spacer(modifier = Modifier.height(24.dp))

            // Quadrante Circolare Termostato
            Surface(
                shape = CircleShape,
                color = tempColor.copy(alpha = 0.12f),
                border = androidx.compose.foundation.BorderStroke(3.dp, tempColor),
                modifier = Modifier.size(200.dp)
            ) {
                Column(
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Text(
                        text = "TARGET",
                        fontSize = 11.sp,
                        fontWeight = FontWeight.Bold,
                        letterSpacing = 1.sp,
                        color = tempColor
                    )
                    Text(
                        text = "${String.format("%.1f", targetTemp)}°",
                        fontSize = 44.sp,
                        fontWeight = FontWeight.Black,
                        color = MaterialTheme.colorScheme.onSurface
                    )

                    // Pulsanti + e -
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(16.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        IconButton(
                            onClick = {
                                if (targetTemp > 10.0) {
                                    targetTemp -= 0.5
                                    onSetTemperature(entity.id, targetTemp)
                                }
                            },
                            modifier = Modifier.background(MaterialTheme.colorScheme.surfaceVariant, CircleShape)
                        ) {
                            Text("-", fontSize = 24.sp, fontWeight = FontWeight.Bold)
                        }

                        IconButton(
                            onClick = {
                                if (targetTemp < 32.0) {
                                    targetTemp += 0.5
                                    onSetTemperature(entity.id, targetTemp)
                                }
                            },
                            modifier = Modifier.background(MaterialTheme.colorScheme.surfaceVariant, CircleShape)
                        ) {
                            Text("+", fontSize = 20.sp, fontWeight = FontWeight.Bold)
                        }
                    }
                }
            }

            Spacer(modifier = Modifier.height(24.dp))

            // Selettore Modalità HVAC
            val modes = listOf("off", "heat", "cool", "auto")
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                for (mode in modes) {
                    val isSelected = selectedMode == mode
                    Surface(
                        shape = RoundedCornerShape(12.dp),
                        color = if (isSelected) tempColor else MaterialTheme.colorScheme.surfaceVariant,
                        modifier = Modifier
                            .weight(1f)
                            .clickable {
                                selectedMode = mode
                                onSetMode(entity.id, mode)
                            }
                    ) {
                        Box(
                            contentAlignment = Alignment.Center,
                            modifier = Modifier.padding(vertical = 12.dp)
                        ) {
                            Text(
                                text = mode.uppercase(),
                                fontSize = 12.sp,
                                fontWeight = FontWeight.Bold,
                                color = if (isSelected) Color.Black else MaterialTheme.colorScheme.onSurface
                            )
                        }
                    }
                }
            }

            Spacer(modifier = Modifier.height(24.dp))
        }
    }
}

@Composable
fun ThermostatDial(
    entity: Entity,
    onSetTemperature: (String, Double) -> Unit,
    onSetMode: (String, String) -> Unit = { _, _ -> },
    onDismiss: () -> Unit
) {
    ThermostatSheet(
        entity = entity,
        onSetTemperature = onSetTemperature,
        onSetMode = onSetMode,
        onDismiss = onDismiss
    )
}
