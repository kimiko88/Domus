import { describe, it, expect, beforeEach } from 'vitest'
import itLocale from './it.json'
import enLocale from './en.json'
import { i18n, setLanguage, getAvailableLanguages } from './index'

describe('Localization (i18n)', () => {
  beforeEach(() => {
    localStorage.clear()
    setLanguage('it')
  })

  it('contains consistent translation keys between Italian and English', () => {
    const itKeys = Object.keys(itLocale).sort()
    const enKeys = Object.keys(enLocale).sort()
    expect(itKeys).toEqual(enKeys)

    // Deep check top namespaces
    for (const key of itKeys) {
      const itSubKeys = Object.keys((itLocale as Record<string, any>)[key]).sort()
      const enSubKeys = Object.keys((enLocale as Record<string, any>)[key]).sort()
      expect(itSubKeys).toEqual(enSubKeys)
    }
  })

  it('defaults to Italian or saved language', () => {
    expect(i18n.global.locale.value).toBe('it')
  })

  it('switches language and persists to localStorage', () => {
    setLanguage('en')
    expect(i18n.global.locale.value).toBe('en')
    expect(localStorage.getItem('domus_language')).toBe('en')

    setLanguage('it')
    expect(i18n.global.locale.value).toBe('it')
    expect(localStorage.getItem('domus_language')).toBe('it')
  })

  it('exposes available languages metadata', () => {
    const langs = getAvailableLanguages()
    expect(langs).toEqual([
      { code: 'it', name: 'Italiano', flag: '🇮🇹' },
      { code: 'en', name: 'English', flag: '🇬🇧' }
    ])
  })
})
