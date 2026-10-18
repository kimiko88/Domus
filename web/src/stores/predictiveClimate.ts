import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '../api/client'
import type {
  PredictiveZoneConfig,
  PredictiveEvaluation,
  SystemPredictiveStatus,
  ZoneDashboardStatus
} from '../api/types'

export const usePredictiveClimateStore = defineStore('predictiveClimate', () => {
  const status = ref<SystemPredictiveStatus | null>(null)
  const zones = ref<PredictiveZoneConfig[]>([])
  const loading = ref(false)
  const evaluating = ref(false)
  const error = ref<string | null>(null)

  // --- COMPUTED ---

  const totalZones = computed(() => zones.value.length)
  const enabledZones = computed(() => zones.value.filter((z) => z.enabled).length)
  const coldFrontActive = computed(() => status.value?.cold_front_active ?? false)
  const solarCutoffActive = computed(() => status.value?.solar_cutoff_active ?? false)

  const zoneStatuses = computed<ZoneDashboardStatus[]>(() => {
    if (status.value?.zones && status.value.zones.length > 0) {
      return status.value.zones
    }
    return zones.value.map((z) => ({ config: z }))
  })

  // --- ACTIONS ---

  async function fetchStatus() {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get<SystemPredictiveStatus>('/api/v1/climate/predictive/status')
      status.value = data
      if (data.zones) {
        zones.value = data.zones.map((zs) => zs.config)
      }
    } catch (err: any) {
      error.value = err.message || 'Errore durante il caricamento dello stato predittivo'
    } finally {
      loading.value = false
    }
  }

  async function fetchZones() {
    loading.value = true
    error.value = null
    try {
      const res = await apiClient.get<{ zones: PredictiveZoneConfig[]; total: number }>(
        '/api/v1/climate/predictive/zones'
      )
      zones.value = res.zones || []
    } catch (err: any) {
      error.value = err.message || 'Errore durante il caricamento delle zone predittive'
    } finally {
      loading.value = false
    }
  }

  async function saveZone(cfg: PredictiveZoneConfig): Promise<PredictiveZoneConfig> {
    loading.value = true
    error.value = null
    try {
      let saved: PredictiveZoneConfig
      if (cfg.id) {
        saved = await apiClient.put<PredictiveZoneConfig>(
          `/api/v1/climate/predictive/zones/${cfg.id}`,
          cfg
        )
        const idx = zones.value.findIndex((z) => z.id === cfg.id)
        if (idx !== -1) {
          zones.value[idx] = saved
        }
      } else {
        saved = await apiClient.post<PredictiveZoneConfig>(
          '/api/v1/climate/predictive/zones',
          cfg
        )
        zones.value.push(saved)
      }
      await fetchStatus()
      return saved
    } catch (err: any) {
      error.value = err.message || 'Errore durante il salvataggio della zona predittiva'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteZone(id: string) {
    loading.value = true
    error.value = null
    try {
      await apiClient.delete(`/api/v1/climate/predictive/zones/${id}`)
      zones.value = zones.value.filter((z) => z.id !== id)
      await fetchStatus()
    } catch (err: any) {
      error.value = err.message || 'Errore durante l\'eliminazione della zona predittiva'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function evaluateZone(id: string): Promise<PredictiveEvaluation> {
    evaluating.value = true
    error.value = null
    try {
      const evalResult = await apiClient.post<PredictiveEvaluation>(
        `/api/v1/climate/predictive/zones/${id}/evaluate`,
        {}
      )
      await fetchStatus()
      return evalResult
    } catch (err: any) {
      error.value = err.message || 'Errore durante la valutazione predittiva della zona'
      throw err
    } finally {
      evaluating.value = false
    }
  }

  async function evaluateAll() {
    evaluating.value = true
    error.value = null
    try {
      await apiClient.post('/api/v1/climate/predictive/evaluate-all', {})
      await fetchStatus()
    } catch (err: any) {
      error.value = err.message || 'Errore durante la valutazione di tutte le zone'
      throw err
    } finally {
      evaluating.value = false
    }
  }

  return {
    status,
    zones,
    loading,
    evaluating,
    error,
    totalZones,
    enabledZones,
    coldFrontActive,
    solarCutoffActive,
    zoneStatuses,
    fetchStatus,
    fetchZones,
    saveZone,
    deleteZone,
    evaluateZone,
    evaluateAll
  }
})
