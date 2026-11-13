import Foundation
import Security

/// Gestore del token di autenticazione nel Keychain di iOS.
/// I token sono conservati nel Keychain con protezione Data Protection di iOS.
public actor KeychainManager: Sendable {

    private let service = "com.fabriziochimienti.domus"
    private let tokenKey = "auth_token"
    private let usernameKey = "username"

    public init() {}

    // MARK: - Token

    @discardableResult
    public func saveToken(_ token: String) -> Bool { save(value: token, forKey: tokenKey) }

    public func getToken() -> String? {
        load(forKey: tokenKey)
    }

    @discardableResult
    public func deleteToken() -> Bool { delete(forKey: tokenKey) }

    // MARK: - Username

    public func saveUsername(_ username: String) {
        save(value: username, forKey: usernameKey)
    }

    public func getUsername() -> String? {
        load(forKey: usernameKey)
    }

    // MARK: - Private Keychain helpers

    @discardableResult
    private func save(value: String, forKey key: String) -> Bool {
        guard let data = value.data(using: .utf8) else { return false }

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
        if updateStatus == errSecSuccess { return true }
        guard updateStatus == errSecItemNotFound else { return false }
        var addQuery = query
        addQuery[kSecValueData as String] = data
        addQuery[kSecAttrAccessible as String] = kSecAttrAccessibleWhenUnlocked
        let addStatus = SecItemAdd(addQuery as CFDictionary, nil)
        if addStatus == errSecDuplicateItem {
            return SecItemUpdate(query as CFDictionary, attributes as CFDictionary) == errSecSuccess
        }
        return addStatus == errSecSuccess
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

    @discardableResult
    private func delete(forKey key: String) -> Bool {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key
        ]
        let status = SecItemDelete(query as CFDictionary)
        return status == errSecSuccess || status == errSecItemNotFound
    }
}
