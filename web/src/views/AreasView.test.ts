import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AreasView from './AreasView.vue'
import { useAreasStore } from '../stores/areas'
import { useEntitiesStore } from '../stores/entities'
import { useDevicesStore } from '../stores/devices'
import { useAuthStore } from '../stores/auth'
import { i18n } from '../locales'

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

    expect(updateAreaSpy).toHaveBeenNthCalledWith(1, 'tuya_sensor_1', null)
    expect(updateAreaSpy).toHaveBeenNthCalledWith(2, 'shelly_office_1', 'living-room')
    expect(wrapper.find('.area-device-modal').exists()).toBe(false)
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
