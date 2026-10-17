import Foundation

/// Modello articolo presente in dispensa o frigo
public struct PantryItem: Identifiable, Codable, Sendable, Equatable {
    public let id: String
    public var name: String
    public var category: String
    public var location: String
    public var quantity: Double
    public var unit: String
    public var minThreshold: Double
    public var expiryDate: String?
    public var unitPrice: Double
    public var costCurrency: String
    public var barcode: String?
    public var notes: String
    public var isLowStock: BooleanLiteralType
    public var isExpiringSoon: BooleanLiteralType
    public var createdAt: String
    public var updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case category
        case location
        case quantity
        case unit
        case minThreshold = "min_threshold"
        case expiryDate = "expiry_date"
        case unitPrice = "unit_price"
        case costCurrency = "cost_currency"
        case barcode
        case notes
        case isLowStock = "is_low_stock"
        case isExpiringSoon = "is_expiring_soon"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    public init(
        id: String = UUID().uuidString,
        name: String,
        category: String = "dispensa",
        location: String = "dispensa",
        quantity: Double = 1.0,
        unit: String = "pz",
        minThreshold: Double = 1.0,
        expiryDate: String? = nil,
        unitPrice: Double = 0.0,
        costCurrency: String = "EUR",
        barcode: String? = nil,
        notes: String = "",
        isLowStock: Bool = false,
        isExpiringSoon: Bool = false,
        createdAt: String = ISO8601DateFormatter().string(from: Date()),
        updatedAt: String = ISO8601DateFormatter().string(from: Date())
    ) {
        self.id = id
        self.name = name
        self.category = category
        self.location = location
        self.quantity = quantity
        self.unit = unit
        self.minThreshold = minThreshold
        self.expiryDate = expiryDate
        self.unitPrice = unitPrice
        self.costCurrency = costCurrency
        self.barcode = barcode
        self.notes = notes
        self.isLowStock = isLowStock
        self.isExpiringSoon = isExpiringSoon
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }
}

/// Ingrediente associato a una ricetta
public struct RecipeIngredient: Identifiable, Codable, Sendable, Equatable {
    public let id: String
    public var recipeId: String
    public var pantryItemId: String?
    public var itemName: String
    public var quantity: Double
    public var unit: String
    public var isOptional: Bool

    enum CodingKeys: String, CodingKey {
        case id
        case recipeId = "recipe_id"
        case pantryItemId = "pantry_item_id"
        case itemName = "item_name"
        case quantity
        case unit
        case isOptional = "is_optional"
    }

    public init(
        id: String = UUID().uuidString,
        recipeId: String = "",
        pantryItemId: String? = nil,
        itemName: String,
        quantity: Double = 1.0,
        unit: String = "g",
        isOptional: Bool = false
    ) {
        self.id = id
        self.recipeId = recipeId
        self.pantryItemId = pantryItemId
        self.itemName = itemName
        self.quantity = quantity
        self.unit = unit
        self.isOptional = isOptional
    }
}

/// Ricetta salvata nel ricettario
public struct Recipe: Identifiable, Codable, Sendable, Equatable {
    public let id: String
    public var title: String
    public var description: String
    public var category: String
    public var prepTimeMin: Int
    public var cookTimeMin: Int
    public var portionsBase: Int
    public var instructions: String
    public var isFavorite: Bool
    public var ingredients: [RecipeIngredient]
    public var createdAt: String
    public var updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case title
        case description
        case category
        case prepTimeMin = "prep_time_min"
        case cookTimeMin = "cook_time_min"
        case portionsBase = "portions_base"
        case instructions
        case isFavorite = "is_favorite"
        case ingredients
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    public init(
        id: String = UUID().uuidString,
        title: String,
        description: String = "",
        category: String = "primo",
        prepTimeMin: Int = 10,
        cookTimeMin: Int = 15,
        portionsBase: Int = 2,
        instructions: String = "",
        isFavorite: Bool = false,
        ingredients: [RecipeIngredient] = [],
        createdAt: String = ISO8601DateFormatter().string(from: Date()),
        updatedAt: String = ISO8601DateFormatter().string(from: Date())
    ) {
        self.id = id
        self.title = title
        self.description = description
        self.category = category
        self.prepTimeMin = prepTimeMin
        self.cookTimeMin = cookTimeMin
        self.portionsBase = portionsBase
        self.instructions = instructions
        self.isFavorite = isFavorite
        self.ingredients = ingredients
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }
}

