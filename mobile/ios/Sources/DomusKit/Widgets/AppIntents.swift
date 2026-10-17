import Foundation
import AppIntents
import SwiftUI

/// AppIntent per il controllo interattivo rapido dai Widget della schermata home e Centro di Controllo.
public struct ToggleEntityIntent: AppIntent {
    public static var title: LocalizedStringResource = "Attiva/Disattiva Dispositivo"
    public static var description = IntentDescription("Accende o spegne un dispositivo Domus direttamente dal widget senza aprire l'app.")

    @Parameter(title: "ID Entità")
    public var entityId: String

    @Parameter(title: "Nome Dispositivo")
    public var entityName: String?

    public init() {
        self.entityId = ""
        self.entityName = ""
    }

    public init(entityId: String, entityName: String? = nil) {
        self.entityId = entityId
        self.entityName = entityName
    }

    public func perform() async throws -> some IntentResult {
        #if canImport(UIKit)
        Haptics.medium()
        #endif
        if !entityId.isEmpty {
            try? await DomusHttpClient.shared.toggleEntity(id: entityId)
        }
        return .result()
    }
}

/// AppIntent per l'esecuzione di scenari domotici dai widget (es. Scena Uscita, Buonanotte, Cinema).
public struct RunSceneIntent: AppIntent {
    public static var title: LocalizedStringResource = "Esegui Scenario Domus"
    public static var description = IntentDescription("Attiva uno scenario istantaneamente dal widget o da Siri.")

    @Parameter(title: "ID Scenario")
    public var sceneId: String

    public init() {
        self.sceneId = ""
    }

    public init(sceneId: String) {
        self.sceneId = sceneId
    }

    public func perform() async throws -> some IntentResult {
        #if canImport(UIKit)
        Haptics.success()
        #endif
        if !sceneId.isEmpty {
            try? await DomusHttpClient.shared.callService(domain: "scene", service: "turn_on", entityId: sceneId)
        }
        return .result()
    }
}

/// AppIntent per il controllo di porte, cancelli e garage via Siri e Comandi.
public struct ControlCoverIntent: AppIntent {
    public static var title: LocalizedStringResource = "Controlla Apertura (Cancello / Garage)"
    public static var description = IntentDescription("Apre o chiude porte, cancelli e garage direttamente con Siri.")

    @Parameter(title: "ID Entità (es. cover.gate o cover.garage)")
    public var entityId: String

    @Parameter(title: "Azione (open/close)")
    public var action: String

    public init() {
        self.entityId = "cover.gate"
        self.action = "open"
    }

    public init(entityId: String, action: String = "open") {
        self.entityId = entityId
        self.action = action
    }

    public func perform() async throws -> some IntentResult {
        #if canImport(UIKit)
        Haptics.success()
        #endif
        let service = (action.lowercased() == "close") ? "close_cover" : "open_cover"
        try? await DomusHttpClient.shared.callService(domain: "cover", service: service, entityId: entityId)
        return .result()
    }
}

/// AppIntent per la regolazione della temperatura del termostato tramite Siri.
public struct SetClimateTemperatureIntent: AppIntent {
    public static var title: LocalizedStringResource = "Imposta Temperatura Clima"
    public static var description = IntentDescription("Imposta la temperatura del termostato con comandi vocali diretti.")

    @Parameter(title: "ID Termostato (es. climate.living_room)")
    public var entityId: String

    @Parameter(title: "Temperatura Desiderata (°C)")
    public var temperature: Double

    public init() {
        self.entityId = "climate.living_room"
        self.temperature = 21.0
    }

    public init(entityId: String, temperature: Double) {
        self.entityId = entityId
        self.temperature = temperature
    }

    public func perform() async throws -> some IntentResult {
        #if canImport(UIKit)
        Haptics.medium()
        #endif
        let data: [String: Any] = ["temperature": temperature]
        try? await DomusHttpClient.shared.callService(domain: "climate", service: "set_temperature", entityId: entityId, data: data)
        return .result()
    }
}

#if os(iOS)
@available(iOS 16.1, *)
extension ToggleEntityIntent: LiveActivityIntent {}

@available(iOS 16.1, *)
extension ControlCoverIntent: LiveActivityIntent {}
#endif


