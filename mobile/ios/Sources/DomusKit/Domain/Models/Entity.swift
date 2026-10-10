import Foundation

public struct Entity: Identifiable, Codable, Hashable, Sendable {
    public let id: String
    public let name: String
    public let domain: String
    public var state: String
    public var attributes: [String: AnyCodable]
    public let areaId: String?
    public let lastChanged: String
    public let lastUpdated: String

    public init(
        id: String,
        name: String,
        domain: String,
        state: String,
        attributes: [String: AnyCodable] = [:],
        areaId: String? = nil,
        lastChanged: String = "",
        lastUpdated: String = ""
    ) {
        self.id = id
        self.name = name
        self.domain = domain
        self.state = state
        self.attributes = attributes
        self.areaId = areaId
        self.lastChanged = lastChanged
        self.lastUpdated = lastUpdated
    }

    public var isLight: Boolean { domain == "light" }
    public var isSwitch: Boolean { domain == "switch" }
    public var isSensor: Boolean { domain == "sensor" }
    public var isClimate: Boolean { domain == "climate" }
    public var isCamera: Boolean { domain == "camera" }
    public var isAlarm: Boolean { domain == "alarm_control_panel" }

    public var isOn: Bool {
        let s = state.lowercased()
        return s == "on" || s == "open" || s == "armed_home" || s == "armed_away"
    }

    public var brightness: Int? {
        if let val = attributes["brightness"]?.value as? Int { return val }
        if let val = attributes["brightness"]?.value as? Double { return Int(val) }
        return nil
    }

    public var currentTemperature: Double? {
        if let val = attributes["current_temperature"]?.value as? Double { return val }
        if let val = attributes["current_temperature"]?.value as? Int { return Double(val) }
        return nil
    }
}

public struct AnyCodable: Codable, Hashable, Sendable {
    public let value: Any

    public init(_ value: Any) {
        self.value = value
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if let intVal = try? container.decode(Int.self) {
            value = intVal
        } else if let dblVal = try? container.decode(Double.self) {
            value = dblVal
        } else if let boolVal = try? container.decode(Bool.self) {
            value = boolVal
        } else if let strVal = try? container.decode(String.self) {
            value = strVal
        } else {
            value = ""
        }
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        if let intVal = value as? Int {
            try container.encode(intVal)
        } else if let dblVal = value as? Double {
            try container.encode(dblVal)
        } else if let boolVal = value as? Bool {
            try container.encode(boolVal)
        } else if let strVal = value as? String {
            try container.encode(strVal)
        } else {
            try container.encode("\(value)")
        }
    }

    public func hash(into hasher: inout Hasher) {
        hasher.combine("\(value)")
    }

    public static func == (lhs: AnyCodable, rhs: AnyCodable) -> Bool {
        "\(lhs.value)" == "\(rhs.value)"
    }
}

public typealias Boolean = Bool
