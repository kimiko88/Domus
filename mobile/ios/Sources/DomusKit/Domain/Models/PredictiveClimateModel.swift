import Foundation

/// Orientamento geografico della stanza
public enum RoomOrientation: String, Codable, CaseIterable, Sendable {
    case north = "north"
    case south = "south"
    case east = "east"
    case west = "west"

    public var localizedName: String {
        switch self {
        case .north: return NSLocalizedString("orientation_north", comment: "Nord")
        case .south: return NSLocalizedString("orientation_south", comment: "Sud")
        case .east: return NSLocalizedString("orientation_east", comment: "Est")
        case .west: return NSLocalizedString("orientation_west", comment: "Ovest")
        }
    }
}

/// Tipologia di corpo scaldante
public enum EmitterType: String, Codable, CaseIterable, Sendable {
    case radiator = "radiator"
    case floorRadiant = "floor_radiant"
    case fancoil = "fancoil"

    public var localizedName: String {
        switch self {
        case .radiator: return NSLocalizedString("emitter_radiator", comment: "Termosifoni")
        case .floorRadiant: return NSLocalizedString("emitter_floor_radiant", comment: "Riscaldamento a Pavimento")
        case .fancoil: return NSLocalizedString("emitter_fancoil", comment: "Ventilconvettore (Fan-Coil)")
        }
    }
}

/// Modalità operativa calcolata dall'algoritmo predittivo
public enum PredictiveMode: String, Codable, CaseIterable, Sendable {
    case comfort = "comfort"
    case coldFrontBoost = "cold_front_boost"
    case solarCutoff = "solar_cutoff"
    case preheat = "preheat"
    case eco = "eco"
    case frostProtection = "frost_protection"

    public var localizedName: String {
        switch self {
        case .comfort: return NSLocalizedString("climate_mode_comfort", comment: "Comfort")
        case .coldFrontBoost: return NSLocalizedString("climate_mode_cold_front_boost", comment: "Boost Fronte Freddo")
        case .solarCutoff: return NSLocalizedString("climate_mode_solar_cutoff", comment: "Cutoff Solare")
        case .preheat: return NSLocalizedString("climate_mode_preheat", comment: "Pre-Riscaldamento")
        case .eco: return NSLocalizedString("climate_mode_eco", comment: "Eco Risparmio")
        case .frostProtection: return NSLocalizedString("climate_mode_frost_protection", comment: "Protezione Antigelo")
        }
    }

    public var iconName: String {
        switch self {
        case .comfort: return "thermometer.medium"
        case .coldFrontBoost: return "wind.snow"
        case .solarCutoff: return "sun.max.fill"
        case .preheat: return "flame.fill"
        case .eco: return "leaf.fill"
        case .frostProtection: return "snowflake"
        }
    }
}

/// Configurazione della zona termica per il controllo predittivo
public struct PredictiveZoneConfig: Identifiable, Codable, Sendable, Equatable {
    public var id: String
    public var name: String
    public var areaId: String?
    public var trvEntityKey: String
    public var tempSensorKey: String
    public var pressureSensorKey: String?
    public var humiditySensorKey: String?
    public var windowSensorKey: String?
    public var targetTemperature: Double
    public var orientation: RoomOrientation
    public var windowAreaSqm: Double
    public var emitterType: EmitterType
    public var thermalInertiaHours: Double
    public var preheatEnabled: Bool
    public var solarCutoffEnabled: Bool
    public var pressureBoostEnabled: Bool
    public var enabled: Bool
    public var createdAt: String?
    public var updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case areaId = "area_id"
        case trvEntityKey = "trv_entity_key"
        case tempSensorKey = "temp_sensor_key"
        case pressureSensorKey = "pressure_sensor_key"
        case humiditySensorKey = "humidity_sensor_key"
        case windowSensorKey = "window_sensor_key"
        case targetTemperature = "target_temperature"
        case orientation
        case windowAreaSqm = "window_area_sqm"
        case emitterType = "emitter_type"
        case thermalInertiaHours = "thermal_inertia_hours"
        case preheatEnabled = "preheat_enabled"
        case solarCutoffEnabled = "solar_cutoff_enabled"
        case pressureBoostEnabled = "pressure_boost_enabled"
        case enabled
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    public init(
        id: String = UUID().uuidString,
        name: String,
        areaId: String? = nil,
        trvEntityKey: String,
        tempSensorKey: String,
        pressureSensorKey: String? = nil,
        humiditySensorKey: String? = nil,
        windowSensorKey: String? = nil,
        targetTemperature: Double = 20.0,
        orientation: RoomOrientation = .south,
        windowAreaSqm: Double = 2.0,
        emitterType: EmitterType = .radiator,
        thermalInertiaHours: Double = 1.5,
        preheatEnabled: Bool = true,
        solarCutoffEnabled: Bool = true,
        pressureBoostEnabled: Bool = true,
        enabled: Bool = true,
        createdAt: String? = nil,
        updatedAt: String? = nil
    ) {
        self.id = id
        self.name = name
        self.areaId = areaId
        self.trvEntityKey = trvEntityKey
        self.tempSensorKey = tempSensorKey
        self.pressureSensorKey = pressureSensorKey
        self.humiditySensorKey = humiditySensorKey
        self.windowSensorKey = windowSensorKey
        self.targetTemperature = targetTemperature
        self.orientation = orientation
        self.windowAreaSqm = windowAreaSqm
        self.emitterType = emitterType
        self.thermalInertiaHours = thermalInertiaHours
        self.preheatEnabled = preheatEnabled
        self.solarCutoffEnabled = solarCutoffEnabled
        self.pressureBoostEnabled = pressureBoostEnabled
        self.enabled = enabled
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }
}

