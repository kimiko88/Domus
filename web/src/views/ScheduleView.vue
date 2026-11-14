<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Calendar as CalendarIcon,
  Clock,
  Flame,
  Plus,
  Upload,
  CheckCircle,
  Thermometer,
  Power,
  Trash2,
  Copy,
  Save,
  Sun,
  Moon,
  Sparkles,
  X
} from '@lucide/vue'
import { apiClient } from '../api/client'
import { useEntitiesStore } from '../stores/entities'
import { useNotificationsStore } from '../stores/notifications'

const { t } = useI18n()
const entitiesStore = useEntitiesStore()
const notifStore = useNotificationsStore()

export interface TimeSlot {
  start: string
  end: string
  value: number
}

export interface Schedule {
  id: string
  name: string
  entity_key: string
  enabled: boolean
  grid: Record<string, TimeSlot[]>
}

export interface CalendarEvent {
  id: string
  summary: string
  description?: string
  start_time: string
  end_time: string
  location?: string
  source: string
}

const activeTab = ref<'cronotermostato' | 'calendar'>('cronotermostato')

const daysOfWeek = [
  { key: 'monday', label: 'Lunedì' },
  { key: 'tuesday', label: 'Martedì' },
  { key: 'wednesday', label: 'Mercoledì' },
  { key: 'thursday', label: 'Giovedì' },
  { key: 'friday', label: 'Venerdì' },
  { key: 'saturday', label: 'Sabato' },
  { key: 'sunday', label: 'Domenica' }
]

const schedules = ref<Schedule[]>([])
const selectedScheduleId = ref<string>('')
const calendarEvents = ref<CalendarEvent[]>([])
const icsUploadStatus = ref('')
const isSavingSchedule = ref(false)

// Connected Smart Thermostats from entities store
const connectedThermostats = computed(() => {
  return Object.values(entitiesStore.entities).filter(
    (e) => e.domain === 'climate' || e.id.startsWith('climate.')
  )
})

// Active regulated thermostat (either from connectedThermostats or schedule target)
const activeThermostatId = ref<string>('')

// Live thermostat regulation state
const liveTargetTemp = ref<number>(20.5)
const liveHvacMode = ref<'heat' | 'auto' | 'off'>('heat')
const isAdjustingLive = ref(false)

// Modal for adding a time slot
const showAddSlotModal = ref(false)
const targetSlotDay = ref('monday')
const newSlotForm = ref<TimeSlot>({
  start: '06:30',
  end: '08:30',
  value: 21.0
})

const selectedSchedule = computed(() => {
  return schedules.value.find((s) => s.id === selectedScheduleId.value) || null
})

// Current room temperature from thermostat
const currentAmbientTemp = computed(() => {
  if (!activeThermostatId.value) return 20.8
  const ent = entitiesStore.entities[activeThermostatId.value]
  if (!ent) return 20.8
  if (ent.attributes?.current_temperature) return Number(ent.attributes.current_temperature)
  if (ent.attributes?.temperature) return Number(ent.attributes.temperature)
  const parsed = parseFloat(ent.state)
  return isNaN(parsed) ? 20.8 : parsed
})

watchThermostatChange()

function watchThermostatChange() {
  if (connectedThermostats.value.length > 0 && !activeThermostatId.value) {
    activeThermostatId.value = connectedThermostats.value[0].id
    syncLiveThermostatValues()
  }
}

function syncLiveThermostatValues() {
  if (!activeThermostatId.value) return
  const ent = entitiesStore.entities[activeThermostatId.value]
  if (!ent) return

  if (ent.attributes?.temperature) {
    liveTargetTemp.value = Number(ent.attributes.temperature)
  }
  if (ent.state === 'heat' || ent.state === 'auto' || ent.state === 'off') {
    liveHvacMode.value = ent.state as any
  }
}

function adjustLiveTemp(delta: number) {
  liveTargetTemp.value = Math.round((liveTargetTemp.value + delta) * 2) / 2
}

function setLivePreset(temp: number) {
  liveTargetTemp.value = temp
}

