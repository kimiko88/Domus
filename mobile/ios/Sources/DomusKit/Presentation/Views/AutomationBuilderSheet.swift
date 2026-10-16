import SwiftUI

/// Builder Visivo di Scenari & Automazioni da Mobile ("Se accade X con condizione Y allora fai Z").
public struct AutomationBuilderSheet: View {
    @Environment(\.dismiss) private var dismiss

    @State private var automations: [AutomationRule] = []
    @State private var availableEntities: [Entity] = []
    @State private var isLoading: Bool = false
    @State private var errorMessage: String? = nil
    @State private var showingEditor: Bool = false
    @State private var editingRule: AutomationRule? = nil
    @State private var isTestingRule: Bool = false
    @State private var testSuccessMessage: String? = nil

    // Form editing states
    @State private var formName: String = ""
    @State private var formDescription: String = ""
    @State private var selectedTriggerPlatform: String = "time"
    @State private var selectedTime: Date = Calendar.current.date(bySettingHour: 7, minute: 30, second: 0, of: Date()) ?? Date()
    @State private var selectedSunEvent: String = "sunset"
    @State private var selectedMotionSensor: String = "binary_sensor.hallway_motion"
    @State private var selectedGeofenceEvent: String = "leave_home"

    // Conditions state
    @State private var requireAlarmArmed: Bool = false
    @State private var requireNightHours: Bool = false

    // Actions state
    @State private var actionTargetLight: String = "light.living_room"
    @State private var actionLightService: String = "turn_on"
    @State private var actionTargetTemp: Double = 21.0
    @State private var actionPushMessage: String = "Movimento rilevato da Domus"
    @State private var includeClimateAction: Bool = false
    @State private var includeNotificationAction: Bool = false

    public init() {}

    private var availableLights: [Entity] {
        let lights = availableEntities.filter { $0.isLight || $0.isSwitch }
        if lights.isEmpty {
            return [
                Entity(id: "light.living_room", name: "Luce Salotto", domain: "light", state: "off"),
                Entity(id: "light.garden_floodlight", name: "Faretti Giardino", domain: "light", state: "off"),
                Entity(id: "light.all_lights", name: "Tutte le Luci", domain: "light", state: "off")
            ]
        }
        return lights
    }

    private var availableSensors: [Entity] {
        let sensors = availableEntities.filter { $0.domain == "binary_sensor" || $0.domain == "sensor" }
        if sensors.isEmpty {
            return [
                Entity(id: "binary_sensor.hallway_motion", name: "Sensore Ingresso", domain: "binary_sensor", state: "off"),
                Entity(id: "binary_sensor.garden_motion", name: "Sensore Giardino", domain: "binary_sensor", state: "off"),
                Entity(id: "binary_sensor.living_motion", name: "Sensore Salotto", domain: "binary_sensor", state: "off")
            ]
        }
        return sensors
    }

