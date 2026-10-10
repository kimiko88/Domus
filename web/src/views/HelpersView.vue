<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Sliders,
  ToggleLeft,
  Hash,
  Timer,
  Play,
  Pause,
  RotateCcw,
  Plus,
  Trash2,
  CheckCircle,
  Sparkles,
  Thermometer
} from '@lucide/vue'
import { apiClient } from '../api/client'

const { t } = useI18n()

interface Helper {
  id: string
  entity_key: string
  name: string
  domain: 'input_boolean' | 'input_number' | 'timer' | 'climate_room_compensator'
  state: string
  config_json: string
}

const helpers = ref<Helper[]>([
  {
    id: 'h_bool_1',
    entity_key: 'input_boolean.guest_mode',
    name: 'Modalità Ospiti',
    domain: 'input_boolean',
    state: 'off',
    config_json: '{}'
  },
  {
    id: 'h_bool_2',
    entity_key: 'input_boolean.vacation_mode',
    name: 'Modalità Vacanza',
    domain: 'input_boolean',
    state: 'off',
    config_json: '{}'
  },
  {
    id: 'h_num_1',
    entity_key: 'input_number.target_climate',
    name: 'Soglia Riscaldamento',
    domain: 'input_number',
    state: '20.50',
    config_json: '{"min": 15, "max": 25, "step": 0.5, "unit": "°C"}'
  },
  {
    id: 'h_timer_1',
    entity_key: 'timer.irrigation',
    name: 'Timer Irrigazione Giardino',
    domain: 'timer',
    state: 'idle',
    config_json: '{"duration_seconds": 900, "remaining_seconds": 900, "status": "idle"}'
  }
])

const showCreateModal = ref(false)
const newName = ref('')
const newDomain = ref<'input_boolean' | 'input_number' | 'timer' | 'climate_room_compensator'>('input_boolean')
const newMin = ref(10)
const newMax = ref(30)
const newStep = ref(0.5)
const newUnit = ref('°C')
const newDurationSec = ref(300)
const newTrvKey = ref('')
const newRoomTempKey = ref('')
const newWindowKey = ref('')
const newTargetTemp = ref(20.0)
const newDeltaFactor = ref(1.5)

async function loadHelpers() {
  try {
    const list = await apiClient.get<Helper[]>('/api/v1/helpers')
    if (list && list.length > 0) {
      helpers.value = list
    }
  } catch {}
}

async function toggleBoolean(h: Helper) {
  const next = h.state === 'on' ? 'turn_off' : 'turn_on'
  h.state = h.state === 'on' ? 'off' : 'on'
  try {
    await apiClient.post(`/api/v1/helpers/${h.id}/action`, { action: next })
  } catch {}
}

async function setNumber(h: Helper, val: number) {
  h.state = val.toFixed(2)
  try {
    await apiClient.post(`/api/v1/helpers/${h.id}/action`, {
      action: 'set_value',
      params: { value: val }
    })
  } catch {}
}

async function timerAction(h: Helper, action: 'start' | 'pause' | 'cancel') {
  if (action === 'start') h.state = 'active'
  if (action === 'pause') h.state = 'paused'
  if (action === 'cancel') h.state = 'idle'

  try {
    await apiClient.post(`/api/v1/helpers/${h.id}/action`, { action })
  } catch {}
}

async function deleteHelper(id: string) {
  helpers.value = helpers.value.filter(h => h.id !== id)
  try {
    await apiClient.delete(`/api/v1/helpers/${id}`)
  } catch {}
}

async function createHelper() {
  if (!newName.value.trim()) return

  const payload: any = {
    name: newName.value,
    domain: newDomain.value
  }

  if (newDomain.value === 'input_number') {
    payload.min = Number(newMin.value)
    payload.max = Number(newMax.value)
    payload.step = Number(newStep.value)
    payload.unit = newUnit.value
    payload.initial_val = Number(newMin.value)
  } else if (newDomain.value === 'timer') {
    payload.duration_sec = Number(newDurationSec.value)
  } else if (newDomain.value === 'climate_room_compensator') {
    payload.trv_entity_key = newTrvKey.value
    payload.room_temp_sensor_key = newRoomTempKey.value
    payload.window_sensor_key = newWindowKey.value
    payload.target_temperature = Number(newTargetTemp.value)
    payload.delta_factor = Number(newDeltaFactor.value)
  }

  try {
    const created = await apiClient.post<Helper>('/api/v1/helpers', payload)
    if (created) helpers.value.push(created)
  } catch {}

  showCreateModal.value = false
  newName.value = ''
}

