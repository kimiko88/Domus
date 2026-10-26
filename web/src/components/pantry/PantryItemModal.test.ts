import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import PantryItemModal from './PantryItemModal.vue'
import { usePantryStore } from '../../stores/pantry'
import type { PantryItem } from '../../api/types'

describe('PantryItemModal.vue - Autocomplete & Threshold Confirmation', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const sampleItems: PantryItem[] = [
    {
      id: 'item-1',
      name: 'Mele Golden',
      category: 'produce',
      quantity: 5,
      unit: 'kg',
      min_threshold: 2,
      location: 'pantry',
      expiration_date: '2026-10-28',
      created_at: '2026-10-20T00:00:00Z',
      updated_at: '2026-10-20T00:00:00Z'
    },
    {
      id: 'item-2',
      name: 'Mele Golden',
      category: 'produce',
      quantity: 3,
      unit: 'kg',
      min_threshold: 2,
      location: 'pantry',
      expiration_date: '2026-11-05',
      created_at: '2026-10-24T00:00:00Z',
      updated_at: '2026-10-24T00:00:00Z'
    },
    {
      id: 'item-3',
      name: 'Latte Intero',
      category: 'dairy',
      quantity: 2,
      unit: 'l',
      min_threshold: 1,
      location: 'fridge',
      expiration_date: '2026-11-02',
      created_at: '2026-10-25T00:00:00Z',
      updated_at: '2026-10-25T00:00:00Z'
    }
  ]

  it('renders modal when isOpen is true', () => {
    const store = usePantryStore()
    store.items = sampleItems

    const wrapper = mount(PantryItemModal, {
      props: {
        isOpen: true
      },
      global: {
        stubs: { Teleport: true }
      }
    })

    expect(wrapper.find('h2.modal-title').text()).toBe('Nuovo Alimento in Dispensa')
    expect(wrapper.find('#pantry-name').exists()).toBe(true)
  })

  it('filters and displays autocomplete suggestions based on existing items', async () => {
    const store = usePantryStore()
    store.items = sampleItems

    const wrapper = mount(PantryItemModal, {
      props: {
        isOpen: true
      },
      global: {
        stubs: { Teleport: true }
      }
    })

    const input = wrapper.find('#pantry-name')
    await input.setValue('mel')
    await input.trigger('input')
    await input.trigger('focus')
    await wrapper.vm.$nextTick()

    const dropdown = wrapper.find('.autocomplete-dropdown')
    expect(dropdown.exists()).toBe(true)

    const items = wrapper.findAll('.autocomplete-item')
    expect(items.length).toBe(1)
    expect(items[0].text()).toContain('Mele Golden')
    expect(items[0].text()).toContain('Soglia: 2 kg')
    expect(items[0].text()).toContain('2 lotti')
  })

  it('auto-fills fields when clicking an autocomplete suggestion', async () => {
    const store = usePantryStore()
    store.items = sampleItems

    const wrapper = mount(PantryItemModal, {
      props: {
        isOpen: true
      },
      global: {
        stubs: { Teleport: true }
      }
    })

    const input = wrapper.find('#pantry-name')
    await input.setValue('mel')
    await input.trigger('input')
    await input.trigger('focus')
    await wrapper.vm.$nextTick()

    const suggestion = wrapper.find('.autocomplete-item')
    await suggestion.trigger('mousedown')
    await wrapper.vm.$nextTick()

    // Assert auto-filled fields
    expect((wrapper.find('#pantry-name').element as HTMLInputElement).value).toBe('Mele Golden')
    expect((wrapper.find('#pantry-category').element as HTMLSelectElement).value).toBe('produce')
    expect((wrapper.find('#pantry-unit').element as HTMLSelectElement).value).toBe('kg')
    expect((wrapper.find('#pantry-threshold').element as HTMLInputElement).value).toBe('2')

    // Matching hint banner should be visible
    expect(wrapper.find('.match-hint-banner').exists()).toBe(true)
    expect(wrapper.find('.match-hint-banner').text()).toContain('Alimento già presente in dispensa')
  })

  it('submits directly without prompt if threshold matches established threshold', async () => {
    const store = usePantryStore()
    store.items = sampleItems

    const wrapper = mount(PantryItemModal, {
      props: {
        isOpen: true
      },
      global: {
        stubs: { Teleport: true }
      }
    })

    const input = wrapper.find('#pantry-name')
    await input.setValue('Mele Golden')
    await input.trigger('input')
    await wrapper.vm.$nextTick()

    // Threshold remains at 2 (matches existing item)
    const thresholdInput = wrapper.find('#pantry-threshold')
    await thresholdInput.setValue(2)

    const form = wrapper.find('form')
    await form.trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    // Should NOT show threshold confirmation
    expect(wrapper.find('.threshold-confirm-overlay').exists()).toBe(false)

    // Should emit save event
    expect(wrapper.emitted('save')).toBeTruthy()
    const payload = wrapper.emitted('save')![0][0] as Partial<PantryItem>
    expect(payload.name).toBe('Mele Golden')
    expect(payload.min_threshold).toBe(2)
  })

  it('prompts user when threshold is modified for existing item, and allows keeping established threshold', async () => {
    const store = usePantryStore()
    store.items = sampleItems

    const wrapper = mount(PantryItemModal, {
      props: {
        isOpen: true
      },
      global: {
        stubs: { Teleport: true }
      }
    })

    const input = wrapper.find('#pantry-name')
    await input.setValue('Mele Golden')
    await input.trigger('input')
    await wrapper.vm.$nextTick()

    // User changes threshold to 5 (different from established threshold 2)
    const thresholdInput = wrapper.find('#pantry-threshold')
    await thresholdInput.setValue(5)
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.thresh-diff-badge').exists()).toBe(true)

    // Submit form
    const form = wrapper.find('form')
    await form.trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    // Threshold confirmation overlay MUST be displayed
    const overlay = wrapper.find('.threshold-confirm-overlay')
    expect(overlay.exists()).toBe(true)
    expect(overlay.text()).toContain('Conferma Soglia di Avviso')
    expect(overlay.text()).toContain('2 kg')
    expect(overlay.text()).toContain('5 kg')

    // Click "Mantieni Soglia Esistente"
    const keepBtn = wrapper.find('.btn-thresh-keep')
    await keepBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('save')).toBeTruthy()
    const payload = wrapper.emitted('save')![0][0] as Partial<PantryItem>
    expect(payload.min_threshold).toBe(2)
  })

  it('prompts user and allows confirming the new modified threshold', async () => {
    const store = usePantryStore()
    store.items = sampleItems
    const updateSpy = vi.spyOn(store, 'updatePantryItem').mockResolvedValue({} as any)

    const wrapper = mount(PantryItemModal, {
      props: {
        isOpen: true
      },
      global: {
        stubs: { Teleport: true }
      }
    })

    const input = wrapper.find('#pantry-name')
    await input.setValue('Mele Golden')
    await input.trigger('input')
    await wrapper.vm.$nextTick()

    // Change threshold to 6
    const thresholdInput = wrapper.find('#pantry-threshold')
    await thresholdInput.setValue(6)
    await wrapper.vm.$nextTick()

    // Submit form
    const form = wrapper.find('form')
    await form.trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.threshold-confirm-overlay').exists()).toBe(true)

    // Click "Modifica Soglia a 6 kg"
    const modifyBtn = wrapper.find('.btn-thresh-modify')
    await modifyBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('save')).toBeTruthy()
    const payload = wrapper.emitted('save')![0][0] as Partial<PantryItem>
    expect(payload.min_threshold).toBe(6)

    // Verify other existing batches of Mele Golden are updated to maintain consistent alert threshold
    expect(updateSpy).toHaveBeenCalledWith('item-1', { min_threshold: 6 })
    expect(updateSpy).toHaveBeenCalledWith('item-2', { min_threshold: 6 })
  })
})
