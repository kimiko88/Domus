import SwiftUI

public struct InAppNotificationItem: Identifiable, Sendable {
    public let id: String
    public let title: String
    public let message: String
    public let category: String // "security", "energy", "devices", "system"
    public let timestamp: String
    public var isRead: Bool
    public let quickAction: String?

    public init(
        id: String = UUID().uuidString,
        title: String,
        message: String,
        category: String,
        timestamp: String,
        isRead: Bool = false,
        quickAction: String? = nil
    ) {
        self.id = id
        self.title = title
        self.message = message
        self.category = category
        self.timestamp = timestamp
        self.isRead = isRead
        self.quickAction = quickAction
    }
}

public struct NotificationHistoryView: View {
    @Environment(\.dismiss) private var dismiss

    @State private var notifications: [InAppNotificationItem] = []
    @State private var selectedCategory: String = "Tutte"
    @State private var isLoading: Bool = false

    private let categories = ["Tutte", "security", "energy", "devices", "system"]

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
                                NotificationRowView(item: item) {
                                    markAsRead(item)
                                }
                            }
                        }
                        .listStyle(.insetGrouped)
                    }
                }
            }
            .navigationTitle("Centro Notifiche")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Chiudi") {
                        dismiss()
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
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
                    timestamp: $0.createdAt,
                    isRead: false,
                    quickAction: $0.action
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
        return Button(action: { selectedCategory = cat }) {
            Text(label)
                .font(.caption)
                .fontWeight(.semibold)
                .padding(.horizontal, 14)
                .padding(.vertical, 7)
                .background(isSelected ? Color.cyan : Color.domusSecondarySystemGroupedBackground)
                .foregroundColor(isSelected ? .white : .primary)
                .clipShape(Capsule())
        }
    }

    private func categoryLabel(_ cat: String) -> String {
        switch cat {
        case "security": return "🛡️ Sicurezza"
        case "energy":   return "⚡ Energia"
        case "devices":  return "🔌 Dispositivi"
        case "system":   return "⚙️ Sistema"
        default:         return "Tutte"
        }
    }

    private func markAsRead(_ item: InAppNotificationItem) {
        if let idx = notifications.firstIndex(where: { $0.id == item.id }) {
            notifications[idx].isRead = true
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

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 8) {
                if !item.isRead {
                    Circle()
                        .fill(Color.cyan)
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

            if let action = item.quickAction {
                HStack {
                    Spacer()
                    Button(action: onMarkRead) {
                        Text(action)
                            .font(.caption)
                            .fontWeight(.bold)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 5)
                            .background(Color.cyan.opacity(0.15))
                            .foregroundColor(.cyan)
                            .clipShape(Capsule())
                    }
                    .buttonStyle(.plain)
                }
                .padding(.top, 4)
            }
        }
        .padding(.vertical, 4)
        .contentShape(Rectangle())
        .onTapGesture(perform: onMarkRead)
    }
}
