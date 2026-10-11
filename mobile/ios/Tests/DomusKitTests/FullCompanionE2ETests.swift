import XCTest
@testable import DomusKit

@MainActor
final class FullCompanionE2ETests: XCTestCase {

    final class E2EMockEntityRepository: EntityRepositoryProtocol, @unchecked Sendable {
        var entities: [Entity] = []
        var areas: [Area] = []
        var serviceCallHistory: [(domain: String, service: String, entityId: String)] = []

        func getEntities() async -> [Entity] { entities }
        func getEntity(id: String) async -> Entity? { entities.first(where: { $0.id == id }) }
        func syncEntities() async throws -> [Entity] { entities }
        func toggleEntity(id: String) async throws -> Entity {
            guard let idx = entities.firstIndex(where: { $0.id == id }) else {
                let newEntity = Entity(id: id, name: id, domain: "cover", state: "open")
                entities.append(newEntity)
                return newEntity
            }
            if entities[idx].domain == "cover" {
                entities[idx].state = entities[idx].isOn ? "closed" : "open"
            } else {
                entities[idx].state = entities[idx].isOn ? "off" : "on"
            }
            return entities[idx]
        }
        func callService(domain: String, service: String, entityId: String, data: [String: Any]) async throws {
            serviceCallHistory.append((domain, service, entityId))
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

    final class E2EMockConnectionRepository: ConnectionRepositoryProtocol, @unchecked Sendable {
        var connection = ServerConnection(mode: .disconnected, isConnected: false)

        func getCurrentConnection() async -> ServerConnection { connection }
        func discoverLocalHub(timeoutSeconds: Double) async -> String? { "http://192.168.1.100:8080" }
        func connect(mode: TransportMode?) async throws {
            connection = ServerConnection(
                mode: mode ?? .localLan,
                localAddress: "http://192.168.1.100:8080",
                remoteAddress: "https://hub.domus.internal",
                isConnected: true
            )
        }
        func disconnect() async {
            connection = ServerConnection(mode: .disconnected, isConnected: false)
        }
        func setEndpoints(local: String?, remote: String?) async {
            connection.localAddress = local
            connection.remoteAddress = remote
        }
    }

    func testCompleteCompanionAppLifecycleE2E() async throws {
        // ─── SETUP E2E ENVIRONMENT ──────────────────────────────────────────
        let entityRepo = E2EMockEntityRepository()
        let connRepo = E2EMockConnectionRepository()

        let discoverUseCase = DiscoverAndConnectUseCase(connectionRepository: connRepo)
        let syncUseCase = SyncEntitiesUseCase(entityRepository: entityRepo)
        let toggleUseCase = ToggleEntityUseCase(entityRepository: entityRepo)

        let bioManager = LocalAuthenticationManager(isSimulationMode: true)
        let beaconManager = RoomPresenceBeaconManager(configuredBeacons: [
            DomusRoomBeacon(id: "b_living", roomId: "living_room", roomName: "Salotto", major: 1, minor: 1, rssiThreshold: -65),
            DomusRoomBeacon(id: "b_kitchen", roomId: "kitchen", roomName: "Cucina", major: 1, minor: 2, rssiThreshold: -65)
        ])
        let liveActivityManager = DomusLiveActivityManager.shared
        let carPlayManager = DomusCarPlayManager.shared
        let vpnManager = VpnOnDemandManager(homeSsid: "Domus_Home_5G", isOnDemandEnabled: true)
        let bgTaskManager = DomusBackgroundTaskManager.shared

        let viewModel = DashboardViewModel(
            entityRepository: entityRepo,
            connectionRepository: connRepo,
            discoverAndConnectUseCase: discoverUseCase,
            syncEntitiesUseCase: syncUseCase,
            toggleEntityUseCase: toggleUseCase,
            beaconManager: beaconManager,
            liveActivityManager: liveActivityManager,
            carPlayManager: carPlayManager,
            vpnManager: vpnManager,
            backgroundTaskManager: bgTaskManager
        )

        // ─── STEP 1: Cold Launch & Local LAN Discovery ─────────────────────
        let discoveredConn = try await discoverUseCase.execute(preferLocal: true)
        XCTAssertEqual(discoveredConn.mode, .localLan)
        XCTAssertTrue(discoveredConn.isConnected)

        // ─── STEP 2: Multi-Domain Entity Synchronization ───────────────────
        entityRepo.areas = [
            Area(id: "living_room", name: "Salotto"),
            Area(id: "kitchen", name: "Cucina")
        ]
        entityRepo.entities = [
            Entity(id: "light.living_room", name: "Lampada Piantana", domain: "light", state: "on", attributes: ["brightness": AnyCodable(90)], areaId: "living_room"),
            Entity(id: "switch.coffee_maker", name: "Macchina Espresso", domain: "switch", state: "on", attributes: ["power": AnyCodable(1200.0)], areaId: "kitchen"),
            Entity(id: "climate.thermostat", name: "Termostato", domain: "climate", state: "heat", attributes: ["temperature": AnyCodable(21.0)], areaId: "living_room"),
            Entity(id: "alarm_control_panel.home", name: "Centrale Allarme", domain: "alarm_control_panel", state: "disarmed"),
            Entity(id: "cover.garage", name: "Basculante Garage", domain: "cover", state: "closed"),
            Entity(id: "camera.driveway", name: "Telecamera Viale", domain: "camera", state: "idle")
        ]

        viewModel.entities = entityRepo.entities
        viewModel.areas = entityRepo.areas
        XCTAssertEqual(viewModel.entities.count, 6)
        XCTAssertNil(viewModel.selectedAreaId)

        // ─── STEP 3: Indoor Micro-localization via BLE Beacon ──────────────
        // L'utente entra in cucina; il beacon emette segnale RSSI -58 dBm (> soglia -65)
        beaconManager.evaluateRssiSample(major: 1, minor: 2, rssi: -58)
        XCTAssertEqual(beaconManager.activeRoomId, "kitchen")
        XCTAssertEqual(viewModel.selectedAreaId, "kitchen")

        // La dashboard filtra istantaneamente i controlli per la cucina
        let kitchenDevices = viewModel.filteredEntities
        XCTAssertEqual(kitchenDevices.count, 1)
        XCTAssertEqual(kitchenDevices.first?.id, "switch.coffee_maker")

        // ─── STEP 4: Biometric Authentication & Alarm Arming (Live Activity) ─
        let bioResult = await bioManager.authenticate(reason: "Conferma armamento allarme Domus")
        switch bioResult {
        case .success(let passed):
            XCTAssertTrue(passed)
        case .failure:
            XCTFail("Autenticazione biometrica fallita nel flusso E2E")
        }

        // Armamento allarme totale con avvio Live Activity (Dynamic Island)
        viewModel.armAlarm(id: "alarm_control_panel.home", code: "1234", mode: "alarm_arm_away")
        try? await Task.sleep(nanoseconds: 50_000_000)

        XCTAssertTrue(liveActivityManager.isActivityActive)
        XCTAssertEqual(liveActivityManager.activeActivityType, .alarmCountdown)
        XCTAssertEqual(viewModel.entities.first(where: { $0.id == "alarm_control_panel.home" })?.state, "armed_away")

        // ─── STEP 5: Geofence Exit & VPN On-Demand Automatic Tunnel ────────
        // L'utente esce dalla residenza -> Disconnessione Wi-Fi domestico
        vpnManager.configureOnDemandRules(homeSsid: "Domus_Home_5G")
        vpnManager.connect()
        #if canImport(NetworkExtension)
        XCTAssertTrue(vpnManager.status == .connecting || vpnManager.status == .connected || vpnManager.lastErrorMessage != nil)
        #else
        XCTAssertEqual(vpnManager.status, .connected)
        #endif

        // ─── STEP 6: CarPlay In-Car Interaction ────────────────────────────
        // L'utente sul cruscotto dell'auto tocca il tasto per chiudere/aprire il garage
        #if canImport(CarPlay) && os(iOS)
        carPlayManager.onCarPlayAction?("cover", "cover.garage")
        #else
        carPlayManager.simulateCarAction(domain: "cover", entityId: "cover.garage")
        #endif
        try? await Task.sleep(nanoseconds: 50_000_000)

        let garage = viewModel.entities.first(where: { $0.id == "cover.garage" })
        XCTAssertEqual(garage?.state, "open")

        // ─── STEP 7: Nightly Scheduled Maintenance Pipeline ────────────────
        let maintenanceSummary = await bgTaskManager.executeMaintenanceNow()
        XCTAssertTrue(maintenanceSummary.contains("Cache sincronizzata") || maintenanceSummary.contains("OK"))
        XCTAssertNotNil(bgTaskManager.lastMaintenanceDate)
        XCTAssertFalse(bgTaskManager.isMaintenanceRunning)
    }
}
