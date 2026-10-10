package com.domus.app.integration

import com.domus.app.data.network.DomusApiClient
import com.domus.app.data.network.DualTransportManager
import com.domus.app.data.repository.DefaultEntityRepository
import com.domus.app.data.storage.InMemoryTokenStorage
import com.domus.app.data.storage.OfflineEntityCache
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class EntityRepositoryIntegrationTest {

    private lateinit var server: MockWebServer

    @Before
    fun setup() {
        server = MockWebServer()
        server.start()
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun testSyncEntitiesAndToggleEntity() = runTest {
        val entitiesJson = """
            [
                {
                    "id": "switch.garden_lights",
                    "name": "Luci Giardino",
                    "domain": "switch",
                    "state": "off",
                    "areaId": "garden"
                }
            ]
        """.trimIndent()

        // 1. Response for GET /api/v1/entities
        server.enqueue(MockResponse().setResponseCode(200).setBody(entitiesJson))
        // 2. Response for GET /api/v1/areas
        server.enqueue(MockResponse().setResponseCode(200).setBody("[]"))
        // 3. Response for POST /api/v1/services/switch/turn_on
        server.enqueue(MockResponse().setResponseCode(200).setBody("""{"status":"ok"}"""))

        val tokenStorage = InMemoryTokenStorage(localEndpoint = server.url("").toString().removeSuffix("/"))
        val cache = OfflineEntityCache()
        val apiClient = DomusApiClient()
        val transport = DualTransportManager()

        val repository = DefaultEntityRepository(
            apiClient = apiClient,
            transportManager = transport,
            entityCache = cache,
            tokenStorage = tokenStorage
        )

        // Sync entities
        val syncResult = repository.syncEntities()
        assertTrue(syncResult.isSuccess)
        assertEquals(1, syncResult.getOrNull()?.size)

        // Check cached flow
        val flowEntities = repository.entitiesFlow.first()
        assertEquals(1, flowEntities.size)
        assertEquals("switch.garden_lights", flowEntities[0].id)
        assertEquals("off", flowEntities[0].state)

        // Toggle entity
        val toggleResult = repository.toggleEntity("switch.garden_lights")
        assertTrue(toggleResult.isSuccess)
        assertEquals("on", toggleResult.getOrNull()?.state)
    }
}
