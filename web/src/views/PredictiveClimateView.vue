<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { usePredictiveClimateStore } from '../stores/predictiveClimate'
import type { PredictiveZoneConfig, RoomOrientation, EmitterType } from '../api/types'

const store = usePredictiveClimateStore()

const showModal = ref(false)
const editingZone = ref<PredictiveZoneConfig | null>(null)

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
          <button
            type="button"
            class="btn btn-sm btn-outline"
            :disabled="store.evaluating"
            @click="item.config.id && handleEvaluateZone(item.config.id)"
            aria-label="Calcola setpoint per questa stanza"
          >
            ⚡ Valuta Ora
          </button>
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
  background: linear-gradient(135deg, #fff 0%, #cbd5e1 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
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
  background: rgba(0, 0, 0, 0.2);
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
  color: #fff;
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
  background: rgba(255, 255, 255, 0.04);
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
  border-top: 1px solid rgba(255, 255, 255, 0.06);
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
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
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
  background: var(--bg-surface, #1e293b);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-xl, 16px);
  width: 100%;
  max-width: 620px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
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
  background: rgba(15, 23, 42, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 8px;
  padding: 0.65rem 0.85rem;
  color: #fff;
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
</style>
