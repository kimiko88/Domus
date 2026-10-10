<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  Volume2,
  VolumeX,
  Play,
  Pause,
  Cast,
  Radio,
  Bell,
  Sliders,
  Music
} from '@lucide/vue'
import { apiClient } from '../api/client'

interface MediaPlayer {
  id: string
  name: string
  platform: 'cast' | 'airplay' | 'dlna' | 'local'
  state: 'idle' | 'playing' | 'paused'
  volume: number
  muted: boolean
  current_title?: string
  current_artist?: string
}

const players = ref<MediaPlayer[]>([
  {
    id: 'media_living',
    name: 'Google Nest Hub Salotto',
    platform: 'cast',
    state: 'playing',
    volume: 0.65,
    muted: false,
    current_title: 'Radio Monte Carlo',
    current_artist: 'Live Streaming Web'
  },
  {
    id: 'media_kitchen',
    name: 'HomePod Mini Cucina',
    platform: 'airplay',
    state: 'paused',
    volume: 0.40,
    muted: false,
    current_title: 'Podcast Mattutino',
    current_artist: 'Notizie & Meteo'
  },
  {
    id: 'media_bedroom',
    name: 'Speaker Bluetooth Camera',
    platform: 'local',
    state: 'idle',
    volume: 0.30,
    muted: false
  }
])

const isBroadcastModalOpen = ref(false)
const broadcastMessage = ref('Attenzione: Qualcuno ha suonato al campanello!')
const broadcastStatus = ref('')

async function togglePlay(player: MediaPlayer) {
  player.state = player.state === 'playing' ? 'paused' : 'playing'
  try {
    await apiClient.post(`/api/v1/media/players/${player.id}/state`, { state: player.state })
  } catch {}
}

async function handleVolumeChange(player: MediaPlayer, e: Event) {
  const val = parseFloat((e.target as HTMLInputElement).value)
  player.volume = val
  try {
    await apiClient.post(`/api/v1/media/players/${player.id}/volume`, { volume: val })
  } catch {}
}

async function sendBroadcast() {
  if (!broadcastMessage.value.trim()) return
  try {
    await apiClient.post('/api/v1/media/broadcast', {
      message: broadcastMessage.value,
      volume: 0.85,
      audio_url: '/tts/bell_chime.wav'
    })
    broadcastStatus.value = 'Annuncio diffuso con successo su tutti gli speaker!'
    setTimeout(() => {
      broadcastStatus.value = ''
      isBroadcastModalOpen.value = false
    }, 2500)
  } catch {
    broadcastStatus.value = 'Errore diffusione audio'
  }
}

onMounted(async () => {
  try {
    const list = await apiClient.get<MediaPlayer[]>('/api/v1/media/players')
    if (list && list.length > 0) players.value = list
  } catch {}
})
</script>

