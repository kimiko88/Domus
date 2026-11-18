import { beforeEach } from 'vitest'
import { config } from '@vue/test-utils'
import { i18n } from './locales/index'

// Register the real i18n instance globally for @vue/test-utils.
config.global.plugins = [i18n]

// Keep tests deterministic even when the browser-like storage contains a saved locale.
beforeEach(() => {
  i18n.global.locale.value = 'it'
})
