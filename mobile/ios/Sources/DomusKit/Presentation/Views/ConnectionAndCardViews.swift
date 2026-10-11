import SwiftUI

/// Banner di stato connessione — vive in DomusKit, importata dal target Xcode.
public struct ConnectionStatusBannerView: View {
    public let connection: ServerConnection
    public var onReconnect: () -> Void

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
        case .localLan:     return .green
        case .remoteTls:    return .blue
        case .connecting:   return .orange
        case .disconnected: return .red
        }
    }

    private var statusTitle: String {
        switch connection.mode {
        case .localLan:     return "LAN • \(connection.latencyMs)ms"
        case .remoteTls:    return "Remoto • \(connection.latencyMs)ms"
        case .connecting:   return "Ricerca Hub..."
        case .disconnected: return "Disconnesso"
        }
    }
}

/// Card per una singola entità — vive in DomusKit, con supporto ad apertura dettaglio nativo Fase 2.
public struct EntityCardView: View {
    public let entity: Entity
    public var onToggle: (String) -> Void
    public var onOpenDetail: (Entity) -> Void

    public init(
        entity: Entity,
        onToggle: @escaping (String) -> Void,
        onOpenDetail: @escaping (Entity) -> Void = { _ in }
    ) {
        self.entity = entity
        self.onToggle = onToggle
        self.onOpenDetail = onOpenDetail
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                ZStack {
                    RoundedRectangle(cornerRadius: 10)
                        .fill(cardAccentColor.opacity(0.2))
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
                    .toggleStyle(SwitchToggleStyle(tint: cardAccentColor))
                } else {
                    statusBadge
                }
            }

            VStack(alignment: .leading, spacing: 3) {
                Text(entity.name)
                    .font(.system(size: 15, weight: .bold))
                    .lineLimit(1)

                Text(subtitle)
                    .font(.system(size: 12))
                    .foregroundColor(.secondary)
                    .lineLimit(1)
            }

            // Barra progressiva luminosità o badge potenza istantanea
            if entity.isLight && entity.isOn, let b = entity.brightness {
                GeometryReader { geo in
                    ZStack(alignment: .leading) {
                        Capsule()
                            .fill(Color.secondary.opacity(0.2))
                            .frame(height: 4)
                        Capsule()
                            .fill(cardAccentColor)
                            .frame(width: geo.size.width * CGFloat(b) / 100.0, height: 4)
                    }
                }
                .frame(height: 4)
            } else if entity.isSwitch && entity.isOn, let watts = entity.powerWatts {
                HStack(spacing: 4) {
                    Image(systemName: "bolt.fill")
                        .font(.system(size: 9))
                    Text(String(format: "%.0f W", watts))
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                }
                .foregroundColor(.yellow)
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(Color.yellow.opacity(0.12))
                .clipShape(Capsule())
            }
        }
        .padding(14)
        .background(entity.isOn ? cardAccentColor.opacity(0.08) : cardBackground)
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .contentShape(Rectangle())
        .onTapGesture {
            onOpenDetail(entity)
        }
    }

    // MARK: - Helpers

    private var cardAccentColor: Color {
        if entity.isLight { return .cyan }
        if entity.isSwitch { return .green }
        if entity.isClimate { return entity.isCooling ? .blue : .orange }
        if entity.isAlarm { return entity.isArmed ? .red : .green }
        if entity.isCamera { return .purple }
        return .cyan
    }

    private var domainIcon: String {
        switch entity.domain {
        case "light":               return "💡"
        case "switch":              return "🔌"
        case "climate":             return "🌡️"
        case "sensor":              return "📊"
        case "alarm_control_panel": return "🛡️"
        case "camera":              return "📹"
        default:                    return "⚙️"
        }
    }

    private var subtitle: String {
        if entity.isLight && entity.isOn    { return "Luminosità • \(entity.brightness ?? 100)%" }
        if entity.isLight && !entity.isOn   { return "Spenta" }
        if entity.isSwitch && entity.isOn   { return "Presa Attiva" }
        if entity.isSwitch && !entity.isOn  { return "Presa Disattivata" }
        if entity.isClimate                 {
            let curr = entity.currentTemperature.map { String(format: "%.1f°C", $0) } ?? "--"
            let target = entity.targetTemperature.map { String(format: "%.1f°C", $0) } ?? "--"
            return "\(curr) → Target \(target)"
        }
        if entity.isAlarm                   { return entity.isArmed ? "Armato" : "Disarmato" }
        if entity.isCamera                  { return "Live Streaming" }
        return entity.domain.capitalized
    }

    private var statusBadge: some View {
        Text(entity.state.uppercased())
            .font(.system(size: 10, weight: .bold))
            .padding(.horizontal, 6)
            .padding(.vertical, 3)
            .background(cardAccentColor.opacity(0.2))
            .foregroundColor(cardAccentColor)
            .clipShape(RoundedRectangle(cornerRadius: 6))
    }

    private var cardBackground: Color {
        #if canImport(UIKit)
        return Color(UIColor.secondarySystemBackground)
        #else
        return Color(.windowBackgroundColor)
        #endif
    }
}
