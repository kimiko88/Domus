import Foundation
import SwiftUI
#if canImport(WidgetKit)
import WidgetKit
#endif
#if canImport(AppIntents)
import AppIntents
#endif

// MARK: - iOS 18 Control Center SetValueIntent

#if canImport(AppIntents)
public struct ControlToggleEntityIntent: SetValueIntent {
    public static var title: LocalizedStringResource = "Commuta Dispositivo"
    public static var description = IntentDescription("Accende o spegne un dispositivo Domus dal Centro di Controllo.")

    @Parameter(title: "ID Entità")
    public var entityId: String

    @Parameter(title: "Stato")
    public var value: Bool

    public init() {
        self.entityId = ""
        self.value = false
    }

    public init(entityId: String, value: Bool = false) {
        self.entityId = entityId
        self.value = value
    }

    public func perform() async throws -> some IntentResult {
        #if canImport(UIKit)
        Haptics.medium()
        #endif
        if !entityId.isEmpty {
            try? await DomusHttpClient.shared.callService(
                domain: "homeassistant",
                service: value ? "turn_on" : "turn_off",
                entityId: entityId
            )
        }
        return .result()
    }
}
#endif

// MARK: - iOS 18 Control Center Widgets & Lock Screen Controls

#if canImport(WidgetKit) && canImport(AppIntents) && os(iOS)
@available(iOS 18.0, *)
public struct DomusDeviceControlWidget: ControlWidget {
    public static var kind: String = "com.domus.deviceControl"

    public init() {}

    public var body: some ControlWidgetConfiguration {
        StaticControlConfiguration(kind: Self.kind) {
            ControlWidgetToggle(
                "Dispositivo Rapido",
                isOn: false,
                action: ControlToggleEntityIntent(entityId: "light.living_room", value: true)
            ) { isOn in
                Label(isOn ? "Acceso" : "Spento", systemImage: isOn ? "lightbulb.fill" : "lightbulb")
            }
        }
        .displayName("Dispositivo Domus")
        .description("Attiva o disattiva un dispositivo Domus direttamente dal Centro di Controllo o dalla schermata di blocco.")
    }
}

@available(iOS 18.0, *)
public struct DomusSceneControlWidget: ControlWidget {
    public static var kind: String = "com.domus.sceneControl"

    public init() {}

    public var body: some ControlWidgetConfiguration {
        StaticControlConfiguration(kind: Self.kind) {
            ControlWidgetButton(
                action: RunSceneIntent(sceneId: "scene.goodnight")
            ) {
                Label("Scenario Buonanotte", systemImage: "moon.stars.fill")
            }
        }
        .displayName("Scenario Domus")
        .description("Esegui uno scenario rapido Domus dal Centro di Controllo.")
    }
}
#endif
