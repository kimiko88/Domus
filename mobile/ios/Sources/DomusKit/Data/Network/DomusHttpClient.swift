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

    /// PUT generico
    public func put<B: Encodable, R: Decodable>(path: String, body: B, as type: R.Type) async throws -> R {
        let url = try makeURL(path: path)
        var request = URLRequest(url: url)
        request.httpMethod = "PUT"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        addAuth(to: &request)
        request.httpBody = try JSONEncoder().encode(body)

        let (data, response) = try await session.data(for: request)
        try validateResponse(response)
        return try JSONDecoder().decode(R.self, from: data)
    }

    /// PUT senza corpo di risposta
    public func putVoid<B: Encodable>(path: String, body: B) async throws {
        let url = try makeURL(path: path)
        var request = URLRequest(url: url)
        request.httpMethod = "PUT"
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

    public func createChore(title: String, areaName: String = "", assignedTo: String = "", priority: String = "medium", karmaPoints: Int = 20, description: String = "") async throws -> ChoreDTO {
        let body = CreateChoreDTO(title: title, areaName: areaName, assignedTo: assignedTo, priority: priority, karmaPoints: karmaPoints, description: description)
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

    public func createShopping(title: String, quantity: String, category: String, addedBy: String = "iOS Companion") async throws -> ShoppingItemDTO {
        let body = CreateShoppingDTO(title: title, quantity: quantity, category: category, addedBy: addedBy)
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

    public func handleNotificationAction(notificationId: String, actionId: String, user: String = "iOS") async throws {
        struct ActionBody: Encodable {
            let notificationId: String
            let user: String
            enum CodingKeys: String, CodingKey {
                case notificationId = "notification_id"
                case user
            }
        }
        try await postVoid(path: "/api/v1/notifications/actionable/actions/\(actionId)", body: ActionBody(notificationId: notificationId, user: user))
    }

    public func applyScene(id: String) async throws {
        try await postVoid(path: "/api/v1/scenes/\(id)/apply", body: EmptyBodyDTO())
    }

    public func scanTag(id: String, scannedBy: String = "iOS_Companion") async throws {
        struct TagScanBody: Encodable {
            let scannedBy: String
            enum CodingKeys: String, CodingKey {
                case scannedBy = "scanned_by"
            }
        }
        try await postVoid(path: "/api/v1/tags/\(id)/scan", body: TagScanBody(scannedBy: scannedBy))
    }

    public func registerPushToken(deviceToken: String, deviceName: String = "Apple Device") async throws {
        struct PushTokenBody: Encodable {
            let platform: String
            let deviceToken: String
            let deviceName: String
            let appVersion: String
            enum CodingKeys: String, CodingKey {
                case platform
                case deviceToken = "device_token"
                case deviceName = "device_name"
                case appVersion = "app_version"
            }
        }
        try await postVoid(
            path: "/api/v1/notifications/push/tokens",
            body: PushTokenBody(platform: "apns", deviceToken: deviceToken, deviceName: deviceName, appVersion: "1.0.0")
        )
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

    // MARK: - Floorplan & Markers API

    public func fetchFloors() async throws -> [FloorDTO] {
        try await get(path: "/api/v1/floors", as: [FloorDTO].self)
    }

    public func createFloor(name: String, level: Int, planSvg: String? = nil, planImage: String? = nil, sortOrder: Int = 0) async throws -> FloorDTO {
        struct CreateFloorReq: Encodable {
            let name: String
            let level: Int
            let planSvg: String?
            let planImage: String?
            let sortOrder: Int
            enum CodingKeys: String, CodingKey {
                case name, level
                case planSvg = "plan_svg"
                case planImage = "plan_image"
                case sortOrder = "sort_order"
            }
        }
        return try await post(path: "/api/v1/floors", body: CreateFloorReq(name: name, level: level, planSvg: planSvg, planImage: planImage, sortOrder: sortOrder), as: FloorDTO.self)
    }

    public func updateFloor(id: String, name: String, level: Int, planSvg: String? = nil, planImage: String? = nil, sortOrder: Int = 0) async throws -> FloorDTO {
        struct UpdateFloorReq: Encodable {
            let name: String
            let level: Int
            let planSvg: String?
            let planImage: String?
            let sortOrder: Int
            enum CodingKeys: String, CodingKey {
                case name, level
                case planSvg = "plan_svg"
                case planImage = "plan_image"
                case sortOrder = "sort_order"
            }
        }
        return try await put(path: "/api/v1/floors/\(id)", body: UpdateFloorReq(name: name, level: level, planSvg: planSvg, planImage: planImage, sortOrder: sortOrder), as: FloorDTO.self)
    }

    public func deleteFloor(id: String) async throws {
        try await delete(path: "/api/v1/floors/\(id)")
    }

    public func fetchFloorMarkers(floorId: String) async throws -> [FloorMarkerDTO] {
        try await get(path: "/api/v1/floors/\(floorId)/markers", as: [FloorMarkerDTO].self)
    }

    public func saveFloorMarker(floorId: String, markerId: String? = nil, entityKey: String, xPercent: Double, yPercent: Double, iconOverride: String? = nil, showBadge: Bool = true) async throws -> FloorMarkerDTO {
        struct SaveMarkerReq: Encodable {
            let id: String?
            let entityKey: String
            let xPercent: Double
            let yPercent: Double
            let iconOverride: String?
            let showBadge: Bool
            enum CodingKeys: String, CodingKey {
                case id
                case entityKey = "entity_key"
                case xPercent = "x_percent"
                case yPercent = "y_percent"
                case iconOverride = "icon_override"
                case showBadge = "show_badge"
            }
        }
        return try await put(path: "/api/v1/floors/\(floorId)/markers", body: SaveMarkerReq(id: markerId, entityKey: entityKey, xPercent: xPercent, yPercent: yPercent, iconOverride: iconOverride, showBadge: showBadge), as: FloorMarkerDTO.self)
    }

    public func deleteFloorMarker(floorId: String, markerId: String) async throws {
        try await delete(path: "/api/v1/floors/\(floorId)/markers/\(markerId)")
    }

    // MARK: - Automations API

    public func fetchAutomations() async throws -> [AutomationRule] {
        let res = try await get(path: "/api/v1/automations", as: AutomationsResponseDTO.self)
        return res.automations
    }

    public func createAutomation(rule: AutomationRule) async throws -> AutomationRule {
        try await post(path: "/api/v1/automations", body: rule, as: AutomationRule.self)
    }

    public func updateAutomation(id: String, rule: AutomationRule) async throws -> AutomationRule {
        try await put(path: "/api/v1/automations/\(id)", body: rule, as: AutomationRule.self)
    }

    public func deleteAutomation(id: String) async throws {
        try await delete(path: "/api/v1/automations/\(id)")
    }

    public func triggerAutomation(id: String) async throws {
        try await postVoid(path: "/api/v1/automations/\(id)/trigger", body: EmptyBodyDTO())
    }

    // MARK: - Timeline & Logbook API

    public func fetchLogbook(limit: Int = 100) async throws -> [TimelineEvent] {
        let res = try await get(path: "/api/v1/logbook?limit=\(limit)", as: LogbookResponseDTO.self)
        return res.entries
    }

    // MARK: - RF Mesh & Hardware Diagnostics API

    public func fetchMeshTopology(protocol: String? = nil) async throws -> MeshTopologyDTO {
        let query = `protocol`.flatMap { "?protocol=\($0)" } ?? ""
        return try await get(path: "/api/v1/mesh/topology\(query)", as: MeshTopologyDTO.self)
    }

    public func fetchSystemHealth() async throws -> (health: SystemHealthDTO, latencyMs: Int) {
        let start = CFAbsoluteTimeGetCurrent()
        let health = try await get(path: "/api/v1/system/health", as: SystemHealthDTO.self)
        let latency = max(1, Int((CFAbsoluteTimeGetCurrent() - start) * 1000))
        return (health, latency)
    }

    // MARK: - Family & Users API

    public func fetchFamilyMembers() async throws -> [FamilyMemberDTO] {
        let res = try await get(path: "/api/v1/family/members", as: FamilyMembersResponseDTO.self)
        return res.members
    }

    public func fetchFamilyInvites() async throws -> [FamilyInviteDTO] {
        let res = try await get(path: "/api/v1/family/invites", as: FamilyInvitesResponseDTO.self)
        return res.invites
    }

    public func createFamilyInvite(role: String, expiresInHours: Int) async throws -> FamilyInviteDTO {
        try await post(
            path: "/api/v1/family/invites",
            body: CreateInviteRequestDTO(role: role, expiresInHours: expiresInHours),
            as: FamilyInviteDTO.self
        )
    }

    public func revokeFamilyInvite(id: String) async throws {
        try await delete(path: "/api/v1/family/invites/\(id)")
    }

    public func fetchAreas() async throws -> [Area] {
        try await get(path: "/api/v1/areas", as: [Area].self)
    }

    public func fetchEntities() async throws -> [Entity] {
        let dtos = try await get(path: "/api/v1/entities", as: [EntityResponseDTO].self)
        return dtos.map { dto in
            Entity(
                id: dto.id,
                name: dto.name,
                domain: dto.domain,
                state: dto.state,
                attributes: dto.attributes,
                areaId: dto.areaId
            )
        }
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

public struct AssistantActionDTO: Codable, Sendable {
    public let entityKey: String
    public let service: String
    public let success: Bool
    public let message: String

    enum CodingKeys: String, CodingKey {
        case entityKey = "entity_key"
        case service
        case success
        case message
    }

    public init(entityKey: String = "", service: String = "", success: Bool = true, message: String = "") {
        self.entityKey = entityKey
        self.service = service
        self.success = success
        self.message = message
    }
}

public struct AssistantChatResponseDTO: Decodable, Sendable {
    public let userPrompt: String?
    public let responseText: String
    public let actionsExecuted: [AssistantActionDTO]?
    public let engineUsed: String?
    public let executionTimeMs: Int64?

    enum CodingKeys: String, CodingKey {
        case userPrompt = "user_prompt"
        case responseText = "response_text"
        case actionsExecuted = "actions_executed"
        case engineUsed = "engine_used"
        case executionTimeMs = "execution_time_ms"
    }

    public init(responseText: String, actionsExecuted: [AssistantActionDTO]? = nil, userPrompt: String? = nil, engineUsed: String? = nil, executionTimeMs: Int64? = nil) {
        self.responseText = responseText
        self.actionsExecuted = actionsExecuted
        self.userPrompt = userPrompt
        self.engineUsed = engineUsed
        self.executionTimeMs = executionTimeMs
    }

    public var actionSummary: String? {
        guard let actions = actionsExecuted, !actions.isEmpty else { return nil }
        return actions.map { "\($0.entityKey) → \($0.service)" }.joined(separator: ", ")
    }
}

public struct ChoreDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let title: String
    public let description: String?
    public let areaId: String?
    public let areaName: String
    public let assignedUserId: String?
    public let assignedTo: String
    public let priority: String
    public let status: String?
    public let completed: Bool
    public let karmaPoints: Int

    enum CodingKeys: String, CodingKey {
        case id
        case title
        case description
        case areaId = "area_id"
        case areaName = "area_name"
        case assignedUserId = "assigned_user_id"
        case assignedTo = "assigned_to"
        case priority
        case status
        case completed
        case karmaPoints = "karma_points"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try container.decode(String.self, forKey: .id)
        self.title = try container.decode(String.self, forKey: .title)
        self.description = try container.decodeIfPresent(String.self, forKey: .description)
        self.areaId = try container.decodeIfPresent(String.self, forKey: .areaId)
        self.areaName = try container.decodeIfPresent(String.self, forKey: .areaName) ?? self.areaId ?? "Casa"
        self.assignedUserId = try container.decodeIfPresent(String.self, forKey: .assignedUserId)
        self.assignedTo = try container.decodeIfPresent(String.self, forKey: .assignedTo) ?? self.assignedUserId ?? "Tutti"
        self.priority = try container.decodeIfPresent(String.self, forKey: .priority) ?? "medium"
        self.status = try container.decodeIfPresent(String.self, forKey: .status)

        if let comp = try container.decodeIfPresent(Bool.self, forKey: .completed) {
            self.completed = comp
        } else if let st = self.status {
            self.completed = (st == "completed")
        } else {
            self.completed = false
        }

        if let kp = try container.decodeIfPresent(Int.self, forKey: .karmaPoints) {
            self.karmaPoints = kp
        } else {
            switch self.priority.lowercased() {
            case "urgent": self.karmaPoints = 50
            case "high":   self.karmaPoints = 30
            case "medium": self.karmaPoints = 20
            default:       self.karmaPoints = 10
            }
        }
    }

    public init(id: String, title: String, areaName: String = "Casa", assignedTo: String = "Tutti", priority: String = "medium", completed: Bool = false, karmaPoints: Int = 20, description: String? = nil, areaId: String? = nil, assignedUserId: String? = nil, status: String? = nil) {
        self.id = id
        self.title = title
        self.areaName = areaName
        self.assignedTo = assignedTo
        self.priority = priority
        self.completed = completed
        self.karmaPoints = karmaPoints
        self.description = description
        self.areaId = areaId
        self.assignedUserId = assignedUserId
        self.status = status
    }
}

public struct CreateChoreDTO: Encodable, Sendable {
    public let title: String
    public let description: String
    public let areaId: String
    public let assignedUserId: String
    public let priority: String

    enum CodingKeys: String, CodingKey {
        case title
        case description
        case areaId = "area_id"
        case assignedUserId = "assigned_user_id"
        case priority
    }

    public init(title: String, areaName: String = "", assignedTo: String = "", priority: String = "medium", karmaPoints: Int = 20, description: String = "") {
        self.title = title
        self.description = description
        self.areaId = areaName
        self.assignedUserId = assignedTo
        self.priority = priority
    }
}

public struct ShoppingItemDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let title: String
    public let quantity: String
    public let category: String
    public let completed: Bool
    public let addedBy: String?
    public let completedBy: String?
    public let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case title
        case quantity
        case category
        case completed
        case addedBy = "added_by"
        case completedBy = "completed_by"
        case createdAt = "created_at"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try container.decode(String.self, forKey: .id)
        if let name = try container.decodeIfPresent(String.self, forKey: .name) {
            self.title = name
        } else if let title = try container.decodeIfPresent(String.self, forKey: .title) {
            self.title = title
        } else {
            self.title = ""
        }
        self.quantity = try container.decodeIfPresent(String.self, forKey: .quantity) ?? "1"
        self.category = try container.decodeIfPresent(String.self, forKey: .category) ?? "Dispensa"
        self.completed = try container.decodeIfPresent(Bool.self, forKey: .completed) ?? false
        self.addedBy = try container.decodeIfPresent(String.self, forKey: .addedBy)
        self.completedBy = try container.decodeIfPresent(String.self, forKey: .completedBy)
        self.createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt)
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(title, forKey: .name)
        try container.encode(quantity, forKey: .quantity)
        try container.encode(category, forKey: .category)
        try container.encode(completed, forKey: .completed)
        try container.encodeIfPresent(addedBy, forKey: .addedBy)
        try container.encodeIfPresent(completedBy, forKey: .completedBy)
        try container.encodeIfPresent(createdAt, forKey: .createdAt)
    }

    public init(id: String, title: String, quantity: String, category: String, completed: Bool, completedBy: String? = nil, createdAt: String? = nil, addedBy: String? = nil) {
        self.id = id
        self.title = title
        self.quantity = quantity
        self.category = category
        self.completed = completed
        self.completedBy = completedBy
        self.createdAt = createdAt
        self.addedBy = addedBy
    }
}

public struct CreateShoppingDTO: Encodable, Sendable {
    public let name: String
    public let quantity: String
    public let category: String
    public let addedBy: String

    enum CodingKeys: String, CodingKey {
        case name
        case quantity
        case category
        case addedBy = "added_by"
    }

    public init(title: String, quantity: String, category: String, addedBy: String = "iOS Companion") {
        self.name = title
        self.quantity = quantity
        self.category = category
        self.addedBy = addedBy
    }
}

public struct EnergyFlowDTO: Codable, Sendable {
    public let solarWatts: Double
    public let gridWatts: Double
    public let batteryWatts: Double
    public let batterySoc: Double
    public let houseConsumptionWatts: Double
    public let solarToHouseWatts: Double?
    public let solarToBatteryWatts: Double?
    public let solarToGridWatts: Double?
    public let batteryToHouseWatts: Double?
    public let gridToHouseWatts: Double?
    public let gridToBatteryWatts: Double?

    enum CodingKeys: String, CodingKey {
        case solarWatts = "solar_watts"
        case gridWatts = "grid_watts"
        case batteryWatts = "battery_watts"
        case batterySoc = "battery_soc"
        case houseConsumptionWatts = "house_consumption_watts"
        case solarToHouseWatts = "solar_to_house_watts"
        case solarToBatteryWatts = "solar_to_battery_watts"
        case solarToGridWatts = "solar_to_grid_watts"
        case batteryToHouseWatts = "battery_to_house_watts"
        case gridToHouseWatts = "grid_to_house_watts"
        case gridToBatteryWatts = "grid_to_battery_watts"
    }

    public init(
        solarWatts: Double,
        gridWatts: Double,
        batteryWatts: Double,
        batterySoc: Double,
        houseConsumptionWatts: Double,
        solarToHouseWatts: Double? = nil,
        solarToBatteryWatts: Double? = nil,
        solarToGridWatts: Double? = nil,
        batteryToHouseWatts: Double? = nil,
        gridToHouseWatts: Double? = nil,
        gridToBatteryWatts: Double? = nil
    ) {
        self.solarWatts = solarWatts
        self.gridWatts = gridWatts
        self.batteryWatts = batteryWatts
        self.batterySoc = batterySoc
        self.houseConsumptionWatts = houseConsumptionWatts
        self.solarToHouseWatts = solarToHouseWatts
        self.solarToBatteryWatts = solarToBatteryWatts
        self.solarToGridWatts = solarToGridWatts
        self.batteryToHouseWatts = batteryToHouseWatts
        self.gridToHouseWatts = gridToHouseWatts
        self.gridToBatteryWatts = gridToBatteryWatts
    }
}

public struct NotificationActionDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let title: String
    public let destructive: Bool?

    public init(id: String, title: String, destructive: Bool? = nil) {
        self.id = id
        self.title = title
        self.destructive = destructive
    }
}

