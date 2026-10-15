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

    @State private var items: [ShoppingItem] = [
        ShoppingItem(title: "Latte di avena", quantity: "2L", category: "Latticini & Bevande", completed: false),
        ShoppingItem(title: "Mele Golden", quantity: "1 kg", category: "Ortofrutta", completed: false),
        ShoppingItem(title: "Pasta integrale", quantity: "500g", category: "Dispensa", completed: true),
        ShoppingItem(title: "Detersivo piatti ecologico", quantity: "1 flacone", category: "Cura Casa", completed: false)
    ]
    @State private var newTitle: String = ""
    @State private var newQuantity: String = "1"
    @State private var selectedCategory: String = "Dispensa"
    @State private var isAdding: Bool = false

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

                    // Lista raggruppata per categoria
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

    private func addItem() {
        guard !newTitle.trimmingCharacters(in: .whitespaces).isEmpty else { return }
        let newItem = ShoppingItem(
            title: newTitle.trimmingCharacters(in: .whitespaces),
            quantity: newQuantity.isEmpty ? "1" : newQuantity,
            category: selectedCategory,
            completed: false
        )
        withAnimation(.spring(response: 0.3, dampingFraction: 0.7)) {
            items.insert(newItem, at: 0)
            newTitle = ""
            newQuantity = "1"
        }
        #if canImport(UIKit)
        UIImpactFeedbackGenerator(style: .medium).impactOccurred()
        #endif
    }

    private func toggleItem(_ item: ShoppingItem) {
        if let idx = items.firstIndex(where: { $0.id == item.id }) {
            withAnimation(.spring(response: 0.35, dampingFraction: 0.75)) {
                items[idx].completed.toggle()
            }
            #if canImport(UIKit)
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
            #endif
        }
    }

    private func deleteItems(at offsets: IndexSet, in category: String) {
        let catItems = items.filter { $0.category == category }
        let toRemove = offsets.map { catItems[$0] }
        withAnimation {
            items.removeAll { toRemove.contains($0) }
        }
    }

    private func clearCompleted() {
        withAnimation(.easeInOut) {
            items.removeAll { $0.completed }
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
