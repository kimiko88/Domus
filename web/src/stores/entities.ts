import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '../api/client'
import type { Entity, EntityDomain, EntityStateEventPayload } from '../api/types'

export const useEntitiesStore = defineStore('entities', () => {
  const entities = ref<Record<string, Entity>>({})
  const loading = ref(false)
  const error = ref<string | null>(null)

  const allEntities = computed(() => Object.values(entities.value))
  const lights = computed(() => allEntities.value.filter((e) => e.domain === 'light'))
  const switches = computed(() => allEntities.value.filter((e) => e.domain === 'switch'))
  const sensors = computed(() => allEntities.value.filter((e) => e.domain === 'sensor'))
  const binarySensors = computed(() => allEntities.value.filter((e) => e.domain === 'binary_sensor'))

  const byId = (id: string): Entity | undefined => entities.value[id]

  const byDomain = (domain: EntityDomain | string): Entity[] =>
    allEntities.value.filter((e) => e.domain === domain)

  async function fetchEntities(): Promise<Entity[]> {
    try {
      loading.value = true
      error.value = null
      const raw = await apiClient.get<Entity[] | null>('/api/v1/entities')
      const data = raw ?? []
      const dict: Record<string, Entity> = {}
      data.forEach((e) => {
        dict[e.id] = e
      })
      entities.value = dict
      return data
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch entities'
      throw err
    } finally {
      loading.value = false
    }
  }

  function handleStateChange(payload: EntityStateEventPayload) {
    const existing = entities.value[payload.entity_id]
    if (existing) {
      existing.state = payload.new_state
      existing.attributes = { ...existing.attributes, ...payload.new_attributes }
      existing.last_updated = payload.timestamp
    } else {
      // Parse domain from entity_id if not previously loaded
      const parts = payload.entity_id.split('.')
      const domain = (parts[0] || 'sensor') as EntityDomain
      entities.value[payload.entity_id] = {
        id: payload.entity_id,
        domain,
        name: payload.new_attributes?.friendly_name || payload.entity_id,
        state: payload.new_state,
        attributes: payload.new_attributes || {},
        last_changed: payload.timestamp,
        last_updated: payload.timestamp
      }
    }
  }

  async function callService(
    domain: string,
    service: string,
    entityId: string,
    data?: Record<string, any>
  ): Promise<void> {
    const existing = entities.value[entityId]
    const backupState = existing ? { ...existing, attributes: { ...existing.attributes } } : null

    // Optimistic Update
    if (existing) {
      if (service === 'turn_on') {
        existing.state = 'on'
        if (data?.brightness !== undefined) {
          existing.attributes.brightness = data.brightness
        }
      } else if (service === 'turn_off') {
        existing.state = 'off'
      } else if (service === 'toggle') {
        existing.state = existing.state === 'on' ? 'off' : 'on'
      }
      existing.last_updated = new Date().toISOString()
    }

    try {
      await apiClient.post(`/api/v1/entities/${entityId}/command`, {
        service: `${domain}.${service}`,
        params: data
      })
    } catch (err) {
      // Revert optimistic update on failure
      if (existing && backupState) {
        existing.state = backupState.state
        existing.attributes = backupState.attributes
        existing.last_updated = backupState.last_updated
      }
      throw err
    }
  }

  return {
    entities,
    loading,
    error,
    allEntities,
    lights,
    switches,
    sensors,
    binarySensors,
    byId,
    byDomain,
    fetchEntities,
    handleStateChange,
    callService
  }
})
