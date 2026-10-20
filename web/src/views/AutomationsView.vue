<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAutomationsStore } from '../stores/automations'
import { useNotificationsStore } from '../stores/notifications'
import type { Automation } from '../api/types'
import AutomationCard from '../components/cards/AutomationCard.vue'
import AutomationEditorModal from '../components/modals/AutomationEditorModal.vue'
import AutomationRunsModal from '../components/modals/AutomationRunsModal.vue'
import AutomationFlowchartModal from '../components/modals/AutomationFlowchartModal.vue'
import BaseButton from '../components/ui/BaseButton.vue'
import { Zap, Plus, Search, Globe, Copy, Check, Trash2, Key, Link2, X, Layers, Workflow } from '@lucide/vue'
import { apiClient } from '../api/client'
import type { InboundWebhook } from '../api/types'

const { t } = useI18n()
const automationsStore = useAutomationsStore()
const notificationsStore = useNotificationsStore()

const searchQuery = ref('')
const filterStatus = ref<'all' | 'active' | 'disabled'>('all')

const isEditorOpen = ref(false)
const editingAutomation = ref<Automation | null>(null)
const isFlowchartOpen = ref(false)

const isRunsModalOpen = ref(false)
const activeRunAutomation = ref<Automation | null>(null)

// Inbound Webhooks State
const webhooks = ref<InboundWebhook[]>([])
const isWebhookModalOpen = ref(false)
const newWhName = ref('')
const newWhSecret = ref('')
const newWhMethods = ref('POST')
const copiedWhId = ref<string | null>(null)

async function loadAutomations() {
  try {
    await automationsStore.fetchAutomations()
  } catch (err) {
    console.warn('Failed to load automations:', err)
  }
}

async function loadWebhooks() {
  try {
    const list = await apiClient.get<InboundWebhook[]>('/api/v1/webhook-configs')
    if (list) webhooks.value = list
  } catch {}
}

async function handleCreateWebhook() {
  if (!newWhName.value.trim()) return
  try {
    const created = await apiClient.post<InboundWebhook>('/api/v1/webhook-configs', {
      name: newWhName.value.trim(),
      secret_token: newWhSecret.value.trim(),
      allowed_methods: newWhMethods.value
    })
    if (created) webhooks.value.push(created)
    isWebhookModalOpen.value = false
    newWhName.value = ''
    newWhSecret.value = ''
    notificationsStore.success('Webhook Inbound Creato', created.name)
  } catch (err: any) {
    notificationsStore.error('Errore creazione webhook', err?.message)
  }
}

async function handleDeleteWebhook(wh: InboundWebhook) {
  if (!confirm(`Sei sicuro di voler eliminare il webhook "${wh.name}"?`)) return
  try {
    await apiClient.delete(`/api/v1/webhook-configs/${wh.id}`)
    webhooks.value = webhooks.value.filter(w => w.id !== wh.id)
    notificationsStore.success('Webhook eliminato', wh.name)
  } catch (err: any) {
    notificationsStore.error('Errore eliminazione', err?.message)
  }
}

async function handleCopyWebhookUrl(whId: string) {
  const base = typeof window !== 'undefined' ? window.location.origin : ''
  const fullUrl = `${base}/api/v1/webhooks/${whId}`
  try {
    if (navigator?.clipboard) {
      await navigator.clipboard.writeText(fullUrl)
    }
    copiedWhId.value = whId
    setTimeout(() => { copiedWhId.value = null }, 2500)
  } catch {}
}

onMounted(() => {
  loadAutomations()
  loadWebhooks()
})

const filteredAutomations = computed(() => {
  return automationsStore.automations.filter((a) => {
    if (filterStatus.value === 'active' && !a.enabled) return false
    if (filterStatus.value === 'disabled' && a.enabled) return false

    if (searchQuery.value.trim()) {
      const q = searchQuery.value.toLowerCase()
      const matchName = a.name.toLowerCase().includes(q)
      const matchDesc = a.description?.toLowerCase().includes(q) || false
      if (!matchName && !matchDesc) return false
    }

    return true
  })
})

function openNewModal() {
  editingAutomation.value = null
  isEditorOpen.value = true
}

function handleEdit(automation: Automation) {
  editingAutomation.value = automation
  isEditorOpen.value = true
}

async function handleDelete(automation: Automation) {
  if (confirm(t('automations.confirmDelete', { name: automation.name }))) {
    try {
      await automationsStore.deleteAutomation(automation.id)
      notificationsStore.success(t('automations.deleteSuccess'), `"${automation.name}"`)
    } catch (err: any) {
      notificationsStore.error(t('common.error'), err?.message || t('automations.deleteError'))
    }
  }
}

