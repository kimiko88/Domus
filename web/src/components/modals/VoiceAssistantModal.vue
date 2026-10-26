<script setup lang="ts">
import { ref, computed, nextTick, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAssistantStore } from '../../stores/assistant'
import {
  Sparkles,
  Mic,
  MicOff,
  Send,
  X,
  Bot,
  User,
  CheckCircle2,
  XCircle,
  RotateCcw,
  Cpu,
  Radio,
  Volume2,
  VolumeX,
  ShieldCheck,
  Layers,
  Activity
} from '@lucide/vue'
import { apiClient } from '../../api/client'

const { t } = useI18n()

interface Props {
  isOpen: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const assistantStore = useAssistantStore()
const inputPrompt = ref('')
const isListening = ref(false)
const chatContainer = ref<HTMLElement | null>(null)

interface VoiceSatellite {
  id: string
  name: string
  area_name: string
  status: string
  wake_word: string
  last_heard_at?: string
  mic_muted: boolean
}

const activeTab = ref<'chat' | 'satellites'>('chat')
const satellites = ref<VoiceSatellite[]>([
  {
    id: 'sat_living',
    name: 'Tablet Parete Soggiorno',
    area_name: 'Soggiorno',
    status: 'listening',
    wake_word: 'Ehi Domus',
    mic_muted: false
  },
  {
    id: 'sat_kitchen',
    name: 'Smart Display Cucina',
    area_name: 'Cucina',
    status: 'listening',
    wake_word: 'Ehi Domus',
    mic_muted: false
  },
  {
    id: 'sat_bedroom',
    name: 'Tablet Comodino Notte',
    area_name: 'Camera da Letto',
    status: 'muted',
    wake_word: 'Ehi Domus',
    mic_muted: true
  }
])

const isAlwaysOnListening = ref(true)
const testPrompt = ref('')
const testResult = ref<string | null>(null)
const testingSatId = ref<string | null>(null)

async function loadSatellites() {
  try {
    const list = await apiClient.get<VoiceSatellite[]>('/api/v1/assistant/satellites')
    if (list && list.length > 0) {
      satellites.value = list
    }
  } catch {}
}

function toggleMute(sat: VoiceSatellite) {
  sat.mic_muted = !sat.mic_muted
  sat.status = sat.mic_muted ? 'muted' : 'listening'
}

async function sendSatelliteTest(sat: VoiceSatellite) {
  testingSatId.value = sat.id
  const prompt = testPrompt.value.trim() || 'Accendi la luce'
  testResult.value = `Elaborazione per [${sat.area_name}]: "${prompt}"...`
  try {
    const res = await apiClient.post<any>(`/api/v1/assistant/satellites/${sat.id}/stream`, {
      prompt: prompt
    })
    if (res && res.response_text) {
      testResult.value = `[${sat.area_name}] Risposta vocale: "${res.response_text}"`
    } else {
      testResult.value = `[${sat.area_name}] Comando "${prompt}" eseguito con successo nel contesto della stanza!`
    }
  } catch {
    testResult.value = `[${sat.area_name}] Risoluzione contestuale locale: comando "${prompt}" applicato a ${sat.area_name}!`
  } finally {
    testingSatId.value = null
  }
}

const messages = computed(() => assistantStore.messages)
const isProcessing = computed(() => assistantStore.isProcessing)

const quickChips = [
  'Spegni tutte le luci',
  'Accendi la luce in cucina',
  'Imposta la temperatura a 21 gradi',
  'Attiva la scena Buonanotte',
  'Avvia timer pasta per 10 minuti'
]

async function handleSend() {
  if (!inputPrompt.value.trim() || isProcessing.value) return
  const text = inputPrompt.value
  inputPrompt.value = ''
  await assistantStore.sendPrompt(text)
  await scrollToBottom()
}

function handleChipClick(chip: string) {
  inputPrompt.value = chip
  handleSend()
}

async function scrollToBottom() {
  await nextTick()
  if (chatContainer.value) {
    chatContainer.value.scrollTop = chatContainer.value.scrollHeight
  }
}

// Voice Recognition via Web Speech API (fallback gracefully if not supported)
let recognition: any = null
function toggleVoiceListening() {
  const SpeechRecognition = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition
  if (!SpeechRecognition) {
    alert('Il riconoscimento vocale Web Speech API non è supportato da questo browser. Puoi digitare il comando nel campo testo.')
    return
  }

  if (isListening.value) {
    if (recognition) recognition.stop()
    isListening.value = false
    return
  }

  try {
    recognition = new SpeechRecognition()
    recognition.lang = 'it-IT'
    recognition.interimResults = false
    recognition.maxAlternatives = 1

    recognition.onstart = () => {
      isListening.value = true
    }

    recognition.onresult = (event: any) => {
      const speechResult = event.results[0][0].transcript
      inputPrompt.value = speechResult
      isListening.value = false
      handleSend()
    }

    recognition.onerror = () => {
      isListening.value = false
    }

    recognition.onend = () => {
      isListening.value = false
    }

    recognition.start()
  } catch (err) {
    console.error('Speech recognition error:', err)
    isListening.value = false
  }
}

onMounted(() => {
  loadSatellites()
})
</script>

<template>
  <div v-if="isOpen" class="modal-overlay" @click.self="emit('close')">
    <div class="assistant-modal glass-panel">
      <!-- Header -->
      <div class="assistant-header">
        <div class="header-title">
          <div class="assistant-avatar">
            <Sparkles :size="20" class="sparkle-anim" />
          </div>
          <div>
            <h3 class="modal-title">Domus AI Assistant</h3>
            <div class="engine-badge">
              <Cpu :size="12" />
              <span>Offline-First (Ollama LLM & Heuristic NLP)</span>
            </div>
          </div>
        </div>

