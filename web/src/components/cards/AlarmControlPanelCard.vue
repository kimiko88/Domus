<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAlarmStore } from '../../stores/alarm'
import {
  Shield,
  ShieldCheck,
  ShieldAlert,
  Moon,
  Home,
  LogOut,
  Delete,
  RotateCcw,
  AlertTriangle,
  Siren,
  Bell
} from '@lucide/vue'
import BaseButton from '../ui/BaseButton.vue'

const alarmStore = useAlarmStore()

const pinInput = ref<string>('')
const actionError = ref<string | null>(null)
const actionSuccess = ref<string | null>(null)

onMounted(async () => {
  await alarmStore.fetchPanel()
})

const panel = computed(() => alarmStore.panel)
const currentState = computed(() => panel.value?.state || 'disarmed')

const isTriggered = computed(() => currentState.value === 'triggered')
const isPending = computed(() => currentState.value === 'pending')

function appendDigit(digit: string) {
  if (pinInput.value.length < 8) {
    pinInput.value += digit
  }
}

function clearPin() {
  pinInput.value = ''
}

function backspacePin() {
  pinInput.value = pinInput.value.slice(0, -1)
}

async function handleArm(mode: 'armed_home' | 'armed_away' | 'armed_night') {
  actionError.value = null
  actionSuccess.value = null
  try {
    await alarmStore.arm(mode, pinInput.value)
    pinInput.value = ''
    actionSuccess.value = `Allarme armato in modalità ${getModeLabel(mode)}`
    setTimeout(() => { actionSuccess.value = null }, 3000)
  } catch (err: any) {
    actionError.value = err?.message || 'Errore durante l\'inserimento allarme'
  }
}

async function handleDisarm() {
  actionError.value = null
  actionSuccess.value = null
  if (!pinInput.value) {
    actionError.value = 'Inserisci il PIN per disarmare'
    return
  }
  try {
    await alarmStore.disarm(pinInput.value)
    pinInput.value = ''
    actionSuccess.value = 'Allarme disarmato con successo'
    setTimeout(() => { actionSuccess.value = null }, 3000)
  } catch (err: any) {
    actionError.value = err?.message || 'PIN errato o non valido'
  }
}

async function handlePanic() {
  if (!confirm('ATTENZIONE: Stai per innescare l\'allarme SOS manuale con sirene. Confermi?')) {
    return
  }
  try {
    await alarmStore.trigger('SOS Manuale da Pannello')
  } catch (err: any) {
    actionError.value = err?.message || 'Errore innesco SOS'
  }
}

function getModeLabel(state: string): string {
  switch (state) {
    case 'disarmed': return 'Disarmato'
    case 'armed_home': return 'In Casa'
    case 'armed_away': return 'Fuori Casa'
    case 'armed_night': return 'Notte'
    case 'pending': return 'Ritardo Uscita/Ingresso'
    case 'triggered': return 'ALLARME IN CORSO!'
    default: return state
  }
}

const statusClass = computed(() => {
  switch (currentState.value) {
    case 'disarmed': return 'status-disarmed'
    case 'armed_home': return 'status-home'
    case 'armed_away': return 'status-away'
    case 'armed_night': return 'status-night'
    case 'pending': return 'status-pending'
    case 'triggered': return 'status-triggered'
    default: return ''
  }
})
</script>