/// Pianificazione pasto nel calendario
public struct MealPlan: Identifiable, Codable, Sendable, Equatable {
    public let id: String
    public var date: String
    public var mealType: String
    public var recipeId: String?
    public var recipeTitle: String
    public var customNotes: String
    public var personsCount: Int
    public var isConsumed: Bool
    public var consumedAt: String?
    public var createdAt: String
    public var updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case date
        case mealType = "meal_type"
        case recipeId = "recipe_id"
        case recipeTitle = "recipe_title"
        case customNotes = "custom_notes"
        case personsCount = "persons_count"
        case isConsumed = "is_consumed"
        case consumedAt = "consumed_at"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    public init(
        id: String = UUID().uuidString,
        date: String,
        mealType: String = "lunch",
        recipeId: String? = nil,
        recipeTitle: String = "",
        customNotes: String = "",
        personsCount: Int = 2,
        isConsumed: Bool = false,
        consumedAt: String? = nil,
        createdAt: String = ISO8601DateFormatter().string(from: Date()),
        updatedAt: String = ISO8601DateFormatter().string(from: Date())
    ) {
        self.id = id
        self.date = date
        self.mealType = mealType
        self.recipeId = recipeId
        self.recipeTitle = recipeTitle
        self.customNotes = customNotes
        self.personsCount = personsCount
        self.isConsumed = isConsumed
        self.consumedAt = consumedAt
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }
}

/// Articolo estratto dallo scontrino
public struct ReceiptItem: Identifiable, Codable, Sendable, Equatable {
    public var id: String { rawLine }
    public var rawLine: String
    public var itemName: String
    public var quantity: Double
    public var unit: String
    public var unitPrice: Double
    public var totalPrice: Double
    public var matchedPantryId: String?
    public var suggestedCategory: String

    enum CodingKeys: String, CodingKey {
        case rawLine = "raw_line"
        case itemName = "item_name"
        case quantity
        case unit
        case unitPrice = "unit_price"
        case totalPrice = "total_price"
        case matchedPantryId = "matched_pantry_id"
        case suggestedCategory = "suggested_category"
    }

    public init(
        rawLine: String = "",
        itemName: String,
        quantity: Double = 1.0,
        unit: String = "pz",
        unitPrice: Double = 0.0,
        totalPrice: Double = 0.0,
        matchedPantryId: String? = nil,
        suggestedCategory: String = "altro"
    ) {
        self.rawLine = rawLine.isEmpty ? "\(itemName)_\(totalPrice)" : rawLine
        self.itemName = itemName
        self.quantity = quantity
        self.unit = unit
        self.unitPrice = unitPrice
        self.totalPrice = totalPrice
        self.matchedPantryId = matchedPantryId
        self.suggestedCategory = suggestedCategory
    }
}

/// Scontrino analizzato
public struct Receipt: Codable, Sendable, Equatable {
    public var storeName: String
    public var date: String
    public var totalAmount: Double
    public var currency: String
    public var items: [ReceiptItem]
    public var rawText: String

    enum CodingKeys: String, CodingKey {
        case storeName = "store_name"
        case date
        case totalAmount = "total_amount"
        case currency
        case items
        case rawText = "raw_text"
    }

    public init(
        storeName: String = "",
        date: String = "",
        totalAmount: Double = 0.0,
        currency: String = "EUR",
        items: [ReceiptItem] = [],
        rawText: String = ""
    ) {
        self.storeName = storeName
        self.date = date
        self.totalAmount = totalAmount
        self.currency = currency
        self.items = items
        self.rawText = rawText
    }
}

/// Risultato importazione scontrino
public struct ReceiptImportResult: Codable, Sendable, Equatable {
    public var importedItemsCount: Int
    public var pantryItemsUpdated: [PantryItem]
    public var expenseId: String?
    public var totalAmount: Double
    public var message: String

    enum CodingKeys: String, CodingKey {
        case importedItemsCount = "imported_items_count"
        case pantryItemsUpdated = "pantry_items_updated"
        case expenseId = "expense_id"
        case totalAmount = "total_amount"
        case message
    }
}
