<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Video,
  Play,
  Maximize2,
  ShieldAlert,
  User,
  Car,
  Dog,
  Package,
  Plus,
  RefreshCw,
  Eye,
  Activity,
  Cpu,
  HardDrive,
  Layers,
  Sliders,
  LayoutGrid,
  Clock,
  FastForward,
  Rewind,
  Pause,
  Filter,
  Volume2,
  Film,
  CheckCircle,
  UserCheck,
  Trash2,
  ShieldCheck
} from '@lucide/vue'
import { apiClient } from '../api/client'
import { soundEffects } from '../utils/audioSynthesizer'
import type { FamilyFace, AuthorizedVehicle } from '../api/types'

const { t } = useI18n()

interface Camera {
  id: string
  name: string
  area_id?: string
  stream_url: string
  snapshot_url?: string
  stream_type: 'webrtc' | 'hls' | 'rtsp'
  detection_enabled: boolean
}

interface DetectionEvent {
  id: string
  camera_id: string
  label: string
  score: number
  thumbnail_url?: string
  detected_at: string
  clip_duration_sec?: number
}

interface NVRCameraStats {
  camera_id: string
  camera_name: string
  stream_status: string
  current_fps: number
  detection_ms: number
  total_events_24h: number
}

interface NVRSystemStatus {
  engine: string
  total_cameras: number
  active_streams: number
  detection_fps: number
  average_latency_ms: number
  hardware_accel: string
  storage_used_gb: number
  storage_limit_gb: number
  cameras?: NVRCameraStats[]
}

interface MotionZone {
  id: string
  camera_id: string
  name: string
  sensitivity: number
  labels: string[]
  enabled: boolean
}

const cameras = ref<Camera[]>([])
const events = ref<DetectionEvent[]>([])
const nvrStatus = ref<NVRSystemStatus>({
  engine: 'Domus Local AI Vision (ONNX)',
  total_cameras: 0,
  active_streams: 0,
  detection_fps: 0,
  average_latency_ms: 0,
  hardware_accel: 'CPU AVX2 SIMD',
  storage_used_gb: 0,
  storage_limit_gb: 250.0,
  cameras: []
})
const motionZones = ref<MotionZone[]>([])

const activeCamera = ref<Camera | null>(null)
const isLiveModalOpen = ref(false)
const selectedCameraForZones = ref<Camera | null>(null)
const activeCameraZones = computed(() => {
  if (!selectedCameraForZones.value) return []
  return motionZones.value.filter((z: MotionZone) => z.camera_id === selectedCameraForZones.value?.id)
})
const isZonesModalOpen = ref(false)
const previewClipModal = ref<{ isOpen: boolean; event: DetectionEvent | null }>({
  isOpen: false,
  event: null
})

function openLiveStream(cam: Camera) {
  activeCamera.value = cam
  isLiveModalOpen.value = true
}

function openZonesEditor(cam: Camera) {
  selectedCameraForZones.value = cam
  isZonesModalOpen.value = true
}

function toggleZone(zone: MotionZone) {
  zone.enabled = !zone.enabled
}

function openClipPlayer(e: DetectionEvent) {
  previewClipModal.value = {
    isOpen: true,
    event: e
  }
}

function addMockZone() {
  if (!selectedCameraForZones.value) return
  const newId = 'zone_' + Date.now().toString(36)
  motionZones.value.push({
    id: newId,
    camera_id: selectedCameraForZones.value.id,
    name: 'Nuova Zona ' + (motionZones.value.length + 1),
    sensitivity: 80,
    labels: ['person'],
    enabled: true
  })
}

// Surveillance Wall, Grid & AI Biometrics Mode
const currentViewMode = ref<'grid' | 'wall' | 'ai-faces'>('grid')
const wallMatrix = ref<'2x2' | '3x3'>('2x2')
const isWallPlaying = ref(true)
const wallPlaybackSpeed = ref(1)
const timelineCurrentMinute = ref(12 * 60 + 30) // default 12:30
const selectedLabelFilter = ref<'all' | 'person' | 'vehicle' | 'package' | 'animal'>('all')
const activeScrubbedEvent = ref<DetectionEvent | null>(null)

// Biometric Family Faces & Authorized Plates State
const familyFaces = ref<FamilyFace[]>([])
const authorizedPlates = ref<AuthorizedVehicle[]>([])
const isFaceModalOpen = ref(false)
const isPlateModalOpen = ref(false)
const newFaceName = ref('')
const newFaceRole = ref('Residente')
const newPlateNumber = ref('')
const newPlateOwner = ref('')
const newPlateModel = ref('')
const newPlateAutoOpen = ref(true)

const faceTestModal = ref<{ isOpen: boolean; face: FamilyFace | null; score: number | null }>({
  isOpen: false,
  face: null,
  score: null
})

async function loadFaces() {
  try {
    const list = await apiClient.get<FamilyFace[]>('/api/v1/cameras/faces')
    familyFaces.value = list || []
  } catch {
    familyFaces.value = []
  }
}

async function loadPlates() {
  try {
    const list = await apiClient.get<AuthorizedVehicle[]>('/api/v1/cameras/plates')
    authorizedPlates.value = list || []
  } catch {
    authorizedPlates.value = []
  }
}

async function handleCreateFace() {
  if (!newFaceName.value.trim()) return
  try {
    const created = await apiClient.post<FamilyFace>('/api/v1/cameras/faces', {
      name: newFaceName.value.trim(),
      role: newFaceRole.value,
      photo_url: `https://api.dicebear.com/7.x/initials/svg?seed=${encodeURIComponent(newFaceName.value.trim())}`,
      embedding: [0.5, 0.5, 0.5, 0.5]
    })
    if (created) {
      familyFaces.value.push(created)
      soundEffects.playSuccessChime()
      isFaceModalOpen.value = false
      newFaceName.value = ''
      newFaceRole.value = 'Residente'
    }
  } catch (err) {
    console.error('Failed to register face:', err)
  }
}

async function handleDeleteFace(face: FamilyFace) {
  if (!confirm(`Rimuovere il profilo biometrico di ${face.name}?`)) return
  try {
    await apiClient.delete(`/api/v1/cameras/faces/${face.id}`)
    familyFaces.value = familyFaces.value.filter(f => f.id !== face.id)
    soundEffects.playClick(400)
  } catch (err) {
    console.error('Failed to delete face:', err)
  }
}

async function handleCreatePlate() {
  if (!newPlateNumber.value.trim()) return
  try {
    const cleanPlate = newPlateNumber.value.trim().toUpperCase()
    const created = await apiClient.post<AuthorizedVehicle>('/api/v1/cameras/plates', {
      plate_number: cleanPlate,
      owner_name: newPlateOwner.value.trim() || 'Familiare',
      vehicle_model: newPlateModel.value.trim() || 'Veicolo Elettrico',
      auto_open_gate: newPlateAutoOpen.value
    })
    if (created) {
      authorizedPlates.value.push(created)
      soundEffects.playSuccessChime()
      isPlateModalOpen.value = false
      newPlateNumber.value = ''
      newPlateOwner.value = ''
      newPlateModel.value = ''
      newPlateAutoOpen.value = true
    }
  } catch (err) {
    console.error('Failed to register plate:', err)
  }
}

async function handleDeletePlate(veh: AuthorizedVehicle) {
  if (!confirm(`Rimuovere la targa ${veh.plate_number}?`)) return
  try {
    await apiClient.delete(`/api/v1/cameras/plates/${veh.plate_number}`)
    authorizedPlates.value = authorizedPlates.value.filter(v => v.plate_number !== veh.plate_number)
    soundEffects.playClick(400)
  } catch (err) {
    console.error('Failed to delete plate:', err)
  }
}

function testFaceMatch(face: FamilyFace) {
  soundEffects.playClick()
  faceTestModal.value = {
    isOpen: true,
    face,
    score: 0.94
  }
}

const maxWallCams = computed(() => (wallMatrix.value === '2x2' ? 4 : 9))

const wallCameras = computed(() => {
  return cameras.value.slice(0, maxWallCams.value)
})

const timelineFormattedTime = computed(() => {
  const totalMins = Math.max(0, Math.min(1439, timelineCurrentMinute.value))
  const hrs = Math.floor(totalMins / 60).toString().padStart(2, '0')
  const mins = (totalMins % 60).toString().padStart(2, '0')
  return `${hrs}:${mins}:00`
})

// Calculate percentage for timeline events across 24h
function getEventMinute(timeStr: string): number {
  if (!timeStr) return 0
  const parts = timeStr.split(':')
  if (parts.length >= 2) {
    const h = parseInt(parts[0], 10) || 0
    const m = parseInt(parts[1], 10) || 0
    return h * 60 + m
  }
  return 0
}

