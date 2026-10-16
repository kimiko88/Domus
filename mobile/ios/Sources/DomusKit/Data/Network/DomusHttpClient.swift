import Foundation

/// Client HTTP semplice per le chiamate REST all'API di Domus.
/// Usa `URLSession` nativa.
public actor DomusHttpClient: Sendable {

    public static let shared = DomusHttpClient()

    private var baseURL: String = ""
    private var token: String?
    private let session: URLSession

    public init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 10
        config.timeoutIntervalForResource = 30
        self.session = URLSession(configuration: config)
        if let savedBase = UserDefaults.standard.string(forKey: "domus_server_url"), !savedBase.isEmpty {
            self.baseURL = savedBase
        }
        if let savedToken = UserDefaults.standard.string(forKey: "domus_auth_token"), !savedToken.isEmpty {
            self.token = savedToken
        }
    }

    public func configure(baseURL: String, token: String?) {
        self.baseURL = baseURL
        self.token = token
        UserDefaults.standard.set(baseURL, forKey: "domus_server_url")
        if let token {
            UserDefaults.standard.set(token, forKey: "domus_auth_token")
        }
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

    /// DELETE generico
    public func delete(path: String) async throws {
        let url = try makeURL(path: path)
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        addAuth(to: &request)

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
        UserDefaults.standard.set(baseUrl, forKey: "domus_server_url")
        let res = try await post(
            path: "/api/v1/auth/login",
            body: LoginRequestDTO(username: username, password: password),
            as: LoginResponseDTO.self
        )
        self.token = res.token
        UserDefaults.standard.set(res.token, forKey: "domus_auth_token")
        return AuthSession(
            token: res.token,
            username: res.user?.username ?? username,
            role: res.user?.role ?? "admin",
            isAuthenticated: true
        )
    }

    // MARK: - Typed Endpoints for Companion Modules

    public struct EmptyBodyDTO: Encodable, Sendable {
        public init() {}
    }

    public func sendAssistantPrompt(prompt: String) async throws -> AssistantChatResponseDTO {
        try await post(
            path: "/api/v1/assistant/chat",
            body: AssistantChatRequestDTO(prompt: prompt),
            as: AssistantChatResponseDTO.self
        )
    }

    public func fetchChores() async throws -> [ChoreDTO] {
        try await get(path: "/api/v1/chores", as: [ChoreDTO].self)
    }

    public func createChore(title: String, areaName: String, assignedTo: String, priority: String, karmaPoints: Int) async throws -> ChoreDTO {
        let body = CreateChoreDTO(title: title, areaName: areaName, assignedTo: assignedTo, priority: priority, karmaPoints: karmaPoints)
        return try await post(path: "/api/v1/chores", body: body, as: ChoreDTO.self)
    }

    public func completeChore(id: String) async throws {
        try await postVoid(path: "/api/v1/chores/\(id)/complete", body: EmptyBodyDTO())
    }

    public func deleteChore(id: String) async throws {
        try await delete(path: "/api/v1/chores/\(id)")
    }

    public func fetchShopping() async throws -> [ShoppingItemDTO] {
        try await get(path: "/api/v1/shopping", as: [ShoppingItemDTO].self)
    }

    public func createShopping(title: String, quantity: String, category: String) async throws -> ShoppingItemDTO {
        let body = CreateShoppingDTO(title: title, quantity: quantity, category: category)
        return try await post(path: "/api/v1/shopping", body: body, as: ShoppingItemDTO.self)
    }

    public func toggleShopping(id: String) async throws -> ShoppingItemDTO {
        try await post(path: "/api/v1/shopping/\(id)/toggle", body: EmptyBodyDTO(), as: ShoppingItemDTO.self)
    }

    public func deleteShopping(id: String) async throws {
        try await delete(path: "/api/v1/shopping/\(id)")
    }

    public func clearCompletedShopping() async throws {
        try await postVoid(path: "/api/v1/shopping/clear-completed", body: EmptyBodyDTO())
    }

    public func fetchEnergyFlow() async throws -> EnergyFlowDTO {
        try await get(path: "/api/v1/energy/flow", as: EnergyFlowDTO.self)
    }

    public func fetchNotifications() async throws -> [InAppNotificationDTO] {
        try await get(path: "/api/v1/notifications", as: [InAppNotificationDTO].self)
    }

    public func toggleEntity(id: String, domain: String = "") async throws {
        struct ToggleBody: Encodable {
            let entityId: String
            let domain: String
            enum CodingKeys: String, CodingKey {
                case entityId = "entity_id"
                case domain
            }
        }
        try await postVoid(path: "/api/v1/entities/\(id)/toggle", body: ToggleBody(entityId: id, domain: domain))
    }

    public func callService(domain: String, service: String, entityId: String, data: [String: String] = [:]) async throws {
        struct ServiceBody: Encodable {
            let entityId: String
            let data: [String: String]
            enum CodingKeys: String, CodingKey {
                case entityId = "entity_id"
                case data
            }
        }
        try await postVoid(path: "/api/v1/services/\(domain)/\(service)", body: ServiceBody(entityId: entityId, data: data))
    }

    // MARK: - Private helpers

    private func makeURL(path: String) throws -> URL {
        let base = !baseURL.isEmpty ? baseURL : (UserDefaults.standard.string(forKey: "domus_server_url") ?? "http://127.0.0.1:8080")
        guard let url = URL(string: base + path) else {
            throw DomusNetworkError.invalidURL(base + path)
        }
        return url
    }

    private func addAuth(to request: inout URLRequest) {
        let effectiveToken = token ?? UserDefaults.standard.string(forKey: "domus_auth_token")
        if let effectiveToken, !effectiveToken.isEmpty {
            request.setValue("Bearer \(effectiveToken)", forHTTPHeaderField: "Authorization")
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

// MARK: - Companion DTOs

public struct AssistantChatRequestDTO: Encodable, Sendable {
    public let prompt: String
    public init(prompt: String) { self.prompt = prompt }
}

public struct AssistantChatResponseDTO: Decodable, Sendable {
    public let responseText: String
    public let actionsExecuted: [String]?

    enum CodingKeys: String, CodingKey {
        case responseText = "response_text"
        case actionsExecuted = "actions_executed"
    }

    public init(responseText: String, actionsExecuted: [String]? = nil) {
        self.responseText = responseText
        self.actionsExecuted = actionsExecuted
    }
}

public struct ChoreDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let title: String
    public let areaName: String
    public let assignedTo: String
    public let priority: String
    public let completed: Bool
    public let karmaPoints: Int

    enum CodingKeys: String, CodingKey {
        case id
        case title
        case areaName = "area_name"
        case assignedTo = "assigned_to"
        case priority
        case completed
        case karmaPoints = "karma_points"
    }

    public init(id: String, title: String, areaName: String, assignedTo: String, priority: String, completed: Bool, karmaPoints: Int) {
        self.id = id
        self.title = title
        self.areaName = areaName
        self.assignedTo = assignedTo
        self.priority = priority
        self.completed = completed
        self.karmaPoints = karmaPoints
    }
}

public struct CreateChoreDTO: Encodable, Sendable {
    public let title: String
    public let areaName: String
    public let assignedTo: String
    public let priority: String
    public let karmaPoints: Int

    enum CodingKeys: String, CodingKey {
        case title
        case areaName = "area_name"
        case assignedTo = "assigned_to"
        case priority
        case karmaPoints = "karma_points"
    }
}

public struct ShoppingItemDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let title: String
    public let quantity: String
    public let category: String
    public let completed: Bool
    public let completedBy: String?
    public let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case title
        case quantity
        case category
        case completed
        case completedBy = "completed_by"
        case createdAt = "created_at"
    }

    public init(id: String, title: String, quantity: String, category: String, completed: Bool, completedBy: String? = nil, createdAt: String? = nil) {
        self.id = id
        self.title = title
        self.quantity = quantity
        self.category = category
        self.completed = completed
        self.completedBy = completedBy
        self.createdAt = createdAt
    }
}

