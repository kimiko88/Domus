import SwiftUI

/// Scheda di controllo avanzato per Luci — Fase 2: Luminosità fluida, temperatura colore e colori RGB/HSL.
public struct LightDetailSheet: View {
    public let entity: Entity
    public var onToggle: (String) -> Void
    public var onBrightnessChange: (String, Int) -> Void
    public var onColorChange: (String, Double, Double, Double) -> Void
    public var onColorTempChange: (String, Int) -> Void

    @Environment(\.accessibilityDifferentiateWithoutColor) private var differentiateWithoutColor
    @Environment(\.accessibilityReduceTransparency) private var reduceTransparency
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

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
                            .blur(radius: (entity.isOn && !reduceMotion) ? 18 : 0)

                        Circle()
                            .fill(entity.isOn ? selectedColor.opacity(0.2) : Color.gray.opacity(0.1))
                            .frame(width: 90, height: 90)

                        Image(systemName: entity.isOn ? "lightbulb.fill" : "lightbulb")
                            .font(.system(size: 44))
                            .foregroundColor(entity.isOn ? selectedColor : .secondary)
                    }
                    .accessibilityHidden(true)
                    .padding(.top, 16)

                    // Nome e stato
                    VStack(spacing: 4) {
                        Text(entity.name)
                            .font(.title2.bold())
                        Text(entity.isOn ? "Accesa • \(Int(brightness))%" : "Spenta")
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                    }
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(entity.name): \(entity.isOn ? "Accesa al \(Int(brightness)) percento" : "Spenta")")

                    // Tasto On/Off principale
                    Button(action: {
                        Haptics.medium()
                        onToggle(entity.id)
                        DomusAccessibility.announce(entity.isOn ? "Luce spenta" : "Luce accesa")
                    }) {
                        HStack {
                            Image(systemName: "power")
                            Text(entity.isOn ? "Spegni" : "Accendi")
                        }
                        .font(.headline)
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .padding(.vertical, 14)
                        .background(entity.isOn ? Color.red.opacity(reduceTransparency ? 0.3 : 0.15) : Color.cyan.opacity(reduceTransparency ? 0.3 : 0.15))
                        .foregroundColor(entity.isOn ? .red : .cyan)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                        .overlay(
                            RoundedRectangle(cornerRadius: 14)
                                .stroke(differentiateWithoutColor ? (entity.isOn ? Color.red : Color.cyan) : Color.clear, lineWidth: 1.5)
                        )
                    }
                    .accessibilityLabel(entity.isOn ? "Spegni luce" : "Accendi luce")
                    .accessibilityAddTraits(.isButton)
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
                                    DomusAccessibility.announce("Luminosità impostata al \(Int(brightness)) percento")
                                }
                            }
                        )
                        .tint(selectedColor)
                        .accessibilityLabel("Luminosità")
                        .accessibilityValue("\(Int(brightness)) percento")
                    }
                    .padding(16)
                    .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.1))
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
                                    DomusAccessibility.announce("Colore impostato su \(preset.name)")
                                }) {
                                    Circle()
                                        .fill(preset.color)
                                        .frame(width: 44, height: 44)
                                        .overlay(
                                            Circle()
                                                .stroke(Color.white, lineWidth: selectedColor == preset.color ? 3 : 0)
                                        )
                                        .overlay(
                                            Circle()
                                                .stroke(Color.primary.opacity(0.3), lineWidth: (differentiateWithoutColor && selectedColor != preset.color) ? 1 : 0)
                                        )
                                        .shadow(color: reduceMotion ? .clear : preset.color.opacity(0.4), radius: 6, x: 0, y: 3)
                                }
                                .accessibilityLabel("Colore \(preset.name)")
                                .accessibilityValue(selectedColor == preset.color ? "Selezionato" : "Non selezionato")
                                .accessibilityAddTraits(selectedColor == preset.color ? [.isSelected, .isButton] : .isButton)
                            }
                        }
                    }
                    .padding(16)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.1))
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
                                    DomusAccessibility.announce("Temperatura luce impostata su \(preset.name), \(preset.kelvin)")
                                }) {
                                    VStack(spacing: 4) {
                                        Text(preset.name)
                                            .font(.caption2.weight(.semibold))
                                        Text(preset.kelvin)
                                            .font(.caption2)
                                            .foregroundColor(.secondary)
                                    }
                                    .frame(maxWidth: .infinity, minHeight: 44)
                                    .padding(.vertical, 10)
                                    .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.15))
                                    .foregroundColor(.primary)
                                    .clipShape(RoundedRectangle(cornerRadius: 12))
                                }
                                .accessibilityLabel("Temperatura luce \(preset.name), \(preset.kelvin)")
                                .accessibilityAddTraits(.isButton)
                            }
                        }
                    }
                    .padding(16)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.1))
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