public struct InAppNotificationDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let title: String
    public let message: String
    public let category: String
    public let critical: Bool
    public let imageUrl: String?
    public let actions: [NotificationActionDTO]?
    public let createdAt: String
    public let resolvedAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case title
        case message
        case category
        case critical
        case imageUrl = "image_url"
        case actions
        case createdAt = "created_at"
        case resolvedAt = "resolved_at"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try container.decode(String.self, forKey: .id)
        self.title = try container.decode(String.self, forKey: .title)
        self.message = try container.decode(String.self, forKey: .message)
        let isCrit = try container.decodeIfPresent(Bool.self, forKey: .critical) ?? false
        self.critical = isCrit
        self.imageUrl = try container.decodeIfPresent(String.self, forKey: .imageUrl)
        self.actions = try container.decodeIfPresent([NotificationActionDTO].self, forKey: .actions)
        self.createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        self.resolvedAt = try container.decodeIfPresent(String.self, forKey: .resolvedAt)

        if let cat = try container.decodeIfPresent(String.self, forKey: .category), !cat.isEmpty {
            self.category = cat
        } else {
            let lower = self.title.lowercased()
            if isCrit || lower.contains("allarme") || lower.contains("sicurezza") {
                self.category = "Sicurezza"
            } else if lower.contains("energia") || lower.contains("kw") || lower.contains("consum") {
                self.category = "Energia"
            } else if lower.contains("aggiornamento") || lower.contains("sistema") {
                self.category = "Sistema"
            } else {
                self.category = "Dispositivi"
            }
        }
    }

    public init(id: String, title: String, message: String, category: String, createdAt: String, critical: Bool = false, imageUrl: String? = nil, actions: [NotificationActionDTO]? = nil, resolvedAt: String? = nil) {
        self.id = id
        self.title = title
        self.message = message
        self.category = category
        self.createdAt = createdAt
        self.critical = critical
        self.imageUrl = imageUrl
        self.actions = actions
        self.resolvedAt = resolvedAt
    }
}

