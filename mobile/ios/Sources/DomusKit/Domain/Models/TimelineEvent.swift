import Foundation

/// Categoria dell'evento nella cronologia della casa per il filtraggio rapido.
public enum TimelineCategory: String, CaseIterable, Identifiable, Sendable {
    case all = "Tutti"
    case security = "Sicurezza"
    case access = "Accessi"
    case climateAndLights = "Luci & Clima"
    case energy = "Energia"

    public var id: String { rawValue }

    public var icon: String {
        switch self {
        case .all: return "list.bullet.rectangle.portrait.fill"
        case .security: return "shield.checkered"
        case .access: return "door.left.hand.open"
        case .climateAndLights: return "lightbulb.fill"
        case .energy: return "bolt.fill"
        }
    }
}

/// Modello per una voce nella Timeline / Registro Eventi Storici (Logbook & Audit).
public struct TimelineEvent: Identifiable, Codable, Hashable, Sendable {
    public let id: String
    public let entityId: String
    public let entityName: String
    public let domain: String
    public let state: String
    public let message: String
    public let contextType: String // "user", "automation", "system"
    public let contextName: String? // "Marco", "Automazione Notturna", "Domus Core"
    public let timestamp: Date

    public init(
        id: String = UUID().uuidString,
        entityId: String,
        entityName: String,
        domain: String,
        state: String,
        message: String,
        contextType: String = "user",
        contextName: String? = nil,
        timestamp: Date = Date()
    ) {
        self.id = id
        self.entityId = entityId
        self.entityName = entityName
        self.domain = domain
        self.state = state
        self.message = message
        self.contextType = contextType
        self.contextName = contextName
        self.timestamp = timestamp
    }

    enum CodingKeys: String, CodingKey {
        case id
        case entityId = "entity_id"
        case entityName = "entity_name"
        case domain
        case state
        case message
        case contextType = "context_type"
        case contextName = "context_name"
        case timestamp
    }

    public var category: TimelineCategory {
        switch domain {
        case "alarm_control_panel", "camera", "binary_sensor":
            return .security
        case "lock", "cover":
            return .access
        case "light", "climate":
            return .climateAndLights
        case "sensor":
            if entityId.contains("power") || entityId.contains("energy") || entityId.contains("potenza") {
                return .energy
            }
            return .security
        default:
            return .security
        }
    }

    public var categoryIcon: String {
        switch category {
        case .security: return "shield.fill"
        case .access: return "door.left.hand.open"
        case .climateAndLights: return "lightbulb.fill"
        case .energy: return "bolt.fill"
        case .all: return "clock.fill"
        }
    }

    public var formattedTime: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm"
        return formatter.string(from: timestamp)
    }

    public static let sampleTimeline: [TimelineEvent] = [
        TimelineEvent(
            entityId: "lock.garage_door",
            entityName: "Porta Garage",
            domain: "lock",
            state: "unlocked",
            message: "Porta garage aperta",
            contextType: "user",
            contextName: "Marco",
            timestamp: Calendar.current.date(byAdding: .minute, value: -22, to: Date()) ?? Date()
        ),
        TimelineEvent(
            entityId: "binary_sensor.garden_motion",
            entityName: "Sensore Giardino",
            domain: "binary_sensor",
            state: "on",
            message: "Rilevato movimento in giardino",
            contextType: "system",
            contextName: "Radar PIR",
            timestamp: Calendar.current.date(byAdding: .minute, value: -64, to: Date()) ?? Date()
        ),
        TimelineEvent(
            entityId: "sensor.home_power",
            entityName: "Consumo Generale",
            domain: "sensor",
            state: "3820 W",
            message: "Superata soglia potenza contrattuale: consumati 3.8 kW",
            contextType: "system",
            contextName: "Energy Monitor",
            timestamp: Calendar.current.date(byAdding: .hour, value: -2, to: Date()) ?? Date()
        ),
        TimelineEvent(
            entityId: "scene.dinner",
            entityName: "Scenario Cena",
            domain: "scene",
            state: "active",
            message: "Attivato scenario Cena e luci calde",
            contextType: "automation",
            contextName: "Regola Oraria",
            timestamp: Calendar.current.date(byAdding: .hour, value: -3, to: Date()) ?? Date()
        ),
        TimelineEvent(
            entityId: "alarm_control_panel.home",
            entityName: "Allarme Domus",
            domain: "alarm_control_panel",
            state: "armed_away",
            message: "Allarme perimetrale inserito",
            contextType: "user",
            contextName: "App iOS",
            timestamp: Calendar.current.date(byAdding: .hour, value: -5, to: Date()) ?? Date()
        )
    ]
}
