import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import MedicinesView from './MedicinesView.vue'
import { apiClient } from '../api/client'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

describe('MedicinesView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const mockMedicines = [
    {
      id: 'med-1',
      name: 'Tachipirina 1000 mg',
      active_ingredient: 'Paracetamolo',
      dosage: '1000 mg',
      form: 'compresse',
      aic: '034234035',
      expiration_date: '2028-10-31',
      pao_days: 0,
      effective_expiration_date: '2028-10-31',
      location: 'armadietto_bagno',
      requires_cold_chain: false,
      quantity: 16,
      unit: 'compresse',
      min_threshold: 4,
      prescription_required: false,
      created_at: '2026-10-18T00:00:00Z',
      updated_at: '2026-10-18T00:00:00Z'
    },
    {
      id: 'med-2',
      name: 'Tobral Collirio 5ml',
      active_ingredient: 'Tobramicina',
      dosage: '0.3%',
      form: 'collirio',
      expiration_date: '2028-12-31',
      pao_days: 28,
      opened_at: '2026-10-01',
      effective_expiration_date: '2026-10-29',
      location: 'frigo',
      requires_cold_chain: true,
      quantity: 1,
      unit: 'ml',
      min_threshold: 0,
      prescription_required: true,
      created_at: '2026-10-18T00:00:00Z',
      updated_at: '2026-10-18T00:00:00Z'
    }
  ]

  it('renders page header and medicine cards', async () => {
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/api/v1/medicines') return Promise.resolve(mockMedicines)
      if (url === '/api/v1/medicines/alerts') return Promise.resolve([])
      return Promise.resolve([])
    })

    const wrapper = mount(MedicinesView)
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('h1.page-title').text()).toBe('Armadietto Farmaci & Scadenze')
    expect(wrapper.text()).toContain('Tachipirina 1000 mg')
    expect(wrapper.text()).toContain('Tobral Collirio 5ml')
    expect(wrapper.text()).toContain('Frigo 2-8°C')
  })

  it('opens quick add modal and fills Datamatrix code', async () => {
    vi.mocked(apiClient.get).mockResolvedValue([])
    vi.mocked(apiClient.post).mockImplementation((url: string) => {
      if (url === '/api/v1/medicines/parse-code') {
        return Promise.resolve({
          aic: '034234035',
          expiration_date: '2028-04-30',
          lot_number: 'LOT789',
          raw_code: 'sample'
        })
      }
      return Promise.resolve({})
    })

    const wrapper = mount(MedicinesView)
    await wrapper.vm.$nextTick()

    // Click Inserimento Rapido button
    const addBtn = wrapper.find('.btn-primary-add')
    await addBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.modal-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('Scansione Datamatrix GS1 o Codice AIC')

    // Trigger decodifica
    const codeInput = wrapper.find('.input-code')
    await codeInput.setValue('034234035')
    const parseBtn = wrapper.find('.btn-parse-code')
    await parseBtn.trigger('click')
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/medicines/parse-code', { code: '034234035' })
  })

  it('allows consuming a dose and calls apiClient.post', async () => {
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/api/v1/medicines') return Promise.resolve(mockMedicines)
      return Promise.resolve([])
    })
    vi.mocked(apiClient.post).mockResolvedValue({
      ...mockMedicines[0],
      quantity: 15
    })

    const wrapper = mount(MedicinesView)
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    const consumeBtn = wrapper.find('.btn-consume-dose')
    expect(consumeBtn.exists()).toBe(true)
    await consumeBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/medicines/med-1/consume', { amount: 1 })
  })
})
