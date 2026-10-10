package com.domus.app.unit

import com.domus.app.data.storage.OfflineEntityCache
import com.domus.app.domain.model.Area
import com.domus.app.domain.model.Entity
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class OfflineEntityCacheTest {

    @Test
    fun testUpdateAndRetrieveEntities() = runTest {
        val cache = OfflineEntityCache()

        val entities = listOf(
            Entity(id = "light.living_room", name = "Luce Salotto", domain = "light", state = "off"),
            Entity(id = "switch.coffee_maker", name = "Macchina Caffè", domain = "switch", state = "on")
        )

        cache.updateEntities(entities)

        assertEquals(2, cache.getAllEntities().size)
        val light = cache.getEntity("light.living_room")
        assertNotNull(light)
        assertEquals("off", light?.state)

        // Upsert state change
        cache.upsertEntity(light!!.copy(state = "on"))
        val updated = cache.getEntity("light.living_room")
        assertEquals("on", updated?.state)
        assertTrue(updated?.isOn == true)

        // StateFlow emits update
        val flowValue = cache.entitiesFlow.first()
        assertEquals(2, flowValue.size)
    }

    @Test
    fun testAreaManagement() = runTest {
        val cache = OfflineEntityCache()
        val areas = listOf(
            Area(id = "living_room", name = "Salotto"),
            Area(id = "kitchen", name = "Cucina")
        )

        cache.updateAreas(areas)
        assertEquals(2, cache.getAllAreas().size)
    }
}
