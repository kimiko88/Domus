import Foundation

/// Luoghi di conservazione dei farmaci
public enum MedicineLocation: String, Codable, CaseIterable, Sendable {
    case armadiettoBagno = "armadietto_bagno"
    case cassettoNotte = "cassetto_notte"
    case borsaViaggio = "borsa_viaggio"
    case frigo = "frigo"
    case altro = "altro"

    public var localizedName: String {
        switch self {
        case .armadiettoBagno: return NSLocalizedString("medicine_loc_bathroom", comment: "Armadietto Bagno")
        case .cassettoNotte: return NSLocalizedString("medicine_loc_nightstand", comment: "Cassetto Notte")
        case .borsaViaggio: return NSLocalizedString("medicine_loc_travel_bag", comment: "Borsa Viaggio")
        case .frigo: return NSLocalizedString("medicine_loc_fridge", comment: "Frigorifero")
        case .altro: return NSLocalizedString("medicine_loc_other", comment: "Altro")
        }
    }

    public var iconName: String {
        switch self {
        case .armadiettoBagno: return "cross.case.fill"
        case .cassettoNotte: return "bed.double.fill"
        case .borsaViaggio: return "bag.fill"
        case .frigo: return "snowflake"
        case .altro: return "shippingbox.fill"
        }
    }
}

/// Unità di misura del farmaco
public enum MedicineUnit: String, Codable, CaseIterable, Sendable {
    case compresse = "compresse"
    case bustine = "bustine"
    case gocce = "gocce"
    case ml = "ml"
    case pz = "pz"
    case confezione = "confezione"

    public var localizedName: String {
        return NSLocalizedString("medicine_unit_\(rawValue)", comment: rawValue)
    }
}

/// Modello principale farmaco presente nell'armadietto
public struct Medicine: Identifiable, Codable, Sendable, Equatable {
    public let id: String
    public var name: String
    public var activeIngredient: String?
    public var dosage: String?
    public var form: String?
    public var aic: String?
    public var barcode: String?
    public var lotNumber: String?
    public var expirationDate: String
    public var paoDays: Int
    public var openedAt: String?
    public var effectiveExpirationDate: String
    public var location: MedicineLocation
    public var requiresColdChain: Bool
    public var quantity: Double
    public var unit: MedicineUnit
    public var minThreshold: Double
    public var prescriptionRequired: Bool
    public var notes: String?
    public var createdAt: String
    public var updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case activeIngredient = "active_ingredient"
        case dosage
        case form
        case aic
        case barcode
        case lotNumber = "lot_number"
        case expirationDate = "expiration_date"
        case paoDays = "pao_days"
        case openedAt = "opened_at"
        case effectiveExpirationDate = "effective_expiration_date"
        case location
        case requiresColdChain = "requires_cold_chain"
        case quantity
        case unit
        case minThreshold = "min_threshold"
        case prescriptionRequired = "prescription_required"
        case notes
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    public init(
        id: String = UUID().uuidString,
        name: String,
        activeIngredient: String? = nil,
        dosage: String? = nil,
        form: String? = nil,
        aic: String? = nil,
        barcode: String? = nil,
        lotNumber: String? = nil,
        expirationDate: String,
        paoDays: Int = 0,
        openedAt: String? = nil,
        effectiveExpirationDate: String? = nil,
        location: MedicineLocation = .armadiettoBagno,
        requiresColdChain: Bool = false,
        quantity: Double = 1.0,
        unit: MedicineUnit = .compresse,
        minThreshold: Double = 1.0,
        prescriptionRequired: Bool = false,
        notes: String? = nil,
        createdAt: String = ISO8601DateFormatter().string(from: Date()),
        updatedAt: String = ISO8601DateFormatter().string(from: Date())
    ) {
        self.id = id
        self.name = name
        self.activeIngredient = activeIngredient
        self.dosage = dosage
        self.form = form
        self.aic = aic
        self.barcode = barcode
        self.lotNumber = lotNumber
        self.expirationDate = expirationDate
        self.paoDays = paoDays
        self.openedAt = openedAt
        self.effectiveExpirationDate = effectiveExpirationDate ?? expirationDate
        self.location = location
        self.requiresColdChain = requiresColdChain
        self.quantity = quantity
        self.unit = unit
        self.minThreshold = minThreshold
        self.prescriptionRequired = prescriptionRequired
        self.notes = notes
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }

    // MARK: - Business Logic Helpers

    public var isOpened: Bool {
        return openedAt != nil && !(openedAt?.isEmpty ?? true)
    }

    public var isLowStock: Bool {
        return quantity <= minThreshold
    }

    public func parsedEffectiveExpirationDate() -> Date? {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(secondsFromGMT: 0)

        // Try YYYY-MM-DD
        formatter.dateFormat = "yyyy-MM-dd"
        if let d = formatter.date(from: effectiveExpirationDate) {
            return d
        }
        // Try ISO8601
        formatter.dateFormat = "yyyy-MM-dd'T'HH:mm:ssZ"
        if let d = formatter.date(from: effectiveExpirationDate) {
            return d
        }
        return ISO8601DateFormatter().date(from: effectiveExpirationDate)
    }

