import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseCard from './BaseCard.vue'

describe('BaseCard.vue', () => {
  it('renders default slot', () => {
    const wrapper = mount(BaseCard, {
      slots: {
        default: '<div class="test-content">Card Body</div>'
      }
    })
    expect(wrapper.find('.test-content').text()).toBe('Card Body')
    expect(wrapper.classes()).toContain('base-card')
  })

  it('renders header and footer slots when provided', () => {
    const wrapper = mount(BaseCard, {
      slots: {
        header: '<h3>Title</h3>',
        default: 'Main Content',
        footer: '<button>Action</button>'
      }
    })
    expect(wrapper.find('.card-header').text()).toBe('Title')
    expect(wrapper.find('.card-footer').text()).toBe('Action')
  })

  it('applies active and clickable modifier classes', () => {
    const wrapper = mount(BaseCard, {
      props: {
        active: true,
        clickable: true
      }
    })
    expect(wrapper.classes()).toContain('is-active')
    expect(wrapper.classes()).toContain('is-clickable')
  })

  it('emits click event when clickable is true', async () => {
    const wrapper = mount(BaseCard, {
      props: {
        clickable: true
      }
    })
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toHaveLength(1)
  })
})
