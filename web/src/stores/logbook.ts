import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiClient } from '../api/client'
import type { LogbookEntry, LogbookFilter } from '../api/types'

export const useLogbookStore = defineStore('logbook', () => {
  const entries = ref<LogbookEntry[]>([])
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  async function fetchEntries(filter?: LogbookFilter): Promise<LogbookEntry[]> {
    loading.value = true
    error.value = null
    try {
      const params: Record<string, any> = {}
      if (filter?.entity_id) params.entity_id = filter.entity_id
      if (filter?.domain) params.domain = filter.domain
      if (filter?.from) params.from = filter.from
      if (filter?.to) params.to = filter.to
      if (filter?.limit) params.limit = filter.limit

      const data = await apiClient.get<{ entries: LogbookEntry[]; total: number }>('/api/v1/logbook', params)
      entries.value = data.entries || []
      return entries.value
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch logbook entries'
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    entries,
    loading,
    error,
    fetchEntries,
  }
})
