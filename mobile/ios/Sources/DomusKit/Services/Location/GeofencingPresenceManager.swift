import Foundation
import CoreLocation
import Network

/// Stato di presenza dell'utente rispetto all'abitazione.
public enum PresenceState: String, Codable, Sendable {
    case home = "home"
    case away = "away"
    case unknown = "unknown"
}

/// Modello di configurazione del perimetro domestico.
public struct HomePerimeterConfig: Codable, Sendable {
    public var latitude: Double
    public var longitude: Double
    public var radiusMeters: Double
    public var homeWifiSSID: String?

    public init(
        latitude: Double = 41.9028,
        longitude: Double = 12.4964,
        radiusMeters: Double = 150.0,
        homeWifiSSID: String? = nil
    ) {
        self.latitude = latitude
        self.longitude = longitude
        self.radiusMeters = radiusMeters
        self.homeWifiSSID = homeWifiSSID
    }
}

/// Orchestratore di presenza Multi-Tier a batteria zero — Fase 3.
/// Combina rilevamento Wi-Fi locale (Tier 1) e Geofencing circolare CoreLocation (Tier 2).
@MainActor
public final class GeofencingPresenceManager: NSObject, ObservableObject {
    @Published public var presenceState: PresenceState = .unknown
    @Published public var isMonitoringActive: Bool = false
    @Published public var authorizationStatus: CLAuthorizationStatus = .notDetermined
    @Published public var lastLocationTimestamp: Date? = nil
    @Published public var config: HomePerimeterConfig

    private let locationManager = CLLocationManager()
    private let pathMonitor = NWPathMonitor()
    private let monitorQueue = DispatchQueue(label: "com.domus.network.presence")
    private let httpClient: DomusHttpClient?

    public var onPresenceChanged: ((PresenceState) -> Void)?

    public init(config: HomePerimeterConfig = HomePerimeterConfig(), httpClient: DomusHttpClient? = nil) {
        self.config = config
        self.httpClient = httpClient
        super.init()

        locationManager.delegate = self
        locationManager.desiredAccuracy = kCLLocationAccuracyHundredMeters
        locationManager.pausesLocationUpdatesAutomatically = true
        #if os(iOS)
        locationManager.allowsBackgroundLocationUpdates = false // per geofencing puro non serve tracciamento continuo
        #endif
        self.authorizationStatus = locationManager.authorizationStatus

        setupNetworkPathMonitoring()
    }

    /// Richiede le autorizzazioni di geolocalizzazione (Always per geofence in background).
    public func requestPermissions() {
        #if os(iOS)
        locationManager.requestAlwaysAuthorization()
        #else
        locationManager.requestWhenInUseAuthorization()
        #endif
    }

    /// Avvia il monitoraggio della geofence domestica.
    public func startMonitoring() {
        guard CLLocationManager.isMonitoringAvailable(for: CLCircularRegion.self) else { return }

        let center = CLLocationCoordinate2D(latitude: config.latitude, longitude: config.longitude)
        let region = CLCircularRegion(center: center, radius: config.radiusMeters, identifier: "domus_home_geofence")
        region.notifyOnEntry = true
        region.notifyOnExit = true

        locationManager.startMonitoring(for: region)
        locationManager.requestState(for: region)
        isMonitoringActive = true
    }

    /// Interrompe il monitoraggio geofence.
    public func stopMonitoring() {
        for region in locationManager.monitoredRegions {
            locationManager.stopMonitoring(for: region)
        }
        isMonitoringActive = false
    }

    /// Aggiorna le coordinate della casa e riavvia il monitoraggio.
    public func updateHomeLocation(latitude: Double, longitude: Double, radius: Double) {
        config.latitude = latitude
        config.longitude = longitude
        config.radiusMeters = radius
        if isMonitoringActive {
            startMonitoring()
        }
    }

    // MARK: - Tier 1: Local Network Presence

    private func setupNetworkPathMonitoring() {
        pathMonitor.pathUpdateHandler = { [weak self] path in
            guard let self else { return }
            Task { @MainActor in
                if path.status == .satisfied && !path.isExpensive {
                    // Connesso a Wi-Fi locale non-metred
                    if self.presenceState != .home {
                        self.updatePresence(.home, source: "Wi-Fi LAN")
                    }
                }
            }
        }
        pathMonitor.start(queue: monitorQueue)
    }

    private func updatePresence(_ newState: PresenceState, source: String) {
        guard presenceState != newState else { return }
        presenceState = newState
        lastLocationTimestamp = Date()
        onPresenceChanged?(newState)

        // Sincronizza con il backend Domus
        Task {
            struct PresencePayload: Encodable {
                let state: String
                let source: String
                let timestamp: String
            }
            let payload = PresencePayload(
                state: newState.rawValue,
                source: source,
                timestamp: ISO8601DateFormatter().string(from: Date())
            )
            _ = try? await self.httpClient?.postVoid(path: "/api/v1/presence/update", body: payload)
        }
    }
}

// MARK: - CLLocationManagerDelegate

extension GeofencingPresenceManager: CLLocationManagerDelegate {
    public nonisolated func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        Task { @MainActor in
            self.authorizationStatus = manager.authorizationStatus
            #if os(iOS)
            if manager.authorizationStatus == .authorizedAlways || manager.authorizationStatus == .authorizedWhenInUse {
                self.startMonitoring()
            }
            #else
            if manager.authorizationStatus == .authorizedAlways {
                self.startMonitoring()
            }
            #endif
        }

    }

    public nonisolated func locationManager(_ manager: CLLocationManager, didEnterRegion region: CLRegion) {
        if region.identifier == "domus_home_geofence" {
            Task { @MainActor in
                self.updatePresence(.home, source: "Geofence Enter")
            }
        }
    }

    public nonisolated func locationManager(_ manager: CLLocationManager, didExitRegion region: CLRegion) {
        if region.identifier == "domus_home_geofence" {
            Task { @MainActor in
                self.updatePresence(.away, source: "Geofence Exit")
            }
        }
    }

    public nonisolated func locationManager(_ manager: CLLocationManager, didDetermineState state: CLRegionState, for region: CLRegion) {
        if region.identifier == "domus_home_geofence" {
            Task { @MainActor in
                switch state {
                case .inside:
                    self.updatePresence(.home, source: "Geofence State Inside")
                case .outside:
                    self.updatePresence(.away, source: "Geofence State Outside")
                case .unknown:
                    break
                }
            }
        }
    }
}
