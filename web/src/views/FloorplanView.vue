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
  Edit2,
  X,
  Maximize2,
  Tv,
  Utensils,
  Bed,
  Bath,
  Briefcase,
  Home,
  LayoutGrid,
  Palette,
  Sparkles
} from '@lucide/vue'
import { useFloorplanStore } from '../stores/floorplan'
import { useEntitiesStore } from '../stores/entities'
import { useNotificationsStore } from '../stores/notifications'
import { audioSynthesizer } from '../utils/audioSynthesizer'
import type { Floor, FloorMarker } from '../api/types'

const { t } = useI18n()
const floorStore = useFloorplanStore()
const entitiesStore = useEntitiesStore()
const notifStore = useNotificationsStore()

// Modes: 'view' | 'edit_markers' | 'edit_rooms'
const activeMode = ref<'view' | 'edit_markers' | 'edit_rooms'>('view')

const showFloorModal = ref(false)
const showRoomModal = ref(false)

const floorForm = ref({
  name: '',
  level: 0,
  plan_svg: ''
})

// Room Definition
export interface FloorRoom {
  id: string
  name: string
  type: 'living' | 'kitchen' | 'bedroom' | 'bathroom' | 'office' | 'hallway' | 'balcony' | 'other'
  x: number // percent 0-100
  y: number // percent 0-100
  width: number // percent 10-100
  height: number // percent 10-100
  color: string
}

const rooms = ref<FloorRoom[]>([])
const selectedRoomId = ref<string | null>(null)

// Room Form
const roomForm = ref<{
  name: string
  type: FloorRoom['type']
  color: string
  width: number
  height: number
}>({
  name: 'Nuova Stanza',
  type: 'living',
  color: '#3b82f6',
  width: 35,
  height: 35
})

const roomColorPresets = [
  { name: 'Blu Oceano', value: '#3b82f6' },
  { name: 'Smeraldo', value: '#10b981' },
  { name: 'Ambra', value: '#f59e0b' },
  { name: 'Viola Notte', value: '#8b5cf6' },
  { name: 'Ciano Cielo', value: '#06b6d4' },
  { name: 'Rosa Corallo', value: '#f43f5e' }
]

const roomTypeOptions = [
  { key: 'living', label: 'Soggiorno / Salotto', icon: Tv },
  { key: 'kitchen', label: 'Cucina / Pranzo', icon: Utensils },
  { key: 'bedroom', label: 'Camera da Letto', icon: Bed },
  { key: 'bathroom', label: 'Bagno', icon: Bath },
  { key: 'office', label: 'Studio / Ufficio', icon: Briefcase },
  { key: 'hallway', label: 'Corridoio / Ingresso', icon: Home },
  { key: 'balcony', label: 'Terrazzo / Balcone', icon: LayoutGrid }
]

function getRoomIcon(type: string) {
  switch (type) {
    case 'kitchen': return Utensils
    case 'bedroom': return Bed
    case 'bathroom': return Bath
    case 'office': return Briefcase
    case 'hallway': return Home
    case 'balcony': return LayoutGrid
    default: return Tv
  }
}

// Drag state for marker placement
const draggedEntityKey = ref<string | null>(null)
const mapContainerRef = ref<HTMLDivElement | null>(null)

// Spatial Heatmap 2.5D State
const showHeatmap = ref(false)
const heatmapMetric = ref<'temperature' | 'humidity'>('temperature')
const heatmapCanvasRef = ref<HTMLCanvasElement | null>(null)

// Persistence of Rooms
function loadRoomsForFloor(floorId: string) {
  try {
    const raw = localStorage.getItem(`domus_floor_rooms_${floorId}`)
    if (raw) {
      rooms.value = JSON.parse(raw)
    } else {
      rooms.value = []
    }
  } catch {
    rooms.value = []
  }
}

function saveRoomsForFloor(floorId: string) {
  try {
    localStorage.setItem(`domus_floor_rooms_${floorId}`, JSON.stringify(rooms.value))
  } catch {}
}

