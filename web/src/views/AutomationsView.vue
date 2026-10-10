<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAutomationsStore } from '../stores/automations'
import { useNotificationsStore } from '../stores/notifications'
import type { Automation } from '../api/types'
import AutomationCard from '../components/cards/AutomationCard.vue'
import AutomationEditorModal from '../components/modals/AutomationEditorModal.vue'
import AutomationRunsModal from '../components/modals/AutomationRunsModal.vue'
import BaseButton from '../components/ui/BaseButton.vue'
import { Zap, Plus, Search } from '@lucide/vue'

const { t } = useI18n()
const automationsStore = useAutomationsStore()
const notificationsStore = useNotificationsStore()

const searchQuery = ref('')
const filterStatus = ref<'all' | 'active' | 'disabled'>('all')

const isEditorOpen = ref(false)
const editingAutomation = ref<Automation | null>(null)

const isRunsModalOpen = ref(false)
const activeRunAutomation = ref<Automation | null>(null)

async function loadAutomations() {
  try {
    await automationsStore.fetchAutomations()
  } catch (err) {
    console.warn('Failed to load automations:', err)
  }
}

onMounted(() => {
  loadAutomations()
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
</style>
