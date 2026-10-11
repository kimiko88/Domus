import Foundation
import Security

/// Gestore del token di autenticazione nel Keychain di iOS.
/// I token sono cifrati dall'hardware Secure Enclave.
public actor KeychainManager: Sendable {

    private let service = "com.fabriziochimienti.domus"
    private let tokenKey = "auth_token"
    private let usernameKey = "username"

    public init() {}

    // MARK: - Token

    public func saveToken(_ token: String) {
        save(value: token, forKey: tokenKey)
    }

    public func getToken() -> String? {
        load(forKey: tokenKey)
    }

    public func deleteToken() {
        delete(forKey: tokenKey)
    }

    // MARK: - Username

    public func saveUsername(_ username: String) {
        save(value: username, forKey: usernameKey)
    }

    public func getUsername() -> String? {
        load(forKey: usernameKey)
    }

    // MARK: - Private Keychain helpers

    private func save(value: String, forKey key: String) {
        guard let data = value.data(using: .utf8) else { return }

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key
        ]

        let attributes: [String: Any] = [
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlocked
        ]

        // Prima prova ad aggiornare
        let updateStatus = SecItemUpdate(query as CFDictionary, attributes as CFDictionary)
        if updateStatus == errSecItemNotFound {
            // Non esiste ancora: aggiunge
            var addQuery = query
            addQuery[kSecValueData as String] = data
            addQuery[kSecAttrAccessible as String] = kSecAttrAccessibleWhenUnlocked
            SecItemAdd(addQuery as CFDictionary, nil)
        }
    }

    private func load(forKey key: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        guard status == errSecSuccess, let data = result as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    private func delete(forKey key: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key
        ]
        SecItemDelete(query as CFDictionary)
    }
}
