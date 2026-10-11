import Foundation

// MARK: - DTO (Data Transfer Objects)

private struct EntityDTO: Decodable {
    let entityId: String
    let name: String
    let domain: String
    let state: String
    let attributes: [String: AnyCodable]
    let areaId: String?
    let lastChanged: String
    let lastUpdated: String

    enum CodingKeys: String, CodingKey {
        case entityId = "entity_id"
        case name = "friendly_name"
        case domain
        case state
        case attributes
        case areaId = "area_id"
        case lastChanged = "last_changed"
        case lastUpdated = "last_updated"
    }

    func toDomain() -> Entity {
        Entity(
            id: entityId,
            name: name,
            domain: domain,
            state: state,
            attributes: attributes,
            areaId: areaId,
            lastChanged: lastChanged,
            lastUpdated: lastUpdated
        )
    }
}

private struct AreaDTO: Decodable {
    let areaId: String
    let name: String
    let icon: String?

    enum CodingKeys: String, CodingKey {
        case areaId = "area_id"
        case name
        case icon
    }

    func toDomain() -> Area {
        Area(id: areaId, name: name, icon: icon ?? "home")
    }
}

private struct ToggleRequestDTO: Encodable {
    let entityId: String
    let domain: String

    enum CodingKeys: String, CodingKey {
        case entityId = "entity_id"
        case domain
    }
}

private struct ServiceCallDTO: Encodable {
    let entityId: String
    let data: [String: String]

    enum CodingKeys: String, CodingKey {
        case entityId = "entity_id"
        case data
    }
}

// MARK: - DefaultEntityRepository

/// Implementazione concreta del repository entità.
/// Combina cache offline, HTTP REST e aggiornamenti real-time via WebSocket.
public actor DefaultEntityRepository: EntityRepositoryProtocol {

    private let httpClient: DomusHttpClient
    private let webSocketClient: DualTransportWebSocketClient
    private let cache: OfflineEntityCache
    private let tokenStorage: KeychainManager

    public init(
        httpClient: DomusHttpClient,
        webSocketClient: DualTransportWebSocketClient,
        cache: OfflineEntityCache,
        tokenStorage: KeychainManager
    ) {
        self.httpClient = httpClient
        self.webSocketClient = webSocketClient
        self.cache = cache
        self.tokenStorage = tokenStorage
    }

    // MARK: - EntityRepositoryProtocol

    public func getEntities() async -> [Entity] {
        await cache.getEntities()
    }

    public func getEntity(id: String) async -> Entity? {
        await cache.getEntity(id: id)
    }

    public func syncEntities() async throws -> [Entity] {
        let dtos = try await httpClient.get(path: "/api/v1/entities", as: [EntityDTO].self)
        let entities = dtos.map { $0.toDomain() }
        await cache.saveEntities(entities)
        return entities
    }

    public func toggleEntity(id: String, domain: String = "") async throws -> Entity {
        let effectiveDomain = domain.isEmpty ? (await cache.getEntity(id: id)?.domain ?? "homeassistant") : domain
        let dto = ToggleRequestDTO(entityId: id, domain: effectiveDomain)
        let updated = try await httpClient.post(path: "/api/v1/entities/\(id)/toggle", body: dto, as: EntityDTO.self)
        let entity = updated.toDomain()
        await cache.updateEntity(entity)
        return entity
    }

    public func toggleEntity(id: String) async throws -> Entity {
        try await toggleEntity(id: id, domain: "")
    }

    public func callService(domain: String, service: String, entityId: String, data: [String: Any]) async throws {
        let bodyData = data.compactMapValues { "\($0)" }
        let dto = ServiceCallDTO(entityId: entityId, data: bodyData)
        try await httpClient.postVoid(path: "/api/v1/services/\(domain)/\(service)", body: dto)
    }

    public func updateLocalEntity(_ entity: Entity) async {
        await cache.updateEntity(entity)
    }

    public func getAreas() async -> [Area] {
        let cached = await cache.getAreas()
        if !cached.isEmpty { return cached }

        // Prova a caricare dall'API
        if let dtos = try? await httpClient.get(path: "/api/v1/areas", as: [AreaDTO].self) {
            let areas = dtos.map { $0.toDomain() }
            await cache.saveAreas(areas)
            return areas
        }
        return []
    }
}
