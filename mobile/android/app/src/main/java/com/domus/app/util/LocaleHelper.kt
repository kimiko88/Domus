package com.domus.app.util

import android.content.Context
import android.content.res.Configuration
import java.util.Locale

data class SupportedLanguage(
    val code: String,
    val displayName: String,
    val flag: String
)

object LocaleHelper {
    private const val PREF_KEY_LANG = "domus_language"
    private const val PREFS_NAME = "domus_locale_prefs"

    val supportedLanguages = listOf(
        SupportedLanguage("it", "Italiano", "🇮🇹"),
        SupportedLanguage("en", "English", "🇬🇧"),
        SupportedLanguage("fr", "Français", "🇫🇷"),
        SupportedLanguage("es", "Español", "🇪🇸"),
        SupportedLanguage("de", "Deutsch", "🇩🇪")
    )

    fun getLanguage(context: Context): String {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        return prefs.getString(PREF_KEY_LANG, null) ?: Locale.getDefault().language.let { lang ->
            if (supportedLanguages.any { it.code == lang }) lang else "it"
        }
    }

    fun setLocale(context: Context, languageCode: String): Context {
        persist(context, languageCode)
        return updateResources(context, languageCode)
    }

    private fun persist(context: Context, languageCode: String) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        prefs.edit().putString(PREF_KEY_LANG, languageCode).apply()
    }

    fun updateResources(context: Context, languageCode: String): Context {
        val locale = Locale(languageCode)
        Locale.setDefault(locale)

        val configuration = Configuration(context.resources.configuration)
        configuration.setLocale(locale)
        configuration.setLayoutDirection(locale)

        return context.createConfigurationContext(configuration)
    }

    fun onAttach(context: Context): Context {
        val lang = getLanguage(context)
        return updateResources(context, lang)
    }
}
