<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { usePredictiveClimateStore } from '../stores/predictiveClimate'
import type {
  PredictiveZoneConfig,
  RoomOrientation,
  EmitterType,
  ThermalProfile,
  PreheatRecommendation,
  CoastDownRecommendation
} from '../api/types'

const store = usePredictiveClimateStore()

const showModal = ref(false)
const editingZone = ref<PredictiveZoneConfig | null>(null)

// Thermal Profile Modal state
const showProfileModal = ref(false)
const selectedProfileZone = ref<PredictiveZoneConfig | null>(null)
const currentProfile = ref<ThermalProfile | null>(null)
const profileLoading = ref(false)

// Optimal Start & Stop state
const calcTargetTime = ref('07:30')
const calcTargetTemp = ref(21.0)
const preheatResult = ref<PreheatRecommendation | null>(null)
const calcDepartureTime = ref('08:30')
const coastResult = ref<CoastDownRecommendation | null>(null)

// Cycle training simulator state
const trainCycleType = ref<'heating' | 'cooling'>('heating')
const trainStartTemp = ref(19.0)
const trainEndTemp = ref(20.5)
const trainDurationMin = ref(60)
const trainLoading = ref(false)
const trainSuccessMsg = ref('')

// Form fields
const formName = ref('')
const formTRVKey = ref('')
const formTempKey = ref('')
const formPressureKey = ref('')
const formHumidityKey = ref('')
const formWindowKey = ref('')
const formTargetTemp = ref(21.0)
const formOrientation = ref<RoomOrientation>('south')
const formWindowArea = ref(4.0)
const formEmitter = ref<EmitterType>('radiator')
const formInertia = ref(2.0)
const formPreheat = ref(true)
const formSolarCutoff = ref(true)
const formPressureBoost = ref(true)
const formEnabled = ref(true)

onMounted(async () => {
  await store.fetchStatus()
})

async function openThermalProfileModal(zone: PredictiveZoneConfig) {
  if (!zone.id) return
  selectedProfileZone.value = zone
  calcTargetTemp.value = zone.target_temperature || 21.0
  preheatResult.value = null
  coastResult.value = null
  trainSuccessMsg.value = ''
  showProfileModal.value = true
  profileLoading.value = true

  try {
    currentProfile.value = await store.fetchProfile(zone.id)
    await handleCalculateOptimalStart()
    await handleCalculateCoastDown()
  } catch (e) {
    console.error('Failed to load thermal profile', e)
  } finally {
    profileLoading.value = false
  }
}

async function handleCalculateOptimalStart() {
  if (!selectedProfileZone.value?.id) return
  try {
    const today = new Date()
    const parts = calcTargetTime.value.split(':').map(Number)
    const hours = parts[0] ?? 7
    const minutes = parts[1] ?? 30
    const targetDate = new Date(today.getFullYear(), today.getMonth(), today.getDate(), hours, minutes)
    preheatResult.value = await store.getOptimalStart(
      selectedProfileZone.value.id,
      targetDate.toISOString(),
      Number(calcTargetTemp.value)
    )
  } catch (err: any) {
    console.error('Error calculating preheat', err)
  }
}

async function handleCalculateCoastDown() {
  if (!selectedProfileZone.value?.id) return
  try {
    const today = new Date()
    const parts = calcDepartureTime.value.split(':').map(Number)
    const hours = parts[0] ?? 8
    const minutes = parts[1] ?? 30
    const depDate = new Date(today.getFullYear(), today.getMonth(), today.getDate(), hours, minutes)
    coastResult.value = await store.getOptimalStop(
      selectedProfileZone.value.id,
      depDate.toISOString()
    )
  } catch (err: any) {
    console.error('Error calculating coast down', err)
  }
}

async function handleTrainCycle() {
  if (!selectedProfileZone.value?.id) return
  trainLoading.value = true
  trainSuccessMsg.value = ''
  try {
    const updated = await store.trainZone(selectedProfileZone.value.id, {
      cycle_type: trainCycleType.value,
      start_temp: Number(trainStartTemp.value),
      end_temp: Number(trainEndTemp.value),
      duration_minutes: Number(trainDurationMin.value),
      observed_at: new Date().toISOString()
    })
    currentProfile.value = updated
    trainSuccessMsg.value = `Modello aggiornato! Nuova velocità: ${updated.heat_rate_per_hr} °C/h (Campioni: ${updated.samples_count})`
    await handleCalculateOptimalStart()
    await handleCalculateCoastDown()
  } catch (err: any) {
    console.error('Error training cycle', err)
  } finally {
    trainLoading.value = false
  }
}