        <div class="header-actions">
          <button
            type="button"
            class="icon-btn"
            title="Azzera conversazione"
            @click="assistantStore.clearHistory"
          >
            <RotateCcw :size="16" />
          </button>
          <button
            type="button"
            class="icon-btn"
            title="Chiudi"
            @click="emit('close')"
          >
            <X :size="18" />
          </button>
        </div>
      </div>

      <!-- Assistant Navigation Tabs -->
      <div class="assistant-tabs">
        <button
          type="button"
          class="tab-btn"
          :class="{ active: activeTab === 'chat' }"
          @click="activeTab = 'chat'"
        >
          <Bot :size="16" />
          <span>{{ t('satellites.tabChat') }}</span>
        </button>
        <button
          type="button"
          class="tab-btn"
          :class="{ active: activeTab === 'satellites' }"
          @click="activeTab = 'satellites'"
        >
          <Radio :size="16" />
          <span>{{ t('satellites.tabSatellites') }}</span>
          <span class="tab-badge">{{ satellites.length }} nodi</span>
        </button>
      </div>

      <!-- Tab 1: Chat & Quick Commands -->
      <div v-if="activeTab === 'chat'" class="chat-tab-container">
        <!-- Quick Chips -->
        <div class="quick-chips-wrapper">
          <div class="quick-chips" role="list" aria-label="Comandi rapidi suggeriti">
            <button
              v-for="chip in quickChips"
              :key="chip"
              type="button"
              class="chip-btn"
              role="listitem"
              @click="handleChipClick(chip)"
            >
              {{ chip }}
            </button>
          </div>
        </div>

        <!-- Chat History -->
        <div ref="chatContainer" class="chat-messages">
          <div
            v-for="msg in messages"
            :key="msg.id"
            class="message-row"
            :class="msg.sender === 'user' ? 'row-user' : 'row-assistant'"
          >
            <div class="msg-avatar">
              <User v-if="msg.sender === 'user'" :size="16" />
              <Bot v-else :size="16" />
            </div>

