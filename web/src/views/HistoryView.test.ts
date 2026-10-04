import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import HistoryView from './HistoryView.vue'
import { useEntitiesStore } from '@/stores/entities'
import { useHistoryStore } from '@/stores/history'
import { i18n } from '@/locales'

// Mock vue-router
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} })
}))

// Mock Chart.js
vi.mock('chart.js/auto', () => {
  return {
    default: class MockChart {
      destroy = vi.fn()
      update = vi.fn()
    }
  }
})

describe('HistoryView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('renders history view controls and entity selector', async () => {
    const entStore = useEntitiesStore()
    entStore.entities = {
      'sensor.temp': {
        id: 'sensor.temp',
        name: 'Sensore Salotto',
        domain: 'sensor',
        state: '21.5',
        attributes: { unit_of_measurement: '°C' },
        last_changed: '',
        last_updated: ''
      }
    }

    const histStore = useHistoryStore()
    vi.spyOn(histStore, 'fetchHistory').mockResolvedValue([])

    const wrapper = mount(HistoryView, {
      global: {
        plugins: [i18n]
      }
    })

    expect(wrapper.text()).toContain('Storico & Analisi')
    expect(wrapper.find('select.entity-picker').exists()).toBe(true)
    expect(wrapper.findAll('.range-btn')).toHaveLength(4)
  })

  it('computes min, max, average statistics for numerical records', async () => {
    const entStore = useEntitiesStore()
    entStore.entities = {
      'sensor.temp': {
        id: 'sensor.temp',
        name: 'Temperatura',
        domain: 'sensor',
        state: '22.0',
        attributes: { unit_of_measurement: '°C' },
        last_changed: '',
        last_updated: ''
      }
    }

    const histStore = useHistoryStore()
    const mockData = [
      { id: 1, entity_id: 'sensor.temp', state: '18.0', recorded_at: '2026-10-04T10:00:00Z' },
      { id: 2, entity_id: 'sensor.temp', state: '22.0', recorded_at: '2026-10-04T11:00:00Z' }
    ]
    vi.spyOn(histStore, 'fetchHistory').mockImplementation(async (id) => {
      histStore.historyByEntity[id] = mockData
      return mockData
    })

    const wrapper = mount(HistoryView, {
      global: {
        plugins: [i18n]
      }
    })

    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    // Check stat cards
    expect(wrapper.text()).toContain('18')
    expect(wrapper.text()).toContain('22')
    expect(wrapper.text()).toContain('20') // average (18 + 22)/2
  })

  it('switches time range and re-fetches history', async () => {
    const entStore = useEntitiesStore()
    entStore.entities = {
      'sensor.temp': {
        id: 'sensor.temp',
        name: 'Temperatura',
        domain: 'sensor',
        state: '22.0',
        attributes: {},
        last_changed: '',
        last_updated: ''
      }
    }

    const histStore = useHistoryStore()
    const fetchSpy = vi.spyOn(histStore, 'fetchHistory').mockResolvedValue([])

    const wrapper = mount(HistoryView, {
      global: {
        plugins: [i18n]
      }
    })

    await wrapper.vm.$nextTick()
    fetchSpy.mockClear()

    // Click on 7 Days range button (index 2)
    const rangeBtns = wrapper.findAll('.range-btn')
    await rangeBtns[2].trigger('click')

    expect(fetchSpy).toHaveBeenCalledTimes(1)
  })
})
