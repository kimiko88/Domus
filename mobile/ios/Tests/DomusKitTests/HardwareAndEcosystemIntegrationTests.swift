import XCTest
@testable import DomusKit

@MainActor
final class HardwareAndEcosystemIntegrationTests: XCTestCase {

    final class IntegrationMockEntityRepository: EntityRepositoryProtocol, @unchecked Sendable {
        var entities: [Entity] = []
        var areas: [Area] = []
        var lastServiceCall: (domain: String, service: String, entityId: String, data: [String: Any])?
        var toggleCallCount: Int = 0

        func getEntities() async -> [Entity] { entities }
        func getEntity(id: String) async -> Entity? { entities.first(where: { $0.id == id }) }
        func syncEntities() async throws -> [Entity] { entities }
        func toggleEntity(id: String) async throws -> Entity {
            toggleCallCount += 1
            if let idx = entities.firstIndex(where: { $0.id == id }) {
                if entities[idx].domain == "cover" {
                    entities[idx].state = entities[idx].isOn ? "closed" : "open"
                } else {
                    entities[idx].state = entities[idx].isOn ? "off" : "on"
                }
                return entities[idx]
            }
            let fallback = Entity(id: id, name: id, domain: "cover", state: "open")
            entities.append(fallback)
            return fallback
        }
        func callService(domain: String, service: String, entityId: String, data: [String: Any]) async throws {
            lastServiceCall = (domain, service, entityId, data)
        }
        func updateLocalEntity(_ entity: Entity) async {
            if let idx = entities.firstIndex(where: { $0.id == entity.id }) {
                entities[idx] = entity
            } else {
                entities.append(entity)
            }
        }
        func getAreas() async -> [Area] { areas }
    }

    final class IntegrationMockConnectionRepository: ConnectionRepositoryProtocol, @unchecked Sendable {
        var mode: TransportMode = .localLan

        func getCurrentConnection() async -> ServerConnection {
            ServerConnection(mode: mode, isConnected: true)
        }
        func discoverLocalHub(timeoutSeconds: Double) async -> String? { "http://192.168.1.100:8080" }
        func connect(mode: TransportMode?) async throws {
            self.mode = mode ?? .localLan
        }
        func disconnect() async {
            self.mode = .disconnected
        }
        func setEndpoints(local: String?, remote: String?) async {}
    }

    private func createTestViewModel(
        entityRepo: IntegrationMockEntityRepository,
        connRepo: IntegrationMockConnectionRepository
    ) -> DashboardViewModel {
        let discoverUseCase = DiscoverAndConnectUseCase(connectionRepository: connRepo)
        let syncUseCase = SyncEntitiesUseCase(entityRepository: entityRepo)
        let toggleUseCase = ToggleEntityUseCase(entityRepository: entityRepo)

        let vm = DashboardViewModel(
            entityRepository: entityRepo,
            connectionRepository: connRepo,
            discoverAndConnectUseCase: discoverUseCase,
            syncEntitiesUseCase: syncUseCase,
            toggleEntityUseCase: toggleUseCase
        )
        return vm
    }

    // MARK: - 1. Beacon Proximity -> Dashboard Area Auto-Focus Integration

    func testBeaconRoomPresenceIntegrationWithDashboard() async {
        let entityRepo = IntegrationMockEntityRepository()
        entityRepo.areas = [
            Area(id: "living_room", name: "Salotto"),
            Area(id: "kitchen", name: "Cucina")
        ]
        entityRepo.entities = [
            Entity(id: "light.living_room", name: "Luce Salotto", domain: "light", state: "on", areaId: "living_room"),
            Entity(id: "light.kitchen", name: "Luce Cucina", domain: "light", state: "off", areaId: "kitchen"),
            Entity(id: "switch.coffee_machine", name: "Macchina Caffè", domain: "switch", state: "on", areaId: "kitchen")
        ]
        let connRepo = IntegrationMockConnectionRepository()
        let vm = createTestViewModel(entityRepo: entityRepo, connRepo: connRepo)

        vm.entities = entityRepo.entities
        vm.areas = entityRepo.areas
        vm.autoFocusNearbyRoom = true

        XCTAssertNil(vm.selectedAreaId)
        XCTAssertEqual(vm.filteredEntities.count, 3)

        // Simula il rilevamento del Beacon della Cucina
        vm.beaconManager.simulateRoomProximity(roomId: "kitchen", distanceMeters: 0.6)

        XCTAssertEqual(vm.selectedAreaId, "kitchen")
        // La lista filtrata della dashboard ora include esclusivamente i dispositivi della cucina
        let filtered = vm.filteredEntities
        XCTAssertEqual(filtered.count, 2)
        XCTAssertTrue(filtered.allSatisfy { $0.areaId == "kitchen" })

        // Se autoFocusNearbyRoom viene disabilitato dall'utente, il cambio stanza non altera la visuale
        vm.autoFocusNearbyRoom = false
        vm.beaconManager.simulateRoomProximity(roomId: "living_room", distanceMeters: 0.4)
        XCTAssertEqual(vm.selectedAreaId, "kitchen") // rimane invariato
    }

