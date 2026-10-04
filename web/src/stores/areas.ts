import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '../api/client'
import type { Area } from '../api/types'

export const useAreasStore = defineStore('areas', () => {
  const areas = ref<Area[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const sortedAreas = computed(() =>
    [...areas.value].sort((a, b) => (a.order_index ?? 0) - (b.order_index ?? 0))
  )

  const byId = (id: string): Area | undefined => areas.value.find((a) => a.id === id)

  async function fetchAreas(): Promise<Area[]> {
    try {
      loading.value = true
      error.value = null
      const data = await apiClient.get<Area[]>('/api/v1/areas')
      areas.value = data
      return data
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch areas'
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    areas,
    sortedAreas,
    loading,
    error,
    byId,
    fetchAreas
  }
})
