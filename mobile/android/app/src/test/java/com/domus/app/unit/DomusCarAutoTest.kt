package com.domus.app.unit

import com.domus.app.services.auto.DomusCarActions
import org.junit.Assert.*
import org.junit.Test

class DomusCarAutoTest {

    @Test
    fun testQuickActionsConfigured() {
        val actions = DomusCarActions.quickActions
        assertEquals(4, actions.size)

        val gate = actions.find { it.id == "car_gate" }
        assertNotNull(gate)
        assertEquals("cover", gate?.domain)
        assertEquals("cover.gate", gate?.entityId)

        val garage = actions.find { it.id == "car_garage" }
        assertNotNull(garage)
        assertEquals("cover", garage?.domain)
        assertEquals("cover.garage", garage?.entityId)

        val disarm = actions.find { it.id == "car_disarm" }
        assertNotNull(disarm)
        assertEquals("alarm_control_panel", disarm?.domain)
    }

    @Test
    fun testGeofenceStateToggle() {
        DomusCarActions.isNearHomeGeofence = true
        assertTrue(DomusCarActions.isNearHomeGeofence)

        DomusCarActions.isNearHomeGeofence = false
        assertFalse(DomusCarActions.isNearHomeGeofence)
    }
}
