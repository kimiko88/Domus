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
        AuthSession(token = null, username = null, role = "user", isAuthenticated = false)
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
        restoreSessionAndConnect()
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

    private fun restoreSessionAndConnect() {
        viewModelScope.launch {
            val token = tokenStorage?.getToken()
            if (!token.isNullOrBlank()) {
                val endpoint = tokenStorage.getRemoteEndpoint()
                    ?: tokenStorage.getLocalEndpoint()
                    ?: "https://domus.local.myhome.net"
                val restored = apiClient.validateSession(endpoint, token)
                if (restored.isSuccess) {
                    _authSession.value = restored.getOrThrow()
                } else {
                    tokenStorage.saveToken(null)
                    _authSession.value = AuthSession(token = null, username = null, role = "user", isAuthenticated = false)
                }
            }
            initialConnectAndSync()
        }
    }

    fun initialConnectAndSync() {
        viewModelScope.launch {
            _isRefreshing.value = true
            discoverAndConnectUseCase(preferLocal = true)
            syncEntitiesUseCase()
            loadFinance()
            _isRefreshing.value = false
        }
    }

    fun selectArea(areaId: String?) {
        _selectedAreaId.value = areaId
    }

    fun setSearchQuery(query: String) {
        _searchQuery.value = query
    }

    val isBiometricSecurityGateEnabled = MutableStateFlow(true)

    fun isEntitySensitive(domain: String, entityId: String): Boolean {
        return com.domus.app.services.biometrics.BiometricAuthHelper.isSensitiveEntity(domain, entityId)
    }

    fun toggleEntity(entityId: String) {
        viewModelScope.launch {
            toggleEntityUseCase(entityId)
        }
    }

    fun toggleEntityWithBiometricCheck(
        domain: String,
        entityId: String,
        onRequireBiometric: (title: String, subtitle: String, onConfirmed: () -> Unit) -> Unit
    ) {
        if (isBiometricSecurityGateEnabled.value && isEntitySensitive(domain, entityId)) {
            val (title, subtitle) = com.domus.app.services.biometrics.BiometricAuthHelper.createPromptTitle(
                if (domain == "alarm_control_panel") "disarm" else "lock"
            )
            onRequireBiometric(title, subtitle) {
                toggleEntity(entityId)
            }
        } else {
            toggleEntity(entityId)
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
    fun login(hubUrl: String, user: String, pass: String, twoFactorCode: String? = null, onResult: (Boolean, String?) -> Unit) {
        viewModelScope.launch {
            val res = apiClient.login(hubUrl, user, pass, twoFactorCode)
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

    // --- HOME FINANCE ---
    private val _financeSummary = MutableStateFlow<com.domus.app.domain.model.FinancialSummaryDto?>(null)
    val financeSummary: StateFlow<com.domus.app.domain.model.FinancialSummaryDto?> = _financeSummary.asStateFlow()

    private val _financeExpenses = MutableStateFlow<List<com.domus.app.domain.model.ExpenseDto>>(emptyList())
    val financeExpenses: StateFlow<List<com.domus.app.domain.model.ExpenseDto>> = _financeExpenses.asStateFlow()

    fun loadFinance(period: String = "month") {
        viewModelScope.launch {
            val resSum = apiClient.fetchFinancialSummary(getBaseUrl(), getToken(), period)
            if (resSum.isSuccess) {
                _financeSummary.value = resSum.getOrNull()
            }
            val resExp = apiClient.fetchExpenses(getBaseUrl(), getToken())
            if (resExp.isSuccess) {
                _financeExpenses.value = resExp.getOrDefault(emptyList())
            }
        }
    }

    fun addExpense(
        title: String,
        amount: Double,
        category: String = "food",
        subcategory: String = "groceries",
        isFixed: Boolean = false,
        kwh: Double? = null,
        expiry: String? = null
    ) {
        viewModelScope.launch {
            val req = com.domus.app.domain.model.CreateExpenseRequestDto(
                title = title,
                amount = amount,
                category = category,
                subcategory = subcategory,
                expense_date = java.time.Instant.now().toString(),
                is_fixed = isFixed,
                energy_kwh = kwh,
                contract_expiry_date = expiry
            )
            val res = apiClient.createExpense(getBaseUrl(), getToken(), req)
            if (res.isSuccess) {
                loadFinance()
            }
        }
    }

    fun setMonthlyBudget(limit: Double) {
        viewModelScope.launch {
            val req = com.domus.app.domain.model.SetBudgetRequestDto(monthly_limit = limit)
            val res = apiClient.setBudget(getBaseUrl(), getToken(), req)
            if (res.isSuccess) {
                loadFinance()
            }
        }
    }

    // --- PANTRY, RECIPES, MEAL PLANS & RECEIPT OCR ---
    private val _pantryItems = MutableStateFlow<List<PantryItemDto>>(emptyList())
    val pantryItems: StateFlow<List<PantryItemDto>> = _pantryItems.asStateFlow()

    private val _recipes = MutableStateFlow<List<RecipeDto>>(emptyList())
    val recipes: StateFlow<List<RecipeDto>> = _recipes.asStateFlow()

    private val _mealPlans = MutableStateFlow<List<MealPlanDto>>(emptyList())
    val mealPlans: StateFlow<List<MealPlanDto>> = _mealPlans.asStateFlow()

    private val _isPantryLoading = MutableStateFlow(false)
    val isPantryLoading: StateFlow<Boolean> = _isPantryLoading.asStateFlow()

    private val _lastConsumedResult = MutableStateFlow<ConsumeMealPlanResponseDto?>(null)
    val lastConsumedResult: StateFlow<ConsumeMealPlanResponseDto?> = _lastConsumedResult.asStateFlow()

    private val _parsedReceipt = MutableStateFlow<ReceiptDto?>(null)
    val parsedReceipt: StateFlow<ReceiptDto?> = _parsedReceipt.asStateFlow()

    private val _receiptImportResult = MutableStateFlow<ReceiptImportResultDto?>(null)
    val receiptImportResult: StateFlow<ReceiptImportResultDto?> = _receiptImportResult.asStateFlow()

    fun loadPantry() {
        viewModelScope.launch {
            _isPantryLoading.value = true
            val res = apiClient.fetchPantryItems(getBaseUrl(), getToken())
            if (res.isSuccess) {
                _pantryItems.value = res.getOrDefault(emptyList())
            }
            _isPantryLoading.value = false
        }
    }

    fun updatePantryStock(id: String, delta: Double) {
        viewModelScope.launch {
            val res = apiClient.updatePantryStock(getBaseUrl(), getToken(), id, delta = delta)
            if (res.isSuccess) {
                loadPantry()
            }
        }
    }

    fun createPantryItem(req: CreatePantryItemRequestDto, onComplete: ((Boolean) -> Unit)? = null) {
        viewModelScope.launch {
            val res = apiClient.createPantryItem(getBaseUrl(), getToken(), req)
            val success = res.isSuccess
            if (success) {
                loadPantry()
            }
            onComplete?.invoke(success)
        }
    }

    fun deletePantryItem(id: String) {
        viewModelScope.launch {
            val res = apiClient.deletePantryItem(getBaseUrl(), getToken(), id)
            if (res.isSuccess) {
                loadPantry()
            }
        }
    }

    fun loadRecipes() {
        viewModelScope.launch {
            val res = apiClient.fetchRecipes(getBaseUrl(), getToken())
            if (res.isSuccess) {
                _recipes.value = res.getOrDefault(emptyList())
            }
        }
    }

    fun createRecipe(req: CreateRecipeRequestDto, onComplete: ((Boolean) -> Unit)? = null) {
        viewModelScope.launch {
            val res = apiClient.createRecipe(getBaseUrl(), getToken(), req)
            val success = res.isSuccess
            if (success) {
                loadRecipes()
            }
            onComplete?.invoke(success)
        }
    }

    fun deleteRecipe(id: String) {
        viewModelScope.launch {
            val res = apiClient.deleteRecipe(getBaseUrl(), getToken(), id)
            if (res.isSuccess) {
                loadRecipes()
            }
        }
    }

    fun loadMealPlans(startDate: String? = null, endDate: String? = null) {
        viewModelScope.launch {
            val res = apiClient.fetchMealPlans(getBaseUrl(), getToken(), startDate, endDate)
            if (res.isSuccess) {
                _mealPlans.value = res.getOrDefault(emptyList())
            }
        }
    }

    fun createMealPlan(req: CreateMealPlanRequestDto, onComplete: ((Boolean) -> Unit)? = null) {
        viewModelScope.launch {
            val res = apiClient.createMealPlan(getBaseUrl(), getToken(), req)
            val success = res.isSuccess
            if (success) {
                loadMealPlans()
            }
            onComplete?.invoke(success)
        }
    }

    fun consumeMealPlan(planId: String, personsCount: Int? = null, onComplete: ((ConsumeMealPlanResponseDto?) -> Unit)? = null) {
        viewModelScope.launch {
            val res = apiClient.consumeMealPlan(getBaseUrl(), getToken(), planId, personsCount)
            if (res.isSuccess) {
                val data = res.getOrNull()
                _lastConsumedResult.value = data
                loadMealPlans()
                loadPantry()
                onComplete?.invoke(data)
            } else {
                onComplete?.invoke(null)
            }
        }
    }

    fun deleteMealPlan(id: String) {
        viewModelScope.launch {
            val res = apiClient.deleteMealPlan(getBaseUrl(), getToken(), id)
            if (res.isSuccess) {
                loadMealPlans()
            }
        }
    }

    fun parseReceipt(rawText: String, onComplete: ((ReceiptDto?) -> Unit)? = null) {
        viewModelScope.launch {
            _isPantryLoading.value = true
            val res = apiClient.parseReceipt(getBaseUrl(), getToken(), rawText)
            _isPantryLoading.value = false
            if (res.isSuccess) {
                val parsed = res.getOrNull()
                _parsedReceipt.value = parsed
                onComplete?.invoke(parsed)
            } else {
                onComplete?.invoke(null)
            }
        }
    }

    fun importReceipt(receipt: ReceiptDto, autoFinance: Boolean = true, autoPantry: Boolean = true, onComplete: ((ReceiptImportResultDto?) -> Unit)? = null) {
        viewModelScope.launch {
            _isPantryLoading.value = true
            val req = ReceiptImportRequestDto(
                receipt = receipt,
                auto_add_to_finance = autoFinance,
                auto_update_pantry = autoPantry
            )
            val res = apiClient.importReceipt(getBaseUrl(), getToken(), req)
            _isPantryLoading.value = false
            if (res.isSuccess) {
                val result = res.getOrNull()
                _receiptImportResult.value = result
                loadPantry()
                loadFinance()
                onComplete?.invoke(result)
            } else {
                onComplete?.invoke(null)
            }
        }
    }

    // --- MEDICINE CABINET & EXPIRATION ---
    private val _medicines = MutableStateFlow<List<MedicineDto>>(emptyList())
    val medicines: StateFlow<List<MedicineDto>> = _medicines.asStateFlow()

    private val _medicineAlerts = MutableStateFlow<List<MedicineAlertDto>>(emptyList())
    val medicineAlerts: StateFlow<List<MedicineAlertDto>> = _medicineAlerts.asStateFlow()

    private val _isMedicineLoading = MutableStateFlow(false)
    val isMedicineLoading: StateFlow<Boolean> = _isMedicineLoading.asStateFlow()

    fun loadMedicines(location: String? = null, status: String? = null, query: String? = null) {
        viewModelScope.launch {
            _isMedicineLoading.value = true
            val res = apiClient.getMedicines(getBaseUrl(), getToken(), location = location, status = status, query = query)
            if (res.isSuccess) {
                _medicines.value = res.getOrDefault(emptyList())
            }
            _isMedicineLoading.value = false
        }
    }

    fun loadMedicineAlerts(fridgeTemp: Double? = null) {
        viewModelScope.launch {
            val res = apiClient.getMedicineAlerts(getBaseUrl(), getToken(), fridgeTemp = fridgeTemp)
            if (res.isSuccess) {
                _medicineAlerts.value = res.getOrDefault(emptyList())
            }
        }
    }

    fun createMedicine(req: CreateMedicineRequestDto, onComplete: ((Boolean) -> Unit)? = null) {
        viewModelScope.launch {
            val res = apiClient.createMedicine(getBaseUrl(), getToken(), req)
            val success = res.isSuccess
            if (success) {
                loadMedicines()
                loadMedicineAlerts()
            }
            onComplete?.invoke(success)
        }
    }

    fun openMedicine(id: String, openedAt: String? = null, onComplete: ((Boolean) -> Unit)? = null) {
        viewModelScope.launch {
            val res = apiClient.openMedicine(getBaseUrl(), getToken(), id, openedAt = openedAt)
            val success = res.isSuccess
            if (success) {
                loadMedicines()
                loadMedicineAlerts()
            }
            onComplete?.invoke(success)
        }
    }

    fun consumeMedicine(id: String, amount: Double = 1.0, onComplete: ((Boolean) -> Unit)? = null) {
        viewModelScope.launch {
            val res = apiClient.consumeMedicine(getBaseUrl(), getToken(), id, amount = amount)
            val success = res.isSuccess
            if (success) {
                loadMedicines()
                loadMedicineAlerts()
            }
            onComplete?.invoke(success)
        }
    }

    fun deleteMedicine(id: String, onComplete: ((Boolean) -> Unit)? = null) {
        viewModelScope.launch {
            val res = apiClient.deleteMedicine(getBaseUrl(), getToken(), id)
            val success = res.isSuccess
            if (success) {
                loadMedicines()
                loadMedicineAlerts()
            }
            onComplete?.invoke(success)
        }
    }

    // --- PREDICTIVE CLIMATE ---
    private val _predictiveStatus = MutableStateFlow<SystemPredictiveStatusDto?>(null)
    val predictiveStatus: StateFlow<SystemPredictiveStatusDto?> = _predictiveStatus.asStateFlow()

    private val _isPredictiveLoading = MutableStateFlow(false)
    val isPredictiveLoading: StateFlow<Boolean> = _isPredictiveLoading.asStateFlow()

    fun loadPredictiveStatus() {
        viewModelScope.launch {
            _isPredictiveLoading.value = true
            val res = apiClient.getPredictiveStatus(getBaseUrl(), getToken())
            if (res.isSuccess) {
                _predictiveStatus.value = res.getOrNull()
            }
            _isPredictiveLoading.value = false
        }
    }

    fun evaluatePredictiveZone(id: String, onComplete: ((PredictiveEvaluationDto?) -> Unit)? = null) {
        viewModelScope.launch {
            val res = apiClient.evaluatePredictiveZone(getBaseUrl(), getToken(), id)
            if (res.isSuccess) {
                loadPredictiveStatus()
                onComplete?.invoke(res.getOrNull())
            } else {
                onComplete?.invoke(null)
            }
        }
    }

    fun evaluateAllPredictiveZones(onComplete: ((Boolean) -> Unit)? = null) {
        viewModelScope.launch {
            _isPredictiveLoading.value = true
            val res = apiClient.evaluateAllPredictiveZones(getBaseUrl(), getToken())
            val success = res.isSuccess
            if (success) {
                loadPredictiveStatus()
            }
            _isPredictiveLoading.value = false
            onComplete?.invoke(success)
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
