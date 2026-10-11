import XCTest
@testable import DomusKit

@MainActor
final class Phase4And5Tests: XCTestCase {

    // MARK: - Fase 4: Widget & AppIntents Tests

    func testAppIntentsParameters() {
        let toggleIntent = ToggleEntityIntent(entityId: "light.bedroom", entityName: "Luce Camera")
        XCTAssertEqual(toggleIntent.entityId, "light.bedroom")
        XCTAssertEqual(toggleIntent.entityName, "Luce Camera")

        let sceneIntent = RunSceneIntent(sceneId: "good_morning")
        XCTAssertEqual(sceneIntent.sceneId, "good_morning")
    }

    func testWidgetEntryDefaultsAndSamples() {
        let entry = DomusWidgetEntry()
        XCTAssertEqual(entry.homeTemperature, 21.5)
        XCTAssertEqual(entry.activeDevicesCount, 3)
        XCTAssertTrue(entry.isAlarmArmed)
        XCTAssertEqual(entry.entities.count, 4)

        let first = entry.entities.first
        XCTAssertEqual(first?.id, "light.living_room")
        XCTAssertEqual(first?.domain, "light")
        XCTAssertTrue(first?.isOn == true)
    }

    // MARK: - Fase 4: WatchComplication Tests

    func testWatchComplicationSnapshot() {
        let snapshot = WatchComplicationSnapshot(
            indoorTemperature: 22.0,
            instantPowerKW: 1.25,
            isAlarmArmed: false
        )
        XCTAssertEqual(snapshot.indoorTemperature, 22.0)
        XCTAssertEqual(snapshot.instantPowerKW, 1.25)
        XCTAssertFalse(snapshot.isAlarmArmed)
    }

    // MARK: - Fase 5: Kiosk Mode & Battery Protection Tests

    func testKioskModeManagerBatteryRule() {
        let kiosk = KioskModeManager()
        kiosk.smartPlugEntityId = "switch.wall_charger"
        kiosk.isBatteryProtectionEnabled = true

        var lastPlugCommand: (id: String, state: Bool)?
        kiosk.onSmartPlugCommand = { id, state in
            lastPlugCommand = (id, state)
        }

        // Test attivazione Kiosk
        kiosk.setKioskMode(enabled: true)
        XCTAssertTrue(kiosk.isKioskModeEnabled)
        XCTAssertEqual(kiosk.displayState, .fullDashboard)

        // Test ingresso e risveglio dallo screensaver
        kiosk.enterScreensaver()
        XCTAssertEqual(kiosk.displayState, .ambientScreensaver)

        kiosk.wakeFromScreensaver()
        XCTAssertEqual(kiosk.displayState, .fullDashboard)

        // Disattiva Kiosk
        kiosk.setKioskMode(enabled: false)
        XCTAssertFalse(kiosk.isKioskModeEnabled)
        _ = lastPlugCommand
    }

}
