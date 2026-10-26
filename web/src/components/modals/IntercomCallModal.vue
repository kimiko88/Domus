<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Phone, PhoneOff, Mic, MicOff, Key, Volume2, ShieldCheck, Video, Sparkles, X } from '@lucide/vue'
import { apiClient } from '../../api/client'
import { useNotificationsStore } from '../../stores/notifications'

interface Props {
  isOpen: boolean
  cameraId?: string
  callerName?: string
}

const props = withDefaults(defineProps<Props>(), {
  isOpen: false,
  cameraId: 'cam_front_door',
  callerName: 'Visitatore al Cancello'
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'unlocked'): void
}>()

const notifStore = useNotificationsStore()

const isMuted = ref(false)
const isTwoWayAudioActive = ref(true)
const isUnlocking = ref(false)
const unlockSuccess = ref(false)
const latencyMs = ref(142)
const isAudioNegotiated = ref(false)

onMounted(() => {
  if (props.isOpen) {
    negotiateIntercom()
  }
})

async function negotiateIntercom() {
  try {
    const res = await apiClient.post<{ latency_ms: number; aec_enabled: boolean }>(
      `/api/v1/intercom/${props.cameraId}/audio`,
      {
        offer_sdp: 'v=0\r\no=Client 1234 2 IN IP4 127.0.0.1\r\nm=audio 9 UDP/TLS/RTP/SAVPF 111',
        aec: true
      }
    )
    if (res && res.latency_ms) {
      latencyMs.value = res.latency_ms
      isAudioNegotiated.value = true
    }
  } catch (err) {
    console.warn('Intercom audio negotiation fallback:', err)
  }
}

async function unlockGate() {
  isUnlocking.value = true
  try {
    await apiClient.post(`/api/v1/intercom/${props.cameraId}/unlock`, {})
    unlockSuccess.value = true
    notifStore.success('Cancello Sbloccato', 'Il cancelletto pedonale è stato aperto')
    emit('unlocked')
    setTimeout(() => {
      unlockSuccess.value = false
    }, 3500)
  } catch (err: any) {
    notifStore.error('Errore Sblocco', err?.message || 'Impossibile sbloccare il cancello')
  } finally {
    isUnlocking.value = false
  }
}

function toggleMute() {
  isMuted.value = !isMuted.value
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.isOpen) {
    emit('close')
  }
}

onMounted(() => window.addEventListener('keydown', handleKeydown))
onUnmounted(() => window.removeEventListener('keydown', handleKeydown))
</script>

<template>
  <div v-if="isOpen" class="intercom-backdrop" @click.self="emit('close')">
    <div
      class="intercom-modal glass-card"
      role="dialog"
      aria-modal="true"
      aria-labelledby="intercom-title"
    >
      <!-- Header -->
      <div class="intercom-header">
        <div class="header-left">
          <div class="live-badge">
            <span class="live-dot"></span>
            <span>LIVE WebRTC</span>
          </div>
          <span class="latency-text">{{ latencyMs }}ms • AEC Attivo</span>
        </div>
        <button class="btn-close-mini" @click="emit('close')" aria-label="Chiudi chiamata">
          <X :size="18" />
        </button>
      </div>

      <!-- Video Feed Container -->
      <div class="video-container">
        <div class="video-overlay-top">
          <div class="caller-box">
            <h3 id="intercom-title" class="caller-name">{{ callerName }}</h3>
            <span class="caller-sub">Campanello Ingresso Principale</span>
          </div>
          <div class="stream-encryption-badge" title="Flusso WebRTC crittografato SRTP end-to-end">
            <ShieldCheck :size="14" />
            <span>SRTP</span>
          </div>
        </div>

        <!-- Simulated Low-Latency Feed Graphic -->
        <div class="mock-video-canvas">
          <div class="video-placeholder-graphics">
            <div class="crosshair crosshair-tl"></div>
            <div class="crosshair crosshair-tr"></div>
            <div class="crosshair crosshair-bl"></div>
            <div class="crosshair crosshair-br"></div>
            <div class="person-silhouette">
              <div class="head"></div>
              <div class="body"></div>
            </div>
          </div>
          <div class="feed-timestamp">
            <span>{{ new Date().toLocaleTimeString() }} • 1080p 30fps H.264</span>
          </div>
        </div>

        <!-- Two-Way Audio Indicator Banner -->
        <div class="audio-banner">
          <div class="audio-left">
            <Volume2 :size="16" class="text-primary" />
            <span>Audio Bi-direzionale (Cancellazione Eco AEC)</span>
          </div>
          <button
            :class="['btn-mute-toggle', { muted: isMuted }]"
            @click="toggleMute"
            :aria-label="isMuted ? 'Riattiva microfono' : 'Disattiva microfono'"
          >
            <component :is="isMuted ? MicOff : Mic" :size="16" />
            <span>{{ isMuted ? 'Muto' : 'Microfono ON' }}</span>
          </button>
        </div>
      </div>

      <!-- Footer Action Controls -->
      <div class="intercom-footer">
        <button
          class="btn-intercom-end"
          @click="emit('close')"
          aria-label="Termina chiamata citofono"
        >
          <PhoneOff :size="18" />
          <span>Chiudi</span>
        </button>

        <button
          :class="['btn-intercom-unlock', { success: unlockSuccess }]"
          :disabled="isUnlocking"
          @click="unlockGate"
          aria-label="Apri Cancello Pedonale"
        >
          <Key :size="20" :class="{ 'spin-icon': isUnlocking }" />
          <span>{{ unlockSuccess ? 'Cancello Aperto!' : isUnlocking ? 'Sblocco...' : 'Apri Cancello' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.intercom-backdrop {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: rgba(0, 0, 0, 0.85);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  animation: fadeIn 0.2s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.intercom-modal {
  width: 100%;
  max-width: 540px;
  background: #0f172a;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: var(--radius-2xl, 20px);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.7);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.intercom-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.live-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 8px;
  border-radius: var(--radius-full);
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
}

.live-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #ef4444;
  animation: blink 1.2s ease-in-out infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.85); }
}

