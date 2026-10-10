import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiClient } from '../api/client'
import type { FamilyMember, FamilyInvite, CreateInviteResponse, UserRole } from '../api/types'

export const useFamilyStore = defineStore('family', () => {
  const members = ref<FamilyMember[]>([])
  const invites = ref<FamilyInvite[]>([])
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  async function fetchMembers(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get<{ members: FamilyMember[], total: number }>('/api/v1/family/members')
      members.value = data.members || []
    } catch (err: any) {
      error.value = err?.message || 'Failed to load family members'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchInvites(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get<{ invites: FamilyInvite[], total: number }>('/api/v1/family/invites')
      invites.value = data.invites || []
    } catch (err: any) {
      error.value = err?.message || 'Failed to load invitations'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createInvite(role: UserRole, expiresInHours: number = 48): Promise<CreateInviteResponse> {
    loading.value = true
    error.value = null
    try {
      const resp = await apiClient.post<CreateInviteResponse>('/api/v1/family/invites', {
        role,
        expires_in_hours: expiresInHours
      })
      await fetchInvites()
      return resp
    } catch (err: any) {
      error.value = err?.message || 'Failed to create invite'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function revokeInvite(id: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await apiClient.delete(`/api/v1/family/invites/${encodeURIComponent(id)}`)
      invites.value = invites.value.filter(i => i.id !== id)
    } catch (err: any) {
      error.value = err?.message || 'Failed to revoke invite'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateMemberRole(id: string, role: UserRole): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await apiClient.put(`/api/v1/family/members/${encodeURIComponent(id)}/role`, { role })
      const m = members.value.find(member => member.id === id)
      if (m) {
        m.role = role
      }
    } catch (err: any) {
      error.value = err?.message || 'Failed to update member role'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function removeMember(id: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await apiClient.delete(`/api/v1/family/members/${encodeURIComponent(id)}`)
      members.value = members.value.filter(m => m.id !== id)
    } catch (err: any) {
      error.value = err?.message || 'Failed to remove member'
      throw err
    } finally {
      loading.value = false
    }
  }

  const guests = ref<FamilyMember[]>([])
  const guestLoading = ref<boolean>(false)

  async function fetchGuests(): Promise<void> {
    guestLoading.value = true
    try {
      const data = await apiClient.get<{ guests: FamilyMember[], total: number }>('/api/v1/family/guests')
      guests.value = data.guests || []
    } catch {
      // Graceful fallback for guests
    } finally {
      guestLoading.value = false
    }
  }

  async function createGuest(username: string, pinCode: string, validHours: number = 24, kioskMode: boolean = false): Promise<FamilyMember> {
    guestLoading.value = true
    try {
      const resp = await apiClient.post<FamilyMember>('/api/v1/family/guests', {
        username,
        pin_code: pinCode,
        valid_hours: validHours,
        kiosk_mode: kioskMode
      })
      await fetchGuests()
      return resp
    } catch (err: any) {
      error.value = err?.message || 'Failed to create guest'
      throw err
    } finally {
      guestLoading.value = false
    }
  }

  return {
    members,
    invites,
    guests,
    guestLoading,
    loading,
    error,
    fetchMembers,
    fetchInvites,
    fetchGuests,
    createGuest,
    createInvite,
    revokeInvite,
    updateMemberRole,
    removeMember
  }
})
