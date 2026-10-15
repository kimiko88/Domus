package com.domus.app.services.auto

/**
 * Driver-safe action items for Android Auto (androidx.car.app GridTemplate)
 * strictly adhering to Google Driver Distraction Guidelines (DDG).
 */
data class CarActionItem(
    val id: String,
    val title: String,
    val iconEmoji: String,
    val category: String
)

object DomusCarActions {
    val quickActions = listOf(
        CarActionItem("car_gate", "Apri Cancello", "🚪", "Accesso"),
        CarActionItem("car_garage", "Apri Garage", "🚗", "Accesso"),
        CarActionItem("car_disarm", "Disarma Perimetrale", "🛡️", "Sicurezza"),
        CarActionItem("car_lights", "Luci Esterne Vialetto", "💡", "Illuminazione")
    )
}
