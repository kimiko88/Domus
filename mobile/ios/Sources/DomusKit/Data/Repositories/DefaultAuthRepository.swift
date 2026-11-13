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
        try await login(username: username, password: password, localEndpoint: nil, remoteEndpoint: nil, twoFactorCode: nil)
    }

    public func login(username: String, password: String, localEndpoint: String?, remoteEndpoint: String?) async throws -> AuthSession {
        try await login(username: username, password: password, localEndpoint: localEndpoint, remoteEndpoint: remoteEndpoint, twoFactorCode: nil)
    }

    public func login(username: String, password: String, localEndpoint: String?, remoteEndpoint: String?, twoFactorCode: String?) async throws -> AuthSession {
        if localEndpoint != nil || remoteEndpoint != nil {
            await connectionRepository.setEndpoints(local: localEndpoint, remote: remoteEndpoint)
        }
        let conn = await connectionRepository.getCurrentConnection()
        let localCandidate = localEndpoint ?? conn.localAddress ?? UserDefaults.standard.string(forKey: "domus_local_endpoint")
        let remoteCandidate = remoteEndpoint ?? conn.remoteAddress ?? UserDefaults.standard.string(forKey: "domus_remote_endpoint")

        var sessionResult: AuthSession?
        var lastError: Error?

        // 1. Prova prima l'endpoint locale se configurato (LAN)
        if let local = localCandidate, !local.trimmingCharacters(in: .whitespaces).isEmpty {
            do {
                sessionResult = try await httpClient.login(baseUrl: local, username: username, password: password, twoFactorCode: twoFactorCode)
            } catch {
                lastError = error
            }
        }

        // 2. Se l'endpoint locale fallisce o non c'è, prova l'endpoint remoto TLS (accesso fuori casa / 4G / 5G)
        if sessionResult == nil, let remote = remoteCandidate, !remote.trimmingCharacters(in: .whitespaces).isEmpty {
            do {
                sessionResult = try await httpClient.login(baseUrl: remote, username: username, password: password, twoFactorCode: twoFactorCode)
            } catch {
                lastError = error
            }
        }

        // 3. Fallback su activeUrl o default
        if sessionResult == nil {
            if let active = conn.activeUrl, active != localCandidate && active != remoteCandidate {
                sessionResult = try await httpClient.login(baseUrl: active, username: username, password: password, twoFactorCode: twoFactorCode)
            } else if let err = lastError {
                throw err
            } else {
                let defaultUrl = "http://192.168.1.100:8080"
                sessionResult = try await httpClient.login(baseUrl: defaultUrl, username: username, password: password, twoFactorCode: twoFactorCode)
            }
        }

        guard let loggedSession = sessionResult else {
            throw DomusNetworkError.notConnected
        }

        if let token = loggedSession.token, !(await keychainManager.saveToken(token)) {
            throw DomusNetworkError.invalidResponse
        }
        await keychainManager.saveUsername(username)
        self.session = loggedSession
        try? await connectionRepository.connect(mode: nil)
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
            let candidates = [
                UserDefaults.standard.string(forKey: "domus_local_endpoint"),
                UserDefaults.standard.string(forKey: "domus_remote_endpoint"),
                UserDefaults.standard.string(forKey: "domus_server_url")
            ].compactMap { $0 }.filter { !$0.isEmpty }
            var seenEndpoints = Set<String>()
            let endpoints = candidates.filter { seenEndpoints.insert($0).inserted }
            for endpoint in endpoints {
                do {
                    self.session = try await httpClient.validateSession(baseUrl: endpoint, token: token)
                    return self.session
                } catch DomusNetworkError.httpError(let code) where code == 401 || code == 403 {
                    _ = await keychainManager.deleteToken()
                    self.session = AuthSession()
                    return self.session
                } catch {
                    continue
                }
            }
            // Keep the Keychain token on transient network errors; it may work
            // again when the device reaches the hub or remote endpoint.
            self.session = AuthSession()
        } else {
            self.session = AuthSession()
        }
        return self.session
    }
}
