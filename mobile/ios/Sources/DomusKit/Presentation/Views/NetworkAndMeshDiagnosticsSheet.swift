import SwiftUI

/// Modello di un nodo wireless o dispositivo di rete nella diagnostica dell'hub.
public struct MeshNodeDiagnostic: Identifiable, Sendable {
    public let id: String
    public let name: String
    public let room: String
    public let protocolType: String // "Zigbee 3.0", "Thread / Matter", "Z-Wave Plus", "BLE Proxy"
    public let lqi: Int // Link Quality Index 0-255
    public let rssi: Int // dBm (es. -52)
    public let batteryLevel: Int? // % (nil se alimentato a rete fissa)
    public let isRouter: Bool // true se router/ripetitore, false se end-device a batteria

    public init(
        id: String,
        name: String,
        room: String,
        protocolType: String,
        lqi: Int,
        rssi: Int,
        batteryLevel: Int? = nil,
        isRouter: Bool = false
    ) {
        self.id = id
        self.name = name
        self.room = room
        self.protocolType = protocolType
        self.lqi = lqi
        self.rssi = rssi
        self.batteryLevel = batteryLevel
        self.isRouter = isRouter
    }

    public var signalQuality: String {
        if rssi >= -60 { return "Eccellente" }
        if rssi >= -75 { return "Buono" }
        if rssi >= -85 { return "Accettabile" }
        return "Critico"
    }

    public var signalColor: Color {
        if rssi >= -60 { return .green }
        if rssi >= -75 { return .cyan }
        if rssi >= -85 { return .orange }
        return .red
    }

    public static let sampleNodes: [MeshNodeDiagnostic] = [
        MeshNodeDiagnostic(id: "zigbee.pir_corridor", name: "Sensore PIR Corridoio", room: "Corridoio", protocolType: "Zigbee 3.0", lqi: 235, rssi: -58, batteryLevel: 12, isRouter: false),
        MeshNodeDiagnostic(id: "zigbee.temp_bedroom", name: "Termo-Igrometro Letto", room: "Camera Padronale", protocolType: "Zigbee 3.0", lqi: 190, rssi: -68, batteryLevel: 14, isRouter: false),
        MeshNodeDiagnostic(id: "thread.door_sensor", name: "Sensore Finestra Studio", room: "Studio", protocolType: "Thread / Matter", lqi: 248, rssi: -52, batteryLevel: 88, isRouter: false),
        MeshNodeDiagnostic(id: "zwave.water_leak", name: "Allagamento Lavanderia", room: "Lavanderia", protocolType: "Z-Wave Plus", lqi: 180, rssi: -72, batteryLevel: 92, isRouter: false),
        MeshNodeDiagnostic(id: "ble.beacon_living", name: "BLE Proxy ESP32 Salotto", room: "Salotto", protocolType: "BLE Proxy", lqi: 255, rssi: -45, batteryLevel: nil, isRouter: true),
        MeshNodeDiagnostic(id: "zigbee.bulb_kitchen", name: "Luce Cucina (Router)", room: "Cucina", protocolType: "Zigbee 3.0", lqi: 250, rssi: -48, batteryLevel: nil, isRouter: true)
    ]
}

/// Vista "Stato dell'Hub e della Rete" per qualità del segnale RSSI/LQI, latenza e salute batterie wireless.
public struct NetworkAndMeshDiagnosticsSheet: View {
    @Environment(\.dismiss) private var dismiss

    @State private var nodes: [MeshNodeDiagnostic] = MeshNodeDiagnostic.sampleNodes
    @State private var selectedTab: Int = 0 // 0: Segnale & Mesh, 1: Batterie (<15%), 2: Latenza & Hub
    @State private var localLatencyMs: Int = 12
    @State private var remoteLatencyMs: Int = 64
    @State private var selectedProtocolFilter: String = "Tutti"

    private let protocolOptions = ["Tutti", "Zigbee 3.0", "Thread / Matter", "Z-Wave Plus", "BLE Proxy"]

    public init() {}

    private var lowBatteryNodes: [MeshNodeDiagnostic] {
        nodes.filter { ($0.batteryLevel ?? 100) < 15 }
    }

