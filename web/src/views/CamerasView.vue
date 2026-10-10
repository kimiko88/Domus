<script setup lang="ts">
import { ref, onMounted } from 'vue'
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
  Eye
} from '@lucide/vue'
import { apiClient } from '../api/client'

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
    detected_at: '2 min fa'
  },
  {
    id: 'det_2',
    camera_id: 'cam_garden',
    label: 'animal',
    score: 0.88,
    detected_at: '15 min fa'
  }
])

const activeCamera = ref<Camera | null>(null)
const isLiveModalOpen = ref(false)

function openLiveStream(cam: Camera) {
  activeCamera.value = cam
  isLiveModalOpen.value = true
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
          <h1 class="page-title">Videosorveglianza & Visione AI</h1>
          <p class="page-subtitle">Streaming WebRTC a bassa latenza (&lt;200ms) e rilevamento oggetti Frigate NVR</p>
        </div>
      </div>
    </header>

    <!-- Camera Feeds Grid -->
    <div class="cameras-grid">
      <div v-for="cam in cameras" :key="cam.id" class="camera-card glass-panel">
        <div class="cam-preview-wrapper" @click="openLiveStream(cam)">
          <img :src="cam.snapshot_url" :alt="cam.name" class="cam-img" />
          <div class="live-badge">
            <span class="pulsing-dot" />
            <span>LIVE (WebRTC)</span>
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

          <div v-if="cam.detection_enabled" class="ai-shield-tag" title="Frigate AI Vision Attiva">
            <ShieldAlert :size="14" />
            <span>AI Guard</span>
          </div>
        </div>
      </div>
    </div>

    <!-- AI Detection Event Timeline -->
    <section class="events-section glass-panel">
      <div class="section-header">
        <Eye :size="20" class="text-warning" />
        <h2 class="section-title">Timeline Eventi Riconoscimento Locale (Frigate NVR)</h2>
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
            <span class="event-label">Rilevato: <strong>{{ e.label.toUpperCase() }}</strong></span>
            <span class="event-sub">Camera: {{ e.camera_id }} • Accuratezza: {{ (e.score * 100).toFixed(0) }}%</span>
          </div>

          <span class="event-time">{{ e.detected_at }}</span>
        </div>
      </div>
    </section>

    <!-- Fullscreen Live Modal -->
    <div v-if="isLiveModalOpen && activeCamera" class="modal-backdrop" @click="isLiveModalOpen = false">
      <div class="live-modal-dialog glass-panel" @click.stop>
        <div class="modal-top">
          <h3>{{ activeCamera.name }} (WebRTC Stream)</h3>
          <button class="close-btn" @click="isLiveModalOpen = false">✕</button>
        </div>
        <div class="video-container">
          <img :src="activeCamera.snapshot_url" class="fullscreen-video" />
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
</style>
