import Foundation
import SwiftUI

#if canImport(ActivityKit) && os(iOS)
import ActivityKit

/// Tipi di eventi Live Activity supportati in Domus.
public enum DomusActivityType: String, Codable, Sendable {
    case alarmCountdown = "alarm_countdown"
    case applianceCycle = "appliance_cycle"
    case criticalAlert = "critical_alert"
    case evCharging = "ev_charging"
}

/// Modello ActivityAttributes per Live Activities e Dynamic Island.
public struct DomusLiveActivityAttributes: ActivityAttributes {
    public struct ContentState: Codable, Hashable, Sendable {
        public var title: String
        public var subtitle: String
        public var progress: Double        // 0.0 ... 1.0
        public var remainingSeconds: Int
        public var iconName: String
        public var activityType: DomusActivityType
        public var isCritical: Bool
        public var targetEntityId: String?
        public var evKw: Double?
        public var evBatteryPercent: Int?
        public var evEstimatedEndTime: String?

        public init(
            title: String,
            subtitle: String,
            progress: Double = 0.0,
            remainingSeconds: Int = 0,
            iconName: String = "shield.fill",
            activityType: DomusActivityType = .alarmCountdown,
            isCritical: Bool = false,
            targetEntityId: String? = nil,
            evKw: Double? = nil,
            evBatteryPercent: Int? = nil,
            evEstimatedEndTime: String? = nil
        ) {
            self.title = title
            self.subtitle = subtitle
            self.progress = progress
            self.remainingSeconds = remainingSeconds
            self.iconName = iconName
            self.activityType = activityType
            self.isCritical = isCritical
            self.targetEntityId = targetEntityId
            self.evKw = evKw
            self.evBatteryPercent = evBatteryPercent
            self.evEstimatedEndTime = evEstimatedEndTime
        }
    }

    public var activityId: String
    public var locationName: String

    public init(activityId: String, locationName: String = "Domus Hub") {
        self.activityId = activityId
        self.locationName = locationName
    }
}

// MARK: - Viste SwiftUI per Dynamic Island e Lock Screen

/// Vista per il Banner di Blocco Schermo (Lock Screen Live Activity).
public struct DomusLiveActivityLockScreenView: View {
    public let state: DomusLiveActivityAttributes.ContentState

    public init(state: DomusLiveActivityAttributes.ContentState) {
        self.state = state
    }

    public var body: some View {
        HStack(spacing: 14) {
            ZStack {
                Circle()
                    .fill(state.isCritical ? Color.red.opacity(0.2) : Color.blue.opacity(0.2))
                    .frame(width: 44, height: 44)
                Image(systemName: state.iconName)
                    .font(.system(size: 20, weight: .bold))
                    .foregroundColor(state.isCritical ? .red : .blue)
            }

            VStack(alignment: .leading, spacing: 3) {
                Text(state.title)
                    .font(.headline)
                    .foregroundColor(.primary)
                Text(state.subtitle)
                    .font(.subheadline)
                    .foregroundColor(.secondary)

                if state.progress > 0 {
                    ProgressView(value: state.progress)
                        .tint(state.isCritical ? .red : .blue)
                        .padding(.top, 2)
                }
            }

            Spacer()

            if state.remainingSeconds > 0 {
                VStack(alignment: .trailing, spacing: 2) {
                    Text("\(state.remainingSeconds)s")
                        .font(.title3.bold().monospacedDigit())
                        .foregroundColor(state.isCritical ? .red : .primary)
                    Text("rimanenti")
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }
            }
        }
        .padding(16)
        .background(Color.domusSecondarySystemBackground)
    }
}

/// Viste compatte e minime per la Dynamic Island
public struct DomusDynamicIslandCompactViews {
    public static func compactLeading(state: DomusLiveActivityAttributes.ContentState) -> some View {
        Image(systemName: state.iconName)
            .foregroundColor(state.isCritical ? .red : .cyan)
            .font(.system(size: 13, weight: .bold))
    }

    public static func compactTrailing(state: DomusLiveActivityAttributes.ContentState) -> some View {
        Group {
            if state.activityType == .evCharging {
                Text("\(state.evBatteryPercent ?? 0)%")
                    .font(.system(size: 13, weight: .bold, design: .rounded))
                    .foregroundColor(.green)
            } else if state.remainingSeconds > 0 {
                Text("\(state.remainingSeconds)s")
                    .font(.system(size: 13, weight: .bold, design: .rounded))
                    .foregroundColor(state.isCritical ? .red : .white)
            } else {
                Text(state.isCritical ? "ALERT" : "OK")
                    .font(.system(size: 11, weight: .heavy))
                    .foregroundColor(state.isCritical ? .red : .green)
            }
        }
    }

