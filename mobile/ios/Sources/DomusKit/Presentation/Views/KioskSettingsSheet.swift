import SwiftUI

/// Foglio di configurazione per la Modalità Kiosk (Tablet a Parete) — Fase 5.
public struct KioskSettingsSheet: View {
    @ObservedObject public var kioskManager: KioskModeManager

    public init(kioskManager: KioskModeManager) {
        self.kioskManager = kioskManager
    }

    public var body: some View {
        NavigationStack {
            List {
                Section(header: Label("Impostazioni Schermo a Parete", systemImage: "ipad.landscape")) {
                    Toggle("Modalità Kiosk (Schermo Sempre Attivo)", isOn: Binding(
                        get: { kioskManager.isKioskModeEnabled },
                        set: { kioskManager.setKioskMode(enabled: $0) }
                    ))

                    Button("Avvia Screensaver Notturno Adesso") {
                        kioskManager.enterScreensaver()
                    }
                    .foregroundColor(.cyan)

                    HStack {
                        Text("Accesso Guidato (Blocco Kiosk)")
                        Spacer()
                        Text(kioskManager.isGuidedAccessActive ? "Attivo" : "Disattivato")
                            .font(.caption.bold())
                            .foregroundColor(kioskManager.isGuidedAccessActive ? .green : .secondary)
                    }
                }

                Section(header: Label("Protezione Ciclo Batteria", systemImage: "battery.100.bolt")) {
                    Toggle("Protezione 80% / 20%", isOn: $kioskManager.isBatteryProtectionEnabled)

                    HStack {
                        Text("Livello Batteria Attuale")
                        Spacer()
                        let pct = Int(kioskManager.batteryLevel * 100)
                        Text(pct >= 0 ? "\(pct)%" : "100%")
                            .font(.subheadline.bold())
                    }

                    HStack {
                        Text("Stato Carica")
                        Spacer()
                        Text(kioskManager.isCharging ? "In Carica ⚡️" : "Batteria 🔋")
                            .font(.caption.bold())
                            .foregroundColor(kioskManager.isCharging ? .green : .secondary)
                    }

                    VStack(alignment: .leading, spacing: 4) {
                        Text("ID Presa Smart Ricarica")
                            .font(.caption)
                            .foregroundColor(.secondary)
                        TextField("switch.tablet_charger", text: $kioskManager.smartPlugEntityId)
                            .textFieldStyle(.roundedBorder)
                    }

                    Text("Quando la carica raggiunge l'80%, l'app invia il comando di spegnimento alla presa smart per preservare la batteria al litio; la riaccende al 20%.")
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }
            }
            .navigationTitle("Modalità Kiosk")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
        }
    }
}
