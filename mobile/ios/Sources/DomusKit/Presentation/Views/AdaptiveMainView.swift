import SwiftUI

/// Layout adattivo nativo per iPadOS e macOS (NavigationSplitView a 3 colonne) con fallback fluido per iPhone.
public struct AdaptiveMainView: View {
    @ObservedObject public var viewModel: DashboardViewModel
    #if os(iOS)
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    #endif

    @State private var selectedCategory: String? = nil
    @State private var showFloorplan: Bool = false
    @State private var columnVisibility: NavigationSplitViewVisibility = .all
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

    public init(viewModel: DashboardViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        #if os(iOS)
        if horizontalSizeClass == .regular {
            ipadMultiColumnLayout
        } else {
            DashboardView(viewModel: viewModel)
        }
        #else
        ipadMultiColumnLayout
        #endif
    }

    // MARK: - iPadOS & macOS 3-Column NavigationSplitView Layout

    private var ipadMultiColumnLayout: some View {
        NavigationSplitView(columnVisibility: $columnVisibility) {
            // Colonna 1: Sidebar (Stanze e Categorie)
            sidebarView
                .navigationTitle("Domus")
        } content: {
            // Colonna 2: Contenuto Principale (Griglia o Pianta 2D)
            contentGridView
                .navigationTitle(viewModel.selectedAreaId != nil ? (viewModel.areas.first(where: { $0.id == viewModel.selectedAreaId })?.name ?? "Stanza") : "Tutti i Dispositivi")
        } detail: {
            // Colonna 3: Inspector / Dettaglio Rapido (Telecamere, Clima, Allarme)
            detailInspectorView
        }
        .navigationSplitViewStyle(.balanced)
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
        .sheet(isPresented: $viewModel.showLoginSheet) {
            LoginView(viewModel: viewModel)
        }
    }

    // MARK: - Sidebar

