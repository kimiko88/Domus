import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useEntitiesStore } from './entities'
import { useNotificationsStore } from './notifications'
import { useAlarmStore } from './alarm'
import type { EntityStateEventPayload } from '../api/types'
import i18n from '../locales'

const t = (...args: Parameters<typeof i18n.global.t>) => i18n.global.t(...args)

const DISCOVERY_NOTIFICATION_COOLDOWN_MS = 10 * 60 * 1000

export const useWsStore = defineStore('ws', () => {
  const isConnected = ref(false)
  const isConnecting = ref(false)
  const lastError = ref<string | null>(null)

  const entitiesStore = useEntitiesStore()
  const notificationsStore = useNotificationsStore()
  const alarmStore = useAlarmStore()
  const discoveryNotificationLastShownAt = new Map<string, number>()

  function setConnected(connected: boolean) {
    isConnected.value = connected
  }

  function setConnecting(connecting: boolean) {
    isConnecting.value = connecting
  }

  function setError(error: string | null) {
    lastError.value = error
  }

  function notifyDiscoveredDevice(payload: any) {
    const deviceKey = String(payload.device_id || payload.id || payload.name || '').trim()
    if (!deviceKey) return

    const now = Date.now()
    const lastShownAt = discoveryNotificationLastShownAt.get(deviceKey)
    if (lastShownAt !== undefined && now - lastShownAt < DISCOVERY_NOTIFICATION_COOLDOWN_MS) {
      return
    }

    // Keep the map bounded when many different devices are discovered over time.
    for (const [key, timestamp] of discoveryNotificationLastShownAt) {
      if (now - timestamp >= DISCOVERY_NOTIFICATION_COOLDOWN_MS) {
        discoveryNotificationLastShownAt.delete(key)
      }
    }

    discoveryNotificationLastShownAt.set(deviceKey, now)
    notificationsStore.info(
      'Device Discovered',
      `New device discovered: ${payload.name || payload.device_id || deviceKey}`
    )
  }

  function handleIncomingEvent(eventType: string, payload: any) {
    switch (eventType) {
      case 'entity.state_changed':
        entitiesStore.handleStateChange(payload as EntityStateEventPayload)
        break

      case 'alarm.state_changed':
      case 'alarm.pending_entry':
        alarmStore.handleAlarmEvent(eventType, payload)
        notificationsStore.info(t('common.wsNotifications.alarmTitle'), t('common.wsNotifications.alarmState', { state: payload.state || t('common.wsNotifications.updatedState') }))
        break

      case 'alarm.triggered':
        alarmStore.handleAlarmEvent(eventType, payload)
        notificationsStore.error(t('common.wsNotifications.alarmTriggeredTitle'), t('common.wsNotifications.alarmTriggeredMessage', { reason: payload.reason || t('common.wsNotifications.defaultAlarmReason') }))
        break

      case 'alarm.disarmed':
        alarmStore.handleAlarmEvent(eventType, payload)
        notificationsStore.success(t('common.wsNotifications.alarmDisarmedTitle'), t('common.wsNotifications.alarmDisarmedMessage'))
        break

      case 'tag.scanned':
        notificationsStore.info(t('common.wsNotifications.tagScannedTitle'), t('common.wsNotifications.tagScannedMessage', { tag: payload.tag_name || payload.tag_id }))
        break

      case 'device.online':
        notificationsStore.info(t('common.wsNotifications.deviceOnlineTitle'), t('common.wsNotifications.deviceOnlineMessage', { name: payload.name || payload.device_id }))
        break

      case 'device.offline':
        notificationsStore.warning(t('common.wsNotifications.deviceOfflineTitle'), t('common.wsNotifications.deviceOfflineMessage', { name: payload.name || payload.device_id }))
        break

      case 'device.discovered':
        notifyDiscoveredDevice(payload)
        break

      case 'scene.activated':
        notificationsStore.success(t('common.wsNotifications.sceneActivatedTitle'), t('common.wsNotifications.sceneActivatedMessage', { name: payload.name || payload.scene_id }))
        break

      case 'automation.run_failed':
        notificationsStore.error(t('common.wsNotifications.automationFailedTitle'), t('common.wsNotifications.automationFailedMessage', { name: payload.name || payload.automation_id, error: payload.error }))
        break

      case 'notification.created':
        notificationsStore.addNotification({
          title: payload.title || t('common.wsNotifications.systemNotification'),
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
