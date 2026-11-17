import { ref, computed, watch } from 'vue'

export type Theme = 'dark' | 'light' | 'oled' | 'system'

export interface ThemeOption {
  id: Theme
  nameKey: string
  descriptionKey: string
  icon: string
  previewBg: string
  previewBorder: string
}

export const themeOptions: ThemeOption[] = [
  {
    id: 'dark',
    nameKey: 'settings.themes.dark',
    descriptionKey: 'settings.themeDescriptions.dark',
    icon: 'Moon',
    previewBg: '#0f172a',
    previewBorder: '#0ea5e9'
  },
  {
    id: 'light',
    nameKey: 'settings.themes.light',
    descriptionKey: 'settings.themeDescriptions.light',
    icon: 'Sun',
    previewBg: '#ffffff',
    previewBorder: '#38bdf8'
  },
  {
    id: 'oled',
    nameKey: 'settings.themes.oled',
    descriptionKey: 'settings.themeDescriptions.oled',
    icon: 'Smartphone',
    previewBg: '#000000',
    previewBorder: '#3b82f6'
  },
  {
    id: 'system',
    nameKey: 'settings.themes.system',
    descriptionKey: 'settings.themeDescriptions.system',
    icon: 'Monitor',
    previewBg: 'linear-gradient(135deg, #0f172a 50%, #f8fafc 50%)',
    previewBorder: '#a855f7'
  }
]

const STORAGE_KEY = 'domus-theme'

const mql = typeof window !== 'undefined' && typeof window.matchMedia === 'function'
  ? window.matchMedia('(prefers-color-scheme: dark)')
  : null

// Keep the operating-system preference reactive: MediaQueryList.matches is
// not a Vue dependency by itself, even though the browser updates it live.
const prefersDark = ref(mql?.matches ?? false)

function getInitialTheme(): Theme {
  try {
    const stored = localStorage.getItem(STORAGE_KEY) as Theme | null
    if (stored && ['dark', 'light', 'oled', 'system'].includes(stored)) {
      return stored
    }
  } catch {
    // fallback if localStorage not accessible
  }
  return 'system'
}

// Shared reactive state
const theme = ref<Theme>(getInitialTheme())

function resolveActualTheme(t: Theme): 'dark' | 'light' | 'oled' {
  if (t === 'system') {
    return prefersDark.value ? 'dark' : 'light'
  }
  return t
}

const resolvedTheme = computed<'dark' | 'light' | 'oled'>(() => resolveActualTheme(theme.value))
const isDark = computed(() => resolvedTheme.value !== 'light')

function applyTheme(t: Theme) {
  if (typeof document === 'undefined') return
  const actual = resolveActualTheme(t)
  const metaThemeColor = document.querySelector('meta[name="theme-color"]')

  if (actual === 'light') {
    document.documentElement.setAttribute('data-theme', 'light')
    metaThemeColor?.setAttribute('content', '#f8fafc')
  } else if (actual === 'oled') {
    document.documentElement.setAttribute('data-theme', 'oled')
    metaThemeColor?.setAttribute('content', '#000000')
  } else {
    document.documentElement.removeAttribute('data-theme')
    metaThemeColor?.setAttribute('content', '#0b1120')
  }
}

// Apply immediately on module load
applyTheme(theme.value)

// Sync DOM whenever reactive value changes
watch(theme, (t) => {
  try {
    localStorage.setItem(STORAGE_KEY, t)
  } catch {}
  applyTheme(t)
})

// Listen to system changes if in system mode
if (mql) {
  const handleSystemThemeChange = (event: MediaQueryListEvent | MediaQueryList) => {
    prefersDark.value = event.matches
    if (theme.value === 'system') applyTheme('system')
  }
  if (typeof mql.addEventListener === 'function') {
    mql.addEventListener('change', handleSystemThemeChange)
  } else {
    // Legacy Safari/iOS and older embedded browsers.
    mql.addListener(handleSystemThemeChange)
  }
}

export function useTheme() {
  function toggleTheme() {
    if (resolvedTheme.value === 'light') {
      theme.value = 'dark'
    } else {
      theme.value = 'light'
    }
  }

  function setTheme(t: Theme) {
    theme.value = t
  }

  return {
    theme,
    resolvedTheme,
    isDark,
    themeOptions,
    toggleTheme,
    setTheme
  }
}
