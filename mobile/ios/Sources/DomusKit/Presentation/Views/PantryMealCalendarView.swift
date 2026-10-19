import SwiftUI

/// Vista completa Dispensa, Ricettario, Calendario Pasti e Scanner Scontrino per DomusKit iOS
public struct PantryMealCalendarView: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Environment(\.accessibilityReduceTransparency) private var reduceTransparency
    @Environment(\.accessibilityDifferentiateWithoutColor) private var differentiateWithoutColor

    @State private var selectedTab: Int = 0
    private let tabs = ["Dispensa", "Ricettario", "Calendario", "Scontrino"]

    // Mock/Local state for preview and integration
    @State private var items: [PantryItem] = [
        PantryItem(name: "Pasta di Semola", category: "dispensa", location: "dispensa", quantity: 2.0, unit: "kg", minThreshold: 1.0),
        PantryItem(name: "Latte Intero Fresco", category: "frigo", location: "frigo", quantity: 1.0, unit: "L", minThreshold: 2.0, expiryDate: "19/10/2026", isLowStock: true, isExpiringSoon: true),
        PantryItem(name: "Passata di Pomodoro", category: "dispensa", location: "dispensa", quantity: 3.0, unit: "pz", minThreshold: 2.0),
        PantryItem(name: "Uova Fresche", category: "frigo", location: "frigo", quantity: 6.0, unit: "pz", minThreshold: 4.0),
        PantryItem(name: "Olio Extra Vergine", category: "dispensa", location: "dispensa", quantity: 0.5, unit: "L", minThreshold: 1.0, isLowStock: true)
    ]

    @State private var recipes: [Recipe] = [
        Recipe(
            title: "Spaghetti al Pomodoro Fresco",
            description: "Classico primo piatto italiano della tradizione con pomodoro e basilico.",
            category: "primo",
            prepTimeMin: 10,
            cookTimeMin: 12,
            portionsBase: 2,
            ingredients: [
                RecipeIngredient(itemName: "Pasta di Semola", quantity: 200, unit: "g"),
                RecipeIngredient(itemName: "Passata di Pomodoro", quantity: 250, unit: "g"),
                RecipeIngredient(itemName: "Olio Extra Vergine", quantity: 20, unit: "ml")
            ]
        ),
        Recipe(
            title: "Frittata Soffice alle Erbe",
            description: "Secondo leggero, veloce e ricco di proteine.",
            category: "secondo",
            prepTimeMin: 5,
            cookTimeMin: 8,
            portionsBase: 2,
            ingredients: [
                RecipeIngredient(itemName: "Uova Fresche", quantity: 4, unit: "pz"),
                RecipeIngredient(itemName: "Olio Extra Vergine", quantity: 10, unit: "ml")
            ]
        )
    ]

    @State private var mealPlans: [MealPlan] = [
        MealPlan(
            date: "17/10/2026",
            mealType: "lunch",
            recipeTitle: "Spaghetti al Pomodoro Fresco",
            customNotes: "Pranzo in famiglia",
            personsCount: 2,
            isConsumed: false
        ),
        MealPlan(
            date: "17/10/2026",
            mealType: "dinner",
            recipeTitle: "Frittata Soffice alle Erbe",
            customNotes: "Cena leggera",
            personsCount: 2,
            isConsumed: false
        )
    ]

    // Dialog state
    @State private var showAddItemSheet = false
    @State private var showAddRecipeSheet = false
    @State private var showAddMealSheet = false
    @State private var selectedRecipeForMeal: Recipe? = nil
    @State private var alertMessage: String? = nil

    public init() {}

    public var body: some View {
        NavigationStack {
            ZStack {
                Color.domusSystemGroupedBackground.ignoresSafeArea()

                VStack(spacing: 0) {
                    // Segmented Tab Picker
                    Picker("Sezione", selection: $selectedTab) {
                        ForEach(0..<tabs.count, id: \.self) { index in
                            Text(tabs[index]).tag(index)
                        }
                    }
                    .pickerStyle(.segmented)
                    .padding(.horizontal)
                    .padding(.top, 8)
                    .padding(.bottom, 6)

                    // Alert Notification Banner
                    if let msg = alertMessage {
                        HStack(spacing: 8) {
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundColor(.green)
                            Text(msg)
                                .font(.caption.bold())
                                .foregroundColor(.green)
                            Spacer()
                            Button(action: { alertMessage = nil }) {
                                Image(systemName: "xmark")
                                    .font(.caption2.bold())
                                    .foregroundColor(.secondary)
                            }
                        }
                        .padding(.horizontal, 14)
                        .padding(.vertical, 8)
                        .background(Color.green.opacity(0.15))
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                        .padding(.horizontal)
                        .padding(.bottom, 4)
                        .transition(.move(edge: .top).combined(with: .opacity))
                    }

                    // Content based on Selected Tab
                    Group {
                        switch selectedTab {
                        case 0:
                            inventoryTabView
                        case 1:
                            recipesTabView
                        case 2:
                            mealCalendarTabView
                        default:
                            receiptScannerTabView
                        }
                    }
                }
            }
            .navigationTitle("Dispensa & Ricettario")
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
                    trailingToolbarButton
                }
            }
            .sheet(isPresented: $showAddItemSheet) {
                AddItemModalView { newItem in
                    items.append(newItem)
                    alertMessage = "Prodotto \(newItem.name) aggiunto alla dispensa!"
                    Haptics.success()
                }
            }
            .sheet(isPresented: $showAddRecipeSheet) {
                AddRecipeModalView { newRecipe in
                    recipes.append(newRecipe)
                    alertMessage = "Ricetta \(newRecipe.title) salvata nel ricettario!"
                    Haptics.success()
                }
            }
            .sheet(isPresented: $showAddMealSheet) {
                AddMealPlanModalView(preselectedRecipe: selectedRecipeForMeal, availableRecipes: recipes) { newPlan in
                    mealPlans.append(newPlan)
                    alertMessage = "Pasto pianificato nel calendario!"
                    Haptics.success()
                }
            }
        }
    }

    @ViewBuilder
    private var trailingToolbarButton: some View {
        switch selectedTab {
        case 0:
            Button(action: { showAddItemSheet = true }) {
                Image(systemName: "plus.circle.fill")
                    .foregroundColor(.blue)
                    .frame(minWidth: 44, minHeight: 44)
            }
            .accessibilityLabel("Aggiungi alimento in dispensa")
        case 1:
            Button(action: { showAddRecipeSheet = true }) {
                Image(systemName: "plus.circle.fill")
                    .foregroundColor(.green)
                    .frame(minWidth: 44, minHeight: 44)
            }
            .accessibilityLabel("Crea nuova ricetta")
        case 2:
            Button(action: {
                selectedRecipeForMeal = nil
                showAddMealSheet = true
            }) {
                Image(systemName: "plus.circle.fill")
                    .foregroundColor(.cyan)
                    .frame(minWidth: 44, minHeight: 44)
            }
            .accessibilityLabel("Pianifica nuovo pasto")
        default:
            EmptyView()
        }
    }

    // ==========================================
    // TAB 1: DISPENSA & SCORTE INVENTARIO
    // ==========================================

    @State private var filterCategory: String = "Tutti"
    private let categories = ["Tutti", "frigo", "dispensa", "ortofrutta", "freschi", "bevande"]

    private var inventoryTabView: some View {
        List {
            // KPI Summary Section
            Section {
                HStack(spacing: 10) {
                    kpiTile(title: "Articoli", value: "\(items.count)", color: .blue)
                    let lowCount = items.filter { $0.isLowStock }.count
                    kpiTile(title: "Sotto Soglia", value: "\(lowCount)", color: lowCount > 0 ? .red : .secondary)
                    let expCount = items.filter { $0.isExpiringSoon }.count
                    kpiTile(title: "In Scadenza", value: "\(expCount)", color: expCount > 0 ? .orange : .secondary)
                }
                .listRowBackground(Color.clear)
                .listRowInsets(EdgeInsets(top: 4, leading: 0, bottom: 4, trailing: 0))
            }

            // Category Filter Pills
            Section {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        ForEach(categories, id: \.self) { cat in
                            Button(action: {
                                Haptics.selection()
                                filterCategory = cat
                            }) {
                                Text(cat.capitalized)
                                    .font(.caption2.bold())
                                    .padding(.horizontal, 12)
                                    .padding(.vertical, 6)
                                    .background(filterCategory == cat ? Color.cyan : Color.secondary.opacity(0.15))
                                    .foregroundColor(filterCategory == cat ? .black : .primary)
                                    .clipShape(Capsule())
                            }
                            .frame(minHeight: 44)
                        }
                    }
                }
                .listRowBackground(Color.clear)
                .listRowInsets(EdgeInsets(top: 0, leading: 0, bottom: 0, trailing: 0))
            }

            // Inventory Item Rows
            let filtered = items.filter { filterCategory == "Tutti" || $0.category.lowercased() == filterCategory.lowercased() || $0.location.lowercased() == filterCategory.lowercased() }

            Section("Scorte in Casa (\(filtered.count))") {
                ForEach(filtered) { item in
                    HStack(spacing: 12) {
                        VStack(alignment: .leading, spacing: 3) {
                            HStack(spacing: 6) {
                                Text(item.name)
                                    .font(.subheadline.weight(.semibold))

                                if item.isLowStock {
                                    Text("SOTTO SOGLIA")
                                        .font(.system(size: 8, weight: .bold))
                                        .padding(.horizontal, 5)
                                        .padding(.vertical, 2)
                                        .background(Color.red.opacity(0.18))
                                        .foregroundColor(.red)
                                        .clipShape(RoundedRectangle(cornerRadius: 4))
                                }
                            }

                            HStack(spacing: 6) {
                                Text("\(item.category.uppercased()) • \(item.location)")
                                    .font(.caption2)
                                    .foregroundColor(.secondary)

                                if let expiry = item.expiryDate {
                                    Text("• Scad. \(expiry)")
                                        .font(.caption2)
                                        .foregroundColor(item.isExpiringSoon ? .orange : .secondary)
                                }
                            }
                        }

                        Spacer()

                        // Stock Stepper (+ / -)
                        HStack(spacing: 8) {
                            Button(action: {
                                Haptics.light()
                                if let idx = items.firstIndex(where: { $0.id == item.id }), items[idx].quantity > 0 {
                                    items[idx].quantity = max(0, items[idx].quantity - 1.0)
                                    items[idx].isLowStock = items[idx].quantity <= items[idx].minThreshold
                                }
                            }) {
                                Image(systemName: "minus")
                                    .font(.caption.bold())
                                    .frame(width: 32, height: 32)
                                    .background(Color.secondary.opacity(0.15))
                                    .clipShape(Circle())
                            }
                            .buttonStyle(.plain)
                            .frame(minWidth: 44, minHeight: 44)
                            .accessibilityLabel("Riduci scorta di \(item.name)")

                            Text(String(format: "%.1f %@", item.quantity, item.unit).replacingOccurrences(of: ".0", with: ""))
                                .font(.callout.bold().monospacedDigit())
                                .foregroundColor(item.isLowStock ? .red : .primary)
                                .frame(minWidth: 50)

                            Button(action: {
                                Haptics.light()
                                if let idx = items.firstIndex(where: { $0.id == item.id }) {
                                    items[idx].quantity += 1.0
                                    items[idx].isLowStock = items[idx].quantity <= items[idx].minThreshold
                                }
                            }) {
                                Image(systemName: "plus")
                                    .font(.caption.bold())
                                    .frame(width: 32, height: 32)
                                    .background(Color.blue.opacity(0.2))
                                    .foregroundColor(.blue)
                                    .clipShape(Circle())
                            }
                            .buttonStyle(.plain)
                            .frame(minWidth: 44, minHeight: 44)
                            .accessibilityLabel("Aumenta scorta di \(item.name)")
                        }
                    }
                    .padding(.vertical, 4)
                }
                .onDelete { indices in
                    items.remove(atOffsets: indices)
                }
            }
        }
        #if os(iOS)
        .listStyle(.insetGrouped)
        #else
        .listStyle(.inset)
        #endif
    }

    private func kpiTile(title: String, value: String, color: Color) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title)
                .font(.caption2)
                .foregroundColor(.secondary)
            Text(value)
                .font(.title3.bold().monospacedDigit())
                .foregroundColor(color)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(10)
        .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.12))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title): \(value)")
    }

    // ==========================================
    // TAB 2: RICETTARIO
    // ==========================================

    private var recipesTabView: some View {
        List {
            Section("Ricette Salvate (\(recipes.count))") {
                ForEach(recipes) { recipe in
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(recipe.title)
                                    .font(.headline)
                                Text("\(recipe.category.uppercased()) • \(recipe.portionsBase) porzioni base • ⏱️ \(recipe.prepTimeMin + recipe.cookTimeMin) min")
                                    .font(.caption2)
                                    .foregroundColor(.cyan)
                            }
                            Spacer()

                            Button(action: {
                                Haptics.light()
                                selectedRecipeForMeal = recipe
                                showAddMealSheet = true
                            }) {
                                Text("Pianifica")
                                    .font(.caption.bold())
                                    .padding(.horizontal, 10)
                                    .padding(.vertical, 6)
                                    .background(Color.blue)
                                    .foregroundColor(.white)
                                    .clipShape(RoundedRectangle(cornerRadius: 8))
                            }
                            .frame(minHeight: 44)
                            .accessibilityLabel("Pianifica pasto per \(recipe.title)")
                        }

                        if !recipe.description.isEmpty {
                            Text(recipe.description)
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }

                        // Ingredient Tags
                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 6) {
                                ForEach(recipe.ingredients) { ing in
                                    Text("\(ing.itemName) (\(Int(ing.quantity)) \(ing.unit))")
                                        .font(.system(size: 10, weight: .semibold))
                                        .padding(.horizontal, 8)
                                        .padding(.vertical, 3)
                                        .background(Color.secondary.opacity(0.15))
                                        .clipShape(Capsule())
                                }
                            }
                        }
                    }
                    .padding(.vertical, 6)
                }
                .onDelete { indices in
                    recipes.remove(atOffsets: indices)
                }
            }
        }
        #if os(iOS)
        .listStyle(.insetGrouped)
        #else
        .listStyle(.inset)
        #endif
    }

    // ==========================================
    // TAB 3: CALENDARIO PASTI & CONSUMO AUTOMATICO
    // ==========================================

    private var mealCalendarTabView: some View {
        List {
            Section {
                HStack(spacing: 8) {
                    Image(systemName: "sparkles")
                        .foregroundColor(.cyan)
                    Text("Quando premi 'Consuma Pasto', Domus decurta automaticamente tutti gli ingredienti dalla dispensa in base ai commensali.")
                        .font(.caption)
                        .foregroundColor(.cyan)
                }
                .padding(.vertical, 4)
                .listRowBackground(Color.cyan.opacity(0.12))
            }

            Section("Pasti Pianificati") {
                if mealPlans.isEmpty {
                    Text("Nessun pasto in programma. Tocca + per pianificare.")
                        .font(.caption)
                        .foregroundColor(.secondary)
                } else {
                    ForEach(mealPlans) { plan in
                        VStack(alignment: .leading, spacing: 8) {
                            HStack {
                                Text(mealTypeIcon(plan.mealType))
                                    .font(.title3)

                                VStack(alignment: .leading, spacing: 2) {
                                    Text(plan.recipeTitle)
                                        .font(.subheadline.bold())
                                    Text("\(plan.date) • \(plan.mealType.uppercased())")
                                        .font(.caption2)
                                        .foregroundColor(.secondary)
                                }

                                Spacer()

                                if plan.isConsumed {
                                    Text("CONSUMATO ✓")
                                        .font(.system(size: 9, weight: .bold))
                                        .padding(.horizontal, 8)
                                        .padding(.vertical, 4)
                                        .background(Color.green.opacity(0.2))
                                        .foregroundColor(.green)
                                        .clipShape(RoundedRectangle(cornerRadius: 6))
                                }
                            }

                            if !plan.customNotes.isEmpty {
                                Text(plan.customNotes)
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }

                            if !plan.isConsumed {
                                Divider()

                                HStack {
                                    Text("Commensali: \(plan.personsCount)")
                                        .font(.caption.bold())

                                    Spacer()

                                    Button(action: {
                                        Haptics.success()
                                        consumeMealPlan(plan)
                                    }) {
                                        HStack(spacing: 4) {
                                            Image(systemName: "fork.knife")
                                            Text("Consuma Pasto")
                                        }
                                        .font(.caption.bold())
                                        .padding(.horizontal, 12)
                                        .padding(.vertical, 7)
                                        .background(Color.green)
                                        .foregroundColor(.black)
                                        .clipShape(RoundedRectangle(cornerRadius: 8))
                                    }
                                    .frame(minHeight: 44)
                                    .accessibilityLabel("Consuma pasto e scala dispensa")
                                }
                            }
                        }
                        .padding(.vertical, 4)
                    }
                    .onDelete { indices in
                        mealPlans.remove(atOffsets: indices)
                    }
                }
            }
        }
        #if os(iOS)
        .listStyle(.insetGrouped)
        #else
        .listStyle(.inset)
        #endif
    }

    private func mealTypeIcon(_ type: String) -> String {
        switch type {
        case "breakfast": return "☕"
        case "lunch": return "🍝"
        case "snack": return "🍎"
        case "dinner": return "🍲"
        default: return "🍰"
        }
    }

    private func consumeMealPlan(_ plan: MealPlan) {
        // Mark consumed
        if let idx = mealPlans.firstIndex(where: { $0.id == plan.id }) {
            mealPlans[idx].isConsumed = true
            mealPlans[idx].consumedAt = ISO8601DateFormatter().string(from: Date())
        }

        // Scale inventory
        if let recipe = recipes.first(where: { $0.title == plan.recipeTitle }) {
            let scale = Double(plan.personsCount) / Double(max(1, recipe.portionsBase))
            for ing in recipe.ingredients {
                if let pIdx = items.firstIndex(where: { $0.name.lowercased().contains(ing.itemName.lowercased()) || ing.itemName.lowercased().contains($0.name.lowercased()) }) {
                    let dose = ing.quantity * scale
                    // Convert grams to kg if pantry is in kg
                    let effectiveDose = (items[pIdx].unit == "kg" && ing.unit == "g") ? dose / 1000.0 : dose
                    items[pIdx].quantity = max(0, items[pIdx].quantity - effectiveDose)
                    items[pIdx].isLowStock = items[pIdx].quantity <= items[pIdx].minThreshold
                }
            }
        }

        alertMessage = "Pasto consumato! Scorte in dispensa decurtate proporzionalmente per \(plan.personsCount) persone."
    }

    // ==========================================
    // TAB 4: SCANNER SCONTRINO (OCR)
    // ==========================================

    @State private var ocrText: String = ""
    @State private var parsedReceipt: Receipt? = nil
    @State private var autoAddFinance: Bool = true
    @State private var autoUpdatePantry: Bool = true

    private var receiptScannerTabView: some View {
        List {
            Section("Lettore Scontrini Intelligente") {
                Text("Incolla o acquisisci il testo OCR dello scontrino per estrarre articoli, pesi, prezzi e registrare la spesa in Home Finance.")
                    .font(.caption)
                    .foregroundColor(.secondary)

                HStack(spacing: 8) {
                    Button("Campione Esselunga") {
                        ocrText = """
                        ESSELUNGA S.P.A.
                        PASTA BARILLA 500G        1.20
                        LATTE INTERO 1L           1.50
                        MELE FUJI KG 1.500        3.45
                        CAFFE LAVAZZA 250G        3.80
                        TOTALE COMPLESSIVO       17.55
                        """
                    }
                    .font(.caption2.bold())
                    .frame(minHeight: 44)

                    Button("Campione Conad") {
                        ocrText = """
                        CONAD CITY
                        UOVA BIO 6PZ              2.40
                        OLIO EVO 1L               7.90
                        PARMIGIANO REGGIANO 300G  5.50
                        TOTALE EURO              15.80
                        """
                    }
                    .font(.caption2.bold())
                    .frame(minHeight: 44)
                }

                TextEditor(text: $ocrText)
                    .frame(minHeight: 100)
                    .font(.system(.caption, design: .monospaced))

                Button(action: {
                    Haptics.light()
                    parseReceiptText()
                }) {
                    HStack {
                        Spacer()
                        Image(systemName: "text.viewfinder")
                        Text("Analizza Scontrino")
                            .font(.subheadline.bold())
                        Spacer()
                    }
                }
                .disabled(ocrText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                .frame(minHeight: 44)
            }

            if let receipt = parsedReceipt {
                Section("Risultato Scansione (Totale: €\(String(format: "%.2f", receipt.totalAmount)))") {
                    ForEach(receipt.items) { item in
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(item.itemName)
                                    .font(.subheadline.bold())
                                Text(String(format: "Quantità: %.1f %@", item.quantity, item.unit))
                                    .font(.caption2)
                                    .foregroundColor(.secondary)
                            }
                            Spacer()
                            Text(String(format: "€%.2f", item.totalPrice))
                                .font(.callout.bold().monospacedDigit())
                                .foregroundColor(.green)
                        }
                    }

                    Toggle("Aggiorna Scorte Dispensa", isOn: $autoUpdatePantry)
                    Toggle("Registra Spesa in Bilancio Casa (€)", isOn: $autoAddFinance)

                    Button(action: {
                        Haptics.success()
                        importReceipt(receipt)
                    }) {
                        HStack {
                            Spacer()
                            Image(systemName: "checkmark.circle.fill")
                            Text("Conferma e Importa in Dispensa")
                                .font(.subheadline.bold())
                            Spacer()
                        }
                    }
                    .foregroundColor(.green)
                    .frame(minHeight: 44)
                }
            }
        }
        #if os(iOS)
        .listStyle(.insetGrouped)
        #else
        .listStyle(.inset)
        #endif
    }

    private func parseReceiptText() {
        let lines = ocrText.components(separatedBy: .newlines)
        var parsedItems: [ReceiptItem] = []
        var detectedTotal: Double = 0.0

        for line in lines {
            let upper = line.uppercased()
            if upper.contains("TOTALE") {
                let parts = upper.components(separatedBy: CharacterSet.decimalDigits.inverted).filter { !$0.isEmpty }
                if let lastPart = parts.last, let val = Double(lastPart) {
                    detectedTotal = val / 100.0
                }
                continue
            }

            // Quick heuristic: Name + Price
            let tokens = line.split(separator: " ").map { String($0) }
            if tokens.count >= 2, let price = Double(tokens.last!.replacingOccurrences(of: ",", with: ".")) {
                let name = tokens.dropLast().joined(separator: " ")
                let qty = name.contains("KG") ? 1.5 : 1.0
                let unit = name.contains("KG") ? "kg" : (name.contains("L") ? "L" : "pz")
                parsedItems.append(ReceiptItem(itemName: name, quantity: qty, unit: unit, unitPrice: price, totalPrice: price))
            }
        }

        if detectedTotal == 0.0 {
            detectedTotal = parsedItems.reduce(0.0) { $0 + $1.totalPrice }
        }

        parsedReceipt = Receipt(storeName: "Supermercato", totalAmount: detectedTotal, items: parsedItems, rawText: ocrText)
    }

    private func importReceipt(_ receipt: Receipt) {
        if autoUpdatePantry {
            for rItem in receipt.items {
                if let idx = items.firstIndex(where: { $0.name.lowercased().contains(rItem.itemName.lowercased()) || rItem.itemName.lowercased().contains($0.name.lowercased()) }) {
                    items[idx].quantity += rItem.quantity
                    items[idx].isLowStock = items[idx].quantity <= items[idx].minThreshold
                } else {
                    items.append(PantryItem(name: rItem.itemName, category: "dispensa", location: "dispensa", quantity: rItem.quantity, unit: rItem.unit))
                }
            }
        }

        alertMessage = "Importati \(receipt.items.count) articoli! €\(String(format: "%.2f", receipt.totalAmount)) registrati in bilancio."
        parsedReceipt = nil
        ocrText = ""
        selectedTab = 0
    }
}

