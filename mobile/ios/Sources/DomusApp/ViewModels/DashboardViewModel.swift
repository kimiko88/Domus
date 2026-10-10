import Foundation
import SwiftUI
import DomusKit

@MainActor
public final class DashboardViewModel: ObservableObject {
    @Published public var connection: ServerConnection = ServerConnection()
    @Published public var entities: [Entity] = []
    @Published public var areas: [Area] = []
    @Published public var selectedAreaId: String? = nil
    @Published public var searchQuery: String = ""
    @Published public var isRefreshing: Bool = false

    private let entityRepository: EntityRepositoryProtocol
    private let connectionRepository: ConnectionRepositoryProtocol
    private let discoverAndConnectUseCase: DiscoverAndConnectUseCase
    private let syncEntitiesUseCase: SyncEntitiesUseCase
    private let toggleEntityUseCase: ToggleEntityUseCase

    public init(
        entityRepository: EntityRepositoryProtocol,
        connectionRepository: ConnectionRepositoryProtocol,
        discoverAndConnectUseCase: DiscoverAndConnectUseCase,
        syncEntitiesUseCase: SyncEntitiesUseCase,
        toggleEntityUseCase: ToggleEntityUseCase
    ) {
        self.entityRepository = entityRepository
        self.connectionRepository = connectionRepository
        self.discoverAndConnectUseCase = discoverAndConnectUseCase
        self.syncEntitiesUseCase = syncEntitiesUseCase
        self.toggleEntityUseCase = toggleEntityUseCase
    }

    public var filteredEntities: [Entity] {
        entities.filter { entity in
            let matchesArea = selectedAreaId == nil || entity.areaId == selectedAreaId
            let matchesQuery = searchQuery.trimmingCharacters(in: .whitespaces).isEmpty ||
                entity.name.localizedCaseInsensitiveContains(searchQuery) ||
                entity.domain.localizedCaseInsensitiveContains(searchQuery)
            return matchesArea && matchesQuery
        }
    }

    public func initialConnectAndSync() {
        Task {
            isRefreshing = true
            _ = try? await discoverAndConnectUseCase.execute(preferLocal: true)
            self.connection = await connectionRepository.getCurrentConnection()
            self.entities = (try? await syncEntitiesUseCase.execute()) ?? []
            self.areas = await entityRepository.getAreas()
            isRefreshing = false
        }
    }

    public func toggleEntity(id: String) {
        Task {
            if let updated = try? await toggleEntityUseCase.execute(entityId: id) {
                if let index = entities.firstIndex(where: { $0.id == id }) {
                    entities[index] = updated
                }
            }
        }
    }

    public func selectArea(_ areaId: String?) {
        self.selectedAreaId = areaId
    }

    public func reconnect() {
        initialConnectAndSync()
    }
}
