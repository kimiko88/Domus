import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAreasStore } from './areas'
import { apiClient } from '../api/client'
import type { Area } from '../api/types'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn()
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
})