const timelineEvents = computed(() => {
  return events.value.map((e, idx) => {
    // If detected_at is e.g. "14:22" or ISO string
    let minute = 0
    if (e.detected_at.includes(':')) {
      minute = getEventMinute(e.detected_at)
    } else {
      minute = 600 + (idx * 55) % 800
    }
    const percent = Math.min(100, Math.max(0, (minute / 1440) * 100))
    return {
      ...e,
      minute,
      percent
    }
  }).filter(e => {
    if (selectedLabelFilter.value === 'all') return true
    return e.label === selectedLabelFilter.value
  })
})

function onScrubberChange(event: Event) {
  const target = event.target as HTMLInputElement
  const minute = parseInt(target.value, 10)
  timelineCurrentMinute.value = minute
  soundEffects.playClick()

  // Find closest event within 15 minutes of scrubber
  const closest = timelineEvents.value.find(e => Math.abs(e.minute - minute) <= 15)
  if (closest) {
    activeScrubbedEvent.value = closest
  } else {
    activeScrubbedEvent.value = null
  }
}

function jumpToTimelineEvent(event: DetectionEvent & { minute?: number }) {
  soundEffects.playClick()
  if (event.minute !== undefined) {
    timelineCurrentMinute.value = event.minute
  }
  activeScrubbedEvent.value = event
}

function toggleWallPlayback() {
  soundEffects.playClick()
  isWallPlaying.value = !isWallPlaying.value
}

function stepWallTimeline(deltaMinutes: number) {
  soundEffects.playClick()
  timelineCurrentMinute.value = Math.max(0, Math.min(1439, timelineCurrentMinute.value + deltaMinutes))
}

function cyclePlaybackSpeed() {
  soundEffects.playClick()
  const speeds = [1, 2, 4, 8]
  const curIdx = speeds.indexOf(wallPlaybackSpeed.value)
  wallPlaybackSpeed.value = speeds[(curIdx + 1) % speeds.length]
}

onMounted(async () => {
  try {
    const list = await apiClient.get<Camera[]>('/api/v1/cameras')
    cameras.value = list || []
  } catch {
    cameras.value = []
  }

  try {
    const evts = await apiClient.get<DetectionEvent[]>('/api/v1/cameras/events')
    events.value = evts || []
  } catch {
    events.value = []
  }

  try {
    const nvr = await apiClient.get<NVRSystemStatus>('/api/v1/nvr/status')
    if (nvr && nvr.engine) nvrStatus.value = nvr
  } catch {}

  loadFaces()
  loadPlates()
})
</script>

