import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import SettingsView from './SettingsView.vue'
import { useAuthStore } from '@/stores/auth'
import { useSystemStore } from '@/stores/system'
import { i18n, getCurrentLanguage } from '@/locales'

describe('SettingsView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('renders user info, live system metrics, and language options', async () => {
    const authStore = useAuthStore()
    authStore.user = {
      id: 'usr_admin',
      username: 'admin',
      role: 'admin',
      created_at: '',
      updated_at: ''
    }

    const mockMetrics = {
      status: 'ok',
      uptime_s: 7200,
      entity_count: 18,
      device_count: 7,
      mqtt_connected: true,
      ws_clients: 4,
      db_size_bytes: 1048576
    }

    const systemStore = useSystemStore()
    vi.spyOn(systemStore, 'fetchMetrics').mockImplementation(async () => {
      systemStore.metrics = mockMetrics
      return mockMetrics
    })

    const wrapper = mount(SettingsView, {
      global: {
        plugins: [i18n]
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('admin')
    expect(wrapper.text()).toContain('Lingua Applicazione')
    expect(wrapper.find('.language-selector').exists()).toBe(true)
    expect(wrapper.text()).toContain('Metriche Operative')
    expect(wrapper.text()).toContain('2h') // formatted uptime
  })

  it('switches application language when clicking a language button', async () => {
    const systemStore = useSystemStore()
    vi.spyOn(systemStore, 'fetchMetrics').mockResolvedValue({
      status: 'ok',
      uptime_s: 100,
      entity_count: 0,
      device_count: 0,
      mqtt_connected: false,
      ws_clients: 0,
      db_size_bytes: 0
    })

    const wrapper = mount(SettingsView, {
      global: {
        plugins: [i18n]
      }
    })

    const langBtns = wrapper.findAll('.lang-btn')
    expect(langBtns.length).toBeGreaterThanOrEqual(2)

    // Click English button
    const enBtn = langBtns.find(b => b.text().includes('English'))
    expect(enBtn).toBeDefined()
    await enBtn!.trigger('click')

    expect(getCurrentLanguage()).toBe('en')
    expect(wrapper.text()).toContain('Application Language')
  })
})
