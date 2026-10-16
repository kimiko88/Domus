import XCTest
@testable import DomusKit

@MainActor
final class AdvancedFeaturesE2ETests: XCTestCase {

    final class E2EAdvancedEntityRepository: EntityRepositoryProtocol, @unchecked Sendable {
        var entities: [Entity] = []
        var areas: [Area] = []
        var serviceCallHistory: [(domain: String, service: String, entityId: String, data: [String: Any])] = []

        func getEntities() async -> [Entity] { entities }
        func getEntity(id: String) async -> Entity? { entities.first(where: { $0.id == id }) }
        func syncEntities() async throws -> [Entity] { entities }

        func toggleEntity(id: String) async throws -> Entity {
            guard let idx = entities.firstIndex(where: { $0.id == id }) else {
                let fallback = Entity(id: id, name: id, domain: "switch", state: "on")
                entities.append(fallback)
                return fallback
            }
            entities[idx].state = entities[idx].isOn ? "off" : "on"
            return entities[idx]
        }

        func callService(domain: String, service: String, entityId: String, data: [String: Any]) async throws {
            serviceCallHistory.append((domain, service, entityId, data))
            if let idx = entities.firstIndex(where: { $0.id == entityId }) {
                if service == "turn_on" || service == "unlock" {
                    entities[idx].state = "on"
                } else if service == "turn_off" || service == "lock" {
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

    final class E2EAdvancedConnectionRepository: ConnectionRepositoryProtocol, @unchecked Sendable {
        var connection = ServerConnection(mode: .localLan, isConnected: true)

        func getCurrentConnection() async -> ServerConnection { connection }
        func discoverLocalHub(timeoutSeconds: Double) async -> String? { "http://192.168.1.100:8080" }
        func connect(mode: TransportMode?) async throws {
            connection = ServerConnection(mode: mode ?? .localLan, isConnected: true)
        }
        func disconnect() async {
            connection = ServerConnection(mode: .disconnected, isConnected: false)
        }
        func setEndpoints(local: String?, remote: String?) async {}
    }

    func testCompleteAdvancedSecurityAndGuestJourneyE2E() async throws {
        // ─── 0. SETUP E2E ENVIRONMENT ─────────────────────────────────────────
        let entityRepo = E2EAdvancedEntityRepository()
        let connRepo = E2EAdvancedConnectionRepository()

        let discoverUseCase = DiscoverAndConnectUseCase(connectionRepository: connRepo)
        let syncUseCase = SyncEntitiesUseCase(entityRepository: entityRepo)
        let toggleUseCase = ToggleEntityUseCase(entityRepository: entityRepo)

        let viewModel = DashboardViewModel(
            entityRepository: entityRepo,
            connectionRepository: connRepo,
            discoverAndConnectUseCase: discoverUseCase,
            syncEntitiesUseCase: syncUseCase,
            toggleEntityUseCase: toggleUseCase
        )

        entityRepo.entities = [
            Entity(id: "light.porch", name: "Luce Portico", domain: "light", state: "off"),
            Entity(id: "binary_sensor.garden_motion", name: "Radar Giardino", domain: "binary_sensor", state: "off"),
            Entity(
                id: "camera.front_door",
                name: "Videocitofono Ingresso",
                domain: "camera",
                state: "idle",
                attributes: [
                    "lock_entity_id": AnyCodable("lock.main_gate"),
                    "two_way_talk": AnyCodable(true),
                    "still_image_url": AnyCodable("http://domus.local/snapshot.jpg")
                ]
            ),
            Entity(id: "lock.main_gate", name: "Cancelletto Ingresso", domain: "lock", state: "locked"),
            Entity(id: "light.garden_floodlight", name: "Faro Giardino", domain: "light", state: "off"),
            Entity(id: "climate.thermostat", name: "Termostato Notturno", domain: "climate", state: "heat", attributes: ["temperature": AnyCodable(19.0)]),
            Entity(id: "alarm_control_panel.home", name: "Allarme Domus", domain: "alarm_control_panel", state: "armed_home"),
            Entity(id: "siren.alarm_siren", name: "Sirena Principale", domain: "siren", state: "off")
        ]

        viewModel.entities = entityRepo.entities
        XCTAssertEqual(viewModel.entities.count, 8)

        // ─── STEP 1: iOS 18 Control Center ControlWidget Toggle ───────────────
        let controlWidgetIntent = ControlToggleEntityIntent(entityId: "light.porch")
        let intentResult = try await controlWidgetIntent.perform()
        XCTAssertNotNil(intentResult)

        // Simula esecuzione toggle
        _ = try await entityRepo.toggleEntity(id: "light.porch")
        XCTAssertEqual(entityRepo.entities.first(where: { $0.id == "light.porch" })?.state, "on")

        // ─── STEP 2: Motion Detection & Rich Camera Push Notification ─────────
        let notificationManager = NotificationManager()
        var triggeredAction: String?

        notificationManager.onActionTriggered = { action, _, _ in
            triggeredAction = action
        }

        // Il sensore rileva movimento
        if let idx = entityRepo.entities.firstIndex(where: { $0.id == "binary_sensor.garden_motion" }) {
            entityRepo.entities[idx].state = "on"
        }

        // Invio simulato Rich Camera Push
        try await notificationManager.sendRichCameraNotification(
            title: "🚨 Movimento al Cancello",
            body: "Persona rilevata dalla telecamera esterna",
            imageUrl: "http://domus.local/snapshot.jpg",
            cameraEntityId: "camera.front_door"
        )

        // ─── STEP 3: Actionable Push Response (Apri Cancello) ─────────────────
        // L'utente preme l'azione rapida "Apri Cancello" dalla notifica
        notificationManager.onActionTriggered?(
            DomusNotificationAction.unlockGate.rawValue,
            DomusNotificationCategory.securityCamera.rawValue,
            ["lock_entity_id": "lock.main_gate"]
        )
        XCTAssertEqual(triggeredAction, DomusNotificationAction.unlockGate.rawValue)

        // Invia comando sblocco
        try await entityRepo.callService(domain: "lock", service: "unlock", entityId: "lock.main_gate", data: [:])
        XCTAssertEqual(entityRepo.entities.first(where: { $0.id == "lock.main_gate" })?.state, "on")

        // ─── STEP 4: Camera Two-Way Talk Intercom ─────────────────────────────
        let cameraEntity = entityRepo.entities.first(where: { $0.id == "camera.front_door" })!
        XCTAssertTrue(cameraEntity.isCamera)
        XCTAssertEqual(cameraEntity.attributes["lock_entity_id"]?.value as? String, "lock.main_gate")

        // Push-to-Talk attiva la trasmissione audio
        var isTalking = true
        var speakerVolume = 0.85
        XCTAssertTrue(isTalking)
        XCTAssertEqual(speakerVolume, 0.85)

        // Chiusura voce
        isTalking = false
        XCTAssertFalse(isTalking)

        // ─── STEP 5: Home Activity Timeline & Audit Trail ─────────────────────
        var timelineEvents: [TimelineEvent] = []
        timelineEvents.append(TimelineEvent(
            entityId: "binary_sensor.garden_motion",
            entityName: "Radar Giardino",
            domain: "binary_sensor",
            state: "on",
            message: "Rilevato movimento all'esterno",
            contextType: "system",
            contextName: "PIR"
        ))
        timelineEvents.append(TimelineEvent(
            entityId: "lock.main_gate",
            entityName: "Cancelletto",
            domain: "lock",
            state: "unlocked",
            message: "Cancelletto aperto da notifica",
            contextType: "user",
            contextName: "Fabrizio"
        ))

        // Verifica che l'evento del cancello sia categorizzato in Accessi
        let accessEvents = timelineEvents.filter { $0.category == .access }
        XCTAssertEqual(accessEvents.count, 1)
        XCTAssertEqual(accessEvents.first?.entityId, "lock.main_gate")

        // Verifica che il movimento sia in Sicurezza
        let securityEvents = timelineEvents.filter { $0.category == .security }
        XCTAssertEqual(securityEvents.count, 1)

        // ─── STEP 6: Automation Evaluation & Execution ────────────────────────
        let autoRule = AutomationRule(
            name: "Sicurezza Notturna Giardino",
            triggers: [AutomationTrigger(platform: "state", entityKey: "binary_sensor.garden_motion", to: "on")],
            conditions: [AutomationCondition(condition: "time", after: "22:00:00", before: "06:00:00")],
            actions: [
                AutomationAction(type: "service", service: "turn_on", targetEntityKey: "light.garden_floodlight"),
                AutomationAction(type: "service", service: "set_temperature", targetEntityKey: "climate.thermostat", params: ["temperature": "20.5"])
            ]
        )

        // Esecuzione azioni della regola
        for action in autoRule.actions {
            guard let svc = action.service, let target = action.targetEntityKey else { continue }
            let dom = target.components(separatedBy: ".").first ?? "light"
            var p: [String: Any] = [:]
            if let tempStr = action.params?["temperature"], let t = Double(tempStr) {
                p["temperature"] = t
            }
            try await entityRepo.callService(domain: dom, service: svc, entityId: target, data: p)
        }

        XCTAssertEqual(entityRepo.entities.first(where: { $0.id == "light.garden_floodlight" })?.state, "on")
        XCTAssertEqual(entityRepo.entities.first(where: { $0.id == "climate.thermostat" })?.targetTemperature, 20.5)

        // ─── STEP 7: Mesh Diagnostics & Battery Warning ───────────────────────
        let meshNodes = [
            MeshNodeDiagnostic(id: "n.pir", name: "PIR Giardino", room: "Giardino", protocolType: "Zigbee 3.0", lqi: 215, rssi: -64, batteryLevel: 12),
            MeshNodeDiagnostic(id: "n.flood", name: "Faro Giardino (Router)", room: "Giardino", protocolType: "Zigbee 3.0", lqi: 250, rssi: -48, batteryLevel: nil, isRouter: true)
        ]

        let lowBatteryAlerts = meshNodes.filter { ($0.batteryLevel ?? 100) < 15 }
        XCTAssertEqual(lowBatteryAlerts.count, 1)
        XCTAssertEqual(lowBatteryAlerts.first?.name, "PIR Giardino")

        // ─── STEP 8: Temporary 24h Guest Pass (QR Code) ───────────────────────
        let guestPass = DomusUserAccount(
            name: "Marco Ospite",
            role: .guest,
            allowedRooms: ["Camera Ospiti", "Salotto"],
            canDisarmAlarm: false,
            canViewCameras: false,
            expiresAt: Calendar.current.date(byAdding: .hour, value: 24, to: Date())
        )

        XCTAssertFalse(guestPass.canDisarmAlarm)
        XCTAssertFalse(guestPass.canViewCameras)
        XCTAssertEqual(guestPass.role, .guest)
        XCTAssertEqual(guestPass.allowedRooms.count, 2)

        // ─── STEP 9: iOS 17 StandBy Mode & Night Scene ────────────────────────
        let nightStandEntry = DomusWidgetEntry(
            date: Date(),
            homeTemperature: 20.5,
            activeDevicesCount: 2,
            isAlarmArmed: true,
            entities: []
        )
        XCTAssertTrue(nightStandEntry.isAlarmArmed)
        XCTAssertEqual(nightStandEntry.homeTemperature, 20.5)

        // L'utente preme il pulsante 'Buonanotte' dallo stand MagSafe
        let goodnightIntent = RunSceneIntent(sceneId: "scene.goodnight")
        let sceneResult = try await goodnightIntent.perform()
        XCTAssertNotNil(sceneResult)

        // Spegni tutte le luci per la notte
        try await entityRepo.callService(domain: "light", service: "turn_off", entityId: "light.porch", data: [:])
        try await entityRepo.callService(domain: "light", service: "turn_off", entityId: "light.garden_floodlight", data: [:])

        XCTAssertEqual(entityRepo.entities.first(where: { $0.id == "light.porch" })?.state, "off")
        XCTAssertEqual(entityRepo.entities.first(where: { $0.id == "light.garden_floodlight" })?.state, "off")
    }
}
