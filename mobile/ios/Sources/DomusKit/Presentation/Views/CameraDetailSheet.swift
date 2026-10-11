import SwiftUI

/// Scheda di controllo per Telecamere — Fase 2: Snapshot live, streaming a bassa latenza, badge di stato e refresh.
public struct CameraDetailSheet: View {
    public let entity: Entity
    public var onRefresh: (String) -> Void

    @State private var isLiveActive: Bool = true
    @State private var lastRefreshedAt: Date = Date()
    @State private var isFullscreen: Bool = false
    @ObservedObject private var pipManager = PictureInPictureManager.shared

    public init(entity: Entity, onRefresh: @escaping (String) -> Void) {
        self.entity = entity
        self.onRefresh = onRefresh
    }

    public var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 20) {
                    // Video Feed Viewport
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
                                    .font(.system(size: 10, weight: .bold))
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
                                    .font(.system(size: 10, weight: .bold))
                            }
                            .foregroundColor(.green)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(Color.black.opacity(0.7))
                            .clipShape(Capsule())
                        }
                        .padding(12)
                    }
                    .padding(.horizontal)
                    .padding(.top, 12)

                    // Barra Controlli Rapidi Streaming
                    HStack(spacing: 12) {
                        Button(action: {
                            Haptics.light()
                            lastRefreshedAt = Date()
                            onRefresh(entity.id)
                        }) {
                            Label("Aggiorna Frame", systemImage: "arrow.clockwise")
                                .font(.subheadline.bold())
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 12)
                                .background(Color.secondary.opacity(0.12))
                                .foregroundColor(.primary)
                                .clipShape(RoundedRectangle(cornerRadius: 12))
                        }

                        Button(action: {
                            Haptics.medium()
                            isLiveActive.toggle()
                        }) {
                            Label(isLiveActive ? "Pausa Stream" : "Avvia Stream", systemImage: isLiveActive ? "pause.fill" : "play.fill")
                                .font(.subheadline.bold())
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 12)
                                .background(isLiveActive ? Color.orange.opacity(0.15) : Color.green.opacity(0.15))
                                .foregroundColor(isLiveActive ? .orange : .green)
                                .clipShape(RoundedRectangle(cornerRadius: 12))
                        }

                        Button(action: {
                            Haptics.medium()
                            pipManager.togglePictureInPicture(cameraEntityId: entity.id)
                        }) {
                            Image(systemName: pipManager.isPiPActive ? "pip.exit" : "pip.enter")
                                .font(.system(size: 16, weight: .bold))
                                .padding(12)
                                .background(pipManager.isPiPActive ? Color.blue.opacity(0.2) : Color.secondary.opacity(0.12))
                                .foregroundColor(pipManager.isPiPActive ? .blue : .primary)
                                .clipShape(RoundedRectangle(cornerRadius: 12))
                        }
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
                            statusRow(title: "Codec Video", value: "H.264 / WebRTC", icon: "film.fill", color: .blue)
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
