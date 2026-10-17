<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Volume2,
  VolumeX,
  Play,
  Pause,
  Cast,
  Radio,
  Bell,
  Music,
  Plus,
  Trash2,
  Speaker,
  X,
  CheckCircle,
  AlertCircle
} from '@lucide/vue'
import { apiClient } from '../api/client'

const { t } = useI18n()

interface MediaPlayer {
  id: string
  name: string
  platform: 'cast' | 'airplay' | 'dlna' | 'local'
  state: 'idle' | 'playing' | 'paused'
  volume: number
  muted: boolean
  current_title?: string
  current_artist?: string
  area_id?: string
}

const players = ref<MediaPlayer[]>([])
const loading = ref(false)

// Broadcast Modal
const isBroadcastModalOpen = ref(false)
const broadcastMessage = ref('Attenzione: Qualcuno ha suonato al campanello!')
const broadcastStatus = ref('')

// Add Player Modal
const isAddModalOpen = ref(false)
const newPlayerName = ref('')
const newPlayerPlatform = ref<'cast' | 'airplay' | 'dlna' | 'local'>('cast')
const newPlayerArea = ref('salotto')
const addError = ref('')
const addSuccess = ref('')

async function fetchPlayers() {
  loading.value = true
  try {
    const list = await apiClient.get<MediaPlayer[]>('/api/v1/media/players')
    players.value = list || []
  } catch {
    players.value = []
  } finally {
    loading.value = false
  }
}

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
    broadcastStatus.value = 'Annuncio diffuso con successo su tutti gli altoparlanti!'
    setTimeout(() => {
      broadcastStatus.value = ''
      isBroadcastModalOpen.value = false
    }, 2000)
  } catch {
    broadcastStatus.value = 'Errore durante la diffusione audio'
  }
}

async function createPlayer() {
  if (!newPlayerName.value.trim()) {
    addError.value = 'Inserisci un nome per il media player'
    return
  }
  addError.value = ''
  try {
    await apiClient.post('/api/v1/media/players', {
      name: newPlayerName.value.trim(),
      platform: newPlayerPlatform.value,
      area_id: newPlayerArea.value ? newPlayerArea.value : undefined
    })
    addSuccess.value = 'Altoparlante registrato con successo nel database!'
    newPlayerName.value = ''
    await fetchPlayers()
    setTimeout(() => {
      addSuccess.value = ''
      isAddModalOpen.value = false
    }, 1200)
  } catch (err: any) {
    addError.value = err?.message || 'Errore nella registrazione del player'
  }
}

async function deletePlayer(id: string, name: string) {
  if (!confirm(`Sei sicuro di voler eliminare "${name}" dal database di Domus?`)) {
    return
  }
  try {
    await apiClient.delete(`/api/v1/media/players/${id}`)
    players.value = players.value.filter(p => p.id !== id)
  } catch (err) {
    console.error('Failed to delete media player:', err)
  }
}

