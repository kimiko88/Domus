import Foundation

/// Implementazione concreta del repository di connessione.
/// Gestisce il dual-transport: LAN locale (mDNS) con fallback remoto TLS.
public actor DefaultConnectionRepository: ConnectionRepositoryProtocol {

    private let mdnsEngine: MdnsDiscoveryEngine
    private let webSocketClient: DualTransportWebSocketClient
    private let tokenStorage: KeychainManager

    private var connection: ServerConnection = ServerConnection()
    private var localAddress: String?
    private var remoteAddress: String?

    public init(
        mdnsEngine: MdnsDiscoveryEngine,
        webSocketClient: DualTransportWebSocketClient,
        tokenStorage: KeychainManager
    ) {
        self.mdnsEngine = mdnsEngine
        self.webSocketClient = webSocketClient
        self.tokenStorage = tokenStorage
    }

    // MARK: - ConnectionRepositoryProtocol

    public func getCurrentConnection() async -> ServerConnection {
        connection
    }

    public func discoverLocalHub(timeoutSeconds: Double) async -> String? {
        let found = await mdnsEngine.discoverLocalHub(timeoutSeconds: timeoutSeconds)
        if let found {
            self.localAddress = found.httpBaseUrl
        }
        return found?.httpBaseUrl
    }

    public func connect(mode: TransportMode?) async throws {
        let targetMode = mode ?? .localLan

        connection.mode = .connecting
        connection.targetMode = targetMode

        let token = await tokenStorage.getToken()

        switch targetMode {
        case .localLan:
            guard let local = localAddress else {
                // Prova prima la discovery
                if let discovered = await discoverLocalHub(timeoutSeconds: 1.5) {
                    let wsURL = discovered.replacingOccurrences(of: "http://", with: "ws://") + "/api/v1/ws"
                    try await openWebSocket(url: wsURL, token: token)
                    connection.mode = .localLan
                    connection.localAddress = discovered
                    connection.activeUrl = wsURL
                    connection.isConnected = true
                    return
                }
                throw DomusNetworkError.notConnected
            }
            let wsURL = local.replacingOccurrences(of: "http://", with: "ws://") + "/api/v1/ws"
            try await openWebSocket(url: wsURL, token: token)
            connection.mode = .localLan
            connection.localAddress = local
            connection.activeUrl = wsURL
            connection.isConnected = true

        case .remoteTls:
            guard let remote = remoteAddress else {
                throw DomusNetworkError.notConnected
            }
            let wsURL = remote + "/api/v1/ws"
            try await openWebSocket(url: wsURL, token: token)
            connection.mode = .remoteTls
            connection.remoteAddress = remote
            connection.activeUrl = wsURL
            connection.isConnected = true

        default:
            throw DomusNetworkError.notConnected
        }
    }

    public func disconnect() async {
        await webSocketClient.disconnect()
        connection.mode = .disconnected
        connection.isConnected = false
        connection.activeUrl = nil
    }

    public func setEndpoints(local: String?, remote: String?) async {
        if let local { self.localAddress = local }
        if let remote { self.remoteAddress = remote }
    }

    // MARK: - Private

    private func openWebSocket(url: String, token: String?) async throws {
        // Il token viene aggiunto come query parameter o header nel WebSocket handshake
        var urlString = url
        if let token {
            urlString += "?token=\(token)"
        }
        try await webSocketClient.connect(to: urlString)
    }
}
