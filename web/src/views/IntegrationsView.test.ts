import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import IntegrationsView from './IntegrationsView.vue'
import { apiClient } from '../api/client'

describe('IntegrationsView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders integration cards with statuses', async () => {
    const mockIntegrations = [
      {
        id: 'int_mqtt',
        name: 'MQTT Generic Broker',
        type: 'mqtt_generic',
        enabled: true,
        status: 'connected',
        last_error: null
      },
      {
        id: 'int_z2m',
        name: 'Zigbee2MQTT',
        type: 'zigbee2mqtt',
        enabled: true,
        status: 'disconnected',
        last_error: 'Connection timeout'
      }
    ]

    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url === '/api/v1/integrations') return mockIntegrations
      if (url === '/api/v1/integrations/catalog') return []
      if (url === '/api/v1/integrations/discovery/devices') return []
      return []
    })

    const wrapper = mount(IntegrationsView)
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    expect(wrapper.text()).toContain('MQTT Generic Broker')
    expect(wrapper.text()).toContain('Zigbee2MQTT')
    expect(wrapper.text()).toContain('connected')
    expect(wrapper.text()).toContain('disconnected')
  })

  it('triggers reconnect call when reconnect button is clicked', async () => {
    const mockIntegrations = [
      {
        id: 'int_mqtt',
        name: 'MQTT Generic',
        type: 'mqtt_generic',
        enabled: true,
        status: 'disconnected',
        last_error: null
      }
    ]

    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url === '/api/v1/integrations') return mockIntegrations
      return []
    })
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ status: 'reconnecting' })

    const wrapper = mount(IntegrationsView)
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    const reconnectBtn = wrapper.find('.btn-reconnect')
    expect(reconnectBtn.exists()).toBe(true)

    await reconnectBtn.trigger('click')
    expect(postSpy).toHaveBeenCalledWith('/api/v1/integrations/int_mqtt/reconnect')
  })

  it('switches to LAN discovery tab and displays discovered brand devices ready for 1-click pairing', async () => {
    const mockDiscovered = [
      {
        id: 'shelly_aabbcc',
        brand: 'shelly',
        name: 'Shelly Plus 1PM Cucina',
        ip: '192.168.1.145',
        port: 80,
        model: 'SNSW-001P16EU',
        firmware: '1.4.2',
        status: 'discovered',
        capabilities: ['switch', 'power_meter'],
        suggested_entities: [
          { domain: 'switch', name: 'Interruttore', key_suffix: 'relay_0', writable: true }
        ],
        discovered_at: new Date().toISOString()
      }
    ]

    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url === '/api/v1/integrations/discovery/devices') return mockDiscovered
      return []
    })
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ status: 'paired' })

    const wrapper = mount(IntegrationsView)
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    // Switch to discovery tab
    const tabs = wrapper.findAll('.tab-btn')
    expect(tabs.length).toBe(5)
    await tabs[1].trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Shelly Plus 1PM Cucina')
    expect(wrapper.text()).toContain('192.168.1.145:80')
    expect(wrapper.text()).toContain('Associa con 1-Click')

    const pairBtn = wrapper.find('.btn-pair')
    expect(pairBtn.exists()).toBe(true)
    await pairBtn.trigger('click')

    expect(postSpy).toHaveBeenCalledWith('/api/v1/integrations/discovery/pair', {
      brand: 'shelly',
      ip: '192.168.1.145',
      port: 80,
      name: 'Shelly Plus 1PM Cucina'
    })
  })

  it('switches to brand catalog tab and renders supported smart home brands', async () => {
    const mockCatalog = [
      {
        id: 'shelly',
        name: 'Shelly (Allterco)',
        category: 'Relè & Prese',
        protocol: 'HTTP RPC',
        zero_cloud: true,
        default_port: 80,
        description: 'Controllo 100% locale senza cloud',
        supported_models: ['Shelly Plus 1PM', 'Shelly Pro 4PM'],
        pairing_guide: 'Collega al Wi-Fi locale',
        icon: 'zap'
      }
    ]

    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url === '/api/v1/integrations/catalog') return mockCatalog
      return []
    })

    const wrapper = mount(IntegrationsView)
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    const tabs = wrapper.findAll('.tab-btn')
    await tabs[2].trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Shelly (Allterco)')
    expect(wrapper.text()).toContain('Zero-Cloud')
    expect(wrapper.text()).toContain('Shelly Plus 1PM')
  })

  it('switches to LAN & IoT Guardian tab and allows device quarantine', async () => {
    const mockHosts = [
      {
        ip: '192.168.1.55',
        mac: '34:94:54:11:22:33',
        vendor: 'Shelly (Allterco)',
        suggested_plugin: 'mqtt_generic',
        confidence: 0.85,
        open_ports: [80, 1883],
        is_iot: true
      }
    ]

    const mockLeaks = {
      count: 1,
      leaks: [
        {
          id: 'leak_1',
          source_ip: '192.168.1.55',
          source_mac: '34:94:54:11:22:33',
          device_name: 'Shelly Plug Infiltrato',
          destination_ip: '104.26.4.12',
          port: 1883,
          protocol: 'tcp',
          detected_at: new Date().toISOString(),
          reason: 'Isolated IoT node attempted outbound WAN connection'
        }
      ],
      isolated_macs: []
    }

    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url === '/api/v1/network/devices') return mockHosts
      if (url === '/api/v1/network/leaks') return mockLeaks
      return []
    })
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ status: 'quarantined' })

    const wrapper = mount(IntegrationsView)
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    const tabs = wrapper.findAll('.tab-btn')
    // 4th tab is Guardian
    await tabs[3].trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('IoT Network Guardian & Confinamento LAN')
    expect(wrapper.text()).toContain('Shelly Plug Infiltrato')
    expect(wrapper.text()).toContain('104.26.4.12:1883')
    expect(wrapper.text()).toContain('34:94:54:11:22:33')

    // Click on Quarantine button
    const quarantineBtn = wrapper.find('.btn-quarantine')
    expect(quarantineBtn.exists()).toBe(true)
    await quarantineBtn.trigger('click')

    expect(postSpy).toHaveBeenCalledWith('/api/v1/network/quarantine', expect.objectContaining({
      mac: '34:94:54:11:22:33',
      ip: '192.168.1.55'
    }))
  })
  it('loads Smart Life Cloud devices and toggles boolean datapoints', async () => {
    const cloudDevices = [{
      id: 'tuya-device-1',
      name: 'Lampada soggiorno',
      product_name: 'Wi-Fi Smart Bulb',
      online: true,
      status: [{ code: 'switch_led', value: false }]
    }]
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url === '/api/v1/integrations/tuya-cloud/config') {
        return { configured: true, client_id: 'test-cid', uid: 'test-uid', endpoint: 'https://openapi.tuyaeu.com' }
      }
      if (url === '/api/v1/integrations/tuya-oauth/status') return { connected: false }
      if (url === '/api/v1/integrations/tuya-cloud/devices') return { devices: cloudDevices }
      return []
    })
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ success: true })

    const wrapper = mount(IntegrationsView)
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))
    const tabs = wrapper.findAll('.tab-btn')
    await tabs[4].trigger('click')
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    // The Cloud tab exposes devices behind the direct server-to-server connection
    const refreshDevices = wrapper.findAll('.btn-scan').find((button) => button.text().includes('Aggiorna dispositivi'))
    expect(refreshDevices).toBeDefined()
    await refreshDevices!.trigger('click')
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    expect(wrapper.text()).toContain('Lampada soggiorno')
    expect(wrapper.text()).toContain('switch_led')
    const toggle = wrapper.find('.btn-host-probe')
    expect(toggle.exists()).toBe(true)
    await toggle.trigger('click')
    expect(postSpy).toHaveBeenCalledWith(
      '/api/v1/integrations/tuya-cloud/devices/tuya-device-1/commands',
      { commands: [{ code: 'switch_led', value: true }] }
    )
  })

  it('triggers local LAN sync and sends commands directly via local LAN when local_configured is true', async () => {
    const cloudDevices = [{
      id: 'tuya-local-dev-1',
      name: 'Presa Studio',
      product_name: 'Smart Socket',
      online: true,
      local_configured: true,
      local_active: true,
      local_ip: '192.168.1.120',
      status: [{ code: 'switch_1', value: false }]
    }]
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url === '/api/v1/integrations/tuya-cloud/config') {
        return { configured: true, client_id: 'test-cid', uid: 'test-uid', endpoint: 'https://openapi.tuyaeu.com' }
      }
      if (url === '/api/v1/integrations/tuya-oauth/status') return { connected: false }
      if (url === '/api/v1/integrations/tuya-cloud/devices') return { devices: cloudDevices }
      return []
    })
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ success: true, synced_count: 1 })

    const wrapper = mount(IntegrationsView)
    await wrapper.vm.$nextTick()
    const tabs = wrapper.findAll('.tab-btn')
    await tabs[4].trigger('click')
    await wrapper.vm.$nextTick()

    // Find and click sync button
    const syncBtn = wrapper.findAll('button').find((button) => button.text().includes('Applica selezione'))
    expect(syncBtn).toBeDefined()
    await syncBtn!.trigger('click')
    expect(postSpy).toHaveBeenCalledWith('/api/v1/integrations/tuya-cloud/sync-local', { device_ids: ['tuya-local-dev-1'] })

    // Now click the toggle on the locally configured device
    const toggle = wrapper.find('.btn-host-probe')
    expect(toggle.exists()).toBe(true)
    await toggle.trigger('click')
    // Must route to LAN adapter endpoint
    expect(postSpy).toHaveBeenCalledWith(
      '/api/v1/integrations/tuya-local/devices/tuya-local-dev-1/command',
      { command: 'turn_on' }
    )
    expect(wrapper.text()).toContain('LAN: 192.168.1.120')
    expect(wrapper.text()).toContain('LAN Locale')
  })

  it('allows saving Tuya configuration directly from the UI without OAuth redirect', async () => {
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url === '/api/v1/integrations/tuya-cloud/config') {
        return { configured: false, client_id: '', uid: '', endpoint: 'https://openapi.tuyaeu.com' }
      }
      return []
    })
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({
      success: true,
      message: 'Configurazione Tuya Smart Life salvata con successo'
    })

    const wrapper = mount(IntegrationsView)
    await wrapper.vm.$nextTick()
    const tabs = wrapper.findAll('.tab-btn')
    await tabs[4].trigger('click')
    await wrapper.vm.$nextTick()

    // Form must be visible when not configured
    expect(wrapper.text()).toContain('Come collegare Smart Life a Domus (Senza OAuth 2.0)')
    expect(wrapper.find('#tuya-client-id').exists()).toBe(true)
    expect(wrapper.find('#tuya-client-secret').exists()).toBe(true)
    expect(wrapper.find('#tuya-uid').exists()).toBe(true)

    await wrapper.find('#tuya-client-id').setValue('my-access-id')
    await wrapper.find('#tuya-client-secret').setValue('my-secret-key')
    await wrapper.find('#tuya-uid').setValue('eu123456789')
    await wrapper.find('form.tuya-form-grid').trigger('submit')

    expect(postSpy).toHaveBeenCalledWith('/api/v1/integrations/tuya-cloud/config', {
      client_id: 'my-access-id',
      client_secret: 'my-secret-key',
      uid: 'eu123456789',
      endpoint: 'https://openapi.tuyaeu.com'
    })
  })


  it('filters online and offline Tuya devices, selects devices and loads the API function set', async () => {
    const cloudDevices = [
      { id: 'tuya-online-1', name: 'Presa online', category: 'cz', online: true, status: [] },
      { id: 'tuya-offline-1', name: 'Presa offline', category: 'cz', online: false, status: [] }
    ]
    const getSpy = vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url === '/api/v1/integrations/tuya-cloud/config') {
        return { configured: true, client_id: 'test-cid', uid: 'test-uid', endpoint: 'https://openapi.tuyaeu.com' }
      }
      if (url === '/api/v1/integrations/tuya-cloud/devices') return { devices: cloudDevices }
      if (url.endsWith('/tuya-online-1/functions')) {
        return { functions: [{ code: 'switch_led', name: 'Interruttore', desc: 'Accensione luce', type: 'Boolean', values: '{}' }] }
      }
      return []
    })
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ success: true, synced_count: 1 })
    const wrapper = mount(IntegrationsView)
    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 10))
    await wrapper.findAll('.tab-btn')[4].trigger('click')
    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 10))

    expect(wrapper.text()).toContain('Presa online')
    expect(wrapper.text()).toContain('Presa offline')
    await wrapper.find('.tuya-device-filters button:nth-child(3)').trigger('click')
    expect(wrapper.text()).toContain('Presa offline')
    expect(wrapper.text()).not.toContain('Presa online')
    await wrapper.find('.tuya-device-filters button:nth-child(1)').trigger('click')

    const checkbox = wrapper.find('.tuya-device-selection input[type="checkbox"]')
    await checkbox.setValue(true)
    const syncButton = wrapper.findAll('button').find(button => button.text().includes('Applica selezione'))
    expect(syncButton).toBeDefined()
    await syncButton!.trigger('click')
    expect(postSpy).toHaveBeenCalledWith('/api/v1/integrations/tuya-cloud/sync-local', { device_ids: ['tuya-online-1'] })

    const functionsButton = wrapper.findAll('button').find(button => button.text().includes('Comandi disponibili via API'))
    expect(functionsButton).toBeDefined()
    await functionsButton!.trigger('click')
    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(getSpy).toHaveBeenCalledWith('/api/v1/integrations/tuya-cloud/devices/tuya-online-1/functions')
    expect(wrapper.text()).toContain('Accensione luce')
    expect(wrapper.text()).toContain('switch_led')
  })

  it('filters Smart Life devices by selected and unselected LocalTuya state', async () => {
    const cloudDevices = [
      {
        id: 'tuya-selected-1',
        name: 'Presa già selezionata',
        online: true,
        local_configured: true,
        status: []
      },
      {
        id: 'tuya-unselected-1',
        name: 'Lampada non selezionata',
        online: false,
        local_configured: false,
        status: []
      }
    ]
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url === '/api/v1/integrations/tuya-cloud/config') {
        return { configured: true, client_id: 'test-cid', uid: 'test-uid', endpoint: 'https://openapi.tuyaeu.com' }
      }
      if (url === '/api/v1/integrations/tuya-cloud/devices') return { devices: cloudDevices }
      return []
    })

    const wrapper = mount(IntegrationsView)
    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 10))
    await wrapper.findAll('.tab-btn')[4].trigger('click')
    await wrapper.vm.$nextTick()

    await wrapper.find('[data-filter="selected"]').trigger('click')
    expect(wrapper.text()).toContain('Presa già selezionata')
    expect(wrapper.text()).not.toContain('Lampada non selezionata')

    await wrapper.find('[data-filter="unselected"]').trigger('click')
    expect(wrapper.text()).toContain('Lampada non selezionata')
    expect(wrapper.text()).not.toContain('Presa già selezionata')

    await wrapper.find('[data-filter="all"]').trigger('click')
    await wrapper.find('#tuya-device-search').setValue('Presa')
    expect(wrapper.text()).toContain('Presa già selezionata')
    expect(wrapper.text()).not.toContain('Lampada non selezionata')
  })

  it('exposes LocalTuya thermostat DP4 automatic/manual and target temperature controls', async () => {
    const cloudDevices = [{
      id: 'tuya-thermostat-1',
      name: 'Termostato camera',
      product_name: 'Wi-Fi Thermostat',
      category: 'wk',
      online: true,
      local_configured: true,
      local_active: true,
      local_ip: '192.168.1.88',
      status: [{ code: 'temp_set', value: 215 }, { code: 'temp_current', value: 205 }]
    }]
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url === '/api/v1/integrations/tuya-cloud/config') {
        return { configured: true, client_id: 'test-cid', uid: 'test-uid', endpoint: 'https://openapi.tuyaeu.com' }
      }
      if (url === '/api/v1/integrations/tuya-cloud/devices') return { devices: cloudDevices }
      return []
    })
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ success: true })
    const wrapper = mount(IntegrationsView)
    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 10))
    await wrapper.findAll('.tab-btn')[4].trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Automatico · DP4 = 0')
    expect(wrapper.text()).toContain('Manuale · DP4 = 1')
    const autoButton = wrapper.findAll('button').find(button => button.text().includes('Automatico · DP4 = 0'))
    const manualButton = wrapper.findAll('button').find(button => button.text().includes('Manuale · DP4 = 1'))
    expect(autoButton).toBeDefined()
    expect(manualButton).toBeDefined()
    await autoButton!.trigger('click')
    expect(postSpy).toHaveBeenCalledWith(
      '/api/v1/integrations/tuya-local/devices/tuya-thermostat-1/command',
      { command: 'set_dp', params: { dp_id: 4, value: 0 } }
    )
    await manualButton!.trigger('click')
    expect(postSpy).toHaveBeenCalledWith(
      '/api/v1/integrations/tuya-local/devices/tuya-thermostat-1/command',
      { command: 'set_dp', params: { dp_id: 4, value: 1 } }
    )

    const temperature = wrapper.find('#tuya-temp-tuya-thermostat-1')
    await temperature.setValue('22.5')
    const setTemperature = wrapper.findAll('button').find(button => button.text().includes('Imposta temperatura'))
    await setTemperature!.trigger('click')
    expect(postSpy).toHaveBeenCalledWith(
      '/api/v1/integrations/tuya-local/devices/tuya-thermostat-1/command',
      { command: 'set_temperature', params: { temperature: 22.5 } }
    )
  })


  it('sends discovered supported thermostat commands through LocalTuya when cloud reports the device offline', async () => {
    const cloudDevices = [{
      id: 'tuya-offline-thermostat',
      name: 'Termostato locale',
      category: 'wk',
      online: false,
      local_configured: true,
      local_active: true,
      local_ip: '192.168.1.90',
      status: [{ code: 'temp_set', value: 215 }]
    }]
    const getSpy = vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url === '/api/v1/integrations/tuya-cloud/config') {
        return { configured: true, client_id: 'test-cid', uid: 'test-uid', endpoint: 'https://openapi.tuyaeu.com' }
      }
      if (url === '/api/v1/integrations/tuya-cloud/devices') return { devices: cloudDevices }
      if (url.endsWith('/tuya-offline-thermostat/functions')) {
        return {
          functions: [{
            code: 'temp_set',
            name: 'Temperatura obiettivo',
            type: 'Integer',
            values: '{"min":50,"max":350,"scale":1}'
          }]
        }
      }
      return []
    })
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ success: true })
    const wrapper = mount(IntegrationsView)
    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 10))
    await wrapper.findAll('.tab-btn')[4].trigger('click')
    await wrapper.vm.$nextTick()

    const functionsButton = wrapper.findAll('button').find(button => button.text().includes('Comandi disponibili via API'))
    await functionsButton!.trigger('click')
    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(getSpy).toHaveBeenCalledWith('/api/v1/integrations/tuya-cloud/devices/tuya-offline-thermostat/functions')

    const temperatureInput = wrapper.find('input[aria-label="Valore per Temperatura obiettivo"]')
    await temperatureInput.setValue('22.5')
    const executeButton = wrapper.findAll('button').find(button => button.text().includes('Invia'))
    expect(executeButton).toBeDefined()
    await executeButton!.trigger('click')
    expect(postSpy).toHaveBeenCalledWith(
      '/api/v1/integrations/tuya-local/devices/tuya-offline-thermostat/command',
      { command: 'set_temperature', params: { temperature: 22.5 } }
    )
    expect(postSpy).not.toHaveBeenCalledWith(
      '/api/v1/integrations/tuya-cloud/devices/tuya-offline-thermostat/commands',
      expect.anything()
    )
  })

})

