import { defineStore } from 'pinia'
import i18n from '../locales'
import { ref, computed } from 'vue'
import { apiClient } from '../api/client'
import type { Device } from '../api/types'

export const useDevicesStore = defineStore('devices', () => {
  const devices = ref<Device[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const onlineDevices = computed(() => (devices.value || []).filter((d) => d.status === 'online'))
  const offlineDevices = computed(() => (devices.value || []).filter((d) => d.status === 'offline'))

  const byId = (id: string): Device | undefined => (devices.value || []).find((d) => d.id === id)
  const byArea = (areaId: string): Device[] => (devices.value || []).filter((d) => d.area_id === areaId)

  async function updateDeviceArea(deviceId: string, areaId: string | null): Promise<{
    device_id: string
    area_id: string
    device_updated: boolean
    updated_entities: number
  }> {
    try {
      error.value = null
      const result = await apiClient.put<{
        device_id: string
        area_id: string
        device_updated: boolean
        updated_entities: number
      }>(`/api/v1/devices/${encodeURIComponent(deviceId)}/area`, {
        area_id: areaId || null
      })
      const existing = devices.value.find((device) => device.id === deviceId)
      if (existing) existing.area_id = result.area_id || undefined
      return result
    } catch (err: any) {
      error.value = i18n.global.t('devices.view.roomAssignErrorMessage')
      throw err
    }
  }

  async function fetchDevices(): Promise<Device[]> {
    try {
      loading.value = true
      error.value = null
      type ApiDevice = Omit<Device, 'status'> & {
        status?: Device['status']
        online?: boolean
      }
      const data = await apiClient.get<ApiDevice[] | null>('/api/v1/devices')
      devices.value = (data ?? []).map((device) => ({
        ...device,
        status: device.status === 'online' || device.status === 'offline' || device.status === 'unknown'
          ? device.status
          : device.online === true
            ? 'online'
            : device.online === false
              ? 'offline'
              : 'unknown'
      }))
      return devices.value
    } catch (err: any) {
      error.value = i18n.global.t('common.errors.loadFailed')
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
    fetchDevices,
    updateDeviceArea
  }
})
