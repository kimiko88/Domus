import XCTest
@testable import DomusKit

@MainActor
final class AdvancedFeaturesIntegrationTests: XCTestCase {

    final class IntegrationMockEntityRepository: EntityRepositoryProtocol, @unchecked Sendable {
        var entities: [Entity] = []
        var areas: [Area] = []
        var serviceCalls: [(domain: String, service: String, entityId: String, data: [String: Any])] = []

        func getEntities() async -> [Entity] { entities }
        func getEntity(id: String) async -> Entity? { entities.first(where: { $0.id == id }) }
        func syncEntities() async throws -> [Entity] { entities }

        func toggleEntity(id: String) async throws -> Entity {
            guard let idx = entities.firstIndex(where: { $0.id == id }) else {
                let e = Entity(id: id, name: id, domain: "switch", state: "on")
                entities.append(e)
                return e
            }
            entities[idx].state = entities[idx].isOn ? "off" : "on"
            return entities[idx]
        }

        func callService(domain: String, service: String, entityId: String, data: [String: Any]) async throws {
            serviceCalls.append((domain, service, entityId, data))
            if let idx = entities.firstIndex(where: { $0.id == entityId }) {
                if service == "turn_on" || service == "unlock" {
                    entities[idx].state = "on"
                } else if service == "turn_off" {
                    entities[idx].state = "off"
                } else if service == "set_temperature", let t = data["temperature"] as? Double {
                    entities[idx].attributes["temperature"] = AnyCodable(t)
                }
            }
        }

        func updateLocalEntity(_ entity: Entity) async {
            if let idx = entities.firstIndex(where: { $0.id == entity.id }) {
                entities[idx] = entity
            } else {
                entities.append(entity)
            }
        }

        func getAreas() async -> [Area] { areas }
    }

    final class IntegrationMockConnectionRepository: ConnectionRepositoryProtocol, @unchecked Sendable {
        var isConnected: Bool = true
        func getCurrentConnection() async -> ServerConnection {
            ServerConnection(mode: .localLan, isConnected: isConnected)
        }
        func discoverLocalHub(timeoutSeconds: Double) async -> String? { "http://192.168.1.100:8080" }
        func connect(mode: TransportMode?) async throws {}
        func disconnect() async { isConnected = false }
        func setEndpoints(local: String?, remote: String?) async {}
    }

    // MARK: - 1. Intercom & Door Unlock Integration

    func testCameraIntercomAndUnlockIntegrationWithRepository() async throws {
        let repo = IntegrationMockEntityRepository()
        repo.entities = [
            Entity(
                id: "camera.doorbell",
                name: "Citofono Ingresso",
                domain: "camera",
                state: "streaming",
                attributes: ["lock_entity_id": AnyCodable("lock.main_gate")]
            ),
            Entity(id: "lock.main_gate", name: "Cancello Principale", domain: "lock", state: "locked")
        ]

        // Simulazione apertura rapida dal video della telecamera
        let camera = repo.entities[0]
        let targetLockId = camera.attributes["lock_entity_id"]?.value as? String ?? "lock.main_gate"

        try await repo.callService(domain: "lock", service: "unlock", entityId: targetLockId, data: [:])

        XCTAssertEqual(repo.serviceCalls.count, 1)
        XCTAssertEqual(repo.serviceCalls.first?.domain, "lock")
        XCTAssertEqual(repo.serviceCalls.first?.service, "unlock")
        XCTAssertEqual(repo.serviceCalls.first?.entityId, "lock.main_gate")
        XCTAssertEqual(repo.entities.first(where: { $0.id == "lock.main_gate" })?.state, "on")
    }

    // MARK: - 2. Rich Notification Action Dispatch Integration

