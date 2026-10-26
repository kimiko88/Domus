<script setup lang="ts">
import { computed } from 'vue'
import { Bell, CheckCircle, AlertTriangle, X, ShieldAlert, Sparkles, Clock } from '@lucide/vue'
import { useNotificationsStore } from '../../stores/notifications'
import { useAuthStore } from '../../stores/auth'

const props = defineProps<{
  isOpen: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const notifStore = useNotificationsStore()
const authStore = useAuthStore()

const activeItems = computed(() => notifStore.activeActionable)

async function handleActionClick(notifId: string, actionId: string) {
  const username = authStore.user?.username || 'user'
  await notifStore.executeAction(notifId, actionId, username)
}

function handleDismiss(notifId: string) {
  notifStore.dismissActionable(notifId)
}

function formatTime(isoStr: string) {
  try {
    const d = new Date(isoStr)
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  } catch {
    return ''
  }
}
</script>

<template>
  <Transition name="fade">
    <div v-if="isOpen" class="modal-backdrop" @click="emit('close')">
      <div class="notif-drawer glass-panel" role="dialog" aria-modal="true" @click.stop>
        <!-- Drawer Header -->
        <div class="drawer-header">
          <div class="header-title-box">
            <div class="bell-icon-badge">
              <Bell :size="18" />
              <span v-if="notifStore.unreadActionableCount > 0" class="badge-count">
                {{ notifStore.unreadActionableCount }}
              </span>
            </div>
            <div>
              <h2 class="drawer-title">Notifiche Azionabili</h2>
              <p class="drawer-sub">Comandi rapidi e avvisi interattivi della casa</p>
            </div>
          </div>
          <button class="btn-close" aria-label="Chiudi notifiche" @click="emit('close')">
            <X :size="18" />
          </button>
        </div>

        <!-- Drawer Content -->
        <div class="drawer-body">
          <div v-if="activeItems.length === 0" class="empty-state">
            <div class="empty-icon-circle">
              <CheckCircle :size="32" class="text-success" />
            </div>
            <h3>Nessuna notifica in sospeso</h3>
            <p>Tutti gli eventi e gli allarmi della casa sono risolti o archiviati.</p>
          </div>

          <div v-else class="notif-list">
            <div
              v-for="item in activeItems"
              :key="item.id"
              class="notif-card"
              :class="{ 'is-critical': item.critical }"
            >
              <div class="notif-card-header">
                <div class="notif-type-tag">
                  <ShieldAlert v-if="item.critical" :size="15" class="text-danger" />
                  <Sparkles v-else :size="15" class="text-primary" />
                  <span class="type-text">{{ item.critical ? 'Critico / Urgente' : 'Notifica Intelligente' }}</span>
                </div>
                <div class="time-meta">
                  <Clock :size="12" />
                  <span>{{ formatTime(item.created_at) }}</span>
                  <button class="btn-dismiss" title="Archivia" @click="handleDismiss(item.id)">
                    <X :size="14" />
                  </button>
                </div>
              </div>

              <div class="notif-card-body">
                <h4 class="notif-title">{{ item.title }}</h4>
                <p class="notif-msg">{{ item.message }}</p>
                <img v-if="item.image_url" :src="item.image_url" alt="Anteprima evento" class="notif-preview-img" />
              </div>

              <!-- Actionable Buttons -->
              <div v-if="item.actions && item.actions.length > 0" class="actions-row">
                <button
                  v-for="act in item.actions"
                  :key="act.id"
                  class="btn-action"
                  :class="{ 'is-destructive': act.destructive }"
                  @click="handleActionClick(item.id, act.id)"
                >
                  {{ act.title }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: var(--bg-overlay, rgba(0, 0, 0, 0.65));
  backdrop-filter: blur(8px);
  z-index: 9999;
  display: flex;
  justify-content: flex-end;
}

.notif-drawer {
  width: 100%;
  max-width: 440px;
  height: 100%;
  background: var(--bg-surface);
  border-left: 1px solid var(--border-color);
  box-shadow: -10px 0 35px rgba(0, 0, 0, 0.4);
  display: flex;
  flex-direction: column;
}

.drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border-color);
}

