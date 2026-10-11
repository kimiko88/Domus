import XCTest
@testable import DomusKit

@MainActor
final class EcosystemAndHardwareTests: XCTestCase {

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

    func testAppShortcutsConfiguration() {
        let shortcuts = DomusShortcuts.appShortcuts
        XCTAssertEqual(shortcuts.count, 2)
        XCTAssertEqual(DomusShortcuts.shortcutTileColor, .lightBlue)
    }

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

    func testCarPlayActionTriggering() {
        let carPlay = DomusCarPlayManager.shared
        var receivedAction: (domain: String, entityId: String)?

        carPlay.onCarPlayAction = { domain, entityId in
            receivedAction = (domain, entityId)
        }

        #if canImport(CarPlay) && os(iOS)
        // Se CarPlay è supportato, possiamo invocare la simulazione o trigger
        carPlay.onCarPlayAction?("cover", "cover.garage")
        #else
        carPlay.simulateCarAction(domain: "cover", entityId: "cover.garage")
        #endif

        XCTAssertEqual(receivedAction?.domain, "cover")
        XCTAssertEqual(receivedAction?.entityId, "cover.garage")
    }

    func testPictureInPictureManager() {
        let pip = PictureInPictureManager.shared
        pip.startPictureInPicture(cameraEntityId: "camera.front_door")
        XCTAssertTrue(pip.isPiPActive)
        XCTAssertEqual(pip.activeCameraEntityId, "camera.front_door")

        pip.togglePictureInPicture(cameraEntityId: "camera.front_door")
        XCTAssertFalse(pip.isPiPActive)
        XCTAssertNil(pip.activeCameraEntityId)
    }

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

    func testVpnOnDemandManagerConfiguration() {
        let vpn = VpnOnDemandManager.shared
        vpn.configureOnDemandRules(homeSsid: "Domus_Home_WiFi")
        XCTAssertEqual(vpn.homeSsid, "Domus_Home_WiFi")
        XCTAssertTrue(vpn.isOnDemandEnabled)
    }

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

    func testFloorplanRoomDataStructures() {
        let room = FloorplanRoom(
            id: "living_room",
            name: "Salotto",
            rect: CGRect(x: 0.1, y: 0.1, width: 0.5, height: 0.5),
            lightEntityId: "light.living_room",
            climateEntityId: "climate.living_room"
        )

        XCTAssertEqual(room.name, "Salotto")
        XCTAssertEqual(room.lightEntityId, "light.living_room")
        XCTAssertEqual(room.climateEntityId, "climate.living_room")
    }
}
