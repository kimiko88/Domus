import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import RegisterView from './RegisterView.vue'
import { useAuthStore } from '../stores/auth'
import { i18n } from '../locales'

const mockRoute = {
  query: { token: 'valid-token-123' }
}

const mockRouter = {
  push: vi.fn()
}

vi.mock('vue-router', () => ({
  useRoute: () => mockRoute,
  useRouter: () => mockRouter
}))

describe('RegisterView.vue', () => {
  let pinia: any

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    vi.restoreAllMocks()
    mockRoute.query = { token: 'valid-token-123' }
  })

  it('validates invite token on mount and displays role banner', async () => {
    const authStore = useAuthStore()
    vi.spyOn(authStore, 'validateInvite').mockResolvedValue({
      valid: true,
      role: 'operator',
      expires_at: '2026-10-06T00:00:00Z'
    })

    const wrapper = mount(RegisterView, {
      global: {
        plugins: [pinia, i18n],
        stubs: ['router-link']
      }
    })

    await vi.waitFor(() => {
      expect(wrapper.find('.role-banner').exists()).toBe(true)
    })

    expect(wrapper.find('.role-notice-val').text()).toContain('Operatore')
    expect(wrapper.find('input#username').exists()).toBe(true)
  })

  it('displays error when invite token is invalid or expired', async () => {
    const authStore = useAuthStore()
    vi.spyOn(authStore, 'validateInvite').mockRejectedValue(new Error('Invite expired'))

    const wrapper = mount(RegisterView, {
      global: {
        plugins: [pinia, i18n],
        stubs: ['router-link']
      }
    })

    await vi.waitFor(() => {
      expect(wrapper.find('.error-box').exists()).toBe(true)
    })

    expect(wrapper.find('.error-text').text()).toContain('Invite expired')
  })

  it('submits registration form successfully', async () => {
    const authStore = useAuthStore()
    vi.spyOn(authStore, 'validateInvite').mockResolvedValue({
      valid: true,
      role: 'operator',
      expires_at: '2026-10-06T00:00:00Z'
    })
    const regSpy = vi.spyOn(authStore, 'registerWithInvite').mockResolvedValue(true)

    const wrapper = mount(RegisterView, {
      global: {
        plugins: [pinia, i18n],
        stubs: ['router-link']
      }
    })

    await vi.waitFor(() => {
      expect(wrapper.find('input#username').exists()).toBe(true)
    })

    await wrapper.find('input#username').setValue('new_family_member')
    await wrapper.find('input#password').setValue('SuperPassword123!')
    await wrapper.find('input#confirmPassword').setValue('SuperPassword123!')
    await wrapper.find('form').trigger('submit.prevent')

    expect(regSpy).toHaveBeenCalledWith({
      token: 'valid-token-123',
      username: 'new_family_member',
      password: 'SuperPassword123!',
      email: undefined
    })
  })
})
