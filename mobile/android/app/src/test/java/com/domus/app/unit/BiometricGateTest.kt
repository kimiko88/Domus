package com.domus.app.unit

import com.domus.app.services.biometrics.BiometricAuthHelper
import org.junit.Assert.*
import org.junit.Test

class BiometricGateTest {

    @Test
    fun testSensitiveEntityIdentification() {
        // Critical entities requiring biometric gate
        assertTrue(BiometricAuthHelper.isSensitiveEntity("lock", "lock.front_door"))
        assertTrue(BiometricAuthHelper.isSensitiveEntity("alarm_control_panel", "alarm_control_panel.home"))
        assertTrue(BiometricAuthHelper.isSensitiveEntity("cover", "cover.main_gate"))
        assertTrue(BiometricAuthHelper.isSensitiveEntity("cover", "cover.garage_door"))
        assertTrue(BiometricAuthHelper.isSensitiveEntity("cover", "cover.portone_ingresso"))
        assertTrue(BiometricAuthHelper.isSensitiveEntity("cover", "cover.basculante_box"))

        // Non-critical entities
        assertFalse(BiometricAuthHelper.isSensitiveEntity("light", "light.salotto"))
        assertFalse(BiometricAuthHelper.isSensitiveEntity("switch", "switch.fan"))
        assertFalse(BiometricAuthHelper.isSensitiveEntity("climate", "climate.living_room"))
    }

    @Test
    fun testPromptMetadataGeneration() {
        val (disarmTitle, disarmSubtitle) = BiometricAuthHelper.createPromptTitle("disarm")
        assertTrue(disarmTitle.contains("Sicurezza"))
        assertTrue(disarmSubtitle.contains("disarmare"))

        val (lockTitle, lockSubtitle) = BiometricAuthHelper.createPromptTitle("lock")
        assertTrue(lockTitle.contains("Sblocco"))
        assertTrue(lockSubtitle.contains("cancello") || lockSubtitle.contains("portone"))
    }
}
