import Foundation
import SwiftUI
import Combine

public enum DomusLanguage: String, CaseIterable, Identifiable, Codable, Sendable {
    case it = "it"
    case en = "en"
    case fr = "fr"
    case es = "es"
    case de = "de"

    public var id: String { rawValue }

    public var displayName: String {
        switch self {
        case .it: return "Italiano"
        case .en: return "English"
        case .fr: return "Français"
        case .es: return "Español"
        case .de: return "Deutsch"
        }
    }

    public var flag: String {
        switch self {
        case .it: return "🇮🇹"
        case .en: return "🇬🇧"
        case .fr: return "🇫🇷"
        case .es: return "🇪🇸"
        case .de: return "🇩🇪"
        }
    }
}

@MainActor
public final class DomusLocalizationManager: ObservableObject {
    public static let shared = DomusLocalizationManager()

    private let storageKey = "domus_language"

    @Published public var currentLanguage: DomusLanguage {
        didSet {
            UserDefaults.standard.set(currentLanguage.rawValue, forKey: storageKey)
            updateBundle()
        }
    }

    private var currentBundle: Bundle?

    public init() {
        if let saved = UserDefaults.standard.string(forKey: storageKey),
           let lang = DomusLanguage(rawValue: saved) {
            self.currentLanguage = lang
        } else {
            // Check system preferred language
            let preferred = Locale.preferredLanguages.first ?? "it"
            let code = String(preferred.prefix(2)).lowercased()
            self.currentLanguage = DomusLanguage(rawValue: code) ?? .it
        }
        updateBundle()
    }

    private func updateBundle() {
        #if SWIFT_PACKAGE
        let baseBundle = Bundle.module
        #else
        let baseBundle = Bundle(for: DomusLocalizationManager.self)
        #endif

        if let path = baseBundle.path(forResource: currentLanguage.rawValue, ofType: "lproj"),
           let bundle = Bundle(path: path) {
            self.currentBundle = bundle
        } else if let mainPath = Bundle.main.path(forResource: currentLanguage.rawValue, ofType: "lproj"),
                  let bundle = Bundle(path: mainPath) {
            self.currentBundle = bundle
        } else {
            self.currentBundle = baseBundle
        }
    }

    public func tr(_ key: String, default defaultValue: String? = nil) -> String {
        if let bundle = currentBundle {
            let res = bundle.localizedString(forKey: key, value: defaultValue, table: nil)
            if res != key { return res }
        }
        #if SWIFT_PACKAGE
        let fallback = Bundle.module.localizedString(forKey: key, value: defaultValue ?? key, table: nil)
        #else
        let fallback = NSLocalizedString(key, value: defaultValue ?? key, comment: "")
        #endif
        return fallback
    }
}

public struct DomusL10n {
    @MainActor
    public static func tr(_ key: String, default defaultValue: String? = nil) -> String {
        DomusLocalizationManager.shared.tr(key, default: defaultValue)
    }
}

public extension String {
    @MainActor
    var localized: String {
        DomusL10n.tr(self)
    }
}