    private var allBatteryNodes: [MeshNodeDiagnostic] {
        nodes.filter { $0.batteryLevel != nil }
            .sorted(by: { ($0.batteryLevel ?? 0) < ($1.batteryLevel ?? 0) })
    }

    private var filteredMeshNodes: [MeshNodeDiagnostic] {
        if selectedProtocolFilter == "Tutti" {
            return nodes
        }
        return nodes.filter { $0.protocolType == selectedProtocolFilter }
    }

    public var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Selettore Sezione Superiore
                Picker("Sezione Diagnostica", selection: $selectedTab) {
                    Text("Mesh & RSSI").tag(0)
                    Text("Batterie (\(lowBatteryNodes.count) critiche)").tag(1)
                    Text("Latenza & Hub").tag(2)
                }
                .pickerStyle(.segmented)
                .padding(.horizontal)
                .padding(.vertical, 10)

                // Banner Avviso Batterie Scariche
                if !lowBatteryNodes.isEmpty && selectedTab != 1 {
                    HStack(spacing: 8) {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .foregroundColor(.orange)
                        Text("\(lowBatteryNodes.count) sensori con batteria residua sotto al 15%!")
                            .font(.caption.bold())
                            .foregroundColor(.orange)
                        Spacer()
                        Button(action: {
                            selectedTab = 1
                        }) {
                            Text("Vedi")
                                .font(.caption.bold())
                                .foregroundColor(.orange)
                                .underline()
                        }
                    }
                    .padding(.horizontal, 14)
                    .padding(.vertical, 8)
                    .background(Color.orange.opacity(0.12))
                    .padding(.horizontal)
                    .padding(.bottom, 6)
                }

