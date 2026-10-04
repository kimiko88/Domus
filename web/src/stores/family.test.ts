import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useFamilyStore } from './family'
import { apiClient } from '../api/client'
import type { FamilyMember, FamilyInvite, CreateInviteResponse } from '../api/types'

describe('useFamilyStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('fetches family members and populates list', async () => {
    const mockMembers: FamilyMember[] = [
      {
        id: 'user-1',
        username: 'admin_parent',
        email: 'admin@domus.local',
        role: 'admin',
        enabled: true,
        created_at: '2026-10-04T00:00:00Z'
      },
      {
        id: 'user-2',
        username: 'child_viewer',
        email: 'child@domus.local',
        role: 'viewer',
        enabled: true,
        created_at: '2026-10-04T01:00:00Z'
      }
    ]

    vi.spyOn(apiClient, 'get').mockResolvedValue({
      members: mockMembers,
      total: 2
    })

    const store = useFamilyStore()
    await store.fetchMembers()

    expect(store.members).toHaveLength(2)
    expect(store.members[0].username).toBe('admin_parent')
    expect(store.loading).toBe(false)
  })

  it('fetches pending invites', async () => {
    const mockInvites: FamilyInvite[] = [
      {
        id: 'inv-1',
        token: 'token-abc',
        role: 'operator',
        expires_at: '2026-10-06T00:00:00Z',
        created_at: '2026-10-04T00:00:00Z'
      }
    ]

    vi.spyOn(apiClient, 'get').mockResolvedValue({
      invites: mockInvites,
      total: 1
    })

    const store = useFamilyStore()
    await store.fetchInvites()

    expect(store.invites).toHaveLength(1)
    expect(store.invites[0].role).toBe('operator')
  })

  it('creates an invite and refetches invite list', async () => {
    const mockResponse: CreateInviteResponse = {
      id: 'inv-2',
      token: 'token-xyz',
      role: 'operator',
      expires_at: '2026-10-06T00:00:00Z',
      invite_url: '/register?token=token-xyz'
    }

    vi.spyOn(apiClient, 'post').mockResolvedValue(mockResponse)
    const getSpy = vi.spyOn(apiClient, 'get').mockResolvedValue({
      invites: [mockResponse as any],
      total: 1
    })

    const store = useFamilyStore()
    const result = await store.createInvite('operator', 48)

    expect(result.token).toBe('token-xyz')
    expect(getSpy).toHaveBeenCalledWith('/api/v1/family/invites')
  })

  it('revokes an invite and removes it from store', async () => {
    vi.spyOn(apiClient, 'delete').mockResolvedValue(undefined)

    const store = useFamilyStore()
    store.invites = [
      {
        id: 'inv-to-remove',
        token: 'tok-1',
        role: 'viewer',
        expires_at: '2026-10-06T00:00:00Z'
      }
    ]

    await store.revokeInvite('inv-to-remove')
    expect(store.invites).toHaveLength(0)
  })

  it('updates a member role', async () => {
    vi.spyOn(apiClient, 'put').mockResolvedValue({ status: 'ok' })

    const store = useFamilyStore()
    store.members = [
      {
        id: 'user-kid',
        username: 'kid',
        role: 'viewer',
        enabled: true,
        created_at: '2026-10-04T00:00:00Z'
      }
    ]

    await store.updateMemberRole('user-kid', 'operator')
    expect(store.members[0].role).toBe('operator')
  })

  it('removes a family member', async () => {
    vi.spyOn(apiClient, 'delete').mockResolvedValue(undefined)

    const store = useFamilyStore()
    store.members = [
      {
        id: 'user-guest',
        username: 'guest',
        role: 'viewer',
        enabled: true,
        created_at: '2026-10-04T00:00:00Z'
      }
    ]

    await store.removeMember('user-guest')
    expect(store.members).toHaveLength(0)
  })
})
