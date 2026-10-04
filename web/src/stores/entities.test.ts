import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useEntitiesStore } from './entities'
import { apiClient } from '../api/client'
import type { Entity } from '../api/types'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn()
  }
}))

describe('Entities Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const mockEntities: Entity[] = [
    {
      id: 'light.living_room',
      device_id: 'dev-1',
      domain: 'light',
      name: 'Living Room Light',
      state: 'off',
      attributes: { brightness: 0, color_temp: 300 },
      last_changed: '2026-10-03T18:00:00Z',
      last_updated: '2026-10-03T18:00:00Z'
    },
    {
      id: 'switch.coffee_maker',
      device_id: 'dev-2',
      domain: 'switch',
      name: 'Coffee Maker',
      state: 'off',
      attributes: {},
      last_changed: '2026-10-03T18:00:00Z',
      last_updated: '2026-10-03T18:00:00Z'
    },
    {
      id: 'sensor.living_room_temp',
      device_id: 'dev-1',
      domain: 'sensor',
      name: 'Living Room Temperature',
      state: '21.5',
      attributes: { unit_of_measurement: '°C', device_class: 'temperature' },
      last_changed: '2026-10-03T18:00:00Z',
      last_updated: '2026-10-03T18:00:00Z'
    },
    {
      id: 'binary_sensor.front_door',
      device_id: 'dev-3',
      domain: 'binary_sensor',
      name: 'Front Door',
      state: 'off',
      attributes: { device_class: 'door' },
      last_changed: '2026-10-03T18:00:00Z',
      last_updated: '2026-10-03T18:00:00Z'
    }
  ]

  it('fetches entities and maps them into state dictionary and categorized getters', async () => {
    const store = useEntitiesStore()
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockEntities)

    await store.fetchEntities()
    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/entities')
    expect(store.allEntities.length).toBe(4)
    expect(store.lights.length).toBe(1)
    expect(store.lights[0].id).toBe('light.living_room')
    expect(store.switches.length).toBe(1)
    expect(store.sensors.length).toBe(1)
    expect(store.binarySensors.length).toBe(1)
  })

  it('handles state_changed event and updates entity reactively', async () => {
    const store = useEntitiesStore()
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockEntities)
    await store.fetchEntities()

    store.handleStateChange({
      entity_id: 'light.living_room',
      old_state: 'off',
      new_state: 'on',
      old_attributes: { brightness: 0 },
      new_attributes: { brightness: 255 },
      timestamp: '2026-10-03T18:05:00Z'
    })

    const updated = store.byId('light.living_room')
    expect(updated).toBeDefined()
    expect(updated?.state).toBe('on')
    expect(updated?.attributes.brightness).toBe(255)
    expect(updated?.last_updated).toBe('2026-10-03T18:05:00Z')
  })

  it('performs optimistic update on callService', async () => {
    const store = useEntitiesStore()
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockEntities)
    await store.fetchEntities()

    vi.mocked(apiClient.post).mockResolvedValueOnce({ success: true })

    await store.callService('light', 'turn_on', 'light.living_room', { brightness: 200 })

    const updated = store.byId('light.living_room')
    expect(updated?.state).toBe('on')
    expect(updated?.attributes.brightness).toBe(200)

    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/entities/light.living_room/command', {
      domain: 'light',
      service: 'turn_on',
      data: { brightness: 200 }
    })
  })
})