function parseConfig(h: Helper) {
  try {
    return JSON.parse(h.config_json || '{}')
  } catch {
    return {}
  }
}

onMounted(() => {
  loadHelpers()
})
</script>

<template>
  <div class="helpers-view-container">
    <header class="helpers-header glass-panel">
      <div class="header-title-box">
        <div class="icon-circle">
          <Sliders :size="24" class="text-primary" />
        </div>
        <div>
          <h1 class="page-title">{{ t('helpers.title') }}</h1>
          <p class="page-subtitle">{{ t('helpers.subtitle') }}</p>
        </div>
      </div>

      <button class="create-btn" @click="showCreateModal = true">
        <Plus :size="18" />
        <span>{{ t('helpers.newHelper') }}</span>
      </button>
    </header>

    <!-- Helpers Grid -->
    <div class="helpers-grid">
      <div v-for="h in helpers" :key="h.id" class="helper-card glass-panel">
        <div class="card-top">
          <div class="helper-info">
            <component
              :is="h.domain === 'input_boolean' ? ToggleLeft : h.domain === 'input_number' ? Hash : h.domain === 'timer' ? Timer : Thermometer"
              :size="20"
              :class="h.domain === 'input_boolean' ? 'text-primary' : h.domain === 'input_number' ? 'text-warning' : h.domain === 'timer' ? 'text-info' : 'text-danger'"
            />
            <div>
              <h3 class="helper-name">{{ h.name }}</h3>
              <span class="helper-key">{{ h.entity_key }}</span>
            </div>
          </div>

          <button class="delete-btn" @click="deleteHelper(h.id)" title="Elimina Helper">
            <Trash2 :size="16" />
          </button>
        </div>

        <!-- Body per Domain -->
        <!-- 1. Input Boolean -->
        <div v-if="h.domain === 'input_boolean'" class="card-action boolean-action">
          <span class="status-text">{{ h.state === 'on' ? 'ATTIVO' : 'SPENTO' }}</span>
          <button :class="['toggle-pill', { active: h.state === 'on' }]" @click="toggleBoolean(h)">
            <span class="pill-thumb" />
          </button>
        </div>

        <!-- 2. Input Number -->
        <div v-else-if="h.domain === 'input_number'" class="card-action number-action">
          <div class="slider-val-row">
            <span>Valore:</span>
            <span class="slider-display">{{ h.state }} {{ parseConfig(h).unit || '' }}</span>
          </div>
          <input
            type="range"
            class="slider-input"
            :min="parseConfig(h).min ?? 0"
            :max="parseConfig(h).max ?? 100"
            :step="parseConfig(h).step ?? 1"
            :value="parseFloat(h.state)"
            @change="(e: any) => setNumber(h, parseFloat(e.target.value))"
          />
        </div>

        <!-- 3. Timer -->
        <div v-else-if="h.domain === 'timer'" class="card-action timer-action">
          <div class="timer-status-row">
            <span class="timer-state-badge" :class="h.state">{{ h.state.toUpperCase() }}</span>
            <span class="timer-time">{{ parseConfig(h).remaining_seconds ?? parseConfig(h).duration_seconds ?? 60 }}s</span>
          </div>
          <div class="timer-controls">
            <button class="t-btn play" @click="timerAction(h, 'start')" title="Avvia">
              <Play :size="16" />
            </button>
            <button class="t-btn pause" @click="timerAction(h, 'pause')" title="Pausa">
              <Pause :size="16" />
            </button>
            <button class="t-btn reset" @click="timerAction(h, 'cancel')" title="Azzera">
              <RotateCcw :size="16" />
            </button>
          </div>
        </div>

        <!-- 4. Climate Room Compensator (Better Thermostat) -->
        <div v-else-if="h.domain === 'climate_room_compensator'" class="card-action climate-action">
          <div class="climate-metric-row">
            <div>
              <span class="metric-label">Target Stanza</span>
              <div class="metric-val">{{ parseConfig(h).target_temperature || 20 }}°C</div>
            </div>
            <div>
              <span class="metric-label">TRV Dinamico</span>
              <div class="metric-val text-warning">{{ h.state }}°C</div>
            </div>
          </div>
          <div class="climate-targets-info">
            <span>Sensore: <code>{{ parseConfig(h).room_temp_sensor_key }}</code></span>
            <span>Valvola: <code>{{ parseConfig(h).trv_entity_key }}</code></span>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal Crea Helper -->
    <div v-if="showCreateModal" class="modal-backdrop">
      <div class="modal-dialog glass-panel">
        <h2 class="modal-title">Crea Nuovo Helper Virtuale</h2>

        <div class="form-group">
          <label>Nome Helper</label>
          <input v-model="newName" type="text" placeholder="Es. Modalità Pulizie" class="form-input" />
        </div>

        <div class="form-group">
          <label>Tipo di Entità</label>
          <select v-model="newDomain" class="form-input">
            <option value="input_boolean">Interruttore Booleano (input_boolean)</option>
            <option value="input_number">Cursore Numerico (input_number)</option>
            <option value="timer">Conto alla Rovescia (timer)</option>
            <option value="climate_room_compensator">Compensatore Termostati (Better Thermostat)</option>
          </select>
        </div>

        <div v-if="newDomain === 'climate_room_compensator'" class="compensator-opts">
          <div class="form-group">
            <label>Valvola Termostatica (TRV)</label>
            <input v-model="newTrvKey" type="text" placeholder="climate.valvola_salotto" class="form-input" />
          </div>
          <div class="form-group">
            <label>Sensore Temperatura Stanza</label>
            <input v-model="newRoomTempKey" type="text" placeholder="sensor.temperatura_salotto" class="form-input" />
          </div>
          <div class="form-group">
            <label>Sensore Finestra Aperta (opzionale)</label>
            <input v-model="newWindowKey" type="text" placeholder="binary_sensor.finestra_salotto" class="form-input" />
          </div>
          <div class="number-opts-grid">
            <div class="form-group">
              <label>Temperatura Target (°C)</label>
              <input v-model.number="newTargetTemp" type="number" step="0.5" class="form-input" />
            </div>
            <div class="form-group">
              <label>Fattore Offset (1.0 - 2.5)</label>
              <input v-model.number="newDeltaFactor" type="number" step="0.1" class="form-input" />
            </div>
          </div>
        </div>

        <div v-if="newDomain === 'input_number'" class="number-opts-grid">
          <div class="form-group">
            <label>Minimo</label>
            <input v-model.number="newMin" type="number" class="form-input" />
          </div>
          <div class="form-group">
            <label>Massimo</label>
            <input v-model.number="newMax" type="number" class="form-input" />
          </div>
          <div class="form-group">
            <label>Passo (Step)</label>
            <input v-model.number="newStep" type="number" step="0.1" class="form-input" />
          </div>
          <div class="form-group">
            <label>Unità</label>
            <input v-model="newUnit" type="text" placeholder="°C, %, kW" class="form-input" />
          </div>
        </div>

        <div v-if="newDomain === 'timer'" class="form-group">
          <label>Durata (secondi)</label>
          <input v-model.number="newDurationSec" type="number" class="form-input" />
        </div>

        <div class="modal-actions">
          <button class="cancel-btn" @click="showCreateModal = false">Annulla</button>
          <button class="submit-btn" @click="createHelper">Crea Helper</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.helpers-view-container {
  padding: 1.5rem;
  max-width: 1400px;
  margin: 0 auto;
}

