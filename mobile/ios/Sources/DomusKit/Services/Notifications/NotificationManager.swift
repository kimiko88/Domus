import Foundation
import UserNotifications

/// Categorie di notifiche interattive Domus — Fase 3.
public enum DomusNotificationCategory: String, CaseIterable, Sendable {
    case alarmTriggered = "ALARM_TRIGGERED"
    case windowRainAlert = "WINDOW_RAIN_ALERT"
    case securityCamera = "SECURITY_CAMERA"
    case applianceFinished = "APPLIANCE_FINISHED"
}

/// Identificatori delle azioni rapide collegate alle notifiche.
public enum DomusNotificationAction: String, Sendable {
    case disarmAlarm = "ACTION_DISARM_ALARM"
    case closeCovers = "ACTION_CLOSE_COVERS"
    case viewCamera = "ACTION_VIEW_CAMERA"
    case snooze = "ACTION_SNOOZE"
}

/// Gestore notifiche interattive e Allarmi Critici (Critical Alerts) — Fase 3.
@MainActor
public final class NotificationManager: NSObject, ObservableObject {
    @Published public var authorizationStatus: UNAuthorizationStatus = .notDetermined
    @Published public var criticalAlertsAllowed: Bool = false
    @Published public var lastReceivedNotification: [AnyHashable: Any]? = nil

    public var onActionTriggered: ((String, String, [AnyHashable: Any]) -> Void)?

    private var center: UNUserNotificationCenter? {
        guard isEnvironmentSupported else { return nil }
        return UNUserNotificationCenter.current()
    }

    private var isEnvironmentSupported: Bool {
        // UNUserNotificationCenter richiede un bundle applicativo iOS (.app / .appex)
        // e va in eccezione se chiamato dentro test CLI headless o xctest runner
        guard let _ = Bundle.main.bundleIdentifier,
              Bundle.main.bundleURL.pathExtension == "app" || Bundle.main.bundleURL.pathExtension == "appex" else {
            return false
        }
        return true
    }

    public override init() {
        super.init()
        if isEnvironmentSupported {
            center?.delegate = self
            setupNotificationCategories()
            checkAuthorizationStatus()
        } else {
            // Modalità mock per test unitari e preview
            self.authorizationStatus = .authorized
            self.criticalAlertsAllowed = true
        }
    }

    /// Controlla lo stato attuale dei permessi di notifica.
    public func checkAuthorizationStatus() {
        guard let center else { return }
        center.getNotificationSettings { [weak self] settings in
            Task { @MainActor in
                self?.authorizationStatus = settings.authorizationStatus
                self?.criticalAlertsAllowed = (settings.criticalAlertSetting == .enabled)
            }
        }
    }

    /// Richiede autorizzazione per notifiche push, badge, suoni e Allarmi Critici.
    public func requestAuthorization() async -> Bool {
        guard let center else { return true }
        do {
            var options: UNAuthorizationOptions = [.alert, .badge, .sound]
            #if os(iOS)
            options.insert(.criticalAlert)
            #endif

            let granted = try await center.requestAuthorization(options: options)
            checkAuthorizationStatus()
            return granted
        } catch {
            return false
        }
    }

    /// Registra le categorie di notifica con i pulsanti di azione rapida.
    private func setupNotificationCategories() {
        guard let center else { return }

        // 1. Allarme Scattato
        let disarmAction = UNNotificationAction(
            identifier: DomusNotificationAction.disarmAlarm.rawValue,
            title: "Disarma Subito",
            options: [.authenticationRequired, .destructive]
        )
        let snoozeAction = UNNotificationAction(
            identifier: DomusNotificationAction.snooze.rawValue,
            title: "Silenzia Notifica",
            options: []
        )
        let alarmCategory = UNNotificationCategory(
            identifier: DomusNotificationCategory.alarmTriggered.rawValue,
            actions: [disarmAction, snoozeAction],
            intentIdentifiers: [],
            options: [.customDismissAction]
        )

        // 2. Finestra aperta con pioggia
        let closeCoversAction = UNNotificationAction(
            identifier: DomusNotificationAction.closeCovers.rawValue,
            title: "Chiudi Tapparelle",
            options: [.authenticationRequired]
        )
        let windowCategory = UNNotificationCategory(
            identifier: DomusNotificationCategory.windowRainAlert.rawValue,
            actions: [closeCoversAction, snoozeAction],
            intentIdentifiers: [],
            options: []
        )

        // 3. Telecamera Movimento
        let viewCameraAction = UNNotificationAction(
            identifier: DomusNotificationAction.viewCamera.rawValue,
            title: "Guarda Live",
            options: [.foreground]
        )
        let cameraCategory = UNNotificationCategory(
            identifier: DomusNotificationCategory.securityCamera.rawValue,
            actions: [viewCameraAction],
            intentIdentifiers: [],
            options: []
        )

        center.setNotificationCategories([alarmCategory, windowCategory, cameraCategory])
    }

    /// Invia una notifica di test locale (Standard o Critical Alert con bypass del silenzioso).
    public func sendTestNotification(isCritical: Bool = false) async throws {
        guard let center else { return }

        let content = UNMutableNotificationContent()
        content.title = isCritical ? "🚨 ALLARME CRITICO: Fumo Rilevato" : "Domus: Finestra Aperta"
        content.body = isCritical
            ? "Rilevato fumo in Cucina! Pressione sonora massima anche in modalità Non Disturbare."
            : "Rilevata pioggia in arrivo e finestra studio aperta. Vuoi chiudere le tapparelle?"
        content.categoryIdentifier = isCritical
            ? DomusNotificationCategory.alarmTriggered.rawValue
            : DomusNotificationCategory.windowRainAlert.rawValue

        #if os(iOS)
        if isCritical && criticalAlertsAllowed {
            content.sound = UNNotificationSound.defaultCriticalSound(withAudioVolume: 1.0)
        } else {
            content.sound = .default
        }
        #else
        content.sound = .default
        #endif

        let trigger = UNTimeIntervalNotificationTrigger(timeInterval: 1, repeats: false)
        let request = UNNotificationRequest(identifier: UUID().uuidString, content: content, trigger: trigger)
        try await center.add(request)
    }
}

// MARK: - UNUserNotificationCenterDelegate

extension NotificationManager: UNUserNotificationCenterDelegate {
    public nonisolated func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification
    ) async -> UNNotificationPresentationOptions {
        return [.banner, .sound, .badge]
    }

    public nonisolated func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse
    ) async {
        let actionId = response.actionIdentifier
        let categoryId = response.notification.request.content.categoryIdentifier
        let userInfo = response.notification.request.content.userInfo

        Task { @MainActor in
            self.lastReceivedNotification = userInfo
            self.onActionTriggered?(actionId, categoryId, userInfo)
        }
    }
}
