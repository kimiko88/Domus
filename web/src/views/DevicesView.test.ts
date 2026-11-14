import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import DevicesView from './DevicesView.vue'
import { apiClient } from '../api/client'
import { useDevicesStore } from '../stores/devices'
import { i18n } from '../locales'

describe('DevicesView.vue', () => {
  let devicesStore: ReturnType<typeof useDevicesStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    devicesStore = useDevicesStore()
    vi.spyOn(apiClient, 'get').mockResolvedValue([])
  })

  it('renders device list with status and integration info', async () => {
    devicesStore.devices = [
      {
        id: 'dev_1',
        name: 'Aqara Motion Sensor',
        integration_id: 'zigbee2mqtt',
        manufacturer: 'Xiaomi',
        model: 'RTCGQ11LM',
        status: 'online',
        created_at: '',
        updated_at: ''
      },
      {
        id: 'dev_2',
        name: 'Sonoff Smart Plug',
        integration_id: 'mqtt_generic',
        manufacturer: 'Sonoff',
        model: 'S26',
        status: 'offline',
        created_at: '',
        updated_at: ''
      }
    ]

    vi.spyOn(devicesStore, 'fetchDevices').mockResolvedValue(devicesStore.devices)

    const wrapper = mount(DevicesView, {
      global: { plugins: [i18n] }
    })
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Aqara Motion Sensor')
    expect(wrapper.text()).toContain('Sonoff Smart Plug')
    expect(wrapper.text()).toContain('Xiaomi')
    expect(wrapper.text()).toContain('zigbee2mqtt')
  })

  it('filters devices by status', async () => {
    devicesStore.devices = [
      {
        id: 'dev_1',
        name: 'Online Bulb',
        integration_id: 'mqtt_generic',
        status: 'online',
        created_at: '',
        updated_at: ''
      },
      {
        id: 'dev_2',
        name: 'Offline Switch',
        integration_id: 'mqtt_generic',
        status: 'offline',
        created_at: '',
        updated_at: ''
      }
    ]

    const wrapper = mount(DevicesView, {
      global: { plugins: [i18n] }
    })
    await wrapper.vm.$nextTick()

    const onlineFilter = wrapper.findAll('.filter-pill').find((p) => p.text().includes('Online'))
    expect(onlineFilter).toBeDefined()
    await onlineFilter!.trigger('click')

    expect(wrapper.text()).toContain('Online Bulb')
    expect(wrapper.text()).not.toContain('Offline Switch')
  })

  it('shows registry and Tuya Cloud devices together without duplicating LocalTuya devices', async () => {
    devicesStore.devices = [
      {
        id: 'tuya_device-1',
        external_id: 'device-1',
        name: 'Lampada soggiorno',
        integration_id: 'tuya_local',
        manufacturer: 'Tuya',
        status: 'offline',
        created_at: '',
        updated_at: ''
      },
      {
        id: 'zigbee-1',
        name: 'Sensore ingresso',
        integration_id: 'zigbee2mqtt',
        manufacturer: 'Aqara',
        status: 'online',
        created_at: '',
        updated_at: ''
      }
    ]
    vi.spyOn(devicesStore, 'fetchDevices').mockResolvedValue(devicesStore.devices)
    vi.mocked(apiClient.get).mockImplementation(async (url: string) => {
      if (url === '/api/v1/integrations/tuya-cloud/config') return { configured: true }
      if (url === '/api/v1/integrations/tuya-cloud/devices') {
        return {
          devices: [
            {
              id: 'device-1',
              name: 'Lampada soggiorno',
              product_name: 'Smart Bulb',
              online: true,
              local_configured: true,
              local_active: true,
              local_ip: '192.168.1.31'
            },
            {
              id: 'device-2',
              name: 'Presa studio',
              product_name: 'Smart Plug',
              online: false,
              local_configured: false
            }
          ]
        }
      }
      return []
    })

    const wrapper = mount(DevicesView, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 10))

    expect(wrapper.findAll('.device-card')).toHaveLength(3)
    expect(wrapper.text()).toContain('Lampada soggiorno')
    expect(wrapper.text()).toContain('Presa studio')
    expect(wrapper.text()).toContain('Sensore ingresso')
    expect(wrapper.text()).toContain('Tuya Cloud · LocalTuya')
    expect(wrapper.text()).toContain('192.168.1.31')

    await wrapper.find('#devices-search').setValue('Presa studio')
    expect(wrapper.findAll('.device-card')).toHaveLength(1)
    expect(wrapper.text()).toContain('Presa studio')
    expect(wrapper.text()).not.toContain('Sensore ingresso')

    await wrapper.find('#devices-search').setValue('')
    await wrapper.find('#devices-integration-filter').setValue('tuya_cloud')
    expect(wrapper.findAll('.device-card')).toHaveLength(1)
    expect(wrapper.text()).toContain('Presa studio')
    expect(wrapper.text()).not.toContain('Lampada soggiorno')
  })

  it('renders empty state without crashing when devices list is empty or null', async () => {
    devicesStore.devices = [] as any
    vi.spyOn(devicesStore, 'fetchDevices').mockResolvedValue([])

    const wrapper = mount(DevicesView, {
      global: { plugins: [i18n] }
    })
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.empty-state').exists()).toBe(true)
  })
})
