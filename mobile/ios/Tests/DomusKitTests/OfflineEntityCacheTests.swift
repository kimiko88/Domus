import XCTest
@testable import DomusKit

final class OfflineEntityCacheTests: XCTestCase {

    func testUpsertAndRetrieve() async {
        let cache = OfflineEntityCache()

        let entity = Entity(
            id: "light.living_room",
            name: "Luce Salotto",
            domain: "light",
            state: "off",
            attributes: ["brightness": AnyCodable(75)]
        )

        await cache.updateEntities([entity])

        let retrieved = await cache.getEntity(id: "light.living_room")
        XCTAssertNotNil(retrieved)
        XCTAssertEqual(retrieved?.state, "off")
        XCTAssertEqual(retrieved?.brightness, 75)

        // Toggle / update
        var updated = entity
        updated.state = "on"
        await cache.upsertEntity(updated)

        let afterUpdate = await cache.getEntity(id: "light.living_room")
        XCTAssertEqual(afterUpdate?.state, "on")
        XCTAssertTrue(afterUpdate?.isOn == true)
    }

    func testAreaManagement() async {
        let cache = OfflineEntityCache()
        let area = Area(id: "kitchen", name: "Cucina")

        await cache.updateAreas([area])
        let allAreas = await cache.getAllAreas()
        XCTAssertEqual(allAreas.count, 1)
        XCTAssertEqual(allAreas.first?.name, "Cucina")
    }
}