    // MARK: - 2. CarPlay Action -> Entity State & Service Integration

    func testCarPlayActionIntegrationWithDashboard() async {
        let entityRepo = IntegrationMockEntityRepository()
        let garageEntity = Entity(id: "cover.garage", name: "Garage", domain: "cover", state: "closed")
        entityRepo.entities = [garageEntity]
        let connRepo = IntegrationMockConnectionRepository()
        let vm = createTestViewModel(entityRepo: entityRepo, connRepo: connRepo)
        vm.entities = entityRepo.entities

        XCTAssertEqual(vm.entities.first(where: { $0.id == "cover.garage" })?.state, "closed")

        // Invocazione azione da cruscotto CarPlay
        #if canImport(CarPlay) && os(iOS)
        vm.carPlayManager.onCarPlayAction?("cover", "cover.garage")
        #else
        vm.carPlayManager.simulateCarAction(domain: "cover", entityId: "cover.garage")
        #endif

        // Attesa esecuzione asincrona del task
        try? await Task.sleep(nanoseconds: 50_000_000)

        XCTAssertEqual(entityRepo.toggleCallCount, 1)
        XCTAssertEqual(vm.entities.first(where: { $0.id == "cover.garage" })?.state, "open")
    }

    // MARK: - 3. Alarm Arm / Disarm -> Live Activity Lifecycle Integration

    func testAlarmArmAndDisarmWithLiveActivityIntegration() async {
        let entityRepo = IntegrationMockEntityRepository()
        let alarmEntity = Entity(id: "alarm_control_panel.home", name: "Allarme Casa", domain: "alarm_control_panel", state: "disarmed")
        entityRepo.entities = [alarmEntity]
        let connRepo = IntegrationMockConnectionRepository()
        let vm = createTestViewModel(entityRepo: entityRepo, connRepo: connRepo)
        vm.entities = entityRepo.entities

        // 1. Armamento allarme
        vm.armAlarm(id: "alarm_control_panel.home", code: "9988", mode: "alarm_arm_away")
        try? await Task.sleep(nanoseconds: 50_000_000)

        // Verifica che Live Activity countdown sia stato attivato
        XCTAssertTrue(vm.liveActivityManager.isActivityActive)
        XCTAssertEqual(vm.liveActivityManager.activeActivityType, .alarmCountdown)
        XCTAssertEqual(vm.entities.first(where: { $0.id == "alarm_control_panel.home" })?.state, "armed_away")
        XCTAssertEqual(entityRepo.lastServiceCall?.domain, "alarm_control_panel")
        XCTAssertEqual(entityRepo.lastServiceCall?.service, "alarm_arm_away")

        // 2. Disarmo allarme
        vm.disarmAlarm(id: "alarm_control_panel.home", code: "9988")
        try? await Task.sleep(nanoseconds: 50_000_000)

        // Verifica che Live Activity sia terminata
        XCTAssertFalse(vm.liveActivityManager.isActivityActive)
        XCTAssertEqual(vm.entities.first(where: { $0.id == "alarm_control_panel.home" })?.state, "disarmed")
        XCTAssertEqual(entityRepo.lastServiceCall?.service, "alarm_disarm")
    }

    // MARK: - 4. Background Maintenance -> Cache Sync Integration

