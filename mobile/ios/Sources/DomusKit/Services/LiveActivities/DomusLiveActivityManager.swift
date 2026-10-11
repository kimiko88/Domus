import Foundation
import Combine

#if canImport(ActivityKit) && os(iOS)
import ActivityKit
#endif

/// Manager per la gestione del ciclo di vita delle Live Activities e Dynamic Island.
@MainActor
public final class DomusLiveActivityManager: ObservableObject {
    public static let shared = DomusLiveActivityManager()

    @Published public private(set) var isActivityActive: Bool = false
    @Published public private(set) var currentActivityTitle: String? = nil
    @Published public private(set) var activeActivityType: DomusActivityType? = nil

    #if canImport(ActivityKit) && os(iOS)
    private var currentActivity: Activity<DomusLiveActivityAttributes>? = nil
    #endif

    public init() {}

    /// Avvia il conto alla rovescia di inserimento/disarmo allarme (30 secondi).
    public func startAlarmCountdown(seconds: Int = 30, isArming: Bool = true) {
        let title = isArming ? "Armamento Allarme" : "Ritardo Ingresso Allarme"
        let subtitle = isArming ? "Esci dall'abitazione prima dello scadere" : "Inserisci il PIN per disarmare"

        let state = DomusLiveActivityAttributes.ContentState(
            title: title,
            subtitle: subtitle,
            progress: 1.0,
            remainingSeconds: seconds,
            iconName: "shield.lefthalf.filled",
            activityType: .alarmCountdown,
            isCritical: !isArming,
            targetEntityId: "alarm_control_panel.home"
        )

        startActivity(attributes: DomusLiveActivityAttributes(activityId: "alarm-\(UUID().uuidString)"), initialState: state)
    }

    /// Avvia il monitoraggio di un elettrodomestico (es. Lavatrice/Asciugatrice).
    public func startApplianceCycle(name: String, remainingMinutes: Int, iconName: String = "washer.fill") {
        let state = DomusLiveActivityAttributes.ContentState(
            title: name,
            subtitle: "Fine ciclo tra \(remainingMinutes) min",
            progress: 0.65,
            remainingSeconds: remainingMinutes * 60,
            iconName: iconName,
            activityType: .applianceCycle,
            isCritical: false,
            targetEntityId: nil
        )

        startActivity(attributes: DomusLiveActivityAttributes(activityId: "appliance-\(UUID().uuidString)"), initialState: state)
    }

    /// Avvia un banner persistente per allarme critico (fuga gas, allagamento, intrusione).
    public func startCriticalAlert(title: String, detail: String, iconName: String = "exclamationmark.shield.fill") {
        let state = DomusLiveActivityAttributes.ContentState(
            title: title,
            subtitle: detail,
            progress: 1.0,
            remainingSeconds: 0,
            iconName: iconName,
            activityType: .criticalAlert,
            isCritical: true,
            targetEntityId: "alarm_control_panel.home"
        )

        startActivity(attributes: DomusLiveActivityAttributes(activityId: "alert-\(UUID().uuidString)"), initialState: state)
    }

    /// Aggiorna lo stato della Live Activity in esecuzione.
    public func updateState(remainingSeconds: Int, progress: Double, subtitle: String? = nil) {
        guard isActivityActive else { return }

        #if canImport(ActivityKit) && os(iOS)
        guard let activity = currentActivity else { return }
        var current = activity.content.state
        current.remainingSeconds = remainingSeconds
        current.progress = progress
        if let subtitle = subtitle {
            current.subtitle = subtitle
        }

        Task {
            let updatedContent = ActivityContent(state: current, staleDate: nil)
            await activity.update(updatedContent)
        }
        #endif
    }

    /// Termina la Live Activity in corso.
    public func stopActivity() {
        #if canImport(ActivityKit) && os(iOS)
        if let activity = currentActivity {
            Task {
                await activity.end(nil, dismissalPolicy: .immediate)
            }
            self.currentActivity = nil
        }
        #endif

        self.isActivityActive = false
        self.currentActivityTitle = nil
        self.activeActivityType = nil
    }

    // MARK: - Private Helpers

    private func startActivity(attributes: DomusLiveActivityAttributes, initialState: DomusLiveActivityAttributes.ContentState) {
        self.isActivityActive = true
        self.currentActivityTitle = initialState.title
        self.activeActivityType = initialState.activityType

        #if canImport(ActivityKit) && os(iOS)
        guard ActivityAuthorizationInfo().areActivitiesEnabled else { return }

        do {
            let content = ActivityContent(state: initialState, staleDate: nil)
            self.currentActivity = try Activity.request(
                attributes: attributes,
                content: content,
                pushType: nil
            )
        } catch {
            print("Errore avvio Live Activity: \(error)")
        }
        #endif
    }
}
