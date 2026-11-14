package com.domus.app.data.repository

import com.domus.app.data.network.DomusApiClient
import com.domus.app.data.network.DualTransportManager
import com.domus.app.data.storage.OfflineEntityCache
import com.domus.app.data.storage.TokenStorage
import com.domus.app.domain.model.Area
import com.domus.app.domain.model.Entity
import com.domus.app.domain.repository.EntityRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.launch

class DefaultEntityRepository(
    private val apiClient: DomusApiClient,
    private val transportManager: DualTransportManager,
    private val entityCache: OfflineEntityCache,
    private val tokenStorage: TokenStorage,
    private val coroutineScope: CoroutineScope = CoroutineScope(Dispatchers.IO)
) : EntityRepository {

    override val entitiesFlow: Flow<List<Entity>> = entityCache.entitiesFlow
    override val areasFlow: Flow<List<Area>> = entityCache.areasFlow

    init {
        // Listen to WebSocket entity changes in real-time
        coroutineScope.launch {
            transportManager.eventFlow.collect { event ->
                val existing = entityCache.getEntity(event.entityId)
                val updated = if (existing != null) {
                    existing.copy(
                        state = event.newState.state,
                        attributes = event.newState.attributes,
                        lastChanged = event.newState.lastChanged,
                        lastUpdated = event.newState.lastUpdated
                    )
                } else {
                    Entity(
                        id = event.entityId,
                        name = event.entityId.substringAfter("."),
                        domain = event.entityId.substringBefore("."),
                        state = event.newState.state,
                        attributes = event.newState.attributes,
                        lastChanged = event.newState.lastChanged,
                        lastUpdated = event.newState.lastUpdated
                    )
                }
                entityCache.upsertEntity(updated)
            }
        }
    }

    override suspend fun getEntity(id: String): Entity? = entityCache.getEntity(id)

    override suspend fun getAllEntities(): List<Entity> = entityCache.getAllEntities()

    override suspend fun syncEntities(): Result<List<Entity>> {
        val baseUrl = transportManager.connectionFlow.value.activeUrl
            ?: tokenStorage.getLocalEndpoint()
            ?: "http://192.168.1.100:8080"
        val token = tokenStorage.getToken()

        val entitiesRes = apiClient.fetchEntities(baseUrl, token)
        if (entitiesRes.isSuccess) {
            val entities = entitiesRes.getOrNull() ?: emptyList()
            entityCache.updateEntities(entities)

            // Also fetch areas if available
            val areasRes = apiClient.fetchAreas(baseUrl, token)
            if (areasRes.isSuccess) {
                entityCache.updateAreas(areasRes.getOrNull() ?: emptyList())
            }

            return Result.success(entities)
        }

        // If network fetch fails, return cached entities for offline-first resilience
        val cached = entityCache.getAllEntities()
        return if (cached.isNotEmpty()) {
            Result.success(cached)
        } else {
            Result.failure(entitiesRes.exceptionOrNull() ?: Exception("Impossibile caricare le entità"))
        }
    }

    override suspend fun toggleEntity(entityId: String): Result<Entity> {
        val entity = entityCache.getEntity(entityId)
            ?: return Result.failure(Exception("Entità non trovata"))

        val domain = entity.domain
        val service = if (entity.isOn) "turn_off" else "turn_on"
        val newState = if (entity.isOn) "off" else "on"

        // Optimistic UI update
        val optimisticEntity = entity.copy(state = newState)
        entityCache.upsertEntity(optimisticEntity)

        // Try WebSocket first
        if (transportManager.connectionFlow.value.isConnected &&
            transportManager.sendCommand(domain, service, entityId)
        ) {
            return Result.success(optimisticEntity)
        }

        // Fallback to HTTP REST
        val baseUrl = transportManager.connectionFlow.value.activeUrl
            ?: tokenStorage.getLocalEndpoint()
            ?: "http://192.168.1.100:8080"
        val token = tokenStorage.getToken()

        val callRes = apiClient.callService(baseUrl, token, domain, service, entityId)
        return if (callRes.isSuccess) {
            Result.success(optimisticEntity)
        } else {
            // Revert optimistic update on failure
            entityCache.upsertEntity(entity)
            Result.failure(callRes.exceptionOrNull() ?: Exception("Comando fallito"))
        }
    }

    override suspend fun callService(domain: String, service: String, entityId: String, data: Map<String, Any>): Result<Unit> {
        if (transportManager.connectionFlow.value.isConnected &&
            transportManager.sendCommand(domain, service, entityId, data)
        ) {
            return Result.success(Unit)
        }

        val baseUrl = transportManager.connectionFlow.value.activeUrl
            ?: tokenStorage.getLocalEndpoint()
            ?: "http://192.168.1.100:8080"
        val token = tokenStorage.getToken()

        return apiClient.callService(baseUrl, token, domain, service, entityId, data)
    }

    override suspend fun updateLocalEntity(entity: Entity) {
        entityCache.upsertEntity(entity)
    }
}
