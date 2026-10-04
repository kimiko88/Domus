import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAreasStore } from './areas'
import { apiClient } from '../api/client'
import type { Area } from '../api/types'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

describe('Areas Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const mockAreas: Area[] = [
    { id: 'area-1', name: 'Living Room', icon: 'sofa', floor: 1, order_index: 0, created_at: '', updated_at: '' },
    { id: 'area-2', name: 'Kitchen', icon: 'utensils', floor: 1, order_index: 1, created_at: '', updated_at: '' }
  ]

  it('fetches areas and sorts them by order_index', async () => {
    const store = useAreasStore()
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockAreas)

    await store.fetchAreas()
    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/areas')
    expect(store.areas.length).toBe(2)
    expect(store.areas[0].name).toBe('Living Room')
    expect(store.byId('area-1')?.name).toBe('Living Room')
  })

  it('creates an area and appends it to store', async () => {
    const store = useAreasStore()
    const newArea: Area = { id: 'area-3', name: 'Garden', icon: 'trees', sort_order: 2, created_at: '', updated_at: '' }
    vi.mocked(apiClient.post).mockResolvedValueOnce(newArea)

    const result = await store.createArea({ name: 'Garden', icon: 'trees', sort_order: 2 })
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/areas', { name: 'Garden', icon: 'trees', sort_order: 2 })
    expect(result.id).toBe('area-3')
    expect(store.areas.length).toBe(1)
    expect(store.areas[0].name).toBe('Garden')
  })

  it('updates an existing area', async () => {
    const store = useAreasStore()
    store.areas = [...mockAreas]
    const updatedArea: Area = { ...mockAreas[0], name: 'Big Living Room' }
    vi.mocked(apiClient.put).mockResolvedValueOnce(updatedArea)

    const result = await store.updateArea('area-1', { name: 'Big Living Room' })
    expect(apiClient.put).toHaveBeenCalledWith('/api/v1/areas/area-1', { name: 'Big Living Room' })
    expect(result.name).toBe('Big Living Room')
    expect(store.byId('area-1')?.name).toBe('Big Living Room')
  })

  it('deletes an area from store', async () => {
    const store = useAreasStore()
    store.areas = [...mockAreas]
    vi.mocked(apiClient.delete).mockResolvedValueOnce({})

    await store.deleteArea('area-1')
    expect(apiClient.delete).toHaveBeenCalledWith('/api/v1/areas/area-1')
    expect(store.areas.length).toBe(1)
    expect(store.byId('area-1')).toBeUndefined()
  })
})
