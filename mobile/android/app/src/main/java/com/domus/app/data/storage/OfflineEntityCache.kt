package com.domus.app.data.storage

import com.domus.app.domain.model.Area
import com.domus.app.domain.model.Entity
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import java.util.concurrent.ConcurrentHashMap

class OfflineEntityCache {
    private val entityMap = ConcurrentHashMap<String, Entity>()
    private val areaMap = ConcurrentHashMap<String, Area>()

    private val _entitiesFlow = MutableStateFlow<List<Entity>>(emptyList())
    val entitiesFlow: StateFlow<List<Entity>> = _entitiesFlow.asStateFlow()

    private val _areasFlow = MutableStateFlow<List<Area>>(emptyList())
    val areasFlow: StateFlow<List<Area>> = _areasFlow.asStateFlow()

    fun updateEntities(entities: List<Entity>) {
        entityMap.clear()
        entities.forEach { entityMap[it.id] = it }
        publishEntities()
    }

    fun upsertEntity(entity: Entity) {
        entityMap[entity.id] = entity
        publishEntities()
    }

    fun getEntity(id: String): Entity? = entityMap[id]

    fun getAllEntities(): List<Entity> = entityMap.values.toList().sortedBy { it.name }

    fun updateAreas(areas: List<Area>) {
        areaMap.clear()
        areas.forEach { areaMap[it.id] = it }
        publishAreas()
    }

    fun getAllAreas(): List<Area> = areaMap.values.toList().sortedBy { it.name }

    fun clear() {
        entityMap.clear()
        areaMap.clear()
        publishEntities()
        publishAreas()
    }

    private fun publishEntities() {
        _entitiesFlow.value = entityMap.values.toList().sortedBy { it.name }
    }

    private fun publishAreas() {
        _areasFlow.value = areaMap.values.toList().sortedBy { it.name }
    }
}
