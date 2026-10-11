import SwiftUI

/// Screensaver notturno OLED per Tablet a parete — Fase 5.
/// Mostra orologio minimale, meteo e consumi riducendo al minimo la luminosità.
public struct KioskScreensaverView: View {
    public var onWake: () -> Void

    @State private var currentTime: String = ""
    @State private var currentDate: String = ""
    private let timer = Timer.publish(every: 1, on: .main, in: .common).autoconnect()

    public init(onWake: @escaping () -> Void) {
        self.onWake = onWake
    }

    public var body: some View {
        ZStack {
            Color.black
                .ignoresSafeArea()

            VStack(spacing: 16) {
                Spacer()

                // Orologio minimale
                Text(currentTime.isEmpty ? "12:00" : currentTime)
                    .font(.system(size: 80, weight: .thin, design: .rounded))
                    .foregroundColor(.white.opacity(0.85))

                // Data
                Text(currentDate.isEmpty ? "Lunedì, 11 Ottobre" : currentDate)
                    .font(.title3)
                    .foregroundColor(.secondary)

                // Badge Casa
                HStack(spacing: 12) {
                    Label("21.5°C Interna", systemImage: "thermometer.medium")
                    Text("•")
                    Label("Tutto Spento", systemImage: "moon.fill")
                }
                .font(.subheadline)
                .foregroundColor(.white.opacity(0.6))
                .padding(.top, 8)

                Spacer()

                // Indicatore di risveglio
                HStack(spacing: 6) {
                    Image(systemName: "hand.tap.fill")
                    Text("Tocca per risvegliare Domus")
                }
                .font(.caption2)
                .foregroundColor(.white.opacity(0.3))
                .padding(.bottom, 24)
            }
        }
        .contentShape(Rectangle())
        .onTapGesture {
            #if canImport(UIKit)
            Haptics.light()
            #endif
            onWake()
        }
        .onAppear {
            updateClock()
        }
        .onReceive(timer) { _ in
            updateClock()
        }
    }

    private func updateClock() {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "it_IT")
        formatter.dateFormat = "HH:mm"
        currentTime = formatter.string(from: Date())

        formatter.dateFormat = "EEEE, d MMMM"
        currentDate = formatter.string(from: Date()).capitalized
    }
}
