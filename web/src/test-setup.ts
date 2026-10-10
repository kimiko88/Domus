import { config } from '@vue/test-utils'
import { i18n } from './locales/index'

// Register real i18n instance globally for @vue/test-utils
config.global.plugins = [i18n]