async function applyLiveThermostatSettings() {
  if (!activeThermostatId.value) {
    notifStore.error('Nessun Termostato', 'Seleziona un termostato smart collegato')
    return
  }

  try {
    isAdjustingLive.value = true
    const ent = entitiesStore.entities[activeThermostatId.value]
    const domain = ent?.domain || 'climate'

    // Call set_temperature service
    await entitiesStore.callService(domain, 'set_temperature', activeThermostatId.value, {
      temperature: liveTargetTemp.value
    })

    // Call set_hvac_mode if changed
    if (liveHvacMode.value) {
      try {
        await entitiesStore.callService(domain, 'set_hvac_mode', activeThermostatId.value, {
          hvac_mode: liveHvacMode.value
        })
      } catch {}
    }

    notifStore.success(
      'Termostato Regolato',
      `Impostata temperatura obiettivo a ${liveTargetTemp.value.toFixed(1)}°C (${liveHvacMode.value.toUpperCase()})`
    )
  } catch (err: any) {
    notifStore.error('Errore Regolazione', err?.message || 'Impossibile comunicare con il termostato smart')
  } finally {
    isAdjustingLive.value = false
  }
}

// Slot Management
function openAddSlot(dayKey: string) {
  targetSlotDay.value = dayKey
  newSlotForm.value = {
    start: '07:00',
    end: '09:00',
    value: 21.0
  }
  showAddSlotModal.value = true
}

function saveSlot() {
  if (!selectedSchedule.value) return
  const sched = selectedSchedule.value
  if (!sched.grid[targetSlotDay.value]) {
    sched.grid[targetSlotDay.value] = []
  }

  sched.grid[targetSlotDay.value].push({
    start: newSlotForm.value.start,
    end: newSlotForm.value.end,
    value: Number(newSlotForm.value.value)
  })

  // Sort slots by start time
  sched.grid[targetSlotDay.value].sort((a, b) => a.start.localeCompare(b.start))

  showAddSlotModal.value = false
  notifStore.success('Fascia Aggiunta', `Fascia ${newSlotForm.value.start}-${newSlotForm.value.end} (${newSlotForm.value.value}°C) aggiunta a ${targetSlotDay.value}`)
}

function removeSlot(dayKey: string, index: number) {
  if (!selectedSchedule.value) return
  const sched = selectedSchedule.value
  if (sched.grid[dayKey]) {
    sched.grid[dayKey].splice(index, 1)
  }
}

// Preset Profiles for Schedule
function applyWinterWorkingProfile() {
  if (!selectedSchedule.value) return
  const sched = selectedSchedule.value

  const workdaySlots: TimeSlot[] = [
    { start: '06:00', end: '08:30', value: 21.0 }, // Sveglia comfort
    { start: '08:30', end: '17:30', value: 18.5 }, // Lavoro eco
    { start: '17:30', end: '22:30', value: 21.5 }, // Rientro serale
    { start: '22:30', end: '06:00', value: 17.5 }  // Notte
  ]

  const weekendSlots: TimeSlot[] = [
    { start: '07:30', end: '23:00', value: 21.0 }, // Giorno comfort continuo
    { start: '23:00', end: '07:30', value: 17.5 }  // Notte
  ]

  sched.grid = {
    monday: JSON.parse(JSON.stringify(workdaySlots)),
    tuesday: JSON.parse(JSON.stringify(workdaySlots)),
    wednesday: JSON.parse(JSON.stringify(workdaySlots)),
    thursday: JSON.parse(JSON.stringify(workdaySlots)),
    friday: JSON.parse(JSON.stringify(workdaySlots)),
    saturday: JSON.parse(JSON.stringify(weekendSlots)),
    sunday: JSON.parse(JSON.stringify(weekendSlots))
  }

  notifStore.success('Profilo Applicato', 'Impostato profilo Inverno Lavorativo (Comfort mattina/sera, Eco giorno)')
}

function copyMondayToWorkdays() {
  if (!selectedSchedule.value) return
  const sched = selectedSchedule.value
  const mondaySlots = sched.grid['monday'] || []

  sched.grid['tuesday'] = JSON.parse(JSON.stringify(mondaySlots))
  sched.grid['wednesday'] = JSON.parse(JSON.stringify(mondaySlots))
  sched.grid['thursday'] = JSON.parse(JSON.stringify(mondaySlots))
  sched.grid['friday'] = JSON.parse(JSON.stringify(mondaySlots))

  notifStore.success('Copiato', 'Fasce del Lunedì copiate su tutti i giorni lavorativi (Mar-Ven)')
}

