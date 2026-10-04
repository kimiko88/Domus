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

    vi.spyOn(apiClient, 'get').mockResolvedValue(mockIntegrations)

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

    vi.spyOn(apiClient, 'get').mockResolvedValue(mockIntegrations)
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ status: 'reconnecting' })

    const wrapper = mount(IntegrationsView)
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    const reconnectBtn = wrapper.find('.btn-reconnect')
    expect(reconnectBtn.exists()).toBe(true)

    await reconnectBtn.trigger('click')
    expect(postSpy).toHaveBeenCalledWith('/api/v1/integrations/int_mqtt/reconnect')
  })
})
