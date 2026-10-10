import Foundation

public struct DiscoverAndConnectUseCase: Sendable {
    private let connectionRepository: ConnectionRepositoryProtocol

    public init(connectionRepository: ConnectionRepositoryProtocol) {
        self.connectionRepository = connectionRepository
    }

    public func execute(preferLocal: Bool = true, timeoutSeconds: Double = 1.5) async throws -> ServerConnection {
        if preferLocal {
            if let _ = await connectionRepository.discoverLocalHub(timeoutSeconds: timeoutSeconds) {
                do {
                    try await connectionRepository.connect(mode: .localLan)
                    return await connectionRepository.getCurrentConnection()
                } catch {
                    // Fall through to remote
                }
            }
        }

        // Fallback to Remote
        try await connectionRepository.connect(mode: .remoteTls)
        return await connectionRepository.getCurrentConnection()
    }
}

public struct SyncEntitiesUseCase: Sendable {
    private let entityRepository: EntityRepositoryProtocol

    public init(entityRepository: EntityRepositoryProtocol) {
        self.entityRepository = entityRepository
    }

    public func execute() async throws -> [Entity] {
        return try await entityRepository.syncEntities()
    }
}

public struct ToggleEntityUseCase: Sendable {
    private let entityRepository: EntityRepositoryProtocol

    public init(entityRepository: EntityRepositoryProtocol) {
        self.entityRepository = entityRepository
    }

    public func execute(entityId: String) async throws -> Entity {
        return try await entityRepository.toggleEntity(id: entityId)
    }
}

public struct AuthenticateUseCase: Sendable {
    private let authRepository: AuthRepositoryProtocol

    public init(authRepository: AuthRepositoryProtocol) {
        self.authRepository = authRepository
    }

    public func login(username: String, password: String) async throws -> AuthSession {
        return try await authRepository.login(username: username, password: password)
    }

    public func logout() async {
        await authRepository.logout()
    }

    public func checkSession() async -> AuthSession {
        return await authRepository.restoreSession()
    }
}
