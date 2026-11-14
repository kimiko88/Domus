import Foundation
import SwiftUI
import Combine

/// Dashboard ViewModel per iOS — gestisce lo stato reattivo e i comandi della Fase 2.
@MainActor
public final class DashboardViewModel: ObservableObject {
    @Published public var connection: ServerConnection = ServerConnection()
    @Published public var entities: [Entity] = []
    @Published public var areas: [Area] = []
    @Published public var selectedAreaId: String? = nil
    @Published public var searchQuery: String = ""
    @Published public var isRefreshing: Bool = false
    @Published public var selectedEntityForDetail: Entity? = nil
    @Published public var showAutomationSettings: Bool = false
    @Published public var showKioskSettings: Bool = false
    @Published public var showFloorplanModal: Bool = false
    @Published public var customEntityOrder: [String] = []
    @Published public var autoFocusNearbyRoom: Bool = true
    @Published public var authSession: AuthSession = AuthSession()
    @Published public var showLoginSheet: Bool = false
    @Published public var isLoggingIn: Bool = false
    @Published public var loginError: String? = nil
    @Published public var requiresTwoFactor: Bool = false
    @Published public var financeSummary: FinancialSummaryDTO? = nil
    @Published public var financeExpenses: [ExpenseDTO] = []
    @Published public var showFinanceSheet: Bool = false

    public let presenceManager: GeofencingPresenceManager
    public let notificationManager: NotificationManager
    public let nfcManager: NfcTagManager
    public let kioskManager: KioskModeManager
    public let watchSessionManager: WatchSessionManager
    public let beaconManager: RoomPresenceBeaconManager
    public let liveActivityManager: DomusLiveActivityManager
    public let carPlayManager: DomusCarPlayManager
    public let vpnManager: VpnOnDemandManager
    public let backgroundTaskManager: DomusBackgroundTaskManager

    private let entityRepository: EntityRepositoryProtocol
    private let connectionRepository: ConnectionRepositoryProtocol
    private let authRepository: AuthRepositoryProtocol?
    private let discoverAndConnectUseCase: DiscoverAndConnectUseCase
    private let syncEntitiesUseCase: SyncEntitiesUseCase
    private let toggleEntityUseCase: ToggleEntityUseCase

    public init(
        entityRepository: EntityRepositoryProtocol,
        connectionRepository: ConnectionRepositoryProtocol,
        discoverAndConnectUseCase: DiscoverAndConnectUseCase,
        syncEntitiesUseCase: SyncEntitiesUseCase,
        toggleEntityUseCase: ToggleEntityUseCase,
        authRepository: AuthRepositoryProtocol? = nil,
        presenceManager: GeofencingPresenceManager? = nil,
        notificationManager: NotificationManager? = nil,
        nfcManager: NfcTagManager? = nil,
        kioskManager: KioskModeManager? = nil,
        watchSessionManager: WatchSessionManager? = nil,
        beaconManager: RoomPresenceBeaconManager? = nil,
        liveActivityManager: DomusLiveActivityManager? = nil,
        carPlayManager: DomusCarPlayManager? = nil,
        vpnManager: VpnOnDemandManager? = nil,
        backgroundTaskManager: DomusBackgroundTaskManager? = nil
    ) {
        self.entityRepository = entityRepository
        self.connectionRepository = connectionRepository
        self.discoverAndConnectUseCase = discoverAndConnectUseCase
        self.syncEntitiesUseCase = syncEntitiesUseCase
        self.toggleEntityUseCase = toggleEntityUseCase
        self.authRepository = authRepository
        self.presenceManager = presenceManager ?? GeofencingPresenceManager()
        self.notificationManager = notificationManager ?? NotificationManager()
        self.nfcManager = nfcManager ?? NfcTagManager()
        self.kioskManager = kioskManager ?? KioskModeManager()
        self.watchSessionManager = watchSessionManager ?? WatchSessionManager()
        self.beaconManager = beaconManager ?? RoomPresenceBeaconManager.shared
        self.liveActivityManager = liveActivityManager ?? DomusLiveActivityManager.shared
        self.carPlayManager = carPlayManager ?? DomusCarPlayManager.shared
        self.vpnManager = vpnManager ?? VpnOnDemandManager.shared
        self.backgroundTaskManager = backgroundTaskManager ?? DomusBackgroundTaskManager.shared

        if let savedOrder = UserDefaults.standard.stringArray(forKey: "domus_custom_card_order") {
            self.customEntityOrder = savedOrder
        }

        setupAutomationCallbacks()
    }

