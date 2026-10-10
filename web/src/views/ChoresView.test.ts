import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ChoresView from './ChoresView.vue'
import { useChoresStore } from '../stores/chores'
import { useAreasStore } from '../stores/areas'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

describe('ChoresView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders chores list and stats', async () => {
    const choresStore = useChoresStore()
    const areasStore = useAreasStore()

    const mockData = [
      {
        id: 'c-1',
        title: 'Pulire filtri',
        status: 'pending' as const,
        priority: 'high' as const,
        recurrence_type: 'weekly' as const,
        created_at: '',
        updated_at: ''
      }
    ]
    vi.spyOn(choresStore, 'fetchChores').mockImplementation(async () => {
      choresStore.chores = mockData
      return mockData
    })
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue([])

    const wrapper = mount(ChoresView)
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('h1').text()).toBe('Faccende Domestiche')
    expect(wrapper.find('.stats-grid').exists()).toBe(true)
    expect(wrapper.text()).toContain('Pulire filtri')
  })

  it('opens create modal on button click', async () => {
    const choresStore = useChoresStore()
    const areasStore = useAreasStore()

    vi.spyOn(choresStore, 'fetchChores').mockResolvedValue([])
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue([])

    const wrapper = mount(ChoresView)

    expect(wrapper.find('.modal-card').exists()).toBe(false)
    await wrapper.find('.btn-primary').trigger('click')
    expect(wrapper.find('.modal-card').exists()).toBe(true)
  })
})
