import Foundation
import Combine

#if canImport(CarPlay) && os(iOS)
import CarPlay
import UIKit

/// Gestore dell'interfaccia Apple CarPlay sul cruscotto dell'automobile.
/// Fornisce pulsanti rapidi ad alta visibilità per cancello, garage, luci esterne e disarmo allarme.
@MainActor
public final class DomusCarPlayManager: NSObject, ObservableObject {
    public static let shared = DomusCarPlayManager()

    public var interfaceController: CPInterfaceController?
    public var onCarPlayAction: ((String, String) -> Void)? // (domain, entityId)

    @Published public private(set) var isConnectedToCar: Bool = false

    public func connect(interfaceController: CPInterfaceController) {
        self.interfaceController = interfaceController
        self.isConnectedToCar = true
        setupCarPlayRootTemplate()
    }

    public func disconnect() {
        self.interfaceController = nil
        self.isConnectedToCar = false
    }

    /// Configura il template a griglia rapida per l'infotainment auto.
    public func setupCarPlayRootTemplate() {
        guard let interfaceController = interfaceController else { return }

        let gateButton = CPGridButton(
            titleVariants: ["Cancello", "Apri Cancello"],
            image: UIImage(systemName: "door.garage.open") ?? UIImage()
        ) { [weak self] _ in
            self?.triggerAction(domain: "cover", entityId: "cover.gate")
        }

        let garageButton = CPGridButton(
            titleVariants: ["Garage", "Basculante"],
            image: UIImage(systemName: "car.side.fill") ?? UIImage()
        ) { [weak self] _ in
            self?.triggerAction(domain: "cover", entityId: "cover.garage")
        }

        let outdoorLightsButton = CPGridButton(
            titleVariants: ["Luci Viale", "Luci Esterne"],
            image: UIImage(systemName: "lightbulb.fill") ?? UIImage()
        ) { [weak self] _ in
            self?.triggerAction(domain: "light", entityId: "light.outdoor_lights")
        }

        let disarmAlarmButton = CPGridButton(
            titleVariants: ["Disarma", "Disarma Allarme"],
            image: UIImage(systemName: "lock.open.shield.fill") ?? UIImage()
        ) { [weak self] _ in
            self?.triggerAction(domain: "alarm_control_panel", entityId: "alarm_control_panel.home")
        }

        let gridTemplate = CPGridTemplate(
            title: "Domus Auto",
            gridButtons: [gateButton, garageButton, outdoorLightsButton, disarmAlarmButton]
        )

        interfaceController.setRootTemplate(gridTemplate, animated: false, completion: nil)
    }

    private func triggerAction(domain: String, entityId: String) {
        onCarPlayAction?(domain, entityId)
    }
}

#else

/// Stub compatibile per target senza CarPlay o macOS
@MainActor
public final class DomusCarPlayManager: ObservableObject {
    public static let shared = DomusCarPlayManager()

    @Published public private(set) var isConnectedToCar: Bool = false
    public var onCarPlayAction: ((String, String) -> Void)?

    public init() {}

    public func simulateCarAction(domain: String, entityId: String) {
        onCarPlayAction?(domain, entityId)
    }
}

#endif
