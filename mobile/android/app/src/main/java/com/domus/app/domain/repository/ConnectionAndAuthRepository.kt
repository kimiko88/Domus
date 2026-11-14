package com.domus.app.domain.repository

import com.domus.app.domain.model.AuthSession
import com.domus.app.domain.model.ServerConnection
import com.domus.app.domain.model.TransportMode
import kotlinx.coroutines.flow.Flow

interface ConnectionRepository {
    val connectionFlow: Flow<ServerConnection>

    suspend fun discoverLocalHub(timeoutMs: Long = 2000L): String?
    suspend fun connect(targetMode: TransportMode? = null): Result<Unit>
    suspend fun disconnect()
    suspend fun setEndpoints(localAddress: String?, remoteAddress: String?)
    fun getCurrentConnection(): ServerConnection
}

interface AuthRepository {
    val sessionFlow: Flow<AuthSession>

    suspend fun login(username: String, password: String, twoFactorCode: String? = null): Result<AuthSession>
    suspend fun logout()
    suspend fun getSavedToken(): String?
    suspend fun restoreSession(): AuthSession
}
