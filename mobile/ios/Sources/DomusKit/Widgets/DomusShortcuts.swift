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
    }
}