onMounted(async () => {
  await Promise.all([
    floorStore.fetchFloors(),
    entitiesStore.fetchEntities()
  ])
  if (floorStore.activeFloor) {
    await floorStore.fetchMarkers(floorStore.activeFloor.id)
    loadRoomsForFloor(floorStore.activeFloor.id)
  }
})

watch(() => floorStore.activeFloorId, async (newId) => {
  if (newId) {
    await floorStore.fetchMarkers(newId)
    loadRoomsForFloor(newId)
    selectedRoomId.value = null
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
  if (activeMode.value !== 'view') return

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
  // If placing a device marker
  if (activeMode.value === 'edit_markers' && draggedEntityKey.value && mapContainerRef.value && floorStore.activeFloor) {
    const rect = mapContainerRef.value.getBoundingClientRect()
    const xPercent = Math.max(0, Math.min(100, ((e.clientX - rect.left) / rect.width) * 100))
    const yPercent = Math.max(0, Math.min(100, ((e.clientY - rect.top) / rect.height) * 100))

    saveMarkerPosition(draggedEntityKey.value, xPercent, yPercent)
    draggedEntityKey.value = null
    return
  }

  // If clicked outside any room in room edit mode, deselect
  if (activeMode.value === 'edit_rooms') {
    selectedRoomId.value = null
  }
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

// Room Management Methods
function openAddRoomModal() {
  roomForm.value = {
    name: 'Soggiorno',
    type: 'living',
    color: '#3b82f6',
    width: 38,
    height: 38
  }
  showRoomModal.value = true
}

function handleAddRoom() {
  if (!floorStore.activeFloor) return

  const newRoom: FloorRoom = {
    id: 'room_' + Date.now(),
    name: roomForm.value.name.trim() || 'Stanza',
    type: roomForm.value.type,
    color: roomForm.value.color,
    x: 10 + (rooms.value.length % 3) * 25,
    y: 10 + Math.floor(rooms.value.length / 3) * 25,
    width: roomForm.value.width,
    height: roomForm.value.height
  }

  rooms.value.push(newRoom)
  saveRoomsForFloor(floorStore.activeFloor.id)
  selectedRoomId.value = newRoom.id
  showRoomModal.value = false
  notifStore.success('Stanza Creata', `Stanza "${newRoom.name}" aggiunta alla planimetria`)
}

function generateStandardFloorplan() {
  if (!floorStore.activeFloor) return

  const defaultRooms: FloorRoom[] = [
    {
      id: 'room_living_' + Date.now(),
      name: 'Soggiorno Open Space',
      type: 'living',
      color: '#3b82f6',
      x: 3,
      y: 3,
      width: 46,
      height: 45
    },
    {
      id: 'room_kitchen_' + (Date.now() + 1),
      name: 'Cucina & Pranzo',
      type: 'kitchen',
      color: '#f59e0b',
      x: 51,
      y: 3,
      width: 46,
      height: 45
    },
    {
      id: 'room_bed_' + (Date.now() + 2),
      name: 'Camera da Letto',
      type: 'bedroom',
      color: '#8b5cf6',
      x: 3,
      y: 51,
      width: 32,
      height: 46
    },
    {
      id: 'room_bath_' + (Date.now() + 3),
      name: 'Bagno Padronale',
      type: 'bathroom',
      color: '#06b6d4',
      x: 37,
      y: 51,
      width: 24,
      height: 46
    },
    {
      id: 'room_office_' + (Date.now() + 4),
      name: 'Studio / Ufficio',
      type: 'office',
      color: '#10b981',
      x: 63,
      y: 51,
      width: 34,
      height: 46
    }
  ]

  rooms.value = defaultRooms
  saveRoomsForFloor(floorStore.activeFloor.id)
  selectedRoomId.value = defaultRooms[0].id
  notifStore.success('Planimetria Generata', '5 stanze predisposte con layout architettonico standard')
}

function deleteSelectedRoom() {
  if (!selectedRoomId.value || !floorStore.activeFloor) return
  rooms.value = rooms.value.filter(r => r.id !== selectedRoomId.value)
  saveRoomsForFloor(floorStore.activeFloor.id)
  selectedRoomId.value = null
  notifStore.success('Stanza Rimossa', 'Stanza eliminata dalla planimetria')
}

const selectedRoom = computed(() => {
  return rooms.value.find(r => r.id === selectedRoomId.value) || null
})

function selectRoom(r: FloorRoom, e: MouseEvent) {
  if (activeMode.value === 'edit_rooms') {
    e.stopPropagation()
    selectedRoomId.value = r.id
  }
}

// 2.5D Heatmap
function toggleHeatmap() {
  audioSynthesizer.playClick()
  showHeatmap.value = !showHeatmap.value
  if (showHeatmap.value) {
    setTimeout(renderSpatialHeatmap, 50)
  }
}

function renderSpatialHeatmap() {
  const canvas = heatmapCanvasRef.value
  if (!canvas || !mapContainerRef.value) return
  const rect = mapContainerRef.value.getBoundingClientRect()
  canvas.width = rect.width
  canvas.height = rect.height
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  ctx.clearRect(0, 0, canvas.width, canvas.height)

  const points: Array<{ x: number; y: number; val: number }> = []
  for (const m of floorStore.currentMarkers) {
    const ent = getEntity(m.entity_key)
    if (!ent) continue
    const k = m.entity_key.toLowerCase()
    const isTemp = k.includes('temp') || ent.domain === 'climate'
    const isHum = k.includes('humidity') || k.includes('umidita')

    if ((heatmapMetric.value === 'temperature' && isTemp) || (heatmapMetric.value === 'humidity' && isHum)) {
      let val = parseFloat(ent.state)
      if (isNaN(val) && ent.attributes) {
        val = parseFloat(ent.attributes.current_temperature || ent.attributes.temperature || '21')
      }
      if (!isNaN(val)) {
        points.push({
          x: (m.x_percent / 100) * canvas.width,
          y: (m.y_percent / 100) * canvas.height,
          val
        })
      }
    }
  }

  // Realistic fallback room points
  if (points.length === 0) {
    points.push(
      { x: canvas.width * 0.25, y: canvas.height * 0.3, val: heatmapMetric.value === 'temperature' ? 20.2 : 48 },
      { x: canvas.width * 0.75, y: canvas.height * 0.3, val: heatmapMetric.value === 'temperature' ? 22.4 : 54 },
      { x: canvas.width * 0.5, y: canvas.height * 0.7, val: heatmapMetric.value === 'temperature' ? 19.1 : 44 }
    )
  }

  for (const p of points) {
    const radius = Math.max(canvas.width, canvas.height) * 0.4
    const grad = ctx.createRadialGradient(p.x, p.y, 10, p.x, p.y, radius)

    let color = 'rgba(16, 185, 129, 0.45)'
    if (heatmapMetric.value === 'temperature') {
      if (p.val < 18.5) color = 'rgba(56, 189, 248, 0.45)'
      else if (p.val <= 21.5) color = 'rgba(16, 185, 129, 0.45)'
      else if (p.val <= 23.5) color = 'rgba(245, 158, 11, 0.45)'
      else color = 'rgba(239, 68, 68, 0.5)'
    } else {
      if (p.val < 40) color = 'rgba(245, 158, 11, 0.45)'
      else if (p.val <= 60) color = 'rgba(6, 182, 212, 0.45)'
      else color = 'rgba(59, 130, 246, 0.5)'
    }

    grad.addColorStop(0, color)
    grad.addColorStop(1, 'rgba(0, 0, 0, 0)')

    ctx.fillStyle = grad
    ctx.fillRect(0, 0, canvas.width, canvas.height)
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
    <!-- Header with Floor Tabs & Mode Controls -->
    <header class="floorplan-header">
      <div class="header-left">
        <div class="title-with-badge">
          <h1>{{ t('floorplan.title') }}</h1>
          <span class="floor-count-badge">{{ floorStore.sortedFloors.length }} Piani</span>
          <span v-if="rooms.length > 0" class="room-count-badge">{{ rooms.length }} Stanze</span>
        </div>
        <p class="subtitle">Disegna e posiziona le stanze della tua casa, organizza i dispositivi e visualizza la planimetria in tempo reale.</p>
      </div>

      <div class="header-actions">
        <!-- Mode Switcher Buttons -->
        <div class="mode-pill-group">
          <button
            class="mode-btn"
            :class="{ active: activeMode === 'view' }"
            @click="activeMode = 'view'"
          >
            Vista Mappa
          </button>
          <button
            class="mode-btn"
            :class="{ active: activeMode === 'edit_rooms' }"
            @click="activeMode = 'edit_rooms'"
          >
            <LayoutGrid :size="14" />
            <span>Planimetria Stanze</span>
          </button>
          <button
            class="mode-btn btn-mode"
            :class="{ active: activeMode === 'edit_markers' }"
            @click="activeMode = activeMode === 'edit_markers' ? 'view' : 'edit_markers'"
          >
            <Move :size="14" />
            <span>{{ activeMode === 'edit_markers' ? 'Termina Modifica' : 'Modifica Posizioni' }}</span>
          </button>
        </div>

        <button
          class="btn-heatmap-mode"
          :class="{ active: showHeatmap }"
          @click="toggleHeatmap"
          title="Mostra gradiente termico o umidità"
        >
          <Thermometer class="btn-icon" />
          <span>{{ showHeatmap ? 'Chiudi Heatmap' : 'Heatmap 2.5D' }}</span>
        </button>

        <button class="btn-primary" @click="showFloorModal = true">
          <Plus class="btn-icon" />
          <span>Nuovo Piano</span>
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

    <!-- Room Editing Toolbar (Visible when in edit_rooms mode) -->
    <div v-if="activeMode === 'edit_rooms'" class="room-editor-toolbar glass-panel">
      <div class="toolbar-left">
        <span class="toolbar-title">
          <LayoutGrid :size="18" class="text-primary" />
          <span>Editor Architettonico Stanze</span>
        </span>
        <button class="btn-action-tool" @click="openAddRoomModal">
          <Plus :size="16" />
          <span>Aggiungi Stanza</span>
        </button>
        <button class="btn-action-tool secondary" @click="generateStandardFloorplan">
          <Sparkles :size="16" />
          <span>Genera Planimetria Standard</span>
        </button>
      </div>

      <!-- Selected Room Customizer Controls -->
      <div v-if="selectedRoom" class="selected-room-controls">
        <span class="selected-label">Stanza: <strong>{{ selectedRoom.name }}</strong></span>

        <!-- X, Y, Width, Height quick sliders -->
        <div class="control-inputs">
          <label>X: <input v-model.number="selectedRoom.x" type="range" min="0" max="90" @input="saveRoomsForFloor(floorStore.activeFloor!.id)" /></label>
          <label>Y: <input v-model.number="selectedRoom.y" type="range" min="0" max="90" @input="saveRoomsForFloor(floorStore.activeFloor!.id)" /></label>
          <label>L: <input v-model.number="selectedRoom.width" type="range" min="15" max="80" @input="saveRoomsForFloor(floorStore.activeFloor!.id)" /></label>
          <label>A: <input v-model.number="selectedRoom.height" type="range" min="15" max="80" @input="saveRoomsForFloor(floorStore.activeFloor!.id)" /></label>
        </div>

        <!-- Color presets for room -->
        <div class="room-color-chips">
          <button
            v-for="c in roomColorPresets"
            :key="c.value"
            class="color-dot"
            :style="{ background: c.value }"
            :class="{ active: selectedRoom.color === c.value }"
            :title="c.name"
            @click="selectedRoom.color = c.value; saveRoomsForFloor(floorStore.activeFloor!.id)"
          />
        </div>

        <button class="btn-delete-room" title="Elimina questa stanza" @click="deleteSelectedRoom">
          <Trash2 :size="16" />
        </button>
      </div>
      <div v-else class="toolbar-hint">
        Fai clic su una stanza nella planimetria per modificarne posizione, dimensioni e colore.
      </div>
    </div>

    <!-- Main Workspace -->
    <div class="floorplan-workspace" :class="{ 'with-sidebar': activeMode === 'edit_markers' }">
      <!-- Interactive Floor Plan Map Area -->
      <div
        ref="mapContainerRef"
        class="floorplan-canvas"
        :class="{
          'editing-markers': activeMode === 'edit_markers',
          'editing-rooms': activeMode === 'edit_rooms',
          'placing-marker': !!draggedEntityKey
        }"
        @click="onMapClick"
      >
        <!-- Blueprint Grid Background -->
        <div class="blueprint-grid"></div>

        <!-- Architectural Rooms Layer -->
        <div class="rooms-layer">
          <div
            v-for="room in rooms"
            :key="room.id"
            class="architectural-room"
            :class="{
              'is-selected': selectedRoomId === room.id && activeMode === 'edit_rooms'
            }"
            :style="{
              left: `${room.x}%`,
              top: `${room.y}%`,
              width: `${room.width}%`,
              height: `${room.height}%`,
              borderColor: room.color,
              backgroundColor: `${room.color}15`,
              boxShadow: `inset 0 0 24px ${room.color}25`
            }"
            @click="selectRoom(room, $event)"
          >
            <!-- Room Header Badge -->
            <div class="room-header-badge" :style="{ background: `${room.color}35`, borderColor: room.color }">
              <component :is="getRoomIcon(room.type)" :size="14" :style="{ color: room.color }" />
              <span class="room-title">{{ room.name }}</span>
            </div>

            <!-- Architectural Wall Corner Markers -->
            <div class="wall-corner top-left" :style="{ borderColor: room.color }"></div>
            <div class="wall-corner top-right" :style="{ borderColor: room.color }"></div>
            <div class="wall-corner bottom-left" :style="{ borderColor: room.color }"></div>
            <div class="wall-corner bottom-right" :style="{ borderColor: room.color }"></div>
          </div>
        </div>

        <!-- Optional Custom SVG Layer -->
        <div v-if="floorStore.activeFloor?.plan_svg" class="svg-layer" v-html="floorStore.activeFloor.plan_svg" />

        <!-- 2.5D Spatial Heatmap Canvas Layer -->
        <canvas
          v-if="showHeatmap"
          ref="heatmapCanvasRef"
          class="spatial-heatmap-canvas"
        />

        <!-- Floating Heatmap Legend -->
        <div v-if="showHeatmap" class="heatmap-legend">
          <span class="legend-title">{{ heatmapMetric === 'temperature' ? 'Gradiente Termico (°C)' : 'Umidità (%RH)' }}</span>
          <div class="gradient-bar"></div>
          <div class="legend-labels">
            <span>{{ heatmapMetric === 'temperature' ? '< 18°C' : '< 35%' }}</span>
            <span>{{ heatmapMetric === 'temperature' ? '21°C' : '50%' }}</span>
            <span>{{ heatmapMetric === 'temperature' ? '> 24°C' : '> 70%' }}</span>
          </div>
          <button class="metric-switch-btn" @click.stop="heatmapMetric = heatmapMetric === 'temperature' ? 'humidity' : 'temperature'; renderSpatialHeatmap()">
            {{ heatmapMetric === 'temperature' ? 'Passa a Umidità' : 'Passa a Temperatura' }}
          </button>
        </div>

        <!-- Render Entity Markers on Map -->
        <div
          v-for="marker in floorStore.currentMarkers"
          :key="marker.id"
          class="map-marker"
          :class="{
            'is-on': getEntityState(marker.entity_key) === 'on',
            'is-edit-mode': activeMode === 'edit_markers'
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
            v-if="activeMode === 'edit_markers'"
            class="marker-del-btn"
            title="Rimuovi dispositivo dalla mappa"
            @click.stop="removeMarker(marker.id)"
          >
            <X class="del-icon" />
          </button>
        </div>
      </div>

      <!-- Right Drawer: Unplaced Devices in Markers Edit Mode -->
      <aside v-if="activeMode === 'edit_markers'" class="edit-sidebar">
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

    <!-- Create Room Modal -->
    <div v-if="showRoomModal" class="modal-backdrop" @click="showRoomModal = false">
      <div class="modal-card" @click.stop role="dialog" aria-modal="true" aria-labelledby="modal-room-title">
        <div class="modal-header">
          <h2 id="modal-room-title">Aggiungi Stanza alla Planimetria</h2>
          <button class="modal-close-btn" @click="showRoomModal = false">
            <X class="icon-close" />
          </button>
        </div>

        <form @submit.prevent="handleAddRoom" class="modal-body">
          <div class="form-group">
            <label for="room-name">Nome della Stanza *</label>
            <input
              id="room-name"
              v-model="roomForm.name"
              type="text"
              placeholder="es. Soggiorno, Cucina, Camera Padronale"
              required
            />
          </div>

          <div class="form-group">
            <label for="room-type">Tipologia Stanza</label>
            <select id="room-type" v-model="roomForm.type">
              <option v-for="t in roomTypeOptions" :key="t.key" :value="t.key">
                {{ t.label }}
              </option>
            </select>
          </div>

          <div class="form-group">
            <label>Colore / Identificativo Architettonico</label>
            <div class="color-picker-row">
              <button
                v-for="c in roomColorPresets"
                :key="c.value"
                type="button"
                class="color-dot-large"
                :style="{ background: c.value }"
                :class="{ active: roomForm.color === c.value }"
                :title="c.name"
                @click="roomForm.color = c.value"
              />
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label for="room-w">Larghezza Iniziale (%)</label>
              <input id="room-w" v-model.number="roomForm.width" type="number" min="15" max="80" />
            </div>
            <div class="form-group">
              <label for="room-h">Altezza Iniziale (%)</label>
              <input id="room-h" v-model.number="roomForm.height" type="number" min="15" max="80" />
            </div>
          </div>

          <div class="modal-footer">
            <button type="button" class="btn-cancel" @click="showRoomModal = false">Annulla</button>
            <button type="submit" class="btn-primary">Posiziona Stanza</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Create Floor Modal -->
    <div v-if="showFloorModal" class="modal-backdrop" @click="showFloorModal = false">
      <div class="modal-card" @click.stop role="dialog" aria-modal="true" aria-labelledby="modal-floor-title">
        <div class="modal-header">
          <h2 id="modal-floor-title">Nuovo Piano</h2>
          <button class="modal-close-btn" @click="showFloorModal = false">
            <X class="icon-close" />
          </button>
        </div>

        <form @submit.prevent="handleCreateFloor" class="modal-body">
          <div class="form-group">
            <label for="floor-name">Nome del Piano</label>
            <input
              id="floor-name"
              v-model="floorForm.name"
              type="text"
              placeholder="es. Piano Terra, Mansarda, Giardino"
              required
            />
          </div>

          <div class="form-group">
            <label for="floor-level">Livello Piano</label>
            <input
              id="floor-level"
              v-model.number="floorForm.level"
              type="number"
              placeholder="0 per Piano Terra, 1 per Primo Piano"
            />
          </div>

          <div class="form-group">
            <label for="floor-svg">Markup SVG Planimetria (Opzionale)</label>
            <textarea
              id="floor-svg"
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
  flex-wrap: wrap;
}

.title-with-badge h1 {
  font-size: 1.75rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-main, #f8fafc);
}

.floor-count-badge {
  font-size: 0.75rem;
  padding: 0.2rem 0.6rem;
  background: rgba(59, 130, 246, 0.2);
  color: var(--primary, #3b82f6);
  border-radius: 9999px;
  font-weight: 600;
}

.room-count-badge {
  font-size: 0.75rem;
  padding: 0.2rem 0.6rem;
  background: rgba(16, 185, 129, 0.2);
  color: #10b981;
  border-radius: 9999px;
  font-weight: 600;
}

.subtitle {
  color: var(--text-muted, #94a3b8);
  font-size: 0.875rem;
  margin-top: 0.25rem;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.mode-pill-group {
  display: flex;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 0.5rem;
  padding: 0.25rem;
  gap: 0.25rem;
}

.mode-btn {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.4rem 0.85rem;
  background: transparent;
  border: none;
  color: var(--text-muted, #94a3b8);
  font-size: 0.8rem;
  font-weight: 600;
  border-radius: 0.35rem;
  cursor: pointer;
  transition: all 0.2s;
}

.mode-btn:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.05);
}

.mode-btn.active {
  background: var(--primary, #3b82f6);
  color: #fff;
}

.btn-heatmap-mode {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #cbd5e1;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-heatmap-mode:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.btn-heatmap-mode.active {
  background: rgba(245, 158, 11, 0.2);
  border-color: #f59e0b;
  color: #fbbf24;
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: var(--primary, #3b82f6);
  border: none;
  color: #fff;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary:hover {
  background: var(--primary-hover, #2563eb);
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
  background: var(--bg-surface, rgba(15, 23, 42, 0.6));
  border: 1px solid var(--border-card, rgba(255, 255, 255, 0.12));
  border-radius: 0.5rem;
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
  transition: all 0.2s ease;
}

.floor-tab-btn:hover {
  color: var(--text-main, #f8fafc);
  background: var(--bg-surface-hover, rgba(255, 255, 255, 0.05));
}

.floor-tab-btn.active {
  background: var(--primary-light, rgba(59, 130, 246, 0.15));
  border-color: var(--primary, #3b82f6);
  color: var(--primary, #3b82f6);
  font-weight: 600;
}

.floor-level-tag {
  font-size: 0.7rem;
  padding: 0.1rem 0.4rem;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 4px;
}

/* Room Editor Toolbar */
.room-editor-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 18px;
  border-radius: 12px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  flex-wrap: wrap;
  gap: 12px;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.toolbar-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  font-size: 0.9rem;
  color: #f8fafc;
}

.btn-action-tool {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  border: 1px solid rgba(59, 130, 246, 0.4);
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-action-tool:hover {
  background: rgba(59, 130, 246, 0.3);
}

.btn-action-tool.secondary {
  border-color: rgba(245, 158, 11, 0.4);
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
}

.btn-action-tool.secondary:hover {
  background: rgba(245, 158, 11, 0.25);
}

.selected-room-controls {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.selected-label {
  font-size: 0.85rem;
  color: #cbd5e1;
}

.control-inputs {
  display: flex;
  gap: 10px;
}

.control-inputs label {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.75rem;
  color: #94a3b8;
}

.control-inputs input[type="range"] {
  width: 70px;
  accent-color: #3b82f6;
}

.room-color-chips {
  display: flex;
  gap: 6px;
}

.color-dot {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
}

.color-dot.active {
  border-color: #fff;
  transform: scale(1.15);
}

.btn-delete-room {
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
  padding: 6px;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-delete-room:hover {
  background: rgba(239, 68, 68, 0.3);
}

.toolbar-hint {
  font-size: 0.8rem;
  color: #94a3b8;
  font-style: italic;
}

/* Workspace Layout */
.floorplan-workspace {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1.25rem;
  min-height: 650px;
}

.floorplan-workspace.with-sidebar {
  grid-template-columns: 1fr 340px;
}

/* Floorplan Canvas */
.floorplan-canvas {
  position: relative;
  background: #070d1e;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 0.75rem;
  min-height: 650px;
  overflow: hidden;
  box-shadow: inset 0 2px 16px rgba(0, 0, 0, 0.6);
}

.floorplan-canvas.editing-rooms {
  cursor: default;
}

.floorplan-canvas.placing-marker {
  cursor: crosshair;
}

.blueprint-grid {
  position: absolute;
  inset: 0;
  background-image: 
    linear-gradient(rgba(59, 130, 246, 0.05) 1px, transparent 1px),
    linear-gradient(90deg, rgba(59, 130, 246, 0.05) 1px, transparent 1px);
  background-size: 32px 32px;
  pointer-events: none;
}

/* Rooms Layer */
.rooms-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.architectural-room {
  position: absolute;
  border: 2px solid;
  border-radius: 10px;
  box-sizing: border-box;
  pointer-events: auto;
  transition: border-color 0.2s, box-shadow 0.2s;
  cursor: pointer;
  overflow: hidden;
}

.architectural-room:hover {
  filter: brightness(1.1);
}

.architectural-room.is-selected {
  outline: 2px dashed #fff;
  outline-offset: 3px;
  z-index: 5;
}

.room-header-badge {
  position: absolute;
  top: 8px;
  left: 8px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 8px;
  border-radius: 6px;
  border: 1px solid;
  backdrop-filter: blur(4px);
  pointer-events: none;
}

.room-title {
  font-size: 0.75rem;
  font-weight: 700;
  color: #fff;
  white-space: nowrap;
}

.wall-corner {
  position: absolute;
  width: 8px;
  height: 8px;
  border: 2px solid;
  pointer-events: none;
}

.wall-corner.top-left { top: -2px; left: -2px; border-right: none; border-bottom: none; }
.wall-corner.top-right { top: -2px; right: -2px; border-left: none; border-bottom: none; }
.wall-corner.bottom-left { bottom: -2px; left: -2px; border-right: none; border-top: none; }
.wall-corner.bottom-right { bottom: -2px; right: -2px; border-left: none; border-top: none; }

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
  background: rgba(15, 23, 42, 0.95);
  border: 2px solid rgba(255, 255, 255, 0.3);
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
  background: var(--surface-card);
  border: 1px solid var(--border-color);
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
  color: var(--text-main);
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
  max-height: 520px;
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
  padding: 1rem;
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

.modal-header h2 {
  font-size: 1.25rem;
  margin: 0;
  color: #f8fafc;
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
.form-group select,
.form-group textarea {
  padding: 0.6rem;
  background: #1e293b;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 0.375rem;
  color: #fff;
  font-size: 0.875rem;
}

.color-picker-row {
  display: flex;
  gap: 10px;
}

.color-dot-large {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
}

.color-dot-large.active {
  border-color: #fff;
  transform: scale(1.15);
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
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

/* 2.5D Spatial Heatmap Styles */
.spatial-heatmap-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 6;
  border-radius: 1rem;
}

.heatmap-legend {
  position: absolute;
  bottom: 1.5rem;
  left: 1.5rem;
  z-index: 10;
  padding: 0.85rem 1.25rem;
  border-radius: 0.85rem;
  background: rgba(15, 23, 42, 0.85);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.12);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  min-width: 220px;
}

.legend-title {
  font-size: 0.8rem;
  font-weight: 700;
  color: #f8fafc;
}

.gradient-bar {
  height: 8px;
  width: 100%;
  border-radius: 4px;
  background: linear-gradient(90deg, #38bdf8 0%, #10b981 50%, #f59e0b 80%, #ef4444 100%);
}

.legend-labels {
  display: flex;
  justify-content: space-between;
  font-size: 0.7rem;
  color: #94a3b8;
  font-weight: 600;
}

.metric-switch-btn {
  margin-top: 0.25rem;
  padding: 0.35rem 0.65rem;
  font-size: 0.75rem;
  font-weight: 600;
  border-radius: 0.5rem;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #38bdf8;
  cursor: pointer;
  transition: all 0.2s ease;
}

.metric-switch-btn:hover {
  background: rgba(56, 189, 248, 0.2);
}
</style>
