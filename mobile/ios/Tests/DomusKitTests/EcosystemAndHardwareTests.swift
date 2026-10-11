import XCTest
@testable import DomusKit

@MainActor
final class EcosystemAndHardwareTests: XCTestCase {

    // MARK: - Biometric Authentication Tests (Face ID / Touch ID)

    func testBiometricAuthenticationSimulation() async {
        let bioManager = LocalAuthenticationManager(isSimulationMode: true)
        XCTAssertEqual(bioManager.biometryType, .faceID)
        XCTAssertTrue(bioManager.isBiometryAvailable)

        let result = await bioManager.authenticate(reason: "Test Face ID")
        switch result {
        case .success(let authenticated):
            XCTAssertTrue(authenticated)
            XCTAssertTrue(bioManager.lastAuthSuccess)
            XCTAssertNil(bioManager.lastErrorMessage)
        case .failure:
            XCTFail("Autenticazione biometrica simulata dovrebbe avere successo")
        }
    }

    func testBiometricAuthenticationNonSimulatedState() {
        let bioManager = LocalAuthenticationManager(isSimulationMode: false)
        // Sul runtime di test o CI senza hardware biometrico registrato, non deve crashare
        XCTAssertNotNil(bioManager.biometryType)
        bioManager.refreshBiometryStatus()
    }

    // MARK: - App Shortcuts & Siri Tests

    func testAppShortcutsConfiguration() {
        let shortcuts = DomusShortcuts.appShortcuts
        XCTAssertEqual(shortcuts.count, 2)
        XCTAssertEqual(DomusShortcuts.shortcutTileColor, .lightBlue)
    }

    // MARK: - Live Activities & Dynamic Island Tests

    func testLiveActivityContentStateSerialization() throws {
        let state = DomusLiveActivityAttributes.ContentState(
            title: "Ritardo Allarme",
            subtitle: "30 secondi rimanenti",
            progress: 0.5,
            remainingSeconds: 15,
            iconName: "shield.fill",
            activityType: .alarmCountdown,
            isCritical: true,
            targetEntityId: "alarm_control_panel.home"
        )

        let data = try JSONEncoder().encode(state)
        let decoded = try JSONDecoder().decode(DomusLiveActivityAttributes.ContentState.self, from: data)

        XCTAssertEqual(decoded.title, "Ritardo Allarme")
        XCTAssertEqual(decoded.remainingSeconds, 15)
        XCTAssertEqual(decoded.activityType, .alarmCountdown)
        XCTAssertTrue(decoded.isCritical)
        XCTAssertEqual(decoded.targetEntityId, "alarm_control_panel.home")
    }

    func testLiveActivityManagerTransitions() {
        let manager = DomusLiveActivityManager.shared
        manager.startAlarmCountdown(seconds: 30, isArming: true)
        XCTAssertTrue(manager.isActivityActive)
        XCTAssertEqual(manager.activeActivityType, .alarmCountdown)
        XCTAssertEqual(manager.currentActivityTitle, "Armamento Allarme")

        manager.updateState(remainingSeconds: 20, progress: 0.66)
        XCTAssertTrue(manager.isActivityActive)

        manager.stopActivity()
        XCTAssertFalse(manager.isActivityActive)
        XCTAssertNil(manager.currentActivityTitle)
        XCTAssertNil(manager.activeActivityType)
    }

    func testLiveActivityApplianceAndCriticalAlerts() {
        let manager = DomusLiveActivityManager.shared

        // Test Appliance Cycle
        manager.startApplianceCycle(name: "Lavatrice Smart", remainingMinutes: 45, iconName: "washer.fill")
        XCTAssertTrue(manager.isActivityActive)
        XCTAssertEqual(manager.activeActivityType, .applianceCycle)
        XCTAssertEqual(manager.currentActivityTitle, "Lavatrice Smart")

        manager.stopActivity()
        XCTAssertFalse(manager.isActivityActive)

        // Test Critical Alert (Allagamento o Gas)
        manager.startCriticalAlert(title: "Allagamento Lavanderia", detail: "Sensore acqua attivo", iconName: "exclamationmark.triangle.fill")
        XCTAssertTrue(manager.isActivityActive)
        XCTAssertEqual(manager.activeActivityType, .criticalAlert)
        XCTAssertEqual(manager.currentActivityTitle, "Allagamento Lavanderia")

        manager.stopActivity()
        XCTAssertFalse(manager.isActivityActive)
    }

