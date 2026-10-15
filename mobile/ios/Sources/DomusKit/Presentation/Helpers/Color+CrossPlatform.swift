import SwiftUI

#if canImport(UIKit)
import UIKit
#endif

#if canImport(AppKit)
import AppKit
#endif

public extension Color {
    /// Sfondo principale raggruppato (es. schermate modali, liste e form)
    static var domusSystemGroupedBackground: Color {
        #if canImport(UIKit)
        return Color(uiColor: .systemGroupedBackground)
        #elseif canImport(AppKit)
        return Color(nsColor: .windowBackgroundColor)
        #else
        return Color(white: 0.08)
        #endif
    }

    /// Sfondo secondario raggruppato per card, bolle di chat e pillole
    static var domusSecondarySystemGroupedBackground: Color {
        #if canImport(UIKit)
        return Color(uiColor: .secondarySystemGroupedBackground)
        #elseif canImport(AppKit)
        return Color(nsColor: .controlBackgroundColor)
        #else
        return Color(white: 0.14)
        #endif
    }

    /// Sfondo standard di pagina
    static var domusSystemBackground: Color {
        #if canImport(UIKit)
        return Color(uiColor: .systemBackground)
        #elseif canImport(AppKit)
        return Color(nsColor: .windowBackgroundColor)
        #else
        return Color(white: 0.08)
        #endif
    }

    /// Sfondo secondario per superfici sopraelevate e card
    static var domusSecondarySystemBackground: Color {
        #if canImport(UIKit)
        return Color(uiColor: .secondarySystemBackground)
        #elseif canImport(AppKit)
        return Color(nsColor: .underPageBackgroundColor)
        #else
        return Color(white: 0.16)
        #endif
    }
}

#if !canImport(UIKit)
/// Shim di compatibilità per compilazione cross-platform su macOS e SPM
public enum DomusUIColorShim: Sendable {
    case systemGroupedBackground
    case secondarySystemGroupedBackground
    case systemBackground
    case secondarySystemBackground
}

public typealias UIColor = DomusUIColorShim

public extension Color {
    init(uiColor: DomusUIColorShim) {
        #if canImport(AppKit)
        switch uiColor {
        case .systemGroupedBackground:
            self.init(nsColor: .windowBackgroundColor)
        case .secondarySystemGroupedBackground:
            self.init(nsColor: .controlBackgroundColor)
        case .systemBackground:
            self.init(nsColor: .windowBackgroundColor)
        case .secondarySystemBackground:
            self.init(nsColor: .underPageBackgroundColor)
        }
        #else
        switch uiColor {
        case .systemGroupedBackground:
            self.init(white: 0.08)
        case .secondarySystemGroupedBackground:
            self.init(white: 0.14)
        case .systemBackground:
            self.init(white: 0.08)
        case .secondarySystemBackground:
            self.init(white: 0.16)
        }
        #endif
    }

    init(_ uiColor: DomusUIColorShim) {
        self.init(uiColor: uiColor)
    }
}
#endif
