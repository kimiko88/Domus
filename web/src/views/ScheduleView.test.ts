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

  it('uses Smart Life thermostat setpoint and only its reported LocalTuya modes', async () => {
    const thermostat = {
      id: 'climate.tuya_room', entity_key: 'climate.tuya_room', domain: 'climate',
      name: 'Termostato Smart Life', state: 'on',
      attributes: { target_temperature: 21, current_temperature: 19, hvac_mode: 'auto', hvac_modes: ['auto', 'manual'] },
      last_changed: '', last_updated: ''
    }
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/entities')) return [thermostat] as any
      if (url.includes('/api/v1/schedules')) return [{ id: 'sched-tuya', name: 'Smart Life', entity_key: 'climate.tuya_room', enabled: true, grid: { monday: [] } }] as any
      return [] as any
    })
    vi.spyOn(apiClient, 'post').mockResolvedValue({ status: 'accepted' } as any)
    const serviceSpy = vi.spyOn(useEntitiesStore(), 'callService').mockResolvedValue(undefined)
    const wrapper = mount(ScheduleView)
    await flushPromises()

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
