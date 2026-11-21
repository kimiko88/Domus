import { afterAll, describe, expect, it } from 'vitest'
import { i18n, setLanguage, supportedLanguages } from './index'

describe('climate history localization', () => {
  const originalLocale = i18n.global.locale.value

  afterAll(() => setLanguage(originalLocale as any))

  it.each(supportedLanguages)('provides complete climate history labels in %s', language => {
    setLanguage(language.code)
    expect(i18n.global.t('nav.climateHistory')).not.toBe('nav.climateHistory')
    expect(i18n.global.t('climateHistory.title')).not.toBe('climateHistory.title')
    expect(i18n.global.t('climateHistory.coolingRate')).not.toBe('climateHistory.coolingRate')
    expect(i18n.global.t('climateHistory.settings.roomMap')).not.toBe('climateHistory.settings.roomMap')
    expect(i18n.global.t('climateHistory.insights.humidity_high')).not.toBe('climateHistory.insights.humidity_high')
  })
})
