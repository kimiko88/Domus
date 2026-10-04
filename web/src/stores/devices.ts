import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '../api/client'
import type { Device } from '../api/types'

export const useDevicesStore = defineStore('devices', () => {
  const devices = ref<Device[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const onlineDevices = computed(() => devices.value.filter((d) => d.status === 'online'))
  const offlineDevices = computed(() => devices.value.filter((d) => d.status === 'offline'))

  const byId = (id: string): Device | undefined => devices.value.find((d) => d.id === id)
  const byArea = (areaId: string): Device[] => devices.value.filter((d) => d.area_id === areaId)

  async function fetchDevices(): Promise<Device[]> {
    try {
      loading.value = true
      error.value = null
      const data = await apiClient.get<Device[]>('/api/v1/devices')
      devices.value = data
      return data
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch devices'
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    devices,
    onlineDevices,
    offlineDevices,
    loading,
    error,
    byId,
    byArea,
    fetchDevices
  }
})
