import SwiftUI

/// Vista nativa iOS per Bilancio Casa & Gestione Spese (Home Finance)
public struct FinanceSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Environment(\.accessibilityReduceTransparency) private var reduceTransparency

    @State private var selectedPeriod: String = "month"
    @State private var summary: FinancialSummaryDTO? = nil
    @State private var expenses: [ExpenseDTO] = []
    @State private var isLoading: Bool = false
    @State private var showAddExpenseSheet: Bool = false
    @State private var showSetBudgetSheet: Bool = false

    private let periods: [(id: String, label: String)] = [
        ("day", "Giorno"),
        ("week", "Settimana"),
        ("month", "Mese"),
        ("year", "Anno")
    ]

    public init() {}

    public var body: some View {
        NavigationStack {
            ZStack {
                Color.domusSystemGroupedBackground.ignoresSafeArea()

                ScrollView {
                    VStack(spacing: 16) {
                        // Selettore Periodo
                        periodSelectorView
                            .padding(.horizontal)
                            .padding(.top, 8)

                        if isLoading && summary == nil {
                            VStack(spacing: 12) {
                                ProgressView()
                                Text("Caricamento bilancio e spese...")
                                    .font(.subheadline)
                                    .foregroundColor(.secondary)
                            }
                            .frame(maxWidth: .infinity, minHeight: 200)
                        } else {
                            // KPI Cards
                            kpiCardsView
                                .padding(.horizontal)

                            // Fisse vs Variabili
                            if let fvv = summary?.fixedVsVariable {
                                fixedVsVariableCard(fvv)
                                    .padding(.horizontal)
                            }

                            // Smart Savings Insights
                            if let insights = summary?.insights, !insights.isEmpty {
                                insightsSection(insights)
                                    .padding(.horizontal)
                            }

                            // Ripartizione per Categorie
                            if let categories = summary?.categories, !categories.isEmpty {
                                categoryBreakdownSection(categories)
                                    .padding(.horizontal)
                            }

                            // Transazioni Recenti
                            recentExpensesSection
                                .padding(.horizontal)
                        }
                    }
                    .padding(.bottom, 32)
                }
                .refreshable {
                    await reloadData()
                }
            }
            .navigationTitle("Bilancio Casa & Spese")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Chiudi") {
                        dismiss()
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }

                ToolbarItem(placement: .primaryAction) {
                    HStack(spacing: 12) {
                        Button(action: {
                            Haptics.light()
                            showSetBudgetSheet = true
                        }) {
                            Image(systemName: "slider.horizontal.3")
                                .font(.body.weight(.semibold))
                        }
                        .frame(minWidth: 44, minHeight: 44)
                        .accessibilityLabel("Imposta budget mensile")

                        Button(action: {
                            Haptics.medium()
                            showAddExpenseSheet = true
                        }) {
                            Image(systemName: "plus.circle.fill")
                                .font(.title3)
                                .foregroundColor(.cyan)
                        }
                        .frame(minWidth: 44, minHeight: 44)
                        .accessibilityLabel("Aggiungi nuova spesa")
                    }
                }
            }
            .sheet(isPresented: $showAddExpenseSheet) {
                AddExpenseSheet {
                    Task { await reloadData() }
                }
            }
            .sheet(isPresented: $showSetBudgetSheet) {
                SetBudgetSheet(currentBudget: summary?.monthlyBudget ?? 1500) {
                    Task { await reloadData() }
                }
            }
            .task {
                await reloadData()
            }
        }
    }

    // MARK: - Subviews

    private var periodSelectorView: some View {
        HStack(spacing: 8) {
            ForEach(periods, id: \.id) { item in
                let isSelected = selectedPeriod == item.id
                Button(action: {
                    Haptics.selection()
                    selectedPeriod = item.id
                    Task { await reloadData() }
                }) {
                    Text(item.label)
                        .font(.subheadline.weight(isSelected ? .bold : .medium))
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .background(isSelected ? Color.cyan : Color.secondary.opacity(0.12))
                        .foregroundColor(isSelected ? .black : .primary)
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                }
                .accessibilityLabel("Filtra periodo: \(item.label)")
                .accessibilityAddTraits(isSelected ? .isSelected : [])
            }
        }
    }

    private var kpiCardsView: some View {
        VStack(spacing: 10) {
            HStack(spacing: 10) {
                // Totale Speso
                kpiCard(
                    title: "Totale Speso",
                    value: String(format: "€%.2f", summary?.totalSpent ?? 0.0),
                    subtitle: formatPercentageChange(summary?.percentageChange),
                    subtitleColor: (summary?.percentageChange ?? 0) <= 0 ? .green : .red,
                    icon: "eurosign.circle.fill",
                    accentColor: .blue
                )

                // Budget Residuo
                let remaining = summary?.remainingBudget ?? 0.0
                let hasBudget = (summary?.monthlyBudget ?? 0.0) > 0
                kpiCard(
                    title: hasBudget ? (remaining >= 0 ? "Budget Residuo" : "Sforamento") : "Media Giorno",
                    value: hasBudget ? String(format: "€%.2f", abs(remaining)) : String(format: "€%.2f", summary?.dailyAverage ?? 0.0),
                    subtitle: hasBudget ? (remaining >= 0 ? "Disponibile" : "Oltre il limite") : "Spesa media",
                    subtitleColor: remaining >= 0 ? .green : .red,
                    icon: remaining >= 0 ? "checkmark.seal.fill" : "exclamationmark.triangle.fill",
                    accentColor: remaining >= 0 ? .green : .red
                )
            }

            if let budget = summary?.monthlyBudget, budget > 0 {
                // Barra di avanzamento Budget
                VStack(alignment: .leading, spacing: 6) {
                    HStack {
                        Text("Budget Mensile Impostato")
                            .font(.caption)
                            .foregroundColor(.secondary)
                        Spacer()
                        Text(String(format: "€%.2f / €%.2f", summary?.totalSpent ?? 0.0, budget))
                            .font(.caption.bold())
                    }

                    let ratio = min(1.0, max(0.0, (summary?.totalSpent ?? 0.0) / budget))
                    GeometryReader { geo in
                        ZStack(alignment: .leading) {
                            RoundedRectangle(cornerRadius: 4)
                                .fill(Color.secondary.opacity(0.2))
                                .frame(height: 8)

                            RoundedRectangle(cornerRadius: 4)
                                .fill(ratio >= 1.0 ? Color.red : (ratio > 0.85 ? Color.orange : Color.green))
                                .frame(width: geo.size.width * CGFloat(ratio), height: 8)
                        }
                    }
                    .frame(height: 8)
                }
                .padding(12)
                .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.08))
                .clipShape(RoundedRectangle(cornerRadius: 12))
            }
        }
    }

    private func kpiCard(title: String, value: String, subtitle: String, subtitleColor: Color, icon: String, accentColor: Color) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(title)
                    .font(.caption.weight(.semibold))
                    .foregroundColor(.secondary)
                Spacer()
                Image(systemName: icon)
                    .font(.body)
                    .foregroundColor(accentColor)
            }

            Text(value)
                .font(.title2.bold())
                .foregroundColor(.primary)

            Text(subtitle)
                .font(.caption2.weight(.medium))
                .foregroundColor(subtitleColor)
        }
        .padding(14)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.08))
        .clipShape(RoundedRectangle(cornerRadius: 14))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title): \(value), \(subtitle)")
    }

    private func fixedVsVariableCard(_ fvv: FixedVsVariableDTO) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Spese Fisse vs Variabili")
                .font(.headline)

            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text("Fisse (Bollette, Assicurazioni)")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(String(format: "€%.2f (%.1f%%)", fvv.fixedAmount, fvv.fixedPercentage))
                        .font(.subheadline.bold())
                        .foregroundColor(.blue)
                }
                Spacer()
                VStack(alignment: .trailing, spacing: 2) {
                    Text("Variabili (Cibo, Extra)")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(String(format: "€%.2f (%.1f%%)", fvv.variableAmount, fvv.variablePercentage))
                        .font(.subheadline.bold())
                        .foregroundColor(.orange)
                }
            }

            // Barra proporzionale
            GeometryReader { geo in
                HStack(spacing: 2) {
                    let fixedRatio = max(0.0, min(1.0, fvv.fixedPercentage / 100.0))
                    RoundedRectangle(cornerRadius: 4)
                        .fill(Color.blue)
                        .frame(width: max(4, (geo.size.width - 2) * CGFloat(fixedRatio)), height: 8)
                    RoundedRectangle(cornerRadius: 4)
                        .fill(Color.orange)
                        .frame(height: 8)
                }
            }
            .frame(height: 8)
        }
        .padding(14)
        .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.08))
        .clipShape(RoundedRectangle(cornerRadius: 14))
    }

    private func insightsSection(_ insights: [SavingsInsightDTO]) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Label("Suggerimenti Intelligenti di Risparmio", systemImage: "sparkles")
                .font(.headline)
                .foregroundColor(.cyan)

            ForEach(insights) { insight in
                HStack(alignment: .top, spacing: 12) {
                    Image(systemName: insight.severity == "warning" ? "exclamationmark.triangle.fill" : "lightbulb.fill")
                        .foregroundColor(insight.severity == "warning" ? .orange : .cyan)
                        .font(.title3)
                        .padding(.top, 2)

                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(insight.title)
                                .font(.subheadline.bold())
                            Spacer()
                            if let saving = insight.potentialSaving, saving > 0 {
                                Text(String(format: "-€%.0f", saving))
                                    .font(.caption.bold())
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 3)
                                    .background(Color.green.opacity(0.18))
                                    .foregroundColor(.green)
                                    .clipShape(Capsule())
                            }
                        }

                        Text(insight.message)
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }
                .padding(12)
                .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.06))
                .clipShape(RoundedRectangle(cornerRadius: 12))
                .accessibilityElement(children: .combine)
            }
        }
    }

    private func categoryBreakdownSection(_ categories: [CategoryBreakdownDTO]) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Ripartizione per Categoria")
                .font(.headline)

            ForEach(categories) { cat in
                VStack(spacing: 6) {
                    HStack {
                        Image(systemName: iconForCategory(cat.category))
                            .foregroundColor(colorForCategory(cat.category))
                            .frame(width: 24)

                        Text(cat.name)
                            .font(.subheadline.weight(.semibold))

                        Spacer()

                        Text(String(format: "€%.2f", cat.totalAmount))
                            .font(.subheadline.bold())

                        Text(String(format: "%.1f%%", cat.percentage))
                            .font(.caption)
                            .foregroundColor(.secondary)
                            .frame(width: 48, alignment: .trailing)
                    }

                    GeometryReader { geo in
                        let ratio = min(1.0, max(0.0, cat.percentage / 100.0))
                        ZStack(alignment: .leading) {
                            RoundedRectangle(cornerRadius: 3)
                                .fill(Color.secondary.opacity(0.15))
                                .frame(height: 6)

                            RoundedRectangle(cornerRadius: 3)
                                .fill(colorForCategory(cat.category))
                                .frame(width: geo.size.width * CGFloat(ratio), height: 6)
                        }
                    }
                    .frame(height: 6)
                }
                .padding(.vertical, 4)
            }
        }
        .padding(14)
        .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.08))
        .clipShape(RoundedRectangle(cornerRadius: 14))
    }

    private var recentExpensesSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text("Ultime Transazioni")
                    .font(.headline)
                Spacer()
                Text("\(expenses.count) registrate")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            if expenses.isEmpty {
                Text("Nessuna spesa registrata per questo periodo.")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .padding(.vertical, 8)
            } else {
                ForEach(expenses) { exp in
                    HStack(spacing: 12) {
                        Image(systemName: iconForCategory(exp.category))
                            .font(.title3)
                            .foregroundColor(colorForCategory(exp.category))
                            .frame(width: 36, height: 36)
                            .background(colorForCategory(exp.category).opacity(0.14))
                            .clipShape(Circle())

                        VStack(alignment: .leading, spacing: 3) {
                            Text(exp.title)
                                .font(.subheadline.weight(.semibold))
                            HStack(spacing: 6) {
                                Text(exp.category.capitalized)
                                    .font(.caption2)
                                    .foregroundColor(.secondary)
                                if exp.isFixed {
                                    Text("Fissa")
                                        .font(.caption2.bold())
                                        .padding(.horizontal, 5)
                                        .padding(.vertical, 1)
                                        .background(Color.blue.opacity(0.15))
                                        .foregroundColor(.blue)
                                        .clipShape(RoundedRectangle(cornerRadius: 4))
                                }
                                if let kwh = exp.energyKwh, kwh > 0 {
                                    Text(String(format: "%.1f kWh", kwh))
                                        .font(.caption2)
                                        .foregroundColor(.yellow)
                                }
                            }
                        }

                        Spacer()

                        Text(String(format: "€%.2f", exp.amount))
                            .font(.subheadline.bold())
                            .foregroundColor(.primary)
                    }
                    .padding(.vertical, 6)
                    Divider()
                }
            }
        }
        .padding(14)
        .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.08))
        .clipShape(RoundedRectangle(cornerRadius: 14))
    }

    // MARK: - Helpers

    private func reloadData() async {
        isLoading = true
        defer { isLoading = false }
        let client = DomusHttpClient.shared
        if let sum = try? await client.fetchFinancialSummary(period: selectedPeriod) {
            self.summary = sum
        }
        if let exps = try? await client.fetchExpenses() {
            self.expenses = exps
        }
    }

    private func formatPercentageChange(_ change: Double?) -> String {
        guard let change = change else { return "N/D vs prec." }
        if change > 0 {
            return String(format: "+%.1f%% vs prec.", change)
        } else if change < 0 {
            return String(format: "%.1f%% vs prec.", change)
        } else {
            return "Stabile vs prec."
        }
    }

    private func iconForCategory(_ category: String) -> String {
        switch category.lowercased() {
        case "food": return "cart.fill"
        case "utilities": return "bolt.fill"
        case "transport": return "fuelpump.fill"
        case "maintenance": return "hammer.fill"
        case "insurance": return "shield.fill"
        default: return "tag.fill"
        }
    }

    private func colorForCategory(_ category: String) -> Color {
        switch category.lowercased() {
        case "food": return .green
        case "utilities": return .yellow
        case "transport": return .cyan
        case "maintenance": return .orange
        case "insurance": return .purple
        default: return .secondary
        }
    }
}

