package com.domus.app.presentation.screens.assistant

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.*
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.domus.app.presentation.theme.CyanAccent
import com.domus.app.presentation.theme.EmeraldSuccess
import com.domus.app.presentation.theme.IndigoPrimary
import com.domus.app.presentation.theme.RoseDanger
import com.domus.app.presentation.theme.Slate700
import com.domus.app.presentation.theme.Slate800
import com.domus.app.presentation.theme.Slate900
import com.domus.app.presentation.viewmodel.DashboardViewModel
import kotlinx.coroutines.launch

data class ChatMessage(
    val id: String,
    val text: String,
    val isUser: Boolean,
    val timestamp: String,
    val executedAction: ActionCard? = null
)

data class ActionCard(
    val title: String,
    val domain: String,
    val status: String,
    val isSuccess: Boolean = true
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AssistantChatScreen(
    viewModel: DashboardViewModel? = null,
    onDismiss: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    val coroutineScope = rememberCoroutineScope()
    val listState = rememberLazyListState()

    var inputText by remember { mutableStateOf("") }
    var isListening by remember { mutableStateOf(false) }

    val messages = remember {
        mutableStateListOf(
            ChatMessage(
                id = "welcome",
                text = "Ciao! Sono il tuo assistente locale Domus. Come posso aiutarti con la tua casa?",
                isUser = false,
                timestamp = "Ora"
            )
        )
    }

    val suggestions = listOf(
        "Spegni tutte le luci",
        "Arma l'allarme notte",
        "Metti il termostato a 21°C",
        "Stato telecamere esterne"
    )

    // Animated waveform bars for listening state
    val infiniteTransition = rememberInfiniteTransition(label = "audio_bars")
    val barHeight1 by infiniteTransition.animateFloat(
        initialValue = 12f,
        targetValue = 38f,
        animationSpec = infiniteRepeatable(tween(400, easing = FastOutSlowInEasing), RepeatMode.Reverse),
        label = "bar1"
    )
    val barHeight2 by infiniteTransition.animateFloat(
        initialValue = 28f,
        targetValue = 10f,
        animationSpec = infiniteRepeatable(tween(350, easing = FastOutSlowInEasing), RepeatMode.Reverse),
        label = "bar2"
    )
    val barHeight3 by infiniteTransition.animateFloat(
        initialValue = 15f,
        targetValue = 42f,
        animationSpec = infiniteRepeatable(tween(480, easing = FastOutSlowInEasing), RepeatMode.Reverse),
        label = "bar3"
    )

    fun sendMessage(text: String) {
        if (text.isBlank()) return
        val userMsg = ChatMessage(
            id = System.currentTimeMillis().toString(),
            text = text.trim(),
            isUser = true,
            timestamp = "Ora"
        )
        messages.add(userMsg)
        inputText = ""

        coroutineScope.launch {
            val response = viewModel?.sendAssistantPrompt(text.trim())
            val reply = if (response != null) {
                val action = response.actions_executed.firstOrNull()?.let {
                    ActionCard(
                        title = "${it.entity_key} • ${it.service}",
                        domain = it.entity_key.substringBefore("."),
                        status = if (it.success) "Eseguito con successo" else "Errore esecuzione",
                        isSuccess = it.success
                    )
                }
                ChatMessage(
                    id = System.currentTimeMillis().toString(),
                    text = response.response_text.ifBlank { "Comando elaborato con successo." },
                    isUser = false,
                    timestamp = "Ora",
                    executedAction = action
                )
            } else {
                ChatMessage(
                    id = System.currentTimeMillis().toString(),
                    text = "Impossibile contattare l'assistente locale. Verifica la connessione al server Domus.",
                    isUser = false,
                    timestamp = "Ora"
                )
            }
            messages.add(reply)
            listState.animateScrollToItem(messages.size - 1)
        }
    }

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
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = "Assistente Domus",
                        fontWeight = FontWeight.Bold,
                        fontSize = 17.sp,
                        color = Color.White
                    )
                    Spacer(modifier = Modifier.width(6.dp))
                    Surface(
                        shape = RoundedCornerShape(6.dp),
                        color = CyanAccent.copy(alpha = 0.15f)
                    ) {
                        Text(
                            text = "Local LLM",
                            color = CyanAccent,
                            fontSize = 10.sp,
                            fontWeight = FontWeight.Bold,
                            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                        )
                    }
                }
                Text(
                    text = "Privacy Totale • Zero Cloud",
                    fontSize = 11.sp,
                    color = Color.White.copy(alpha = 0.6f)
                )
            }

            Spacer(modifier = Modifier.size(40.dp))
        }

        // Suggestions Horizontal Scroll
        Row(
            modifier = Modifier
                .horizontalScroll(rememberScrollState())
                .padding(horizontal = 16.dp, vertical = 6.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            suggestions.forEach { suggestion ->
                Surface(
                    shape = RoundedCornerShape(20.dp),
                    color = Slate800,
                    border = BorderStroke(1.dp, Slate700),
                    modifier = Modifier.clickable { sendMessage(suggestion) }
                ) {
                    Text(
                        text = suggestion,
                        fontSize = 12.sp,
                        color = CyanAccent,
                        modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp)
                    )
                }
            }
        }

        // Chat Message Stream
        LazyColumn(
            state = listState,
            modifier = Modifier
                .fillMaxWidth()
                .weight(1f)
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            contentPadding = PaddingValues(vertical = 12.dp)
        ) {
            items(messages, key = { it.id }) { msg ->
                Column(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalAlignment = if (msg.isUser) Alignment.End else Alignment.Start
                ) {
                    Surface(
                        shape = RoundedCornerShape(
                            topStart = 16.dp,
                            topEnd = 16.dp,
                            bottomStart = if (msg.isUser) 16.dp else 4.dp,
                            bottomEnd = if (msg.isUser) 4.dp else 16.dp
                        ),
                        color = if (msg.isUser) IndigoPrimary else Slate800,
                        border = BorderStroke(1.dp, if (msg.isUser) CyanAccent.copy(alpha = 0.5f) else Slate700),
                        modifier = Modifier.widthIn(max = 300.dp)
                    ) {
                        Column(modifier = Modifier.padding(12.dp)) {
                            Text(
                                text = msg.text,
                                color = Color.White,
                                fontSize = 14.sp,
                                lineHeight = 20.sp
                            )

                            // Executed Action Card
                            msg.executedAction?.let { action ->
                                Spacer(modifier = Modifier.height(8.dp))
                                Surface(
                                    shape = RoundedCornerShape(10.dp),
                                    color = Slate900.copy(alpha = 0.8f),
                                    border = BorderStroke(1.dp, EmeraldSuccess.copy(alpha = 0.5f)),
                                    modifier = Modifier.fillMaxWidth()
                                ) {
                                    Row(
                                        modifier = Modifier.padding(8.dp),
                                        verticalAlignment = Alignment.CenterVertically
                                    ) {
                                        Text("⚡", fontSize = 16.sp)
                                        Spacer(modifier = Modifier.width(6.dp))
                                        Column {
                                            Text(
                                                text = action.title,
                                                fontWeight = FontWeight.Bold,
                                                color = Color.White,
                                                fontSize = 12.sp
                                            )
                                            Text(
                                                text = action.status,
                                                color = EmeraldSuccess,
                                                fontSize = 11.sp
                                            )
                                        }
                                    }
                                }
                            }
                        }
                    }

                    Text(
                        text = msg.timestamp,
                        color = Color.White.copy(alpha = 0.4f),
                        fontSize = 10.sp,
                        modifier = Modifier.padding(top = 2.dp, start = 4.dp, end = 4.dp)
                    )
                }
            }
        }

        // Voice Visualizer HUD when listening
        AnimatedVisibility(visible = isListening) {
            Surface(
                color = Slate800,
                border = BorderStroke(1.dp, CyanAccent.copy(alpha = 0.5f)),
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 6.dp)
                    .clip(RoundedCornerShape(14.dp))
            ) {
                Row(
                    modifier = Modifier.padding(12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.Center
                ) {
                    Text("Ascolto in corso...", color = CyanAccent, fontWeight = FontWeight.Bold, fontSize = 13.sp)
                    Spacer(modifier = Modifier.width(16.dp))
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(4.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Box(modifier = Modifier.width(4.dp).height(barHeight1.dp).background(CyanAccent, CircleShape))
                        Box(modifier = Modifier.width(4.dp).height(barHeight2.dp).background(CyanAccent, CircleShape))
                        Box(modifier = Modifier.width(4.dp).height(barHeight3.dp).background(CyanAccent, CircleShape))
                        Box(modifier = Modifier.width(4.dp).height(barHeight2.dp).background(CyanAccent, CircleShape))
                        Box(modifier = Modifier.width(4.dp).height(barHeight1.dp).background(CyanAccent, CircleShape))
                    }
                }
            }
        }

        // Bottom Input Bar & Microphone
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            OutlinedTextField(
                value = inputText,
                onValueChange = { inputText = it },
                placeholder = { Text("Chiedi qualcosa all'assistente...", fontSize = 14.sp) },
                singleLine = true,
                shape = RoundedCornerShape(24.dp),
                colors = OutlinedTextFieldDefaults.colors(
                    focusedBorderColor = CyanAccent,
                    unfocusedBorderColor = Slate700
                ),
                modifier = Modifier.weight(1f)
            )

            Spacer(modifier = Modifier.width(8.dp))

            // Microphone Button with push-to-talk
            Surface(
                shape = CircleShape,
                color = if (isListening) RoseDanger else Slate800,
                border = BorderStroke(1.5.dp, if (isListening) RoseDanger else CyanAccent),
                modifier = Modifier
                    .size(48.dp)
                    .clickable {
                        isListening = !isListening
                    }
            ) {
                Box(contentAlignment = Alignment.Center) {
                    Text(if (isListening) "⏹" else "🎙️", fontSize = 20.sp)
                }
            }

            if (inputText.isNotBlank()) {
                Spacer(modifier = Modifier.width(6.dp))
                Button(
                    onClick = { sendMessage(inputText) },
                    shape = CircleShape,
                    colors = ButtonDefaults.buttonColors(containerColor = CyanAccent),
                    contentPadding = PaddingValues(0.dp),
                    modifier = Modifier.size(48.dp)
                ) {
                    Text("➤", color = Color.Black, fontSize = 16.sp)
                }
            }
        }
    }
}
