import SwiftUI

public struct ChoreItem: Identifiable, Sendable {
    public let id: String
    public var title: String
    public var areaName: String
    public var assignedTo: String
    public var priority: String // "urgent", "high", "medium", "low"
    public var completed: Bool
    public var karmaPoints: Int

    public init(
        id: String = UUID().uuidString,
        title: String,
        areaName: String,
        assignedTo: String,
        priority: String = "medium",
        completed: Bool = false,
        karmaPoints: Int = 20
    ) {
        self.id = id
        self.title = title
        self.areaName = areaName
        self.assignedTo = assignedTo
        self.priority = priority
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

    private let filters = ["Tutti", "Da Fare", "Completati"]

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
                        }
                        .listStyle(.insetGrouped)
                    }
                }
            }
            .navigationTitle("Faccende & Promemoria")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Chiudi") {
                        dismiss()
                    }
                }
            }
            .task {
                await loadChores()
            }
            .refreshable {
                await loadChores()
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
            Image(systemName: "medal.fill")
                .font(.system(size: 34))
                .foregroundColor(.orange)
        }
        .padding(14)
        .background(Color.domusSecondarySystemGroupedBackground)
        .clipShape(RoundedRectangle(cornerRadius: 16))
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
                    areaName: $0.areaName,
                    assignedTo: $0.assignedTo,
                    priority: $0.priority,
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
                    Text(chore.areaName)
                        .font(.caption)
                        .foregroundColor(.secondary)

                    Text("•")
                        .font(.caption)
                        .foregroundColor(.secondary)

                    Text(chore.assignedTo)
                        .font(.caption)
                        .fontWeight(.semibold)
                        .foregroundColor(.cyan)
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
        switch chore.priority {
        case "urgent": return (.red, "URGENTE")
        case "high":   return (.orange, "ALTA")
        case "medium": return (.blue, "MEDIA")
        default:       return (.gray, "BASSA")
        }
    }
}
