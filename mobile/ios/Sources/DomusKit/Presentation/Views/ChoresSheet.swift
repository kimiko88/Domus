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

    @State private var chores: [ChoreItem] = [
        ChoreItem(title: "Pulizia filtro aspirapolvere", areaName: "Soggiorno", assignedTo: "Fabrizio", priority: "high", completed: false, karmaPoints: 30),
        ChoreItem(title: "Innaffiare le piante sul balcone", areaName: "Terrazzo", assignedTo: "Sara", priority: "medium", completed: false, karmaPoints: 15),
        ChoreItem(title: "Svuotare la lavastoviglie", areaName: "Cucina", assignedTo: "Tutti", priority: "low", completed: true, karmaPoints: 10),
        ChoreItem(title: "Sostituzione pastiglie anticalcare", areaName: "Bagno", assignedTo: "Fabrizio", priority: "urgent", completed: false, karmaPoints: 50)
    ]
    @State private var selectedFilter: String = "Tutti"
    @State private var totalKarma: Int = 145

    private let filters = ["Tutti", "Da Fare", "Completati"]

    public init() {}

    public var body: some View {
        NavigationStack {
            ZStack {
                Color(uiColor: .systemGroupedBackground).ignoresSafeArea()

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
            .navigationTitle("Faccende & Promemoria")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Chiudi") {
                        dismiss()
                    }
                }
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
        .background(Color(uiColor: .secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 16))
    }

    private var filteredChores: [ChoreItem] {
        switch selectedFilter {
        case "Da Fare":    return chores.filter { !$0.completed }
        case "Completati": return chores.filter { $0.completed }
        default:           return chores
        }
    }

    private func toggleChore(_ chore: ChoreItem) {
        if let idx = chores.firstIndex(where: { $0.id == chore.id }) {
            withAnimation(.spring(response: 0.35, dampingFraction: 0.75)) {
                chores[idx].completed.toggle()
                if chores[idx].completed {
                    totalKarma += chores[idx].karmaPoints
                } else {
                    totalKarma -= chores[idx].karmaPoints
                }
            }
            #if canImport(UIKit)
            UINotificationFeedbackGenerator().notificationOccurred(.success)
            #endif
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