    func testBackgroundMaintenanceIntegrationWithCacheSync() async {
        let entityRepo = IntegrationMockEntityRepository()
        entityRepo.entities = [
            Entity(id: "light.hallway", name: "Luce Ingresso", domain: "light", state: "on")
        ]
        let connRepo = IntegrationMockConnectionRepository()
        let vm = createTestViewModel(entityRepo: entityRepo, connRepo: connRepo)

        let summary = await vm.backgroundTaskManager.executeMaintenanceNow()

        XCTAssertTrue(summary.contains("Cache sincronizzata"))
        XCTAssertNotNil(vm.backgroundTaskManager.lastMaintenanceDate)
        XCTAssertEqual(vm.backgroundTaskManager.lastMaintenanceSummary, summary)
    }

    // MARK: - 5. Kiosk Battery Protection -> Smart Plug Command Integration

    func testKioskBatteryProtectionIntegrationWithSmartPlug() async {
        let entityRepo = IntegrationMockEntityRepository()
        let connRepo = IntegrationMockConnectionRepository()
        let vm = createTestViewModel(entityRepo: entityRepo, connRepo: connRepo)

        vm.kioskManager.smartPlugEntityId = "switch.tablet_power"

        // 1. Batteria raggiunge soglia massima (85% > 80%) -> Stacco alimentazione
        vm.kioskManager.onSmartPlugCommand?("switch.tablet_power", false)
        try? await Task.sleep(nanoseconds: 50_000_000)

        XCTAssertEqual(entityRepo.lastServiceCall?.domain, "switch")
        XCTAssertEqual(entityRepo.lastServiceCall?.service, "turn_off")
        XCTAssertEqual(entityRepo.lastServiceCall?.entityId, "switch.tablet_power")

        // 2. Batteria scende sotto soglia minima (15% < 20%) -> Riattacco alimentazione
        vm.kioskManager.onSmartPlugCommand?("switch.tablet_power", true)
        try? await Task.sleep(nanoseconds: 50_000_000)

        XCTAssertEqual(entityRepo.lastServiceCall?.service, "turn_on")
        XCTAssertEqual(entityRepo.lastServiceCall?.entityId, "switch.tablet_power")
    }

    // MARK: - 6. NFC Tag Scanning & Watch Sessions Integration

    func testNfcTagScanningAndWatchCommandsIntegration() async {
        let entityRepo = IntegrationMockEntityRepository()
        let light = Entity(id: "light.desk", name: "Luce Scrivania", domain: "light", state: "off")
        entityRepo.entities = [light]
        let connRepo = IntegrationMockConnectionRepository()
        let vm = createTestViewModel(entityRepo: entityRepo, connRepo: connRepo)
        vm.entities = entityRepo.entities

        // 1. Scansione NFC Tag "toggle"
        let tag = DomusNfcTag(id: "tag_1", uri: "domus://tag/tag_1", actionType: "toggle", targetId: "light.desk")
        vm.nfcManager.onTagScanned?(tag)
        try? await Task.sleep(nanoseconds: 50_000_000)

        XCTAssertEqual(entityRepo.toggleCallCount, 1)

        // 2. Ricezione comando toggle da Apple Watch
        vm.watchSessionManager.onWatchCommandReceived?("toggle", "light.desk")
        try? await Task.sleep(nanoseconds: 50_000_000)

        XCTAssertEqual(entityRepo.toggleCallCount, 2)
    }

    // MARK: - 7. Drag-and-Drop Card Reordering & Filter Sorting Integration

    func testDashboardCardReorderingAndFilteringIntegration() {
        UserDefaults.standard.removeObject(forKey: "domus_custom_card_order")
        let entityRepo = IntegrationMockEntityRepository()
        let connRepo = IntegrationMockConnectionRepository()
        let vm = createTestViewModel(entityRepo: entityRepo, connRepo: connRepo)
        vm.customEntityOrder = []

        vm.entities = [
            Entity(id: "e1", name: "Alfa", domain: "light", state: "on"),
            Entity(id: "e2", name: "Beta", domain: "switch", state: "off"),
            Entity(id: "e3", name: "Gamma", domain: "climate", state: "heat")
        ]

        // Spostamento del primo elemento all'ultima posizione
        vm.moveEntity(from: 0, to: 2)

        let ordered = vm.filteredEntities.map { $0.id }
        XCTAssertEqual(ordered, ["e2", "e3", "e1"])
        XCTAssertEqual(vm.customEntityOrder, ["e2", "e3", "e1"])
        UserDefaults.standard.removeObject(forKey: "domus_custom_card_order")
    }
}
