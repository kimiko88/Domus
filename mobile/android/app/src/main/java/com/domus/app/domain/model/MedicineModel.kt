package com.domus.app.domain.model

import kotlinx.serialization.Serializable

@Serializable
enum class MedicineLocation(val value: String) {
    ARMADIETTO_BAGNO("armadietto_bagno"),
    CASSETTO_NOTTE("cassetto_notte"),
    BORSA_VIAGGIO("borsa_viaggio"),
    FRIGO("frigo"),
    ALTRO("altro");

    companion object {
        fun fromValue(v: String?): MedicineLocation {
            return entries.firstOrNull { it.value.equals(v, ignoreCase = true) } ?: ARMADIETTO_BAGNO
        }
    }
}

@Serializable
enum class MedicineUnit(val value: String) {
    COMPRESSE("compresse"),
    BUSTINE("bustine"),
    GOCCE("gocce"),
    ML("ml"),
    PZ("pz"),
    CONFEZIONE("confezione");

    companion object {
        fun fromValue(v: String?): MedicineUnit {
            return entries.firstOrNull { it.value.equals(v, ignoreCase = true) } ?: COMPRESSE
        }
    }
}

@Serializable
data class MedicineDto(
    val id: String = "",
    val name: String = "",
    val active_ingredient: String? = null,
    val dosage: String? = null,
    val form: String? = null,
    val aic: String? = null,
    val barcode: String? = null,
    val lot_number: String? = null,
    val expiration_date: String = "",
    val pao_days: Int = 0,
    val opened_at: String? = null,
    val effective_expiration_date: String = "",
    val location: String = "armadietto_bagno",
    val requires_cold_chain: Boolean = false,
    val quantity: Double = 1.0,
    val unit: String = "compresse",
    val min_threshold: Double = 1.0,
    val prescription_required: Boolean = false,
    val notes: String? = null,
    val created_at: String = "",
    val updated_at: String = ""
) {
    val isOpened: Boolean
        get() = !opened_at.isNullOrBlank()

    val isLowStock: Boolean
        get() = quantity <= min_threshold
}

@Serializable
data class CreateMedicineRequestDto(
    val name: String,
    val active_ingredient: String? = null,
    val dosage: String? = null,
    val form: String? = null,
    val aic: String? = null,
    val barcode: String? = null,
    val lot_number: String? = null,
    val expiration_date: String,
    val pao_days: Int = 0,
    val location: String = "armadietto_bagno",
    val requires_cold_chain: Boolean = false,
    val quantity: Double = 1.0,
    val unit: String = "compresse",
    val min_threshold: Double = 1.0,
    val prescription_required: Boolean = false,
    val notes: String? = null
)

@Serializable
data class UpdateMedicineRequestDto(
    val name: String? = null,
    val active_ingredient: String? = null,
    val dosage: String? = null,
    val expiration_date: String? = null,
    val pao_days: Int? = null,
    val location: String? = null,
    val requires_cold_chain: Boolean? = null,
    val quantity: Double? = null,
    val unit: String? = null,
    val min_threshold: Double? = null,
    val notes: String? = null
)

@Serializable
data class OpenMedicineRequestDto(
    val opened_at: String? = null
)

@Serializable
data class ConsumeMedicineRequestDto(
    val amount: Double = 1.0
)

@Serializable
data class DatamatrixParseResultDto(
    val aic: String? = null,
    val gtin: String? = null,
    val expiration_date: String? = null,
    val lot_number: String? = null,
    val serial_number: String? = null,
    val raw_code: String = ""
)

@Serializable
data class AIFARecordDto(
    val aic: String = "",
    val name: String = "",
    val active_ingredient: String = "",
    val dosage: String = "",
    val form: String = "",
    val unit: String = "compresse",
    val default_quantity: Double = 1.0,
    val requires_cold_chain: Boolean = false,
    val default_location: String = "armadietto_bagno",
    val default_pao_days: Int = 0,
    val prescription_required: Boolean = false
)

@Serializable
data class MedicineAlertDto(
    val medicine_id: String = "",
    val medicine_name: String = "",
    val type: String = "expiring_soon_30d",
    val severity: String = "info",
    val days_remaining: Int = 0,
    val message: String = "",
    val location: String = "armadietto_bagno"
) {
    val isCritical: Boolean
        get() = severity.equals("critical", ignoreCase = true)
}