function openCreateModal() {
  editingZone.value = null
  formName.value = ''
  formTRVKey.value = ''
  formTempKey.value = ''
  formPressureKey.value = ''
  formHumidityKey.value = ''
  formWindowKey.value = ''
  formTargetTemp.value = 21.0
  formOrientation.value = 'south'
  formWindowArea.value = 4.0
  formEmitter.value = 'radiator'
  formInertia.value = 2.0
  formPreheat.value = true
  formSolarCutoff.value = true
  formPressureBoost.value = true
  formEnabled.value = true
  showModal.value = true
}

function openEditModal(cfg: PredictiveZoneConfig) {
  editingZone.value = cfg
  formName.value = cfg.name
  formTRVKey.value = cfg.trv_entity_key
  formTempKey.value = cfg.temp_sensor_key
  formPressureKey.value = cfg.pressure_sensor_key || ''
  formHumidityKey.value = cfg.humidity_sensor_key || ''
  formWindowKey.value = cfg.window_sensor_key || ''
  formTargetTemp.value = cfg.target_temperature
  formOrientation.value = cfg.orientation
  formWindowArea.value = cfg.window_area_sqm
  formEmitter.value = cfg.emitter_type
  formInertia.value = cfg.thermal_inertia_hours
  formPreheat.value = cfg.preheat_enabled
  formSolarCutoff.value = cfg.solar_cutoff_enabled
  formPressureBoost.value = cfg.pressure_boost_enabled
  formEnabled.value = cfg.enabled
  showModal.value = true
}

async function handleSaveZone() {
  if (!formName.value || !formTRVKey.value || !formTempKey.value) return

  const payload: PredictiveZoneConfig = {
    id: editingZone.value?.id,
    name: formName.value.trim(),
    trv_entity_key: formTRVKey.value.trim(),
    temp_sensor_key: formTempKey.value.trim(),
    pressure_sensor_key: formPressureKey.value.trim() || undefined,
    humidity_sensor_key: formHumidityKey.value.trim() || undefined,
    window_sensor_key: formWindowKey.value.trim() || undefined,
    target_temperature: Number(formTargetTemp.value),
    orientation: formOrientation.value,
    window_area_sqm: Number(formWindowArea.value),
    emitter_type: formEmitter.value,
    thermal_inertia_hours: Number(formInertia.value),
    preheat_enabled: formPreheat.value,
    solar_cutoff_enabled: formSolarCutoff.value,
    pressure_boost_enabled: formPressureBoost.value,
    enabled: formEnabled.value
  }

  await store.saveZone(payload)
  showModal.value = false
}

async function handleDeleteZone(id: string) {
  if (confirm('Sei sicuro di voler eliminare questa zona predittiva?')) {
    await store.deleteZone(id)
  }
}

async function handleEvaluateZone(id: string) {
  await store.evaluateZone(id)
}

async function handleEvaluateAll() {
  await store.evaluateAll()
}

function getModeBadgeClass(mode?: string): string {
  switch (mode) {
    case 'cold_front_boost':
      return 'badge-cold-front'
    case 'solar_cutoff':
      return 'badge-solar'
    case 'frost_protection':
      return 'badge-frost'
    case 'preheat':
      return 'badge-preheat'
    case 'eco':
      return 'badge-eco'
    default:
      return 'badge-comfort'
  }
}

function getModeLabel(mode?: string): string {
  switch (mode) {
    case 'cold_front_boost':
      return 'Fronte Freddo (+Boost)'
    case 'solar_cutoff':
      return 'Taglio Solare (-Eco)'
    case 'frost_protection':
      return 'Finestra Aperta (Spento)'
    case 'preheat':
      return 'Pre-riscaldamento'
    case 'eco':
      return 'A Regime (Eco)'
    default:
      return 'Comfort Standard'
  }
}

function getOrientationLabel(orient: RoomOrientation): string {
  switch (orient) {
    case 'south':
      return 'Sud'
    case 'north':
      return 'Nord'
    case 'east':
      return 'Est'
    case 'west':
      return 'Ovest'
  }
}

function getEmitterLabel(emitter: EmitterType): string {
  switch (emitter) {
    case 'floor_radiant':
      return 'Pavimento Radiante'
    case 'fancoil':
      return 'Fan-coil'
    default:
      return 'Radiatori'
  }
}
</script>

