import SwiftUI

/// Vista principale della dashboard — include schede interattive Fase 2, filtro stanze e apertura modale di dettaglio.
public struct DashboardView: View {
    @ObservedObject public var viewModel: DashboardViewModel

    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Environment(\.accessibilityDifferentiateWithoutColor) private var differentiateWithoutColor
    @Environment(\.accessibilityReduceTransparency) private var reduceTransparency
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    @State private var isFloorplanMode: Bool = false
    @State private var draggedEntityId: String? = nil
    @State private var showShoppingList: Bool = false
    @State private var showAssistant: Bool = false
    @State private var showChores: Bool = false
    @State private var showEnergyFlow: Bool = false
    @State private var showNotifications: Bool = false
    @State private var showAutomationBuilder: Bool = false
    @State private var showActivityTimeline: Bool = false
    @State private var showMeshDiagnostics: Bool = false
    @State private var showGuestAccess: Bool = false
    @State private var showFinance: Bool = false
    @State private var showPantry: Bool = false

    private var columns: [GridItem] {
        if dynamicTypeSize.isAccessibilitySize {
            return [GridItem(.flexible(), spacing: 12)]
        }
        return [
            GridItem(.flexible(), spacing: 12),
            GridItem(.flexible(), spacing: 12)
        ]
    }

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
                    .padding(.vertical, 8)
                    .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.cyan.opacity(0.12))
                    .clipShape(Capsule())
                    .overlay(
                        Capsule()
                            .stroke(differentiateWithoutColor ? Color.cyan : Color.clear, lineWidth: 1)
                    )
                    .padding(.horizontal)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Stanza rilevata: \(roomName)\(viewModel.beaconManager.currentDistanceMeters != nil ? String(format: ", distanza %.1f metri", viewModel.beaconManager.currentDistanceMeters!) : "")")
                }

                // Search Bar
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("Cerca dispositivi o stanze...", text: $viewModel.searchQuery)
                        .textFieldStyle(.plain)
                        .accessibilityLabel("Cerca dispositivi o stanze")
                    if !viewModel.searchQuery.isEmpty {
                        Button(action: { viewModel.searchQuery = "" }) {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundColor(.secondary)
                                .frame(width: 44, height: 44)
                        }
                        .accessibilityLabel("Cancella testo di ricerca")
                    }
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.12))
                .clipShape(RoundedRectangle(cornerRadius: 12))
                .padding(.horizontal)

                // Barra Accesso Rapido Moduli Casa
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        quickFeaturePill(icon: "sparkles", label: "Assistant", color: .cyan) {
                            showAssistant = true
                        }
                        quickFeaturePill(icon: "wand.and.stars", label: "Automazioni", color: .purple) {
                            showAutomationBuilder = true
                        }
                        quickFeaturePill(icon: "clock.arrow.circlepath", label: "Timeline", color: .blue) {
                            showActivityTimeline = true
                        }
                        quickFeaturePill(icon: "bolt.fill", label: "Energia", color: .yellow) {
                            showEnergyFlow = true
                        }
                        quickFeaturePill(icon: "antenna.radiowaves.left.and.right", label: "Rete & Mesh", color: .green) {
                            showMeshDiagnostics = true
                        }
                        quickFeaturePill(icon: "person.2.fill", label: "Ospiti", color: .indigo) {
                            showGuestAccess = true
                        }
                        quickFeaturePill(icon: "cart.fill", label: "Spesa", color: .mint) {
                            showShoppingList = true
                        }
                        quickFeaturePill(icon: "fork.knife", label: "Dispensa & Pasti", color: .orange) {
                            showPantry = true
                        }
                        quickFeaturePill(icon: "dollarsign.circle.fill", label: "Spese", color: .green) {
                            showFinance = true
                        }
                        quickFeaturePill(icon: "checklist", label: "Faccende", color: .orange) {
                            showChores = true
                        }
                        quickFeaturePill(icon: "bell.fill", label: "Notifiche", color: .red) {
                            showNotifications = true
                        }
                    }
                    .padding(.horizontal)
                }

                // Mini Finance Summary Card (Spese del Mese)
                if let summary = viewModel.financeSummary, summary.totalSpent > 0 {
                    Button(action: {
                        Haptics.light()
                        showFinance = true
                    }) {
                        HStack {
                            Image(systemName: "dollarsign.circle.fill")
                                .font(.title2)
                                .foregroundColor(.green)

                            VStack(alignment: .leading, spacing: 2) {
                                Text("Spese del Mese")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                                Text(String(format: "€%.2f", summary.totalSpent))
                                    .font(.subheadline.bold())
                                    .foregroundColor(.primary)
                            }

                            Spacer()

                            if summary.monthlyBudget > 0 {
                                VStack(alignment: .trailing, spacing: 2) {
                                    Text(summary.remainingBudget >= 0 ? "Budget Residuo" : "Sforamento")
                                        .font(.caption)
                                        .foregroundColor(summary.remainingBudget >= 0 ? .green : .red)
                                    Text(String(format: "€%.2f", abs(summary.remainingBudget)))
                                        .font(.subheadline.bold())
                                        .foregroundColor(summary.remainingBudget >= 0 ? .green : .red)
                                }
                            }

                            Image(systemName: "chevron.right")
                                .font(.caption.bold())
                                .foregroundColor(.secondary)
                        }
                        .padding(.horizontal, 14)
                        .padding(.vertical, 8)
                        .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.1))
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                        .padding(.horizontal)
                    }
                    .accessibilityLabel("Bilancio Casa e Spese del Mese, totale speso: \(String(format: "€%.2f", summary.totalSpent))")
                }

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
                                        withAnimation(reduceMotion ? .none : .spring(response: 0.35, dampingFraction: 0.8)) {
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
                            withAnimation(reduceMotion ? .none : .spring(response: 0.35, dampingFraction: 0.8)) {
                                isFloorplanMode.toggle()
                            }
                        }) {
                            Image(systemName: isFloorplanMode ? "square.grid.2x2.fill" : "map.fill")
                                .font(.subheadline.bold())
                                .foregroundColor(.blue)
                                .frame(width: 44, height: 44)
                                .background(Color.blue.opacity(0.12))
                                .clipShape(Circle())
                        }
                        .accessibilityLabel(isFloorplanMode ? "Passa a vista griglia dispositivi" : "Passa a vista planimetria 2D")
                        .accessibilityAddTraits(.isButton)

                        ConnectionStatusBannerView(connection: viewModel.connection) {
                            viewModel.reconnect()
                        }

                        Button(action: {
                            Haptics.light()
                            viewModel.showAutomationSettings = true
                        }) {
                            Image(systemName: "antenna.radiowaves.left.and.right")
                                .font(.subheadline.bold())
                                .foregroundColor(.cyan)
                                .frame(width: 44, height: 44)
                                .background(Color.cyan.opacity(0.12))
                                .clipShape(Circle())
                        }
                        .accessibilityLabel("Automazioni e Rilevamento Presenza")
                        .accessibilityAddTraits(.isButton)

                        Button(action: {
                            Haptics.light()
                            viewModel.showKioskSettings = true
                        }) {
                            Image(systemName: "ipad.landscape")
                                .font(.subheadline.bold())
                                .foregroundColor(.primary)
                                .frame(width: 44, height: 44)
                                .background(Color.secondary.opacity(0.12))
                                .clipShape(Circle())
                        }
                        .accessibilityLabel("Impostazioni Chiosco")
                        .accessibilityAddTraits(.isButton)

                        Button(action: {
                            Haptics.light()
                            viewModel.showLoginSheet = true
                        }) {
                            Image(systemName: viewModel.authSession.isAuthenticated ? "person.crop.circle.badge.checkmark" : "person.crop.circle.badge.exclamationmark")
                                .font(.subheadline.bold())
                                .foregroundColor(viewModel.authSession.isAuthenticated ? .blue : .orange)
                                .frame(width: 44, height: 44)
                                .background((viewModel.authSession.isAuthenticated ? Color.blue : Color.orange).opacity(0.12))
                                .clipShape(Circle())
                        }
                        .accessibilityLabel(viewModel.authSession.isAuthenticated ? "Profilo e Server Domus (\(viewModel.authSession.username ?? "Utente"))" : "Accedi a Domus")
                        .accessibilityAddTraits(.isButton)
                    }
                }
            }
            .sheet(item: $viewModel.selectedEntityForDetail) { entity in
                EntityDetailRouterView(entity: entity, viewModel: viewModel)
            }
            .sheet(isPresented: $viewModel.showLoginSheet) {
                LoginView(viewModel: viewModel)
            }
            .sheet(isPresented: $viewModel.showAutomationSettings) {
                PresenceAndNfcSheet(
                    presenceManager: viewModel.presenceManager,
                    notificationManager: viewModel.notificationManager,
                    nfcManager: viewModel.nfcManager
                )
            }
            .sheet(isPresented: $showShoppingList) {
                ShoppingListView()
            }
            .sheet(isPresented: $showAssistant) {
                AssistantChatSheet()
            }
            .sheet(isPresented: $showChores) {
                ChoresSheet()
            }
            .sheet(isPresented: $showEnergyFlow) {
                EnergyFlowSheet()
            }
            .sheet(isPresented: $showNotifications) {
                NotificationHistoryView()
            }
            .sheet(isPresented: $showAutomationBuilder) {
                AutomationBuilderSheet()
            }
            .sheet(isPresented: $showActivityTimeline) {
                ActivityTimelineSheet()
            }
            .sheet(isPresented: $showMeshDiagnostics) {
                NetworkAndMeshDiagnosticsSheet()
            }
            .sheet(isPresented: $showGuestAccess) {
                GuestAccessAndFamilySheet()
            }
            .sheet(isPresented: $showFinance) {
                FinanceSheet()
            }
            .sheet(isPresented: $showPantry) {
                PantryMealCalendarView()
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
        .animation(reduceMotion ? .none : .easeInOut(duration: 0.3), value: viewModel.kioskManager.displayState)
        .onAppear {
            viewModel.initialConnectAndSync()
        }
    }

    // MARK: - Subviews

    private func quickFeaturePill(icon: String, label: String, color: Color, action: @escaping () -> Void) -> some View {
        Button(action: {
            Haptics.light()
            action()
        }) {
            HStack(spacing: 5) {
                Image(systemName: icon)
                    .font(.caption2.weight(.bold))
                Text(label)
                    .font(.footnote.weight(.semibold))
            }
            .frame(minHeight: 44)
            .padding(.horizontal, 12)
            .padding(.vertical, 7)
            .background(reduceTransparency ? Color.domusSecondarySystemGroupedBackground : color.opacity(0.14))
            .foregroundColor(color)
            .clipShape(Capsule())
            .overlay(
                Capsule()
                    .stroke(differentiateWithoutColor ? color.opacity(0.5) : Color.clear, lineWidth: 1)
            )
        }
        .accessibilityLabel("\(label), modulo rapido")
        .accessibilityAddTraits(.isButton)
    }

    private func areaButton(label: String, areaId: String?) -> some View {
        Button(action: { viewModel.selectArea(areaId) }) {
            Text(label)
                .font(.subheadline.weight(.semibold))
                .frame(minHeight: 44)
                .padding(.horizontal, 14)
                .padding(.vertical, 8)
                .background(viewModel.selectedAreaId == areaId ? Color.blue : (reduceTransparency ? Color.domusSecondarySystemGroupedBackground : Color.secondary.opacity(0.15)))
                .foregroundColor(viewModel.selectedAreaId == areaId ? .white : .primary)
                .clipShape(Capsule())
                .overlay(
                    Capsule()
                        .stroke(differentiateWithoutColor && viewModel.selectedAreaId == areaId ? Color.white : Color.clear, lineWidth: 1.5)
                )
        }
        .accessibilityLabel("Stanza: \(label)")
        .accessibilityAddTraits(viewModel.selectedAreaId == areaId ? [.isSelected, .isButton] : .isButton)
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
            .font(.subheadline.bold())
            .frame(minHeight: 44)
            .padding(.top, 10)
        }
    }
}
