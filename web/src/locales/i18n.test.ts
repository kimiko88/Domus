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
    const locales = [
      { code: 'it', catalog: itLocale },
      { code: 'fr', catalog: frLocale },
      { code: 'es', catalog: esLocale },
      { code: 'de', catalog: deLocale },
      { code: 'ru', catalog: ruLocale }
    ]
    const samples = [
      ['devices', 'view', 'scanLanTitle'],
      ['shopping', 'view', 'receiptIntro'],
      ['chores', 'view', 'googleStep1'],
      ['media', 'view', 'broadcastDesc'],
      ['dashboard', 'view', 'climateThermostat'],
      ['dashboard', 'view', 'batteryLabel'],
      ['dashboard', 'view', 'defaultProfileLivingRoom'],
      ['dashboard', 'view', 'defaultProfileKitchen'],
      ['dashboard', 'view', 'defaultProfileNight'],
      ['dashboard', 'view', 'roomLivingRoom'],
      ['dashboard', 'view', 'roomKitchen'],
      ['dashboard', 'view', 'roomBedroom'],
      ['integrations', 'view', 'genericLanHost'],
      ['integrations', 'view', 'selectedForLocalTuya'],
      ['integrations', 'view', 'filterAll'],
      ['integrations', 'view', 'filterOnline'],
      ['integrations', 'view', 'filterOffline'],
      ['integrations', 'view', 'filterSelected'],
      ['integrations', 'view', 'filterUnselected'],
      ['integrations', 'view', 'unblock'],
      ['integrations', 'view', 'messages', 'tuyaConnectedTitle'],
      ['integrations', 'view', 'messages', 'deviceOfflineDesc'],
      ['integrations', 'view', 'messages', 'arpScanCompleteDesc'],
      ['integrations', 'view', 'messages', 'deviceUnblockedTitle'],
      ['common', 'wsNotifications', 'deviceDiscoveredTitle'],
      ['common', 'wsNotifications', 'deviceDiscoveredMessage'],
      ['common', 'wsNotifications', 'automationFailedMessage'],
      ['common', 'pwa', 'updateAvailableTitle'],
      ['common', 'pwa', 'updateAvailableDescription'],
      ['common', 'pwa', 'updateNow'],
      ['common', 'pwa', 'later'],
      ['common', 'notificationRegion'],
      ['common', 'notifications', 'actionExecutedTitle'],
      ['common', 'notifications', 'actionExecutedMessage'],
      ['common', 'notifications', 'actionFailedMessage'],
      ['common', 'errors', 'fileReadFailed'],
      ['integrations', 'view', 'deviceOnline'],
      ['integrations', 'view', 'valueToSend'],
      ['integrations', 'view', 'messages', 'unsupportedTemperatureTitle'],
      ['integrations', 'view', 'messages', 'setpointUpdatedDesc'],
      ['integrations', 'view', 'messages', 'commandSentDesc'],
      ['scenes', 'editor', 'actionRequired'],
      ['chores', 'view', 'completedMessage'],
      ['predictiveClimate', 'view', 'invalidSample'],
      ['cameras', 'view', 'filterPerson'],
      ['media', 'view', 'stateBuffering'],
      ['areas', 'nameRequired'],
      ['flowchart', 'view', 'palette', 'stateChangeTitle'],
      ['flowchart', 'view', 'validation', 'unsupportedAction'],
      ['energy', 'powerWheelTitle'],
      ['energy', 'recalculateModulation'],
      ['floorplan', 'view', 'commandSentTitle'],
      ['floorplan', 'view', 'roomTypes', 'kitchen'],
      ['devices', 'view', 'scanCompletedTitle'],
      ['devices', 'view', 'filteredCount'],
      ['integrations', 'view', 'tuyaCloudSetupInstruction'],
      ['shopping', 'view', 'transferCompleted'],
      ['store', 'view', 'rejectedByAdministrator'],
      ['dashboard', 'view', 'customize'],
      ['dashboard', 'view', 'deleteConfirm'],
      ['automations', 'editor', 'titleEdit'],
      ['automations', 'editor', 'errors', 'invalidJson'],
      ['automations', 'validation', 'objectExpected'],
      ['automations', 'graphErrors', 'conditionCycle'],
      ['flowchart', 'conditionEditor', 'entityLabel'],
      ['flowchart', 'conditionEditor', 'stateLabel'],
      ['guide', 'searchAria'],
      ['pantry', 'view', 'kpis', 'lowStock'],
      ['pantry', 'view', 'kpis', 'filterLowStock'],
      ['satellites', 'askPlaceholder'],
      ['satellites', 'quickChips', 'pastaTimer'],
      ['satellites', 'fallbackSatellites', 'kitchenArea'],
      ['integrations', 'view', 'tuyaProjectIntro'],
      ['integrations', 'view', 'tuyaProjectPath'],
      ['common', 'done'],
      ['automations', 'cooldown'],
      ['integrations', 'view', 'deviceOnline'],
      ['flowchart', 'view', 'nodeTypes', 'trigger'],
      ['pantry', 'view', 'mealCalendar', 'mealTypes', 'dessert'],
      ['guide', 'sec7RouteLabel'],
      ['automations', 'graphErrors', 'conditionTitles', 'state'],
      ['pantry', 'view', 'kpis', 'filterExpiring']
    ]

    for (const { code, catalog } of locales) {
      for (const path of samples) {
        const keyPath = path.join('.')
        const translated = path.reduce((value: any, key) => value?.[key], catalog as any)
        const english = path.reduce((value: any, key) => value?.[key], enLocale as any)
        expect(typeof translated, keyPath).toBe('string')
        expect(translated.length, keyPath).toBeGreaterThan(0)
        // “Dessert” is the correct French word too; matching English is not a fallback here.
        if (code === 'fr' && keyPath === 'pantry.view.mealCalendar.mealTypes.dessert') {
          expect(translated, keyPath).toBe('Dessert')
        } else {
          expect(translated, `Missing translation for ${keyPath}`).not.toBe(english)
        }
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
