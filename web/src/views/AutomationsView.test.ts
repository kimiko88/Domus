import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AutomationsView from './AutomationsView.vue'
import { useAutomationsStore } from '../stores/automations'
import type { Automation } from '../api/types'

describe('AutomationsView.vue', () => {
  let automationsStore: ReturnType<typeof useAutomationsStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    automationsStore = useAutomationsStore()
  })

  it('renders automations list and controls', async () => {
    const mockAutomations: Automation[] = [
      {
        id: 'auto_1',
        name: 'Motion in Corridor',
        description: 'Turns on light for 2 mins',
        enabled: true,
        execution_mode: 'restart',
        cooldown_seconds: 5,
        triggers: [{ platform: 'state', entity_key: 'binary_sensor.motion' }],
        conditions: [],
        actions: [{ type: 'service', service: 'turn_on', target_entity_key: 'light.corridor' }],
        created_at: '',
        updated_at: ''
      },
      {
        id: 'auto_2',
        name: 'Morning Routine',
        description: 'Coffee and blinds up',
        enabled: false,
        execution_mode: 'single',
        cooldown_seconds: 0,
        triggers: [{ platform: 'time', at: '07:00:00' }],
        conditions: [],
        actions: [],
        created_at: '',
        updated_at: ''
      }
    ]

    vi.spyOn(automationsStore, 'fetchAutomations').mockResolvedValue(mockAutomations)
    automationsStore.automations = mockAutomations

    const wrapper = mount(AutomationsView)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Motion in Corridor')
    expect(wrapper.text()).toContain('Morning Routine')
    expect(wrapper.find('.btn-new-automation').exists()).toBe(true)
  })

  it('filters automations by active/disabled status and search query', async () => {
    const mockAutomations: Automation[] = [
      {
        id: 'auto_1',
        name: 'Active Rule',
        enabled: true,
        execution_mode: 'single',
        cooldown_seconds: 0,
        triggers: [],
        conditions: [],
        actions: [],
        created_at: '',
        updated_at: ''
      },
      {
        id: 'auto_2',
        name: 'Disabled Rule',
        enabled: false,
        execution_mode: 'single',
        cooldown_seconds: 0,
        triggers: [],
        conditions: [],
        actions: [],
        created_at: '',
        updated_at: ''
      }
    ]

    automationsStore.automations = mockAutomations

    const wrapper = mount(AutomationsView)
    await wrapper.vm.$nextTick()

    // Filter to active only
    const activePill = wrapper.findAll('.filter-pill').find((p) => p.text().includes('Attive'))
    expect(activePill).toBeDefined()
    await activePill!.trigger('click')

    expect(wrapper.text()).toContain('Active Rule')
    expect(wrapper.text()).not.toContain('Disabled Rule')

    // Search query
    const searchInput = wrapper.find('input[type="search"]')
    await searchInput.setValue('nonexistent')

    expect(wrapper.text()).not.toContain('Active Rule')
  })

  it('opens editor modal when new automation button is clicked', async () => {
    const wrapper = mount(AutomationsView)
    await wrapper.vm.$nextTick()

    await wrapper.find('.btn-new-automation').trigger('click')

    expect(wrapper.findComponent({ name: 'AutomationEditorModal' }).props('isOpen')).toBe(true)
  })

  it('opens runs history modal when view-runs event is emitted from card', async () => {
    const mockAutomations: Automation[] = [
      {
        id: 'auto_1',
        name: 'Test History',
        enabled: true,
        execution_mode: 'single',
        cooldown_seconds: 0,
        triggers: [],
        conditions: [],
        actions: [],
        created_at: '',
        updated_at: ''
      }
    ]
    automationsStore.automations = mockAutomations

    const wrapper = mount(AutomationsView)
    await wrapper.vm.$nextTick()

    const card = wrapper.findComponent({ name: 'AutomationCard' })
    card.vm.$emit('view-runs', mockAutomations[0])
    await wrapper.vm.$nextTick()

    expect(wrapper.findComponent({ name: 'AutomationRunsModal' }).props('isOpen')).toBe(true)
  })
})
