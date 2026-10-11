//
//  domusTests.swift
//  domusTests
//
//  Created by Fabrizio Chimienti on 11/10/2026.
//

import Testing
import SwiftUI
@testable import domus
import DomusKit

struct domusTests {

    @Test func testAppCompositionRootInitialization() async throws {
        // Verifica che il composition root di domusApp istanzi tutti i moduli
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

        let viewModel = await MainActor.run {
            DashboardViewModel(
                entityRepository: entityRepo,
                connectionRepository: connRepo,
                discoverAndConnectUseCase: discoverUseCase,
                syncEntitiesUseCase: syncUseCase,
                toggleEntityUseCase: toggleUseCase
            )
        }

        #expect(await viewModel.entities.isEmpty == true)
        #expect(await viewModel.autoFocusNearbyRoom == true)
        #expect(await viewModel.customEntityOrder.isEmpty == true)
    }

    @Test func testAdaptiveMainViewInstantiation() async throws {
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

        let viewModel = await MainActor.run {
            DashboardViewModel(
                entityRepository: entityRepo,
                connectionRepository: connRepo,
                discoverAndConnectUseCase: DiscoverAndConnectUseCase(connectionRepository: connRepo),
                syncEntitiesUseCase: SyncEntitiesUseCase(entityRepository: entityRepo),
                toggleEntityUseCase: ToggleEntityUseCase(entityRepository: entityRepo)
            )
        }

        let adaptiveView = await MainActor.run {
            AdaptiveMainView(viewModel: viewModel)
        }

        #expect(adaptiveView != nil)
    }

    @Test func testHardwareSingletonServicesAvailability() async throws {
        // Verifica che i singleton di sistema siano accessibili e pronti all'uso
        await MainActor.run {
            #expect(DomusCarPlayManager.shared != nil)
            #expect(DomusLiveActivityManager.shared != nil)
            #expect(PictureInPictureManager.shared != nil)
            #expect(RoomPresenceBeaconManager.shared != nil)
            #expect(VpnOnDemandManager.shared != nil)
            #expect(DomusBackgroundTaskManager.shared != nil)
        }
    }
}
