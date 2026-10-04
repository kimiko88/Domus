import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AutomationCard from './AutomationCard.vue'
import { useAutomationsStore } from '../../stores/automations'
import type { Automation } from '../../api/types'

describe('AutomationCard.vue', () => {
  let automationsStore: ReturnType<typeof useAutomationsStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    automationsStore = useAutomationsStore()
  })

  it('renders automation name, mode, cooldown, and last status badge', () => {
    const automation: Automation = {
      id: 'auto_1',
      name: 'Hallway Motion Lights',
      description: 'Turns on lights when motion is detected',
      enabled: true,
      execution_mode: 'restart',
      cooldown_seconds: 15,
      triggers: [{ platform: 'state', entity_key: 'binary_sensor.motion' }],
      conditions: [],
      actions: [{ type: 'service', service: 'turn_on', target_entity_key: 'light.hall' }],
      last_run_at: '2026-10-04T10:00:00Z',
      last_status: 'completed',
      created_at: '',
      updated_at: ''
    }

    const wrapper = mount(AutomationCard, {
      props: { automation }
    })

    expect(wrapper.text()).toContain('Hallway Motion Lights')
    expect(wrapper.text()).toContain('Turns on lights when motion is detected')
    expect(wrapper.text()).toContain('restart')
    expect(wrapper.text()).toContain('15s')
    expect(wrapper.text()).toContain('completed')
  })

  it('toggles automation enabled state when toggle button is clicked', async () => {
    const automation: Automation = {
      id: 'auto_toggle',
      name: 'Auto Toggle',
      enabled: true,
      execution_mode: 'single',
      cooldown_seconds: 0,
      triggers: [],
      conditions: [],
      actions: [],
      created_at: '',
      updated_at: ''
    }

    const toggleSpy = vi.spyOn(automationsStore, 'toggleAutomation').mockResolvedValue(false)

    const wrapper = mount(AutomationCard, {
      props: { automation }
    })

    const toggleBtn = wrapper.find('.toggle-btn')
    await toggleBtn.trigger('click')

    expect(toggleSpy).toHaveBeenCalledWith('auto_toggle')
  })

  it('triggers manual run when test button is clicked', async () => {
    const automation: Automation = {
      id: 'auto_trigger',
      name: 'Manual Run',
      enabled: true,
      execution_mode: 'single',
      cooldown_seconds: 0,
      triggers: [],
      conditions: [],
      actions: [],
      created_at: '',
      updated_at: ''
    }

    const triggerSpy = vi.spyOn(automationsStore, 'triggerAutomation').mockResolvedValue(undefined)

    const wrapper = mount(AutomationCard, {
      props: { automation }
    })

    const testBtn = wrapper.find('.btn-trigger')
    await testBtn.trigger('click')

    expect(triggerSpy).toHaveBeenCalledWith('auto_trigger')
  })

  it('emits view-runs, edit, and delete events', async () => {
    const automation: Automation = {
      id: 'auto_events',
      name: 'Events Test',
      enabled: false,
      execution_mode: 'single',
      cooldown_seconds: 0,
      triggers: [],
      conditions: [],
      actions: [],
      created_at: '',
      updated_at: ''
    }

    const wrapper = mount(AutomationCard, {
      props: { automation }
    })

    await wrapper.find('.btn-runs').trigger('click')
    expect(wrapper.emitted('view-runs')?.[0]).toEqual([automation])

    await wrapper.find('.btn-edit').trigger('click')
    expect(wrapper.emitted('edit')?.[0]).toEqual([automation])

    await wrapper.find('.btn-delete').trigger('click')
    expect(wrapper.emitted('delete')?.[0]).toEqual([automation])
  })
})
