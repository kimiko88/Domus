import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseButton from './BaseButton.vue'

describe('BaseButton.vue', () => {
  it('renders default slot text correctly', () => {
    const wrapper = mount(BaseButton, {
      slots: {
        default: 'Click Me'
      }
    })
    expect(wrapper.text()).toContain('Click Me')
    expect(wrapper.classes()).toContain('btn')
    expect(wrapper.classes()).toContain('btn-primary')
  })

  it('applies variant classes correctly', () => {
    const wrapper = mount(BaseButton, {
      props: {
        variant: 'secondary'
      }
    })
    expect(wrapper.classes()).toContain('btn-secondary')

    const dangerWrapper = mount(BaseButton, {
      props: {
        variant: 'danger'
      }
    })
    expect(dangerWrapper.classes()).toContain('btn-danger')

    const ghostWrapper = mount(BaseButton, {
      props: {
        variant: 'ghost'
      }
    })
    expect(ghostWrapper.classes()).toContain('btn-ghost')
  })

  it('shows loading spinner and disables button when loading is true', () => {
    const wrapper = mount(BaseButton, {
      props: {
        loading: true
      },
      slots: {
        default: 'Save'
      }
    })
    expect(wrapper.attributes('disabled')).toBeDefined()
    expect(wrapper.find('.btn-spinner').exists()).toBe(true)
  })

  it('emits click event when clicked while enabled', async () => {
    const wrapper = mount(BaseButton)
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toHaveLength(1)
  })

  it('does not emit click when disabled', async () => {
    const wrapper = mount(BaseButton, {
      props: {
        disabled: true
      }
    })
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toBeUndefined()
  })
})
