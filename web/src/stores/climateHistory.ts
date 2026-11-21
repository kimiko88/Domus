import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiClient } from '@/api/client'
import type {
  ClimateDashboardSummary,
  ClimateHistoryPoint,
  ClimateHistoryResponse,
  ClimateHistorySettings
} from '@/api/types'

export interface ClimateHistoryQuery {
  from?: string
  to?: string
  area_id?: string
  entity_id?: string
  limit?: number
}

export const defaultClimateHistorySettings: ClimateHistorySettings = {
  enabled: true,
  interval_minutes: 10,
  record_temperature: true,
  record_humidity: true,
  record_thermostat: true,
  smart_sampling: false,
  temperature_change_threshold: 0.2,
  humidity_change_threshold: 1,
  raw_retention_days: 7,
  hourly_retention_days: 90,
  temperature_alerts_enabled: true,
  humidity_alerts_enabled: true,
  comfort_analysis_enabled: true,
  room_comparison_enabled: true,
  thermal_analysis_enabled: true,
  anomaly_detection_enabled: true,
  forecast_enabled: true,
  room_map_enabled: true,
  low_temperature_limit: 16,
  high_temperature_limit: 28,
  low_humidity_limit: 30,
  high_humidity_limit: 70
}

export const useClimateHistoryStore = defineStore('climateHistory', () => {
  const settings = ref<ClimateHistorySettings>({ ...defaultClimateHistorySettings })
  const points = ref<ClimateHistoryPoint[]>([])
  const summary = ref<ClimateDashboardSummary | null>(null)
  const loading = ref(false)
  const saving = ref(false)
  const error = ref<string | null>(null)

  async function fetchSettings() {
    settings.value = await apiClient.get<ClimateHistorySettings>('/api/v1/climate/history/settings')
    return settings.value
  }

  async function saveSettings(next: ClimateHistorySettings) {
    saving.value = true
    error.value = null
    try {
      settings.value = await apiClient.put<ClimateHistorySettings>('/api/v1/climate/history/settings', next)
      return settings.value
    } catch (err: any) {
      error.value = err?.message || 'Unable to save climate history settings'
      throw err
    } finally {
      saving.value = false
    }
  }

  async function fetchHistory(query: ClimateHistoryQuery = {}) {
    loading.value = true
    error.value = null
    try {
      const response = await apiClient.get<ClimateHistoryResponse>('/api/v1/climate/history/history', query)
      points.value = Array.isArray(response?.points) ? response.points : []
      return points.value
    } catch (err: any) {
      error.value = err?.message || 'Unable to load climate history'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchSummary(query: ClimateHistoryQuery = {}) {
    summary.value = await apiClient.get<ClimateDashboardSummary>('/api/v1/climate/history/summary', query)
    return summary.value
  }

  async function exportCSV(query: ClimateHistoryQuery = {}) {
    const csv = await apiClient.get<string>('/api/v1/climate/history/export.csv', query)
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = 'domus-climate-history.csv'
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
    URL.revokeObjectURL(url)
  }

  async function refresh(query: ClimateHistoryQuery = {}) {
    loading.value = true
    try {
      await Promise.all([fetchHistory(query), fetchSummary(query)])
    } finally {
      loading.value = false
    }
  }

  return {
    settings, points, summary, loading, saving, error,
    fetchSettings, saveSettings, fetchHistory, fetchSummary, exportCSV, refresh
  }
})
