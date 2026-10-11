import XCTest
@testable import DomusKit

@MainActor
final class Phase2ControlsTests: XCTestCase {

    final class MockEntityRepository: EntityRepositoryProtocol, @unchecked Sendable {
        var entities: [Entity] = []
        var lastServiceCall: (domain: String, service: String, entityId: String, data: [String: Any])?

        func getEntities() async -> [Entity] { entities }
        func getEntity(id: String) async -> Entity? { entities.first(where: { $0.id == id }) }
        func syncEntities() async throws -> [Entity] { entities }
        func toggleEntity(id: String) async throws -> Entity {
            guard let idx = entities.firstIndex(where: { $0.id == id }) else {
                fatalError("Entity not found")
            }
            entities[idx].state = entities[idx].isOn ? "off" : "on"
            return entities[idx]
        }
        func callService(domain: String, service: String, entityId: String, data: [String: Any]) async throws {
            lastServiceCall = (domain, service, entityId, data)
        }
        func updateLocalEntity(_ entity: Entity) async {
            if let idx = entities.firstIndex(where: { $0.id == entity.id }) {
                entities[idx] = entity
            }
        }
        func getAreas() async -> [Area] { [] }
    }

    final class MockConnectionRepository: ConnectionRepositoryProtocol, @unchecked Sendable {
        func getCurrentConnection() async -> ServerConnection { ServerConnection() }
        func discoverLocalHub(timeoutSeconds: Double) async -> String? { "http://192.168.1.100:8080" }
        func connect(mode: TransportMode?) async throws {}
        func disconnect() async {}
        func setEndpoints(local: String?, remote: String?) async {}
    }

    func testEntityDomainComputedProperties() {
        let light = Entity(
            id: "light.living_room",
            name: "Luce Salotto",
            domain: "light",
            state: "on",
            attributes: [
                "brightness": AnyCodable(85),
                "rgb_color": AnyCodable([56, 189, 248]),
                "color_temp": AnyCodable(250)
            ]
        )
        XCTAssertTrue(light.isLight)
        XCTAssertEqual(light.brightness, 85)
        XCTAssertEqual(light.colorTemp, 250)
        XCTAssertNotNil(light.rgbColor)

        let socket = Entity(
            id: "switch.coffee_machine",
            name: "Macchina Caffè",
            domain: "switch",
            state: "on",
            attributes: [
                "power": AnyCodable(1450.0),
                "energy": AnyCodable(0.85)
            ]
        )
        XCTAssertTrue(socket.isSwitch)
        XCTAssertEqual(socket.powerWatts, 1450.0)
        XCTAssertEqual(socket.energyKWh, 0.85)

        let climate = Entity(
            id: "climate.thermostat",
            name: "Termostato",
            domain: "climate",
            state: "heat",
            attributes: [
                "current_temperature": AnyCodable(19.8),
                "temperature": AnyCodable(21.5),
                "hvac_action": AnyCodable("heating")
            ]
        )
        XCTAssertTrue(climate.isClimate)
        XCTAssertEqual(climate.currentTemperature, 19.8)
        XCTAssertEqual(climate.targetTemperature, 21.5)
        XCTAssertTrue(climate.isHeating)

        let alarm = Entity(
            id: "alarm_control_panel.home",
            name: "Allarme Domus",
            domain: "alarm_control_panel",
            state: "armed_home"
        )
        XCTAssertTrue(alarm.isAlarm)
        XCTAssertTrue(alarm.isArmed)
    }

    func testLightControlUpdatesViaViewModel() async {
        let repo = MockEntityRepository()
        let light = Entity(id: "light.1", name: "Luce", domain: "light", state: "off")
        repo.entities = [light]

        let connRepo = MockConnectionRepository()
        let vm = DashboardViewModel(
            entityRepository: repo,
            connectionRepository: connRepo,
            discoverAndConnectUseCase: DiscoverAndConnectUseCase(connectionRepository: connRepo),
            syncEntitiesUseCase: SyncEntitiesUseCase(entityRepository: repo),
            toggleEntityUseCase: ToggleEntityUseCase(entityRepository: repo)
        )
        vm.entities = [light]

        // Imposta luminosità a 75%
        vm.setLightBrightness(id: "light.1", brightness: 75)
        XCTAssertEqual(vm.entities.first?.brightness, 75)
        XCTAssertEqual(vm.entities.first?.state, "on")

        // Imposta colore
        vm.setLightColor(id: "light.1", r: 0.2, g: 0.8, b: 1.0)
        XCTAssertNotNil(vm.entities.first?.rgbColor)

        // Imposta temperatura colore
        vm.setLightColorTemp(id: "light.1", mireds: 300)
        XCTAssertEqual(vm.entities.first?.colorTemp, 300)
    }

    func testThermostatControlUpdatesViaViewModel() async {
        let repo = MockEntityRepository()
        let climate = Entity(id: "climate.1", name: "Clima", domain: "climate", state: "off")
        repo.entities = [climate]

        let connRepo = MockConnectionRepository()
        let vm = DashboardViewModel(
            entityRepository: repo,
            connectionRepository: connRepo,
            discoverAndConnectUseCase: DiscoverAndConnectUseCase(connectionRepository: connRepo),
            syncEntitiesUseCase: SyncEntitiesUseCase(entityRepository: repo),
            toggleEntityUseCase: ToggleEntityUseCase(entityRepository: repo)
        )
        vm.entities = [climate]

        vm.setThermostatTemperature(id: "climate.1", targetTemp: 22.5)
        XCTAssertEqual(vm.entities.first?.targetTemperature, 22.5)

        vm.setThermostatHvacMode(id: "climate.1", mode: "heat")
        XCTAssertEqual(vm.entities.first?.state, "heat")
    }
}
