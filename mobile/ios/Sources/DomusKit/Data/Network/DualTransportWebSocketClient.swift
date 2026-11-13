import Foundation

/// Client WebSocket bidirezionale con supporto dual-transport (LAN locale / Remoto TLS).
/// Usa `URLSessionWebSocketTask` con riconnessione automatica ad esponenziale backoff.
public actor DualTransportWebSocketClient: Sendable {

    // MARK: - State

    private var webSocketTask: URLSessionWebSocketTask?
    private var session: URLSession?
    private var currentURL: URL?
    private var isConnected: Bool = false
    private var reconnectAttempts: Int = 0
    private var authToken: String?
    private let maxReconnectAttempts: Int = 5
    private var eventHandlers: [(String) -> Void] = []

    public init() {}

    // MARK: - Public API

    /// Connette al WebSocket all'URL specificato.
    public func connect(to urlString: String) async throws {
        guard let url = URL(string: urlString) else {
            throw DomusNetworkError.invalidURL(urlString)
        }
        self.currentURL = url
        self.authToken = token
        self.reconnectAttempts = 0
        try await openConnection(url: url, token: token)
    }

    /// Disconnette il WebSocket.
    public func disconnect() {
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        webSocketTask = nil
        isConnected = false
        reconnectAttempts = 0
    }

    /// Invia un messaggio JSON al server.
    public func send<T: Encodable>(_ message: T) async throws {
        guard let task = webSocketTask, isConnected else {
            throw DomusNetworkError.notConnected
        }
        let data = try JSONEncoder().encode(message)
        let jsonString = String(data: data, encoding: .utf8) ?? "{}"
        try await task.send(.string(jsonString))
    }

    /// Aggiunge un handler chiamato ogni volta che arriva un messaggio dal server.
    public func addMessageHandler(_ handler: @escaping (String) -> Void) {
        eventHandlers.append(handler)
    }

    public var connected: Bool { isConnected }

    // MARK: - Private

    private func openConnection(url: URL, token: String? = nil) async throws {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 15
        let urlSession = URLSession(configuration: config)
        self.session = urlSession

        var request = URLRequest(url: url)
        request.timeoutInterval = 15
        if let token, !token.isEmpty { request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }

        let task = urlSession.webSocketTask(with: request)
        self.webSocketTask = task
        task.resume()
        self.isConnected = true
        self.reconnectAttempts = 0

        // Avvia il loop di ricezione messaggi
        receiveMessage()
    }

    private func receiveMessage() {
        webSocketTask?.receive { [weak self] result in
            guard let self else { return }
            Task {
                switch result {
                case .success(let message):
                    switch message {
                    case .string(let text):
                        await self.dispatchMessage(text)
                    case .data(let data):
                        if let text = String(data: data, encoding: .utf8) {
                            await self.dispatchMessage(text)
                        }
                    @unknown default:
                        break
                    }
                    await self.receiveMessage() // ricevi il prossimo messaggio

                case .failure:
                    await self.handleDisconnect()
                }
            }
        }
    }

    private func dispatchMessage(_ text: String) {
        for handler in eventHandlers {
            handler(text)
        }
    }

    private func handleDisconnect() async {
        isConnected = false
        guard reconnectAttempts < maxReconnectAttempts, let url = currentURL else { return }

        reconnectAttempts += 1
        let delay = min(pow(2.0, Double(reconnectAttempts)), 30.0) // backoff esponenziale, max 30s
        try? await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))

        try? await openConnection(url: url, token: authToken)
    }
}
