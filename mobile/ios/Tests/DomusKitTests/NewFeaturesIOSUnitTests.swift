import XCTest
@testable import DomusKit

@MainActor
final class NewFeaturesIOSUnitTests: XCTestCase {

    func testAppIntentsInitialization() {
        let coverIntent = ControlCoverIntent(entityId: "cover.garage", action: "open")
        XCTAssertEqual(coverIntent.entityId, "cover.garage")
        XCTAssertEqual(coverIntent.action, "open")

        let climateIntent = SetClimateTemperatureIntent(entityId: "climate.living_room", temperature: 21.5)
        XCTAssertEqual(climateIntent.entityId, "climate.living_room")
        XCTAssertEqual(climateIntent.temperature, 21.5)
    }

    func testDomusShortcutsConfigured() {
        let shortcuts = DomusShortcuts.appShortcuts
        XCTAssertGreaterThanOrEqual(shortcuts.count, 4)
    }

    func testLiveActivityEVCharging() {
        let manager = DomusLiveActivityManager.shared
        manager.startEVCharging(carName: "Tesla Model 3", currentKw: 11.0, batteryPercent: 82, estimatedEnd: "16:00")

        XCTAssertTrue(manager.isActivityActive)
        XCTAssertEqual(manager.activeActivityType, .evCharging)
        XCTAssertEqual(manager.currentActivityTitle, "Ricarica Tesla Model 3")

        manager.stopActivity()
        XCTAssertFalse(manager.isActivityActive)
    }

    func testCarPlayDynamicStateBadges() {
        let carPlay = DomusCarPlayManager.shared
        carPlay.updateLiveState(gateOpen: true, garageOpen: false, lightsOn: true, alarmArmed: true, alarmMode: "Armato Perimetrale")

        XCTAssertTrue(carPlay.isGateOpen)
        XCTAssertFalse(carPlay.isGarageOpen)
        XCTAssertTrue(carPlay.isOutdoorLightsOn)
        XCTAssertTrue(carPlay.isAlarmArmed)
        XCTAssertEqual(carPlay.alarmMode, "Armato Perimetrale")
    }

    func testNfcTagManagerNDEFWriteSimulation() {
        let nfc = NfcTagManager()
        let result = nfc.simulateTagWrite(uri: "domus://action/cover.gate")
        XCTAssertEqual(result, "domus://action/cover.gate")
        XCTAssertTrue(nfc.lastStatusMessage.contains("programmato con successo"))
    }

    func testWatchThermostatDigitalCrown() {
        let thermostat = DomusWatchThermostatState(current: 20.0, target: 20.5)
        XCTAssertEqual(thermostat.targetTemperature, 20.5)

        // Rotate crown clockwise by +1.0°C
        thermostat.rotateCrown(deltaDegrees: 1.0)
        XCTAssertEqual(thermostat.targetTemperature, 21.5)

        // Rotate crown counter-clockwise by -0.5°C
        thermostat.rotateCrown(deltaDegrees: -0.5)
        XCTAssertEqual(thermostat.targetTemperature, 21.0)
    }

    func testWatchSmartStackHomeArrivalRelevance() {
        let smartStack = DomusSmartStackHomeArrivalModel.shared
        smartStack.isNearHome = true
        XCTAssertEqual(smartStack.smartStackRelevanceScore, 1.0)
        XCTAssertEqual(smartStack.suggestedActions.count, 3)

        smartStack.isNearHome = false
        XCTAssertEqual(smartStack.smartStackRelevanceScore, 0.1)
    }
}
