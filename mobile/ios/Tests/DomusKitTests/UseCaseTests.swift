import XCTest
@testable import DomusKit

final class DiscoverAndConnectUseCaseTests: XCTestCase {

    final class MockConnectionRepo: ConnectionRepositoryProtocol, @unchecked Sendable {
        var connectedMode: TransportMode?
        let shouldDiscover: Bool

        init(shouldDiscover: Bool) {
            self.shouldDiscover = shouldDiscover
        }

        func getCurrentConnection() async -> ServerConnection {
            ServerConnection(mode: connectedMode ?? .disconnected, isConnected: true)
        }

        func discoverLocalHub(timeoutSeconds: Double) async -> String? {
            shouldDiscover ? "http://192.168.1.100:8080" : nil
        }

        func connect(mode: TransportMode?) async throws {
            self.connectedMode = mode
        }

        func disconnect() async {}
        func setEndpoints(local: String?, remote: String?) async {}
    }

    func testPrefersLocalWhenDiscovered() async throws {
        let repo = MockConnectionRepo(shouldDiscover: true)
        let useCase = DiscoverAndConnectUseCase(connectionRepository: repo)

        let conn = try await useCase.execute(preferLocal: true)
        XCTAssertEqual(conn.mode, .localLan)
        XCTAssertEqual(repo.connectedMode, .localLan)
    }

    func testFallsBackToRemoteWhenLocalUnavailable() async throws {
        let repo = MockConnectionRepo(shouldDiscover: false)
        let useCase = DiscoverAndConnectUseCase(connectionRepository: repo)

        let conn = try await useCase.execute(preferLocal: true)
        XCTAssertEqual(conn.mode, .remoteTls)
        XCTAssertEqual(repo.connectedMode, .remoteTls)
    }
}

final class ToggleEntityUseCaseTests: XCTestCase {

    final class MockEntityRepo: EntityRepositoryProtocol, @unchecked Sendable {
        var toggledId: String?

        func getEntities() async -> [Entity] { [] }
        func getEntity(id: String) async -> Entity? { nil }
        func syncEntities() async throws -> [Entity] { [] }
        func toggleEntity(id: String) async throws -> Entity {
            self.toggledId = id
            return Entity(id: id, name: "Luce", domain: "light", state: "on")
        }
        func callService(domain: String, service: String, entityId: String, data: [String : Any]) async throws {}
        func updateLocalEntity(_ entity: Entity) async {}
        func getAreas() async -> [Area] { [] }
    }

    func testToggleDispatchesId() async throws {
        let repo = MockEntityRepo()
        let useCase = ToggleEntityUseCase(entityRepository: repo)

        let updated = try await useCase.execute(entityId: "light.balcony")
        XCTAssertEqual(repo.toggledId, "light.balcony")
        XCTAssertEqual(updated.state, "on")
    }
}
