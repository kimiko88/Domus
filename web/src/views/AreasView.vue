<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAreasStore } from '../stores/areas'
import { useEntitiesStore } from '../stores/entities'
import { useNotificationsStore } from '../stores/notifications'
import type { Area } from '../api/types'
import BaseButton from '../components/ui/BaseButton.vue'
import AreaEditorModal from '../components/modals/AreaEditorModal.vue'
import {
  Grid,
  Layers,
  Plus,
  Pencil,
  Trash2,
  Sofa,
  Utensils,
  Bed,
  Bath,
  Laptop,
  Trees,
  Sun,
  Car,
  Tv
} from '@lucide/vue'

const { t } = useI18n()

const areasStore = useAreasStore()
const entitiesStore = useEntitiesStore()
const notificationsStore = useNotificationsStore()

const isEditorOpen = ref(false)
const editingArea = ref<Area | null>(null)

onMounted(async () => {
  try {
    await Promise.all([
      areasStore.fetchAreas(),
      entitiesStore.fetchEntities()
    ])
  } catch (err) {
    console.warn('AreasView data fetch warning:', err)
  }
})

function getEntitiesForArea(areaId: string) {
  return entitiesStore.allEntities.filter(
    (e) => (e.attributes as any)?.area_id === areaId || (e as any).area_id === areaId
  )
}

function getAreaIcon(icon?: string) {
  switch (icon?.toLowerCase()) {
    case 'sofa':
    case 'living':
      return Sofa
    case 'utensils':
    case 'kitchen':
      return Utensils
    case 'bed':
    case 'bedroom':
      return Bed
    case 'bath':
    case 'bathroom':
      return Bath
    case 'laptop':
    case 'office':
      return Laptop
    case 'trees':
    case 'garden':
      return Trees
    case 'sun':
    case 'balcony':
    case 'terrace':
      return Sun
    case 'car':
    case 'garage':
      return Car
    case 'tv':
    case 'cinema':
      return Tv
    default:
      return Grid
  }
}

function openNewModal() {
  editingArea.value = null
  isEditorOpen.value = true
}

function openEditModal(area: Area) {
  editingArea.value = area
  isEditorOpen.value = true
}

async function handleDelete(area: Area) {
  if (confirm(t('areas.confirmDelete', { name: area.name }))) {
    try {
      await areasStore.deleteArea(area.id)
      notificationsStore.success(t('areas.deleteSuccess'), `"${area.name}"`)
    } catch (err: any) {
      notificationsStore.error(t('common.error'), err?.message || t('areas.deleteError'))
    }
  }
}

async function handleSaveArea(payload: { name: string; icon: string; sort_order: number }) {
  try {
    if (editingArea.value) {
      await areasStore.updateArea(editingArea.value.id, payload)
      notificationsStore.success(t('areas.saveSuccess'), `"${payload.name}"`)
    } else {
      await areasStore.createArea(payload)
      notificationsStore.success(t('areas.saveSuccess'), `"${payload.name}"`)
    }
    isEditorOpen.value = false
    editingArea.value = null
  } catch (err: any) {
    notificationsStore.error(t('common.error'), err?.message || t('areas.saveError'))
  }
}
</script>

