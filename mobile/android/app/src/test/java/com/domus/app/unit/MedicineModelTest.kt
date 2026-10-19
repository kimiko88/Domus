package com.domus.app.unit

import com.domus.app.domain.model.*
import kotlinx.serialization.json.Json
import org.junit.Assert.*
import org.junit.Test

class MedicineModelTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testMedicineDtoSerializationAndLogic() {
        val med = MedicineDto(
            id = "med-test-1",
            name = "Paracetamolo 500mg",
            active_ingredient = "Paracetamolo",
            dosage = "500mg",
            form = "compresse",
            aic = "012345678",
            barcode = "800000000001",
            lot_number = "LOT2026",
            expiration_date = "2027-12-31",
            pao_days = 0,
            opened_at = null,
            effective_expiration_date = "2027-12-31",
            location = "armadietto_bagno",
            requires_cold_chain = false,
            quantity = 16.0,
            unit = "compresse",
            min_threshold = 4.0,
            prescription_required = false
        )

        assertFalse(med.isOpened)
        assertFalse(med.isLowStock)

        val encoded = json.encodeToString(MedicineDto.serializer(), med)
        assertTrue(encoded.contains("Paracetamolo 500mg"))
        assertTrue(encoded.contains("armadietto_bagno"))

        val decoded = json.decodeFromString(MedicineDto.serializer(), encoded)
        assertEquals(med.id, decoded.id)
        assertEquals(med.name, decoded.name)
        assertEquals(16.0, decoded.quantity, 0.001)
        assertEquals(4.0, decoded.min_threshold, 0.001)
        assertFalse(decoded.isOpened)
        assertFalse(decoded.isLowStock)
    }

    @Test
    fun testMedicinePAOAndLowStockFlags() {
        val openedMed = MedicineDto(
            id = "med-test-2",
            name = "Collirio Antibiotico",
            pao_days = 28,
            opened_at = "2026-10-19",
            effective_expiration_date = "2026-11-16",
            quantity = 1.0,
            min_threshold = 2.0
        )

        assertTrue(openedMed.isOpened)
        assertTrue(openedMed.isLowStock)
        assertEquals(28, openedMed.pao_days)
        assertEquals("2026-11-16", openedMed.effective_expiration_date)
    }

    @Test
    fun testMedicineAlertDto() {
        val alert = MedicineAlertDto(
            medicine_id = "med-99",
            medicine_name = "Insulina Glargine",
            type = "cold_chain_warning",
            severity = "critical",
            days_remaining = 45,
            message = "Temperatura frigo superiore a 8°C",
            location = "frigo"
        )

        assertTrue(alert.isCritical)
        assertEquals("cold_chain_warning", alert.type)

        val encoded = json.encodeToString(MedicineAlertDto.serializer(), alert)
        val decoded = json.decodeFromString(MedicineAlertDto.serializer(), encoded)
        assertEquals(alert.medicine_id, decoded.medicine_id)
        assertTrue(decoded.isCritical)
    }

    @Test
    fun testDatamatrixAndAIFARecordParsing() {
        val dm = DatamatrixParseResultDto(
            aic = "029028014",
            gtin = "08001234567890",
            expiration_date = "2027-05-31",
            lot_number = "LOT2026B",
            rawCode = "01080012345678901727053110LOT2026B"
        )
        assertEquals("029028014", dm.aic)
        assertEquals("2027-05-31", dm.expiration_date)

        val aifa = AIFARecordDto(
            aic = "029028014",
            name = "Augmentin",
            active_ingredient = "Amoxicillina + Clavulanato",
            dosage = "1g",
            form = "compresse",
            unit = "compresse",
            default_quantity = 12.0,
            requires_cold_chain = false,
            default_location = "armadietto_bagno",
            default_pao_days = 0,
            prescription_required = true
        )
        assertEquals("Augmentin", aifa.name)
        assertTrue(aifa.prescription_required)
    }

    @Test
    fun testEnumsFromValue() {
        assertEquals(MedicineLocation.ARMADIETTO_BAGNO, MedicineLocation.fromValue("armadietto_bagno"))
        assertEquals(MedicineLocation.FRIGO, MedicineLocation.fromValue("frigo"))
        assertEquals(MedicineLocation.ARMADIETTO_BAGNO, MedicineLocation.fromValue("sconosciuto"))

        assertEquals(MedicineUnit.COMPRESSE, MedicineUnit.fromValue("compresse"))
        assertEquals(MedicineUnit.ML, MedicineUnit.fromValue("ml"))
        assertEquals(MedicineUnit.COMPRESSE, MedicineUnit.fromValue("invalid"))
    }
}
