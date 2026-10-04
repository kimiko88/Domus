import { ref } from 'vue'
import { useEntitiesStore } from '../stores/entities'
import { useNotificationsStore } from '../stores/notifications'
import type { Entity } from '../api/types'

export function useEntityCommand() {
  const isExecuting = ref(false)
  const entitiesStore = useEntitiesStore()
  const notificationsStore = useNotificationsStore()

  async function execute(
    domain: string,
    service: string,
    entityId: string,
    data?: Record<string, any>
  ): Promise<void> {
    isExecuting.value = true
    try {
      await entitiesStore.callService(domain, service, entityId, data)
    } catch (err: any) {
      notificationsStore.error('Command Failed', err?.message || 'Action could not be completed')
      throw err
    } finally {
      isExecuting.value = false
    }
  }

  async function toggle(entity: Entity): Promise<void> {
    const isCurrentlyOn = entity.state === 'on'
    const service = isCurrentlyOn ? 'turn_off' : 'turn_on'
    return execute(entity.domain, service, entity.id)
  }

  async function turnOn(entity: Entity, data?: Record<string, any>): Promise<void> {
    return execute(entity.domain, 'turn_on', entity.id, data)
  }

  async function turnOff(entity: Entity): Promise<void> {
    return execute(entity.domain, 'turn_off', entity.id)
  }

  async function setBrightness(entity: Entity, brightness: number): Promise<void> {
    return execute(entity.domain, 'turn_on', entity.id, { brightness })
  }

  async function setTemperature(entity: Entity, temperature: number): Promise<void> {
    return execute(entity.domain, 'set_temperature', entity.id, { temperature })
  }

  async function setCoverPosition(entity: Entity, position: number): Promise<void> {
    return execute(entity.domain, 'set_cover_position', entity.id, { position })
  }

  return {
    isExecuting,
    execute,
    toggle,
    turnOn,
    turnOff,
    setBrightness,
    setTemperature,
    setCoverPosition
  }
}