<template>
  <div class="predictive-climate-container" role="main" aria-label="Controllo Termostato Predittivo">
    <!-- Header -->
    <header class="view-header">
      <div class="title-section">
        <h1 class="view-title">Termostato Predittivo Multizona</h1>
        <p class="view-subtitle">
          Controllo intelligente con inerzia termica, nowcasting barometrico locale e previsioni solari
        </p>
      </div>

      <div class="header-actions">
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="store.evaluating"
          @click="handleEvaluateAll"
          aria-label="Rivaluta tutte le zone"
        >
          {{ store.evaluating ? 'Calcolo in corso...' : '⚡ Valuta Tutte le Zone' }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          @click="openCreateModal"
          aria-label="Aggiungi nuova zona predittiva"
        >
          + Nuova Zona
        </button>
      </div>
    </header>

    <!-- Top KPI Banners -->
    <section class="kpi-banner-grid" aria-label="Riepilogo condizioni ambientali">
      <div class="kpi-card">
        <div class="kpi-icon">🌡️</div>
        <div class="kpi-content">
          <div class="kpi-value">{{ store.enabledZones }} / {{ store.totalZones }}</div>
          <div class="kpi-label">Zone Attive</div>
        </div>
      </div>

      <div
        class="kpi-card"
        :class="{ 'alert-active': store.coldFrontActive }"
        role="status"
        aria-live="polite"
      >
        <div class="kpi-icon">📉</div>
        <div class="kpi-content">
          <div class="kpi-value">
            {{ store.coldFrontActive ? 'Fronte Freddo Rilevato' : 'Pressione Stabile' }}
          </div>
          <div class="kpi-label">
            {{ store.coldFrontActive ? 'Anticipo termico barometrico attivo' : 'Nessun crollo barometrico' }}
          </div>
        </div>
      </div>

      <div
        class="kpi-card"
        :class="{ 'solar-active': store.solarCutoffActive }"
        role="status"
        aria-live="polite"
      >
        <div class="kpi-icon">☀️</div>
        <div class="kpi-content">
          <div class="kpi-value">
            {{ store.solarCutoffActive ? 'Taglio Solare Attivo' : 'Sole Moderato' }}
          </div>
          <div class="kpi-label">
            {{ store.solarCutoffActive ? 'Risparmio attivo su stanze esposte' : 'Apporto solare standard' }}
          </div>
        </div>
      </div>
    </section>

    <!-- Zone Cards Grid -->
    <section class="zones-grid" aria-label="Zone termiche predittive">
      <div v-if="store.loading && store.zoneStatuses.length === 0" class="empty-state">
        Caricamento zone in corso...
      </div>

      <div v-else-if="store.zoneStatuses.length === 0" class="empty-state">
        <p>Nessuna zona predittiva configurata.</p>
        <button type="button" class="btn btn-primary" @click="openCreateModal">
          Configura la prima zona
        </button>
      </div>

      <article
        v-for="item in store.zoneStatuses"
        :key="item.config.id"
        class="zone-card"
        :class="{ disabled: !item.config.enabled }"
        :aria-labelledby="`zone-title-${item.config.id}`"
      >
        <header class="zone-card-header">
          <div>
            <h2 :id="`zone-title-${item.config.id}`" class="zone-title">
              {{ item.config.name }}
            </h2>
            <div class="zone-tags">
              <span class="tag tag-orientation">{{ getOrientationLabel(item.config.orientation) }}</span>
              <span class="tag tag-emitter">{{ getEmitterLabel(item.config.emitter_type) }}</span>
              <span class="tag tag-window">{{ item.config.window_area_sqm }} m² vetrate</span>
            </div>
          </div>

          <div
            v-if="item.latest_evaluation"
            class="mode-badge"
            :class="getModeBadgeClass(item.latest_evaluation.mode)"
          >
            {{ getModeLabel(item.latest_evaluation.mode) }}
          </div>
        </header>

        <!-- Main Temperature Display -->
        <div class="zone-metrics-row">
          <div class="metric-box">
            <span class="metric-label">Stanza / Percepita</span>
            <span class="metric-temp">
              {{ item.latest_evaluation ? item.latest_evaluation.current_temp : '--' }} °C
            </span>
            <span v-if="item.latest_evaluation" class="metric-sub">
              (percepita {{ item.latest_evaluation.apparent_temp }} °C)
            </span>
          </div>

          <div class="metric-box target-box">
            <span class="metric-label">Target Base</span>
            <span class="metric-target">{{ item.config.target_temperature }} °C</span>
          </div>

          <div class="metric-box calculated-box">
            <span class="metric-label">Setpoint Calcolato</span>
            <span class="metric-setpoint">
              {{ item.latest_evaluation ? item.latest_evaluation.calculated_setpoint : item.config.target_temperature }} °C
            </span>
          </div>
        </div>

        <!-- Environmental Context -->
        <div v-if="item.latest_evaluation" class="zone-context-grid">
          <div class="context-item">
            <span class="context-icon">⏲️</span>
            <div>
              <div class="context-val">{{ item.latest_evaluation.current_pressure }} hPa</div>
              <div class="context-sub">
                Trend: {{ item.latest_evaluation.pressure_trend_hpa_hr > 0 ? '+' : '' }}{{ item.latest_evaluation.pressure_trend_hpa_hr }} hPa/h
              </div>
            </div>
          </div>

          <div class="context-item">
            <span class="context-icon">☀️</span>
            <div>
              <div class="context-val">{{ item.latest_evaluation.direct_radiation_forecast }} W/m²</div>
              <div class="context-sub">
                Guadagno: +{{ item.latest_evaluation.estimated_solar_gain_kw }} kW
              </div>
            </div>
          </div>

          <div class="context-item">
            <span class="context-icon">💧</span>
            <div>
              <div class="context-val">{{ item.latest_evaluation.humidity }}% UR</div>
              <div class="context-sub">Est. {{ item.latest_evaluation.outdoor_temp_forecast }} °C</div>
            </div>
          </div>
        </div>

        <!-- Explainability Note -->
        <div v-if="item.latest_evaluation?.reason" class="zone-reason">
          <span class="reason-icon">💡</span>
          <span class="reason-text">{{ item.latest_evaluation.reason }}</span>
        </div>

        <!-- Actions -->
        <footer class="zone-card-footer">
          <div class="footer-start-actions">
            <button
              type="button"
              class="btn btn-sm btn-outline"
              :disabled="store.evaluating"
              @click="item.config.id && handleEvaluateZone(item.config.id)"
              aria-label="Calcola setpoint per questa stanza"
            >
              ⚡ Valuta Ora
            </button>
            <button
              type="button"
              class="btn btn-sm btn-secondary"
              @click="openThermalProfileModal(item.config)"
              aria-label="Visualizza modello termico e start ottimale"
            >
              🧠 Modello & Start
            </button>
          </div>
          <div class="footer-end-actions">
            <button
              type="button"
              class="btn btn-sm btn-ghost"
              @click="openEditModal(item.config)"
              aria-label="Modifica parametri zona"
            >
              Modifica
            </button>
            <button
              type="button"
              class="btn btn-sm btn-ghost text-danger"
              @click="item.config.id && handleDeleteZone(item.config.id)"
              aria-label="Elimina zona"
            >
              Elimina
            </button>
          </div>
        </footer>
      </article>
    </section>

    <!-- Modal Form -->
    <div
      v-if="showModal"
      class="modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
      @click.self="showModal = false"
    >
      <div class="modal-card">
        <header class="modal-header">
          <h2 id="modal-title">{{ editingZone ? 'Modifica Zona Predittiva' : 'Nuova Zona Predittiva' }}</h2>
          <button
            type="button"
            class="btn-close"
            @click="showModal = false"
            aria-label="Chiudi finestra"
          >
            &times;
          </button>
        </header>

        <form @submit.prevent="handleSaveZone" class="modal-form">
          <div class="form-group">
            <label for="zone-name">Nome Zona / Stanza</label>
            <input
              id="zone-name"
              v-model="formName"
              type="text"
              class="form-control"
              placeholder="es. Salone Principale"
              required
            />
          </div>

          <div class="form-row">
            <div class="form-group col">
              <label for="zone-trv">Entità TRV / Termostato</label>
              <input
                id="zone-trv"
                v-model="formTRVKey"
                type="text"
                class="form-control"
                placeholder="climate.valvola_salone"
                required
              />
            </div>

            <div class="form-group col">
              <label for="zone-temp">Sensore Temperatura Stanza</label>
              <input
                id="zone-temp"
                v-model="formTempKey"
                type="text"
                class="form-control"
                placeholder="sensor.salone_temperatura"
                required
              />
            </div>
          </div>

          <div class="form-row">
            <div class="form-group col">
              <label for="zone-press">Sensore Pressione (Barometro)</label>
              <input
                id="zone-press"
                v-model="formPressureKey"
                type="text"
                class="form-control"
                placeholder="sensor.salone_pressione"
              />
            </div>

            <div class="form-group col">
              <label for="zone-hum">Sensore Umidità Relativa</label>
              <input
                id="zone-hum"
                v-model="formHumidityKey"
                type="text"
                class="form-control"
                placeholder="sensor.salone_umidita"
              />
            </div>
          </div>

          <div class="form-group">
            <label for="zone-win">Sensore Contatto Finestra (Opzionale)</label>
            <input
              id="zone-win"
              v-model="formWindowKey"
              type="text"
              class="form-control"
              placeholder="binary_sensor.finestra_salone"
            />
          </div>

          <div class="form-row">
            <div class="form-group col">
              <label for="zone-target">Temperatura Obiettivo (°C)</label>
              <input
                id="zone-target"
                v-model.number="formTargetTemp"
                type="number"
                step="0.5"
                min="10"
                max="30"
                class="form-control"
                required
              />
            </div>

            <div class="form-group col">
              <label for="zone-orient">Esposizione Principale</label>
              <select id="zone-orient" v-model="formOrientation" class="form-control">
                <option value="south">Sud (Forte Sole Pomeriggio)</option>
                <option value="east">Est (Sole Mattutino)</option>
                <option value="west">Ovest (Sole Tardo Pomeriggio)</option>
                <option value="north">Nord (Dispersivo, Zero Sole)</option>
              </select>
            </div>
          </div>

          <div class="form-row">
            <div class="form-group col">
              <label for="zone-emitter">Tipo Terminale Riscaldamento</label>
              <select id="zone-emitter" v-model="formEmitter" class="form-control">
                <option value="radiator">Radiatori (Inerzia media 2h)</option>
                <option value="floor_radiant">Pavimento Radiante (Inerzia alta 4h)</option>
                <option value="fancoil">Fan-coil (Inerzia rapida 1h)</option>
              </select>
            </div>

            <div class="form-group col">
              <label for="zone-area">Superficie Vetrate (m²)</label>
              <input
                id="zone-area"
                v-model.number="formWindowArea"
                type="number"
                step="0.5"
                min="0"
                max="50"
                class="form-control"
                required
              />
            </div>
          </div>

          <div class="toggles-box">
            <label class="toggle-label">
              <input type="checkbox" v-model="formPressureBoost" />
              <span>Anticipo Fronte Freddo (Nowcasting Barometrico)</span>
            </label>

            <label class="toggle-label">
              <input type="checkbox" v-model="formSolarCutoff" />
              <span>Taglio Solare Preventivo (Anti-Surriscaldamento)</span>
            </label>

            <label class="toggle-label">
              <input type="checkbox" v-model="formPreheat" />
              <span>Pre-riscaldamento Meteo (Gelo in arrivo)</span>
            </label>

            <label class="toggle-label">
              <input type="checkbox" v-model="formEnabled" />
              <span>Zona Attiva</span>
            </label>
          </div>

          <footer class="modal-footer">
            <button type="button" class="btn btn-secondary" @click="showModal = false">
              Annulla
            </button>
            <button type="submit" class="btn btn-primary">
              Salva Configurazione
            </button>
          </footer>
        </form>
      </div>
    </div>

    <!-- Modal Profilo Termico & Auto-Apprendimento Grey-box -->
    <div
      v-if="showProfileModal && selectedProfileZone"
      class="modal-backdrop"
      role="dialog"
      aria-modal="true"
      :aria-labelledby="`profile-modal-title-${selectedProfileZone.id}`"
      @click.self="showProfileModal = false"
    >
      <div class="modal-card modal-card-lg">
        <header class="modal-header">
          <div>
            <h2 :id="`profile-modal-title-${selectedProfileZone.id}`">
              🧠 Modello Termico & Start: {{ selectedProfileZone.name }}
            </h2>
            <p class="modal-subtitle">
              Stima fisica 1R-1C, pre-heating intelligente e coast-down a inerzia
            </p>
          </div>
          <button
            type="button"
            class="btn-close"
            @click="showProfileModal = false"
            aria-label="Chiudi finestra"
          >
            ✕
          </button>
        </header>

        <div v-if="profileLoading" class="modal-body modal-loading">
          Caricamento parametri termici in corso...
        </div>

        <div v-else class="modal-body profile-modal-content">
          <!-- 1. Parametri Appresi -->
          <section class="profile-section" aria-label="Parametri termici appresi">
            <h3 class="section-heading">📊 Parametri Appresi (Fisica Grey-Box 1R-1C)</h3>
            <div class="profile-metrics-grid">
              <div class="profile-metric-card">
                <span class="p-label">Velocità Riscaldamento</span>
                <span class="p-value heat-val">
                  {{ currentProfile ? currentProfile.heat_rate_per_hr : '--' }} °C/h
                </span>
                <span class="p-sub">In potenza attiva nominale</span>
              </div>

              <div class="profile-metric-card">
                <span class="p-label">Velocità Dispersione</span>
                <span class="p-value cool-val">
                  {{ currentProfile ? currentProfile.cool_rate_per_hr : '--' }} °C/h
                </span>
                <span class="p-sub">Normalizzata a Δ10°C esterno</span>
              </div>

              <div class="profile-metric-card">
                <span class="p-label">Cicli di Apprendimento</span>
                <span class="p-value samples-val">
                  {{ currentProfile ? currentProfile.samples_count : 0 }}
                </span>
                <span class="p-sub">Aggiornamenti EMA registrati</span>
              </div>
            </div>
          </section>

          <!-- 2. Ottimizzatori: Pre-heating e Coast-Down -->
          <section class="profile-section" aria-label="Calcolatori dinamici">
            <div class="optimizers-grid">
              <!-- Pre-heating Card -->
              <div class="optimizer-card">
                <div class="opt-header">
                  <span class="opt-icon">⚡</span>
                  <div>
                    <h4 class="opt-title">Pre-Heating (Start Ottimale)</h4>
                    <p class="opt-desc">Anticipa l'accensione per raggiungere il comfort esattamente all'orario prefissato</p>
                  </div>
                </div>

                <div class="opt-inputs">
                  <div class="form-group">
                    <label for="preheat-time">Orario Desiderato</label>
                    <input
                      id="preheat-time"
                      v-model="calcTargetTime"
                      type="time"
                      class="form-control form-control-sm"
                      @change="handleCalculateOptimalStart"
                    />
                  </div>
                  <div class="form-group">
                    <label for="preheat-temp">Temp Obiettivo (°C)</label>
                    <input
                      id="preheat-temp"
                      v-model.number="calcTargetTemp"
                      type="number"
                      step="0.5"
                      class="form-control form-control-sm"
                      @change="handleCalculateOptimalStart"
                    />
                  </div>
                </div>

                <div v-if="preheatResult" class="opt-result">
                  <div class="result-badge badge-preheat">
                    Anticipo Richiesto: <strong>{{ preheatResult.preheat_minutes }} min</strong>
                    (Avvio: {{ new Date(preheatResult.start_time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }})
                  </div>
                  <p class="result-reason">{{ preheatResult.reason }}</p>
                </div>
              </div>

              <!-- Coast-Down Card -->
              <div class="optimizer-card">
                <div class="opt-header">
                  <span class="opt-icon">🛑</span>
                  <div>
                    <h4 class="opt-title">Coast-Down (Inerzia & Stop)</h4>
                    <p class="opt-desc">Spegne il riscaldamento prima dell'uscita o del sonno sfruttando il volano termico</p>
                  </div>
                </div>

                <div class="opt-inputs">
                  <div class="form-group">
                    <label for="coast-time">Orario Uscita / Sonno</label>
                    <input
                      id="coast-time"
                      v-model="calcDepartureTime"
                      type="time"
                      class="form-control form-control-sm"
                      @change="handleCalculateCoastDown"
                    />
                  </div>
                </div>

                <div v-if="coastResult" class="opt-result">
                  <div class="result-badge badge-eco">
                    Spegnimento Anticipato: <strong>{{ coastResult.coast_down_minutes }} min</strong>
                    (Stop: {{ new Date(coastResult.stop_time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }})
                  </div>
                  <p class="result-reason">
                    {{ coastResult.reason }}
                  </p>
                </div>
              </div>
            </div>
          </section>

          <!-- 3. Allenatore Cicli (Simulazione o Registrazione) -->
          <section class="profile-section training-box" aria-label="Simulatore ciclo termico">
            <h3 class="section-heading">🧪 Allenamento Modello da Ciclo Reale</h3>
            <p class="section-subtext">
              Inserisci i dati osservati da una sessione per calibrare i parametri con livellamento esponenziale (EMA).
            </p>

            <div class="training-form-row">
              <div class="form-group">
                <label for="train-type">Tipo Ciclo</label>
                <select id="train-type" v-model="trainCycleType" class="form-control form-control-sm">
                  <option value="heating">Riscaldamento Attivo</option>
                  <option value="cooling">Raffreddamento Passivo</option>
                </select>
              </div>

              <div class="form-group">
                <label for="train-start">Temp Inizio (°C)</label>
                <input id="train-start" v-model.number="trainStartTemp" type="number" step="0.1" class="form-control form-control-sm" />
              </div>

              <div class="form-group">
                <label for="train-end">Temp Fine (°C)</label>
                <input id="train-end" v-model.number="trainEndTemp" type="number" step="0.1" class="form-control form-control-sm" />
              </div>

              <div class="form-group">
                <label for="train-dur">Durata (Minuti)</label>
                <input id="train-dur" v-model.number="trainDurationMin" type="number" min="15" step="5" class="form-control form-control-sm" />
              </div>

              <div class="form-group training-btn-group">
                <button
                  type="button"
                  class="btn btn-sm btn-primary"
                  :disabled="trainLoading"
                  @click="handleTrainCycle"
                >
                  {{ trainLoading ? 'Calcolo...' : '🚀 Registra Ciclo' }}
                </button>
              </div>
            </div>

            <div v-if="trainSuccessMsg" class="train-alert-success" role="alert">
              {{ trainSuccessMsg }}
            </div>
          </section>
        </div>

        <footer class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="showProfileModal = false">
            Chiudi
          </button>
        </footer>
      </div>
    </div>
  </div>
</template>

<style scoped>
.predictive-climate-container {
  padding: 1.5rem;
  max-width: 1300px;
  margin: 0 auto;
  color: var(--text-primary);
}

.view-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
  gap: 1rem;
}

.view-title {
  font-size: 1.75rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-main);
  /* gradient only in dark/oled themes where white text is readable */
  background: linear-gradient(135deg, var(--text-main) 0%, var(--text-muted) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.view-subtitle {
  color: var(--text-muted);
  font-size: 0.95rem;
  margin-top: 0.25rem;
}

.header-actions {
  display: flex;
  gap: 0.75rem;
}

/* KPI Banners */
.kpi-banner-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.kpi-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.25rem;
  border-radius: var(--radius-lg, 12px);
  background: var(--bg-card, rgba(30, 41, 59, 0.7));
  backdrop-filter: blur(12px);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
}

.kpi-icon {
  font-size: 2rem;
}

.kpi-value {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-primary);
}

.kpi-label {
  font-size: 0.85rem;
  color: var(--text-muted);
}

.alert-active {
  border-color: #f59e0b;
  background: rgba(245, 158, 11, 0.12);
}

.solar-active {
  border-color: #38bdf8;
  background: rgba(56, 189, 248, 0.12);
}

/* Zones Grid */
.zones-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  gap: 1.5rem;
}

