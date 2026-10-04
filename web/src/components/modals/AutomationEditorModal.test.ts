import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import AutomationEditorModal from './AutomationEditorModal.vue'

describe('AutomationEditorModal.vue', () => {
  it('renders modal with initial values for editing', () => {
    const existing = {
      id: 'auto_1',
      name: 'Night Mode',
      description: 'Turn off all',
      enabled: true,
      execution_mode: 'single' as const,
      cooldown_seconds: 5,
      triggers: [{ platform: 'time', at: '23:00:00' }],
      conditions: [],
      actions: [{ type: 'service', service: 'turn_off', target_entity_key: 'light.all' }],
      created_at: '',
      updated_at: ''
    }

    const wrapper = mount(AutomationEditorModal, {
      props: {
        isOpen: true,
        automation: existing
      }
    })

    expect((wrapper.find('input[name="auto-name"]').element as HTMLInputElement).value).toBe('Night Mode')
    expect((wrapper.find('textarea[name="dsl-json"]').element as HTMLTextAreaElement).value).toContain('23:00:00')
  })

  it('detects and displays JSON syntax errors', async () => {
    const wrapper = mount(AutomationEditorModal, {
      props: {
        isOpen: true
      }
    })

    await wrapper.find('input[name="auto-name"]').setValue('Test')
    await wrapper.find('textarea[name="dsl-json"]').setValue('{ invalid json here')
    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.text()).toContain('Sintassi JSON non valida')
    expect(wrapper.emitted('save')).toBeUndefined()
  })

  it('validates triggers and actions invariants', async () => {
    const wrapper = mount(AutomationEditorModal, {
      props: {
        isOpen: true
      }
    })

    await wrapper.find('input[name="auto-name"]').setValue('Test')
    // Valid JSON but missing triggers or actions
    await wrapper.find('textarea[name="dsl-json"]').setValue(JSON.stringify({
      triggers: [],
      conditions: [],
      actions: []
    }))
    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.text()).toContain('Almeno un trigger è richiesto')
    expect(wrapper.emitted('save')).toBeUndefined()
  })

  it('emits save with parsed valid data', async () => {
    const wrapper = mount(AutomationEditorModal, {
      props: {
        isOpen: true
      }
    })

    await wrapper.find('input[name="auto-name"]').setValue('Morning Sunlight')
    await wrapper.find('textarea[name="dsl-json"]').setValue(JSON.stringify({
      triggers: [{ platform: 'time', at: '07:00:00' }],
      conditions: [],
      actions: [{ type: 'service', service: 'turn_on', target_entity_key: 'light.bedroom' }]
    }))

    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.emitted('save')).toHaveLength(1)
    const saved = wrapper.emitted('save')?.[0]?.[0] as any
    expect(saved.name).toBe('Morning Sunlight')
    expect(saved.triggers).toHaveLength(1)
    expect(saved.actions).toHaveLength(1)
  })
})
