import Foundation
import SwiftUI
import Combine

#if canImport(UIKit)
import UIKit
#endif

/// Stato operativo della modalità Kiosk.
public enum KioskDisplayState: String, Sendable {
    case fullDashboard
    case ambientScreensaver
}

/// Gestore per la Modalità Kiosk su tablet e iPad a parete — Fase 5.
/// Include protezione ciclo batteria (80%-20%), schermo sempre attivo e screensaver notturno.
@MainActor
public final class KioskModeManager: ObservableObject {
    @Published public var isKioskModeEnabled: Bool = false
    @Published public var displayState: KioskDisplayState = .fullDashboard
    @Published public var batteryLevel: Float = 1.0
    @Published public var isCharging: Bool = false
    @Published public var isBatteryProtectionEnabled: Bool = true
    @Published public var smartPlugEntityId: String = "switch.tablet_charger"
    @Published public var isNightDimmingEnabled: Bool = true
    @Published public var isGuidedAccessActive: Bool = false

    public var onSmartPlugCommand: ((String, Bool) -> Void)?

    private var cancellables = Set<AnyCancellable>()
    private var sleepTimer: Timer?

    public init() {
        #if canImport(UIKit) && os(iOS)
        UIDevice.current.isBatteryMonitoringEnabled = true
        self.batteryLevel = UIDevice.current.batteryLevel
        self.isCharging = UIDevice.current.batteryState == .charging || UIDevice.current.batteryState == .full
        self.isGuidedAccessActive = UIAccessibility.isGuidedAccessEnabled

        setupBatteryObservers()
        #endif
    }

    /// Attiva o disattiva la modalità Kiosk a parete.
    public func setKioskMode(enabled: Bool) {
        self.isKioskModeEnabled = enabled
        #if canImport(UIKit) && os(iOS)
        // Impedisce lo spegnimento dello schermo dell'iPad se abilitato
        UIApplication.shared.isIdleTimerDisabled = enabled
        #endif

        if enabled {
            startInactivityTimer()
        } else {
            stopInactivityTimer()
            displayState = .fullDashboard
        }
    }

    /// Risveglia lo schermo dallo screensaver alla dashboard completa al tocco o rilevamento.
    public func wakeFromScreensaver() {
        displayState = .fullDashboard
        if isKioskModeEnabled {
            startInactivityTimer()
        }
    }

    /// Forza l'attivazione dello screensaver notturno.
    public func enterScreensaver() {
        displayState = .ambientScreensaver
    }

    // MARK: - Gestione Inattività

    private func startInactivityTimer() {
        sleepTimer?.invalidate()
        // Dopo 3 minuti di inattività attiva lo screensaver OLED minimale
        sleepTimer = Timer.scheduledTimer(withTimeInterval: 180, repeats: false) { [weak self] _ in
            Task { @MainActor [weak self] in
                if self?.isKioskModeEnabled == true {
                    self?.displayState = .ambientScreensaver
                }
            }
        }
    }

    private func stopInactivityTimer() {
        sleepTimer?.invalidate()
        sleepTimer = nil
    }

    // MARK: - Protezione Ciclo Batteria (80% / 20%)

    #if canImport(UIKit) && os(iOS)
    private func setupBatteryObservers() {
        NotificationCenter.default.publisher(for: UIDevice.batteryLevelDidChangeNotification)
            .sink { [weak self] _ in
                self?.handleBatteryChange()
            }
            .store(in: &cancellables)

        NotificationCenter.default.publisher(for: UIDevice.batteryStateDidChangeNotification)
            .sink { [weak self] _ in
                self?.handleBatteryChange()
            }
            .store(in: &cancellables)

        NotificationCenter.default.publisher(for: UIAccessibility.guidedAccessStatusDidChangeNotification)
            .sink { [weak self] _ in
                self?.isGuidedAccessActive = UIAccessibility.isGuidedAccessEnabled
            }
            .store(in: &cancellables)
    }

    private func handleBatteryChange() {
        let level = UIDevice.current.batteryLevel
        let state = UIDevice.current.batteryState
        self.batteryLevel = level
        self.isCharging = (state == .charging || state == .full)

        guard isBatteryProtectionEnabled, !smartPlugEntityId.isEmpty else { return }

        // Se la batteria supera l'80% e sta caricando -> stacca alimentazione alla presa smart
        if level >= 0.80 && (state == .charging || state == .full) {
            onSmartPlugCommand?(smartPlugEntityId, false)
        }
        // Se la batteria scende sotto il 20% ed è scollegata -> riattiva la presa di ricarica
        else if level <= 0.20 && state == .unplugged {
            onSmartPlugCommand?(smartPlugEntityId, true)
        }
    }
    #endif
}
