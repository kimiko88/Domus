import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useEntitiesStore } from './entities'
import { useNotificationsStore } from './notifications'
import type { EntityStateEventPayload } from '../api/types'

export const useWsStore = defineStore('ws', () => {
  const isConnected = ref(false)
  const isConnecting = ref(false)
  const lastError = ref<string | null>(null)

  const entitiesStore = useEntitiesStore()
  const notificationsStore = useNotificationsStore()

  function setConnected(connected: boolean) {
    isConnected.value = connected
  }

  function setConnecting(connecting: boolean) {
    isConnecting.value = connecting
  }

  function setError(error: string | null) {
    lastError.value = error
  }

  function handleIncomingEvent(eventType: string, payload: any) {
    switch (eventType) {
      case 'entity.state_changed':
        entitiesStore.handleStateChange(payload as EntityStateEventPayload)
        break

      case 'device.online':
        notificationsStore.info('Device Online', `${payload.name || payload.device_id} is now online.`)
        break

      case 'device.offline':
        notificationsStore.warning('Device Offline', `${payload.name || payload.device_id} is offline.`)
        break

      case 'device.discovered':
        notificationsStore.info('Device Discovered', `New device discovered: ${payload.name || payload.device_id}`)
        break

      case 'scene.activated':
        notificationsStore.success('Scene Activated', `${payload.name || payload.scene_id} was activated.`)
        break

      case 'automation.run_failed':
        notificationsStore.error('Automation Failed', `Automation "${payload.name || payload.automation_id}" failed: ${payload.error}`)
        break

      case 'notification.created':
        notificationsStore.addNotification({
          title: payload.title || 'System Notification',
          message: payload.message,
          type: payload.severity === 'critical' || payload.severity === 'error' ? 'error' : (payload.severity || 'info')
        })
        break

      default:
        // Ignore unhandled event types
        break
    }
  }

  return {
    isConnected,
    isConnecting,
    lastError,
    setConnected,
    setConnecting,
    setError,
    handleIncomingEvent
  }
})
