<script setup lang="ts">
import { ref } from 'vue'
import type { Automation } from '../../api/types'
import { useAutomationsStore } from '../../stores/automations'
import { useNotificationsStore } from '../../stores/notifications'
import {
  Zap,
  Play,
  History,
  Pencil,
  Trash2,
  Clock,
  Layers,
  CheckCircle,
  AlertCircle,
  Activity
} from '@lucide/vue'

interface Props {
  automation: Automation
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'view-runs', automation: Automation): void
  (e: 'edit', automation: Automation): void
  (e: 'delete', automation: Automation): void
}>()

const automationsStore = useAutomationsStore()
const notificationsStore = useNotificationsStore()

const isTriggering = ref(false)

async function handleToggle() {
  try {
    const newState = await automationsStore.toggleAutomation(props.automation.id)
    notificationsStore.info(
      'Automazione',
      `"${props.automation.name}" ${newState ? 'abilitata' : 'disabilitata'}`
    )
  } catch (err: any) {
    notificationsStore.error('Errore', err?.message || 'Impossibile aggiornare l\'automazione')
  }
}

async function handleTrigger() {
  isTriggering.value = true
  try {
    await automationsStore.triggerAutomation(props.automation.id)
    notificationsStore.success('Automazione Avviata', `Trigger manuale inviato per "${props.automation.name}"`)
  } catch (err: any) {
    notificationsStore.error('Errore Trigger', err?.message || 'Impossibile avviare l\'automazione')
  } finally {
    isTriggering.value = false
  }
}
</script>

<template>
  <div :class="['automation-card glass-card', { 'is-disabled': !automation.enabled }]">
    <div class="card-top">
      <div class="header-left">
        <div class="icon-box" :class="{ active: automation.enabled }">
          <Zap :size="20" />
        </div>
        <div class="title-group">
          <h3 class="automation-name" :title="automation.name">{{ automation.name }}</h3>
          <p v-if="automation.description" class="automation-desc">{{ automation.description }}</p>
        </div>
      </div>

      <!-- Toggle Button -->
      <button
        class="toggle-btn"
        :class="{ active: automation.enabled }"
        aria-label="Toggle automation enabled"
        @click="handleToggle"
      >
        <span class="toggle-indicator" />
      </button>
    </div>

    <!-- Metadata Badges -->
    <div class="metadata-row">
      <span class="badge badge-mode">
        <Layers :size="12" />
        {{ automation.execution_mode }}
      </span>

      <span v-if="automation.cooldown_seconds > 0" class="badge badge-cooldown">
        <Clock :size="12" />
        {{ automation.cooldown_seconds }}s cooldown
      </span>

      <span
        v-if="automation.last_status"
        :class="['badge badge-status', `status-${automation.last_status}`]"
      >
        <CheckCircle v-if="automation.last_status === 'completed'" :size="12" />
        <AlertCircle v-else-if="automation.last_status === 'failed'" :size="12" />
        <Activity v-else :size="12" />
        {{ automation.last_status }}
      </span>
    </div>

    <!-- Card Footer Actions -->
    <div class="card-footer">
      <div class="primary-buttons">
        <button
          class="btn-action btn-trigger"
          :disabled="isTriggering"
          title="Esegui manualmente"
          @click="handleTrigger"
        >
          <Play :size="13" :class="{ 'animate-spin': isTriggering }" />
          <span>Esegui</span>
        </button>

        <button
          class="btn-action btn-runs"
          title="Visualizza cronologia esecuzioni"
          @click="emit('view-runs', automation)"
        >
          <History :size="13" />
          <span>Cronologia</span>
        </button>
      </div>

      <div class="secondary-buttons">
        <button
          class="btn-icon btn-edit"
          title="Modifica regola"
          aria-label="Edit automation"
          @click="emit('edit', automation)"
        >
          <Pencil :size="14" />
        </button>
        <button
          class="btn-icon btn-delete"
          title="Elimina automazione"
          aria-label="Delete automation"
          @click="emit('delete', automation)"
        >
          <Trash2 :size="14" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.automation-card {
  padding: 18px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 16px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
  transition: all var(--transition-normal);
}

.automation-card.is-disabled {
  opacity: 0.75;
}

.automation-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
}

.card-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.header-left {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
  flex: 1;
}

.icon-box {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-dim);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all var(--transition-normal);
}

.icon-box.active {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  box-shadow: 0 0 16px rgba(245, 158, 11, 0.3);
}

.title-group {
  flex: 1;
  min-width: 0;
}

.automation-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.automation-desc {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 3px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.4;
}

/* Toggle Switch Button */
.toggle-btn {
  width: 44px;
  height: 24px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.1);
  position: relative;
  cursor: pointer;
  border: 1px solid var(--border-card);
  transition: all var(--transition-fast);
  flex-shrink: 0;
  padding: 2px;
}

.toggle-indicator {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: white;
  transition: all var(--transition-fast);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.4);
}

.toggle-btn.active {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  border-color: #d97706;
}

.toggle-btn.active .toggle-indicator {
  transform: translateX(20px);
}

/* Metadata Badges */
.metadata-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 500;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-muted);
}

.badge-status {
  text-transform: capitalize;
}

.status-completed {
  background: var(--color-success-bg);
  color: var(--color-success);
  border-color: rgba(16, 185, 129, 0.3);
}

.status-failed {
  background: var(--color-danger-bg);
  color: var(--color-danger);
  border-color: rgba(244, 63, 94, 0.3);
}

.status-running {
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
  border-color: rgba(14, 165, 233, 0.3);
}

/* Footer Actions */
.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}

.primary-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-action {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-action:hover:not(:disabled) {
  background: var(--bg-surface-hover);
  border-color: rgba(255, 255, 255, 0.2);
}

.secondary-buttons {
  display: flex;
  align-items: center;
  gap: 4px;
}

.btn-icon {
  width: 30px;
  height: 30px;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-dim);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-icon:hover {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-main);
}

.btn-delete:hover {
  background: rgba(244, 63, 94, 0.1);
  color: var(--color-danger);
}
</style>
