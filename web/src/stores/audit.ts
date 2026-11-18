import { defineStore } from 'pinia'
import i18n from '../locales'
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
      error.value = i18n.global.t('common.errors.loadFailed')
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
