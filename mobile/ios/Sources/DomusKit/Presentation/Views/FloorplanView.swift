import SwiftUI

/// Modello geometrico per una stanza nella planimetria 2D (mantenuto per calcoli geometrici e test unitari).
public struct FloorplanRoom: Identifiable, Sendable {
    public var id: String
    public var name: String
    public var rect: CGRect // Coordinate relative normalizzate (0.0 ... 1.0)
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

/// Vista interattiva Pianta 2D / Floorplan alimentata al 100% dal server Domus.
/// Supporta caricamento piani e marker da `/api/v1/floors`, modalità di modifica con drag-and-drop
/// per aggiustare e impostare la planimetria reale, e interazione live con i dispositivi.
public struct FloorplanView: View {
    public let entities: [Entity]
    public var onToggleEntity: (String) -> Void
    public var onSelectEntityForDetail: (Entity) -> Void

    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Environment(\.accessibilityDifferentiateWithoutColor) private var differentiateWithoutColor
    @Environment(\.accessibilityReduceTransparency) private var reduceTransparency

    @State private var floors: [FloorDTO] = []
    @State private var selectedFloorId: String = ""
    @State private var markers: [FloorMarkerDTO] = []
    @State private var isEditMode: Bool = false
    @State private var isLoading: Bool = false
    @State private var showAddFloorSheet: Bool = false
    @State private var showAddDeviceSheet: Bool = false
    @State private var showAccessibleListView: Bool = false
    @State private var scale: CGFloat = 1.0
    @State private var offset: CGSize = .zero
    @State private var dragOffsets: [String: CGSize] = [:]
    @State private var newFloorName: String = ""
    @State private var newFloorLevel: Int = 0
    @State private var statusFeedback: String? = nil

    public init(
        entities: [Entity],
        onToggleEntity: @escaping (String) -> Void,
        onSelectEntityForDetail: @escaping (Entity) -> Void
    ) {
        self.entities = entities
        self.onToggleEntity = onToggleEntity
        self.onSelectEntityForDetail = onSelectEntityForDetail
    }

    private var currentFloor: FloorDTO? {
        floors.first(where: { $0.id == selectedFloorId }) ?? floors.first
    }

    public var body: some View {
        VStack(spacing: 12) {
            // Barra Superiore: Selettore Piano & Toggle Modifica
            headerControlBar

            // Selettore Piani (se multipli o per crearne di nuovi)
            floorSelectorBar

            if isEditMode {
                editModeBanner
            }

            // Canvas Interattivo della Planimetria o Elenco Accessibile
            if showAccessibleListView {
                accessibleDeviceListView
            } else {
                floorplanCanvas
            }

            // Barra delle Azioni in Modalità Modifica
            if isEditMode {
                HStack(spacing: 12) {
                    Button(action: { showAddDeviceSheet = true }) {
                        Label("Aggiungi Dispositivo", systemImage: "plus.circle.fill")
                            .font(.subheadline.bold())
                            .padding(.horizontal, 14)
                            .padding(.vertical, 8)
                            .background(Color.blue)
                            .foregroundColor(.white)
                            .clipShape(Capsule())
                    }
                    .accessibilityLabel("Aggiungi un dispositivo su questo piano")

                    Spacer()

                    if let feedback = statusFeedback {
                        Text(feedback)
                            .font(.caption2.bold())
                            .foregroundColor(.secondary)
                    }
                }
                .padding(.horizontal)
            } else {
                // Legenda Dispositivi
                legendBar
            }
        }
        .task {
            await loadFloorsAndMarkers()
        }
        .sheet(isPresented: $showAddFloorSheet) {
            addFloorSheetView
        }
        .sheet(isPresented: $showAddDeviceSheet) {
            addDeviceSheetView
        }
    }

    // MARK: - Header Controls

    private var headerControlBar: some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text(currentFloor?.name ?? "Planimetria Casa")
                    .font(.headline)
                    .foregroundColor(.primary)
                Text(isEditMode ? "Trascina i nodi per posizionarli sulla pianta" : "Tocca un dispositivo per interagire")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            Spacer()

