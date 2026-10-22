import SwiftUI

/// Vista per l'estensione di notifica ricca (Notification Content Extension / Banner espanso).
/// Mostra l'istantanea acquisita dalla telecamera o dal videocitofono con pulsanti di azione immediata.
public struct RichCameraNotificationView: View {
    public let cameraName: String
    public let timestamp: String
    public let imageUrl: String?
    public let detectedObject: String
    public var onSoundSiren: () -> Void
    public var onDismissFalseAlarm: () -> Void
    public var onUnlockGate: () -> Void

    @State private var sirenActivated: Bool = false
    @State private var gateUnlocked: Bool = false

    public init(
        cameraName: String = "Telecamera Esterna Cancello",
        timestamp: String = "Adesso • 14:12",
        imageUrl: String? = nil,
        detectedObject: String = "Movimento persona rilevato",
        onSoundSiren: @escaping () -> Void = {},
        onDismissFalseAlarm: @escaping () -> Void = {},
        onUnlockGate: @escaping () -> Void = {}
    ) {
        self.cameraName = cameraName
        self.timestamp = timestamp
        self.imageUrl = imageUrl
        self.detectedObject = detectedObject
        self.onSoundSiren = onSoundSiren
        self.onDismissFalseAlarm = onDismissFalseAlarm
        self.onUnlockGate = onUnlockGate
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Header Notifica
            HStack(spacing: 8) {
                Image(systemName: "video.fill")
                    .foregroundColor(.red)
                    .font(.subheadline)

                Text(cameraName)
                    .font(.headline)
                    .lineLimit(1)

                Spacer()

                HStack(spacing: 4) {
                    Circle()
                        .fill(Color.red)
                        .frame(width: 6, height: 6)
                    Text("EVENTO LIVE")
                        .font(.caption2.bold())
                        .foregroundColor(.red)
                }
                .padding(.horizontal, 6)
                .padding(.vertical, 3)
                .background(Color.red.opacity(0.12))
                .clipShape(Capsule())
            }

            // Snapshot Video Viewport
            ZStack(alignment: .bottomLeading) {
                RoundedRectangle(cornerRadius: 14)
                    .fill(Color.black.opacity(0.85))
                    .aspectRatio(16/9, contentMode: .fit)
                    .overlay(
                        Group {
                            if let img = imageUrl, let url = URL(string: img) {
                                AsyncImage(url: url) { phase in
                                    if let image = phase.image {
                                        image
                                            .resizable()
                                            .aspectRatio(contentMode: .fill)
                                    } else {
                                        placeholderSnapshot
                                    }
                                }
                            } else {
                                placeholderSnapshot
                            }
                        }
                    )
                    .clipShape(RoundedRectangle(cornerRadius: 14))

                // Tag Rilevamento Sovrapposto in Basso a Sinistra
                HStack(spacing: 6) {
                    Image(systemName: "figure.walk.motion")
                        .font(.caption2.bold())
                    Text(detectedObject)
                        .font(.caption2.bold())
                }
                .foregroundColor(.white)
                .padding(.horizontal, 10)
                .padding(.vertical, 6)
                .background(Color.black.opacity(0.75))
                .clipShape(Capsule())
                .padding(10)
            }

            // Tasti Rapidi di Reazione
            HStack(spacing: 8) {
                Button(action: {
                    sirenActivated = true
                    Haptics.heavy()
                    onSoundSiren()
                    DomusAccessibility.announce("Sirena di allarme attivata!")
                }) {
                    HStack(spacing: 4) {
                        Image(systemName: "speaker.wave.3.fill")
                        Text(sirenActivated ? "Sirena ON!" : "Suona Sirena")
                    }
                    .font(.caption.bold())
                    .frame(maxWidth: .infinity, minHeight: 44)
                    .background(sirenActivated ? Color.red : Color.red.opacity(0.15))
                    .foregroundColor(sirenActivated ? .white : .red)
                    .clipShape(RoundedRectangle(cornerRadius: 10))
                }
                .accessibilityLabel("Suona sirena di sicurezza")

                Button(action: {
                    gateUnlocked = true
                    Haptics.success()
                    onUnlockGate()
                    DomusAccessibility.announce("Cancelletto sbloccato!")
                }) {
                    HStack(spacing: 4) {
                        Image(systemName: gateUnlocked ? "lock.open.fill" : "lock.fill")
                        Text(gateUnlocked ? "Aperto" : "Apri Cancello")
                    }
                    .font(.caption.bold())
                    .frame(maxWidth: .infinity, minHeight: 44)
                    .background(gateUnlocked ? Color.green : Color.blue.opacity(0.15))
                    .foregroundColor(gateUnlocked ? .white : .blue)
                    .clipShape(RoundedRectangle(cornerRadius: 10))
                }
                .accessibilityLabel("Apri cancello o portone")

                Button(action: {
                    Haptics.light()
                    onDismissFalseAlarm()
                    DomusAccessibility.announce("Allarme archiviato come falso positivo")
                }) {
                    Text("Falso Allarme")
                        .font(.caption.bold())
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .background(Color.secondary.opacity(0.12))
                        .foregroundColor(.secondary)
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                }
                .accessibilityLabel("Ignora come falso allarme")
            }
        }
        .padding(14)
        .background(Color.domusSecondarySystemGroupedBackground)
        .clipShape(RoundedRectangle(cornerRadius: 18))
        .shadow(color: Color.black.opacity(0.1), radius: 8, x: 0, y: 4)
    }

    private var placeholderSnapshot: some View {
        VStack(spacing: 6) {
            Image(systemName: "camera.viewfinder")
                .font(.system(size: 32))
                .foregroundColor(.white.opacity(0.6))
            Text("Istantanea Rilevamento Domus")
                .font(.caption.bold())
                .foregroundColor(.white.opacity(0.8))
        }
    }
}

