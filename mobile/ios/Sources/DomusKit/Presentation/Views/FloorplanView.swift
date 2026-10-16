import SwiftUI

/// Modello geometrico per una stanza nella planimetria 2D.
public struct FloorplanRoom: Identifiable, Sendable {
    public var id: String
    public var name: String
    public var rect: CGRect         // Coordinate relative normalizzate (0.0 ... 1.0)
    public var lightEntityId: String?
    public var climateEntityId: String?
    public var motionEntityId: String?

    public init(
        id: String,
        name: String,
        rect: CGRect,
        lightEntityId: String? = nil,
        climateEntityId: String? = nil,
        motionEntityId: String? = nil
    ) {
        self.id = id
        self.name = name
        self.rect = rect
        self.lightEntityId = lightEntityId
        self.climateEntityId = climateEntityId
        self.motionEntityId = motionEntityId
    }
}

/// Vista interattiva Pianta 2D / Floorplan con illuminazione reattiva e nodi interattivi.
public struct FloorplanView: View {
    public let entities: [Entity]
    public var onToggleEntity: (String) -> Void
    public var onSelectEntityForDetail: (Entity) -> Void

    @State private var scale: CGFloat = 1.0
    @State private var offset: CGSize = .zero
    @State private var selectedRoomId: String? = nil

    private let rooms: [FloorplanRoom] = [
        FloorplanRoom(
            id: "living_room",
            name: "Salotto",
            rect: CGRect(x: 0.05, y: 0.05, width: 0.55, height: 0.48),
            lightEntityId: "light.living_room",
            climateEntityId: "climate.living_room",
            motionEntityId: "binary_sensor.motion_living_room"
        ),
        FloorplanRoom(
            id: "kitchen",
            name: "Cucina",
            rect: CGRect(x: 0.63, y: 0.05, width: 0.32, height: 0.48),
            lightEntityId: "light.kitchen",
            climateEntityId: nil,
            motionEntityId: nil
        ),
        FloorplanRoom(
            id: "bedroom",
            name: "Camera da Letto",
            rect: CGRect(x: 0.05, y: 0.56, width: 0.45, height: 0.38),
            lightEntityId: "light.bedroom",
            climateEntityId: "climate.bedroom",
            motionEntityId: nil
        ),
        FloorplanRoom(
            id: "bathroom",
            name: "Bagno",
            rect: CGRect(x: 0.53, y: 0.56, width: 0.42, height: 0.38),
            lightEntityId: "light.bathroom",
            climateEntityId: nil,
            motionEntityId: nil
        )
    ]

    public init(
        entities: [Entity],
        onToggleEntity: @escaping (String) -> Void,
        onSelectEntityForDetail: @escaping (Entity) -> Void
    ) {
        self.entities = entities
        self.onToggleEntity = onToggleEntity
        self.onSelectEntityForDetail = onSelectEntityForDetail
    }

