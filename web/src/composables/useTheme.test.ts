import { beforeEach, describe, expect, it, vi } from 'vitest'

describe('useTheme system preference', () => {
  beforeEach(() => {
    vi.resetModules()
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
  })

  it('follows live operating-system color-scheme changes in system mode', async () => {
    const listeners: Array<(event: MediaQueryListEvent) => void> = []
    const mediaQuery = {
      matches: true,
      addEventListener: vi.fn((_type: string, listener: (event: MediaQueryListEvent) => void) => {
        listeners.push(listener)
      }),
      removeEventListener: vi.fn(),
      addListener: vi.fn()
    } as unknown as MediaQueryList

    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: vi.fn(() => mediaQuery)
    })

    const { useTheme } = await import('./useTheme')
    const { theme, resolvedTheme, setTheme } = useTheme()

    expect(theme.value).toBe('system')
    expect(resolvedTheme.value).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBeNull()

    listeners[0]({ matches: false } as MediaQueryListEvent)
    expect(resolvedTheme.value).toBe('light')
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')

    listeners[0]({ matches: true } as MediaQueryListEvent)
    expect(resolvedTheme.value).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBeNull()

    setTheme('light')
    listeners[0]({ matches: true } as MediaQueryListEvent)
    expect(theme.value).toBe('light')
    expect(resolvedTheme.value).toBe('light')
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  })
})