onMounted(() => {
  fetchPlayers()
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
          <h1 class="page-title">{{ t('media.title') }}</h1>
          <p class="page-subtitle">{{ t('media.subtitle') }}</p>
        </div>
      </div>

      <div class="header-actions">
        <button class="add-player-btn" @click="isAddModalOpen = true">
          <Plus :size="18" />
          <span>Aggiungi Player</span>
        </button>

        <button class="broadcast-trigger-btn" @click="isBroadcastModalOpen = true">
          <Bell :size="18" />
          <span>Diffondi Annuncio Vocale</span>
        </button>
      </div>
    </header>

    <!-- Players Grid -->
    <div v-if="players.length === 0" class="empty-players glass-panel">
      <Speaker :size="48" class="empty-icon text-muted" />
      <p class="empty-state">Nessun media player registrato nel database.</p>
      <p class="empty-sub">Registra i tuoi altoparlanti Apple AirPlay, Google Cast, ricevitori DLNA o altoparlanti locali.</p>
      <button class="btn-primary add-first-btn" @click="isAddModalOpen = true">
        <Plus :size="18" />
        <span>Registra Nuovo Media Player</span>
      </button>
    </div>

    <div v-else class="players-grid">
      <div v-for="p in players" :key="p.id" class="player-card glass-panel">
        <div class="card-head">
          <div class="player-meta">
            <Cast v-if="p.platform === 'cast'" :size="22" class="text-primary" />
            <Radio v-else-if="p.platform === 'airplay'" :size="22" class="text-warning" />
            <Speaker v-else-if="p.platform === 'local'" :size="22" class="text-success" />
            <Music v-else :size="22" class="text-info" />
            <div>
              <h3 class="player-name">{{ p.name }}</h3>
              <span class="platform-badge">{{ p.platform.toUpperCase() }}</span>
            </div>
          </div>

          <div class="head-right">
            <span class="state-pill" :class="p.state">{{ p.state.toUpperCase() }}</span>
            <button
              class="delete-player-btn"
              title="Elimina dal database"
              aria-label="Elimina player"
              @click="deletePlayer(p.id, p.name)"
            >
              <Trash2 :size="16" />
            </button>
          </div>
        </div>

        <!-- Track Info -->
        <div class="track-info-box">
          <span class="track-title">{{ p.current_title || 'Nessun brano in riproduzione' }}</span>
          <span class="track-artist">{{ p.current_artist || 'Speaker connesso e pronto' }}</span>
        </div>

        <!-- Controls -->
        <div class="player-controls-row">
          <button class="play-btn" :title="p.state === 'playing' ? 'Metti in pausa' : 'Riproduci'" @click="togglePlay(p)">
            <Pause v-if="p.state === 'playing'" :size="20" />
            <Play v-else :size="20" />
          </button>

          <div class="volume-slider-group">
            <Volume2 :size="18" class="vol-icon" />
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
      <div class="modal-dialog" @click.stop>
        <div class="modal-top">
          <div class="modal-title-box">
            <div class="modal-icon-badge warning">
              <Bell :size="20" />
            </div>
            <h3>Diffusione Vocale Sincronizzata</h3>
          </div>
          <button class="close-btn" aria-label="Chiudi finestra" @click="isBroadcastModalOpen = false">
            <X :size="18" />
          </button>
        </div>

        <p class="modal-desc">
          Il messaggio verrà sintetizzato istantaneamente con Piper TTS e diffuso a volume prioritario su tutti gli altoparlanti, interrompendo momentaneamente la musica in corso.
        </p>

        <div class="form-group">
          <label class="form-label" for="broadcast-text">Messaggio vocale da diffondere:</label>
          <textarea
            id="broadcast-text"
            v-model="broadcastMessage"
            class="broadcast-textarea"
            rows="3"
            placeholder="Scrivi il messaggio vocale da pronunciare..."
          />
        </div>

        <div v-if="broadcastStatus" class="status-banner success">
          <CheckCircle :size="18" />
          <span>{{ broadcastStatus }}</span>
        </div>

        <div class="modal-actions">
          <button class="btn-cancel" @click="isBroadcastModalOpen = false">Annulla</button>
          <button class="btn-send" @click="sendBroadcast">Diffondi Ora</button>
        </div>
      </div>
    </div>

    <!-- Add Media Player Modal -->
    <div v-if="isAddModalOpen" class="modal-backdrop" @click="isAddModalOpen = false">
      <div class="modal-dialog" @click.stop>
        <div class="modal-top">
          <div class="modal-title-box">
            <div class="modal-icon-badge primary">
              <Plus :size="20" />
            </div>
            <h3>Registra Media Player nel Database</h3>
          </div>
          <button class="close-btn" aria-label="Chiudi finestra" @click="isAddModalOpen = false">
            <X :size="18" />
          </button>
        </div>

        <p class="modal-desc">
          Aggiunge una nuova periferica audio alla tabella <code>media_players</code> di SQLite per la gestione multi-room e notifiche vocali.
        </p>

        <form class="modal-form" @submit.prevent="createPlayer">
          <div class="form-group">
            <label class="form-label" for="player-name">Nome Altoparlante / Speaker *</label>
            <input
              id="player-name"
              v-model="newPlayerName"
              type="text"
              class="form-input"
              placeholder="es. Apple HomePod Mini Cucina, Sonos Salotto"
              required
            />
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label" for="player-platform">Piattaforma / Protocollo</label>
              <select id="player-platform" v-model="newPlayerPlatform" class="form-select">
                <option value="cast">Google Cast (Chromecast / Nest)</option>
                <option value="airplay">Apple AirPlay 2 (HomePod / Ricevitori)</option>
                <option value="dlna">DLNA / UPnP (Hi-Fi / TV)</option>
                <option value="local">Altoparlante Locale Hub (Jack / USB / DAC)</option>
              </select>
            </div>

            <div class="form-group">
              <label class="form-label" for="player-area">Stanza / Area</label>
              <select id="player-area" v-model="newPlayerArea" class="form-select">
                <option value="salotto">Salotto</option>
                <option value="cucina">Cucina</option>
                <option value="camera">Camera da Letto</option>
                <option value="studio">Studio</option>
                <option value="bagno">Bagno</option>
                <option value="giardino">Giardino</option>
              </select>
            </div>
          </div>

          <div v-if="addError" class="status-banner error">
            <AlertCircle :size="18" />
            <span>{{ addError }}</span>
          </div>

          <div v-if="addSuccess" class="status-banner success">
            <CheckCircle :size="18" />
            <span>{{ addSuccess }}</span>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn-cancel" @click="isAddModalOpen = false">Annulla</button>
            <button type="submit" class="btn-primary">Salva nel Database</button>
          </div>
        </form>
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
  padding: 22px 28px;
  border-radius: var(--radius-xl, 20px);
  background: var(--bg-surface);
  border: 1.5px solid var(--border-card);
  box-shadow: var(--shadow-sm);
  flex-wrap: wrap;
  gap: 16px;
}

.header-title-box {
  display: flex;
  align-items: center;
  gap: 16px;
}

.icon-circle {
  width: 50px;
  height: 50px;
  border-radius: 14px;
  background: rgba(14, 165, 233, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-title {
  font-size: 1.45rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-main);
}

.page-subtitle {
  font-size: 0.88rem;
  color: var(--text-muted);
  margin: 4px 0 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.add-player-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border-radius: var(--radius-md, 10px);
  background: var(--primary);
  color: #ffffff;
  border: none;
  font-weight: 600;
  font-size: 0.92rem;
  cursor: pointer;
  box-shadow: 0 4px 14px var(--primary-glow);
  transition: all var(--transition-fast);
}

.add-player-btn:hover {
  filter: brightness(1.1);
  transform: translateY(-1px);
}

.broadcast-trigger-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border-radius: var(--radius-md, 10px);
  background: linear-gradient(135deg, #f59e0b, #ef4444);
  color: #ffffff;
  border: none;
  font-weight: 600;
  font-size: 0.92rem;
  cursor: pointer;
  box-shadow: 0 4px 14px rgba(245, 158, 11, 0.3);
  transition: all var(--transition-fast);
}

.broadcast-trigger-btn:hover {
  filter: brightness(1.1);
  transform: translateY(-1px);
}

.empty-players {
  padding: 60px 24px;
  text-align: center;
  border-radius: var(--radius-xl, 20px);
  background: var(--bg-surface);
  border: 1.5px dashed var(--border-card);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.empty-icon {
  opacity: 0.5;
  margin-bottom: 8px;
}

.empty-state {
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--text-main);
}

.empty-sub {
  font-size: 0.9rem;
  color: var(--text-muted);
  max-width: 480px;
}

.add-first-btn {
  margin-top: 10px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
}

.players-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 20px;
}

.player-card {
  padding: 22px;
  border-radius: var(--radius-xl, 20px);
  background: var(--bg-surface);
  border: 1.5px solid var(--border-card);
  box-shadow: var(--shadow-sm);
  display: flex;
  flex-direction: column;
  gap: 16px;
  transition: transform var(--transition-normal), box-shadow var(--transition-normal);
}

.player-card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.player-meta {
  display: flex;
  align-items: center;
  gap: 14px;
}

.player-name {
  font-size: 1.05rem;
  font-weight: 600;
  margin: 0;
  color: var(--text-main);
}

.platform-badge {
  font-size: 0.72rem;
  font-weight: 600;
  color: var(--primary);
  background: rgba(14, 165, 233, 0.12);
  padding: 2px 6px;
  border-radius: 4px;
  letter-spacing: 0.05em;
}

.head-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.state-pill {
  font-size: 0.72rem;
  font-weight: 700;
  padding: 3px 9px;
  border-radius: var(--radius-full);
}

.state-pill.playing {
  background: rgba(16, 185, 129, 0.18);
  color: #10b981;
  border: 1px solid rgba(16, 185, 129, 0.35);
}

.state-pill.paused {
  background: rgba(245, 158, 11, 0.18);
  color: #f59e0b;
  border: 1px solid rgba(245, 158, 11, 0.35);
}

.state-pill.idle {
  background: rgba(148, 163, 184, 0.15);
  color: var(--text-muted);
  border: 1px solid var(--border-subtle);
}

.delete-player-btn {
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-dim);
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
  cursor: pointer;
}

