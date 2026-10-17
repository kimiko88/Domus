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

    // Live entity states for real-time CarPlay dashboard badges
    @Published public var isGateOpen: Bool = false
    @Published public var isGarageOpen: Bool = false
    @Published public var isOutdoorLightsOn: Bool = false
    @Published public var isAlarmArmed: Bool = true
    @Published public var alarmMode: String = "Armato Perimetrale"

    public func connect(interfaceController: CPInterfaceController) {
        self.interfaceController = interfaceController
        self.isConnectedToCar = true
        setupCarPlayRootTemplate()
    }

    public func disconnect() {
        self.interfaceController = nil
        self.isConnectedToCar = false
    }

    /// Aggiorna lo stato in tempo reale e riflette i badge su CarPlay.
    public func updateLiveState(
        gateOpen: Bool? = nil,
        garageOpen: Bool? = nil,
        lightsOn: Bool? = nil,
        alarmArmed: Bool? = nil,
        alarmMode: String? = nil
    ) {
        if let g = gateOpen { self.isGateOpen = g }
        if let gr = garageOpen { self.isGarageOpen = gr }
        if let l = lightsOn { self.isOutdoorLightsOn = l }
        if let a = alarmArmed { self.isAlarmArmed = a }
        if let m = alarmMode { self.alarmMode = m }

        if isConnectedToCar {
            setupCarPlayRootTemplate()
        }
    }

    /// Configura il template a griglia rapida con stato dinamico in tempo reale per l'infotainment auto.
    public func setupCarPlayRootTemplate() {
        guard let interfaceController = interfaceController else { return }

        let gateBadge = isGateOpen ? "Aperto" : "Chiuso"
        let gateButton = CPGridButton(
            titleVariants: ["Cancello [\(gateBadge)]", "Cancello", isGateOpen ? "Chiudi Cancello" : "Apri Cancello"],
            image: UIImage(systemName: isGateOpen ? "door.garage.open" : "door.garage.closed") ?? UIImage()
        ) { [weak self] _ in
            self?.triggerAction(domain: "cover", entityId: "cover.gate")
        }

        let garageBadge = isGarageOpen ? "Aperto" : "Chiuso"
        let garageButton = CPGridButton(
            titleVariants: ["Garage [\(garageBadge)]", "Garage", isGarageOpen ? "Chiudi Garage" : "Apri Garage"],
            image: UIImage(systemName: isGarageOpen ? "car.side.arrowtriangle.up" : "car.side.fill") ?? UIImage()
        ) { [weak self] _ in
            self?.triggerAction(domain: "cover", entityId: "cover.garage")
        }

        let lightsBadge = isOutdoorLightsOn ? "Accese" : "Spente"
        let outdoorLightsButton = CPGridButton(
            titleVariants: ["Luci [\(lightsBadge)]", "Luci Viale", isOutdoorLightsOn ? "Spegni Luci" : "Accendi Luci"],
            image: UIImage(systemName: isOutdoorLightsOn ? "lightbulb.fill" : "lightbulb") ?? UIImage()
        ) { [weak self] _ in
            self?.triggerAction(domain: "light", entityId: "light.outdoor_lights")
        }

        let alarmBadge = isAlarmArmed ? alarmMode : "Disarmato"
        let disarmAlarmButton = CPGridButton(
            titleVariants: ["Allarme [\(alarmBadge)]", isAlarmArmed ? "Disarma" : "Arma Allarme"],
            image: UIImage(systemName: isAlarmArmed ? "shield.lefthalf.filled" : "lock.open.shield.fill") ?? UIImage()
        ) { [weak self] _ in
            self?.triggerAction(domain: "alarm_control_panel", entityId: "alarm_control_panel.home")
        }

        let gridTemplate = CPGridTemplate(
            title: "Domus Auto • \(isAlarmArmed ? "🛡️ Protetta" : "Disarmata")",
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

    // Live entity states for real-time CarPlay dashboard badges
    @Published public var isGateOpen: Bool = false
    @Published public var isGarageOpen: Bool = false
    @Published public var isOutdoorLightsOn: Bool = false
    @Published public var isAlarmArmed: Bool = true
    @Published public var alarmMode: String = "Armato Perimetrale"

    public var onCarPlayAction: ((String, String) -> Void)?

    public init() {}

    public func updateLiveState(
        gateOpen: Bool? = nil,
        garageOpen: Bool? = nil,
        lightsOn: Bool? = nil,
        alarmArmed: Bool? = nil,
        alarmMode: String? = nil
    ) {
        if let g = gateOpen { self.isGateOpen = g }
        if let gr = garageOpen { self.isGarageOpen = gr }
        if let l = lightsOn { self.isOutdoorLightsOn = l }
        if let a = alarmArmed { self.isAlarmArmed = a }
        if let m = alarmMode { self.alarmMode = m }
    }

    public func simulateCarAction(domain: String, entityId: String) {
        onCarPlayAction?(domain, entityId)
    }
}

#endif

