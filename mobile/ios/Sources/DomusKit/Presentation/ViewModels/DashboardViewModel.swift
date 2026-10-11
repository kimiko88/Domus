import Foundation
import SwiftUI
import Combine

/// Dashboard ViewModel per iOS — gestisce lo stato reattivo e i comandi della Fase 2.
@MainActor
public final class DashboardViewModel: ObservableObject {
    @Published public var connection: ServerConnection = ServerConnection()
    @Published public var entities: [Entity] = []
    @Published public var areas: [Area] = []
    @Published public var selectedAreaId: String? = nil
    @Published public var searchQuery: String = ""
    @Published public var isRefreshing: Bool = false
    @Published public var selectedEntityForDetail: Entity? = nil
    @Published public var showAutomationSettings: Bool = false
    @Published public var showKioskSettings: Bool = false

    public let presenceManager: GeofencingPresenceManager
    public let notificationManager: NotificationManager
    public let nfcManager: NfcTagManager
    public let kioskManager: KioskModeManager
    public let watchSessionManager: WatchSessionManager

    private let entityRepository: EntityRepositoryProtocol
    private let connectionRepository: ConnectionRepositoryProtocol
    private let discoverAndConnectUseCase: DiscoverAndConnectUseCase
    private let syncEntitiesUseCase: SyncEntitiesUseCase
    private let toggleEntityUseCase: ToggleEntityUseCase

    public init(
        entityRepository: EntityRepositoryProtocol,
        connectionRepository: ConnectionRepositoryProtocol,
        discoverAndConnectUseCase: DiscoverAndConnectUseCase,
        syncEntitiesUseCase: SyncEntitiesUseCase,
        toggleEntityUseCase: ToggleEntityUseCase,
        presenceManager: GeofencingPresenceManager? = nil,
        notificationManager: NotificationManager? = nil,
        nfcManager: NfcTagManager? = nil,
        kioskManager: KioskModeManager? = nil,
        watchSessionManager: WatchSessionManager? = nil
    ) {
        self.entityRepository = entityRepository
        self.connectionRepository = connectionRepository
        self.discoverAndConnectUseCase = discoverAndConnectUseCase
        self.syncEntitiesUseCase = syncEntitiesUseCase
        self.toggleEntityUseCase = toggleEntityUseCase
        self.presenceManager = presenceManager ?? GeofencingPresenceManager()
        self.notificationManager = notificationManager ?? NotificationManager()
        self.nfcManager = nfcManager ?? NfcTagManager()
        self.kioskManager = kioskManager ?? KioskModeManager()
        self.watchSessionManager = watchSessionManager ?? WatchSessionManager()

        setupAutomationCallbacks()
    }

    public var filteredEntities: [Entity] {
        entities.filter { entity in
            let matchesArea = selectedAreaId == nil || entity.areaId == selectedAreaId
            let matchesQuery = searchQuery.trimmingCharacters(in: .whitespaces).isEmpty ||
                entity.name.localizedCaseInsensitiveContains(searchQuery) ||
                entity.domain.localizedCaseInsensitiveContains(searchQuery)
            return matchesArea && matchesQuery
        }
    }

    public func initialConnectAndSync() {
        Task {
            isRefreshing = true
            _ = try? await discoverAndConnectUseCase.execute(preferLocal: true)
            self.connection = await connectionRepository.getCurrentConnection()
            self.entities = (try? await syncEntitiesUseCase.execute()) ?? []
            self.areas = await entityRepository.getAreas()
            isRefreshing = false
        }
    }

    public func toggleEntity(id: String) {
        Haptics.medium()
        Task {
            if let updated = try? await toggleEntityUseCase.execute(entityId: id) {
                if let index = entities.firstIndex(where: { $0.id == id }) {
                    entities[index] = updated
                    if selectedEntityForDetail?.id == id {
                        selectedEntityForDetail = updated
                    }
                }
            }
        }
    }

    public func selectArea(_ areaId: String?) {
        Haptics.selection()
        self.selectedAreaId = areaId
    }

    public func selectEntityForDetail(_ entity: Entity?) {
        if entity != nil {
            Haptics.light()
        }
        self.selectedEntityForDetail = entity
    }

    public func reconnect() {
        initialConnectAndSync()
    }

    // MARK: - Fase 2: Controlli Luce (Luminosità, HSL, Temperatura Colore)

    public func setLightBrightness(id: String, brightness: Int) {
        guard let index = entities.firstIndex(where: { $0.id == id }) else { return }
        var current = entities[index]
        current.attributes["brightness"] = AnyCodable(brightness)
        if brightness > 0 { current.state = "on" }
        entities[index] = current
        if selectedEntityForDetail?.id == id { selectedEntityForDetail = current }

        Task {
            try? await entityRepository.callService(
                domain: "light",
                service: "turn_on",
                entityId: id,
                data: ["brightness": brightness]
            )
        }
    }

    public func setLightColor(id: String, r: Double, g: Double, b: Double) {
        guard let index = entities.firstIndex(where: { $0.id == id }) else { return }
        var current = entities[index]
        let rgbArray = [Int(r * 255), Int(g * 255), Int(b * 255)]
        current.attributes["rgb_color"] = AnyCodable(rgbArray)
        current.state = "on"
        entities[index] = current
        if selectedEntityForDetail?.id == id { selectedEntityForDetail = current }

        Task {
            try? await entityRepository.callService(
                domain: "light",
                service: "turn_on",
                entityId: id,
                data: ["rgb_color": "\(rgbArray)"]
            )
        }
    }

