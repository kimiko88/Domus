import Foundation

public struct ExpenseDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let title: String
    public let amount: Double
    public let currency: String
    public let category: String
    public let subcategory: String
    public let expenseDate: String
    public let isFixed: Bool
    public let paymentMethod: String
    public let payerUserId: String?
    public let notes: String?
    public let contractExpiryDate: String?
    public let energyKwh: Double?
    public let gasSmc: Double?
    public let foodWasteValue: Double?
    public let createdAt: String?
    public let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case title
        case amount
        case currency
        case category
        case subcategory
        case expenseDate = "expense_date"
        case isFixed = "is_fixed"
        case paymentMethod = "payment_method"
        case payerUserId = "payer_user_id"
        case notes
        case contractExpiryDate = "contract_expiry_date"
        case energyKwh = "energy_kwh"
        case gasSmc = "gas_smc"
        case foodWasteValue = "food_waste_value"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    public init(
        id: String = UUID().uuidString,
        title: String,
        amount: Double,
        currency: String = "EUR",
        category: String = "other",
        subcategory: String = "",
        expenseDate: String = ISO8601DateFormatter().string(from: Date()),
        isFixed: Bool = false,
        paymentMethod: String = "card",
        payerUserId: String? = nil,
        notes: String? = nil,
        contractExpiryDate: String? = nil,
        energyKwh: Double? = nil,
        gasSmc: Double? = nil,
        foodWasteValue: Double? = nil,
        createdAt: String? = nil,
        updatedAt: String? = nil
    ) {
        self.id = id
        self.title = title
        self.amount = amount
        self.currency = currency
        self.category = category
        self.subcategory = subcategory
        self.expenseDate = expenseDate
        self.isFixed = isFixed
        self.paymentMethod = paymentMethod
        self.payerUserId = payerUserId
        self.notes = notes
        self.contractExpiryDate = contractExpiryDate
        self.energyKwh = energyKwh
        self.gasSmc = gasSmc
        self.foodWasteValue = foodWasteValue
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }
}

public struct CreateExpenseRequestDTO: Codable, Sendable {
    public let title: String
    public let amount: Double
    public let currency: String
    public let category: String
    public let subcategory: String
    public let expenseDate: String
    public let isFixed: Bool
    public let paymentMethod: String
    public let contractExpiryDate: String?
    public let energyKwh: Double?
    public let gasSmc: Double?
    public let foodWasteValue: Double?
    public let notes: String?

    enum CodingKeys: String, CodingKey {
        case title
        case amount
        case currency
        case category
        case subcategory
        case expenseDate = "expense_date"
        case isFixed = "is_fixed"
        case paymentMethod = "payment_method"
        case contractExpiryDate = "contract_expiry_date"
        case energyKwh = "energy_kwh"
        case gasSmc = "gas_smc"
        case foodWasteValue = "food_waste_value"
        case notes
    }

    public init(
        title: String,
        amount: Double,
        currency: String = "EUR",
        category: String = "other",
        subcategory: String = "",
        expenseDate: String = ISO8601DateFormatter().string(from: Date()),
        isFixed: Bool = false,
        paymentMethod: String = "card",
        contractExpiryDate: String? = nil,
        energyKwh: Double? = nil,
        gasSmc: Double? = nil,
        foodWasteValue: Double? = nil,
        notes: String? = nil
    ) {
        self.title = title
        self.amount = amount
        self.currency = currency
        self.category = category
        self.subcategory = subcategory
        self.expenseDate = expenseDate
        self.isFixed = isFixed
        self.paymentMethod = paymentMethod
        self.contractExpiryDate = contractExpiryDate
        self.energyKwh = energyKwh
        self.gasSmc = gasSmc
        self.foodWasteValue = foodWasteValue
        self.notes = notes
    }
}

public struct CategoryBreakdownDTO: Codable, Identifiable, Sendable {
    public var id: String { category }
    public let category: String
    public let name: String
    public let totalAmount: Double
    public let percentage: Double
    public let count: Int

    enum CodingKeys: String, CodingKey {
        case category
        case name
        case totalAmount = "total_amount"
        case percentage
        case count
    }

    public init(
        category: String,
        name: String = "",
        totalAmount: Double = 0.0,
        percentage: Double = 0.0,
        count: Int = 0
    ) {
        self.category = category
        self.name = name.isEmpty ? category.capitalized : name
        self.totalAmount = totalAmount
        self.percentage = percentage
        self.count = count
    }
}

public struct MonthlyTrendDTO: Codable, Identifiable, Sendable {
    public var id: String { yearMonth }
    public let yearMonth: String
    public let monthName: String
    public let totalAmount: Double
    public let utilitiesAmount: Double
    public let foodAmount: Double
    public let transportAmount: Double

    enum CodingKeys: String, CodingKey {
        case yearMonth = "year_month"
        case monthName = "month_name"
        case totalAmount = "total_amount"
        case utilitiesAmount = "utilities_amount"
        case foodAmount = "food_amount"
        case transportAmount = "transport_amount"
    }

