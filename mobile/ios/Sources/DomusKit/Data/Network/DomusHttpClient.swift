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

    // MARK: - Auth

    private struct LoginRequestDTO: Encodable {
        let username: String
        let password: String
    }

    private struct UserResponseDTO: Decodable {
        let username: String?
        let role: String?
    }

    private struct LoginResponseDTO: Decodable {
        let token: String
        let user: UserResponseDTO?
    }

    public func login(baseUrl: String, username: String, password: String) async throws -> AuthSession {
        self.baseURL = baseUrl
        let res = try await post(
            path: "/api/v1/auth/login",
            body: LoginRequestDTO(username: username, password: password),
            as: LoginResponseDTO.self
        )
        self.token = res.token
        return AuthSession(
            token: res.token,
            username: res.user?.username ?? username,
            role: res.user?.role ?? "admin",
            isAuthenticated: true
        )
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