.delete-player-btn:hover {
  background: rgba(244, 63, 94, 0.15);
  border-color: rgba(244, 63, 94, 0.3);
  color: var(--color-danger);
}

.track-info-box {
  padding: 14px 16px;
  background: var(--bg-surface-elevated, rgba(255, 255, 255, 0.04));
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md, 10px);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.track-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-main);
}

.track-artist {
  font-size: 0.82rem;
  color: var(--text-muted);
}

.player-controls-row {
  display: flex;
  align-items: center;
  gap: 16px;
}

.play-btn {
  width: 42px;
  height: 42px;
  border-radius: var(--radius-full);
  background: var(--primary);
  border: none;
  color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 4px 12px var(--primary-glow);
  transition: transform var(--transition-fast), filter var(--transition-fast);
}

.play-btn:hover {
  filter: brightness(1.12);
  transform: scale(1.05);
}

.volume-slider-group {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 10px;
}

.vol-icon {
  color: var(--text-muted);
}

.vol-slider {
  flex: 1;
  accent-color: var(--primary);
  height: 6px;
  border-radius: 3px;
  cursor: pointer;
}

.vol-percent {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--text-muted);
  min-width: 36px;
  text-align: right;
}

/* ── Modal Dialogs ─────────────────────────────────────────────────────── */
.modal-dialog {
  width: 95%;
  max-width: 540px;
  padding: 28px;
  box-sizing: border-box;
}

