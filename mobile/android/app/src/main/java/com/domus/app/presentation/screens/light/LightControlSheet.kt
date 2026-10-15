package com.domus.app.presentation.screens.light

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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LightControlSheet(
    entity: Entity,
    onToggle: (String) -> Unit,
    onSetBrightness: (String, Int) -> Unit,
    onDismiss: () -> Unit
) {
    var brightness by remember { mutableStateOf((entity.brightness ?: 100).toFloat()) }
    val isOn = entity.isOn

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
                text = if (isOn) "Luce Accesa • ${brightness.toInt()}%" else "Spenta",
                fontSize = 13.sp,
                color = if (isOn) CyanAccent else MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
            )

            Spacer(modifier = Modifier.height(24.dp))

            // Icona Luminosa Centrale
            Surface(
                shape = CircleShape,
                color = if (isOn) CyanAccent.copy(alpha = 0.2f) else MaterialTheme.colorScheme.surfaceVariant,
                modifier = Modifier
                    .size(90.dp)
                    .clickable { onToggle(entity.id) }
            ) {
                Box(contentAlignment = Alignment.Center) {
                    Text("💡", fontSize = 40.sp)
                }
            }

            Spacer(modifier = Modifier.height(28.dp))

            // Slider Luminosità
            Text(
                text = "Luminosità",
                fontWeight = FontWeight.SemiBold,
                fontSize = 14.sp,
                modifier = Modifier.align(Alignment.Start)
            )

            Slider(
                value = brightness,
                onValueChange = { brightness = it },
                onValueChangeFinished = {
                    onSetBrightness(entity.id, brightness.toInt())
                },
                valueRange = 1f..100f,
                colors = SliderDefaults.colors(
                    thumbColor = CyanAccent,
                    activeTrackColor = CyanAccent
                )
            )

            Spacer(modifier = Modifier.height(20.dp))

            // Preset Rapidi
            Text(
                text = "Scene e Temperature Rapide",
                fontWeight = FontWeight.SemiBold,
                fontSize = 14.sp,
                modifier = Modifier.align(Alignment.Start)
            )

            Spacer(modifier = Modifier.height(10.dp))

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                presetButton(label = "Relax", color = Color(0xFFFFB74D), modifier = Modifier.weight(1f)) {
                    brightness = 40f
                    onSetBrightness(entity.id, 40)
                }
                presetButton(label = "Lettura", color = Color(0xFFFFF176), modifier = Modifier.weight(1f)) {
                    brightness = 80f
                    onSetBrightness(entity.id, 80)
                }
                presetButton(label = "Massima", color = Color(0xFF81D4FA), modifier = Modifier.weight(1f)) {
                    brightness = 100f
                    onSetBrightness(entity.id, 100)
                }
                presetButton(label = "Notte", color = Color(0xFFCE93D8), modifier = Modifier.weight(1f)) {
                    brightness = 10f
                    onSetBrightness(entity.id, 10)
                }
            }

            Spacer(modifier = Modifier.height(28.dp))
        }
    }
}

@Composable
private fun presetButton(label: String, color: Color, modifier: Modifier = Modifier, onClick: () -> Unit) {
    Surface(
        shape = RoundedCornerShape(12.dp),
        color = color.copy(alpha = 0.2f),
        modifier = modifier.clickable(onClick = onClick)
    ) {
        Box(contentAlignment = Alignment.Center, modifier = Modifier.padding(vertical = 10.dp)) {
            Text(label, fontSize = 12.sp, fontWeight = FontWeight.Bold, color = color)
        }
    }
}
