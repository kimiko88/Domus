<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Layers,
  Plus,
  Move,
  Check,
  Trash2,
  Lightbulb,
  ToggleLeft,
  Thermometer,
  Zap,
  Eye,
  Edit2,
  X
} from '@lucide/vue'
import { useFloorplanStore } from '../stores/floorplan'
import { useEntitiesStore } from '../stores/entities'
import { useNotificationsStore } from '../stores/notifications'
import type { Floor, FloorMarker } from '../api/types'

const { t } = useI18n()
const floorStore = useFloorplanStore()
const entitiesStore = useEntitiesStore()
const notifStore = useNotificationsStore()

const isEditMode = ref(false)
const showFloorModal = ref(false)
const floorForm = ref({
  name: '',
  level: 0,
  plan_svg: ''
})

// Drag state for marker placement
const draggedEntityKey = ref<string | null>(null)
const mapContainerRef = ref<HTMLDivElement | null>(null)

onMounted(async () => {
  await Promise.all([
    floorStore.fetchFloors(),
    entitiesStore.fetchEntities()
  ])
  if (floorStore.activeFloor) {
    await floorStore.fetchMarkers(floorStore.activeFloor.id)
  }
})

watch(() => floorStore.activeFloorId, async (newId) => {
  if (newId) {
    await floorStore.fetchMarkers(newId)
  }
})

function selectFloor(floor: Floor) {
  floorStore.activeFloorId = floor.id
}

function getEntity(key: string) {
  return entitiesStore.entities[key]
}

function getEntityState(key: string): string {
  const ent = getEntity(key)
  return ent ? ent.state : 'offline'
}

function getEntityIcon(key: string) {
  if (key.startsWith('light.')) return Lightbulb
  if (key.startsWith('switch.')) return ToggleLeft
  if (key.startsWith('climate.') || key.includes('temperature')) return Thermometer
  return Zap
}

async function handleMarkerClick(marker: FloorMarker) {
  if (isEditMode.value) return

  const ent = getEntity(marker.entity_key)
  if (!ent) return

  if (ent.domain === 'light' || ent.domain === 'switch') {
    const nextState = ent.state === 'on' ? 'turn_off' : 'turn_on'
    try {
      await entitiesStore.callService(ent.domain, nextState, ent.id)
      notifStore.success('Comando Inviato', `${ent.name}: ${nextState === 'turn_on' ? 'Acceso' : 'Spento'}`)
    } catch {
      notifStore.error('Errore', 'Impossibile inviare il comando')
    }
  }
}

function onMapClick(e: MouseEvent) {
  if (!isEditMode.value || !draggedEntityKey.value || !mapContainerRef.value || !floorStore.activeFloor) return

  const rect = mapContainerRef.value.getBoundingClientRect()
  const xPercent = Math.max(0, Math.min(100, ((e.clientX - rect.left) / rect.width) * 100))
  const yPercent = Math.max(0, Math.min(100, ((e.clientY - rect.top) / rect.height) * 100))

  saveMarkerPosition(draggedEntityKey.value, xPercent, yPercent)
  draggedEntityKey.value = null
}

async function saveMarkerPosition(entityKey: string, x: number, y: number) {
  if (!floorStore.activeFloor) return
  try {
    await floorStore.saveMarker(floorStore.activeFloor.id, {
      entity_key: entityKey,
      x_percent: Math.round(x * 10) / 10,
      y_percent: Math.round(y * 10) / 10,
      show_badge: true
    })
    notifStore.success('Posizione Salvata', 'Dispositivo posizionato sulla mappa')
  } catch {
    notifStore.error('Errore', 'Impossibile salvare il marker')
  }
}

async function removeMarker(markerId: string) {
  if (!floorStore.activeFloor) return
  try {
    await floorStore.deleteMarker(floorStore.activeFloor.id, markerId)
    notifStore.success('Rimosso', 'Dispositivo rimosso dalla mappa')
  } catch {
    notifStore.error('Errore', 'Impossibile rimuovere il marker')
  }
}

async function handleCreateFloor() {
  if (!floorForm.value.name.trim()) return
  try {
    await floorStore.createFloor({
      name: floorForm.value.name.trim(),
      level: floorForm.value.level,
      plan_svg: floorForm.value.plan_svg.trim()
    })
    showFloorModal.value = false
    floorForm.value = { name: '', level: 0, plan_svg: '' }
    notifStore.success('Piano Creato', 'Nuovo piano aggiunto alla planimetria')
  } catch {
    notifStore.error('Errore', 'Impossibile creare il piano')
  }
}