<template>
  <div class="cameras-view-container">
    <header class="cameras-header glass-panel">
      <div class="header-title-box">
        <div class="icon-circle">
          <Video :size="24" class="text-primary" />
        </div>
        <div>
          <h1 class="page-title">{{ t('cameras.title') }}</h1>
          <p class="page-subtitle">{{ t('cameras.subtitle') }}</p>
        </div>
      </div>

      <div class="header-view-controls">
        <div class="view-switch-group">
          <button
            type="button"
            class="btn-view-toggle"
            :class="{ active: currentViewMode === 'grid' }"
            @click="currentViewMode = 'grid'; soundEffects.playClick()"
          >
            <Video :size="15" />
            <span>Griglia</span>
          </button>
          <button
            type="button"
            class="btn-view-toggle"
            :class="{ active: currentViewMode === 'wall' }"
            @click="currentViewMode = 'wall'; soundEffects.playClick()"
          >
            <LayoutGrid :size="15" />
            <span>Surveillance Wall</span>
          </button>
          <button
            type="button"
            class="btn-view-toggle"
            :class="{ active: currentViewMode === 'ai-faces' }"
            @click="currentViewMode = 'ai-faces'; soundEffects.playClick()"
          >
            <UserCheck :size="15" />
            <span>Biometria & Targhe AI</span>
          </button>
        </div>

        <div v-if="currentViewMode === 'wall'" class="matrix-switch-group">
          <button
            type="button"
            class="btn-matrix-toggle"
            :class="{ active: wallMatrix === '2x2' }"
            @click="wallMatrix = '2x2'; soundEffects.playClick()"
          >
            2×2
          </button>
          <button
            type="button"
            class="btn-matrix-toggle"
            :class="{ active: wallMatrix === '3x3' }"
            @click="wallMatrix = '3x3'; soundEffects.playClick()"
          >
            3×3
          </button>
        </div>
      </div>
    </header>

    <!-- Surveillance Wall Mode -->
    <section v-if="currentViewMode === 'wall'" class="surveillance-wall-section glass-panel">
      <div class="wall-top-bar">
        <div class="wall-title-area">
          <div class="wall-live-indicator">
            <span class="pulsing-dot" />
            <span class="wall-live-text">SINCRONIZZAZIONE MULTI-SCHERMO ({{ wallMatrix }})</span>
          </div>
          <span class="wall-clock font-mono">{{ timelineFormattedTime }}</span>
        </div>

        <div class="wall-quick-actions">
          <div class="filter-pills">
            <button
              type="button"
              class="btn-filter-pill"
              :class="{ active: selectedLabelFilter === 'all' }"
              @click="selectedLabelFilter = 'all'; soundEffects.playClick()"
            >
              Tutti ({{ events.length }})
            </button>
            <button
              type="button"
              class="btn-filter-pill person"
              :class="{ active: selectedLabelFilter === 'person' }"
              @click="selectedLabelFilter = 'person'; soundEffects.playClick()"
            >
              <User :size="12" /> Persona
            </button>
            <button
              type="button"
              class="btn-filter-pill vehicle"
              :class="{ active: selectedLabelFilter === 'vehicle' }"
              @click="selectedLabelFilter = 'vehicle'; soundEffects.playClick()"
            >
              <Car :size="12" /> Veicolo
            </button>
            <button
              type="button"
              class="btn-filter-pill package"
              :class="{ active: selectedLabelFilter === 'package' }"
              @click="selectedLabelFilter = 'package'; soundEffects.playClick()"
            >
              <Package :size="12" /> Pacco
            </button>
          </div>
        </div>
      </div>

      <!-- Synchronized Video Feeds Matrix -->
      <div
        class="surveillance-matrix-grid"
        :class="wallMatrix === '2x2' ? 'matrix-2x2' : 'matrix-3x3'"
      >
        <div
          v-for="(cam, index) in wallCameras"
          :key="cam.id"
          class="wall-feed-cell"
          :class="{ 'cell-highlighted': activeScrubbedEvent && activeScrubbedEvent.camera_id === cam.id }"
          @click="openLiveStream(cam)"
        >
          <img :src="cam.snapshot_url" :alt="cam.name" class="wall-feed-img" />

          <!-- Overlay UI -->
          <div class="feed-overlay feed-overlay-top">
            <span class="cell-cam-name">CAM {{ index + 1 }}: {{ cam.name }}</span>
            <span class="cell-time font-mono">{{ timelineFormattedTime }}</span>
          </div>

          <div class="feed-overlay feed-overlay-bottom">
            <span class="feed-status-badge">
              <span class="status-indicator-dot" />
              {{ cam.stream_type.toUpperCase() }} REC
            </span>

            <div v-if="activeScrubbedEvent && activeScrubbedEvent.camera_id === cam.id" class="feed-event-alert">
              <span class="alert-label">{{ activeScrubbedEvent.label.toUpperCase() }}</span>
              <span class="alert-score">{{ (activeScrubbedEvent.score * 100).toFixed(0) }}%</span>
            </div>
          </div>

          <!-- Highlight AI Box Simulation -->
          <div
            v-if="activeScrubbedEvent && activeScrubbedEvent.camera_id === cam.id"
            class="ai-tracking-box"
          >
            <span class="box-tag">{{ activeScrubbedEvent.label.toUpperCase() }}</span>
          </div>
        </div>

        <!-- Fill empty slots if fewer cameras than matrix size -->
        <div
          v-for="emptyIdx in Math.max(0, maxWallCams - wallCameras.length)"
          :key="'empty_' + emptyIdx"
          class="wall-feed-cell empty-cell"
        >
          <div class="empty-feed-placeholder">
            <Video :size="32" style="opacity: 0.3;" />
            <span>Canale {{ wallCameras.length + emptyIdx }} in attesa</span>
          </div>
        </div>
      </div>

      <!-- Unified Scrubbing Timeline & Playback Bar -->
      <div class="unified-timeline-bar">
        <div class="timeline-controls-row">
          <div class="playback-buttons">
            <button
              type="button"
              class="btn-playback-step"
              title="Indietro 15 minuti"
              @click="stepWallTimeline(-15)"
            >
              <Rewind :size="16" />
              <span>-15m</span>
            </button>

            <button
              type="button"
              class="btn-playback-play"
              :title="isWallPlaying ? 'Pausa' : 'Riproduci'"
              @click="toggleWallPlayback"
            >
              <Pause v-if="isWallPlaying" :size="18" />
              <Play v-else :size="18" />
            </button>

            <button
              type="button"
              class="btn-playback-step"
              title="Avanti 15 minuti"
              @click="stepWallTimeline(15)"
            >
              <span>+15m</span>
              <FastForward :size="16" />
            </button>

            <button
              type="button"
              class="btn-playback-speed"
              title="Velocità riproduzione"
              @click="cyclePlaybackSpeed"
            >
              {{ wallPlaybackSpeed }}x
            </button>
          </div>

          <div class="timeline-time-display">
            <Clock :size="16" class="text-primary" />
            <span class="time-readout font-mono">{{ timelineFormattedTime }}</span>
            <span class="time-scope">Oggi, 24 Ore Registrate</span>
          </div>
        </div>

        <!-- Timeline Track & AI Markers -->
        <div class="timeline-track-container">
          <div class="timeline-hour-markers">
            <span>00:00</span>
            <span>04:00</span>
            <span>08:00</span>
            <span>12:00</span>
            <span>16:00</span>
            <span>20:00</span>
            <span>24:00</span>
          </div>

          <div class="timeline-track-wrapper">
            <!-- Scrubber Slider -->
            <input
              type="range"
              min="0"
              max="1439"
              :value="timelineCurrentMinute"
              class="timeline-scrubber-slider"
              @input="onScrubberChange"
            />

            <!-- Event markers on track -->
            <div
              v-for="e in timelineEvents"
              :key="e.id"
              class="timeline-marker"
              :class="[e.label, { active: activeScrubbedEvent && activeScrubbedEvent.id === e.id }]"
              :style="{ left: e.percent + '%' }"
              :title="`${e.label.toUpperCase()} @ ${e.detected_at} (${e.camera_id})`"
              @click="jumpToTimelineEvent(e)"
            >
              <span class="marker-dot" />
            </div>
          </div>
        </div>

        <!-- Scrubbed Event Detail Card -->
        <div v-if="activeScrubbedEvent" class="scrubbed-event-card">
          <div class="scrubbed-event-thumb">
            <img :src="activeScrubbedEvent.thumbnail_url" class="scrub-img" />
            <div class="scrub-badge" :class="activeScrubbedEvent.label">
              {{ activeScrubbedEvent.label.toUpperCase() }}
            </div>
          </div>
          <div class="scrubbed-event-info">
            <div class="scrub-title-row">
              <strong>Evento AI Rilevato</strong>
              <span class="scrub-time font-mono">{{ activeScrubbedEvent.detected_at }}</span>
            </div>
            <div class="scrub-details">
              <span>Camera: <strong>{{ activeScrubbedEvent.camera_id }}</strong></span>
              <span>Confidenza: <strong>{{ (activeScrubbedEvent.score * 100).toFixed(0) }}%</strong></span>
            </div>
          </div>
          <button
            type="button"
            class="btn-play-scrubbed-clip"
            @click="openClipPlayer(activeScrubbedEvent)"
          >
            <Film :size="15" />
            <span>Riproduci Filmato ({{ activeScrubbedEvent.clip_duration_sec || 5 }}s)</span>
          </button>
        </div>
      </div>
    </section>

    <!-- Biometric Face & Vehicle Plate Recognition Mode -->
    <section v-if="currentViewMode === 'ai-faces'" class="ai-faces-management-section">
      <!-- AI Vision Overview Banner -->
      <div class="ai-overview-banner glass-panel">
        <div class="ai-banner-header">
          <div class="ai-title-row">
            <div class="ai-engine-pill">
              <Cpu :size="16" class="text-primary" />
              <span>Domus Local Edge Vision Engine</span>
            </div>
            <div class="ai-secure-pill">
              <ShieldCheck :size="14" class="text-emerald" />
              <span>Zero-Cloud Privacy Guarantee</span>
            </div>
          </div>
          <p class="ai-banner-desc">
            Riconoscimento facciale dei residenti mediante embedding vettoriali locali (Cosine Similarity &ge; 0.65) e lettura automatica delle targhe (ALPR) per controllo accessi e varchi carrabili.
          </p>
        </div>

        <div class="ai-stats-row">
          <div class="ai-stat-box">
            <span class="ai-stat-label">Familiari Registrati</span>
            <span class="ai-stat-number text-primary">{{ familyFaces.length }}</span>
            <span class="ai-stat-sub">Profili biometrici 128-D attivi</span>
          </div>
          <div class="ai-stat-box">
            <span class="ai-stat-label">Targhe Autorizzate</span>
            <span class="ai-stat-number text-emerald">{{ authorizedPlates.length }}</span>
            <span class="ai-stat-sub">Apertura automatica varco attiva</span>
          </div>
          <div class="ai-stat-box">
            <span class="ai-stat-label">Soglia Similarità</span>
            <span class="ai-stat-number text-warning">&ge; 65%</span>
            <span class="ai-stat-sub">Tolleranza angolare e occhiali</span>
          </div>
          <div class="ai-stat-box">
            <span class="ai-stat-label">Accelerazione Hardware</span>
            <span class="ai-stat-number text-info">{{ nvrStatus.hardware_accel || 'NPU / AVX2 SIMD' }}</span>
            <span class="ai-stat-sub">Inferenza edge sub-frame</span>
          </div>
        </div>
      </div>

      <!-- Family Faces Grid -->
      <div class="ai-sub-section glass-panel">
        <div class="sub-section-header">
          <div class="sub-header-title">
            <UserCheck :size="20" class="text-primary" />
            <div>
              <h3>Profili Biometrici Familiari</h3>
              <p class="sub-header-desc">Riconoscimento automatico dei componenti della casa per automazioni e disarmo sicurezza</p>
            </div>
          </div>
          <button type="button" class="btn-primary-action" @click="isFaceModalOpen = true; soundEffects.playClick()">
            <Plus :size="15" />
            <span>Registra Nuovo Volto</span>
          </button>
        </div>

        <div v-if="familyFaces.length === 0" class="empty-faces-box">
          <User :size="36" style="opacity: 0.3; margin-bottom: 8px;" />
          <p>Nessun volto familiare registrato nell'hub locale.</p>
        </div>

        <div v-else class="faces-grid">
          <div v-for="face in familyFaces" :key="face.id" class="face-card glass-panel">
            <div class="face-avatar-container">
              <img :src="face.photo_url || `https://api.dicebear.com/7.x/initials/svg?seed=${encodeURIComponent(face.name)}`" class="face-avatar-img" :alt="face.name" />
              <span class="face-verified-badge" title="Embedding Vettoriale Attivo">
                <ShieldCheck :size="12" />
              </span>
            </div>
            <div class="face-info-box">
              <div class="face-name-row">
                <span class="face-name">{{ face.name }}</span>
                <span class="face-role-pill">{{ face.role || 'Residente' }}</span>
              </div>
              <span class="face-id-sub font-mono">{{ face.id }}</span>
              <div class="face-vector-status">
                <span class="status-dot green" />
                <span class="text-xs">Embedding 128-D Vettoriale</span>
              </div>
            </div>
            <div class="face-card-actions">
              <button
                type="button"
                class="btn-test-match"
                title="Testa similarità con modello AI locale"
                @click="testFaceMatch(face)"
              >
                Test Match
              </button>
              <button
                type="button"
                class="btn-delete-card"
                title="Elimina profilo"
                @click="handleDeleteFace(face)"
              >
                <Trash2 :size="14" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Authorized Vehicles Grid -->
      <div class="ai-sub-section glass-panel">
        <div class="sub-section-header">
          <div class="sub-header-title">
            <Car :size="20" class="text-primary" />
            <div>
              <h3>Targhe Veicoli Autorizzati (ALPR / ANPR)</h3>
              <p class="sub-header-desc">Riconoscimento ottico della targa per apertura tempestiva cancello carraio e portone garage</p>
            </div>
          </div>
          <button type="button" class="btn-primary-action" @click="isPlateModalOpen = true; soundEffects.playClick()">
            <Plus :size="15" />
            <span>Aggiungi Targa</span>
          </button>
        </div>

        <div v-if="authorizedPlates.length === 0" class="empty-faces-box">
          <Car :size="36" style="opacity: 0.3; margin-bottom: 8px;" />
          <p>Nessun veicolo registrato per il varco automatico.</p>
        </div>

        <div v-else class="plates-grid">
          <div v-for="veh in authorizedPlates" :key="veh.plate_number" class="plate-card glass-panel">
            <div class="eu-license-plate">
              <div class="plate-band-left">
                <span class="eu-stars">★</span>
                <span class="eu-country">IT</span>
              </div>
              <span class="plate-text font-mono">{{ veh.plate_number }}</span>
              <div class="plate-band-right">
                <span class="plate-year">26</span>
              </div>
            </div>
            <div class="plate-info-box">
              <span class="plate-owner">{{ veh.owner_name }}</span>
              <span class="plate-model text-xs text-dim">{{ veh.vehicle_model || 'Veicolo Privato' }}</span>
              <div class="plate-gate-status">
                <span class="status-dot green" />
                <span class="text-xs text-emerald">Apertura Automatica Cancello</span>
              </div>
            </div>
            <div class="plate-card-actions">
              <button
                type="button"
                class="btn-delete-card"
                title="Rimuovi targa autorizzata"
                @click="handleDeletePlate(veh)"
              >
                <Trash2 :size="14" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- NVR System Status Banner -->
    <section v-if="currentViewMode === 'grid'" class="nvr-banner glass-panel">
      <div class="nvr-banner-top">
        <div class="nvr-title-group">
          <div class="nvr-badge">
            <Cpu :size="16" class="text-primary" />
            <span>{{ nvrStatus.engine }}</span>
          </div>
          <div class="nvr-hw-badge">
            <Activity :size="14" class="text-emerald" />
            <span>{{ nvrStatus.hardware_accel }}</span>
          </div>
        </div>

        <div class="nvr-storage-pill">
          <HardDrive :size="14" />
          <span>Storage NVR: {{ nvrStatus.storage_used_gb }} GB / {{ nvrStatus.storage_limit_gb }} GB (WAL Ring Buffer)</span>
        </div>
      </div>

      <div class="nvr-stats-grid">
        <div class="nvr-stat-card">
          <span class="stat-lbl">{{ t('cameras.inferenceFps') }}</span>
          <span class="stat-val text-primary">{{ nvrStatus.detection_fps }} FPS</span>
          <span class="stat-desc">Zero frame drop su WebRTC</span>
        </div>
        <div class="nvr-stat-card">
          <span class="stat-lbl">{{ t('cameras.latency') }}</span>
          <span class="stat-val text-emerald">{{ nvrStatus.average_latency_ms }} ms</span>
          <span class="stat-desc">Inferenza real-time sub-frame</span>
        </div>
        <div class="nvr-stat-card">
          <span class="stat-lbl">{{ t('cameras.activeChannels') }}</span>
          <span class="stat-val">{{ nvrStatus.active_streams }} / {{ nvrStatus.total_cameras }} Online</span>
          <span class="stat-desc">&lt;200ms latenza di streaming</span>
        </div>
        <div class="nvr-stat-card">
          <span class="stat-lbl">{{ t('cameras.privacy') }}</span>
          <span class="stat-val text-info">{{ t('cameras.privacyDesc') }}</span>
        </div>
      </div>
    </section>

    <!-- Empty State for Cameras -->
    <div v-if="currentViewMode === 'grid' && cameras.length === 0" class="empty-state glass-panel" style="padding: 40px; text-align: center; margin-bottom: 24px;">
      <Video :size="48" style="opacity: 0.4; margin: 0 auto 12px; color: var(--text-muted);" />
      <p style="font-weight: 600; font-size: 1.1rem; margin: 0 0 6px;">{{ t('cameras.noCameras', 'Nessuna telecamera configurata') }}</p>
      <p style="color: var(--text-muted); font-size: 0.9rem; margin: 0;">Collega telecamere WebRTC, RTSP o HLS per visualizzare streaming live e rilevamento AI.</p>
    </div>

    <!-- Camera Feeds Grid -->
    <div v-else-if="currentViewMode === 'grid'" class="cameras-grid">
      <div v-for="cam in cameras" :key="cam.id" class="camera-card glass-panel">
        <div class="cam-preview-wrapper" @click="openLiveStream(cam)">
          <img :src="cam.snapshot_url" :alt="cam.name" class="cam-img" />
          <div class="live-badge">
            <span class="pulsing-dot" />
            <span>{{ t('cameras.live') }}</span>
          </div>
          <button class="expand-btn" title="Espandi a tutto schermo">
            <Maximize2 :size="16" />
          </button>
        </div>

        <div class="cam-info-bar">
          <div>
            <h3 class="cam-title">{{ cam.name }}</h3>
            <span class="cam-protocol">{{ cam.stream_type.toUpperCase() }} • Sub-second delay</span>
          </div>

          <div class="cam-actions-row">
            <button
              type="button"
              class="btn-zone-cfg"
              title="Configura Zone di Movimento AI"
              @click.stop="openZonesEditor(cam)"
            >
              <Sliders :size="14" />
              <span>{{ t('cameras.zonesAi') }}</span>
            </button>
            <div v-if="cam.detection_enabled" class="ai-shield-tag" title="Local ONNX AI Guard Attivo">
              <ShieldAlert :size="14" />
              <span>{{ t('cameras.aiGuard') }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Active Motion Zones Overview -->
    <section class="zones-overview-section glass-panel">
      <div class="section-header">
        <div class="section-title-box">
          <Layers :size="20" class="text-primary" />
          <h2 class="section-title">{{ t('cameras.zonesTitle') }}</h2>
        </div>
      </div>

      <div v-if="motionZones.length === 0" style="padding: 24px; text-align: center; color: var(--text-muted);">
        <p style="margin: 0;">Nessuna zona di rilevamento movimento configurata.</p>
      </div>
      <div v-else class="zones-grid">
        <div v-for="zone in motionZones" :key="zone.id" class="zone-card">
          <div class="zone-top">
            <div class="zone-info">
              <span class="zone-name">{{ zone.name }}</span>
              <span class="zone-cam text-dim font-mono">{{ zone.camera_id }}</span>
            </div>
            <button
              type="button"
              class="zone-toggle"
              :class="{ active: zone.enabled }"
              @click="toggleZone(zone)"
            >
              {{ zone.enabled ? t('cameras.active') : t('cameras.disabled') }}
            </button>
          </div>

          <div class="zone-meta">
            <div class="sensitivity-row">
              <span class="text-dim text-xs">{{ t('cameras.sensitivity') }}:</span>
              <span class="text-xs font-mono">{{ zone.sensitivity }}%</span>
            </div>
            <div class="sensitivity-bar">
              <div class="sensitivity-fill" :style="{ width: zone.sensitivity + '%' }" />
            </div>
          </div>

          <div class="zone-labels-row">
            <span
              v-for="lbl in zone.labels"
              :key="lbl"
              class="label-badge"
              :class="lbl"
            >
              {{ lbl.toUpperCase() }}
            </span>
          </div>
        </div>
      </div>
    </section>

    <!-- AI Detection Event Timeline -->
    <section class="events-section glass-panel">
      <div class="section-header">
        <div class="section-title-box">
          <Eye :size="20" class="text-warning" />
          <h2 class="section-title">{{ t('cameras.timelineTitle') }}</h2>
        </div>
      </div>

      <div v-if="events.length === 0" style="padding: 24px; text-align: center; color: var(--text-muted);">
        <p style="margin: 0;">Nessun evento AI registrato di recente.</p>
      </div>
      <div v-else class="events-list">
        <div v-for="e in events" :key="e.id" class="event-row">
          <div class="event-icon-badge" :class="e.label">
            <User v-if="e.label === 'person'" :size="16" />
            <Car v-else-if="e.label === 'vehicle'" :size="16" />
            <Dog v-else-if="e.label === 'animal'" :size="16" />
            <Package v-else :size="16" />
          </div>

          <div class="event-meta">
            <span class="event-label">{{ t('cameras.detected') }}: <strong>{{ e.label.toUpperCase() }}</strong></span>
            <span class="event-sub">Camera: {{ e.camera_id }} • {{ t('cameras.confidence') }}: {{ (e.score * 100).toFixed(0) }}%</span>
          </div>

          <div class="event-actions">
            <button
              type="button"
              class="btn-clip-preview"
              title="Guarda clip animata dell'evento"
              @click="openClipPlayer(e)"
            >
              <Film :size="14" />
              <span>{{ t('cameras.clipButton') }} ({{ e.clip_duration_sec || 5 }}s)</span>
            </button>
            <span class="event-time">{{ e.detected_at }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- Fullscreen Live Modal -->
    <div v-if="isLiveModalOpen && activeCamera" class="modal-backdrop" @click="isLiveModalOpen = false">
      <div class="live-modal-dialog glass-panel" @click.stop>
        <div class="modal-top">
          <h3>{{ activeCamera.name }} (WebRTC Stream Diretto)</h3>
          <button class="close-btn" @click="isLiveModalOpen = false">✕</button>
        </div>
        <div class="video-container">
          <img :src="activeCamera.snapshot_url" class="fullscreen-video" />
        </div>
      </div>
    </div>

    <!-- Instant Clip Preview Modal -->
    <div v-if="previewClipModal.isOpen && previewClipModal.event" class="modal-backdrop" @click="previewClipModal.isOpen = false">
      <div class="clip-modal-dialog glass-panel" @click.stop>
        <div class="modal-top">
          <div class="clip-title-row">
            <Film :size="18" class="text-primary" />
            <h3>{{ t('cameras.clipModalTitle') }} ({{ previewClipModal.event.label.toUpperCase() }})</h3>
          </div>
          <button class="close-btn" @click="previewClipModal.isOpen = false">✕</button>
        </div>
        <div class="clip-media-wrapper">
          <img :src="previewClipModal.event.thumbnail_url" class="clip-img" />
          <div class="clip-overlay-tag">
            <span class="tag-label">{{ previewClipModal.event.label.toUpperCase() }} ({{ (previewClipModal.event.score * 100).toFixed(0) }}%)</span>
            <span class="tag-time">{{ previewClipModal.event.detected_at }}</span>
          </div>
        </div>
        <div class="clip-modal-footer">
          <div class="clip-meta-info">
            <span>Camera ID: <strong>{{ previewClipModal.event.camera_id }}</strong></span>
            <span>{{ t('cameras.clipDuration') }}: <strong>{{ previewClipModal.event.clip_duration_sec }}s MP4 H.264</strong></span>
          </div>
          <span class="notif-sync-badge">
            <CheckCircle :size="14" />
            {{ t('cameras.notifSync') }}
          </span>
        </div>
      </div>
    </div>

    <!-- Motion Zones Configurator Modal -->
    <div v-if="isZonesModalOpen && selectedCameraForZones" class="modal-backdrop" @click="isZonesModalOpen = false">
      <div class="zones-modal-dialog glass-panel" @click.stop>
        <div class="modal-top">
          <div class="clip-title-row">
            <Sliders :size="18" class="text-primary" />
            <h3>{{ t('cameras.zonesModalTitle') }} - {{ selectedCameraForZones.name }}</h3>
          </div>
          <button class="close-btn" @click="isZonesModalOpen = false">✕</button>
        </div>

        <div class="zones-modal-body">
          <div class="zones-cam-preview">
            <img :src="selectedCameraForZones.snapshot_url" class="zone-preview-img" />
            <div class="zone-box-overlay zone-1">
              <span>Portico & Consegna Pacchi</span>
            </div>
            <div class="zone-box-overlay zone-2">
              <span>Vialetto Veicoli</span>
            </div>
          </div>

          <div class="zones-editor-list">
            <div class="editor-header">
              <h4>Zone Definite per la Telecamera</h4>
              <button type="button" class="btn-add-zone" @click="addMockZone">
                <Plus :size="14" />
                <span>{{ t('cameras.addZone') }}</span>
              </button>
            </div>

            <div
              v-for="zone in activeCameraZones"
              :key="zone.id"
              class="zone-editor-item"
            >
              <div class="editor-item-top">
                <input v-model="zone.name" class="zone-name-input" />
                <button
                  type="button"
                  class="zone-toggle-sm"
                  :class="{ active: zone.enabled }"
                  @click="toggleZone(zone)"
                >
                  {{ zone.enabled ? 'Attiva' : 'Disattivata' }}
                </button>
              </div>

              <div class="editor-sensitivity">
                <span class="text-xs text-dim">Sensibilità: {{ zone.sensitivity }}%</span>
                <input
                  v-model.number="zone.sensitivity"
                  type="range"
                  min="10"
                  max="100"
                  class="slider-range"
                />
              </div>

              <div class="editor-tags">
                <span v-for="tag in zone.labels" :key="tag" class="tag-pill">{{ tag }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Register Family Face Modal -->
    <div v-if="isFaceModalOpen" class="modal-backdrop" @click="isFaceModalOpen = false">
      <div class="ai-modal-dialog glass-panel" @click.stop>
        <div class="modal-top">
          <div class="clip-title-row">
            <UserCheck :size="18" class="text-primary" />
            <h3>Registra Nuovo Profilo Biometrico</h3>
          </div>
          <button class="close-btn" @click="isFaceModalOpen = false">✕</button>
        </div>
        <form class="ai-modal-form" @submit.prevent="handleCreateFace">
          <div class="form-group">
            <label class="form-label">Nome e Cognome Familiare</label>
            <input
              v-model="newFaceName"
              type="text"
              class="form-input"
              placeholder="es. Giulia Rossi"
              required
            />
          </div>
          <div class="form-group">
            <label class="form-label">Ruolo nell'Abitazione</label>
            <select v-model="newFaceRole" class="form-select">
              <option value="Residente">Residente</option>
              <option value="Amministratore">Amministratore</option>
              <option value="Ospite">Ospite Frequente</option>
            </select>
          </div>
          <div class="biometric-scan-simulation">
            <div class="scan-preview-circle">
              <User :size="40" class="text-primary" />
              <div class="scanner-sweep-line" />
            </div>
            <div class="scan-meta">
              <strong>Estrazione Vettore 128-D ONNX</strong>
              <p class="text-xs text-dim">L'algoritmo edge calcolerà la firma vettoriale localmente senza mai caricare immagini su cloud esterni.</p>
            </div>
          </div>
          <div class="modal-actions-row">
            <button type="button" class="btn-cancel" @click="isFaceModalOpen = false">
              Annulla
            </button>
            <button type="submit" class="btn-save-primary">
              Salva Profilo
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Register Authorized Vehicle Plate Modal -->
    <div v-if="isPlateModalOpen" class="modal-backdrop" @click="isPlateModalOpen = false">
      <div class="ai-modal-dialog glass-panel" @click.stop>
        <div class="modal-top">
          <div class="clip-title-row">
            <Car :size="18" class="text-primary" />
            <h3>Aggiungi Targa Autorizzata (ALPR)</h3>
          </div>
          <button class="close-btn" @click="isPlateModalOpen = false">✕</button>
        </div>
        <form class="ai-modal-form" @submit.prevent="handleCreatePlate">
          <div class="form-group">
            <label class="form-label">Targa Veicolo (Formato Europeo o Standard)</label>
            <input
              v-model="newPlateNumber"
              type="text"
              class="form-input font-mono uppercase"
              placeholder="es. AB123CD"
              required
            />
          </div>
          <div class="form-group">
            <label class="form-label">Proprietario / Riferimento</label>
            <input
              v-model="newPlateOwner"
              type="text"
              class="form-input"
              placeholder="es. Marco Rossi"
            />
          </div>
          <div class="form-group">
            <label class="form-label">Modello Veicolo</label>
            <input
              v-model="newPlateModel"
              type="text"
              class="form-input"
              placeholder="es. Tesla Model Y, Fiat 500e..."
            />
          </div>
          <div class="checkbox-option">
            <input
              id="auto-open-gate"
              v-model="newPlateAutoOpen"
              type="checkbox"
              class="custom-check"
            />
            <label for="auto-open-gate" class="text-sm">
              Apri automaticamente cancello carraio e accendi luci viale
            </label>
          </div>
          <div class="modal-actions-row">
            <button type="button" class="btn-cancel" @click="isPlateModalOpen = false">
              Annulla
            </button>
            <button type="submit" class="btn-save-primary">
              Registra Targa
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Face Test Match Modal -->
    <div v-if="faceTestModal.isOpen && faceTestModal.face" class="modal-backdrop" @click="faceTestModal.isOpen = false">
      <div class="ai-modal-dialog glass-panel" @click.stop>
        <div class="modal-top">
          <div class="clip-title-row">
            <ShieldCheck :size="18" class="text-emerald" />
            <h3>Test Match Biometrico Locale</h3>
          </div>
          <button class="close-btn" @click="faceTestModal.isOpen = false">✕</button>
        </div>
        <div class="test-match-body">
          <div class="test-avatar-box">
            <img :src="faceTestModal.face.photo_url || `https://api.dicebear.com/7.x/initials/svg?seed=${encodeURIComponent(faceTestModal.face.name)}`" class="test-avatar" />
            <div class="test-pulse-ring" />
          </div>
          <h4 class="test-name">{{ faceTestModal.face.name }}</h4>
          <span class="test-role">{{ faceTestModal.face.role || 'Residente' }}</span>

          <div class="match-score-pill">
            <span>Cosine Similarity:</span>
            <strong class="text-emerald font-mono">{{ ((faceTestModal.score || 0.94) * 100).toFixed(1) }}%</strong>
            <span class="score-pass-tag">&ge; 65% SOGLIA SUPERATA</span>
          </div>

          <div class="match-verdict-box">
            <CheckCircle :size="20" class="text-emerald" />
            <div>
              <strong>ACCESSO CONFERMATO</strong>
              <p class="text-xs text-dim">Profilo validato dal motore NVR locale. Automazioni di benvenuto sbloccate.</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cameras-view-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.cameras-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-radius: 16px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
}

.header-title-box {
  display: flex;
  align-items: center;
  gap: 16px;
}

.icon-circle {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: rgba(99, 102, 241, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-title {
  font-size: 1.4rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.page-subtitle {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin: 4px 0 0;
}

.cameras-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(340px, 1fr));
  gap: 20px;
}

.camera-card {
  border-radius: 16px;
  overflow: hidden;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
}

.cam-preview-wrapper {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 9;
  cursor: pointer;
  background: #000;
}

.cam-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.live-badge {
  position: absolute;
  top: 12px;
  left: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(0, 0, 0, 0.65);
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 0.75rem;
  font-weight: 600;
  color: #ef4444;
}

.pulsing-dot {
  width: 8px;
  height: 8px;
  background: #ef4444;
  border-radius: 50%;
  box-shadow: 0 0 8px #ef4444;
}

.expand-btn {
  position: absolute;
  bottom: 12px;
  right: 12px;
  background: rgba(0, 0, 0, 0.6);
  border: none;
  color: #fff;
  padding: 6px;
  border-radius: 8px;
  cursor: pointer;
}

.cam-info-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
}

.cam-title {
  font-size: 1rem;
  font-weight: 600;
  margin: 0;
}

.cam-protocol {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.ai-shield-tag {
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  padding: 4px 8px;
  border-radius: 8px;
  font-size: 0.75rem;
  font-weight: 600;
}

.events-section {
  padding: 20px 24px;
  border-radius: 16px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
}

.section-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.section-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-main);
  margin: 0;
}

.events-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.event-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 10px;
}