                // Contenuto Selezionato
                Group {
                    switch selectedTab {
                    case 0:
                        meshSignalView
                    case 1:
                        batteryHealthView
                    default:
                        hubAndLatencyView
                    }
                }
            }
            .navigationTitle("Diagnostica Mesh & Hardware")
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
                    Button(action: {
                        refreshDiagnostics()
                    }) {
                        Image(systemName: "arrow.clockwise")
                    }
                    .accessibilityLabel("Aggiorna diagnostica")
                }
            }
        }
    }

    // MARK: - 1. Vista Segnale Mesh & RSSI

    private var meshSignalView: some View {
        List {
            Section {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        ForEach(protocolOptions, id: \.self) { proto in
                            Button(action: {
                                Haptics.selection()
                                selectedProtocolFilter = proto
                            }) {
                                Text(proto)
                                    .font(.caption.bold())
                                    .padding(.horizontal, 12)
                                    .padding(.vertical, 6)
                                    .background(selectedProtocolFilter == proto ? Color.blue : Color.secondary.opacity(0.12))
                                    .foregroundColor(selectedProtocolFilter == proto ? .white : .primary)
                                    .clipShape(Capsule())
                            }
                        }
                    }
                }
                .listRowInsets(EdgeInsets(top: 8, leading: 16, bottom: 8, trailing: 16))
            }

            Section(header: Text("Nodi della Rete (\(filteredMeshNodes.count))")) {
                ForEach(filteredMeshNodes) { node in
                    HStack(spacing: 12) {
                        Image(systemName: node.isRouter ? "point.3.filled.connected.trianglepath.dotted" : "sensor.fill")
                            .foregroundColor(node.signalColor)
                            .font(.title3)
                            .frame(width: 32)

                        VStack(alignment: .leading, spacing: 3) {
                            Text(node.name)
                                .font(.subheadline.bold())
                            HStack(spacing: 6) {
                                Text(node.room)
                                    .font(.caption2)
                                    .foregroundColor(.secondary)
                                Text("•")
                                    .foregroundColor(.secondary)
                                Text(node.protocolType)
                                    .font(.caption2.bold())
                                    .foregroundColor(.blue)
                            }
                        }

                        Spacer()

                        VStack(alignment: .trailing, spacing: 2) {
                            HStack(spacing: 4) {
                                Image(systemName: "cellularbars")
                                    .font(.caption2)
                                Text("\(node.rssi) dBm")
                                    .font(.caption.monospacedDigit().bold())
                            }
                            .foregroundColor(node.signalColor)

                            Text("LQI: \(node.lqi)")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }
                    }
                    .padding(.vertical, 2)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(node.name), segnale \(node.signalQuality), \(node.rssi) decibel milliwatt, LQI \(node.lqi)")
                }
            }
        }
        #if os(iOS)
        .listStyle(.insetGrouped)
        #else
        .listStyle(.inset)
        #endif
    }

    // MARK: - 2. Vista Salute Batterie (<15% Alert)

    private var batteryHealthView: some View {
        List {
            Section(header: Text("Sensori a Batteria (\(allBatteryNodes.count))")) {
                ForEach(allBatteryNodes) { node in
                    let level = node.batteryLevel ?? 0
                    let isCritical = level < 15

                    HStack(spacing: 12) {
                        Image(systemName: isCritical ? "battery.0" : (level < 50 ? "battery.50" : "battery.100"))
                            .foregroundColor(isCritical ? .red : (level < 50 ? .orange : .green))
                            .font(.title3)
                            .frame(width: 32)

                        VStack(alignment: .leading, spacing: 3) {
                            HStack(spacing: 6) {
                                Text(node.name)
                                    .font(.subheadline.bold())
                                if isCritical {
                                    Text("SOSTITUIRE")
                                        .font(.system(size: 9, weight: .heavy))
                                        .padding(.horizontal, 5)
                                        .padding(.vertical, 1)
                                        .background(Color.red)
                                        .foregroundColor(.white)
                                        .clipShape(Capsule())
                                }
                            }
                            Text("\(node.room) • \(node.protocolType)")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }

                        Spacer()

                        Text("\(level)%")
                            .font(.subheadline.monospacedDigit().bold())
                            .foregroundColor(isCritical ? .red : (level < 50 ? .orange : .primary))
                    }
                    .padding(.vertical, 2)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(node.name), batteria residua al \(level) percento\(isCritical ? ", allarme batteria scarica" : "")")
                }
            }
        }
        #if os(iOS)
        .listStyle(.insetGrouped)
        #else
        .listStyle(.inset)
        #endif
    }

    // MARK: - 3. Vista Latenza & Stato Hub

    private var hubAndLatencyView: some View {
        List {
            Section(header: Text("Latenza di Rete & Trasporto")) {
                HStack {
                    Label("LAN Locale (mDNS)", systemImage: "bolt.horizontal.fill")
                        .foregroundColor(.green)
                    Spacer()
                    Text("\(localLatencyMs) ms (Attiva)")
                        .font(.subheadline.bold())
                        .foregroundColor(.green)
                }

                HStack {
                    Label("Fallback Remoto (TLS / WireGuard)", systemImage: "globe")
                        .foregroundColor(.blue)
                    Spacer()
                    Text("\(remoteLatencyMs) ms (Standby)")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
            }

            Section(header: Text("Parametri Appliance Domus OS")) {
                HStack {
                    Text("Uptime Sistema")
                    Spacer()
                    Text("14 giorni, 6 ore")
                        .font(.subheadline.bold())
                }

                HStack {
                    Text("Carico CPU Hub")
                    Spacer()
                    Text("8.4%")
                        .font(.subheadline.bold())
                }

                HStack {
                    Text("Memoria RAM")
                    Spacer()
                    Text("1.2 GB / 4.0 GB (30%)")
                        .font(.subheadline.bold())
                }

                HStack {
                    Text("Database SQLite WAL")
                    Spacer()
                    Text("Integrità OK • 42 MB")
                        .font(.subheadline.bold())
                        .foregroundColor(.green)
                }
            }
        }
        #if os(iOS)
        .listStyle(.insetGrouped)
        #else
        .listStyle(.inset)
        #endif
    }

    private func refreshDiagnostics() {
        Haptics.medium()
        localLatencyMs = Int.random(in: 9...15)
        remoteLatencyMs = Int.random(in: 60...75)
        DomusAccessibility.announce("Diagnostica di rete aggiornata")
    }
}