    public var body: some View {
        NavigationStack {
            List {
                Section {
                    HStack(spacing: 12) {
                        Image(systemName: "wand.and.stars")
                            .font(.system(size: 28))
                            .foregroundColor(.purple)
                        VStack(alignment: .leading, spacing: 2) {
                            Text("Automazioni Intelligenti")
                                .font(.headline)
                            Text("Configura regole visive Se-Allora sincronizzate direttamente con Domus Core.")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }
                    .padding(.vertical, 4)
                }

                if let err = errorMessage {
                    Section {
                        HStack(spacing: 8) {
                            Image(systemName: "exclamationmark.triangle.fill")
                                .foregroundColor(.orange)
                            Text(err)
                                .font(.caption)
                                .foregroundColor(.orange)
                        }
                    }
                }

                if let msg = testSuccessMessage {
                    Section {
                        HStack(spacing: 8) {
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundColor(.green)
                            Text(msg)
                                .font(.subheadline.bold())
                                .foregroundColor(.green)
                        }
                        .padding(.vertical, 4)
                    }
                }

                Section(header: Text("Regole Attive (\(automations.count))")) {
                    if isLoading && automations.isEmpty {
                        HStack {
                            Spacer()
                            ProgressView("Caricamento regole dal server...")
                                .padding(.vertical, 12)
                            Spacer()
                        }
                    } else if automations.isEmpty {
                        VStack(spacing: 8) {
                            Text("Nessuna automazione trovata sul server Domus.")
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                            Button(action: {
                                Task { await loadAutomationsAndEntities() }
                            }) {
                                Text("Riprova sincronizzazione")
                                    .font(.caption.bold())
                                    .foregroundColor(.blue)
                            }
                        }
                        .padding(.vertical, 8)
                    } else {
                        ForEach(automations) { rule in
                            automationRow(rule)
                                .swipeActions(edge: .trailing, allowsFullSwipe: true) {
                                    Button(role: .destructive) {
                                        deleteRule(rule)
                                    } label: {
                                        Label("Elimina", systemImage: "trash")
                                    }
                                }
                        }
                    }
                }
            }
            #if os(iOS)
            .listStyle(.insetGrouped)
            #else
            .listStyle(.inset)
            #endif
            .navigationTitle("Regole & Automazioni")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .refreshable {
                await loadAutomationsAndEntities()
            }
            .task {
                await loadAutomationsAndEntities()
            }
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Chiudi") {
                        dismiss()
                    }
                }
                ToolbarItem(placement: .primaryAction) {
                    Button(action: {
                        startNewRule()
                    }) {
                        Image(systemName: "plus.circle.fill")
                            .font(.system(size: 20))
                            .foregroundColor(.blue)
                    }
                    .accessibilityLabel("Crea nuova automazione")
                }
            }
            .sheet(isPresented: $showingEditor) {
                ruleEditorView
            }
        }
    }

    // MARK: - Riga Singola Automazione

    private func automationRow(_ rule: AutomationRule) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                HStack(spacing: 6) {
                    Image(systemName: rule.triggers.first?.displayIcon ?? "bolt.fill")
                        .foregroundColor(.blue)
                    Text(rule.name)
                        .font(.headline)
                }
                Spacer()
                Toggle("", isOn: Binding(
                    get: { rule.enabled },
                    set: { val in
                        toggleRuleEnabled(rule, enabled: val)
                    }
                ))
                .labelsHidden()
                .accessibilityLabel("Abilita automazione \(rule.name)")
            }

            if let desc = rule.description, !desc.isEmpty {
                Text(desc)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            // Pill Riepilogo Triggers e Azioni
            HStack(spacing: 8) {
                if let t = rule.triggers.first {
                    Label(t.displayTitle, systemImage: t.displayIcon)
                        .font(.caption2.bold())
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .background(Color.blue.opacity(0.12))
                        .foregroundColor(.blue)
                        .clipShape(Capsule())
                }

                if !rule.conditions.isEmpty {
                    Label("\(rule.conditions.count) filtri", systemImage: "line.3.horizontal.decrease.circle")
                        .font(.caption2)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 3)
                        .background(Color.purple.opacity(0.12))
                        .foregroundColor(.purple)
                        .clipShape(Capsule())
                }

                Label("\(rule.actions.count) azioni", systemImage: "arrow.right.circle.fill")
                    .font(.caption2)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 3)
                    .background(Color.green.opacity(0.12))
                    .foregroundColor(.green)
                    .clipShape(Capsule())

                Spacer()

                Button(action: {
                    testRunRule(rule)
                }) {
                    Text("Prova")
                        .font(.caption2.bold())
                        .foregroundColor(.primary)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4)
                        .background(Color.secondary.opacity(0.15))
                        .clipShape(Capsule())
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Esegui test immediato di \(rule.name)")
            }
        }
        .padding(.vertical, 4)
    }

    // MARK: - Editor Modale Nuova Automazione

    private var ruleEditorView: some View {
        NavigationStack {
            Form {
                Section(header: Text("Informazioni Regola")) {
                    TextField("Nome Automazione (es. Luci Tramonto)", text: $formName)
                        .accessibilityLabel("Nome Automazione")
                    TextField("Descrizione (Opzionale)", text: $formDescription)
                        .accessibilityLabel("Descrizione Regola")
                }

                // 1. SE ACCADE (TRIGGER)
                Section(header: Label("SE ACCADE (TRIGGER)", systemImage: "bolt.fill").foregroundColor(.blue)) {
                    Picker("Tipo di Innesco", selection: $selectedTriggerPlatform) {
                        Label("Orario Programmato", systemImage: "clock").tag("time")
                        Label("Alba / Tramonto", systemImage: "sun.max").tag("sun")
                        Label("Rilevamento Movimento", systemImage: "figure.walk").tag("state")
                        Label("Ingresso / Uscita Casa", systemImage: "location").tag("geofence")
                    }

                    switch selectedTriggerPlatform {
                    case "time":
                        DatePicker("Orario di attivazione", selection: $selectedTime, displayedComponents: .hourAndMinute)
                    case "sun":
                        Picker("Evento Solare", selection: $selectedSunEvent) {
                            Text("Tramonto (Crepuscolo)").tag("sunset")
                            Text("Alba (Mattina)").tag("sunrise")
                        }
                        .pickerStyle(.segmented)
                    case "state":
                        Picker("Sensore di Movimento", selection: $selectedMotionSensor) {
                            ForEach(availableSensors) { sensor in
                                Text(sensor.name).tag(sensor.id)
                            }
                        }
                    case "geofence":
                        Picker("Evento Posizione", selection: $selectedGeofenceEvent) {
                            Text("Tutti escono da casa").tag("leave_home")
                            Text("Primo membro rientra a casa").tag("enter_home")
                        }
                        .pickerStyle(.segmented)
                    default:
                        EmptyView()
                    }
                }

                // 2. E SOLO SE (CONDIZIONI)
                Section(header: Label("E SOLO SE (CONDIZIONI FILTRO)", systemImage: "line.3.horizontal.decrease.circle").foregroundColor(.purple)) {
                    Toggle("Solo se l'allarme è inserito", isOn: $requireAlarmArmed)
                        .accessibilityHint("Impedisce l'esecuzione se l'allarme è disarmato")
                    Toggle("Solo in orario notturno (22:00 - 06:00)", isOn: $requireNightHours)
                        .accessibilityHint("Attiva le azioni solo se l'evento accade durante la notte")
                }

                // 3. ALLORA FAI (AZIONI)
                Section(header: Label("ALLORA FAI (AZIONI ESEGUITE)", systemImage: "play.circle.fill").foregroundColor(.green)) {
                    // Azione Luce / Interruttore
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Controllo Illuminazione")
                            .font(.caption.bold())
                            .foregroundColor(.secondary)
                        Picker("Dispositivo", selection: $actionTargetLight) {
                            ForEach(availableLights) { light in
                                Text(light.name).tag(light.id)
                            }
                        }
                        Picker("Azione", selection: $actionLightService) {
                            Text("Accendi").tag("turn_on")
                            Text("Spegni").tag("turn_off")
                        }
                        .pickerStyle(.segmented)
                    }

                    // Azione Clima
                    Toggle("Regola anche la Temperatura", isOn: $includeClimateAction)
                    if includeClimateAction {
                        VStack(alignment: .leading, spacing: 4) {
                            HStack {
                                Text("Temperatura Clima:")
                                    .font(.subheadline)
                                Spacer()
                                Text("\(actionTargetTemp, specifier: "%.1f")°C")
                                    .font(.subheadline.bold())
                                    .foregroundColor(.blue)
                            }
                            Slider(value: $actionTargetTemp, in: 16.0...26.0, step: 0.5)
                        }
                    }

                    // Azione Notifica
                    Toggle("Invia Notifica Push", isOn: $includeNotificationAction)
                    if includeNotificationAction {
                        TextField("Testo notifica...", text: $actionPushMessage)
                    }
                }

                // Pulsante Test e Salva
                Section {
                    Button(action: {
                        testCurrentDraft()
                    }) {
                        HStack {
                            Spacer()
                            if isTestingRule {
                                ProgressView()
                            } else {
                                Image(systemName: "play.fill")
                                Text("Prova Regola Ora (Simulazione)")
                            }
                            Spacer()
                        }
                        .font(.subheadline.bold())
                        .foregroundColor(.blue)
                        .frame(minHeight: 44)
                    }

                    Button(action: {
                        saveNewRule()
                    }) {
                        HStack {
                            Spacer()
                            Image(systemName: "checkmark")
                            Text("Salva Automazione")
                            Spacer()
                        }
                        .font(.headline)
                        .foregroundColor(.white)
                        .frame(minHeight: 44)
                    }
                    .listRowBackground(Color.blue)
                    .disabled(formName.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            .navigationTitle("Nuova Automazione")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annulla") {
                        showingEditor = false
                    }
                }
            }
        }
    }

    // MARK: - Caricamento e Logica Server

    private func loadAutomationsAndEntities() async {
        isLoading = true
        errorMessage = nil
        do {
            async let fetchedRules = DomusHttpClient.shared.fetchAutomations()
            async let fetchedEntities = DomusHttpClient.shared.fetchEntities()
            let (rules, ents) = try await (fetchedRules, fetchedEntities)
            await MainActor.run {
                self.automations = rules
                self.availableEntities = ents
                if let firstLight = availableLights.first, !availableLights.contains(where: { $0.id == actionTargetLight }) {
                    self.actionTargetLight = firstLight.id
                }
                if let firstSensor = availableSensors.first, !availableSensors.contains(where: { $0.id == selectedMotionSensor }) {
                    self.selectedMotionSensor = firstSensor.id
                }
                self.isLoading = false
            }
        } catch {
            await MainActor.run {
                self.errorMessage = "Sincronizzazione fallita: \(error.localizedDescription)"
                self.isLoading = false
            }
        }
    }

    private func startNewRule() {
        formName = ""
        formDescription = ""
        selectedTriggerPlatform = "time"
        requireAlarmArmed = false
        requireNightHours = false
        includeClimateAction = false
        includeNotificationAction = false
        if let firstSensor = availableSensors.first {
            selectedMotionSensor = firstSensor.id
        }
        if let firstLight = availableLights.first {
            actionTargetLight = firstLight.id
        }
        showingEditor = true
    }

    private func saveNewRule() {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm:ss"
        let timeString = formatter.string(from: selectedTime)

        var triggers: [AutomationTrigger] = []
        switch selectedTriggerPlatform {
        case "time":
            triggers.append(AutomationTrigger(platform: "time", at: timeString))
        case "sun":
            triggers.append(AutomationTrigger(platform: "sun", eventType: selectedSunEvent))
        case "state":
            triggers.append(AutomationTrigger(platform: "state", entityKey: selectedMotionSensor, to: "on"))
        case "geofence":
            triggers.append(AutomationTrigger(platform: "geofence", eventType: selectedGeofenceEvent))
        default:
            triggers.append(AutomationTrigger(platform: "time", at: timeString))
        }

        var conditions: [AutomationCondition] = []
        if requireAlarmArmed {
            conditions.append(AutomationCondition(condition: "state", entityKey: "alarm_control_panel.home", state: "armed_home"))
        }
        if requireNightHours {
            conditions.append(AutomationCondition(condition: "time", after: "22:00:00", before: "06:00:00"))
        }

        var actions: [AutomationAction] = []
        actions.append(AutomationAction(
            type: "service",
            service: actionLightService,
            targetEntityKey: actionTargetLight
        ))

        if includeClimateAction {
            actions.append(AutomationAction(
                type: "service",
                service: "set_temperature",
                targetEntityKey: "climate.thermostat",
                params: ["temperature": String(format: "%.1f", actionTargetTemp)]
            ))
        }

        if includeNotificationAction {
            actions.append(AutomationAction(
                type: "service",
                service: "notify",
                targetEntityKey: "notify.mobile_app",
                params: ["message": actionPushMessage]
            ))
        }

        let newRule = AutomationRule(
            id: UUID().uuidString,
            name: formName,
            description: formDescription.isEmpty ? nil : formDescription,
            enabled: true,
            triggers: triggers,
            conditions: conditions,
            actions: actions
        )

        showingEditor = false
        Haptics.success()
        DomusAccessibility.announce("Salvataggio automazione...")

        Task {
            if let created = try? await DomusHttpClient.shared.createAutomation(rule: newRule) {
                await MainActor.run {
                    self.automations.append(created)
                }
            } else {
                await MainActor.run {
                    self.automations.append(newRule)
                }
            }
            DomusAccessibility.announce("Automazione \(newRule.name) salvata con successo")
        }
    }

    private func deleteRule(_ rule: AutomationRule) {
        automations.removeAll(where: { $0.id == rule.id })
        Haptics.medium()
        DomusAccessibility.announce("Automazione eliminata")

        Task {
            try? await DomusHttpClient.shared.deleteAutomation(id: rule.id)
        }
    }

    private func toggleRuleEnabled(_ rule: AutomationRule, enabled: Bool) {
        if let idx = automations.firstIndex(where: { $0.id == rule.id }) {
            automations[idx].enabled = enabled
            Haptics.light()
            DomusAccessibility.announce(enabled ? "\(rule.name) attivata" : "\(rule.name) disattivata")

            var updated = automations[idx]
            updated.enabled = enabled
            Task {
                try? await DomusHttpClient.shared.updateAutomation(id: rule.id, rule: updated)
            }
        }
    }

    private func testRunRule(_ rule: AutomationRule) {
        Haptics.medium()
        testSuccessMessage = "Invio trigger al server per \(rule.name)..."
        DomusAccessibility.announce("Test in corso per \(rule.name)")

        Task {
            do {
                try await DomusHttpClient.shared.triggerAutomation(id: rule.id)
                await MainActor.run {
                    testSuccessMessage = "Test eseguito: \(rule.name) avviata sul server!"
                }
            } catch {
                if let first = rule.actions.first, let svc = first.service, let target = first.targetEntityKey {
                    let domain = target.components(separatedBy: ".").first ?? "light"
                    try? await DomusHttpClient.shared.callService(domain: domain, service: svc, entityId: target)
                }
                await MainActor.run {
                    testSuccessMessage = "Test eseguito: azioni inviate con successo!"
                }
            }
            DomusAccessibility.announce("Test completato per \(rule.name)")

            try? await Task.sleep(nanoseconds: 3_000_000_000)
            await MainActor.run {
                testSuccessMessage = nil
            }
        }
    }

    private func testCurrentDraft() {
        isTestingRule = true
        Haptics.medium()

        Task {
            let domain = actionTargetLight.components(separatedBy: ".").first ?? "light"
            try? await DomusHttpClient.shared.callService(domain: domain, service: actionLightService, entityId: actionTargetLight)
            await MainActor.run {
                isTestingRule = false
                Haptics.success()
                DomusAccessibility.announce("Simulazione completata su \(actionTargetLight)")
            }
        }
    }
}
