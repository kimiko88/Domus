import Foundation

public final class DefaultEntityRepository: EntityRepositoryProtocol, @unchecked Sendable {
    private let httpClient: DomusHttpClient
    private let webSocketClient: DualTransportWebSocketClient
    private let cache: OfflineEntityCache
    private let tokenStorage: any TokenStorageProtocol

    public init(
        httpClient: DomusHttpClient,
        webSocketClient: DualTransportWebSocketClient,
        cache: OfflineEntityCache,
        tokenStorage: any TokenStorageProtocol
    ) {
        self.httpClient = httpClient
        self.webSocketClient = webSocketClient
        self.cache = cache
        self.tokenStorage = tokenStorage

        Task {
            await webSocketClient.setOnEventCallback { [weak self] snapshot in
                guard let self = self else { return }
                Task {
                    let existing = await self.cache.getEntity(id: snapshot.entityId)
                    let updated: Entity
                    if let ex = existing {
                        updated = Entity(
                            id: snapshot.entityId,
                            name: ex.name,
                            domain: ex.domain,
                            state: snapshot.state,
                            attributes: snapshot.attributes,
                            areaId: ex.areaId,
                            lastChanged: snapshot.lastChanged,
                            lastUpdated: snapshot.lastUpdated
                        )
                    } else {
                        let domain = snapshot.entityId.components(separatedBy: ".").first ?? "switch"
                        let name = snapshot.entityId.components(separatedBy: ".").last ?? snapshot.entityId
                        updated = Entity(
                            id: snapshot.entityId,
                            name: name,
                            domain: domain,
                            state: snapshot.state,
                            attributes: snapshot.attributes,
                            areaId: nil,
                            lastChanged: snapshot.lastChanged,
                            lastUpdated: snapshot.lastUpdated
                        )
                    }
                    await self.cache.upsertEntity(updated)
                }
            }
        }
    }

    public func getEntities() async -> [Entity] {
        await cache.getAllEntities()
    }

    public func getEntity(id: String) async -> Entity? {
        await cache.getEntity(id: id)
    }

    public func syncEntities() async throws -> [Entity] {
        let conn = await webSocketClient.getCurrentConnection()
        let baseUrl = conn.activeUrl ?? (await tokenStorage.getLocalEndpoint()) ?? "http://192.168.1.100:8080"
        let token = await tokenStorage.getToken()

        do {
            let entities = try await httpClient.fetchEntities(baseUrl: baseUrl, token: token)
            await cache.updateEntities(entities)

            let areas = await httpClient.fetchAreas(baseUrl: baseUrl, token: token)
            await cache.updateAreas(areas)

            return entities
        } catch {
            let cached = await cache.getAllEntities()
            if !cached.isEmpty {
                return cached
            }
            throw error
        }
    }

    public func toggleEntity(id: String) async throws -> Entity {
        guard let entity = await cache.getEntity(id: id) else {
            throw NSError(domain: "Domus", code: 404, userInfo: [NSLocalizedDescriptionKey: "Entità non trovata"])
        }

        let newState = entity.isOn ? "off" : "on"
        let service = entity.isOn ? "turn_off" : "turn_on"

        var optimistic = entity
        optimistic.state = newState
        await cache.upsertEntity(optimistic)

        let conn = await webSocketClient.getCurrentConnection()
        if conn.isConnected {
            _ = await webSocketClient.sendCommand(domain: entity.domain, service: service, entityId: id)
            return optimistic
        }

        let baseUrl = conn.activeUrl ?? (await tokenStorage.getLocalEndpoint()) ?? "http://192.168.1.100:8080"
        let token = await tokenStorage.getToken()

        do {
            try await httpClient.callService(baseUrl: baseUrl, token: token, domain: entity.domain, service: service, entityId: id)
            return optimistic
        } catch {
            await cache.upsertEntity(entity)
            throw error
        }
    }

    public func callService(domain: String, service: String, entityId: String, data: [String: Any]) async throws {
        let conn = await webSocketClient.getCurrentConnection()
        if conn.isConnected {
            _ = await webSocketClient.sendCommand(domain: domain, service: service, entityId: entityId)
            return
        }

        let baseUrl = conn.activeUrl ?? (await tokenStorage.getLocalEndpoint()) ?? "http://192.168.1.100:8080"
        let token = await tokenStorage.getToken()
        try await httpClient.callService(baseUrl: baseUrl, token: token, domain: domain, service: service, entityId: entityId)
    }