.zone-card {
  border-radius: var(--radius-lg, 12px);
  background: var(--bg-card, rgba(30, 41, 59, 0.65));
  backdrop-filter: blur(16px);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  transition: transform 0.2s, box-shadow 0.2s;
}

.zone-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.3);
}

.zone-card.disabled {
  opacity: 0.6;
}

.zone-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.zone-title {
  font-size: 1.25rem;
  font-weight: 600;
  margin: 0 0 0.5rem 0;
}

.zone-tags {
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
}

.tag {
  font-size: 0.75rem;
  padding: 0.2rem 0.5rem;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-secondary);
}

.mode-badge {
  font-size: 0.8rem;
  padding: 0.3rem 0.6rem;
  border-radius: 999px;
  font-weight: 600;
}

.badge-cold-front {
  background: rgba(239, 68, 68, 0.2);
  color: #f87171;
  border: 1px solid rgba(239, 68, 68, 0.4);
}

.badge-solar {
  background: rgba(56, 189, 248, 0.2);
  color: #38bdf8;
  border: 1px solid rgba(56, 189, 248, 0.4);
}

.badge-comfort {
  background: rgba(34, 197, 94, 0.2);
  color: #4ade80;
  border: 1px solid rgba(34, 197, 94, 0.4);
}

.badge-frost {
  background: rgba(148, 163, 184, 0.2);
  color: #94a3b8;
  border: 1px solid rgba(148, 163, 184, 0.4);
}

