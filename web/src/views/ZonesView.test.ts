import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ZonesView from './ZonesView.vue'
import { useZonesStore } from '@/stores/zones'
import { apiClient } from '../api/client'

describe('ZonesView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/presence/vacation')) {
        return { enabled: false, jitter_minutes: 15 } as any
      }
      return null as any
    })
    vi.spyOn(apiClient, 'post').mockResolvedValue({ enabled: true, jitter_minutes: 15 } as any)
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
    await flushPromises()

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
    await flushPromises()

    const btn = wrapper.find('button[aria-label="Crea nuova zona"]')
    await btn.trigger('click')

    expect(wrapper.text()).toContain('Crea Nuova Zona Geografica')
  })

  it('renders Vacation Mode Simulator section', async () => {
    const store = useZonesStore()
    vi.spyOn(store, 'fetchZones').mockResolvedValue([])
    vi.spyOn(store, 'fetchPersons').mockResolvedValue([])

    const wrapper = mount(ZonesView)
    await flushPromises()

    expect(wrapper.text()).toContain('Simulatore di Presenza')
    expect(wrapper.text()).toContain('Modalità Vacanza')
    expect(wrapper.text()).toContain('jitter gaussiano')
    expect(wrapper.text()).toContain('Azioni Simulate Registrate')
    expect(wrapper.find('.vacation-toggle').exists()).toBe(true)
    expect(wrapper.find('.vacation-toggle[aria-checked="false"]').exists()).toBe(true)
  })

  it('toggles Vacation Mode on button click', async () => {
    const store = useZonesStore()
    vi.spyOn(store, 'fetchZones').mockResolvedValue([])
    vi.spyOn(store, 'fetchPersons').mockResolvedValue([])

    const wrapper = mount(ZonesView)
    await flushPromises()

    const toggleBtn = wrapper.find('.vacation-toggle')
    expect(toggleBtn.exists()).toBe(true)

    await toggleBtn.trigger('click')
    await flushPromises()

    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/presence/vacation', { enabled: true })
  })

  it('opens Add Simulated Action modal', async () => {
    const store = useZonesStore()
    vi.spyOn(store, 'fetchZones').mockResolvedValue([])
    vi.spyOn(store, 'fetchPersons').mockResolvedValue([])

    const wrapper = mount(ZonesView)
    await flushPromises()

    const addBtn = wrapper.find('button[aria-label="Aggiungi azione simulata"]')
    expect(addBtn.exists()).toBe(true)
    await addBtn.trigger('click')

    expect(wrapper.text()).toContain('Aggiungi Azione Simulata')
    expect(wrapper.text()).toContain('Entity ID')
    expect(wrapper.text()).toContain('Orario Tipico')
  })
})