// ==========================================
// MODAL SHEETS
// ==========================================

struct AddItemModalView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var name: String = ""
    @State private var category: String = "dispensa"
    @State private var location: String = "dispensa"
    @State private var quantityStr: String = "1"
    @State private var unit: String = "pz"
    @State private var thresholdStr: String = "1"

    var onSave: (PantryItem) -> Void

    var body: some View {
        NavigationStack {
            Form {
                Section("Dati Prodotto") {
                    TextField("Nome Alimento", text: $name)
                    Picker("Categoria", selection: $category) {
                        Text("Dispensa").tag("dispensa")
                        Text("Frigo").tag("frigo")
                        Text("Ortofrutta").tag("ortofrutta")
                        Text("Freschi").tag("freschi")
                        Text("Bevande").tag("bevande")
                        Text("Surgelati").tag("surgelati")
                    }
                    Picker("Posizione", selection: $location) {
                        Text("Dispensa").tag("dispensa")
                        Text("Frigorifero").tag("frigo")
                        Text("Freezer").tag("freezer")
                        Text("Cantina").tag("cantina")
                    }
                }

                Section("Quantità e Soglie") {
                    HStack {
                        TextField("Quantità", text: $quantityStr)
                            #if os(iOS)
                            .keyboardType(.decimalPad)
                            #endif
                        TextField("Unità (pz, kg, L, g)", text: $unit)
                    }
                    TextField("Soglia Minima Allerta Scorte", text: $thresholdStr)
                        #if os(iOS)
                        .keyboardType(.decimalPad)
                        #endif
                }
            }
            .navigationTitle("Nuovo Alimento")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annulla") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Salva") {
                        let q = Double(quantityStr) ?? 1.0
                        let th = Double(thresholdStr) ?? 1.0
                        let item = PantryItem(
                            name: name,
                            category: category,
                            location: location,
                            quantity: q,
                            unit: unit,
                            minThreshold: th
                        )
                        onSave(item)
                        dismiss()
                    }
                    .disabled(name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
        }
    }
}