    public func updateLocalEntity(_ entity: Entity) async {
        await cache.upsertEntity(entity)
    }

    public func getAreas() async -> [Area] {
        await cache.getAllAreas()
    }
}

public final class DefaultConnectionRepository: ConnectionRepositoryProtocol, @unchecked Sendable {
    private let mdnsEngine: MdnsDiscoveryEngine
    private let webSocketClient: DualTransportWebSocketClient
    private let tokenStorage: any TokenStorageProtocol

    public init(
        mdnsEngine: MdnsDiscoveryEngine,
        webSocketClient: DualTransportWebSocketClient,
        tokenStorage: any TokenStorageProtocol
    ) {
        self.mdnsEngine = mdnsEngine
        self.webSocketClient = webSocketClient
        self.tokenStorage = tokenStorage
    }

    public func getCurrentConnection() async -> ServerConnection {
        await webSocketClient.getCurrentConnection()
    }

    public func discoverLocalHub(timeoutSeconds: Double) async -> String? {
        if let hub = await mdnsEngine.discoverLocalHub(timeoutSeconds: timeoutSeconds) {
            await tokenStorage.saveLocalEndpoint(hub.httpBaseUrl)
            return hub.httpBaseUrl
        }
        return nil
    }

    public func connect(mode: TransportMode?) async throws {
        let target = mode ?? .localLan
        let token = await tokenStorage.getToken()

        let endpoint: String
        switch target {
        case .localLan:
            endpoint = (await tokenStorage.getLocalEndpoint()) ?? "http://192.168.1.100:8080"
        case .remoteTls:
            endpoint = (await tokenStorage.getRemoteEndpoint()) ?? "https://domus.local.myhome.net"
        default:
            endpoint = (await tokenStorage.getLocalEndpoint()) ?? "http://192.168.1.100:8080"
        }

        await webSocketClient.connect(mode: target, endpointUrl: endpoint, token: token)
    }

    public func disconnect() async {
        await webSocketClient.disconnect()
    }

    public func setEndpoints(local: String?, remote: String?) async {
        if let local = local { await tokenStorage.saveLocalEndpoint(local) }
        if let remote = remote { await tokenStorage.saveRemoteEndpoint(remote) }
    }
}

public final class DefaultAuthRepository: AuthRepositoryProtocol, @unchecked Sendable {
    private let httpClient: DomusHttpClient
    private let tokenStorage: any TokenStorageProtocol
    private let connectionRepository: ConnectionRepositoryProtocol
    private var session: AuthSession = AuthSession()

    public init(
        httpClient: DomusHttpClient,
        tokenStorage: any TokenStorageProtocol,
        connectionRepository: ConnectionRepositoryProtocol
    ) {
        self.httpClient = httpClient
        self.tokenStorage = tokenStorage
        self.connectionRepository = connectionRepository
    }

    public func getSession() async -> AuthSession {
        return session
    }

    public func login(username: String, password: String) async throws -> AuthSession {
        let conn = await connectionRepository.getCurrentConnection()
        let baseUrl = conn.activeUrl ?? (await tokenStorage.getLocalEndpoint()) ?? "http://192.168.1.100:8080"

        let loggedSession = try await httpClient.login(baseUrl: baseUrl, username: username, password: password)
        await tokenStorage.saveToken(loggedSession.token)
        self.session = loggedSession
        try? await connectionRepository.connect(mode: conn.mode)
        return loggedSession
    }

    public func logout() async {
        await tokenStorage.clear()
        self.session = AuthSession()
        await connectionRepository.disconnect()
    }

    public func getSavedToken() async -> String? {
        await tokenStorage.getToken()
    }

    public func restoreSession() async -> AuthSession {
        if let token = await tokenStorage.getToken(), !token.isEmpty {
            self.session = AuthSession(token: token, username: "admin", role: "admin", isAuthenticated: true)
        } else {
            self.session = AuthSession()
        }
        return self.session
    }
}
