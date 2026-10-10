import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '../api/client'
import type { Chore, ChorePriority, ChoreRecurrence } from '../api/types'

export const useChoresStore = defineStore('chores', () => {
  const chores = ref<Chore[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const pendingChores = computed(() =>
    chores.value.filter((c) => c.status === 'pending' || c.status === 'overdue')
  )

  const overdueChores = computed(() =>
    chores.value.filter((c) => c.status === 'overdue')
  )

  const completedChores = computed(() =>
    chores.value.filter((c) => c.status === 'completed')
  )

  async function fetchChores(filter?: { status?: string; user_id?: string; area_id?: string }): Promise<Chore[]> {
    try {
      loading.value = true
      error.value = null
      const raw = await apiClient.get<Chore[] | null>('/api/v1/chores', filter)
      const data = raw ?? []
      chores.value = data
      return data
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch chores'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createChore(payload: {
    title: string
    description?: string
    area_id?: string
    assigned_user_id?: string
    priority?: ChorePriority
    due_date?: string
    recurrence_type?: ChoreRecurrence
    recurrence_rule?: string
  }): Promise<Chore> {
    try {
      loading.value = true
      error.value = null
      const created = await apiClient.post<Chore>('/api/v1/chores', payload)
      chores.value.unshift(created)
      return created
    } catch (err: any) {
      error.value = err?.message || 'Failed to create chore'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateChore(id: string, payload: Partial<Chore>): Promise<Chore> {
    try {
      loading.value = true
      error.value = null
      const updated = await apiClient.put<Chore>(`/api/v1/chores/${id}`, payload)
      const idx = chores.value.findIndex((c) => c.id === id)
      if (idx !== -1) {
        chores.value[idx] = updated
      }
      return updated
    } catch (err: any) {
      error.value = err?.message || 'Failed to update chore'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function completeChore(id: string): Promise<void> {
    try {
      await apiClient.post(`/api/v1/chores/${id}/complete`)
      const chore = chores.value.find((c) => c.id === id)
      if (chore) {
        chore.status = 'completed'
        chore.last_completed_at = new Date().toISOString()
      }
      // Re-fetch to pull newly spawned recurring instance if any
      await fetchChores()
    } catch (err: any) {
      error.value = err?.message || 'Failed to complete chore'
      throw err
    }
  }

  async function deleteChore(id: string): Promise<void> {
    try {
      loading.value = true
      error.value = null
      await apiClient.delete(`/api/v1/chores/${id}`)
      chores.value = chores.value.filter((c) => c.id !== id)
    } catch (err: any) {
      error.value = err?.message || 'Failed to delete chore'
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    chores,
    loading,
    error,
    pendingChores,
    overdueChores,
    completedChores,
    fetchChores,
    createChore,
    updateChore,
    completeChore,
    deleteChore
  }
})
