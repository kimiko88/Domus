import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import DevicesView from './DevicesView.vue'
import { useDevicesStore } from '../stores/devices'
import { i18n } from '../locales'

describe('DevicesView.vue', () => {
  let devicesStore: ReturnType<typeof useDevicesStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    devicesStore = useDevicesStore()
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