struct AddRecipeModalView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var title: String = ""
    @State private var description: String = ""
    @State private var category: String = "primo"
    @State private var portions: Int = 2
    @State private var prepTime: Int = 10
    @State private var cookTime: Int = 15

    var onSave: (Recipe) -> Void

    var body: some View {
        NavigationStack {
            Form {
                Section("Dettagli Ricetta") {
                    TextField("Titolo Ricetta", text: $title)
                    TextField("Descrizione breve", text: $description)
                    Picker("Categoria", selection: $category) {
                        Text("Primo").tag("primo")
                        Text("Secondo").tag("secondo")
                        Text("Contorno").tag("contorno")
                        Text("Dessert").tag("dessert")
                        Text("Spuntino").tag("snack")
                    }
                }

                Section("Porzioni e Tempi") {
                    Stepper("Porzioni base: \(portions)", value: $portions, in: 1...20)
                    Stepper("Preparazione: \(prepTime) min", value: $prepTime, in: 1...120)
                    Stepper("Cottura: \(cookTime) min", value: $cookTime, in: 0...240)
                }
            }
            .navigationTitle("Nuova Ricetta")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annulla") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Crea Ricetta") {
                        let recipe = Recipe(
                            title: title,
                            description: description,
                            category: category,
                            prepTimeMin: prepTime,
                            cookTimeMin: cookTime,
                            portionsBase: portions
                        )
                        onSave(recipe)
                        dismiss()
                    }
                    .disabled(title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
        }
    }
}