public struct FloorDTO: Codable, Identifiable, Sendable {
    public let id: String
    public var name: String
    public var level: Int
    public var planSvg: String?
    public var planImage: String?
    public var sortOrder: Int?

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case level
        case planSvg = "plan_svg"
        case planImage = "plan_image"
        case sortOrder = "sort_order"
    }

    public init(id: String, name: String, level: Int = 0, planSvg: String? = nil, planImage: String? = nil, sortOrder: Int? = 0) {
        self.id = id
        self.name = name
        self.level = level
        self.planSvg = planSvg
        self.planImage = planImage
        self.sortOrder = sortOrder
    }
}

public struct FloorMarkerDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let floorId: String?
    public var entityKey: String
    public var xPercent: Double
    public var yPercent: Double
    public var iconOverride: String?
    public var showBadge: Bool

    enum CodingKeys: String, CodingKey {
        case id
        case floorId = "floor_id"
        case entityKey = "entity_key"
        case xPercent = "x_percent"
        case yPercent = "y_percent"
        case iconOverride = "icon_override"
        case showBadge = "show_badge"
    }

    public init(id: String, floorId: String? = nil, entityKey: String, xPercent: Double, yPercent: Double, iconOverride: String? = nil, showBadge: Bool = true) {
        self.id = id
        self.floorId = floorId
        self.entityKey = entityKey
        self.xPercent = xPercent
        self.yPercent = yPercent
        self.iconOverride = iconOverride
        self.showBadge = showBadge
    }
}

