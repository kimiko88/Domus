import SwiftUI

/// Vista principale della dashboard — include schede interattive Fase 2, filtro stanze e apertura modale di dettaglio.
public struct DashboardView: View {
    @ObservedObject public var viewModel: DashboardViewModel

    @State private var isFloorplanMode: Bool = false
    @State private var draggedEntityId: String? = nil

    private let columns = [
        GridItem(.flexible(), spacing: 12),
        GridItem(.flexible(), spacing: 12)
    ]

    public init(viewModel: DashboardViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        NavigationStack {
            VStack(spacing: 12) {
                // Indicatore Stanza Rilevata via Beacon / UWB
                if let roomName = viewModel.beaconManager.activeRoomName {
                    HStack(spacing: 6) {
                        Image(systemName: "location.fill")
                            .font(.caption2)
                            .foregroundColor(.cyan)
                        Text("Stanza Rilevata: \(roomName)")
                            .font(.caption.bold())
                            .foregroundColor(.cyan)
                        Spacer()
                        if let meters = viewModel.beaconManager.currentDistanceMeters {
                            Text(String(format: "%.1fm", meters))
                                .font(.caption2.monospacedDigit())
                                .foregroundColor(.secondary)
                        }
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .background(Color.cyan.opacity(0.12))
                    .clipShape(Capsule())
                    .padding(.horizontal)
                }

                // Search Bar
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("Cerca dispositivi o stanze...", text: $viewModel.searchQuery)
                        .textFieldStyle(.plain)
                    if !viewModel.searchQuery.isEmpty {
                        Button(action: { viewModel.searchQuery = "" }) {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundColor(.secondary)
                        }
                    }
                }
                .padding(10)
                .background(Color.secondary.opacity(0.12))
                .clipShape(RoundedRectangle(cornerRadius: 12))
                .padding(.horizontal)

                if isFloorplanMode {
                    // Vista Pianta 2D Interattiva
                    ScrollView {
                        FloorplanView(
                            entities: viewModel.entities,
                            onToggleEntity: { id in
                                viewModel.toggleEntity(id: id)
                            },
                            onSelectEntityForDetail: { entity in
                                viewModel.selectEntityForDetail(entity)
                            }
                        )
                        .padding(.top, 8)
                    }
                } else {
                    // Selettore Stanze / Aree
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: 8) {
                            areaButton(label: "Tutte le Stanze", areaId: nil)
                            ForEach(viewModel.areas) { area in
                                areaButton(label: area.name, areaId: area.id)
                            }
                        }
                        .padding(.horizontal)
                    }

                    // Griglia Dispositivi con Drag & Drop
                    if viewModel.isRefreshing && viewModel.entities.isEmpty {
                        Spacer()
                        ProgressView("Sincronizzazione Domus...")
                        Spacer()
                    } else if viewModel.filteredEntities.isEmpty {
                        Spacer()
                        emptyStateView
                        Spacer()
                    } else {
                        ScrollView {
                            LazyVGrid(columns: columns, spacing: 12) {
                                ForEach(Array(viewModel.filteredEntities.enumerated()), id: \.element.id) { index, entity in
                                    EntityCardView(
                                        entity: entity,
                                        onToggle: { id in
                                            viewModel.toggleEntity(id: id)
                                        },
                                        onOpenDetail: { selected in
                                            viewModel.selectEntityForDetail(selected)
                                        }
                                    )
                                    .draggable(entity.id)
                                    .dropDestination(for: String.self) { items, _ in
                                        guard let dragged = items.first,
                                              let fromIndex = viewModel.filteredEntities.firstIndex(where: { $0.id == dragged }) else {
                                            return false
                                        }
                                        withAnimation(.spring(response: 0.35, dampingFraction: 0.8)) {
                                            viewModel.moveEntity(from: fromIndex, to: index)
                                        }
                                        return true
                                    }
                                }
                            }
                            .padding(.horizontal)
                            .padding(.bottom, 24)
                        }
                        .refreshable {
                            viewModel.initialConnectAndSync()
                        }
                    }
                }
            }
            .navigationTitle("Domus")
            .toolbar {
                ToolbarItem(placement: .automatic) {
                    HStack(spacing: 8) {
                        Button(action: {
                            Haptics.selection()
                            withAnimation(.spring(response: 0.35, dampingFraction: 0.8)) {
                                isFloorplanMode.toggle()
                            }
                        }) {
                            Image(systemName: isFloorplanMode ? "square.grid.2x2.fill" : "map.fill")
                                .font(.system(size: 13, weight: .bold))
                                .foregroundColor(.blue)
                                .padding(6)
                                .background(Color.blue.opacity(0.12))
                                .clipShape(Circle())
                        }

                        ConnectionStatusBannerView(connection: viewModel.connection) {
                            viewModel.reconnect()
                        }

                        Button(action: {
                            Haptics.light()
                            viewModel.showAutomationSettings = true
                        }) {
                            Image(systemName: "antenna.radiowaves.left.and.right")
                                .font(.system(size: 13, weight: .bold))
                                .foregroundColor(.cyan)
                                .padding(6)
                                .background(Color.cyan.opacity(0.12))
                                .clipShape(Circle())
                        }

                        Button(action: {
                            Haptics.light()
                            viewModel.showKioskSettings = true
                        }) {
                            Image(systemName: "ipad.landscape")
                                .font(.system(size: 13, weight: .bold))
                                .foregroundColor(.primary)
                                .padding(6)
                                .background(Color.secondary.opacity(0.12))
                                .clipShape(Circle())
                        }
                    }
                }
            }
            .sheet(item: $viewModel.selectedEntityForDetail) { entity in
                EntityDetailRouterView(entity: entity, viewModel: viewModel)
            }
            .sheet(isPresented: $viewModel.showAutomationSettings) {
                PresenceAndNfcSheet(
                    presenceManager: viewModel.presenceManager,
                    notificationManager: viewModel.notificationManager,
                    nfcManager: viewModel.nfcManager
                )
            }
            .sheet(isPresented: $viewModel.showKioskSettings) {
                KioskSettingsSheet(kioskManager: viewModel.kioskManager)
            }
            .overlay {
                if viewModel.kioskManager.displayState == .ambientScreensaver {
                    KioskScreensaverView {
                        viewModel.kioskManager.wakeFromScreensaver()
                    }
                    .transition(.opacity)
                }
            }
        }
        .animation(.easeInOut(duration: 0.3), value: viewModel.kioskManager.displayState)
        .onAppear {
            viewModel.initialConnectAndSync()
        }

    }

    // MARK: - Subviews

    private func areaButton(label: String, areaId: String?) -> some View {
        Button(action: { viewModel.selectArea(areaId) }) {
            Text(label)
                .font(.system(size: 13, weight: .semibold))
                .padding(.horizontal, 14)
                .padding(.vertical, 8)
                .background(viewModel.selectedAreaId == areaId ? Color.blue : Color.secondary.opacity(0.15))
                .foregroundColor(viewModel.selectedAreaId == areaId ? .white : .primary)
                .clipShape(Capsule())
        }
    }

    private var emptyStateView: some View {
        VStack(spacing: 8) {
            Text("📡 Nessun dispositivo trovato")
                .font(.headline)
            Text("Verifica che il server Domus sia avviato sulla rete locale")
                .font(.subheadline)
                .foregroundColor(.secondary)
            Button("Riprova Scansione LAN") {
                viewModel.reconnect()
            }
            .padding(.top, 10)
        }
    }
}
