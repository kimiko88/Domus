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
  Volume2
} from '@lucide/vue'
import { apiClient } from '../api/client'
import { soundEffects } from '../utils/audioSynthesizer'

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

// Surveillance Wall & AI Timeline Scrubbing
const currentViewMode = ref<'grid' | 'wall'>('grid')
const wallMatrix = ref<'2x2' | '3x3'>('2x2')
const isWallPlaying = ref(true)
const wallPlaybackSpeed = ref(1)
const timelineCurrentMinute = ref(12 * 60 + 30) // default 12:30
const selectedLabelFilter = ref<'all' | 'person' | 'vehicle' | 'package' | 'animal'>('all')
const activeScrubbedEvent = ref<DetectionEvent | null>(null)

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
  background: rgba(17, 24, 39, 0.7);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.08);
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
  color: #ffffff;
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
  color: #94a3b8;
  background: rgba(255, 255, 255, 0.04);
  padding: 0.35rem 0.75rem;
  border-radius: 0.5rem;
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.nvr-stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.nvr-stat-card {
  background: rgba(31, 41, 55, 0.4);
  padding: 1rem 1.25rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.05);
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.stat-lbl {
  font-size: 0.75rem;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-val {
  font-size: 1.35rem;
  font-weight: 800;
  color: #ffffff;
}

.stat-desc {
  font-size: 0.7rem;
  color: #64748b;
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
  background: rgba(17, 24, 39, 0.7);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.08);
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
  background: rgba(31, 41, 55, 0.45);
  border: 1px solid rgba(255, 255, 255, 0.06);
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
  color: #ffffff;
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
</style>
