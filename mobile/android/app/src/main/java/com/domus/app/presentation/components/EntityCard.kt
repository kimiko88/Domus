package com.domus.app.presentation.components

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.domain.model.Entity
import com.domus.app.presentation.theme.AmberWarning
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.Slate700

@Composable
fun EntityCard(
    entity: Entity,
    onToggle: (String) -> Unit,
    onClick: ((Entity) -> Unit)? = null,
    modifier: Modifier = Modifier
) {
    val haptic = LocalHapticFeedback.current
    val isOn = entity.isOn
    val activeColor by animateColorAsState(
        targetValue = if (isOn) CyanAccent.copy(alpha = 0.14f) else MaterialTheme.colorScheme.surface,
        animationSpec = tween(durationMillis = 250),
        label = "card_color"
    )

    val borderColor by animateColorAsState(
        targetValue = if (isOn) CyanAccent.copy(alpha = 0.45f) else MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.25f),
        animationSpec = tween(durationMillis = 250),
        label = "card_border"
    )

    // Dynamic subtitle calculation
    val subtitleText = when {
        entity.isLight && isOn -> "Accesa • ${entity.brightness ?: 100}%"
        entity.isLight && !isOn -> "Spenta"
        entity.isSwitch && isOn -> "Attivo"
        entity.isSwitch && !isOn -> "Disattivato"
        entity.isClimate -> "Target: ${entity.currentTemperature ?: 21.0}°C"
        entity.isSensor -> "${entity.state} ${entity.attributes["unit_of_measurement"]?.toString()?.replace("\"", "") ?: ""}".trim()
        entity.isAlarm -> if (entity.isArmed) "Allarme Armato" else "Disarmato"
        else -> entity.domain.replaceFirstChar { it.uppercase() }
    }

    val a11yDescription = "${entity.name}, ${if (isOn) "acceso" else "spento"}, $subtitleText"

    Card(
        shape = RoundedCornerShape(20.dp),
        colors = CardDefaults.cardColors(containerColor = activeColor),
        border = BorderStroke(1.dp, borderColor),
        elevation = CardDefaults.cardElevation(
            defaultElevation = if (isOn) 4.dp else 1.5.dp,
            pressedElevation = 6.dp
        ),
        modifier = modifier
            .fillMaxWidth()
            .defaultMinSize(minHeight = 72.dp)
            .semantics(mergeDescendants = true) {
                contentDescription = a11yDescription
                role = if (entity.isLight || entity.isSwitch) Role.Switch else Role.Button
            }
            .clickable(
                onClickLabel = if (isOn) "Gestisci ${entity.name}" else "Accendi ${entity.name}"
            ) {
                haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                if (onClick != null) {
                    onClick(entity)
                } else {
                    onToggle(entity.id)
                }
            }
    ) {
        Column(
            modifier = Modifier.padding(16.dp)
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween,
                modifier = Modifier.fillMaxWidth()
            ) {
                // Domain / Icon Symbol
                Surface(
                    shape = RoundedCornerShape(12.dp),
                    color = if (isOn) CyanAccent.copy(alpha = 0.22f) else Slate700.copy(alpha = 0.35f),
                    modifier = Modifier.size(40.dp)
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Text(
                            text = when (entity.domain) {
                                "light" -> "💡"
                                "switch" -> "🔌"
                                "climate" -> "🌡️"
                                "sensor" -> "📊"
                                "alarm_control_panel" -> "🛡️"
                                "camera" -> "📹"
                                else -> "⚙️"
                            },
                            fontSize = 20.sp
                        )
                    }
                }

                // Interactive Switch for togglable domains
                if (entity.isLight || entity.isSwitch) {
                    Switch(
                        checked = isOn,
                        onCheckedChange = { onToggle(entity.id) },
                        colors = SwitchDefaults.colors(
                            checkedThumbColor = Color.White,
                            checkedTrackColor = CyanAccent
                        )
                    )
                } else {
                    // Badge State
                    Surface(
                        shape = RoundedCornerShape(8.dp),
                        color = MaterialTheme.colorScheme.surfaceVariant
                    ) {
                        Text(
                            text = entity.state.uppercase(),
                            fontSize = 11.sp,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(14.dp))

            Text(
                text = entity.name,
                fontWeight = FontWeight.Bold,
                fontSize = 15.sp,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )

            Spacer(modifier = Modifier.height(3.dp))

            Text(
                text = subtitleText,
                fontSize = 12.sp,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.65f),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
        }
    }
}
