import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ShoppingListView from './ShoppingListView.vue'
import { apiClient } from '../api/client'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

describe('ShoppingListView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders page title and initial or fetched items', async () => {
    const mockItems = [
      {
        id: 'item-1',
        name: 'Pane Integrale',
        quantity: '500g',
        category: 'alimentari',
        completed: false,
        added_by: 'Marco',
        created_at: 'Oggi'
      },
      {
        id: 'item-2',
        name: 'Mela Rossa',
        quantity: '1kg',
        category: 'frutta_verdura',
        completed: true,
        added_by: 'Chiara',
        completed_by: 'Marco',
        created_at: 'Ieri'
      }
    ]
    vi.mocked(apiClient.get).mockResolvedValue(mockItems)

    const wrapper = mount(ShoppingListView)
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('h1.page-title').text()).toBe('Lista della Spesa Condivisa')
    expect(wrapper.text()).toContain('Pane Integrale')
    expect(wrapper.text()).toContain('Mela Rossa')
  })

  it('allows adding a new item and calls apiClient.post', async () => {
    vi.mocked(apiClient.get).mockResolvedValue([])
    vi.mocked(apiClient.post).mockResolvedValue({
      id: 'item-new',
      name: 'Olio Extravergine',
      quantity: '1L',
      category: 'alimentari',
      completed: false,
      added_by: 'Io',
      created_at: 'Adesso'
    })

    const wrapper = mount(ShoppingListView)
    await wrapper.vm.$nextTick()

    const nameInput = wrapper.find('input.input-name')
    const qtyInput = wrapper.find('input.input-qty')
    await nameInput.setValue('Olio Extravergine')
    await qtyInput.setValue('1L')

    await wrapper.find('form.add-form').trigger('submit')
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/shopping', expect.objectContaining({
      name: 'Olio Extravergine',
      quantity: '1L',
      category: 'alimentari',
      added_by: 'Io'
    }))
    expect(wrapper.text()).toContain('Olio Extravergine')
  })

  it('toggles item status and clears completed items', async () => {
    const mockItems = [
      {
        id: 'item-10',
        name: 'Sapone Mani',
        quantity: '1 flacone',
        category: 'igiene',
        completed: false,
        added_by: 'Io',
        created_at: 'Oggi'
      }
    ]
    vi.mocked(apiClient.get).mockResolvedValue(mockItems)
    vi.mocked(apiClient.post).mockResolvedValue({})

    const wrapper = mount(ShoppingListView)
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    // Toggle the checkbox button
    const toggleBtn = wrapper.find('.checkbox-btn')
    expect(toggleBtn.exists()).toBe(true)
    await toggleBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/shopping/item-10/toggle', expect.objectContaining({
      completed_by: 'Io'
    }))

    // Clear completed items button should appear
    const clearBtn = wrapper.find('.btn-clear-completed')
    expect(clearBtn.exists()).toBe(true)
    await clearBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/shopping/clear-completed', {})
  })

  it('deletes an item when delete button is clicked', async () => {
    const mockItems = [
      {
        id: 'item-to-delete',
        name: 'Detersivo Piatti',
        quantity: '1L',
        category: 'casa',
        completed: false,
        added_by: 'Io',
        created_at: 'Oggi'
      }
    ]
    vi.mocked(apiClient.get).mockResolvedValue(mockItems)
    vi.mocked(apiClient.delete).mockResolvedValue({})

    const wrapper = mount(ShoppingListView)
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    const deleteBtn = wrapper.find('.btn-delete')
    expect(deleteBtn.exists()).toBe(true)
    await deleteBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(apiClient.delete).toHaveBeenCalledWith('/api/v1/shopping/item-to-delete')
    expect(wrapper.text()).not.toContain('Detersivo Piatti')
  })

  it('opens receipt modal, parses receipt, and allows matching items', async () => {
    const mockItems = [
      {
        id: 'item-pasta',
        name: 'Pasta Penne',
        quantity: '500g',
        category: 'alimentari',
        completed: false,
        added_by: 'Io',
        created_at: 'Oggi'
      }
    ]
    vi.mocked(apiClient.get).mockResolvedValue(mockItems)

    const wrapper = mount(ShoppingListView)
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    // Find and click the Scan Receipt button
    const scanBtn = wrapper.find('.btn-scan-receipt')
    expect(scanBtn.exists()).toBe(true)
    await scanBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.receipt-modal-backdrop').exists()).toBe(true)
    expect(wrapper.text()).toContain('Scansiona Scontrino Spesa')
    expect(wrapper.find('#receipt-image-input').exists()).toBe(true)
    expect(wrapper.find('#receipt-image-input').attributes('capture')).toBe('environment')

    // Mock receipt parsing
    vi.mocked(apiClient.post).mockResolvedValueOnce({
      store_name: 'Esselunga',
      date: '2026-10-18',
      items: [
        { name: 'Pasta Penne Rigate', quantity: 1, unit: 'pz', total_price: 1.30, category: 'pantry' }
      ],
      total_amount: 1.30,
      confidence: 0.95
    })

    const textarea = wrapper.find('.receipt-textarea')
    await textarea.setValue('ESSELUNGA PASTA PENNE 1.30')
    const parseBtn = wrapper.find('.scanner-input-card button.btn-primary')
    await parseBtn.trigger('click')
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/receipts/parse', expect.objectContaining({
      raw_text: 'ESSELUNGA PASTA PENNE 1.30'
    }))
  })
})
