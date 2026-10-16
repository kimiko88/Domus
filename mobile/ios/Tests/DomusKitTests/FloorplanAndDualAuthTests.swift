import XCTest
@testable import DomusKit

final class FloorplanAndDualAuthTests: XCTestCase {

    func testFloorAndMarkerDTOCodable() throws {
        let floorJson = """
        {
            "id": "floor_ground",
            "name": "Piano Terra",
            "level": 0,
            "plan_svg": "<svg></svg>",
            "sort_order": 1
        }
        """
        let decoder = JSONDecoder()
        let floor = try decoder.decode(FloorDTO.self, from: floorJson.data(using: .utf8)!)
        XCTAssertEqual(floor.id, "floor_ground")
        XCTAssertEqual(floor.name, "Piano Terra")
        XCTAssertEqual(floor.level, 0)
        XCTAssertEqual(floor.sortOrder, 1)

        let markerJson = """
        {
            "id": "marker_1",
            "floor_id": "floor_ground",
            "entity_key": "light.living_room_ceiling",
            "x_percent": 45.5,
            "y_percent": 60.2,
            "icon_override": "lightbulb.fill",
            "show_badge": true
        }
        """
        let marker = try decoder.decode(FloorMarkerDTO.self, from: markerJson.data(using: .utf8)!)
        XCTAssertEqual(marker.id, "marker_1")
        XCTAssertEqual(marker.floorId, "floor_ground")
        XCTAssertEqual(marker.entityKey, "light.living_room_ceiling")
        XCTAssertEqual(marker.xPercent, 45.5)
        XCTAssertEqual(marker.yPercent, 60.2)
        XCTAssertTrue(marker.showBadge)
    }

    func testTransportModeDisplayNames() {
        XCTAssertEqual(TransportMode.localLan.displayName, "LAN Locale")
        XCTAssertEqual(TransportMode.remoteTls.displayName, "TLS Remoto (Fuori Casa)")
        XCTAssertEqual(TransportMode.connecting.displayName, "Connessione...")
        XCTAssertEqual(TransportMode.disconnected.displayName, "Disconnesso")
    }

    func testFloorplanRoomHitTesting() {
        let room = FloorplanRoom(
            id: "studio",
            name: "Studio",
            rect: CGRect(x: 0.2, y: 0.2, width: 0.4, height: 0.4),
            lightEntityId: "light.studio",
            climateEntityId: "climate.studio"
        )

        XCTAssertTrue(room.rect.contains(CGPoint(x: 0.3, y: 0.3)))
        XCTAssertFalse(room.rect.contains(CGPoint(x: 0.1, y: 0.1)))
        XCTAssertEqual(room.lightEntityId, "light.studio")
    }

    func testViewModelAuthStates() async {
        let entityRepo = MockEntityRepo()
        let connRepo = MockConnRepo()
        let discoverUseCase = DiscoverAndConnectUseCase(connectionRepository: connRepo)
        let syncUseCase = SyncEntitiesUseCase(entityRepository: entityRepo)
        let toggleUseCase = ToggleEntityUseCase(entityRepository: entityRepo)
        let authRepo = MockAuthRepo()

        let vm = await MainActor.run {
            DashboardViewModel(
                entityRepository: entityRepo,
                connectionRepository: connRepo,
                discoverAndConnectUseCase: discoverUseCase,
                syncEntitiesUseCase: syncUseCase,
                toggleEntityUseCase: toggleUseCase,
                authRepository: authRepo
            )
        }

        let isAuthedBefore = await MainActor.run { vm.authSession.isAuthenticated }
        XCTAssertFalse(isAuthedBefore)

        let loginSuccess = await vm.login(
            username: "admin",
            password: "password123",
            localEndpoint: "http://192.168.1.100:8080",
            remoteEndpoint: "https://myhome.domus.net:8443"
        )
        XCTAssertTrue(loginSuccess)

        let isAuthedAfter = await MainActor.run { vm.authSession.isAuthenticated }
        let usernameAfter = await MainActor.run { vm.authSession.username }
        XCTAssertTrue(isAuthedAfter)
        XCTAssertEqual(usernameAfter, "admin")

        await MainActor.run { vm.logout() }
        // Attendi che il task asincrono di logout venga processato
        try? await Task.sleep(nanoseconds: 50_000_000)
        let isAuthedLoggedOut = await MainActor.run { vm.authSession.isAuthenticated }
        XCTAssertFalse(isAuthedLoggedOut)
    }
}

// MARK: - Mocks for Testing Isolation

private actor MockEntityRepo: EntityRepositoryProtocol {
    func getEntities() async -> [Entity] { [] }
    func getEntity(id: String) async -> Entity? { nil }
    func syncEntities() async throws -> [Entity] { [] }
    func toggleEntity(id: String) async throws -> Entity {
        Entity(id: id, name: "Test", domain: "switch", state: "on")
    }
    func callService(domain: String, service: String, entityId: String, data: [String : Any]) async throws {}
    func updateLocalEntity(_ entity: Entity) async {}
    func getAreas() async -> [Area] { [] }
}

private actor MockConnRepo: ConnectionRepositoryProtocol {
    private var conn = ServerConnection()
    func getCurrentConnection() async -> ServerConnection { conn }
    func discoverLocalHub(timeoutSeconds: Double) async -> String? { "http://192.168.1.100:8080" }
    func connect(mode: TransportMode?) async throws {
        conn.isConnected = true
        conn.mode = mode ?? .localLan
    }
    func disconnect() async {
        conn.isConnected = false
        conn.mode = .disconnected
    }
    func setEndpoints(local: String?, remote: String?) async {
        conn.localAddress = local
        conn.remoteAddress = remote
    }
}

private actor MockAuthRepo: AuthRepositoryProtocol {
    private var session = AuthSession()

    func getSession() async -> AuthSession { session }
    func login(username: String, password: String) async throws -> AuthSession {
        session = AuthSession(token: "test_token", username: username, role: "admin", isAuthenticated: true)
        return session
    }
    func login(username: String, password: String, localEndpoint: String?, remoteEndpoint: String?) async throws -> AuthSession {
        session = AuthSession(token: "test_token", username: username, role: "admin", isAuthenticated: true)
        return session
    }
    func logout() async {
        session = AuthSession()
    }
    func getSavedToken() async -> String? { session.token }
    func restoreSession() async -> AuthSession { session }
}
