import Foundation
import Combine
import CoreLocation

#if canImport(NearbyInteraction) && os(iOS)
import NearbyInteraction
#endif

/// Modello di configurazione per un Beacon o Tag BLE associato a una stanza domestica.
public struct DomusRoomBeacon: Identifiable, Codable, Equatable, Sendable {
    public var id: String
    public var roomId: String
    public var roomName: String
    public var proximityUUID: String
    public var major: Int
    public var minor: Int
    public var rssiThreshold: Int // es. -65 dBm per considerare l'utente presente nella stanza

    public init(
        id: String,
        roomId: String,
        roomName: String,
        proximityUUID: String = "E2C56DB5-DFFB-48D2-B060-D0F5A71096E0",
        major: Int,
        minor: Int,
        rssiThreshold: Int = -68
    ) {
        self.id = id
        self.roomId = roomId
        self.roomName = roomName
        self.proximityUUID = proximityUUID
        self.major = major
        self.minor = minor
        self.rssiThreshold = rssiThreshold
    }
}

/// Gestore della micro-localizzazione indoor stanza per stanza.
/// Utilizza iBeacon BLE e chip UWB NearbyInteraction per identificare la presenza fisica dell'utente.
@MainActor
public final class RoomPresenceBeaconManager: NSObject, ObservableObject {
    public static let shared = RoomPresenceBeaconManager()

    @Published public private(set) var activeRoomId: String? = nil
    @Published public private(set) var activeRoomName: String? = nil
    @Published public private(set) var currentDistanceMeters: Double? = nil
    @Published public private(set) var configuredBeacons: [DomusRoomBeacon] = []
    @Published public var isScanning: Bool = false

    public var onRoomChanged: ((String?) -> Void)?

    private var locationManager: CLLocationManager?

    public init(configuredBeacons: [DomusRoomBeacon] = []) {
        super.init()
        if configuredBeacons.isEmpty {
            self.configuredBeacons = [
                DomusRoomBeacon(id: "b1", roomId: "living_room", roomName: "Salotto", major: 1, minor: 1, rssiThreshold: -65),
                DomusRoomBeacon(id: "b2", roomId: "kitchen", roomName: "Cucina", major: 1, minor: 2, rssiThreshold: -65),
                DomusRoomBeacon(id: "b3", roomId: "bedroom", roomName: "Camera da Letto", major: 1, minor: 3, rssiThreshold: -68)
            ]
        } else {
            self.configuredBeacons = configuredBeacons
        }
    }

    /// Avvia il monitoraggio dei beacon indoor
    public func startMonitoring() {
        self.isScanning = true
        #if os(iOS)
        if locationManager == nil {
            locationManager = CLLocationManager()
            locationManager?.delegate = self
        }
        locationManager?.requestWhenInUseAuthorization()
        #endif
    }

    /// Arresta il monitoraggio dei beacon
    public func stopMonitoring() {
        self.isScanning = false
        self.activeRoomId = nil
        self.activeRoomName = nil
        self.currentDistanceMeters = nil
    }

    /// Aggiorna manualmente o via simulazione la stanza attiva (usato anche nei test unitari e simulatore)
    public func simulateRoomProximity(roomId: String?, distanceMeters: Double? = 0.8) {
        let matched = configuredBeacons.first(where: { $0.roomId == roomId })
        self.activeRoomId = roomId
        self.activeRoomName = matched?.roomName
        self.currentDistanceMeters = distanceMeters
        onRoomChanged?(roomId)
    }

    /// Valuta il pacchetto RSSI rilevato e aggiorna la stanza corrente se supera la soglia
    public func evaluateRssiSample(major: Int, minor: Int, rssi: Int) {
        guard let beacon = configuredBeacons.first(where: { $0.major == major && $0.minor == minor }) else {
            return
        }

        if rssi >= beacon.rssiThreshold {
            if activeRoomId != beacon.roomId {
                self.activeRoomId = beacon.roomId
                self.activeRoomName = beacon.roomName
                // Stima approssimativa distanza in metri basata su RSSI a 1m (-59 dBm)
                let ratio = Double(-59 - rssi) / (10.0 * 2.0)
                self.currentDistanceMeters = max(0.2, pow(10.0, ratio))
                onRoomChanged?(beacon.roomId)
            }
        }
    }
}

extension RoomPresenceBeaconManager: CLLocationManagerDelegate {
    nonisolated public func locationManager(_ manager: CLLocationManager, didChangeAuthorization status: CLAuthorizationStatus) {
        // Callback autorizzazione
    }
}
