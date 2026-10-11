import Foundation
import LocalAuthentication

/// Tipi di biometria supportati dal dispositivo.
public enum DomusBiometryType: String, Sendable {
    case faceID = "Face ID"
    case touchID = "Touch ID"
    case opticID = "Optic ID"
    case none = "Nessuna Biometria"
}

/// Errori durante l'autenticazione biometrica.
public enum DomusBiometricError: Error, Equatable, Sendable {
    case notAvailable
    case canceled
    case failed
    case lockout
    case unknown(String)
}

/// Manager per l'autenticazione biometrica nativa con LocalAuthentication.
/// Permette lo sblocco immediato dell'allarme o di comandi critici con Face ID / Touch ID.
@MainActor
public final class LocalAuthenticationManager: ObservableObject {
    @Published public private(set) var biometryType: DomusBiometryType = .none
    @Published public private(set) var isBiometryAvailable: Bool = false
    @Published public private(set) var lastAuthSuccess: Bool = false
    @Published public private(set) var lastErrorMessage: String? = nil

    public var isSimulationMode: Bool = false

    public init(isSimulationMode: Bool = false) {
        self.isSimulationMode = isSimulationMode
        refreshBiometryStatus()
    }

    public func refreshBiometryStatus() {
        if isSimulationMode {
            self.biometryType = .faceID
            self.isBiometryAvailable = true
            return
        }

        let context = LAContext()
        var error: NSError?
        let canEvaluate = context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error)
        self.isBiometryAvailable = canEvaluate

        switch context.biometryType {
        case .faceID:
            self.biometryType = .faceID
        case .touchID:
            self.biometryType = .touchID
        case .opticID:
            self.biometryType = .opticID
        case .none:
            self.biometryType = .none
        @unknown default:
            self.biometryType = .none
        }
    }

    /// Autentica l'utente tramite biometria con motivazione mostrata nel prompt di sistema.
    public func authenticate(reason: String = "Conferma la tua identità per controllare Domus") async -> Result<Bool, DomusBiometricError> {
        if isSimulationMode {
            self.lastAuthSuccess = true
            self.lastErrorMessage = nil
            return .success(true)
        }

        let context = LAContext()
        context.localizedCancelTitle = "Annulla"
        context.localizedFallbackTitle = "Usa PIN"

        var authError: NSError?
        guard context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &authError) else {
            let mappedErr = mapLaError(authError)
            self.lastAuthSuccess = false
            self.lastErrorMessage = "Biometria non configurata o non disponibile"
            return .failure(mappedErr)
        }

        do {
            let success = try await context.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, localizedReason: reason)
            self.lastAuthSuccess = success
            self.lastErrorMessage = nil
            return .success(success)
        } catch let err as LAError {
            self.lastAuthSuccess = false
            let mapped = mapLaError(err as NSError)
            self.lastErrorMessage = err.localizedDescription
            return .failure(mapped)
        } catch {
            self.lastAuthSuccess = false
            self.lastErrorMessage = error.localizedDescription
            return .failure(.unknown(error.localizedDescription))
        }
    }

    private func mapLaError(_ error: NSError?) -> DomusBiometricError {
        guard let error = error else { return .failed }
        switch error.code {
        case LAError.userCancel.rawValue, LAError.appCancel.rawValue:
            return .canceled
        case LAError.biometryNotAvailable.rawValue, LAError.biometryNotEnrolled.rawValue:
            return .notAvailable
        case LAError.biometryLockout.rawValue:
            return .lockout
        default:
            return .failed
        }
    }
}
