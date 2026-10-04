import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuditStore } from './audit'
import { apiClient } from '../api/client'

describe('useAuditStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('initializes with empty entries and not loading', () => {
    const store = useAuditStore()
    expect(store.entries).toEqual([])
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
    expect(store.total).toBe(0)
  })

  it('fetches audit logs and populates state', async () => {
    const mockData = {
      entries: [
        {
          id: 1,
          actor_type: 'local_user',
          actor_id: 'admin_1',
          action: 'auth.login',
          target_type: 'user',
          target_id: 'admin_1',
          payload: { ip: '127.0.0.1' },
          created_at: '2026-10-04T12:00:00Z'
        }
      ],
      total: 1
    }

    vi.spyOn(apiClient, 'get').mockResolvedValueOnce(mockData)

    const store = useAuditStore()
    await store.fetchAuditLogs(50)

    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/audit', { limit: 50 })
    expect(store.entries).toEqual(mockData.entries)
    expect(store.total).toBe(1)
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('handles fetch errors properly', async () => {
    vi.spyOn(apiClient, 'get').mockRejectedValueOnce(new Error('Network error'))

    const store = useAuditStore()
    await store.fetchAuditLogs()

    expect(store.entries).toEqual([])
    expect(store.loading).toBe(false)
    expect(store.error).toBe('Network error')
  })
})