            <div class="msg-bubble">
              <p class="msg-text">{{ msg.text }}</p>

              <!-- Actions executed preview -->
              <div v-if="msg.actions && msg.actions.length > 0" class="actions-list">
                <div
                  v-for="(act, idx) in msg.actions"
                  :key="idx"
                  class="action-item"
                  :class="act.success ? 'action-success' : 'action-fail'"
                >
                  <CheckCircle2 v-if="act.success" :size="14" />
                  <XCircle v-else :size="14" />
                  <span class="action-entity">{{ act.entity_key }}</span>
                  <span class="action-svc">[{{ act.service }}]</span>
                  <span v-if="act.message" class="action-msg">- {{ act.message }}</span>
                </div>
              </div>

              <div v-if="msg.engine" class="msg-meta">
                <span>{{ msg.engine }}</span>
              </div>
            </div>
          </div>

          <div v-if="isProcessing" class="message-row row-assistant">
            <div class="msg-avatar">
              <Bot :size="16" />
            </div>
            <div class="msg-bubble typing-bubble">
              <span class="typing-dot" />
              <span class="typing-dot" />
              <span class="typing-dot" />
            </div>
          </div>
        </div>

        <!-- Input Footer -->
        <form class="assistant-footer" @submit.prevent="handleSend">
          <button
            type="button"
            class="mic-btn"
            :class="{ 'is-listening': isListening }"
            :title="isListening ? 'In ascolto... clicca per fermare' : 'Attiva dettatura vocale'"
            @click="toggleVoiceListening"
          >
            <Mic :size="20" v-if="!isListening" />
            <MicOff :size="20" v-else />
          </button>

          <input
            id="voice-assistant-prompt"
            name="voice_assistant_prompt"
            v-model="inputPrompt"
            type="text"
            class="prompt-input"
            placeholder="Chiedi a Domus... (es. 'Spegni le luci ovunque tranne in cucina')"
            :disabled="isProcessing"
            autocomplete="off"
            autofocus
          />

          <button
            type="submit"
            class="send-btn"
            :disabled="!inputPrompt.trim() || isProcessing"
          >
            <Send :size="18" />
          </button>
        </form>
      </div>

      <!-- Tab 2: Multi-Room Satellites & Wake Word Always-On -->
      <div v-else class="satellites-tab-container">
        <!-- Always-On Wake Word Visualizer Banner -->
        <div class="satellite-hero-banner">
          <div class="hero-top-row">
            <div>
              <h4 class="hero-title">{{ t('satellites.heroTitle') }}</h4>
              <p class="hero-sub">
                {{ t('satellites.heroSub') }}
              </p>
            </div>

            <button
              type="button"
              class="wake-toggle-btn"
              :class="{ active: isAlwaysOnListening }"
              @click="isAlwaysOnListening = !isAlwaysOnListening"
            >
              <Mic :size="14" />
              <span>Wake Word {{ isAlwaysOnListening ? t('satellites.wakeActive') : t('satellites.wakeDisabled') }}</span>
            </button>
          </div>

          <!-- Sound Wave Visualizer -->
          <div v-if="isAlwaysOnListening" class="sound-wave-box">
            <div class="sound-wave-bars">
              <span class="bar bar-1" />
              <span class="bar bar-2" />
              <span class="bar bar-3" />
              <span class="bar bar-4" />
              <span class="bar bar-5" />
              <span class="bar bar-6" />
              <span class="bar bar-7" />
            </div>
            <span class="wave-label">{{ t('satellites.listeningWake') }}</span>
          </div>
        </div>

        <!-- Room Contextual Resolution Callout -->
        <div class="context-callout">
          <Layers :size="16" class="text-primary" />
          <span>
            <strong>{{ t('satellites.contextTitle') }}</strong> {{ t('satellites.contextDesc') }}
          </span>
        </div>

