import SwiftUI

/// Vista per la gestione delle funzionalità di Fase 3: Geofencing, Notifiche Critiche e Tag NFC.
public struct PresenceAndNfcSheet: View {
    @ObservedObject public var presenceManager: GeofencingPresenceManager
    @ObservedObject public var notificationManager: NotificationManager
    @ObservedObject public var nfcManager: NfcTagManager

    @State private var radiusSlider: Double = 150
    @State private var selectedNfcAction: String = "scene:goodnight"
    @State private var feedbackBanner: String? = nil

    // ── Stato Simulatore Vacanza & Pendolarismo ──
    @State private var isVacationModeEnabled: Bool = false
    @State private var vacationJitterMinutes: Int = 15
    @State private var vacationActions: [VacationActionDTO] = [
        VacationActionDTO(entityId: "light.living_room", command: "turn_on", typicalTime: "20:30"),
        VacationActionDTO(entityId: "cover.bedroom", command: "close", typicalTime: "22:15"),
        VacationActionDTO(entityId: "light.kitchen", command: "turn_off", typicalTime: "23:00")
    ]
    @State private var newActionEntity: String = "light.living_room"
    @State private var newActionCommand: String = "turn_on"
    @State private var newActionTime: String = "21:00"
    @State private var isAddingVacationAction: Bool = false
    @State private var commuteStatus: CommuteStatusDTO? = nil

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

                // ── SEZIONE 4: Simulatore di Presenza Vacanza (Habit Replay) ──
                Section(header: Label("Simulatore di Presenza Vacanza", systemImage: "sun.max.fill")) {
                    Toggle("Simulazione Presenza Reale", isOn: Binding(
                        get: { isVacationModeEnabled },
                        set: { enable in
                            isVacationModeEnabled = enable
                            Task {
                                do {
                                    try await DomusHttpClient.shared.setVacationMode(enabled: enable)
                                    showBanner(enable ? "Modalità Vacanza attivata (jitter ±15 min)" : "Modalità Vacanza disattivata")
                                } catch {
                                    showBanner("Errore aggiornamento simulatore")
                                }
                            }
                        }
                    ))

                    HStack {
                        Image(systemName: "shuffle")
                            .foregroundColor(.orange)
                        VStack(alignment: .leading, spacing: 2) {
                            Text("Jitter Temporale Casuale")
                                .font(.subheadline)
                            Text("±\(vacationJitterMinutes) minuti di variazione stocastica anti-effrazione")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }

                    ForEach(vacationActions) { act in
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(act.entityId)
                                    .font(.subheadline.bold())
                                Text("Azione: \(act.command)")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                            Spacer()
                            Text(act.typicalTime)
                                .font(.system(.caption, design: .monospaced).bold())
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(Color.cyan.opacity(0.15))
                                .cornerRadius(6)
                        }
                    }

                    if isAddingVacationAction {
                        VStack(alignment: .leading, spacing: 8) {
                            TextField("Entity ID (es. light.studio)", text: $newActionEntity)
                                .textFieldStyle(.roundedBorder)
                                .font(.caption)
                            HStack {
                                TextField("Comando (turn_on/close)", text: $newActionCommand)
                                    .textFieldStyle(.roundedBorder)
                                    .font(.caption)
                                TextField("Ora (HH:mm)", text: $newActionTime)
                                    .textFieldStyle(.roundedBorder)
                                    .font(.caption)
                            }
                            HStack {
                                Button("Salva Azione") {
                                    let newAct = VacationActionDTO(entityId: newActionEntity, command: newActionCommand, typicalTime: newActionTime)
                                    vacationActions.append(newAct)
                                    isAddingVacationAction = false
                                    Task {
                                        try? await DomusHttpClient.shared.registerVacationAction(
                                            entityId: newActionEntity,
                                            command: newActionCommand,
                                            typicalTime: newActionTime
                                        )
                                        showBanner("Azione vacanza registrata!")
                                    }
                                }
                                .font(.caption.bold())
                                .foregroundColor(.green)

                                Spacer()

                                Button("Annulla") {
                                    isAddingVacationAction = false
                                }
                                .font(.caption)
                                .foregroundColor(.secondary)
                            }
                        }
                        .padding(.vertical, 4)
                    } else {
                        Button(action: { isAddingVacationAction = true }) {
                            HStack {
                                Image(systemName: "plus.circle.fill")
                                Text("Aggiungi Routine Replay")
                            }
                            .font(.caption.bold())
                        }
                        .foregroundColor(.cyan)
                    }
                }

                // ── SEZIONE 5: Pendolarismo & ETA Rientro Casa ──
                Section(header: Label("Pendolarismo & ETA Casa (Pre-clima)", systemImage: "car.fill")) {
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("Stato Tragitto Casa")
                                .font(.subheadline)
                            Text(commuteStatus?.phase.capitalized ?? "In monitoraggio")
                                .font(.headline.bold())
                                .foregroundColor(.cyan)
                        }
                        Spacer()
                        if let eta = commuteStatus?.etaMinutes, eta > 0 {
                            VStack(alignment: .trailing, spacing: 2) {
                                Text("\(eta) min")
                                    .font(.title2.bold())
                                    .foregroundColor(.green)
                                Text("all'arrivo")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                        }
                    }

                    if let dist = commuteStatus?.distanceKm, dist > 0 {
                        HStack {
                            Text("Distanza stimata:")
                                .font(.caption)
                            Spacer()
                            Text(String(format: "%.1f km", dist))
                                .font(.caption.bold())
                        }
                    }

                    HStack {
                        Image(systemName: "thermometer.snowflake")
                            .foregroundColor(.blue)
                        Text("Pre-riscaldamento automatico attivo se ETA < 20 min")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }
            }
            .navigationTitle("Presenza & Automazioni")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .task {
                await loadVacationAndCommute()
            }
            .refreshable {
                await loadVacationAndCommute()
            }
        }
    }

    private func loadVacationAndCommute() async {
        if let vac = try? await DomusHttpClient.shared.fetchVacationMode() {
            self.isVacationModeEnabled = vac.enabled
            if let j = vac.jitterMinutes { self.vacationJitterMinutes = j }
        }
        if let com = try? await DomusHttpClient.shared.fetchCommuteStatus(userId: "user_main") {
            self.commuteStatus = com
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
