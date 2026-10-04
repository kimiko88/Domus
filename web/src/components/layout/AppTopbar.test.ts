import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AppTopbar from './AppTopbar.vue'
import { useAuthStore } from '../../stores/auth'
import { useWsStore } from '../../stores/ws'
import { i18n, setLanguage } from '@/locales'

describe('AppTopbar.vue', () => {
  let authStore: ReturnType<typeof useAuthStore>
  let wsStore: ReturnType<typeof useWsStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    authStore = useAuthStore()
    wsStore = useWsStore()
  })

  it('renders page title', () => {
    const wrapper = mount(AppTopbar, {
      props: {
        title: 'Dispositivi'
      }
    })
    expect(wrapper.text()).toContain('Dispositivi')
  })

  it('shows WebSocket connection status indicator', () => {
    wsStore.setConnected(true)
    const wrapper = mount(AppTopbar)
    expect(wrapper.find('.ws-status').classes()).toContain('is-connected')
    expect(wrapper.text()).toContain('Live')
  })

  it('displays user profile info when logged in', () => {
    authStore.user = {
      id: 'u1',
      username: 'admin_master',
      role: 'admin',
      created_at: '2026-10-04T00:00:00Z',
      updated_at: '2026-10-04T00:00:00Z'
    }

    const wrapper = mount(AppTopbar)
    expect(wrapper.text()).toContain('admin_master')
    expect(wrapper.text()).toContain('admin')
  })

  it('calls authStore.logout when logout button is clicked', async () => {
    authStore.user = {
      id: 'u1',
      username: 'user1',
      role: 'viewer',
      created_at: '2026-10-04T00:00:00Z',
      updated_at: '2026-10-04T00:00:00Z'
    }
    const logoutSpy = vi.spyOn(authStore, 'logout').mockResolvedValue(undefined)

    const wrapper = mount(AppTopbar)
    const logoutBtn = wrapper.find('.btn-logout')
    expect(logoutBtn.exists()).toBe(true)

    await logoutBtn.trigger('click')
    expect(logoutSpy).toHaveBeenCalled()
  })

  it('emits toggle-menu when menu button is clicked', async () => {
    const wrapper = mount(AppTopbar)
    const menuBtn = wrapper.find('.btn-menu-toggle')
    expect(menuBtn.exists()).toBe(true)

    await menuBtn.trigger('click')
    expect(wrapper.emitted('toggle-menu')).toHaveLength(1)
  })

  it('toggles language when clicking the language quick switch button', async () => {
    setLanguage('it')
    const wrapper = mount(AppTopbar, {
      global: {
        plugins: [i18n]
      }
    })
    const langBtn = wrapper.find('.lang-quick-toggle')
    expect(langBtn.exists()).toBe(true)
    expect(langBtn.text()).toContain('IT')

    await langBtn.trigger('click')
    expect(langBtn.text()).toContain('EN')
  })
})
