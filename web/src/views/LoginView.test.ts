import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import LoginView from './LoginView.vue'
import { useAuthStore } from '../stores/auth'

describe('LoginView.vue', () => {
  let authStore: ReturnType<typeof useAuthStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    authStore = useAuthStore()
  })

  it('renders login form elements', () => {
    const wrapper = mount(LoginView, {
      global: {
        stubs: {
          RouterLink: true
        }
      }
    })

    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
    expect(wrapper.find('input[type="password"]').exists()).toBe(true)
    expect(wrapper.find('button[type="submit"]').text()).toContain('Accedi')
  })

  it('calls authStore.login on form submit', async () => {
    const loginSpy = vi.spyOn(authStore, 'login').mockResolvedValue(true)

    const wrapper = mount(LoginView, {
      global: {
        stubs: {
          RouterLink: true
        }
      }
    })

    await wrapper.find('input[type="text"]').setValue('admin')
    await wrapper.find('input[type="password"]').setValue('secret123')
    await wrapper.find('form').trigger('submit.prevent')

    expect(loginSpy).toHaveBeenCalledWith('admin', 'secret123')
  })

  it('displays error message when login fails', async () => {
    vi.spyOn(authStore, 'login').mockRejectedValue(new Error('Invalid username or password'))

    const wrapper = mount(LoginView, {
      global: {
        stubs: {
          RouterLink: true
        }
      }
    })

    await wrapper.find('input[type="text"]').setValue('admin')
    await wrapper.find('input[type="password"]').setValue('wrong')
    await wrapper.find('form').trigger('submit.prevent')

    // Wait for promise resolution
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    expect(wrapper.text()).toContain('Invalid username or password')
  })
})
