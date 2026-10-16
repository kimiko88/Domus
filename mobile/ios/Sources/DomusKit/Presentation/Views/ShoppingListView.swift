import SwiftUI

/// Modello articolo lista della spesa condivisa
public struct ShoppingItem: Identifiable, Codable, Sendable, Equatable {
    public let id: String
    public var title: String
    public var quantity: String
    public var category: String
    public var completed: Bool
    public var completedBy: String?
    public var createdAt: String

    public init(
        id: String = UUID().uuidString,
        title: String,
        quantity: String = "1",
        category: String = "Dispensa",
        completed: Bool = false,
        completedBy: String? = nil,
        createdAt: String = ISO8601DateFormatter().string(from: Date())
    ) {
        self.id = id
        self.title = title
        self.quantity = quantity
        self.category = category
        self.completed = completed
        self.completedBy = completedBy
        self.createdAt = createdAt
    }
}

/// Vista nativa Lista della Spesa per Domus iOS
public struct ShoppingListView: View {
    @Environment(\.dismiss) private var dismiss

    @State private var items: [ShoppingItem] = []
    @State private var newTitle: String = ""
    @State private var newQuantity: String = "1"
    @State private var selectedCategory: String = "Dispensa"
    @State private var isAdding: Bool = false
    @State private var isLoading: Bool = false

    private let categories = ["Ortofrutta", "Latticini & Bevande", "Dispensa", "Cura Casa", "Altro"]

    public init() {}

