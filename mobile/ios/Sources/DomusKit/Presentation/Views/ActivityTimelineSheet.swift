import SwiftUI

/// Vista Timeline/Cronologia degli eventi recenti e Audit Log della casa ("Timeline della Casa").
public struct ActivityTimelineSheet: View {
    @Environment(\.dismiss) private var dismiss

    @State private var events: [TimelineEvent] = TimelineEvent.sampleTimeline
    @State private var selectedCategory: TimelineCategory = .all
    @State private var searchText: String = ""
    @State private var isRefreshing: Bool = false

    public init() {}

    private var filteredEvents: [TimelineEvent] {
        events.filter { event in
            let matchesCategory = (selectedCategory == .all || event.category == selectedCategory)
            let matchesSearch = searchText.isEmpty
                || event.message.localizedCaseInsensitiveContains(searchText)
                || event.entityName.localizedCaseInsensitiveContains(searchText)
                || (event.contextName?.localizedCaseInsensitiveContains(searchText) ?? false)
            return matchesCategory && matchesSearch
        }
    }

    public var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Barra di Ricerca
                HStack(spacing: 8) {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("Cerca negli eventi recenti...", text: $searchText)
                        .textFieldStyle(.plain)
                        .accessibilityLabel("Cerca eventi nella cronologia")
                    if !searchText.isEmpty {
                        Button(action: { searchText = "" }) {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundColor(.secondary)
                        }
                    }
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .background(Color.secondary.opacity(0.1))
                .clipShape(RoundedRectangle(cornerRadius: 12))
                .padding(.horizontal)
                .padding(.top, 10)

                // Filtri Categoria a Pills Orizzontali
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        ForEach(TimelineCategory.allCases) { cat in
                            Button(action: {
                                Haptics.selection()
                                selectedCategory = cat
                            }) {
                                HStack(spacing: 5) {
                                    Image(systemName: cat.icon)
                                        .font(.caption2.bold())
                                    Text(cat.rawValue)
                                        .font(.caption.bold())
                                }
                                .padding(.horizontal, 12)
                                .padding(.vertical, 7)
                                .background(selectedCategory == cat ? Color.blue : Color.secondary.opacity(0.12))
                                .foregroundColor(selectedCategory == cat ? .white : .primary)
                                .clipShape(Capsule())
                            }
                            .accessibilityLabel("Filtra per \(cat.rawValue)")
                            .accessibilityAddTraits(selectedCategory == cat ? [.isSelected] : [])
                        }
                    }
                    .padding(.horizontal)
                    .padding(.vertical, 10)
                }

                Divider()

                // Lista Cronologia Eventi
                if filteredEvents.isEmpty {
                    Spacer()
                    VStack(spacing: 12) {
                        Image(systemName: "clock.arrow.circlepath")
                            .font(.system(size: 44))
                            .foregroundColor(.secondary.opacity(0.6))
                        Text("Nessun evento trovato")
                            .font(.headline)
                            .foregroundColor(.secondary)
                        Text("Modifica i filtri o la ricerca per consultare lo storico.")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    Spacer()
                } else {
                    List {
                        ForEach(filteredEvents) { event in
                            timelineEventRow(event)
                                .listRowSeparator(.hidden)
                                .listRowInsets(EdgeInsets(top: 6, leading: 16, bottom: 6, trailing: 16))
                        }
                    }
                    .listStyle(.plain)
                    .refreshable {
                        await refreshTimeline()
                    }
                }
            }
            .navigationTitle("Timeline della Casa")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Chiudi") {
                        dismiss()
                    }
                }
            }
        }
    }

    // MARK: - Riga Evento Timeline

    private func timelineEventRow(_ event: TimelineEvent) -> some View {
        HStack(alignment: .top, spacing: 14) {
            // Timestamp orario
            VStack(alignment: .trailing, spacing: 2) {
                Text(event.formattedTime)
                    .font(.subheadline.monospacedDigit().bold())
                    .foregroundColor(.primary)
                Text("Oggi")
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }
            .frame(width: 48, alignment: .trailing)

            // Indicatore ad asse verticale
            VStack(spacing: 0) {
                Circle()
                    .fill(colorForCategory(event.category))
                    .frame(width: 12, height: 12)
                    .overlay(
                        Circle()
                            .stroke(Color.white, lineWidth: 2)
                    )
                Rectangle()
                    .fill(Color.secondary.opacity(0.2))
                    .frame(width: 2)
                    .frame(maxHeight: .infinity)
            }

            // Scheda Contenuto Evento
            VStack(alignment: .leading, spacing: 6) {
                HStack {
                    Text(event.message)
                        .font(.subheadline.bold())
                        .foregroundColor(.primary)
                    Spacer()
                    Image(systemName: event.categoryIcon)
                        .font(.caption)
                        .foregroundColor(colorForCategory(event.category))
                }

                HStack(spacing: 6) {
                    if let actor = event.contextName {
                        HStack(spacing: 4) {
                            Image(systemName: actorIcon(event.contextType))
                                .font(.system(size: 9))
                            Text(actor)
                                .font(.caption2.bold())
                        }
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color.secondary.opacity(0.12))
                        .clipShape(Capsule())
                    }

                    Text(event.entityName)
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }
            }
            .padding(12)
            .background(Color.domusSecondarySystemGroupedBackground)
            .clipShape(RoundedRectangle(cornerRadius: 14))
            .shadow(color: Color.black.opacity(0.04), radius: 4, x: 0, y: 2)
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(event.formattedTime): \(event.message), eseguito da \(event.contextName ?? "sistema") su \(event.entityName)")
    }

    private func colorForCategory(_ category: TimelineCategory) -> Color {
        switch category {
        case .security: return .red
        case .access: return .blue
        case .climateAndLights: return .orange
        case .energy: return .yellow
        case .all: return .purple
        }
    }

    private func actorIcon(_ contextType: String) -> String {
        switch contextType {
        case "user": return "person.fill"
        case "automation": return "wand.and.stars"
        case "system": return "cpu"
        default: return "app.fill"
        }
    }

    private func refreshTimeline() async {
        isRefreshing = true
        Haptics.light()

        // Sync da endpoint logbook se disponibile
        if let liveEntries = try? await DomusHttpClient.shared.get(path: "/api/v1/logbook", as: [TimelineEvent].self), !liveEntries.isEmpty {
            self.events = liveEntries
        }
        isRefreshing = false
        DomusAccessibility.announce("Cronologia aggiornata")
    }
}