/// Valutazione predittiva istantanea di una zona
public struct PredictiveEvaluation: Identifiable, Codable, Sendable, Equatable {
    public let id: String
    public let zoneId: String
    public let zoneName: String
    public let currentTemp: Double
    public let targetTemp: Double
    public let currentPressure: Double
    public let pressureTrendHpaHr: Double
    public let humidity: Double
    public let apparentTemp: Double
    public let outdoorTempForecast: Double
    public let directRadiationForecast: Double
    public let estimatedSolarGainKw: Double
    public let coldFrontDetected: Bool
    public let windowOpen: Bool
    public let calculatedSetpoint: Double
    public let mode: String
    public let reason: String
    public let evaluatedAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case zoneId = "zone_id"
        case zoneName = "zone_name"
        case currentTemp = "current_temp"
        case targetTemp = "target_temp"
        case currentPressure = "current_pressure"
        case pressureTrendHpaHr = "pressure_trend_hpa_hr"
        case humidity
        case apparentTemp = "apparent_temp"
        case outdoorTempForecast = "outdoor_temp_forecast"
        case directRadiationForecast = "direct_radiation_forecast"
        case estimatedSolarGainKw = "estimated_solar_gain_kw"
        case coldFrontDetected = "cold_front_detected"
        case windowOpen = "window_open"
        case calculatedSetpoint = "calculated_setpoint"
        case mode
        case reason
        case evaluatedAt = "evaluated_at"
    }

    public init(
        id: String = UUID().uuidString,
        zoneId: String,
        zoneName: String,
        currentTemp: Double,
        targetTemp: Double,
        currentPressure: Double = 1013.25,
        pressureTrendHpaHr: Double = 0.0,
        humidity: Double = 50.0,
        apparentTemp: Double = 20.0,
        outdoorTempForecast: Double = 10.0,
        directRadiationForecast: Double = 0.0,
        estimatedSolarGainKw: Double = 0.0,
        coldFrontDetected: Bool = false,
        windowOpen: Bool = false,
        calculatedSetpoint: Double,
        mode: String = "comfort",
        reason: String = "",
        evaluatedAt: String = ISO8601DateFormatter().string(from: Date())
    ) {
        self.id = id
        self.zoneId = zoneId
        self.zoneName = zoneName
        self.currentTemp = currentTemp
        self.targetTemp = targetTemp
        self.currentPressure = currentPressure
        self.pressureTrendHpaHr = pressureTrendHpaHr
        self.humidity = humidity
        self.apparentTemp = apparentTemp
        self.outdoorTempForecast = outdoorTempForecast
        self.directRadiationForecast = directRadiationForecast
        self.estimatedSolarGainKw = estimatedSolarGainKw
        self.coldFrontDetected = coldFrontDetected
        self.windowOpen = windowOpen
        self.calculatedSetpoint = calculatedSetpoint
        self.mode = mode
        self.reason = reason
        self.evaluatedAt = evaluatedAt
    }

    public var parsedMode: PredictiveMode {
        return PredictiveMode(rawValue: mode) ?? .comfort
    }
}

/// Stato dashboard di una singola zona predittiva
public struct ZoneDashboardStatus: Identifiable, Codable, Sendable, Equatable {
    public var id: String { config.id }
    public let config: PredictiveZoneConfig
    public let latestEvaluation: PredictiveEvaluation?

    enum CodingKeys: String, CodingKey {
        case config
        case latestEvaluation = "latest_evaluation"
    }

    public init(config: PredictiveZoneConfig, latestEvaluation: PredictiveEvaluation? = nil) {
        self.config = config
        self.latestEvaluation = latestEvaluation
    }
}

/// Stato aggregato di tutte le zone del sistema predittivo
public struct SystemPredictiveStatus: Codable, Sendable, Equatable {
    public let totalZones: Int
    public let enabledZones: Int
    public let coldFrontActive: Bool
    public let solarCutoffActive: Bool
    public let zones: [ZoneDashboardStatus]
    public let evaluatedAt: String

    enum CodingKeys: String, CodingKey {
        case totalZones = "total_zones"
        case enabledZones = "enabled_zones"
        case coldFrontActive = "cold_front_active"
        case solarCutoffActive = "solar_cutoff_active"
        case zones
        case evaluatedAt = "evaluated_at"
    }