    public var body: some View {
        NavigationStack {
            ZStack {
                Color.domusSystemGroupedBackground.ignoresSafeArea()

                VStack(spacing: 0) {
                    // Barra di inserimento rapido
                    inputBar
                        .padding(.horizontal)
                        .padding(.top, 10)
                        .padding(.bottom, 6)

                    // Categoria rapida
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: 6) {
                            ForEach(categories, id: \.self) { cat in
                                Button(action: {
                                    selectedCategory = cat
                                }) {
                                    Text(cat)
                                        .font(.caption2.bold())
                                        .padding(.horizontal, 10)
                                        .padding(.vertical, 5)
                                        .background(selectedCategory == cat ? Color.cyan : Color.secondary.opacity(0.12))
                                        .foregroundColor(selectedCategory == cat ? .black : .primary)
                                        .clipShape(Capsule())
                                }
                            }
                        }
                        .padding(.horizontal)
                        .padding(.vertical, 4)
                    }

                    // Lista raggruppata per categoria
                    if isLoading && items.isEmpty {
                        Spacer()
                        ProgressView("Caricamento lista della spesa...")
                        Spacer()
                    } else if items.isEmpty {
                        Spacer()
                        VStack(spacing: 8) {
                            Image(systemName: "cart")
                                .font(.system(size: 44))
                                .foregroundColor(.secondary)
                            Text("Lista della spesa vuota")
                                .font(.headline)
                                .foregroundColor(.secondary)
                            Text("Aggiungi articoli usando la barra in alto")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                        Spacer()
                    } else {
                        List {
                            ForEach(categories, id: \.self) { cat in
                                let catItems = items.filter { $0.category == cat }
                                if !catItems.isEmpty {
                                    Section(header: Text(cat).font(.headline).foregroundColor(.primary)) {
                                        ForEach(catItems) { item in
                                            ShoppingItemRow(item: item) {
                                                toggleItem(item)
                                            }
                                        }
                                        .onDelete { indexSet in
                                            deleteItems(at: indexSet, in: cat)
                                        }
                                    }
                                }
                            }
                        }
                        .listStyle(.insetGrouped)
                    }
                }
            }
            .navigationTitle("Lista della Spesa")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Chiudi") {
                        dismiss()
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Menu {
                        Button(role: .destructive, action: clearCompleted) {
                            Label("Rimuovi completati", systemImage: "trash")
                        }
                    } label: {
                        Image(systemName: "ellipsis.circle")
                            .font(.system(size: 18))
                    }
                }
            }
            .task {
                await loadItems()
            }
            .refreshable {
                await loadItems()
            }
        }
    }

    private var inputBar: some View {
        HStack(spacing: 8) {
            TextField("Nuovo articolo...", text: $newTitle)
                .textFieldStyle(.plain)
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .background(Color.domusSecondarySystemGroupedBackground)
                .clipShape(RoundedRectangle(cornerRadius: 10))

            TextField("Qtà", text: $newQuantity)
                .textFieldStyle(.plain)
                .frame(width: 50)
                .padding(.horizontal, 8)
                .padding(.vertical, 8)
                .background(Color.domusSecondarySystemGroupedBackground)
                .clipShape(RoundedRectangle(cornerRadius: 10))

            Button(action: addItem) {
                Image(systemName: "plus.circle.fill")
                    .font(.system(size: 28))
                    .foregroundColor(newTitle.isEmpty ? .secondary : .cyan)
            }
            .disabled(newTitle.isEmpty)
        }
    }

    private func loadItems() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let dtos = try await DomusHttpClient.shared.fetchShopping()
            self.items = dtos.map {
                ShoppingItem(
                    id: $0.id,
                    title: $0.title,
                    quantity: $0.quantity,
                    category: $0.category,
                    completed: $0.completed,
                    completedBy: $0.completedBy,
                    createdAt: $0.createdAt ?? ""
                )
            }
        } catch {
            // Rete non raggiungibile
        }
    }

    private func addItem() {
        let title = newTitle.trimmingCharacters(in: .whitespaces)
        guard !title.isEmpty else { return }
        let qty = newQuantity.isEmpty ? "1" : newQuantity
        let cat = selectedCategory
        newTitle = ""
        newQuantity = "1"

        Task {
            do {
                _ = try await DomusHttpClient.shared.createShopping(title: title, quantity: qty, category: cat)
                #if canImport(UIKit)
                UIImpactFeedbackGenerator(style: .medium).impactOccurred()
                #endif
                await loadItems()
            } catch {
                #if canImport(UIKit)
                UINotificationFeedbackGenerator().notificationOccurred(.error)
                #endif
            }
        }
    }

    private func toggleItem(_ item: ShoppingItem) {
        Task {
            do {
                _ = try await DomusHttpClient.shared.toggleShopping(id: item.id)
                #if canImport(UIKit)
                UIImpactFeedbackGenerator(style: .light).impactOccurred()
                #endif
                await loadItems()
            } catch {
                #if canImport(UIKit)
                UINotificationFeedbackGenerator().notificationOccurred(.error)
                #endif
            }
        }
    }

    private func deleteItems(at offsets: IndexSet, in category: String) {
        let catItems = items.filter { $0.category == category }
        for index in offsets {
            let item = catItems[index]
            Task {
                try? await DomusHttpClient.shared.deleteShopping(id: item.id)
                await loadItems()
            }
        }
    }

    private func clearCompleted() {
        Task {
            do {
                try await DomusHttpClient.shared.clearCompletedShopping()
                await loadItems()
            } catch {
                // Rete non raggiungibile
            }
        }
    }
}

private struct ShoppingItemRow: View {
    let item: ShoppingItem
    var onToggle: () -> Void

    var body: some View {
        HStack(spacing: 12) {
            Button(action: onToggle) {
                Image(systemName: item.completed ? "checkmark.circle.fill" : "circle")
                    .font(.system(size: 22))
                    .foregroundColor(item.completed ? .green : .secondary)
            }
            .buttonStyle(.plain)

            VStack(alignment: .leading, spacing: 2) {
                Text(item.title)
                    .font(.body)
                    .strikethrough(item.completed, color: .secondary)
                    .foregroundColor(item.completed ? .secondary : .primary)

                if !item.quantity.isEmpty {
                    Text(item.quantity)
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }

            Spacer()
        }
        .padding(.vertical, 4)
        .contentShape(Rectangle())
        .onTapGesture(perform: onToggle)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(item.title), quantità \(item.quantity)")
        .accessibilityValue(item.completed ? "Completato" : "Da acquistare")
        .accessibilityAction(named: item.completed ? "Segna da acquistare" : "Segna completato", onToggle)
    }
}