function handleViewRuns(automation: Automation) {
  activeRunAutomation.value = automation
  isRunsModalOpen.value = true
}

async function handleSaveAutomation(data: Partial<Automation>) {
  try {
    if (editingAutomation.value) {
      await automationsStore.updateAutomation(editingAutomation.value.id, data)
      notificationsStore.success(t('automations.saveSuccess'), `"${data.name}"`)
    } else {
      await automationsStore.createAutomation(data)
      notificationsStore.success(t('automations.saveSuccess'), `"${data.name}"`)
    }
    isEditorOpen.value = false
    editingAutomation.value = null
  } catch (err: any) {
    notificationsStore.error(t('common.error'), err?.message || t('automations.saveError'))
  }
}
</script>

<template>
  <div class="automations-view">
    <div class="header-row">
      <div>
        <h2 class="section-title">{{ t('automations.title') }}</h2>
        <p class="section-subtitle">{{ t('automations.subtitle') }}</p>
      </div>

      <div class="header-actions">
        <router-link to="/automations/flow" class="btn-flow-editor glass-panel" title="Apri Editor Visuale a Nodi">
          <Workflow :size="16" />
          <span>Editor a Nodi</span>
        </router-link>
        <button class="btn-flowchart glass-panel" @click="isFlowchartOpen = true">
          <Layers :size="16" />
          <span>{{ t('flowchart.btnFlowchart') }}</span>
        </button>
        <BaseButton variant="primary" class="btn-new-automation" @click="openNewModal">
          <Plus :size="16" />
          <span>{{ t('automations.newAutomation') }}</span>
        </BaseButton>
      </div>
    </div>

    <!-- Search & Filters Toolbar -->
    <div class="toolbar-row">
      <div class="search-box">
        <Search :size="16" class="search-icon" />
        <input
          v-model="searchQuery"
          type="search"
          :placeholder="t('automations.searchPlaceholder')"
        />
      </div>

      <div class="filter-pills">
        <button
          :class="['filter-pill', { active: filterStatus === 'all' }]"
          @click="filterStatus = 'all'"
        >
          {{ t('common.all') }} ({{ automationsStore.automations.length }})
        </button>
        <button
          :class="['filter-pill', { active: filterStatus === 'active' }]"
          @click="filterStatus = 'active'"
        >
          {{ t('common.active') }} ({{ automationsStore.automations.filter((a) => a.enabled).length }})
        </button>
        <button
          :class="['filter-pill', { active: filterStatus === 'disabled' }]"
          @click="filterStatus = 'disabled'"
        >
          {{ t('common.disabled') }} ({{ automationsStore.automations.filter((a) => !a.enabled).length }})
        </button>
      </div>
    </div>

    <!-- Automations Grid -->
    <div v-if="filteredAutomations.length > 0" class="automations-grid">
      <AutomationCard
        v-for="automation in filteredAutomations"
        :key="automation.id"
        :automation="automation"
        @edit="handleEdit"
        @delete="handleDelete"
        @view-runs="handleViewRuns"
      />
    </div>

    <!-- Empty State -->
    <div v-else-if="!automationsStore.loading" class="empty-state glass-card">
      <Zap :size="36" class="empty-icon" />
      <h3 class="empty-title">{{ t('automations.emptyTitle') }}</h3>
      <p class="empty-desc">
        {{ searchQuery ? t('automations.noResults') : t('automations.emptyDesc') }}
      </p>
      <BaseButton v-if="!searchQuery" variant="primary" style="margin-top: 16px;" @click="openNewModal">
        <Plus :size="16" />
        <span>{{ t('automations.newAutomation') }}</span>
      </BaseButton>
    </div>

    <!-- Inbound Webhooks Section -->
    <div class="webhooks-section glass-card">
      <div class="webhooks-header">
        <div class="webhooks-title-group">
          <div class="webhook-icon-box">
            <Globe :size="20" />
          </div>
          <div>
            <h3 class="wh-title">Webhook Esterni Riceventi (Inbound)</h3>
            <p class="wh-subtitle">Ricevi payload HTTP protetti da HMAC da IFTTT, n8n, OwnTracks o sensori esterni per innescare eventi</p>
          </div>
        </div>
        <BaseButton variant="secondary" size="sm" @click="isWebhookModalOpen = true">
          <Plus :size="16" />
          <span>Nuovo Webhook</span>
        </BaseButton>
      </div>

      <div v-if="webhooks.length === 0" class="wh-empty">
        <Globe :size="28" class="empty-icon" />
        <p>Nessun webhook configurato. Crea un endpoint per ricevere richieste esterne in sicurezza.</p>
      </div>

      <div v-else class="wh-list">
        <div v-for="wh in webhooks" :key="wh.id" class="wh-card">
          <div class="wh-card-top">
            <span class="wh-name">{{ wh.name }}</span>
            <span class="wh-method-badge">{{ wh.allowed_methods }}</span>
            <span class="wh-count-badge">{{ wh.trigger_count }} esecuzioni</span>
          </div>

          <div class="wh-url-box">
            <span class="wh-url">/api/v1/webhooks/{{ wh.webhook_id }}</span>
            <button
              class="wh-copy-btn"
              :title="copiedWhId === wh.webhook_id ? 'Copiato!' : 'Copia URL Webhook'"
              @click="handleCopyWebhookUrl(wh.webhook_id)"
            >
              <Check v-if="copiedWhId === wh.webhook_id" :size="14" class="text-success" />
              <Copy v-else :size="14" />
            </button>
          </div>

          <div class="wh-card-bottom">
            <span class="wh-secret">Secret: <code>{{ wh.secret_token ? 'HMAC-SHA256 protetto' : 'Libero' }}</code></span>
            <button class="wh-delete-btn" title="Elimina" @click="handleDeleteWebhook(wh)">
              <Trash2 :size="15" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Modals -->
    <AutomationEditorModal
      :is-open="isEditorOpen"
      :automation="editingAutomation"
      @save="handleSaveAutomation"
      @close="isEditorOpen = false"
    />

    <AutomationRunsModal
      :is-open="isRunsModalOpen"
      :automation="activeRunAutomation"
      @close="isRunsModalOpen = false"
    />

    <AutomationFlowchartModal
      :is-open="isFlowchartOpen"
      @close="isFlowchartOpen = false"
      @save="notificationsStore.success('Flusso Salvato', 'Automazione da Flowchart creata con successo!')"
    />

    <!-- New Webhook Modal -->
    <Teleport to="body">
      <div v-if="isWebhookModalOpen" class="modal-backdrop" @click.self="isWebhookModalOpen = false">
        <div class="modal-card glass-card">
          <div class="modal-header">
            <div class="modal-title-box">
              <Globe :size="20" class="modal-icon" />
              <h3 class="modal-title">Nuovo Webhook Inbound</h3>
            </div>
            <button class="modal-close" aria-label="Chiudi" @click="isWebhookModalOpen = false">
              <X :size="18" />
            </button>
          </div>

          <div class="modal-body">
            <div class="form-group">
              <label for="webhook-name" class="form-label">Nome Webhook</label>
              <input
                id="webhook-name"
                name="webhook_name"
                v-model="newWhName"
                type="text"
                class="text-input"
                placeholder="es. IFTTT Arrivo a Casa, OwnTracks GPS"
              />
            </div>

            <div class="form-group">
              <label for="webhook-secret" class="form-label">Secret HMAC Token (opzionale per firma X-Domus-Signature)</label>
              <input
                id="webhook-secret"
                name="webhook_secret"
                v-model="newWhSecret"
                type="text"
                class="text-input"
                placeholder="es. secret_my_token_123"
              />
            </div>

            <div class="form-group">
              <label for="webhook-method" class="form-label">Metodo HTTP Consentito</label>
              <select id="webhook-method" name="webhook_method" v-model="newWhMethods" class="select-input">
                <option value="POST">POST (Standard JSON payload)</option>
                <option value="GET">GET (Query string parameters)</option>
                <option value="PUT">PUT</option>
              </select>
            </div>
          </div>

          <div class="modal-footer">
            <BaseButton variant="ghost" size="md" @click="isWebhookModalOpen = false">Annulla</BaseButton>
            <BaseButton variant="primary" size="md" @click="handleCreateWebhook">Crea Webhook</BaseButton>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.automations-view {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 16px;
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

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.btn-flowchart {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: var(--radius-md);
  background: rgba(14, 165, 233, 0.15);
  border: 1px solid rgba(14, 165, 233, 0.3);
  color: var(--primary);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-flowchart:hover {
  background: rgba(14, 165, 233, 0.25);
  transform: translateY(-1px);
}

.btn-flow-editor {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: var(--radius-md);
  background: rgba(168, 85, 247, 0.15);
  border: 1px solid rgba(168, 85, 247, 0.35);
  color: #c084fc;
  font-size: 13px;
  font-weight: 600;
  text-decoration: none;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-flow-editor:hover {
  background: rgba(168, 85, 247, 0.25);
  transform: translateY(-1px);
}

.toolbar-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 240px;
  max-width: 440px;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--text-dim);
  pointer-events: none;
}

.search-box input {
  width: 100%;
  padding: 9px 14px 9px 38px;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  color: var(--text-main);
  font-size: 13px;
}

.search-box input:focus {
  border-color: var(--primary);
  outline: none;
}

.filter-pills {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-pill {
  padding: 8px 14px;
  border-radius: var(--radius-full);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-muted);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.filter-pill:hover {
  background: var(--bg-surface-hover);
  color: var(--text-main);
}

.filter-pill.active {
  background: #f59e0b;
  color: white;
  border-color: #f59e0b;
  box-shadow: 0 0 12px rgba(245, 158, 11, 0.4);
}

.automations-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 18px;
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
  color: #f59e0b;
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

/* Inbound Webhooks Section Styling */
.webhooks-section {
  padding: 24px 28px;
  border-radius: var(--radius-xl, 16px);
  margin-top: 36px;
  background: var(--surface-card, rgba(15, 23, 42, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.webhooks-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.webhooks-title-group {
  display: flex;
  align-items: center;
  gap: 16px;
}

.webhook-icon-box {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: rgba(14, 165, 233, 0.15);
  color: #0ea5e9;
  display: flex;
  align-items: center;
  justify-content: center;
}

.wh-title {
  font-size: 1.15rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-main);
}

.wh-subtitle {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin: 4px 0 0;
}

.wh-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 36px 20px;
  text-align: center;
  background: rgba(255, 255, 255, 0.02);
  border: 1px dashed var(--border-card, rgba(255, 255, 255, 0.15));
  border-radius: var(--radius-lg, 12px);
  gap: 10px;
}

.wh-empty .empty-icon {
  color: #f59e0b;
  margin-bottom: 4px;
}

.wh-empty p {
  color: var(--text-muted);
  font-size: 0.9rem;
  margin: 0;
}

.wh-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.wh-card {
  padding: 18px;
  border-radius: var(--radius-lg, 12px);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.wh-card-top {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.wh-name {
  font-weight: 600;
  color: var(--text-main);
  flex: 1;
}

.wh-method-badge {
  font-size: 0.7rem;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
}

.wh-count-badge {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.wh-url-box {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: rgba(0, 0, 0, 0.2);
  padding: 8px 12px;
  border-radius: 8px;
  font-family: var(--font-mono, monospace);
  font-size: 0.8rem;
  color: var(--text-muted);
}

.wh-copy-btn,
.wh-delete-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  color: var(--text-muted);
  padding: 4px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.wh-copy-btn:hover {
  color: var(--text-main);
  background: rgba(255, 255, 255, 0.08);
}

.wh-delete-btn:hover {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.15);
}

.wh-card-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.75rem;
  color: var(--text-dim);
}

/* Webhook Inbound Modal Styles */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: var(--bg-overlay, rgba(0, 0, 0, 0.75));
  backdrop-filter: blur(8px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.modal-card {
  width: 100%;
  max-width: 520px;
  background: var(--bg-surface, #1e293b);
  border: 1px solid var(--border-card, rgba(255, 255, 255, 0.12));
  border-radius: var(--radius-xl, 16px);
  padding: 24px;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.modal-title-box {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--primary, #0ea5e9);
}

.modal-title {
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--text-main, #f8fafc);
  margin: 0;
}

.modal-close {
  background: transparent;
  border: none;
  color: var(--text-muted, #94a3b8);
  padding: 6px;
  border-radius: var(--radius-sm, 6px);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast, 0.2s);
}

.modal-close:hover {
  color: var(--text-main, #fff);
  background: rgba(255, 255, 255, 0.08);
}

.modal-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text-muted, #94a3b8);
}

.text-input,
.select-input {
  width: 100%;
  box-sizing: border-box;
  padding: 10px 14px;
  background: rgba(15, 23, 42, 0.85);
  border: 1px solid var(--border-card, rgba(255, 255, 255, 0.15));
  border-radius: var(--radius-md, 8px);
  color: var(--text-main, #f8fafc);
  font-size: 0.9rem;
  outline: none;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.text-input:focus,
.select-input:focus {
  border-color: var(--primary, #0ea5e9);
  box-shadow: 0 0 0 2px rgba(14, 165, 233, 0.25);
}

.select-input option {
  background: #1e293b;
  color: #f8fafc;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.08));
}
</style>
