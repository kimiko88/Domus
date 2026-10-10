import SwiftUI
import DomusKit

public struct ConnectionStatusBannerView: View {
    public let connection: ServerConnection
    public var onReconnect: () -> Void = {}

    public init(connection: ServerConnection, onReconnect: @escaping () -> Void = {}) {
        self.connection = connection
        self.onReconnect = onReconnect
    }

    public var body: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(statusColor)
                .frame(width: 8, height: 8)

            Text(statusTitle)
                .font(.system(size: 12, weight: .semibold))
                .foregroundColor(.primary)

            if !connection.isConnected && connection.mode != .connecting {
                Button("Riconnetti", action: onReconnect)
                    .font(.system(size: 11, weight: .bold))
                    .foregroundColor(.blue)
            }
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 5)
        .background(statusColor.opacity(0.15))
        .clipShape(Capsule())
    }

    private var statusColor: Color {
        switch connection.mode {
        case .localLan: return .green
        case .remoteTls: return .blue
        case .connecting: return .orange
        case .disconnected: return .red
        }
    }

    private var statusTitle: String {
        switch connection.mode {
        case .localLan: return "LAN Diretta • \(connection.latencyMs)ms"
        case .remoteTls: return "Remoto TLS • \(connection.latencyMs)ms"
        case .connecting: return "Ricerca Hub..."
        case .disconnected: return "Disconnesso"
        }
    }
}

public struct EntityCardView: View {
    public let entity: Entity
    public var onToggle: (String) -> Void

    public init(entity: Entity, onToggle: @escaping (String) -> Void) {
        self.entity = entity
        self.onToggle = onToggle
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                ZStack {
                    RoundedRectangle(cornerRadius: 10)
                        .fill(entity.isOn ? Color.cyan.opacity(0.2) : Color.gray.opacity(0.2))
                        .frame(width: 38, height: 38)
                    Text(domainIcon)
                        .font(.system(size: 18))
                }

                Spacer()

                if entity.isLight || entity.isSwitch {
                    Toggle("", isOn: Binding(
                        get: { entity.isOn },
                        set: { _ in onToggle(entity.id) }
                    ))
                    .labelsHidden()
                    .toggleStyle(SwitchToggleStyle(tint: .cyan))
                } else {
                    Text(entity.state.uppercased())
                        .font(.system(size: 10, weight: .bold))
                        .padding(.horizontal, 6)
                        .padding(.vertical, 3)
                        .background(Color.secondary.opacity(0.2))
                        .clipShape(RoundedRectangle(cornerRadius: 6))
                }
            }

            Text(entity.name)
                .font(.system(size: 15, weight: .bold))
                .lineLimit(1)

            Text(subtitle)
                .font(.system(size: 12))
                .foregroundColor(.secondary)
                .lineLimit(1)
        }
        .padding(14)
        .background(entity.isOn ? Color.cyan.opacity(0.08) : Color(uiColorOrFallback: .secondarySystemBackground))
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .contentShape(Rectangle())
        .onTapGesture {
            onToggle(entity.id)
        }
    }

    private var domainIcon: String {
        switch entity.domain {
        case "light": return "💡"
        case "switch": return "🔌"
        case "climate": return "🌡️"
        case "sensor": return "📊"
        case "alarm_control_panel": return "🛡️"
        default: return "⚙️"
        }
    }

    private var subtitle: String {
        if entity.isLight && entity.isOn {
            return "Accesa • \(entity.brightness ?? 100)%"
        } else if entity.isLight && !entity.isOn {
            return "Spenta"
        } else if entity.isClimate {
            return "Target: \(entity.currentTemperature ?? 21.0)°C"
        } else {
            return entity.domain.capitalized
        }
    }
}

extension Color {
    init(uiColorOrFallback: Any) {
        self = Color.gray.opacity(0.12)
    }
}