    public static func minimal(state: DomusLiveActivityAttributes.ContentState) -> some View {
        Image(systemName: state.iconName)
            .foregroundColor(state.isCritical ? .red : (state.activityType == .evCharging ? .green : .cyan))
            .font(.system(size: 12, weight: .bold))
    }
}

/// Vista espansa per la Dynamic Island quando l'utente tiene premuto sull'isola.
public struct DomusDynamicIslandExpandedView: View {
    public let state: DomusLiveActivityAttributes.ContentState

    public init(state: DomusLiveActivityAttributes.ContentState) {
        self.state = state
    }

    public var body: some View {
        VStack(spacing: 8) {
            HStack {
                HStack(spacing: 6) {
                    Image(systemName: state.iconName)
                        .foregroundColor(state.isCritical ? .red : (state.activityType == .evCharging ? .green : .cyan))
                    Text(state.title)
                        .font(.headline)
                        .foregroundColor(.white)
                }
                Spacer()
                if let evKw = state.evKw {
                    Text(String(format: "%.1f kW", evKw))
                        .font(.headline.monospacedDigit())
                        .foregroundColor(.green)
                } else if state.remainingSeconds > 0 {
                    Text("\(state.remainingSeconds)s")
                        .font(.headline.monospacedDigit())
                        .foregroundColor(state.isCritical ? .red : .cyan)
                }
            }

            Text(state.subtitle)
                .font(.subheadline)
                .foregroundColor(.white.opacity(0.8))
                .frame(maxWidth: .infinity, alignment: .leading)

            if state.activityType == .evCharging {
                HStack {
                    if let soc = state.evBatteryPercent {
                        Text("\(soc)% Batteria")
                            .font(.caption.bold())
                            .foregroundColor(.green)
                    }
                    Spacer()
                    if let end = state.evEstimatedEndTime {
                        Text("Fine stimata: \(end)")
                            .font(.caption)
                            .foregroundColor(.white.opacity(0.7))
                    }
                }
            }

            if state.progress > 0 {
                ProgressView(value: state.progress)
                    .tint(state.isCritical ? .red : (state.activityType == .evCharging ? .green : .cyan))
            }

            // Interactive quick disarm button on Alarm Countdown
            if state.activityType == .alarmCountdown {
                Button(intent: ControlCoverIntent(entityId: state.targetEntityId ?? "alarm_control_panel.home", action: "disarm")) {
                    HStack(spacing: 6) {
                        Image(systemName: "lock.open.fill")
                        Text("Disarma al Volo")
                    }
                    .font(.caption.bold())
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 6)
                    .background(Color.red.opacity(0.8))
                    .foregroundColor(.white)
                    .cornerRadius(8)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(12)
    }
}

#else

// Fallback leggero per macOS / target non-iOS
public enum DomusActivityType: String, Codable, Sendable {
    case alarmCountdown = "alarm_countdown"
    case applianceCycle = "appliance_cycle"
    case criticalAlert = "critical_alert"
    case evCharging = "ev_charging"
}

public struct DomusLiveActivityAttributes: Sendable {
    public struct ContentState: Codable, Hashable, Sendable {
        public var title: String
        public var subtitle: String
        public var progress: Double
        public var remainingSeconds: Int
        public var iconName: String
        public var activityType: DomusActivityType
        public var isCritical: Bool
        public var targetEntityId: String?
        public var evKw: Double?
        public var evBatteryPercent: Int?
        public var evEstimatedEndTime: String?

        public init(
            title: String,
            subtitle: String,
            progress: Double = 0.0,
            remainingSeconds: Int = 0,
            iconName: String = "shield.fill",
            activityType: DomusActivityType = .alarmCountdown,
            isCritical: Bool = false,
            targetEntityId: String? = nil,
            evKw: Double? = nil,
            evBatteryPercent: Int? = nil,
            evEstimatedEndTime: String? = nil
        ) {
            self.title = title
            self.subtitle = subtitle
            self.progress = progress
            self.remainingSeconds = remainingSeconds
            self.iconName = iconName
            self.activityType = activityType
            self.isCritical = isCritical
            self.targetEntityId = targetEntityId
            self.evKw = evKw
            self.evBatteryPercent = evBatteryPercent
            self.evEstimatedEndTime = evEstimatedEndTime
        }
    }

    public var activityId: String
    public var locationName: String

    public init(activityId: String, locationName: String = "Domus Hub") {
        self.activityId = activityId
        self.locationName = locationName
    }
}

#endif
