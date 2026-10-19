import XCTest
@testable import DomusKit

final class FinanceTests: XCTestCase {

    func testExpenseDTOCoding() throws {
        let expense = ExpenseDTO(
            id: "exp-1",
            title: "Spesa Supermercato",
            amount: 78.50,
            currency: "EUR",
            category: "food",
            subcategory: "groceries",
            expenseDate: "2026-10-17T11:00:00Z",
            isFixed: false,
            paymentMethod: "card",
            notes: "Spesa settimanale",
            energyKwh: nil
        )

        let data = try JSONEncoder().encode(expense)
        let decoded = try JSONDecoder().decode(ExpenseDTO.self, from: data)

        XCTAssertEqual(decoded.id, "exp-1")
        XCTAssertEqual(decoded.title, "Spesa Supermercato")
        XCTAssertEqual(decoded.amount, 78.50, accuracy: 0.001)
        XCTAssertEqual(decoded.category, "food")
        XCTAssertFalse(decoded.isFixed)

        // Request DTO
        let req = CreateExpenseRequestDTO(
            title: "Bolletta Luce",
            amount: 110.0,
            category: "utilities",
            subcategory: "electricity",
            isFixed: true,
            energyKwh: 280.0
        )
        let reqData = try JSONEncoder().encode(req)
        let reqDecoded = try JSONDecoder().decode(CreateExpenseRequestDTO.self, from: reqData)
        XCTAssertEqual(reqDecoded.title, "Bolletta Luce")
        XCTAssertEqual(reqDecoded.amount, 110.0, accuracy: 0.001)
        XCTAssertTrue(reqDecoded.isFixed)
        XCTAssertEqual(reqDecoded.energyKwh, 280.0)
    }

    func testFinancialSummaryDTODecoding() throws {
        let jsonString = """
        {
            "period": "month",
            "start_date": "2026-10-01",
            "end_date": "2026-10-31",
            "total_spent": 1250.75,
            "previous_period_total": 1400.00,
            "percentage_change": -10.66,
            "daily_average": 40.35,
            "monthly_budget": 1800.00,
            "remaining_budget": 549.25,
            "categories": [
                {
                    "category": "utilities",
                    "name": "Bollette & Utenze",
                    "total_amount": 320.00,
                    "percentage": 25.58,
                    "count": 2
                },
                {
                    "category": "food",
                    "name": "Cibo & Alimentari",
                    "total_amount": 550.00,
                    "percentage": 43.97,
                    "count": 6
                }
            ],
            "monthly_trends": [
                {
                    "year_month": "2026-09",
                    "month_name": "Settembre 2026",
                    "total_amount": 1400.00
                },
                {
                    "year_month": "2026-10",
                    "month_name": "Ottobre 2026",
                    "total_amount": 1250.75
                }
            ],
            "fixed_vs_variable": {
                "fixed_amount": 420.00,
                "variable_amount": 830.75,
                "fixed_percentage": 33.58,
                "variable_percentage": 66.42
            },
            "insights": [
                {
                    "id": "ins-1",
                    "type": "tariff_optimization",
                    "title": "Ottimizzazione Tariffa Energia",
                    "message": "Spostando i carichi pesanti in fascia notturna risparmi circa 35€ al mese",
                    "potential_saving": 35.00,
                    "severity": "info"
                }
            ]
        }
        """

        let data = jsonString.data(using: .utf8)!
        let summary = try JSONDecoder().decode(FinancialSummaryDTO.self, from: data)

        XCTAssertEqual(summary.period, "month")
        XCTAssertEqual(summary.totalSpent, 1250.75, accuracy: 0.001)
        XCTAssertEqual(summary.monthlyBudget, 1800.00, accuracy: 0.001)
        XCTAssertEqual(summary.remainingBudget, 549.25, accuracy: 0.001)
        XCTAssertEqual(summary.percentageChange, -10.66, accuracy: 0.001)
        XCTAssertEqual(summary.categories.count, 2)
        XCTAssertEqual(summary.categories[0].category, "utilities")
        XCTAssertEqual(summary.fixedVsVariable.fixedAmount, 420.00, accuracy: 0.001)
        XCTAssertEqual(summary.monthlyTrends.count, 2)
        XCTAssertEqual(summary.insights.count, 1)
        XCTAssertEqual(summary.insights[0].potentialSaving, 35.00)
        XCTAssertEqual(summary.insights[0].severity, "info")
    }

    func testBudgetTargetDTO() throws {
        let budget = BudgetTargetDTO(
            id: "b-100",
            category: "total",
            monthlyLimit: 2200.0,
            monthYear: "global"
        )
        let data = try JSONEncoder().encode(budget)
        let decoded = try JSONDecoder().decode(BudgetTargetDTO.self, from: data)

        XCTAssertEqual(decoded.id, "b-100")
        XCTAssertEqual(decoded.monthlyLimit, 2200.0, accuracy: 0.001)
        XCTAssertEqual(decoded.category, "total")

        let req = SetBudgetRequestDTO(monthlyLimit: 2500.0)
        let reqData = try JSONEncoder().encode(req)
        let reqDecoded = try JSONDecoder().decode(SetBudgetRequestDTO.self, from: reqData)
        XCTAssertEqual(reqDecoded.monthlyLimit, 2500.0, accuracy: 0.001)
    }
}