    public var filteredEntities: [Entity] {
        let base = entities.filter { entity in
            let matchesArea = selectedAreaId == nil || entity.areaId == selectedAreaId
            let matchesQuery = searchQuery.trimmingCharacters(in: .whitespaces).isEmpty ||
                entity.name.localizedCaseInsensitiveContains(searchQuery) ||
                entity.domain.localizedCaseInsensitiveContains(searchQuery)
            return matchesArea && matchesQuery
        }

        if customEntityOrder.isEmpty {
            return base
        }

        return base.sorted { a, b in
            let idxA = customEntityOrder.firstIndex(of: a.id) ?? Int.max
            let idxB = customEntityOrder.firstIndex(of: b.id) ?? Int.max
            if idxA != idxB {
                return idxA < idxB
            }
            return a.name < b.name
        }
    }

    public func initialConnectAndSync() {
        Task {
            isRefreshing = true
            if let authRepo = authRepository {
                let sess = await authRepo.restoreSession()
                self.authSession = sess
                if !sess.isAuthenticated {
                    self.showLoginSheet = true
                    self.isRefreshing = false
                    return
                }
            }
            _ = try? await discoverAndConnectUseCase.execute(preferLocal: true)
            self.connection = await connectionRepository.getCurrentConnection()
            self.entities = (try? await syncEntitiesUseCase.execute()) ?? []
            self.areas = await entityRepository.getAreas()
            await self.loadFinance()
            isRefreshing = false
        }
    }

    public func login(username: String, password: String, localEndpoint: String? = nil, remoteEndpoint: String? = nil, twoFactorCode: String? = nil) async -> Bool {
        guard let authRepository else { return false }
        isLoggingIn = true
        loginError = nil
        defer { isLoggingIn = false }
        do {
            let session = try await authRepository.login(username: username, password: password, localEndpoint: localEndpoint, remoteEndpoint: remoteEndpoint, twoFactorCode: twoFactorCode)
            self.authSession = session
            self.requiresTwoFactor = false
            self.connection = await connectionRepository.getCurrentConnection()
            self.showLoginSheet = false
            self.entities = (try? await syncEntitiesUseCase.execute()) ?? []
            self.areas = await entityRepository.getAreas()
            return true
        } catch {
            if let networkError = error as? DomusNetworkError, case .twoFactorRequired = networkError {
                self.requiresTwoFactor = true
            }
            self.loginError = error.localizedDescription
            return false
        }
    }

    public func logout() {
        Task {
            await authRepository?.logout()
            self.authSession = AuthSession()
            self.entities = []
            self.areas = []
            self.showLoginSheet = true
        }
    }

    public func discoverLocalHub() async -> String? {
        return await connectionRepository.discoverLocalHub(timeoutSeconds: 2.0)
    }

    public func toggleEntity(id: String) {
        Haptics.medium()
        Task {
            if let updated = try? await toggleEntityUseCase.execute(entityId: id) {
                if let index = entities.firstIndex(where: { $0.id == id }) {
                    entities[index] = updated
                    if selectedEntityForDetail?.id == id {
                        selectedEntityForDetail = updated
                    }
                }
            }
        }
    }

    public func selectArea(_ areaId: String?) {
        Haptics.selection()
        self.selectedAreaId = areaId
    }

    public func selectEntityForDetail(_ entity: Entity?) {
        if entity != nil {
            Haptics.light()
        }
        self.selectedEntityForDetail = entity
    }

    public func reconnect() {
        initialConnectAndSync()
    }

    // MARK: - Fase 2: Controlli Luce (Luminosità, HSL, Temperatura Colore)

    public func setLightBrightness(id: String, brightness: Int) {
        guard let index = entities.firstIndex(where: { $0.id == id }) else { return }
        var current = entities[index]
        current.attributes["brightness"] = AnyCodable(brightness)
        if brightness > 0 { current.state = "on" }
        entities[index] = current
        if selectedEntityForDetail?.id == id { selectedEntityForDetail = current }

        Task {
            try? await entityRepository.callService(
                domain: "light",
                service: "turn_on",
                entityId: id,
                data: ["brightness": brightness]
            )
        }
    }