const unplacedEntities = computed(() => {
  const currentKeySet = new Set(floorStore.currentMarkers.map((m) => m.entity_key))
  return Object.values(entitiesStore.entities).filter((e) => !currentKeySet.has(e.id))
})
</script>

<template>
  <div class="floorplan-view">
    <!-- Header with Floor Tabs & Controls -->
    <header class="floorplan-header">
      <div class="header-left">
        <div class="title-with-badge">
          <h1>{{ t('floorplan.title') }}</h1>
          <span class="floor-count-badge">{{ floorStore.sortedFloors.length }} Piani</span>
        </div>
        <p class="subtitle">{{ t('floorplan.subtitle') }}</p>
      </div>

      <div class="header-actions">
        <button
          class="btn-mode"
          :class="{ active: isEditMode }"
          @click="isEditMode = !isEditMode"
        >
          <component :is="isEditMode ? Check : Edit2" class="btn-icon" />
          {{ isEditMode ? 'Termina Modifica' : 'Modifica Posizioni' }}
        </button>

        <button class="btn-primary" @click="showFloorModal = true">
          <Plus class="btn-icon" />
          Aggiungi Piano
        </button>
      </div>
    </header>

    <!-- Floor Tabs Navigation -->
    <div class="floor-tabs-bar">
      <div class="floor-tabs">
        <button
          v-for="floor in floorStore.sortedFloors"
          :key="floor.id"
          class="floor-tab-btn"
          :class="{ active: floorStore.activeFloorId === floor.id }"
          @click="selectFloor(floor)"
        >
          <Layers class="tab-icon" />
          <span class="floor-name">{{ floor.name }}</span>
          <span class="floor-level-tag">Lvl {{ floor.level }}</span>
        </button>

        <button
          v-if="floorStore.sortedFloors.length === 0"
          class="floor-tab-btn empty"
          @click="showFloorModal = true"
        >
          <Plus class="tab-icon" />
          <span>Crea il primo piano</span>
        </button>
      </div>
    </div>

    <!-- Main Workspace -->
    <div class="floorplan-workspace" :class="{ 'with-sidebar': isEditMode }">
      <!-- Interactive Floor Plan Map Area -->
      <div
        ref="mapContainerRef"
        class="floorplan-canvas"
        :class="{ 'editing-canvas': isEditMode, 'placing-marker': !!draggedEntityKey }"
        @click="onMapClick"
      >
        <!-- SVG Layout or Blueprint Grid Background -->
        <div v-if="floorStore.activeFloor?.plan_svg" class="svg-layer" v-html="floorStore.activeFloor.plan_svg" />
        <div v-else class="blueprint-grid">
          <div class="blueprint-label">
            <Layers class="blueprint-icon" />
            <p>{{ floorStore.activeFloor?.name || 'Nessun Piano Selezionato' }}</p>
            <span>Fai clic sulla mappa per posizionare i dispositivi in modalità modifica</span>
          </div>
        </div>

        <!-- Render Entity Markers on Map -->
        <div
          v-for="marker in floorStore.currentMarkers"
          :key="marker.id"
          class="map-marker"
          :class="{
            'is-on': getEntityState(marker.entity_key) === 'on',
            'is-edit-mode': isEditMode
          }"
          :style="{
            left: `${marker.x_percent}%`,
            top: `${marker.y_percent}%`
          }"
          @click.stop="handleMarkerClick(marker)"
        >
          <div class="marker-bubble">
            <component :is="getEntityIcon(marker.entity_key)" class="marker-svg" />
          </div>

          <div v-if="marker.show_badge" class="marker-tooltip">
            <span class="tooltip-name">{{ getEntity(marker.entity_key)?.name || marker.entity_key }}</span>
            <span class="tooltip-state" :class="getEntityState(marker.entity_key)">
              {{ getEntityState(marker.entity_key) }}
            </span>
          </div>

          <!-- Quick remove button in edit mode -->
          <button
            v-if="isEditMode"
            class="marker-del-btn"
            title="Rimuovi dispositivo dalla mappa"
            @click.stop="removeMarker(marker.id)"
          >
            <X class="del-icon" />
          </button>
        </div>
      </div>

      <!-- Right Drawer: Unplaced Devices in Edit Mode -->
      <aside v-if="isEditMode" class="edit-sidebar">
        <div class="sidebar-header">
          <h3>Dispositivi Disponibili</h3>
          <span class="badge-count">{{ unplacedEntities.length }}</span>
        </div>
        <p class="sidebar-hint">Seleziona un dispositivo e fai clic sulla mappa per posizionarlo.</p>

        <div class="unplaced-list">
          <div
            v-for="ent in unplacedEntities"
            :key="ent.id"
            class="unplaced-item"
            :class="{ selected: draggedEntityKey === ent.id }"
            @click="draggedEntityKey = draggedEntityKey === ent.id ? null : ent.id"
          >
            <div class="item-icon">
              <component :is="getEntityIcon(ent.id)" class="icon-small" />
            </div>
            <div class="item-info">
              <span class="item-name">{{ ent.name }}</span>
              <span class="item-id">{{ ent.id }}</span>
            </div>
            <Move class="move-icon" />
          </div>

          <div v-if="unplacedEntities.length === 0" class="empty-unplaced">
            Tutti i dispositivi sono già posizionati su questa mappa!
          </div>
        </div>
      </aside>
    </div>

    <!-- Create Floor Modal -->
    <div v-if="showFloorModal" class="modal-backdrop" @click="showFloorModal = false">
      <div class="modal-card" @click.stop>
        <div class="modal-header">
          <h2>Nuovo Piano</h2>
          <button class="modal-close-btn" @click="showFloorModal = false">
            <X class="icon-close" />
          </button>
        </div>

        <form @submit.prevent="handleCreateFloor" class="modal-body">
          <div class="form-group">
            <label>Nome del Piano</label>
            <input
              v-model="floorForm.name"
              type="text"
              placeholder="es. Piano Terra, Mansarda, Giardino"
              required
            />
          </div>

          <div class="form-group">
            <label>Livello Piano</label>
            <input
              v-model.number="floorForm.level"
              type="number"
              placeholder="0 per Piano Terra, 1 per Primo Piano"
            />
          </div>

          <div class="form-group">
            <label>Markup SVG Planimetria (Opzionale)</label>
            <textarea
              v-model="floorForm.plan_svg"
              rows="4"
              placeholder="Incolla il codice vettoriale <svg>...</svg> della planimetria"
            ></textarea>
          </div>

          <div class="modal-footer">
            <button type="button" class="btn-cancel" @click="showFloorModal = false">Annulla</button>
            <button type="submit" class="btn-primary">Salva Piano</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.floorplan-view {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  padding: 1.5rem;
  max-width: 1600px;
  margin: 0 auto;
  color: #f1f5f9;
}