.modal-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}

.modal-title-box {
  display: flex;
  align-items: center;
  gap: 12px;
}

.modal-title-box h3 {
  font-size: 1.2rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-main);
}

.modal-icon-badge {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-icon-badge.warning {
  background: rgba(245, 158, 11, 0.18);
  color: #f59e0b;
  border: 1px solid rgba(245, 158, 11, 0.35);
}

.modal-icon-badge.primary {
  background: rgba(14, 165, 233, 0.18);
  color: var(--primary);
  border: 1px solid rgba(14, 165, 233, 0.35);
}

.modal-desc {
  font-size: 0.92rem;
  color: var(--text-main);
  opacity: 0.9;
  line-height: 1.5;
  margin-bottom: 20px;
}

.modal-desc code {
  background: rgba(255, 255, 255, 0.1);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.85em;
  color: var(--primary);
}

.modal-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.form-label {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-main);
}

.form-input,
.form-select,
.broadcast-textarea {
  width: 100%;
  background: var(--bg-surface-elevated, #1e293b);
  border: 1.5px solid var(--border-card);
  border-radius: var(--radius-md, 10px);
  color: var(--text-main);
  padding: 12px 14px;
  font-size: 0.92rem;
  box-sizing: border-box;
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
}

.form-input:focus,
.form-select:focus,
.broadcast-textarea:focus {
  border-color: var(--primary);
  outline: none;
  box-shadow: 0 0 0 3px rgba(14, 165, 233, 0.25);
}

.broadcast-textarea {
  resize: vertical;
  min-height: 85px;
}

.status-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-radius: var(--radius-md);
  font-size: 0.88rem;
  font-weight: 500;
  margin-top: 10px;
}

.status-banner.success {
  background: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.4);
  color: #10b981;
}

.status-banner.error {
  background: rgba(244, 63, 94, 0.15);
  border: 1px solid rgba(244, 63, 94, 0.4);
  color: var(--color-danger);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 22px;
}

.btn-cancel {
  background: transparent;
  border: 1.5px solid var(--border-card);
  color: var(--text-main);
  padding: 10px 18px;
  border-radius: var(--radius-md);
  font-weight: 600;
  font-size: 0.92rem;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-cancel:hover {
  background: rgba(255, 255, 255, 0.08);
}

.btn-send,
.btn-primary {
  background: var(--primary);
  border: none;
  color: #ffffff;
  padding: 10px 20px;
  border-radius: var(--radius-md);
  font-weight: 600;
  font-size: 0.92rem;
  cursor: pointer;
  box-shadow: 0 4px 12px var(--primary-glow);
  transition: all var(--transition-fast);
}

.btn-send:hover,
.btn-primary:hover {
  filter: brightness(1.1);
  transform: translateY(-1px);
}
</style>