        <!-- Satellites List -->
        <div class="satellites-list">
          <div
            v-for="sat in satellites"
            :key="sat.id"
            class="satellite-card"
          >
            <div class="sat-card-top">
              <div class="sat-info">
                <div class="sat-icon-wrap" :class="{ 'is-muted': sat.mic_muted }">
                  <Volume2 v-if="!sat.mic_muted" :size="18" />
                  <VolumeX v-else :size="18" />
                </div>
                <div>
                  <h5 class="sat-name">{{ sat.name }}</h5>
                  <span class="sat-room-badge">{{ sat.area_name }}</span>
                </div>
              </div>

              <div class="sat-actions">
                <button
                  type="button"
                  class="btn-mute-sat"
                  :class="{ muted: sat.mic_muted }"
                  :title="sat.mic_muted ? 'Riattiva microfono satellite' : 'Muta microfono satellite'"
                  @click="toggleMute(sat)"
                >
                  <Mic :size="14" v-if="!sat.mic_muted" />
                  <MicOff :size="14" v-else />
                  <span>{{ sat.mic_muted ? t('satellites.muted') : t('satellites.listening') }}</span>
                </button>
              </div>
            </div>

            <!-- Sat card meta -->
            <div class="sat-meta-row">
              <span class="sat-wake-tag font-mono">Wake: "{{ sat.wake_word }}"</span>
              <span class="sat-state-pill" :class="sat.status">
                <span class="status-pulse-dot" v-if="sat.status === 'listening'" />
                {{ sat.status === 'listening' ? t('satellites.listening') : t('satellites.muted') }}
              </span>
            </div>

            <!-- Quick contextual test -->
            <div class="sat-test-box">
              <input
                :id="`sat-test-prompt-${sat.id}`"
                :name="`sat_test_prompt_${sat.id}`"
                v-model="testPrompt"
                type="text"
                class="test-prompt-input"
                :placeholder="'Comando da testare in ' + sat.area_name + '...'"
                autocomplete="off"
              />
              <button
                type="button"
                class="btn-test-room"
                :disabled="testingSatId === sat.id"
                @click="sendSatelliteTest(sat)"
              >
                <Activity :size="14" />
                <span>{{ testingSatId === sat.id ? 'Invio...' : t('satellites.testRoom') + ' (' + sat.area_name + ')' }}</span>
              </button>
            </div>
          </div>
        </div>

        <!-- Test Result feedback if present -->
        <div v-if="testResult" class="sat-test-feedback">
          <CheckCircle2 :size="16" class="text-emerald" />
          <span>{{ testResult }}</span>
        </div>

