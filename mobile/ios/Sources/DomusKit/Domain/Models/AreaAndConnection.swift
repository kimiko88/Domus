import Foundation

public struct Area: Identifiable, Codable, Hashable, Sendable {
    public let id: String
    public let name: String
    public let icon: String

    public init(id: String, name: String, icon: String = "home") {
        self.id = id
        self.name = name
        self.icon = icon
    }

    enum CodingKeys: String, CodingKey {
        case id
        case areaId = "area_id"
        case name
        case icon
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        self.id = (try? container.decodeIfPresent(String.self, forKey: .id))
            ?? (try? container.decodeIfPresent(String.self, forKey: .areaId))
            ?? ""
        self.name = (try? container.decodeIfPresent(String.self, forKey: .name)) ?? ""
        self.icon = (try? container.decodeIfPresent(String.self, forKey: .icon)) ?? "home"
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(name, forKey: .name)
        try container.encode(icon, forKey: .icon)
    }
}

public struct Device: Identifiable, Codable, Hashable, Sendable {
    public let id: String
    public let name: String
    public let integration: String
    public let manufacturer: String
    public let model: String
    public let areaId: String?
    public let swVersion: String

    public init(
        id: String,
        name: String,
        integration: String = "",
        manufacturer: String = "",
        model: String = "",
        areaId: String? = nil,
        swVersion: String = ""
    ) {
        self.id = id
        self.name = name
        self.integration = integration
        self.manufacturer = manufacturer
        self.model = model
        self.areaId = areaId
        self.swVersion = swVersion
    }
}

public enum TransportMode: String, Codable, Sendable {
    case localLan = "LOCAL_LAN"
    case remoteTls = "REMOTE_TLS"
    case connecting = "CONNECTING"
    case disconnected = "DISCONNECTED"

    public var displayName: String {
        switch self {
        case .localLan: return "LAN Locale"
        case .remoteTls: return "TLS Remoto (Fuori Casa)"
        case .connecting: return "Connessione..."
        case .disconnected: return "Disconnesso"
        }
    }
}

public struct ServerConnection: Equatable, Sendable {
    public var mode: TransportMode
    public var targetMode: TransportMode
    public var localAddress: String?
    public var remoteAddress: String?
    public var activeUrl: String?
    public var latencyMs: Int64
    public var isConnected: Bool
    public var lastError: String?

    public init(
        mode: TransportMode = .disconnected,
        targetMode: TransportMode = .disconnected,
        localAddress: String? = nil,
        remoteAddress: String? = nil,
        activeUrl: String? = nil,
        latencyMs: Int64 = 0,
        isConnected: Bool = false,
        lastError: String? = nil
    ) {
        self.mode = mode
        self.targetMode = targetMode
        self.localAddress = localAddress
        self.remoteAddress = remoteAddress
        self.activeUrl = activeUrl
        self.latencyMs = latencyMs
        self.isConnected = isConnected
        self.lastError = lastError
    }

    public var isLocal: Bool {
        mode == .localLan || targetMode == .localLan
    }

    public var isRemote: Bool {
        mode == .remoteTls || targetMode == .remoteTls
    }
}

public struct AuthSession: Equatable, Codable, Sendable {
    public var token: String?
    public var username: String?
    public var role: String
    public var isAuthenticated: Bool

    public init(
        token: String? = nil,
        username: String? = nil,
        role: String = "user",
        isAuthenticated: Bool = false
    ) {
        self.token = token
        self.username = username
        self.role = role
        self.isAuthenticated = isAuthenticated
    }
}
