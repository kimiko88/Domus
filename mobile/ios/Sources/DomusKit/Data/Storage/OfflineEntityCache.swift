import Foundation

/// Cache in-memory delle entità Domus per il funzionamento offline.
/// In futuro può essere sostituita con SwiftData per persistenza su disco.
public actor OfflineEntityCache: Sendable {

    private var entities: [String: Entity] = [:]
    private var areas: [String: Area] = [:]

    public init() {}

    // MARK: - Entities

    public func saveEntities(_ list: [Entity]) {
        for entity in list {
            entities[entity.id] = entity
        }
    }

    public func getEntities() -> [Entity] {
        Array(entities.values).sorted { $0.name < $1.name }
    }

    public func getEntity(id: String) -> Entity? {
        entities[id]
    }

    public func updateEntity(_ entity: Entity) {
        entities[entity.id] = entity
    }

    public func deleteEntity(id: String) {
        entities.removeValue(forKey: id)
    }

    public func updateEntities(_ list: [Entity]) {
        saveEntities(list)
    }

    public func upsertEntity(_ entity: Entity) {
        updateEntity(entity)
    }

    // MARK: - Areas

    public func saveAreas(_ list: [Area]) {
        for area in list {
            areas[area.id] = area
        }
    }

    public func updateAreas(_ list: [Area]) {
        saveAreas(list)
    }

    public func getAreas() -> [Area] {
        Array(areas.values).sorted { $0.name < $1.name }
    }

    public func getAllAreas() -> [Area] {
        getAreas()
    }

    public func clearAreas() {
        areas.removeAll()
    }
}

