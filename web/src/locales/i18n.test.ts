import { describe, it, expect, beforeEach } from 'vitest'
import itLocale from './it.json'
import enLocale from './en.json'
import frLocale from './fr.json'
import esLocale from './es.json'
import deLocale from './de.json'
import { i18n, setLanguage, getAvailableLanguages, type SupportedLanguage } from './index'

describe('Localization (i18n)', () => {
  beforeEach(() => {
    localStorage.clear()
    setLanguage('it')
  })

  it('contains consistent top-level and nested translation keys across all 5 locales', () => {
    const locales: Record<string, Record<string, any>> = {
      it: itLocale,
      en: enLocale,
      fr: frLocale,
      es: esLocale,
      de: deLocale
    }

    const baselineKeys = Object.keys(itLocale).sort()

    for (const [code, catalog] of Object.entries(locales)) {
      const keys = Object.keys(catalog).sort()
      expect(keys, `Locale ${code} top keys mismatch`).toEqual(baselineKeys)

      for (const section of baselineKeys) {
        const baselineSubKeys = Object.keys(itLocale[section as keyof typeof itLocale]).sort()
        const currentSubKeys = Object.keys(catalog[section]).sort()
        expect(currentSubKeys, `Locale ${code} section ${section} keys mismatch`).toEqual(baselineSubKeys)
      }
    }
  })

  it('defaults to Italian or saved language', () => {
    expect(i18n.global.locale.value).toBe('it')
  })

  it('switches between all 5 languages and persists to localStorage', () => {
    const languages: SupportedLanguage[] = ['en', 'fr', 'es', 'de', 'it']

    for (const lang of languages) {
      setLanguage(lang)
      expect(i18n.global.locale.value).toBe(lang)
      expect(localStorage.getItem('domus_language')).toBe(lang)
    }
  })

  it('exposes all 5 available languages metadata with codes, names, and flags', () => {
    const langs = getAvailableLanguages()
    expect(langs).toEqual([
      { code: 'it', name: 'Italiano', flag: '🇮🇹' },
      { code: 'en', name: 'English', flag: '🇬🇧' },
      { code: 'fr', name: 'Français', flag: '🇫🇷' },
      { code: 'es', name: 'Español', flag: '🇪🇸' },
      { code: 'de', name: 'Deutsch', flag: '🇩🇪' }
    ])
  })
})
