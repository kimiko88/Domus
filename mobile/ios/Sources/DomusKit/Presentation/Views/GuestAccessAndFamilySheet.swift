import SwiftUI
import CoreImage

/// Ruolo o profilo di autorizzazione per gli utenti della casa.
public enum UserAccessRole: String, CaseIterable, Identifiable, Sendable {
    case admin = "Amministratore"
    case family = "Membro Famiglia"
    case child = "Bambino"
    case guest = "Ospite Temporaneo"

    public var id: String { rawValue }

    public var description: String {
        switch self {
        case .admin:
            return "Accesso completo a tutti i dispositivi, allarmi, impostazioni e backup."
        case .family:
            return "Controllo di tutta la casa e allarme. Nessun accesso a impostazioni di sistema."
        case .child:
            return "Controllo limitato alla camera da letto e salotto. Allarme e serrature bloccati."
        case .guest:
            return "Accesso limitato alle stanze selezionate, con scadenza temporale automatica."
        }
    }

    public var icon: String {
        switch self {
        case .admin: return "person.badge.key.fill"
        case .family: return "person.2.fill"
        case .child: return "figure.and.child.holdinghands"
        case .guest: return "qrcode"
        }
    }
}

/// Modello di un account o ospite autorizzato.
public struct DomusUserAccount: Identifiable, Sendable {
    public let id: String
    public var name: String
    public var role: UserAccessRole
    public var allowedRooms: [String]
    public var canDisarmAlarm: Bool
    public var canViewCameras: Bool
    public var expiresAt: Date?

    public init(
        id: String = UUID().uuidString,
        name: String,
        role: UserAccessRole,
        allowedRooms: [String] = [],
        canDisarmAlarm: Bool = false,
        canViewCameras: Bool = false,
        expiresAt: Date? = nil
    ) {
        self.id = id
        self.name = name
        self.role = role
        self.allowedRooms = allowedRooms
        self.canDisarmAlarm = canDisarmAlarm
        self.canViewCameras = canViewCameras
        self.expiresAt = expiresAt
    }

    public static let sampleAccounts: [DomusUserAccount] = [
        DomusUserAccount(
            id: "user.fabrizio",
            name: "Fabrizio (Tu)",
            role: .admin,
            allowedRooms: ["Tutte le stanze"],
            canDisarmAlarm: true,
            canViewCameras: true
        ),
        DomusUserAccount(
            id: "user.elena",
            name: "Elena",
            role: .family,
            allowedRooms: ["Tutte le stanze"],
            canDisarmAlarm: true,
            canViewCameras: true
        ),
        DomusUserAccount(
            id: "user.leo",
            name: "Leo (Bambino)",
            role: .child,
            allowedRooms: ["Cameretta", "Salotto"],
            canDisarmAlarm: false,
            canViewCameras: false
        ),
        DomusUserAccount(
            id: "user.guest_bnb",
            name: "Ospite Weekend",
            role: .guest,
            allowedRooms: ["Camera Ospiti", "Bagno Ospiti", "Salotto"],
            canDisarmAlarm: false,
            canViewCameras: false,
            expiresAt: Calendar.current.date(byAdding: .hour, value: 24, to: Date())
        )
    ]
}

/// Gestione Accesso Ospiti e Multi-Account con permessi granulari e condivisione QR Code 24h.
public struct GuestAccessAndFamilySheet: View {
    @Environment(\.dismiss) private var dismiss

    @State private var accounts: [DomusUserAccount] = []
    @State private var availableRoomsList: [String] = []
    @State private var showingQRGenerator: Bool = false
    @State private var guestName: String = "Ospite"
    @State private var guestDurationHours: Int = 24
    @State private var selectedGuestRooms: Set<String> = []
    @State private var generatedQRImage: CGImage? = nil
    @State private var latestCreatedInvite: FamilyInviteDTO? = nil
    @State private var isLoading: Bool = false
    @State private var isCreatingPass: Bool = false
    @State private var errorMessage: String? = nil

    private var availableRooms: [String] {
        if availableRoomsList.isEmpty {
            return ["Salotto", "Cucina", "Camera Padronale", "Camera Ospiti", "Bagno Ospiti", "Giardino", "Garage"]
        }
        return availableRoomsList
    }

    public init() {}

