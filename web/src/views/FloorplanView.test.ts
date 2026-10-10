import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import FloorplanView from './FloorplanView.vue'
import { useFloorplanStore } from '../stores/floorplan'
import { useEntitiesStore } from '../stores/entities'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

describe('FloorplanView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders floor tabs and canvas', async () => {
    const floorStore = useFloorplanStore()
    const entitiesStore = useEntitiesStore()

    vi.spyOn(floorStore, 'fetchFloors').mockResolvedValue([
      { id: 'f-1', name: 'Piano Terra', level: 0, sort_order: 0, created_at: '', updated_at: '' }
    ])
    vi.spyOn(floorStore, 'fetchMarkers').mockResolvedValue([])
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])

    const wrapper = mount(FloorplanView)

    expect(wrapper.find('h1').text()).toBe('Mappa della Casa')
    expect(wrapper.find('.floor-tab-btn').exists()).toBe(true)
  })

  it('toggles edit mode when button is clicked', async () => {
    const floorStore = useFloorplanStore()
    const entitiesStore = useEntitiesStore()

    vi.spyOn(floorStore, 'fetchFloors').mockResolvedValue([])
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])

    const wrapper = mount(FloorplanView)

    const toggleBtn = wrapper.find('.btn-mode')
    expect(toggleBtn.text()).toContain('Modifica Posizioni')

    await toggleBtn.trigger('click')
    expect(toggleBtn.text()).toContain('Termina Modifica')
    expect(wrapper.find('.edit-sidebar').exists()).toBe(true)
  })
})
