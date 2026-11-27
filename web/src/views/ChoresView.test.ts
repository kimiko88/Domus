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
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('h1').text()).toBe('Faccende Domestiche')
    expect(wrapper.find('.stats-grid').exists()).toBe(true)
    expect(wrapper.text()).toContain('Pulire filtri')
    expect(wrapper.find('.priority-badge').text()).toContain('Alta')
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

  it('opens an existing chore for editing and saves its changes', async () => {
    const choresStore = useChoresStore()
    const areasStore = useAreasStore()
    const existing = {
      id: 'c-edit',
      title: 'Pulire il balcone',
      description: 'Una volta al mese',
      priority: 'medium' as const,
      recurrence_type: 'monthly' as const,
      recurrence_rule: 'INTERVAL=1',
      status: 'pending' as const,
      created_at: '',
      updated_at: ''
    }
    vi.spyOn(choresStore, 'fetchChores').mockImplementation(async () => {
      choresStore.chores = [existing]
      return [existing]
    })
    vi.spyOn(choresStore, 'updateChore').mockResolvedValue({
      ...existing,
      title: 'Pulire anche le ringhiere'
    })
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue([])

    const wrapper = mount(ChoresView)
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()
    await wrapper.find('.chore-main').trigger('click')
    expect(wrapper.find('#modal-title').text()).toBe('Modifica faccenda domestica')

    await wrapper.find('#chore-title').setValue('Pulire anche le ringhiere')
    await wrapper.find('#chore-recurrence-interval').setValue('2')
    await wrapper.find('form.modal-body').trigger('submit')

    expect(choresStore.updateChore).toHaveBeenCalledWith('c-edit', expect.objectContaining({
      title: 'Pulire anche le ringhiere',
      recurrence_type: 'monthly',
      recurrence_rule: 'INTERVAL=2'
    }))
  })

  it('completes a pending chore from its check circle', async () => {
    const choresStore = useChoresStore()
    const areasStore = useAreasStore()
    const pending = {
      id: 'c-pending',
      title: 'Annaffiare le piante',
      priority: 'low' as const,
      recurrence_type: 'none' as const,
      status: 'pending' as const,
      created_at: '',
      updated_at: ''
    }
    vi.spyOn(choresStore, 'fetchChores').mockImplementation(async () => {
      choresStore.chores = [pending]
      return [pending]
    })
    const completeSpy = vi.spyOn(choresStore, 'completeChore').mockResolvedValue()
    const reopenSpy = vi.spyOn(choresStore, 'reopenChore').mockResolvedValue()
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue([])

    const wrapper = mount(ChoresView)
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()
    await wrapper.find('.check-btn').trigger('click')

    expect(completeSpy).toHaveBeenCalledWith('c-pending')
    expect(reopenSpy).not.toHaveBeenCalled()
  })

  it('reopens a completed chore when its check circle is clicked again', async () => {
    const choresStore = useChoresStore()
    const areasStore = useAreasStore()
    const completed = {
      id: 'c-completed',
      title: 'Pulire i filtri',
      priority: 'high' as const,
      recurrence_type: 'weekly' as const,
      status: 'completed' as const,
      created_at: '',
      updated_at: ''
    }
    vi.spyOn(choresStore, 'fetchChores').mockImplementation(async () => {
      choresStore.chores = [completed]
      return [completed]
    })
    const reopenSpy = vi.spyOn(choresStore, 'reopenChore').mockResolvedValue()
    const completeSpy = vi.spyOn(choresStore, 'completeChore').mockResolvedValue()
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue([])

    const wrapper = mount(ChoresView)
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()
    await wrapper.find('.check-btn').trigger('click')

    expect(reopenSpy).toHaveBeenCalledWith('c-completed')
    expect(completeSpy).not.toHaveBeenCalled()
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
