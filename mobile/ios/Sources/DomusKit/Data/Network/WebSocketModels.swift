import Foundation

public struct DomusWsMessage: Codable, Sendable {
    public let type: String
    public let id: Int64?
    public let accessToken: String?
    public let eventType: String?
    public let success: Bool?
    public let event: WsEventContainer?

    enum CodingKeys: String, CodingKey {
        case type, id, success, event
        case accessToken = "access_token"
        case eventType = "event_type"
    }
}

public struct WsEventContainer: Codable, Sendable {
    public let entityId: String?
    public let newState: EntitySnapshot?

    enum CodingKeys: String, CodingKey {
        case entityId = "entity_id"
        case newState = "new_state"
    }
}

public struct EntitySnapshot: Codable, Sendable {
    public let entityId: String
    public let state: String
    public let attributes: [String: AnyCodable]
    public let lastChanged: String
    public let lastUpdated: String

    enum CodingKeys: String, CodingKey {
        case entityId = "entity_id"
        case state, attributes
        case lastChanged = "last_changed"
        case lastUpdated = "last_updated"
    }
}
