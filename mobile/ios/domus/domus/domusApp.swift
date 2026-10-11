//
//  domusApp.swift
//  domus
//
//  Created by Fabrizio Chimienti on 11/10/2026.
//

import SwiftUI
import DomusKit

// ─── Composition Root ──────────────────────────────────────────────────────
// Questo è l'unico file Swift nel target Xcode.
// Tutta la logica (Domain, Data, Presentation) vive nel package locale DomusKit.
// Per aggiungere DomusKit: File → Add Package Dependencies → Add Local → seleziona mobile/ios/
// ──────────────────────────────────────────────────────────────────────────

@main
struct domusApp: App {

    @StateObject private var viewModel: DashboardViewModel

    init() {
        // Data Layer
        let mdns         = MdnsDiscoveryEngine()
        let ws           = DualTransportWebSocketClient()
        let http         = DomusHttpClient()
        let tokenStorage = KeychainManager()
        let cache        = OfflineEntityCache()

        // Repositories
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

        // Use Cases
        let discoverUseCase = DiscoverAndConnectUseCase(connectionRepository: connRepo)
        let syncUseCase     = SyncEntitiesUseCase(entityRepository: entityRepo)
        let toggleUseCase   = ToggleEntityUseCase(entityRepository: entityRepo)

        // ViewModel
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
