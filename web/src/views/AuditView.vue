<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuditStore } from '../stores/audit'
import BaseButton from '../components/ui/BaseButton.vue'
import { ShieldCheck, RefreshCw, ChevronDown, ChevronUp, User, Terminal, Cpu, Cog } from '@lucide/vue'

const { t } = useI18n()
const auditStore = useAuditStore()

const limit = ref<number>(50)
const expandedPayloads = ref<Record<number, boolean>>({})

function togglePayload(id: number) {
  expandedPayloads.value[id] = !expandedPayloads.value[id]
}

function formatDate(iso: string): string {
  try {
    const d = new Date(iso)
    return d.toLocaleString()
  } catch {
    return iso
  }
}

function getActorIcon(type: string) {
  switch (type) {
    case 'local_user':
      return User
    case 'api_token':
      return Terminal
    case 'automation':
      return Cpu
    default:
      return Cog
  }
}

function getActorLabel(type: string): string {
  const key = `audit.actorTypes.${type}`
  const translated = t(key)
  return translated !== key ? translated : type
}

async function refresh() {
  await auditStore.fetchAuditLogs(limit.value)
}

onMounted(() => {
  refresh()
})
</script>

<template>
  <div class="audit-view">
    <div class="header-row">
      <div>
        <div class="title-with-icon">
          <div class="icon-badge">
            <ShieldCheck :size="24" class="text-cyan" />
          </div>
          <div>
            <h2 class="section-title">{{ t('audit.title') }}</h2>
            <p class="section-subtitle">{{ t('audit.subtitle') }}</p>
          </div>
        </div>
      </div>

      <div class="header-actions">
        <div class="limit-selector">
          <label class="limit-label">{{ t('audit.limitLabel') }}:</label>
          <select v-model.number="limit" class="limit-select" @change="refresh">
            <option :value="25">25</option>
            <option :value="50">50</option>
            <option :value="100">100</option>
            <option :value="200">200</option>
          </select>
        </div>

        <BaseButton variant="secondary" :loading="auditStore.loading" @click="refresh">
          <RefreshCw :size="16" />
          <span>{{ t('common.refresh') }}</span>
        </BaseButton>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="auditStore.loading && auditStore.entries.length === 0" class="loading-state glass-card">
      <div class="spinner" />
      <p>{{ t('common.loading') }}</p>
    </div>

    <!-- Empty State -->
    <div v-else-if="auditStore.entries.length === 0" class="empty-state glass-card">
      <ShieldCheck :size="48" class="empty-icon" />
      <p class="empty-text">{{ t('audit.noEntries') }}</p>
    </div>

    <!-- Audit Log Entries Table / List -->
    <div v-else class="audit-table-card glass-card">
      <table class="audit-table">
        <thead>
          <tr>
            <th>{{ t('audit.timestamp') }}</th>
            <th>{{ t('audit.actor') }}</th>
            <th>{{ t('audit.action') }}</th>
            <th>{{ t('audit.target') }}</th>
            <th>{{ t('audit.payload') }}</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="entry in auditStore.entries" :key="entry.id">
            <tr class="audit-row">
              <td class="font-mono text-dim time-cell">
                {{ formatDate(entry.created_at) }}
              </td>
              <td>
                <div class="actor-cell">
                  <component :is="getActorIcon(entry.actor_type)" :size="14" class="actor-icon" />
                  <span class="actor-badge">{{ getActorLabel(entry.actor_type) }}</span>
                  <span v-if="entry.actor_id" class="actor-id font-mono text-dim">({{ entry.actor_id }})</span>
                </div>
              </td>
              <td>
                <span class="action-tag font-mono">{{ entry.action }}</span>
              </td>
              <td>
                <span v-if="entry.target_type" class="target-badge">
                  {{ entry.target_type }}<span v-if="entry.target_id">: {{ entry.target_id }}</span>
                </span>
                <span v-else class="text-dim">-</span>
              </td>
              <td>
                <button
                  v-if="entry.payload && Object.keys(entry.payload).length > 0"
                  class="btn-toggle-payload"
                  @click="togglePayload(entry.id)"
                >
                  <span>{{ expandedPayloads[entry.id] ? t('audit.hidePayload') : t('audit.showPayload') }}</span>
                  <component :is="expandedPayloads[entry.id] ? ChevronUp : ChevronDown" :size="14" />
                </button>
                <span v-else class="text-dim">-</span>
              </td>
            </tr>
            <tr v-if="expandedPayloads[entry.id]" class="payload-row">
              <td colspan="5">
                <pre class="payload-viewer font-mono">{{ JSON.stringify(entry.payload, null, 2) }}</pre>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.audit-view {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}

.title-with-icon {
  display: flex;
  align-items: center;
  gap: 16px;
}

.icon-badge {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: rgba(14, 165, 233, 0.12);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(14, 165, 233, 0.25);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 16px;
}

.limit-selector {
  display: flex;
  align-items: center;
  gap: 8px;
}

.limit-label {
  font-size: 13px;
  color: var(--text-dim);
}

.limit-select {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  color: var(--text-main);
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  cursor: pointer;
}

.loading-state,
.empty-state {
  padding: 60px 20px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.empty-icon {
  color: var(--text-dim);
}

.empty-text {
  color: var(--text-dim);
  font-size: 14px;
}

.audit-table-card {
  padding: 0;
  overflow-x: auto;
}

.audit-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
  font-size: 13px;
}

.audit-table th {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-dim);
  font-weight: 500;
  background: rgba(255, 255, 255, 0.02);
}

.audit-row td {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-subtle);
  vertical-align: middle;
}

.time-cell {
  white-space: nowrap;
  font-size: 12px;
}

.actor-cell {
  display: flex;
  align-items: center;
  gap: 6px;
}

.actor-badge {
  font-weight: 500;
  color: var(--text-main);
}

.actor-icon {
  color: var(--primary);
}

.action-tag {
  background: rgba(14, 165, 233, 0.1);
  color: var(--primary);
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  border: 1px solid rgba(14, 165, 233, 0.2);
}

.target-badge {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-muted);
}

.btn-toggle-payload {
  background: none;
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  border-radius: var(--radius-sm);
  padding: 3px 8px;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-toggle-payload:hover {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-main);
}

.payload-row td {
  padding: 0 16px 16px;
  background: rgba(0, 0, 0, 0.25);
  border-bottom: 1px solid var(--border-subtle);
}

.payload-viewer {
  background: rgba(0, 0, 0, 0.4);
  padding: 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  color: var(--text-dim);
  border: 1px solid var(--border-subtle);
  overflow-x: auto;
  margin: 8px 0 0;
}
</style>
