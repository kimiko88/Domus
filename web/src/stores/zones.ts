import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiClient } from '../api/client'
import type { SpatialZone, TrackedPerson } from '../api/types'

export const useZonesStore = defineStore('zones', () => {
  const zones = ref<SpatialZone[]>([])
  const persons = ref<TrackedPerson[]>([])
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  async function fetchZones(): Promise<SpatialZone[]> {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get<{ zones: SpatialZone[]; total: number }>('/api/v1/zones')
      zones.value = data.zones || []
      return zones.value
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch zones'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createZone(zone: Partial<SpatialZone>): Promise<SpatialZone> {
    loading.value = true
    error.value = null
    try {
      const created = await apiClient.post<SpatialZone>('/api/v1/zones', zone)
      zones.value.push(created)
      return created
    } catch (err: any) {
      error.value = err?.message || 'Failed to create zone'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateZone(id: string, zone: Partial<SpatialZone>): Promise<SpatialZone> {
    loading.value = true
    error.value = null
    try {
      const updated = await apiClient.put<SpatialZone>(`/api/v1/zones/${id}`, zone)
      const idx = zones.value.findIndex((z) => z.id === id)
      if (idx !== -1) zones.value[idx] = updated
      return updated
    } catch (err: any) {
      error.value = err?.message || 'Failed to update zone'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteZone(id: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await apiClient.delete(`/api/v1/zones/${id}`)
      zones.value = zones.value.filter((z) => z.id !== id)
    } catch (err: any) {
      error.value = err?.message || 'Failed to delete zone'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchPersons(): Promise<TrackedPerson[]> {
    try {
      const data = await apiClient.get<{ persons: TrackedPerson[]; total: number }>('/api/v1/persons')
      persons.value = data.persons || []
      return persons.value
    } catch {
      try {
        const data = await apiClient.get<{ persons: TrackedPerson[]; total: number }>('/api/v1/presence/persons')
        persons.value = data.persons || []
        return persons.value
      } catch {
        return []
      }
    }
  }

  return {
    zones,
    persons,
    loading,
    error,
    fetchZones,
    createZone,
    updateZone,
    deleteZone,
    fetchPersons,
  }
})