            // Pulsante Modalità Elenco Accessibile
            Button(action: {
                Haptics.light()
                withAnimation(reduceMotion ? .none : .spring()) {
                    showAccessibleListView.toggle()
                }
                DomusAccessibility.announce(showAccessibleListView ? "Visualizzazione elenco dispositivi attiva" : "Visualizzazione planimetria grafica attiva")
            }) {
                Image(systemName: showAccessibleListView ? "map" : "list.bullet")
                    .font(.caption.bold())
                    .frame(width: 44, height: 44)
                    .background(showAccessibleListView ? Color.blue.opacity(0.15) : Color.secondary.opacity(0.12))
                    .foregroundColor(showAccessibleListView ? .blue : .primary)
                    .clipShape(Circle())
            }
            .accessibilityLabel(showAccessibleListView ? "Mostra planimetria grafica" : "Mostra elenco dispositivi accessibile")
            .accessibilityHint("Passa a un elenco accessibile con tutti i dispositivi presenti su questo piano")

            // Pulsante Reset Zoom/Pan
            Button(action: {
                withAnimation(reduceMotion ? .none : .spring()) {
                    scale = 1.0
                    offset = .zero
                }
            }) {
                Image(systemName: "arrow.counterclockwise")
                    .font(.caption.bold())
                    .frame(width: 44, height: 44)
                    .background(Color.secondary.opacity(0.12))
                    .clipShape(Circle())
            }
            .accessibilityLabel("Ripristina zoom e posizione")

