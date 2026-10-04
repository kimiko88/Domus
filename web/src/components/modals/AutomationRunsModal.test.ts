import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AutomationRunsModal from './AutomationRunsModal.vue'
import { useAutomationsStore } from '../../stores/automations'
import type { Automation, AutomationRun } from '../../api/types'

describe('AutomationRunsModal.vue', () => {
  let automationsStore: ReturnType<typeof useAutomationsStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    automationsStore = useAutomationsStore()
  })

  it('fetches and displays execution runs list', async () => {
    const automation: Automation = {
      id: 'auto_runs_1',
      name: 'Sunset Lighting',
      enabled: true,
      execution_mode: 'single',
      cooldown_seconds: 0,
      triggers: [],
      conditions: [],
      actions: [],
      created_at: '',
      updated_at: ''
    }

    const mockRuns: AutomationRun[] = [
      {
        id: 'run_101',
        automation_id: 'auto_runs_1',
        started_at: '2026-10-04T10:00:00Z',
        finished_at: '2026-10-04T10:00:01Z',
        status: 'completed',
        trigger_context: '{"entity_key":"sun.elevation"}',
        log: 'All 3 actions executed successfully',
        duration_ms: 120
      },
      {
        id: 'run_102',
        automation_id: 'auto_runs_1',
        started_at: '2026-10-04T09:00:00Z',
        finished_at: '2026-10-04T09:00:00Z',
        status: 'failed',
        trigger_context: '{}',
        log: 'Timeout connecting to light.terrace',
        duration_ms: 500
      }
    ]

    vi.spyOn(automationsStore, 'fetchRuns').mockResolvedValue(mockRuns)
    automationsStore.runs[automation.id] = mockRuns

    const wrapper = mount(AutomationRunsModal, {
      props: {
        isOpen: true,
        automation
      }
    })

    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Sunset Lighting')
    expect(wrapper.text()).toContain('completed')
    expect(wrapper.text()).toContain('failed')
    expect(wrapper.text()).toContain('120ms')
    expect(wrapper.text()).toContain('Timeout connecting to light.terrace')
  })

  it('emits close when close button is clicked', async () => {
    const automation: Automation = {
      id: 'auto_1',
      name: 'Test',
      enabled: true,
      execution_mode: 'single',
      cooldown_seconds: 0,
      triggers: [],
      conditions: [],
      actions: [],
      created_at: '',
      updated_at: ''
    }

    const wrapper = mount(AutomationRunsModal, {
      props: {
        isOpen: true,
        automation
      }
    })

    await wrapper.find('.btn-close').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