public struct FamilyMemberDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let username: String
    public let email: String?
    public let role: String?
    public let enabled: Bool?
    public let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case username
        case email
        case role
        case enabled
        case createdAt = "created_at"
    }

    public init(id: String, username: String, email: String? = nil, role: String? = nil, enabled: Bool? = nil, createdAt: String? = nil) {
        self.id = id
        self.username = username
        self.email = email
        self.role = role
        self.enabled = enabled
        self.createdAt = createdAt
    }
}

public struct FamilyMembersResponseDTO: Codable, Sendable {
    public let members: [FamilyMemberDTO]
    public let total: Int?

    public init(members: [FamilyMemberDTO], total: Int? = nil) {
        self.members = members
        self.total = total
    }
}

public struct AutomationsResponseDTO: Codable, Sendable {
    public let automations: [AutomationRule]
    public let total: Int?

    public init(automations: [AutomationRule], total: Int? = nil) {
        self.automations = automations
        self.total = total
    }
}

public struct LogbookResponseDTO: Codable, Sendable {
    public let entries: [TimelineEvent]
    public let total: Int?

    public init(entries: [TimelineEvent], total: Int? = nil) {
        self.entries = entries
        self.total = total
    }
}

public struct MeshNodeDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let name: String
    public let protocolType: String
    public let type: String
    public let lqi: Int
    public let battery: Int?
    public let parentId: String?

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case protocolType = "protocol"
        case type
        case lqi
        case battery
        case parentId = "parent_id"
    }

    public init(
        id: String,
        name: String,
        protocolType: String,
        type: String = "end_device",
        lqi: Int = 200,
        battery: Int? = nil,
        parentId: String? = nil
    ) {
        self.id = id
        self.name = name
        self.protocolType = protocolType
        self.type = type
        self.lqi = lqi
        self.battery = battery
        self.parentId = parentId
    }
}

