import SwiftUI

/// Vista nativa per il login sul server Domus.
/// Supporta il dual-endpoint: LAN locale con mDNS discovery e indirizzo TLS remoto per l'accesso da fuori casa.
public struct LoginView: View {
    @ObservedObject public var viewModel: DashboardViewModel
    @Environment(\.dismiss) private var dismiss

    @State private var localUrl: String = ""
    @State private var remoteUrl: String = ""
    @State private var username: String = "admin"
    @State private var password: String = ""
    @State private var twoFactorCode: String = ""
    @State private var isDiscoveringMdns: Bool = false
    @State private var discoveryMessage: String? = nil

    public init(viewModel: DashboardViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        NavigationStack {
            ZStack {
                Color.domusSystemGroupedBackground
                    .ignoresSafeArea()

                ScrollView {
                    VStack(spacing: 24) {
                        headerSection

                        VStack(spacing: 16) {
                            endpointsSection
                            credentialsSection

                            if let error = viewModel.loginError {
                                HStack(spacing: 8) {
                                    Image(systemName: "exclamationmark.triangle.fill")
                                        .foregroundColor(.red)
                                    Text(error)
                                        .font(.subheadline)
                                        .foregroundColor(.red)
                                }
                                .padding(12)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(Color.red.opacity(0.12))
                                .clipShape(RoundedRectangle(cornerRadius: 12))
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel("Errore di accesso: \(error)")
                            }

                            loginButton
                        }
                        .padding(.horizontal)
                    }
                    .padding(.vertical, 24)
                }
            }
            .navigationTitle("Accesso Domus")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                if viewModel.authSession.isAuthenticated {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Annulla") {
                            dismiss()
                        }
                    }
                }
            }
            .onAppear {
                let savedLocal = UserDefaults.standard.string(forKey: "domus_local_endpoint")
                    ?? UserDefaults.standard.string(forKey: "domus_server_url")
                    ?? "http://192.168.1.100:8080"
                let savedRemote = UserDefaults.standard.string(forKey: "domus_remote_endpoint") ?? ""
                self.localUrl = savedLocal
                self.remoteUrl = savedRemote
            }
        }
    }

    // MARK: - Sections

    private var headerSection: some View {
        VStack(spacing: 8) {
            ZStack {
                Circle()
                    .fill(
                        LinearGradient(
                            colors: [Color.blue, Color.cyan],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        )
                    )
                    .frame(width: 80, height: 80)
                    .shadow(color: Color.blue.opacity(0.3), radius: 12, x: 0, y: 6)

                Image(systemName: "house.fill")
                    .font(.system(size: 38, weight: .bold))
                    .foregroundColor(.white)
            }
            .accessibilityHidden(true)

            Text("Domus Hub")
                .font(.system(size: 26, weight: .black, design: .rounded))
                .foregroundColor(.primary)

            Text("Connettiti al tuo hub locale o da fuori casa")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
        }
        .padding(.top, 10)
    }

    private var endpointsSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("CONNETTIVITÀ & SERVER")
                .font(.caption.bold())
                .foregroundColor(.secondary)
                .padding(.horizontal, 4)

            VStack(spacing: 12) {
                // LAN Locale
                VStack(alignment: .leading, spacing: 6) {
                    Label("Indirizzo LAN Locale", systemImage: "wifi")
                        .font(.caption.bold())
                        .foregroundColor(.primary)

                    TextField("http://192.168.1.100:8080", text: $localUrl)
                        .textFieldStyle(.plain)
                        .autocorrectionDisabled()
                        #if os(iOS)
                        .textInputAutocapitalization(.never)
                        .keyboardType(.URL)
                        #endif
                        .padding(12)
                        .background(Color.domusSecondarySystemGroupedBackground)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                        .overlay(
                            RoundedRectangle(cornerRadius: 12)
                                .stroke(Color.primary.opacity(0.08), lineWidth: 1)
                        )
                }

                // Bottone mDNS Scan
                Button(action: scanMdns) {
                    HStack {
                        if isDiscoveringMdns {
                            ProgressView()
                                .padding(.trailing, 4)
                        } else {
                            Image(systemName: "dot.radiowaves.left.and.right")
                        }
                        Text(isDiscoveringMdns ? "Scansione rete locale in corso..." : "Trova Hub in LAN (Zero-Config mDNS)")
                            .font(.subheadline.bold())
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .background(Color.blue.opacity(0.12))
                    .foregroundColor(.blue)
                    .clipShape(RoundedRectangle(cornerRadius: 10))
                }
                .disabled(isDiscoveringMdns)
                .accessibilityLabel("Cerca automaticamente hub Domus sulla rete locale")

                if let discoveryMsg = discoveryMessage {
                    Text(discoveryMsg)
                        .font(.caption)
                        .foregroundColor(.green)
                        .padding(.horizontal, 4)
                }

                Divider()
                    .padding(.vertical, 4)

                // Indirizzo Fuori Casa (TLS)
                VStack(alignment: .leading, spacing: 6) {
                    HStack {
                        Label("Accesso Fuori Casa (TLS Remoto)", systemImage: "globe")
                            .font(.caption.bold())
                            .foregroundColor(.primary)
                        Spacer()
                        Text("Opzionale")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }

                    TextField("https://tuo-hub.duckdns.org:8443", text: $remoteUrl)
                        .textFieldStyle(.plain)
                        .autocorrectionDisabled()
                        #if os(iOS)
                        .textInputAutocapitalization(.never)
                        .keyboardType(.URL)
                        #endif
                        .padding(12)
                        .background(Color.domusSecondarySystemGroupedBackground)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                        .overlay(
                            RoundedRectangle(cornerRadius: 12)
                                .stroke(Color.primary.opacity(0.08), lineWidth: 1)
                        )

                    Text("Permette all'app di mantenere la connessione quando sei fuori casa su rete 4G/5G con crittografia end-to-end TLS.")
                        .font(.caption2)
                        .foregroundColor(.secondary)
                        .padding(.horizontal, 4)
                }
            }
            .padding(16)
            .background(Color.domusSystemBackground)
            .clipShape(RoundedRectangle(cornerRadius: 16))
            .shadow(color: Color.black.opacity(0.04), radius: 6, x: 0, y: 3)
        }
    }

    private var credentialsSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("CREDENZIALI")
                .font(.caption.bold())
                .foregroundColor(.secondary)
                .padding(.horizontal, 4)

            VStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 6) {
                    Label("Nome Utente", systemImage: "person.fill")
                        .font(.caption.bold())
                        .foregroundColor(.primary)

                    TextField("Nome utente", text: $username)
                        .textFieldStyle(.plain)
                        .autocorrectionDisabled()
                        #if os(iOS)
                        .textInputAutocapitalization(.never)
                        #endif
                        .padding(12)
                        .background(Color.domusSecondarySystemGroupedBackground)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                        .overlay(
                            RoundedRectangle(cornerRadius: 12)
                                .stroke(Color.primary.opacity(0.08), lineWidth: 1)
                        )
                }

                VStack(alignment: .leading, spacing: 6) {
                    Label("Password", systemImage: "lock.fill")
                        .font(.caption.bold())
                        .foregroundColor(.primary)

                    SecureField("Password", text: $password)
                        .textFieldStyle(.plain)
                        .padding(12)
                        .background(Color.domusSecondarySystemGroupedBackground)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                        .overlay(
                            RoundedRectangle(cornerRadius: 12)
                                .stroke(Color.primary.opacity(0.08), lineWidth: 1)
                        )
                }

                if viewModel.requiresTwoFactor {
                    VStack(alignment: .leading, spacing: 6) {
                        Label("Codice di verifica 2FA", systemImage: "lock.shield")
                            .font(.caption.bold())
                            .foregroundColor(.primary)
                        TextField("Codice TOTP o di recupero", text: $twoFactorCode)
                            .textFieldStyle(.plain)
                            .textContentType(.oneTimeCode)
                            .textInputAutocapitalization(.characters)
                            .autocorrectionDisabled()
                            .padding(12)
                            .background(Color.domusSecondarySystemGroupedBackground)
                            .clipShape(RoundedRectangle(cornerRadius: 12))
                    }
                }
            }
            .padding(16)
            .background(Color.domusSystemBackground)
            .clipShape(RoundedRectangle(cornerRadius: 16))
            .shadow(color: Color.black.opacity(0.04), radius: 6, x: 0, y: 3)
        }
    }

    private var loginButton: some View {
        Button(action: performLogin) {
            HStack {
                if viewModel.isLoggingIn {
                    ProgressView()
                        .tint(.white)
                        .padding(.trailing, 6)
                }
                Text(viewModel.isLoggingIn ? "Connessione in corso..." : "Accedi al Server")
                    .font(.headline.bold())
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .background(
                LinearGradient(
                    colors: [Color.blue, Color.blue.opacity(0.85)],
                    startPoint: .leading,
                    endPoint: .trailing
                )
            )
            .foregroundColor(.white)
            .clipShape(RoundedRectangle(cornerRadius: 14))
            .shadow(color: Color.blue.opacity(0.25), radius: 8, x: 0, y: 4)
        }
        .disabled(viewModel.isLoggingIn || username.isEmpty)
        .accessibilityLabel("Effettua l'accesso a Domus")
    }

    // MARK: - Actions

    private func scanMdns() {
        isDiscoveringMdns = true
        discoveryMessage = nil
        Task {
            if let hubUrl = await viewModel.discoverLocalHub() {
                self.localUrl = hubUrl
                self.discoveryMessage = "✓ Hub rilevato: \(hubUrl)"
            } else {
                self.discoveryMessage = "Nessun hub rilevato con mDNS. Inserisci l'IP manualmente."
            }
            isDiscoveringMdns = false
        }
    }

    private func performLogin() {
        Haptics.medium()
        let cleanLocal = localUrl.trimmingCharacters(in: .whitespaces)
        let cleanRemote = remoteUrl.trimmingCharacters(in: .whitespaces)
        let localCandidate = cleanLocal.isEmpty ? nil : cleanLocal
        let remoteCandidate = cleanRemote.isEmpty ? nil : cleanRemote

        Task {
            let success = await viewModel.login(
                username: username,
                password: password,
                localEndpoint: localCandidate,
                remoteEndpoint: remoteCandidate,
                twoFactorCode: viewModel.requiresTwoFactor ? twoFactorCode : nil
            )
            if success {
                Haptics.success()
                dismiss()
            } else {
                Haptics.error()
            }
        }
    }
}
