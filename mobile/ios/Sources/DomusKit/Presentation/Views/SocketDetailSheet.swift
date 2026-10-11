import SwiftUI
import Charts

/// Scheda di controllo per Prese e Interruttori — Fase 2: Potenza istantanea (W), energia (kWh) e grafico consumi.
public struct SocketDetailSheet: View {
    public let entity: Entity
    public var onToggle: (String) -> Void

    private struct PowerDataPoint: Identifiable {
        let id = UUID()
        let hour: String
        let watts: Double
    }

    private var hourlyData: [PowerDataPoint] {
        let base = entity.isOn ? (entity.powerWatts ?? 65.0) : 0.0
        return [
            PowerDataPoint(hour: "02:00", watts: max(0, base * 0.2)),
            PowerDataPoint(hour: "06:00", watts: max(0, base * 0.1)),
            PowerDataPoint(hour: "10:00", watts: max(0, base * 0.8)),
            PowerDataPoint(hour: "14:00", watts: max(0, base * 0.95)),
            PowerDataPoint(hour: "18:00", watts: max(0, base * 1.1)),
            PowerDataPoint(hour: "Adesso", watts: base)
        ]
    }

    public init(entity: Entity, onToggle: @escaping (String) -> Void) {
        self.entity = entity
        self.onToggle = onToggle
    }

    public var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 24) {
                    // Header con icona e glow
                    ZStack {
                        Circle()
                            .fill(entity.isOn ? Color.green.opacity(0.2) : Color.gray.opacity(0.15))
                            .frame(width: 110, height: 110)
                            .blur(radius: entity.isOn ? 16 : 0)

                        Circle()
                            .fill(entity.isOn ? Color.green.opacity(0.15) : Color.gray.opacity(0.1))
                            .frame(width: 84, height: 84)

                        Image(systemName: "powerplug.fill")
                            .font(.system(size: 40))
                            .foregroundColor(entity.isOn ? .green : .secondary)
                    }
                    .padding(.top, 16)

                    // Titolo e stato
                    VStack(spacing: 4) {
                        Text(entity.name)
                            .font(.title2.bold())
                        Text(entity.isOn ? "Presa Attiva" : "Presa Spenta")
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                    }

                    // Tasto On/Off
                    Button(action: {
                        Haptics.medium()
                        onToggle(entity.id)
                    }) {
                        HStack {
                            Image(systemName: "power")
                            Text(entity.isOn ? "Disattiva Presa" : "Attiva Presa")
                        }
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(entity.isOn ? Color.red.opacity(0.15) : Color.green.opacity(0.15))
                        .foregroundColor(entity.isOn ? .red : .green)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                    }
                    .padding(.horizontal)

                    // Card Metriche Potenza & Energia
                    HStack(spacing: 12) {
                        metricCard(
                            title: "Potenza Istantanea",
                            value: String(format: "%.1f", entity.isOn ? (entity.powerWatts ?? 65.0) : 0.0),
                            unit: "W",
                            icon: "bolt.fill",
                            color: .yellow
                        )

                        metricCard(
                            title: "Energia Oggi",
                            value: String(format: "%.2f", entity.energyKWh ?? 1.42),
                            unit: "kWh",
                            icon: "leaf.fill",
                            color: .green
                        )
                    }
                    .padding(.horizontal)

                    // Grafico Consumi delle Ultime Ore
                    VStack(alignment: .leading, spacing: 12) {
                        HStack {
                            Label("Andamento Carico (W)", systemImage: "chart.xyaxis.line")
                                .font(.headline)
                            Spacer()
                            Text("Ultime 6h")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }

                        Chart(hourlyData) { point in
                            AreaMark(
                                x: .value("Ora", point.hour),
                                y: .value("Watt", point.watts)
                            )
                            .foregroundStyle(
                                LinearGradient(
                                    colors: [Color.green.opacity(0.4), Color.green.opacity(0.02)],
                                    startPoint: .top,
                                    endPoint: .bottom
                                )
                            )

                            LineMark(
                                x: .value("Ora", point.hour),
                                y: .value("Watt", point.watts)
                            )
                            .foregroundStyle(Color.green)
                            .lineStyle(StrokeStyle(lineWidth: 2.5))

                            PointMark(
                                x: .value("Ora", point.hour),
                                y: .value("Watt", point.watts)
                            )
                            .foregroundStyle(Color.green)
                        }
                        .frame(height: 160)
                        .padding(.vertical, 8)
                    }
                    .padding(16)
                    .background(Color.secondary.opacity(0.1))
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                    .padding(.horizontal)

                    // Stima Costo
                    HStack {
                        Image(systemName: "eurosign.circle.fill")
                            .font(.title2)
                            .foregroundColor(.cyan)
                        VStack(alignment: .leading, spacing: 2) {
                            Text("Stima Costo Orario")
                                .font(.caption.bold())
                                .foregroundColor(.secondary)
                            let watts = entity.isOn ? (entity.powerWatts ?? 65.0) : 0.0
                            let costPerHour = (watts / 1000.0) * 0.25 // 0.25 €/kWh
                            Text(String(format: "€ %.3f / ora", costPerHour))
                                .font(.subheadline.bold())
                        }
                        Spacer()
                    }
                    .padding(14)
                    .background(Color.cyan.opacity(0.08))
                    .clipShape(RoundedRectangle(cornerRadius: 14))
                    .padding(.horizontal)
                    .padding(.bottom, 24)
                }
            }
            .navigationTitle("Controllo Presa")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
        }
    }

    private func metricCard(title: String, value: String, unit: String, icon: String, color: Color) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Image(systemName: icon)
                    .foregroundColor(color)
                Spacer()
                Text(unit)
                    .font(.caption2.bold())
                    .foregroundColor(.secondary)
            }
            Text(value)
                .font(.system(.title2, design: .rounded).bold())
            Text(title)
                .font(.caption2)
                .foregroundColor(.secondary)
        }
        .padding(14)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.secondary.opacity(0.1))
        .clipShape(RoundedRectangle(cornerRadius: 14))
    }
}
