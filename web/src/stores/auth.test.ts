import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from './auth'
import { apiClient } from '../api/client'

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>()
  return {
    ...actual,
    apiClient: {
      get: vi.fn(),
      post: vi.fn()
    }
  }
})

describe('Auth Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('initializes with unauthenticated state', () => {
    const store = useAuthStore()
    expect(store.isAuthenticated).toBe(false)
    expect(store.user).toBeNull()
    expect(store.isSetupRequired).toBe(false)
  })

  it('checks setup status and updates isSetupRequired', async () => {
    const store = useAuthStore()
    vi.mocked(apiClient.get).mockResolvedValueOnce({ setup_required: true })

    const required = await store.checkSetup()
    expect(required).toBe(true)
    expect(store.isSetupRequired).toBe(true)
    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/auth/setup')
  })

  it('fetches current user and sets authentication state', async () => {
    const store = useAuthStore()
    const mockUser = { id: 'usr-1', username: 'admin', role: 'admin' as const, created_at: '', updated_at: '' }
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockUser)

    await store.fetchMe()
    expect(store.isAuthenticated).toBe(true)
    expect(store.user).toEqual(mockUser)
    expect(store.error).toBeNull()
  })

  it('logs in successfully and loads user', async () => {
    const store = useAuthStore()
    const mockUser = { id: 'usr-1', username: 'admin', role: 'admin' as const, created_at: '', updated_at: '' }
    vi.mocked(apiClient.post).mockResolvedValueOnce({ message: 'Login successful' })
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockUser)

    await store.login('admin', 'secret123')
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/auth/login', {
      username: 'admin',
      password: 'secret123'
    })
    expect(store.isAuthenticated).toBe(true)
    expect(store.user).toEqual(mockUser)
  })

  it('does not report login success when the session user cannot be loaded', async () => {
    const store = useAuthStore()
    vi.mocked(apiClient.post).mockResolvedValueOnce({ message: 'Login successful' })
    vi.mocked(apiClient.get).mockRejectedValueOnce(new Error('session lookup failed'))

    await expect(store.login('admin', 'secret123')).rejects.toThrow('session lookup failed')
    expect(store.isAuthenticated).toBe(false)
    expect(store.user).toBeNull()
    expect(store.error).toBe('session lookup failed')
  })

  it('logs out and clears user state', async () => {
    const store = useAuthStore()
    store.user = { id: 'usr-1', username: 'admin', role: 'admin', created_at: '', updated_at: '' }

    vi.mocked(apiClient.post).mockResolvedValueOnce({})

    await store.logout()
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/auth/logout')
    expect(store.isAuthenticated).toBe(false)
    expect(store.user).toBeNull()
  })
})
