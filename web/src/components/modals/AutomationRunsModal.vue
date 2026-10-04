<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { Automation } from '../../api/types'
import { useAutomationsStore } from '../../stores/automations'
import {
  History,
  X,
  RefreshCw,
  CheckCircle,
  AlertCircle,
  Clock,
  Code2
} from '@lucide/vue'

interface Props {
  isOpen: boolean
  automation?: Automation | null
}

const props = withDefaults(defineProps<Props>(), {
  isOpen: false,
  automation: null
})

const emit = defineEmits<{
  (e: 'close'): void
}>()

const automationsStore = useAutomationsStore()
const loading = ref(false)

const currentRuns = computed(() => {
  if (!props.automation) return []
  return automationsStore.runs[props.automation.id] || []
})

async function loadRuns() {
  if (!props.automation) return
  loading.value = true
  try {
    await automationsStore.fetchRuns(props.automation.id)
  } catch (err) {
    console.warn('Failed to load runs:', err)
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.isOpen, props.automation?.id],
  ([open, id]) => {
    if (open && id) {
      loadRuns()
    }
  },
  { immediate: true }
)

function formatDate(iso: string): string {
  if (!iso) return '--'
  try {
    const d = new Date(iso)
    return d.toLocaleString()
  } catch {
    return iso
  }
}
</script>

<template>
  <div v-if="isOpen" class="modal-backdrop">
    <div class="modal-card glass-panel animate-enter">
      <div class="modal-header">
        <div class="header-title">
          <History :size="20" class="title-icon" />
          <div>
            <h3>Cronologia Esecuzioni</h3>
            <span v-if="automation" class="subtitle">{{ automation.name }}</span>
          </div>
        </div>

        <div class="header-actions">
          <button
            class="btn-icon"
            title="Aggiorna cronologia"
            :disabled="loading"
            @click="loadRuns"
          >
            <RefreshCw :size="16" :class="{ 'animate-spin': loading }" />
          </button>
          <button class="btn-icon btn-close" aria-label="Close modal" @click="emit('close')">
            <X :size="18" />
          </button>
        </div>
      </div>

      <!-- Runs List -->
      <div v-if="currentRuns.length > 0" class="runs-list">
        <div
          v-for="run in currentRuns"
          :key="run.id"
          class="run-item glass-card"
        >
          <div class="run-item-top">
            <div class="run-status-badge" :class="`status-${run.status}`">
              <CheckCircle v-if="run.status === 'completed'" :size="14" />
              <AlertCircle v-else-if="run.status === 'failed'" :size="14" />
              <Clock v-else :size="14" />
              <span>{{ run.status }}</span>
            </div>

            <div class="run-timing">
              <span class="run-date">{{ formatDate(run.started_at) }}</span>
              <span class="run-duration font-mono">{{ run.duration_ms }}ms</span>
            </div>
          </div>

          <div v-if="run.trigger_context && run.trigger_context !== '{}'" class="run-context">
            <Code2 :size="13" class="context-icon" />
            <span class="font-mono text-dim">{{ run.trigger_context }}</span>
          </div>

          <div v-if="run.log" :class="['run-log font-mono', { 'is-error': run.status === 'failed' }]">
            {{ run.log }}
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else-if="!loading" class="empty-runs">
        <History :size="32" class="empty-icon" />
        <p>Nessuna esecuzione registrata finora per questa automazione.</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: var(--bg-overlay);
  backdrop-filter: blur(8px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.modal-card {
  width: 100%;
  max-width: 600px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  border-radius: var(--radius-xl);
  padding: 24px;
  border: 1px solid var(--border-card);
  box-shadow: var(--shadow-lg);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-subtle);
}

.header-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.title-icon {
  color: var(--primary);
}

h3 {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-main);
  line-height: 1.2;
}

.subtitle {
  font-size: 13px;
  color: var(--text-muted);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-icon {
  background: transparent;
  color: var(--text-dim);
  padding: 6px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
}

.btn-icon:hover {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-main);
}

.runs-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-y: auto;
  padding-right: 4px;
}

.run-item {
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-radius: var(--radius-md);
}

.run-item-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.run-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 600;
  text-transform: capitalize;
}

.status-completed {
  background: var(--color-success-bg);
  color: var(--color-success);
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.status-failed {
  background: var(--color-danger-bg);
  color: var(--color-danger);
  border: 1px solid rgba(244, 63, 94, 0.3);
}

.status-running {
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
  border: 1px solid rgba(14, 165, 233, 0.3);
}

.status-skipped {
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-dim);
}

.run-timing {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  color: var(--text-muted);
}

.run-duration {
  color: var(--primary);
}

.run-context {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  background: rgba(0, 0, 0, 0.2);
  padding: 4px 8px;
  border-radius: var(--radius-sm);
}

.context-icon {
  color: var(--text-dim);
}

.run-log {
  font-size: 12px;
  line-height: 1.4;
  color: var(--text-muted);
  background: #060913;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
  word-break: break-word;
}

.run-log.is-error {
  color: #fb7185;
  border-color: rgba(244, 63, 94, 0.3);
  background: rgba(244, 63, 94, 0.05);
}

.empty-runs {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  text-align: center;
  color: var(--text-dim);
  gap: 12px;
  font-size: 14px;
}

.empty-icon {
  opacity: 0.5;
}
</style>
