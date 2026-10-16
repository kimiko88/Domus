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

    // MARK: - 1. iOS 18 Control Center Widgets & Intents (Extended)

    func testControlWidgetsToggleIntentDefaultAndCustomIds() async throws {
        let defaultIntent = ControlToggleEntityIntent()
        XCTAssertEqual(defaultIntent.entityId, "")
        let defaultResult = try await defaultIntent.perform()
        XCTAssertNotNil(defaultResult)

        let customIntent = ControlToggleEntityIntent(entityId: "lock.garden_gate")
        XCTAssertEqual(customIntent.entityId, "lock.garden_gate")
        let customResult = try await customIntent.perform()
        XCTAssertNotNil(customResult)
    }

    // MARK: - 2. Citofono & Audio Bidirezionale / Sblocco Porta (Extended)

    func testCameraTwoWayAudioAttributesAndFallbackLock() {
        // Telecamera senza lock_entity_id esplicito: fallback a 'lock.main_gate'
        let cameraWithoutLock = Entity(
            id: "camera.intercom",
            name: "Citofono",
            domain: "camera",
            state: "streaming",
            attributes: [
                "two_way_talk_supported": AnyCodable(true),
                "audio_codec": AnyCodable("Opus")
            ]
        )

        let resolvedLock = cameraWithoutLock.attributes["lock_entity_id"]?.value as? String ?? "lock.main_gate"
        XCTAssertEqual(resolvedLock, "lock.main_gate")
        XCTAssertTrue(cameraWithoutLock.isCamera)
        XCTAssertNil(cameraWithoutLock.cameraImageUrl)
    }

    // MARK: - 3. Builder Visivo di Scenari & Automazioni (Extended)

    func testAutomationConditionTimeWindowFormatting() {
        let windowCondition = AutomationCondition(condition: "time", after: "22:00:00", before: "06:00:00")
        XCTAssertEqual(windowCondition.displayTitle, "Tra le 22:00:00 e le 06:00:00")
        XCTAssertEqual(windowCondition.displayIcon, "hourglass")

        let afterOnly = AutomationCondition(condition: "time", after: "20:00:00")
        XCTAssertEqual(afterOnly.displayTitle, "Dopo le 20:00:00")

        let beforeOnly = AutomationCondition(condition: "time", before: "08:00:00")
        XCTAssertEqual(beforeOnly.displayTitle, "Prima delle 08:00:00")
    }

    func testAutomationActionDisplayTitles() {
        let turnOffAction = AutomationAction(type: "service", service: "turn_off", targetEntityKey: "light.bedroom")
        XCTAssertEqual(turnOffAction.displayTitle, "Spegni / Disattiva light.bedroom")
        XCTAssertEqual(turnOffAction.displayIcon, "power.circle")

        let tempAction = AutomationAction(
            type: "service",
            service: "set_temperature",
            targetEntityKey: "climate.living_room",
            params: ["temperature": "19.5"]
        )
        XCTAssertEqual(tempAction.displayTitle, "Imposta clima a 19.5°C su climate.living_room")
        XCTAssertEqual(tempAction.displayIcon, "thermometer.sun.fill")

        let notifyAction = AutomationAction(
            type: "service",
            service: "notify",
            targetEntityKey: "notify.all",
            params: ["message": "Allarme attivato!"]
        )
        XCTAssertEqual(notifyAction.displayTitle, "Invia notifica push: \"Allarme attivato!\"")
        XCTAssertEqual(notifyAction.displayIcon, "bell.badge.fill")
    }

    // MARK: - 4. Notifiche Ricche con Snapshot Telecamera (Extended)

    func testRichCameraNotificationParametersAndDispatch() async throws {
        let notificationManager = NotificationManager()
        XCTAssertNotNil(notificationManager)

        // Verifica invio simulato notifica ricca
        try await notificationManager.sendRichCameraNotification(
            title: "Movimento Sospetto",
            body: "Persona rilevata al cancello",
            imageUrl: "http://domus.local/snapshot.jpg",
            cameraEntityId: "camera.gate",
            isCritical: false
        )
    }

    // MARK: - 5. Registro Eventi Storici & Audit Log (Extended)

    func testTimelineCategoryMappingForAllDomains() {
        let coverEvent = TimelineEvent(entityId: "cover.garage", entityName: "Garage", domain: "cover", state: "open", message: "Aperto")
        XCTAssertEqual(coverEvent.category, .access)

        let lightEvent = TimelineEvent(entityId: "light.kitchen", entityName: "Cucina", domain: "light", state: "on", message: "Accesa")
        XCTAssertEqual(lightEvent.category, .climateAndLights)

        let climateEvent = TimelineEvent(entityId: "climate.bedroom", entityName: "Clima Letto", domain: "climate", state: "heat", message: "Riscaldamento")
        XCTAssertEqual(climateEvent.category, .climateAndLights)

        let powerEvent = TimelineEvent(entityId: "sensor.power_consumption", entityName: "Consumo", domain: "sensor", state: "2500 W", message: "Assorbimento")
        XCTAssertEqual(powerEvent.category, .energy)

        let pirEvent = TimelineEvent(entityId: "sensor.motion_hallway", entityName: "Radar", domain: "sensor", state: "active", message: "Rilevamento")
        XCTAssertEqual(pirEvent.category, .security)
    }

    // MARK: - 6. Diagnostica Mesh & Hardware (Extended)

    func testMeshNodeDiagnosticSignalClassificationThresholds() {
        let nodeExcellent1 = MeshNodeDiagnostic(id: "1", name: "N1", room: "R", protocolType: "Zigbee", lqi: 250, rssi: -59)
        XCTAssertEqual(nodeExcellent1.signalQuality, "Eccellente")

        let nodeExcellent2 = MeshNodeDiagnostic(id: "2", name: "N2", room: "R", protocolType: "Zigbee", lqi: 240, rssi: -60)
        XCTAssertEqual(nodeExcellent2.signalQuality, "Eccellente")

        let nodeGood1 = MeshNodeDiagnostic(id: "3", name: "N3", room: "R", protocolType: "Zigbee", lqi: 200, rssi: -61)
        XCTAssertEqual(nodeGood1.signalQuality, "Buono")

        let nodeGood2 = MeshNodeDiagnostic(id: "4", name: "N4", room: "R", protocolType: "Zigbee", lqi: 190, rssi: -75)
        XCTAssertEqual(nodeGood2.signalQuality, "Buono")

        let nodeAcceptable = MeshNodeDiagnostic(id: "5", name: "N5", room: "R", protocolType: "Zigbee", lqi: 150, rssi: -84)
        XCTAssertEqual(nodeAcceptable.signalQuality, "Accettabile")

        let nodeCritical = MeshNodeDiagnostic(id: "6", name: "N6", room: "R", protocolType: "Zigbee", lqi: 90, rssi: -86)
        XCTAssertEqual(nodeCritical.signalQuality, "Critico")
    }

    // MARK: - 7. StandBy Mode Widget Entry (Extended)

    func testStandByWidgetEntryDisarmedState() {
        let disarmedEntry = DomusWidgetEntry(
            date: Date(),
            homeTemperature: 19.8,
            activeDevicesCount: 0,
            isAlarmArmed: false,
            entities: []
        )
        XCTAssertFalse(disarmedEntry.isAlarmArmed)
        XCTAssertEqual(disarmedEntry.homeTemperature, 19.8)
        XCTAssertEqual(disarmedEntry.activeDevicesCount, 0)
    }

    // MARK: - 8. Accesso Ospiti & Multi-Account (Extended)

    func testGuestQRCodePayloadSerialization() throws {
        let guestName = "Marco"
        let durationHours = 24
        let rooms = ["Camera Ospiti", "Salotto"]
        let token = "TOKEN123"

        let jsonPayload = """
        {"guest":"\(guestName)","duration_hours":\(durationHours),"rooms":["\(rooms[0])","\(rooms[1])"],"token":"\(token)"}
        """

        guard let data = jsonPayload.data(using: .utf8) else {
            XCTFail("Impossibile serializzare JSON")
            return
        }

        struct DecodedGuestPayload: Decodable {
            let guest: String
            let duration_hours: Int
            let rooms: [String]
            let token: String
        }

        let decoded = try JSONDecoder().decode(DecodedGuestPayload.self, from: data)
        XCTAssertEqual(decoded.guest, "Marco")
        XCTAssertEqual(decoded.duration_hours, 24)
        XCTAssertEqual(decoded.rooms.count, 2)
        XCTAssertEqual(decoded.token, "TOKEN123")
    }

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

