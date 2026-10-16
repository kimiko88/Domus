import Foundation

/// Modello per una condizione o sorgente di innesco (Trigger) di un'automazione.
public struct AutomationTrigger: Identifiable, Codable, Hashable, Sendable {
    public var id: String
    public var platform: String // "time", "state", "sun", "geofence", "event"
    public var entityKey: String?
    public var from: String?
    public var to: String?
    public var at: String? // "HH:mm:ss"
    public var eventType: String?
    public var forSeconds: Int?

    public init(
        id: String = UUID().uuidString,
        platform: String = "time",
        entityKey: String? = nil,
        from: String? = nil,
        to: String? = nil,
        at: String? = nil,
        eventType: String? = nil,
        forSeconds: Int? = nil
    ) {
        self.id = id
        self.platform = platform
        self.entityKey = entityKey
        self.from = from
        self.to = to
        self.at = at
        self.eventType = eventType
        self.forSeconds = forSeconds
    }

    enum CodingKeys: String, CodingKey {
        case id
        case platform
        case entityKey = "entity_key"
        case from
        case to
        case at
        case eventType = "event_type"
        case forSeconds = "for_seconds"
    }

    public var displayTitle: String {
        switch platform {
        case "time":
            return "Orario: \(at ?? "07:30")"
        case "sun":
            return "Alba / Tramonto"
        case "state":
            let e = entityKey ?? "Sensore"
            return "\(e) diventa \(to ?? "attivo")"
        case "geofence":
            return "Presenza: \(eventType ?? "Uscita/Ingresso")"
        default:
            return "Evento: \(platform)"
        }
    }

    public var displayIcon: String {
        switch platform {
        case "time": return "clock.fill"
        case "sun": return "sun.max.fill"
        case "state": return "figure.walk.motion"
        case "geofence": return "location.fill"
        default: return "bolt.badge.clock.fill"
        }
    }
}

/// Modello per un filtro condizionale che deve risultare vero per procedere con le azioni.
public struct AutomationCondition: Identifiable, Codable, Hashable, Sendable {
    public var id: String
    public var condition: String // "state", "time", "numeric_state"
    public var entityKey: String?
    public var state: String?
    public var after: String?
    public var before: String?

    public init(
        id: String = UUID().uuidString,
        condition: String = "state",
        entityKey: String? = nil,
        state: String? = nil,
        after: String? = nil,
        before: String? = nil
    ) {
        self.id = id
        self.condition = condition
        self.entityKey = entityKey
        self.state = state
        self.after = after
        self.before = before
    }

    enum CodingKeys: String, CodingKey {
        case id
        case condition
        case entityKey = "entity_key"
        case state
        case after
        case before
    }

    public var displayTitle: String {
        switch condition {
        case "state":
            let e = entityKey ?? "Entità"
            return "\(e) è '\(state ?? "on")'"
        case "time":
            if let a = after, let b = before {
                return "Tra le \(a) e le \(b)"
            } else if let a = after {
                return "Dopo le \(a)"
            } else if let b = before {
                return "Prima delle \(b)"
            }
            return "Fascia oraria"
        default:
            return "Condizione: \(condition)"
        }
    }

    public var displayIcon: String {
        switch condition {
        case "state": return "checkmark.shield.fill"
        case "time": return "hourglass"
        default: return "slider.horizontal.3"
        }
    }
}

/// Modello per un'azione eseguita dall'automazione.
public struct AutomationAction: Identifiable, Codable, Hashable, Sendable {
    public var id: String
    public var type: String // "service", "delay", "scene"
    public var service: String? // "turn_on", "turn_off", "set_temperature", "notify"
    public var targetEntityKey: String?
    public var params: [String: String]?

    public init(
        id: String = UUID().uuidString,
        type: String = "service",
        service: String? = "turn_on",
        targetEntityKey: String? = nil,
        params: [String: String]? = nil
    ) {
        self.id = id
        self.type = type
        self.service = service
        self.targetEntityKey = targetEntityKey
        self.params = params
    }

    enum CodingKeys: String, CodingKey {
        case id
        case type
        case service
        case targetEntityKey = "target_entity_key"
        case params
    }

