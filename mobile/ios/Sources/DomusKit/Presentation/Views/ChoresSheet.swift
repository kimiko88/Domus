import SwiftUI

public struct ChoreItem: Identifiable, Sendable {
    public let id: String
    public var title: String
    public var areaName: String
    public var assignedTo: String
    public var priority: String // "urgent", "high", "medium", "low"
    public var frequency: String
    public var completed: Bool
    public var karmaPoints: Int

    public init(
        id: String = UUID().uuidString,
        title: String,
        areaName: String = "Casa",
        assignedTo: String = "Tutti",
        priority: String = "medium",
        frequency: String = "Una tantum",
        completed: Bool = false,
        karmaPoints: Int = 20
    ) {
        self.id = id
        self.title = title
        self.areaName = areaName
        self.assignedTo = assignedTo
        self.priority = priority
        self.frequency = frequency
        self.completed = completed
        self.karmaPoints = karmaPoints
    }
}

public struct ChoresSheet: View {
    @Environment(\.dismiss) private var dismiss

    @State private var chores: [ChoreItem] = []
    @State private var selectedFilter: String = "Tutti"
    @State private var totalKarma: Int = 0
    @State private var isLoading: Bool = false
    @State private var showingAddSheet: Bool = false

    // Form per nuova faccenda
    @State private var newTitle: String = ""
    @State private var newArea: String = "Casa"
    @State private var newAssignee: String = "Tutti"
    @State private var newPriority: String = "medium"
    @State private var isSubmitting: Bool = false

    private let filters = ["Tutti", "Da Fare", "Completati"]
    private let roomSuggestions = ["Casa", "Cucina", "Salotto", "Bagno", "Camera", "Giardino"]
    private let familyMembers = ["Tutti", "Marco", "Giulia"]
    private let priorities = [
        ("low", "Bassa", Color.gray),
        ("medium", "Media", Color.blue),
        ("high", "Alta", Color.orange),
        ("urgent", "Urgente", Color.red)
    ]

    public init() {}

    public var body: some View {
        NavigationStack {
            ZStack {
                Color.domusSystemGroupedBackground.ignoresSafeArea()

                VStack(spacing: 0) {
                    // Karma Header Card
                    karmaHeader
                        .padding(.horizontal)
                        .padding(.top, 10)

                    // Leaderboard Card
                    leaderboardCard
                        .padding(.horizontal)
                        .padding(.top, 8)

                    // Picker filtro
                    Picker("Filtro", selection: $selectedFilter) {
                        ForEach(filters, id: \.self) { f in
                            Text(f).tag(f)
                        }
                    }
                    .pickerStyle(.segmented)
                    .padding(.horizontal)
                    .padding(.vertical, 10)

                    // Lista Faccende
                    if isLoading && chores.isEmpty {
                        Spacer()
                        ProgressView("Caricamento faccende...")
                        Spacer()
                    } else if filteredChores.isEmpty {
                        Spacer()
                        VStack(spacing: 8) {
                            Image(systemName: "checklist.checked")
                                .font(.system(size: 40))
                                .foregroundColor(.secondary)
                            Text("Nessuna faccenda presente")
                                .font(.headline)
                                .foregroundColor(.secondary)
                        }
                        Spacer()
                    } else {
                        List {
                            ForEach(filteredChores) { chore in
                                ChoreRowView(chore: chore) {
                                    toggleChore(chore)
                                }
                            }
                            .onDelete(perform: deleteChores)
                        }
#if os(iOS)
                        .listStyle(.insetGrouped)
#else
                        .listStyle(.inset)
#endif
                    }
                }
            }
            .navigationTitle("Faccende & Promemoria")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Chiudi") {
                        dismiss()
                    }
                }
                ToolbarItem(placement: .primaryAction) {
                    Button {
                        showingAddSheet = true
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
            .task {
                await loadChores()
            }
            .refreshable {
                await loadChores()
            }
            .sheet(isPresented: $showingAddSheet) {
                addChoreNavigationStack
            }
        }
    }

