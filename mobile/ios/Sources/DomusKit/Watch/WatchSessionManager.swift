import Foundation
#if canImport(WatchConnectivity)
import WatchConnectivity
#endif

/// Modello sintetico per la complicazione Apple Watch (quadrante).
public struct WatchComplicationSnapshot: Codable, Sendable {
    public let indoorTemperature: Double
    public let instantPowerKW: Double
    public let isAlarmArmed: Bool
    public let updatedDate: Date

    public init(indoorTemperature: Double = 21.5, instantPowerKW: Double = 0.45, isAlarmArmed: Bool = true, updatedDate: Date = Date()) {
        self.indoorTemperature = indoorTemperature
        self.instantPowerKW = instantPowerKW
        self.isAlarmArmed = isAlarmArmed
        self.updatedDate = updatedDate
    }
}

/// Gestore della connessione WatchConnectivity tra iPhone e Apple Watch — Fase 4.
@MainActor
public final class WatchSessionManager: NSObject, ObservableObject {
    @Published public var isWatchPaired: Bool = false
    @Published public var isWatchAppInstalled: Bool = false
    @Published public var lastReceivedCommand: String? = nil

    public var onWatchCommandReceived: ((String, String) -> Void)?

    #if canImport(WatchConnectivity) && os(iOS)
    private var session: WCSession?
    #endif

    public override init() {
        super.init()
        #if canImport(WatchConnectivity) && os(iOS)
        if WCSession.isSupported() {
            let session = WCSession.default
            self.session = session
            session.delegate = self
            session.activate()
            self.isWatchPaired = session.isPaired
            self.isWatchAppInstalled = session.isWatchAppInstalled
        }
        #endif
    }

    /// Invia uno snapshot aggiornato dei dati per le complicazioni dell'orologio.
    public func updateComplications(snapshot: WatchComplicationSnapshot) {
        #if canImport(WatchConnectivity) && os(iOS)
        guard let session, session.activationState == .activated else { return }

        let dict: [String: Any] = [
            "temp": snapshot.indoorTemperature,
            "powerKW": snapshot.instantPowerKW,
            "alarmArmed": snapshot.isAlarmArmed,
            "timestamp": snapshot.updatedDate.timeIntervalSince1970
        ]

        try? session.updateApplicationContext(dict)
        #endif
    }

    /// Invia l'elenco dei dispositivi preferiti per la visualizzazione standalone sull'Apple Watch.
    public func sendFavoritesToWatch(entities: [Entity]) {
        #if canImport(WatchConnectivity) && os(iOS)
        guard let session, session.activationState == .activated else { return }

        let payload = entities.prefix(8).map { [
            "id": $0.id,
            "name": $0.name,
            "isOn": $0.isOn,
            "domain": $0.domain
        ] }

        session.sendMessage(["favorites": payload], replyHandler: nil, errorHandler: nil)
        #endif
    }
}

// MARK: - WCSessionDelegate

#if canImport(WatchConnectivity) && os(iOS)
extension WatchSessionManager: WCSessionDelegate {
    public nonisolated func session(_ session: WCSession, activationDidCompleteWith activationState: WCSessionActivationState, error: Error?) {
        Task { @MainActor in
            self.isWatchPaired = session.isPaired
            self.isWatchAppInstalled = session.isWatchAppInstalled
        }
    }

    public nonisolated func sessionDidBecomeInactive(_ session: WCSession) {}

    public nonisolated func sessionDidDeactivate(_ session: WCSession) {
        WCSession.default.activate()
    }

    public nonisolated func session(_ session: WCSession, didReceiveMessage message: [String : Any]) {
        // Riceve comando di toggle o attivazione scena dall'orologio
        if let action = message["action"] as? String, let targetId = message["targetId"] as? String {
            Task { @MainActor in
                self.lastReceivedCommand = "\(action): \(targetId)"
                self.onWatchCommandReceived?(action, targetId)
            }
        }
    }
}
#endif
