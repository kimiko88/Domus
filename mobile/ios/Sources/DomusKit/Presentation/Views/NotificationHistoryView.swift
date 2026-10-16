import SwiftUI

public struct InAppNotificationItem: Identifiable, Sendable {
    public let id: String
    public let title: String
    public let message: String
    public let category: String // "Sicurezza", "Energia", "Dispositivi", "Sistema"
    public let timestamp: String
    public var isRead: Bool
    public let actions: [NotificationActionDTO]

    public init(
        id: String = UUID().uuidString,
        title: String,
        message: String,
        category: String,
        timestamp: String,
        isRead: Bool = false,
        actions: [NotificationActionDTO] = []
    ) {
        self.id = id
        self.title = title
        self.message = message
        self.category = category
        self.timestamp = timestamp
        self.isRead = isRead
        self.actions = actions
    }
}

public struct NotificationHistoryView: View {
    @Environment(\.dismiss) private var dismiss

    @State private var notifications: [InAppNotificationItem] = []
    @State private var selectedCategory: String = "Tutte"
    @State private var isLoading: Bool = false

    private let categories = ["Tutte", "Sicurezza", "Energia", "Dispositivi", "Sistema"]

    public init() {}

    public var body: some View {
        NavigationStack {
            ZStack {
                Color.domusSystemGroupedBackground.ignoresSafeArea()

                VStack(spacing: 0) {
                    // Category filter pills
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: 8) {
                            ForEach(categories, id: \.self) { cat in
                                categoryPill(cat)
                            }
                        }
                        .padding(.horizontal)
                        .padding(.vertical, 10)
                    }

                    // Notifications list
                    if isLoading && notifications.isEmpty {
                        Spacer()
                        ProgressView("Caricamento notifiche...")
                        Spacer()
                    } else if filteredNotifications.isEmpty {
                        Spacer()
                        VStack(spacing: 8) {
                            Image(systemName: "bell.slash")
                                .font(.system(size: 40))
                                .foregroundColor(.secondary)
                            Text("Nessuna notifica presente")
                                .font(.headline)
                                .foregroundColor(.secondary)
                        }
                        Spacer()
                    } else {
                        List {
                            ForEach(filteredNotifications) { item in
                                NotificationRowView(
                                    item: item,
                                    onMarkRead: { markAsRead(item) },
                                    onAction: { action in executeAction(item: item, action: action) }
                                )
                            }
                        }
#if os(iOS)
                        .listStyle(.insetGrouped)
#else
                        .listStyle(.inset)
#endif
                    }
                }
            }
            .navigationTitle("Centro Notifiche")
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
                    Button("Segna lette") {
                        markAllAsRead()
                    }
                }
            }
            .task {
                await loadNotifications()
            }
            .refreshable {
                await loadNotifications()
            }
        }
    }

    private func loadNotifications() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let dtos = try await DomusHttpClient.shared.fetchNotifications()
            self.notifications = dtos.map {
                InAppNotificationItem(
                    id: $0.id,
                    title: $0.title,
                    message: $0.message,
                    category: $0.category,
                    timestamp: $0.createdAt.isEmpty ? "Recente" : $0.createdAt,
                    isRead: $0.resolvedAt != nil,
                    actions: $0.actions ?? []
                )
            }
        } catch {
            // Rete non raggiungibile
        }
    }

    private var filteredNotifications: [InAppNotificationItem] {
        if selectedCategory == "Tutte" {
            return notifications
        }
        return notifications.filter { $0.category == selectedCategory }
    }

    private func categoryPill(_ cat: String) -> some View {
        let isSelected = selectedCategory == cat
        let label = categoryLabel(cat)
        let color = categoryColor(cat)
        return Button(action: { selectedCategory = cat }) {
            Text(label)
                .font(.caption)
                .fontWeight(.semibold)
                .padding(.horizontal, 14)
                .padding(.vertical, 7)
                .background(isSelected ? color : Color.domusSecondarySystemGroupedBackground)
                .foregroundColor(isSelected ? .white : .primary)
                .clipShape(Capsule())
        }
    }

    private func categoryLabel(_ cat: String) -> String {
        switch cat {
        case "Sicurezza": return "🛡️ Sicurezza"
        case "Energia":   return "⚡ Energia"
        case "Dispositivi": return "🔌 Dispositivi"
        case "Sistema":   return "⚙️ Sistema"
        default:          return "🔔 Tutte"
        }
    }

    private func categoryColor(_ cat: String) -> Color {
        switch cat {
        case "Sicurezza": return .red
        case "Energia":   return .orange
        case "Sistema":   return .indigo
        default:          return .cyan
        }
    }

    private func markAsRead(_ item: InAppNotificationItem) {
        if let idx = notifications.firstIndex(where: { $0.id == item.id }) {
            notifications[idx].isRead = true
        }
    }

    private func executeAction(item: InAppNotificationItem, action: NotificationActionDTO) {
        markAsRead(item)
        Task {
            try? await DomusHttpClient.shared.handleNotificationAction(notificationId: item.id, actionId: action.id)
            #if canImport(UIKit)
            UINotificationFeedbackGenerator().notificationOccurred(.success)
            #endif
        }
    }

    private func markAllAsRead() {
        withAnimation {
            for i in notifications.indices {
                notifications[i].isRead = true
            }
        }
    }
}

private struct NotificationRowView: View {
    let item: InAppNotificationItem
    var onMarkRead: () -> Void
    var onAction: (NotificationActionDTO) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 8) {
                if !item.isRead {
                    Circle()
                        .fill(categoryColor(item.category))
                        .frame(width: 8, height: 8)
                }

                Text(item.title)
                    .font(.subheadline)
                    .fontWeight(item.isRead ? .medium : .bold)

                Spacer()

                Text(item.timestamp)
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }

            Text(item.message)
                .font(.caption)
                .foregroundColor(.secondary)

            if !item.actions.isEmpty {
                HStack(spacing: 8) {
                    Spacer()
                    ForEach(item.actions) { action in
                        Button(action: { onAction(action) }) {
                            Text(action.title)
                                .font(.caption)
                                .fontWeight(.bold)
                                .padding(.horizontal, 12)
                                .padding(.vertical, 5)
                                .background((action.destructive == true ? Color.red : Color.cyan).opacity(0.15))
                                .foregroundColor(action.destructive == true ? .red : .cyan)
                                .clipShape(Capsule())
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.top, 4)
            }
        }
        .padding(.vertical, 4)
        .contentShape(Rectangle())
        .onTapGesture(perform: onMarkRead)
    }

    private func categoryColor(_ cat: String) -> Color {
        switch cat {
        case "Sicurezza": return .red
        case "Energia":   return .orange
        case "Sistema":   return .indigo
        default:          return .cyan
        }
    }
}