    public func setLightColor(id: String, r: Double, g: Double, b: Double) {
        guard let index = entities.firstIndex(where: { $0.id == id }) else { return }
        var current = entities[index]
        let rgbArray = [Int(r * 255), Int(g * 255), Int(b * 255)]
        current.attributes["rgb_color"] = AnyCodable(rgbArray)
        current.state = "on"
        entities[index] = current
        if selectedEntityForDetail?.id == id { selectedEntityForDetail = current }

        Task {
            try? await entityRepository.callService(
                domain: "light",
                service: "turn_on",
                entityId: id,
                data: ["rgb_color": "\(rgbArray)"]
            )
        }
    }

    public func setLightColorTemp(id: String, mireds: Int) {
        guard let index = entities.firstIndex(where: { $0.id == id }) else { return }
        var current = entities[index]
        current.attributes["color_temp"] = AnyCodable(mireds)
        current.state = "on"
        entities[index] = current
        if selectedEntityForDetail?.id == id { selectedEntityForDetail = current }

        Task {
            try? await entityRepository.callService(
                domain: "light",
                service: "turn_on",
                entityId: id,
                data: ["color_temp": mireds]
            )
        }
    }

    // MARK: - Fase 2: Termostato Circolare & HVAC

    public func setThermostatTemperature(id: String, targetTemp: Double) {
        guard let index = entities.firstIndex(where: { $0.id == id }) else { return }
        var current = entities[index]
        current.attributes["temperature"] = AnyCodable(targetTemp)
        entities[index] = current
        if selectedEntityForDetail?.id == id { selectedEntityForDetail = current }

        Task {
            try? await entityRepository.callService(
                domain: "climate",
                service: "set_temperature",
                entityId: id,
                data: ["temperature": targetTemp]
            )
        }
    }

    public func setThermostatHvacMode(id: String, mode: String) {
        guard let index = entities.firstIndex(where: { $0.id == id }) else { return }
        var current = entities[index]
        current.state = mode
        entities[index] = current
        if selectedEntityForDetail?.id == id { selectedEntityForDetail = current }

        Task {
            try? await entityRepository.callService(
                domain: "climate",
                service: "set_hvac_mode",
                entityId: id,
                data: ["hvac_mode": mode]
            )
        }
    }

    public func moveEntity(from sourceIndex: Int, to destinationIndex: Int) {
        let currentList = filteredEntities
        guard sourceIndex < currentList.count, destinationIndex < currentList.count else { return }
        Haptics.medium()
        var currentIds = currentList.map { $0.id }
        let moved = currentIds.remove(at: sourceIndex)
        currentIds.insert(moved, at: destinationIndex)
        self.customEntityOrder = currentIds
        UserDefaults.standard.set(currentIds, forKey: "domus_custom_card_order")
    }

    // MARK: - Fase 2: Tastierino Allarme (PIN, Arm Home/Away, Disarm)

    public func armAlarm(id: String, code: String, mode: String) {
        liveActivityManager.startAlarmCountdown(seconds: 30, isArming: true)
        Task {
            do {
                try await entityRepository.callService(
                    domain: "alarm_control_panel",
                    service: mode,
                    entityId: id,
                    data: ["code": code]
                )
                Haptics.success()
                if let index = self.entities.firstIndex(where: { $0.id == id }) {
                    let targetState = mode == "alarm_arm_home" ? "armed_home" : "armed_away"
                    self.entities[index].state = targetState
                    if self.selectedEntityForDetail?.id == id {
                        self.selectedEntityForDetail?.state = targetState
                    }
                }
            } catch {
                liveActivityManager.stopActivity()
                Haptics.error()
            }
        }
    }

    public func disarmAlarm(id: String, code: String) {
        liveActivityManager.stopActivity()
        Task {
            do {
                try await entityRepository.callService(
                    domain: "alarm_control_panel",
                    service: "alarm_disarm",
                    entityId: id,
                    data: ["code": code]
                )
                Haptics.success()
                if let index = self.entities.firstIndex(where: { $0.id == id }) {
                    self.entities[index].state = "disarmed"
                    if self.selectedEntityForDetail?.id == id {
                        self.selectedEntityForDetail?.state = "disarmed"
                    }
                }
            } catch {
                Haptics.error()
            }
        }
    }

