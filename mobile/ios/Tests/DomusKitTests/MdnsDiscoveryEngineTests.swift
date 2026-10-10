import XCTest
@testable import DomusKit

final class MdnsDiscoveryEngineTests: XCTestCase {

    struct MockScanner: MdnsScannerProtocol {
        let returnHub: DiscoveredHub?

        func discover(serviceType: String, timeoutSeconds: Double) async -> DiscoveredHub? {
            return returnHub
        }
    }

    func testDiscoverySuccess() async {
        let mockHub = DiscoveredHub(name: "Domus Hub LAN", host: "192.168.1.100", port: 8080)
        let engine = MdnsDiscoveryEngine(customScanner: MockScanner(returnHub: mockHub))

        let result = await engine.discoverLocalHub(timeoutSeconds: 1.0)
        XCTAssertNotNil(result)
        XCTAssertEqual(result?.name, "Domus Hub LAN")
        XCTAssertEqual(result?.httpBaseUrl, "http://192.168.1.100:8080")
        XCTAssertEqual(result?.wsBaseUrl, "ws://192.168.1.100:8080/api/v1/ws")
    }

    func testDiscoveryTimeout() async {
        let engine = MdnsDiscoveryEngine(customScanner: MockScanner(returnHub: nil))

        let result = await engine.discoverLocalHub(timeoutSeconds: 0.1)
        XCTAssertNil(result)
    }
}
