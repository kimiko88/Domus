package com.domus.app.presentation.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

val Slate900 = Color(0xFF0F172A)
val Slate800 = Color(0xFF1E293B)
val Slate700 = Color(0xFF334155)
val Slate400 = Color(0xFF94A3B8)
val Slate100 = Color(0xFFF1F5F9)

val CyanAccent = Color(0xFF00D2FF)
val IndigoPrimary = Color(0xFF6366F1)
val EmeraldSuccess = Color(0xFF10B981)
val AmberWarning = Color(0xFFF59E0B)
val RoseError = Color(0xFFF43F5E)

private val DarkColorScheme = darkColorScheme(
    primary = CyanAccent,
    secondary = IndigoPrimary,
    background = Slate900,
    surface = Slate800,
    onPrimary = Color.Black,
    onSecondary = Color.White,
    onBackground = Slate100,
    onSurface = Slate100
)

private val LightColorScheme = lightColorScheme(
    primary = IndigoPrimary,
    secondary = CyanAccent,
    background = Color(0xFFF8FAFC),
    surface = Color.White,
    onPrimary = Color.White,
    onSecondary = Color.Black,
    onBackground = Slate900,
    onSurface = Slate900
)

@Composable
fun DomusTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit
) {
    val colorScheme = if (darkTheme) DarkColorScheme else LightColorScheme
    MaterialTheme(
        colorScheme = colorScheme,
        content = content
    )
}