    // MARK: - Fase 2: Telecamere

    public func refreshCamera(id: String) {
        Haptics.light()
        Task {
            // Trigger refresh
            _ = try? await syncEntitiesUseCase.execute()
        }
    }

    // MARK: - Fase 3 & 6: Automazioni NFC, Notifiche, Beacon & CarPlay

    private func setupAutomationCallbacks() {
        beaconManager.onRoomChanged = { [weak self] newRoomId in
            guard let self = self, self.autoFocusNearbyRoom, let newRoomId = newRoomId else { return }
            self.selectedAreaId = newRoomId
        }

        carPlayManager.onCarPlayAction = { [weak self] domain, entityId in
            guard let self = self else { return }
            self.toggleEntity(id: entityId)
        }

        backgroundTaskManager.onMaintenanceTriggered = { [weak self] in
            guard let self = self else { return "Hub non raggiungibile" }
            _ = try? await self.syncEntitiesUseCase.execute()
            return "Cache sincronizzata con successo ed entità aggiornate"
        }

        nfcManager.onTagScanned = { [weak self] tag in
            guard let self else { return }
            self.handleNfcTag(tag)
        }

        notificationManager.onActionTriggered = { [weak self] actionId, categoryId, userInfo in
            guard let self else { return }
            self.handleNotificationAction(actionId: actionId, categoryId: categoryId, userInfo: userInfo)
        }

        kioskManager.onSmartPlugCommand = { [weak self] plugId, turnOn in
            guard let self else { return }
            Task {
                let service = turnOn ? "turn_on" : "turn_off"
                try? await self.entityRepository.callService(
                    domain: "switch",
                    service: service,
                    entityId: plugId,
                    data: [:]
                )
            }
        }

        watchSessionManager.onWatchCommandReceived = { [weak self] action, targetId in
            guard let self else { return }
            if action == "toggle" {
                self.toggleEntity(id: targetId)
            }
        }
    }


    private func handleNfcTag(_ tag: DomusNfcTag) {
        switch tag.actionType {
        case "toggle":
            toggleEntity(id: tag.targetId)
        case "scene":
            Task {
                try? await entityRepository.callService(
                    domain: "scene",
                    service: "turn_on",
                    entityId: tag.targetId,
                    data: [:]
                )
            }
        case "alarm":
            Task {
                try? await entityRepository.callService(
                    domain: "alarm_control_panel",
                    service: tag.targetId,
                    entityId: "alarm_control_panel.home",
                    data: [:]
                )
            }
        default:
            break
        }
    }

    private func handleNotificationAction(actionId: String, categoryId: String, userInfo: [AnyHashable: Any]) {
        switch actionId {
        case DomusNotificationAction.closeCovers.rawValue:
            Task {
                try? await entityRepository.callService(
                    domain: "cover",
                    service: "close_cover",
                    entityId: "all",
                    data: [:]
                )
            }
        case DomusNotificationAction.disarmAlarm.rawValue:
            disarmAlarm(id: "alarm_control_panel.home", code: "")
        default:
            break
        }
    }

    // MARK: - Home Finance

    public func loadFinance(period: String = "month") async {
        let client = DomusHttpClient.shared
        if let summary = try? await client.fetchFinancialSummary(period: period) {
            self.financeSummary = summary
        }
        if let expenses = try? await client.fetchExpenses() {
            self.financeExpenses = expenses
        }
    }

    public func addExpense(
        title: String,
        amount: Double,
        category: String = "food",
        subcategory: String = "groceries",
        isFixed: Bool = false,
        energyKwh: Double? = nil,
        contractExpiryDate: String? = nil
    ) async -> Bool {
        let req = CreateExpenseRequestDTO(
            title: title,
            amount: amount,
            category: category,
            subcategory: subcategory,
            isFixed: isFixed,
            contractExpiryDate: contractExpiryDate,
            energyKwh: energyKwh
        )
        do {
            _ = try await DomusHttpClient.shared.createExpense(req: req)
            await loadFinance()
            return true
        } catch {
            return false
        }
    }

    public func setMonthlyBudget(limit: Double) async -> Bool {
        let req = SetBudgetRequestDTO(monthlyLimit: limit)
        do {
            _ = try await DomusHttpClient.shared.setBudget(req: req)
            await loadFinance()
            return true
        } catch {
            return false
        }
    }
}

