import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '../api/client'
import type { Area } from '../api/types'

export const useAreasStore = defineStore('areas', () => {
  const areas = ref<Area[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const sortedAreas = computed(() =>
    [...areas.value].sort((a, b) => ((a.sort_order ?? a.order_index ?? 0) - (b.sort_order ?? b.order_index ?? 0)))
  )

  const byId = (id: string): Area | undefined => areas.value.find((a) => a.id === id)

  async function fetchAreas(): Promise<Area[]> {
    try {
      loading.value = true
      error.value = null
      const raw = await apiClient.get<Area[] | null>('/api/v1/areas')
      const data = raw ?? []
      areas.value = data
      return data
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch areas'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createArea(payload: { name: string; icon?: string; sort_order?: number; parent_area_id?: string }): Promise<Area> {
    try {
      loading.value = true
      error.value = null
      const created = await apiClient.post<Area>('/api/v1/areas', payload)
      areas.value.push(created)
      return created
    } catch (err: any) {
      error.value = err?.message || 'Failed to create area'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateArea(id: string, payload: { name: string; icon?: string; sort_order?: number; parent_area_id?: string }): Promise<Area> {
    try {
      loading.value = true
      error.value = null
      const updated = await apiClient.put<Area>(`/api/v1/areas/${id}`, payload)
      const idx = areas.value.findIndex((a) => a.id === id)
      if (idx !== -1) {
        areas.value[idx] = updated
      }
      return updated
    } catch (err: any) {
      error.value = err?.message || 'Failed to update area'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteArea(id: string): Promise<void> {
    try {
      loading.value = true
      error.value = null
      await apiClient.delete(`/api/v1/areas/${id}`)
      areas.value = areas.value.filter((a) => a.id !== id)
    } catch (err: any) {
      error.value = err?.message || 'Failed to delete area'
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
    fetchAreas,
    createArea,
    updateArea,
    deleteArea
  }
})
