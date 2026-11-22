import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AreasView from './AreasView.vue'
import { useAreasStore } from '../stores/areas'
import { useEntitiesStore } from '../stores/entities'
import { useDevicesStore } from '../stores/devices'
import { useAuthStore } from '../stores/auth'
import { i18n } from '../locales'
import { apiClient } from '../api/client'

describe('AreasView.vue', () => {
  let areasStore: ReturnType<typeof useAreasStore>
  let entitiesStore: ReturnType<typeof useEntitiesStore>
  let devicesStore: ReturnType<typeof useDevicesStore>
  let authStore: ReturnType<typeof useAuthStore>

  beforeEach(() => {
    vi.restoreAllMocks()
    setActivePinia(createPinia())
    areasStore = useAreasStore()
    entitiesStore = useEntitiesStore()
    devicesStore = useDevicesStore()
    authStore = useAuthStore()
    vi.spyOn(devicesStore, 'fetchDevices').mockImplementation(async () => devicesStore.devices)
  })

  it('renders areas and displays their names and entity counts', async () => {
    areasStore.areas = [
      { id: 'area_living', name: 'Soggiorno', order_index: 1, created_at: '', updated_at: '' },
      { id: 'area_kitchen', name: 'Cucina', order_index: 2, created_at: '', updated_at: '' }
    ]

    entitiesStore.entities = {
      'light.living': {
        id: 'light.living',
        domain: 'light',
        name: 'Lampadario Soggiorno',
        state: 'on',
        attributes: { area_id: 'area_living' },
        last_changed: '',
        last_updated: ''
      }
    }

    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue(areasStore.areas)
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])

    const wrapper = mount(AreasView)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Soggiorno')
    expect(wrapper.text()).toContain('Cucina')
    expect(wrapper.text()).toContain('1 Dispositivi/Entità')
  })

  it('opens editor modal when "Nuova Area" button is clicked', async () => {
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue([])
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])

    const wrapper = mount(AreasView)
    await wrapper.vm.$nextTick()

    const newBtn = wrapper.find('.header-actions button')
    expect(newBtn.exists()).toBe(true)
    expect(newBtn.text()).toContain('Nuova Area')

    await newBtn.trigger('click')
    expect(wrapper.find('.modal-backdrop').exists()).toBe(true)
    expect(wrapper.text()).toContain(i18n.global.t('areas.createArea'))
  })

  it('opens editor modal when "Crea la prima area" in empty state is clicked', async () => {
    areasStore.areas = []
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue([])
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])

    const wrapper = mount(AreasView)
    await wrapper.vm.$nextTick()

    const emptyBtn = wrapper.find('.empty-btn')
    expect(emptyBtn.exists()).toBe(true)
    await emptyBtn.trigger('click')

    expect(wrapper.find('.modal-backdrop').exists()).toBe(true)
  })

  it('opens modal in edit mode when edit button is clicked', async () => {
    areasStore.areas = [
      { id: 'area_living', name: 'Soggiorno', icon: 'sofa', order_index: 1, created_at: '', updated_at: '' }
    ]
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue(areasStore.areas)
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])

    const wrapper = mount(AreasView)
    await wrapper.vm.$nextTick()

    const editBtn = wrapper.find('.btn-card-action:not(.is-danger)')
    expect(editBtn.exists()).toBe(true)
    await editBtn.trigger('click')

    expect(wrapper.find('.modal-backdrop').exists()).toBe(true)
    expect(wrapper.text()).toContain('Modifica Area')
    const input = wrapper.find<HTMLInputElement>('#area-name')
    expect(input.element.value).toBe('Soggiorno')
  })

  it('calls deleteArea when delete button is confirmed', async () => {
    areasStore.areas = [
      { id: 'area_living', name: 'Soggiorno', icon: 'sofa', order_index: 1, created_at: '', updated_at: '' }
    ]
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue(areasStore.areas)
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])
    vi.stubGlobal('confirm', vi.fn(() => true))
    const deleteSpy = vi.spyOn(areasStore, 'deleteArea').mockResolvedValue()

    const wrapper = mount(AreasView)
    await wrapper.vm.$nextTick()

    const deleteBtn = wrapper.find('.btn-card-action.is-danger')
    expect(deleteBtn.exists()).toBe(true)
    await deleteBtn.trigger('click')

    expect(deleteSpy).toHaveBeenCalledWith('area_living')
  })
  it('lets administrators assign integration devices from an area and move or unassign them', async () => {
    areasStore.areas = [
      { id: 'living-room', name: 'Soggiorno' },
      { id: 'office', name: 'Studio' }
    ]
    devicesStore.devices = [{
      id: 'tuya_sensor_1',
      external_id: 'sensor_1',
      name: 'Sensore Tuya',
      integration_id: 'tuya_local',
      area_id: 'living-room',
      status: 'online',
      created_at: '',
      updated_at: ''
    }]
    entitiesStore.entities = {
      'sensor.tuya_sensor_1': {
        id: 'sensor.tuya_sensor_1',
        entity_key: 'sensor.tuya_sensor_1',
        device_id: 'tuya_sensor_1',
        domain: 'sensor',
        name: 'Sensore Tuya',
        state: 'active',
        area_id: 'living-room',
        attributes: {},
        last_changed: '',
        last_updated: ''
      },
      'sensor.shelly_office': {
        id: 'sensor.shelly_office',
        entity_key: 'sensor.shelly_office',
        device_id: 'shelly_office_1',
        domain: 'sensor',
        name: 'Sensore Shelly Studio',
        state: '21.5',
        area_id: 'office',
        attributes: { integration_id: 'shelly' },
        last_changed: '',
        last_updated: ''
      }
    }
    authStore.user = {
      id: 'admin',
      username: 'admin',
      role: 'admin',
      created_at: '',
      updated_at: ''
    }

    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue(areasStore.areas)
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue(Object.values(entitiesStore.entities))
    const updateAreaSpy = vi.spyOn(devicesStore, 'updateDeviceArea').mockImplementation(async (deviceId, areaId) => ({
      device_id: deviceId,
      area_id: areaId || '',
      device_updated: true,
      updated_entities: 1
    }))

    const wrapper = mount(AreasView, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()

    await wrapper.find('.manage-devices-action').trigger('click')
    expect(wrapper.find('.area-device-modal').exists()).toBe(true)
    const options = wrapper.findAll('.area-device-option')
    expect(options).toHaveLength(2)
    expect(wrapper.text()).toContain('Sensore Shelly Studio')

    const tuyaCheckbox = options.find(option => option.text().includes('Sensore Tuya'))!.find('input')
    const shellyCheckbox = options.find(option => option.text().includes('Sensore Shelly Studio'))!.find('input')
    expect((tuyaCheckbox.element as HTMLInputElement).checked).toBe(true)
    expect((shellyCheckbox.element as HTMLInputElement).checked).toBe(false)

    await tuyaCheckbox.setValue(false)
    await shellyCheckbox.setValue(true)
    await wrapper.find('.area-device-primary-btn').trigger('click')

    expect(updateAreaSpy).toHaveBeenCalledTimes(2)
    expect(updateAreaSpy).toHaveBeenCalledWith('tuya_sensor_1', null, expect.objectContaining({ external_id: 'sensor_1', integration_id: 'tuya_local' }))
    expect(updateAreaSpy).toHaveBeenCalledWith('shelly_office_1', 'living-room')
    expect(wrapper.find('.area-device-modal').exists()).toBe(false)
  })

  it('finds and unassigns a legacy Tuya entity without device_id from its area', async () => {
    areasStore.areas = [{ id: 'bedroom', name: 'Camera' }]
    entitiesStore.entities['sensor.tuya_legacy_thermo'] = {
      id: 'sensor.tuya_legacy_thermo',
      entity_key: 'sensor.tuya_legacy_thermo',
      domain: 'sensor',
      name: 'Termoigrometro Tuya',
      state: 'active',
      area_id: 'bedroom',
      attributes: { integration_id: 'tuya_local' },
      last_changed: '',
      last_updated: ''
    }
    authStore.user = {
      id: 'admin-legacy',
      username: 'admin',
      role: 'admin',
      created_at: '',
      updated_at: ''
    }
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue(areasStore.areas)
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue(Object.values(entitiesStore.entities))
    const updateAreaSpy = vi.spyOn(devicesStore, 'updateDeviceArea').mockResolvedValue({
      device_id: 'tuya_legacy_thermo',
      area_id: '',
      device_updated: false,
      updated_entities: 1
    })

    const wrapper = mount(AreasView, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('1 Dispositivi/Entità')

    await wrapper.find('.manage-devices-action').trigger('click')
    const legacyDevice = wrapper.findAll('.area-device-option').find(option =>
      option.text().includes('Termoigrometro Tuya')
    )
    expect(legacyDevice).toBeDefined()
    const checkbox = legacyDevice!.find('input')
    expect((checkbox.element as HTMLInputElement).checked).toBe(true)
    await checkbox.setValue(false)
    await wrapper.find('.area-device-primary-btn').trigger('click')

    expect(updateAreaSpy).toHaveBeenCalledWith('tuya_legacy_thermo', null, expect.objectContaining({ external_id: 'legacy_thermo', integration_id: 'tuya_local' }))
  })

  it('lists and assigns online Tuya Cloud and Smart Life OAuth devices that are not in the local registry', async () => {
    areasStore.areas = [{ id: 'living-room', name: 'Soggiorno' }]
    authStore.user = {
      id: 'admin-cloud',
      username: 'admin',
      role: 'admin',
      created_at: '',
      updated_at: ''
    }
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue(areasStore.areas)
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])
    vi.spyOn(devicesStore, 'fetchDevices').mockResolvedValue([])
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url === '/api/v1/integrations/tuya-cloud/config') return { configured: true }
      if (url === '/api/v1/integrations/tuya-cloud/devices') {
        return {
          devices: [{
            id: 'cloud-only-thermo',
            name: 'Termoigrometro Cloud',
            product_name: 'TH01',
            category: 'wsdcgq',
            online: true,
            local_configured: false
          }]
        }
      }
      if (url === '/api/v1/integrations/tuya-oauth/status') return { connected: true }
      if (url === '/api/v1/integrations/tuya-oauth/devices') {
        return {
          devices: [{
            id: 'oauth-only-sensor',
            name: 'Sensore Smart Life',
            product_name: 'Temperature sensor',
            category: 'temperature_sensor',
            online: true,
            local_configured: false
          }]
        }
      }
      throw new Error(`Unexpected API GET: ${url}`)
    })
    const putSpy = vi.spyOn(apiClient, 'put').mockResolvedValue({
      device_id: '',
      area_id: 'living-room',
      device_updated: true,
      device_created: true,
      updated_entities: 0
    })

    const wrapper = mount(AreasView, { global: { plugins: [i18n] } })
    await vi.waitFor(() => expect(areasStore.areas).toHaveLength(1))
    await new Promise(resolve => setTimeout(resolve, 0))
    await wrapper.find('.manage-devices-action').trigger('click')

    await vi.waitFor(() => expect(wrapper.findAll('.area-device-option')).toHaveLength(2))
    expect(wrapper.text()).toContain('Termoigrometro Cloud')
    expect(wrapper.text()).toContain('Sensore Smart Life')

    for (const option of wrapper.findAll('.area-device-option')) {
      await option.find('input').setValue(true)
    }
    await wrapper.find('.area-device-primary-btn').trigger('click')

    expect(putSpy).toHaveBeenCalledWith('/api/v1/devices/tuya_cloud-only-thermo/area', {
      area_id: 'living-room',
      external_id: 'cloud-only-thermo',
      name: 'Termoigrometro Cloud',
      integration_id: 'tuya_cloud',
      online: true,
      manufacturer: 'Tuya',
      model: 'TH01'
    })
    expect(putSpy).toHaveBeenCalledWith('/api/v1/devices/tuya_oauth-only-sensor/area', {
      area_id: 'living-room',
      external_id: 'oauth-only-sensor',
      name: 'Sensore Smart Life',
      integration_id: 'tuya_oauth',
      online: true,
      manufacturer: 'Tuya',
      model: 'Temperature sensor'
    })
  })

  it('does not expose area-device assignment to non-admin users', async () => {
    areasStore.areas = [{ id: 'living-room', name: 'Soggiorno' }]
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue(areasStore.areas)
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])

    const wrapper = mount(AreasView, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.manage-devices-action').exists()).toBe(false)
  })

})
