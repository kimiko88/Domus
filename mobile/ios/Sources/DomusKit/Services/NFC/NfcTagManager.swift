import Foundation
#if canImport(CoreNFC) && os(iOS)
import CoreNFC
#endif

/// Modello di un Tag NFC letto o programmato.
public struct DomusNfcTag: Identifiable, Codable, Sendable {
    public let id: String
    public let uri: String
    public let actionType: String // "scene", "toggle", "alarm", "custom"
    public let targetId: String
    public let createdAt: Date

    public init(id: String = UUID().uuidString, uri: String, actionType: String, targetId: String, createdAt: Date = Date()) {
        self.id = id
        self.uri = uri
        self.actionType = actionType
        self.targetId = targetId
        self.createdAt = createdAt
    }
}

/// Modalità operativa della sessione NFC.
public enum NfcSessionMode: Sendable {
    case read
    case write(uri: String)
}

/// Gestore Scansione e Scrittura Tag NFC nativi — Fase 3.
@MainActor
public final class NfcTagManager: NSObject, ObservableObject {
    @Published public var isNfcAvailable: Bool = false
    @Published public var isSessionActive: Bool = false
    @Published public var lastReadTag: DomusNfcTag? = nil
    @Published public var lastStatusMessage: String = ""

    public var onTagScanned: ((DomusNfcTag) -> Void)?

    #if canImport(CoreNFC) && os(iOS)
    private var session: NFCNDEFReaderSession?
    private var currentMode: NfcSessionMode = .read
    #endif

    public override init() {
        super.init()
        #if canImport(CoreNFC) && os(iOS)
        self.isNfcAvailable = NFCNDEFReaderSession.readingAvailable
        #else
        self.isNfcAvailable = false
        #endif
    }

    /// Avvia la lettura di un tag NFC (foreground o background shortcut).
    public func startReading() {
        #if canImport(CoreNFC) && os(iOS)
        guard NFCNDEFReaderSession.readingAvailable else {
            lastStatusMessage = "NFC non supportato su questo dispositivo"
            return
        }

        currentMode = .read
        session = NFCNDEFReaderSession(
            delegate: self,
            queue: nil,
            invalidateAfterFirstRead: true
        )
        session?.alertMessage = "Avvicina la parte superiore dell'iPhone al Tag Domus."
        session?.begin()
        isSessionActive = true
        #else
        lastStatusMessage = "NFC disponibile solo su dispositivi iOS fisici"
        #endif
    }

    /// Avvia la scrittura di un payload Domus su un tag NFC fisico.
    public func writeTag(uri: String) {
        #if canImport(CoreNFC) && os(iOS)
        guard NFCNDEFReaderSession.readingAvailable else {
            lastStatusMessage = "NFC non supportato su questo dispositivo"
            return
        }

        currentMode = .write(uri: uri)
        session = NFCNDEFReaderSession(
            delegate: self,
            queue: nil,
            invalidateAfterFirstRead: false
        )
        session?.alertMessage = "Avvicina il Tag NFC vuoto da programmare."
        session?.begin()
        isSessionActive = true
        #else
        lastStatusMessage = "NFC disponibile solo su dispositivi iOS fisici"
        #endif
    }

    /// Simula la lettura di un tag (utile per test e anteprima).
    public func simulateTagScan(uri: String = "domus://scene/relax_evening") {
        let tag = parseDomusUri(uri)
        self.lastReadTag = tag
        self.lastStatusMessage = "Tag simulato letto: \(uri)"
        Haptics.success()
        self.onTagScanned?(tag)
    }

    private func parseDomusUri(_ uri: String) -> DomusNfcTag {
        // Formato tipico: domus://{type}/{targetId} es: domus://scene/night o domus://toggle/light.kitchen
        var action = "custom"
        var target = uri
        if uri.hasPrefix("domus://scene/") {
            action = "scene"
            target = String(uri.dropFirst("domus://scene/".count))
        } else if uri.hasPrefix("domus://toggle/") {
            action = "toggle"
            target = String(uri.dropFirst("domus://toggle/".count))
        } else if uri.hasPrefix("domus://alarm/") {
            action = "alarm"
            target = String(uri.dropFirst("domus://alarm/".count))
        }

        return DomusNfcTag(uri: uri, actionType: action, targetId: target)
    }
}

// MARK: - NFCNDEFReaderSessionDelegate

#if canImport(CoreNFC) && os(iOS)
extension NfcTagManager: NFCNDEFReaderSessionDelegate {
    public nonisolated func readerSession(_ session: NFCNDEFReaderSession, didDetectNDEFs messages: [NFCNDEFMessage]) {
        guard let message = messages.first, let record = message.records.first else { return }

        // Estrai URI o Payload testuale
        var extractedUri = ""
        if let uri = record.wellKnownTypeURIPayload() {
            extractedUri = uri.absoluteString
        } else if let text = String(data: record.payload, encoding: .utf8) {
            extractedUri = text
        }

        Task { @MainActor in
            let tag = self.parseDomusUri(extractedUri)
            self.lastReadTag = tag
            self.lastStatusMessage = "Tag letto con successo!"
            Haptics.success()
            self.onTagScanned?(tag)
            session.alertMessage = "Operazione Domus eseguita!"
            session.invalidate()
            self.isSessionActive = false
        }
    }

    public nonisolated func readerSession(_ session: NFCNDEFReaderSession, didDetect tags: [NFCNDEFTag]) {
        Task { @MainActor in
            guard case .write(let uri) = self.currentMode else { return }
            guard let tag = tags.first else { return }

            session.connect(to: tag) { error in
                if error != nil {
                    session.invalidate(errorMessage: "Errore di connessione al tag")
                    return
                }

                tag.queryNDEFStatus { status, capacity, error in
                    guard error == nil else {
                        session.invalidate(errorMessage: "Errore durante la verifica del tag")
                        return
                    }

                    switch status {
                    case .notSupported:
                        session.invalidate(errorMessage: "Tag non compatibile NDEF")
                    case .readOnly:
                        session.invalidate(errorMessage: "Tag in sola lettura")
                    case .readWrite:
                        guard let url = URL(string: uri), let payload = NFCNDEFPayload.wellKnownTypeURIPayload(url: url) else {
                            session.invalidate(errorMessage: "URL non valido")
                            return
                        }
                        let message = NFCNDEFMessage(records: [payload])
                        tag.writeNDEF(message) { writeError in
                            if writeError != nil {
                                session.invalidate(errorMessage: "Scrittura fallita")
                            } else {
                                session.alertMessage = "Tag Domus programmato con successo!"
                                session.invalidate()
                                Task { @MainActor in
                                    Haptics.success()
                                    self.lastStatusMessage = "Tag scritto: \(uri)"
                                    self.isSessionActive = false
                                }
                            }
                        }
                    @unknown default:
                        session.invalidate(errorMessage: "Stato tag sconosciuto")
                    }
                }
            }
        }
    }


    public nonisolated func readerSession(_ session: NFCNDEFReaderSession, didInvalidateWithError error: Error) {
        Task { @MainActor in
            self.isSessionActive = false
            self.lastStatusMessage = "Sessione NFC terminata"
        }
    }
}
#endif
