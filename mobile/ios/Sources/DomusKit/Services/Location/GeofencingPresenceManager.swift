import Foundation
import CoreLocation

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
        latitude: Double = 0.0,
        longitude: Double = 0.0,
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
        guard CLLocationManager.isMonitoringAvailable(for: CLCircularRegion.self),
              isHomeLocationConfigured else { return }

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
        guard latitude.isFinite, longitude.isFinite,
              (-90.0...90.0).contains(latitude),
              (-180.0...180.0).contains(longitude),
              radius.isFinite, radius > 0, radius <= 100_000,
              latitude != 0 || longitude != 0 else { return }

        config.latitude = latitude
        config.longitude = longitude
        config.radiusMeters = radius
        if isMonitoringActive {
            stopMonitoring()
            startMonitoring()
        }
    }

    /// A zero/zero location is the unconfigured default, never Rome or another
    /// inferred home address. Wi-Fi reachability alone cannot prove presence
    /// on the home LAN because NWPathMonitor does not expose the SSID.
    private var isHomeLocationConfigured: Bool {
        config.latitude.isFinite && config.longitude.isFinite &&
        (-90.0...90.0).contains(config.latitude) &&
        (-180.0...180.0).contains(config.longitude) &&
        config.radiusMeters.isFinite && config.radiusMeters > 0 &&
        config.radiusMeters <= 100_000 &&
        (config.latitude != 0 || config.longitude != 0)
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
