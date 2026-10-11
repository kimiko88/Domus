import SwiftUI

/// Scheda di controllo avanzato per Luci — Fase 2: Luminosità fluida, temperatura colore e colori RGB/HSL.
public struct LightDetailSheet: View {
    public let entity: Entity
    public var onToggle: (String) -> Void
    public var onBrightnessChange: (String, Int) -> Void
    public var onColorChange: (String, Double, Double, Double) -> Void
    public var onColorTempChange: (String, Int) -> Void

    @State private var brightness: Double = 100
    @State private var selectedColor: Color = .cyan

    private let colorPresets: [(name: String, color: Color, rgb: (Double, Double, Double))] = [
        ("Domus Ciano", Color(red: 0.22, green: 0.74, blue: 0.97), (0.22, 0.74, 0.97)),
        ("Caldo Ambra", Color(red: 0.98, green: 0.75, blue: 0.35), (0.98, 0.75, 0.35)),
        ("Viola Neon", Color(red: 0.66, green: 0.33, blue: 0.97), (0.66, 0.33, 0.97)),
        ("Smeraldo", Color(red: 0.06, green: 0.73, blue: 0.51), (0.06, 0.73, 0.51)),
        ("Rosso Corallo", Color(red: 0.96, green: 0.25, blue: 0.37), (0.96, 0.25, 0.37)),
        ("Luce Bianca", Color.white, (1.0, 1.0, 1.0))
    ]

    private let tempPresets: [(name: String, kelvin: String, mireds: Int)] = [
        ("Calda", "2700K", 370),
        ("Naturale", "4000K", 250),
        ("Fredda", "6500K", 153)
    ]

    public init(
        entity: Entity,
        onToggle: @escaping (String) -> Void,
        onBrightnessChange: @escaping (String, Int) -> Void,
        onColorChange: @escaping (String, Double, Double, Double) -> Void,
        onColorTempChange: @escaping (String, Int) -> Void
    ) {
        self.entity = entity
        self.onToggle = onToggle
        self.onBrightnessChange = onBrightnessChange
        self.onColorChange = onColorChange
        self.onColorTempChange = onColorTempChange
        _brightness = State(initialValue: Double(entity.brightness ?? 100))
    }

    public var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 24) {
                    // Header con icona e glow
                    ZStack {
                        Circle()
                            .fill(entity.isOn ? selectedColor.opacity(0.25) : Color.gray.opacity(0.15))
                            .frame(width: 120, height: 120)
                            .blur(radius: entity.isOn ? 18 : 0)

                        Circle()
                            .fill(entity.isOn ? selectedColor.opacity(0.2) : Color.gray.opacity(0.1))
                            .frame(width: 90, height: 90)

                        Image(systemName: entity.isOn ? "lightbulb.fill" : "lightbulb")
                            .font(.system(size: 44))
                            .foregroundColor(entity.isOn ? selectedColor : .secondary)
                    }
                    .padding(.top, 16)

                    // Nome e stato
                    VStack(spacing: 4) {
                        Text(entity.name)
                            .font(.title2.bold())
                        Text(entity.isOn ? "Accesa • \(Int(brightness))%" : "Spenta")
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                    }

                    // Tasto On/Off principale
                    Button(action: {
                        Haptics.medium()
                        onToggle(entity.id)
                    }) {
                        HStack {
                            Image(systemName: "power")
                            Text(entity.isOn ? "Spegni" : "Accendi")
                        }
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(entity.isOn ? Color.red.opacity(0.15) : Color.cyan.opacity(0.15))
                        .foregroundColor(entity.isOn ? .red : .cyan)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                    }
                    .padding(.horizontal)

                    // Slider Luminosità fluida
                    VStack(alignment: .leading, spacing: 10) {
                        HStack {
                            Label("Luminosità", systemImage: "sun.max.fill")
                                .font(.headline)
                            Spacer()
                            Text("\(Int(brightness))%")
                                .font(.system(.subheadline, design: .monospaced).bold())
                                .foregroundColor(.secondary)
                        }

                        Slider(
                            value: $brightness,
                            in: 1...100,
                            step: 1,
                            onEditingChanged: { isEditing in
                                if !isEditing {
                                    Haptics.light()
                                    onBrightnessChange(entity.id, Int(brightness))
                                }
                            }
                        )
                        .tint(selectedColor)
                    }
                    .padding(16)
                    .background(Color.secondary.opacity(0.1))
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                    .padding(.horizontal)

                    // Palette Colori
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Colori Rapidi")
                            .font(.headline)

                        HStack(spacing: 12) {
                            ForEach(colorPresets, id: \.name) { preset in
                                Button(action: {
                                    Haptics.selection()
                                    selectedColor = preset.color
                                    onColorChange(entity.id, preset.rgb.0, preset.rgb.1, preset.rgb.2)
                                }) {
                                    Circle()
                                        .fill(preset.color)
                                        .frame(width: 44, height: 44)
                                        .overlay(
                                            Circle()
                                                .stroke(Color.white, lineWidth: selectedColor == preset.color ? 3 : 0)
                                        )
                                        .shadow(color: preset.color.opacity(0.4), radius: 6, x: 0, y: 3)
                                }
                            }
                        }
                    }
                    .padding(16)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(Color.secondary.opacity(0.1))
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                    .padding(.horizontal)

                    // Temperatura Colore (Kelvin)
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Temperatura Luce")
                            .font(.headline)

                        HStack(spacing: 10) {
                            ForEach(tempPresets, id: \.name) { preset in
                                Button(action: {
                                    Haptics.selection()
                                    onColorTempChange(entity.id, preset.mireds)
                                }) {
                                    VStack(spacing: 4) {
                                        Text(preset.name)
                                            .font(.system(size: 13, weight: .semibold))
                                        Text(preset.kelvin)
                                            .font(.caption2)
                                            .foregroundColor(.secondary)
                                    }
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 10)
                                    .background(Color.secondary.opacity(0.15))
                                    .foregroundColor(.primary)
                                    .clipShape(RoundedRectangle(cornerRadius: 12))
                                }
                            }
                        }
                    }
                    .padding(16)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(Color.secondary.opacity(0.1))
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                    .padding(.horizontal)
                    .padding(.bottom, 24)
                }
            }
            .navigationTitle("Controllo Luce")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
        }
    }
}
