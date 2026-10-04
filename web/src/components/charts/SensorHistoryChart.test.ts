import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import SensorHistoryChart from './SensorHistoryChart.vue'
import { i18n } from '@/locales'

// Mock Chart.js constructor
vi.mock('chart.js/auto', () => {
  return {
    default: class MockChart {
      destroy = vi.fn()
      update = vi.fn()
    }
  }
})

describe('SensorHistoryChart.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders empty state when history is empty', () => {
    const wrapper = mount(SensorHistoryChart, {
      props: {
        history: [],
        unit: '°C'
      },
      global: {
        plugins: [i18n]
      }
    })

    expect(wrapper.find('.empty-state').exists()).toBe(true)
    expect(wrapper.text()).toContain('Nessun dato registrato')
  })

  it('renders canvas and initializes chart when history is provided', () => {
    const wrapper = mount(SensorHistoryChart, {
      props: {
        history: [
          { entity_id: 'sensor.temp', state: '20.0', recorded_at: '2026-10-04T10:00:00Z' },
          { entity_id: 'sensor.temp', state: '21.5', recorded_at: '2026-10-04T11:00:00Z' }
        ],
        unit: '°C',
        label: 'Temperatura Soggiorno'
      },
      global: {
        plugins: [i18n]
      }
    })

    expect(wrapper.find('canvas').exists()).toBe(true)
    expect(wrapper.find('.empty-state').exists()).toBe(false)
  })

  it('calculates numerical data points correctly', () => {
    const wrapper = mount(SensorHistoryChart, {
      props: {
        history: [
          { entity_id: 'sensor.humidity', state: '55', recorded_at: '2026-10-04T10:00:00Z' },
          { entity_id: 'sensor.humidity', state: '60', recorded_at: '2026-10-04T10:30:00Z' }
        ],
        unit: '%'
      },
      global: {
        plugins: [i18n]
      }
    })

    const vm = wrapper.vm as any
    const points = vm.processedPoints
    expect(points).toHaveLength(2)
    expect(points[0].value).toBe(55)
    expect(points[1].value).toBe(60)
  })
})
