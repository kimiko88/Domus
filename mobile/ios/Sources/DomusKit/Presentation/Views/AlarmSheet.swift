import SwiftUI

/// Scheda di controllo per Tastierino Allarme — Fase 2: PIN numerico, inserimento totale/parziale e disarmo.
public struct AlarmSheet: View {
    public let entity: Entity
    public var onArm: (String, String, String) -> Void   // (id, code, mode)
    public var onDisarm: (String, String) -> Void        // (id, code)

    @State private var pinCode: String = ""
    @State private var selectedAction: String? = nil    // "alarm_arm_home", "alarm_arm_away", "alarm_disarm"
    @State private var isError: Bool = false

    private let maxPinLength = 6

    public init(
        entity: Entity,
        onArm: @escaping (String, String, String) -> Void,
        onDisarm: @escaping (String, String) -> Void
    ) {
        self.entity = entity
        self.onArm = onArm
        self.onDisarm = onDisarm
    }

    public var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 20) {
                    // Badge stato allarme
                    statusHeader
                        .padding(.top, 16)

                    // Selettore Azione Desiderata
                    actionSelector

                    // Indicatore cifre PIN inserite (dots)
                    pinDotsView

                    // Tastierino Numerico
                    numericKeypad
                        .padding(.horizontal, 24)
                        .padding(.bottom, 24)
                }
            }
            .navigationTitle("Centrale Allarme")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
        }
    }

    // MARK: - Subviews

    private var statusHeader: some View {
        VStack(spacing: 8) {
            ZStack {
                Circle()
                    .fill(statusColor.opacity(0.2))
                    .frame(width: 80, height: 80)
                Image(systemName: statusIcon)
                    .font(.system(size: 36))
                    .foregroundColor(statusColor)
            }

            Text(statusTitle)
                .font(.title3.bold())

            Text(entity.name)
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }

    private var actionSelector: some View {
        HStack(spacing: 10) {
            actionButton(
                title: "Disarma",
                mode: "alarm_disarm",
                icon: "lock.open.fill",
                color: .green
            )

            actionButton(
                title: "Parziale (Notte)",
                mode: "alarm_arm_home",
                icon: "moon.fill",
                color: .orange
            )

            actionButton(
                title: "Totale (Fuori)",
                mode: "alarm_arm_away",
                icon: "shield.fill",
                color: .red
            )
        }
        .padding(.horizontal)
    }

    private func actionButton(title: String, mode: String, icon: String, color: Color) -> some View {
        let isSelected = selectedAction == mode
        return Button(action: {
            Haptics.selection()
            selectedAction = mode
        }) {
            VStack(spacing: 6) {
                Image(systemName: icon)
                    .font(.system(size: 20))
                Text(title)
                    .font(.system(size: 11, weight: .bold))
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 12)
            .background(isSelected ? color : Color.secondary.opacity(0.12))
            .foregroundColor(isSelected ? .white : .primary)
            .clipShape(RoundedRectangle(cornerRadius: 12))
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(color, lineWidth: isSelected ? 2 : 0)
            )
        }
    }

    private var pinDotsView: some View {
        VStack(spacing: 8) {
            Text(selectedAction == nil ? "Seleziona un'operazione sopra" : "Inserisci PIN di sicurezza")
                .font(.caption.bold())
                .foregroundColor(.secondary)

            HStack(spacing: 16) {
                ForEach(0..<4, id: \.self) { index in
                    Circle()
                        .fill(index < pinCode.count ? (isError ? Color.red : Color.cyan) : Color.secondary.opacity(0.2))
                        .frame(width: 14, height: 14)
                }
            }
            .padding(.vertical, 8)
        }
    }

    private var numericKeypad: some View {
        let keys = [
            ["1", "2", "3"],
            ["4", "5", "6"],
            ["7", "8", "9"],
            ["C", "0", "✓"]
        ]

        return VStack(spacing: 12) {
            ForEach(keys, id: \.self) { row in
                HStack(spacing: 12) {
                    ForEach(row, id: \.self) { key in
                        Button(action: { handleKey(key) }) {
                            Text(key)
                                .font(.system(size: 24, weight: .bold, design: .rounded))
                                .frame(width: 72, height: 72)
                                .background(keyColor(key))
                                .foregroundColor(keyTextColor(key))
                                .clipShape(Circle())
                        }
                        .disabled(selectedAction == nil && key != "C")
                    }
                }
            }
        }
    }

    // MARK: - Logic & Key Handling

    private func handleKey(_ key: String) {
        if key == "C" {
            Haptics.light()
            pinCode = ""
            isError = false
        } else if key == "✓" {
            submitAction()
        } else {
            if pinCode.count < maxPinLength {
                Haptics.light()
                pinCode.append(key)
                if pinCode.count == 4 {
                    // Auto submit when 4 digits entered if an action is picked
                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.15) {
                        submitAction()
                    }
                }
            }
        }
    }

    private func submitAction() {
        guard let action = selectedAction, !pinCode.isEmpty else {
            Haptics.warning()
            return
        }

        if action == "alarm_disarm" {
            onDisarm(entity.id, pinCode)
        } else {
            onArm(entity.id, pinCode, action)
        }

        pinCode = ""
    }

    private func keyColor(_ key: String) -> Color {
        if key == "✓" { return Color.cyan }
        if key == "C" { return Color.secondary.opacity(0.2) }
        return Color.secondary.opacity(0.12)
    }

    private func keyTextColor(_ key: String) -> Color {
        if key == "✓" { return .black }
        return .primary
    }

    private var statusColor: Color {
        switch entity.state.lowercased() {
        case "armed_away":  return .red
        case "armed_home":  return .orange
        case "armed_night": return .purple
        case "triggered":   return .red
        default:            return .green
        }
    }

    private var statusIcon: String {
        switch entity.state.lowercased() {
        case "armed_away":  return "shield.fill"
        case "armed_home":  return "house.and.flag.fill"
        case "armed_night": return "moon.fill"
        case "triggered":   return "exclamationmark.shield.fill"
        default:            return "lock.open.fill"
        }
    }

    private var statusTitle: String {
        switch entity.state.lowercased() {
        case "armed_away":  return "ARMATO TOTALE"
        case "armed_home":  return "ARMATO PARZIALE"
        case "armed_night": return "ARMATO NOTTE"
        case "triggered":   return "ALLARME IN CORSO"
        default:            return "DISARMATO"
        }
    }
}
