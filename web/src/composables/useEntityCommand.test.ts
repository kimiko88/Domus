import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useEntityCommand } from './useEntityCommand'
import { useEntitiesStore } from '../stores/entities'
import { useNotificationsStore } from '../stores/notifications'
import type { Entity } from '../api/types'

describe('useEntityCommand', () => {
  let entitiesStore: ReturnType<typeof useEntitiesStore>
  let notificationsStore: ReturnType<typeof useNotificationsStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    entitiesStore = useEntitiesStore()
    notificationsStore = useNotificationsStore()
  })

  it('toggles light entity state from off to on', async () => {
    const mockEntity: Entity = {
      id: 'light.living_room',
      domain: 'light',
      name: 'Living Room Light',
      state: 'off',
      attributes: {},
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }
    entitiesStore.entities[mockEntity.id] = mockEntity

    const callServiceSpy = vi.spyOn(entitiesStore, 'callService').mockResolvedValue(undefined)

    const { toggle, isExecuting } = useEntityCommand()
    expect(isExecuting.value).toBe(false)

    await toggle(mockEntity)

    expect(callServiceSpy).toHaveBeenCalledWith('light', 'turn_on', 'light.living_room', undefined)
  })

  it('toggles light entity state from on to off', async () => {
    const mockEntity: Entity = {
      id: 'light.kitchen',
      domain: 'light',
      name: 'Kitchen Light',
      state: 'on',
      attributes: {},
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }
    entitiesStore.entities[mockEntity.id] = mockEntity

    const callServiceSpy = vi.spyOn(entitiesStore, 'callService').mockResolvedValue(undefined)

    const { toggle } = useEntityCommand()
    await toggle(mockEntity)

    expect(callServiceSpy).toHaveBeenCalledWith('light', 'turn_off', 'light.kitchen', undefined)
  })

  it('sets brightness for light entity', async () => {
    const mockEntity: Entity = {
      id: 'light.bedroom',
      domain: 'light',
      name: 'Bedroom Light',
      state: 'on',
      attributes: { brightness: 100 },
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }
    entitiesStore.entities[mockEntity.id] = mockEntity

    const callServiceSpy = vi.spyOn(entitiesStore, 'callService').mockResolvedValue(undefined)

    const { setBrightness } = useEntityCommand()
    await setBrightness(mockEntity, 200)

    expect(callServiceSpy).toHaveBeenCalledWith('light', 'turn_on', 'light.bedroom', { brightness: 200 })
  })

  it('notifies on command failure and restores executing state', async () => {
    const mockEntity: Entity = {
      id: 'switch.fan',
      domain: 'switch',
      name: 'Ceiling Fan',
      state: 'off',
      attributes: {},
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }
    entitiesStore.entities[mockEntity.id] = mockEntity

    vi.spyOn(entitiesStore, 'callService').mockRejectedValue(new Error('Network offline'))

    const { toggle, isExecuting } = useEntityCommand()

    await expect(toggle(mockEntity)).rejects.toThrow('Network offline')
    expect(isExecuting.value).toBe(false)
    expect(notificationsStore.notifications.length).toBe(1)
    expect(notificationsStore.notifications[0].type).toBe('error')
    expect(notificationsStore.notifications[0].title).toBe('Command Failed')
  })
})
