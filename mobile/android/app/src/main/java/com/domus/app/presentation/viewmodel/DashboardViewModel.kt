package com.domus.app.presentation.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.domus.app.domain.model.Area
import com.domus.app.domain.model.Entity
import com.domus.app.domain.model.ServerConnection
import com.domus.app.domain.repository.ConnectionRepository
import com.domus.app.domain.usecase.DiscoverAndConnectUseCase
import com.domus.app.domain.usecase.SyncEntitiesUseCase
import com.domus.app.domain.usecase.ToggleEntityUseCase
import com.domus.app.domain.repository.EntityRepository
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch

class DashboardViewModel(
    private val entityRepository: EntityRepository,
    private val connectionRepository: ConnectionRepository,
    private val discoverAndConnectUseCase: DiscoverAndConnectUseCase,
    private val syncEntitiesUseCase: SyncEntitiesUseCase,
    private val toggleEntityUseCase: ToggleEntityUseCase
) : ViewModel() {

    val connection: StateFlow<ServerConnection> = connectionRepository.connectionFlow
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5000), ServerConnection())

    val areas: StateFlow<List<Area>> = entityRepository.areasFlow
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5000), emptyList())

    private val _selectedAreaId = MutableStateFlow<String?>(null)
    val selectedAreaId: StateFlow<String?> = _selectedAreaId.asStateFlow()

    private val _searchQuery = MutableStateFlow("")
    val searchQuery: StateFlow<String> = _searchQuery.asStateFlow()

    private val _isRefreshing = MutableStateFlow(false)
    val isRefreshing: StateFlow<Boolean> = _isRefreshing.asStateFlow()

    val filteredEntities: StateFlow<List<Entity>> = combine(
        entityRepository.entitiesFlow,
        _selectedAreaId,
        _searchQuery
    ) { entities, areaId, query ->
        entities.filter { entity ->
            val matchesArea = areaId == null || entity.areaId == areaId
            val matchesQuery = query.isBlank() ||
                    entity.name.contains(query, ignoreCase = true) ||
                    entity.domain.contains(query, ignoreCase = true)
            matchesArea && matchesQuery
        }
    }.stateIn(viewModelScope, SharingStarted.WhileSubscribed(5000), emptyList())

    init {
        initialConnectAndSync()
    }

    fun initialConnectAndSync() {
        viewModelScope.launch {
            _isRefreshing.value = true
            discoverAndConnectUseCase(preferLocal = true)
            syncEntitiesUseCase()
            _isRefreshing.value = false
        }
    }

    fun selectArea(areaId: String?) {
        _selectedAreaId.value = areaId
    }

    fun setSearchQuery(query: String) {
        _searchQuery.value = query
    }

    fun toggleEntity(entityId: String) {
        viewModelScope.launch {
            toggleEntityUseCase(entityId)
        }
    }

    fun reconnect() {
        viewModelScope.launch {
            _isRefreshing.value = true
            discoverAndConnectUseCase(preferLocal = true)
            syncEntitiesUseCase()
            _isRefreshing.value = false
        }
    }
}
