package com.domus.app.domain.model

enum class TransportMode {
    LOCAL_LAN,
    REMOTE_TLS,
    CONNECTING,
    DISCONNECTED
}

data class ServerConnection(
    val mode: TransportMode = TransportMode.DISCONNECTED,
    val targetMode: TransportMode = TransportMode.DISCONNECTED,
    val localAddress: String? = null,
    val remoteAddress: String? = null,
    val activeUrl: String? = null,
    val latencyMs: Long = 0L,
    val isConnected: Boolean = false,
    val lastError: String? = null
) {
    val isLocal: Boolean get() = (mode == TransportMode.LOCAL_LAN || targetMode == TransportMode.LOCAL_LAN)
    val isRemote: Boolean get() = (mode == TransportMode.REMOTE_TLS || targetMode == TransportMode.REMOTE_TLS)
}

data class AuthSession(
    val token: String? = null,
    val username: String? = null,
    val role: String = "user",
    val isAuthenticated: Boolean = false
)