public struct MeshTopologyDTO: Codable, Sendable {
    public let protocolType: String?
    public let nodes: [MeshNodeDTO]

    enum CodingKeys: String, CodingKey {
        case protocolType = "protocol"
        case nodes
    }

    public init(protocolType: String? = nil, nodes: [MeshNodeDTO] = []) {
        self.protocolType = protocolType
        self.nodes = nodes
    }
}

public struct SystemHealthDTO: Codable, Sendable {
    public let status: String
    public let version: String?
    public let uptimeSeconds: Int64?
    public let database: String?

    enum CodingKeys: String, CodingKey {
        case status
        case version
        case uptimeSeconds = "uptime_s"
        case database
    }

    public init(status: String = "ok", version: String? = "1.0.0", uptimeSeconds: Int64? = 0, database: String? = "healthy") {
        self.status = status
        self.version = version
        self.uptimeSeconds = uptimeSeconds
        self.database = database
    }
}

public struct FamilyInviteDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let token: String
    public let role: String
    public let expiresAt: String?
    public let inviteUrl: String?

    enum CodingKeys: String, CodingKey {
        case id
        case token
        case role
        case expiresAt = "expires_at"
        case inviteUrl = "invite_url"
    }

    public init(id: String, token: String, role: String, expiresAt: String? = nil, inviteUrl: String? = nil) {
        self.id = id
        self.token = token
        self.role = role
        self.expiresAt = expiresAt
        self.inviteUrl = inviteUrl
    }
}