// MARK: - Modale Aggiungi Spesa

public struct AddExpenseSheet: View {
    @Environment(\.dismiss) private var dismiss
    public let onSaved: () -> Void

    @State private var title: String = ""
    @State private var amountText: String = ""
    @State private var selectedCategory: String = "food"
    @State private var subcategory: String = ""
    @State private var isFixed: Bool = false
    @State private var kwhText: String = ""
    @State private var isSaving: Bool = false

    private let categories = [
        ("food", "Cibo & Alimentari"),
        ("utilities", "Bollette & Utenze"),
        ("transport", "Carburante & Mobilità"),
        ("maintenance", "Manutenzione Casa"),
        ("insurance", "Assicurazioni"),
        ("other", "Altro / Extra")
    ]

    public init(onSaved: @escaping () -> Void) {
        self.onSaved = onSaved
    }

    public var body: some View {
        NavigationStack {
            Form {
                Section(header: Text("Dati Spesa")) {
                    TextField("Titolo (es. Spesa Supermercato)", text: $title)
                        .accessibilityLabel("Titolo della spesa")
                    TextField("Importo in € (es. 45.50)", text: $amountText)
                        #if os(iOS)
                        .keyboardType(.decimalPad)
                        #endif
                        .accessibilityLabel("Importo in euro")

                    Picker("Categoria", selection: $selectedCategory) {
                        ForEach(categories, id: \.0) { cat in
                            Text(cat.1).tag(cat.0)
                        }
                    }

                    TextField("Sottocategoria (opzionale)", text: $subcategory)
                }

                Section(header: Text("Dettagli Aggiuntivi")) {
                    Toggle("Spesa Fissa / Ricorrente", isOn: $isFixed)

                    if selectedCategory == "utilities" {
                        TextField("Consumo kWh (opzionale)", text: $kwhText)
                            #if os(iOS)
                            .keyboardType(.decimalPad)
                            #endif
                    }
                }
            }
            .navigationTitle("Nuova Spesa")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annulla") {
                        dismiss()
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }

                ToolbarItem(placement: .confirmationAction) {
                    Button("Salva") {
                        saveExpense()
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .disabled(title.trimmingCharacters(in: .whitespaces).isEmpty || Double(amountText.replacingOccurrences(of: ",", with: ".")) == nil || isSaving)
                }
            }
        }
    }

    private func saveExpense() {
        guard let amount = Double(amountText.replacingOccurrences(of: ",", with: ".")) else { return }
        isSaving = true
        let kwh = Double(kwhText.replacingOccurrences(of: ",", with: "."))

        Task {
            let req = CreateExpenseRequestDTO(
                title: title.trimmingCharacters(in: .whitespaces),
                amount: amount,
                category: selectedCategory,
                subcategory: subcategory,
                isFixed: isFixed,
                energyKwh: kwh
            )
            _ = try? await DomusHttpClient.shared.createExpense(req: req)
            Haptics.success()
            isSaving = false
            onSaved()
            dismiss()
        }
    }
}