    public var displayTitle: String {
        let target = targetEntityKey ?? "Dispositivo"
        switch service {
        case "turn_on":
            return "Accendi / Attiva \(target)"
        case "turn_off":
            return "Spegni / Disattiva \(target)"
        case "set_temperature":
            let t = params?["temperature"] ?? "20"
            return "Imposta clima a \(t)°C su \(target)"
        case "notify":
            return "Invia notifica push: \"\(params?["message"] ?? "Avviso Domus")\""
        default:
            return "\(service ?? "Azione") -> \(target)"
        }
    }

    public var displayIcon: String {
        switch service {
        case "turn_on": return "power.circle.fill"
        case "turn_off": return "power.circle"
        case "set_temperature": return "thermometer.sun.fill"
        case "notify": return "bell.badge.fill"
        default: return "play.circle.fill"
        }
    }
}

/// Regola completa di automazione ("Se accade X con condizione Y allora fai Z").
public struct AutomationRule: Identifiable, Codable, Hashable, Sendable {
    public var id: String
    public var name: String
    public var description: String?
    public var enabled: Bool
    public var triggers: [AutomationTrigger]
    public var conditions: [AutomationCondition]
    public var actions: [AutomationAction]
    public var executionMode: String
    public var lastRunAt: String?
    public var lastStatus: String?

    public init(
        id: String = UUID().uuidString,
        name: String,
        description: String? = nil,
        enabled: Bool = true,
        triggers: [AutomationTrigger] = [],
        conditions: [AutomationCondition] = [],
        actions: [AutomationAction] = [],
        executionMode: String = "single",
        lastRunAt: String? = nil,
        lastStatus: String? = nil
    ) {
        self.id = id
        self.name = name
        self.description = description
        self.enabled = enabled
        self.triggers = triggers
        self.conditions = conditions
        self.actions = actions
        self.executionMode = executionMode
        self.lastRunAt = lastRunAt
        self.lastStatus = lastStatus
    }

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case description
        case enabled
        case triggers
        case conditions
        case actions
        case executionMode = "execution_mode"
        case lastRunAt = "last_run_at"
        case lastStatus = "last_status"
    }

    public static let sampleRules: [AutomationRule] = [
        AutomationRule(
            id: "auto.goodmorning",
            name: "Buongiorno e Riscaldamento",
            description: "Alle 07:00 nei giorni feriali accende il riscaldamento e prepara il caffè",
            enabled: true,
            triggers: [
                AutomationTrigger(platform: "time", at: "07:00:00")
            ],
            conditions: [
                AutomationCondition(condition: "state", entityKey: "alarm_control_panel.home", state: "disarmed")
            ],
            actions: [
                AutomationAction(type: "service", service: "set_temperature", targetEntityKey: "climate.thermostat", params: ["temperature": "21.0"]),
                AutomationAction(type: "service", service: "turn_on", targetEntityKey: "switch.coffee_machine")
            ]
        ),
        AutomationRule(
            id: "auto.night_security",
            name: "Sicurezza Notturna Giardino",
            description: "Dopo le 23:00, se il radar perimetrale rileva movimento, accende i faretti",
            enabled: true,
            triggers: [
                AutomationTrigger(platform: "state", entityKey: "binary_sensor.garden_motion", to: "on")
            ],
            conditions: [
                AutomationCondition(condition: "time", after: "23:00:00", before: "06:00:00")
            ],
            actions: [
                AutomationAction(type: "service", service: "turn_on", targetEntityKey: "light.garden_floodlight"),
                AutomationAction(type: "service", service: "notify", targetEntityKey: "notify.mobile_app", params: ["message": "Movimento rilevato in giardino di notte!"])
            ]
        ),
        AutomationRule(
            id: "auto.leave_home",
            name: "Uscita da Casa",
            description: "Quando tutti i membri escono dal perimetro di casa, spegne tutte le luci e arma l'allarme",
            enabled: true,
            triggers: [
                AutomationTrigger(platform: "geofence", eventType: "leave_home")
            ],
            conditions: [],
            actions: [
                AutomationAction(type: "service", service: "turn_off", targetEntityKey: "light.all_lights"),
                AutomationAction(type: "service", service: "turn_on", targetEntityKey: "alarm_control_panel.home", params: ["arm_mode": "armed_away"])
            ]
        )
    ]
}