.floorplan-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
}

.title-with-badge {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.title-with-badge h1 {
  font-size: 1.75rem;
  font-weight: 700;
  margin: 0;
  color: #fff;
}

.floor-count-badge {
  font-size: 0.75rem;
  padding: 0.2rem 0.6rem;
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
  border-radius: 9999px;
  font-weight: 600;
}

.subtitle {
  color: #94a3b8;
  font-size: 0.875rem;
  margin-top: 0.25rem;
}

.header-actions {
  display: flex;
  gap: 0.75rem;
}

.btn-mode {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: rgba(30, 41, 59, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #cbd5e1;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-mode:hover {
  background: rgba(51, 65, 85, 0.8);
  color: #fff;
}

.btn-mode.active {
  background: #3b82f6;
  color: #fff;
  border-color: #3b82f6;
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: #3b82f6;
  border: none;
  color: #fff;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary:hover {
  background: #2563eb;
}

.btn-icon {
  width: 1rem;
  height: 1rem;
}

/* Floor Tabs */
.floor-tabs-bar {
  display: flex;
  overflow-x: auto;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  padding-bottom: 0.5rem;
}

.floor-tabs {
  display: flex;
  gap: 0.5rem;
}

.floor-tab-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.6rem 1.2rem;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 0.5rem;
  color: #94a3b8;
  cursor: pointer;
  transition: all 0.2s ease;
}

.floor-tab-btn:hover {
  color: #fff;
  background: rgba(30, 41, 59, 0.8);
}

.floor-tab-btn.active {
  background: rgba(59, 130, 246, 0.15);
  border-color: #3b82f6;
  color: #60a5fa;
  font-weight: 600;
}

.floor-level-tag {
  font-size: 0.7rem;
  padding: 0.1rem 0.4rem;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 4px;
}

/* Workspace Layout */
.floorplan-workspace {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1.25rem;
  min-height: 600px;
}

.floorplan-workspace.with-sidebar {
  grid-template-columns: 1fr 340px;
}

/* Floorplan Canvas */
.floorplan-canvas {
  position: relative;
  background: #0b1329;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 0.75rem;
  min-height: 600px;
  overflow: hidden;
  box-shadow: inset 0 2px 8px rgba(0, 0, 0, 0.5);
}

.floorplan-canvas.placing-marker {
  cursor: crosshair;
}

.blueprint-grid {
  position: absolute;
  inset: 0;
  background-image: 
    linear-gradient(rgba(255, 255, 255, 0.04) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.04) 1px, transparent 1px);
  background-size: 40px 40px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.blueprint-label {
  text-align: center;
  color: #64748b;
}

.blueprint-icon {
  width: 3rem;
  height: 3rem;
  margin-bottom: 0.5rem;
  color: #475569;
}

.blueprint-label p {
  font-size: 1.25rem;
  font-weight: 600;
  color: #cbd5e1;
  margin: 0;
}

.blueprint-label span {
  font-size: 0.8rem;
  color: #64748b;
}

/* Markers */
.map-marker {
  position: absolute;
  transform: translate(-50%, -50%);
  cursor: pointer;
  z-index: 10;
  transition: transform 0.2s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

.map-marker:hover {
  transform: translate(-50%, -50%) scale(1.15);
  z-index: 20;
}

.marker-bubble {
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 50%;
  background: rgba(30, 41, 59, 0.9);
  border: 2px solid rgba(255, 255, 255, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.5);
  transition: all 0.2s ease;
}

.map-marker.is-on .marker-bubble {
  background: #eab308;
  color: #000;
  border-color: #fef08a;
  box-shadow: 0 0 20px rgba(234, 179, 8, 0.6);
}

.marker-svg {
  width: 1.25rem;
  height: 1.25rem;
}

.marker-tooltip {
  position: absolute;
  bottom: 110%;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(15, 23, 42, 0.95);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 4px;
  padding: 0.25rem 0.5rem;
  white-space: nowrap;
  pointer-events: none;
  font-size: 0.75rem;
  display: flex;
  gap: 0.35rem;
  align-items: center;
}

.tooltip-state.on {
  color: #4ade80;
  font-weight: 600;
}

.marker-del-btn {
  position: absolute;
  top: -8px;
  right: -8px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #ef4444;
  border: none;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.del-icon {
  width: 10px;
  height: 10px;
}

/* Edit Sidebar */
.edit-sidebar {
  background: rgba(15, 23, 42, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 0.75rem;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.sidebar-header h3 {
  margin: 0;
  font-size: 1rem;
  color: #fff;
}

.badge-count {
  font-size: 0.75rem;
  padding: 0.15rem 0.5rem;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 9999px;
}

.sidebar-hint {
  font-size: 0.75rem;
  color: #94a3b8;
  margin: 0;
}

.unplaced-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-height: 500px;
  overflow-y: auto;
}

.unplaced-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.6rem 0.75rem;
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.2s;
}

.unplaced-item:hover {
  background: rgba(51, 65, 85, 0.8);
}

.unplaced-item.selected {
  border-color: #3b82f6;
  background: rgba(59, 130, 246, 0.2);
}

.item-info {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.item-name {
  font-size: 0.85rem;
  font-weight: 500;
  color: #f1f5f9;
}

.item-id {
  font-size: 0.7rem;
  color: #64748b;
}

.move-icon {
  width: 1rem;
  height: 1rem;
  color: #64748b;
}

/* Modals */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.modal-card {
  background: #0f172a;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 0.75rem;
  width: 100%;
  max-width: 500px;
  padding: 1.5rem;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.7);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.modal-close-btn {
  background: none;
  border: none;
  color: #94a3b8;
  cursor: pointer;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-bottom: 1rem;
}

.form-group label {
  font-size: 0.8rem;
  font-weight: 500;
  color: #cbd5e1;
}

.form-group input,
.form-group textarea {
  padding: 0.6rem;
  background: #1e293b;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 0.375rem;
  color: #fff;
  font-size: 0.875rem;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1.5rem;
}

.btn-cancel {
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #94a3b8;
  border-radius: 0.375rem;
  cursor: pointer;
}
</style>
