package com.domus.app.unit

import com.domus.app.services.wear.DomusWearTileService
import org.junit.Assert.*
import org.junit.Test

class WearOsTilesTest {

    @Test
    fun testWearQuickScenesConfigured() {
        val scenes = DomusWearTileService.quickScenes
        assertEquals(3, scenes.size)

        val leave = scenes.find { it.name == "Esco" }
        assertNotNull(leave)
        assertEquals("scene.leave_home", leave?.entityId)

        val night = scenes.find { it.name == "Buonanotte" }
        assertNotNull(night)
        assertEquals("scene.goodnight", night?.entityId)

        val cinema = scenes.find { it.name == "Cinema" }
        assertNotNull(cinema)
        assertEquals("scene.cinema", cinema?.entityId)
    }

    @Test
    fun testComplicationFormatting() {
        val service = DomusWearTileService.shared
        service.updateComplication(temp = 22.4, lights = 4, alarmArmed = true, mode = "Armato Notte")

        val complicationText = service.getWatchFaceComplicationText()
        assertTrue(complicationText.contains("22.4°C"))
        assertTrue(complicationText.contains("4 Luci"))

        val alarmText = service.getAlarmTileText()
        assertTrue(alarmText.contains("Armato Notte"))
    }
}
