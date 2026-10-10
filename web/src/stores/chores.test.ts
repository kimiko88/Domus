import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useChoresStore } from './chores'
import { apiClient } from '../api/client'
import type { Chore } from '../api/types'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

describe('Chores Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const mockChores: Chore[] = [
    {
      id: 'chore-1',
      title: 'Pulire filtri',
      status: 'pending',
      priority: 'high',
      recurrence_type: 'weekly',
      created_at: '',
      updated_at: ''
    },
    {
      id: 'chore-2',
      title: 'Lavare pavimenti',
      status: 'overdue',
      priority: 'urgent',
      recurrence_type: 'none',
      created_at: '',
      updated_at: ''
    },
    {
      id: 'chore-3',
      title: 'Annaffiare piante',
      status: 'completed',
      priority: 'low',
      recurrence_type: 'daily',
      created_at: '',
      updated_at: ''
    }
  ]

  it('fetches chores and filters pending/overdue/completed', async () => {
    const store = useChoresStore()
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockChores)

    await store.fetchChores()
    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/chores', undefined)
    expect(store.chores.length).toBe(3)
    expect(store.pendingChores.length).toBe(2) // pending + overdue
    expect(store.overdueChores.length).toBe(1)
    expect(store.completedChores.length).toBe(1)
  })

  it('creates a new chore and prepends to list', async () => {
    const store = useChoresStore()
    const newChore: Chore = {
      id: 'chore-4',
      title: 'Cambiare sacchetto aspirapolvere',
      status: 'pending',
      priority: 'medium',
      recurrence_type: 'none',
      created_at: '',
      updated_at: ''
    }
    vi.mocked(apiClient.post).mockResolvedValueOnce(newChore)

    const res = await store.createChore({
      title: 'Cambiare sacchetto aspirapolvere',
      priority: 'medium'
    })
    expect(res.id).toBe('chore-4')
    expect(store.chores[0].title).toBe('Cambiare sacchetto aspirapolvere')
  })

  it('completes a chore', async () => {
    const store = useChoresStore()
    store.chores = [...mockChores]
    vi.mocked(apiClient.post).mockResolvedValueOnce({})
    vi.mocked(apiClient.get).mockResolvedValueOnce([
      { ...mockChores[0], status: 'completed' }
    ])

    await store.completeChore('chore-1')
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/chores/chore-1/complete')
  })
})
