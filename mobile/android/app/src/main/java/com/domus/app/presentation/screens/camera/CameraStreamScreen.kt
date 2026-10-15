package com.domus.app.presentation.screens.camera

import android.app.Activity
import android.app.PictureInPictureParams
import android.os.Build
import android.util.Rational
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.presentation.theme.AmberWarning
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.RoseDanger
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.theme.Slate900

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CameraStreamScreen(
    cameraName: String = "Ingresso Principale",
    onDismiss: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current
    var isPtzActive by remember { mutableStateOf(false) }
    var isNightVision by remember { mutableStateOf(false) }
    var isAiTrackingEnabled by remember { mutableStateOf(true) }
    var isMicActive by remember { mutableStateOf(false) }
    var snapshotTakenNotification by remember { mutableStateOf(false) }

    // Pulsing recording dot
    val infiniteTransition = rememberInfiniteTransition(label = "cam_rec")
    val recAlpha by infiniteTransition.animateFloat(
        initialValue = 0.3f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(800, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "rec_pulse"
    )

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
                    text = cameraName,
                    color = Color.White,
                    fontWeight = FontWeight.Bold,
                    fontSize = 16.sp
                )
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Box(
                        modifier = Modifier
                            .size(8.dp)
                            .background(RoseDanger.copy(alpha = recAlpha), CircleShape)
                    )
                    Spacer(modifier = Modifier.width(6.dp))
                    Text(
                        text = "LIVE • WebRTC 1080p @ 30fps",
                        color = Color.White.copy(alpha = 0.7f),
                        fontSize = 11.sp
                    )
                }
            }

            // PiP Button
            IconButton(
                onClick = {
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                        val activity = context as? Activity
                        val params = PictureInPictureParams.Builder()
                            .setAspectRatio(Rational(16, 9))
                            .build()
                        activity?.enterPictureInPictureMode(params)
                    }
                }
            ) {
                Text("⤢", fontSize = 22.sp, color = CyanAccent)
            }
        }

        // Camera Video Surface & AI HUD Overlay
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .aspectRatio(16f / 9f)
                .background(Color.Black)
                .border(1.dp, Slate700)
        ) {
            // Simulated live feed background with IR night mode tint
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(
                        if (isNightVision) Color(0xFF1E2822) else Color(0xFF121820)
                    )
            ) {
                // Camera Grid lines
                Column(modifier = Modifier.fillMaxSize()) {
                    Spacer(modifier = Modifier.weight(1f))
                    Divider(color = Color.White.copy(alpha = 0.1f), thickness = 1.dp)
                    Spacer(modifier = Modifier.weight(1f))
                    Divider(color = Color.White.copy(alpha = 0.1f), thickness = 1.dp)
                    Spacer(modifier = Modifier.weight(1f))
                }
                Row(modifier = Modifier.fillMaxSize()) {
                    Spacer(modifier = Modifier.weight(1f))
                    Box(modifier = Modifier.fillMaxHeight().width(1.dp).background(Color.White.copy(alpha = 0.1f)))
                    Spacer(modifier = Modifier.weight(1f))
                    Box(modifier = Modifier.fillMaxHeight().width(1.dp).background(Color.White.copy(alpha = 0.1f)))
                    Spacer(modifier = Modifier.weight(1f))
                }

                // AI Detection Bounding Box (Persona rilevata)
                if (isAiTrackingEnabled) {
                    Box(
                        modifier = Modifier
                            .offset(x = 60.dp, y = 35.dp)
                            .size(width = 110.dp, height = 130.dp)
                            .border(2.dp, CyanAccent, RoundedCornerShape(8.dp))
                            .background(CyanAccent.copy(alpha = 0.08f))
                    ) {
                        Surface(
                            shape = RoundedCornerShape(topStart = 8.dp, bottomEnd = 8.dp),
                            color = CyanAccent,
                            modifier = Modifier.align(Alignment.TopStart)
                        ) {
                            Text(
                                text = "👤 Persona • 98%",
                                color = Color.Black,
                                fontSize = 10.sp,
                                fontWeight = FontWeight.Bold,
                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                            )
                        }
                    }
                }
            }

            // Timestamp HUD
            Surface(
                shape = RoundedCornerShape(6.dp),
                color = Color.Black.copy(alpha = 0.65f),
                modifier = Modifier
                    .align(Alignment.TopStart)
                    .padding(12.dp)
            ) {
                Text(
                    text = "CAM_01 • 2026-10-15 12:00:44",
                    color = Color.White,
                    fontSize = 11.sp,
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                )
            }

            // Snapshot feedback toast
            AnimatedVisibility(
                visible = snapshotTakenNotification,
                modifier = Modifier.align(Alignment.Center)
            ) {
                Surface(
                    shape = RoundedCornerShape(12.dp),
                    color = EmeraldSuccess.copy(alpha = 0.9f)
                ) {
                    Text(
                        text = "📸 Fotogramma salvato in Galleria",
                        color = Color.White,
                        fontWeight = FontWeight.Bold,
                        fontSize = 13.sp,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
                    )
                }
            }
        }

        // Camera Quick Action Bar
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.SpaceEvenly
        ) {
            // Snapshot Button
            FilterChip(
                selected = false,
                onClick = { snapshotTakenNotification = true },
                label = { Text("Scatta Foto") },
                leadingIcon = { Text("📸") }
            )

            // Night Vision Toggle
            FilterChip(
                selected = isNightVision,
                onClick = { isNightVision = !isNightVision },
                label = { Text(if (isNightVision) "IR On" else "IR Auto") },
                leadingIcon = { Text(if (isNightVision) "🌙" else "☀️") }
            )

            // AI Smart Tracking Toggle
            FilterChip(
                selected = isAiTrackingEnabled,
                onClick = { isAiTrackingEnabled = !isAiTrackingEnabled },
                label = { Text("Smart AI") },
                leadingIcon = { Text("🤖") }
            )

            // PTZ D-Pad Toggle
            FilterChip(
                selected = isPtzActive,
                onClick = { isPtzActive = !isPtzActive },
                label = { Text("PTZ") },
                leadingIcon = { Text("🕹️") }
            )
        }

        Spacer(modifier = Modifier.height(10.dp))

        // Intercom Two-Way Talk & PTZ D-Pad Controls
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 24.dp),
            horizontalArrangement = Arrangement.SpaceAround,
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Intercom Push-To-Talk
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                modifier = Modifier
                    .semantics { contentDescription = "Pulsante Interfono Bidirezionale" }
            ) {
                Surface(
                    shape = CircleShape,
                    color = if (isMicActive) RoseDanger else Slate800,
                    border = BorderStroke(2.dp, if (isMicActive) RoseDanger else CyanAccent),
                    modifier = Modifier
                        .size(68.dp)
                        .clickable { isMicActive = !isMicActive }
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Text(
                            text = if (isMicActive) "🎙️" else "🎤",
                            fontSize = 28.sp
                        )
                    }
                }
                Spacer(modifier = Modifier.height(6.dp))
                Text(
                    text = if (isMicActive) "Parla ora..." else "Tocca per parlare",
                    color = if (isMicActive) RoseDanger else Color.White.copy(alpha = 0.7f),
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold
                )
            }

            // Directional PTZ D-Pad
            if (isPtzActive) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Button(
                        onClick = { /* Pan Up */ },
                        shape = RoundedCornerShape(8.dp),
                        colors = ButtonDefaults.buttonColors(containerColor = Slate800),
                        modifier = Modifier.size(48.dp),
                        contentPadding = PaddingValues(0.dp)
                    ) {
                        Text("▲", color = CyanAccent, fontSize = 16.sp)
                    }

                    Row(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Button(
                            onClick = { /* Pan Left */ },
                            shape = RoundedCornerShape(8.dp),
                            colors = ButtonDefaults.buttonColors(containerColor = Slate800),
                            modifier = Modifier.size(48.dp),
                            contentPadding = PaddingValues(0.dp)
                        ) {
                            Text("◀", color = CyanAccent, fontSize = 16.sp)
                        }

                        Surface(
                            shape = CircleShape,
                            color = Slate700,
                            modifier = Modifier.size(36.dp)
                        ) {
                            Box(contentAlignment = Alignment.Center) {
                                Text("⌖", color = Color.White, fontSize = 18.sp)
                            }
                        }

                        Button(
                            onClick = { /* Pan Right */ },
                            shape = RoundedCornerShape(8.dp),
                            colors = ButtonDefaults.buttonColors(containerColor = Slate800),
                            modifier = Modifier.size(48.dp),
                            contentPadding = PaddingValues(0.dp)
                        ) {
                            Text("▶", color = CyanAccent, fontSize = 16.sp)
                        }
                    }

                    Button(
                        onClick = { /* Pan Down */ },
                        shape = RoundedCornerShape(8.dp),
                        colors = ButtonDefaults.buttonColors(containerColor = Slate800),
                        modifier = Modifier.size(48.dp),
                        contentPadding = PaddingValues(0.dp)
                    ) {
                        Text("▼", color = CyanAccent, fontSize = 16.sp)
                    }
                }
            }
        }
    }
}
