<script setup lang="ts">
import { computed } from 'vue'
import type { Scene } from '../../api/types'
import { useScenesStore } from '../../stores/scenes'
import { useNotificationsStore } from '../../stores/notifications'
import BaseButton from '../ui/BaseButton.vue'
import {
  Clapperboard,
  Play,
  Pencil,
  Trash2,
  Moon,
  Sun,
  Tv,
  Coffee,
  Sparkles
} from '@lucide/vue'

interface Props {
  scene: Scene
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'edit', scene: Scene): void
  (e: 'delete', scene: Scene): void
}>()

const scenesStore = useScenesStore()
const notificationsStore = useNotificationsStore()

const isApplying = computed(() => scenesStore.applyingId === props.scene.id)

const sceneIcon = computed(() => {
  const icon = props.scene.icon?.toLowerCase()
  if (icon === 'moon' || icon === 'night') return Moon
  if (icon === 'sun' || icon === 'morning') return Sun
  if (icon === 'film' || icon === 'tv' || icon === 'cinema') return Tv
  if (icon === 'coffee') return Coffee
  return Clapperboard
})

async function handleApply() {
  try {
    await scenesStore.applyScene(props.scene.id)
    notificationsStore.success('Scena Attivata', `"${props.scene.name}" è stata applicata`)
  } catch (err: any) {
    notificationsStore.error('Errore Scena', err?.message || 'Impossibile applicare la scena')
  }
}
</script>

<template>
  <div class="scene-card glass-card">
    <div class="card-main">
      <div class="scene-icon-box">
        <component :is="sceneIcon" :size="24" />
      </div>

      <div class="scene-info">
        <h3 class="scene-name" :title="scene.name">{{ scene.name }}</h3>
        <span class="scene-meta">
          {{ scene.actions.length }} Azioni
        </span>
      </div>
    </div>

    <div class="card-actions">
      <BaseButton
        variant="primary"
        size="sm"
        class="btn-apply"
        :loading="isApplying"
        @click="handleApply"
      >
        <Play :size="14" class="play-icon" />
        <span>Attiva</span>
      </BaseButton>

      <div class="secondary-actions">
        <button
          class="btn-icon btn-edit"
          title="Modifica scena"
          aria-label="Edit scene"
          @click="emit('edit', scene)"
        >
          <Pencil :size="15" />
        </button>
        <button
          class="btn-icon btn-delete"
          title="Elimina scena"
          aria-label="Delete scene"
          @click="emit('delete', scene)"
        >
          <Trash2 :size="15" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.scene-card {
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

.scene-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
}

.card-main {
  display: flex;
  align-items: center;
  gap: 14px;
}

.scene-icon-box {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, rgba(168, 85, 247, 0.2) 0%, rgba(99, 102, 241, 0.15) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #c084fc;
  flex-shrink: 0;
}

.scene-info {
  flex: 1;
  min-width: 0;
}

.scene-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.scene-meta {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 2px;
}

.card-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}

.play-icon {
  margin-right: 2px;
}

.secondary-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-icon {
  width: 32px;
  height: 32px;
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