.latency-text {
  font-size: 12px;
  color: #94a3b8;
  font-family: var(--font-mono);
}

.btn-close-mini {
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 6px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.btn-close-mini:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #ffffff;
}

/* Video Feed Box */
.video-container {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 10;
  background: #020617;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  overflow: hidden;
}

.video-overlay-top {
  position: absolute;
  top: 14px;
  left: 14px;
  right: 14px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  z-index: 10;
  pointer-events: none;
}

.caller-box {
  background: rgba(15, 23, 42, 0.75);
  backdrop-filter: blur(6px);
  padding: 8px 12px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.caller-name {
  font-size: 15px;
  font-weight: 700;
  color: #f8fafc;
}

.caller-sub {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 1px;
  display: block;
}

.stream-encryption-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border-radius: 6px;
  background: rgba(16, 185, 129, 0.2);
  color: #34d399;
  font-size: 11px;
  font-weight: 600;
}

.mock-video-canvas {
  flex: 1;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(circle at center, #1e293b 0%, #020617 100%);
}

.video-placeholder-graphics {
  width: 180px;
  height: 180px;
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.crosshair {
  position: absolute;
  width: 16px;
  height: 16px;
  border-color: rgba(14, 165, 233, 0.6);
  border-style: solid;
}

.crosshair-tl { top: 0; left: 0; border-width: 2px 0 0 2px; }
.crosshair-tr { top: 0; right: 0; border-width: 2px 2px 0 0; }
.crosshair-bl { bottom: 0; left: 0; border-width: 0 0 2px 2px; }
.crosshair-br { bottom: 0; right: 0; border-width: 0 2px 2px 0; }

.person-silhouette {
  display: flex;
  flex-direction: column;
  align-items: center;
  opacity: 0.35;
}

.person-silhouette .head {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: #cbd5e1;
  margin-bottom: 6px;
}

.person-silhouette .body {
  width: 80px;
  height: 50px;
  border-radius: 36px 36px 0 0;
  background: #cbd5e1;
}

.feed-timestamp {
  position: absolute;
  bottom: 8px;
  left: 14px;
  font-size: 11px;
  color: #64748b;
  font-family: var(--font-mono);
}

/* Audio Banner */
.audio-banner {
  padding: 10px 16px;
  background: rgba(15, 23, 42, 0.9);
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  align-items: center;
  justify-content: space-between;
  z-index: 5;
}

.audio-left {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #e2e8f0;
}

.btn-mute-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #f8fafc;
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-mute-toggle.muted {
  background: rgba(239, 68, 68, 0.2);
  border-color: rgba(239, 68, 68, 0.4);
  color: #fca5a5;
}

/* Footer Controls */
.intercom-footer {
  padding: 18px 24px;
  background: #0f172a;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.btn-intercom-end {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 12px 24px;
  border-radius: 12px;
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #f87171;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-intercom-end:hover {
  background: #ef4444;
  color: #ffffff;
}

.btn-intercom-unlock {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 12px 24px;
  border-radius: 12px;
  background: #10b981;
  border: 1px solid #10b981;
  color: #ffffff;
  font-size: 15px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
  box-shadow: 0 4px 14px rgba(16, 185, 129, 0.35);
}

.btn-intercom-unlock:hover:not(:disabled) {
  background: #059669;
  transform: translateY(-1px);
}

.btn-intercom-unlock.success {
  background: #047857;
  border-color: #047857;
}

.btn-intercom-unlock:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Light Theme Contrast Overrides */
[data-theme="light"] .intercom-modal {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
}

[data-theme="light"] .intercom-header {
  border-color: #e2e8f0;
}

[data-theme="light"] .caller-box {
  background: rgba(255, 255, 255, 0.9);
  border-color: #cbd5e1;
}

[data-theme="light"] .caller-name {
  color: #0f172a;
}

[data-theme="light"] .caller-sub {
  color: #475569;
}

[data-theme="light"] .audio-banner {
  background: #f8fafc;
  border-color: #e2e8f0;
}

[data-theme="light"] .audio-left {
  color: #1e293b;
}

[data-theme="light"] .btn-mute-toggle {
  background: #e2e8f0;
  border-color: #cbd5e1;
  color: #0f172a;
}

[data-theme="light"] .intercom-footer {
  background: #ffffff;
  border-color: #e2e8f0;
}
</style>
