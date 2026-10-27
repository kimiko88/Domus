<script setup lang="ts">
import { useNotificationsStore } from '../../stores/notifications'
import {
  CheckCircle,
  AlertCircle,
  AlertTriangle,
  Info,
  X
} from '@lucide/vue'

const notificationsStore = useNotificationsStore()

function getIcon(type: string) {
  switch (type) {
    case 'success':
      return CheckCircle
    case 'error':
      return AlertCircle
    case 'warning':
      return AlertTriangle
    default:
      return Info
  }
}
</script>

<template>
  <div class="toast-container" role="region" aria-label="Notifications" aria-live="polite">
    <TransitionGroup name="toast">
      <div
        v-for="item in notificationsStore.notifications"
        :key="item.id"
        :class="['toast-item glass-panel', `toast-${item.type}`]"
      >
        <component :is="getIcon(item.type)" class="toast-icon" :size="20" />
        <div class="toast-body">
          <div class="toast-title">{{ item.title }}</div>
          <div v-if="item.message" class="toast-message">{{ item.message }}</div>
        </div>
        <button
          class="toast-dismiss"
          aria-label="Dismiss notification"
          @click="notificationsStore.removeNotification(item.id)"
        >
          <X :size="16" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-container {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: 10px;
  pointer-events: none;
  max-width: 380px;
  width: calc(100% - 40px);
}

.toast-item {
  pointer-events: auto;
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 16px;
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  backdrop-filter: var(--blur-card);
  -webkit-backdrop-filter: var(--blur-card);
  border-left: 4px solid var(--primary);
  animation: slide-in-up 0.25s ease-out;
}

.toast-icon {
  flex-shrink: 0;
  margin-top: 1px;
}

.toast-body {
  flex: 1;
  min-width: 0;
}

.toast-title {
  font-weight: 600;
  font-size: 14px;
  color: var(--text-main);
  line-height: 1.3;
}

.toast-message {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 2px;
  line-height: 1.4;
  word-break: break-word;
}

.toast-dismiss {
  flex-shrink: 0;
  background: transparent;
  color: var(--text-dim);
  width: 36px;
  height: 36px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color var(--transition-fast);
  position: relative;
  cursor: pointer;
}

.toast-dismiss::after {
  content: '';
  position: absolute;
  inset: -4px;
  min-width: 44px;
  min-height: 44px;
}

.toast-dismiss:hover {
  color: var(--text-main);
}

/* Notification Types */
.toast-success {
  border-left-color: var(--color-success);
}
.toast-success .toast-icon {
  color: var(--color-success);
}

.toast-error {
  border-left-color: var(--color-danger);
}
.toast-error .toast-icon {
  color: var(--color-danger);
}

.toast-warning {
  border-left-color: var(--color-warning);
}
.toast-warning .toast-icon {
  color: var(--color-warning);
}

.toast-info {
  border-left-color: var(--primary);
}
.toast-info .toast-icon {
  color: var(--primary);
}

/* Transition Animations */
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(30px);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(30px);
}
</style>