<template>
  <div class="areas-view">
    <div class="header-row">
      <div>
        <h2 class="section-title">{{ t('areas.title') }}</h2>
        <p class="section-subtitle">{{ t('areas.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <BaseButton variant="primary" @click="openNewModal">
          <Plus :size="16" />
          {{ t('areas.newArea') }}
        </BaseButton>
      </div>
    </div>

    <!-- Areas Grid -->
    <div v-if="areasStore.sortedAreas.length > 0" class="areas-grid">
      <div
        v-for="area in areasStore.sortedAreas"
        :key="area.id"
        class="area-card glass-card"
      >
        <div class="area-card-top">
          <div class="area-icon-box">
            <component :is="getAreaIcon(area.icon)" :size="24" />
          </div>
          <div class="area-info">
            <h3 class="area-name">{{ area.name }}</h3>
            <span class="area-count">
              {{ getEntitiesForArea(area.id).length }} {{ t('areas.devicesCount') }}
            </span>
          </div>

          <div class="card-actions">
            <button
              class="btn-card-action"
              :title="t('areas.editArea')"
              :aria-label="t('areas.editArea')"
              @click="openEditModal(area)"
            >
              <Pencil :size="15" />
            </button>
            <button
              class="btn-card-action is-danger"
              :title="t('areas.deleteArea')"
              :aria-label="t('areas.deleteArea')"
              @click="handleDelete(area)"
            >
              <Trash2 :size="15" />
            </button>
          </div>
        </div>

        <div v-if="getEntitiesForArea(area.id).length > 0" class="area-entities-preview">
          <div
            v-for="entity in getEntitiesForArea(area.id).slice(0, 4)"
            :key="entity.id"
            class="entity-pill"
          >
            <span class="pill-dot" :class="{ 'is-on': entity.state === 'on' }" />
            <span class="pill-name">{{ entity.name }}</span>
          </div>
          <span v-if="getEntitiesForArea(area.id).length > 4" class="more-count">
            +{{ getEntitiesForArea(area.id).length - 4 }} {{ t('areas.moreEntities') }}
          </span>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="empty-state glass-card">
      <Layers :size="42" class="empty-icon" />
      <h3 class="empty-title">{{ t('areas.emptyTitle') }}</h3>
      <p class="empty-desc">{{ t('areas.emptyDesc') }}</p>
      <BaseButton variant="primary" class="empty-btn" @click="openNewModal">
        <Plus :size="16" />
        {{ t('areas.createFirst') }}
      </BaseButton>
    </div>

    <!-- Editor Modal -->
    <AreaEditorModal
      :is-open="isEditorOpen"
      :area="editingArea"
      @close="isEditorOpen = false"
      @save="handleSaveArea"
    />
  </div>
</template>

<style scoped>
.areas-view {
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
  gap: 12px;
}

.areas-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
}

.area-card {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
  transition: all var(--transition-normal);
  position: relative;
}

.area-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
}

.area-card-top {
  display: flex;
  align-items: center;
  gap: 16px;
}

.area-icon-box {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.2) 0%, rgba(99, 102, 241, 0.15) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
  flex-shrink: 0;
}

.area-info {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.area-name {
  font-size: 17px;
  font-weight: 600;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.area-count {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 2px;
}

.card-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
}

.btn-card-action {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-surface-elevated, rgba(255, 255, 255, 0.05));
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-card-action:hover {
  background: rgba(14, 165, 233, 0.15);
  border-color: rgba(14, 165, 233, 0.4);
  color: var(--primary);
}

.btn-card-action.is-danger:hover {
  background: var(--color-danger-bg);
  border-color: var(--color-danger);
  color: var(--color-danger);
}

.area-entities-preview {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}

.entity-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  font-size: 12px;
  color: var(--text-main);
}

.pill-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: var(--text-dim);
}

.pill-dot.is-on {
  background-color: #fbbf24;
  box-shadow: 0 0 6px rgba(251, 191, 36, 0.6);
}

.pill-name {
  max-width: 120px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.more-count {
  font-size: 12px;
  color: var(--text-dim);
  font-family: var(--font-mono);
}

.empty-state {
  padding: 48px 24px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  max-width: 520px;
  margin: 40px auto;
  border-radius: var(--radius-xl);
}

.empty-icon {
  color: var(--primary);
  margin-bottom: 16px;
  opacity: 0.85;
}

.empty-title {
  font-size: 19px;
  font-weight: 700;
  color: var(--text-main);
}

.empty-desc {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 8px;
  margin-bottom: 24px;
  line-height: 1.6;
}

.empty-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
</style>