    public init(
        totalZones: Int,
        enabledZones: Int,
        coldFrontActive: Bool,
        solarCutoffActive: Bool,
        zones: [ZoneDashboardStatus],
        evaluatedAt: String = ISO8601DateFormatter().string(from: Date())
    ) {
        self.totalZones = totalZones
        self.enabledZones = enabledZones
        self.coldFrontActive = coldFrontActive
        self.solarCutoffActive = solarCutoffActive
        self.zones = zones
        self.evaluatedAt = evaluatedAt
    }
}

/// Profilo termico appreso da osservazioni sul campo
public struct ThermalProfile: Codable, Sendable, Equatable {
    public let zoneId: String
    public let heatRatePerHr: Double
    public let coolRatePerHr: Double
    public let outdoorLossCoeff: Double
    public let solarBoostCoeff: Double
    public let samplesCount: Int
    public let lastTrainedAt: String

    enum CodingKeys: String, CodingKey {
        case zoneId = "zone_id"
        case heatRatePerHr = "heat_rate_per_hr"
        case coolRatePerHr = "cool_rate_per_hr"
        case outdoorLossCoeff = "outdoor_loss_coeff"
        case solarBoostCoeff = "solar_boost_coeff"
        case samplesCount = "samples_count"
        case lastTrainedAt = "last_trained_at"
    }

    public init(
        zoneId: String,
        heatRatePerHr: Double,
        coolRatePerHr: Double,
        outdoorLossCoeff: Double,
        solarBoostCoeff: Double,
        samplesCount: Int,
        lastTrainedAt: String
    ) {
        self.zoneId = zoneId
        self.heatRatePerHr = heatRatePerHr
        self.coolRatePerHr = coolRatePerHr
        self.outdoorLossCoeff = outdoorLossCoeff
        self.solarBoostCoeff = solarBoostCoeff
        self.samplesCount = samplesCount
        self.lastTrainedAt = lastTrainedAt
    }
}

/// Raccomandazione di avvio ottimale (Pre-riscaldamento intelligente)
public struct PreheatRecommendation: Identifiable, Codable, Sendable, Equatable {
    public var id: String { "\(zoneId)-\(targetTime)" }
    public let zoneId: String
    public let targetTime: String
    public let targetTemp: Double
    public let currentTemp: Double
    public let outdoorTemp: Double
    public let preheatMinutes: Int
    public let startTime: String
    public let effectiveHeatRate: Double
    public let reason: String

    enum CodingKeys: String, CodingKey {
        case zoneId = "zone_id"
        case targetTime = "target_time"
        case targetTemp = "target_temp"
        case currentTemp = "current_temp"
        case outdoorTemp = "outdoor_temp"
        case preheatMinutes = "preheat_minutes"
        case startTime = "start_time"
        case effectiveHeatRate = "effective_heat_rate"
        case reason
    }

    public init(
        zoneId: String,
        targetTime: String,
        targetTemp: Double,
        currentTemp: Double,
        outdoorTemp: Double,
        preheatMinutes: Int,
        startTime: String,
        effectiveHeatRate: Double,
        reason: String
    ) {
        self.zoneId = zoneId
        self.targetTime = targetTime
        self.targetTemp = targetTemp
        self.currentTemp = currentTemp
        self.outdoorTemp = outdoorTemp
        self.preheatMinutes = preheatMinutes
        self.startTime = startTime
        self.effectiveHeatRate = effectiveHeatRate
        self.reason = reason
    }
}

/// Raccomandazione di spegnimento anticipato (Coast-Down)
public struct CoastDownRecommendation: Identifiable, Codable, Sendable, Equatable {
    public var id: String { "\(zoneId)-\(departureTime)" }
    public let zoneId: String
    public let departureTime: String
    public let comfortLimitTemp: Double
    public let coastDownMinutes: Int
    public let stopTime: String
    public let energySavedKwh: Double
    public let reason: String

    enum CodingKeys: String, CodingKey {
        case zoneId = "zone_id"
        case departureTime = "departure_time"
        case comfortLimitTemp = "comfort_limit_temp"
        case coastDownMinutes = "coast_down_minutes"
        case stopTime = "stop_time"
        case energySavedKwh = "energy_saved_kwh"
        case reason
    }

    public init(
        zoneId: String,
        departureTime: String,
        comfortLimitTemp: Double,
        coastDownMinutes: Int,
        stopTime: String,
        energySavedKwh: Double,
        reason: String
    ) {
        self.zoneId = zoneId
        self.departureTime = departureTime
        self.comfortLimitTemp = comfortLimitTemp
        self.coastDownMinutes = coastDownMinutes
        self.stopTime = stopTime
        self.energySavedKwh = energySavedKwh
        self.reason = reason
    }
}

/// Risposta per l'elenco delle zone predittive
public struct PredictiveZonesResponseDTO: Codable, Sendable {
    public let zones: [PredictiveZoneConfig]
    public let total: Int

    public init(zones: [PredictiveZoneConfig], total: Int) {
        self.zones = zones
        self.total = total
    }
}
