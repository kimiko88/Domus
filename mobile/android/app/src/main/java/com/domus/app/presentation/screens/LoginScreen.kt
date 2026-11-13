package com.domus.app.presentation.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.IndigoPrimary

@Composable
fun LoginScreen(
    onLogin: (String, String, String, String?) -> Unit,
    onDiscoverLan: () -> Unit,
    discoveredHubUrl: String?,
    isLoading: Boolean,
    errorMessage: String?,
    requiresTwoFactor: Boolean = false,
    modifier: Modifier = Modifier
) {
    var hubUrl by remember { mutableStateOf(discoveredHubUrl ?: "http://192.168.1.100:8080") }
    var username by remember { mutableStateOf("admin") }
    var password by remember { mutableStateOf("")
    var twoFactorCode by remember { mutableStateOf("") }

    LaunchedEffect(discoveredHubUrl) {
        if (!discoveredHubUrl.isNullOrBlank()) {
            hubUrl = discoveredHubUrl
        }
    }

    Box(
        modifier = modifier
            .fillMaxSize()
            .padding(24.dp),
        contentAlignment = Alignment.Center
    ) {
        Card(
            shape = RoundedCornerShape(24.dp),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
            elevation = CardDefaults.cardElevation(defaultElevation = 6.dp),
            modifier = Modifier.fillMaxWidth()
        ) {
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                modifier = Modifier.padding(24.dp)
            ) {
                Text(
                    "Domus Companion",
                    fontWeight = FontWeight.Black,
                    fontSize = 24.sp,
                    color = MaterialTheme.colorScheme.onSurface
                )
                Text(
                    "Connettiti al tuo hub domestico",
                    fontSize = 13.sp,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                )

                Spacer(modifier = Modifier.height(24.dp))

                OutlinedTextField(
                    value = hubUrl,
                    onValueChange = { hubUrl = it },
                    label = { Text("Indirizzo Hub (LAN o Remoto)") },
                    singleLine = true,
                    shape = RoundedCornerShape(12.dp),
                    modifier = Modifier.fillMaxWidth()
                )

                Spacer(modifier = Modifier.height(8.dp))

                OutlinedButton(
                    onClick = onDiscoverLan,
                    shape = RoundedCornerShape(12.dp),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Text("🔍 Trova Hub in LAN (Zero-Config mDNS)")
                }

                Spacer(modifier = Modifier.height(16.dp))

                OutlinedTextField(
                    value = username,
                    onValueChange = { username = it },
                    label = { Text("Nome Utente") },
                    singleLine = true,
                    shape = RoundedCornerShape(12.dp),
                    modifier = Modifier.fillMaxWidth()
                )

                Spacer(modifier = Modifier.height(12.dp))

                OutlinedTextField(
                    value = password,
                    onValueChange = { password = it },
                    label = { Text("Password") },
                    visualTransformation = PasswordVisualTransformation(),
                    singleLine = true,
                    shape = RoundedCornerShape(12.dp),
                    modifier = Modifier.fillMaxWidth()
                )

                if (requiresTwoFactor) {
                    Spacer(modifier = Modifier.height(12.dp))
                    OutlinedTextField(
                        value = twoFactorCode,
                        onValueChange = { twoFactorCode = it.trim().take(32) },
                        label = { Text("Codice 2FA / recupero") },
                        singleLine = true,
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth()
                    )
                }

                if (!errorMessage.isNullOrBlank()) {
                    Spacer(modifier = Modifier.height(12.dp))
                    Text(
                        text = errorMessage,
                        color = MaterialTheme.colorScheme.error,
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold
                    )
                }

                Spacer(modifier = Modifier.height(24.dp))

                Button(
                    onClick = { onLogin(hubUrl, username, password, twoFactorCode.takeIf { requiresTwoFactor }) },
                    enabled = !isLoading && username.isNotBlank() && password.isNotBlank(),
                    shape = RoundedCornerShape(12.dp),
                    colors = ButtonDefaults.buttonColors(containerColor = IndigoPrimary),
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(50.dp)
                ) {
                    if (isLoading) {
                        CircularProgressIndicator(color = CyanAccent, modifier = Modifier.size(24.dp))
                    } else {
                        Text("Accedi all'Hub", fontWeight = FontWeight.Bold, fontSize = 15.sp)
                    }
                }
            }
        }
    }
}
