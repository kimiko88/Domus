package com.domus.app.integration

import com.domus.app.data.network.DualTransportManager
import com.domus.app.domain.model.TransportMode
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
import org.junit.Before
import org.junit.Test

class WebSocketProtocolIntegrationTest {

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
    fun testWebSocketConnectAndReceiveStateChangedEvent() = runTest {
        val eventPayload = """
            {
                "type": "event",
                "event": {
                    "entity_id": "light.bedroom",
                    "new_state": {
                        "entity_id": "light.bedroom",
                        "state": "on",
                        "attributes": {"brightness": 85},
                        "last_changed": "2026-10-10T20:00:00Z"
                    }
                }
            }
        """.trimIndent()

        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                return if (request.path?.startsWith("/api/v1/ws") == true) {
                    MockResponse().withWebSocketUpgrade(object : WebSocketListener() {
                        override fun onOpen(webSocket: WebSocket, response: Response) {
                            webSocket.send(eventPayload)
                        }
                        override fun onMessage(webSocket: WebSocket, text: String) {}
                        override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                            webSocket.close(code, reason)
                        }
                    })
                } else {
                    MockResponse().setResponseCode(404)
                }
            }
        }
        server.start()

        val transport = DualTransportManager()
        val serverUrl = server.url("/api/v1/ws").toString()

        transport.connect(TransportMode.LOCAL_LAN, serverUrl, token = "test_jwt_token")

        var event: com.domus.app.data.network.EntityStateChangedEvent? = null
        val deadline = System.currentTimeMillis() + 4000L
        while (System.currentTimeMillis() < deadline && event == null) {
            event = transport.eventFlow.replayCache.firstOrNull()
            if (event == null) {
                delay(50L)
            }
        }

        assertNotNull("Event should be received via WebSocket", event)
        assertEquals("light.bedroom", event?.entityId)
        assertEquals("on", event?.newState?.state)

        transport.disconnect()
    }
}
