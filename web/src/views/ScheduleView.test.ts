import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ScheduleView from './ScheduleView.vue'
import { apiClient } from '../api/client'
import { useEntitiesStore } from '../stores/entities'

describe('ScheduleView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/schedules')) {
        return [
          {
            id: 'sched_climate_living',
            name: 'Cronotermostato Salotto',
            entity_key: 'climate.living_room',
            enabled: true,
            grid: {
              monday: [{ start: '06:30', end: '08:30', value: 21.0 }]
            }
          }
        ] as any
      }
      if (url.includes('/api/v1/calendar/events')) {
        return [
          {
            id: 'evt_1',
            summary: 'Smart Working da Casa',
            description: 'Imposta riscaldamento a 21.5°C',
            start_time: '2026-10-15T09:00:00Z',
            end_time: '2026-10-15T18:00:00Z',
            location: 'Studio',
            source: 'local'
          }
        ] as any
      }
      return null as any
    })
  })

  it('renders weekly matrix cronotermostato planner', async () => {
    const wrapper = mount(ScheduleView)
    await flushPromises()

    expect(wrapper.text()).toContain('Programmazione Oraria & Calendari')
    expect(wrapper.text()).toContain('Cronotermostato Salotto')
    expect(wrapper.text()).toContain('climate.living_room')
    expect(wrapper.text()).toContain('06:30 - 08:30')
    expect(wrapper.text()).toContain('21.0°C')
    expect(wrapper.find('[data-testid="sensor-automation-link"]').exists()).toBe(true)
    await wrapper.find('.btn-add-slot').trigger('click')
    expect(wrapper.find('#slot-temp').attributes('step')).toBe('0.1')
  })

  it('switches to calendar view and shows events', async () => {
    const wrapper = mount(ScheduleView)
    await flushPromises()

    const calTabBtn = wrapper.findAll('.tab-btn')[1]
    await calTabBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Eventi sincronizzati e trigger delle automazioni')
    expect(wrapper.text()).toContain('Smart Working da Casa')
  })
})


