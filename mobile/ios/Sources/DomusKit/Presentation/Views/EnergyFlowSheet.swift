import SwiftUI

public struct EnergyFlowSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.accessibilityDifferentiateWithoutColor) private var differentiateWithoutColor
    @Environment(\.accessibilityReduceTransparency) private var reduceTransparency
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    @State private var solarPowerWatts: Double = 0.0
    @State private var gridPowerWatts: Double = 0.0
    @State private var batteryPowerWatts: Double = 0.0
    @State private var batteryLevelPercent: Int = 0
    @State private var homeConsumptionWatts: Double = 0.0
    @State private var loadSheddingLimitKw: Double = 4.5
    @State private var currentTariffEurKwh: Double = 0.185

    public init() {}

    public var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 16) {
                    // Live Power Header Card
                    liveSummaryCard

                    // Canvas Flusso Energetico Interattivo
                    energyFlowCanvas
                        .frame(height: 230)
                        .padding(.horizontal)

                    // Tariffa Attiva & PUN
                    tariffCard

                    // Soglia contrattuale Load Shedding
                    loadSheddingCard

                    // Elenco Carichi Prioritari
                    applianceLoadsCard
                }
                .padding(.vertical)
            }
            .background(Color.domusSystemGroupedBackground.ignoresSafeArea())
            .navigationTitle("Cruscotto Energetico")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Chiudi") {
                        dismiss()
                    }
                    .accessibilityLabel("Chiudi cruscotto energetico")
                }
            }
            .task {
                await loadEnergyFlow()
            }
            .refreshable {
                await loadEnergyFlow()
            }
        }
    }

    private func loadEnergyFlow() async {
        do {
            let flow = try await DomusHttpClient.shared.fetchEnergyFlow()
            self.solarPowerWatts = flow.solarWatts
            self.gridPowerWatts = flow.gridWatts
            self.batteryPowerWatts = flow.batteryWatts
            self.batteryLevelPercent = Int(flow.batterySoc)
            self.homeConsumptionWatts = flow.houseConsumptionWatts
            DomusAccessibility.announce("Dati energetici aggiornati: autoconsumo \(String(format: "%.1f", flow.houseConsumptionWatts / 1000.0)) kilowatt")
        } catch {
            // Rete non raggiungibile
        }
    }

    private var liveSummaryCard: some View {
        HStack {
            VStack(alignment: .leading, spacing: 4) {
                Text("Autoconsumo Attivo")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Text(String(format: "%.1f kW", homeConsumptionWatts / 1000.0))
                    .font(.system(.title, design: .rounded).bold())
                    .foregroundColor(.primary)
            }
            Spacer()
            VStack(alignment: .trailing, spacing: 4) {
                Text("Produzione Solare")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Text(String(format: "%.1f kW", solarPowerWatts / 1000.0))
                    .font(.system(.title2, design: .rounded).bold())
                    .foregroundColor(.yellow)
            }
        }
        .padding(16)
        .background(Color.domusSecondarySystemGroupedBackground)
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .padding(.horizontal)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Autoconsumo attivo: \(String(format: "%.1f", homeConsumptionWatts / 1000.0)) kilowatt. Produzione solare: \(String(format: "%.1f", solarPowerWatts / 1000.0)) kilowatt")
    }

    private var energyFlowCanvas: some View {
        ZStack {
            RoundedRectangle(cornerRadius: 18)
                .fill(Color.domusSecondarySystemGroupedBackground)

            VStack {
                // Top: Solare
                flowNode(title: "Solare", value: String(format: "%.1f kW", solarPowerWatts / 1000.0), icon: "sun.max.fill", color: .yellow)

                Spacer()

                // Middle: Batteria (sx) e Rete (dx)
                HStack {
                    flowNode(title: "Batteria (\(batteryLevelPercent)%)", value: String(format: "%.1f kW", batteryPowerWatts / 1000.0), icon: "battery.75", color: .green)
                    Spacer()
                    flowNode(title: "Rete Nazionale", value: gridPowerWatts < 0 ? String(format: "Immette %.1f kW", abs(gridPowerWatts) / 1000.0) : String(format: "Preleva %.1f kW", gridPowerWatts / 1000.0), icon: "bolt.fill", color: .blue)
                }
                .padding(.horizontal, 24)

                Spacer()

                // Bottom: Casa
                flowNode(title: "Carichi Casa", value: String(format: "%.1f kW", homeConsumptionWatts / 1000.0), icon: "house.fill", color: .cyan)
            }
            .padding(.vertical, 16)
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel("Diagramma di flusso energetico della casa")
    }

    private func flowNode(title: String, value: String, icon: String, color: Color) -> some View {
        VStack(spacing: 4) {
            ZStack {
                Circle()
                    .fill(color.opacity(reduceTransparency ? 0.35 : 0.18))
                    .frame(width: 44, height: 44)
                    .overlay(
                        Circle()
                            .stroke(color, lineWidth: differentiateWithoutColor ? 1.5 : 0)
                    )
                Image(systemName: icon)
                    .foregroundColor(color)
                    .font(.system(size: 20))
            }
            Text(title)
                .font(.caption2.weight(.semibold))
                .foregroundColor(.secondary)
            Text(value)
                .font(.footnote.bold())
                .foregroundColor(.primary)
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title): \(value)")
    }

    private var tariffCard: some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text("Tariffa Dinamica Attuale (Fascia F1)")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Text(String(format: "€ %.3f / kWh", currentTariffEurKwh))
                    .font(.title3)
                    .fontWeight(.bold)
            }
            Spacer()
            Text("Economica")
                .font(.caption2.bold())
                .padding(.horizontal, 10)
                .padding(.vertical, 4)
                .background(Color.green.opacity(reduceTransparency ? 0.3 : 0.15))
                .foregroundColor(.green)
                .clipShape(Capsule())
                .overlay(
                    Capsule()
                        .stroke(differentiateWithoutColor ? Color.green : Color.clear, lineWidth: 1)
                )
        }
        .padding(14)
        .background(Color.domusSecondarySystemGroupedBackground)
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .padding(.horizontal)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Tariffa dinamica attuale: \(String(format: "%.3f", currentTariffEurKwh)) euro per kilowattora, fascia F1 economica")
    }

    private var loadSheddingCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text("Soglia Contrattuale Antidistacco")
                    .font(.headline)
                Spacer()
                Text(String(format: "%.1f kW", loadSheddingLimitKw))
                    .font(.headline)
                    .foregroundColor(.cyan)
            }

            Slider(value: $loadSheddingLimitKw, in: 3.0...6.6, step: 0.1)
                .tint(.cyan)
                .accessibilityLabel("Soglia contrattuale antidistacco")
                .accessibilityValue(String(format: "%.1f kilowatt", loadSheddingLimitKw))

            Text("Se l'assorbimento totale supera la soglia per oltre 2 minuti, Domus spegne automaticamente i carichi a bassa priorità.")
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding(16)
        .background(Color.domusSecondarySystemGroupedBackground)
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .padding(.horizontal)
    }

    private var applianceLoadsCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Priorità Distacco Carichi")
                .font(.headline)
                .padding(.horizontal)

            VStack(spacing: 8) {
                applianceRow(name: "Pompa di Calore HVAC", power: "1.2 kW", priority: "Priorità 1 (Critico)", safe: true)
                applianceRow(name: "Lavatrice Cucina", power: "0.9 kW", priority: "Priorità 2 (Flessibile)", safe: false)
                applianceRow(name: "Wallbox Ricarica Auto", power: "2.3 kW", priority: "Priorità 3 (Sheddabile)", safe: false)
            }
            .padding(.horizontal)
        }
    }

    private func applianceRow(name: String, power: String, priority: String, safe: Bool) -> some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text(name)
                    .font(.subheadline)
                    .fontWeight(.medium)
                HStack(spacing: 4) {
                    if differentiateWithoutColor {
                        Image(systemName: safe ? "checkmark.shield.fill" : "exclamationmark.triangle.fill")
                            .font(.caption2)
                            .foregroundColor(safe ? .green : .orange)
                    }
                    Text(priority)
                        .font(.caption2)
                        .foregroundColor(safe ? .green : .orange)
                }
            }
            Spacer()
            Text(power)
                .font(.subheadline)
                .fontWeight(.bold)
                .foregroundColor(.primary)
        }
        .padding(12)
        .background(Color.domusSecondarySystemGroupedBackground)
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(name), consumo: \(power), \(priority)")
    }
}
