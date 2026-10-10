import SwiftUI
import DomusKit

@main
struct DomusApp: App {
    @StateObject private var viewModel: DashboardViewModel

    init() {
        let mdns = MdnsDiscoveryEngine()
        let ws = DualTransportWebSocketClient()
        let http = DomusHttpClient()
        let tokenStorage = KeychainManager()
        let cache = OfflineEntityCache()

        let entityRepo = DefaultEntityRepository(
            httpClient: http,
            webSocketClient: ws,
            cache: cache,
            tokenStorage: tokenStorage
        )
        let connRepo = DefaultConnectionRepository(
            mdnsEngine: mdns,
            webSocketClient: ws,
            tokenStorage: tokenStorage
        )

        let discoverUseCase = DiscoverAndConnectUseCase(connectionRepository: connRepo)
        let syncUseCase = SyncEntitiesUseCase(entityRepository: entityRepo)
        let toggleUseCase = ToggleEntityUseCase(entityRepository: entityRepo)

        _viewModel = StateObject(wrappedValue: DashboardViewModel(
            entityRepository: entityRepo,
            connectionRepository: connRepo,
            discoverAndConnectUseCase: discoverUseCase,
            syncEntitiesUseCase: syncUseCase,
            toggleEntityUseCase: toggleUseCase
        ))
    }

    var body: some Scene {
        WindowGroup {
            DashboardView(viewModel: viewModel)
        }
    }
}
