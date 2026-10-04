import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { StateHistory } from '@/api/types'
import { apiClient } from '@/api/client'

export interface HistoryQueryOptions {
  from?: string
  to?: string
  limit?: number
}

export const useHistoryStore = defineStore('history', () => {
  const historyByEntity = ref<Record<string, StateHistory[]>>({})
  const loading = ref(false)
  const error = ref<string | null>(null)

  function getHistory(entityId: string): StateHistory[] {
    return historyByEntity.value[entityId] || []
  }

  async function fetchHistory(entityId: string, options?: HistoryQueryOptions): Promise<StateHistory[]> {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get<StateHistory[]>(`/entities/${encodeURIComponent(entityId)}/history`, {
        params: options
      })
      const list = Array.isArray(data) ? data : []
      historyByEntity.value[entityId] = list
      return list
    } catch (err: any) {
      error.value = err.message || 'Failed to fetch history'
      throw err
    } finally {
      loading.value = false
    }
  }

  function clearHistory(entityId?: string) {
    if (entityId) {
      delete historyByEntity.value[entityId]
    } else {
      historyByEntity.value = {}
    }
  }

  return {
    historyByEntity,
    loading,
    error,
    getHistory,
    fetchHistory,
    clearHistory
  }
})
