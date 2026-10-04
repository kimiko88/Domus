import { ref, watch } from 'vue'

export type Theme = 'dark' | 'light'

const STORAGE_KEY = 'domus-theme'

function getInitialTheme(): Theme {
  const stored = localStorage.getItem(STORAGE_KEY) as Theme | null
  if (stored === 'light' || stored === 'dark') return stored
  // Respect OS preference on first load
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

// Shared reactive state across all composable instances
const theme = ref<Theme>(getInitialTheme())

// Apply theme attribute to the root element
function applyTheme(t: Theme) {
  if (t === 'light') {
    document.documentElement.setAttribute('data-theme', 'light')
    document
      .querySelector('meta[name="theme-color"]')
      ?.setAttribute('content', '#f1f5f9')
  } else {
    document.documentElement.removeAttribute('data-theme')
    document
      .querySelector('meta[name="theme-color"]')
      ?.setAttribute('content', '#0b1120')
  }
}

// Apply immediately on module load
applyTheme(theme.value)

// Sync DOM whenever reactive value changes
watch(theme, (t) => {
  localStorage.setItem(STORAGE_KEY, t)
  applyTheme(t)
})

export function useTheme() {
  const isDark = ref(theme.value === 'dark')

  watch(theme, (t) => {
    isDark.value = t === 'dark'
  })

  function toggleTheme() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
  }

  function setTheme(t: Theme) {
    theme.value = t
  }

  return {
    theme,
    isDark,
    toggleTheme,
    setTheme
  }
}