<template>
  <div class="alarm-panel-card glass-card" :class="{ 'is-alarm-triggered': isTriggered }">
    <!-- Header with State Badge -->
    <div class="alarm-header">
      <div class="alarm-title-group">
        <div class="alarm-icon-box" :class="statusClass">
          <ShieldAlert v-if="isTriggered" :size="24" class="pulse-icon" />
          <ShieldCheck v-else-if="currentState === 'disarmed'" :size="24" />
          <Shield v-else :size="24" />
        </div>
        <div>
          <h3 class="panel-name">{{ panel?.name || 'Sistema Antifurto' }}</h3>
          <p class="panel-sensors-info">
            Perimetro: {{ panel?.perimeter_sensors?.length || 0 }} sensori | Interno: {{ panel?.interior_sensors?.length || 0 }} sensori
          </p>
        </div>
      </div>

      <div class="state-pill" :class="statusClass">
        <span class="state-dot" />
        <span class="state-text">{{ getModeLabel(currentState) }}</span>
      </div>
    </div>

    <!-- Alarm Triggered Alert -->
    <div v-if="isTriggered" class="triggered-banner">
      <Siren :size="22" class="siren-animate" />
      <div class="triggered-info">
        <strong>ALLARME ATTIVO - SIRENE ACCESE!</strong>
        <p>Causa: {{ panel?.last_triggered_reason || 'Rilevamento intrusione' }}</p>
      </div>
    </div>

    <!-- Delay Countdown Banner -->
    <div v-if="isPending" class="pending-banner">
      <Bell :size="20" class="pulse-icon" />
      <span>Ritardo di attivazione / disattivazione in corso... Inserisci il PIN!</span>
    </div>

    <!-- Feedback alerts -->
    <div v-if="actionError" class="alert alert-error">
      <AlertTriangle :size="16" />
      <span>{{ actionError }}</span>
    </div>
    <div v-if="actionSuccess" class="alert alert-success">
      <ShieldCheck :size="16" />
      <span>{{ actionSuccess }}</span>
    </div>

    <!-- PIN Display Dots -->
    <div class="pin-display-container">
      <label class="pin-label">Codice PIN Tastierino</label>
      <div class="pin-dots">
        <span
          v-for="i in 6"
          :key="i"
          class="pin-dot"
          :class="{ filled: i <= pinInput.length }"
        />
      </div>
    </div>

    <!-- Numeric Keypad -->
    <div class="keypad-grid">
      <button v-for="d in ['1','2','3','4','5','6','7','8','9']" :key="d" type="button" class="key-btn" @click="appendDigit(d)">
        {{ d }}
      </button>
      <button type="button" class="key-btn key-action" title="Azzera" @click="clearPin">
        <RotateCcw :size="18" />
      </button>
      <button type="button" class="key-btn" @click="appendDigit('0')">
        0
      </button>
      <button type="button" class="key-btn key-action" title="Cancella" @click="backspacePin">
        <Delete :size="18" />
      </button>
    </div>

    <!-- Arm / Disarm Action Buttons -->
    <div class="actions-grid">
      <BaseButton
        variant="primary"
        size="md"
        class="arm-btn btn-disarm"
        :disabled="currentState === 'disarmed'"
        @click="handleDisarm"
      >
        <ShieldCheck :size="18" />
        Disarma
      </BaseButton>

      <BaseButton
        variant="primary"
        size="md"
        class="arm-btn btn-home"
        :disabled="currentState === 'armed_home'"
        @click="handleArm('armed_home')"
      >
        <Home :size="18" />
        In Casa
      </BaseButton>

      <BaseButton
        variant="primary"
        size="md"
        class="arm-btn btn-away"
        :disabled="currentState === 'armed_away'"
        @click="handleArm('armed_away')"
      >
        <LogOut :size="18" />
        Fuori Casa
      </BaseButton>

      <BaseButton
        variant="primary"
        size="md"
        class="arm-btn btn-night"
        :disabled="currentState === 'armed_night'"
        @click="handleArm('armed_night')"
      >
        <Moon :size="18" />
        Notte
      </BaseButton>
    </div>

    <!-- SOS Panic Button -->
    <div class="panic-container">
      <button type="button" class="panic-btn" @click="handlePanic">
        <Siren :size="18" />
        SOS Antipallottola / Panico Immediato
      </button>
    </div>
  </div>
</template>

<style scoped>
.alarm-panel-card {
  padding: 1.5rem;
  border-radius: 1rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  background: var(--card-bg, rgba(20, 24, 33, 0.75));
  backdrop-filter: blur(16px);
  transition: all 0.3s ease;
}

.is-alarm-triggered {
  border-color: #ef4444;
  box-shadow: 0 0 25px rgba(239, 68, 68, 0.35);
  animation: alarmPulse 1.5s infinite;
}

@keyframes alarmPulse {
  0% { box-shadow: 0 0 10px rgba(239, 68, 68, 0.2); }
  50% { box-shadow: 0 0 35px rgba(239, 68, 68, 0.6); }
  100% { box-shadow: 0 0 10px rgba(239, 68, 68, 0.2); }
}