function copySaturdayToSunday() {
  if (!selectedSchedule.value) return
  const sched = selectedSchedule.value
  const satSlots = sched.grid['saturday'] || []

  sched.grid['sunday'] = JSON.parse(JSON.stringify(satSlots))
  notifStore.success('Copiato', 'Fasce del Sabato duplicate sulla Domenica')
}

async function saveCurrentSchedule() {
  if (!selectedSchedule.value) return
  const sched = selectedSchedule.value

  try {
    isSavingSchedule.value = true
    await apiClient.put(`/api/v1/schedules/${sched.id}`, {
      name: sched.name,
      entity_key: sched.entity_key || activeThermostatId.value || 'climate.termostato_principale',
      enabled: sched.enabled,
      grid: sched.grid
    })
    notifStore.success('Cronotermostato Salvato', `Programmazione oraria per "${sched.name}" aggiornata con successo`)
  } catch (err: any) {
    notifStore.error('Errore Salvataggio', err?.message || 'Impossibile salvare la programmazione')
  } finally {
    isSavingSchedule.value = false
  }
}

async function createNewSchedule() {
  const newSchedName = `Cronotermostato ${schedules.value.length + 1}`
  const targetEntity = activeThermostatId.value || 'climate.termostato_principale'

  try {
    const created = await apiClient.post<Schedule>('/api/v1/schedules', {
      name: newSchedName,
      entity_key: targetEntity,
      enabled: true,
      grid: {
        monday: [{ start: '06:30', end: '08:30', value: 21.0 }, { start: '17:30', end: '22:30', value: 21.0 }],
        tuesday: [{ start: '06:30', end: '08:30', value: 21.0 }, { start: '17:30', end: '22:30', value: 21.0 }],
        wednesday: [{ start: '06:30', end: '08:30', value: 21.0 }, { start: '17:30', end: '22:30', value: 21.0 }],
        thursday: [{ start: '06:30', end: '08:30', value: 21.0 }, { start: '17:30', end: '22:30', value: 21.0 }],
        friday: [{ start: '06:30', end: '08:30', value: 21.0 }, { start: '17:30', end: '22:30', value: 21.0 }],
        saturday: [{ start: '08:00', end: '23:00', value: 21.0 }],
        sunday: [{ start: '08:00', end: '23:00', value: 21.0 }]
      }
    })

    if (created) {
      schedules.value.push(created)
      selectedScheduleId.value = created.id
      notifStore.success('Nuovo Cronotermostato', `Creato ${created.name}`)
    }
  } catch {
    // Fallback if offline
    const fallbackSched: Schedule = {
      id: 'sched_' + Date.now(),
      name: newSchedName,
      entity_key: targetEntity,
      enabled: true,
      grid: {
        monday: [{ start: '06:30', end: '08:30', value: 21.0 }, { start: '17:30', end: '22:30', value: 21.0 }]
      }
    }
    schedules.value.push(fallbackSched)
    selectedScheduleId.value = fallbackSched.id
  }
}

// iCal Upload Handler
async function handleICalUpload(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return

  const text = await file.text()
  try {
    await apiClient.post('/api/v1/calendar/import', text)
    icsUploadStatus.value = 'Calendario importato con successo!'
    setTimeout(() => (icsUploadStatus.value = ''), 4000)
    const e = await apiClient.get<CalendarEvent[]>('/api/v1/calendar/events')
    calendarEvents.value = e || []
  } catch {
    icsUploadStatus.value = "Errore durante l'importazione iCal"
  }
}

onMounted(async () => {
  await entitiesStore.fetchEntities()

  try {
    const s = await apiClient.get<Schedule[]>('/api/v1/schedules')
    schedules.value = s || []
    if (schedules.value.length > 0) {
      selectedScheduleId.value = schedules.value[0].id
    }
  } catch {
    schedules.value = []
  }

  // If no schedules exist, generate an initial default one so the user has immediate visual controls
  if (schedules.value.length === 0) {
    await createNewSchedule()
  }

  watchThermostatChange()

  try {
    const e = await apiClient.get<CalendarEvent[]>('/api/v1/calendar/events')
    calendarEvents.value = e || []
  } catch {
    calendarEvents.value = []
  }
})
</script>

