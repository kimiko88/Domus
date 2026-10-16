package com.domus.app.presentation.screens.floorplan

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectTransformGestures
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.domain.model.Entity
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.IndigoPrimary
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.theme.Slate900

data class FloorplanRoom(
    val id: String,
    val name: String,
    val rectX: Float,
    val rectY: Float,
    val rectWidth: Float,
    val rectHeight: Float,
    val isLit: Boolean = false
)

data class FloorplanMarker(
    val entityId: String,
    val name: String,
    val domain: String,
    val posX: Float,
    val posY: Float,
    var isOn: Boolean = false,
    var valueText: String = ""
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FloorplanCanvas(
    entities: List<Entity> = emptyList(),
    onEntityClick: (String) -> Unit = {},
    onDismiss: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    // Transform gesture state (Zoom & Pan)
    var scale by remember { mutableStateOf(1f) }
    var offset by remember { mutableStateOf(Offset.Zero) }

    // Selected marker for detail popup
    var selectedMarker by remember { mutableStateOf<FloorplanMarker?>(null) }

    val rooms = remember {
        listOf(
            FloorplanRoom("living", "Salotto & Cucina", 0.05f, 0.08f, 0.55f, 0.42f, isLit = true),
            FloorplanRoom("bedroom", "Camera Matrimoniale", 0.63f, 0.08f, 0.32f, 0.42f, isLit = false),
            FloorplanRoom("bath", "Bagno Principale", 0.05f, 0.53f, 0.28f, 0.38f, isLit = true),
            FloorplanRoom("office", "Studio / Domotica", 0.36f, 0.53f, 0.30f, 0.38f, isLit = true),
            FloorplanRoom("balcony", "Terrazzo Panoramico", 0.69f, 0.53f, 0.26f, 0.38f, isLit = false)
        )
    }

    val markers = remember(entities) {
        if (entities.isEmpty()) {
            emptyList()
        } else {
            entities.take(12).mapIndexed { index, entity ->
                val x = 0.15f + (index % 3) * 0.35f
                val y = 0.18f + (index / 3) * 0.26f
                FloorplanMarker(
                    entityId = entity.id,
                    name = entity.name,
                    domain = entity.domain,
                    posX = x.coerceIn(0.08f, 0.92f),
                    posY = y.coerceIn(0.12f, 0.88f),
                    isOn = entity.isOn,
                    valueText = when {
                        entity.domain == "climate" -> "${entity.attributes["temperature"] ?: "21.0"}°C"
                        entity.domain == "light" && entity.isOn -> "${entity.attributes["brightness"] ?: "On"}"
                        entity.isOn -> "On"
                        else -> "Off"
                    }
                )
            }
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(Slate900)
    ) {
        // Floorplan Top Bar
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
                    text = "Planimetria Interattiva",
                    fontWeight = FontWeight.Bold,
                    fontSize = 17.sp,
                    color = Color.White
                )
                Text(
                    text = "Piano Terra • 5 Ambienti",
                    fontSize = 12.sp,
                    color = CyanAccent
                )
            }

            // Reset Zoom
            IconButton(onClick = { scale = 1f; offset = Offset.Zero }) {
                Text("↺", fontSize = 22.sp, color = CyanAccent)
            }
        }

        // Canvas Area with Zoom & Pan Gestures
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .weight(1f)
                .pointerInput(Unit) {
                    detectTransformGestures { _, pan, zoom, _ ->
                        scale = (scale * zoom).coerceIn(0.7f, 3.5f)
                        offset = Offset(
                            x = offset.x + pan.x,
                            y = offset.y + pan.y
                        )
                    }
                }
        ) {
            Canvas(modifier = Modifier.fillMaxSize()) {
                val canvasWidth = size.width
                val canvasHeight = size.height

                // Draw Background Architectural Grid
                val gridStep = 40f * scale
                var x = (offset.x % gridStep)
                while (x < canvasWidth) {
                    drawLine(
                        color = Color.White.copy(alpha = 0.04f),
                        start = Offset(x, 0f),
                        end = Offset(x, canvasHeight),
                        strokeWidth = 1f
                    )
                    x += gridStep
                }
                var y = (offset.y % gridStep)
                while (y < canvasHeight) {
                    drawLine(
                        color = Color.White.copy(alpha = 0.04f),
                        start = Offset(0f, y),
                        end = Offset(canvasWidth, y),
                        strokeWidth = 1f
                    )
                    y += gridStep
                }

                // Draw Rooms
                rooms.forEach { room ->
                    val rx = offset.x + (room.rectX * canvasWidth * scale)
                    val ry = offset.y + (room.rectY * canvasHeight * scale)
                    val rw = room.rectWidth * canvasWidth * scale
                    val rh = room.rectHeight * canvasHeight * scale

                    // Room fill color (glows warm when lit)
                    val fillColor = if (room.isLit) {
                        CyanAccent.copy(alpha = 0.12f)
                    } else {
                        Slate800.copy(alpha = 0.45f)
                    }

                    drawRect(
                        color = fillColor,
                        topLeft = Offset(rx, ry),
                        size = Size(rw, rh)
                    )

                    // Room boundary walls
                    drawRect(
                        color = Slate700,
                        topLeft = Offset(rx, ry),
                        size = Size(rw, rh),
                        style = Stroke(width = 3f * scale)
                    )
                }

                // Draw Markers on Canvas
                markers.forEach { marker ->
                    val mx = offset.x + (marker.posX * canvasWidth * scale)
                    val my = offset.y + (marker.posY * canvasHeight * scale)

                    val markerColor = if (marker.isOn) CyanAccent else Slate700
                    val radius = 14f * scale

                    // Outer pulse/halo
                    if (marker.isOn) {
                        drawCircle(
                            color = CyanAccent.copy(alpha = 0.25f),
                            radius = radius * 1.6f,
                            center = Offset(mx, my)
                        )
                    }

                    // Main Marker Circle
                    drawCircle(
                        color = markerColor,
                        radius = radius,
                        center = Offset(mx, my)
                    )
                    drawCircle(
                        color = Color.White,
                        radius = radius,
                        center = Offset(mx, my),
                        style = Stroke(width = 2f)
                    )
                }
            }

            // Interactive Marker Overlays for touch events
            BoxWithConstraints(modifier = Modifier.fillMaxSize()) {
                val boxWidth = maxWidth
                val boxHeight = maxHeight

                markers.forEach { marker ->
                    val mx = (marker.posX * boxWidth.value * scale) + (offset.x / 2.5f)
                    val my = (marker.posY * boxHeight.value * scale) + (offset.y / 2.5f)

                    Surface(
                        shape = CircleShape,
                        color = if (marker.isOn) CyanAccent else Slate800,
                        border = BorderStroke(1.5.dp, Color.White),
                        modifier = Modifier
                            .offset(x = mx.dp - 18.dp, y = my.dp - 18.dp)
                            .size(36.dp)
                            .clickable {
                                marker.isOn = !marker.isOn
                                selectedMarker = marker
                                onEntityClick(marker.entityId)
                            }
                    ) {
                        Box(contentAlignment = Alignment.Center) {
                            Text(
                                text = when (marker.domain) {
                                    "light" -> "💡"
                                    "climate" -> "🌡️"
                                    "camera" -> "📹"
                                    else -> "🔌"
                                },
                                fontSize = 16.sp
                            )
                        }
                    }
                }
            }

            // Selected Marker Popup Inspector
            selectedMarker?.let { marker ->
                Surface(
                    shape = RoundedCornerShape(16.dp),
                    color = Slate800.copy(alpha = 0.95f),
                    border = BorderStroke(1.dp, CyanAccent.copy(alpha = 0.5f)),
                    modifier = Modifier
                        .align(Alignment.BottomCenter)
                        .padding(16.dp)
                        .fillMaxWidth(0.9f)
                ) {
                    Row(
                        modifier = Modifier.padding(16.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Column {
                            Text(
                                text = marker.name,
                                fontWeight = FontWeight.Bold,
                                color = Color.White,
                                fontSize = 15.sp
                            )
                            Spacer(modifier = Modifier.height(2.dp))
                            Text(
                                text = "Stato: ${if (marker.isOn) "Attivo (${marker.valueText})" else "Disattivato"}",
                                color = if (marker.isOn) CyanAccent else Color.White.copy(alpha = 0.6f),
                                fontSize = 12.sp
                            )
                        }

                        Button(
                            onClick = {
                                marker.isOn = !marker.isOn
                                onEntityClick(marker.entityId)
                            },
                            colors = ButtonDefaults.buttonColors(
                                containerColor = if (marker.isOn) CyanAccent else IndigoPrimary
                            )
                        ) {
                            Text(if (marker.isOn) "Spegni" else "Accendi", color = if (marker.isOn) Color.Black else Color.White)
                        }
                    }
                }
            }
        }
    }
}
