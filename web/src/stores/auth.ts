import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient, ApiError } from '../api/client'
import type { User, SetupStatus } from '../api/types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const isSetupRequired = ref<boolean>(false)
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  const isAuthenticated = computed(() => user.value !== null)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const isOperator = computed(() => user.value?.role === 'admin' || user.value?.role === 'operator')

  async function checkSetup(): Promise<boolean> {
    try {
      loading.value = true
      const data = await apiClient.get<SetupStatus>('/api/v1/auth/setup')
      isSetupRequired.value = data.setup_required
      return data.setup_required
    } catch (err: any) {
      error.value = err?.message || 'Failed to check setup status'
      return false
    } finally {
      loading.value = false
    }
  }

  async function fetchMe(): Promise<User | null> {
    try {
      loading.value = true
      error.value = null
      const userData = await apiClient.get<User>('/api/v1/auth/me')
      user.value = userData
      return userData
    } catch (err: any) {
      user.value = null
      if (err instanceof ApiError && err.status === 401) {
        // Expected when unauthenticated
      } else {
        error.value = err?.message || 'Failed to fetch user'
      }
      return null
    } finally {
      loading.value = false
    }
  }

  async function login(username: string, password: string): Promise<boolean> {
    try {
      loading.value = true
      error.value = null
      await apiClient.post('/api/v1/auth/login', { username, password })
      await fetchMe()
      return true
    } catch (err: any) {
      error.value = err?.message || 'Login failed'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function registerFirstAdmin(username: string, password: string): Promise<boolean> {
    try {
      loading.value = true
      error.value = null
      await apiClient.post('/api/v1/auth/setup', { username, password })
      isSetupRequired.value = false
      await fetchMe()
      return true
    } catch (err: any) {
      error.value = err?.message || 'First admin setup failed'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function logout(): Promise<void> {
    try {
      loading.value = true
      await apiClient.post('/api/v1/auth/logout')
    } catch (err) {
      console.warn('Logout request failed:', err)
    } finally {
      user.value = null
      loading.value = false
    }
  }

  return {
    user,
    isAuthenticated,
    isAdmin,
    isOperator,
    isSetupRequired,
    loading,
    error,
    checkSetup,
    fetchMe,
    login,
    registerFirstAdmin,
    logout
  }
})
