import SwiftUI
import WidgetKit
import AppIntents

/// Rappresentazione sintetica di un'entità per il rendering rapido nei Widget.
public struct WidgetEntitySummary: Identifiable, Codable, Sendable {
    public let id: String
    public let name: String
    public let domain: String
    public let isOn: Bool
    public let icon: String
    public let subtitle: String

    public init(id: String, name: String, domain: String, isOn: Bool, icon: String, subtitle: String) {
        self.id = id
        self.name = name
        self.domain = domain
        self.isOn = isOn
        self.icon = icon
        self.subtitle = subtitle
    }
}

/// Dati timeline forniti ai widget della schermata home e blocco.
public struct DomusWidgetEntry: TimelineEntry, Sendable {
    public let date: Date
    public let homeTemperature: Double
    public let activeDevicesCount: Int
    public let isAlarmArmed: Bool
    public let entities: [WidgetEntitySummary]

    public init(
        date: Date = Date(),
        homeTemperature: Double = 0.0,
        activeDevicesCount: Int = 0,
        isAlarmArmed: Bool = false,
        entities: [WidgetEntitySummary] = []
    ) {
        self.date = date
        self.homeTemperature = homeTemperature
        self.activeDevicesCount = activeDevicesCount
        self.isAlarmArmed = isAlarmArmed
        self.entities = entities
    }

    public static let sampleEntities: [WidgetEntitySummary] = [
        WidgetEntitySummary(id: "light.living_room", name: "Salotto", domain: "light", isOn: true, icon: "lightbulb.fill", subtitle: "85%"),
        WidgetEntitySummary(id: "switch.coffee", name: "Caffè", domain: "switch", isOn: false, icon: "powerplug.fill", subtitle: "Spenta"),
        WidgetEntitySummary(id: "switch.gate", name: "Cancello", domain: "switch", isOn: false, icon: "door.left.hand.open", subtitle: "Chiuso"),
        WidgetEntitySummary(id: "alarm.home", name: "Allarme", domain: "alarm", isOn: true, icon: "shield.fill", subtitle: "Armato")
    ]

    public static let sample = DomusWidgetEntry(
        date: Date(),
        homeTemperature: 21.5,
        activeDevicesCount: 3,
        isAlarmArmed: true,
        entities: sampleEntities
    )
}

// MARK: - Widget Schermata Home: Taglia Piccola (systemSmall)

public struct DomusSmallWidgetView: View {
    public let entry: DomusWidgetEntry

    public init(entry: DomusWidgetEntry) {
        self.entry = entry
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Image(systemName: "house.fill")
                    .foregroundColor(.cyan)
                    .font(.caption.bold())
                Text("Domus")
                    .font(.caption.bold())
                Spacer()
                if entry.isAlarmArmed {
                    Image(systemName: "shield.fill")
                        .foregroundColor(.red)
                        .font(.caption2)
                }
            }

            Spacer()

            VStack(alignment: .leading, spacing: 2) {
                Text(String(format: "%.1f°C", entry.homeTemperature))
                    .font(.system(size: 26, weight: .bold, design: .rounded))
                Text("\(entry.activeDevicesCount) accesi")
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }

            if let first = entry.entities.first {
                Button(intent: ToggleEntityIntent(entityId: first.id, entityName: first.name)) {
                    HStack {
                        Image(systemName: first.icon)
                        Text(first.name)
                            .lineLimit(1)
                    }
                    .font(.caption2.bold())
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 6)
                    .background(first.isOn ? Color.cyan : Color.secondary.opacity(0.18))
                    .foregroundColor(first.isOn ? .black : .primary)
                    .clipShape(Capsule())
                }
                .buttonStyle(.plain)
            }
        }
        .padding(12)
        .containerBackground(for: .widget) {
            Color(red: 0.05, green: 0.08, blue: 0.14)
        }
    }
}

// MARK: - Widget Schermata Home: Taglia Media (systemMedium)