    private var sidebarView: some View {
        List {
            Section(header: Text("Riepilogo Casa")) {
                statusPillRow(
                    icon: "lightbulb.fill",
                    title: "Luci Accese",
                    value: "\(viewModel.entities.filter({ $0.domain == "light" && $0.isOn }).count)",
                    color: .yellow
                )

                statusPillRow(
                    icon: "shield.checkered",
                    title: "Stato Allarme",
                    value: viewModel.entities.first(where: { $0.domain == "alarm_control_panel" })?.state.capitalized ?? "Disarmato",
                    color: .green
                )

                let powerSensor = viewModel.entities.first(where: {
                    $0.domain == "sensor" && ($0.id.contains("power") || $0.id.contains("energy") || $0.id.contains("potenza"))
                })
                let powerValue = powerSensor.map { "\($0.state) \($0.attributes["unit_of_measurement"]?.description ?? "W")" } ?? "0 W"
                statusPillRow(
                    icon: "bolt.fill",
                    title: "Potenza Attuale",
                    value: powerValue,
                    color: .cyan
                )
            }

            Section(header: Text("Visualizzazione")) {
                Button(action: {
                    Haptics.selection()
                    showFloorplan.toggle()
                }) {
                    Label(showFloorplan ? "Mostra Griglia Dispositivi" : "Mostra Pianta 2D", systemImage: showFloorplan ? "square.grid.2x2" : "map.fill")
                        .foregroundColor(.blue)
                }
            }

            Section(header: Text("Funzionalità Casa")) {
                Button(action: {
                    Haptics.selection()
                    showAssistant = true
                }) {
                    Label("Assistente AI", systemImage: "sparkles")
                        .foregroundColor(.cyan)
                }

                Button(action: {
                    Haptics.selection()
                    showEnergyFlow = true
                }) {
                    Label("Cruscotto Energia", systemImage: "bolt.fill")
                        .foregroundColor(.yellow)
                }

                Button(action: {
                    Haptics.selection()
                    showShoppingList = true
                }) {
                    Label("Lista della Spesa", systemImage: "cart.fill")
                        .foregroundColor(.green)
                }

                Button(action: {
                    Haptics.selection()
                    showPantry = true
                }) {
                    Label("Dispensa & Pasti", systemImage: "fork.knife")
                        .foregroundColor(.orange)
                }

                Button(action: {
                    Haptics.selection()
                    showFinance = true
                }) {
                    Label("Bilancio & Spese", systemImage: "dollarsign.circle.fill")
                        .foregroundColor(.green)
                }

                Button(action: {
                    Haptics.selection()
                    showChores = true
                }) {
                    Label("Faccende Domestiche", systemImage: "checklist")
                        .foregroundColor(.orange)
                }

                Button(action: {
                    Haptics.selection()
                    showAutomationBuilder = true
                }) {
                    Label("Regole & Automazioni", systemImage: "wand.and.stars")
                        .foregroundColor(.purple)
                }

                Button(action: {
                    Haptics.selection()
                    showActivityTimeline = true
                }) {
                    Label("Timeline della Casa", systemImage: "clock.arrow.circlepath")
                        .foregroundColor(.blue)
                }

                Button(action: {
                    Haptics.selection()
                    showMeshDiagnostics = true
                }) {
                    Label("Diagnostica Mesh & Hub", systemImage: "antenna.radiowaves.left.and.right")
                        .foregroundColor(.green)
                }

                Button(action: {
                    Haptics.selection()
                    showGuestAccess = true
                }) {
                    Label("Accesso Ospiti & Membri", systemImage: "person.2.fill")
                        .foregroundColor(.indigo)
                }

                Button(action: {
                    Haptics.selection()
                    showNotifications = true
                }) {
                    Label("Centro Notifiche", systemImage: "bell.fill")
                        .foregroundColor(.red)
                }
            }

            Section(header: Text("Stanze")) {
                Button(action: {
                    Haptics.selection()
                    viewModel.selectArea(nil)
                }) {
                    HStack {
                        Label("Tutte le Stanze", systemImage: "house.fill")
                        Spacer()
                        if viewModel.selectedAreaId == nil {
                            Image(systemName: "checkmark")
                                .foregroundColor(.blue)
                        }
                    }
                }
                .foregroundColor(viewModel.selectedAreaId == nil ? .blue : .primary)

                ForEach(viewModel.areas) { area in
                    Button(action: {
                        Haptics.selection()
                        viewModel.selectArea(area.id)
                    }) {
                        HStack {
                            Label(area.name, systemImage: "door.left.hand.closed")
                            Spacer()
                            if viewModel.selectedAreaId == area.id {
                                Image(systemName: "checkmark")
                                    .foregroundColor(.blue)
                            }
                        }
                    }
                }
            }

            Section(header: Text("Server & Account")) {
                Button(action: {
                    Haptics.selection()
                    viewModel.showLoginSheet = true
                }) {
                    HStack {
                        Label(
                            viewModel.authSession.isAuthenticated ? (viewModel.authSession.username ?? "Utente") : "Accedi al Server",
                            systemImage: viewModel.authSession.isAuthenticated ? "person.crop.circle.badge.checkmark" : "person.crop.circle.badge.exclamationmark"
                        )
                        .foregroundColor(viewModel.authSession.isAuthenticated ? .primary : .orange)
                        Spacer()
                        Text(viewModel.connection.mode.displayName)
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }
                }

                if viewModel.authSession.isAuthenticated {
                    Button(action: {
                        Haptics.selection()
                        viewModel.logout()
                    }) {
                        Label("Disconnetti", systemImage: "rectangle.portrait.and.arrow.right")
                            .foregroundColor(.red)
                    }
                }
            }
        }
        .listStyle(.sidebar)
    }

    // MARK: - Content Grid / Floorplan

    private var contentGridView: some View {
        VStack(spacing: 12) {
            if showFloorplan {
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
                    .padding()
                }
            } else {
                ScrollView {
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 160), spacing: 14)], spacing: 14) {
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
                    .padding()
                }
            }
        }
        .toolbar {
            ToolbarItem(placement: .automatic) {
                Button(action: {
                    Haptics.light()
                    showFloorplan.toggle()
                }) {
                    Image(systemName: showFloorplan ? "square.grid.2x2" : "map.fill")
                }
            }
        }
    }

    // MARK: - Detail / Inspector

    private var detailInspectorView: some View {
        Group {
            if let entity = viewModel.selectedEntityForDetail {
                EntityDetailRouterView(entity: entity, viewModel: viewModel)
            } else {
                VStack(spacing: 16) {
                    Image(systemName: "homekit")
                        .font(.system(size: 48))
                        .foregroundColor(.secondary)
                    Text("Pannello Dettagli")
                        .font(.title2.bold())
                    Text("Seleziona un dispositivo dalla griglia per visualizzare i controlli dedicati.")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
    }

    private func statusPillRow(icon: String, title: String, value: String, color: Color) -> some View {
        HStack(spacing: 10) {
            Image(systemName: icon)
                .foregroundColor(color)
            Text(title)
                .font(.subheadline)
            Spacer()
            Text(value)
                .font(.subheadline.bold())
                .foregroundColor(.secondary)
        }
    }
}
