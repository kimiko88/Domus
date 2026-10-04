import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import AreaEditorModal from './AreaEditorModal.vue'

describe('AreaEditorModal.vue', () => {
  it('does not render when isOpen is false', () => {
    const wrapper = mount(AreaEditorModal, {
      props: { isOpen: false }
    })
    expect(wrapper.find('.modal-backdrop').exists()).toBe(false)
  })

  it('renders correctly in create mode', () => {
    const wrapper = mount(AreaEditorModal, {
      props: { isOpen: true }
    })
    expect(wrapper.find('.modal-backdrop').exists()).toBe(true)
    expect(wrapper.text()).toContain('Nuova Area')
    expect(wrapper.text()).toContain('Crea Area')
  })

  it('renders prefilled fields in edit mode', () => {
    const wrapper = mount(AreaEditorModal, {
      props: {
        isOpen: true,
        area: {
          id: 'area-1',
          name: 'Cucina Rustica',
          icon: 'utensils',
          sort_order: 3,
          created_at: '',
          updated_at: ''
        }
      }
    })
    expect(wrapper.text()).toContain('Modifica Area')
    const input = wrapper.find<HTMLInputElement>('#area-name')
    expect(input.element.value).toBe('Cucina Rustica')
  })

  it('emits save event with correct payload on form submit', async () => {
    const wrapper = mount(AreaEditorModal, {
      props: { isOpen: true }
    })

    const input = wrapper.find<HTMLInputElement>('#area-name')
    await input.setValue('Soggiorno Moderno')

    // Click sofa icon
    const sofaBtn = wrapper.findAll('.icon-option')[0]
    await sofaBtn.trigger('click')

    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.emitted('save')).toBeTruthy()
    const payload = wrapper.emitted('save')![0][0] as any
    expect(payload.name).toBe('Soggiorno Moderno')
    expect(payload.icon).toBe('sofa')
  })

  it('shows error if name is empty', async () => {
    const wrapper = mount(AreaEditorModal, {
      props: { isOpen: true }
    })

    const input = wrapper.find<HTMLInputElement>('#area-name')
    await input.setValue('   ')
    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.emitted('save')).toBeFalsy()
    expect(wrapper.text()).toContain("Il nome dell'area è obbligatorio")
  })
})
