import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiClient } from '../api/client'
import type { AuditEntry, AuditListResponse } from '../api/types'

export const useAuditStore = defineStore('audit', () => {
  const entries = ref<AuditEntry[]>([])
  const total = ref<number>(0)
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  async function fetchAuditLogs(limit: number = 50): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const response = await apiClient.get<AuditListResponse>('/api/v1/audit', { limit })
      entries.value = response.entries || []
      total.value = response.total ?? entries.value.length
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch audit log'
      entries.value = []
    } finally {
      loading.value = false
    }
  }

  return {
    entries,
    total,
    loading,
    error,
    fetchAuditLogs
  }
})
