import Foundation
import AppIntents

/// Fornitore di Comandi Rapidi di sistema (AppShortcutsProvider).
/// Espone automaticamente le azioni vocali per Siri e l'indicizzazione Spotlight senza configurazione manuale.
public struct DomusShortcuts: AppShortcutsProvider {
    public static var shortcutTileColor: ShortcutTileColor = .lightBlue

    public static var appShortcuts: [AppShortcut] {
        AppShortcut(
            intent: ToggleEntityIntent(),
            phrases: [
                "Spegni salotto con \(.applicationName)",
                "Accendi salotto con \(.applicationName)",
                "Commuta dispositivi con \(.applicationName)",
                "Attiva allarme con \(.applicationName)"
            ],
            shortTitle: "Attiva Dispositivo",
            systemImageName: "power"
        )

        AppShortcut(
            intent: RunSceneIntent(),
            phrases: [
                "Attiva Buonanotte con \(.applicationName)",
                "Esegui Buonanotte con \(.applicationName)",
                "Attiva Buongiorno con \(.applicationName)",
                "Attiva Esco di casa con \(.applicationName)"
            ],
            shortTitle: "Esegui Scenario Domus",
            systemImageName: "sparkles"
        )

        AppShortcut(
            intent: ControlCoverIntent(),
            phrases: [
                "Apri il garage con \(.applicationName)",
                "Chiudi il garage con \(.applicationName)",
                "Apri il cancello con \(.applicationName)",
                "Chiudi il cancello con \(.applicationName)"
            ],
            shortTitle: "Apri o Chiudi Ingresso",
            systemImageName: "door.garage.open"
        )

        AppShortcut(
            intent: SetClimateTemperatureIntent(),
            phrases: [
                "Imposta 21 gradi in soggiorno con \(.applicationName)",
                "Imposta temperatura con \(.applicationName)",
                "Regola il termostato con \(.applicationName)"
            ],
            shortTitle: "Imposta Temperatura",
            systemImageName: "thermometer.medium"
        )
    }
}