public struct CreateShoppingDTO: Encodable, Sendable {
    public let title: String
    public let quantity: String
    public let category: String

    public init(title: String, quantity: String, category: String) {
        self.title = title
        self.quantity = quantity
        self.category = category
    }
}

public struct EnergyFlowDTO: Codable, Sendable {
    public let solarWatts: Double
    public let gridWatts: Double
    public let batteryWatts: Double
    public let batterySoc: Double
    public let houseConsumptionWatts: Double

    enum CodingKeys: String, CodingKey {
        case solarWatts = "solar_watts"
        case gridWatts = "grid_watts"
        case batteryWatts = "battery_watts"
        case batterySoc = "battery_soc"
        case houseConsumptionWatts = "house_consumption_watts"
    }

    public init(solarWatts: Double, gridWatts: Double, batteryWatts: Double, batterySoc: Double, houseConsumptionWatts: Double) {
        self.solarWatts = solarWatts
        self.gridWatts = gridWatts
        self.batteryWatts = batteryWatts
        self.batterySoc = batterySoc
        self.houseConsumptionWatts = houseConsumptionWatts
    }
}

public struct InAppNotificationDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let title: String
    public let message: String
    public let category: String
    public let createdAt: String
    public let action: String?

    enum CodingKeys: String, CodingKey {
        case id
        case title
        case message
        case category
        case createdAt = "created_at"
        case action
    }

    public init(id: String, title: String, message: String, category: String, createdAt: String, action: String? = nil) {
        self.id = id
        self.title = title
        self.message = message
        self.category = category
        self.createdAt = createdAt
        self.action = action
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