            // Toggle Modalità Modifica / Aggiusta Planimetria
            Button(action: {
                Haptics.medium()
                withAnimation(reduceMotion ? .none : .spring(response: 0.35, dampingFraction: 0.8)) {
                    isEditMode.toggle()
                }
            }) {
                HStack(spacing: 4) {
                    Image(systemName: isEditMode ? "checkmark.circle.fill" : "slider.horizontal.3")
                    Text(isEditMode ? "Fine" : "Aggiusta")
                }
                .font(.caption.bold())
                .frame(minHeight: 44)
                .padding(.horizontal, 12)
                .padding(.vertical, 7)
                .background(isEditMode ? Color.green : Color.blue.opacity(0.15))
                .foregroundColor(isEditMode ? .white : .blue)
                .clipShape(Capsule())
            }
            .accessibilityLabel(isEditMode ? "Termina aggiustamento planimetria" : "Aggiusta e imposta planimetria")
        }
        .padding(.horizontal)
    }

    private var floorSelectorBar: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(floors) { floor in
                    let isSelected = (floor.id == selectedFloorId) || (selectedFloorId.isEmpty && floor.id == floors.first?.id)
                    Button(action: {
                        Haptics.selection()
                        selectedFloorId = floor.id
                        Task { await loadMarkersForCurrentFloor() }
                        DomusAccessibility.announce("Selezionato piano \(floor.name)")
                    }) {
                        HStack(spacing: 4) {
                            Image(systemName: "square.2.layers.3d")
                                .font(.caption2)
                            Text(floor.name)
                                .font(.caption.bold())
                        }
                        .frame(minHeight: 44)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 6)
                        .background(isSelected ? Color.blue : (reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.12)))
                        .foregroundColor(isSelected ? .white : .primary)
                        .clipShape(Capsule())
                        .overlay(
                            Capsule()
                                .stroke(differentiateWithoutColor && isSelected ? Color.white : Color.clear, lineWidth: 1.5)
                        )
                    }
                    .accessibilityLabel("Piano: \(floor.name)")
                    .accessibilityAddTraits(isSelected ? [.isSelected, .isButton] : .isButton)
                }

                Button(action: {
                    newFloorName = ""
                    newFloorLevel = floors.count
                    showAddFloorSheet = true
                }) {
                    HStack(spacing: 4) {
                        Image(systemName: "plus")
                        Text("Nuovo Piano")
                    }
                    .font(.caption.bold())
                    .frame(minHeight: 44)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6)
                    .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.12))
                    .foregroundColor(.blue)
                    .clipShape(Capsule())
                }
                .accessibilityLabel("Crea un nuovo piano")
                .accessibilityAddTraits(.isButton)
            }
            .padding(.horizontal)
        }
    }

    private var editModeBanner: some View {
        HStack(spacing: 8) {
            Image(systemName: "hand.draw.fill")
                .foregroundColor(.orange)
            Text("Trascina qualsiasi dispositivo per regolarne la posizione. I cambiamenti vengono salvati automaticamente sul server.")
                .font(.caption)
                .foregroundColor(.primary)
            Spacer()
        }
        .padding(10)
        .background(Color.orange.opacity(0.12))
        .clipShape(RoundedRectangle(cornerRadius: 10))
        .padding(.horizontal)
    }

    // MARK: - Canvas

    private var floorplanCanvas: some View {
        GeometryReader { geo in
            let w = geo.size.width
            let h = geo.size.height

            ZStack {
                // Sfondo Grigio Architettonico con Griglia
                blueprintBackground(width: w, height: h)

                // Render dei Marker dal Server
                ForEach(markers) { marker in
                    markerNodeView(marker: marker, totalWidth: w, totalHeight: h)
                }

                if markers.isEmpty && !isLoading {
                    VStack(spacing: 8) {
                        Image(systemName: "map")
                            .font(.system(size: 32))
                            .foregroundColor(.secondary)
                        Text("Nessun dispositivo posizionato su questo piano")
                            .font(.caption)
                            .foregroundColor(.secondary)
                        Button("Aggiungi Dispositivo") {
                            showAddDeviceSheet = true
                        }
                        .font(.caption.bold())
                        .foregroundColor(.blue)
                    }
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
                            if !isEditMode {
                                offset = val.translation
                            }
                        }
                )
            )
        }
        .frame(height: 380)
        .clipShape(RoundedRectangle(cornerRadius: 20))
        .padding(.horizontal)
    }

    private func blueprintBackground(width: CGFloat, height: CGFloat) -> some View {
        ZStack {
            RoundedRectangle(cornerRadius: 18)
                .fill(Color.domusSystemBackground)
                .overlay(
                    RoundedRectangle(cornerRadius: 18)
                        .stroke(Color.primary.opacity(isEditMode ? 0.35 : 0.15), lineWidth: 2)
                )
                .shadow(color: Color.black.opacity(0.06), radius: 8, x: 0, y: 4)

            // Griglia millimetrata visibile in modalità modifica
            if isEditMode {
                Canvas { context, size in
                    let step: CGFloat = 30.0
                    var path = Path()
                    for x in stride(from: step, to: size.width, by: step) {
                        path.move(to: CGPoint(x: x, y: 0))
                        path.addLine(to: CGPoint(x: x, y: size.height))
                    }
                    for y in stride(from: step, to: size.height, by: step) {
                        path.move(to: CGPoint(x: 0, y: y))
                        path.addLine(to: CGPoint(x: size.width, y: y))
                    }
                    context.stroke(path, with: .color(Color.blue.opacity(0.08)), lineWidth: 1)
                }
                .clipShape(RoundedRectangle(cornerRadius: 18))
            }
        }
    }

    // MARK: - Marker Node

    private func markerNodeView(marker: FloorMarkerDTO, totalWidth: CGFloat, totalHeight: CGFloat) -> some View {
        let matchingEntity = entities.first(where: { $0.id == marker.entityKey })
        let currentDrag = dragOffsets[marker.id] ?? .zero

        let baseX = (marker.xPercent / 100.0) * totalWidth
        let baseY = (marker.yPercent / 100.0) * totalHeight
        let posX = baseX + currentDrag.width
        let posY = baseY + currentDrag.height

        let isOn = matchingEntity?.isOn ?? false
        let domain = matchingEntity?.domain ?? "switch"

        return ZStack(alignment: .topTrailing) {
            VStack(spacing: 2) {
                ZStack {
                    // Alone luminoso se attivo
                    if isOn {
                        Circle()
                            .fill(markerActiveColor(domain: domain).opacity(0.3))
                            .frame(width: 44, height: 44)
                    }

                    Circle()
                        .fill(isOn ? markerActiveColor(domain: domain) : Color.domusSecondarySystemGroupedBackground)
                        .frame(width: 36, height: 36)
                        .overlay(
                            Circle()
                                .stroke(isEditMode ? Color.orange : Color.primary.opacity(0.2), lineWidth: isEditMode ? 2 : 1)
                        )
                        .shadow(color: isOn ? markerActiveColor(domain: domain).opacity(0.4) : Color.clear, radius: 6)

                    Image(systemName: markerIcon(marker: marker, entity: matchingEntity))
                        .font(.system(size: 15, weight: .bold))
                        .foregroundColor(isOn ? .black : .primary)
                }

                // Nome ed eventuale stato / temperatura
                if marker.showBadge {
                    VStack(spacing: 1) {
                        Text(matchingEntity?.name ?? marker.entityKey)
                            .font(.system(size: 9, weight: .bold))
                            .foregroundColor(.primary)
                            .lineLimit(1)
                            .frame(maxWidth: 80)

                        if let badge = markerBadgeText(matchingEntity) {
                            Text(badge)
                                .font(.system(size: 8, weight: .heavy))
                                .foregroundColor(isOn ? .orange : .secondary)
                        }
                    }
                    .padding(.horizontal, 4)
                    .padding(.vertical, 2)
                    .background(Color.domusSystemBackground.opacity(0.85))
                    .clipShape(RoundedRectangle(cornerRadius: 6))
                }
            }
            .contentShape(Rectangle())
            .onTapGesture {
                if !isEditMode {
                    handleMarkerTap(marker: marker, entity: matchingEntity)
                }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel("\(matchingEntity?.name ?? marker.entityKey), dispositivo \(domain)")
            .accessibilityValue(isOn ? "Attivo / Acceso" : "Spento / Inattivo")
            .accessibilityHint(isEditMode ? "Trascina per riposizionare" : "Tocca due volte per controllare")
            .accessibilityAddTraits(.isButton)
            .accessibilityAction(named: isOn ? "Spegni" : "Accendi") {
                handleMarkerTap(marker: marker, entity: matchingEntity)
            }

            // Pulsante elimina marker in modalità modifica
            if isEditMode {
                Button(action: {
                    Task { await deleteMarker(marker) }
                }) {
                    Image(systemName: "xmark.circle.fill")
                        .font(.system(size: 16))
                        .foregroundColor(.red)
                        .background(Circle().fill(Color.white))
                }
                .offset(x: 10, y: -10)
                .accessibilityLabel("Rimuovi dispositivo dal piano")
            }
        }
        .position(x: posX, y: posY)
        .gesture(
            isEditMode ? DragGesture()
                .onChanged { val in
                    dragOffsets[marker.id] = val.translation
                }
                .onEnded { val in
                    let finalX = baseX + val.translation.width
                    let finalY = baseY + val.translation.height
                    let clampedX = max(10, min(totalWidth - 10, finalX))
                    let clampedY = max(10, min(totalHeight - 10, finalY))
                    let newXPercent = Double(clampedX / totalWidth) * 100.0
                    let newYPercent = Double(clampedY / totalHeight) * 100.0

                    dragOffsets[marker.id] = .zero
                    Task {
                        await saveUpdatedMarkerPosition(marker: marker, xPercent: newXPercent, yPercent: newYPercent)
                    }
                }
            : nil
        )
    }

    // MARK: - Actions & Logic

    private func handleMarkerTap(marker: FloorMarkerDTO, entity: Entity?) {
        guard let entity else {
            Haptics.light()
            onToggleEntity(marker.entityKey)
            return
        }

        switch entity.domain {
        case "light", "switch":
            Haptics.medium()
            onToggleEntity(entity.id)
        default:
            Haptics.light()
            onSelectEntityForDetail(entity)
        }
    }

    private func loadFloorsAndMarkers() async {
        isLoading = true
        defer { isLoading = false }
        do {
            var fetched = try await DomusHttpClient.shared.fetchFloors()
            if fetched.isEmpty {
                // Crea il piano di default se il server non ne ha ancora registrati
                let defaultFloor = try await DomusHttpClient.shared.createFloor(name: "Piano Terra", level: 0)
                fetched = [defaultFloor]
            }
            self.floors = fetched
            if selectedFloorId.isEmpty || !fetched.contains(where: { $0.id == selectedFloorId }) {
                self.selectedFloorId = fetched.first?.id ?? ""
            }
            await loadMarkersForCurrentFloor()
        } catch {
            self.statusFeedback = "Errore di sincronizzazione con il server"
        }
    }

    private func loadMarkersForCurrentFloor() async {
        guard !selectedFloorId.isEmpty else { return }
        do {
            let fetched = try await DomusHttpClient.shared.fetchFloorMarkers(floorId: selectedFloorId)
            self.markers = fetched
        } catch {
            // Ignora o mostra feedback
        }
    }

    private func saveUpdatedMarkerPosition(marker: FloorMarkerDTO, xPercent: Double, yPercent: Double) async {
        guard !selectedFloorId.isEmpty else { return }
        Haptics.selection()
        do {
            let saved = try await DomusHttpClient.shared.saveFloorMarker(
                floorId: selectedFloorId,
                markerId: marker.id,
                entityKey: marker.entityKey,
                xPercent: xPercent,
                yPercent: yPercent,
                iconOverride: marker.iconOverride,
                showBadge: marker.showBadge
            )
            if let idx = markers.firstIndex(where: { $0.id == marker.id }) {
                markers[idx] = saved
            }
            statusFeedback = "Posizione salvata ✓"
        } catch {
            statusFeedback = "Errore salvataggio posizione"
        }
    }

    private func deleteMarker(_ marker: FloorMarkerDTO) async {
        guard !selectedFloorId.isEmpty else { return }
        Haptics.medium()
        do {
            try await DomusHttpClient.shared.deleteFloorMarker(floorId: selectedFloorId, markerId: marker.id)
            markers.removeAll(where: { $0.id == marker.id })
            statusFeedback = "Dispositivo rimosso ✓"
        } catch {
            statusFeedback = "Errore cancellazione"
        }
    }

    private func markerIcon(marker: FloorMarkerDTO, entity: Entity?) -> String {
        if let icon = marker.iconOverride, !icon.isEmpty {
            return icon
        }
        guard let entity else { return "power" }
        switch entity.domain {
        case "light": return "lightbulb.fill"
        case "climate": return "thermometer.medium"
        case "switch": return "powerplug.fill"
        case "cover": return "blinds.vertical.closed"
        case "binary_sensor": return "sensor.fill"
        case "camera": return "video.fill"
        case "lock": return entity.isOn ? "lock.open.fill" : "lock.fill"
        default: return "circle.fill"
        }
    }

    private func markerActiveColor(domain: String) -> Color {
        switch domain {
        case "light": return .yellow
        case "climate": return .orange
        case "switch": return .cyan
        case "lock": return .red
        default: return .blue
        }
    }

    private func markerBadgeText(_ entity: Entity?) -> String? {
        guard let entity else { return nil }
        if entity.domain == "climate" {
            if let temp = entity.targetTemperature ?? entity.currentTemperature {
                return String(format: "%.1f°", temp)
            }
        }
        return entity.isOn ? "ON" : "OFF"
    }

    // MARK: - Subviews & Sheets

    private var legendBar: some View {
        HStack(spacing: 16) {
            legendItem(title: "Luce", color: .yellow, icon: "lightbulb.fill")
            legendItem(title: "Clima", color: .orange, icon: "thermometer.medium")
            legendItem(title: "Presa", color: .cyan, icon: "powerplug.fill")
        }
        .font(.caption2)
        .padding(.vertical, 4)
    }

    private func legendItem(title: String, color: Color, icon: String) -> some View {
        HStack(spacing: 4) {
            Image(systemName: icon)
                .foregroundColor(color)
            Text(title)
                .foregroundColor(.secondary)
        }
    }

    private var addFloorSheetView: some View {
        NavigationStack {
            Form {
                Section(header: Text("Dati Nuovo Piano")) {
                    TextField("Nome Piano (es. Piano Primo)", text: $newFloorName)
                    Stepper("Livello: \(newFloorLevel)", value: $newFloorLevel, in: -3...10)
                }
            }
            .navigationTitle("Aggiungi Piano")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annulla") { showAddFloorSheet = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Crea") {
                        Task {
                            guard !newFloorName.trimmingCharacters(in: .whitespaces).isEmpty else { return }
                            if let created = try? await DomusHttpClient.shared.createFloor(name: newFloorName, level: newFloorLevel) {
                                floors.append(created)
                                selectedFloorId = created.id
                                await loadMarkersForCurrentFloor()
                            }
                            showAddFloorSheet = false
                        }
                    }
                    .disabled(newFloorName.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
        }
    }

    private var addDeviceSheetView: some View {
        NavigationStack {
            List {
                Section(header: Text("Tocca un dispositivo per posizionarlo al centro della pianta")) {
                    ForEach(entities) { entity in
                        let isAlreadyPlaced = markers.contains(where: { $0.entityKey == entity.id })
                        Button(action: {
                            Task {
                                guard !selectedFloorId.isEmpty else { return }
                                if let saved = try? await DomusHttpClient.shared.saveFloorMarker(
                                    floorId: selectedFloorId,
                                    entityKey: entity.id,
                                    xPercent: 50.0,
                                    yPercent: 50.0,
                                    showBadge: true
                                ) {
                                    markers.append(saved)
                                    statusFeedback = "Dispositivo posizionato al centro ✓"
                                }
                                showAddDeviceSheet = false
                            }
                        }) {
                            HStack {
                                Image(systemName: markerIcon(marker: FloorMarkerDTO(id: "", entityKey: entity.id, xPercent: 50, yPercent: 50), entity: entity))
                                    .foregroundColor(.blue)
                                    .frame(width: 24)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(entity.name)
                                        .font(.subheadline.bold())
                                        .foregroundColor(.primary)
                                    Text(entity.id)
                                        .font(.caption2)
                                        .foregroundColor(.secondary)
                                }
                                Spacer()
                                if isAlreadyPlaced {
                                    Text("Posizionato")
                                        .font(.caption2)
                                        .foregroundColor(.secondary)
                                }
                            }
                        }
                    }
                }
            }
            .navigationTitle("Posiziona Dispositivo")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Chiudi") { showAddDeviceSheet = false }
                }
            }
        }
    }

    private var accessibleDeviceListView: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Dispositivi su \(currentFloor?.name ?? "questo piano") (\(markers.count))")
                .font(.subheadline.bold())
                .padding(.horizontal)

            if markers.isEmpty {
                Text("Nessun dispositivo posizionato su questo piano.")
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .padding(.horizontal)
            } else {
                ForEach(markers) { marker in
                    let matchingEntity = entities.first(where: { $0.id == marker.entityKey })
                    let isOn = matchingEntity?.isOn ?? false
                    HStack {
                        Image(systemName: markerIcon(marker: marker, entity: matchingEntity))
                            .font(.headline)
                            .foregroundColor(isOn ? markerActiveColor(domain: matchingEntity?.domain ?? "switch") : .secondary)
                            .frame(width: 44, height: 44)
                            .background(Color.secondary.opacity(0.12))
                            .clipShape(Circle())

                        VStack(alignment: .leading, spacing: 2) {
                            Text(matchingEntity?.name ?? marker.entityKey)
                                .font(.subheadline.bold())
                            Text(isOn ? "Acceso / Attivo" : "Spento / Inattivo")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }

                        Spacer()

                        Button(action: {
                            handleMarkerTap(marker: marker, entity: matchingEntity)
                        }) {
                            Text(isOn ? "Spegni" : "Accendi")
                                .font(.caption.bold())
                                .padding(.horizontal, 14)
                                .padding(.vertical, 8)
                                .background(isOn ? Color.orange.opacity(0.15) : Color.blue.opacity(0.15))
                                .foregroundColor(isOn ? .orange : .blue)
                                .clipShape(Capsule())
                        }
                        .frame(minHeight: 44)
                    }
                    .padding(12)
                    .background(Color.domusSecondarySystemGroupedBackground)
                    .clipShape(RoundedRectangle(cornerRadius: 12))
                    .padding(.horizontal)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("\(matchingEntity?.name ?? marker.entityKey), stato: \(isOn ? "Acceso" : "Spento")")
                    .accessibilityHint("Tocca due volte per controllare il dispositivo")
                    .accessibilityAction(named: isOn ? "Spegni" : "Accendi") {
                        handleMarkerTap(marker: marker, entity: matchingEntity)
                    }
                }
            }
        }
        .padding(.vertical, 8)
    }
}
