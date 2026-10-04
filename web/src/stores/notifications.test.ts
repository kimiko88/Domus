import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useNotificationsStore } from './notifications'

describe('useNotificationsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
  })

  it('initializes with empty notifications', () => {
    const store = useNotificationsStore()
    expect(store.notifications).toEqual([])
  })

  it('adds notification with unique id and default duration', () => {
    const store = useNotificationsStore()
    const id = store.addNotification({
      title: 'Device Online',
      message: 'Living room light is online',
      type: 'success'
    })

    expect(store.notifications.length).toBe(1)
    expect(store.notifications[0].id).toBe(id)
    expect(store.notifications[0].title).toBe('Device Online')
    expect(store.notifications[0].type).toBe('success')
  })

  it('removes notification by id manually', () => {
    const store = useNotificationsStore()
    const id = store.addNotification({
      title: 'Alert',
      type: 'error'
    })
    expect(store.notifications.length).toBe(1)

    store.removeNotification(id)
    expect(store.notifications.length).toBe(0)
  })

  it('auto-removes notification after specified timeout', () => {
    const store = useNotificationsStore()
    store.addNotification({
      title: 'Auto-dismiss toast',
      type: 'info',
      timeout: 3000
    })

    expect(store.notifications.length).toBe(1)
    vi.advanceTimersByTime(2999)
    expect(store.notifications.length).toBe(1)

    vi.advanceTimersByTime(1)
    expect(store.notifications.length).toBe(0)
  })

  it('supports helper methods success, error, info, warning', () => {
    const store = useNotificationsStore()
    store.success('Success Title', 'Success message')
    store.error('Error Title', 'Error message')
    store.info('Info Title', 'Info message')
    store.warning('Warning Title', 'Warning message')

    expect(store.notifications.length).toBe(4)
    expect(store.notifications[0].type).toBe('success')
    expect(store.notifications[1].type).toBe('error')
    expect(store.notifications[2].type).toBe('info')
    expect(store.notifications[3].type).toBe('warning')
  })
})
