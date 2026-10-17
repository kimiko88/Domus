import Foundation
import SwiftUI
import Combine

/// Azione rapida contestuale per lo Smart Stack di watchOS 10+.
public struct WatchSmartStackAction: Identifiable, Sendable {
    public let id: String
    public let title: String
    public let systemImage: String
    public let domain: String
    public let service: String
    public let entityId: String

    public init(id: String, title: String, systemImage: String, domain: String, service: String, entityId: String) {
        self.id = id
        self.title = title
        self.systemImage = systemImage
        self.domain = domain
        self.service = service
        self.entityId = entityId
    }
}

/// Modello dello Smart Stack di watchOS 10+ che appare automaticamente al ritorno a casa.
@MainActor
public final class DomusSmartStackHomeArrivalModel: ObservableObject {
    public static let shared = DomusSmartStackHomeArrivalModel()

    @Published public var isNearHome: Bool = true
    @Published public var currentIndoorTemp: Double = 21.0
    @Published public var isAlarmArmed: Bool = true

    public var smartStackRelevanceScore: Float {
        // watchOS 10+ Smart Stack WidgetKit relevance: massimo punteggio quando si entra nella geofence di casa
        return isNearHome ? 1.0 : 0.1
    }

    public var suggestedActions: [WatchSmartStackAction] {
        return [
            WatchSmartStackAction(
                id: "act_gate",
                title: "Apri Cancello",
                systemImage: "door.garage.open",
                domain: "cover",
                service: "open_cover",
                entityId: "cover.gate"
            ),
            WatchSmartStackAction(
                id: "act_disarm",
                title: "Disarma Allarme",
                systemImage: "lock.open.shield.fill",
                domain: "alarm_control_panel",
                service: "alarm_disarm",
                entityId: "alarm_control_panel.home"
            ),
            WatchSmartStackAction(
                id: "act_lights",
                title: "Luci Ingresso",
                systemImage: "lightbulb.fill",
                domain: "light",
                service: "turn_on",
                entityId: "light.entryway"
            )
        ]
    }

    public init() {}

    public func executeAction(_ action: WatchSmartStackAction) async throws {
        try await DomusHttpClient.shared.callService(
            domain: action.domain,
            service: action.service,
            entityId: action.entityId
        )
    }
}

/// Stato del termostato controllato tramite la Digital Crown di Apple Watch.
@MainActor
public final class DomusWatchThermostatState: ObservableObject {
    @Published public var currentTemperature: Double = 20.4
    @Published public var targetTemperature: Double = 21.0
    @Published public var hvacMode: String = "heat"
    @Published public var isHeating: Bool = true

    public let minTemperature: Double = 16.0
    public let maxTemperature: Double = 28.0
    public let step: Double = 0.5

    public init(current: Double = 20.4, target: Double = 21.0) {
        self.currentTemperature = current
        self.targetTemperature = target
    }

    /// Incrementa o decrementa la temperatura target con rotazione della Digital Crown.
    public func rotateCrown(deltaDegrees: Double) {
        let newTarget = targetTemperature + deltaDegrees
        targetTemperature = max(minTemperature, min(maxTemperature, round(newTarget * 2) / 2))
    }

    /// Invia il nuovo setpoint di temperatura al backend locale Domus.
    public func commitTargetTemperature(entityId: String = "climate.living_room") async throws {
        let payload: [String: Any] = ["temperature": targetTemperature]
        try await DomusHttpClient.shared.callService(
            domain: "climate",
            service: "set_temperature",
            entityId: entityId,
            data: payload
        )
    }
}
