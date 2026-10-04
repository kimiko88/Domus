import { createI18n } from 'vue-i18n'
import it from './it.json'
import en from './en.json'
import fr from './fr.json'
import es from './es.json'
import de from './de.json'

export type SupportedLanguage = 'it' | 'en' | 'fr' | 'es' | 'de'

export interface LanguageMeta {
  code: SupportedLanguage
  name: string
  flag: string
}

export const supportedLanguages: LanguageMeta[] = [
  { code: 'it', name: 'Italiano', flag: '🇮🇹' },
  { code: 'en', name: 'English', flag: '🇬🇧' },
  { code: 'fr', name: 'Français', flag: '🇫🇷' },
  { code: 'es', name: 'Español', flag: '🇪🇸' },
  { code: 'de', name: 'Deutsch', flag: '🇩🇪' }
]

const STORAGE_KEY = 'domus_language'

export function getSavedLanguage(): SupportedLanguage {
  try {
    const saved = localStorage.getItem(STORAGE_KEY) as SupportedLanguage | null
    if (saved && ['it', 'en', 'fr', 'es', 'de'].includes(saved)) {
      return saved
    }
  } catch {
    // fallback if localStorage is unavailable
  }
  return 'it'
}

export const i18n = createI18n({
  legacy: false,
  locale: getSavedLanguage(),
  fallbackLocale: 'en',
  messages: {
    it,
    en,
    fr,
    es,
    de
  }
})

export function setLanguage(lang: SupportedLanguage) {
  i18n.global.locale.value = lang
  try {
    localStorage.setItem(STORAGE_KEY, lang)
  } catch {
    // localStorage might be unavailable in some test environments
  }
}

export function getCurrentLanguage(): SupportedLanguage {
  return i18n.global.locale.value as SupportedLanguage
}

export function getAvailableLanguages(): LanguageMeta[] {
  return supportedLanguages
}

export default i18n
