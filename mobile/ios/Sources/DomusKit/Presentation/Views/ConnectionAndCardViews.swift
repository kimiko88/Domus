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

    @Environment(\.accessibilityDifferentiateWithoutColor) private var differentiateWithoutColor
    @Environment(\.accessibilityReduceTransparency) private var reduceTransparency
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

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
            headerSection
            titleSection
            bottomIndicatorSection
        }
        .padding(14)
        .background(cardBackgroundView)
        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
        .shadow(color: shadowColor, radius: shadowRadius, x: 0, y: shadowY)
        .contentShape(Rectangle())
        .onTapGesture {
            onOpenDetail(entity)
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(Text("\(entity.name), \(domainAccessibilityLabel)"))
        .accessibilityValue(Text(accessibilityValueText))
        .accessibilityHint(Text("Tocca due volte per visualizzare i dettagli o controlla con le azioni del rotore"))
        .accessibilityAddTraits(entity.domain == "sensor" ? [] : .isButton)
        .accessibilityAction(named: accessibilityActionName) {
            if entity.isLight || entity.isSwitch {
                onToggle(entity.id)
            }
        }
    }

    // MARK: - Subviews

    private var headerSection: some View {
        HStack {
            ZStack {
                RoundedRectangle(cornerRadius: 10)
                    .fill(cardAccentColor.opacity(reduceTransparency ? 0.35 : 0.2))
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
    }

    private var titleSection: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(entity.name)
                .font(.subheadline.bold())
                .lineLimit(2)

            Text(subtitle)
                .font(.caption)
                .foregroundColor(.secondary)
                .lineLimit(1)
        }
    }

    @ViewBuilder
    private var bottomIndicatorSection: some View {
        if entity.isLight && entity.isOn, let b = entity.brightness {
            brightnessProgressBar(brightness: b)
        } else if entity.isSwitch && entity.isOn, let watts = entity.powerWatts {
            powerWattsBadge(watts: watts)
        }
    }

    private func brightnessProgressBar(brightness: Int) -> some View {
        GeometryReader { geo in
            ZStack(alignment: .leading) {
                Capsule()
                    .fill(Color.secondary.opacity(0.2))
                    .frame(height: 4)
                Capsule()
                    .fill(cardAccentColor)
                    .frame(width: geo.size.width * CGFloat(brightness) / 100.0, height: 4)
            }
        }
        .frame(height: 4)
    }

    private func powerWattsBadge(watts: Double) -> some View {
        HStack(spacing: 4) {
            Image(systemName: "bolt.fill")
                .font(.system(size: 9))
            Text(String(format: "%.0f W", watts))
                .font(.caption2.monospaced().bold())
        }
        .foregroundColor(.yellow)
        .padding(.horizontal, 6)
        .padding(.vertical, 2)
        .background(Color.yellow.opacity(0.12))
        .clipShape(Capsule())
    }

    private var cardBackgroundView: some View {
        RoundedRectangle(cornerRadius: 18, style: .continuous)
            .fill(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.domusSecondarySystemGroupedBackground.opacity(0.85))
            .overlay(cardGradientBorder)
            .overlay(cardHighContrastBorder)
            .overlay(cardActiveTintOverlay)
    }

    private var cardGradientBorder: some View {
        let opacity1: Double = entity.isOn ? 0.45 : 0.15
        let opacity2: Double = entity.isOn ? 0.15 : 0.05
        let gradient = LinearGradient(
            colors: [cardAccentColor.opacity(opacity1), cardAccentColor.opacity(opacity2)],
            startPoint: .topLeading,
            endPoint: .bottomTrailing
        )
        return RoundedRectangle(cornerRadius: 18, style: .continuous)
            .stroke(gradient, lineWidth: 1)
    }

    private var cardHighContrastBorder: some View {
        let strokeColor: Color = differentiateWithoutColor ? cardAccentColor.opacity(entity.isOn ? 0.9 : 0.4) : Color.clear
        return RoundedRectangle(cornerRadius: 18, style: .continuous)
            .stroke(strokeColor, lineWidth: 1.5)
    }

    private var cardActiveTintOverlay: some View {
        let fillColor: Color = entity.isOn ? cardAccentColor.opacity(0.08) : Color.clear
        return RoundedRectangle(cornerRadius: 18, style: .continuous)
            .fill(fillColor)
    }

    private var shadowColor: Color {
        if reduceMotion { return .clear }
        return entity.isOn ? cardAccentColor.opacity(0.18) : Color.black.opacity(0.06)
    }

    private var shadowRadius: CGFloat {
        entity.isOn ? 10 : 4
    }

    private var shadowY: CGFloat {
        entity.isOn ? 4 : 2
    }

    private var accessibilityActionName: Text {
        guard entity.isLight || entity.isSwitch else {
            return Text("")
        }
        return entity.isOn ? Text("Spegni") : Text("Accendi")
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
        return Color.domusSecondarySystemBackground
    }

    private var domainAccessibilityLabel: String {
        switch entity.domain {
        case "light":               return "Luce"
        case "switch":              return "Interruttore"
        case "climate":             return "Climatizzazione"
        case "sensor":              return "Sensore"
        case "alarm_control_panel": return "Pannello Allarme"
        case "camera":              return "Telecamera"
        default:                    return entity.domain.capitalized
        }
    }

    private var accessibilityValueText: String {
        if entity.isLight {
            return entity.isOn ? "Accesa al \(entity.brightness ?? 100)%" : "Spenta"
        }
        if entity.isSwitch {
            return entity.isOn ? "Attivo" : "Disattivato"
        }
        if entity.isAlarm {
            return entity.isArmed ? "Sistema allarme armato" : "Disarmato"
        }
        return subtitle
    }
}
