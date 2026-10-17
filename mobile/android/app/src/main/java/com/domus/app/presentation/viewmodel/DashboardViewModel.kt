package com.domus.app.presentation.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.domus.app.data.network.*
import com.domus.app.data.storage.TokenStorage
import com.domus.app.domain.model.*
import com.domus.app.domain.repository.ConnectionRepository
import com.domus.app.domain.repository.EntityRepository
import com.domus.app.domain.usecase.DiscoverAndConnectUseCase
import com.domus.app.domain.usecase.SyncEntitiesUseCase
import com.domus.app.domain.usecase.ToggleEntityUseCase
import com.domus.app.services.kiosk.DomusKioskController
import com.domus.app.services.location.RoomPresenceBeaconManager
import com.domus.app.services.nfc.NfcTagManager
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch

class DashboardViewModel(
    private val entityRepository: EntityRepository,
    private val connectionRepository: ConnectionRepository,
    private val discoverAndConnectUseCase: DiscoverAndConnectUseCase,
    private val syncEntitiesUseCase: SyncEntitiesUseCase,
    private val toggleEntityUseCase: ToggleEntityUseCase,
    val apiClient: DomusApiClient = DomusApiClient(),
    val tokenStorage: TokenStorage? = null,
    val beaconManager: RoomPresenceBeaconManager = RoomPresenceBeaconManager.shared,
    val nfcManager: NfcTagManager = NfcTagManager.shared,
    var kioskController: DomusKioskController? = null
) : ViewModel() {

    val connection: StateFlow<ServerConnection> = connectionRepository.connectionFlow
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5000), ServerConnection())

    val areas: StateFlow<List<Area>> = entityRepository.areasFlow
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5000), emptyList())

    val allEntities: StateFlow<List<Entity>> = entityRepository.entitiesFlow
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

    // Auth state
    private val _authSession = MutableStateFlow(
        AuthSession(
            token = tokenStorage?.getToken(),
            username = "admin",
            role = "admin",
            isAuthenticated = tokenStorage?.getToken()?.isNotBlank() == true
        )
    )
    val authSession: StateFlow<AuthSession> = _authSession.asStateFlow()

    // Automations state
    private val _automations = MutableStateFlow<List<AutomationRule>>(AutomationRule.sampleRules)
    val automations: StateFlow<List<AutomationRule>> = _automations.asStateFlow()

    // Timeline / Logbook state
    private val _timelineEvents = MutableStateFlow<List<TimelineEvent>>(TimelineEvent.sampleTimeline)
    val timelineEvents: StateFlow<List<TimelineEvent>> = _timelineEvents.asStateFlow()

    // Mesh & Diagnostics state
    private val _meshNodes = MutableStateFlow<List<MeshNodeDiagnostic>>(MeshNodeDiagnostic.sampleNodes)
    val meshNodes: StateFlow<List<MeshNodeDiagnostic>> = _meshNodes.asStateFlow()

    private val _systemHealth = MutableStateFlow<SystemHealthDto?>(SystemHealthDto())
    val systemHealth: StateFlow<SystemHealthDto?> = _systemHealth.asStateFlow()

    private val _lanLatencyMs = MutableStateFlow(12)
    val lanLatencyMs: StateFlow<Int> = _lanLatencyMs.asStateFlow()

    // Family and Guests
    private val _familyAccounts = MutableStateFlow<List<DomusUserAccount>>(DomusUserAccount.sampleAccounts)
    val familyAccounts: StateFlow<List<DomusUserAccount>> = _familyAccounts.asStateFlow()

    private val _familyInvites = MutableStateFlow<List<FamilyInviteDto>>(emptyList())
    val familyInvites: StateFlow<List<FamilyInviteDto>> = _familyInvites.asStateFlow()

    // Kiosk state observables
    private val _isScreensaverActive = MutableStateFlow(false)
    val isScreensaverActive: StateFlow<Boolean> = _isScreensaverActive.asStateFlow()

    private val _isKioskModeEnabled = MutableStateFlow(false)
    val isKioskModeEnabled: StateFlow<Boolean> = _isKioskModeEnabled.asStateFlow()

    private val _isBatteryProtectionEnabled = MutableStateFlow(true)
    val isBatteryProtectionEnabled: StateFlow<Boolean> = _isBatteryProtectionEnabled.asStateFlow()

    private val _kioskSmartPlugId = MutableStateFlow("switch.tablet_charger")
    val kioskSmartPlugId: StateFlow<String> = _kioskSmartPlugId.asStateFlow()

    init {
        initialConnectAndSync()
        beaconManager.startMonitoring()
        setupKioskListeners()
    }

    private fun setupKioskListeners() {
        kioskController?.let { controller ->
            viewModelScope.launch {
                controller.isScreensaverActive.collect { _isScreensaverActive.value = it }
            }
            viewModelScope.launch {
                controller.isKioskModeEnabled.collect { _isKioskModeEnabled.value = it }
            }
            controller.onBatteryProtectionAction = { entityId, turnOn ->
                callService("switch", if (turnOn) "turn_on" else "turn_off", entityId)
            }
        }
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

    // --- AUTOMATIONS ---
    fun loadAutomations() {
        viewModelScope.launch {
            val res = apiClient.fetchAutomations(getBaseUrl(), getToken())
            if (res.isSuccess) {
                _automations.value = res.getOrDefault(AutomationRule.sampleRules)
            }
        }
    }

    fun createAutomation(rule: AutomationRule) {
        viewModelScope.launch {
            val res = apiClient.createAutomation(getBaseUrl(), getToken(), rule)
            if (res.isSuccess) {
                loadAutomations()
            } else {
                _automations.value = _automations.value + rule
            }
        }
    }

    fun updateAutomation(rule: AutomationRule) {
        viewModelScope.launch {
            val res = apiClient.updateAutomation(getBaseUrl(), getToken(), rule.id, rule)
            if (res.isSuccess) {
                loadAutomations()
            } else {
                _automations.value = _automations.value.map { if (it.id == rule.id) rule else it }
            }
        }
    }

    fun deleteAutomation(ruleId: String) {
        viewModelScope.launch {
            apiClient.deleteAutomation(getBaseUrl(), getToken(), ruleId)
            _automations.value = _automations.value.filterNot { it.id == ruleId }
        }
    }

    fun triggerAutomation(ruleId: String) {
        viewModelScope.launch {
            apiClient.triggerAutomation(getBaseUrl(), getToken(), ruleId)
        }
    }

    // --- TIMELINE / LOGBOOK ---
    fun loadTimelineEvents() {
        viewModelScope.launch {
            val res = apiClient.fetchLogbook(getBaseUrl(), getToken(), 100)
            if (res.isSuccess) {
                _timelineEvents.value = res.getOrDefault(TimelineEvent.sampleTimeline)
            }
        }
    }

    // --- MESH & HARDWARE DIAGNOSTICS ---
    fun loadMeshDiagnostics() {
        viewModelScope.launch {
            val healthRes = apiClient.fetchSystemHealth(getBaseUrl(), getToken())
            if (healthRes.isSuccess) {
                val (h, latency) = healthRes.getOrNull() ?: Pair(SystemHealthDto(), 12)
                _systemHealth.value = h
                _lanLatencyMs.value = latency
            }

            val meshRes = apiClient.fetchMeshTopology(getBaseUrl(), getToken())
            if (meshRes.isSuccess) {
                val nodesDto = meshRes.getOrNull()?.nodes ?: emptyList()
                if (nodesDto.isNotEmpty()) {
                    _meshNodes.value = nodesDto.map { dto ->
                        MeshNodeDiagnostic(
                            id = dto.id,
                            name = dto.name,
                            room = "Casa",
                            protocolType = dto.protocolType,
                            lqi = dto.lqi,
                            rssi = -60,
                            batteryLevel = dto.battery,
                            isRouter = dto.type == "router" || dto.type == "coordinator"
                        )
                    }
                }
            }
        }
    }

    // --- FAMILY & GUESTS ---
    fun loadFamilyAndInvites() {
        viewModelScope.launch {
            val membersRes = apiClient.fetchFamilyMembers(getBaseUrl(), getToken())
            if (membersRes.isSuccess && membersRes.getOrDefault(emptyList()).isNotEmpty()) {
                _familyAccounts.value = membersRes.getOrDefault(emptyList()).map { mem ->
                    DomusUserAccount(
                        id = mem.id,
                        name = mem.username,
                        role = when (mem.role) {
                            "admin" -> UserAccessRole.ADMIN
                            "child" -> UserAccessRole.CHILD
                            "guest" -> UserAccessRole.GUEST
                            else -> UserAccessRole.FAMILY
                        },
                        allowedRooms = listOf("Tutte le stanze"),
                        canDisarmAlarm = mem.role == "admin" || mem.role == "family",
                        canViewCameras = true
                    )
                }
            }

            val invitesRes = apiClient.fetchFamilyInvites(getBaseUrl(), getToken())
            if (invitesRes.isSuccess) {
                _familyInvites.value = invitesRes.getOrDefault(emptyList())
            }
        }
    }

    fun createFamilyInvite(role: String, expiresInHours: Int) {
        viewModelScope.launch {
            val res = apiClient.createFamilyInvite(getBaseUrl(), getToken(), role, expiresInHours)
            if (res.isSuccess) {
                loadFamilyAndInvites()
            }
        }
    }

    fun revokeFamilyInvite(inviteId: String) {
        viewModelScope.launch {
            apiClient.revokeFamilyInvite(getBaseUrl(), getToken(), inviteId)
            _familyInvites.value = _familyInvites.value.filterNot { it.id == inviteId }
        }
    }

    // --- KIOSK CONTROLS ---
    fun enterScreensaver() {
        _isScreensaverActive.value = true
        kioskController?.enterScreensaver()
    }

    fun wakeFromScreensaver() {
        _isScreensaverActive.value = false
        kioskController?.wakeFromScreensaver()
    }

    fun setKioskMode(enabled: Boolean) {
        _isKioskModeEnabled.value = enabled
        kioskController?.setKioskMode(enabled)
    }

    fun setBatteryProtection(enabled: Boolean) {
        _isBatteryProtectionEnabled.value = enabled
        kioskController?.setBatteryProtection(enabled)
    }

    fun setKioskSmartPlugId(id: String) {
        _kioskSmartPlugId.value = id
        kioskController?.setSmartPlugId(id)
    }

    // --- AUTH ---
    fun login(hubUrl: String, user: String, pass: String, onResult: (Boolean, String?) -> Unit) {
        viewModelScope.launch {
            val res = apiClient.login(hubUrl, user, pass)
            if (res.isSuccess) {
                val sess = res.getOrThrow()
                tokenStorage?.saveToken(sess.token ?: "")
                tokenStorage?.saveLocalEndpoint(hubUrl)
                _authSession.value = sess
                initialConnectAndSync()
                onResult(true, null)
            } else {
                onResult(false, res.exceptionOrNull()?.message ?: "Errore di autenticazione")
            }
        }
    }

    fun logout() {
        tokenStorage?.clear()
        _authSession.value = AuthSession(token = null, username = null, role = "user", isAuthenticated = false)
    }

    // --- CHORES ---
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

    // --- SHOPPING ---
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

    // --- ENERGY ---
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

    // --- NOTIFICATIONS ---
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