    public var body: some View {
        NavigationStack {
            List {
                // Sezione Introduttiva
                Section {
                    HStack(spacing: 12) {
                        Image(systemName: "person.2.badge.gearshape.fill")
                            .font(.system(size: 28))
                            .foregroundColor(.blue)
                        VStack(alignment: .leading, spacing: 2) {
                            Text("Membri & Accesso Ospiti")
                                .font(.headline)
                            Text("Configura permessi differenziati per bambini, familiari e genera pass temporanei sincronizzati sul server.")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }
                    .padding(.vertical, 4)
                }

                if let err = errorMessage {
                    Section {
                        HStack(spacing: 8) {
                            Image(systemName: "exclamationmark.triangle.fill")
                                .foregroundColor(.orange)
                            Text(err)
                                .font(.caption)
                                .foregroundColor(.orange)
                        }
                    }
                }

                // Generatore Rapido Pass Ospite QR Code
                Section(header: Text("Condivisione Rapida Ospiti")) {
                    Button(action: {
                        startGuestPassCreation()
                    }) {
                        HStack(spacing: 12) {
                            Image(systemName: "qrcode.viewfinder")
                                .font(.title2)
                                .foregroundColor(.purple)
                            VStack(alignment: .leading, spacing: 2) {
                                Text("Crea Pass Ospite (QR Code)")
                                    .font(.subheadline.bold())
                                    .foregroundColor(.primary)
                                Text("Genera un token crittografico sul server Domus per limitare l'accesso alle stanze scelte.")
                                    .font(.caption2)
                                    .foregroundColor(.secondary)
                            }
                            Spacer()
                            Image(systemName: "chevron.right")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                        .padding(.vertical, 4)
                    }
                }

                // Elenco Profili Esistenti
                Section(header: Text("Profili e Permessi (\(accounts.count))")) {
                    if isLoading && accounts.isEmpty {
                        HStack {
                            Spacer()
                            ProgressView("Caricamento profili autorizzati...")
                                .padding(.vertical, 12)
                            Spacer()
                        }
                    } else if accounts.isEmpty {
                        VStack(spacing: 8) {
                            Text("Nessun profilo o ospite configurato sull'hub.")
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                            Button(action: {
                                Task { await loadFamilyAndInvites() }
                            }) {
                                Text("Ricarica")
                                    .font(.caption.bold())
                                    .foregroundColor(.blue)
                            }
                        }
                        .padding(.vertical, 8)
                    } else {
                        ForEach(accounts) { account in
                            accountRow(account)
                        }
                    }
                }
            }
            #if os(iOS)
            .listStyle(.insetGrouped)
            #else
            .listStyle(.inset)
            #endif
            .navigationTitle("Accesso & Ospiti")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .refreshable {
                await loadFamilyAndInvites()
            }
            .task {
                await loadFamilyAndInvites()
            }
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Chiudi") {
                        dismiss()
                    }
                }
                ToolbarItem(placement: .primaryAction) {
                    Button(action: {
                        Task { await loadFamilyAndInvites() }
                    }) {
                        if isLoading {
                            ProgressView()
                        } else {
                            Image(systemName: "arrow.clockwise")
                        }
                    }
                    .accessibilityLabel("Aggiorna membri e pass")
                }
            }
            .sheet(isPresented: $showingQRGenerator) {
                guestQRModalView
            }
        }
    }

    // MARK: - Riga Profilo Account

    private func accountRow(_ account: DomusUserAccount) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                HStack(spacing: 8) {
                    Image(systemName: account.role.icon)
                        .foregroundColor(roleColor(account.role))
                    Text(account.name)
                        .font(.headline)
                }
                Spacer()
                Text(account.role.rawValue)
                    .font(.caption2.bold())
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(roleColor(account.role).opacity(0.12))
                    .foregroundColor(roleColor(account.role))
                    .clipShape(Capsule())
            }

            Text(account.role.description)
                .font(.caption)
                .foregroundColor(.secondary)

            // Permessi Chiave
            HStack(spacing: 8) {
                Label(account.allowedRooms.joined(separator: ", "), systemImage: "door.left.hand.open")
                    .font(.caption2)
                    .foregroundColor(.secondary)

                Spacer()

                if let exp = account.expiresAt {
                    Label("Scade tra \(hoursRemaining(exp))h", systemImage: "timer")
                        .font(.caption2.bold())
                        .foregroundColor(.orange)
                }
            }

            if account.role == .guest {
                Divider()
                HStack {
                    Spacer()
                    Button(role: .destructive, action: {
                        revokeGuest(account)
                    }) {
                        Text("Revoca Accesso Subito")
                            .font(.caption2.bold())
                            .foregroundColor(.red)
                    }
                }
            }
        }
        .padding(.vertical, 4)
    }

    // MARK: - Modale Generatore Pass Ospite QR Code

    private var guestQRModalView: some View {
        NavigationStack {
            Form {
                Section(header: Text("Configurazione Pass")) {
                    TextField("Nome Ospite (es. Marco)", text: $guestName)
                        .accessibilityLabel("Nome Ospite")

                    Picker("Validità del Pass", selection: $guestDurationHours) {
                        Text("12 Ore").tag(12)
                        Text("24 Ore (Standard)").tag(24)
                        Text("48 Ore (Weekend)").tag(48)
                        Text("7 Giorni").tag(168)
                    }
                    .onChange(of: guestDurationHours) { _, _ in
                        Task { await generateServerGuestPass() }
                    }
                }

                Section(header: Text("Stanze Autorizzate")) {
                    ForEach(availableRooms, id: \.self) { room in
                        HStack {
                            Text(room)
                            Spacer()
                            if selectedGuestRooms.contains(room) {
                                Image(systemName: "checkmark")
                                    .foregroundColor(.blue)
                            }
                        }
                        .contentShape(Rectangle())
                        .onTapGesture {
                            Haptics.selection()
                            if selectedGuestRooms.contains(room) {
                                selectedGuestRooms.remove(room)
                            } else {
                                selectedGuestRooms.insert(room)
                            }
                            if let inv = latestCreatedInvite {
                                renderQRCode(for: inv)
                            }
                        }
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("\(room), \(selectedGuestRooms.contains(room) ? "selezionata" : "non selezionata")")
                    }
                }

                Section(header: Text("Codice QR da Scansionare")) {
                    VStack(spacing: 14) {
                        if isCreatingPass {
                            ProgressView("Generazione token sicuro sul server...")
                                .frame(height: 200)
                        } else if let qr = generatedQRImage {
                            Image(decorative: qr, scale: 1.0)
                                .resizable()
                                .interpolation(.none)
                                .scaledToFit()
                                .frame(width: 200, height: 200)
                                .padding(12)
                                .background(Color.white)
                                .clipShape(RoundedRectangle(cornerRadius: 16))
                                .shadow(color: Color.black.opacity(0.15), radius: 8, x: 0, y: 4)
                        } else {
                            VStack(spacing: 8) {
                                Image(systemName: "exclamationmark.triangle")
                                    .foregroundColor(.orange)
                                Text("Nessun token generato")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                                Button("Genera Token") {
                                    Task { await generateServerGuestPass() }
                                }
                                .font(.caption.bold())
                            }
                            .frame(height: 200)
                        }

                        Text("Fai inquadrare questo codice con la fotocamera dell'ospite per consentire l'accesso immediato senza password.")
                            .font(.caption)
                            .foregroundColor(.secondary)
                            .multilineTextAlignment(.center)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 8)
                }

                Section {
                    Button(action: {
                        confirmCreateGuest()
                    }) {
                        HStack {
                            Spacer()
                            Image(systemName: "checkmark")
                            Text("Salva e Attiva Pass")
                            Spacer()
                        }
                        .font(.headline)
                        .foregroundColor(.white)
                        .frame(minHeight: 44)
                    }
                    .listRowBackground(Color.blue)
                }
            }
            .navigationTitle("Pass Ospite QR")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Chiudi") {
                        showingQRGenerator = false
                    }
                }
            }
            .onAppear {
                if latestCreatedInvite == nil {
                    Task { await generateServerGuestPass() }
                }
            }
        }
    }

    // MARK: - Logica Sincronizzazione Server

    private func loadFamilyAndInvites() async {
        isLoading = true
        errorMessage = nil
        do {
            async let membersReq = DomusHttpClient.shared.fetchFamilyMembers()
            async let invitesReq = DomusHttpClient.shared.fetchFamilyInvites()
            async let areasReq = DomusHttpClient.shared.fetchAreas()

            let (members, invites, areas) = try await (membersReq, invitesReq, areasReq)

            var parsedAccounts: [DomusUserAccount] = []

            for m in members {
                let role: UserAccessRole
                switch (m.role ?? "").lowercased() {
                case "admin": role = .admin
                case "child": role = .child
                case "guest": role = .guest
                default: role = .family
                }

                parsedAccounts.append(
                    DomusUserAccount(
                        id: m.id,
                        name: m.username,
                        role: role,
                        allowedRooms: ["Tutte le stanze"],
                        canDisarmAlarm: role == .admin || role == .family,
                        canViewCameras: role == .admin || role == .family
                    )
                )
            }

            let isoFormatter = ISO8601DateFormatter()
            for inv in invites {
                let expDate = inv.expiresAt.flatMap { isoFormatter.date(from: $0) }
                let role: UserAccessRole = inv.role.lowercased() == "guest" ? .guest : .family
                let tokenShort = inv.token.count >= 6 ? String(inv.token.prefix(6)) : inv.token
                parsedAccounts.append(
                    DomusUserAccount(
                        id: inv.id,
                        name: "Pass Ospite (\(tokenShort))",
                        role: role,
                        allowedRooms: ["Stanze Ospiti"],
                        canDisarmAlarm: false,
                        canViewCameras: false,
                        expiresAt: expDate
                    )
                )
            }

            let roomNames = areas.map(\.name).filter { !$0.isEmpty }

            await MainActor.run {
                self.accounts = parsedAccounts
                self.availableRoomsList = roomNames
                if self.selectedGuestRooms.isEmpty, let firstRoom = roomNames.first {
                    self.selectedGuestRooms = [firstRoom]
                }
                self.isLoading = false
            }
        } catch {
            await MainActor.run {
                self.errorMessage = "Impossibile caricare profili dal server: \(error.localizedDescription)"
                self.isLoading = false
            }
        }
    }

    private func startGuestPassCreation() {
        if selectedGuestRooms.isEmpty {
            selectedGuestRooms = Set(availableRooms.prefix(3))
        }
        latestCreatedInvite = nil
        generatedQRImage = nil
        showingQRGenerator = true
        Task {
            await generateServerGuestPass()
        }
    }

    private func generateServerGuestPass() async {
        isCreatingPass = true
        generatedQRImage = nil
        do {
            let invite = try await DomusHttpClient.shared.createFamilyInvite(role: "guest", expiresInHours: guestDurationHours)
            await MainActor.run {
                self.latestCreatedInvite = invite
                self.renderQRCode(for: invite)
                self.isCreatingPass = false
            }
        } catch {
            await MainActor.run {
                self.errorMessage = "Creazione pass fallita sul server: \(error.localizedDescription)"
                self.isCreatingPass = false
            }
        }
    }

    private func renderQRCode(for invite: FamilyInviteDTO) {
        let payload = """
        {"hub":"domus","guest":"\(guestName)","duration_hours":\(guestDurationHours),"rooms":\(Array(selectedGuestRooms)),"token":"\(invite.token)"}
        """

        let filter = CIFilter(name: "CIQRCodeGenerator")
        filter?.setValue(payload.data(using: .utf8), forKey: "inputMessage")
        filter?.setValue("M", forKey: "inputCorrectionLevel")

        if let outputImage = filter?.outputImage {
            let context = CIContext()
            let scaledImage = outputImage.transformed(by: CGAffineTransform(scaleX: 10, y: 10))
            if let cgImage = context.createCGImage(scaledImage, from: scaledImage.extent) {
                self.generatedQRImage = cgImage
            }
        }
    }

    private func confirmCreateGuest() {
        if let inv = latestCreatedInvite {
            let isoFormatter = ISO8601DateFormatter()
            let expDate = inv.expiresAt.flatMap { isoFormatter.date(from: $0) }
                ?? Calendar.current.date(byAdding: .hour, value: guestDurationHours, to: Date())
            let tokenShort = inv.token.count >= 6 ? String(inv.token.prefix(6)) : inv.token
            let newGuest = DomusUserAccount(
                id: inv.id,
                name: guestName.isEmpty ? "Ospite (\(tokenShort))" : "\(guestName) (\(tokenShort))",
                role: .guest,
                allowedRooms: Array(selectedGuestRooms),
                canDisarmAlarm: false,
                canViewCameras: false,
                expiresAt: expDate
            )
            accounts.append(newGuest)
        }
        showingQRGenerator = false
        Haptics.success()
        DomusAccessibility.announce("Pass ospite attivato con successo sul server")
    }

    private func revokeGuest(_ account: DomusUserAccount) {
        accounts.removeAll(where: { $0.id == account.id })
        Haptics.heavy()
        DomusAccessibility.announce("Accesso ospite revocato")

        Task {
            try? await DomusHttpClient.shared.revokeFamilyInvite(id: account.id)
        }
    }

    private func roleColor(_ role: UserAccessRole) -> Color {
        switch role {
        case .admin: return .blue
        case .family: return .green
        case .child: return .orange
        case .guest: return .purple
        }
    }

    private func hoursRemaining(_ date: Date) -> Int {
        let diff = date.timeIntervalSince(Date())
        return max(0, Int(diff / 3600.0))
    }
}