<template>
  <div class="schedule-view-container">
    <header class="schedule-header glass-panel">
      <div class="header-title-box">
        <div class="icon-circle">
          <Flame :size="26" class="text-flame" />
        </div>
        <div>
          <h1 class="page-title">{{ t('schedules.title') }}</h1>
          <p class="page-subtitle">{{ t('schedules.subtitle') }}</p>
        </div>
      </div>

      <div class="tab-selector" role="tablist" aria-label="Sezioni Cronotermostato">
        <button
          :class="['tab-btn', { active: activeTab === 'cronotermostato' }]"
          @click="activeTab = 'cronotermostato'"
          aria-label="Cronotermostato Settimanale"
        >
          <Flame :size="16" />
          <span>Cronotermostato</span>
        </button>
        <button
          :class="['tab-btn', { active: activeTab === 'calendar' }]"
          @click="activeTab = 'calendar'"
          aria-label="Calendario & iCal"
        >
          <CalendarIcon :size="16" />
          <span>Calendario & Eventi</span>
        </button>
      </div>
    </header>

    <!-- Tab 1: Cronotermostato Live & Weekly Matrix -->
    <div v-if="activeTab === 'cronotermostato'" class="cronotermostato-layout">
      <!-- SECTION 1: LIVE SMART THERMOSTAT REGULATOR WIDGET -->
      <section class="live-thermostat-card glass-panel" aria-label="Regolazione Termostato Smart">
        <div class="thermostat-top-row">
          <div class="thermostat-title-box">
            <Thermometer :size="20" class="text-primary" />
            <h3>Regola Termostato Smart Collegato</h3>
          </div>

          <!-- Selector of Connected Thermostats -->
          <div class="thermostat-selector-wrap">
            <label for="thermostat-select">Dispositivo:</label>
            <select
              id="thermostat-select"
              v-model="activeThermostatId"
              class="thermostat-select"
              @change="syncLiveThermostatValues"
            >
              <option v-for="t in connectedThermostats" :key="t.id" :value="t.id">
                🌡️ {{ t.name }} ({{ t.id }})
              </option>
              <option v-if="connectedThermostats.length === 0" value="climate.termostato_salotto">
                🌡️ Termostato Salotto (Simulato)
              </option>
            </select>
          </div>
        </div>

        <div class="thermostat-dial-grid">
          <!-- Current Reading -->
          <div class="temp-display-block current">
            <span class="temp-sublabel">Temperatura Rilevata</span>
            <div class="temp-value-huge">
              <span class="number">{{ currentAmbientTemp.toFixed(1) }}</span>
              <span class="unit">°C</span>
            </div>
            <span class="status-indicator">
              <span class="dot-online"></span> In monitoraggio continuo
            </span>
          </div>

          <!-- Interactive Setpoint Control -->
          <div class="temp-display-block target">
            <span class="temp-sublabel">Obiettivo Setpoint Desiderato</span>

            <div class="setpoint-adjust-row">
              <button
                class="btn-temp-circle"
                @click="adjustLiveTemp(-0.5)"
                title="Diminuisci 0.5°C"
                aria-label="Diminuisci temperatura di 0.5 gradi"
              >
                -
              </button>
              <div class="temp-value-huge text-target">
                <span class="number">{{ liveTargetTemp.toFixed(1) }}</span>
                <span class="unit">°C</span>
              </div>
              <button
                class="btn-temp-circle"
                @click="adjustLiveTemp(0.5)"
                title="Aumenta 0.5°C"
                aria-label="Aumenta temperatura di 0.5 gradi"
              >
                +
              </button>
            </div>

            <!-- Quick Presets -->
            <div class="quick-temp-pills">
              <button
                class="temp-pill"
                :class="{ active: liveTargetTemp === 18.0 }"
                @click="setLivePreset(18.0)"
              >
                <Moon :size="12" />
                <span>Eco 18°C</span>
              </button>
              <button
                class="temp-pill"
                :class="{ active: liveTargetTemp === 20.0 }"
                @click="setLivePreset(20.0)"
              >
                <Sun :size="12" />
                <span>Normale 20°C</span>
              </button>
              <button
                class="temp-pill"
                :class="{ active: liveTargetTemp === 21.5 }"
                @click="setLivePreset(21.5)"
              >
                <Flame :size="12" />
                <span>Comfort 21.5°C</span>
              </button>
              <button
                class="temp-pill"
                :class="{ active: liveTargetTemp === 23.0 }"
                @click="setLivePreset(23.0)"
              >
                <Sparkles :size="12" />
                <span>Boost 23°C</span>
              </button>
            </div>
          </div>

          <!-- HVAC Mode & Execution Column -->
          <div class="temp-display-block actions-col">
            <span class="temp-sublabel">Modalità Operativa</span>

            <div class="hvac-mode-buttons">
              <button
                class="mode-btn"
                :class="{ active: liveHvacMode === 'heat' }"
                @click="liveHvacMode = 'heat'"
              >
                <Flame :size="14" />
                <span>Riscaldamento</span>
              </button>
              <button
                class="mode-btn"
                :class="{ active: liveHvacMode === 'auto' }"
                @click="liveHvacMode = 'auto'"
              >
                <Clock :size="14" />
                <span>Automatico</span>
              </button>
              <button
                class="mode-btn off"
                :class="{ active: liveHvacMode === 'off' }"
                @click="liveHvacMode = 'off'"
              >
                <Power :size="14" />
                <span>Spento</span>
              </button>
            </div>

            <button
              class="btn-apply-thermostat"
              :disabled="isAdjustingLive"
              @click="applyLiveThermostatSettings"
            >
              <Save :size="16" />
              <span>{{ isAdjustingLive ? 'Invio comando...' : 'Regola Termostato Ora' }}</span>
            </button>
          </div>
        </div>
      </section>

      <!-- SECTION 2: WEEKLY SCHEDULE MATRIX (7 GIORNI X 24H) -->
      <section v-if="selectedSchedule" class="planner-card glass-panel" aria-label="Programmatore Settimanale">
        <div class="planner-top">
          <div class="schedule-title-block">
            <h2 class="schedule-title">{{ selectedSchedule.name }}</h2>
            <span class="schedule-target">Entità controllata: {{ selectedSchedule.entity_key }}</span>
          </div>

          <div class="schedule-selector-row">
            <div class="form-group inline">
              <label for="sched-picker">Profilo:</label>
              <select id="sched-picker" v-model="selectedScheduleId" class="sched-select">
                <option v-for="s in schedules" :key="s.id" :value="s.id">
                  {{ s.name }} ({{ s.entity_key }})
                </option>
              </select>
            </div>

            <button class="btn-tool-secondary" @click="createNewSchedule">
              <Plus :size="14" />
              <span>Nuovo Profilo</span>
            </button>
          </div>

          <div class="planner-top-actions">
            <!-- Preset Profile Buttons -->
            <button class="btn-profile-preset" @click="applyWinterWorkingProfile" title="Imposta orari per settimana lavorativa">
              <Sparkles :size="14" />
              <span>Profilo Inverno Lavoro</span>
            </button>
            <button class="btn-profile-preset" @click="copyMondayToWorkdays" title="Copia Lunedì da Martedì a Venerdì">
              <Copy :size="14" />
              <span>Copia Lun &gt; Mar-Ven</span>
            </button>
            <button class="btn-profile-preset" @click="copySaturdayToSunday" title="Copia Sabato a Domenica">
              <Copy :size="14" />
              <span>Copia Sab &gt; Dom</span>
            </button>

            <!-- Save Schedule Button -->
            <button class="btn-primary" :disabled="isSavingSchedule" @click="saveCurrentSchedule">
              <Save :size="16" />
              <span>{{ isSavingSchedule ? 'Salvataggio...' : 'Salva Programmazione' }}</span>
            </button>
          </div>
        </div>

        <!-- 7-Day Matrix Grid -->
        <div class="weekly-matrix">
          <div v-for="day in daysOfWeek" :key="day.key" class="day-row">
            <div class="day-col-left">
              <span class="day-label">{{ day.label }}</span>
              <button class="btn-add-slot" @click="openAddSlot(day.key)" title="Aggiungi fascia oraria">
                <Plus :size="12" />
                <span>Fascia</span>
              </button>
            </div>

            <div class="slots-track">
              <div
                v-for="(slot, idx) in (selectedSchedule.grid[day.key] || [])"
                :key="idx"
                class="slot-block"
              >
                <div class="slot-info">
                  <span class="slot-time">{{ slot.start }} - {{ slot.end }}</span>
                  <span class="slot-temp">{{ Number(slot.value).toFixed(1) }}°C</span>
                </div>
                <button
                  class="btn-del-slot"
                  @click="removeSlot(day.key, idx)"
                  title="Elimina fascia"
                  aria-label="Elimina fascia oraria"
                >
                  <X :size="12" />
                </button>
              </div>

              <div
                v-if="!selectedSchedule.grid[day.key] || selectedSchedule.grid[day.key].length === 0"
                class="no-slots"
              >
                Nessuna fascia oraria (mantenimento eco 18.0°C)
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- Tab 2: Calendar & iCal Sync -->
    <div v-else class="calendar-layout">
      <div class="calendar-card glass-panel">
        <div class="cal-top-bar">
          <h2 class="section-title">Eventi Sincronizzati & Trigger Automazioni</h2>
          <label class="import-btn">
            <Upload :size="16" />
            <span>Importa .ics / CalDAV</span>
            <input type="file" accept=".ics" class="hidden-input" @change="handleICalUpload" />
          </label>
        </div>

        <p v-if="icsUploadStatus" class="status-msg text-success">{{ icsUploadStatus }}</p>

        <p v-if="calendarEvents.length === 0" class="empty-state">
          Nessun evento sincronizzato nel database.
        </p>

        <div v-else class="calendar-events-list">
          <div v-for="evt in calendarEvents" :key="evt.id" class="cal-event-card">
            <div class="cal-time-col">
              <span class="cal-date">{{ new Date(evt.start_time).toLocaleDateString('it-IT') }}</span>
              <span class="cal-hours">{{
                new Date(evt.start_time).toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' })
              }}</span>
            </div>

            <div class="cal-info-col">
              <h3 class="evt-summary">{{ evt.summary }}</h3>
              <p class="evt-desc">{{ evt.description }}</p>
              <span v-if="evt.location" class="evt-loc">Stanza: {{ evt.location }}</span>
            </div>

            <span class="source-tag" :class="evt.source">{{ evt.source.toUpperCase() }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal for Adding a Time Slot -->
    <div v-if="showAddSlotModal" class="modal-backdrop" @click="showAddSlotModal = false">
      <div class="modal-card" @click.stop role="dialog" aria-modal="true" aria-labelledby="modal-slot-title">
        <div class="modal-header">
          <h2 id="modal-slot-title">Nuova Fascia Oraria</h2>
          <button class="modal-close-btn" @click="showAddSlotModal = false" aria-label="Chiudi finestra">
            <X class="icon-close" />
          </button>
        </div>

        <form @submit.prevent="saveSlot" class="modal-body">
          <div class="form-row">
            <div class="form-group">
              <label for="slot-start">Ora Inizio</label>
              <input id="slot-start" v-model="newSlotForm.start" type="time" required />
            </div>
            <div class="form-group">
              <label for="slot-end">Ora Fine</label>
              <input id="slot-end" v-model="newSlotForm.end" type="time" required />
            </div>
          </div>

          <div class="form-group">
            <label for="slot-temp">Temperatura Obiettivo (°C)</label>
            <input
              id="slot-temp"
              v-model.number="newSlotForm.value"
              type="number"
              step="0.5"
              min="14"
              max="28"
              required
            />
          </div>

          <div class="modal-footer">
            <button type="button" class="btn-cancel" @click="showAddSlotModal = false">Annulla</button>
            <button type="submit" class="btn-primary">Aggiungi Fascia</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.schedule-view-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.schedule-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-radius: 16px;
  background: var(--surface-card, rgba(15, 23, 42, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  flex-wrap: wrap;
  gap: 16px;
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
  background: rgba(245, 158, 11, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
}

.text-flame {
  color: #f59e0b;
}

.page-title {
  font-size: 1.4rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-main, #f8fafc);
}

.page-subtitle {
  font-size: 0.85rem;
  color: var(--text-muted, #94a3b8);
  margin: 4px 0 0;
}

.tab-selector {
  display: flex;
  gap: 8px;
  background: rgba(255, 255, 255, 0.05);
  padding: 4px;
  border-radius: 10px;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: var(--text-muted, #94a3b8);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.tab-btn.active {
  background: var(--primary, #3b82f6);
  color: #fff;
}

/* LIVE SMART THERMOSTAT CARD */
.cronotermostato-layout {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.live-thermostat-card {
  padding: 24px;
  border-radius: 16px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  box-shadow: var(--shadow-md);
}

.thermostat-top-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  flex-wrap: wrap;
  gap: 12px;
}

.thermostat-title-box {
  display: flex;
  align-items: center;
  gap: 8px;
}

.thermostat-title-box h3 {
  margin: 0;
  font-size: 1.1rem;
  color: var(--text-main);
}

.thermostat-selector-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  color: var(--text-muted);
}

.thermostat-select {
  padding: 6px 12px;
  background: var(--bg-surface);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-main);
  font-size: 0.85rem;
}

.thermostat-dial-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 20px;
}

.temp-display-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
  border-radius: 12px;
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-color);
}

.temp-sublabel {
  font-size: 0.8rem;
  color: var(--text-muted);
  font-weight: 600;
  margin-bottom: 12px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.temp-value-huge {
  display: flex;
  align-items: baseline;
  gap: 4px;
}

.temp-value-huge .number {
  font-size: 3rem;
  font-weight: 800;
  color: var(--text-main);
  line-height: 1;
}

.temp-value-huge .unit {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-muted);
}

.temp-value-huge.text-target .number {
  color: #f59e0b;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  color: #10b981;
  margin-top: 12px;
}

.dot-online {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 8px #10b981;
}

.setpoint-adjust-row {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 12px;
}

.btn-temp-circle {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: 2px solid var(--border-color);
  background: var(--bg-surface-hover);
  color: var(--text-main);
  font-size: 1.5rem;
  font-weight: 700;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.btn-temp-circle:hover {
  background: rgba(245, 158, 11, 0.3);
  border-color: #f59e0b;
  transform: scale(1.08);
}

.quick-temp-pills {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: center;
}

.temp-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.05);
  color: #cbd5e1;
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.2s;
}