    // MARK: - CarPlay Tests

    func testCarPlayActionTriggering() {
        let carPlay = DomusCarPlayManager.shared
        var receivedAction: (domain: String, entityId: String)?

        carPlay.onCarPlayAction = { domain, entityId in
            receivedAction = (domain, entityId)
        }

        #if canImport(CarPlay) && os(iOS)
        carPlay.onCarPlayAction?("cover", "cover.garage")
        #else
        carPlay.simulateCarAction(domain: "cover", entityId: "cover.garage")
        #endif

        XCTAssertEqual(receivedAction?.domain, "cover")
        XCTAssertEqual(receivedAction?.entityId, "cover.garage")
    }

    func testCarPlayMultipleQuickActions() {
        let carPlay = DomusCarPlayManager.shared
        var actionsCount = 0

        carPlay.onCarPlayAction = { domain, entityId in
            actionsCount += 1
        }

        #if canImport(CarPlay) && os(iOS)
        carPlay.onCarPlayAction?("cover", "cover.gate")
        carPlay.onCarPlayAction?("light", "light.outdoor_lights")
        carPlay.onCarPlayAction?("alarm_control_panel", "alarm_control_panel.home")
        #else
        carPlay.simulateCarAction(domain: "cover", entityId: "cover.gate")
        carPlay.simulateCarAction(domain: "light", entityId: "light.outdoor_lights")
        carPlay.simulateCarAction(domain: "alarm_control_panel", entityId: "alarm_control_panel.home")
        #endif

        XCTAssertEqual(actionsCount, 3)
    }

    // MARK: - Picture-in-Picture Tests

    func testPictureInPictureManager() {
        let pip = PictureInPictureManager.shared
        pip.startPictureInPicture(cameraEntityId: "camera.front_door")
        XCTAssertTrue(pip.isPiPActive)
        XCTAssertEqual(pip.activeCameraEntityId, "camera.front_door")

        pip.togglePictureInPicture(cameraEntityId: "camera.front_door")
        XCTAssertFalse(pip.isPiPActive)
        XCTAssertNil(pip.activeCameraEntityId)
    }

    func testPictureInPictureCameraSwitching() {
        let pip = PictureInPictureManager.shared
        pip.startPictureInPicture(cameraEntityId: "camera.front_door")
        XCTAssertEqual(pip.activeCameraEntityId, "camera.front_door")

        // Switch diretto a telecamera giardino
        pip.startPictureInPicture(cameraEntityId: "camera.garden")
        XCTAssertTrue(pip.isPiPActive)
        XCTAssertEqual(pip.activeCameraEntityId, "camera.garden")

        pip.stopPictureInPicture()
        XCTAssertFalse(pip.isPiPActive)
    }

    // MARK: - Indoor Beacon Micro-localization Tests

    func testRoomPresenceBeaconDetection() {
        let beaconManager = RoomPresenceBeaconManager(configuredBeacons: [
            DomusRoomBeacon(id: "b1", roomId: "living_room", roomName: "Salotto", major: 1, minor: 1, rssiThreshold: -65),
            DomusRoomBeacon(id: "b2", roomId: "kitchen", roomName: "Cucina", major: 1, minor: 2, rssiThreshold: -60)
        ])

        var roomNotified: String?
        beaconManager.onRoomChanged = { room in
            roomNotified = room
        }

        // Segnale debole sotto soglia: non deve attivare
        beaconManager.evaluateRssiSample(major: 1, minor: 1, rssi: -75)
        XCTAssertNil(beaconManager.activeRoomId)

        // Segnale forte sopra soglia per Salotto
        beaconManager.evaluateRssiSample(major: 1, minor: 1, rssi: -58)
        XCTAssertEqual(beaconManager.activeRoomId, "living_room")
        XCTAssertEqual(beaconManager.activeRoomName, "Salotto")
        XCTAssertEqual(roomNotified, "living_room")
        XCTAssertNotNil(beaconManager.currentDistanceMeters)

        // Spostamento in Cucina
        beaconManager.evaluateRssiSample(major: 1, minor: 2, rssi: -55)
        XCTAssertEqual(beaconManager.activeRoomId, "kitchen")
        XCTAssertEqual(beaconManager.activeRoomName, "Cucina")
        XCTAssertEqual(roomNotified, "kitchen")
    }

