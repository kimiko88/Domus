package com.domus.app.unit

import com.domus.app.util.LocaleHelper
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class LocaleHelperTest {
    @Test
    fun testSupportedLanguagesCountAndCodes() {
        val langs = LocaleHelper.supportedLanguages
        assertEquals(5, langs.size)
        val codes = langs.map { it.code }
        assertTrue(codes.contains("it"))
        assertTrue(codes.contains("en"))
        assertTrue(codes.contains("fr"))
        assertTrue(codes.contains("es"))
        assertTrue(codes.contains("de"))
    }

    @Test
    fun testDisplayNamesAndFlags() {
        val langs = LocaleHelper.supportedLanguages.associateBy { it.code }
        assertEquals("Italiano", langs["it"]?.displayName)
        assertEquals("English", langs["en"]?.displayName)
        assertEquals("Français", langs["fr"]?.displayName)
        assertEquals("Español", langs["es"]?.displayName)
        assertEquals("Deutsch", langs["de"]?.displayName)

        assertEquals("🇮🇹", langs["it"]?.flag)
        assertEquals("🇬🇧", langs["en"]?.flag)
        assertEquals("🇫🇷", langs["fr"]?.flag)
        assertEquals("🇪🇸", langs["es"]?.flag)
        assertEquals("🇩🇪", langs["de"]?.flag)
    }
}
