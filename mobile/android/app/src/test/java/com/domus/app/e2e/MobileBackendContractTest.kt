package com.domus.app.e2e

import com.domus.app.data.mdns.DiscoveredHub
import com.domus.app.data.mdns.MdnsDiscoveryEngine
import com.domus.app.data.mdns.MdnsScanner
import com.domus.app.data.network.DomusApiClient
import com.domus.app.data.network.DualTransportManager
import com.domus.app.data.repository.DefaultAuthRepository
import com.domus.app.data.repository.DefaultConnectionRepository
import com.domus.app.data.repository.DefaultEntityRepository
import com.domus.app.data.storage.InMemoryTokenStorage
import com.domus.app.data.storage.OfflineEntityCache
import com.domus.app.domain.usecase.AuthenticateUseCase
import com.domus.app.domain.usecase.DiscoverAndConnectUseCase
import com.domus.app.domain.usecase.SyncEntitiesUseCase
import com.domus.app.domain.usecase.ToggleEntityUseCase
import kotlinx.coroutines.delay
import kotlinx.coroutines.test.runTest
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import okhttp3.mockwebserver.Dispatcher
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.RecordedRequest
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class MobileBackendContractTest {

    private lateinit var server: MockWebServer

    @Before
    fun setup() {
        server = MockWebServer()
    }

    @After
    fun tearDown() {
        try {
            server.shutdown()
        } catch (_: Exception) {}
    }

    @Test
    fun testCompleteMobileLifecycleE2E() = runTest {
        val wsEvent = """
            {
                "type": "event",
                "event": {
                    "entity_id": "light.kitchen",
                    "new_state": {
                        "entity_id": "light.kitchen",
                        "state": "on",
                        "attributes": {"brightness": 100},
                        "last_changed": "2026-10-10T20:00:00Z"
                    }
                }
            }
        """.trimIndent()

        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                val path = request.path ?: ""
                return when {
                    path.startsWith("/api/v1/ws") -> {
                        MockResponse().withWebSocketUpgrade(object : WebSocketListener() {
                            override fun onOpen(webSocket: WebSocket, response: Response) {
                                webSocket.send(wsEvent)
                            }
                            override fun onMessage(webSocket: WebSocket, text: String) {}
                            override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                                webSocket.close(code, reason)
                            }
                        })
                    }
                    path.startsWith("/api/v1/auth/login") -> {
                        MockResponse()
                            .setResponseCode(200)
                            .setBody("""{"token":"jwt_mobile_secret_token","user":{"id":"usr_1","username":"admin","role":"admin"}}""")
                    }
                    path.startsWith("/api/v1/entities") -> {
                        MockResponse()
                            .setResponseCode(200)
                            .setBody("""[{"id":"light.kitchen","name":"Luce Cucina","domain":"light","state":"off"}]""")
                    }
                    path.startsWith("/api/v1/areas") -> {
                        MockResponse().setResponseCode(200).setBody("[]")
                    }
                    path.startsWith("/api/v1/services") -> {
                        MockResponse().setResponseCode(200).setBody("""{"status":"ok"}""")
                    }
                    else -> MockResponse().setResponseCode(404)
                }
            }
        }
        server.start()

        val serverHost = server.hostName
        val serverPort = server.port
        val baseUrl = server.url("").toString().removeSuffix("/")

        val mockScanner = object : MdnsScanner {
            override suspend fun discoverService(serviceType: String, timeoutMs: Long): DiscoveredHub {
                return DiscoveredHub("Domus Mock Hub", serverHost, serverPort)
            }
        }

        // Build stack
        val tokenStorage = InMemoryTokenStorage(localEndpoint = baseUrl)
        val cache = OfflineEntityCache()
        val mdnsEngine = MdnsDiscoveryEngine(customScanner = mockScanner)
        val transport = DualTransportManager()
        val apiClient = DomusApiClient()

        val connectionRepo = DefaultConnectionRepository(mdnsEngine, transport, tokenStorage)
        val authRepo = DefaultAuthRepository(apiClient, tokenStorage, connectionRepo)
        val entityRepo = DefaultEntityRepository(apiClient, transport, cache, tokenStorage)

        val discoverUseCase = DiscoverAndConnectUseCase(connectionRepo)
        val authUseCase = AuthenticateUseCase(authRepo)
        val syncUseCase = SyncEntitiesUseCase(entityRepo)
        val toggleUseCase = ToggleEntityUseCase(entityRepo)

        // Step 1: Discover and Connect LAN
        val connResult = discoverUseCase(preferLocal = true)
        assertTrue(connResult.isSuccess)
        val conn = connResult.getOrNull()
        assertNotNull(conn)
        assertTrue("La connessione dovrebbe essere indirizzata a LAN", conn?.isLocal == true)

        // Step 2: Login and get JWT token
        val authResult = authUseCase.login("admin", "supersecret")
        assertTrue(authResult.isSuccess)
        assertEquals("jwt_mobile_secret_token", authResult.getOrNull()?.token)
        assertEquals("jwt_mobile_secret_token", tokenStorage.getToken())

        // Step 3: Fetch initial entity snapshot
        val syncResult = syncUseCase()
        assertTrue(syncResult.isSuccess)
        val initialEntities = syncResult.getOrNull() ?: emptyList()
        assertEquals(1, initialEntities.size)
        assertEquals("light.kitchen", initialEntities[0].id)

        // Step 4: Verify WebSocket real-time update arrived into cache
        val deadline = System.currentTimeMillis() + 4000L
        while (System.currentTimeMillis() < deadline && cache.getEntity("light.kitchen")?.state != "on") {
            delay(50L)
        }
        val updated = cache.getEntity("light.kitchen")
        assertNotNull("L'entità deve essere aggiornata dal WebSocket", updated)
        assertEquals("on", updated?.state)

        // Step 5: Optimistic toggle
        val toggleResult = toggleUseCase("light.kitchen")
        assertTrue(toggleResult.isSuccess)
        assertEquals("off", toggleResult.getOrNull()?.state)

        transport.disconnect()
    }
}