public struct DomusMediumWidgetView: View {
    public let entry: DomusWidgetEntry

    public init(entry: DomusWidgetEntry) {
        self.entry = entry
    }

    public var body: some View {
        VStack(spacing: 10) {
            HStack {
                HStack(spacing: 6) {
                    Image(systemName: "house.fill")
                        .foregroundColor(.cyan)
                    Text("Domus Hub")
                        .font(.subheadline.bold())
                }

                Spacer()

                HStack(spacing: 12) {
                    Label(String(format: "%.1f°C", entry.homeTemperature), systemImage: "thermometer.medium")
                        .font(.caption.bold())
                        .foregroundColor(.primary)

                    Label("\(entry.activeDevicesCount) attivi", systemImage: "bolt.fill")
                        .font(.caption.bold())
                        .foregroundColor(.yellow)
                }
            }

            // Griglia di 4 pulsanti interattivi
            HStack(spacing: 8) {
                ForEach(entry.entities.prefix(4)) { item in
                    Button(intent: ToggleEntityIntent(entityId: item.id, entityName: item.name)) {
                        VStack(spacing: 4) {
                            Image(systemName: item.icon)
                                .font(.system(size: 16))
                            Text(item.name)
                                .font(.system(size: 11, weight: .semibold))
                                .lineLimit(1)
                            Text(item.subtitle)
                                .font(.system(size: 9))
                                .foregroundColor(.secondary)
                        }
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                        .padding(.vertical, 8)
                        .background(item.isOn ? Color.cyan.opacity(0.25) : Color.secondary.opacity(0.12))
                        .foregroundColor(item.isOn ? .cyan : .primary)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                        .overlay(
                            RoundedRectangle(cornerRadius: 12)
                                .stroke(item.isOn ? Color.cyan : Color.clear, lineWidth: 1.5)
                        )
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .padding(14)
        .containerBackground(for: .widget) {
            Color(red: 0.05, green: 0.08, blue: 0.14)
        }
    }
}

// MARK: - Widget Schermata di Blocco (Lock Screen)

public struct DomusLockScreenCircularView: View {
    public let entry: DomusWidgetEntry

    public init(entry: DomusWidgetEntry) {
        self.entry = entry
    }

    public var body: some View {
        ZStack {
            AccessoryWidgetBackground()
            VStack(spacing: 1) {
                Image(systemName: "thermometer.medium")
                    .font(.system(size: 10))
                Text(String(format: "%.0f°", entry.homeTemperature))
                    .font(.system(size: 15, weight: .bold, design: .rounded))
            }
        }
    }
}

public struct DomusLockScreenRectangularView: View {
    public let entry: DomusWidgetEntry

    public init(entry: DomusWidgetEntry) {
        self.entry = entry
    }

    public var body: some View {
        HStack(spacing: 8) {
            Image(systemName: entry.isAlarmArmed ? "shield.fill" : "house.fill")
                .font(.title3)
            VStack(alignment: .leading, spacing: 2) {
                Text(entry.isAlarmArmed ? "Allarme Inserito" : "Casa Sicura")
                    .font(.caption.bold())
                Text("Temp: \(entry.homeTemperature, specifier: "%.1f")°C • \(entry.activeDevicesCount) accesi")
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }
        }
    }
}

// MARK: - iOS 18 Control Center Widget

#if os(iOS)
@available(iOS 18.0, *)
public struct DomusMasterLightControl: ControlWidget {
    public init() {}

    public var body: some ControlWidgetConfiguration {
        StaticControlConfiguration(kind: "com.fabriziochimienti.domus.masterlight") {
            ControlWidgetButton(
                action: ToggleEntityIntent(entityId: "light.living_room", entityName: "Luce Master")
            ) {
                Label("Luce Master", systemImage: "lightbulb.fill")
            }
        }

        .displayName("Luce Master Domus")
        .description("Controlla istantaneamente la luce principale dal Centro di Controllo.")
    }
}
#endif
