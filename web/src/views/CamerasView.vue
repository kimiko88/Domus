<script setup lang="ts">
import { ref, onMounted } from 'vue'
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
  Film,
  CheckCircle,
  Sliders
} from '@lucide/vue'
import { apiClient } from '../api/client'

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

const cameras = ref<Camera[]>([
  {
    id: 'cam_front',
    name: 'Ingresso Principale',
    stream_url: 'rtsp://192.168.1.50/live/ch0',
    snapshot_url: 'https://images.unsplash.com/photo-1558036117-15d82a90b9b1?auto=format&fit=crop&w=600&q=80',
    stream_type: 'webrtc',
    detection_enabled: true
  },
  {
    id: 'cam_garden',
    name: 'Giardino Posteriore',
    stream_url: 'rtsp://192.168.1.51/live/ch0',
    snapshot_url: 'https://images.unsplash.com/photo-1584467735871-8e85353a8413?auto=format&fit=crop&w=600&q=80',
    stream_type: 'webrtc',
    detection_enabled: true
  }
])

const events = ref<DetectionEvent[]>([
  {
    id: 'det_1',
    camera_id: 'cam_front',
    label: 'person',
    score: 0.94,
    detected_at: '2 min fa',
    clip_duration_sec: 5.4,
    thumbnail_url: 'https://images.unsplash.com/photo-1558036117-15d82a90b9b1?auto=format&fit=crop&w=600&q=80'
  },
  {
    id: 'det_2',
    camera_id: 'cam_garden',
    label: 'animal',
    score: 0.88,
    detected_at: '15 min fa',
    clip_duration_sec: 4.1,
    thumbnail_url: 'https://images.unsplash.com/photo-1584467735871-8e85353a8413?auto=format&fit=crop&w=600&q=80'
  },
  {
    id: 'det_3',
    camera_id: 'cam_front',
    label: 'package',
    score: 0.96,
    detected_at: '1 ora fa',
    clip_duration_sec: 6.0,
    thumbnail_url: 'https://images.unsplash.com/photo-1558036117-15d82a90b9b1?auto=format&fit=crop&w=600&q=80'
  }
])

const nvrStatus = ref<NVRSystemStatus>({
  engine: 'Domus Local AI Vision (ONNX)',
  total_cameras: 2,
  active_streams: 2,
  detection_fps: 28.5,
  average_latency_ms: 18,
  hardware_accel: 'CPU AVX2 SIMD & OpenVINO NPU',
  storage_used_gb: 34.6,
  storage_limit_gb: 250.0,
  cameras: [
    {
      camera_id: 'cam_front',
      camera_name: 'Ingresso Principale',
      stream_status: 'online',
      current_fps: 25.0,
      detection_ms: 17,
      total_events_24h: 38
    },
    {
      camera_id: 'cam_garden',
      camera_name: 'Giardino Posteriore',
      stream_status: 'online',
      current_fps: 25.0,
      detection_ms: 19,
      total_events_24h: 14
    }
  ]
})

const motionZones = ref<MotionZone[]>([
  {
    id: 'zone_porch',
    camera_id: 'cam_front',
    name: 'Portico & Consegna Pacchi',
    sensitivity: 85,
    labels: ['person', 'package'],
    enabled: true
  },
  {
    id: 'zone_driveway',
    camera_id: 'cam_front',
    name: "Vialetto d'Accesso & Veicoli",
    sensitivity: 75,
    labels: ['person', 'vehicle'],
    enabled: true
  },
  {
    id: 'zone_garden_lawn',
    camera_id: 'cam_garden',
    name: 'Prato Posteriore & Perimetro',
    sensitivity: 70,
    labels: ['person', 'animal'],
    enabled: true
  }
])

const activeCamera = ref<Camera | null>(null)
const isLiveModalOpen = ref(false)
const selectedCameraForZones = ref<Camera | null>(null)
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

onMounted(async () => {
  try {
    const list = await apiClient.get<Camera[]>('/api/v1/cameras')
    if (list && list.length > 0) cameras.value = list
  } catch {}

  try {
    const evts = await apiClient.get<DetectionEvent[]>('/api/v1/cameras/events')
    if (evts && evts.length > 0) events.value = evts
  } catch {}

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
    </header>

    <!-- NVR System Status Banner -->
    <section class="nvr-banner glass-panel">
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

    <!-- Camera Feeds Grid -->
    <div class="cameras-grid">
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

      <div class="zones-grid">
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

      <div class="events-list">
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
              v-for="zone in motionZones.filter(z => z.camera_id === selectedCameraForZones?.id)"
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
</style>
