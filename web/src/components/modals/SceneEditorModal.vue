<script setup lang="ts">
import { ref, watch, toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Scene, SceneAction } from '../../api/types'
import BaseButton from '../ui/BaseButton.vue'
import { useModalA11y } from '../../composables/useModalA11y'
import { X, Plus, Trash2, Clapperboard, AlertCircle } from '@lucide/vue'

interface Props {
  isOpen: boolean
  scene?: Scene | null
}

const props = withDefaults(defineProps<Props>(), {
  isOpen: false,
  scene: null
})

const emit = defineEmits<{
  (e: 'save', data: Partial<Scene>): void
  (e: 'close'): void
}>()

const modalContainerRef = ref<HTMLElement | null>(null)
useModalA11y({
  isOpen: toRef(props, 'isOpen'),
  modalRef: modalContainerRef,
  onClose: () => emit('close')
})

const { t } = useI18n()

const name = ref('')
const icon = ref('clapperboard')
const actions = ref<SceneAction[]>([])
const error = ref<string | null>(null)

watch(
  () => props.scene,
  (val) => {
    if (val) {
      name.value = val.name
      icon.value = val.icon || 'clapperboard'
      actions.value = (val.actions || []).map((a) => ({ ...a, params: { ...a.params } }))
    } else {
      name.value = ''
      icon.value = 'clapperboard'
      actions.value = []
    }
    error.value = null
  },
  { immediate: true }
)

function addAction() {
  actions.value.push({
    entity_key: '',
    service: 'turn_on',
    params: {}
  })
}

function removeAction(index: number) {
  actions.value.splice(index, 1)
}

function handleSubmit() {
  error.value = null
  if (!name.value.trim()) {
    error.value = t('scenes.editor.nameRequired')
    return
  }
  if (actions.value.length === 0) {
    error.value = t('scenes.editor.actionRequired')
    return
  }
  for (const a of actions.value) {
    if (!a.entity_key.trim()) {
      error.value = t('scenes.editor.entityRequired')
      return
    }
  }

  emit('save', {
    name: name.value.trim(),
    icon: icon.value,
    actions: actions.value
  })
}
</script>

<template>
  <div v-if="isOpen" class="modal-backdrop" @click="emit('close')">
    <div
      ref="modalContainerRef"
      class="modal-card glass-panel animate-enter"
      role="dialog"
      aria-modal="true"
      aria-labelledby="scene-modal-title"
      @click.stop
    >
      <div class="modal-header">
        <div class="header-title">
          <Clapperboard :size="20" class="title-icon" />
          <h3 id="scene-modal-title">{{ scene ? t('scenes.editor.editTitle') : t('scenes.editor.newTitle') }}</h3>
        </div>
        <button class="btn-close" :aria-label="t('scenes.editor.closeWindow')" @click="emit('close')">
          <X :size="18" />
        </button>
      </div>

      <div v-if="error" class="error-alert">
        <AlertCircle :size="16" />
        <span>{{ error }}</span>
      </div>

      <form class="modal-form" @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="scene-name">{{ t('scenes.editor.name') }}</label>
          <input
            id="scene-name"
            v-model="name"
            name="scene-name"
            type="text"
            :placeholder="t('scenes.editor.namePlaceholder')"
            required
          />
        </div>

        <div class="form-group">
          <label for="scene-icon">{{ t('scenes.editor.icon') }}</label>
          <select id="scene-icon" v-model="icon" name="scene-icon">
            <option value="clapperboard" >{{ t('scenes.editor.icons.clapperboard') }}</option>
            <option value="moon" >{{ t('scenes.editor.icons.night') }}</option>
            <option value="sun" >{{ t('scenes.editor.icons.day') }}</option>
            <option value="tv" >{{ t('scenes.editor.icons.cinema') }}</option>
            <option value="coffee" >{{ t('scenes.editor.icons.morning') }}</option>
          </select>
        </div>

        <!-- Actions List -->
        <div class="actions-section">
          <div class="actions-header">
            <label>{{ t('scenes.editor.configuredActions', { count: actions.length }) }}</label>
            <button type="button" class="btn-add-action" @click="addAction">
              <Plus :size="14" />
              <span>{{ t('scenes.editor.addAction') }}</span>
            </button>
          </div>

          <div v-if="actions.length > 0" class="action-list">
            <div
              v-for="(action, idx) in actions"
              :key="idx"
              class="action-row glass-card"
            >
              <input
                v-model="action.entity_key"
                name="action-entity"
                type="text"
                :placeholder="t('scenes.editor.entityPlaceholder')" :aria-label="t('scenes.editor.entityKey')"
                required
                class="entity-input font-mono"
              />

              <select v-model="action.service" class="service-select" :aria-label="t('scenes.editor.action')">
                <option value="turn_on">{{ t('scenes.editor.actions.turnOn') }}</option>
                <option value="turn_off">{{ t('scenes.editor.actions.turnOff') }}</option>
                <option value="toggle">{{ t('scenes.editor.actions.toggle') }}</option>
              </select>

              <button
                type="button"
                class="btn-remove-action"
                :title="t('scenes.editor.removeAction')" :aria-label="t('scenes.editor.removeAction')"
                @click="removeAction(idx)"
              >
                <Trash2 :size="16" />
              </button>
            </div>
          </div>
          <div v-else class="empty-actions">
            {{ t('scenes.editor.emptyActions') }}
          </div>
        </div>

        <div class="modal-footer">
          <BaseButton type="button" variant="ghost" class="btn-cancel" @click="emit('close')">
            {{ t('common.cancel') }}
          </BaseButton>
          <BaseButton type="submit" variant="primary">
            {{ t('scenes.editor.save') }}
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
  max-width: 540px;
  max-height: 90vh;
  overflow-y: auto;
  border-radius: var(--radius-xl);
  padding: 28px;
  border: 1px solid var(--border-card);
  box-shadow: var(--shadow-lg);
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
}

.btn-close {
  background: transparent;
  color: var(--text-dim);
  padding: 4px;
  border-radius: var(--radius-sm);
  cursor: pointer;
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
  gap: 18px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
}

input, select {
  width: 100%;
  padding: 10px 14px;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  color: var(--text-main);
  font-size: 14px;
}

input:focus, select:focus {
  border-color: var(--primary);
  outline: none;
}

.actions-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.actions-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.btn-add-action {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  font-size: 12px;
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  border: 1px solid rgba(14, 165, 233, 0.3);
}

.action-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 220px;
  overflow-y: auto;
}

.action-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
}

.entity-input {
  flex: 1;
  font-size: 13px;
}

.service-select {
  width: 120px;
  font-size: 13px;
}

.btn-remove-action {
  background: transparent;
  color: var(--text-dim);
  padding: 6px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-remove-action:hover {
  color: var(--color-danger);
  background: rgba(244, 63, 94, 0.1);
}

.empty-actions {
  font-size: 13px;
  color: var(--text-dim);
  text-align: center;
  padding: 16px;
  border: 1px dashed var(--border-card);
  border-radius: var(--radius-md);
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 10px;
  padding-top: 16px;
  border-top: 1px solid var(--border-subtle);
}
</style>
