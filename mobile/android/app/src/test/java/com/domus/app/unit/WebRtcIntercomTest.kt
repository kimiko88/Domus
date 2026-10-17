package com.domus.app.unit

import com.domus.app.services.webrtc.DomusWebRtcIntercomManager
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test

class WebRtcIntercomTest {

    private lateinit var intercomManager: DomusWebRtcIntercomManager

    @Before
    fun setUp() {
        intercomManager = DomusWebRtcIntercomManager(context = null)
    }

    @Test
    fun testLiveStreamConnectionAndSub150msLatency() {
        assertFalse(intercomManager.isConnected.value)

        intercomManager.startLiveStream("camera.front_door")
        assertTrue(intercomManager.isConnected.value)
        assertEquals("camera.front_door", intercomManager.activeCameraId.value)

        // Latency must be strictly below 150ms
        assertTrue(intercomManager.latencyMs.value < 150)
        assertEquals(30, intercomManager.currentFps.value)

        intercomManager.stopLiveStream()
        assertFalse(intercomManager.isConnected.value)
        assertNull(intercomManager.activeCameraId.value)
    }

    @Test
    fun testPushToTalkStateTransitions() {
        assertFalse(intercomManager.isPushToTalkActive.value)

        intercomManager.startPushToTalk("camera.intercom_gate")
        assertTrue(intercomManager.isPushToTalkActive.value)

        intercomManager.stopPushToTalk()
        assertFalse(intercomManager.isPushToTalkActive.value)
    }
}
