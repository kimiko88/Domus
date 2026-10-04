import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { SystemMetrics } from '@/api/types'
import { apiClient } from '@/api/client'

export const useSystemStore = defineStore('system', () => {
  const metrics = ref<SystemMetrics | null>(null)
  const loading = ref(false)
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

  async function fetchMetrics(): Promise<SystemMetrics> {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get<SystemMetrics>('/system/metrics')
      metrics.value = data
      return data
    } catch (err: any) {
      error.value = err.message || 'Failed to fetch metrics'
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    metrics,
    loading,
    error,
    formattedUptime,
    fetchMetrics
  }
})