    public var body: some View {
        VStack(spacing: 12) {
            // Header e indicatore stanza selezionata
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text("Planimetria Interattiva 2D")
                        .font(.headline)
                    Text("Tocca una stanza o un'icona per interagire")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
                Spacer()

                Button(action: {
                    withAnimation(.spring()) {
                        scale = 1.0
                        offset = .zero
                        selectedRoomId = nil
                    }
                }) {
                    Label("Ripristina", systemImage: "arrow.counterclockwise")
                        .font(.caption.bold())
                        .padding(.horizontal, 10)
                        .padding(.vertical, 6)
                        .background(Color.secondary.opacity(0.12))
                        .clipShape(Capsule())
                }
            }
            .padding(.horizontal)

            // Canvas della Pianta
            GeometryReader { geo in
                let w = geo.size.width
                let h = geo.size.height

                ZStack {
                    // Sfondo perimetro casa
                    RoundedRectangle(cornerRadius: 18)
                        .stroke(Color.primary.opacity(0.25), lineWidth: 3)
                        .background(
                            RoundedRectangle(cornerRadius: 18)
                                .fill(Color.domusSystemBackground)
                        )
                        .shadow(color: Color.black.opacity(0.08), radius: 8, x: 0, y: 4)

                    // Stanze disegnate
                    ForEach(rooms) { room in
                        roomShapeView(room: room, totalWidth: w, totalHeight: h)
                    }

                    // Nodi interattivi (Luci, Clima, Movimento)
                    ForEach(rooms) { room in
                        roomDevicesOverlay(room: room, totalWidth: w, totalHeight: h)
                    }
                }
                .scaleEffect(scale)
                .offset(offset)
                .gesture(
                    SimultaneousGesture(
                        MagnificationGesture()
                            .onChanged { val in
                                scale = max(0.8, min(val, 2.5))
                            },
                        DragGesture()
                            .onChanged { val in
                                offset = val.translation
                            }
                    )
                )
            }
            .frame(height: 380)
            .clipShape(RoundedRectangle(cornerRadius: 20))
            .padding(.horizontal)

            // Legenda di stato
            HStack(spacing: 16) {
                legendItem(title: "Luce Accesa", color: .yellow, icon: "lightbulb.fill")
                legendItem(title: "Clima Attivo", color: .orange, icon: "thermometer.medium")
                legendItem(title: "Nessun Movimento", color: .green, icon: "checkmark.circle.fill")
            }
            .font(.caption2)
            .padding(.vertical, 6)
        }
    }

    // MARK: - Subviews

    private func roomShapeView(room: FloorplanRoom, totalWidth: CGFloat, totalHeight: CGFloat) -> some View {
        let isLightOn = isRoomLightOn(room)
        let isSelected = selectedRoomId == room.id

        let rx = room.rect.origin.x * totalWidth
        let ry = room.rect.origin.y * totalHeight
        let rw = room.rect.size.width * totalWidth
        let rh = room.rect.size.height * totalHeight

        return ZStack(alignment: .topLeading) {
            RoundedRectangle(cornerRadius: 12)
                .fill(isLightOn ? Color.yellow.opacity(0.18) : Color.secondary.opacity(0.06))
                .overlay(
                    RoundedRectangle(cornerRadius: 12)
                        .stroke(
                            isSelected ? Color.blue : (isLightOn ? Color.yellow.opacity(0.8) : Color.primary.opacity(0.15)),
                            lineWidth: isSelected ? 2.5 : 1.5
                        )
                )
                .frame(width: rw, height: rh)

            Text(room.name)
                .font(.system(size: 11, weight: .bold))
                .foregroundColor(isSelected ? .blue : .primary)
                .padding(8)
        }
        .position(x: rx + rw / 2, y: ry + rh / 2)
        .onTapGesture {
            Haptics.selection()
            withAnimation(.spring(response: 0.3, dampingFraction: 0.7)) {
                selectedRoomId = (selectedRoomId == room.id) ? nil : room.id
            }
        }
    }

    private func roomDevicesOverlay(room: FloorplanRoom, totalWidth: CGFloat, totalHeight: CGFloat) -> some View {
        let rx = room.rect.origin.x * totalWidth
        let ry = room.rect.origin.y * totalHeight
        let rw = room.rect.size.width * totalWidth
        let rh = room.rect.size.height * totalHeight

        return HStack(spacing: 8) {
            // Nodo Luce
            if let light = findRoomLight(room) {
                Button(action: {
                    Haptics.medium()
                    onToggleEntity(light.id)
                }) {
                    ZStack {
                        Circle()
                            .fill(light.isOn ? Color.yellow : Color.secondary.opacity(0.2))
                            .frame(width: 30, height: 30)
                        Image(systemName: "lightbulb.fill")
                            .font(.system(size: 13, weight: .bold))
                            .foregroundColor(light.isOn ? .black : .secondary)
                    }
                }
            }

            // Nodo Clima
            if let climate = findRoomClimate(room) {
                Button(action: {
                    Haptics.light()
                    onSelectEntityForDetail(climate)
                }) {
                    ZStack {
                        Circle()
                            .fill(Color.orange.opacity(0.2))
                            .frame(width: 30, height: 30)
                        VStack(spacing: 0) {
                            Image(systemName: "thermometer.medium")
                                .font(.system(size: 10, weight: .bold))
                            if let temp = climate.targetTemperature ?? climate.currentTemperature {
                                Text("\(Int(temp))°")
                                    .font(.system(size: 9, weight: .heavy))
                            }
                        }
                        .foregroundColor(.orange)
                    }
                }
            }
        }
        .position(x: rx + rw / 2, y: ry + rh / 2 + 10)
    }

    private func findRoomLight(_ room: FloorplanRoom) -> Entity? {
        if let lightId = room.lightEntityId, let light = entities.first(where: { $0.id == lightId }) {
            return light
        }
        return entities.first { ent in
            ent.domain == "light" && (ent.areaId == room.id || ent.id.lowercased().contains(room.id) || ent.name.lowercased().contains(room.name.lowercased()))
        }
    }

    private func findRoomClimate(_ room: FloorplanRoom) -> Entity? {
        if let climateId = room.climateEntityId, let climate = entities.first(where: { $0.id == climateId }) {
            return climate
        }
        return entities.first { ent in
            ent.domain == "climate" && (ent.areaId == room.id || ent.id.lowercased().contains(room.id) || ent.name.lowercased().contains(room.name.lowercased()))
        }
    }

    private func isRoomLightOn(_ room: FloorplanRoom) -> Bool {
        return findRoomLight(room)?.isOn ?? false
    }

    private func legendItem(title: String, color: Color, icon: String) -> some View {
        HStack(spacing: 4) {
            Image(systemName: icon)
                .foregroundColor(color)
            Text(title)
                .foregroundColor(.secondary)
        }
    }
}
