package com.domus.app.unit

import com.domus.app.domain.model.Area
import com.domus.app.domain.model.Entity
import com.domus.app.domain.repository.EntityRepository
import com.domus.app.domain.usecase.ToggleEntityUseCase
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ToggleEntityUseCaseTest {

    @Test
    fun testToggleEntityDispatchesToRepo() = runTest {
        var toggledId: String? = null

        val fakeEntityRepo = object : EntityRepository {
            override val entitiesFlow: Flow<List<Entity>> = flowOf(emptyList())
            override val areasFlow: Flow<List<Area>> = flowOf(emptyList())
            override suspend fun getEntity(id: String): Entity? = null
            override suspend fun getAllEntities(): List<Entity> = emptyList()
            override suspend fun syncEntities(): Result<List<Entity>> = Result.success(emptyList())
            override suspend fun toggleEntity(entityId: String): Result<Entity> {
                toggledId = entityId
                return Result.success(Entity(id = entityId, name = "Luce", domain = "light", state = "on"))
            }
            override suspend fun callService(domain: String, service: String, entityId: String, data: Map<String, Any>): Result<Unit> = Result.success(Unit)
            override suspend fun updateLocalEntity(entity: Entity) {}
        }

        val useCase = ToggleEntityUseCase(fakeEntityRepo)
        val result = useCase("light.kitchen")

        assertTrue(result.isSuccess)
        assertEquals("light.kitchen", toggledId)
        assertEquals("on", result.getOrNull()?.state)
    }
}
