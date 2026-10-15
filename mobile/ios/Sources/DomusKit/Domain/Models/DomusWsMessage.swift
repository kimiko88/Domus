import Foundation

/// Modello di messaggio WebSocket per eventi dal backend Domus.
public struct DomusWsMessage: Codable, Sendable {
    public let type: String
    public let id: Int64?
    public let accessToken: String?
    public let eventType: String?
    public let success: Bool?
    public let event: DomusWsEvent?

    enum CodingKeys: String, CodingKey {
        case type, id, success, event
        case accessToken = "access_token"
        case eventType = "event_type"
    }

    public init(
        type: String,
        id: Int64? = nil,
        accessToken: String? = nil,
        eventType: String? = nil,
        success: Bool? = nil,
        event: DomusWsEvent? = nil
    ) {
        self.type = type
        self.id = id
        self.accessToken = accessToken
        self.eventType = eventType
        self.success = success
        self.event = event
    }
}

/// Contenuto dell'evento WebSocket inviato dal backend Domus.
public struct DomusWsEvent: Codable, Sendable {
    public let entityId: String?
    public let newState: DomusWsEntityState?

    enum CodingKeys: String, CodingKey {
        case entityId = "entity_id"
        case newState = "new_state"
    }

    public init(entityId: String? = nil, newState: DomusWsEntityState? = nil) {
        self.entityId = entityId
        self.newState = newState
    }
}

/// Snapshot dello stato di un'entità all'interno di un evento WebSocket.
public struct DomusWsEntityState: Codable, Sendable {
    public let entityId: String
    public let state: String
    public let attributes: [String: AnyCodable]
    public let lastChanged: String?
    public let lastUpdated: String?

    enum CodingKeys: String, CodingKey {
        case entityId = "entity_id"
        case state
        case attributes
        case lastChanged = "last_changed"
        case lastUpdated = "last_updated"
    }

    public init(
        entityId: String,
        state: String,
        attributes: [String: AnyCodable] = [:],
        lastChanged: String? = nil,
        lastUpdated: String? = nil
    ) {
        self.entityId = entityId
        self.state = state
        self.attributes = attributes
        self.lastChanged = lastChanged
        self.lastUpdated = lastUpdated
    }
}

// Backward-compatibility aliases
public typealias WsEventContainer = DomusWsEvent
public typealias EntitySnapshot = DomusWsEntityState
