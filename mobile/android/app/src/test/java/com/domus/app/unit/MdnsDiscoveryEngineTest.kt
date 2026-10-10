package com.domus.app.unit

import com.domus.app.data.mdns.DiscoveredHub
import com.domus.app.data.mdns.MdnsDiscoveryEngine
import com.domus.app.data.mdns.MdnsScanner
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Test

class MdnsDiscoveryEngineTest {

    @Test
    fun testDiscoverLocalHubSuccess() = runTest {
        val mockScanner = object : MdnsScanner {
            override suspend fun discoverService(serviceType: String, timeoutMs: Long): DiscoveredHub {
                return DiscoveredHub(
                    name = "Domus Hub Test",
                    host = "192.168.1.150",
                    port = 8080
                )
            }
        }

        val engine = MdnsDiscoveryEngine(customScanner = mockScanner)
        val hub = engine.discoverLocalHub(1000L)

        assertNotNull(hub)
        assertEquals("Domus Hub Test", hub?.name)
        assertEquals("http://192.168.1.150:8080", hub?.httpBaseUrl)
        assertEquals("ws://192.168.1.150:8080/api/v1/ws", hub?.wsBaseUrl)
    }

    @Test
    fun testDiscoverLocalHubTimeoutOrNotFound() = runTest {
        val emptyScanner = object : MdnsScanner {
            override suspend fun discoverService(serviceType: String, timeoutMs: Long): DiscoveredHub? = null
        }

        val engine = MdnsDiscoveryEngine(customScanner = emptyScanner)
        val hub = engine.discoverLocalHub(500L)

        assertNull(hub)
    }
}