    private var karmaHeader: some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text("Punti Karma Famiglia")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                HStack(spacing: 6) {
                    Image(systemName: "star.fill")
                        .foregroundColor(.yellow)
                    Text("\(totalKarma) pts")
                        .font(.title2)
                        .fontWeight(.heavy)
                        .foregroundColor(.primary)
                }
            }
            Spacer()
            VStack(alignment: .trailing, spacing: 2) {
                let pending = chores.count { !$0.completed }
                Text("\(pending) in attesa")
                    .font(.caption)
                    .fontWeight(.bold)
                    .foregroundColor(.orange)
                Image(systemName: "medal.fill")
                    .font(.system(size: 28))
                    .foregroundColor(.orange)
            }
        }
        .padding(14)
        .background(Color.domusSecondarySystemGroupedBackground)
        .clipShape(RoundedRectangle(cornerRadius: 16))
    }

    private var leaderboardCard: some View {
        HStack {
            HStack(spacing: 8) {
                Text("👦 Marco")
                    .font(.subheadline)
                    .fontWeight(.bold)
                Text("75 Karma")
                    .font(.caption)
                    .foregroundColor(.cyan)
                    .fontWeight(.semibold)
            }
            Spacer()
            Divider()
                .frame(height: 20)
            Spacer()
            HStack(spacing: 8) {
                Text("👧 Giulia")
                    .font(.subheadline)
                    .fontWeight(.bold)
                Text("90 Karma 🏆")
                    .font(.caption)
                    .foregroundColor(.orange)
                    .fontWeight(.semibold)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
        .background(Color.domusSecondarySystemGroupedBackground.opacity(0.7))
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }

    private var filteredChores: [ChoreItem] {
        switch selectedFilter {
        case "Da Fare":    return chores.filter { !$0.completed }
        case "Completati": return chores.filter { $0.completed }
        default:           return chores
        }
    }

    private func loadChores() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let dtos = try await DomusHttpClient.shared.fetchChores()
            self.chores = dtos.map {
                ChoreItem(
                    id: $0.id,
                    title: $0.title,
                    areaName: $0.areaName.isEmpty ? "Casa" : $0.areaName,
                    assignedTo: $0.assignedTo.isEmpty ? "Tutti" : $0.assignedTo,
                    priority: $0.priority,
                    frequency: "Una tantum",
                    completed: $0.completed,
                    karmaPoints: $0.karmaPoints
                )
            }
            self.totalKarma = self.chores.filter { $0.completed }.reduce(0) { $0 + $1.karmaPoints }
        } catch {
            // In caso di errore di rete
        }
    }

    private func toggleChore(_ chore: ChoreItem) {
        Task {
            do {
                try await DomusHttpClient.shared.completeChore(id: chore.id)
                #if canImport(UIKit)
                UINotificationFeedbackGenerator().notificationOccurred(.success)
                #endif
                await loadChores()
            } catch {
                #if canImport(UIKit)
                UINotificationFeedbackGenerator().notificationOccurred(.error)
                #endif
            }
        }
    }

    private func deleteChores(at offsets: IndexSet) {
        for index in offsets {
            let chore = filteredChores[index]
            Task {
                try? await DomusHttpClient.shared.deleteChore(id: chore.id)
                await loadChores()
            }
        }
    }

    private var addChoreNavigationStack: some View {
        NavigationStack {
            Form {
                Section("Dettagli Attività") {
                    TextField("Nome attività (es. Svuota lavastoviglie)", text: $newTitle)

                    Picker("Stanza / Area", selection: $newArea) {
                        ForEach(roomSuggestions, id: \.self) { room in
                            Text(room).tag(room)
                        }
                    }

                    Picker("Assegnato a", selection: $newAssignee) {
                        ForEach(familyMembers, id: \.self) { member in
                            Text(member).tag(member)
                        }
                    }

                    Picker("Priorità", selection: $newPriority) {
                        ForEach(priorities, id: \.0) { p in
                            Text(p.1).tag(p.0)
                        }
                    }
                }
            }
            .navigationTitle("Nuova Faccenda")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annulla") {
                        showingAddSheet = false
                    }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Aggiungi") {
                        addNewChore()
                    }
                    .disabled(newTitle.trimmingCharacters(in: .whitespaces).isEmpty || isSubmitting)
                }
            }
        }
    }

    private func addNewChore() {
        let title = newTitle.trimmingCharacters(in: .whitespaces)
        guard !title.isEmpty else { return }
        isSubmitting = true

        Task {
            do {
                _ = try await DomusHttpClient.shared.createChore(
                    title: title,
                    areaName: newArea,
                    assignedTo: newAssignee,
                    priority: newPriority
                )
                #if canImport(UIKit)
                UINotificationFeedbackGenerator().notificationOccurred(.success)
                #endif
                await loadChores()
                await MainActor.run {
                    newTitle = ""
                    isSubmitting = false
                    showingAddSheet = false
                }
            } catch {
                await MainActor.run {
                    isSubmitting = false
                }
                #if canImport(UIKit)
                UINotificationFeedbackGenerator().notificationOccurred(.error)
                #endif
            }
        }
    }
}

private struct ChoreRowView: View {
    let chore: ChoreItem
    var onToggle: () -> Void

    var body: some View {
        HStack(spacing: 12) {
            Button(action: onToggle) {
                Image(systemName: chore.completed ? "checkmark.seal.fill" : "circle")
                    .font(.system(size: 24))
                    .foregroundColor(chore.completed ? .green : .secondary)
            }
            .buttonStyle(.plain)

            VStack(alignment: .leading, spacing: 4) {
                Text(chore.title)
                    .font(.body)
                    .fontWeight(.medium)
                    .strikethrough(chore.completed, color: .secondary)
                    .foregroundColor(chore.completed ? .secondary : .primary)

                HStack(spacing: 6) {
                    Text("📍 \(chore.areaName)")
                        .font(.caption)
                        .foregroundColor(.secondary)

                    Text("•")
                        .font(.caption)
                        .foregroundColor(.secondary)

                    Text("👤 \(chore.assignedTo)")
                        .font(.caption)
                        .fontWeight(.semibold)
                        .foregroundColor(.cyan)

                    Text("•")
                        .font(.caption)
                        .foregroundColor(.secondary)

                    Text("⏱️ \(chore.frequency)")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }

            Spacer()

            VStack(alignment: .trailing, spacing: 4) {
                priorityBadge
                Text("+\(chore.karmaPoints) pts")
                    .font(.caption2)
                    .fontWeight(.bold)
                    .foregroundColor(.orange)
            }
        }
        .padding(.vertical, 4)
    }

    private var priorityBadge: some View {
        let (color, label) = priorityData
        return Text(label)
            .font(.system(size: 9, weight: .bold))
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(color.opacity(0.15))
            .foregroundColor(color)
            .clipShape(Capsule())
    }

    private var priorityData: (Color, String) {
        switch chore.priority.lowercased() {
        case "urgent": return (.red, "URGENTE")
        case "high":   return (.orange, "ALTA")
        case "medium": return (.blue, "MEDIA")
        default:       return (.gray, "BASSA")
        }
    }
}
