import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ClimateHistoryView from './ClimateHistoryView.vue'
import { useClimateHistoryStore, defaultClimateHistorySettings } from '@/stores/climateHistory'
import { useEntitiesStore } from '@/stores/entities'
import { useAreasStore } from '@/stores/areas'
import { useDevicesStore } from '@/stores/devices'
import { useAuthStore } from '@/stores/auth'
import { apiClient } from '@/api/client'
import { i18n, setLanguage } from '@/locales'

vi.mock('chart.js/auto', () => ({
  default: class MockChart {
    destroy = vi.fn()
  }
}))

describe('ClimateHistoryView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    setLanguage('it')
    vi.restoreAllMocks()
  })

  function setupView() {
    const store = useClimateHistoryStore()
    const entities = useEntitiesStore()
    const areas = useAreasStore()
    const devices = useDevicesStore()
    const auth = useAuthStore()
    auth.user = {
      id: 'climate-admin', username: 'admin', role: 'admin',
      created_at: '', updated_at: ''
    }
    const roomFixtures = [
      { id: 'bedroom', name: 'Bedroom', icon: 'bed', sort_order: 1, created_at: '', updated_at: '' },
      { id: 'office', name: 'Office', icon: 'laptop', sort_order: 2, created_at: '', updated_at: '' }
    ]
    entities.entities = {
      'sensor-bedroom': {
        id: 'sensor-bedroom', entity_key: 'sensor.bedroom_temperature', device_id: 'device-bedroom',
        domain: 'sensor', name: 'Bedroom temperature', state: '21.5', unit_of_measurement: '°C',
        attributes: { device_class: 'temperature' },
        last_changed: '', last_updated: ''
      },
      'sensor-office': {
        id: 'sensor-office', entity_key: 'sensor.office_temperature', device_id: 'device-office',
        domain: 'sensor', name: 'Office temperature', state: '19.5',
        attributes: { device_class: 'temperature' },
        last_changed: '', last_updated: ''
      },
      'switch-not-climate': {
        id: 'switch-not-climate', entity_key: 'switch.office', device_id: 'device-switch',
        domain: 'switch', name: 'Office switch', state: 'off',
        attributes: {}, last_changed: '', last_updated: ''
      },
      'sensor-tuya-climate': {
        id: 'sensor.tuya_device-tuya-climate',
        entity_key: 'sensor.tuya_device-tuya-climate',
        domain: 'sensor', name: 'Tuya thermo-hygrometer', state: 'active',
        // Cloud-only Tuya devices must remain selectable before their first
        // temperature sample arrives, using the category metadata from Tuya.
        attributes: { tuya_climate_source: true, tuya_sensor_kind: 'temperature_humidity', device_class: 'temperature', unit_of_measurement: '°C' },
        last_changed: '', last_updated: ''
      }
    }
    devices.devices = [
      { id: 'device-bedroom', name: 'Bedroom sensor', integration_id: 'tuya', status: 'online', created_at: '', updated_at: '' },
      { id: 'device-office', name: 'Office sensor', integration_id: 'mqtt', status: 'offline', created_at: '', updated_at: '' },
      { id: 'device-switch', name: 'Office switch', integration_id: 'mqtt', status: 'online', created_at: '', updated_at: '' },
      { id: 'tuya_device-tuya-climate', external_id: 'device-tuya-climate', name: 'Tuya thermo-hygrometer', integration_id: 'tuya_local', status: 'online', created_at: '', updated_at: '' }
    ]
    vi.spyOn(devices, 'fetchDevices').mockImplementation(async () => devices.devices)
    vi.spyOn(apiClient, 'get').mockImplementation((async (path: string) => {
      if (path.endsWith('/tuya-local/devices')) return { devices: [] }
      if (path.endsWith('/tuya-cloud/config')) return { configured: false }
      if (path.endsWith('/tuya-oauth/status')) return { connected: false }
      throw new Error('Unexpected climate API request: ' + path)
    }) as any)
    vi.spyOn(store, 'fetchSettings').mockImplementation(async () => {
      Object.assign(store.settings, defaultClimateHistorySettings)
      return store.settings
    })
    vi.spyOn(entities, 'fetchEntities').mockResolvedValue([])
    areas.areas = roomFixtures
    vi.spyOn(areas, 'fetchAreas').mockImplementation(async () => {
      areas.areas = roomFixtures
      return roomFixtures
    })
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
          temperature_change_rate_c_per_hour: 0.4, forecast_temperature_1h: 21.9,
          last_updated_at: '2026-11-21T12:00:00Z'
        }],
        insights: [{ code: 'temperature_low', area_id: 'bedroom', value: 15, metric: 'temperature' }],
        sample_count: 1, from: '2026-11-21T00:00:00Z', to: '2026-11-21T23:59:59Z'
      }
    })
    return { store, refresh, devices, wrapper: mount(ClimateHistoryView, { global: { plugins: [i18n] } }) }
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


  it('offers configured rooms even when they have no climate samples yet', async () => {
    const { wrapper, refresh } = setupView()
    await flushPromises()
    refresh.mockClear()

    await wrapper.find('#climate-room-select').trigger('click')
    const roomOptions = wrapper.findAll('[role="option"]').map(option => option.text())
    expect(roomOptions).toContain('Tutte le stanze')
    expect(roomOptions).toContain('Bedroom')
    expect(roomOptions).toContain('Office')

    await wrapper.findAll('[role="option]').find(option => option.text() === 'Office')!.trigger('click')
    await flushPromises()
    expect(refresh).toHaveBeenCalledTimes(1)
    expect(refresh.mock.calls[0][0].area_id).toBe('office')
  })

  it('merges Tuya Cloud, Smart Life and LocalTuya climate sources and flags unsynchronized devices', async () => {
    const { wrapper } = setupView()
    const entities = useEntitiesStore()
    const getSpy = vi.mocked(apiClient.get)
    entities.entities['climate.tuya_cloud-thermostat'] = {
      id: 'climate.tuya_cloud-thermostat',
      entity_key: 'climate.tuya_cloud-thermostat',
      device_id: 'tuya_cloud-thermostat',
      domain: 'climate',
      name: 'Cloud thermostat entity',
      state: 'heat',
      attributes: { current_temperature: 21, target_temperature: 22 },
      last_changed: '', last_updated: ''
    }
    entities.entities['sensor.tuya_local-climate-sensor'] = {
      id: 'sensor.tuya_local-climate-sensor',
      entity_key: 'sensor.tuya_local-climate-sensor',
      device_id: 'local-climate-sensor',
      domain: 'sensor',
      name: 'Local temperature sensor',
      state: '21',
      unit_of_measurement: '°C',
      attributes: { device_class: 'temperature', current_temperature: 21 },
      last_changed: '', last_updated: ''
    }
    getSpy.mockImplementation((async (path: string) => {
      if (path.endsWith('/tuya-local/devices')) {
        return { devices: [{
          device_id: 'local-climate-sensor', name: 'Local temperature sensor',
          category: 'sensor', entity_key: 'sensor.tuya_local-climate-sensor', connected: true
        }] }
      }
      if (path.endsWith('/tuya-cloud/config')) return { configured: true }
      if (path.endsWith('/tuya-cloud/devices')) {
        return { devices: [{
          id: 'cloud-thermostat', name: 'Cloud thermostat', category: 'wk',
          inferred_category: 'climate', climate_source: true, online: true
        }] }
      }
      if (path.endsWith('/tuya-oauth/status')) return { connected: true }
      if (path.endsWith('/tuya-oauth/devices')) {
        return { devices: [{
          id: 'oauth-thermometer', name: 'Smart Life thermometer',
          category: 'thermo_hygrometer', climate_source: true, online: true
        }] }
      }
      throw new Error('Unexpected climate API request: ' + path)
    }) as any)

    await wrapper.find('.header-actions button').trigger('click')
    await flushPromises()
    await wrapper.find('.source-all-toggle input').setValue(false)

    const rows = wrapper.findAll('.source-device')
    expect(rows.some(row => row.text().includes('Cloud thermostat'))).toBe(true)
    expect(rows.some(row => row.text().includes('Local temperature sensor'))).toBe(true)
    expect(rows.some(row => row.text().includes('Smart Life thermometer'))).toBe(true)

    const unsynchronized = rows.find(row => row.text().includes('Smart Life thermometer'))
    expect(unsynchronized?.find('input').element.hasAttribute('disabled')).toBe(true)
    const cloudThermostat = rows.find(row => row.text().includes('Cloud thermostat'))
    expect(cloudThermostat?.find('input').element.hasAttribute('disabled')).toBe(false)
  })

  it('selects climate-capable integration devices and persists an explicit source selection', async () => {
    const { wrapper, store } = setupView()
    const saveSpy = vi.spyOn(store, 'saveSettings').mockImplementation(async settings => {
      Object.assign(store.settings, settings)
      return store.settings
    })
    await flushPromises()

    expect(wrapper.findAll('.source-device')).toHaveLength(3)
    expect((wrapper.find('.source-all-toggle input').element as HTMLInputElement).checked).toBe(true)

    await wrapper.find('.source-all-toggle input').setValue(false)
    const sourceInputs = wrapper.findAll('.source-device input')
    expect(sourceInputs).toHaveLength(3)

    const officeDevice = sourceInputs.find(input => (input.element as HTMLInputElement).value === 'device-office')
    expect(officeDevice).toBeTruthy()
    await officeDevice!.setValue(false)
    await wrapper.find('.sources-panel .primary-btn').trigger('click')
    await flushPromises()

    expect(saveSpy).toHaveBeenCalledTimes(1)
    expect(saveSpy.mock.calls[0][0].tracked_device_ids).toEqual(['device-bedroom', 'tuya_device-tuya-climate'])
  })

  it('includes Tuya entities exposed through device keys even when entity.device_id is absent', async () => {
    const { wrapper } = setupView()
    await flushPromises()

    await wrapper.find('.source-all-toggle input').setValue(false)
    const devices = wrapper.findAll('.source-device')
    expect(devices).toHaveLength(3)
    expect(wrapper.text()).toContain('Tuya thermo-hygrometer')
    expect(devices.some(device => device.text().includes('tuya_local'))).toBe(true)
  })

  it('lists Tuya Local sensors after their DPS telemetry is normalized as temperature and humidity', async () => {
    const { wrapper } = setupView()
    const entities = useEntitiesStore()
    const devicesStore = useDevicesStore()
    entities.entities['sensor.tuya_local_thermo'] = {
      id: 'sensor.tuya_local_thermo',
      entity_key: 'sensor.tuya_local_thermo',
      device_id: 'tuya_local_thermo',
      domain: 'sensor',
      name: 'Sensore temperatura Tuya Local',
      state: 'active',
      unit_of_measurement: '°C',
      attributes: {
        dp_1: 215,
        dp_2: 48,
        current_temperature: 21.5,
        current_humidity: 48,
        device_class: 'temperature',
        unit_of_measurement: '°C',
        tuya_climate_source: true,
        tuya_sensor_kind: 'temperature_humidity'
      },
      last_changed: '',
      last_updated: ''
    }
    devicesStore.devices.push({
      id: 'tuya_local_thermo',
      external_id: 'local_thermo',
      name: 'Sensore temperatura Tuya Local',
      integration_id: 'tuya_local',
      status: 'online',
      created_at: '',
      updated_at: ''
    })
    await flushPromises()

    await wrapper.find('.source-all-toggle input').setValue(false)
    const sourceDevices = wrapper.findAll('.source-device')
    expect(sourceDevices).toHaveLength(4)
    expect(sourceDevices.some(device => device.text().includes('Sensore temperatura Tuya Local'))).toBe(true)
  })

  it('uses app-rendered dropdown menus instead of native dark popups', async () => {
    const { wrapper } = setupView()
    await flushPromises()

    const metric = wrapper.find('#climate-metric-select')
    expect(metric.exists()).toBe(true)
    await metric.trigger('click')
    expect(wrapper.find('[role="listbox"]').exists()).toBe(true)
    expect(wrapper.findAll('[role="option"]').map(option => option.text())).toContain('Umidità')
    await wrapper.findAll('[role="option"]').find(option => option.text().includes('Umidità'))!.trigger('click')
    expect(metric.text()).toContain('Umidità')
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
