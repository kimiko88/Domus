import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ClimateHistoryView from './ClimateHistoryView.vue'
import { useClimateHistoryStore, defaultClimateHistorySettings } from '@/stores/climateHistory'
import { useEntitiesStore } from '@/stores/entities'
import { i18n } from '@/locales'

vi.mock('chart.js/auto', () => ({
  default: class MockChart {
    destroy = vi.fn()
  }
}))

describe('ClimateHistoryView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  function setupView() {
    const store = useClimateHistoryStore()
    const entities = useEntitiesStore()
    vi.spyOn(store, 'fetchSettings').mockImplementation(async () => {
      Object.assign(store.settings, defaultClimateHistorySettings)
      return store.settings
    })
    vi.spyOn(entities, 'fetchEntities').mockResolvedValue([])
    const refresh = vi.spyOn(store, 'refresh').mockImplementation(async () => {
      store.points = [{
        entity_id: 'sensor-bedroom', entity_key: 'sensor.bedroom_temperature',
        entity_name: 'Bedroom temperature', area_id: 'bedroom',
        recorded_at: '2026-11-21T12:00:00Z', temperature_average: 21.5,
        temperature_minimum: 21, temperature_maximum: 22,
        humidity_average: 52, target_temperature_average: 22,
        sample_count: 1, period: 'raw'
      }]
      store.summary = {
        rooms: [{
          area_id: 'bedroom', sample_count: 1, current_temperature: 21.5,
          average_temperature: 21.5, minimum_temperature: 21, maximum_temperature: 22,
          current_humidity: 52, average_humidity: 52,
          heating_rate_c_per_hour: 0.4, forecast_temperature_1h: 21.9,
          last_updated_at: '2026-11-21T12:00:00Z'
        }],
        insights: [{ code: 'temperature_low', area_id: 'bedroom', value: 15, metric: 'temperature' }],
        sample_count: 1, from: '2026-11-21T00:00:00Z', to: '2026-11-21T23:59:59Z'
      }
    })
    return { store, refresh, wrapper: mount(ClimateHistoryView, { global: { plugins: [i18n] } }) }
  }

  it('renders multi-room climate history, range filters and measurements', async () => {
    const { wrapper } = setupView()
    await flushPromises()
    expect(wrapper.text()).toContain('Clima & storico')
    expect(wrapper.text()).toContain('Andamento climatico')
    expect(wrapper.text()).toContain('Bedroom temperature')
    expect(wrapper.text()).toContain('21,5')
    expect(wrapper.findAll('.range-btn')).toHaveLength(4)
    expect(wrapper.find('canvas').exists()).toBe(true)
  })

  it('reloads history when the selected time range changes', async () => {
    const { wrapper, refresh } = setupView()
    await flushPromises()
    refresh.mockClear()
    const sevenDays = wrapper.findAll('.range-btn')[1]
    await sevenDays.trigger('click')
    await flushPromises()
    expect(refresh).toHaveBeenCalledTimes(1)
    expect(refresh.mock.calls[0][0]).toHaveProperty('from')
  })

  it('allows the user to disable optional climate analyses and persists settings', async () => {
    const { wrapper, store } = setupView()
    const saveSpy = vi.spyOn(store, 'saveSettings').mockImplementation(async settings => {
      Object.assign(store.settings, settings)
      return store.settings
    })
    await flushPromises()
    // The room-comparison checkbox is the fourth feature toggle in the analysis group.
    const checkboxes = wrapper.findAll('details.settings-panel input[type="checkbox"]')
    expect(checkboxes.length).toBeGreaterThanOrEqual(10)
    await checkboxes[12].setValue(false)
    const saveButton = wrapper.findAll('details.settings-panel button').find(button => button.text().includes('Salva impostazioni'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await flushPromises()
    expect(saveSpy).toHaveBeenCalledTimes(1)
    expect(saveSpy.mock.calls[0][0].room_map_enabled).toBe(false)
  })
})
