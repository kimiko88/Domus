package com.domus.app.unit

import com.domus.app.domain.model.*
import kotlinx.serialization.json.Json
import org.junit.Assert.*
import org.junit.Test

class FinanceModelTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testExpenseDtoSerializationAndDeserialization() {
        val expense = ExpenseDto(
            id = "exp-1",
            title = "Spesa Esselunga",
            amount = 84.50,
            category = "food",
            subcategory = "groceries",
            expense_date = "2026-10-17T12:00:00Z",
            is_fixed = false,
            energy_kwh = null,
            contract_expiry_date = null
        )

        val encoded = json.encodeToString(ExpenseDto.serializer(), expense)
        assertTrue(encoded.contains("Spesa Esselunga"))
        assertTrue(encoded.contains("84.5"))

        val decoded = json.decodeFromString(ExpenseDto.serializer(), encoded)
        assertEquals(expense.id, decoded.id)
        assertEquals(expense.amount, decoded.amount, 0.001)
        assertEquals(expense.category, decoded.category)
        assertFalse(decoded.is_fixed)
    }

    @Test
    fun testFinancialSummaryDtoCalculationAndDeserialization() {
        val summaryJson = """
            {
                "period": "month",
                "total_spent": 1420.50,
                "previous_period_total": 1600.00,
                "percentage_change": -11.22,
                "monthly_budget": 2000.00,
                "remaining_budget": 579.50,
                "daily_average": 47.35,
                "categories": [
                    {
                        "category": "utilities",
                        "name": "Bollette & Utenze",
                        "total_amount": 350.00,
                        "percentage": 24.64,
                        "count": 3
                    },
                    {
                        "category": "food",
                        "name": "Cibo & Alimentari",
                        "total_amount": 600.00,
                        "percentage": 42.24,
                        "count": 8
                    }
                ],
                "fixed_vs_variable": {
                    "fixed_amount": 450.00,
                    "variable_amount": 970.50,
                    "fixed_percentage": 31.68,
                    "variable_percentage": 68.32
                },
                "monthly_trends": [
                    {
                        "year_month": "2026-09",
                        "month_name": "Settembre 2026",
                        "total_amount": 1600.00
                    },
                    {
                        "year_month": "2026-10",
                        "month_name": "Ottobre 2026",
                        "total_amount": 1420.50
                    }
                ],
                "insights": [
                    {
                        "id": "ins-1",
                        "title": "Picco Luce Superiore alla Media",
                        "message": "Consumo elettrico superiore del 25% rispetto al mese scorso",
                        "severity": "warning",
                        "potential_saving": 45.00,
                        "type": "energy_spike"
                    }
                ]
            }
        """.trimIndent()

        val summary = json.decodeFromString(FinancialSummaryDto.serializer(), summaryJson)

        assertEquals("month", summary.period)
        assertEquals(1420.50, summary.total_spent, 0.001)
        assertEquals(2000.00, summary.monthly_budget, 0.001)
        assertEquals(579.50, summary.remaining_budget, 0.001)
        assertEquals(-11.22, summary.percentage_change, 0.001)
        assertEquals(2, summary.categories.size)
        assertEquals("utilities", summary.categories[0].category)
        assertEquals(450.00, summary.fixed_vs_variable.fixed_amount, 0.001)
        assertEquals(2, summary.monthly_trends.size)
        assertEquals(1, summary.insights.size)
        assertEquals(45.00, summary.insights[0].potential_saving ?: 0.0, 0.001)
        assertEquals("warning", summary.insights[0].severity)
    }

    @Test
    fun testBudgetTargetDto() {
        val req = SetBudgetRequestDto(monthly_limit = 2500.0)
        val encoded = json.encodeToString(SetBudgetRequestDto.serializer(), req)
        assertTrue(encoded.contains("2500"))

        val budget = BudgetTargetDto(
            id = "b-1",
            category = "all",
            monthly_limit = 2500.0,
            month_year = "global"
        )
        assertEquals(2500.0, budget.monthly_limit, 0.001)
        assertEquals("global", budget.month_year)
    }
}
