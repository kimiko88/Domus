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
import kotlin.math.min

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
    private val connectionGeneration = AtomicLong(0)

    private val _connectionFlow = MutableStateFlow(ServerConnection())
    val connectionFlow: StateFlow<ServerConnection> = _connectionFlow.asStateFlow()

    private val _eventFlow = MutableSharedFlow<EntityStateChangedEvent>(replay = 1, extraBufferCapacity = 64)
    val eventFlow: SharedFlow<EntityStateChangedEvent> = _eventFlow.asSharedFlow()

    @Volatile private var currentMode: TransportMode = TransportMode.DISCONNECTED
    @Volatile private var currentUrl: String? = null
    @Volatile private var authToken: String? = null
    @Volatile private var reconnectAttempts: Int = 0
    private var reconnectJob: Job? = null
    private var heartbeatJob: Job? = null

    @Synchronized
    fun connect(mode: TransportMode, endpointUrl: String, token: String? = null) {
        disconnect()
        currentMode = mode
        currentUrl = endpointUrl
        authToken = token
        reconnectAttempts = 0
        _connectionFlow.value = ServerConnection(
            mode = TransportMode.CONNECTING,
            targetMode = mode,
            activeUrl = endpointUrl,
            isConnected = false
        )
        openWebSocket(connectionGeneration.get())
    }

    @Synchronized
    fun disconnect() {
        connectionGeneration.incrementAndGet()
        reconnectJob?.cancel()
        reconnectJob = null
        heartbeatJob?.cancel()
        heartbeatJob = null
        webSocket?.close(1000, "Normal Closure")
        webSocket = null
        currentMode = TransportMode.DISCONNECTED
        currentUrl = null
        authToken = null
        reconnectAttempts = 0
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

    private fun openWebSocket(generation: Long) {
        val endpoint = currentUrl ?: return
        val token = authToken
        val wsUrl = endpoint
            .replace("http://", "ws://")
            .replace("https://", "wss://")
            .let { base -> base.trimEnd('/') + "/ws" }

        val request = Request.Builder()
            .url(wsUrl)
            .apply {
                if (!token.isNullOrBlank()) addHeader("Authorization", "Bearer $token")
            }
            .build()

        val connectStartTime = System.currentTimeMillis()
        val socket = client.newWebSocket(request, createWebSocketListener(generation, connectStartTime))
        webSocket = socket
    }

    private fun createWebSocketListener(generation: Long, connectStartTime: Long): WebSocketListener {
        return object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                if (connectionGeneration.get() != generation) {
                    webSocket.close(1000, "Superseded connection")
                    return
                }
                reconnectAttempts = 0
                val latency = System.currentTimeMillis() - connectStartTime
                _connectionFlow.value = ServerConnection(
                    mode = currentMode,
                    targetMode = currentMode,
                    activeUrl = currentUrl,
                    latencyMs = latency,
                    isConnected = true
                )
                val subMsg = """{"id":${messageIdCounter.getAndIncrement()},"type":"subscribe_events","event_type":"entity_state_changed"}"""
                webSocket.send(subMsg)
                startHeartbeat(webSocket, generation)
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                if (connectionGeneration.get() != generation) return
                try {
                    val msg = json.decodeFromString<DomusWsMessage>(text)
                    when (msg.type) {
                        "event" -> {
                            msg.event?.let { evtObj ->
                                val evt = json.decodeFromJsonElement<EntityStateChangedEvent>(evtObj)
                                coroutineScope.launch {
                                    if (connectionGeneration.get() == generation) _eventFlow.emit(evt)
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
                } catch (_: Exception) {
                    // Ignore non-json or unparseable debug frames.
                }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                if (connectionGeneration.get() != generation) return
                if (this@DualTransportManager.webSocket === webSocket) {
                    this@DualTransportManager.webSocket = null
                }
                _connectionFlow.value = ServerConnection(
                    mode = TransportMode.DISCONNECTED,
                    targetMode = currentMode,
                    activeUrl = currentUrl,
                    isConnected = false,
                    lastError = t.message ?: response?.message
                )
                scheduleReconnect(generation)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                if (connectionGeneration.get() != generation) return
                if (this@DualTransportManager.webSocket === webSocket) {
                    this@DualTransportManager.webSocket = null
                }
                _connectionFlow.value = ServerConnection(
                    mode = TransportMode.DISCONNECTED,
                    targetMode = currentMode,
                    activeUrl = currentUrl,
                    isConnected = false,
                    lastError = "WebSocket closed ($code)"
                )
                // Unexpected clean closes (e.g. server restart) must reconnect too.
                scheduleReconnect(generation)
            }
        }
    }

    private fun startHeartbeat(socket: WebSocket, generation: Long) {
        heartbeatJob?.cancel()
        heartbeatJob = coroutineScope.launch {
            while (isActive && connectionGeneration.get() == generation && _connectionFlow.value.isConnected) {
                delay(30_000L)
                if (connectionGeneration.get() != generation) break
                if (!socket.send("""{"type":"ping"}""")) {
                    scheduleReconnect(generation)
                    break
                }
            }
        }
    }

    @Synchronized
    private fun scheduleReconnect(generation: Long) {
        if (connectionGeneration.get() != generation || currentMode == TransportMode.DISCONNECTED) return
        if (reconnectJob?.isActive == true) return
        val attempt = reconnectAttempts.coerceAtMost(5)
        reconnectAttempts = (reconnectAttempts + 1).coerceAtMost(6)
        val delayMs = min(30_000L, 1_000L * (1L shl attempt))
        reconnectJob = coroutineScope.launch {
            delay(delayMs)
            if (connectionGeneration.get() != generation || currentMode == TransportMode.DISCONNECTED) return@launch
            val endpoint = currentUrl
            if (endpoint == null) return@launch
            _connectionFlow.value = ServerConnection(
                mode = TransportMode.CONNECTING,
                targetMode = currentMode,
                activeUrl = endpoint,
                isConnected = false
            )
            openWebSocket(generation)
        }
    }
}
