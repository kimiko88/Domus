import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import LongTermStatisticsCard from './LongTermStatisticsCard.vue'

describe('LongTermStatisticsCard.vue', () => {
  it('renders title, metric summaries and aggregate bars', () => {
    const wrapper = mount(LongTermStatisticsCard, {
      props: {
        title: 'Consumo Mensile Pompa di Calore',
        entityKey: 'sensor.heatpump_energy',
        unit: 'kWh',
      },
    })

    expect(wrapper.text()).toContain('Consumo Mensile Pompa di Calore')
    expect(wrapper.text()).toContain('sensor.heatpump_energy')
    expect(wrapper.findAll('.range-btn')).toHaveLength(4)
    expect(wrapper.findAll('.metric-box')).toHaveLength(4)
    expect(wrapper.find('svg.stats-svg').exists()).toBe(true)
  })

  it('emits range-change event when button is clicked', async () => {
    const wrapper = mount(LongTermStatisticsCard, {
      props: {
        title: 'Statistiche',
      },
    })

    const weekBtn = wrapper.findAll('.range-btn')[1]
    await weekBtn.trigger('click')

    expect(wrapper.emitted('range-change')).toBeTruthy()
    expect(wrapper.emitted('range-change')![0]).toEqual(['week'])
  })
})
