package com.domus.app.unit

import com.domus.app.domain.model.ServerConnection
import com.domus.app.domain.model.TransportMode
import com.domus.app.domain.repository.ConnectionRepository
import com.domus.app.domain.usecase.DiscoverAndConnectUseCase
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class DiscoverAndConnectUseCaseTest {

    @Test
    fun testConnectPrefersLocalWhenDiscovered() = runTest {
        var connectedMode: TransportMode? = null

        val fakeConnectionRepo = object : ConnectionRepository {
            override val connectionFlow: Flow<ServerConnection> = flowOf(ServerConnection())
            override suspend fun discoverLocalHub(timeoutMs: Long): String = "http://192.168.1.100:8080"
            override suspend fun connect(targetMode: TransportMode?): Result<Unit> {
                connectedMode = targetMode
                return Result.success(Unit)
            }
            override suspend fun disconnect() {}
            override suspend fun setEndpoints(localAddress: String?, remoteAddress: String?) {}
            override fun getCurrentConnection(): ServerConnection {
                return ServerConnection(mode = connectedMode ?: TransportMode.DISCONNECTED, isConnected = true)
            }
        }

        val useCase = DiscoverAndConnectUseCase(fakeConnectionRepo)
        val result = useCase(preferLocal = true)

        assertTrue(result.isSuccess)
        assertEquals(TransportMode.LOCAL_LAN, connectedMode)
    }

    @Test
    fun testConnectFallsBackToRemoteWhenLocalUnavailable() = runTest {
        var connectedMode: TransportMode? = null

        val fakeConnectionRepo = object : ConnectionRepository {
            override val connectionFlow: Flow<ServerConnection> = flowOf(ServerConnection())
            override suspend fun discoverLocalHub(timeoutMs: Long): String? = null
            override suspend fun connect(targetMode: TransportMode?): Result<Unit> {
                connectedMode = targetMode
                return Result.success(Unit)
            }
            override suspend fun disconnect() {}
            override suspend fun setEndpoints(localAddress: String?, remoteAddress: String?) {}
            override fun getCurrentConnection(): ServerConnection {
                return ServerConnection(mode = connectedMode ?: TransportMode.DISCONNECTED, isConnected = true)
            }
        }

        val useCase = DiscoverAndConnectUseCase(fakeConnectionRepo)
        val result = useCase(preferLocal = true)

        assertTrue(result.isSuccess)
        assertEquals(TransportMode.REMOTE_TLS, connectedMode)
    }
}
