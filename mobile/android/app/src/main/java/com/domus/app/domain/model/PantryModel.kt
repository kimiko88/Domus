package com.domus.app.domain.model

import kotlinx.serialization.Serializable

@Serializable
data class PantryItemDto(
    val id: String = "",
    val name: String = "",
    val category: String = "altro",
    val location: String = "dispensa",
    val quantity: Double = 0.0,
    val unit: String = "pz",
    val min_threshold: Double = 0.0,
    val expiry_date: String? = null,
    val unit_price: Double = 0.0,
    val cost_currency: String = "EUR",
    val barcode: String? = null,
    val notes: String = "",
    val is_low_stock: Boolean = false,
    val is_expiring_soon: Boolean = false,
    val created_at: String = "",
    val updated_at: String = ""
)

@Serializable
data class CreatePantryItemRequestDto(
    val name: String,
    val category: String = "altro",
    val location: String = "dispensa",
    val quantity: Double = 1.0,
    val unit: String = "pz",
    val min_threshold: Double = 1.0,
    val expiry_date: String? = null,
    val unit_price: Double = 0.0,
    val cost_currency: String = "EUR",
    val barcode: String? = null,
    val notes: String = ""
)

@Serializable
data class UpdatePantryStockRequestDto(
    val delta: Double? = null,
    val quantity: Double? = null
)

@Serializable
data class RecipeIngredientDto(
    val id: String = "",
    val recipe_id: String = "",
    val pantry_item_id: String? = null,
    val item_name: String = "",
    val quantity: Double = 0.0,
    val unit: String = "g",
    val is_optional: Boolean = false
)

@Serializable
data class RecipeDto(
    val id: String = "",
    val title: String = "",
    val description: String = "",
    val category: String = "primo",
    val prep_time_min: Int = 10,
    val cook_time_min: Int = 15,
    val portions_base: Int = 2,
    val instructions: String = "",
    val is_favorite: Boolean = false,
    val ingredients: List<RecipeIngredientDto> = emptyList(),
    val created_at: String = "",
    val updated_at: String = ""
)

@Serializable
data class CreateRecipeRequestDto(
    val title: String,
    val description: String = "",
    val category: String = "primo",
    val prep_time_min: Int = 10,
    val cook_time_min: Int = 15,
    val portions_base: Int = 2,
    val instructions: String = "",
    val is_favorite: Boolean = false,
    val ingredients: List<RecipeIngredientDto> = emptyList()
)

@Serializable
data class MealPlanDto(
    val id: String = "",
    val date: String = "",
    val meal_type: String = "lunch",
    val recipe_id: String? = null,
    val recipe_title: String = "",
    val custom_notes: String = "",
    val persons_count: Int = 2,
    val is_consumed: Boolean = false,
    val consumed_at: String? = null,
    val created_at: String = "",
    val updated_at: String = ""
)

@Serializable
data class CreateMealPlanRequestDto(
    val date: String,
    val meal_type: String = "lunch",
    val recipe_id: String? = null,
    val custom_notes: String = "",
    val persons_count: Int = 2
)

@Serializable
data class ConsumeMealPlanRequestDto(
    val persons_count: Int? = null
)

@Serializable
data class ConsumedItemSummaryDto(
    val pantry_item_id: String = "",
    val name: String = "",
    val scaled_quantity: Double = 0.0,
    val unit: String = "",
    val remaining_quantity: Double = 0.0,
    val is_low_stock: Boolean = false
)

@Serializable
data class ConsumeMealPlanResponseDto(
    val success: Boolean = true,
    val meal_plan_id: String = "",
    val persons_count: Int = 2,
    val consumed_items: List<ConsumedItemSummaryDto> = emptyList(),
    val low_stock_alerts: List<String> = emptyList(),
    val message: String = ""
)

@Serializable
data class ReceiptItemDto(
    val raw_line: String = "",
    val item_name: String = "",
    val quantity: Double = 1.0,
    val unit: String = "pz",
    val unit_price: Double = 0.0,
    val total_price: Double = 0.0,
    val matched_pantry_id: String? = null,
    val suggested_category: String = "altro"
)

@Serializable
data class ReceiptDto(
    val store_name: String = "",
    val date: String = "",
    val total_amount: Double = 0.0,
    val currency: String = "EUR",
    val items: List<ReceiptItemDto> = emptyList(),
    val raw_text: String = ""
)

@Serializable
data class ReceiptParseRequestDto(
    val raw_text: String
)

@Serializable
data class ReceiptImportRequestDto(
    val receipt: ReceiptDto,
    val auto_add_to_finance: Boolean = true,
    val auto_update_pantry: Boolean = true
)

@Serializable
data class ReceiptImportResultDto(
    val imported_items_count: Int = 0,
    val pantry_items_updated: List<PantryItemDto> = emptyList(),
    val expense_id: String? = null,
    val total_amount: Double = 0.0,
    val message: String = ""
)
