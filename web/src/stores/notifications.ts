import { defineStore } from 'pinia'
import { ref } from 'vue'

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
    addNotification,
    removeNotification,
    success,
    error,
    warning,
    info,
    clearAll
  }
})