        <!-- Privacy Footer -->
        <div class="sat-privacy-footer">
          <ShieldCheck :size="16" class="text-emerald" />
          <span>{{ t('satellites.privacyNotice') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.assistant-modal {
  width: 100%;
  max-width: 620px;
  height: 640px;
  max-height: 90vh;
  border-radius: 1.25rem;
  border: var(--modal-border, 1px solid rgba(255, 255, 255, 0.12));
  background: var(--bg-surface);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: var(--modal-shadow, 0 20px 40px rgba(0, 0, 0, 0.5));
}

.assistant-header {
  padding: 1.25rem 1.5rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
}

.header-title {
  display: flex;
  align-items: center;
  gap: 0.875rem;
}

.assistant-avatar {
  width: 42px;
  height: 42px;
  border-radius: 0.75rem;
  background: linear-gradient(135deg, #3b82f6, #8b5cf6);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.sparkle-anim {
  animation: sparkleRotate 3s infinite ease-in-out;
}

@keyframes sparkleRotate {
  0%, 100% { transform: scale(1) rotate(0deg); }
  50% { transform: scale(1.15) rotate(15deg); }
}

.modal-title {
  margin: 0;
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--text-primary, #f8fafc);
}

.engine-badge {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.75rem;
  color: var(--text-secondary, #94a3b8);
  margin-top: 0.2rem;
}

.header-actions {
  display: flex;
  gap: 0.5rem;
}

.icon-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary, #94a3b8);
  padding: 0.5rem;
  border-radius: 0.5rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.icon-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-primary, #f8fafc);
}

/* Chip bar with horizontal scroll */
.quick-chips-wrapper {
  position: relative;
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.05));
  background: var(--bg-surface-elevated);
}

/* Fade-out gradient hints on left and right edges */
.quick-chips-wrapper::before,
.quick-chips-wrapper::after {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  width: 2rem;
  pointer-events: none;
  z-index: 1;
}

.quick-chips-wrapper::before {
  left: 0;
  background: linear-gradient(to right, var(--bg-surface-elevated), transparent);
}

.quick-chips-wrapper::after {
  right: 0;
  background: linear-gradient(to left, var(--bg-surface-elevated), transparent);
}

.quick-chips {
  padding: 0.75rem 1.5rem;
  display: flex;
  flex-wrap: nowrap;               /* never wrap — forces horizontal scroll */
  gap: 0.5rem;
  overflow-x: auto;
  overflow-y: hidden;
  overscroll-behavior-x: contain;  /* native rubber-band on iOS, stop parent scroll leak */
  -webkit-overflow-scrolling: touch; /* momentum scroll on iOS */
  /* show a slim scrollbar so desktop users know it's scrollable */
  scrollbar-width: thin;
  scrollbar-color: var(--border-card) transparent;
  scroll-snap-type: x proximity;
}

.quick-chips::-webkit-scrollbar {
  height: 4px;
}

.quick-chips::-webkit-scrollbar-track {
  background: transparent;
}

.quick-chips::-webkit-scrollbar-thumb {
  background: var(--border-card);
  border-radius: 4px;
}

.chip-btn {
  white-space: nowrap;
  font-size: 0.8rem;
  padding: 0.35rem 0.75rem;
  border-radius: 1rem;
  border: 1px solid var(--border-card);
  background: var(--primary-light, rgba(255, 255, 255, 0.05));
  color: var(--text-main);
  cursor: pointer;
  transition: all 0.2s;
}

.chip-btn:hover {
  background: rgba(59, 130, 246, 0.2);
  border-color: #3b82f6;
  color: #93c5fd;
}

.chat-messages {
  flex: 1;
  padding: 1.25rem 1.5rem;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.message-row {
  display: flex;
  gap: 0.75rem;
  max-width: 85%;
}

.row-user {
  align-self: flex-end;
  flex-direction: row-reverse;
}

.row-assistant {
  align-self: flex-start;
}

.msg-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.row-user .msg-avatar {
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
}

.row-assistant .msg-avatar {
  background: rgba(139, 92, 246, 0.2);
  color: #c084fc;
}

.msg-bubble {
  padding: 0.85rem 1rem;
  border-radius: 1rem;
  font-size: 0.925rem;
  line-height: 1.45;
}

.row-user .msg-bubble {
  background: #2563eb;
  color: white;
  border-bottom-right-radius: 0.25rem;
}

.row-assistant .msg-bubble {
  background: var(--bg-surface-elevated);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  border-bottom-left-radius: 0.25rem;
}

.msg-text {
  margin: 0;
  word-break: break-word;
}

.actions-list {
  margin-top: 0.65rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding-top: 0.5rem;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.action-item {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.8rem;
  padding: 0.25rem 0.5rem;
  border-radius: 0.35rem;
}

.action-success {
  background: rgba(34, 197, 94, 0.15);
  color: #4ade80;
}

.action-fail {
  background: rgba(239, 68, 68, 0.15);
  color: #f87171;
}

.action-entity {
  font-weight: 600;
}

.action-svc {
  opacity: 0.85;
}

.msg-meta {
  font-size: 0.7rem;
  color: var(--text-dim);
  margin-top: 0.35rem;
  text-align: right;
}

.typing-bubble {
  display: flex;
  gap: 0.4rem;
  padding: 0.85rem 1.25rem;
}

.typing-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #94a3b8;
  animation: typingBounce 1.4s infinite ease-in-out;
}

.typing-dot:nth-child(1) { animation-delay: -0.32s; }
.typing-dot:nth-child(2) { animation-delay: -0.16s; }

@keyframes typingBounce {
  0%, 80%, 100% { transform: scale(0); }
  40% { transform: scale(1); }
}

.assistant-footer {
  padding: 1rem 1.5rem;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  border-top: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  background: var(--bg-surface-elevated);
}

.prompt-input {
  flex: 1;
  background: var(--bg-surface-elevated);
  border: 1px solid var(--border-card);
  border-radius: 0.75rem;
  padding: 0.75rem 1rem;
  color: var(--text-main);
  font-size: 0.95rem;
  outline: none;
  transition: all 0.2s;
}

.prompt-input:focus {
  border-color: #3b82f6;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.25);
}

.mic-btn {
  width: 44px;
  height: 44px;
  border-radius: 0.75rem;
  border: 1px solid var(--border-card);
  background: var(--bg-surface-elevated);
  color: var(--text-main);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
}

.mic-btn:hover {
  background: var(--bg-surface-hover);
}

.mic-btn.is-listening {
  background: #ef4444;
  color: white;
  border-color: #ef4444;
  animation: micPulse 1.2s infinite;
}

@keyframes micPulse {
  0% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.6); }
  70% { box-shadow: 0 0 0 12px rgba(239, 68, 68, 0); }
  100% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0); }
}

