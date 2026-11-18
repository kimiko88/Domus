// Vitest-only fallback for the virtual module normally provided by vite-plugin-pwa.
type RegisterSWOptions = {
  immediate?: boolean
  onNeedRefresh?: () => void
  onOfflineReady?: () => void
  onRegisteredSW?: (serviceWorkerUrl: string, registration?: ServiceWorkerRegistration) => void
  onRegisterError?: (error: unknown) => void
}

export function registerSW(_options: RegisterSWOptions = {}) {
  return async (_reloadPage = false): Promise<void> => undefined
}
