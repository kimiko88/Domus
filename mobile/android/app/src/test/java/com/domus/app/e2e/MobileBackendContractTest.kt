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
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
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
import org.junit.Assert.assertFalse
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
        } catch (e: Exception) {}
    }


    @Test
    fun apiClientSafelyEncodesUserStringsAndReportsHttpFailures() = runTest {
        server.start()
        val baseUrl = server.url("/").toString().removeSuffix("/")
        val apiClient = DomusApiClient()
        val injectedValue = """device", "admin": true, "extra":"owned"""

        server.enqueue(MockResponse().setResponseCode(200))
        val scanResult = apiClient.scanTag(baseUrl, "test-token", "tag-1", injectedValue)
        assertTrue(scanResult.isSuccess)
        val scanRequest = server.takeRequest()
        val scanJson = kotlinx.serialization.json.Json.parseToJsonElement(scanRequest.body.readUtf8()).jsonObject
        assertEquals(setOf("scanned_by"), scanJson.keys)
        assertEquals(injectedValue, scanJson["scanned_by"]?.jsonPrimitive?.content)

        server.enqueue(MockResponse().setResponseCode(500))
        assertFalse(apiClient.scanTag(baseUrl, "test-token", "tag-1").isSuccess)

        server.enqueue(MockResponse().setResponseCode(500))
        assertFalse(apiClient.registerPushToken(baseUrl, "test-token", "push-token", "test-device").isSuccess)
    }

    @Test
    fun apiClientPropagatesReadFailuresAndUsesNotificationFallbackOnlyForMissingRoute() = runTest {
        server.start()
        val baseUrl = server.url("/").toString().removeSuffix("/")
        val apiClient = DomusApiClient()

        server.enqueue(MockResponse().setResponseCode(503))
        assertFalse(apiClient.fetchAreas(baseUrl, "token").isSuccess)
        server.takeRequest()

        server.enqueue(MockResponse().setResponseCode(500))
        assertFalse(apiClient.fetchNotifications(baseUrl, "token").isSuccess)
        server.takeRequest()

        server.enqueue(MockResponse().setResponseCode(404))
        server.enqueue(MockResponse().setResponseCode(200).setBody("[]"))
        val fallbackResult = apiClient.fetchNotifications(baseUrl, "token")
        assertTrue(fallbackResult.isSuccess)
        assertEquals(4, server.requestCount)
        assertEquals("/api/v1/notifications", server.takeRequest().path)
        assertEquals("/api/v1/notifications/actionable", server.takeRequest().path)

        server.enqueue(MockResponse().setResponseCode(503))
        assertFalse(apiClient.fetchSystemHealth(baseUrl, "token").isSuccess)
        server.takeRequest()

        server.enqueue(MockResponse().setResponseCode(403))
        assertFalse(apiClient.fetchFamilyMembers(baseUrl, "token").isSuccess)
        server.takeRequest()

        server.enqueue(MockResponse().setResponseCode(500))
        assertFalse(apiClient.fetchFamilyInvites(baseUrl, "token").isSuccess)
        server.takeRequest()
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

        var activeWs: WebSocket? = null

        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                val path = request.path ?: ""
                return when {
                    path == "/ws" -> {
                        MockResponse().withWebSocketUpgrade(object : WebSocketListener() {
                            override fun onOpen(webSocket: WebSocket, response: Response) {
                                activeWs = webSocket
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
                    path.startsWith("/api/v1/finance/summary") -> {
                        MockResponse().setResponseCode(200).setBody("""
                            {"period":"month","total_spent":450.0,"previous_total":500.0,"percentage_change":-10.0,"monthly_budget":1000.0,"remaining_budget":550.0,"daily_average":15.0,"categories":[],"fixed_vs_variable":{"fixed_total":200.0,"variable_total":250.0,"fixed_percentage":44.4,"variable_percentage":55.6},"monthly_trend":[],"insights":[]}
                        """.trimIndent())
                    }
                    path.startsWith("/api/v1/finance/expenses") -> {
                        MockResponse().setResponseCode(200).setBody("""
                            [{"id":"exp-1","title":"Spesa Supermercato","amount":75.5,"category":"food","subcategory":"groceries","expense_date":"2026-10-17T10:00:00Z","is_fixed":false}]
                        """.trimIndent())
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
        // Wait for active WebSocket connection, then send live event
        val deadlineWs = System.currentTimeMillis() + 3000L
        while (System.currentTimeMillis() < deadlineWs && activeWs == null) {
            delay(50L)
        }
        assertNotNull("WebSocket should be connected", activeWs)
        activeWs?.send(wsEvent)

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

        // Step 6: E2E Home Finance summary & expenses integration
        val financeSummary = apiClient.fetchFinancialSummary(baseUrl, tokenStorage.getToken() ?: "", "month")
        assertTrue(financeSummary.isSuccess)
        assertEquals(450.0, financeSummary.getOrNull()?.total_spent ?: 0.0, 0.001)
        assertEquals(550.0, financeSummary.getOrNull()?.remaining_budget ?: 0.0, 0.001)

        val expenses = apiClient.fetchExpenses(baseUrl, tokenStorage.getToken() ?: "")
        assertTrue(expenses.isSuccess)
        assertEquals(1, expenses.getOrNull()?.size)
        assertEquals("Spesa Supermercato", expenses.getOrNull()?.get(0)?.title)

        transport.disconnect()
    }
}
