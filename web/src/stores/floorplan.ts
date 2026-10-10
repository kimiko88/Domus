import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '../api/client'
import type { Floor, FloorMarker } from '../api/types'

export const useFloorplanStore = defineStore('floorplan', () => {
  const floors = ref<Floor[]>([])
  const activeFloorId = ref<string | null>(null)
  const markers = ref<Record<string, FloorMarker[]>>({}) // floor_id -> markers
  const loading = ref(false)
  const error = ref<string | null>(null)

  const sortedFloors = computed(() =>
    [...floors.value].sort((a, b) => a.level - b.level || a.sort_order - b.sort_order)
  )

  const activeFloor = computed(() =>
    floors.value.find((f) => f.id === activeFloorId.value) ?? sortedFloors.value[0] ?? null
  )

  const currentMarkers = computed(() => {
    if (!activeFloor.value) return []
    return markers.value[activeFloor.value.id] ?? []
  })

  async function fetchFloors(): Promise<Floor[]> {
    try {
      loading.value = true
      error.value = null
      const raw = await apiClient.get<Floor[] | null>('/api/v1/floors')
      const data = raw ?? []
      floors.value = data
      if (data.length > 0 && !activeFloorId.value) {
        activeFloorId.value = data[0].id
      }
      return data
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch floors'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchMarkers(floorId: string): Promise<FloorMarker[]> {
    try {
      const raw = await apiClient.get<FloorMarker[] | null>(`/api/v1/floors/${floorId}/markers`)
      const list = raw ?? []
      markers.value[floorId] = list
      return list
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch markers'
      throw err
    }
  }

  async function createFloor(payload: { name: string; level: number; plan_svg?: string; plan_image?: string; sort_order?: number }): Promise<Floor> {
    try {
      loading.value = true
      error.value = null
      const created = await apiClient.post<Floor>('/api/v1/floors', payload)
      floors.value.push(created)
      activeFloorId.value = created.id
      return created
    } catch (err: any) {
      error.value = err?.message || 'Failed to create floor'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateFloor(id: string, payload: { name: string; level: number; plan_svg?: string; plan_image?: string; sort_order?: number }): Promise<Floor> {
    try {
      loading.value = true
      error.value = null
      const updated = await apiClient.put<Floor>(`/api/v1/floors/${id}`, payload)
      const idx = floors.value.findIndex((f) => f.id === id)
      if (idx !== -1) {
        floors.value[idx] = updated
      }
      return updated
    } catch (err: any) {
      error.value = err?.message || 'Failed to update floor'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteFloor(id: string): Promise<void> {
    try {
      loading.value = true
      error.value = null
      await apiClient.delete(`/api/v1/floors/${id}`)
      floors.value = floors.value.filter((f) => f.id !== id)
      delete markers.value[id]
      if (activeFloorId.value === id) {
        activeFloorId.value = floors.value[0]?.id ?? null
      }
    } catch (err: any) {
      error.value = err?.message || 'Failed to delete floor'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function saveMarker(floorId: string, marker: Partial<FloorMarker> & { entity_key: string; x_percent: number; y_percent: number }): Promise<FloorMarker> {
    try {
      const saved = await apiClient.put<FloorMarker>(`/api/v1/floors/${floorId}/markers`, marker)
      const list = markers.value[floorId] ?? []
      const idx = list.findIndex((m) => m.entity_key === marker.entity_key)
      if (idx !== -1) {
        list[idx] = saved
      } else {
        list.push(saved)
      }
      markers.value[floorId] = [...list]
      return saved
    } catch (err: any) {
      error.value = err?.message || 'Failed to save marker'
      throw err
    }
  }

  async function deleteMarker(floorId: string, markerId: string): Promise<void> {
    try {
      await apiClient.delete(`/api/v1/floors/${floorId}/markers/${markerId}`)
      if (markers.value[floorId]) {
        markers.value[floorId] = markers.value[floorId].filter((m) => m.id !== markerId)
      }
    } catch (err: any) {
      error.value = err?.message || 'Failed to delete marker'
      throw err
    }
  }

  return {
    floors,
    activeFloorId,
    markers,
    loading,
    error,
    sortedFloors,
    activeFloor,
    currentMarkers,
    fetchFloors,
    fetchMarkers,
    createFloor,
    updateFloor,
    deleteFloor,
    saveMarker,
    deleteMarker
  }
})
