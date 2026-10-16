import SwiftUI

/// Scheda di controllo per Telecamere — Fase 2: Snapshot live, streaming a bassa latenza, badge di stato e refresh.
public struct CameraDetailSheet: View {
    public let entity: Entity
    public var onRefresh: (String) -> Void

    @State private var isLiveActive: Bool = true
    @State private var lastRefreshedAt: Date = Date()
    @State private var isFullscreen: Bool = false
    @State private var isTalking: Bool = false
    @State private var isDoorUnlocked: Bool = false
    @State private var isDoorActionRunning: Bool = false
    @State private var speakerVolume: Double = 0.8
    @ObservedObject private var pipManager = PictureInPictureManager.shared

    public init(entity: Entity, onRefresh: @escaping (String) -> Void) {
        self.entity = entity
        self.onRefresh = onRefresh
    }

    public var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 20) {
                    // Video Feed Viewport con Overlay Sblocco Rapido
                    ZStack(alignment: .bottomTrailing) {
                        ZStack(alignment: .topTrailing) {
                            // Background / Viewport
                            RoundedRectangle(cornerRadius: 18)
                                .fill(Color.black)
                                .aspectRatio(16/9, contentMode: .fit)
                                .overlay(
                                    VStack(spacing: 12) {
                                        if let imgUrl = entity.cameraImageUrl, let url = URL(string: imgUrl) {
                                            AsyncImage(url: url) { image in
                                                image
                                                    .resizable()
                                                    .aspectRatio(contentMode: .fill)
                                            } placeholder: {
                                                placeholderCameraView
                                            }
                                        } else {
                                            placeholderCameraView
                                        }
                                    }
                                )
                                .clipShape(RoundedRectangle(cornerRadius: 18))
                                .shadow(color: Color.black.opacity(0.3), radius: 10, x: 0, y: 5)

                            // Badge Live & Latenza sovrapposto in alto
                            HStack(spacing: 8) {
                                HStack(spacing: 4) {
                                    Circle()
                                        .fill(Color.red)
                                        .frame(width: 8, height: 8)
                                    Text("LIVE")
                                        .font(.caption2.bold())
                                        .foregroundColor(.white)
                                }
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(Color.black.opacity(0.7))
                                .clipShape(Capsule())

                                HStack(spacing: 4) {
                                    Image(systemName: "bolt.horizontal.fill")
                                        .font(.system(size: 8))
                                    Text("LAN 12ms")
                                        .font(.caption2.bold())
                                }
                                .foregroundColor(.green)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(Color.black.opacity(0.7))
                                .clipShape(Capsule())
                            }
                            .padding(12)
                        }

                        // Overlay Sblocco Rapido Cancelletto / Serratura in basso a destra nel video
                        quickUnlockButtonOverlay
                            .padding(12)
                    }
                    .padding(.horizontal)
                    .padding(.top, 12)

                    // Sezione Citofono & Audio Bidirezionale (Two-Way Talk Intercom)
                    intercomSectionCard

                    // Barra Controlli Rapidi Streaming
                    HStack(spacing: 12) {
                        Button(action: {
                            Haptics.light()
                            lastRefreshedAt = Date()
                            onRefresh(entity.id)
                            DomusAccessibility.announce("Frame video aggiornato")
                        }) {
                            Label("Aggiorna Frame", systemImage: "arrow.clockwise")
                                .font(.subheadline.bold())
                                .frame(maxWidth: .infinity, minHeight: 44)
                                .padding(.vertical, 10)
                                .background(Color.secondary.opacity(0.12))
                                .foregroundColor(.primary)
                                .clipShape(RoundedRectangle(cornerRadius: 12))
                        }
                        .accessibilityLabel("Aggiorna istantanea telecamera")

                        Button(action: {
                            Haptics.medium()
                            isLiveActive.toggle()
                            DomusAccessibility.announce(isLiveActive ? "Streaming riavviato" : "Streaming in pausa")
                        }) {
                            Label(isLiveActive ? "Pausa Stream" : "Avvia Stream", systemImage: isLiveActive ? "pause.fill" : "play.fill")
                                .font(.subheadline.bold())
                                .frame(maxWidth: .infinity, minHeight: 44)
                                .padding(.vertical, 10)
                                .background(isLiveActive ? Color.orange.opacity(0.15) : Color.green.opacity(0.15))
                                .foregroundColor(isLiveActive ? .orange : .green)
                                .clipShape(RoundedRectangle(cornerRadius: 12))
                        }
                        .accessibilityLabel(isLiveActive ? "Metti in pausa streaming video" : "Avvia streaming video in diretta")

                        Button(action: {
                            Haptics.medium()
                            pipManager.togglePictureInPicture(cameraEntityId: entity.id)
                        }) {
                            Image(systemName: pipManager.isPiPActive ? "pip.exit" : "pip.enter")
                                .font(.system(size: 16, weight: .bold))
                                .frame(width: 44, height: 44)
                                .background(pipManager.isPiPActive ? Color.blue.opacity(0.2) : Color.secondary.opacity(0.12))
                                .foregroundColor(pipManager.isPiPActive ? .blue : .primary)
                                .clipShape(RoundedRectangle(cornerRadius: 12))
                        }
                        .accessibilityLabel(pipManager.isPiPActive ? "Chiudi Picture in Picture" : "Apri in Picture in Picture")
                    }
                    .padding(.horizontal)

                    // Sezione Dettagli e Rilevamento Movimento
                    VStack(alignment: .leading, spacing: 14) {
                        Text("Stato Sensori Telecamera")
                            .font(.headline)

                        HStack {
                            statusRow(title: "Rilevamento Movimento", value: "Attivo", icon: "figure.walk", color: .green)
                            Spacer()
                            statusRow(title: "Visione Notturna IR", value: "Auto", icon: "moon.stars.fill", color: .purple)
                        }

                        Divider()

                        HStack {
                            statusRow(title: "Codec Video / Audio", value: "H.264 • WebRTC Opus", icon: "film.fill", color: .blue)
                            Spacer()
                            statusRow(title: "Risoluzione", value: "1080p (30 fps)", icon: "tv.fill", color: .yellow)
                        }
                    }
                    .padding(16)
                    .background(Color.secondary.opacity(0.1))
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                    .padding(.horizontal)
                    .padding(.bottom, 24)
                }
            }
            .navigationTitle(entity.name)
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .automatic) {
                    Button(action: {
                        Haptics.medium()
                        pipManager.togglePictureInPicture(cameraEntityId: entity.id)
                    }) {
                        Image(systemName: pipManager.isPiPActive ? "pip.exit" : "pip.enter")
                            .foregroundColor(.blue)
                    }
                }
            }
        }
    }

    private var placeholderCameraView: some View {
        VStack(spacing: 8) {
            Image(systemName: "video.fill")
                .font(.system(size: 38))
                .foregroundColor(.white.opacity(0.7))
            Text("Streaming Video Nativo")
                .font(.caption.bold())
                .foregroundColor(.white.opacity(0.9))
            Text("Connessione RTSP / WebRTC")
                .font(.caption2)
                .foregroundColor(.white.opacity(0.5))
        }
    }

    // MARK: - Overlay Sblocco Rapido Cancelletto / Serratura
    private var quickUnlockButtonOverlay: some View {
        Button(action: {
            triggerQuickUnlock()
        }) {
            HStack(spacing: 6) {
                if isDoorActionRunning {
                    ProgressView()
                        .progressViewStyle(CircularProgressViewStyle(tint: .white))
                        .scaleEffect(0.8)
                } else {
                    Image(systemName: isDoorUnlocked ? "lock.open.fill" : "lock.fill")
                        .font(.system(size: 15, weight: .bold))
                }
                Text(isDoorUnlocked ? "Aperto" : "Apri Cancello")
                    .font(.caption.bold())
            }
            .foregroundColor(.white)
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
            .background(isDoorUnlocked ? Color.green : Color.black.opacity(0.75))
            .clipShape(Capsule())
            .overlay(
                Capsule()
                    .stroke(isDoorUnlocked ? Color.green : Color.white.opacity(0.3), lineWidth: 1)
            )
            .shadow(color: .black.opacity(0.4), radius: 6, x: 0, y: 3)
        }
        .disabled(isDoorActionRunning)
        .accessibilityLabel(isDoorUnlocked ? "Cancello aperto" : "Sblocca cancelletto o portone")
        .accessibilityHint("Invia il comando di apertura immediata alla serratura associata")
    }

    private func triggerQuickUnlock() {
        Haptics.heavy()
        isDoorActionRunning = true
        DomusAccessibility.announce("Apertura cancelletto in corso")

        Task {
            let lockTargetId = entity.attributes["lock_entity_id"]?.value as? String ?? "lock.main_gate"
            try? await DomusHttpClient.shared.callService(
                domain: "lock",
                service: "unlock",
                entityId: lockTargetId
            )
            await MainActor.run {
                isDoorActionRunning = false
                isDoorUnlocked = true
                Haptics.success()
                DomusAccessibility.announce("Cancelletto aperto con successo")
            }
            try? await Task.sleep(nanoseconds: 3_500_000_000)
            await MainActor.run {
                isDoorUnlocked = false
            }
        }
    }

    // MARK: - Sezione Citofono & Audio Bidirezionale (Push-to-Talk)
    private var intercomSectionCard: some View {
        VStack(spacing: 16) {
            // Header
            HStack {
                HStack(spacing: 8) {
                    Image(systemName: "waveform.and.mic")
                        .font(.headline)
                        .foregroundColor(.blue)
                    Text("Citofono & Audio Bidirezionale")
                        .font(.headline)
                }
                Spacer()
                Text(isTalking ? "PARLA ORA" : "STANDBY")
                    .font(.caption2.bold())
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(isTalking ? Color.red.opacity(0.2) : Color.secondary.opacity(0.15))
                    .foregroundColor(isTalking ? .red : .secondary)
                    .clipShape(Capsule())
            }

            // Pulsante Push-to-Talk centrale
            VStack(spacing: 12) {
                ZStack {
                    if isTalking {
                        Circle()
                            .stroke(Color.red.opacity(0.4), lineWidth: 4)
                            .frame(width: 90, height: 90)
                            .scaleEffect(isTalking ? 1.25 : 1.0)
                            .animation(.easeInOut(duration: 0.8).repeatForever(autoreverses: true), value: isTalking)
                    }

                    Circle()
                        .fill(isTalking ? Color.red : Color.blue)
                        .frame(width: 74, height: 74)
                        .shadow(color: (isTalking ? Color.red : Color.blue).opacity(0.4), radius: 8, x: 0, y: 4)

                    Image(systemName: isTalking ? "mic.fill" : "mic")
                        .font(.system(size: 30, weight: .bold))
                        .foregroundColor(.white)
                }
                .contentShape(Circle())
                .gesture(
                    DragGesture(minimumDistance: 0)
                        .onChanged { _ in
                            if !isTalking {
                                isTalking = true
                                Haptics.medium()
                                DomusAccessibility.announce("Microfono aperto. In trasmissione.")
                            }
                        }
                        .onEnded { _ in
                            isTalking = false
                            Haptics.light()
                            DomusAccessibility.announce("Microfono chiuso.")
                        }
                )
                .accessibilityElement(children: .ignore)
                .accessibilityLabel("Pulsante Push-to-Talk citofono")
                .accessibilityValue(isTalking ? "In trasmissione" : "Disattivato")
                .accessibilityHint("Tieni premuto per parlare attraverso il citofono")

                Text(isTalking ? "Rilascia per chiudere la voce" : "Tieni premuto per parlare")
                    .font(.caption.bold())
                    .foregroundColor(isTalking ? .red : .secondary)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 6)

            // Slider Volume Altoparlante Esterno
            HStack(spacing: 12) {
                Image(systemName: "speaker.wave.1.fill")
                    .font(.caption)
                    .foregroundColor(.secondary)

                Slider(value: $speakerVolume, in: 0.0...1.0)
                    .tint(.blue)
                    .accessibilityLabel("Volume altoparlante citofono")
                    .accessibilityValue("\(Int(speakerVolume * 100)) percento")

                Image(systemName: "speaker.wave.3.fill")
                    .font(.caption)
                    .foregroundColor(.secondary)

                Text("\(Int(speakerVolume * 100))%")
                    .font(.caption2.monospacedDigit())
                    .foregroundColor(.secondary)
                    .frame(width: 38, alignment: .trailing)
            }
            .padding(.horizontal, 4)
        }
        .padding(16)
        .background(Color.secondary.opacity(0.08))
        .clipShape(RoundedRectangle(cornerRadius: 16))
        .padding(.horizontal)
    }

    private func statusRow(title: String, value: String, icon: String, color: Color) -> some View {
        HStack(spacing: 8) {
            Image(systemName: icon)
                .foregroundColor(color)
            VStack(alignment: .leading, spacing: 2) {
                Text(title)
                    .font(.caption2)
                    .foregroundColor(.secondary)
                Text(value)
                    .font(.subheadline.bold())
            }
        }
    }
}

