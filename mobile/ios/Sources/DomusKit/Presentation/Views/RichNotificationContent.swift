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