describe('manual thermostat overrides', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('sends manual thermostat changes with a server-side override duration', async () => {
  const thermostat = {
    id: 'climate-uuid',
    entity_key: 'climate.living_room',
    domain: 'climate',
    name: 'Termostato soggiorno',
    state: 'heat',
    attributes: { current_temperature: 19, target_temperature: 20 },
    last_changed: '',
    last_updated: ''
  }
  const getSpy = vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
    if (url.includes('/api/v1/entities')) return [thermostat] as any
    if (url.includes('/api/v1/schedules')) return [{
      id: 'sched_climate_living', name: 'Cronotermostato Salotto',
      entity_key: 'climate.living_room', enabled: true, grid: { monday: [] }
    }] as any
    if (url.includes('/api/v1/calendar/events')) return [] as any
    return null as any
  })
  const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ status: 'accepted' } as any)
  const serviceSpy = vi.spyOn(useEntitiesStore(), 'callService').mockResolvedValue(undefined)
  const wrapper = mount(ScheduleView)
  await flushPromises()

  await wrapper.find('[data-testid="manual-climate-override-duration"]').setValue('120')
  await wrapper.find('.btn-apply-thermostat').trigger('click')
  await flushPromises()

  expect(postSpy).toHaveBeenCalledWith(
    '/api/v1/entities/climate-uuid/climate-profile',
    { profile: 'custom', temperature: 20, duration_minutes: 120 }
  )
  expect(wrapper.text()).toContain('Alla scadenza Domus ripristina il setpoint precedente.')

  getSpy.mockRestore()
  postSpy.mockRestore()
  serviceSpy.mockRestore()
  wrapper.unmount()
  })

  it('uses the selected schedule entity UUID when applying a thermostat override', async () => {
    const firstThermostat = {
      id: 'first-tuya-uuid', entity_key: 'climate.tuya_room', domain: 'climate',
      name: 'Termostato soggiorno', state: 'heat',
      attributes: { target_temperature: 21, current_temperature: 19, hvac_modes: ['heat'] },
      last_changed: '', last_updated: ''
    }
    const secondThermostat = {
      id: 'second-tuya-uuid', entity_key: 'climate.kitchen', domain: 'climate',
      name: 'Termostato cucina', state: 'auto',
      attributes: { target_temperature: 20, current_temperature: 18, hvac_modes: ['auto'] },
      last_changed: '', last_updated: ''
    }
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/entities')) return [firstThermostat, secondThermostat] as any
      if (url.includes('/api/v1/schedules')) return [{
        id: 'sched-kitchen', name: 'Cucina', entity_key: 'climate.tuya_room',
        enabled: true, grid: { monday: [] }
      }] as any
      return [] as any
    })
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ status: 'accepted' } as any)
    const serviceSpy = vi.spyOn(useEntitiesStore(), 'callService').mockResolvedValue(undefined)
    const wrapper = mount(ScheduleView)
    await flushPromises()

    await wrapper.find('#schedule-thermostat-select').setValue('climate.kitchen')
    await flushPromises()
    expect((wrapper.find('#thermostat-select').element as HTMLSelectElement).value).toBe('second-tuya-uuid')

    await wrapper.find('.btn-apply-thermostat').trigger('click')
    await flushPromises()

    expect(postSpy).toHaveBeenCalledWith(
      '/api/v1/entities/second-tuya-uuid/climate-profile',
      { profile: 'custom', temperature: 20, duration_minutes: 0 }
    )

    postSpy.mockRestore()
    serviceSpy.mockRestore()
    wrapper.unmount()
  })

  it('sends immediate commands to a Smart Life thermostat without a Domus entity and rejects timed overrides', async () => {
    const cloudThermostat = {
      id: 'cloud-thermostat-uuid',
      name: 'Termostato Cloud',
      product_name: 'Smart thermostat',
      category: 'wk',
      inferred_category: 'climate',
      entity_key: 'climate.tuya_cloud_room',
      online: true,
      local_configured: false,
      status: [
        { code: 'switch', value: true },
        { code: 'mode', value: 'auto' },
        { code: 'temp_set', value: 210 },
        { code: 'temp_current', value: 190 }
      ]
    }
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/entities')) return [] as any
      if (url.includes('/api/v1/schedules')) return [{
        id: 'sched-cloud', name: 'Smart Life Cloud',
        entity_key: 'climate.tuya_cloud_room', enabled: true, grid: { monday: [] }
      }] as any
      if (url.endsWith('/api/v1/integrations/tuya-cloud/devices')) {
        return { devices: [cloudThermostat] } as any
      }
      return { devices: [] } as any
    })
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ success: true } as any)
    const wrapper = mount(ScheduleView)
    await flushPromises()

    const duration = wrapper.find('[data-testid="manual-climate-override-duration"]')
    await duration.setValue('15')
    await wrapper.find('.btn-apply-thermostat').trigger('click')
    await flushPromises()
    expect(postSpy).not.toHaveBeenCalled()

    await duration.setValue('0')
    await wrapper.find('.btn-apply-thermostat').trigger('click')
    await flushPromises()
    expect(postSpy).toHaveBeenCalledWith(
      '/api/v1/integrations/tuya-cloud/devices/cloud-thermostat-uuid/commands',
      {
        commands: [
          { code: 'switch', value: true },
          { code: 'temp_set', value: 210 },
          { code: 'mode', value: 'auto' }
        ]
      }
    )

    postSpy.mockRestore()
    wrapper.unmount()
  })

  it('uses Smart Life thermostat setpoint and only its reported LocalTuya modes', async () => {
    const thermostat = {
      id: 'tuya-device-uuid', entity_key: 'climate.tuya_room', domain: 'climate',
      name: 'Termostato Smart Life', state: 'on',
      attributes: { target_temperature: 21, current_temperature: 19, hvac_mode: 'auto', hvac_modes: ['auto', 'manual'] },
      last_changed: '', last_updated: ''
    }
    const secondThermostat = {
      id: 'second-tuya-uuid', entity_key: 'climate.kitchen', domain: 'climate',
      name: 'Termostato cucina', state: 'on',
      attributes: { target_temperature: 20, current_temperature: 18, hvac_modes: ['auto'] },
      last_changed: '', last_updated: ''
    }
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/entities')) return [thermostat, secondThermostat] as any
      if (url.includes('/api/v1/schedules')) return [{ id: 'sched-tuya', name: 'Smart Life', entity_key: 'climate.tuya_room', enabled: true, grid: { monday: [] } }] as any
      return [] as any
    })
    vi.spyOn(apiClient, 'post').mockResolvedValue({ status: 'accepted' } as any)
    const serviceSpy = vi.spyOn(useEntitiesStore(), 'callService').mockResolvedValue(undefined)
    const wrapper = mount(ScheduleView)
    await flushPromises()

    expect((wrapper.find('#thermostat-select').element as HTMLSelectElement).value).toBe('tuya-device-uuid')
    const scheduleThermostat = wrapper.find('#schedule-thermostat-select')
    expect(scheduleThermostat.exists()).toBe(true)
    expect(scheduleThermostat.text()).toContain('Termostato cucina (climate.kitchen)')
    await scheduleThermostat.setValue('climate.kitchen')
    expect((wrapper.find('#thermostat-select').element as HTMLSelectElement).value).toBe('second-tuya-uuid')
    await scheduleThermostat.setValue('climate.tuya_room')
    expect((wrapper.find('#thermostat-select').element as HTMLSelectElement).value).toBe('tuya-device-uuid')

    expect(wrapper.find('.temp-display-block.target').exists()).toBe(true)
    const modeButtons = wrapper.findAll('.hvac-mode-buttons .mode-btn')
    expect(modeButtons).toHaveLength(2)
    expect(modeButtons[1].text()).toBe('manual')
    expect(wrapper.find('.hvac-mode-buttons').text()).not.toContain('heat')
    expect(wrapper.find('.hvac-mode-buttons').text()).not.toContain('off')

    serviceSpy.mockRestore()
    wrapper.unmount()
  })
})
