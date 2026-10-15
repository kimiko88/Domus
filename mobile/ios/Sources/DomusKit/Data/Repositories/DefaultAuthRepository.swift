import Foundation

/// Implementazione concreta di AuthRepositoryProtocol.
/// Gestisce la persistenza del token sicuro tramite KeychainManager e l'autenticazione via HTTP.
public actor DefaultAuthRepository: AuthRepositoryProtocol {
    private let httpClient: DomusHttpClient
    private let keychainManager: KeychainManager
    private let connectionRepository: ConnectionRepositoryProtocol
    private var session: AuthSession = AuthSession()

    public init(
        httpClient: DomusHttpClient,
        keychainManager: KeychainManager,
        connectionRepository: ConnectionRepositoryProtocol
    ) {
        self.httpClient = httpClient
        self.keychainManager = keychainManager
        self.connectionRepository = connectionRepository
    }

    public func getSession() async -> AuthSession {
        return session
    }

    public func login(username: String, password: String) async throws -> AuthSession {
        let conn = await connectionRepository.getCurrentConnection()
        let baseUrl = conn.activeUrl ?? conn.localAddress ?? "http://192.168.1.100:8080"

        let loggedSession = try await httpClient.login(baseUrl: baseUrl, username: username, password: password)
        if let token = loggedSession.token {
            await keychainManager.saveToken(token)
        }
        await keychainManager.saveUsername(username)
        self.session = loggedSession
        try? await connectionRepository.connect(mode: conn.mode)
        return loggedSession
    }

    public func logout() async {
        await keychainManager.deleteToken()
        self.session = AuthSession()
        await connectionRepository.disconnect()
    }

    public func getSavedToken() async -> String? {
        await keychainManager.getToken()
    }

    public func restoreSession() async -> AuthSession {
        if let token = await keychainManager.getToken(), !token.isEmpty {
            let username = await keychainManager.getUsername() ?? "admin"
            self.session = AuthSession(token: token, username: username, role: "admin", isAuthenticated: true)
        } else {
            self.session = AuthSession()
        }
        return self.session
    }
}
