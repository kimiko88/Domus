import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePredictiveClimateStore } from './predictiveClimate'
import { apiClient } from '../api/client'
import type { PredictiveZoneConfig, SystemPredictiveStatus, PredictiveEvaluation } from '../api/types'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

describe('Predictive Climate Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const mockZone: PredictiveZoneConfig = {
    id: 'pzone_1',
    name: 'Salone Sud',
    trv_entity_key: 'climate.valvola_salone',
    temp_sensor_key: 'sensor.salone_temp',
    pressure_sensor_key: 'sensor.salone_pressione',
    target_temperature: 21.0,
    orientation: 'south',
    window_area_sqm: 5.0,
    emitter_type: 'floor_radiant',
    thermal_inertia_hours: 4.0,
    preheat_enabled: true,
    solar_cutoff_enabled: true,
    pressure_boost_enabled: true,
    enabled: true
  }

  const mockEval: PredictiveEvaluation = {
    id: 'peval_1',
    zone_id: 'pzone_1',
    zone_name: 'Salone Sud',
    current_temp: 20.0,
    target_temp: 21.0,
    current_pressure: 1008.0,
    pressure_trend_hpa_hr: -2.1,
    humidity: 50.0,
    apparent_temp: 20.1,
    outdoor_temp_forecast: 8.0,
    direct_radiation_forecast: 500.0,
    estimated_solar_gain_kw: 1.3,
    cold_front_detected: true,
    window_open: false,
    calculated_setpoint: 22.5,
    mode: 'cold_front_boost',
    reason: 'Fronte freddo in arrivo',
    evaluated_at: '2026-10-18T10:00:00Z'
  }

  const mockStatus: SystemPredictiveStatus = {
    total_zones: 1,
    enabled_zones: 1,
    cold_front_active: true,
    solar_cutoff_active: false,
    zones: [
      {
        config: mockZone,
        latest_evaluation: mockEval
      }
    ],
    evaluated_at: '2026-10-18T10:00:00Z'
  }

  it('fetches status and populates zones and indicators', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockStatus)

    const store = usePredictiveClimateStore()
    await store.fetchStatus()

    expect(store.status).toEqual(mockStatus)
    expect(store.zones).toHaveLength(1)
    expect(store.totalZones).toBe(1)
    expect(store.enabledZones).toBe(1)
    expect(store.coldFrontActive).toBe(true)
    expect(store.solarCutoffActive).toBe(false)
  })

  it('saves new zone and calls post then updates status', async () => {
    const newZone = { ...mockZone, id: undefined }
    vi.mocked(apiClient.post).mockResolvedValueOnce(mockZone)
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockStatus)

    const store = usePredictiveClimateStore()
    const saved = await store.saveZone(newZone)

    expect(saved).toEqual(mockZone)
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/climate/predictive/zones', newZone)
    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/climate/predictive/status')
  })

  it('updates existing zone with put', async () => {
    vi.mocked(apiClient.put).mockResolvedValueOnce(mockZone)
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockStatus)

    const store = usePredictiveClimateStore()
    const saved = await store.saveZone(mockZone)

    expect(saved).toEqual(mockZone)
    expect(apiClient.put).toHaveBeenCalledWith('/api/v1/climate/predictive/zones/pzone_1', mockZone)
  })

  it('evaluates zone on demand', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce(mockEval)
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockStatus)

    const store = usePredictiveClimateStore()
    const res = await store.evaluateZone('pzone_1')

    expect(res).toBeDefined()
    expect(res.calculated_setpoint).toBe(22.5)
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/climate/predictive/zones/pzone_1/evaluate', {})
  })

  it('deletes zone by id', async () => {
    vi.mocked(apiClient.delete).mockResolvedValueOnce({})
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      ...mockStatus,
      total_zones: 0,
      enabled_zones: 0,
      zones: []
    })

    const store = usePredictiveClimateStore()
    store.zones = [mockZone]

    await store.deleteZone('pzone_1')
    expect(apiClient.delete).toHaveBeenCalledWith('/api/v1/climate/predictive/zones/pzone_1')
    expect(store.zones).toHaveLength(0)
  })
})
