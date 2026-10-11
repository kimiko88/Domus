import SwiftUI

/// Vista principale della dashboard — include schede interattive Fase 2, filtro stanze e apertura modale di dettaglio.
public struct DashboardView: View {
    @ObservedObject public var viewModel: DashboardViewModel

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

                // Griglia Dispositivi
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
                            ForEach(viewModel.filteredEntities) { entity in
                                EntityCardView(
                                    entity: entity,
                                    onToggle: { id in
                                        viewModel.toggleEntity(id: id)
                                    },
                                    onOpenDetail: { selected in
                                        viewModel.selectEntityForDetail(selected)
                                    }
                                )
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
            .navigationTitle("Domus")
            .toolbar {
                ToolbarItem(placement: .automatic) {
                    HStack(spacing: 8) {
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
