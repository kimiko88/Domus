import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import PredictiveClimateView from './PredictiveClimateView.vue'
import { apiClient } from '../api/client'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

describe('PredictiveClimateView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const mockStatus = {
    total_zones: 2,
    enabled_zones: 2,
    cold_front_active: true,
    solar_cutoff_active: false,
    zones: [
      {
        config: {
          id: 'zone-1',
          name: 'Salone Sud',
          trv_entity_key: 'climate.valvola_salone',
          temp_sensor_key: 'sensor.salone_temp',
          pressure_sensor_key: 'sensor.salone_pressione',
          target_temperature: 21.0,
          orientation: 'south',
          window_area_sqm: 6.0,
          emitter_type: 'floor_radiant',
          thermal_inertia_hours: 4.0,
          preheat_enabled: true,
          solar_cutoff_enabled: true,
          pressure_boost_enabled: true,
          enabled: true
        },
        latest_evaluation: {
          id: 'eval-1',
          zone_id: 'zone-1',
          zone_name: 'Salone Sud',
          current_temp: 20.2,
          target_temp: 21.0,
          current_pressure: 1008.0,
          pressure_trend_hpa_hr: -2.3,
          humidity: 50.0,
          apparent_temp: 20.3,
          outdoor_temp_forecast: 7.5,
          direct_radiation_forecast: 450.0,
          estimated_solar_gain_kw: 1.1,
          cold_front_detected: true,
          window_open: false,
          calculated_setpoint: 22.5,
          mode: 'cold_front_boost',
          reason: 'Anticipo per crollo barometrico',
          evaluated_at: '2026-10-18T10:00:00Z'
        }
      }
    ],
    evaluated_at: '2026-10-18T10:00:00Z'
  }

  it('renders header, KPI metrics, and zone card with calculated setpoint', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockStatus)

    const wrapper = mount(PredictiveClimateView)
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    expect(wrapper.text()).toContain('Termostato Predittivo Multizona')
    expect(wrapper.text()).toContain('Fronte Freddo Rilevato')
    expect(wrapper.text()).toContain('Salone Sud')
    expect(wrapper.text()).toContain('22.5 °C')
    expect(wrapper.text()).toContain('Fronte Freddo (+Boost)')
  })

  it('opens create zone modal on button click', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockStatus)

    const wrapper = mount(PredictiveClimateView)
    await wrapper.vm.$nextTick()

    const createBtn = wrapper.findAll('button').find((b) => b.text().includes('Nuova Zona'))
    expect(createBtn).toBeDefined()
    await createBtn!.trigger('click')

    expect(wrapper.find('.modal-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('Nome Zona / Stanza')
  })

  it('triggers manual evaluate zone action', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockStatus)
    vi.mocked(apiClient.post).mockResolvedValueOnce({
      ...mockStatus.zones[0].latest_evaluation,
      calculated_setpoint: 23.0
    })
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockStatus)

    const wrapper = mount(PredictiveClimateView)
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    const evalBtn = wrapper.findAll('button').find((b) => b.text().includes('Valuta Ora'))
    expect(evalBtn).toBeDefined()
    await evalBtn!.trigger('click')

    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/climate/predictive/zones/zone-1/evaluate', {})
  })
})