.temp-pill:hover {
  background: rgba(255, 255, 255, 0.1);
}

.temp-pill.active {
  background: rgba(245, 158, 11, 0.25);
  border-color: #f59e0b;
  color: #fbbf24;
  font-weight: 600;
}

.actions-col {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.hvac-mode-buttons {
  display: flex;
  gap: 6px;
  width: 100%;
}

.mode-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 8px 10px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.05);
  color: #cbd5e1;
  font-size: 0.78rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.mode-btn:hover {
  background: rgba(255, 255, 255, 0.1);
}

.mode-btn.active {
  background: rgba(245, 158, 11, 0.3);
  border-color: #f59e0b;
  color: #fbbf24;
}

.mode-btn.off.active {
  background: rgba(239, 68, 68, 0.2);
  border-color: #ef4444;
  color: #f87171;
}

.btn-apply-thermostat {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px;
  border-radius: 10px;
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  border: none;
  color: #000;
  font-weight: 700;
  font-size: 0.9rem;
  cursor: pointer;
  transition: filter 0.2s, transform 0.15s;
}

.btn-apply-thermostat:hover {
  filter: brightness(1.1);
  transform: translateY(-1px);
}

/* SECTION 2: WEEKLY SCHEDULE MATRIX */
.planner-card,
.calendar-card {
  padding: 24px;
  border-radius: 16px;
  background: var(--surface-card, rgba(15, 23, 42, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
}

.planner-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  flex-wrap: wrap;
  gap: 16px;
}

.schedule-selector-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.form-group.inline {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
}

.form-group.inline label {
  font-size: 0.85rem;
  color: var(--text-muted);
}

.sched-select {
  padding: 6px 12px;
  background: var(--bg-surface);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-main);
  font-size: 0.85rem;
}

