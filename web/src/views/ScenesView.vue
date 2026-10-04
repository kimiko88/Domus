<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useScenesStore } from '../stores/scenes'
import { useNotificationsStore } from '../stores/notifications'
import type { Scene } from '../api/types'
import SceneCard from '../components/cards/SceneCard.vue'
import SceneEditorModal from '../components/modals/SceneEditorModal.vue'
import BaseButton from '../components/ui/BaseButton.vue'
import { Clapperboard, Plus, RefreshCw, Sparkles } from '@lucide/vue'

const scenesStore = useScenesStore()
const notificationsStore = useNotificationsStore()

const isEditorOpen = ref(false)
const editingScene = ref<Scene | null>(null)

async function loadScenes() {
  try {
    await scenesStore.fetchScenes()
  } catch (err) {
    console.warn('Failed to fetch scenes:', err)
  }
}

onMounted(() => {
  loadScenes()
})

function openNewModal() {
  editingScene.value = null
  isEditorOpen.value = true
}

function handleEdit(scene: Scene) {
  editingScene.value = scene
  isEditorOpen.value = true
}

async function handleDelete(scene: Scene) {
  if (confirm(`Sei sicuro di voler eliminare la scena "${scene.name}"?`)) {
    try {
      await scenesStore.deleteScene(scene.id)
      notificationsStore.success('Scena Eliminata', `"${scene.name}" è stata rimossa`)
    } catch (err: any) {
      notificationsStore.error('Errore', err?.message || 'Impossibile eliminare la scena')
    }
  }
}

async function handleSaveScene(data: Partial<Scene>) {
  try {
    if (editingScene.value) {
      await scenesStore.updateScene(editingScene.value.id, data)
      notificationsStore.success('Scena Aggiornata', `"${data.name}" aggiornata con successo`)
    } else {
      await scenesStore.createScene(data)
      notificationsStore.success('Scena Creata', `"${data.name}" creata con successo`)
    }
    isEditorOpen.value = false
    editingScene.value = null
  } catch (err: any) {
    notificationsStore.error('Errore Salvataggio', err?.message || 'Impossibile salvare la scena')
  }
}
</script>

<template>
  <div class="scenes-view">
    <div class="header-row">
      <div>
        <h2 class="section-title">Scene</h2>
        <p class="section-subtitle">Preset istantanei per configurare più dispositivi con un solo tocco</p>
      </div>

      <div class="header-actions">
        <BaseButton
          variant="primary"
          class="btn-new-scene"
          @click="openNewModal"
        >
          <Plus :size="16" />
          <span>Nuova Scena</span>
        </BaseButton>
      </div>
    </div>

    <!-- Scenes Grid -->
    <div v-if="scenesStore.scenes.length > 0" class="scenes-grid">
      <SceneCard
        v-for="scene in scenesStore.scenes"
        :key="scene.id"
        :scene="scene"
        @edit="handleEdit"
        @delete="handleDelete"
      />
    </div>

    <!-- Empty State -->
    <div v-else-if="!scenesStore.loading" class="empty-state glass-card">
      <Clapperboard :size="36" class="empty-icon" />
      <h3 class="empty-title">Nessuna scena configurata</h3>
      <p class="empty-desc">
        Crea la tua prima scena (es. "Cinema", "Tutto Spento", "Lavoro") per raggruppare azioni su più luci o interruttori.
      </p>
      <BaseButton variant="primary" style="margin-top: 16px;" @click="openNewModal">
        <Plus :size="16" />
        <span>Crea Scena</span>
      </BaseButton>
    </div>

    <!-- Scene Editor Modal -->
    <SceneEditorModal
      :is-open="isEditorOpen"
      :scene="editingScene"
      @save="handleSaveScene"
      @close="isEditorOpen = false"
    />
  </div>
</template>

<style scoped>
.scenes-view {
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

.scenes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
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
  color: #c084fc;
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