// MARK: - Modale Imposta Budget

public struct SetBudgetSheet: View {
    @Environment(\.dismiss) private var dismiss
    public let currentBudget: Double
    public let onSaved: () -> Void

    @State private var budgetText: String = ""
    @State private var isSaving: Bool = false

    public init(currentBudget: Double, onSaved: @escaping () -> Void) {
        self.currentBudget = currentBudget
        self.onSaved = onSaved
        _budgetText = State(initialValue: String(format: "%.0f", currentBudget))
    }

    public var body: some View {
        NavigationStack {
            Form {
                Section(header: Text("Budget Mensile Globale"), footer: Text("Imposta un limite massimo di spesa per la casa. Riceverai avvisi visivi quando ti avvicini o superi la soglia.")) {
                    TextField("Budget in € (es. 1500)", text: $budgetText)
                        #if os(iOS)
                        .keyboardType(.numberPad)
                        #endif
                        .accessibilityLabel("Budget mensile in euro")
                }
            }
            .navigationTitle("Imposta Budget")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annulla") {
                        dismiss()
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }

                ToolbarItem(placement: .confirmationAction) {
                    Button("Salva") {
                        saveBudget()
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .disabled(Double(budgetText) == nil || isSaving)
                }
            }
        }
    }

    private func saveBudget() {
        guard let limit = Double(budgetText) else { return }
        isSaving = true
        Task {
            let req = SetBudgetRequestDTO(monthlyLimit: limit)
            _ = try? await DomusHttpClient.shared.setBudget(req: req)
            Haptics.success()
            isSaving = false
            onSaved()
            dismiss()
        }
    }
}
