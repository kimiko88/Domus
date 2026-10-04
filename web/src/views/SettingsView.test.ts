import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import SettingsView from './SettingsView.vue'
import { useAuthStore } from '../stores/auth'

describe('SettingsView.vue', () => {
  let authStore: ReturnType<typeof useAuthStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    authStore = useAuthStore()
  })

  it('renders user information and role', () => {
    authStore.user = {
      id: 'usr_admin',
      username: 'administrator',
      role: 'admin',
      created_at: '2026-10-04T00:00:00Z',
      updated_at: '2026-10-04T00:00:00Z'
    }

    const wrapper = mount(SettingsView)
    expect(wrapper.text()).toContain('administrator')
    expect(wrapper.text()).toContain('admin')
    expect(wrapper.text()).toContain('Impostazioni')
  })

  it('calls authStore.logout when logout button is pressed', async () => {
    authStore.user = {
      id: 'usr_admin',
      username: 'administrator',
      role: 'admin',
      created_at: '2026-10-04T00:00:00Z',
      updated_at: '2026-10-04T00:00:00Z'
    }
    const logoutSpy = vi.spyOn(authStore, 'logout').mockResolvedValue(undefined)

    const wrapper = mount(SettingsView)
    const logoutBtn = wrapper.find('.btn-logout-settings')
    expect(logoutBtn.exists()).toBe(true)

    await logoutBtn.trigger('click')
    expect(logoutSpy).toHaveBeenCalled()
  })
})