/// Rich Notification View dedicata agli eventi di intelligenza artificiale locale:
/// Riconoscimento volti familiari (`camera.face_recognized`) e targhe autorizzate (`camera.plate_detected`).
public struct RichAIVisionNotificationView: View {
    public let eventType: String // "face_recognized" | "plate_detected"
    public let subjectLabel: String
    public let cameraName: String
    public let confidence: Double
    public let snapshotUrl: String?
    public let cropUrl: String?
    public let gateEntityId: String?
    public var onOpenGate: () -> Void
    public var onViewCamera: () -> Void
    public var onDismissFalsePositive: () -> Void

    @State private var gateOpened: Bool = false

    public init(
        eventType: String,
        subjectLabel: String,
        cameraName: String = "Telecamera Esterna",
        confidence: Double = 0.95,
        snapshotUrl: String? = nil,
        cropUrl: String? = nil,
        gateEntityId: String? = nil,
        onOpenGate: @escaping () -> Void = {},
        onViewCamera: @escaping () -> Void = {},
        onDismissFalsePositive: @escaping () -> Void = {}
    ) {
        self.eventType = eventType
        self.subjectLabel = subjectLabel
        self.cameraName = cameraName
        self.confidence = confidence
        self.snapshotUrl = snapshotUrl
        self.cropUrl = cropUrl
        self.gateEntityId = gateEntityId
        self.onOpenGate = onOpenGate
        self.onViewCamera = onViewCamera
        self.onDismissFalsePositive = onDismissFalsePositive
    }

