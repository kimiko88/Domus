import XCTest
@testable import DomusKit

final class MobileBackendContractTests: XCTestCase {

    func testDynamicAPIPathSegmentsAreEncodedAsSingleSegments() throws {
        let encoded = try DomusHttpClient.encodePathSegment("tag/../../family/invites?admin=true")
        XCTAssertEqual(encoded, "tag%2F..%2F..%2Ffamily%2Finvites%3Fadmin%3Dtrue")
        XCTAssertThrowsError(try DomusHttpClient.encodePathSegment(""))
        XCTAssertThrowsError(try DomusHttpClient.encodePathSegment("."))
        XCTAssertThrowsError(try DomusHttpClient.encodePathSegment(".."))
    }

    func testWebSocketEventContractDecoding() throws {
        let jsonString = """
        {
            "type": "event",
            "id": 42,
            "event": {
                "entity_id": "light.kitchen",
                "new_state": {
                    "entity_id": "light.kitchen",
                    "state": "on",
                    "attributes": {
                        "brightness": 90,
                        "friendly_name": "Luce Cucina"
                    },
                    "last_changed": "2026-10-10T20:00:00Z",
                    "last_updated": "2026-10-10T20:00:00Z"
                }
            }
        }
        """

        let data = jsonString.data(using: .utf8)!
        let msg = try JSONDecoder().decode(DomusWsMessage.self, from: data)

        XCTAssertEqual(msg.type, "event")
        XCTAssertEqual(msg.id, 42)
        XCTAssertNotNil(msg.event)

        let snapshot = msg.event?.newState
        XCTAssertEqual(snapshot?.entityId, "light.kitchen")
        XCTAssertEqual(snapshot?.state, "on")
        XCTAssertEqual(snapshot?.attributes["brightness"]?.value as? Int, 90)
    }

    func testRestEntitiesContractDecoding() throws {
        let jsonString = """
        [
            {
                "id": "climate.main_thermostat",
                "name": "Termostato Principale",
                "domain": "climate",
                "state": "heat",
                "attributes": {
                    "current_temperature": 21.5,
                    "target_temperature": 22.0
                },
                "areaId": "living_room",
                "lastChanged": "2026-10-10T19:00:00Z",
                "lastUpdated": "2026-10-10T19:00:00Z"
            }
        ]
        """

        let data = jsonString.data(using: .utf8)!
        let entities = try JSONDecoder().decode([Entity].self, from: data)

        XCTAssertEqual(entities.count, 1)
        let thermostat = entities[0]
        XCTAssertEqual(thermostat.id, "climate.main_thermostat")
        XCTAssertEqual(thermostat.isClimate, true)
        XCTAssertEqual(thermostat.currentTemperature, 21.5)
    }

    func testVacationModeContractDecoding() throws {
        let jsonString = """
        {
            "enabled": true,
            "jitter_minutes": 15
        }
        """
        let data = jsonString.data(using: .utf8)!
        let vac = try JSONDecoder().decode(VacationModeStatusDTO.self, from: data)

        XCTAssertTrue(vac.enabled)
        XCTAssertEqual(vac.jitterMinutes, 15)
    }

    func testNetworkLeaksContractDecoding() throws {
        let jsonString = """
        {
            "count": 1,
            "leaks": [
                {
                    "id": "leak_01",
                    "mac": "bc:24:11:22:33:44",
                    "ip": "192.168.1.188",
                    "destination": "api2.tuya.cn:443",
                    "timestamp": "2026-10-25T16:00:00Z",
                    "action": "quarantined"
                }
            ],
            "isolated_macs": [
                "bc:24:11:22:33:44"
            ]
        }
        """
        let data = jsonString.data(using: .utf8)!
        let resp = try JSONDecoder().decode(NetworkLeaksResponseDTO.self, from: data)

        XCTAssertEqual(resp.count, 1)
        XCTAssertEqual(resp.leaks.count, 1)
        XCTAssertEqual(resp.leaks.first?.destination, "api2.tuya.cn:443")
        XCTAssertEqual(resp.isolatedMacs, ["bc:24:11:22:33:44"])
    }
}

