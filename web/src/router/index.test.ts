import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createApp } from 'vue'
import { createRouterInstance } from './index'
import { useAuthStore } from '../stores/auth'

describe('Router navigation guard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('redirects to /setup if setup is required', async () => {
    const authStore = useAuthStore()
    vi.spyOn(authStore, 'checkSetup').mockResolvedValue(true)
    authStore.isSetupRequired = true

    const router = createRouterInstance()
    const app = createApp({})
    app.use(router)

    await router.push('/')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/setup')
  })

  it('redirects to /login if unauthenticated and route requires auth', async () => {
    const authStore = useAuthStore()
    vi.spyOn(authStore, 'checkSetup').mockResolvedValue(false)
    vi.spyOn(authStore, 'fetchMe').mockResolvedValue(null)
    authStore.isSetupRequired = false
    authStore.user = null

    const router = createRouterInstance()
    const app = createApp({})
    app.use(router)

    await router.push('/devices')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('allows access to dashboard when authenticated', async () => {
    const authStore = useAuthStore()
    vi.spyOn(authStore, 'checkSetup').mockResolvedValue(false)
    authStore.isSetupRequired = false
    authStore.user = {
      id: 'u1',
      username: 'admin',
      role: 'admin',
      created_at: '',
      updated_at: ''
    }

    const router = createRouterInstance()
    const app = createApp({})
    app.use(router)

    await router.push('/')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/')
  })
})
