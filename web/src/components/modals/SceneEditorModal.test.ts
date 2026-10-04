import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import SceneEditorModal from './SceneEditorModal.vue'

describe('SceneEditorModal.vue', () => {
  it('renders modal with initial values for editing', () => {
    const existingScene = {
      id: 'scene_test',
      name: 'Modal Test Scene',
      icon: 'film',
      actions: [
        { entity_key: 'light.lamp', service: 'turn_on', params: {} }
      ],
      created_at: '',
      updated_at: ''
    }

    const wrapper = mount(SceneEditorModal, {
      props: {
        isOpen: true,
        scene: existingScene
      }
    })

    expect((wrapper.find('input[name="scene-name"]').element as HTMLInputElement).value).toBe('Modal Test Scene')
    expect(wrapper.findAll('.action-row')).toHaveLength(1)
  })

  it('validates that name and at least one action are present before saving', async () => {
    const wrapper = mount(SceneEditorModal, {
      props: {
        isOpen: true
      }
    })

    // Try submit with empty name
    await wrapper.find('form').trigger('submit.prevent')
    expect(wrapper.text()).toContain('Il nome della scena è obbligatorio')
    expect(wrapper.emitted('save')).toBeUndefined()

    // Fill name but no actions
    await wrapper.find('input[name="scene-name"]').setValue('Valid Name')
    await wrapper.find('form').trigger('submit.prevent')
    expect(wrapper.text()).toContain('Aggiungi almeno un\'azione')
    expect(wrapper.emitted('save')).toBeUndefined()
  })

  it('adds an action row and emits save with valid data', async () => {
    const wrapper = mount(SceneEditorModal, {
      props: {
        isOpen: true
      }
    })

    await wrapper.find('input[name="scene-name"]').setValue('Cinema Time')
    await wrapper.find('.btn-add-action').trigger('click')

    const entityInputs = wrapper.findAll('input[name="action-entity"]')
    expect(entityInputs).toHaveLength(1)
    await entityInputs[0].setValue('light.living_room')

    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.emitted('save')).toHaveLength(1)
    const saved = wrapper.emitted('save')?.[0]?.[0] as any
    expect(saved.name).toBe('Cinema Time')
    expect(saved.actions).toHaveLength(1)
    expect(saved.actions[0].entity_key).toBe('light.living_room')
  })

  it('emits close when cancel is clicked', async () => {
    const wrapper = mount(SceneEditorModal, {
      props: {
        isOpen: true
      }
    })

    await wrapper.find('.btn-cancel').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
