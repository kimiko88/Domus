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

        let savedLocal = UserDefaults.standard.string(forKey: "domus_local_endpoint")
            ?? UserDefaults.standard.string(forKey: "domus_server_url")
        let savedRemote = UserDefaults.standard.string(forKey: "domus_remote_endpoint")
        self.localAddress = savedLocal
        self.remoteAddress = savedRemote
    }

    // MARK: - ConnectionRepositoryProtocol

    public func getCurrentConnection() async -> ServerConnection {
        connection
    }

    public func discoverLocalHub(timeoutSeconds: Double) async -> String? {
        let found = await mdnsEngine.discoverLocalHub(timeoutSeconds: timeoutSeconds)
        if let found {
            self.localAddress = found.httpBaseUrl
            UserDefaults.standard.set(found.httpBaseUrl, forKey: "domus_local_endpoint")
            UserDefaults.standard.set(found.httpBaseUrl, forKey: "domus_server_url")
        }
        return found?.httpBaseUrl
    }

    public func connect(mode: TransportMode?) async throws {
        connection.mode = .connecting
        connection.targetMode = mode ?? .localLan

        let token = await tokenStorage.getToken()

        // 1. Se richiesto esplicitamente TLS/remoto
        if mode == .remoteTls {
            guard let remote = remoteAddress ?? UserDefaults.standard.string(forKey: "domus_remote_endpoint"), !remote.isEmpty else {
                connection.mode = .disconnected
                connection.isConnected = false
                throw DomusNetworkError.notConnected
            }
            do {
                let wsUrl = try toWebSocketUrl(httpUrl: remote, requireTLS: true)
                try await openWebSocket(url: wsUrl, token: token)
            } catch {
                connection.mode = .disconnected
                connection.isConnected = false
                throw error
            }
            connection.mode = .remoteTls
            connection.remoteAddress = remote
            connection.activeUrl = remote
            connection.isConnected = true
            await DomusHttpClient.shared.configure(baseURL: remote, token: token)
            return
        }

        // 2. Se richiesto locale o automatico (preferLocal / fuori casa)
        var localCandidate = localAddress ?? UserDefaults.standard.string(forKey: "domus_local_endpoint")
        if localCandidate == nil {
            if let discovered = await discoverLocalHub(timeoutSeconds: 1.5) {
                localCandidate = discovered
                self.localAddress = discovered
                UserDefaults.standard.set(discovered, forKey: "domus_local_endpoint")
                UserDefaults.standard.set(discovered, forKey: "domus_server_url")
            }
        }

        if let local = localCandidate, !local.isEmpty {
            do {
                let wsUrl = toWebSocketUrl(httpUrl: local)
                try await openWebSocket(url: wsUrl, token: token)
                connection.mode = .localLan
                connection.localAddress = local
                connection.activeUrl = local
                connection.isConnected = true
                await DomusHttpClient.shared.configure(baseURL: local, token: token)
                return
            } catch {
                if mode == .localLan {
                    connection.mode = .disconnected
                    connection.isConnected = false
                    throw error
                }
            }
        }

        // 3. Fallback automatico su endpoint remoto se fuori casa
        if let remote = remoteAddress ?? UserDefaults.standard.string(forKey: "domus_remote_endpoint"), !remote.isEmpty {
            do {
                let wsUrl = try toWebSocketUrl(httpUrl: remote, requireTLS: true)
                try await openWebSocket(url: wsUrl, token: token)
            } catch {
                connection.mode = .disconnected
                connection.isConnected = false
                throw error
            }
            connection.mode = .remoteTls
            connection.remoteAddress = remote
            connection.activeUrl = remote
            connection.isConnected = true
            await DomusHttpClient.shared.configure(baseURL: remote, token: token)
            return
        }

        connection.mode = .disconnected
        connection.isConnected = false
        throw DomusNetworkError.notConnected
    }

    public func disconnect() async {
        await webSocketClient.disconnect()
        connection.mode = .disconnected
        connection.isConnected = false
        connection.activeUrl = nil
    }

    public func setEndpoints(local: String?, remote: String?) async {
        if let local {
            self.localAddress = local
            UserDefaults.standard.set(local, forKey: "domus_local_endpoint")
            UserDefaults.standard.set(local, forKey: "domus_server_url")
        }
        if let remote {
            self.remoteAddress = remote
            UserDefaults.standard.set(remote, forKey: "domus_remote_endpoint")
        }
    }

    // MARK: - Private

    private func toWebSocketUrl(httpUrl: String, requireTLS: Bool = false) throws -> String {
        let trimmed = httpUrl.trimmingCharacters(in: .whitespacesAndNewlines)
        guard var components = URLComponents(string: trimmed),
              let scheme = components.scheme?.lowercased(),
              components.host != nil,
              components.user == nil,
              components.password == nil,
              components.query == nil,
              components.fragment == nil else {
            throw DomusNetworkError.invalidURL(httpUrl)
        }

        switch scheme {
        case "https":
            components.scheme = "wss"
        case "http":
            guard !requireTLS else {
                throw DomusNetworkError.invalidURL("L'endpoint remoto deve usare HTTPS")
            }
            components.scheme = "ws"
        default:
            throw DomusNetworkError.invalidURL(httpUrl)
        }

        let basePath = components.path.hasSuffix("/")
            ? String(components.path.dropLast())
            : components.path
        components.path = basePath + "/ws"
        guard let url = components.url else {
            throw DomusNetworkError.invalidURL(httpUrl)
        }
        return url.absoluteString
    }

    private func openWebSocket(url: String, token: String?) async throws {
        try await webSocketClient.connect(to: url, token: token)
    }
}
