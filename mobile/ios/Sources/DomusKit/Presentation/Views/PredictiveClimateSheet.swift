import SwiftUI

/// Scheda Termostato Predittivo & Clima Multi-Zona — Algoritmi predittivi basati su inerzia termica, fronte freddo barometrico e irraggiamento solare.
public struct PredictiveClimateSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.accessibilityDifferentiateWithoutColor) private var differentiateWithoutColor

    @State private var status: SystemPredictiveStatus? = nil
    @State private var isLoading: Bool = true
    @State private var evaluatingZoneId: String? = nil
    @State private var errorMessage: String? = nil
    @State private var preheatRecs: [String: PreheatRecommendation] = [:]
    @State private var coastdownRecs: [String: CoastDownRecommendation] = [:]

    private let client = DomusHttpClient.shared

    public init() {}

    public var body: some View {
        NavigationStack {
            ZStack {
                Color.domusSystemGroupedBackground.ignoresSafeArea()

                if isLoading && status == nil {
                    ProgressView(NSLocalizedString("common_loading", comment: "Caricamento..."))
                        .progressViewStyle(.circular)
                } else if let status {
                    ScrollView {
                        VStack(spacing: 20) {
                            // Banner Allerta Fronte Freddo (se attivo)
                            if status.coldFrontActive {
                                coldFrontBannerView
                            }

                            // Banner Cutoff Solare (se attivo)
                            if status.solarCutoffActive {
                                solarCutoffBannerView
                            }

                            // Cruscotto Statistiche Predittive
                            summaryHeader(status)

                            // Elenco Zone Termiche Predittive
                            VStack(alignment: .leading, spacing: 14) {
                                HStack {
                                    Text(NSLocalizedString("climate_zones_header", comment: "Zone Termiche"))
                                        .font(.headline)
                                    Spacer()
                                    Button(action: {
                                        Task { await evaluateAllZones() }
                                    }) {
                                        HStack(spacing: 4) {
                                            Image(systemName: "arrow.triangle.2.circlepath")
                                                .font(.caption)
                                            Text(NSLocalizedString("climate_evaluate_all", comment: "Ricalcola Tutte"))
                                                .font(.caption.bold())
                                        }
                                        .padding(.horizontal, 10)
                                        .padding(.vertical, 6)
                                        .background(Color.blue.opacity(0.15))
                                        .foregroundColor(.blue)
                                        .clipShape(Capsule())
                                    }
                                }
                                .padding(.horizontal)

                                if status.zones.isEmpty {
                                    emptyZonesView
                                } else {
                                    ForEach(status.zones) { zoneStatus in
                                        zoneCard(zoneStatus)
                                    }
                                    .padding(.horizontal)
                                }
                            }
                        }
                        .padding(.vertical)
                    }
                    .refreshable {
                        await loadStatus()
                    }
                } else {
                    VStack(spacing: 12) {
                        Image(systemName: "thermometer.slash")
                            .font(.system(size: 40))
                            .foregroundColor(.secondary)
                        Text(errorMessage ?? NSLocalizedString("climate_error_loading", comment: "Impossibile caricare il controllo climatico predittivo"))
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                            .multilineTextAlignment(.center)
                            .padding(.horizontal)
                        Button(NSLocalizedString("common_retry", comment: "Riprova")) {
                            Task { await loadStatus() }
                        }
                    }
                    .padding()
                }
            }
            .navigationTitle(NSLocalizedString("predictive_climate_title", comment: "Termostato Predittivo"))
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(action: { dismiss() }) {
                        Text(NSLocalizedString("common_close", comment: "Chiudi"))
                            .font(.body.weight(.medium))
                    }
                }
            }
            .task {
                await loadStatus()
            }
        }
    }

    // MARK: - Banner Eventi Meteo

    private var coldFrontBannerView: some View {
        HStack(spacing: 12) {
            Image(systemName: "wind.snow")
                .font(.title2)
                .foregroundColor(.cyan)

            VStack(alignment: .leading, spacing: 2) {
                Text(NSLocalizedString("climate_cold_front_alert_title", comment: "Fronte Freddo Rilevato"))
                    .font(.subheadline.bold())
                    .foregroundColor(.primary)
                Text(NSLocalizedString("climate_cold_front_alert_desc", comment: "Calo barometrico rapido. Il sistema sta anticipando l'accensione per contrastare l'inerzia."))
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            Spacer()
        }
        .padding()
        .background(Color.cyan.opacity(0.12))
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .overlay(
            RoundedRectangle(cornerRadius: 16)
                .stroke(Color.cyan.opacity(0.3), lineWidth: 1)
        )
        .padding(.horizontal)
        .accessibilityElement(children: .combine)
    }

    private var solarCutoffBannerView: some View {
        HStack(spacing: 12) {
            Image(systemName: "sun.max.fill")
                .font(.title2)
                .foregroundColor(.orange)

            VStack(alignment: .leading, spacing: 2) {
                Text(NSLocalizedString("climate_solar_cutoff_alert_title", comment: "Cutoff Solare Attivo"))
                    .font(.subheadline.bold())
                    .foregroundColor(.primary)
                Text(NSLocalizedString("climate_solar_cutoff_alert_desc", comment: "Irraggiamento solare elevato rilevato nelle stanze a Sud/Ovest. Setpoint ridotti per evitare surriscaldamento."))
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            Spacer()
        }
        .padding()
        .background(Color.orange.opacity(0.12))
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .overlay(
            RoundedRectangle(cornerRadius: 16)
                .stroke(Color.orange.opacity(0.3), lineWidth: 1)
        )
        .padding(.horizontal)
        .accessibilityElement(children: .combine)
    }

    // MARK: - Sommario

    private func summaryHeader(_ status: SystemPredictiveStatus) -> some View {
        HStack(spacing: 12) {
            summaryMetricCard(
                icon: "square.grid.2x2.fill",
                title: NSLocalizedString("climate_metric_zones", comment: "Zone Attive"),
                value: "\(status.enabledZones)/\(status.totalZones)",
                color: .blue
            )

            summaryMetricCard(
                icon: "bolt.shield.fill",
                title: NSLocalizedString("climate_metric_cold_front", comment: "Fronte Freddo"),
                value: status.coldFrontActive ? NSLocalizedString("common_active", comment: "Attivo") : NSLocalizedString("common_nominal", comment: "Nominale"),
                color: status.coldFrontActive ? .cyan : .green
            )

            summaryMetricCard(
                icon: "sun.dust.fill",
                title: NSLocalizedString("climate_metric_solar_cutoff", comment: "Cutoff Solare"),
                value: status.solarCutoffActive ? NSLocalizedString("common_active", comment: "Attivo") : NSLocalizedString("common_nominal", comment: "Nominale"),
                color: status.solarCutoffActive ? .orange : .green
            )
        }
        .padding(.horizontal)
    }

    private func summaryMetricCard(icon: String, title: String, value: String, color: Color) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Image(systemName: icon)
                .font(.subheadline)
                .foregroundColor(color)
            Text(value)
                .font(.title3.bold())
                .foregroundColor(.primary)
            Text(title)
                .font(.caption2)
                .foregroundColor(.secondary)
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(12)
        .background(Color.secondary.opacity(0.08))
        .clipShape(RoundedRectangle(cornerRadius: 14))
        .overlay(
            RoundedRectangle(cornerRadius: 14)
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)
        )
    }

    // MARK: - Card Singola Zona

    private func zoneCard(_ item: ZoneDashboardStatus) -> some View {
        let eval = item.latestEvaluation
        let mode = eval?.parsedMode ?? .comfort

        return VStack(alignment: .leading, spacing: 14) {
            // Header: Nome Zona, Orientamento e Modalità
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text(item.config.name)
                        .font(.headline)
                        .foregroundColor(.primary)
                    HStack(spacing: 6) {
                        Text(item.config.orientation.localizedName)
                            .font(.caption2.bold())
                            .foregroundColor(.secondary)
                        Text("•")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                        Text(item.config.emitterType.localizedName)
                            .font(.caption2)
                            .foregroundColor(.secondary)
                        Text("•")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                        Text("\(String(format: "%.1f", item.config.thermalInertiaHours))h inerzia")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }
                }

                Spacer()

                // Badge Modalità
                HStack(spacing: 4) {
                    Image(systemName: mode.iconName)
                        .font(.caption2)
                    Text(mode.localizedName)
                        .font(.caption.bold())
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 5)
                .background(modeColor(mode).opacity(0.18))
                .foregroundColor(modeColor(mode))
                .clipShape(Capsule())
            }

            Divider()

            // Griglia Temperature: Attuale, Target e Setpoint Calcolato
            HStack(spacing: 12) {
                tempColumn(
                    label: NSLocalizedString("climate_temp_current", comment: "Attuale"),
                    value: eval != nil ? String(format: "%.1f°C", eval!.currentTemp) : "--",
                    color: .primary
                )

                Image(systemName: "arrow.right")
                    .font(.caption)
                    .foregroundColor(.secondary)

                tempColumn(
                    label: NSLocalizedString("climate_temp_target", comment: "Obiettivo"),
                    value: String(format: "%.1f°C", item.config.targetTemperature),
                    color: .blue
                )

                Image(systemName: "arrow.right")
                    .font(.caption)
                    .foregroundColor(.secondary)

                tempColumn(
                    label: NSLocalizedString("climate_temp_calculated", comment: "Setpoint Predittivo"),
                    value: eval != nil ? String(format: "%.1f°C", eval!.calculatedSetpoint) : "--",
                    color: eval != nil && eval!.calculatedSetpoint != item.config.targetTemperature ? .orange : .green
                )
            }

            // Sensori Ambientali (Pressione, Umidità, Guadagno Solare)
            if let eval {
                HStack(spacing: 16) {
                    HStack(spacing: 4) {
                        Image(systemName: "gauge.with.needle")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                        Text(String(format: "%.1f hPa", eval.currentPressure))
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }

                    HStack(spacing: 4) {
                        Image(systemName: "humidity")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                        Text(String(format: "%.0f%%", eval.humidity))
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }

                    if eval.estimatedSolarGainKw > 0 {
                        HStack(spacing: 4) {
                            Image(systemName: "sun.max")
                                .font(.caption2)
                                .foregroundColor(.orange)
                            Text(String(format: "+%.2f kW", eval.estimatedSolarGainKw))
                                .font(.caption.bold())
                                .foregroundColor(.orange)
                        }
                    }

                    if eval.windowOpen {
                        HStack(spacing: 4) {
                            Image(systemName: "window.casement.open")
                                .font(.caption2)
                                .foregroundColor(.red)
                            Text(NSLocalizedString("climate_window_open", comment: "Finestra Aperta"))
                                .font(.caption.bold())
                                .foregroundColor(.red)
                        }
                    }
                }

                if !eval.reason.isEmpty {
                    Text(eval.reason)
                        .font(.footnote)
                        .foregroundColor(.secondary)
                        .italic()
                        .padding(.top, 2)
                }
            }

            // Raccomandazioni Pre-Heat / Coast-Down se presenti
            if let preheat = preheatRecs[item.config.id] {
                HStack(spacing: 8) {
                    Image(systemName: "flame.fill")
                        .foregroundColor(.orange)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(NSLocalizedString("climate_preheat_title", comment: "Pre-Riscaldamento Ottimale"))
                            .font(.caption.bold())
                        Text("Inizio previsto: \(preheat.startTime) (\(preheat.preheatMinutes) min)")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }
                }
                .padding(8)
                .background(Color.orange.opacity(0.1))
                .clipShape(RoundedRectangle(cornerRadius: 10))
            }

            if let coastdown = coastdownRecs[item.config.id] {
                HStack(spacing: 8) {
                    Image(systemName: "leaf.fill")
                        .foregroundColor(.green)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(NSLocalizedString("climate_coastdown_title", comment: "Spegnimento Anticipato (Coast-Down)"))
                            .font(.caption.bold())
                        Text("Stop riscaldamento: \(coastdown.stopTime) • Risparmio: \(String(format: "%.2f kWh", coastdown.energySavedKwh))")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }
                }
                .padding(8)
                .background(Color.green.opacity(0.1))
                .clipShape(RoundedRectangle(cornerRadius: 10))
            }

            // Azione: Valuta Ora
            HStack {
                Spacer()
                Button(action: {
                    Task { await evaluateSingleZone(id: item.config.id) }
                }) {
                    HStack(spacing: 6) {
                        if evaluatingZoneId == item.config.id {
                            ProgressView()
                                .scaleEffect(0.8)
                        } else {
                            Image(systemName: "waveform.path.ecg")
                                .font(.caption)
                        }
                        Text(NSLocalizedString("climate_action_evaluate", comment: "Valuta Ora"))
                            .font(.caption.bold())
                    }
                    .padding(.horizontal, 14)
                    .padding(.vertical, 8)
                    .background(Color.secondary.opacity(0.15))
                    .foregroundColor(.primary)
                    .clipShape(Capsule())
                }
                .disabled(evaluatingZoneId != nil)
            }
        }
        .padding()
        .background(Color.secondary.opacity(0.08))
        .clipShape(RoundedRectangle(cornerRadius: 18))
        .overlay(
            RoundedRectangle(cornerRadius: 18)
                .stroke(Color.secondary.opacity(0.15), lineWidth: 1)
        )
    }

    private func tempColumn(label: String, value: String, color: Color) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.subheadline.bold())
                .foregroundColor(color)
            Text(label)
                .font(.caption2)
                .foregroundColor(.secondary)
        }
        .frame(maxWidth: .infinity)
    }

    private func modeColor(_ mode: PredictiveMode) -> Color {
        switch mode {
        case .comfort: return .blue
        case .coldFrontBoost: return .cyan
        case .solarCutoff: return .orange
        case .preheat: return .red
        case .eco: return .green
        case .frostProtection: return .indigo
        }
    }

    private var emptyZonesView: some View {
        VStack(spacing: 8) {
            Text(NSLocalizedString("climate_empty_zones", comment: "Nessuna zona termica configurata"))
                .font(.headline)
            Text(NSLocalizedString("climate_empty_zones_subtitle", comment: "Aggiungi le tue valvole TRV e sensori di temperatura dal pannello web."))
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
        }
        .padding(.vertical, 40)
    }

    // MARK: - Chiamate di Rete

    private func loadStatus() async {
        isLoading = status == nil
        errorMessage = nil
        do {
            status = try await client.fetchPredictiveStatus()

            // Carica eventuali raccomandazioni per le zone
            if let zones = status?.zones {
                for z in zones {
                    let zoneId = z.config.id
                    Task {
                        if let optStart = try? await client.fetchOptimalStart(id: zoneId) {
                            await MainActor.run { preheatRecs[zoneId] = optStart }
                        }
                        if let optStop = try? await client.fetchOptimalStop(id: zoneId) {
                            await MainActor.run { coastdownRecs[zoneId] = optStop }
                        }
                    }
                }
            }
        } catch {
            errorMessage = error.localizedDescription
        }
        isLoading = false
    }

    private func evaluateSingleZone(id: String) async {
        Haptics.selection()
        evaluatingZoneId = id
        do {
            let eval = try await client.evaluatePredictiveZone(id: id)
            if let current = status {
                let updatedZones = current.zones.map { zs -> ZoneDashboardStatus in
                    if zs.config.id == id {
                        return ZoneDashboardStatus(config: zs.config, latestEvaluation: eval)
                    }
                    return zs
                }
                status = SystemPredictiveStatus(
                    totalZones: current.totalZones,
                    enabledZones: current.enabledZones,
                    coldFrontActive: current.coldFrontActive,
                    solarCutoffActive: current.solarCutoffActive,
                    zones: updatedZones,
                    evaluatedAt: eval.evaluatedAt
                )
            }
            Haptics.success()
        } catch {
            errorMessage = error.localizedDescription
        }
        evaluatingZoneId = nil
    }

    private func evaluateAllZones() async {
        Haptics.light()
        isLoading = true
        do {
            _ = try await client.evaluateAllPredictiveZones()
            await loadStatus()
            Haptics.success()
        } catch {
            errorMessage = error.localizedDescription
        }
        isLoading = false
    }
}