.alarm-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
}

.alarm-title-group {
  display: flex;
  align-items: center;
  gap: 0.875rem;
}

.alarm-icon-box {
  width: 48px;
  height: 48px;
  border-radius: 0.75rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.status-disarmed { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
.status-home { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.status-away { background: rgba(249, 115, 22, 0.15); color: #f97316; }
.status-night { background: rgba(168, 85, 247, 0.15); color: #a855f7; }
.status-pending { background: rgba(234, 179, 8, 0.15); color: #eab308; }
.status-triggered { background: rgba(239, 68, 68, 0.25); color: #ef4444; }

.panel-name {
  font-size: 1.15rem;
  font-weight: 600;
  margin: 0;
  color: var(--text-primary, #f8fafc);
}

.panel-sensors-info {
  font-size: 0.8rem;
  color: var(--text-secondary, #94a3b8);
  margin: 0.2rem 0 0 0;
}

.state-pill {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.35rem 0.85rem;
  border-radius: 2rem;
  font-size: 0.825rem;
  font-weight: 600;
  border: 1px solid currentColor;
}

.state-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}

.triggered-banner {
  display: flex;
  align-items: center;
  gap: 0.875rem;
  padding: 0.875rem 1rem;
  background: rgba(239, 68, 68, 0.2);
  border: 1px solid #ef4444;
  border-radius: 0.75rem;
  color: #fca5a5;
}

.siren-animate {
  animation: sirenSpin 1s infinite alternate;
  color: #ef4444;
}

@keyframes sirenSpin {
  0% { transform: scale(1) rotate(-10deg); }
  100% { transform: scale(1.2) rotate(10deg); }
}

.pending-banner {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  background: rgba(234, 179, 8, 0.15);
  border: 1px solid #eab308;
  border-radius: 0.75rem;
  color: #fef08a;
  font-size: 0.875rem;
}

.pulse-icon {
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.alert {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.65rem 0.875rem;
  border-radius: 0.5rem;
  font-size: 0.85rem;
}

.alert-error {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.alert-success {
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
  border: 1px solid rgba(34, 197, 94, 0.3);
}

.pin-display-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.pin-label {
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-secondary, #94a3b8);
}

.pin-dots {
  display: flex;
  gap: 0.75rem;
}

.pin-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 2px solid var(--border-color, rgba(255, 255, 255, 0.3));
  background: transparent;
  transition: all 0.2s;
}

.pin-dot.filled {
  background: #3b82f6;
  border-color: #3b82f6;
  box-shadow: 0 0 10px rgba(59, 130, 246, 0.5);
  transform: scale(1.15);
}

.keypad-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.65rem;
  max-width: 260px;
  margin: 0 auto;
  width: 100%;
}

.key-btn {
  height: 52px;
  font-size: 1.25rem;
  font-weight: 700;
  border-radius: 0.75rem;
  border: 1px solid var(--border-card);
  background: var(--bg-surface-elevated);
  color: var(--text-main);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: var(--shadow-sm);
  transition: all 0.15s ease;
}

.key-btn:hover {
  background: var(--bg-surface-hover);
  border-color: var(--primary);
  color: var(--primary);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.key-btn:active {
  transform: translateY(1px) scale(0.97);
}

.key-action {
  color: var(--text-muted);
}

.key-action:hover {
  color: var(--primary);
}

.actions-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.75rem;
}

.arm-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.75rem;
  font-weight: 600;
  border-radius: 0.75rem;
}

.btn-disarm {
  background: #16a34a !important;
  color: white !important;
}

.btn-home {
  background: #2563eb !important;
  color: white !important;
}

.btn-away {
  background: #d97706 !important;
  color: white !important;
}

.btn-night {
  background: #9333ea !important;
  color: white !important;
}

.panic-container {
  display: flex;
  justify-content: center;
}

.panic-btn {
  width: 100%;
  padding: 0.75rem 1rem;
  border-radius: 0.75rem;
  border: 1px solid #ef4444;
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  font-weight: 600;
  font-size: 0.85rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  transition: all 0.2s;
}

.panic-btn:hover {
  background: #ef4444;
  color: white;
}
</style>