<template>
  <div class="media-view-container">
    <header class="media-header glass-panel">
      <div class="header-title-box">
        <div class="icon-circle">
          <Music :size="24" class="text-primary" />
        </div>
        <div>
          <h1 class="page-title">Audio Multi-room & Riproduzione Multimediale</h1>
          <p class="page-subtitle">Sincronizzazione speaker Google Cast / AirPlay 2 e diffusione annunci vocali Piper TTS</p>
        </div>
      </div>

      <button class="broadcast-trigger-btn" @click="isBroadcastModalOpen = true">
        <Bell :size="18" />
        <span>Diffondi Annuncio Vocale</span>
      </button>
    </header>

    <!-- Players Grid -->
    <div class="players-grid">
      <div v-for="p in players" :key="p.id" class="player-card glass-panel">
        <div class="card-head">
          <div class="player-meta">
            <Cast v-if="p.platform === 'cast'" :size="20" class="text-primary" />
            <Radio v-else-if="p.platform === 'airplay'" :size="20" class="text-warning" />
            <Music v-else :size="20" class="text-info" />
            <div>
              <h3 class="player-name">{{ p.name }}</h3>
              <span class="platform-badge">{{ p.platform.toUpperCase() }}</span>
            </div>
          </div>

          <span class="state-pill" :class="p.state">{{ p.state.toUpperCase() }}</span>
        </div>

        <!-- Track Info -->
        <div class="track-info-box">
          <span class="track-title">{{ p.current_title || 'Nessun brano in riproduzione' }}</span>
          <span class="track-artist">{{ p.current_artist || 'Speaker pronto' }}</span>
        </div>

        <!-- Controls -->
        <div class="player-controls-row">
          <button class="play-btn" @click="togglePlay(p)">
            <Pause v-if="p.state === 'playing'" :size="20" />
            <Play v-else :size="20" />
          </button>

          <div class="volume-slider-group">
            <Volume2 :size="16" class="vol-icon" />
            <input
              type="range"
              min="0"
              max="1"
              step="0.05"
              :value="p.volume"
              class="vol-slider"
              @input="handleVolumeChange(p, $event)"
            />
            <span class="vol-percent">{{ Math.round(p.volume * 100) }}%</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Announcement Modal -->
    <div v-if="isBroadcastModalOpen" class="modal-backdrop" @click="isBroadcastModalOpen = false">
      <div class="modal-dialog glass-panel" @click.stop>
        <div class="modal-top">
          <div class="modal-title-box">
            <Bell :size="20" class="text-warning" />
            <h3>Diffusione Vocale Sincronizzata (Tutte le Stanze)</h3>
          </div>
          <button class="close-btn" @click="isBroadcastModalOpen = false">✕</button>
        </div>

        <p class="modal-desc">
          Il messaggio verrà sintetizzato istantaneamente con Piper TTS e diffuso a volume prioritario interrompendo momentaneamente la musica in corso.
        </p>

        <textarea
          v-model="broadcastMessage"
          class="broadcast-textarea"
          rows="3"
          placeholder="Scrivi il messaggio vocale da pronunciare..."
        />

        <p v-if="broadcastStatus" class="status-text text-success">{{ broadcastStatus }}</p>

        <div class="modal-actions">
          <button class="btn-cancel" @click="isBroadcastModalOpen = false">Annulla</button>
          <button class="btn-send" @click="sendBroadcast">Diffondi Ora</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.media-view-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.media-header {
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
  background: rgba(168, 85, 247, 0.15);
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

.broadcast-trigger-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border-radius: 10px;
  background: linear-gradient(135deg, #f59e0b, #ef4444);
  color: #fff;
  border: none;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(245, 158, 11, 0.25);
}

.players-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 20px;
}

.player-card {
  padding: 20px;
  border-radius: 16px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.player-meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.player-name {
  font-size: 1rem;
  font-weight: 600;
  margin: 0;
}

.platform-badge {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.state-pill {
  font-size: 0.7rem;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 6px;
}

.state-pill.playing {
  background: rgba(16, 185, 129, 0.2);
  color: #10b981;
}

.state-pill.paused {
  background: rgba(245, 158, 11, 0.2);
  color: #f59e0b;
}

.state-pill.idle {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-muted);
}

.track-info-box {
  padding: 14px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.track-title {
  font-size: 0.95rem;
  font-weight: 600;
}

.track-artist {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.player-controls-row {
  display: flex;
  align-items: center;
  gap: 14px;
}

.play-btn {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--primary);
  border: none;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.volume-slider-group {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
}

.vol-slider {
  flex: 1;
  accent-color: var(--primary);
}

.vol-percent {
  font-size: 0.8rem;
  color: var(--text-muted);
  min-width: 32px;
}

.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 999;
}

.modal-dialog {
  width: 90%;
  max-width: 520px;
  background: var(--surface-card);
  border-radius: 16px;
  padding: 24px;
  border: 1px solid var(--border-color);
}

.modal-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.modal-title-box {
  display: flex;
  align-items: center;
  gap: 10px;
}

.modal-desc {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin-bottom: 16px;
}

.broadcast-textarea {
  width: 100%;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  color: #fff;
  padding: 12px;
  font-size: 0.9rem;
  box-sizing: border-box;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 16px;
}

.btn-cancel {
  background: transparent;
  border: 1px solid var(--border-color);
  color: #fff;
  padding: 8px 16px;
  border-radius: 8px;
  cursor: pointer;
}

.btn-send {
  background: var(--primary);
  border: none;
  color: #fff;
  padding: 8px 18px;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
}
</style>
