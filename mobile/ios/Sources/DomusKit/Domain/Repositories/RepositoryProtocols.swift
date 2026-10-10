import Foundation

public protocol EntityRepositoryProtocol: Sendable {
    func getEntities() async -> [Entity]
    func getEntity(id: String) async -> Entity?
    func syncEntities() async throws -> [Entity]
    func toggleEntity(id: String) async throws -> Entity
    func callService(domain: String, service: String, entityId: String, data: [String: Any]) async throws
    func updateLocalEntity(_ entity: Entity) async
    func getAreas() async -> [Area]
}

public protocol ConnectionRepositoryProtocol: Sendable {
    func getCurrentConnection() async -> ServerConnection
    func discoverLocalHub(timeoutSeconds: Double) async -> String?
    func connect(mode: TransportMode?) async throws
    func disconnect() async
    func setEndpoints(local: String?, remote: String?) async
}

public protocol AuthRepositoryProtocol: Sendable {
    func getSession() async -> AuthSession
    func login(username: String, password: String) async throws -> AuthSession
    func logout() async
    func getSavedToken() async -> String?
    func restoreSession() async -> AuthSession
}