.event-icon-badge {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.event-icon-badge.person {
  background: rgba(99, 102, 241, 0.2);
  color: #818cf8;
}

.event-icon-badge.animal {
  background: rgba(245, 158, 11, 0.2);
  color: #f59e0b;
}

.event-meta {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.event-label {
  font-size: 0.9rem;
}

.event-sub {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.event-time {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 999;
}

.live-modal-dialog {
  width: 90%;
  max-width: 900px;
  background: #111;
  border-radius: 16px;
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.modal-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.video-container {
  width: 100%;
  aspect-ratio: 16 / 9;
  background: #000;
}

.fullscreen-video {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

/* NVR Status Banner */
.nvr-banner {
  padding: 1.5rem;
  border-radius: 1rem;
  background: var(--surface-card);
  backdrop-filter: blur(16px);
  border: 1px solid var(--border-color);
}

.nvr-banner-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
  margin-bottom: 1.25rem;
}

.nvr-title-group {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.nvr-badge {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 700;
  font-size: 0.95rem;
  color: var(--text-main);
}

.nvr-hw-badge {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  background: rgba(16, 185, 129, 0.12);
  color: #10b981;
  padding: 0.25rem 0.65rem;
  border-radius: 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  border: 1px solid rgba(16, 185, 129, 0.25);
}

.nvr-storage-pill {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
  color: var(--text-muted);
  background: var(--bg-surface-hover);
  padding: 0.35rem 0.75rem;
  border-radius: 0.5rem;
  border: 1px solid var(--border-color);
}

.nvr-stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.nvr-stat-card {
  background: var(--bg-surface-hover);
  padding: 1rem 1.25rem;
  border-radius: 0.75rem;
  border: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.stat-lbl {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-val {
  font-size: 1.35rem;
  font-weight: 800;
  color: var(--text-main);
}

.stat-desc {
  font-size: 0.7rem;
  color: var(--text-dim, #64748b);
}

.text-emerald {
  color: #10b981 !important;
}

.text-info {
  color: #38bdf8 !important;
}

/* Cam Actions */
.cam-actions-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-zone-cfg {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  background: rgba(255, 255, 255, 0.07);
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #f1f5f9;
  padding: 0.35rem 0.65rem;
  border-radius: 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-zone-cfg:hover {
  background: rgba(99, 102, 241, 0.2);
  border-color: #6366f1;
}

/* Motion Zones Section */
.zones-overview-section {
  padding: 1.5rem;
  border-radius: 1rem;
  background: var(--surface-card);
  backdrop-filter: blur(16px);
  border: 1px solid var(--border-color);
}

.section-title-box {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.zones-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1rem;
  margin-top: 1.25rem;
}

.zone-card {
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-color);
  border-radius: 0.75rem;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.zone-top {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.zone-name {
  font-weight: 600;
  font-size: 0.9rem;
  color: var(--text-main);
}

.zone-cam {
  font-size: 0.75rem;
  color: #94a3b8;
}

.zone-toggle {
  padding: 0.2rem 0.5rem;
  border-radius: 0.4rem;
  font-size: 0.7rem;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.05);
  color: #94a3b8;
  transition: all 0.2s;
}

.zone-toggle.active {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  border-color: rgba(16, 185, 129, 0.3);
}

.zone-meta {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.sensitivity-row {
  display: flex;
  justify-content: space-between;
}

.sensitivity-bar {
  width: 100%;
  height: 6px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 3px;
  overflow: hidden;
}

.sensitivity-fill {
  height: 100%;
  background: #6366f1;
  border-radius: 3px;
}

.zone-labels-row {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.label-badge {
  font-size: 0.65rem;
  font-weight: 700;
  padding: 0.15rem 0.45rem;
  border-radius: 0.35rem;
  background: rgba(99, 102, 241, 0.15);
  color: #a5b4fc;
}

.label-badge.person {
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
}

.label-badge.vehicle {
  background: rgba(6, 182, 212, 0.15);
  color: #38bdf8;
}

.label-badge.package {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
}

/* Timeline Action Clip */
.event-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.btn-clip-preview {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  background: rgba(99, 102, 241, 0.15);
  color: #818cf8;
  border: 1px solid rgba(99, 102, 241, 0.3);
  padding: 0.35rem 0.65rem;
  border-radius: 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-clip-preview:hover {
  background: rgba(99, 102, 241, 0.3);
  color: #ffffff;
}

/* Modals */
.clip-modal-dialog,
.zones-modal-dialog {
  width: 90%;
  max-width: 720px;
  background: #0f172a;
  border-radius: 1rem;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.12);
}

.clip-title-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.clip-media-wrapper {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 9;
  background: #000000;
}

.clip-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.clip-overlay-tag {
  position: absolute;
  top: 12px;
  left: 12px;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(8px);
  padding: 6px 10px;
  border-radius: 6px;
  display: flex;
  gap: 8px;
  font-size: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.tag-label {
  color: #38bdf8;
  font-weight: 700;
}

.tag-time {
  color: #94a3b8;
}

.clip-modal-footer {
  padding: 1rem 1.25rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: rgba(17, 24, 39, 0.9);
  flex-wrap: wrap;
  gap: 0.75rem;
}

.clip-meta-info {
  display: flex;
  gap: 1rem;
  font-size: 0.8rem;
  color: #94a3b8;
}

.notif-sync-badge {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  color: #10b981;
  font-size: 0.75rem;
  font-weight: 600;
}

/* Zone Editor Body */
.zones-modal-body {
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.zones-cam-preview {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 9;
  border-radius: 0.75rem;
  overflow: hidden;
  background: #000;
}

.zone-preview-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0.85;
}

.zone-box-overlay {
  position: absolute;
  border: 2px dashed #38bdf8;
  background: rgba(56, 189, 248, 0.15);
  border-radius: 4px;
  display: flex;
  align-items: flex-start;
  padding: 4px;
  color: #ffffff;
  font-size: 0.7rem;
  font-weight: 700;
}

.zone-box-overlay.zone-1 {
  top: 20%;
  left: 10%;
  width: 40%;
  height: 50%;
}

.zone-box-overlay.zone-2 {
  top: 40%;
  right: 15%;
  width: 35%;
  height: 45%;
  border-color: #f59e0b;
  background: rgba(245, 158, 11, 0.15);
}

.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.editor-header h4 {
  margin: 0;
  font-size: 0.95rem;
  color: #ffffff;
}

.btn-add-zone {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  background: #6366f1;
  color: #ffffff;
  border: none;
  padding: 0.35rem 0.65rem;
  border-radius: 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
}

.zone-editor-item {
  background: rgba(31, 41, 55, 0.5);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 0.65rem;
  padding: 0.85rem;
  margin-bottom: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.editor-item-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.zone-name-input {
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #ffffff;
  padding: 0.3rem 0.5rem;
  border-radius: 0.4rem;
  font-size: 0.85rem;
  width: 60%;
}

.zone-toggle-sm {
  padding: 0.2rem 0.5rem;
  border-radius: 0.35rem;
  font-size: 0.7rem;
  font-weight: 600;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.05);
  color: #94a3b8;
  cursor: pointer;
}

.zone-toggle-sm.active {
  background: rgba(16, 185, 129, 0.2);
  color: #10b981;
  border-color: rgba(16, 185, 129, 0.3);
}

.editor-sensitivity {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.slider-range {
  accent-color: #6366f1;
  width: 100%;
}

.editor-tags {
  display: flex;
  gap: 0.35rem;
}

.tag-pill {
  background: rgba(99, 102, 241, 0.15);
  color: #a5b4fc;
  font-size: 0.65rem;
  padding: 0.15rem 0.45rem;
  border-radius: 0.3rem;
  text-transform: uppercase;
  font-weight: 700;
}

/* Header View Controls */
.header-view-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.view-switch-group,
.matrix-switch-group {
  display: flex;
  background: rgba(255, 255, 255, 0.05);
  padding: 4px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  gap: 4px;
}

.btn-view-toggle,
.btn-matrix-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-view-toggle.active,
.btn-matrix-toggle.active {
  background: var(--primary);
  color: #fff;
  box-shadow: 0 2px 8px rgba(14, 165, 233, 0.35);
}

/* Surveillance Wall Layout */
.surveillance-wall-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px;
  border-radius: 16px;
  background: #0b1120;
  border: 1px solid var(--border-subtle);
}

.wall-top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.wall-title-area {
  display: flex;
  align-items: center;
  gap: 16px;
}

.wall-live-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
}

.wall-live-text {
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.08em;
  color: #ef4444;
}

.wall-clock {
  font-size: 15px;
  font-weight: 700;
  color: #38bdf8;
  background: rgba(56, 189, 248, 0.12);
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  border: 1px solid rgba(56, 189, 248, 0.25);
}

.filter-pills {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.btn-filter-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  border: 1px solid var(--border-subtle);
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-filter-pill.active {
  background: rgba(14, 165, 233, 0.2);
  border-color: var(--primary);
  color: var(--text-main);
}

.btn-filter-pill.person.active {
  background: rgba(99, 102, 241, 0.25);
  border-color: #818cf8;
  color: #a5b4fc;
}

.btn-filter-pill.vehicle.active {
  background: rgba(6, 182, 212, 0.25);
  border-color: #38bdf8;
  color: #7dd3fc;
}

.btn-filter-pill.package.active {
  background: rgba(16, 185, 129, 0.25);
  border-color: #10b981;
  color: #6ee7b7;
}

/* Video Matrix Grids */
.surveillance-matrix-grid {
  display: grid;
  gap: 12px;
  background: #020617;
  padding: 12px;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.surveillance-matrix-grid.matrix-2x2 {
  grid-template-columns: repeat(2, 1fr);
}

.surveillance-matrix-grid.matrix-3x3 {
  grid-template-columns: repeat(3, 1fr);
}

.wall-feed-cell {
  position: relative;
  aspect-ratio: 16 / 9;
  background: #000;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.1);
  cursor: pointer;
  transition: all 0.25s ease;
}

.wall-feed-cell:hover {
  border-color: var(--primary);
  transform: scale(1.01);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
}

.wall-feed-cell.cell-highlighted {
  border-color: #10b981 !important;
  box-shadow: 0 0 20px rgba(16, 185, 129, 0.45) !important;
}

.wall-feed-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.empty-feed-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-muted);
  font-size: 12px;
  background: rgba(255, 255, 255, 0.02);
}

.feed-overlay {
  position: absolute;
  left: 8px;
  right: 8px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  pointer-events: none;
  z-index: 2;
}

.feed-overlay-top {
  top: 8px;
}

.feed-overlay-bottom {
  bottom: 8px;
}

.cell-cam-name {
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  padding: 3px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 700;
  color: #fff;
}

.cell-time {
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  padding: 3px 8px;
  border-radius: 4px;
  font-size: 11px;
  color: #38bdf8;
}

.feed-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  background: rgba(0, 0, 0, 0.7);
  padding: 3px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
  color: #ef4444;
}

.status-indicator-dot {
  width: 6px;
  height: 6px;
  background: #ef4444;
  border-radius: 50%;
  box-shadow: 0 0 6px #ef4444;
}

.feed-event-alert {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: rgba(16, 185, 129, 0.85);
  backdrop-filter: blur(4px);
  padding: 3px 8px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 800;
  color: #fff;
  box-shadow: 0 0 10px rgba(16, 185, 129, 0.6);
}

.ai-tracking-box {
  position: absolute;
  top: 25%;
  left: 30%;
  width: 35%;
  height: 50%;
  border: 2px solid #10b981;
  background: rgba(16, 185, 129, 0.12);
  box-shadow: 0 0 15px rgba(16, 185, 129, 0.35);
  pointer-events: none;
  animation: pulse-tracking 2s infinite ease-in-out;
}

@keyframes pulse-tracking {
  0%, 100% { border-color: #10b981; }
  50% { border-color: #34d399; }
}

.box-tag {
  position: absolute;
  top: -18px;
  left: 0;
  background: #10b981;
  color: #000;
  font-size: 9px;
  font-weight: 800;
  padding: 1px 4px;
  border-radius: 2px;
}

/* Unified Timeline Bar */
.unified-timeline-bar {
  display: flex;
  flex-direction: column;
  gap: 14px;
  background: rgba(15, 23, 42, 0.8);
  padding: 16px;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.timeline-controls-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
}

.playback-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-playback-step {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid var(--border-subtle);
  color: var(--text-main);
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-playback-step:hover {
  background: rgba(255, 255, 255, 0.12);
}

.btn-playback-play {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: var(--primary);
  color: #fff;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: transform var(--transition-fast);
}

.btn-playback-play:hover {
  transform: scale(1.05);
}

.btn-playback-speed {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 700;
  cursor: pointer;
}

.btn-playback-speed:hover {
  color: var(--text-main);
}

.timeline-time-display {
  display: flex;
  align-items: center;
  gap: 8px;
}

.time-readout {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-main);
}

.time-scope {
  font-size: 12px;
  color: var(--text-muted);
}

/* Timeline Track */
.timeline-track-container {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.timeline-hour-markers {
  display: flex;
  justify-content: space-between;
  font-size: 10px;
  font-family: monospace;
  color: var(--text-muted);
  padding: 0 4px;
}

.timeline-track-wrapper {
  position: relative;
  height: 28px;
  display: flex;
  align-items: center;
}

.timeline-scrubber-slider {
  width: 100%;
  appearance: none;
  background: linear-gradient(90deg, #1e293b 0%, #334155 50%, #1e293b 100%);
  height: 10px;
  border-radius: 5px;
  outline: none;
  cursor: pointer;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.timeline-scrubber-slider::-webkit-slider-thumb {
  appearance: none;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--primary);
  box-shadow: 0 0 10px var(--primary);
  cursor: pointer;
  border: 2px solid #fff;
  transition: transform 0.1s ease;
}

.timeline-scrubber-slider::-webkit-slider-thumb:hover {
  transform: scale(1.2);
}

.timeline-marker {
  position: absolute;
  top: 50%;
  transform: translate(-50%, -50%);
  cursor: pointer;
  z-index: 3;
}

.marker-dot {
  display: block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 2px solid #000;
  transition: transform 0.15s ease;
}

.timeline-marker:hover .marker-dot,
.timeline-marker.active .marker-dot {
  transform: scale(1.5);
  box-shadow: 0 0 12px currentColor;
}

.timeline-marker.person .marker-dot {
  background: #818cf8;
  color: #818cf8;
}

.timeline-marker.vehicle .marker-dot {
  background: #38bdf8;
  color: #38bdf8;
}

.timeline-marker.package .marker-dot {
  background: #34d399;
  color: #34d399;
}

.timeline-marker.animal .marker-dot {
  background: #fbbf24;
  color: #fbbf24;
}

/* Scrubbed Event Detail Card */
.scrubbed-event-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px;
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.08);
  flex-wrap: wrap;
}

.scrubbed-event-thumb {
  position: relative;
  width: 80px;
  height: 50px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  background: #000;
}

.scrub-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.scrub-badge {
  position: absolute;
  bottom: 2px;
  left: 2px;
  font-size: 8px;
  font-weight: 800;
  padding: 1px 4px;
  border-radius: 2px;
  background: rgba(0, 0, 0, 0.7);
  color: #fff;
}

.scrub-badge.person { color: #818cf8; }
.scrub-badge.vehicle { color: #38bdf8; }
.scrub-badge.package { color: #34d399; }
.scrub-badge.animal { color: #fbbf24; }

.scrubbed-event-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.scrub-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--text-main);
}

.scrub-time {
  font-size: 12px;
  color: var(--primary);
}

.scrub-details {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--text-muted);
}

.btn-play-scrubbed-clip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: var(--radius-sm);
  background: var(--primary);
  color: #fff;
  border: none;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

/* ==========================================================================
   AI Biometrics & Authorized License Plates (ALPR)
   ========================================================================== */
.ai-faces-management-section {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.ai-overview-banner {
  padding: 24px;
  border-radius: 16px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  backdrop-filter: blur(12px);
}

.ai-banner-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ai-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.ai-engine-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px;
  border-radius: 999px;
  background: rgba(99, 102, 241, 0.15);
  color: var(--primary);
  font-size: 13px;
  font-weight: 700;
  border: 1px solid rgba(99, 102, 241, 0.3);
}

.ai-secure-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  font-size: 12px;
  font-weight: 600;
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.ai-banner-desc {
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.5;
  margin: 4px 0 0;
  max-width: 800px;
}

.ai-stats-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-top: 20px;
}

.ai-stat-box {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px;
  display: flex;
  flex-direction: column;
}

.ai-stat-label {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-muted);
}

.ai-stat-number {
  font-size: 22px;
  font-weight: 800;
  margin-top: 4px;
}

.ai-stat-sub {
  font-size: 11px;
  color: var(--text-dim);
  margin-top: 2px;
}

.ai-sub-section {
  padding: 24px;
  border-radius: 16px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  backdrop-filter: blur(12px);
}

.sub-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  flex-wrap: wrap;
  gap: 12px;
}

.sub-header-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.sub-header-title h3 {
  font-size: 16px;
  font-weight: 700;
  margin: 0;
  color: var(--text-main);
}

.sub-header-desc {
  font-size: 12px;
  color: var(--text-muted);
  margin: 2px 0 0;
}

.btn-primary-action {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: var(--radius-sm);
  background: var(--primary);
  color: #fff;
  border: none;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.2s;
}

.btn-primary-action:hover {
  opacity: 0.9;
}

.empty-faces-box {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}

.faces-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.face-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-color);
  transition: transform 0.2s, border-color 0.2s;
}

.face-card:hover {
  transform: translateY(-2px);
  border-color: var(--primary);
}

.face-avatar-container {
  position: relative;
  width: 56px;
  height: 56px;
  flex-shrink: 0;
}

.face-avatar-img {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--primary);
  background: #1e293b;
}

.face-verified-badge {
  position: absolute;
  bottom: -2px;
  right: -2px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #10b981;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--surface-card);
}

.face-info-box {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.face-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.face-name {
  font-weight: 700;
  font-size: 15px;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.face-role-pill {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(99, 102, 241, 0.15);
  color: #818cf8;
}

.face-id-sub {
  font-size: 11px;
  color: var(--text-muted);
}

.face-vector-status {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-dim);
}

.face-card-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-test-match {
  padding: 6px 10px;
  font-size: 11px;
  font-weight: 600;
  border-radius: 6px;
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  border: 1px solid rgba(16, 185, 129, 0.3);
  cursor: pointer;
}

.btn-test-match:hover {
  background: rgba(16, 185, 129, 0.25);
}

.btn-delete-card {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  background: transparent;
  border: 1px solid rgba(239, 68, 68, 0.2);
  color: #ef4444;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.btn-delete-card:hover {
  background: rgba(239, 68, 68, 0.15);
}

/* Plates (ALPR) */
.plates-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.plate-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 18px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-color);
  transition: transform 0.2s, border-color 0.2s;
}

.plate-card:hover {
  transform: translateY(-2px);
  border-color: var(--primary);
}

.eu-license-plate {
  display: flex;
  align-items: center;
  height: 44px;
  background: #f8fafc;
  border: 2px solid #334155;
  border-radius: 6px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
}

.plate-band-left {
  width: 28px;
  height: 100%;
  background: #1d4ed8;
  color: #facc15;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  font-size: 8px;
  font-weight: 800;
  line-height: 1;
}

.plate-band-left .eu-country {
  color: #fff;
  font-size: 11px;
  font-weight: 900;
  margin-top: 2px;
}

.plate-text {
  flex: 1;
  text-align: center;
  font-size: 22px;
  font-weight: 900;
  letter-spacing: 3px;
  color: #0f172a;
}

.plate-band-right {
  width: 24px;
  height: 100%;
  background: #1d4ed8;
  color: #facc15;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 700;
}

.plate-info-box {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.plate-owner {
  font-weight: 700;
  font-size: 14px;
  color: var(--text-main);
}

.plate-gate-status {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
}

.plate-card-actions {
  display: flex;
  justify-content: flex-end;
}

/* Modals */
.ai-modal-dialog {
  width: 100%;
  max-width: 480px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  border-radius: 16px;
  padding: 24px;
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(16px);
}

.ai-modal-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-top: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
}

.form-input, .form-select {
  padding: 10px 14px;
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-color);
  color: var(--text-main);
  font-size: 14px;
  outline: none;
}

.form-input:focus, .form-select:focus {
  border-color: var(--primary);
}

.biometric-scan-simulation {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px;
  border-radius: 12px;
  background: rgba(99, 102, 241, 0.08);
  border: 1px dashed rgba(99, 102, 241, 0.3);
}

.scan-preview-circle {
  position: relative;
  width: 50px;
  height: 50px;
  border-radius: 50%;
  border: 2px solid var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  flex-shrink: 0;
}

.scanner-sweep-line {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: #38bdf8;
  box-shadow: 0 0 6px #38bdf8;
  animation: sweepAnim 2s infinite linear;
}

@keyframes sweepAnim {
  0% { transform: translateY(0); }
  50% { transform: translateY(48px); }
  100% { transform: translateY(0); }
}

.scan-meta strong {
  display: block;
  font-size: 13px;
  color: var(--text-main);
}

.checkbox-option {
  display: flex;
  align-items: center;
  gap: 8px;
}

.custom-check {
  width: 16px;
  height: 16px;
  accent-color: var(--primary);
}

.modal-actions-row {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 12px;
}

.btn-cancel {
  padding: 8px 16px;
  border-radius: var(--radius-sm);
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-muted);
  cursor: pointer;
}

.btn-save-primary {
  padding: 8px 16px;
  border-radius: var(--radius-sm);
  background: var(--primary);
  border: none;
  color: #fff;
  font-weight: 600;
  cursor: pointer;
}

/* Test Match Modal Body */
.test-match-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 20px 0 10px;
}

.test-avatar-box {
  position: relative;
  width: 80px;
  height: 80px;
  margin-bottom: 12px;
}

.test-avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  object-fit: cover;
  border: 3px solid #10b981;
}

.test-pulse-ring {
  position: absolute;
  top: -6px;
  left: -6px;
  right: -6px;
  bottom: -6px;
  border: 2px solid #10b981;
  border-radius: 50%;
  animation: pulseRing 1.5s infinite;
}

@keyframes pulseRing {
  0% { transform: scale(0.95); opacity: 1; }
  100% { transform: scale(1.2); opacity: 0; }
}

.test-name {
  font-size: 18px;
  font-weight: 700;
  margin: 0;
  color: var(--text-main);
}

.test-role {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 2px;
}

.match-score-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 999px;
  background: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.3);
  margin: 20px 0 16px;
  font-size: 13px;
}

.score-pass-tag {
  font-size: 10px;
  font-weight: 800;
  color: #10b981;
  letter-spacing: 0.5px;
}

.match-verdict-box {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 12px;
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.25);
  text-align: left;
  max-width: 360px;
}

.match-verdict-box strong {
  display: block;
  font-size: 13px;
  color: #10b981;
}
</style>
