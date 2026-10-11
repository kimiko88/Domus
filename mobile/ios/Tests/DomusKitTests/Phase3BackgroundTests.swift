import XCTest
@testable import DomusKit

@MainActor
final class Phase3BackgroundTests: XCTestCase {

    final class MockEntityRepository: EntityRepositoryProtocol, @unchecked Sendable {
        var entities: [Entity] = []
        var lastServiceCall: (domain: String, service: String, entityId: String, data: [String: Any])?

        func getEntities() async -> [Entity] { entities }
        func getEntity(id: String) async -> Entity? { entities.first(where: { $0.id == id }) }
        func syncEntities() async throws -> [Entity] { entities }
        func toggleEntity(id: String) async throws -> Entity {
            guard let idx = entities.firstIndex(where: { $0.id == id }) else {
                fatalError("Entity not found")
            }
            entities[idx].state = entities[idx].isOn ? "off" : "on"
            return entities[idx]
        }
        func callService(domain: String, service: String, entityId: String, data: [String: Any]) async throws {
            lastServiceCall = (domain, service, entityId, data)
        }
        func updateLocalEntity(_ entity: Entity) async {}
        func getAreas() async -> [Area] { [] }
    }

    final class MockConnectionRepository: ConnectionRepositoryProtocol, @unchecked Sendable {
        func getCurrentConnection() async -> ServerConnection { ServerConnection() }
        func discoverLocalHub(timeoutSeconds: Double) async -> String? { "http://192.168.1.100:8080" }
        func connect(mode: TransportMode?) async throws {}
        func disconnect() async {}
        func setEndpoints(local: String?, remote: String?) async {}
    }

    func testHomePerimeterConfig() {
        var config = HomePerimeterConfig(latitude: 45.4642, longitude: 9.1900, radiusMeters: 200.0)
        XCTAssertEqual(config.latitude, 45.4642)
        XCTAssertEqual(config.longitude, 9.1900)
        XCTAssertEqual(config.radiusMeters, 200.0)

        config.radiusMeters = 250.0
        XCTAssertEqual(config.radiusMeters, 250.0)

        XCTAssertEqual(PresenceState.home.rawValue, "home")
        XCTAssertEqual(PresenceState.away.rawValue, "away")
    }

    func testNotificationCategoriesAndActions() {
        XCTAssertEqual(DomusNotificationCategory.alarmTriggered.rawValue, "ALARM_TRIGGERED")
        XCTAssertEqual(DomusNotificationCategory.windowRainAlert.rawValue, "WINDOW_RAIN_ALERT")
        XCTAssertEqual(DomusNotificationAction.disarmAlarm.rawValue, "ACTION_DISARM_ALARM")
        XCTAssertEqual(DomusNotificationAction.closeCovers.rawValue, "ACTION_CLOSE_COVERS")
    }

    func testNfcTagUriParsingAndSimulation() {
        let nfc = NfcTagManager()
        var scannedTag: DomusNfcTag?

        nfc.onTagScanned = { tag in
            scannedTag = tag
        }

        // Test simulazione tag per scena
        nfc.simulateTagScan(uri: "domus://scene/cinema_mode")

        XCTAssertNotNil(scannedTag)
        XCTAssertEqual(scannedTag?.actionType, "scene")
        XCTAssertEqual(scannedTag?.targetId, "cinema_mode")
        XCTAssertEqual(scannedTag?.uri, "domus://scene/cinema_mode")

        // Test simulazione tag per toggle dispositivo
        nfc.simulateTagScan(uri: "domus://toggle/light.kitchen")
        XCTAssertEqual(scannedTag?.actionType, "toggle")
        XCTAssertEqual(scannedTag?.targetId, "light.kitchen")
    }

    func testNfcTagExecutionTriggersViewModel() async {
        let repo = MockEntityRepository()
        let light = Entity(id: "light.hallway", name: "Luce Ingresso", domain: "light", state: "off")
        repo.entities = [light]

        let connRepo = MockConnectionRepository()
        let nfc = NfcTagManager()
        let vm = DashboardViewModel(
            entityRepository: repo,
            connectionRepository: connRepo,
            discoverAndConnectUseCase: DiscoverAndConnectUseCase(connectionRepository: connRepo),
            syncEntitiesUseCase: SyncEntitiesUseCase(entityRepository: repo),
            toggleEntityUseCase: ToggleEntityUseCase(entityRepository: repo),
            nfcManager: nfc
        )
        vm.entities = [light]

        // Simula lettura tag associato a switch o scena
        nfc.simulateTagScan(uri: "domus://scene/relax_night")

        // Attendi dispatch asincrono
        try? await Task.sleep(nanoseconds: 50_000_000)

        XCTAssertEqual(repo.lastServiceCall?.domain, "scene")
        XCTAssertEqual(repo.lastServiceCall?.service, "turn_on")
        XCTAssertEqual(repo.lastServiceCall?.entityId, "relax_night")
    }
}