.btn-tool-secondary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-color);
  color: var(--text-muted);
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-tool-secondary:hover {
  background: var(--bg-surface);
  color: var(--text-main);
}

.planner-top-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.btn-profile-preset {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  background: rgba(59, 130, 246, 0.15);
  border: 1px solid rgba(59, 130, 246, 0.3);
  color: #60a5fa;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-profile-preset:hover {
  background: rgba(59, 130, 246, 0.25);
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--primary, #3b82f6);
  border: none;
  color: #fff;
  border-radius: 8px;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary:hover {
  background: var(--primary-hover, #2563eb);
}

.weekly-matrix {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.day-row {
  display: grid;
  grid-template-columns: 140px 1fr;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 10px;
}

.day-col-left {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.day-label {
  font-size: 0.9rem;
  font-weight: 700;
  color: #f1f5f9;
}

.btn-add-slot {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #94a3b8;
  font-size: 0.7rem;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-add-slot:hover {
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
  border-color: #3b82f6;
}

.slots-track {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.slot-block {
  display: flex;
  align-items: center;
  gap: 8px;
  background: rgba(245, 158, 11, 0.15);
  border: 1px solid rgba(245, 158, 11, 0.3);
  padding: 6px 10px;
  border-radius: 8px;
}

.slot-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.slot-time {
  font-size: 0.8rem;
  color: #cbd5e1;
  font-weight: 500;
}

.slot-temp {
  font-size: 0.85rem;
  font-weight: 700;
  color: #fbbf24;
}

.btn-del-slot {
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}

.btn-del-slot:hover {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.15);
}

.no-slots {
  font-size: 0.8rem;
  color: #64748b;
  font-style: italic;
}

/* Tab 2 Calendar Styling */
.cal-top-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-title {
  font-size: 1.1rem;
  font-weight: 600;
  margin: 0;
  color: var(--text-main);
}

.import-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 8px;
  background: rgba(99, 102, 241, 0.15);
  color: #818cf8;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
}

.hidden-input {
  display: none;
}

.calendar-events-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.cal-event-card {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 14px 18px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 12px;
}

.cal-time-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 90px;
}

.cal-date {
  font-size: 0.8rem;
  color: var(--text-muted, #94a3b8);
}

.cal-hours {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--primary, #3b82f6);
}

.cal-info-col {
  flex: 1;
}

.evt-summary {
  font-size: 0.95rem;
  font-weight: 600;
  margin: 0;
}

.evt-desc {
  font-size: 0.8rem;
  color: var(--text-muted, #94a3b8);
  margin: 4px 0;
}

.evt-loc {
  font-size: 0.75rem;
  color: #f59e0b;
}

.source-tag {
  font-size: 0.7rem;
  font-weight: 700;
  padding: 4px 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.1);
}

.empty-state {
  padding: 32px;
  text-align: center;
  color: var(--text-muted, #94a3b8);
}

/* Modals */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
  padding: 1rem;
}

.modal-card {
  background: #0f172a;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 0.75rem;
  width: 100%;
  max-width: 440px;
  padding: 1.5rem;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.7);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.modal-header h2 {
  font-size: 1.25rem;
  margin: 0;
  color: #f8fafc;
}

.modal-close-btn {
  background: none;
  border: none;
  color: #94a3b8;
  cursor: pointer;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-bottom: 1rem;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.form-group label {
  font-size: 0.8rem;
  font-weight: 500;
  color: #cbd5e1;
}

.form-group input {
  padding: 0.6rem;
  background: #1e293b;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 0.375rem;
  color: #fff;
  font-size: 0.875rem;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1.5rem;
}

.btn-cancel {
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #94a3b8;
  border-radius: 0.375rem;
  cursor: pointer;
}

/* Keep schedule tabs reachable on narrow screens and prevent flex overflow. */
.schedule-view-container {
  width: 100%;
  min-width: 0;
}

.tab-selector {
  max-width: 100%;
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
  flex-wrap: nowrap;
  -webkit-overflow-scrolling: touch;
  overscroll-behavior-x: contain;
  scrollbar-width: thin;
  touch-action: pan-x;
}

.tab-selector .tab-btn {
  flex: 0 0 auto;
  min-height: 44px;
  white-space: nowrap;
}

@media (max-width: 640px) {
  .schedule-header {
    align-items: flex-start;
    padding: 16px;
  }

  .header-title-box {
    min-width: 0;
    align-items: flex-start;
  }

  .tab-selector {
    width: 100%;
  }

  .live-thermostat-card {
    padding: 16px;
  }

  .thermostat-dial-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

</style>
