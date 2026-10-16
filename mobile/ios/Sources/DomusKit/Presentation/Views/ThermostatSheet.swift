import SwiftUI

/// Scheda di controllo per Termostato — Fase 2: Termostato circolare interattivo con feedback aptico e selettore HVAC.
public struct ThermostatSheet: View {
    public let entity: Entity
    public var onTemperatureChange: (String, Double) -> Void
    public var onModeChange: (String, String) -> Void

    @Environment(\.accessibilityDifferentiateWithoutColor) private var differentiateWithoutColor
    @Environment(\.accessibilityReduceTransparency) private var reduceTransparency

    @State private var targetTemperature: Double
    @State private var dragAngle: Double = 0.0

    private let minTemp: Double = 15.0
    private let maxTemp: Double = 30.0

    public init(
        entity: Entity,
        onTemperatureChange: @escaping (String, Double) -> Void,
        onModeChange: @escaping (String, String) -> Void
    ) {
        self.entity = entity
        self.onTemperatureChange = onTemperatureChange
        self.onModeChange = onModeChange
        let initialTemp = entity.targetTemperature ?? 21.0
        _targetTemperature = State(initialValue: initialTemp)
    }

    public var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 24) {
                    // Badge stato termico
                    HStack(spacing: 8) {
                        Circle()
                            .fill(statusColor)
                            .frame(width: 10, height: 10)
                            .overlay(
                                Circle()
                                    .stroke(Color.primary, lineWidth: differentiateWithoutColor ? 1.5 : 0)
                            )
                        Text(statusDescription)
                            .font(.footnote.weight(.bold))
                            .foregroundColor(statusColor)
                    }
                    .padding(.horizontal, 14)
                    .padding(.vertical, 8)
                    .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : statusColor.opacity(0.12))
                    .clipShape(Capsule())
                    .overlay(
                        Capsule()
                            .stroke(statusColor.opacity(differentiateWithoutColor ? 0.6 : 0), lineWidth: 1)
                    )
                    .padding(.top, 16)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Stato termostato: \(statusDescription)")

                    // Ghiera circolare interattiva del termostato
                    ZStack {
                        // Traccia di sfondo (arco a 240 gradi)
                        Circle()
                            .trim(from: 0.15, to: 0.85)
                            .stroke(
                                Color.secondary.opacity(0.18),
                                style: StrokeStyle(lineWidth: 18, lineCap: .round)
                            )
                            .rotationEffect(.degrees(90))
                            .frame(width: 230, height: 230)

                        // Traccia attiva colorata
                        Circle()
                            .trim(from: 0.15, to: 0.15 + (0.70 * progressRatio))
                            .stroke(
                                AngularGradient(
                                    gradient: Gradient(colors: gradientColors),
                                    center: .center,
                                    startAngle: .degrees(135),
                                    endAngle: .degrees(405)
                                ),
                                style: StrokeStyle(lineWidth: 18, lineCap: .round)
                            )
                            .rotationEffect(.degrees(90))
                            .frame(width: 230, height: 230)

                        // Contenuto centrale: Valori termici
                        VStack(spacing: 4) {
                            if entity.isHeating {
                                Image(systemName: "flame.fill")
                                    .font(.title2)
                                    .foregroundColor(.orange)
                            } else if entity.isCooling {
                                Image(systemName: "snowflake")
                                    .font(.title2)
                                    .foregroundColor(.cyan)
                            } else {
                                Image(systemName: "thermometer.medium")
                                    .font(.title2)
                                    .foregroundColor(.secondary)
                            }

                            Text(String(format: "%.1f°", targetTemperature))
                                .font(.system(size: 48, weight: .heavy, design: .rounded))

                            Text("TARGET")
                                .font(.caption2.bold())
                                .foregroundColor(.secondary)

                            HStack(spacing: 4) {
                                Text("Rilevata:")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                                Text(String(format: "%.1f°C", entity.currentTemperature ?? 20.4))
                                    .font(.caption.bold())
                            }
                            .padding(.top, 4)
                        }
                    }
                    .frame(width: 250, height: 250)
                    .contentShape(Circle())
                    .gesture(
                        DragGesture()
                            .onChanged { value in
                                updateTemperatureFromDrag(location: value.location, in: CGSize(width: 250, height: 250))
                            }
                            .onEnded { _ in
                                Haptics.medium()
                                onTemperatureChange(entity.id, targetTemperature)
                                DomusAccessibility.announce("Temperatura bersaglio impostata a \(String(format: "%.1f", targetTemperature)) gradi")
                            }
                    )
                    .accessibilityElement(children: .ignore)
                    .accessibilityLabel("Temperatura bersaglio termostato")
                    .accessibilityValue("\(String(format: "%.1f", targetTemperature)) gradi centigradi\(entity.currentTemperature != nil ? ", rilevata \(String(format: "%.1f", entity.currentTemperature!)) gradi centigradi" : "")")
                    .accessibilityAdjustableAction { direction in
                        switch direction {
                        case .increment:
                            adjustTemperature(by: +0.5)
                        case .decrement:
                            adjustTemperature(by: -0.5)
                        @unknown default:
                            break
                        }
                    }

                    // Tasti regolazione fine +/- 0.5 °C
                    HStack(spacing: 28) {
                        Button(action: {
                            adjustTemperature(by: -0.5)
                        }) {
                            Image(systemName: "minus")
                                .font(.title2.bold())
                                .frame(width: 52, height: 52)
                                .background(Color.secondary.opacity(0.12))
                                .foregroundColor(.primary)
                                .clipShape(Circle())
                        }
                        .accessibilityLabel("Diminuisci temperatura bersaglio")
                        .accessibilityHint("Riduce la temperatura di mezzo grado centigrado")

                        VStack(spacing: 2) {
                            Text("Passo 0.5°C")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                            Text("\(minTemp, specifier: "%.0f")° - \(maxTemp, specifier: "%.0f")°")
                                .font(.caption.bold())
                        }
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Intervallo temperatura da \(Int(minTemp)) a \(Int(maxTemp)) gradi con passo di 0.5 gradi")

                        Button(action: {
                            adjustTemperature(by: +0.5)
                        }) {
                            Image(systemName: "plus")
                                .font(.title2.bold())
                                .frame(width: 52, height: 52)
                                .background(Color.secondary.opacity(0.12))
                                .foregroundColor(.primary)
                                .clipShape(Circle())
                        }
                        .accessibilityLabel("Aumenta temperatura bersaglio")
                        .accessibilityHint("Aumenta la temperatura di mezzo grado centigrado")
                    }

                    // Selettore Modalità HVAC
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Modalità Termostato")
                            .font(.headline)
                            .padding(.horizontal)

                        HStack(spacing: 8) {
                            hvacButton(title: "Spento", mode: "off", icon: "power")
                            hvacButton(title: "Riscalda", mode: "heat", icon: "flame.fill")
                            hvacButton(title: "Raffresca", mode: "cool", icon: "snowflake")
                            hvacButton(title: "Auto", mode: "auto", icon: "arrow.triangle.2.circlepath")
                        }
                        .padding(.horizontal)
                    }
                    .padding(.top, 8)
                    .padding(.bottom, 24)
                }
            }
            .navigationTitle(entity.name)
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
        }
    }

    // MARK: - Helpers

    private var progressRatio: Double {
        let clamped = min(max(targetTemperature, minTemp), maxTemp)
        return (clamped - minTemp) / (maxTemp - minTemp)
    }

    private var gradientColors: [Color] {
        if entity.hvacMode == "cool" {
            return [.cyan, .blue]
        }
        return [.yellow, .orange, .red]
    }

    private var statusColor: Color {
        if entity.isHeating { return .orange }
        if entity.isCooling { return .cyan }
        if entity.hvacMode == "off" { return .secondary }
        return .green
    }

    private var statusDescription: String {
        if entity.isHeating { return "RISCALDAMENTO ATTIVO" }
        if entity.isCooling { return "RAFFRESCAMENTO ATTIVO" }
        if entity.hvacMode == "off" { return "TERMOSTATO SPENTO" }
        return "IN TEMPERATURA"
    }

    private func adjustTemperature(by delta: Double) {
        let newTemp = min(max(targetTemperature + delta, minTemp), maxTemp)
        let rounded = (newTemp * 2).rounded() / 2
        if rounded != targetTemperature {
            Haptics.light()
            targetTemperature = rounded
            onTemperatureChange(entity.id, rounded)
            DomusAccessibility.announce("\(String(format: "%.1f", rounded)) gradi")
        }
    }

    private func updateTemperatureFromDrag(location: CGPoint, in size: CGSize) {
        let center = CGPoint(x: size.width / 2, y: size.height / 2)
        let dx = location.x - center.x
        let dy = location.y - center.y

        var angle = atan2(dy, dx) * 180 / .pi
        if angle < 0 { angle += 360 }

        // Mappa l'arco (da 135° a 405°)
        var mappedAngle = angle - 135
        if mappedAngle < 0 { mappedAngle += 360 }

        if mappedAngle <= 270 {
            let ratio = mappedAngle / 270.0
            let rawTemp = minTemp + (ratio * (maxTemp - minTemp))
            let rounded = (rawTemp * 2).rounded() / 2
            if rounded != targetTemperature {
                Haptics.light()
                targetTemperature = rounded
            }
        }
    }

    private func hvacButton(title: String, mode: String, icon: String) -> some View {
        let isSelected = entity.hvacMode == mode
        return Button(action: {
            Haptics.selection()
            onModeChange(entity.id, mode)
            DomusAccessibility.announce("Modalità impostata su \(title)")
        }) {
            VStack(spacing: 6) {
                Image(systemName: icon)
                    .font(.system(size: 18))
                Text(title)
                    .font(.caption2.weight(.semibold))
            }
            .frame(maxWidth: .infinity, minHeight: 44)
            .padding(.vertical, 10)
            .background(isSelected ? Color.blue : (reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.12)))
            .foregroundColor(isSelected ? .white : .primary)
            .clipShape(RoundedRectangle(cornerRadius: 12))
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(isSelected ? Color.white : (differentiateWithoutColor ? Color.primary.opacity(0.3) : Color.clear), lineWidth: 1.5)
            )
        }
        .accessibilityLabel("Modalità \(title)")
        .accessibilityAddTraits(isSelected ? [.isSelected, .isButton] : .isButton)
    }
}
