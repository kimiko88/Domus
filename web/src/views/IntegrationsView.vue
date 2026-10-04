<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClient } from '../api/client'
import { useNotificationsStore } from '../stores/notifications'
import { Network, RefreshCw, AlertCircle, CheckCircle, WifiOff } from '@lucide/vue'

const { t } = useI18n()

interface Integration {
  id: string
  name: string
  type: string
  enabled: boolean
  status: string
  last_error?: string | null
}

const integrations = ref<Integration[]>([])
const loading = ref(false)
const reconnectingId = ref<string | null>(null)
const notificationsStore = useNotificationsStore()

async function fetchIntegrations() {
  loading.value = true
  try {
    const data = await apiClient.get<Integration[]>('/api/v1/integrations')
    integrations.value = data
  } catch (err: any) {
    console.warn('Failed to load integrations:', err)
  } finally {
    loading.value = false
  }
}

async function reconnect(integration: Integration) {
  reconnectingId.value = integration.id
  try {
    await apiClient.post(`/api/v1/integrations/${integration.id}/reconnect`)
    notificationsStore.info(
      t('integrations.reconnect'),
      t('integrations.reconnecting', { name: integration.name })
    )
    await fetchIntegrations()
  } catch (err: any) {
    notificationsStore.error(t('common.error'), err?.message || t('integrations.reconnectError'))
  } finally {
    reconnectingId.value = null
  }
}

onMounted(() => {
  fetchIntegrations()
})
</script>

<template>
  <div class="integrations-view">
    <div class="header-row">
      <div>
        <h2 class="section-title">{{ t('integrations.title') }}</h2>
        <p class="section-subtitle">{{ t('integrations.subtitle') }}</p>
      </div>
      <button class="btn-refresh glass-panel" :disabled="loading" @click="fetchIntegrations">
        <RefreshCw :size="16" :class="{ 'animate-spin': loading }" />
        <span>{{ t('integrations.refresh') }}</span>
      </button>
    </div>

    <!-- Integrations Grid -->
    <div v-if="integrations.length > 0" class="integrations-grid">
      <div
        v-for="item in integrations"
        :key="item.id"
        :class="['integration-card glass-card', `status-${item.status}`]"
      >
        <div class="card-top">
          <div class="icon-box">
            <Network :size="24" />
          </div>
          <div class="info">
            <h3 class="name">{{ item.name }}</h3>
            <span class="type-tag">{{ item.type }}</span>
          </div>
          <div :class="['status-badge', `badge-${item.status}`]">
            <CheckCircle v-if="item.status === 'connected'" :size="14" />
            <WifiOff v-else-if="item.status === 'disconnected'" :size="14" />
            <AlertCircle v-else :size="14" />
            <span>{{ item.status }}</span>
          </div>
        </div>

        <div v-if="item.last_error" class="error-box">
          <AlertCircle :size="14" />
          <span>{{ item.last_error }}</span>
        </div>

        <div class="card-footer">
          <button
            class="btn-reconnect"
            :disabled="reconnectingId === item.id"
            @click="reconnect(item)"
          >
            <RefreshCw :size="14" :class="{ 'animate-spin': reconnectingId === item.id }" />
            <span>{{ t('integrations.reconnect') }}</span>
          </button>
        </div>
      </div>
    </div>

    <div v-else-if="!loading" class="empty-state glass-card">
      <Network :size="36" class="empty-icon" />
      <h3 class="empty-title">{{ t('integrations.emptyTitle') }}</h3>
      <p class="empty-desc">{{ t('integrations.emptyDesc') }}</p>
    </div>
  </div>
</template>

<style scoped>
.integrations-view {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.02em;
}

.section-subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 4px;
}

.btn-refresh {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: var(--radius-md);
  font-size: 13px;
  color: var(--text-main);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-refresh:hover:not(:disabled) {
  background: var(--bg-surface-hover);
}

.integrations-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
}

.integration-card {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
  transition: all var(--transition-normal);
}

.card-top {
  display: flex;
  align-items: center;
  gap: 14px;
}

.icon-box {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: rgba(14, 165, 233, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
  flex-shrink: 0;
}

.info {
  flex: 1;
  min-width: 0;
}

.name {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.type-tag {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-muted);
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 500;
  text-transform: capitalize;
}

.badge-connected {
  background: var(--color-success-bg);
  color: var(--color-success);
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.badge-disconnected {
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-dim);
  border: 1px solid var(--border-card);
}

.badge-error {
  background: var(--color-danger-bg);
  color: var(--color-danger);
  border: 1px solid rgba(244, 63, 94, 0.3);
}

.error-box {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  background: var(--color-danger-bg);
  color: var(--color-danger);
  font-size: 12px;
}

.card-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}

.btn-reconnect {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  font-size: 13px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-reconnect:hover:not(:disabled) {
  background: var(--bg-surface-hover);
  border-color: rgba(255, 255, 255, 0.2);
}

.empty-state {
  padding: 48px 24px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  max-width: 500px;
  margin: 30px auto;
  border-radius: var(--radius-xl);
}

.empty-icon {
  color: var(--primary);
  margin-bottom: 16px;
  opacity: 0.8;
}

.empty-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-main);
}

.empty-desc {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 8px;
  line-height: 1.5;
}
</style>