.badge-preheat {
  background: rgba(249, 115, 22, 0.2);
  color: #fb923c;
  border: 1px solid rgba(249, 115, 22, 0.4);
}

.badge-eco {
  background: rgba(168, 85, 247, 0.2);
  color: #c084fc;
  border: 1px solid rgba(168, 85, 247, 0.4);
}

/* Metrics Row */
.zone-metrics-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 0.5rem;
  background: var(--bg-surface-elevated, rgba(0, 0, 0, 0.2));
  border: 1px solid var(--border-subtle);
  padding: 0.75rem;
  border-radius: 8px;
}

.metric-box {
  display: flex;
  flex-direction: column;
}

.metric-label {
  font-size: 0.7rem;
  color: var(--text-muted);
  text-transform: uppercase;
}

.metric-temp {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-main);
}

.metric-sub {
  font-size: 0.75rem;
  color: var(--text-secondary);
}

.metric-target {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-secondary);
}

.calculated-box .metric-setpoint {
  font-size: 1.35rem;
  font-weight: 800;
  color: #38bdf8;
}

/* Context Grid */
.zone-context-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.5rem;
  font-size: 0.85rem;
}

.context-item {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.context-val {
  font-weight: 600;
}

.context-sub {
  font-size: 0.75rem;
  color: var(--text-muted);
}

/* Reason Box */
.zone-reason {
  background: var(--primary-light, rgba(14, 165, 233, 0.08));
  border-left: 3px solid #38bdf8;
  padding: 0.5rem 0.75rem;
  border-radius: 0 6px 6px 0;
  font-size: 0.85rem;
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.reason-text {
  color: var(--text-secondary);
}

/* Footer Actions */
.zone-card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: auto;
  padding-top: 0.75rem;
  border-top: 1px solid var(--border-subtle);
}

.footer-end-actions {
  display: flex;
  gap: 0.25rem;
}

/* Buttons */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.6rem 1.2rem;
  border-radius: 8px;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  border: none;
  min-height: 44px;
  min-width: 44px;
  transition: opacity 0.2s, background 0.2s;
}

