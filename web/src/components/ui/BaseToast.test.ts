import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import BaseToast from './BaseToast.vue'
import { useNotificationsStore } from '../../stores/notifications'

describe('BaseToast.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders nothing when no notifications', () => {
    const wrapper = mount(BaseToast)
    expect(wrapper.findAll('.toast-item')).toHaveLength(0)
  })

  it('renders toast notifications from store', () => {
    const store = useNotificationsStore()
    store.addNotification({
      title: 'Success!',
      message: 'Action completed',
      type: 'success'
    })

    const wrapper = mount(BaseToast)
    const items = wrapper.findAll('.toast-item')
    expect(items).toHaveLength(1)
    expect(items[0].text()).toContain('Success!')
    expect(items[0].text()).toContain('Action completed')
    expect(items[0].classes()).toContain('toast-success')
  })

  it('removes toast when dismiss button is clicked', async () => {
    const store = useNotificationsStore()
    const id = store.addNotification({
      title: 'Dismiss Me',
      type: 'info'
    })

    const wrapper = mount(BaseToast)
    const dismissBtn = wrapper.find('.toast-dismiss')
    expect(dismissBtn.exists()).toBe(true)

    await dismissBtn.trigger('click')
    expect(store.notifications.find((n) => n.id === id)).toBeUndefined()
  })
})
