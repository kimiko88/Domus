package com.domus.app.domain.model

import kotlinx.serialization.Serializable

@Serializable
data class ExpenseDto(
    val id: String = "",
    val title: String = "",
    val amount: Double = 0.0,
    val currency: String = "EUR",
    val category: String = "other",
    val subcategory: String = "",
    val expense_date: String = "",
    val is_fixed: Boolean = false,
    val payment_method: String = "card",
    val payer_user_id: String = "",
    val notes: String = "",
    val contract_expiry_date: String? = null,
    val energy_kwh: Double? = null,
    val gas_smc: Double? = null,
    val food_waste_value: Double? = null,
    val created_at: String = "",
    val updated_at: String = ""
)

@Serializable
data class CreateExpenseRequestDto(
    val title: String,
    val amount: Double,
    val currency: String = "EUR",
    val category: String = "other",
    val subcategory: String = "",
    val expense_date: String,
    val is_fixed: Boolean = false,
    val payment_method: String = "card",
    val contract_expiry_date: String? = null,
    val energy_kwh: Double? = null,
    val gas_smc: Double? = null,
    val food_waste_value: Double? = null,
    val notes: String? = null
)

@Serializable
data class CategoryBreakdownDto(
    val category: String = "",
    val name: String = "",
    val total_amount: Double = 0.0,
    val percentage: Double = 0.0,
    val count: Int = 0
)

@Serializable
data class MonthlyTrendDto(
    val year_month: String = "",
    val month_name: String = "",
    val total_amount: Double = 0.0,
    val utilities_amount: Double = 0.0,
    val food_amount: Double = 0.0,
    val transport_amount: Double = 0.0
)

@Serializable
data class FixedVsVariableDto(
    val fixed_amount: Double = 0.0,
    val variable_amount: Double = 0.0,
    val fixed_percentage: Double = 0.0,
    val variable_percentage: Double = 0.0
)

@Serializable
data class SavingsInsightDto(
    val id: String = "",
    val type: String = "",
    val title: String = "",
    val message: String = "",
    val potential_saving: Double? = null,
    val severity: String = "info",
    val action_url: String? = null
)

@Serializable
data class FinancialSummaryDto(
    val period: String = "month",
    val start_date: String = "",
    val end_date: String = "",
    val total_spent: Double = 0.0,
    val previous_period_total: Double = 0.0,
    val percentage_change: Double = 0.0,
    val daily_average: Double = 0.0,
    val monthly_budget: Double = 0.0,
    val remaining_budget: Double = 0.0,
    val categories: List<CategoryBreakdownDto> = emptyList(),
    val monthly_trends: List<MonthlyTrendDto> = emptyList(),
    val fixed_vs_variable: FixedVsVariableDto = FixedVsVariableDto(),
    val insights: List<SavingsInsightDto> = emptyList()
)

@Serializable
data class BudgetTargetDto(
    val id: String = "",
    val category: String = "total",
    val monthly_limit: Double = 0.0,
    val month_year: String = "global"
)

@Serializable
data class SetBudgetRequestDto(
    val category: String = "total",
    val monthly_limit: Double,
    val month_year: String = "global"
)
