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

  it('fetches thermal profile and caches it', async () => {
    const mockProfile = {
      zone_id: 'pzone_1',
      heat_rate_per_hr: 1.25,
      cool_rate_per_hr: 0.22,
      outdoor_loss_coeff: 0.02,
      solar_boost_coeff: 0.3,
      samples_count: 3,
      last_trained_at: '2026-10-18T12:00:00Z'
    }
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockProfile)

    const store = usePredictiveClimateStore()
    const p = await store.fetchProfile('pzone_1')

    expect(p).toEqual(mockProfile)
    expect(store.profiles['pzone_1']).toEqual(mockProfile)
    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/climate/predictive/zones/pzone_1/profile')
  })

  it('trains zone and updates cached profile', async () => {
    const trainedProfile = {
      zone_id: 'pzone_1',
      heat_rate_per_hr: 1.35,
      cool_rate_per_hr: 0.22,
      outdoor_loss_coeff: 0.02,
      solar_boost_coeff: 0.3,
      samples_count: 4,
      last_trained_at: '2026-10-18T13:00:00Z'
    }
    vi.mocked(apiClient.post).mockResolvedValueOnce(trainedProfile)

    const store = usePredictiveClimateStore()
    const p = await store.trainZone('pzone_1', {
      cycle_type: 'heating',
      start_temp: 19.0,
      end_temp: 20.5,
      duration_minutes: 60
    })

    expect(p.samples_count).toBe(4)
    expect(store.profiles['pzone_1']).toEqual(trainedProfile)
    expect(apiClient.post).toHaveBeenCalledWith(
      '/api/v1/climate/predictive/zones/pzone_1/train',
      expect.objectContaining({ cycle_type: 'heating' })
    )
  })

  it('calculates optimal start and stop recommendations', async () => {
    const mockStart = {
      zone_id: 'pzone_1',
      target_time: '2026-10-18T16:00:00Z',
      target_temp: 21.0,
      current_temp: 19.5,
      outdoor_temp: 10.0,
      preheat_minutes: 45,
      start_time: '2026-10-18T15:15:00Z',
      effective_heat_rate: 1.2,
      reason: 'Avvio anticipato di 45 min'
    }
    const mockStop = {
      zone_id: 'pzone_1',
      departure_time: '2026-10-18T18:00:00Z',
      comfort_limit_temp: 20.5,
      coast_down_minutes: 30,
      stop_time: '2026-10-18T17:30:00Z',
      energy_saved_kwh: 0.75,
      reason: 'Spegnimento anticipato di 30 min'
    }
    vi.mocked(apiClient.get)
      .mockResolvedValueOnce(mockStart)
      .mockResolvedValueOnce(mockStop)

    const store = usePredictiveClimateStore()
    const start = await store.getOptimalStart('pzone_1', '2026-10-18T16:00:00Z', 21.0)
    expect(start.preheat_minutes).toBe(45)

    const stop = await store.getOptimalStop('pzone_1', '2026-10-18T18:00:00Z')
    expect(stop.coast_down_minutes).toBe(30)
    expect(stop.energy_saved_kwh).toBe(0.75)
  })
})
