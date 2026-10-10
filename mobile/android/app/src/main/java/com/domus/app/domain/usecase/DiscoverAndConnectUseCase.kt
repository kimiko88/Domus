package com.domus.app.domain.usecase

import com.domus.app.domain.model.ServerConnection
import com.domus.app.domain.model.TransportMode
import com.domus.app.domain.repository.ConnectionRepository

class DiscoverAndConnectUseCase(
    private val connectionRepository: ConnectionRepository
) {
    suspend operator fun invoke(
        preferLocal: Boolean = true,
        discoveryTimeoutMs: Long = 1500L
    ): Result<ServerConnection> {
        if (preferLocal) {
            val localHub = connectionRepository.discoverLocalHub(discoveryTimeoutMs)
            if (localHub != null) {
                val connectResult = connectionRepository.connect(TransportMode.LOCAL_LAN)
                if (connectResult.isSuccess) {
                    return Result.success(connectionRepository.getCurrentConnection())
                }
            }
        }

        // Fallback to Remote
        val remoteResult = connectionRepository.connect(TransportMode.REMOTE_TLS)
        return if (remoteResult.isSuccess) {
            Result.success(connectionRepository.getCurrentConnection())
        } else {
            Result.failure(remoteResult.exceptionOrNull() ?: Exception("Impossibile connettersi né in locale né da remoto"))
        }
    }
}