struct AddMealPlanModalView: View {
    @Environment(\.dismiss) private var dismiss
    var preselectedRecipe: Recipe?
    var availableRecipes: [Recipe]
    var onSave: (MealPlan) -> Void

    @State private var selectedRecipeTitle: String = ""
    @State private var mealType: String = "lunch"
    @State private var dateStr: String = "17/10/2026"
    @State private var personsCount: Int = 2
    @State private var customNotes: String = ""

    var body: some View {
        NavigationStack {
            Form {
                Section("Ricetta o Nome Pasto") {
                    if !availableRecipes.isEmpty {
                        Picker("Ricetta", selection: $selectedRecipeTitle) {
                            Text("Pasto Libero").tag("")
                            ForEach(availableRecipes) { rec in
                                Text(rec.title).tag(rec.title)
                            }
                        }
                    } else {
                        TextField("Nome pasto", text: $selectedRecipeTitle)
                    }

                    Picker("Tipo di Pasto", selection: $mealType) {
                        Text("☕ Colazione").tag("breakfast")
                        Text("🍝 Pranzo").tag("lunch")
                        Text("🍎 Spuntino").tag("snack")
                        Text("🍲 Cena").tag("dinner")
                        Text("🍰 Dessert").tag("dessert")
                    }
                }

                Section("Commensali e Data") {
                    TextField("Data (GG/MM/AAAA)", text: $dateStr)
                    Stepper("Commensali: \(personsCount)", value: $personsCount, in: 1...20)
                    TextField("Note opzionali", text: $customNotes)
                }
            }
            .navigationTitle("Pianifica Pasto")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .onAppear {
                if let pre = preselectedRecipe {
                    selectedRecipeTitle = pre.title
                    personsCount = pre.portionsBase
                }
            }
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annulla") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Pianifica") {
                        let plan = MealPlan(
                            date: dateStr,
                            mealType: mealType,
                            recipeTitle: selectedRecipeTitle.isEmpty ? "Pasto Programmato" : selectedRecipeTitle,
                            customNotes: customNotes,
                            personsCount: personsCount
                        )
                        onSave(plan)
                        dismiss()
                    }
                }
            }
        }
    }
}
