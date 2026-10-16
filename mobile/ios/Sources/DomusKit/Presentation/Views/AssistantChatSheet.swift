import SwiftUI

public struct AssistantMessage: Identifiable, Sendable {
    public let id: String
    public let text: String
    public let isUser: Bool
    public let actionExecuted: String?
    public let timestamp: Date

    public init(
        id: String = UUID().uuidString,
        text: String,
        isUser: Bool,
        actionExecuted: String? = nil,
        timestamp: Date = Date()
    ) {
        self.id = id
        self.text = text
        self.isUser = isUser
        self.actionExecuted = actionExecuted
        self.timestamp = timestamp
    }
}

public struct AssistantChatSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.accessibilityReduceTransparency) private var reduceTransparency
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    @State private var messages: [AssistantMessage] = [
        AssistantMessage(
            text: "Ciao! Sono l'assistente locale di Domus basato su LLM. Come posso aiutarti con la tua casa?",
            isUser: false
        )
    ]
    @State private var inputText: String = ""
    @State private var isListening: Bool = false
    @State private var isProcessing: Bool = false

    private let suggestions = [
        "Spegni tutte le luci",
        "Qual è la temperatura in salotto?",
        "Attiva scenario Buonanotte",
        "Chiudi il cancello carraio"
    ]

    public init() {}

    public var body: some View {
        NavigationStack {
            ZStack {
                Color.domusSystemGroupedBackground.ignoresSafeArea()

                VStack(spacing: 0) {
                    // Chat messages scroll view
                    ScrollViewReader { proxy in
                        ScrollView {
                            LazyVStack(spacing: 12) {
                                ForEach(messages) { msg in
                                    MessageBubbleView(message: msg)
                                }
                                if isProcessing {
                                    HStack {
                                        ProgressView()
                                            .padding(.horizontal, 16)
                                            .padding(.vertical, 10)
                                            .background(Color.domusSecondarySystemGroupedBackground)
                                            .clipShape(Capsule())
                                        Spacer()
                                    }
                                    .padding(.horizontal)
                                    .accessibilityElement(children: .combine)
                                    .accessibilityLabel("L'assistente sta elaborando la richiesta...")
                                }
                            }
                            .padding(.vertical, 14)
                        }
                        .onChange(of: messages.count) { _, _ in
                            if let last = messages.last {
                                withAnimation(reduceMotion ? .none : .default) {
                                    proxy.scrollTo(last.id, anchor: .bottom)
                                }
                            }
                        }
                    }

                    // Suggestions pills
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: 8) {
                            ForEach(suggestions, id: \.self) { suggestion in
                                Button(action: { sendPrompt(suggestion) }) {
                                    Text(suggestion)
                                        .font(.footnote)
                                        .fontWeight(.medium)
                                        .padding(.horizontal, 12)
                                        .padding(.vertical, 8)
                                        .background(Color.cyan.opacity(reduceTransparency ? 0.3 : 0.12))
                                        .foregroundColor(.cyan)
                                        .clipShape(Capsule())
                                }
                                .frame(minHeight: 44)
                                .accessibilityLabel("Suggerimento rapido: \(suggestion)")
                                .accessibilityHint("Tocca due volte per inviare questa richiesta all'assistente")
                                .accessibilityAddTraits(.isButton)
                            }
                        }
                        .padding(.horizontal)
                        .padding(.vertical, 6)
                    }

                    Divider()

                    // Input bar with text & voice waveform button
                    HStack(spacing: 10) {
                        TextField("Chiedi a Domus...", text: $inputText)
                            .textFieldStyle(.plain)
                            .padding(.horizontal, 14)
                            .padding(.vertical, 10)
                            .background(Color.domusSecondarySystemGroupedBackground)
                            .clipShape(RoundedRectangle(cornerRadius: 18))
                            .onSubmit {
                                sendCurrentMessage()
                            }
                            .accessibilityLabel("Campo inserimento testo per l'assistente")

                        // Pulsante microfono vocale
                        Button(action: toggleVoiceInput) {
                            ZStack {
                                Circle()
                                    .fill(isListening ? Color.red : Color.cyan)
                                    .frame(width: 44, height: 44)
                                Image(systemName: isListening ? "waveform" : "mic.fill")
                                    .foregroundColor(.white)
                                    .font(.system(size: 18, weight: .bold))
                            }
                        }
                        .accessibilityLabel(isListening ? "Interrompi ascolto vocale" : "Avvia dettatura vocale")
                        .accessibilityHint("Attiva il microfono per formulare una richiesta vocale")
                        .accessibilityAddTraits(.isButton)

                        if !inputText.trimmingCharacters(in: .whitespaces).isEmpty {
                            Button(action: sendCurrentMessage) {
                                Image(systemName: "arrow.up.circle.fill")
                                    .font(.system(size: 34))
                                    .foregroundColor(.cyan)
                            }
                            .frame(width: 44, height: 44)
                            .accessibilityLabel("Invia messaggio")
                            .accessibilityAddTraits(.isButton)
                        }
                    }
                    .padding(.horizontal)
                    .padding(.vertical, 10)
                    .background {
                        if reduceTransparency {
                            Color.domusSecondarySystemGroupedBackground
                        } else {
                            Rectangle().fill(.ultraThinMaterial)
                        }
                    }
                }
            }
            .navigationTitle("Assistente AI Locale")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Chiudi") {
                        dismiss()
                    }
                    .accessibilityLabel("Chiudi assistente")
                }
            }
        }
    }

    private func sendCurrentMessage() {
        let text = inputText.trimmingCharacters(in: .whitespaces)
        guard !text.isEmpty else { return }
        inputText = ""
        sendPrompt(text)
    }

    private func sendPrompt(_ text: String) {
        let userMsg = AssistantMessage(text: text, isUser: true)
        messages.append(userMsg)
        isProcessing = true
        DomusAccessibility.announce("Messaggio inviato: \(text). Elaborazione in corso.")

        #if canImport(UIKit)
        UIImpactFeedbackGenerator(style: .medium).impactOccurred()
        #endif

        Task {
            do {
                let response = try await DomusHttpClient.shared.sendAssistantPrompt(prompt: text)
                await MainActor.run {
                    isProcessing = false
                    let action = response.actionSummary
                    messages.append(AssistantMessage(
                        text: response.responseText,
                        isUser: false,
                        actionExecuted: action
                    ))
                    DomusAccessibility.announce("Risposta ricevuta dall'assistente: \(response.responseText)")
                    #if canImport(UIKit)
                    UINotificationFeedbackGenerator().notificationOccurred(.success)
                    #endif
                }
            } catch {
                await MainActor.run {
                    isProcessing = false
                    let errMsg = "Errore durante la comunicazione con l'assistente: \(error.localizedDescription)"
                    messages.append(AssistantMessage(
                        text: errMsg,
                        isUser: false,
                        actionExecuted: nil
                    ))
                    DomusAccessibility.announce(errMsg)
                    #if canImport(UIKit)
                    UINotificationFeedbackGenerator().notificationOccurred(.error)
                    #endif
                }
            }
        }
    }

    private func toggleVoiceInput() {
        withAnimation(reduceMotion ? .none : .easeInOut) {
            isListening.toggle()
        }
        #if canImport(UIKit)
        UIImpactFeedbackGenerator(style: .heavy).impactOccurred()
        #endif
        if isListening {
            DomusAccessibility.announce("Microfono attivo. Parla adesso.")
            DispatchQueue.main.asyncAfter(deadline: .now() + 2.5) {
                if self.isListening {
                    self.isListening = false
                    self.sendPrompt("Spegni le luci del salotto")
                }
            }
        } else {
            DomusAccessibility.announce("Microfono disattivato.")
        }
    }
}

