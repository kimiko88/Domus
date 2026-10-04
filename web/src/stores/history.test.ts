import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useHistoryStore } from './history'
import { apiClient } from '@/api/client'

describe('useHistoryStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('fetches and caches entity history records', async () => {
    const mockData = [
      { id: 1, entity_id: 'sensor.temp', state: '20.5', recorded_at: '2026-10-04T10:00:00Z' },
      { id: 2, entity_id: 'sensor.temp', state: '21.0', recorded_at: '2026-10-04T10:05:00Z' }
    ]

    vi.spyOn(apiClient, 'get').mockResolvedValueOnce(mockData)

    const store = useHistoryStore()
    const result = await store.fetchHistory('sensor.temp')

    expect(apiClient.get).toHaveBeenCalledWith('/entities/sensor.temp/history', { params: undefined })
    expect(result).toHaveLength(2)
    expect(store.historyByEntity['sensor.temp']).toHaveLength(2)
    expect(store.getHistory('sensor.temp')).toHaveLength(2)
  })

  it('supports from, to, and limit filter parameters', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValueOnce([])

    const store = useHistoryStore()
    await store.fetchHistory('sensor.temp', {
      from: '2026-10-04T00:00:00Z',
      to: '2026-10-04T12:00:00Z',
      limit: 50
    })

    expect(apiClient.get).toHaveBeenCalledWith('/entities/sensor.temp/history', {
      params: {
        from: '2026-10-04T00:00:00Z',
        to: '2026-10-04T12:00:00Z',
        limit: 50
      }
    })
  })

  it('handles fetch errors cleanly', async () => {
    vi.spyOn(apiClient, 'get').mockRejectedValueOnce(new Error('Network error'))

    const store = useHistoryStore()
    await expect(store.fetchHistory('sensor.temp')).rejects.toThrow('Network error')
    expect(store.error).toBe('Network error')
    expect(store.loading).toBe(false)
  })
})
