import SwiftUI

/// Router modale che seleziona e visualizza la scheda di controllo nativa (Fase 2) appropriata per il tipo di entità.
public struct EntityDetailRouterView: View {
    public let entity: Entity
    @ObservedObject public var viewModel: DashboardViewModel

    public init(entity: Entity, viewModel: DashboardViewModel) {
        self.entity = entity
        self.viewModel = viewModel
    }

    public var body: some View {
        Group {
            if entity.isLight {
                LightDetailSheet(
                    entity: entity,
                    onToggle: { id in viewModel.toggleEntity(id: id) },
                    onBrightnessChange: { id, val in viewModel.setLightBrightness(id: id, brightness: val) },
                    onColorChange: { id, r, g, b in viewModel.setLightColor(id: id, r: r, g: g, b: b) },
                    onColorTempChange: { id, mireds in viewModel.setLightColorTemp(id: id, mireds: mireds) }
                )
            } else if entity.isSwitch {
                SocketDetailSheet(
                    entity: entity,
                    onToggle: { id in viewModel.toggleEntity(id: id) }
                )
            } else if entity.isClimate {
                ThermostatSheet(
                    entity: entity,
                    onTemperatureChange: { id, temp in viewModel.setThermostatTemperature(id: id, targetTemp: temp) },
                    onModeChange: { id, mode in viewModel.setThermostatHvacMode(id: id, mode: mode) }
                )
            } else if entity.isAlarm {
                AlarmSheet(
                    entity: entity,
                    onArm: { id, code, mode in viewModel.armAlarm(id: id, code: code, mode: mode) },
                    onDisarm: { id, code in viewModel.disarmAlarm(id: id, code: code) }
                )
            } else if entity.isCamera {
                CameraDetailSheet(
                    entity: entity,
                    onRefresh: { id in viewModel.refreshCamera(id: id) }
                )
            } else {
                genericEntityDetailView
            }
        }
        .presentationDetents([.medium, .large])
        .presentationDragIndicator(.visible)
    }

    private var genericEntityDetailView: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(entity.name)
                                .font(.title2.bold())
                            Text(entity.id)
                                .font(.system(.caption, design: .monospaced))
                                .foregroundColor(.secondary)
                        }
                        Spacer()
                        Text(entity.state.uppercased())
                            .font(.subheadline.bold())
                            .padding(.horizontal, 10)
                            .padding(.vertical, 6)
                            .background(Color.secondary.opacity(0.15))
                            .clipShape(Capsule())
                    }
                    .padding()
                    .background(Color.secondary.opacity(0.1))
                    .clipShape(RoundedRectangle(cornerRadius: 16))

                    Text("Attributi")
                        .font(.headline)

                    VStack(spacing: 8) {
                        ForEach(Array(entity.attributes.keys.sorted()), id: \.self) { key in
                            HStack {
                                Text(key)
                                    .font(.subheadline)
                                Spacer()
                                Text(String(describing: entity.attributes[key]?.value ?? ""))
                                    .font(.subheadline.bold())
                                    .foregroundColor(.secondary)
                            }
                            .padding(.vertical, 4)
                            Divider()
                        }
                    }
                    .padding()
                    .background(Color.secondary.opacity(0.1))
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                }
                .padding()
            }
            .navigationTitle("Dettagli Dispositivo")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
        }
    }
}