.header-title-box {
  display: flex;
  align-items: center;
  gap: 0.85rem;
}

.bell-icon-badge {
  position: relative;
  width: 38px;
  height: 38px;
  border-radius: 10px;
  background: var(--primary-light, rgba(14, 165, 233, 0.15));
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.badge-count {
  position: absolute;
  top: -4px;
  right: -4px;
  background: var(--color-danger, #f43f5e);
  color: #ffffff;
  font-size: 0.65rem;
  font-weight: 800;
  padding: 2px 6px;
  border-radius: 10px;
  box-shadow: 0 0 6px rgba(244, 63, 94, 0.6);
}

.drawer-title {
  font-size: 1.15rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-main);
}

.drawer-sub {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin: 0.15rem 0 0;
}

.btn-close {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 0.4rem;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.btn-close:hover {
  background: var(--bg-surface-hover);
  color: var(--text-main);
}

.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 1.25rem;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 4rem 1.5rem;
  color: var(--text-muted);
}

.empty-icon-circle {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: rgba(16, 185, 129, 0.12);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 1rem;
}

.empty-state h3 {
  font-size: 1.05rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0 0 0.4rem;
}

.empty-state p {
  font-size: 0.85rem;
  margin: 0;
  max-width: 280px;
}

.notif-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.notif-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 0.85rem;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  transition: all 0.2s ease;
}

.notif-card.is-critical {
  border-color: rgba(244, 63, 94, 0.4);
  box-shadow: 0 0 16px rgba(244, 63, 94, 0.12);
}

.notif-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.notif-type-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--text-muted);
}

.time-meta {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.btn-dismiss {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 2px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-dismiss:hover {
  color: var(--text-main);
}

.notif-title {
  font-size: 0.95rem;
  font-weight: 700;
  margin: 0 0 0.25rem;
  color: var(--text-main);
}

.notif-msg {
  font-size: 0.825rem;
  color: var(--text-muted);
  line-height: 1.4;
  margin: 0;
}

.notif-preview-img {
  width: 100%;
  max-height: 180px;
  object-fit: cover;
  border-radius: 8px;
  margin-top: 0.5rem;
  border: 1px solid var(--border-color);
}

.actions-row {
  display: flex;
  gap: 0.6rem;
  flex-wrap: wrap;
  padding-top: 0.5rem;
  border-top: 1px solid var(--border-color);
}

.btn-action {
  flex: 1;
  min-width: 120px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.5rem 0.85rem;
  border-radius: 0.6rem;
  font-size: 0.825rem;
  font-weight: 700;
  cursor: pointer;
  background: var(--primary);
  color: #ffffff;
  border: none;
  transition: all 0.2s ease;
}

.btn-action:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 10px rgba(14, 165, 233, 0.3);
}

.btn-action.is-destructive {
  background: rgba(244, 63, 94, 0.15);
  color: var(--color-danger);
  border: 1px solid rgba(244, 63, 94, 0.3);
}

.btn-action.is-destructive:hover {
  background: rgba(244, 63, 94, 0.25);
  box-shadow: 0 4px 10px rgba(244, 63, 94, 0.2);
}

/* Transitions */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.25s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Light Theme Contrast Overrides */
:root[data-theme="light"] .notif-drawer {
  background: #ffffff;
  border-left: 1px solid #e2e8f0;
  box-shadow: -10px 0 35px rgba(15, 23, 42, 0.1);
}

:root[data-theme="light"] .drawer-title {
  color: #0f172a;
}

:root[data-theme="light"] .drawer-sub,
:root[data-theme="light"] .notif-msg,
:root[data-theme="light"] .time-meta,
:root[data-theme="light"] .notif-type-tag {
  color: #475569;
}

:root[data-theme="light"] .notif-card {
  background: #f8fafc;
  border: 1.5px solid #e2e8f0;
}

:root[data-theme="light"] .notif-title {
  color: #0f172a;
}

:root[data-theme="light"] .btn-action {
  background: #0284c7;
  color: #ffffff;
}

:root[data-theme="light"] .btn-action.is-destructive {
  background: #fff1f2;
  color: #be123c;
  border: 1px solid #fecdd3;
}
</style>