    public init(
        yearMonth: String,
        monthName: String = "",
        totalAmount: Double = 0.0,
        utilitiesAmount: Double = 0.0,
        foodAmount: Double = 0.0,
        transportAmount: Double = 0.0
    ) {
        self.yearMonth = yearMonth
        self.monthName = monthName
        self.totalAmount = totalAmount
        self.utilitiesAmount = utilitiesAmount
        self.foodAmount = foodAmount
        self.transportAmount = transportAmount
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        self.yearMonth = try container.decode(String.self, forKey: .yearMonth)
        self.monthName = try container.decodeIfPresent(String.self, forKey: .monthName) ?? ""
        self.totalAmount = try container.decodeIfPresent(Double.self, forKey: .totalAmount) ?? 0.0
        self.utilitiesAmount = try container.decodeIfPresent(Double.self, forKey: .utilitiesAmount) ?? 0.0
        self.foodAmount = try container.decodeIfPresent(Double.self, forKey: .foodAmount) ?? 0.0
        self.transportAmount = try container.decodeIfPresent(Double.self, forKey: .transportAmount) ?? 0.0
    }
}

public struct FixedVsVariableDTO: Codable, Sendable {
    public let fixedAmount: Double
    public let variableAmount: Double
    public let fixedPercentage: Double
    public let variablePercentage: Double

    enum CodingKeys: String, CodingKey {
        case fixedAmount = "fixed_amount"
        case variableAmount = "variable_amount"
        case fixedPercentage = "fixed_percentage"
        case variablePercentage = "variable_percentage"
    }

    public init(
        fixedAmount: Double = 0.0,
        variableAmount: Double = 0.0,
        fixedPercentage: Double = 0.0,
        variablePercentage: Double = 0.0
    ) {
        self.fixedAmount = fixedAmount
        self.variableAmount = variableAmount
        self.fixedPercentage = fixedPercentage
        self.variablePercentage = variablePercentage
    }
}

public struct SavingsInsightDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let type: String
    public let title: String
    public let message: String
    public let potentialSaving: Double?
    public let severity: String
    public let actionUrl: String?

    enum CodingKeys: String, CodingKey {
        case id
        case type
        case title
        case message
        case potentialSaving = "potential_saving"
        case severity
        case actionUrl = "action_url"
    }

    public init(
        id: String = UUID().uuidString,
        type: String = "general",
        title: String,
        message: String,
        potentialSaving: Double? = nil,
        severity: String = "info",
        actionUrl: String? = nil
    ) {
        self.id = id
        self.type = type
        self.title = title
        self.message = message
        self.potentialSaving = potentialSaving
        self.severity = severity
        self.actionUrl = actionUrl
    }
}

public struct FinancialSummaryDTO: Codable, Sendable {
    public let period: String
    public let startDate: String
    public let endDate: String
    public let totalSpent: Double
    public let previousPeriodTotal: Double
    public let percentageChange: Double
    public let dailyAverage: Double
    public let monthlyBudget: Double
    public let remainingBudget: Double
    public let categories: [CategoryBreakdownDTO]
    public let monthlyTrends: [MonthlyTrendDTO]
    public let fixedVsVariable: FixedVsVariableDTO
    public let insights: [SavingsInsightDTO]

    enum CodingKeys: String, CodingKey {
        case period
        case startDate = "start_date"
        case endDate = "end_date"
        case totalSpent = "total_spent"
        case previousPeriodTotal = "previous_period_total"
        case percentageChange = "percentage_change"
        case dailyAverage = "daily_average"
        case monthlyBudget = "monthly_budget"
        case remainingBudget = "remaining_budget"
        case categories
        case monthlyTrends = "monthly_trends"
        case fixedVsVariable = "fixed_vs_variable"
        case insights
    }

    public init(
        period: String = "month",
        startDate: String = "",
        endDate: String = "",
        totalSpent: Double = 0.0,
        previousPeriodTotal: Double = 0.0,
        percentageChange: Double = 0.0,
        dailyAverage: Double = 0.0,
        monthlyBudget: Double = 0.0,
        remainingBudget: Double = 0.0,
        categories: [CategoryBreakdownDTO] = [],
        monthlyTrends: [MonthlyTrendDTO] = [],
        fixedVsVariable: FixedVsVariableDTO = FixedVsVariableDTO(),
        insights: [SavingsInsightDTO] = []
    ) {
        self.period = period
        self.startDate = startDate
        self.endDate = endDate
        self.totalSpent = totalSpent
        self.previousPeriodTotal = previousPeriodTotal
        self.percentageChange = percentageChange
        self.dailyAverage = dailyAverage
        self.monthlyBudget = monthlyBudget
        self.remainingBudget = remainingBudget
        self.categories = categories
        self.monthlyTrends = monthlyTrends
        self.fixedVsVariable = fixedVsVariable
        self.insights = insights
    }
}

public struct BudgetTargetDTO: Codable, Identifiable, Sendable {
    public let id: String
    public let category: String
    public let monthlyLimit: Double
    public let monthYear: String

    enum CodingKeys: String, CodingKey {
        case id
        case category
        case monthlyLimit = "monthly_limit"
        case monthYear = "month_year"
    }

    public init(
        id: String = UUID().uuidString,
        category: String = "total",
        monthlyLimit: Double = 0.0,
        monthYear: String = "global"
    ) {
        self.id = id
        self.category = category
        self.monthlyLimit = monthlyLimit
        self.monthYear = monthYear
    }
}

public struct SetBudgetRequestDTO: Codable, Sendable {
    public let category: String
    public let monthlyLimit: Double
    public let monthYear: String

    enum CodingKeys: String, CodingKey {
        case category
        case monthlyLimit = "monthly_limit"
        case monthYear = "month_year"
    }

    public init(
        category: String = "total",
        monthlyLimit: Double,
        monthYear: String = "global"
    ) {
        self.category = category
        self.monthlyLimit = monthlyLimit
        self.monthYear = monthYear
    }
}
