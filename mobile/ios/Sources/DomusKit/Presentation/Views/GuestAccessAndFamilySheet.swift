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

    @State private var accounts: [DomusUserAccount] = DomusUserAccount.sampleAccounts
    @State private var showingQRGenerator: Bool = false
    @State private var guestName: String = "Ospite"
    @State private var guestDurationHours: Int = 24
    @State private var selectedGuestRooms: Set<String> = ["Camera Ospiti", "Bagno Ospiti", "Salotto"]
    @State private var generatedQRImage: CGImage? = nil

    private let availableRooms = ["Salotto", "Cucina", "Camera Padronale", "Camera Ospiti", "Bagno Ospiti", "Giardino", "Garage"]

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
                            Text("Configura permessi differenziati per bambini, familiari e genera pass temporanei.")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }
                    .padding(.vertical, 4)
                }

                // Generatore Rapido Pass Ospite QR Code
                Section(header: Text("Condivisione Rapida Ospiti")) {
                    Button(action: {
                        generateGuestQRCode()
                        showingQRGenerator = true
                    }) {
                        HStack(spacing: 12) {
                            Image(systemName: "qrcode.viewfinder")
                                .font(.title2)
                                .foregroundColor(.purple)
                            VStack(alignment: .leading, spacing: 2) {
                                Text("Crea Pass Ospite 24h (QR Code)")
                                    .font(.subheadline.bold())
                                    .foregroundColor(.primary)
                                Text("L'ospite scansiona il codice per controllare solo le sue stanze.")
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
                    ForEach(accounts) { account in
                        accountRow(account)
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
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Chiudi") {
                        dismiss()
                    }
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
                        generateGuestQRCode()
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
                            generateGuestQRCode()
                        }
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("\(room), \(selectedGuestRooms.contains(room) ? "selezionata" : "non selezionata")")
                    }
                }

                Section(header: Text("Codice QR da Scansionare")) {
                    VStack(spacing: 14) {
                        if let qr = generatedQRImage {
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
                            ProgressView("Generazione codice...")
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
                generateGuestQRCode()
            }
        }
    }

    // MARK: - Generatore QR Code con CoreImage

    private func generateGuestQRCode() {
        let payload = """
        {"hub":"domus","guest":"\(guestName)","duration_hours":\(guestDurationHours),"rooms":\(Array(selectedGuestRooms)),"token":"\(UUID().uuidString.prefix(8))"}
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
        let newGuest = DomusUserAccount(
            name: guestName.isEmpty ? "Ospite" : guestName,
            role: .guest,
            allowedRooms: Array(selectedGuestRooms),
            canDisarmAlarm: false,
            canViewCameras: false,
            expiresAt: Calendar.current.date(byAdding: .hour, value: guestDurationHours, to: Date())
        )
        accounts.append(newGuest)
        showingQRGenerator = false
        Haptics.success()
        DomusAccessibility.announce("Pass ospite per \(newGuest.name) attivato")
    }

    private func revokeGuest(_ account: DomusUserAccount) {
        accounts.removeAll(where: { $0.id == account.id })
        Haptics.heavy()
        DomusAccessibility.announce("Accesso ospite revocato")
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
