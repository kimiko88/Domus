package com.domus.app.data.network

import com.domus.app.domain.model.ServerConnection
import com.domus.app.domain.model.TransportMode
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromJsonElement
import okhttp3.*
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicLong

class DualTransportManager(
    private val client: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(5, TimeUnit.SECONDS)
        .readTimeout(10, TimeUnit.SECONDS)
        .build(),
    private val json: Json = Json { ignoreUnknownKeys = true; isLenient = true },
    private val coroutineScope: CoroutineScope = CoroutineScope(Dispatchers.IO + SupervisorJob())
) {
    private var webSocket: WebSocket? = null
    private val messageIdCounter = AtomicLong(1)

    private val _connectionFlow = MutableStateFlow(ServerConnection())
    val connectionFlow: StateFlow<ServerConnection> = _connectionFlow.asStateFlow()

    private val _eventFlow = MutableSharedFlow<EntityStateChangedEvent>(replay = 1, extraBufferCapacity = 64)
    val eventFlow: SharedFlow<EntityStateChangedEvent> = _eventFlow.asSharedFlow()

    private var currentMode: TransportMode = TransportMode.DISCONNECTED
    private var currentUrl: String? = null
    private var authToken: String? = null
    private var reconnectJob: Job? = null
    private var heartbeatJob: Job? = null

    fun connect(mode: TransportMode, endpointUrl: String, token: String? = null) {
        disconnect()

        currentMode = mode
        currentUrl = endpointUrl
        authToken = token

        _connectionFlow.value = ServerConnection(
            mode = TransportMode.CONNECTING,
            targetMode = mode,
            activeUrl = endpointUrl,
            isConnected = false
        )

        val wsUrl = endpointUrl
            .replace("http://", "ws://")
            .replace("https://", "wss://")
            .let { base -> base.trimEnd('/') + "/ws" }

        val request = Request.Builder()
            .url(wsUrl)
            .apply {
                if (!token.isNullOrBlank()) {
                    addHeader("Authorization", "Bearer $token")
                }
            }
            .build()

        webSocket = client.newWebSocket(request, createWebSocketListener())
    }

    fun disconnect() {
        reconnectJob?.cancel()
        heartbeatJob?.cancel()
        webSocket?.close(1000, "Normal Closure")
        webSocket = null
        currentMode = TransportMode.DISCONNECTED
        _connectionFlow.value = ServerConnection(
            mode = TransportMode.DISCONNECTED,
            targetMode = TransportMode.DISCONNECTED,
            isConnected = false
        )
    }

    fun sendCommand(domain: String, service: String, entityId: String, data: Map<String, Any> = emptyMap()): Long {
        val id = messageIdCounter.getAndIncrement()
        val payload = kotlinx.serialization.json.buildJsonObject {
            put("id", kotlinx.serialization.json.JsonPrimitive(id))
            put("type", kotlinx.serialization.json.JsonPrimitive("call_service"))
            put("domain", kotlinx.serialization.json.JsonPrimitive(domain))
            put("service", kotlinx.serialization.json.JsonPrimitive(service))
            put("entity_id", kotlinx.serialization.json.JsonPrimitive(entityId))
            put("data", kotlinx.serialization.json.buildJsonObject {
                data.forEach { (key, value) ->
                    put(key, when (value) {
                        is Boolean -> kotlinx.serialization.json.JsonPrimitive(value)
                        is Number -> kotlinx.serialization.json.JsonPrimitive(value)
                        else -> kotlinx.serialization.json.JsonPrimitive(value.toString())
                    })
                }
            })
        }
        webSocket?.send(payload.toString())
        return id
    }

    private fun createWebSocketListener(): WebSocketListener {
        val connectStartTime = System.currentTimeMillis()

        return object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                val latency = System.currentTimeMillis() - connectStartTime
                _connectionFlow.value = ServerConnection(
                    mode = currentMode,
                    targetMode = currentMode,
                    activeUrl = currentUrl,
                    latencyMs = latency,
                    isConnected = true
                )

                // Subscribe to state change events
                val subMsg = """{"id":${messageIdCounter.getAndIncrement()},"type":"subscribe_events","event_type":"entity_state_changed"}"""
                webSocket.send(subMsg)

                startHeartbeat()
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                try {
                    val msg = json.decodeFromString<DomusWsMessage>(text)
                    when (msg.type) {
                        "event" -> {
                            msg.event?.let { evtObj ->
                                val evt = json.decodeFromJsonElement<EntityStateChangedEvent>(evtObj)
                                coroutineScope.launch {
                                    _eventFlow.emit(evt)
                                }
                            }
                        }
                        "pong" -> {
                            val currentLatency = _connectionFlow.value.latencyMs
                            _connectionFlow.value = _connectionFlow.value.copy(
                                latencyMs = (currentLatency + 12) / 2
                            )
                        }
                    }
                } catch (e: Exception) {
                    // Ignore non-json or unparseable debug frames
                }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                _connectionFlow.value = ServerConnection(
                    mode = TransportMode.DISCONNECTED,
                    isConnected = false,
                    lastError = t.message
                )
                scheduleReconnect()
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                _connectionFlow.value = ServerConnection(
                    mode = TransportMode.DISCONNECTED,
                    isConnected = false
                )
            }
        }
    }

    private fun startHeartbeat() {
        heartbeatJob?.cancel()
        heartbeatJob = coroutineScope.launch {
            while (isActive && _connectionFlow.value.isConnected) {
                delay(30_000L)
                webSocket?.send("""{"type":"ping"}""")
            }
        }
    }

    private fun scheduleReconnect() {
        if (currentMode == TransportMode.DISCONNECTED) return
        reconnectJob?.cancel()
        reconnectJob = coroutineScope.launch {
            delay(3000L)
            currentUrl?.let { url ->
                connect(currentMode, url, authToken)
            }
        }
    }
}