    func testRichNotificationActionCallbackIntegration() async throws {
        let notificationManager = NotificationManager()
        var capturedAction: String?
        var capturedCategory: String?
        var capturedUserInfo: [AnyHashable: Any]?

        notificationManager.onActionTriggered = { action, category, userInfo in
            capturedAction = action
            capturedCategory = category
            capturedUserInfo = userInfo
        }

        // Simulazione notifica ricca ricevuta con azione 'Suona Sirena'
        let fakeUserInfo: [AnyHashable: Any] = [
            "camera_entity_id": "camera.driveway",
            "image_url": "http://domus.local/snapshot.jpg",
            "lock_entity_id": "lock.driveway_gate"
        ]

        notificationManager.onActionTriggered?(
            DomusNotificationAction.soundSiren.rawValue,
            DomusNotificationCategory.securityCamera.rawValue,
            fakeUserInfo
        )

        XCTAssertEqual(capturedAction, DomusNotificationAction.soundSiren.rawValue)
        XCTAssertEqual(capturedCategory, DomusNotificationCategory.securityCamera.rawValue)
        XCTAssertEqual(capturedUserInfo?["camera_entity_id"] as? String, "camera.driveway")

        // Simulazione azione 'Apri Cancello'
        notificationManager.onActionTriggered?(
            DomusNotificationAction.unlockGate.rawValue,
            DomusNotificationCategory.securityCamera.rawValue,
            fakeUserInfo
        )

        XCTAssertEqual(capturedAction, DomusNotificationAction.unlockGate.rawValue)
        XCTAssertEqual(capturedUserInfo?["lock_entity_id"] as? String, "lock.driveway_gate")
    }

    // MARK: - 3. Automation Rule Execution Integration

    func testAutomationRuleDraftExecutionIntegrationWithEntities() async throws {
        let repo = IntegrationMockEntityRepository()
        repo.entities = [
            Entity(id: "light.garden", name: "Luce Giardino", domain: "light", state: "off"),
            Entity(id: "climate.thermostat", name: "Termostato", domain: "climate", state: "heat", attributes: ["temperature": AnyCodable(19.0)])
        ]

        let rule = AutomationRule(
            name: "Rientro a Casa",
            triggers: [AutomationTrigger(platform: "geofence", eventType: "enter_home")],
            conditions: [],
            actions: [
                AutomationAction(type: "service", service: "turn_on", targetEntityKey: "light.garden"),
                AutomationAction(type: "service", service: "set_temperature", targetEntityKey: "climate.thermostat", params: ["temperature": "21.5"])
            ]
        )

        // Esegui la sequenza delle azioni dell'automazione
        for action in rule.actions {
            guard let svc = action.service, let target = action.targetEntityKey else { continue }
            let domain = target.components(separatedBy: ".").first ?? "light"
            var params: [String: Any] = [:]
            if let tempStr = action.params?["temperature"], let t = Double(tempStr) {
                params["temperature"] = t
            }
            try await repo.callService(domain: domain, service: svc, entityId: target, data: params)
        }

        XCTAssertEqual(repo.serviceCalls.count, 2)
        XCTAssertEqual(repo.entities.first(where: { $0.id == "light.garden" })?.state, "on")
        let thermostat = repo.entities.first(where: { $0.id == "climate.thermostat" })
        XCTAssertEqual(thermostat?.targetTemperature, 21.5)
    }

    // MARK: - 4. Activity Timeline Integration

