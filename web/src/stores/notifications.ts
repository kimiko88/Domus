import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '../api/client'
import type { ActionableNotification } from '../api/types'

export type NotificationType = 'success' | 'error' | 'warning' | 'info'

export interface NotificationItem {
  id: string
  title: string
  message?: string
  type: NotificationType
  timeout?: number
  timestamp: number
}

export interface AddNotificationOptions {
  title: string
  message?: string
  type?: NotificationType
  timeout?: number
}

export const useNotificationsStore = defineStore('notifications', () => {
  const notifications = ref<NotificationItem[]>([])
  const actionableNotifications = ref<ActionableNotification[]>([])
  const loadingActionable = ref(false)

  const activeActionable = computed(() => {
    return actionableNotifications.value.filter((n) => !n.resolved_at)
  })

  const unreadActionableCount = computed(() => activeActionable.value.length)

  async function fetchActionable() {
    try {
      loadingActionable.value = true
      const res = await apiClient.get<ActionableNotification[]>('/api/v1/notifications/actionable')
      if (Array.isArray(res)) {
        actionableNotifications.value = res
      }
    } catch {
      // Graceful fallback if offline
    } finally {
      loadingActionable.value = false
    }
  }

  async function executeAction(notificationId: string, actionId: string, username: string = 'current_user'): Promise<boolean> {
    try {
      await apiClient.post(`/api/v1/notifications/actionable/actions/${encodeURIComponent(actionId)}`, {
        notification_id: notificationId,
        user: username
      })

      // Update local state to mark resolved
      const item = actionableNotifications.value.find((n) => n.id === notificationId)
      if (item) {
        item.resolved_at = new Date().toISOString()
      }
      success('Azione eseguita', 'Comando inviato con successo')
      return true
    } catch (err: any) {
      error('Errore azione', err?.message || 'Impossibile eseguire l\'azione')
      return false
    }
  }

  function dismissActionable(notificationId: string) {
    const item = actionableNotifications.value.find((n) => n.id === notificationId)
    if (item) {
      item.resolved_at = new Date().toISOString()
    }
  }

  function addActionable(notif: ActionableNotification) {
    if (!actionableNotifications.value.some((n) => n.id === notif.id)) {
      actionableNotifications.value.unshift(notif)
      // Also show a high priority toast
      addNotification({
        title: notif.title,
        message: notif.message,
        type: notif.critical ? 'error' : 'info',
        timeout: notif.critical ? 12000 : 7000
      })
    }
  }

  function addNotification(options: AddNotificationOptions): string {
    const id = `notif_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`
    const timeout = options.timeout ?? 5000

    const item: NotificationItem = {
      id,
      title: options.title,
      message: options.message,
      type: options.type ?? 'info',
      timeout,
      timestamp: Date.now()
    }

    notifications.value.push(item)

    if (timeout > 0) {
      setTimeout(() => {
        removeNotification(id)
      }, timeout)
    }

    return id
  }

  function removeNotification(id: string) {
    const index = notifications.value.findIndex((n) => n.id === id)
    if (index !== -1) {
      notifications.value.splice(index, 1)
    }
  }

  function success(title: string, message?: string, timeout?: number) {
    return addNotification({ title, message, type: 'success', timeout })
  }

  function error(title: string, message?: string, timeout?: number) {
    return addNotification({ title, message, type: 'error', timeout })
  }

  function warning(title: string, message?: string, timeout?: number) {
    return addNotification({ title, message, type: 'warning', timeout })
  }

  function info(title: string, message?: string, timeout?: number) {
    return addNotification({ title, message, type: 'info', timeout })
  }

  function clearAll() {
    notifications.value = []
  }

  return {
    notifications,
    actionableNotifications,
    activeActionable,
    unreadActionableCount,
    loadingActionable,
    fetchActionable,
    executeAction,
    dismissActionable,
    addActionable,
    addNotification,
    removeNotification,
    success,
    error,
    warning,
    info,
    clearAll
  }
})
