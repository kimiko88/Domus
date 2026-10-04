import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useWsStore } from './ws'
import { useEntitiesStore } from './entities'
import { useNotificationsStore } from './notifications'

describe('useWsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('initializes with disconnected state', () => {
    const wsStore = useWsStore()
    expect(wsStore.isConnected).toBe(false)
  })

  it('updates connection status on connect and disconnect triggers', () => {
    const wsStore = useWsStore()
    wsStore.setConnected(true)
    expect(wsStore.isConnected).toBe(true)

    wsStore.setConnected(false)
    expect(wsStore.isConnected).toBe(false)
  })

  it('routes entity.state_changed events to entitiesStore', () => {
    const wsStore = useWsStore()
    const entitiesStore = useEntitiesStore()
    const spy = vi.spyOn(entitiesStore, 'handleStateChange')

    const eventPayload = {
      entity_id: 'light.living_room',
      old_state: 'off',
      new_state: 'on',
      old_attributes: {},
      new_attributes: { brightness: 255 },
      timestamp: '2026-10-04T08:00:00Z'
    }

    wsStore.handleIncomingEvent('entity.state_changed', eventPayload)

    expect(spy).toHaveBeenCalledWith(eventPayload)
  })

  it('creates notifications for device online / offline events', () => {
    const wsStore = useWsStore()
    const notificationsStore = useNotificationsStore()

    wsStore.handleIncomingEvent('device.online', { device_id: 'dev1', name: 'Kitchen Sensor' })
    expect(notificationsStore.notifications.length).toBe(1)
    expect(notificationsStore.notifications[0].title).toBe('Device Online')
    expect(notificationsStore.notifications[0].message).toContain('Kitchen Sensor')

    wsStore.handleIncomingEvent('device.offline', { device_id: 'dev1', name: 'Kitchen Sensor' })
    expect(notificationsStore.notifications.length).toBe(2)
    expect(notificationsStore.notifications[1].title).toBe('Device Offline')
  })
})
