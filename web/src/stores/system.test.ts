import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSystemStore } from './system'
import { apiClient } from '@/api/client'

describe('useSystemStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('fetches system metrics and formats uptime correctly', async () => {
    const mockMetrics = {
      status: 'ok',
      uptime_s: 3665, // 1h 1m 5s
      entity_count: 14,
      device_count: 6,
      mqtt_connected: true,
      ws_clients: 2,
      db_size_bytes: 524288
    }

    vi.spyOn(apiClient, 'get').mockResolvedValueOnce(mockMetrics)

    const store = useSystemStore()
    const res = await store.fetchMetrics()

    expect(apiClient.get).toHaveBeenCalledWith('/system/metrics')
    expect(res.uptime_s).toBe(3665)
    expect(store.metrics).toEqual(mockMetrics)
    expect(store.formattedUptime).toBe('1h 1m 5s')
  })

  it('formats shorter uptime properly', async () => {
    const mockMetrics = {
      status: 'ok',
      uptime_s: 42,
      entity_count: 0,
      device_count: 0,
      mqtt_connected: false,
      ws_clients: 0,
      db_size_bytes: 0
    }

    vi.spyOn(apiClient, 'get').mockResolvedValueOnce(mockMetrics)

    const store = useSystemStore()
    await store.fetchMetrics()

    expect(store.formattedUptime).toBe('42s')
  })
})
