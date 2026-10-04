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
})
