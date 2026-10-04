import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import SensorCard from './SensorCard.vue'
import type { Entity } from '../../api/types'

describe('SensorCard.vue', () => {
  it('renders temperature sensor value and unit', () => {
    const entity: Entity = {
      id: 'sensor.living_room_temp',
      domain: 'sensor',
      name: 'Living Room Temperature',
      state: '21.5',
      attributes: {
        unit_of_measurement: '°C',
        device_class: 'temperature'
      },
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }

    const wrapper = mount(SensorCard, {
      props: { entity }
    })

    expect(wrapper.text()).toContain('Living Room Temperature')
    expect(wrapper.text()).toContain('21.5')
    expect(wrapper.text()).toContain('°C')
  })

  it('renders binary sensor state (motion detected)', () => {
    const entity: Entity = {
      id: 'binary_sensor.front_door',
      domain: 'binary_sensor',
      name: 'Front Door Motion',
      state: 'on',
      attributes: {
        device_class: 'motion'
      },
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }

    const wrapper = mount(SensorCard, {
      props: { entity }
    })

    expect(wrapper.text()).toContain('Front Door Motion')
    expect(wrapper.text()).toContain('Motion')
    expect(wrapper.find('.sensor-card').classes()).toContain('is-triggered')
  })

  it('displays battery badge when battery attribute is present', () => {
    const entity: Entity = {
      id: 'sensor.balcony_temp',
      domain: 'sensor',
      name: 'Balcony Temp',
      state: '18.0',
      attributes: {
        unit_of_measurement: '°C',
        battery: 85
      },
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }

    const wrapper = mount(SensorCard, {
      props: { entity }
    })

    expect(wrapper.text()).toContain('85%')
  })
})