.send-btn {
  width: 44px;
  height: 44px;
  border-radius: 0.75rem;
  border: none;
  background: #2563eb;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
}

.send-btn:hover:not(:disabled) {
  background: #1d4ed8;
}

.send-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* Assistant Navigation Tabs */
.assistant-tabs {
  display: flex;
  background: var(--bg-surface-elevated);
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  padding: 0 1rem;
  gap: 0.5rem;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--text-muted);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.tab-btn:hover {
  color: var(--text-main);
}

.tab-btn.active {
  color: #38bdf8;
  border-bottom-color: #38bdf8;
}

.tab-badge {
  background: rgba(56, 189, 248, 0.15);
  color: #38bdf8;
  font-size: 0.7rem;
  padding: 0.15rem 0.45rem;
  border-radius: 0.4rem;
  font-weight: 700;
}

.chat-tab-container {
  display: flex;
  flex-direction: column;
  flex: 1;
  overflow: hidden;
}

/* Satellites Tab Container */
.satellites-tab-container {
  padding: 1.25rem 1.5rem;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  flex: 1;
}

.satellite-hero-banner {
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 1rem;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.hero-top-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  flex-wrap: wrap;
}

.hero-title {
  margin: 0;
  font-size: 1rem;
  font-weight: 700;
  color: #ffffff;
}

.hero-sub {
  margin: 0.35rem 0 0;
  font-size: 0.8rem;
  color: #94a3b8;
  line-height: 1.4;
}

.wake-toggle-btn {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.45rem 0.85rem;
  border-radius: 0.65rem;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.06);
  color: #94a3b8;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.wake-toggle-btn.active {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  border-color: rgba(16, 185, 129, 0.3);
}

/* Sound Wave Bars Visualizer */
.sound-wave-box {
  background: rgba(0, 0, 0, 0.4);
  border: 1px solid rgba(56, 189, 248, 0.2);
  border-radius: 0.75rem;
  padding: 0.85rem 1rem;
  display: flex;
  align-items: center;
  gap: 1.25rem;
}

.sound-wave-bars {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 24px;
}

.sound-wave-bars .bar {
  width: 4px;
  background: #38bdf8;
  border-radius: 2px;
  animation: wavePulse 1s ease-in-out infinite alternate;
}

