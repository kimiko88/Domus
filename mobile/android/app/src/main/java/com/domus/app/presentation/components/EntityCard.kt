package com.domus.app.presentation.components

import androidx.compose.animation.animateColorAsState
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
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
    modifier: Modifier = Modifier
) {
    val isOn = entity.isOn
    val activeColor by animateColorAsState(
        targetValue = if (isOn) CyanAccent.copy(alpha = 0.12f) else MaterialTheme.colorScheme.surface,
        label = "card_color"
    )

    Card(
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = activeColor),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp),
        modifier = modifier
            .fillMaxWidth()
            .clickable { onToggle(entity.id) }
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
                    shape = RoundedCornerShape(10.dp),
                    color = if (isOn) CyanAccent.copy(alpha = 0.2f) else Slate700.copy(alpha = 0.4f),
                    modifier = Modifier.size(36.dp)
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Text(
                            text = when (entity.domain) {
                                "light" -> "💡"
                                "switch" -> "🔌"
                                "climate" -> "🌡️"
                                "sensor" -> "📊"
                                "alarm_control_panel" -> "🛡️"
                                else -> "⚙️"
                            },
                            fontSize = 18.sp
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
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            Text(
                text = entity.name,
                fontWeight = FontWeight.Bold,
                fontSize = 15.sp,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )

            Spacer(modifier = Modifier.height(4.dp))

            // Subtitle / attributes
            val subtitleText = when {
                entity.isLight && isOn -> "Accesa • ${entity.brightness ?: 100}%"
                entity.isLight && !isOn -> "Spenta"
                entity.isClimate -> "Target: ${entity.currentTemperature ?: 21.0}°C"
                entity.isSensor -> "${entity.state} ${entity.attributes["unit_of_measurement"]?.toString()?.replace("\"", "") ?: ""}"
                else -> entity.domain.replaceFirstChar { it.uppercase() }
            }

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