.helpers-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.25rem 1.75rem;
  border-radius: 1rem;
  margin-bottom: 2rem;
  background: rgba(17, 24, 39, 0.7);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.header-title-box {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.icon-circle {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: rgba(59, 130, 246, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 700;
  color: #f9fafb;
  margin: 0;
}

.page-subtitle {
  font-size: 0.875rem;
  color: #9ca3af;
  margin: 0.25rem 0 0;
}

.create-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  color: white;
  border: none;
  padding: 0.6rem 1.25rem;
  border-radius: 0.5rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.15s ease;
}

.create-btn:hover {
  transform: translateY(-1px);
}

/* Helpers Grid */
.helpers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 1.5rem;
}

.helper-card {
  padding: 1.5rem;
  border-radius: 1.25rem;
  background: rgba(17, 24, 39, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.card-top {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1.25rem;
}

.helper-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.helper-name {
  font-size: 1.05rem;
  font-weight: 600;
  color: #f3f4f6;
  margin: 0;
}

.helper-key {
  font-size: 0.75rem;
  color: #6b7280;
  font-family: monospace;
}

.delete-btn {
  background: transparent;
  border: none;
  color: #6b7280;
  cursor: pointer;
  transition: color 0.15s;
}

.delete-btn:hover {
  color: #ef4444;
}

/* Card Actions */
.card-action {
  background: rgba(31, 41, 55, 0.4);
  padding: 1rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.04);
}

.boolean-action {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.status-text {
  font-size: 0.85rem;
  font-weight: 700;
  color: #9ca3af;
}

.toggle-pill {
  width: 50px;
  height: 26px;
  background: #374151;
  border-radius: 9999px;
  position: relative;
  border: none;
  cursor: pointer;
  transition: background 0.2s ease;
}

.toggle-pill.active {
  background: #3b82f6;
}

.pill-thumb {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 20px;
  height: 20px;
  background: white;
  border-radius: 50%;
  transition: transform 0.2s ease;
}

.toggle-pill.active .pill-thumb {
  transform: translateX(24px);
}

/* Number Action */
.slider-val-row {
  display: flex;
  justify-content: space-between;
  font-size: 0.85rem;
  color: #9ca3af;
  margin-bottom: 0.5rem;
}

.slider-display {
  font-size: 1.1rem;
  font-weight: 700;
  color: #f59e0b;
}

.slider-input {
  width: 100%;
  accent-color: #f59e0b;
}

/* Timer Action */
.timer-status-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.timer-state-badge {
  font-size: 0.75rem;
  padding: 0.2rem 0.5rem;
  border-radius: 0.35rem;
  font-weight: 700;
}

.timer-state-badge.active {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.timer-state-badge.paused {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
}

.timer-state-badge.idle {
  background: rgba(107, 114, 128, 0.15);
  color: #9ca3af;
}

.timer-time {
  font-size: 1.25rem;
  font-weight: 700;
  color: #06b6d4;
}

.timer-controls {
  display: flex;
  gap: 0.5rem;
}

.t-btn {
  flex: 1;
  padding: 0.4rem;
  border-radius: 0.5rem;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(31, 41, 55, 0.8);
  color: #f3f4f6;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background 0.15s;
}

.t-btn:hover {
  background: rgba(55, 65, 81, 0.9);
}

/* Modal */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal-dialog {
  background: #111827;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 1.25rem;
  padding: 2rem;
  width: 100%;
  max-width: 480px;
}

.modal-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: #f9fafb;
  margin-top: 0;
  margin-bottom: 1.5rem;
}

.form-group {
  margin-bottom: 1rem;
}

.form-group label {
  display: block;
  font-size: 0.85rem;
  color: #9ca3af;
  margin-bottom: 0.35rem;
}

.form-input {
  width: 100%;
  padding: 0.6rem 0.8rem;
  border-radius: 0.5rem;
  background: rgba(31, 41, 55, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: white;
  font-size: 0.9rem;
  box-sizing: border-box;
}

.number-opts-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.75rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1.5rem;
}

.cancel-btn {
  padding: 0.6rem 1rem;
  border-radius: 0.5rem;
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #9ca3af;
  cursor: pointer;
}

.submit-btn {
  padding: 0.6rem 1.25rem;
  border-radius: 0.5rem;
  background: #3b82f6;
  border: none;
  color: white;
  font-weight: 600;
  cursor: pointer;
}
</style>