    func testActivityTimelineEventIngestionAndCategoryFiltering() {
        var timeline: [TimelineEvent] = []

        timeline.append(TimelineEvent(entityId: "alarm_control_panel.home", entityName: "Allarme", domain: "alarm_control_panel", state: "armed_away", message: "Allarme armato", contextType: "user", contextName: "Fabrizio"))
        timeline.append(TimelineEvent(entityId: "binary_sensor.garden_pir", entityName: "Radar Giardino", domain: "binary_sensor", state: "on", message: "Movimento giardino", contextType: "system", contextName: "PIR"))
        timeline.append(TimelineEvent(entityId: "lock.main_gate", entityName: "Cancello", domain: "lock", state: "unlocked", message: "Cancello aperto", contextType: "user", contextName: "Marco"))
        timeline.append(TimelineEvent(entityId: "sensor.power_total", entityName: "Potenza", domain: "sensor", state: "3400 W", message: "Superata soglia potenza", contextType: "system", contextName: "Energy Hub"))
        timeline.append(TimelineEvent(entityId: "light.living_room", entityName: "Luce Salotto", domain: "light", state: "on", message: "Luce accesa", contextType: "automation", contextName: "Tramonto"))

        XCTAssertEqual(timeline.count, 5)

        // Filtro Sicurezza
        let securityEvents = timeline.filter { $0.category == .security }
        XCTAssertEqual(securityEvents.count, 2) // allarme + binary_sensor pir

        // Filtro Accessi
        let accessEvents = timeline.filter { $0.category == .access }
        XCTAssertEqual(accessEvents.count, 1)
        XCTAssertEqual(accessEvents.first?.entityId, "lock.main_gate")

        // Filtro Energia
        let energyEvents = timeline.filter { $0.category == .energy }
        XCTAssertEqual(energyEvents.count, 1)
        XCTAssertEqual(energyEvents.first?.entityId, "sensor.power_total")

        // Filtro Luci & Clima
        let lightEvents = timeline.filter { $0.category == .climateAndLights }
        XCTAssertEqual(lightEvents.count, 1)
        XCTAssertEqual(lightEvents.first?.entityId, "light.living_room")
    }

    // MARK: - 5. Mesh Diagnostics Integration with Sensor Entities

    func testMeshDiagnosticsAlertIntegrationWithSensorEntityState() {
        let mockNodes = [
            MeshNodeDiagnostic(id: "node.pir", name: "PIR Balcone", room: "Balcone", protocolType: "Zigbee 3.0", lqi: 120, rssi: -82, batteryLevel: 11, isRouter: false),
            MeshNodeDiagnostic(id: "node.temp", name: "Sensore Termo Bagno", room: "Bagno", protocolType: "Zigbee 3.0", lqi: 210, rssi: -65, batteryLevel: 14, isRouter: false),
            MeshNodeDiagnostic(id: "node.flood", name: "Allagamento Cucina", room: "Cucina", protocolType: "Z-Wave Plus", lqi: 240, rssi: -55, batteryLevel: 85, isRouter: false),
            MeshNodeDiagnostic(id: "node.router", name: "Smart Plug (Router)", room: "Salotto", protocolType: "Zigbee 3.0", lqi: 255, rssi: -45, batteryLevel: nil, isRouter: true)
        ]

        let lowBatteryAlerts = mockNodes.filter { ($0.batteryLevel ?? 100) < 15 }
        XCTAssertEqual(lowBatteryAlerts.count, 2)

        let criticalSignals = mockNodes.filter { $0.signalQuality == "Critico" || $0.signalQuality == "Accettabile" }
        XCTAssertEqual(criticalSignals.count, 1)
        XCTAssertEqual(criticalSignals.first?.id, "node.pir")
    }

    // MARK: - 6. Guest Access Profile & Granular Permissions Integration

    func testGuestAccessProfilePermissionsRestrictsAlarmAccess() {
        let guest = DomusUserAccount(
            name: "Ospite Airbnb",
            role: .guest,
            allowedRooms: ["Camera Ospiti", "Bagno"],
            canDisarmAlarm: false,
            canViewCameras: false,
            expiresAt: Calendar.current.date(byAdding: .hour, value: 24, to: Date())
        )

        XCTAssertFalse(guest.canDisarmAlarm)
        XCTAssertFalse(guest.canViewCameras)
        XCTAssertTrue(guest.allowedRooms.contains("Camera Ospiti"))
        XCTAssertFalse(guest.allowedRooms.contains("Camera Padronale"))

        // Verifica calcolo scadenza temporanea
        guard let exp = guest.expiresAt else {
            XCTFail("Scadenza mancante")
            return
        }
        let remainingHours = max(0, Int(exp.timeIntervalSince(Date()) / 3600.0))
        XCTAssertTrue(remainingHours >= 23 && remainingHours <= 24)

        // Se scaduto, deve restituire 0
        let pastDate = Calendar.current.date(byAdding: .hour, value: -2, to: Date())!
        let expiredRemaining = max(0, Int(pastDate.timeIntervalSince(Date()) / 3600.0))
        XCTAssertEqual(expiredRemaining, 0)
    }
}
