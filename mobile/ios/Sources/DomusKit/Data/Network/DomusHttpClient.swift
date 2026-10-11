import Foundation

/// Client HTTP semplice per le chiamate REST all'API di Domus.
/// Usa `URLSession` nativa.
public actor DomusHttpClient: Sendable {

    private var baseURL: String = ""
    private var token: String?
    private let session: URLSession

    public init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 10
        config.timeoutIntervalForResource = 30
        self.session = URLSession(configuration: config)
    }

    public func configure(baseURL: String, token: String?) {
        self.baseURL = baseURL
        self.token = token
    }

    /// GET generico
    public func get<T: Decodable>(path: String, as type: T.Type) async throws -> T {
        let url = try makeURL(path: path)
        var request = URLRequest(url: url)
        addAuth(to: &request)

        let (data, response) = try await session.data(for: request)
        try validateResponse(response)
        return try JSONDecoder().decode(T.self, from: data)
    }

    /// POST generico
    public func post<B: Encodable, R: Decodable>(path: String, body: B, as type: R.Type) async throws -> R {
        let url = try makeURL(path: path)
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        addAuth(to: &request)
        request.httpBody = try JSONEncoder().encode(body)

        let (data, response) = try await session.data(for: request)
        try validateResponse(response)
        return try JSONDecoder().decode(R.self, from: data)
    }

    /// POST senza corpo di risposta
    public func postVoid<B: Encodable>(path: String, body: B) async throws {
        let url = try makeURL(path: path)
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        addAuth(to: &request)
        request.httpBody = try JSONEncoder().encode(body)

        let (_, response) = try await session.data(for: request)
        try validateResponse(response)
    }

    // MARK: - Private helpers

    private func makeURL(path: String) throws -> URL {
        guard !baseURL.isEmpty, let url = URL(string: baseURL + path) else {
            throw DomusNetworkError.invalidURL(baseURL + path)
        }
        return url
    }

    private func addAuth(to request: inout URLRequest) {
        if let token {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
    }

    private func validateResponse(_ response: URLResponse) throws {
        guard let httpResponse = response as? HTTPURLResponse else {
            throw DomusNetworkError.invalidResponse
        }
        guard (200..<300).contains(httpResponse.statusCode) else {
            throw DomusNetworkError.httpError(httpResponse.statusCode)
        }
    }
}

public enum DomusNetworkError: Error, LocalizedError {
    case invalidURL(String)
    case invalidResponse
    case httpError(Int)
    case notConnected

    public var errorDescription: String? {
        switch self {
        case .invalidURL(let url): return "URL non valido: \(url)"
        case .invalidResponse: return "Risposta HTTP non valida"
        case .httpError(let code): return "Errore HTTP \(code)"
        case .notConnected: return "Non connesso al server Domus"
        }
    }
}
