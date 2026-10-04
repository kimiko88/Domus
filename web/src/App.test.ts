import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import App from './App.vue'
import { useAuthStore } from './stores/auth'

describe('App.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders router view and toast container', () => {
    const authStore = useAuthStore()
    authStore.user = {
      id: 'u1',
      username: 'admin',
      role: 'admin',
      created_at: '',
      updated_at: ''
    }

    const wrapper = mount(App, {
      global: {
        stubs: {
          RouterView: { template: '<div class="routed-view">Home Content</div>' },
          AppSidebar: true,
          AppTopbar: true,
          BaseToast: true
        }
      }
    })

    expect(wrapper.find('.routed-view').text()).toBe('Home Content')
  })
})
