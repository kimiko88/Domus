import SwiftUI

/// Vista per la gestione delle funzionalità di Fase 3: Geofencing, Notifiche Critiche e Tag NFC.
public struct PresenceAndNfcSheet: View {
    @ObservedObject public var presenceManager: GeofencingPresenceManager
    @ObservedObject public var notificationManager: NotificationManager
    @ObservedObject public var nfcManager: NfcTagManager

    @State private var radiusSlider: Double = 150
    @State private var selectedNfcAction: String = "scene:goodnight"
    @State private var feedbackBanner: String? = nil

    public init(
        presenceManager: GeofencingPresenceManager,
        notificationManager: NotificationManager,
        nfcManager: NfcTagManager
    ) {
        self.presenceManager = presenceManager
        self.notificationManager = notificationManager
        self.nfcManager = nfcManager
        _radiusSlider = State(initialValue: presenceManager.config.radiusMeters)
    }

    public var body: some View {
        NavigationStack {
            List {
                // Banner feedback rapido
                if let banner = feedbackBanner {
                    Section {
                        HStack {
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundColor(.green)
                            Text(banner)
                                .font(.subheadline.bold())
                        }
                    }
                }

                // ── SEZIONE 1: Presenza Multi-Tier & Geofencing ──
                Section(header: Label("Presenza & Geofencing a Batteria Zero", systemImage: "location.circle.fill")) {
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("Stato Presenza")
                                .font(.subheadline)
                            Text(presenceTitle)
                                .font(.headline.bold())
                                .foregroundColor(presenceColor)
                        }
                        Spacer()
                        Circle()
                            .fill(presenceColor)
                            .frame(width: 16, height: 16)
                    }

                    Toggle("Monitoraggio Geofence", isOn: Binding(
                        get: { presenceManager.isMonitoringActive },
                        set: { enable in
                            if enable {
                                presenceManager.requestPermissions()
                                presenceManager.startMonitoring()
                            } else {
                                presenceManager.stopMonitoring()
                            }
                        }
                    ))

                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Text("Raggio Perimetro Casa:")
                                .font(.caption)
                            Spacer()
                            Text("\(Int(radiusSlider)) metri")
                                .font(.caption.bold())
                        }
                        Slider(value: $radiusSlider, in: 50...500, step: 25) { _ in
                            presenceManager.updateHomeLocation(
                                latitude: presenceManager.config.latitude,
                                longitude: presenceManager.config.longitude,
                                radius: radiusSlider
                            )
                        }
                    }

                    HStack {
                        Image(systemName: "wifi")
                            .foregroundColor(.cyan)
                        Text("Tier 1: Rilevamento Wi-Fi LAN attivo")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }

                // ── SEZIONE 2: Notifiche Push & Allarmi Critici ──
                Section(header: Label("Notifiche Azionabili & Allarmi Critici", systemImage: "bell.badge.fill")) {
                    HStack {
                        Text("Allarmi Critici (Bypass Silenzioso)")
                        Spacer()
                        Text(notificationManager.criticalAlertsAllowed ? "Abilitato" : "Disabilitato")
                            .font(.caption.bold())
                            .foregroundColor(notificationManager.criticalAlertsAllowed ? .green : .orange)
                    }

                    Button("Richiedi Autorizzazioni Notifiche") {
                        Task {
                            let granted = await notificationManager.requestAuthorization()
                            showBanner(granted ? "Notifiche e allarmi critici autorizzati" : "Permesso rifiutato")
                        }
                    }
                    .foregroundColor(.cyan)

                    Button(action: {
                        Task {
                            try? await notificationManager.sendTestNotification(isCritical: false)
                            showBanner("Notifica interattiva inviata!")
                        }
                    }) {
                        HStack {
                            Image(systemName: "bell.fill")
                            Text("Test Notifica Interattiva (Pioggia)")
                        }
                    }

                    Button(action: {
                        Task {
                            try? await notificationManager.sendTestNotification(isCritical: true)
                            showBanner("Allarme critico emesso!")
                        }
                    }) {
                        HStack {
                            Image(systemName: "speaker.wave.3.fill")
                            Text("Test Allarme Critico (Suono Alto)")
                        }
                        .foregroundColor(.red)
                    }
                }

                // ── SEZIONE 3: Tag NFC & QR ──
                Section(header: Label("Tag NFC Fisici (CoreNFC)", systemImage: "wave.3.right.circle.fill")) {
                    HStack {
                        Text("Supporto NFC Hardware")
                        Spacer()
                        Text(nfcManager.isNfcAvailable ? "Disponibile" : "Solo su iPhone fisico")
                            .font(.caption.bold())
                            .foregroundColor(nfcManager.isNfcAvailable ? .green : .secondary)
                    }

                    Button(action: {
                        nfcManager.startReading()
                    }) {
                        HStack {
                            Image(systemName: "wave.3.right")
                            Text("Scansiona Tag NFC")
                        }
                        .font(.headline)
                    }
                    .foregroundColor(.cyan)

                    Picker("Programma Tag con Azione", selection: $selectedNfcAction) {
                        Text("Scena Buonanotte").tag("scene:goodnight")
                        Text("Disarma Allarme").tag("alarm:disarm")
                        Text("Toggle Luce Ingresso").tag("toggle:light.entrance")
                    }

                    Button("Programma Tag NFC Fisico") {
                        let uri = "domus://\(selectedNfcAction.replacingOccurrences(of: ":", with: "/"))"
                        nfcManager.writeTag(uri: uri)
                    }

                    Button("Simula Lettura Tag (Test)") {
                        nfcManager.simulateTagScan(uri: "domus://scene/goodnight")
                        showBanner("Tag Buonanotte rilevato!")
                    }
                    .font(.caption)
                    .foregroundColor(.secondary)

                    if let tag = nfcManager.lastReadTag {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("Ultimo Tag Rilevato:")
                                .font(.caption.bold())
                            Text(tag.uri)
                                .font(.system(.caption, design: .monospaced))
                                .foregroundColor(.cyan)
                        }
                        .padding(.vertical, 4)
                    }
                }
            }
            .navigationTitle("Presenza & Automazioni")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
        }
    }

    private func showBanner(_ msg: String) {
        feedbackBanner = msg
        DispatchQueue.main.asyncAfter(deadline: .now() + 3) {
            feedbackBanner = nil
        }
    }

    private var presenceTitle: String {
        switch presenceManager.presenceState {
        case .home:    return "A Casa (Presenza Attiva)"
        case .away:    return "Fuori Casa"
        case .unknown: return "Non Determinato"
        }
    }

    private var presenceColor: Color {
        switch presenceManager.presenceState {
        case .home:    return .green
        case .away:    return .blue
        case .unknown: return .secondary
        }
    }
}