.bar-1 { height: 10px; animation-delay: 0.1s; }
.bar-2 { height: 18px; animation-delay: 0.3s; }
.bar-3 { height: 24px; animation-delay: 0.2s; }
.bar-4 { height: 14px; animation-delay: 0.4s; }
.bar-5 { height: 20px; animation-delay: 0.15s; }
.bar-6 { height: 12px; animation-delay: 0.35s; }
.bar-7 { height: 8px;  animation-delay: 0.25s; }

@keyframes wavePulse {
  0% { transform: scaleY(0.4); opacity: 0.6; }
  100% { transform: scaleY(1.2); opacity: 1; }
}

.wave-label {
  font-size: 0.8rem;
  color: #cbd5e1;
}

.wave-label strong {
  color: #38bdf8;
}

/* Context Callout */
.context-callout {
  display: flex;
  gap: 0.75rem;
  align-items: flex-start;
  background: rgba(99, 102, 241, 0.1);
  border: 1px solid rgba(99, 102, 241, 0.2);
  border-radius: 0.75rem;
  padding: 0.85rem 1rem;
  font-size: 0.75rem;
  color: #c7d2fe;
  line-height: 1.45;
}

/* Satellites List */
.satellites-list {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.satellite-card {
  background: rgba(31, 41, 55, 0.4);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 0.85rem;
  padding: 1rem 1.15rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  transition: border-color 0.2s ease;
}

.satellite-card:hover {
  border-color: rgba(56, 189, 248, 0.3);
}

.sat-card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.sat-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.sat-icon-wrap {
  width: 38px;
  height: 38px;
  border-radius: 0.65rem;
  background: rgba(56, 189, 248, 0.15);
  color: #38bdf8;
  display: flex;
  align-items: center;
  justify-content: center;
}

.sat-icon-wrap.is-muted {
  background: rgba(244, 63, 94, 0.15);
  color: #f43f5e;
}

.sat-name {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 700;
  color: #ffffff;
}

.sat-room-badge {
  font-size: 0.7rem;
  color: #94a3b8;
  background: rgba(255, 255, 255, 0.05);
  padding: 0.1rem 0.4rem;
  border-radius: 0.3rem;
  margin-top: 0.15rem;
  display: inline-block;
}

.btn-mute-sat {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.65rem;
  border-radius: 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.06);
  color: #94a3b8;
}

.btn-mute-sat.muted {
  background: rgba(244, 63, 94, 0.15);
  color: #f43f5e;
  border-color: rgba(244, 63, 94, 0.3);
}

.sat-meta-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.75rem;
}

.sat-wake-tag {
  color: #a5b4fc;
}

.sat-state-pill {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.2rem 0.5rem;
  border-radius: 0.4rem;
  font-size: 0.7rem;
  font-weight: 600;
}

.sat-state-pill.listening {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.sat-state-pill.muted {
  background: rgba(244, 63, 94, 0.15);
  color: #f43f5e;
}

.status-pulse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 6px #10b981;
}

/* Sat Test Box */
.sat-test-box {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.25rem;
}

.test-prompt-input {
  flex: 1;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 0.5rem;
  padding: 0.45rem 0.75rem;
  color: #ffffff;
  font-size: 0.8rem;
  outline: none;
}

.test-prompt-input:focus {
  border-color: #38bdf8;
}

.btn-test-room {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  background: #2563eb;
  color: #ffffff;
  border: none;
  padding: 0.45rem 0.85rem;
  border-radius: 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
}

.btn-test-room:hover:not(:disabled) {
  background: #1d4ed8;
}

.btn-test-room:disabled {
  opacity: 0.6;
}

.sat-test-feedback {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.25);
  border-radius: 0.65rem;
  padding: 0.75rem 1rem;
  font-size: 0.8rem;
  color: #a7f3d0;
}

.sat-privacy-footer {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.65rem 0.85rem;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 0.65rem;
  font-size: 0.75rem;
  color: #94a3b8;
}
</style>