.btn-primary {
  background: var(--color-primary, #3b82f6);
  color: #fff;
}

.btn-secondary {
  background: var(--bg-surface-elevated, rgba(255, 255, 255, 0.08));
  border: 1px solid var(--border-card);
  color: var(--text-main);
}

.btn-outline {
  background: transparent;
  border: 1px solid var(--color-primary, #3b82f6);
  color: var(--color-primary, #3b82f6);
}

.btn-ghost {
  background: transparent;
  color: var(--text-secondary);
}

.btn-sm {
  min-height: 36px;
  padding: 0.35rem 0.75rem;
  font-size: 0.825rem;
}

.text-danger {
  color: #f87171 !important;
}

/* Modal */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-card {
  background: var(--bg-surface);
  border: var(--modal-border);
  border-radius: var(--radius-xl, 16px);
  width: 100%;
  max-width: 620px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: var(--modal-shadow);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.modal-header h2 {
  margin: 0;
  font-size: 1.35rem;
  font-weight: 700;
}

.btn-close {
  background: none;
  border: none;
  font-size: 1.75rem;
  color: var(--text-muted);
  cursor: pointer;
  width: 44px;
  height: 44px;
}

.modal-form {
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.form-group label {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.form-control {
  background: var(--bg-surface-elevated, rgba(15, 23, 42, 0.8));
  border: 1px solid var(--border-card);
  border-radius: 8px;
  padding: 0.65rem 0.85rem;
  color: var(--text-main);
  font-size: 0.95rem;
  min-height: 44px;
}

.form-control:focus-visible {
  outline: 2px solid #38bdf8;
  outline-offset: 2px;
}

.form-row {
  display: flex;
  gap: 1rem;
}

.form-row .col {
  flex: 1;
}

.toggles-box {
  background: rgba(0, 0, 0, 0.2);
  padding: 1rem;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.toggle-label {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-size: 0.9rem;
  cursor: pointer;
}

.toggle-label input[type='checkbox'] {
  width: 20px;
  height: 20px;
  cursor: pointer;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding-top: 1rem;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}

.empty-state {
  text-align: center;
  padding: 3rem;
  color: var(--text-muted);
  grid-column: 1 / -1;
  background: rgba(255, 255, 255, 0.02);
  border-radius: 12px;
}

.footer-start-actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  flex-wrap: wrap;
}

.modal-card-lg {
  max-width: 820px;
}

.modal-subtitle {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin: 0.25rem 0 0 0;
}

.profile-modal-content {
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.profile-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.section-heading {
  font-size: 1.05rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-primary);
}

.section-subtext {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin: 0;
}

.profile-metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 0.75rem;
}

.profile-metric-card {
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.p-label {
  font-size: 0.8rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.p-value {
  font-size: 1.4rem;
  font-weight: 800;
}

.heat-val {
  color: #f97316;
}

.cool-val {
  color: #38bdf8;
}

.samples-val {
  color: #a855f7;
}

.p-sub {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.optimizers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1rem;
}

.optimizer-card {
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.opt-header {
  display: flex;
  gap: 0.75rem;
  align-items: flex-start;
}

.opt-icon {
  font-size: 1.5rem;
}

.opt-title {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--text-primary);
}

.opt-desc {
  margin: 0.2rem 0 0 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
  line-height: 1.3;
}

.opt-inputs {
  display: flex;
  gap: 0.75rem;
}

.form-control-sm {
  min-height: 38px;
  padding: 0.4rem 0.65rem;
  font-size: 0.85rem;
}

.opt-result {
  background: rgba(0, 0, 0, 0.25);
  border-radius: 8px;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.result-badge {
  display: inline-block;
  padding: 0.3rem 0.6rem;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 600;
}

.result-reason {
  margin: 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
  line-height: 1.35;
}

.training-box {
  background: rgba(15, 23, 42, 0.5);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  padding: 1.25rem;
}

.training-form-row {
  display: flex;
  gap: 0.75rem;
  align-items: flex-end;
  flex-wrap: wrap;
  margin-top: 0.5rem;
}

.training-btn-group {
  margin-bottom: 0;
}

.train-alert-success {
  background: rgba(34, 197, 94, 0.15);
  border: 1px solid rgba(34, 197, 94, 0.3);
  color: #86efac;
  border-radius: 8px;
  padding: 0.65rem 0.85rem;
  font-size: 0.85rem;
  margin-top: 0.75rem;
}
</style>
