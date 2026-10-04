import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ClimateCard from './ClimateCard.vue'
import { useEntitiesStore } from '../../stores/entities'
import type { Entity } from '../../api/types'

describe('ClimateCard.vue', () => {
  let entitiesStore: ReturnType<typeof useEntitiesStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    entitiesStore = useEntitiesStore()
  })

  it('renders climate name, current temperature, and target setpoint', () => {
    const entity: Entity = {
      id: 'climate.thermostat',
      domain: 'climate',
      name: 'Main Thermostat',
      state: 'heat',
      attributes: {
        current_temperature: 20.5,
        temperature: 21.0,
        unit_of_measurement: '°C'
      },
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }

    const wrapper = mount(ClimateCard, {
      props: { entity }
    })

    expect(wrapper.text()).toContain('Main Thermostat')
    expect(wrapper.text()).toContain('20.5')
    expect(wrapper.text()).toContain('21.0')
    expect(wrapper.text()).toContain('heat')
  })

  it('increments target temperature by 0.5 when plus button is clicked', async () => {
    const entity: Entity = {
      id: 'climate.living_room',
      domain: 'climate',
      name: 'Living Room HVAC',
      state: 'auto',
      attributes: {
        temperature: 21.0
      },
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }

    const callServiceSpy = vi.spyOn(entitiesStore, 'callService').mockResolvedValue(undefined)

    const wrapper = mount(ClimateCard, {
      props: { entity }
    })

    const plusBtn = wrapper.find('.btn-temp-plus')
    expect(plusBtn.exists()).toBe(true)

    await plusBtn.trigger('click')

    expect(callServiceSpy).toHaveBeenCalledWith('climate', 'set_temperature', 'climate.living_room', {
      temperature: 21.5
    })
  })

  it('decrements target temperature by 0.5 when minus button is clicked', async () => {
    const entity: Entity = {
      id: 'climate.bedroom',
      domain: 'climate',
      name: 'Bedroom HVAC',
      state: 'heat',
      attributes: {
        temperature: 20.0
      },
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }

    const callServiceSpy = vi.spyOn(entitiesStore, 'callService').mockResolvedValue(undefined)

    const wrapper = mount(ClimateCard, {
      props: { entity }
    })

    const minusBtn = wrapper.find('.btn-temp-minus')
    expect(minusBtn.exists()).toBe(true)

    await minusBtn.trigger('click')

    expect(callServiceSpy).toHaveBeenCalledWith('climate', 'set_temperature', 'climate.bedroom', {
      temperature: 19.5
    })
  })
})
