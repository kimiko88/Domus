package com.domus.app.domain.repository

import com.domus.app.domain.model.Area
import com.domus.app.domain.model.Entity
import kotlinx.coroutines.flow.Flow

interface EntityRepository {
    val entitiesFlow: Flow<List<Entity>>
    val areasFlow: Flow<List<Area>>

    suspend fun getEntity(id: String): Entity?
    suspend fun getAllEntities(): List<Entity>
    suspend fun syncEntities(): Result<List<Entity>>
    suspend fun toggleEntity(entityId: String): Result<Entity>
    suspend fun callService(domain: String, service: String, entityId: String, data: Map<String, Any> = emptyMap()): Result<Unit>
    suspend fun updateLocalEntity(entity: Entity)
}
