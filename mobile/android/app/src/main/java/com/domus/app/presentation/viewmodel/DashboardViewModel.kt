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

import com.domus.app.data.network.*
import com.domus.app.data.storage.TokenStorage

class DashboardViewModel(
    private val entityRepository: EntityRepository,
    private val connectionRepository: ConnectionRepository,
    private val discoverAndConnectUseCase: DiscoverAndConnectUseCase,
    private val syncEntitiesUseCase: SyncEntitiesUseCase,
    private val toggleEntityUseCase: ToggleEntityUseCase,
    val apiClient: DomusApiClient = DomusApiClient(),
    val tokenStorage: TokenStorage? = null
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

    fun callService(domain: String, service: String, entityId: String, data: Map<String, Any> = emptyMap()) {
        viewModelScope.launch {
            entityRepository.callService(domain, service, entityId, data)
        }
    }

    private fun getBaseUrl(): String {
        return connection.value.activeUrl ?: tokenStorage?.getLocalEndpoint() ?: "http://192.168.1.100:8080"
    }

    private fun getToken(): String? {
        return tokenStorage?.getToken()
    }

    private val _chores = MutableStateFlow<List<ChoreDto>>(emptyList())
    val chores: StateFlow<List<ChoreDto>> = _chores.asStateFlow()

    fun loadChores() {
        viewModelScope.launch {
            val res = apiClient.fetchChores(getBaseUrl(), getToken())
            if (res.isSuccess) {
                _chores.value = res.getOrDefault(emptyList())
            }
        }
    }

    fun completeChore(choreId: String) {
        viewModelScope.launch {
            val res = apiClient.completeChore(getBaseUrl(), getToken(), choreId)
            if (res.isSuccess) {
                loadChores()
            }
        }
    }

    fun createChore(title: String, area: String = "", user: String = "", priority: String = "medium") {
        viewModelScope.launch {
            val req = CreateChoreRequestDto(title = title, area_id = area, assigned_user_id = user, priority = priority)
            val res = apiClient.createChore(getBaseUrl(), getToken(), req)
            if (res.isSuccess) {
                loadChores()
            }
        }
    }

    private val _shoppingItems = MutableStateFlow<List<ShoppingItemDto>>(emptyList())
    val shoppingItems: StateFlow<List<ShoppingItemDto>> = _shoppingItems.asStateFlow()

    fun loadShopping() {
        viewModelScope.launch {
            val res = apiClient.fetchShopping(getBaseUrl(), getToken())
            if (res.isSuccess) {
                _shoppingItems.value = res.getOrDefault(emptyList())
            }
        }
    }

    fun toggleShopping(itemId: String) {
        viewModelScope.launch {
            val res = apiClient.toggleShopping(getBaseUrl(), getToken(), itemId)
            if (res.isSuccess) {
                loadShopping()
            }
        }
    }

    fun createShopping(name: String, category: String = "Dispensa", quantity: String = "1") {
        viewModelScope.launch {
            val req = CreateShoppingItemDto(name = name, category = category, quantity = quantity)
            val res = apiClient.createShopping(getBaseUrl(), getToken(), req)
            if (res.isSuccess) {
                loadShopping()
            }
        }
    }

    fun deleteShopping(itemId: String) {
        viewModelScope.launch {
            val res = apiClient.deleteShopping(getBaseUrl(), getToken(), itemId)
            if (res.isSuccess) {
                loadShopping()
            }
        }
    }

    fun clearCompletedShopping() {
        viewModelScope.launch {
            val res = apiClient.clearCompletedShopping(getBaseUrl(), getToken())
            if (res.isSuccess) {
                loadShopping()
            }
        }
    }

    private val _energyFlow = MutableStateFlow<EnergyFlowDto?>(null)
    val energyFlow: StateFlow<EnergyFlowDto?> = _energyFlow.asStateFlow()

    fun loadEnergyFlow() {
        viewModelScope.launch {
            val res = apiClient.fetchEnergyFlow(getBaseUrl(), getToken())
            if (res.isSuccess) {
                _energyFlow.value = res.getOrNull()
            }
        }
    }

    private val _notifications = MutableStateFlow<List<NotificationDto>>(emptyList())
    val notifications: StateFlow<List<NotificationDto>> = _notifications.asStateFlow()

    fun loadNotifications() {
        viewModelScope.launch {
            val res = apiClient.fetchNotifications(getBaseUrl(), getToken())
            if (res.isSuccess) {
                _notifications.value = res.getOrDefault(emptyList())
            }
        }
    }

    suspend fun sendAssistantPrompt(prompt: String): AssistantChatResponseDto? {
        val res = apiClient.sendAssistantPrompt(getBaseUrl(), getToken(), prompt)
        return res.getOrNull()
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
