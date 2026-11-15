import { describe, it, expect, beforeEach } from 'vitest'
import itLocale from './it.json'
import enLocale from './en.json'
import frLocale from './fr.json'
import esLocale from './es.json'
import deLocale from './de.json'
import ruLocale from './ru'
import { i18n, setLanguage, getAvailableLanguages, type SupportedLanguage } from './index'

describe('Localization (i18n)', () => {
  beforeEach(() => {
    localStorage.clear()
    setLanguage('it')
  })

  it('contains identical translation keys recursively across all 6 locales', () => {
    const locales: Record<string, Record<string, any>> = {
      it: itLocale,
      en: enLocale,
      fr: frLocale,
      es: esLocale,
      de: deLocale,
      ru: ruLocale
    }

    const compareKeys = (baseline: any, current: any, path: string, code: string) => {
      expect(current, `Locale ${code} is missing ${path}`).toBeDefined()
      if (baseline && typeof baseline === 'object' && !Array.isArray(baseline)) {
        expect(current && typeof current === 'object' && !Array.isArray(current),
          `Locale ${code} has an invalid structure at ${path}`).toBe(true)
        expect(Object.keys(current).sort(), `Locale ${code} keys mismatch at ${path}`)
          .toEqual(Object.keys(baseline).sort())
        for (const key of Object.keys(baseline)) {
          compareKeys(baseline[key], current[key], `${path}.${key}`, code)
        }
      }
    }

    for (const [code, catalog] of Object.entries(locales)) {
      compareKeys(itLocale, catalog, 'root', code)
    }
  })

  it('translates the recently migrated device, shopping, chores and media copy', () => {
    const locales = [itLocale, frLocale, esLocale, deLocale, ruLocale]
    const samples = [
      ['devices', 'view', 'scanLanTitle'],
      ['shopping', 'view', 'receiptIntro'],
      ['chores', 'view', 'googleStep1'],
      ['media', 'view', 'broadcastDesc']
    ]

    for (const catalog of locales) {
      for (const path of samples) {
        const translated = path.reduce((value: any, key) => value?.[key], catalog as any)
        const english = path.reduce((value: any, key) => value?.[key], enLocale as any)
        expect(typeof translated, path.join('.')).toBe('string')
        expect(translated.length, path.join('.')).toBeGreaterThan(0)
        expect(translated, `Missing translation for ${path.join('.')}`).not.toBe(english)
      }
    }
  })

  it('defaults to Italian or saved language', () => {
    expect(i18n.global.locale.value).toBe('it')
  })

  it('switches between all 6 languages and persists to localStorage', () => {
    const languages: SupportedLanguage[] = ['en', 'fr', 'es', 'de', 'ru', 'it']

    for (const lang of languages) {
      setLanguage(lang)
      expect(i18n.global.locale.value).toBe(lang)
      expect(localStorage.getItem('domus_language')).toBe(lang)
    }
  })

  it('exposes all 6 available languages metadata with codes, names, and flags', () => {
    const langs = getAvailableLanguages()
    expect(langs).toEqual([
      { code: 'it', name: 'Italiano', flag: '🇮🇹' },
      { code: 'en', name: 'English', flag: '🇬🇧' },
      { code: 'fr', name: 'Français', flag: '🇫🇷' },
      { code: 'es', name: 'Español', flag: '🇪🇸' },
      { code: 'de', name: 'Deutsch', flag: '🇩🇪' },
      { code: 'ru', name: 'Русский', flag: '🇷🇺' }
    ])
  })
})