    public func setLightColorTemp(id: String, mireds: Int) {
        guard let index = entities.firstIndex(where: { $0.id == id }) else { return }
        var current = entities[index]
        current.attributes["color_temp"] = AnyCodable(mireds)
        current.state = "on"
        entities[index] = current
        if selectedEntityForDetail?.id == id { selectedEntityForDetail = current }

        Task {
            try? await entityRepository.callService(
                domain: "light",
                service: "turn_on",
                entityId: id,
                data: ["color_temp": mireds]
            )
        }
    }

    // MARK: - Fase 2: Termostato Circolare & HVAC

    public func setThermostatTemperature(id: String, targetTemp: Double) {
        guard let index = entities.firstIndex(where: { $0.id == id }) else { return }
        var current = entities[index]
        current.attributes["temperature"] = AnyCodable(targetTemp)
        entities[index] = current
        if selectedEntityForDetail?.id == id { selectedEntityForDetail = current }

        Task {
            try? await entityRepository.callService(
                domain: "climate",
                service: "set_temperature",
                entityId: id,
                data: ["temperature": targetTemp]
            )
        }
    }

    public func setThermostatHvacMode(id: String, mode: String) {
        guard let index = entities.firstIndex(where: { $0.id == id }) else { return }
        var current = entities[index]
        current.state = mode
        entities[index] = current
        if selectedEntityForDetail?.id == id { selectedEntityForDetail = current }

        Task {
            try? await entityRepository.callService(
                domain: "climate",
                service: "set_hvac_mode",
                entityId: id,
                data: ["hvac_mode": mode]
            )
        }
    }

    // MARK: - Fase 2: Tastierino Allarme (PIN, Arm Home/Away, Disarm)

    public func armAlarm(id: String, code: String, mode: String) {
        Task {
            do {
                try await entityRepository.callService(
                    domain: "alarm_control_panel",
                    service: mode,
                    entityId: id,
                    data: ["code": code]
                )
                Haptics.success()
                if let index = self.entities.firstIndex(where: { $0.id == id }) {
                    let targetState = mode == "alarm_arm_home" ? "armed_home" : "armed_away"
                    self.entities[index].state = targetState
                    if self.selectedEntityForDetail?.id == id {
                        self.selectedEntityForDetail?.state = targetState
                    }
                }
            } catch {
                Haptics.error()
            }
        }
    }

    public func disarmAlarm(id: String, code: String) {
        Task {
            do {
                try await entityRepository.callService(
                    domain: "alarm_control_panel",
                    service: "alarm_disarm",
                    entityId: id,
                    data: ["code": code]
                )
                Haptics.success()
                if let index = self.entities.firstIndex(where: { $0.id == id }) {
                    self.entities[index].state = "disarmed"
                    if self.selectedEntityForDetail?.id == id {
                        self.selectedEntityForDetail?.state = "disarmed"
                    }
                }
            } catch {
                Haptics.error()
            }
        }
    }

    // MARK: - Fase 2: Telecamere

    public func refreshCamera(id: String) {
        Haptics.light()
        Task {
            // Trigger refresh
            _ = try? await syncEntitiesUseCase.execute()
        }
    }

    // MARK: - Fase 3: Automazioni NFC & Notifiche Azionabili

    private func setupAutomationCallbacks() {
        nfcManager.onTagScanned = { [weak self] tag in
            guard let self else { return }
            self.handleNfcTag(tag)
        }

        notificationManager.onActionTriggered = { [weak self] actionId, categoryId, userInfo in
            guard let self else { return }
            self.handleNotificationAction(actionId: actionId, categoryId: categoryId, userInfo: userInfo)
        }

        kioskManager.onSmartPlugCommand = { [weak self] plugId, turnOn in
            guard let self else { return }
            Task {
                let service = turnOn ? "turn_on" : "turn_off"
                try? await self.entityRepository.callService(
                    domain: "switch",
                    service: service,
                    entityId: plugId,
                    data: [:]
                )
            }
        }

        watchSessionManager.onWatchCommandReceived = { [weak self] action, targetId in
            guard let self else { return }
            if action == "toggle" {
                self.toggleEntity(id: targetId)
            }
        }
    }


    private func handleNfcTag(_ tag: DomusNfcTag) {
        switch tag.actionType {
        case "toggle":
            toggleEntity(id: tag.targetId)
        case "scene":
            Task {
                try? await entityRepository.callService(
                    domain: "scene",
                    service: "turn_on",
                    entityId: tag.targetId,
                    data: [:]
                )
            }
        case "alarm":
            Task {
                try? await entityRepository.callService(
                    domain: "alarm_control_panel",
                    service: tag.targetId,
                    entityId: "alarm_control_panel.home",
                    data: [:]
                )
            }
        default:
            break
        }
    }

    private func handleNotificationAction(actionId: String, categoryId: String, userInfo: [AnyHashable: Any]) {
        switch actionId {
        case DomusNotificationAction.closeCovers.rawValue:
            Task {
                try? await entityRepository.callService(
                    domain: "cover",
                    service: "close_cover",
                    entityId: "all",
                    data: [:]
                )
            }
        case DomusNotificationAction.disarmAlarm.rawValue:
            disarmAlarm(id: "alarm_control_panel.home", code: "")
        default:
            break
        }
    }
}