    public func daysUntilExpiration(from referenceDate: Date = Date()) -> Int? {
        guard let exp = parsedEffectiveExpirationDate() else { return nil }
        let calendar = Calendar.current
        let startOfRef = calendar.startOfDay(for: referenceDate)
        let startOfExp = calendar.startOfDay(for: exp)
        let components = calendar.dateComponents([.day], from: startOfRef, to: startOfExp)
        return components.day
    }

    public func isExpired(from referenceDate: Date = Date()) -> Bool {
        guard let days = daysUntilExpiration(from: referenceDate) else { return false }
        return days < 0
    }

    public func isExpiringSoon(daysThreshold: Int = 30, from referenceDate: Date = Date()) -> Bool {
        guard let days = daysUntilExpiration(from: referenceDate) else { return false }
        return days >= 0 && days <= daysThreshold
    }
}

/// Risultato del parsing di un codice DataMatrix / Farmadati / GS1
public struct DatamatrixParseResult: Codable, Sendable, Equatable {
    public let aic: String?
    public let gtin: String?
    public let expirationDate: String?
    public let lotNumber: String?
    public let serialNumber: String?
    public let rawCode: String

    enum CodingKeys: String, CodingKey {
        case aic
        case gtin
        case expirationDate = "expiration_date"
        case lotNumber = "lot_number"
        case serialNumber = "serial_number"
        case rawCode = "raw_code"
    }

    public init(
        aic: String? = nil,
        gtin: String? = nil,
        expirationDate: String? = nil,
        lotNumber: String? = nil,
        serialNumber: String? = nil,
        rawCode: String
    ) {
        self.aic = aic
        self.gtin = gtin
        self.expirationDate = expirationDate
        self.lotNumber = lotNumber
        self.serialNumber = serialNumber
        self.rawCode = rawCode
    }
}

/// Scheda AIFA di riferimento per il farmaco
public struct AIFARecord: Codable, Sendable, Equatable {
    public let aic: String
    public let name: String
    public let activeIngredient: String
    public let dosage: String
    public let form: String
    public let unit: MedicineUnit
    public let defaultQuantity: Double
    public let requiresColdChain: Bool
    public let defaultLocation: MedicineLocation
    public let defaultPaoDays: Int
    public let prescriptionRequired: Bool

    enum CodingKeys: String, CodingKey {
        case aic
        case name
        case activeIngredient = "active_ingredient"
        case dosage
        case form
        case unit
        case defaultQuantity = "default_quantity"
        case requiresColdChain = "requires_cold_chain"
        case defaultLocation = "default_location"
        case defaultPaoDays = "default_pao_days"
        case prescriptionRequired = "prescription_required"
    }

    public init(
        aic: String,
        name: String,
        activeIngredient: String,
        dosage: String,
        form: String,
        unit: MedicineUnit = .compresse,
        defaultQuantity: Double = 1.0,
        requiresColdChain: Bool = false,
        defaultLocation: MedicineLocation = .armadiettoBagno,
        defaultPaoDays: Int = 0,
        prescriptionRequired: Bool = false
    ) {
        self.aic = aic
        self.name = name
        self.activeIngredient = activeIngredient
        self.dosage = dosage
        self.form = form
        self.unit = unit
        self.defaultQuantity = defaultQuantity
        self.requiresColdChain = requiresColdChain
        self.defaultLocation = defaultLocation
        self.defaultPaoDays = defaultPaoDays
        self.prescriptionRequired = prescriptionRequired
    }
}

/// Alert e avvisi critici sui medicinali (scaduti, in scadenza, scorta bassa, catena del freddo)
public struct MedicineAlert: Identifiable, Codable, Sendable, Equatable {
    public var id: String { "\(medicineId)-\(type)" }
    public let medicineId: String
    public let medicineName: String
    public let type: String // "expired", "expiring_soon_7d", "expiring_soon_30d", "low_stock", "cold_chain_warning"
    public let severity: String // "critical", "warning", "info"
    public let daysRemaining: Int
    public let message: String
    public let location: MedicineLocation

    enum CodingKeys: String, CodingKey {
        case medicineId = "medicine_id"
        case medicineName = "medicine_name"
        case type
        case severity
        case daysRemaining = "days_remaining"
        case message
        case location
    }

    public init(
        medicineId: String,
        medicineName: String,
        type: String,
        severity: String,
        daysRemaining: Int,
        message: String,
        location: MedicineLocation
    ) {
        self.medicineId = medicineId
        self.medicineName = medicineName
        self.type = type
        self.severity = severity
        self.daysRemaining = daysRemaining
        self.message = message
        self.location = location
    }

    public var isCritical: Bool {
        return severity.lowercased() == "critical"
    }
}
