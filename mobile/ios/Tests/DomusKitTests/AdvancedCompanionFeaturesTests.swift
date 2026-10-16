import XCTest
@testable import DomusKit

@MainActor
final class AdvancedCompanionFeaturesTests: XCTestCase {

    // MARK: - 1. iOS 18 Control Center Widgets & Intents

    func testControlWidgetsAndIntents() async throws {
        let toggleIntent = ControlToggleEntityIntent(entityId: "light.living_room")
        XCTAssertEqual(toggleIntent.entityId, "light.living_room")

        let sceneIntent = RunSceneIntent(sceneId: "scene.goodnight")
        XCTAssertEqual(sceneIntent.sceneId, "scene.goodnight")

        // Simulazione esecuzione intent
        let toggleResult = try await toggleIntent.perform()
        XCTAssertNotNil(toggleResult)

        let sceneResult = try await sceneIntent.perform()
        XCTAssertNotNil(sceneResult)
    }

    // MARK: - 2. Citofono & Audio Bidirezionale / Sblocco Porta

    func testCameraIntercomAttributesAndLockAssociation() {
        let entity = Entity(
            id: "camera.front_door",
            name: "Videocitofono Ingresso",
            domain: "camera",
            state: "idle",
            attributes: [
                "lock_entity_id": AnyCodable("lock.main_gate"),
                "two_way_audio": AnyCodable(true),
                "still_image_url": AnyCodable("http://domus.local/snapshot.jpg")
            ]
        )

        XCTAssertTrue(entity.isCamera)
        XCTAssertEqual(entity.attributes["lock_entity_id"]?.value as? String, "lock.main_gate")
        XCTAssertEqual(entity.cameraImageUrl, "http://domus.local/snapshot.jpg")
    }

    // MARK: - 3. Builder Visivo di Scenari & Automazioni

    func testAutomationRuleEncodingDecodingAndDisplay() throws {
        let trigger = AutomationTrigger(platform: "time", at: "07:30:00")
        XCTAssertEqual(trigger.displayTitle, "Orario: 07:30:00")
        XCTAssertEqual(trigger.displayIcon, "clock.fill")

        let condition = AutomationCondition(condition: "state", entityKey: "alarm_control_panel.home", state: "armed_home")
        XCTAssertTrue(condition.displayTitle.contains("alarm_control_panel.home"))

        let action = AutomationAction(type: "service", service: "turn_on", targetEntityKey: "light.living_room")
        XCTAssertEqual(action.displayTitle, "Accendi / Attiva light.living_room")

        let rule = AutomationRule(
            name: "Luce Mattutina",
            description: "Accende la luce al mattino",
            enabled: true,
            triggers: [trigger],
            conditions: [condition],
            actions: [action]
        )

        let encoder = JSONEncoder()
        let data = try encoder.encode(rule)
        XCTAssertFalse(data.isEmpty)

        let decoder = JSONDecoder()
        let decoded = try decoder.decode(AutomationRule.self, from: data)
        XCTAssertEqual(decoded.name, "Luce Mattutina")
        XCTAssertEqual(decoded.triggers.count, 1)
        XCTAssertEqual(decoded.actions.count, 1)
        XCTAssertEqual(decoded.conditions.count, 1)
    }

    func testAutomationTriggerPlatforms() {
        let sunTrigger = AutomationTrigger(platform: "sun", eventType: "sunset")
        XCTAssertEqual(sunTrigger.displayIcon, "sun.max.fill")

        let motionTrigger = AutomationTrigger(platform: "state", entityKey: "binary_sensor.motion", to: "on")
        XCTAssertEqual(motionTrigger.displayIcon, "figure.walk.motion")

        let geoTrigger = AutomationTrigger(platform: "geofence", eventType: "leave_home")
        XCTAssertEqual(geoTrigger.displayIcon, "location.fill")
    }

    // MARK: - 4. Notifiche Ricche con Snapshot Telecamera

    func testRichCameraNotificationActions() {
        XCTAssertEqual(DomusNotificationAction.soundSiren.rawValue, "ACTION_SOUND_SIREN")
        XCTAssertEqual(DomusNotificationAction.falseAlarm.rawValue, "ACTION_FALSE_ALARM")
        XCTAssertEqual(DomusNotificationAction.unlockGate.rawValue, "ACTION_UNLOCK_GATE")
        XCTAssertEqual(DomusNotificationAction.viewCamera.rawValue, "ACTION_VIEW_CAMERA")

        XCTAssertEqual(DomusNotificationCategory.securityCamera.rawValue, "SECURITY_CAMERA")
    }

    // MARK: - 5. Registro Eventi Storici & Audit Log (Timeline della Casa)

    func testHomeActivityTimelineFiltering() {
        let events = TimelineEvent.sampleTimeline
        XCTAssertGreaterThan(events.count, 2)

        let securityEvents = events.filter { $0.category == .security }
        let accessEvents = events.filter { $0.category == .access }
        let energyEvents = events.filter { $0.category == .energy }

        XCTAssertFalse(securityEvents.isEmpty)
        XCTAssertFalse(accessEvents.isEmpty)
        XCTAssertFalse(energyEvents.isEmpty)

        let first = events[0]
        XCTAssertFalse(first.formattedTime.isEmpty)
    }

    // MARK: - 6. Diagnostica Mesh & Hardware

    func testMeshNodeDiagnosticsSignalAndBatteryThresholds() {
        let nodes = MeshNodeDiagnostic.sampleNodes

        let excellentNode = nodes.first(where: { $0.rssi >= -60 })
        XCTAssertNotNil(excellentNode)
        XCTAssertEqual(excellentNode?.signalQuality, "Eccellente")

        let criticalBattery = nodes.filter { ($0.batteryLevel ?? 100) < 15 }
        XCTAssertEqual(criticalBattery.count, 2)
        XCTAssertTrue(criticalBattery.allSatisfy { ($0.batteryLevel ?? 100) < 15 })
    }

    // MARK: - 7. StandBy Mode Widget Entry

    func testStandByWidgetEntryData() {
        let entry = DomusWidgetEntry.sample
        XCTAssertEqual(entry.homeTemperature, 21.5)
        XCTAssertTrue(entry.isAlarmArmed)
        XCTAssertEqual(entry.activeDevicesCount, 3)
    }

    // MARK: - 8. Accesso Ospiti & Multi-Account

    func testGuestAccessRolesAndDuration() {
        let accounts = DomusUserAccount.sampleAccounts

        let admin = accounts.first(where: { $0.role == .admin })
        XCTAssertNotNil(admin)
        XCTAssertTrue(admin?.canDisarmAlarm ?? false)
        XCTAssertTrue(admin?.canViewCameras ?? false)

        let child = accounts.first(where: { $0.role == .child })
        XCTAssertNotNil(child)
        XCTAssertFalse(child?.canDisarmAlarm ?? true)
        XCTAssertFalse(child?.canViewCameras ?? true)

        let guest = accounts.first(where: { $0.role == .guest })
        XCTAssertNotNil(guest)
        XCTAssertNotNil(guest?.expiresAt)
        XCTAssertFalse(guest?.canDisarmAlarm ?? true)
    }
}
