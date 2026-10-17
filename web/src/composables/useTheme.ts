import { ref, computed, watch } from 'vue'

export type Theme = 'dark' | 'light' | 'oled' | 'system'

export interface ThemeOption {
  id: Theme
  nameKey: string
  label: string
  description: string
  icon: string
  previewBg: string
  previewBorder: string
}

export const themeOptions: ThemeOption[] = [
  {
    id: 'dark',
    nameKey: 'settings.themes.dark',
    label: 'Tema Scuro',
    description: 'Tonalità blu notte e ardesia, ideale per la maggior parte delle stanze',
    icon: 'Moon',
    previewBg: '#0f172a',
    previewBorder: '#0ea5e9'
  },
  {
    id: 'light',
    nameKey: 'settings.themes.light',
    label: 'Tema Chiaro',
    description: 'Interfaccia nitida ad alto contrasto per ambienti molto illuminati',
    icon: 'Sun',
    previewBg: '#ffffff',
    previewBorder: '#38bdf8'
  },
  {
    id: 'oled',
    nameKey: 'settings.themes.oled',
    label: 'Midnight OLED',
    description: 'Nero assoluto #000000, perfetto per display a parete OLED e minor consumo',
    icon: 'Smartphone',
    previewBg: '#000000',
    previewBorder: '#3b82f6'
  },
  {
    id: 'system',
    nameKey: 'settings.themes.system',
    label: 'Automatico (Sistema)',
    description: 'Segue automaticamente le preferenze di colore del dispositivo o del browser',
    icon: 'Monitor',
    previewBg: 'linear-gradient(135deg, #0f172a 50%, #f8fafc 50%)',
    previewBorder: '#a855f7'
  }
]

const STORAGE_KEY = 'domus-theme'

const mql = typeof window !== 'undefined' && window.matchMedia
  ? window.matchMedia('(prefers-color-scheme: dark)')
  : null

function getInitialTheme(): Theme {
  try {
    const stored = localStorage.getItem(STORAGE_KEY) as Theme | null
    if (stored && ['dark', 'light', 'oled', 'system'].includes(stored)) {
      return stored
    }
  } catch {
    // fallback if localStorage not accessible
  }
  return 'dark'
}

// Shared reactive state
const theme = ref<Theme>(getInitialTheme())

function resolveActualTheme(t: Theme): 'dark' | 'light' | 'oled' {
  if (t === 'system') {
    return mql && mql.matches ? 'dark' : 'light'
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
  mql.addEventListener('change', () => {
    if (theme.value === 'system') {
      applyTheme('system')
    }
  })
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
