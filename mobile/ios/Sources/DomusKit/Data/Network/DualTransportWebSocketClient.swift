import Foundation

/// Client WebSocket bidirezionale con supporto dual-transport (LAN locale / Remoto TLS).
/// La connessione è considerata attiva solo dopo il completamento dell'handshake ping.
public actor DualTransportWebSocketClient: Sendable {

    private var webSocketTask: URLSessionWebSocketTask?
    private var session: URLSession?
    private var currentURL: URL?
    private var isConnected: Bool = false
    private var reconnectAttempts: Int = 0
    private var authToken: String?
    private var shouldReconnect: Bool = false
    private var connectionGeneration: UInt64 = 0
    private let maxReconnectAttempts: Int = 5
    private var eventHandlers: [(String) -> Void] = []

    public init() {}

    /// Connette al WebSocket all'URL specificato con autenticazione Bearer opzionale.
    public func connect(to urlString: String, token: String? = nil) async throws {
        disconnect()
        guard let url = URL(string: urlString),
              let scheme = url.scheme?.lowercased(),
              scheme == "ws" || scheme == "wss",
              url.host != nil,
              url.user == nil,
              url.password == nil,
              url.query == nil,
              url.fragment == nil else {
            throw DomusNetworkError.invalidURL(urlString)
        }

        let generation = connectionGeneration
        currentURL = url
        authToken = token
        shouldReconnect = true
        reconnectAttempts = 0
        do {
            try await openConnection(url: url, token: token, generation: generation)
        } catch {
            // Un tentativo più vecchio non deve invalidare una connessione più recente.
            if generation == connectionGeneration {
                shouldReconnect = false
                currentURL = nil
                authToken = nil
                webSocketTask?.cancel(with: .goingAway, reason: nil)
                webSocketTask = nil
                session?.invalidateAndCancel()
                session = nil
                isConnected = false
            }
            throw error
        }
    }

    /// Disconnette il WebSocket e impedisce qualsiasi riconnessione automatica.
    public func disconnect() {
        // Invalida anche callback e tentativi di connessione già sospesi.
        connectionGeneration &+= 1
        shouldReconnect = false
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        webSocketTask = nil
        session?.invalidateAndCancel()
        session = nil
        currentURL = nil
        authToken = nil
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

    private func openConnection(url: URL, token: String?, generation: UInt64) async throws {
        guard generation == connectionGeneration, shouldReconnect else {
            throw DomusNetworkError.notConnected
        }

        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 15
        let urlSession = URLSession(configuration: config)
        session = urlSession

        var request = URLRequest(url: url)
        request.timeoutInterval = 15
        if let token, !token.isEmpty {
            request.setValue("Bearer \\(token)", forHTTPHeaderField: "Authorization")
        }

        let task = urlSession.webSocketTask(with: request)
        webSocketTask = task
        task.resume()

        do {
            // URLSessionWebSocketTask.resume() non prova che l'handshake sia riuscito.
            // Un ping completato conferma che il server ha accettato la connessione.
            try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
                task.sendPing { error in
                    if let error {
                        continuation.resume(throwing: error)
                    } else {
                        continuation.resume(returning: ())
                    }
                }
            }
        } catch {
            if generation == connectionGeneration, webSocketTask === task {
                webSocketTask = nil
                isConnected = false
                if session === urlSession {
                    session = nil
                }
            }
            urlSession.invalidateAndCancel()
            throw error
        }

        guard generation == connectionGeneration,
              shouldReconnect,
              webSocketTask === task else {
            task.cancel(with: .goingAway, reason: nil)
            throw DomusNetworkError.notConnected
        }
        isConnected = true
        reconnectAttempts = 0
        receiveMessage(from: task, generation: generation)
    }

    private func receiveMessage(from task: URLSessionWebSocketTask, generation: UInt64) {
        guard generation == connectionGeneration, webSocketTask === task else { return }
        task.receive { [weak self] result in
            guard let self else { return }
            Task {
                switch result {
                case .success(let message):
                    switch message {
                    case .string(let text):
                        await self.dispatchMessage(text, from: task, generation: generation)
                    case .data(let data):
                        if let text = String(data: data, encoding: .utf8) {
                            await self.dispatchMessage(text, from: task, generation: generation)
                        }
                    @unknown default:
                        break
                    }
                    await self.receiveMessage(from: task, generation: generation)

                case .failure:
                    await self.handleDisconnect(from: task, generation: generation)
                }
            }
        }
    }

    private func dispatchMessage(
        _ text: String,
        from task: URLSessionWebSocketTask,
        generation: UInt64
    ) {
        guard generation == connectionGeneration, webSocketTask === task else { return }
        for handler in eventHandlers {
            handler(text)
        }
    }

    private func handleDisconnect(from task: URLSessionWebSocketTask, generation: UInt64) async {
        guard generation == connectionGeneration, webSocketTask === task else { return }
        webSocketTask = nil
        isConnected = false
        await scheduleReconnect(generation: generation)
    }

    private func scheduleReconnect(generation: UInt64) async {
        guard generation == connectionGeneration,
              shouldReconnect,
              let url = currentURL,
              reconnectAttempts < maxReconnectAttempts else { return }

        reconnectAttempts += 1
        let delay = min(pow(2.0, Double(reconnectAttempts)), 30.0)
        try? await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))

        guard generation == connectionGeneration,
              shouldReconnect,
              currentURL == url,
              webSocketTask == nil else { return }
        do {
            try await openConnection(url: url, token: authToken, generation: generation)
        } catch {
            guard generation == connectionGeneration, shouldReconnect else { return }
            await scheduleReconnect(generation: generation)
        }
    }
}
