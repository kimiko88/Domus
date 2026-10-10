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
    expect(tabs.length).toBe(3)
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
})