private struct MessageBubbleView: View {
    let message: AssistantMessage

    var body: some View {
        HStack {
            if message.isUser { Spacer() }

            VStack(alignment: message.isUser ? .trailing : .leading, spacing: 4) {
                Text(message.text)
                    .font(.subheadline)
                    .foregroundColor(message.isUser ? .white : .primary)
                    .padding(.horizontal, 14)
                    .padding(.vertical, 10)
                    .background(message.isUser ? Color.cyan : Color.domusSecondarySystemGroupedBackground)
                    .clipShape(RoundedRectangle(cornerRadius: 18))

                if let action = message.actionExecuted {
                    HStack(spacing: 4) {
                        Image(systemName: "bolt.fill")
                            .font(.system(size: 10))
                        Text(action)
                            .font(.caption2.monospaced().bold())
                    }
                    .foregroundColor(.green)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(Color.green.opacity(0.12))
                    .clipShape(Capsule())
                }
            }
            .padding(.horizontal)
            .accessibilityElement(children: .combine)
            .accessibilityLabel(message.isUser ? "Il tuo messaggio: \(message.text)" : "Risposta assistente: \(message.text)\(message.actionExecuted != nil ? ". Azione eseguita: \(message.actionExecuted!)" : "")")

            if !message.isUser { Spacer() }
        }
    }
}