    private var isFace: Bool {
        eventType == "face_recognized"
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Header con icona IA, Badge evento e telecamera
            HStack(spacing: 8) {
                Image(systemName: isFace ? "person.crop.rectangle.fill" : "car.rear.fill")
                    .foregroundColor(isFace ? .blue : .purple)
                    .font(.subheadline)

                Text(isFace ? "Volto Riconosciuto" : "Targa Rilevata")
                    .font(.headline)
                    .lineLimit(1)

                Spacer()

                HStack(spacing: 4) {
                    Circle()
                        .fill(isFace ? Color.blue : Color.purple)
                        .frame(width: 6, height: 6)
                    Text(String(format: "%.0f%% IA", confidence * 100))
                        .font(.caption2.bold())
                        .foregroundColor(isFace ? .blue : .purple)
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 3)
                .background((isFace ? Color.blue : Color.purple).opacity(0.12))
                .clipShape(Capsule())
            }

            // Snapshot con Picture-in-Picture del ritaglio volto/targa
            ZStack(alignment: .bottomLeading) {
                RoundedRectangle(cornerRadius: 14)
                    .fill(Color.black.opacity(0.85))
                    .aspectRatio(16/9, contentMode: .fit)
                    .overlay(
                        Group {
                            if let img = snapshotUrl, let url = URL(string: img) {
                                AsyncImage(url: url) { phase in
                                    if let image = phase.image {
                                        image.resizable().aspectRatio(contentMode: .fill)
                                    } else {
                                        visionPlaceholder
                                    }
                                }
                            } else {
                                visionPlaceholder
                            }
                        }
                    )
                    .clipShape(RoundedRectangle(cornerRadius: 14))

                // Tag Soggetto Riconosciuto con Ritaglio Bounding Box
                HStack(spacing: 8) {
                    if let crop = cropUrl, let cropUrlParsed = URL(string: crop) {
                        AsyncImage(url: cropUrlParsed) { phase in
                            if let img = phase.image {
                                img.resizable()
                                    .aspectRatio(contentMode: .fill)
                                    .frame(width: 32, height: 32)
                                    .clipShape(Circle())
                                    .overlay(Circle().stroke(Color.white, lineWidth: 1.5))
                            }
                        }
                    }

                    VStack(alignment: .leading, spacing: 1) {
                        Text(subjectLabel)
                            .font(.subheadline.bold())
                            .foregroundColor(.white)
                        Text(cameraName)
                            .font(.caption2)
                            .foregroundColor(.white.opacity(0.8))
                    }
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 6)
                .background(Color.black.opacity(0.75))
                .clipShape(RoundedRectangle(cornerRadius: 10))
                .padding(10)
            }

            // Azioni Rapide: Apri Cancello / Vedi Telecamera / Falso Positivo
            HStack(spacing: 8) {
                if gateEntityId != nil || !isFace {
                    Button(action: {
                        gateOpened = true
                        Haptics.success()
                        onOpenGate()
                        DomusAccessibility.announce("Cancello o portone aperto")
                    }) {
                        HStack(spacing: 4) {
                            Image(systemName: gateOpened ? "lock.open.fill" : "door.garage.open")
                            Text(gateOpened ? "Aperto!" : "Apri Cancello")
                        }
                        .font(.caption.bold())
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .background(gateOpened ? Color.green : Color.green.opacity(0.18))
                        .foregroundColor(gateOpened ? .white : .green)
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                    }
                    .accessibilityLabel("Apri cancello per \(subjectLabel)")
                }

                Button(action: {
                    Haptics.light()
                    onViewCamera()
                }) {
                    HStack(spacing: 4) {
                        Image(systemName: "video.fill")
                        Text("Vedi Live")
                    }
                    .font(.caption.bold())
                    .frame(maxWidth: .infinity, minHeight: 44)
                    .background(Color.blue.opacity(0.15))
                    .foregroundColor(.blue)
                    .clipShape(RoundedRectangle(cornerRadius: 10))
                }
                .accessibilityLabel("Visualizza telecamera live")

                Button(action: {
                    Haptics.light()
                    onDismissFalsePositive()
                    DomusAccessibility.announce("Archiviato come falso positivo")
                }) {
                    Text("Ignora")
                        .font(.caption.bold())
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .background(Color.secondary.opacity(0.12))
                        .foregroundColor(.secondary)
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                }
                .accessibilityLabel("Ignora rilevamento")
            }
        }
        .padding(14)
        .background(Color.domusSecondarySystemGroupedBackground)
        .clipShape(RoundedRectangle(cornerRadius: 18))
        .shadow(color: Color.black.opacity(0.1), radius: 8, x: 0, y: 4)
    }

    private var visionPlaceholder: some View {
        VStack(spacing: 6) {
            Image(systemName: isFace ? "person.crop.rectangle" : "car.side")
                .font(.system(size: 32))
                .foregroundColor(.white.opacity(0.6))
            Text("Istantanea Riconoscimento Edge AI")
                .font(.caption.bold())
                .foregroundColor(.white.opacity(0.8))
        }
    }
}

