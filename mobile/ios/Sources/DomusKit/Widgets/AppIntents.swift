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

#if os(iOS)
@available(iOS 16.1, *)
extension ToggleEntityIntent: LiveActivityIntent {}
#endif

