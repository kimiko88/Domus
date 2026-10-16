import Foundation
import SwiftUI
#if canImport(Accessibility)
import Accessibility
#endif
#if canImport(UIKit)
import UIKit
#endif

/// Helper per la gestione centralizzata delle notifiche e comportamenti di accessibilità in DomusKit.
public enum DomusAccessibility {
    /// Invia un annuncio vocale a VoiceOver in tempo reale.
    @MainActor
    public static func announce(_ message: String) {
        if #available(iOS 17.0, macOS 14.0, watchOS 10.0, tvOS 17.0, *) {
            AccessibilityNotification.Announcement(message).post()
        } else {
            #if canImport(UIKit)
            UIAccessibility.post(notification: .announcement, argument: message)
            #endif
        }
    }
}
