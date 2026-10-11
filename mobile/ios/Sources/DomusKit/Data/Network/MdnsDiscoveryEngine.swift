import Foundation
import Network

/// Modello di un Hub Domus scoperto sulla rete locale via mDNS.
public struct DiscoveredHub: Equatable, Hashable, Sendable {
    public let name: String
    public let host: String
    public let port: Int

    public init(name: String, host: String, port: Int) {
        self.name = name
        self.host = host
        self.port = port
    }

    public var httpBaseUrl: String {
        "http://\(host):\(port)"
    }

    public var wsBaseUrl: String {
        "ws://\(host):\(port)/api/v1/ws"
    }
}

/// Protocollo per scanner mDNS, utile per mocking e testing.
public protocol MdnsScannerProtocol: Sendable {
    func discover(serviceType: String, timeoutSeconds: Double) async -> DiscoveredHub?
}

/// Flag atomico per evitare doppia resume della continuation.
private final class AtomicFlag: @unchecked Sendable {
    private let lock = NSLock()
    private var _value: Bool = false

    var value: Bool {
        lock.lock()
        defer { lock.unlock() }
        return _value
    }

    /// Imposta a true e restituisce se era già stato impostato prima.
    func testAndSet() -> Bool {
        lock.lock()
        defer { lock.unlock() }
        if _value { return true }
        _value = true
        return false
    }
}

/// Motore di discovery mDNS per trovare il server Domus sulla rete locale.
/// Usa `NWBrowser` di Network.framework per cercare il servizio `_domus._tcp`.
public actor MdnsDiscoveryEngine: Sendable {

    private let customScanner: (any MdnsScannerProtocol)?
    private var browser: NWBrowser?

    public init(customScanner: (any MdnsScannerProtocol)? = nil) {
        self.customScanner = customScanner
    }

    /// Cerca il servizio `_domus._tcp.local` per `timeoutSeconds` secondi.
    /// - Returns: `DiscoveredHub` con host, porta e URL se trovato, `nil` altrimenti.
    public func discoverLocalHub(timeoutSeconds: Double = 1.5) async -> DiscoveredHub? {
        if let customScanner {
            return await customScanner.discover(serviceType: "_domus._tcp", timeoutSeconds: timeoutSeconds)
        }

        return await withCheckedContinuation { continuation in
            let params = NWParameters()
            params.includePeerToPeer = true

            let descriptor = NWBrowser.Descriptor.bonjour(type: "_domus._tcp", domain: "local.")
            let newBrowser = NWBrowser(for: descriptor, using: params)
            self.browser = newBrowser

            let done = AtomicFlag()

            newBrowser.stateUpdateHandler = { state in
                switch state {
                case .failed, .cancelled:
                    if !done.testAndSet() {
                        continuation.resume(returning: nil)
                    }
                default:
                    break
                }
            }

            newBrowser.browseResultsChangedHandler = { results, _ in
                guard !done.value else { return }
                if let first = results.first {
                    switch first.endpoint {
                    case .service(let name, _, _, _):
                        let conn = NWConnection(to: first.endpoint, using: .tcp)
                        conn.stateUpdateHandler = { connState in
                            if case .ready = connState {
                                if let innerEndpoint = conn.currentPath?.remoteEndpoint,
                                   case let .hostPort(host, port) = innerEndpoint {
                                    let hub = DiscoveredHub(
                                        name: name,
                                        host: "\(host)",
                                        port: Int(port.rawValue)
                                    )
                                    if !done.testAndSet() {
                                        continuation.resume(returning: hub)
                                    }
                                }
                                conn.cancel()
                            } else if case .failed = connState {
                                if !done.testAndSet() {
                                    continuation.resume(returning: nil)
                                }
                                conn.cancel()
                            }
                        }
                        conn.start(queue: .global())

                    default:
                        break
                    }
                }
            }

            newBrowser.start(queue: .global())

            // Timeout
            Task {
                try? await Task.sleep(nanoseconds: UInt64(timeoutSeconds * 1_000_000_000))
                newBrowser.cancel()
                if !done.testAndSet() {
                    continuation.resume(returning: nil)
                }
            }
        }
    }
}