public struct FamilyInvitesResponseDTO: Codable, Sendable {
    public let invites: [FamilyInviteDTO]
    public let total: Int?

    public init(invites: [FamilyInviteDTO], total: Int? = nil) {
        self.invites = invites
        self.total = total
    }
}

public struct CreateInviteRequestDTO: Encodable, Sendable {
    public let role: String
    public let expiresInHours: Int

    enum CodingKeys: String, CodingKey {
        case role
        case expiresInHours = "expires_in_hours"
    }

    public init(role: String, expiresInHours: Int) {
        self.role = role
        self.expiresInHours = expiresInHours
    }
}

public struct EntityResponseDTO: Decodable, Sendable {
    public let id: String
    public let name: String
    public let domain: String
    public let state: String
    public let attributes: [String: AnyCodable]
    public let areaId: String?

    enum CodingKeys: String, CodingKey {
        case id
        case entityId = "entity_id"
        case entityKey = "entity_key"
        case name
        case friendlyName = "friendly_name"
        case domain
        case state
        case attributes
        case areaId = "area_id"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        self.id = (try? container.decodeIfPresent(String.self, forKey: .id))
            ?? (try? container.decodeIfPresent(String.self, forKey: .entityId))
            ?? (try? container.decodeIfPresent(String.self, forKey: .entityKey))
            ?? UUID().uuidString
        self.name = (try? container.decodeIfPresent(String.self, forKey: .name))
            ?? (try? container.decodeIfPresent(String.self, forKey: .friendlyName))
            ?? self.id
        self.domain = (try? container.decodeIfPresent(String.self, forKey: .domain))
            ?? (self.id.contains(".") ? String(self.id.split(separator: ".")[0]) : "switch")
        self.state = (try? container.decodeIfPresent(String.self, forKey: .state)) ?? "off"
        self.areaId = try? container.decodeIfPresent(String.self, forKey: .areaId)
        self.attributes = (try? container.decodeIfPresent([String: AnyCodable].self, forKey: .attributes)) ?? [:]
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
