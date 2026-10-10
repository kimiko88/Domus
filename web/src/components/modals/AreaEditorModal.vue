<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Area } from '../../api/types'
import BaseButton from '../ui/BaseButton.vue'
import {
  X,
  Grid,
  Sofa,
  Utensils,
  Bed,
  Bath,
  Trees,
  Laptop,
  Car,
  Sun,
  Tv,
  AlertCircle,
  Sparkles
} from '@lucide/vue'

const { t } = useI18n()

const props = defineProps<{
  isOpen: boolean
  area?: Area | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', payload: { name: string; icon: string; sort_order: number }): void
}>()

const name = ref('')
const selectedIcon = ref('grid')
const sortOrder = ref(0)
const error = ref<string | null>(null)

const presetIcons = [
  { id: 'sofa', label: 'Soggiorno', component: Sofa },
  { id: 'utensils', label: 'Cucina', component: Utensils },
  { id: 'bed', label: 'Camera', component: Bed },
  { id: 'bath', label: 'Bagno', component: Bath },
  { id: 'laptop', label: 'Studio', component: Laptop },
  { id: 'trees', label: 'Giardino', component: Trees },
  { id: 'sun', label: 'Terrazzo', component: Sun },
  { id: 'car', label: 'Garage', component: Car },
  { id: 'tv', label: 'Cinema / TV', component: Tv },
  { id: 'grid', label: 'Altro', component: Grid },
]

watch(
  () => props.area,
  (val) => {
    if (val) {
      name.value = val.name || ''
      selectedIcon.value = val.icon || 'grid'
      sortOrder.value = val.sort_order ?? val.order_index ?? 0
    } else {
      name.value = ''
      selectedIcon.value = 'grid'
      sortOrder.value = 0
    }
    error.value = null
  },
  { immediate: true }
)

watch(
  () => props.isOpen,
  (val) => {
    if (val && !props.area) {
      name.value = ''
      selectedIcon.value = 'grid'
      sortOrder.value = 0
      error.value = null
    }
  }
)

function handleSubmit() {
  const trimmed = name.value.trim()
  if (!trimmed) {
    error.value = "Il nome dell'area è obbligatorio"
    return
  }
  error.value = null
  emit('save', {
    name: trimmed,
    icon: selectedIcon.value,
    sort_order: Number(sortOrder.value) || 0
  })
}
</script>

<template>
  <div v-if="isOpen" class="modal-backdrop" @click.self="emit('close')">
    <div class="modal-card glass-panel animate-enter">
      <div class="modal-header">
        <div class="header-title">
          <Sparkles :size="20" class="title-icon" />
          <h3>{{ area ? t('areas.editArea') : t('areas.newArea') }}</h3>
        </div>
        <button class="btn-close" aria-label="Close modal" @click="emit('close')">
          <X :size="18" />
        </button>
      </div>

      <div v-if="error" class="error-alert">
        <AlertCircle :size="16" />
        <span>{{ error }}</span>
      </div>

      <form class="modal-form" @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="area-name">{{ t('areas.areaName') }} *</label>
          <input
            id="area-name"
            v-model="name"
            name="area-name"
            type="text"
            :placeholder="t('areas.areaNamePlaceholder')"
            required
            autocomplete="off"
          />
        </div>

        <div class="form-group">
          <label>{{ t('areas.icon') }}</label>
          <div class="icon-selector-grid">
            <button
              v-for="item in presetIcons"
              :key="item.id"
              type="button"
              class="icon-option"
              :class="{ 'is-selected': selectedIcon === item.id }"
              :title="item.label"
              @click="selectedIcon = item.id"
            >
              <component :is="item.component" :size="20" />
              <span class="icon-label">{{ item.label }}</span>
            </button>
          </div>
        </div>

        <div class="form-group">
          <label for="area-sort-order">{{ t('areas.sortOrder') }}</label>
          <input
            id="area-sort-order"
            v-model.number="sortOrder"
            name="area-sort-order"
            type="number"
            min="0"
            step="1"
            placeholder="0"
          />
        </div>

        <div class="modal-footer">
          <BaseButton type="button" variant="ghost" class="btn-cancel" @click="emit('close')">
            {{ t('common.cancel') }}
          </BaseButton>
          <BaseButton type="submit" variant="primary">
            {{ area ? t('common.save') : 'Crea Area' }}
          </BaseButton>
        </div>
      </form>
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
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
  border-radius: var(--radius-xl);
  padding: 28px;
  border: 1px solid var(--border-card);
  box-shadow: var(--shadow-lg);
  background: var(--bg-surface);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.title-icon {
  color: var(--primary);
}

h3 {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.btn-close {
  background: transparent;
  color: var(--text-dim);
  padding: 4px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color var(--transition-fast);
}

.btn-close:hover {
  color: var(--text-main);
}

.error-alert {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--radius-md);
  background: var(--color-danger-bg);
  color: var(--color-danger);
  font-size: 13px;
  margin-bottom: 16px;
}

.modal-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

label {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-muted);
}

input {
  width: 100%;
  padding: 10px 14px;
  background: var(--bg-surface-elevated, var(--bg-surface));
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  color: var(--text-main);
  font-size: 14px;
  box-sizing: border-box;
}

input:focus {
  border-color: var(--primary);
  outline: none;
}

.field-hint {
  font-size: 11px;
  color: var(--text-dim);
}

.icon-selector-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 8px;
}

@media (max-width: 480px) {
  .icon-selector-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

.icon-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 6px;
  background: var(--bg-surface-elevated, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.icon-option:hover {
  background: rgba(14, 165, 233, 0.08);
  border-color: rgba(14, 165, 233, 0.3);
  color: var(--text-main);
}

.icon-option.is-selected {
  background: rgba(14, 165, 233, 0.18);
  border-color: var(--primary);
  color: var(--primary);
  box-shadow: 0 0 10px rgba(14, 165, 233, 0.2);
}

.icon-label {
  font-size: 10px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}
</style>
