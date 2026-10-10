import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ScheduleView from './ScheduleView.vue'
import { apiClient } from '../api/client'

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
  })

  it('switches to calendar view and shows events', async () => {
    const wrapper = mount(ScheduleView)
    await flushPromises()

    const calTabBtn = wrapper.findAll('.tab-btn')[1]
    await calTabBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Eventi Sincronizzati & Trigger Automazioni')
    expect(wrapper.text()).toContain('Smart Working da Casa')
  })
})
