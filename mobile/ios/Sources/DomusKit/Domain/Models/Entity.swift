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

    public var isLight: Bool { domain == "light" }
    public var isSwitch: Bool { domain == "switch" }
    public var isSensor: Bool { domain == "sensor" }
    public var isClimate: Bool { domain == "climate" }
    public var isCamera: Bool { domain == "camera" }
    public var isAlarm: Bool { domain == "alarm_control_panel" }

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

    public var targetTemperature: Double? {
        if let val = attributes["temperature"]?.value as? Double { return val }
        if let val = attributes["target_temperature"]?.value as? Double { return val }
        if let val = attributes["temperature"]?.value as? Int { return Double(val) }
        return nil
    }

    public var hvacMode: String {
        return state.lowercased()
    }

    public var isHeating: Bool {
        let action = (attributes["hvac_action"]?.value as? String)?.lowercased()
        return action == "heating" || state.lowercased() == "heat"
    }

    public var isCooling: Bool {
        let action = (attributes["hvac_action"]?.value as? String)?.lowercased()
        return action == "cooling" || state.lowercased() == "cool"
    }

    public var powerWatts: Double? {
        if let val = attributes["power"]?.value as? Double { return val }
        if let val = attributes["power"]?.value as? Int { return Double(val) }
        if let val = attributes["current_power_w"]?.value as? Double { return val }
        if let val = attributes["current_power_w"]?.value as? Int { return Double(val) }
        if isSensor && (attributes["unit_of_measurement"]?.value as? String) == "W" {
            return Double(state)
        }
        return nil
    }

    public var energyKWh: Double? {
        if let val = attributes["energy"]?.value as? Double { return val }
        if let val = attributes["energy"]?.value as? Int { return Double(val) }
        if let val = attributes["total_energy_kwh"]?.value as? Double { return val }
        return nil
    }

    public var colorTemp: Int? {
        if let val = attributes["color_temp"]?.value as? Int { return val }
        if let val = attributes["color_temp_kelvin"]?.value as? Int { return val }
        return nil
    }

    public var rgbColor: (r: Double, g: Double, b: Double)? {
        if let arr = attributes["rgb_color"]?.value as? [Int], arr.count >= 3 {
            return (Double(arr[0]) / 255.0, Double(arr[1]) / 255.0, Double(arr[2]) / 255.0)
        }
        if let arr = attributes["rgb_color"]?.value as? [Double], arr.count >= 3 {
            return (arr[0] / 255.0, arr[1] / 255.0, arr[2] / 255.0)
        }
        return nil
    }

    public var isArmed: Bool {
        state.lowercased().hasPrefix("armed")
    }

    public var cameraImageUrl: String? {
        if let pic = attributes["entity_picture"]?.value as? String { return pic }
        if let pic = attributes["still_image_url"]?.value as? String { return pic }
        return nil
    }
}


public struct AnyCodable: Codable, Hashable, @unchecked Sendable {
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

extension AnyCodable: CustomStringConvertible {
    public var description: String {
        "\(value)"
    }
}


