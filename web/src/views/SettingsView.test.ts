import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import SettingsView from './SettingsView.vue'
import { useAuthStore } from '@/stores/auth'
import { useSystemStore } from '@/stores/system'
import { i18n, getCurrentLanguage, setLanguage } from '@/locales'

describe('SettingsView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
    setLanguage('it')
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

  it('renders Domus OS hardware telemetry and maintenance mode card', async () => {
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

    const mockOS = {
      soc_temp_c: 44.8,
      cpu_load_1m: 0.15,
      cpu_load_5m: 0.20,
      cpu_load_15m: 0.10,
      cpu_cores: 4,
      memory_total_bytes: 4294967296,
      memory_available_bytes: 2147483648,
      memory_used_bytes: 2147483648,
      memory_used_percent: 50.0,
      disk_total_bytes: 34359738368,
      disk_free_bytes: 17179869184,
      disk_used_bytes: 17179869184,
      disk_used_percent: 50.0,
      hostname: 'domus',
      os_version: 'Domus OS 1.0.0',
      kernel_version: '6.6.28-v8',
      architecture: 'arm64',
      uptime_seconds: 3600,
      active_slot: 'domus-root-a',
      booted_slot: 'domus-root-a',
      slot_confirmed: true,
      rauc_status: 'good',
      core_healthy: true,
      running_containers: 2,
      total_containers: 2,
      installed_apps: 1,
      maintenance_mode: false,
      timestamp: '2026-10-04T12:00:00Z'
    }

    vi.spyOn(systemStore, 'fetchOSDiagnostics').mockImplementation(async () => {
      systemStore.osDiagnostics = mockOS
      return mockOS
    })

    vi.spyOn(systemStore, 'fetchMaintenanceStatus').mockImplementation(async () => {
      const res = { active: false, remaining_seconds: 0, ssh_port: 22 }
      systemStore.maintenanceStatus = res
      return res
    })

    const wrapper = mount(SettingsView, {
      global: {
        plugins: [i18n]
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Diagnostica Appliance Domus OS')
    expect(wrapper.text()).toContain('44.8 °C')
    expect(wrapper.text()).toContain('Modalità Manutenzione')
    expect(wrapper.text()).toContain('SSH Disattivato')
    expect(wrapper.find('.btn-export-diag').exists()).toBe(true)
  })
})
