import SwiftUI

/// Scheda Armadietto Medicinali — Gestione farmaci di casa, scadenze, PAO, scorte e catena del freddo.
public struct MedicineCabinetSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.accessibilityDifferentiateWithoutColor) private var differentiateWithoutColor

    @State private var medicines: [Medicine] = []
    @State private var alerts: [MedicineAlert] = []
    @State private var isLoading: Bool = true
    @State private var searchQuery: String = ""
    @State private var selectedLocation: MedicineLocation? = nil
    @State private var selectedFilter: MedicineFilter = .all
    @State private var showAddMedicineSheet: Bool = false
    @State private var errorMessage: String? = nil

    private let client = DomusHttpClient.shared

    public enum MedicineFilter: String, CaseIterable, Identifiable {
        case all = "Tutti"
        case expiring = "In Scadenza"
        case lowStock = "Scorta Bassa"
        case coldChain = "Frigo / Freddo"

        public var id: String { rawValue }

        public var localizedName: String {
            switch self {
            case .all: return NSLocalizedString("common_all", comment: "Tutti")
            case .expiring: return NSLocalizedString("medicine_filter_expiring", comment: "In Scadenza")
            case .lowStock: return NSLocalizedString("medicine_filter_low_stock", comment: "Scorta Bassa")
            case .coldChain: return NSLocalizedString("medicine_filter_cold_chain", comment: "Frigo / Freddo")
            }
        }
    }

    public init() {}

    public var body: some View {
        NavigationStack {
            ZStack {
                Color.domusSystemGroupedBackground.ignoresSafeArea()

                if isLoading {
                    ProgressView(NSLocalizedString("common_loading", comment: "Caricamento..."))
                        .progressViewStyle(.circular)
                } else {
                    ScrollView {
                        VStack(spacing: 20) {
                            // Banner Avvisi Critici (se presenti)
                            if !alerts.isEmpty {
                                alertsBannerView
                            }

                            // Filtri & Ricerca
                            searchAndFilterSection

                            // Elenco Farmaci
                            if filteredMedicines.isEmpty {
                                emptyStateView
                            } else {
                                LazyVStack(spacing: 14) {
                                    ForEach(filteredMedicines) { med in
                                        medicineCard(med)
                                    }
                                }
                                .padding(.horizontal)
                            }
                        }
                        .padding(.vertical)
                    }
                    .refreshable {
                        await loadData()
                    }
                }
            }
            .navigationTitle(NSLocalizedString("medicine_cabinet_title", comment: "Armadietto Medicinali"))
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(action: { dismiss() }) {
                        Text(NSLocalizedString("common_close", comment: "Chiudi"))
                            .font(.body.weight(.medium))
                    }
                    .accessibilityLabel(NSLocalizedString("common_close", comment: "Chiudi"))
                }

                ToolbarItem(placement: .primaryAction) {
                    Button(action: {
                        Haptics.light()
                        showAddMedicineSheet = true
                    }) {
                        Image(systemName: "plus")
                            .font(.body.weight(.bold))
                    }
                    .accessibilityLabel(NSLocalizedString("medicine_add_button", comment: "Aggiungi Farmaco"))
                }
            }
            .sheet(isPresented: $showAddMedicineSheet) {
                AddMedicineSheet(onSaved: { newMed in
                    medicines.insert(newMed, at: 0)
                    Task { await loadData() }
                })
            }
            .task {
                await loadData()
            }
        }
    }

    // MARK: - Banner Avvisi Critici

    private var alertsBannerView: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundColor(.red)
                Text(NSLocalizedString("medicine_alerts_banner_title", comment: "Avvisi Farmaci"))
                    .font(.headline)
                    .foregroundColor(.primary)
                Spacer()
                Text("\(alerts.count)")
                    .font(.caption.bold())
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color.red.opacity(0.15))
                    .foregroundColor(.red)
                    .clipShape(Capsule())
            }

            ForEach(alerts.prefix(3)) { alert in
                HStack(spacing: 10) {
                    Circle()
                        .fill(alert.isCritical ? Color.red : Color.orange)
                        .frame(width: 8, height: 8)
                    Text(alert.message)
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .lineLimit(2)
                }
            }
        }
        .padding()
        .background(Color.red.opacity(0.08))
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .overlay(
            RoundedRectangle(cornerRadius: 16)
                .stroke(Color.red.opacity(0.3), lineWidth: 1)
        )
        .padding(.horizontal)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(NSLocalizedString("medicine_alerts_banner_title", comment: "Avvisi Farmaci")): \(alerts.count)")
    }

    // MARK: - Filtri e Ricerca

    private var searchAndFilterSection: some View {
        VStack(spacing: 12) {
            // Barra di ricerca
            HStack {
                Image(systemName: "magnifyingglass")
                    .foregroundColor(.secondary)
                TextField(NSLocalizedString("medicine_search_placeholder", comment: "Cerca farmaco o principio attivo..."), text: $searchQuery)
                    .textFieldStyle(.plain)
                if !searchQuery.isEmpty {
                    Button(action: { searchQuery = "" }) {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundColor(.secondary)
                    }
                }
            }
            .padding(10)
            .background(Color.secondary.opacity(0.12))
            .clipShape(RoundedRectangle(cornerRadius: 12))
            .padding(.horizontal)

            // Chip filtri stato
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(MedicineFilter.allCases) { filter in
                        let isSelected = selectedFilter == filter
                        Button(action: {
                            Haptics.selection()
                            selectedFilter = filter
                        }) {
                            Text(filter.localizedName)
                                .font(.footnote.weight(isSelected ? .bold : .medium))
                                .padding(.horizontal, 14)
                                .padding(.vertical, 8)
                                .background(isSelected ? Color.blue : Color.secondary.opacity(0.15))
                                .foregroundColor(isSelected ? .white : .primary)
                                .clipShape(Capsule())
                        }
                        .accessibilityAddTraits(isSelected ? .isSelected : [])
                    }
                }
                .padding(.horizontal)
            }

            // Chip posizioni
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    let allLocSelected = selectedLocation == nil
                    Button(action: {
                        Haptics.selection()
                        selectedLocation = nil
                    }) {
                        Text(NSLocalizedString("medicine_all_locations", comment: "Tutte le stanze"))
                            .font(.caption.weight(allLocSelected ? .bold : .regular))
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(allLocSelected ? Color.secondary.opacity(0.3) : Color.secondary.opacity(0.1))
                            .foregroundColor(.primary)
                            .clipShape(Capsule())
                    }

                    ForEach(MedicineLocation.allCases, id: \.self) { loc in
                        let isLocSelected = selectedLocation == loc
                        Button(action: {
                            Haptics.selection()
                            selectedLocation = isLocSelected ? nil : loc
                        }) {
                            HStack(spacing: 4) {
                                Image(systemName: loc.iconName)
                                    .font(.caption2)
                                Text(loc.localizedName)
                                    .font(.caption.weight(isLocSelected ? .bold : .regular))
                            }
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(isLocSelected ? Color.secondary.opacity(0.3) : Color.secondary.opacity(0.1))
                            .foregroundColor(.primary)
                            .clipShape(Capsule())
                        }
                    }
                }
                .padding(.horizontal)
            }
        }
    }

    // MARK: - Card Farmaco

    private func medicineCard(_ med: Medicine) -> some View {
        let isExpired = med.isExpired()
        let isExpiring = med.isExpiringSoon(daysThreshold: 30)
        let isLow = med.isLowStock

        return VStack(alignment: .leading, spacing: 12) {
            // Header card con nome e badge scadenza
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(med.name)
                        .font(.headline)
                        .foregroundColor(.primary)

                    if let active = med.activeIngredient, !active.isEmpty {
                        Text(active)
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }

                    if let dosage = med.dosage, !dosage.isEmpty {
                        Text(dosage)
                            .font(.caption2)
                            .foregroundColor(.blue)
                    }
                }

                Spacer()

                // Badge scadenza
                VStack(alignment: .trailing, spacing: 4) {
                    if isExpired {
                        statusBadge(text: NSLocalizedString("medicine_status_expired", comment: "Scaduto"), color: .red)
                    } else if isExpiring {
                        let days = med.daysUntilExpiration() ?? 0
                        statusBadge(text: "\(days) gg", color: days <= 7 ? .red : .orange)
                    } else {
                        statusBadge(text: NSLocalizedString("medicine_status_valid", comment: "Valido"), color: .green)
                    }

                    if med.requiresColdChain {
                        HStack(spacing: 2) {
                            Image(systemName: "snowflake")
                                .font(.caption2)
                            Text("Frigo")
                                .font(.caption2.bold())
                        }
                        .foregroundColor(.cyan)
                    }
                }
            }

            Divider()

            // Dettagli scadenza, posizione, PAO
            HStack(spacing: 16) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(NSLocalizedString("medicine_field_expiration", comment: "Scadenza"))
                        .font(.caption2)
                        .foregroundColor(.secondary)
                    Text(med.effectiveExpirationDate)
                        .font(.footnote.weight(.semibold))
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text(NSLocalizedString("medicine_field_location", comment: "Posizione"))
                        .font(.caption2)
                        .foregroundColor(.secondary)
                    HStack(spacing: 4) {
                        Image(systemName: med.location.iconName)
                            .font(.caption2)
                        Text(med.location.localizedName)
                            .font(.footnote.weight(.semibold))
                    }
                }

                if med.paoDays > 0 {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("PAO")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                        Text(med.isOpened ? "Aperto (\(med.paoDays)gg)" : "Sigillato")
                            .font(.footnote.weight(.semibold))
                            .foregroundColor(med.isOpened ? .orange : .secondary)
                    }
                }

                Spacer()
            }

            // Barra Quantità & Azioni Rapide
            HStack {
                HStack(spacing: 4) {
                    Text("\(String(format: "%.0f", med.quantity))")
                        .font(.headline.bold())
                        .foregroundColor(isLow ? .red : .primary)
                    Text(med.unit.localizedName)
                        .font(.caption)
                        .foregroundColor(.secondary)
                }

                if isLow {
                    Text("• " + NSLocalizedString("medicine_status_low_stock", comment: "Scorta bassa"))
                        .font(.caption.bold())
                        .foregroundColor(.red)
                }

                Spacer()

                // Pulsante Apri Confezione se non aperta e ha PAO
                if med.paoDays > 0 && !med.isOpened {
                    Button(action: {
                        Task { await openMedicine(id: med.id) }
                    }) {
                        Text(NSLocalizedString("medicine_action_open", comment: "Apri"))
                            .font(.caption.bold())
                            .padding(.horizontal, 10)
                            .padding(.vertical, 6)
                            .background(Color.orange.opacity(0.2))
                            .foregroundColor(.orange)
                            .clipShape(Capsule())
                    }
                }

                // Pulsante Assumi 1 Dose
                Button(action: {
                    Task { await consumeDose(id: med.id) }
                }) {
                    HStack(spacing: 4) {
                        Image(systemName: "cross.fill")
                            .font(.caption2)
                        Text(NSLocalizedString("medicine_action_take_dose", comment: "Assumi 1"))
                            .font(.caption.bold())
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .background(Color.blue)
                    .foregroundColor(.white)
                    .clipShape(Capsule())
                }
                .accessibilityLabel("\(NSLocalizedString("medicine_action_take_dose", comment: "Assumi 1")): \(med.name)")
            }
        }
        .padding()
        .background(Color.secondary.opacity(0.08))
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .overlay(
            RoundedRectangle(cornerRadius: 16)
                .stroke(Color.secondary.opacity(0.15), lineWidth: 1)
        )
    }

    private func statusBadge(text: String, color: Color) -> some View {
        Text(text)
            .font(.caption.bold())
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(color.opacity(0.15))
            .foregroundColor(color)
            .clipShape(Capsule())
            .overlay(
                Capsule().stroke(color.opacity(differentiateWithoutColor ? 0.8 : 0), lineWidth: 1)
            )
    }

    // MARK: - Stato Vuoto

    private var emptyStateView: some View {
        VStack(spacing: 12) {
            Image(systemName: "cross.case")
                .font(.system(size: 44))
                .foregroundColor(.secondary)
            Text(NSLocalizedString("medicine_empty_state_title", comment: "Nessun medicinale trovato"))
                .font(.headline)
            Text(NSLocalizedString("medicine_empty_state_subtitle", comment: "Aggiungi il tuo primo farmaco scansionando il codice a barre o inserendolo manualmente."))
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 32)
        }
        .padding(.vertical, 60)
    }

    // MARK: - Filtri Computati

    private var filteredMedicines: [Medicine] {
        medicines.filter { med in
            // Ricerca testuale
            if !searchQuery.isEmpty {
                let q = searchQuery.lowercased()
                let nameMatches = med.name.lowercased().contains(q)
                let activeMatches = med.activeIngredient?.lowercased().contains(q) ?? false
                let aicMatches = med.aic?.lowercased().contains(q) ?? false
                if !nameMatches && !activeMatches && !aicMatches {
                    return false
                }
            }

            // Filtro posizione
            if let loc = selectedLocation, med.location != loc {
                return false
            }

            // Filtro stato
            switch selectedFilter {
            case .all:
                return true
            case .expiring:
                return med.isExpired() || med.isExpiringSoon(daysThreshold: 30)
            case .lowStock:
                return med.isLowStock
            case .coldChain:
                return med.requiresColdChain || med.location == .frigo
            }
        }
    }

    // MARK: - Operazioni di Rete

    private func loadData() async {
        isLoading = medicines.isEmpty
        errorMessage = nil
        do {
            async let fetchedMeds = client.fetchMedicines()
            async let fetchedAlerts = client.fetchMedicineAlerts()
            let (m, a) = try await (fetchedMeds, fetchedAlerts)
            medicines = m
            alerts = a
        } catch {
            errorMessage = error.localizedDescription
        }
        isLoading = false
    }

    private func consumeDose(id: String) async {
        Haptics.light()
        do {
            let updated = try await client.consumeMedicine(id: id, amount: 1.0)
            if let idx = medicines.firstIndex(where: { $0.id == id }) {
                medicines[idx] = updated
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func openMedicine(id: String) async {
        Haptics.selection()
        do {
            let updated = try await client.openMedicine(id: id)
            if let idx = medicines.firstIndex(where: { $0.id == id }) {
                medicines[idx] = updated
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

// MARK: - Sheet Aggiunta Farmaco

private struct AddMedicineSheet: View {
    @Environment(\.dismiss) private var dismiss
    var onSaved: (Medicine) -> Void

    @State private var name: String = ""
    @State private var activeIngredient: String = ""
    @State private var dosage: String = ""
    @State private var aic: String = ""
    @State private var expirationDate: Date = Calendar.current.date(byAdding: .year, value: 1, to: Date()) ?? Date()
    @State private var paoDays: String = "0"
    @State private var quantity: String = "1"
    @State private var minThreshold: String = "1"
    @State private var location: MedicineLocation = .armadiettoBagno
    @State private var unit: MedicineUnit = .compresse
    @State private var requiresColdChain: Bool = false
    @State private var prescriptionRequired: Bool = false
    @State private var notes: String = ""
    @State private var datamatrixCode: String = ""
    @State private var isParsingCode: Bool = false
    @State private var isSaving: Bool = false
    @State private var errorMessage: String? = nil

    private let client = DomusHttpClient.shared

    var body: some View {
        NavigationStack {
            Form {
                Section(header: Text(NSLocalizedString("medicine_section_scan", comment: "Scansione Codice AIC / DataMatrix"))) {
                    HStack {
                        TextField(NSLocalizedString("medicine_field_barcode_placeholder", comment: "Codice DataMatrix o AIC..."), text: $datamatrixCode)
                        if isParsingCode {
                            ProgressView()
                                .padding(.leading, 4)
                        } else {
                            Button(NSLocalizedString("medicine_action_parse", comment: "Analizza")) {
                                Task { await parseCode() }
                            }
                            .disabled(datamatrixCode.isEmpty)
                        }
                    }
                }

                Section(header: Text(NSLocalizedString("medicine_section_info", comment: "Dati Principali"))) {
                    TextField(NSLocalizedString("medicine_field_name", comment: "Nome Commerciale *"), text: $name)
                    TextField(NSLocalizedString("medicine_field_active", comment: "Principio Attivo"), text: $activeIngredient)
                    TextField(NSLocalizedString("medicine_field_dosage", comment: "Dosaggio (es. 500mg, 10ml)"), text: $dosage)
                    TextField(NSLocalizedString("medicine_field_aic", comment: "Codice AIC (9 cifre)"), text: $aic)
                    #if os(iOS)
                    .keyboardType(.numberPad)
                    #endif

                    DatePicker(
                        NSLocalizedString("medicine_field_expiration", comment: "Data di Scadenza"),
                        selection: $expirationDate,
                        displayedComponents: .date
                    )
                }

                Section(header: Text(NSLocalizedString("medicine_section_storage", comment: "Conservazione & Scorte"))) {
                    Picker(NSLocalizedString("medicine_field_location", comment: "Posizione"), selection: $location) {
                        ForEach(MedicineLocation.allCases, id: \.self) { loc in
                            HStack {
                                Image(systemName: loc.iconName)
                                Text(loc.localizedName)
                            }.tag(loc)
                        }
                    }

                    Picker(NSLocalizedString("medicine_field_unit", comment: "Forma Farmaceutica"), selection: $unit) {
                        ForEach(MedicineUnit.allCases, id: \.self) { u in
                            Text(u.localizedName).tag(u)
                        }
                    }

                    HStack {
                        Text(NSLocalizedString("medicine_field_quantity", comment: "Quantità"))
                        Spacer()
                        TextField("1", text: $quantity)
                            .multilineTextAlignment(.trailing)
                            #if os(iOS)
                            .keyboardType(.decimalPad)
                            #endif
                            .frame(width: 80)
                    }

                    HStack {
                        Text(NSLocalizedString("medicine_field_min_threshold", comment: "Scorta Minima"))
                        Spacer()
                        TextField("1", text: $minThreshold)
                            .multilineTextAlignment(.trailing)
                            #if os(iOS)
                            .keyboardType(.decimalPad)
                            #endif
                            .frame(width: 80)
                    }

                    HStack {
                        Text(NSLocalizedString("medicine_field_pao", comment: "PAO (Giorni dopo apertura)"))
                        Spacer()
                        TextField("0", text: $paoDays)
                            .multilineTextAlignment(.trailing)
                            #if os(iOS)
                            .keyboardType(.numberPad)
                            #endif
                            .frame(width: 80)
                    }

                    Toggle(NSLocalizedString("medicine_field_cold_chain", comment: "Richiede Catena del Freddo (2-8°C)"), isOn: $requiresColdChain)
                    Toggle(NSLocalizedString("medicine_field_prescription", comment: "Obbligo di Ricetta Medica"), isOn: $prescriptionRequired)
                }

                Section(header: Text(NSLocalizedString("medicine_section_notes", comment: "Note & Posologia"))) {
                    TextField(NSLocalizedString("medicine_field_notes", comment: "Indicazioni, posologia, allergie..."), text: $notes)
                }

                if let err = errorMessage {
                    Section {
                        Text(err)
                            .foregroundColor(.red)
                            .font(.caption)
                    }
                }
            }
            .navigationTitle(NSLocalizedString("medicine_add_title", comment: "Nuovo Medicinale"))
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(NSLocalizedString("common_cancel", comment: "Annulla")) {
                        dismiss()
                    }
                }

                ToolbarItem(placement: .primaryAction) {
                    Button(action: {
                        Task { await saveMedicine() }
                    }) {
                        if isSaving {
                            ProgressView()
                        } else {
                            Text(NSLocalizedString("common_save", comment: "Salva"))
                                .bold()
                        }
                    }
                    .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty || isSaving)
                }
            }
        }
    }

    private func parseCode() async {
        isParsingCode = true
        errorMessage = nil
        do {
            let res = try await client.parseMedicineCode(code: datamatrixCode)
            if let aicCode = res.aic, !aicCode.isEmpty {
                self.aic = aicCode
                let lookup = try await client.lookupAIFA(aic: aicCode)
                if let record = lookup.first {
                    self.name = record.name
                    self.activeIngredient = record.activeIngredient
                    self.dosage = record.dosage
                    self.unit = record.unit
                    self.requiresColdChain = record.requiresColdChain
                    self.location = record.defaultLocation
                    self.paoDays = "\(record.defaultPaoDays)"
                    self.prescriptionRequired = record.prescriptionRequired
                }
            }
            if let exp = res.expirationDate, !exp.isEmpty {
                let formatter = DateFormatter()
                formatter.dateFormat = "yyyy-MM-dd"
                if let d = formatter.date(from: exp) {
                    self.expirationDate = d
                }
            }
        } catch {
            errorMessage = error.localizedDescription
        }
        isParsingCode = false
    }

    private func saveMedicine() async {
        isSaving = true
        errorMessage = nil

        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let expStr = formatter.string(from: expirationDate)

        let q = Double(quantity.replacingOccurrences(of: ",", with: ".")) ?? 1.0
        let minT = Double(minThreshold.replacingOccurrences(of: ",", with: ".")) ?? 1.0
        let pao = Int(paoDays) ?? 0

        let med = Medicine(
            name: name.trimmingCharacters(in: .whitespaces),
            activeIngredient: activeIngredient.isEmpty ? nil : activeIngredient,
            dosage: dosage.isEmpty ? nil : dosage,
            aic: aic.isEmpty ? nil : aic,
            expirationDate: expStr,
            paoDays: pao,
            location: location,
            requiresColdChain: requiresColdChain,
            quantity: q,
            unit: unit,
            minThreshold: minT,
            prescriptionRequired: prescriptionRequired,
            notes: notes.isEmpty ? nil : notes
        )

        do {
            let saved = try await client.createMedicine(med)
            Haptics.success()
            onSaved(saved)
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
        isSaving = false
    }
}
