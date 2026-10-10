import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useFloorplanStore } from './floorplan'
import { apiClient } from '../api/client'
import type { Floor, FloorMarker } from '../api/types'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

describe('Floorplan Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const mockFloors: Floor[] = [
    { id: 'floor-0', name: 'Piano Terra', level: 0, sort_order: 0, created_at: '', updated_at: '' },
    { id: 'floor-1', name: 'Primo Piano', level: 1, sort_order: 1, created_at: '', updated_at: '' }
  ]

  const mockMarkers: FloorMarker[] = [
    { id: 'm-1', floor_id: 'floor-0', entity_key: 'light.living', x_percent: 25, y_percent: 50, show_badge: true, created_at: '', updated_at: '' }
  ]

  it('fetches floors and sets active floor', async () => {
    const store = useFloorplanStore()
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockFloors)

    await store.fetchFloors()
    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/floors')
    expect(store.floors.length).toBe(2)
    expect(store.activeFloorId).toBe('floor-0')
    expect(store.activeFloor?.name).toBe('Piano Terra')
  })

  it('fetches markers for a specific floor', async () => {
    const store = useFloorplanStore()
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockMarkers)

    const list = await store.fetchMarkers('floor-0')
    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/floors/floor-0/markers')
    expect(list.length).toBe(1)
    expect(store.markers['floor-0'].length).toBe(1)
  })

  it('saves and updates marker position', async () => {
    const store = useFloorplanStore()
    const savedMarker: FloorMarker = {
      id: 'm-2',
      floor_id: 'floor-0',
      entity_key: 'switch.plug',
      x_percent: 40,
      y_percent: 60,
      show_badge: true,
      created_at: '',
      updated_at: ''
    }
    vi.mocked(apiClient.put).mockResolvedValueOnce(savedMarker)

    const res = await store.saveMarker('floor-0', {
      entity_key: 'switch.plug',
      x_percent: 40,
      y_percent: 60,
      show_badge: true
    })
    expect(res.entity_key).toBe('switch.plug')
    expect(store.markers['floor-0']?.length).toBe(1)
  })
})
