import SwiftUI
import DomusKit

public struct DashboardView: View {
    @ObservedObject public var viewModel: DashboardViewModel

    private let columns = [
        GridItem(.flexible(), spacing: 12),
        GridItem(.flexible(), spacing: 12)
    ]

    public var body: some View {
        NavigationStack {
            VStack(spacing: 12) {
                // Search Bar
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("Cerca dispositivi o stanze...", text: $viewModel.searchQuery)
                        .textFieldStyle(.plain)
                }
                .padding(10)
                .background(Color.secondary.opacity(0.12))
                .clipShape(RoundedRectangle(cornerRadius: 12))
                .padding(.horizontal)

                // Area Selector Horizontal Scroll
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        Button(action: { viewModel.selectArea(nil) }) {
                            Text("Tutte le Stanze")
                                .font(.system(size: 13, weight: .semibold))
                                .padding(.horizontal, 14)
                                .padding(.vertical, 8)
                                .background(viewModel.selectedAreaId == nil ? Color.blue : Color.secondary.opacity(0.15))
                                .foregroundColor(viewModel.selectedAreaId == nil ? .white : .primary)
                                .clipShape(Capsule())
                        }

                        ForEach(viewModel.areas) { area in
                            Button(action: { viewModel.selectArea(area.id) }) {
                                Text(area.name)
                                    .font(.system(size: 13, weight: .semibold))
                                    .padding(.horizontal, 14)
                                    .padding(.vertical, 8)
                                    .background(viewModel.selectedAreaId == area.id ? Color.blue : Color.secondary.opacity(0.15))
                                    .foregroundColor(viewModel.selectedAreaId == area.id ? .white : .primary)
                                    .clipShape(Capsule())
                            }
                        }
                    }
                    .padding(.horizontal)
                }

                // Grid of Entities
                if viewModel.isRefreshing && viewModel.entities.isEmpty {
                    Spacer()
                    ProgressView()
                    Spacer()
                } else if viewModel.filteredEntities.isEmpty {
                    Spacer()
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
                    Spacer()
                } else {
                    ScrollView {
                        LazyVGrid(columns: columns, spacing: 12) {
                            ForEach(viewModel.filteredEntities) { entity in
                                EntityCardView(entity: entity) { id in
                                    viewModel.toggleEntity(id: id)
                                }
                            }
                        }
                        .padding(.horizontal)
                        .padding(.bottom, 20)
                    }
                }
            }
            .navigationTitle("Domus")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    ConnectionStatusBannerView(connection: viewModel.connection) {
                        viewModel.reconnect()
                    }
                }
            }
        }
        .onAppear {
            viewModel.initialConnectAndSync()
        }
    }
}