    func testRoomPresenceBeaconStopMonitoringAndSimulation() {
        let beaconManager = RoomPresenceBeaconManager()
        beaconManager.startMonitoring()
        XCTAssertTrue(beaconManager.isScanning)

        beaconManager.simulateRoomProximity(roomId: "living_room", distanceMeters: 0.5)
        XCTAssertEqual(beaconManager.activeRoomId, "living_room")
        XCTAssertEqual(beaconManager.activeRoomName, "Salotto")
        XCTAssertEqual(beaconManager.currentDistanceMeters, 0.5)

        beaconManager.stopMonitoring()
        XCTAssertFalse(beaconManager.isScanning)
        XCTAssertNil(beaconManager.activeRoomId)
        XCTAssertNil(beaconManager.activeRoomName)
        XCTAssertNil(beaconManager.currentDistanceMeters)
    }

    // MARK: - VPN On-Demand Tests

    func testVpnOnDemandManagerConfiguration() {
        let vpn = VpnOnDemandManager.shared
        vpn.configureOnDemandRules(homeSsid: "Domus_Home_WiFi")
        XCTAssertEqual(vpn.homeSsid, "Domus_Home_WiFi")
        XCTAssertTrue(vpn.isOnDemandEnabled)
    }

    func testVpnOnDemandConnectionCycle() {
        let vpn = VpnOnDemandManager(homeSsid: "Domus_Test_Network", isOnDemandEnabled: true)
        XCTAssertEqual(vpn.homeSsid, "Domus_Test_Network")
        XCTAssertTrue(vpn.isOnDemandEnabled)

        vpn.connect()
        #if canImport(NetworkExtension)
        XCTAssertTrue(vpn.status == .connecting || vpn.status == .connected || vpn.lastErrorMessage != nil)
        #else
        XCTAssertEqual(vpn.status, .connected)
        #endif

        vpn.disconnect()
        XCTAssertEqual(vpn.status, .disconnected)
    }

    // MARK: - Background Maintenance Tasks Tests

    func testBackgroundTaskManagerExecution() async {
        let bg = DomusBackgroundTaskManager.shared
        bg.onMaintenanceTriggered = {
            return "Test manutenzione completata con successo"
        }

        let summary = await bg.executeMaintenanceNow()
        XCTAssertEqual(summary, "Test manutenzione completata con successo")
        XCTAssertEqual(bg.lastMaintenanceSummary, "Test manutenzione completata con successo")
        XCTAssertNotNil(bg.lastMaintenanceDate)
        XCTAssertFalse(bg.isMaintenanceRunning)
    }

    func testBackgroundTaskManagerDefaultExecution() async {
        let bg = DomusBackgroundTaskManager()
        bg.onMaintenanceTriggered = nil

        let summary = await bg.executeMaintenanceNow()
        XCTAssertTrue(summary.contains("Cache telecamere pulita"))
        XCTAssertEqual(bg.lastMaintenanceSummary, summary)
        XCTAssertNotNil(bg.lastMaintenanceDate)
        XCTAssertFalse(bg.isMaintenanceRunning)
    }

    // MARK: - 2D Floorplan Room Geometry & Data Structures Tests

    func testFloorplanRoomDataStructures() {
        let room = FloorplanRoom(
            id: "living_room",
            name: "Salotto",
            rect: CGRect(x: 0.1, y: 0.1, width: 0.5, height: 0.5),
            lightEntityId: "light.living_room",
            climateEntityId: "climate.living_room",
            motionEntityId: "binary_sensor.motion_living_room"
        )

        XCTAssertEqual(room.name, "Salotto")
        XCTAssertEqual(room.lightEntityId, "light.living_room")
        XCTAssertEqual(room.climateEntityId, "climate.living_room")
        XCTAssertEqual(room.motionEntityId, "binary_sensor.motion_living_room")
        XCTAssertEqual(room.rect.width, 0.5)
        XCTAssertEqual(room.rect.height, 0.5)
    }

    func testFloorplanRoomHitTesting() {
        let room = FloorplanRoom(
            id: "kitchen",
            name: "Cucina",
            rect: CGRect(x: 0.6, y: 0.1, width: 0.3, height: 0.4)
        )

        let insidePoint = CGPoint(x: 0.7, y: 0.2)
        let outsidePoint = CGPoint(x: 0.1, y: 0.8)

        XCTAssertTrue(room.rect.contains(insidePoint))
        XCTAssertFalse(room.rect.contains(outsidePoint))
    }
}
