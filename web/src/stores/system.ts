import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { SystemMetrics, OSDiagnostics, MaintenanceStatus, EnableMaintenancePayload } from '@/api/types'
import { apiClient } from '@/api/client'

export const useSystemStore = defineStore('system', () => {
  const metrics = ref<SystemMetrics | null>(null)
  const osDiagnostics = ref<OSDiagnostics | null>(null)
  const maintenanceStatus = ref<MaintenanceStatus | null>(null)
  const loading = ref(false)
  const diagLoading = ref(false)
  const maintLoading = ref(false)
  const error = ref<string | null>(null)

  const formattedUptime = computed(() => {
    if (!metrics.value || metrics.value.uptime_s == null) {
      return '0s'
    }
    const s = metrics.value.uptime_s
    const days = Math.floor(s / 86400)
    const hours = Math.floor((s % 86400) / 3600)
    const minutes = Math.floor((s % 3600) / 60)
    const seconds = s % 60

    const parts: string[] = []
    if (days > 0) parts.push(`${days}d`)
    if (hours > 0) parts.push(`${hours}h`)
    if (minutes > 0) parts.push(`${minutes}m`)
    if (seconds > 0 || parts.length === 0) parts.push(`${seconds}s`)

    return parts.join(' ')
  })

  const formattedMemory = computed(() => {
    if (!osDiagnostics.value || !osDiagnostics.value.memory_total_bytes) return 'N/A'
    const totalMB = Math.round(osDiagnostics.value.memory_total_bytes / (1024 * 1024))
    const usedMB = Math.round(osDiagnostics.value.memory_used_bytes / (1024 * 1024))
    return `${usedMB} MB / ${totalMB} MB (${osDiagnostics.value.memory_used_percent}%)`
  })

  const formattedDisk = computed(() => {
    if (!osDiagnostics.value || !osDiagnostics.value.disk_total_bytes) return 'N/A'
    const totalGB = (osDiagnostics.value.disk_total_bytes / (1024 * 1024 * 1024)).toFixed(1)
    const usedGB = (osDiagnostics.value.disk_used_bytes / (1024 * 1024 * 1024)).toFixed(1)
    return `${usedGB} GB / ${totalGB} GB (${osDiagnostics.value.disk_used_percent}%)`
  })

  const tempStatusClass = computed(() => {
    if (!osDiagnostics.value) return 'temp-normal'
    const t = osDiagnostics.value.soc_temp_c
    if (t >= 75) return 'temp-danger'
    if (t >= 60) return 'temp-warning'
    return 'temp-normal'
  })

  const formattedMaintenanceRemaining = computed(() => {
    if (!maintenanceStatus.value || !maintenanceStatus.value.active) return ''
    const s = maintenanceStatus.value.remaining_seconds
    const mins = Math.floor(s / 60)
    const secs = s % 60
    return `${mins}m ${secs}s`
  })

  async function fetchMetrics(): Promise<SystemMetrics> {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get<SystemMetrics>('/api/v1/system/metrics')
      metrics.value = data
      return data
    } catch (err: any) {
      error.value = err.message || 'Failed to fetch metrics'
      return metrics.value || {
        status: 'unknown',
        uptime_s: 0,
        entity_count: 0,
        device_count: 0,
        mqtt_connected: false,
        ws_clients: 0,
        db_size_bytes: 0
      }
    } finally {
      loading.value = false
    }
  }

  async function fetchOSDiagnostics(): Promise<OSDiagnostics | null> {
    diagLoading.value = true
    error.value = null
    try {
      const data = await apiClient.get<OSDiagnostics>('/api/v1/system/diagnostics')
      osDiagnostics.value = data
      return data
    } catch (err: any) {
      error.value = err.message || 'Failed to fetch OS diagnostics'
      return null
    } finally {
      diagLoading.value = false
    }
  }

  async function fetchMaintenanceStatus(): Promise<MaintenanceStatus> {
    try {
      const data = await apiClient.get<MaintenanceStatus>('/api/v1/system/maintenance/status')
      maintenanceStatus.value = data
      return data
    } catch (err: any) {
      // not fatal
      return { active: false, remaining_seconds: 0, ssh_port: 22 }
    }
  }

  async function enableMaintenance(payload: EnableMaintenancePayload): Promise<MaintenanceStatus> {
    maintLoading.value = true
    error.value = null
    try {
      const data = await apiClient.post<MaintenanceStatus>('/api/v1/system/maintenance/enable', payload)
      maintenanceStatus.value = data
      return data
    } catch (err: any) {
      error.value = err.message || 'Failed to enable maintenance mode'
      throw err
    } finally {
      maintLoading.value = false
    }
  }

  async function disableMaintenance(): Promise<MaintenanceStatus> {
    maintLoading.value = true
    error.value = null
    try {
      const data = await apiClient.post<MaintenanceStatus>('/api/v1/system/maintenance/disable', {})
      maintenanceStatus.value = data
      return data
    } catch (err: any) {
      error.value = err.message || 'Failed to disable maintenance mode'
      throw err
    } finally {
      maintLoading.value = false
    }
  }

  async function exportDiagnostics(): Promise<any> {
    try {
      const data = await apiClient.get<any>('/api/v1/system/diagnostics/export')
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `domus-diagnostics-${new Date().toISOString().slice(0, 10)}.json`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      return data
    } catch (err: any) {
      error.value = err.message || 'Failed to export diagnostics'
      throw err
    }
  }

  return {
    metrics,
    osDiagnostics,
    maintenanceStatus,
    loading,
    diagLoading,
    maintLoading,
    error,
    formattedUptime,
    formattedMemory,
    formattedDisk,
    tempStatusClass,
    formattedMaintenanceRemaining,
    fetchMetrics,
    fetchOSDiagnostics,
    fetchMaintenanceStatus,
    enableMaintenance,
    disableMaintenance,
    exportDiagnostics
  }
})
