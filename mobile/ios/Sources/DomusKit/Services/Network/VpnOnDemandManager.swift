import Foundation
import Combine

#if canImport(NetworkExtension)
import NetworkExtension
#endif

/// Stato di connessione della VPN domestica.
public enum DomusVpnStatus: String, Sendable {
    case disconnected = "Disconnesso"
    case connecting = "Connessione..."
    case connected = "Protetto (VPN Attiva)"
    case disconnecting = "Disconnessione..."
    case invalid = "Non configurato"
}

/// Gestore per la VPN On-Demand automatica (NetworkExtension / WireGuardKit).
/// Attiva automaticamente il tunnel crittografato end-to-end non appena lo smartphone
/// esce dalla rete Wi-Fi di casa, senza esporre porte pubbliche su internet.
@MainActor
public final class VpnOnDemandManager: ObservableObject {
    public static let shared = VpnOnDemandManager()

    @Published public private(set) var status: DomusVpnStatus = .disconnected
    @Published public private(set) var homeSsid: String = "Casa_5GHz"
    @Published public private(set) var isOnDemandEnabled: Bool = true
    @Published public private(set) var lastErrorMessage: String? = nil

    #if canImport(NetworkExtension)
    private var vpnManager: NEVPNManager?
    #endif

    public init(homeSsid: String = "Casa_5GHz", isOnDemandEnabled: Bool = true) {
        self.homeSsid = homeSsid
        self.isOnDemandEnabled = isOnDemandEnabled
        setupVpnManager()
    }

    private func setupVpnManager() {
        #if canImport(NetworkExtension)
        let manager = NEVPNManager.shared()
        self.vpnManager = manager
        manager.loadFromPreferences { [weak self] error in
            Task { @MainActor [weak self] in
                guard let self = self else { return }
                if let error = error {
                    self.lastErrorMessage = error.localizedDescription
                } else if let vpn = self.vpnManager {
                    self.updateStatusFromManager(vpn)
                }
            }
        }
        #else
        self.status = .disconnected
        #endif
    }

    /// Configura le regole On-Demand: disconnette su Wi-Fi di casa, connette su reti esterne o 4G/5G.
    public func configureOnDemandRules(homeSsid: String) {
        self.homeSsid = homeSsid

        #if canImport(NetworkExtension)
        guard let manager = vpnManager else { return }

        // Regola 1: Disconnetti se connesso all'SSID Wi-Fi di casa
        let disconnectOnHomeWifi = NEOnDemandRuleDisconnect()
        disconnectOnHomeWifi.interfaceTypeMatch = .wiFi
        disconnectOnHomeWifi.ssidMatch = [homeSsid]

        // Regola 2: Connetti su qualsiasi altra interfaccia (Cellulare o altri Wi-Fi)
        let connectOnExternal = NEOnDemandRuleConnect()
        connectOnExternal.interfaceTypeMatch = .any

        manager.onDemandRules = [disconnectOnHomeWifi, connectOnExternal]
        manager.isOnDemandEnabled = self.isOnDemandEnabled

        manager.saveToPreferences { [weak self] error in
            Task { @MainActor [weak self] in
                if let error = error {
                    self?.lastErrorMessage = error.localizedDescription
                }
            }
        }
        #endif
    }

    /// Connessione manuale forzata al tunnel VPN
    public func connect() {
        #if canImport(NetworkExtension)
        do {
            try vpnManager?.connection.startVPNTunnel()
            self.status = .connecting
        } catch {
            self.lastErrorMessage = error.localizedDescription
        }
        #else
        self.status = .connected
        #endif
    }

    /// Disconnessione manuale dal tunnel VPN
    public func disconnect() {
        #if canImport(NetworkExtension)
        vpnManager?.connection.stopVPNTunnel()
        self.status = .disconnected
        #else
        self.status = .disconnected
        #endif
    }

    #if canImport(NetworkExtension)
    private func updateStatusFromManager(_ manager: NEVPNManager) {
        switch manager.connection.status {
        case .connected:
            self.status = .connected
        case .connecting:
            self.status = .connecting
        case .disconnecting:
            self.status = .disconnecting
        case .disconnected:
            self.status = .disconnected
        case .invalid:
            self.status = .invalid
        case .reasserting:
            self.status = .connecting
        @unknown default:
            self.status = .disconnected
        }
    }
    #endif
}
