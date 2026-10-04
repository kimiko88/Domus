import { createI18n } from 'vue-i18n'
import it from './it.json'
import en from './en.json'

export type SupportedLanguage = 'it' | 'en'

export interface LanguageMeta {
  code: SupportedLanguage
  name: string
  flag: string
}

export const supportedLanguages: LanguageMeta[] = [
  { code: 'it', name: 'Italiano', flag: '🇮🇹' },
  { code: 'en', name: 'English', flag: '🇬🇧' }
]

const STORAGE_KEY = 'domus_language'

export function getSavedLanguage(): SupportedLanguage {
  const saved = localStorage.getItem(STORAGE_KEY) as SupportedLanguage | null
  if (saved && (saved === 'it' || saved === 'en')) {
    return saved
  }
  return 'it'
}

export const i18n = createI18n({
  legacy: false,
  locale: getSavedLanguage(),
  fallbackLocale: 'en',
  messages: {
    it,
    en
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
