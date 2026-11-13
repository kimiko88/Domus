package com.domus.app.data.repository

import com.domus.app.data.mdns.MdnsDiscoveryEngine
import com.domus.app.data.network.DomusApiClient
import com.domus.app.data.network.DualTransportManager
import com.domus.app.data.storage.TokenStorage
import com.domus.app.domain.model.AuthSession
import com.domus.app.domain.model.ServerConnection
import com.domus.app.domain.model.TransportMode
import com.domus.app.domain.repository.AuthRepository
import com.domus.app.domain.repository.ConnectionRepository
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow

class DefaultConnectionRepository(
    private val mdnsEngine: MdnsDiscoveryEngine,
    private val transportManager: DualTransportManager,
    private val tokenStorage: TokenStorage
) : ConnectionRepository {

    override val connectionFlow: Flow<ServerConnection> = transportManager.connectionFlow

    override suspend fun discoverLocalHub(timeoutMs: Long): String? {
        val hub = mdnsEngine.discoverLocalHub(timeoutMs)
        return if (hub != null) {
            tokenStorage.saveLocalEndpoint(hub.httpBaseUrl)
            hub.httpBaseUrl
        } else {
            null
        }
    }

    override suspend fun connect(targetMode: TransportMode?): Result<Unit> {
        val mode = targetMode ?: TransportMode.LOCAL_LAN
        val token = tokenStorage.getToken()

        val endpoint = when (mode) {
            TransportMode.LOCAL_LAN -> tokenStorage.getLocalEndpoint() ?: "http://192.168.1.100:8080"
            TransportMode.REMOTE_TLS -> tokenStorage.getRemoteEndpoint() ?: "https://domus.local.myhome.net"
            else -> tokenStorage.getLocalEndpoint() ?: "http://192.168.1.100:8080"
        }

        transportManager.connect(mode, endpoint, token)
        return Result.success(Unit)
    }

    override suspend fun disconnect() {
        transportManager.disconnect()
    }

    override suspend fun setEndpoints(localAddress: String?, remoteAddress: String?) {
        if (localAddress != null) tokenStorage.saveLocalEndpoint(localAddress)
        if (remoteAddress != null) tokenStorage.saveRemoteEndpoint(remoteAddress)
    }

    override fun getCurrentConnection(): ServerConnection {
        return transportManager.connectionFlow.value
    }
}

class DefaultAuthRepository(
    private val apiClient: DomusApiClient,
    private val tokenStorage: TokenStorage,
    private val connectionRepository: ConnectionRepository
) : AuthRepository {

    private val _sessionFlow = MutableStateFlow(AuthSession())
    override val sessionFlow: Flow<AuthSession> = _sessionFlow.asStateFlow()

    override suspend fun login(username: String, password: String): Result<AuthSession> {
        val activeUrl = connectionRepository.getCurrentConnection().activeUrl
            ?: tokenStorage.getLocalEndpoint()
            ?: "http://192.168.1.100:8080"

        val res = apiClient.login(activeUrl, username, password)
        if (res.isSuccess) {
            val session = res.getOrThrow()
            tokenStorage.saveToken(session.token)
            _sessionFlow.value = session
            // Reconnect transport with newly acquired token
            connectionRepository.connect(connectionRepository.getCurrentConnection().mode)
            return Result.success(session)
        }
        return res
    }

    override suspend fun logout() {
        tokenStorage.clear()
        _sessionFlow.value = AuthSession()
        connectionRepository.disconnect()
    }

    override suspend fun getSavedToken(): String? {
        return tokenStorage.getToken()
    }

    override suspend fun restoreSession(): AuthSession {
        val token = tokenStorage.getToken()
        val session = if (!token.isNullOrBlank()) {
            val endpoint = tokenStorage.getLocalEndpoint() ?: tokenStorage.getRemoteEndpoint() ?: "https://domus.local.myhome.net"
            apiClient.validateSession(endpoint, token).getOrElse {
                tokenStorage.saveToken(null)
                AuthSession()
            }
        } else AuthSession()
        _sessionFlow.value = session
        return session
    }
}
