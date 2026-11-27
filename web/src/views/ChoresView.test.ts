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

  it('allows a configurable recurrence interval and persists it in the create payload', async () => {
    const choresStore = useChoresStore()
    const areasStore = useAreasStore()

    vi.spyOn(choresStore, 'fetchChores').mockResolvedValue([])
    vi.spyOn(choresStore, 'createChore').mockResolvedValue({
      id: 'c-2',
      title: 'Pulire i filtri',
      priority: 'medium',
      recurrence_type: 'weekly',
      recurrence_rule: 'INTERVAL=2',
      status: 'pending',
      created_at: '',
      updated_at: ''
    })
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue([])

    const wrapper = mount(ChoresView)
    await wrapper.find('.header-action-group .btn-primary').trigger('click')

    expect(wrapper.find('#chore-recurrence-interval').exists()).toBe(false)
    await wrapper.find('#chore-title').setValue('Pulire i filtri')
    await wrapper.find('#chore-recurrence').setValue('weekly')
    await wrapper.find('#chore-recurrence-interval').setValue('2')
    await wrapper.find('form.modal-body').trigger('submit')

    expect(choresStore.createChore).toHaveBeenCalledWith(expect.objectContaining({
      title: 'Pulire i filtri',
      recurrence_type: 'weekly',
      recurrence_rule: 'INTERVAL=2'
    }))
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
