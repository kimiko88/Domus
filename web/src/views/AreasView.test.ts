import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AreasView from './AreasView.vue'
import { useAreasStore } from '../stores/areas'
import { useEntitiesStore } from '../stores/entities'

describe('AreasView.vue', () => {
  let areasStore: ReturnType<typeof useAreasStore>
  let entitiesStore: ReturnType<typeof useEntitiesStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    areasStore = useAreasStore()
    entitiesStore = useEntitiesStore()
  })

  it('renders areas and displays their names and entity counts', async () => {
    areasStore.areas = [
      { id: 'area_living', name: 'Soggiorno', order_index: 1, created_at: '', updated_at: '' },
      { id: 'area_kitchen', name: 'Cucina', order_index: 2, created_at: '', updated_at: '' }
    ]

    entitiesStore.entities = {
      'light.living': {
        id: 'light.living',
        domain: 'light',
        name: 'Lampadario Soggiorno',
        state: 'on',
        attributes: { area_id: 'area_living' },
        last_changed: '',
        last_updated: ''
      }
    }

    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue(areasStore.areas)
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])

    const wrapper = mount(AreasView)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Soggiorno')
    expect(wrapper.text()).toContain('Cucina')
    expect(wrapper.text()).toContain('1 Dispositivi/Entità')
  })

  it('opens editor modal when "Nuova Area" button is clicked', async () => {
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue([])
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])

    const wrapper = mount(AreasView)
    await wrapper.vm.$nextTick()

    const newBtn = wrapper.find('.header-actions button')
    expect(newBtn.exists()).toBe(true)
    expect(newBtn.text()).toContain('Nuova Area')

    await newBtn.trigger('click')
    expect(wrapper.find('.modal-backdrop').exists()).toBe(true)
    expect(wrapper.text()).toContain('Crea Area')
  })

  it('opens editor modal when "Crea la prima area" in empty state is clicked', async () => {
    areasStore.areas = []
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue([])
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])

    const wrapper = mount(AreasView)
    await wrapper.vm.$nextTick()

    const emptyBtn = wrapper.find('.empty-btn')
    expect(emptyBtn.exists()).toBe(true)
    await emptyBtn.trigger('click')

    expect(wrapper.find('.modal-backdrop').exists()).toBe(true)
  })

  it('opens modal in edit mode when edit button is clicked', async () => {
    areasStore.areas = [
      { id: 'area_living', name: 'Soggiorno', icon: 'sofa', order_index: 1, created_at: '', updated_at: '' }
    ]
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue(areasStore.areas)
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])

    const wrapper = mount(AreasView)
    await wrapper.vm.$nextTick()

    const editBtn = wrapper.find('.btn-card-action[title="Modifica area"]')
    expect(editBtn.exists()).toBe(true)
    await editBtn.trigger('click')

    expect(wrapper.find('.modal-backdrop').exists()).toBe(true)
    expect(wrapper.text()).toContain('Modifica Area')
    const input = wrapper.find<HTMLInputElement>('#area-name')
    expect(input.element.value).toBe('Soggiorno')
  })

  it('calls deleteArea when delete button is confirmed', async () => {
    areasStore.areas = [
      { id: 'area_living', name: 'Soggiorno', icon: 'sofa', order_index: 1, created_at: '', updated_at: '' }
    ]
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue(areasStore.areas)
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const deleteSpy = vi.spyOn(areasStore, 'deleteArea').mockResolvedValue()

    const wrapper = mount(AreasView)
    await wrapper.vm.$nextTick()

    const deleteBtn = wrapper.find('.btn-card-action.is-danger')
    expect(deleteBtn.exists()).toBe(true)
    await deleteBtn.trigger('click')

    expect(deleteSpy).toHaveBeenCalledWith('area_living')
  })
})
