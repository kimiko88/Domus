package com.domus.app.integration

import com.domus.app.data.network.DomusApiClient
import com.domus.app.domain.model.CreateExpenseRequestDto
import com.domus.app.domain.model.SetBudgetRequestDto
import kotlinx.coroutines.test.runTest
import okhttp3.mockwebserver.Dispatcher
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.RecordedRequest
import org.junit.After
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test

class FinanceIntegrationTest {

    private lateinit var server: MockWebServer
    private lateinit var apiClient: DomusApiClient

    @Before
    fun setup() {
        server = MockWebServer()
        apiClient = DomusApiClient()
    }

    @After
    fun tearDown() {
        try {
            server.shutdown()
        } catch (_: Exception) {}
    }

    @Test
    fun testFinancialSummaryAndExpensesApiIntegration() = runTest {
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                val path = request.path ?: ""
                return when {
                    path.startsWith("/api/v1/finance/summary") -> {
                        MockResponse().setResponseCode(200).setBody("""
                            {
                                "period": "month",
                                "total_spent": 850.00,
                                "previous_total": 900.00,
                                "percentage_change": -5.55,
                                "monthly_budget": 1500.00,
                                "remaining_budget": 650.00,
                                "daily_average": 28.33,
                                "categories": [
                                    {"category": "food", "total": 500.0, "percentage": 58.82, "count": 5},
                                    {"category": "utilities", "total": 350.0, "percentage": 41.18, "count": 2}
                                ],
                                "fixed_vs_variable": {
                                    "fixed_total": 350.0,
                                    "variable_total": 500.0,
                                    "fixed_percentage": 41.18,
                                    "variable_percentage": 58.82
                                },
                                "monthly_trend": [],
                                "insights": [
                                    {
                                        "id": "ins-1",
                                        "title": "Bolletta Gas Ottimizzabile",
                                        "description": "Puoi risparmiare 30€ passando alla tariffa bioraria",
                                        "severity": "info",
                                        "potential_savings": 30.0,
                                        "category": "utilities"
                                    }
                                ]
                            }
                        """.trimIndent())
                    }
                    path.startsWith("/api/v1/finance/expenses") && request.method == "GET" -> {
                        MockResponse().setResponseCode(200).setBody("""
                            [
                                {
                                    "id": "exp-101",
                                    "title": "Bolletta Enel",
                                    "amount": 120.00,
                                    "category": "utilities",
                                    "subcategory": "electricity",
                                    "expense_date": "2026-10-15T09:00:00Z",
                                    "is_fixed": true,
                                    "energy_kwh": 310.5
                                }
                            ]
                        """.trimIndent())
                    }
                    path.startsWith("/api/v1/finance/expenses") && request.method == "POST" -> {
                        MockResponse().setResponseCode(201).setBody("""
                            {
                                "id": "exp-102",
                                "title": "Benzina",
                                "amount": 60.00,
                                "category": "transport",
                                "subcategory": "fuel",
                                "expense_date": "2026-10-17T18:00:00Z",
                                "is_fixed": false
                            }
                        """.trimIndent())
                    }
                    path.startsWith("/api/v1/finance/budgets") && request.method == "POST" -> {
                        MockResponse().setResponseCode(200).setBody("""
                            {
                                "id": "b-1",
                                "category": "all",
                                "monthly_limit": 1800.00,
                                "currency": "EUR"
                            }
                        """.trimIndent())
                    }
                    else -> MockResponse().setResponseCode(404)
                }
            }
        }

        val baseUrl = server.url("/").toString().removeSuffix("/")

        // 1. Fetch Summary
        val summaryResult = apiClient.fetchFinancialSummary(baseUrl, "test-token", "month")
        assertTrue(summaryResult.isSuccess)
        val summary = summaryResult.getOrNull()
        assertNotNull(summary)
        assertEquals(850.00, summary!!.total_spent, 0.001)
        assertEquals(650.00, summary.remaining_budget, 0.001)
        assertEquals(1, summary.insights.size)
        assertEquals("Bolletta Gas Ottimizzabile", summary.insights[0].title)

        // 2. Fetch Expenses List
        val expensesResult = apiClient.fetchExpenses(baseUrl, "test-token")
        assertTrue(expensesResult.isSuccess)
        val expenses = expensesResult.getOrNull()
        assertNotNull(expenses)
        assertEquals(1, expenses!!.size)
        assertEquals("Bolletta Enel", expenses[0].title)
        assertTrue(expenses[0].is_fixed)
        assertEquals(310.5, expenses[0].energy_kwh ?: 0.0, 0.001)

        // 3. Create Expense
        val createResult = apiClient.createExpense(
            baseUrl,
            "test-token",
            CreateExpenseRequestDto(
                title = "Benzina",
                amount = 60.0,
                category = "transport",
                subcategory = "fuel",
                expense_date = "2026-10-17T18:00:00Z",
                is_fixed = false
            )
        )
        assertTrue(createResult.isSuccess)

        // 4. Set Budget
        val budgetResult = apiClient.setBudget(
            baseUrl,
            "test-token",
            SetBudgetRequestDto(monthly_limit = 1800.0)
        )
        assertTrue(budgetResult.isSuccess)
    }
}
