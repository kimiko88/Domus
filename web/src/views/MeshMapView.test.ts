import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import MeshMapView from './MeshMapView.vue'
import { apiClient } from '../api/client'

describe('MeshMapView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/mesh/topology')) {
        return {
          protocol: 'zigbee',
          nodes: [
            {
              id: 'coord_1',
              name: 'Coordinator Zigbee USB',
              protocol: 'zigbee',
              type: 'coordinator',
              lqi: 255
            },
            {
              id: 'router_1',
              name: 'Presa Smart Salotto',
              protocol: 'zigbee',
              type: 'router',
              lqi: 215
            }
          ],
          links: [
            { source: 'coord_1', target: 'router_1', lqi: 215, quality: 'excellent' }
          ]
        } as any
      }
      return null as any
    })
  })

  it('renders mesh diagnostics title and protocol buttons', async () => {
    const wrapper = mount(MeshMapView)
    await flushPromises()

    expect(wrapper.text()).toContain('Diagnostica Radio & Mappa Mesh')
    expect(wrapper.text()).toContain('Zigbee 3.0')
    expect(wrapper.text()).toContain('Z-Wave JS')
    expect(wrapper.text()).toContain('Matter / Thread')
  })

  it('renders SVG topology nodes and LQI indicators', async () => {
    const wrapper = mount(MeshMapView)
    await flushPromises()

    expect(wrapper.find('svg.mesh-svg').exists()).toBe(true)
    expect(wrapper.text()).toContain('LQI 215')
    expect(wrapper.text()).toContain('Coordinator')
  })
})
