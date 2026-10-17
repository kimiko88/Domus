package com.domus.app

import android.content.Context
import android.net.nsd.NsdManager
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.ui.Modifier
import com.domus.app.data.mdns.MdnsDiscoveryEngine
import com.domus.app.data.network.DomusApiClient
import com.domus.app.data.network.DualTransportManager
import com.domus.app.data.repository.DefaultConnectionRepository
import com.domus.app.data.repository.DefaultEntityRepository
import com.domus.app.data.storage.OfflineEntityCache
import com.domus.app.data.storage.SharedPreferencesTokenStorage
import com.domus.app.domain.usecase.DiscoverAndConnectUseCase
import com.domus.app.domain.usecase.SyncEntitiesUseCase
import com.domus.app.domain.usecase.ToggleEntityUseCase
import com.domus.app.presentation.screens.adaptive.AdaptiveMainLayout
import com.domus.app.presentation.theme.DomusTheme
import com.domus.app.presentation.viewmodel.DashboardViewModel
import com.domus.app.services.kiosk.DomusKioskController
import com.domus.app.services.location.RoomPresenceBeaconManager
import com.domus.app.services.nfc.NfcTagManager
import com.domus.app.services.notifications.DomusNotificationHelper

class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        DomusNotificationHelper.createNotificationChannels(this)

        val nsdManager = getSystemService(Context.NSD_SERVICE) as? NsdManager
        val mdnsEngine = MdnsDiscoveryEngine(nsdManager = nsdManager)
        val transportManager = DualTransportManager()
        val apiClient = DomusApiClient()
        val tokenStorage = SharedPreferencesTokenStorage(this)
        val entityCache = OfflineEntityCache()

        val connectionRepo = DefaultConnectionRepository(mdnsEngine, transportManager, tokenStorage)
        val entityRepo = DefaultEntityRepository(apiClient, transportManager, entityCache, tokenStorage)

        val discoverUseCase = DiscoverAndConnectUseCase(connectionRepo)
        val syncUseCase = SyncEntitiesUseCase(entityRepo)
        val toggleUseCase = ToggleEntityUseCase(entityRepo)

        val kioskController = DomusKioskController(this)
        val beaconManager = RoomPresenceBeaconManager(this)
        val nfcManager = NfcTagManager(this)

        val viewModel = DashboardViewModel(
            entityRepository = entityRepo,
            connectionRepository = connectionRepo,
            discoverAndConnectUseCase = discoverUseCase,
            syncEntitiesUseCase = syncUseCase,
            toggleEntityUseCase = toggleUseCase,
            apiClient = apiClient,
            tokenStorage = tokenStorage,
            beaconManager = beaconManager,
            nfcManager = nfcManager,
            kioskController = kioskController
        )

        setContent {
            DomusTheme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    AdaptiveMainLayout(viewModel = viewModel)
                }
            }
        }
    }
}
