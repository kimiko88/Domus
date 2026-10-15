import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ZonesView from './ZonesView.vue'
import { useZonesStore } from '@/stores/zones'

describe('ZonesView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('renders zones header, interactive map canvas, and zones list', async () => {
    const store = useZonesStore()
    vi.spyOn(store, 'fetchZones').mockResolvedValue([])
    vi.spyOn(store, 'fetchPersons').mockResolvedValue([])

    store.zones = [
      {
        id: 'zone-home',
        name: 'Casa',
        latitude: 41.9028,
        longitude: 12.4964,
        radius: 100,
        icon: '🏠',
        passive: false,
        created_at: new Date().toISOString(),
      },
    ]

    store.persons = [
      {
        id: 'person-mario',
        name: 'Mario Rossi',
        state: 'home',
        device_trackers: ['device_tracker.iphone_mario'],
        updated_at: new Date().toISOString(),
      },
    ]

    const wrapper = mount(ZonesView)

    expect(wrapper.text()).toContain('Zone Geografiche & Presenza')
    expect(wrapper.find('svg.radar-svg').exists()).toBe(true)
    expect(wrapper.text()).toContain('Casa')
    expect(wrapper.text()).toContain('Mario Rossi')
    expect(wrapper.text()).toContain('device_tracker.iphone_mario')
  })

  it('opens create zone modal when button is clicked', async () => {
    const store = useZonesStore()
    vi.spyOn(store, 'fetchZones').mockResolvedValue([])
    vi.spyOn(store, 'fetchPersons').mockResolvedValue([])

    const wrapper = mount(ZonesView)

    expect(wrapper.find('.modal-backdrop').exists()).toBe(false)
    const btn = wrapper.find('button[aria-label="Crea nuova zona"]')
    await btn.trigger('click')

    expect(wrapper.find('.modal-backdrop').exists()).toBe(true)
    expect(wrapper.text()).toContain('Crea Nuova Zona Geografica')
  })
})
