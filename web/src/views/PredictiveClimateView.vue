<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePredictiveClimateStore } from '../stores/predictiveClimate'
import { apiClient } from '../api/client'
import type {
  PredictiveZoneConfig,
  Entity,
  RoomOrientation,
  EmitterType,
  ThermalProfile,
  PreheatRecommendation,
  CoastDownRecommendation
} from '../api/types'

const { t, locale } = useI18n()

const store = usePredictiveClimateStore()

const showModal = ref(false)
const editingZone = ref<PredictiveZoneConfig | null>(null)
const availableEntities = ref<Entity[]>([])
const entityLoadError = ref('')
const formError = ref('')
const formSaving = ref(false)
const profileError = ref('')

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
const trainingSampleValid = computed(() => {
  const start = Number(trainStartTemp.value)
  const end = Number(trainEndTemp.value)
  const duration = Number(trainDurationMin.value)
  const delta = end - start
  const maxRate = trainCycleType.value === 'heating' ? 8 : 3
  return Number.isFinite(start) && Number.isFinite(end) && start >= -20 && start <= 50 && end >= -20 && end <= 50 &&
    Number.isInteger(duration) && duration >= 15 && duration <= 720 &&
    (trainCycleType.value === 'heating' ? delta > 0 : delta < 0) && Math.abs(delta) / (duration / 60) <= maxRate
})

// Form fields
const formName = ref('')
const formTRVKey = ref('')
const formTempKey = ref('')
const formPressureKey = ref('')
const formHumidityKey = ref('')
const formWindowKey = ref('')
const formPresenceKey = ref('')
const formAwayTemp = ref(17.0)
const formTargetTemp = ref(21.0)
const formOrientation = ref<RoomOrientation>('south')
const formWindowArea = ref(4.0)
const formEmitter = ref<EmitterType>('radiator')
const formInertia = ref(2.0)
const formPreheat = ref(true)
const formSolarCutoff = ref(true)
const formPressureBoost = ref(true)
const formEnabled = ref(true)

const climateEntityOptions = computed(() => availableEntities.value.filter(entity => entity.domain === 'climate' && entityKey(entity)))
const temperatureEntityOptions = computed(() => availableEntities.value.filter(entity => {
  if (entity.domain !== 'sensor' || !entityKey(entity)) return false
  const unit = String(entity.attributes?.unit_of_measurement || '').toLowerCase()
  return String(entity.attributes?.device_class || '').toLowerCase() === 'temperature' ||
    ['°c', 'c', 'celsius', '°f', 'f', 'fahrenheit'].includes(unit)
}))
const sensorEntityOptions = computed(() => availableEntities.value.filter(entity => entity.domain === 'sensor' && entityKey(entity)))
const contactEntityOptions = computed(() => availableEntities.value.filter(entity => entity.domain === 'binary_sensor' && entityKey(entity)))

function entityKey(entity: Entity): string {
  if (entity.entity_key) return entity.entity_key
  return entity.id.startsWith(entity.domain + '.') ? entity.id : ''
}

function entityOptionLabel(entity: Entity): string {
  return entity.name ? entity.name + ' (' + entityKey(entity) + ')' : entityKey(entity)
}

function formatEvaluationTime(value?: string): string {
  if (!value) return t('predictiveClimate.view.neverEvaluated')
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return t('predictiveClimate.view.dateUnavailable')
  return t('predictiveClimate.view.lastEvaluated', { date: date.toLocaleString(locale.value) })
}

function nextLocalTime(timeValue: string, fallbackHour: number, fallbackMinute: number): Date {
  const parts = timeValue.split(':').map(Number)
  const hour = Number.isInteger(parts[0]) ? parts[0] : fallbackHour
  const minute = Number.isInteger(parts[1]) ? parts[1] : fallbackMinute
  const date = new Date()
  date.setHours(hour, minute, 0, 0)
  if (date.getTime() <= Date.now()) date.setDate(date.getDate() + 1)
  return date
}

onMounted(async () => {
  await Promise.all([
    store.fetchStatus(),
    Promise.resolve().then(() => apiClient.get<Entity[]>('/api/v1/entities')).then(items => { availableEntities.value = items || [] }).catch(() => {
      entityLoadError.value = t('predictiveClimate.view.entityListUnavailable')
    })
  ])
})

async function openThermalProfileModal(zone: PredictiveZoneConfig) {
  profileError.value = ''
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
  } catch (e: any) {
    profileError.value = t('predictiveClimate.view.profileLoadFailed')
    console.error('Failed to load thermal profile', e)
  } finally {
    profileLoading.value = false
  }
}

async function handleCalculateOptimalStart() {
  if (!selectedProfileZone.value?.id) return
  try {
    const targetDate = nextLocalTime(calcTargetTime.value, 7, 30)
    preheatResult.value = await store.getOptimalStart(
      selectedProfileZone.value.id,
      targetDate.toISOString(),
      Number(calcTargetTemp.value)
    )
  } catch (err: any) {
    profileError.value = t('predictiveClimate.view.preheatFailed')
    console.error('Error calculating preheat', err)
  }
}

async function handleCalculateCoastDown() {
  if (!selectedProfileZone.value?.id) return
  try {
    const depDate = nextLocalTime(calcDepartureTime.value, 8, 30)
    coastResult.value = await store.getOptimalStop(
      selectedProfileZone.value.id,
      depDate.toISOString()
    )
  } catch (err: any) {
    profileError.value = t('predictiveClimate.view.coastDownFailed')
    console.error('Error calculating coast down', err)
  }
}

async function handleTrainCycle() {
  if (!selectedProfileZone.value?.id) return
  profileError.value = ''
  if (!trainingSampleValid.value) {
    profileError.value = t('predictiveClimate.view.invalidSample')
    return
  }
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
    trainSuccessMsg.value = t('predictiveClimate.view.modelUpdated', { rate: updated.heat_rate_per_hr, samples: updated.samples_count })
    await handleCalculateOptimalStart()
    await handleCalculateCoastDown()
  } catch (err: any) {
    profileError.value = t('predictiveClimate.view.trainingFailed')
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
  formPresenceKey.value = ''
  formAwayTemp.value = 17.0
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
  formPresenceKey.value = cfg.presence_sensor_key || ''
  formAwayTemp.value = cfg.away_temperature || 17.0
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
  formError.value = ''
  if (!formName.value.trim() || !formTRVKey.value.trim() || !formTempKey.value.trim()) {
    formError.value = t('predictiveClimate.view.requiredFields')
    return
  }
  if (!Number.isFinite(formTargetTemp.value) || formTargetTemp.value < 10 || formTargetTemp.value > 30) {
    formError.value = t('predictiveClimate.view.targetTemperatureRange')
    return
  }
  if (!Number.isFinite(formAwayTemp.value) || formAwayTemp.value < 15 || formAwayTemp.value > 28) {
    formError.value = t('predictiveClimate.view.awayTemperatureRange')
    return
  }
  if (!Number.isFinite(formWindowArea.value) || formWindowArea.value < 0 || formWindowArea.value > 50) {
    formError.value = t('predictiveClimate.view.windowAreaRange')
    return
  }

  const payload: PredictiveZoneConfig = {
    id: editingZone.value?.id,
    name: formName.value.trim(),
    trv_entity_key: formTRVKey.value.trim(),
    temp_sensor_key: formTempKey.value.trim(),
    pressure_sensor_key: formPressureKey.value.trim() || undefined,
    humidity_sensor_key: formHumidityKey.value.trim() || undefined,
    window_sensor_key: formWindowKey.value.trim() || undefined,
    presence_sensor_key: formPresenceKey.value.trim() || undefined,
    away_temperature: Number(formAwayTemp.value),
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

  formSaving.value = true
  try {
    await store.saveZone(payload)
    showModal.value = false
  } catch (err: any) {
    formError.value = t('predictiveClimate.view.saveFailed')
  } finally {
    formSaving.value = false
  }
}

async function handleDeleteZone(id: string) {
  if (confirm(t('predictiveClimate.view.confirmDeleteZone'))) {
    try { await store.deleteZone(id) } catch { /* store.error is rendered in the page */ }
  }
}

async function handleEvaluateZone(id: string) {
  try { await store.evaluateZone(id) } catch { /* store.error is rendered in the page */ }
}

async function handleEvaluateAll() {
  try { await store.evaluateAll() } catch { /* store.error is rendered in the page */ }
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
      return t('predictiveClimate.view.modeLabels.coldFront')
    case 'solar_cutoff':
      return t('predictiveClimate.view.modeLabels.solarCutoff')
    case 'frost_protection':
      return t('predictiveClimate.view.modeLabels.frostProtection')
    case 'preheat':
      return t('predictiveClimate.view.modeLabels.preheat')
    case 'eco':
      return t('predictiveClimate.view.modeLabels.eco')
    default:
      return t('predictiveClimate.view.modeLabels.comfort')
  }
}

function getOrientationLabel(orient: RoomOrientation): string {
  switch (orient) {
    case 'south':
      return t('predictiveClimate.view.orientations.south')
    case 'north':
      return t('predictiveClimate.view.orientations.north')
    case 'east':
      return t('predictiveClimate.view.orientations.east')
    case 'west':
      return t('predictiveClimate.view.orientations.west')
  }
}

function getEmitterLabel(emitter: EmitterType): string {
  switch (emitter) {
    case 'floor_radiant':
      return t('predictiveClimate.view.emitters.floorRadiant')
    case 'fancoil':
      return t('predictiveClimate.view.emitters.fancoil')
    default:
      return t('predictiveClimate.view.emitters.radiator')
  }
}
</script>

<template>
  <div class="predictive-climate-container" role="main" :aria-label="t('predictiveClimate.view.mainLabel')">
    <p v-if="store.error" class="climate-feedback climate-feedback-error" role="alert">{{ store.error }}</p>
    <p v-if="entityLoadError" class="climate-feedback" role="status">{{ entityLoadError }}</p>
    <!-- Header -->
    <header class="view-header">
      <div class="title-section">
        <h1 class="view-title">{{ t('predictiveClimate.view.title') }}</h1>
        <p class="view-subtitle">
          {{ t('predictiveClimate.view.subtitle') }}
        </p>
      </div>

      <div class="header-actions">
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="store.evaluating"
          @click="handleEvaluateAll"
          :aria-label="t('predictiveClimate.view.reassessAllZones')"
        >
          {{ store.evaluating ? t('predictiveClimate.view.calculating') : '⚡ ' + t('predictiveClimate.view.evaluateAllZones') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          @click="openCreateModal"
          :aria-label="t('predictiveClimate.view.addZone')"
        >
          + {{ t('predictiveClimate.view.addZoneCta') }}
        </button>
      </div>
    </header>

    <!-- Top KPI Banners -->
    <section class="kpi-banner-grid" :aria-label="t('predictiveClimate.view.environmentSummary')">
      <div class="kpi-card">
        <div class="kpi-icon">🌡️</div>
        <div class="kpi-content">
          <div class="kpi-value">{{ store.enabledZones }} / {{ store.totalZones }}</div>
          <div class="kpi-label">{{ t('predictiveClimate.view.activeZones') }}</div>
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
            {{ store.coldFrontActive ? t('predictiveClimate.view.coldFrontDetected') : t('predictiveClimate.view.pressureStable') }}
          </div>
          <div class="kpi-label">
            {{ store.coldFrontActive ? t('predictiveClimate.view.barometricPreheatActive') : t('predictiveClimate.view.noBarometricDrop') }}
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
            {{ store.solarCutoffActive ? t('predictiveClimate.view.solarCutoffActive') : t('predictiveClimate.view.moderateSun') }}
          </div>
          <div class="kpi-label">
            {{ store.solarCutoffActive ? t('predictiveClimate.view.solarSavingsActive') : t('predictiveClimate.view.standardSolarGain') }}
          </div>
        </div>
      </div>
    </section>

    <!-- Zone Cards Grid -->
    <section class="zones-grid" :aria-label="t('predictiveClimate.view.thermalZones')">
      <div v-if="store.loading && store.zoneStatuses.length === 0" class="empty-state">
        {{ t('predictiveClimate.view.loadingZones') }}
      </div>

      <div v-else-if="store.zoneStatuses.length === 0" class="empty-state">
        <p>{{ t('predictiveClimate.view.noZones') }}</p>
        <button type="button" class="btn btn-primary" @click="openCreateModal">
          {{ t('predictiveClimate.view.configureFirstZone') }}
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
              <span class="tag tag-window">{{ item.config.window_area_sqm }} m² {{ t('predictiveClimate.view.glazedAreaLabel') }}</span>
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
            <span class="metric-label">{{ t('predictiveClimate.view.roomPerceived') }}</span>
            <span class="metric-temp">
              {{ item.latest_evaluation ? item.latest_evaluation.current_temp : '--' }} °C
            </span>
            <span v-if="item.latest_evaluation" class="metric-sub">
              ({{ t('predictiveClimate.view.perceived') }} {{ item.latest_evaluation.apparent_temp }} °C)
            </span>
          </div>

          <div class="metric-box target-box">
            <span class="metric-label">{{ t('predictiveClimate.view.baseTarget') }}</span>
            <span class="metric-target">{{ item.config.target_temperature }} °C</span>
          </div>

          <div class="metric-box calculated-box">
            <span class="metric-label">{{ t('predictiveClimate.view.calculatedSetpoint') }}</span>
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
                {{ t('predictiveClimate.view.pressureTrend') }}: {{ item.latest_evaluation.pressure_trend_hpa_hr > 0 ? '+' : '' }}{{ item.latest_evaluation.pressure_trend_hpa_hr }} hPa/h
              </div>
            </div>
          </div>

          <div class="context-item">
            <span class="context-icon">☀️</span>
            <div>
              <div class="context-val">{{ item.latest_evaluation.direct_radiation_forecast }} W/m²</div>
              <div class="context-sub">
                {{ t('predictiveClimate.view.gain') }}: +{{ item.latest_evaluation.estimated_solar_gain_kw }} kW
              </div>
            </div>
          </div>

          <div class="context-item">
            <span class="context-icon">💧</span>
            <div>
              <div class="context-val">{{ item.latest_evaluation.humidity }}% UR</div>
              <div class="context-sub">{{ t('predictiveClimate.view.estimated') }} {{ item.latest_evaluation.outdoor_temp_forecast }} °C</div>
            </div>
          </div>
        </div>

        <div class="evaluation-freshness">
          {{ formatEvaluationTime(item.latest_evaluation?.evaluated_at) }}
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
              :aria-label="t('predictiveClimate.view.calculateSetpoint')"
            >
              ⚡ {{ t('predictiveClimate.view.evaluateNow') }}
            </button>
            <button
              type="button"
              class="btn btn-sm btn-secondary"
              @click="openThermalProfileModal(item.config)"
              :aria-label="t('predictiveClimate.view.viewThermalModel')"
            >
              🧠 {{ t('predictiveClimate.view.thermalModelStart') }}
            </button>
          </div>
          <div class="footer-end-actions">
            <button
              type="button"
              class="btn btn-sm btn-ghost"
              @click="openEditModal(item.config)"
              :aria-label="t('predictiveClimate.view.editZone')"
            >
              {{ t('predictiveClimate.view.editAction') }}
            </button>
            <button
              type="button"
              class="btn btn-sm btn-ghost text-danger"
              @click="item.config.id && handleDeleteZone(item.config.id)"
              :aria-label="t('predictiveClimate.view.deleteZone')"
            >
              {{ t('predictiveClimate.view.deleteAction') }}
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
          <h2 id="modal-title">{{ editingZone ? t('predictiveClimate.view.editZoneTitle') : t('predictiveClimate.view.newZoneTitle') }}</h2>
          <button
            type="button"
            class="btn-close"
            @click="showModal = false"
            :aria-label="t('common.close')"
          >
            &times;
          </button>
        </header>

        <form @submit.prevent="handleSaveZone" class="modal-form">
          <div class="form-group">
            <label for="zone-name">{{ t('predictiveClimate.view.zoneName') }}</label>
            <input
              id="zone-name"
              v-model="formName"
              type="text"
              class="form-control"
              :placeholder="t('predictiveClimate.view.zoneNamePlaceholder')"
              required
            />
          </div>

          <div class="form-row">
            <div class="form-group col">
              <label for="zone-trv">{{ t('predictiveClimate.view.trvEntity') }}</label>
              <input
                id="zone-trv"
                v-model="formTRVKey"
                type="text"
                class="form-control"
                :placeholder="t('predictiveClimate.view.trvPlaceholder')"
                list="climate-entity-options"
                autocomplete="off"
                required
              />
              <datalist id="climate-entity-options">
                <option v-for="entity in climateEntityOptions" :key="entity.id" :value="entityKey(entity)">{{ entityOptionLabel(entity) }}</option>
              </datalist>
            </div>

            <div class="form-group col">
              <label for="zone-temp">{{ t('predictiveClimate.view.roomTemperatureSensor') }}</label>
              <input
                id="zone-temp"
                v-model="formTempKey"
                type="text"
                class="form-control"
                :placeholder="t('predictiveClimate.view.temperaturePlaceholder')"
                list="temperature-entity-options"
                autocomplete="off"
                required
              />
              <datalist id="temperature-entity-options">
                <option v-for="entity in temperatureEntityOptions" :key="entity.id" :value="entityKey(entity)">{{ entityOptionLabel(entity) }}</option>
              </datalist>
            </div>
          </div>

          <div class="form-row">
            <div class="form-group col">
              <label for="zone-press">{{ t('predictiveClimate.view.pressureSensor') }}</label>
              <input
                id="zone-press"
                v-model="formPressureKey"
                type="text"
                class="form-control"
                :placeholder="t('predictiveClimate.view.pressurePlaceholder')"
                list="sensor-entity-options"
                autocomplete="off"
              />
              <datalist id="sensor-entity-options">
                <option v-for="entity in sensorEntityOptions" :key="entity.id" :value="entityKey(entity)">{{ entityOptionLabel(entity) }}</option>
              </datalist>
            </div>

            <div class="form-group col">
              <label for="zone-hum">{{ t('predictiveClimate.view.humiditySensor') }}</label>
              <input
                id="zone-hum"
                v-model="formHumidityKey"
                type="text"
                class="form-control"
                :placeholder="t('predictiveClimate.view.humidityPlaceholder')"
                list="sensor-entity-options"
                autocomplete="off"
              />
            </div>
          </div>

          <div class="form-group">
            <label for="zone-win">{{ t('predictiveClimate.view.windowContactSensor') }}</label>
            <input
              id="zone-win"
              v-model="formWindowKey"
              type="text"
              class="form-control"
              :placeholder="t('predictiveClimate.view.windowPlaceholder')"
              list="contact-entity-options"
              autocomplete="off"
            />
            <datalist id="contact-entity-options">
              <option v-for="entity in contactEntityOptions" :key="entity.id" :value="entityKey(entity)">{{ entityOptionLabel(entity) }}</option>
            </datalist>
          </div>

          <div class="form-row">
            <div class="form-group col">
              <label for="zone-presence">{{ t('predictiveClimate.view.presenceSensorOptional') }}</label>
              <input
                id="zone-presence"
                v-model="formPresenceKey"
                type="text"
                class="form-control"
                :placeholder="t('predictiveClimate.view.presencePlaceholder')"
                list="contact-entity-options"
                autocomplete="off"
              />
              <small>{{ t('predictiveClimate.view.emptyRoomHint') }}</small>
            </div>
            <div class="form-group col">
              <label for="zone-away-temp">{{ t('predictiveClimate.view.awayTemperature') }} (°C)</label>
              <input
                id="zone-away-temp"
                v-model.number="formAwayTemp"
                type="number"
                min="15"
                max="28"
                step="0.5"
                class="form-control"
              />
            </div>
          </div>

          <div class="form-row">
            <div class="form-group col">
              <label for="zone-target">{{ t('predictiveClimate.view.targetTemperature') }} (°C)</label>
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
              <label for="zone-orient">{{ t('predictiveClimate.view.mainExposure') }}</label>
              <select id="zone-orient" v-model="formOrientation" class="form-control">
                <option value="south">{{ t('predictiveClimate.view.south') }}</option>
                <option value="east">{{ t('predictiveClimate.view.east') }}</option>
                <option value="west">{{ t('predictiveClimate.view.west') }}</option>
                <option value="north">{{ t('predictiveClimate.view.north') }}</option>
              </select>
            </div>
          </div>

          <div class="form-row">
            <div class="form-group col">
              <label for="zone-emitter">{{ t('predictiveClimate.view.heatingEmitter') }}</label>
              <select id="zone-emitter" v-model="formEmitter" class="form-control">
                <option value="radiator">{{ t('predictiveClimate.view.radiator') }}</option>
                <option value="floor_radiant">{{ t('predictiveClimate.view.floorHeating') }}</option>
                <option value="fancoil">{{ t('predictiveClimate.view.fanCoil') }}</option>
              </select>
            </div>

            <div class="form-group col">
              <label for="zone-area">{{ t('predictiveClimate.view.glazedArea') }} (m²)</label>
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
              <span>{{ t('predictiveClimate.view.coldFrontForecast') }}</span>
            </label>

            <label class="toggle-label">
              <input type="checkbox" v-model="formSolarCutoff" />
              <span>{{ t('predictiveClimate.view.solarCutoff') }}</span>
            </label>

            <label class="toggle-label">
              <input type="checkbox" v-model="formPreheat" />
              <span>{{ t('predictiveClimate.view.weatherPreheat') }}</span>
            </label>

            <label class="toggle-label">
              <input type="checkbox" v-model="formEnabled" />
              <span>{{ t('predictiveClimate.view.zoneActive') }}</span>
            </label>
          </div>

          <p v-if="formError" class="climate-feedback climate-feedback-error" role="alert">{{ formError }}</p>
          <footer class="modal-footer">
            <button type="button" class="btn btn-secondary" @click="showModal = false" :disabled="formSaving">
              {{ t('common.cancel') }}
            </button>
            <button type="submit" class="btn btn-primary" :disabled="formSaving">
              {{ formSaving ? t('common.loading') : t('predictiveClimate.view.saveConfiguration') }}
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
              🧠 {{ t('predictiveClimate.view.profileTitle') }}: {{ selectedProfileZone.name }}
            </h2>
            <p class="modal-subtitle">
              {{ t('predictiveClimate.view.profileSubtitle') }}
            </p>
          </div>
          <button
            type="button"
            class="btn-close"
            @click="showProfileModal = false"
            :aria-label="t('common.close')"
          >
            {{ t('common.close') }}
          </button>
        </header>

        <p v-if="profileError" class="climate-feedback climate-feedback-error" role="alert">{{ profileError }}</p>
        <div v-if="profileLoading" class="modal-body modal-loading">
          {{ t('predictiveClimate.view.loadingProfile') }}
        </div>

        <div v-else class="modal-body profile-modal-content">
          <!-- 1. Parametri Appresi -->
          <section class="profile-section" :aria-label="t('predictiveClimate.view.learnedThermalParameters')">
            <h3 class="section-heading">📊 {{ t('predictiveClimate.view.learnedParametersTitle') }}</h3>
            <div class="profile-metrics-grid">
              <div class="profile-metric-card">
                <span class="p-label">{{ t('predictiveClimate.view.heatingRate') }}</span>
                <span class="p-value heat-val">
                  {{ currentProfile ? currentProfile.heat_rate_per_hr : '--' }} °C/h
                </span>
                <span class="p-sub">{{ t('predictiveClimate.view.nominalActivePower') }}</span>
              </div>

              <div class="profile-metric-card">
                <span class="p-label">{{ t('predictiveClimate.view.heatLossRate') }}</span>
                <span class="p-value cool-val">
                  {{ currentProfile ? currentProfile.cool_rate_per_hr : '--' }} °C/h
                </span>
                <span class="p-sub">{{ t('predictiveClimate.view.normalizedDelta') }}</span>
              </div>

              <div class="profile-metric-card">
                <span class="p-label">{{ t('predictiveClimate.view.learningCycles') }}</span>
                <span class="p-value samples-val">
                  {{ currentProfile ? currentProfile.samples_count : 0 }}
                </span>
                <span class="p-sub">{{ t('predictiveClimate.view.emaUpdates') }}</span>
              </div>
            </div>
          </section>

          <!-- 2. Ottimizzatori: Pre-heating e Coast-Down -->
          <section class="profile-section" :aria-label="t('predictiveClimate.view.dynamicCalculators')">
            <div class="optimizers-grid">
              <!-- Pre-heating Card -->
              <div class="optimizer-card">
                <div class="opt-header">
                  <span class="opt-icon">⚡</span>
                  <div>
                    <h4 class="opt-title">{{ t('predictiveClimate.view.preheatingTitle') }}</h4>
                    <p class="opt-desc">{{ t('predictiveClimate.view.preheatingDesc') }}</p>
                  </div>
                </div>

                <div class="opt-inputs">
                  <div class="form-group">
                    <label for="preheat-time">{{ t('predictiveClimate.view.desiredTime') }}</label>
                    <input
                      id="preheat-time"
                      v-model="calcTargetTime"
                      type="time"
                      class="form-control form-control-sm"
                      @change="handleCalculateOptimalStart"
                    />
                  </div>
                  <div class="form-group">
                    <label for="preheat-temp">{{ t('predictiveClimate.view.targetTemperature') }} (°C)</label>
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
                    {{ t('predictiveClimate.view.requiredPreheat') }}: <strong>{{ preheatResult.preheat_minutes }} min</strong>
                    ({{ t('predictiveClimate.view.startsAt') }}: {{ new Date(preheatResult.start_time).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' }) }})
                  </div>
                  <p class="result-reason">
                    {{ preheatResult.preheat_minutes === 0
                      ? t('predictiveClimate.view.preheatNoAdvanceReason', {
                          current: preheatResult.current_temp.toFixed(1),
                          target: preheatResult.target_temp.toFixed(1)
                        })
                      : t('predictiveClimate.view.preheatScheduledReason', {
                          minutes: preheatResult.preheat_minutes,
                          start: new Date(preheatResult.start_time).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' }),
                          target: preheatResult.target_temp.toFixed(1),
                          targetTime: new Date(preheatResult.target_time).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' }),
                          rate: preheatResult.effective_heat_rate.toFixed(2),
                          outdoor: preheatResult.outdoor_temp.toFixed(1)
                        }) }}
                  </p>
                </div>
              </div>

              <!-- Coast-Down Card -->
              <div class="optimizer-card">
                <div class="opt-header">
                  <span class="opt-icon">🛑</span>
                  <div>
                    <h4 class="opt-title">{{ t('predictiveClimate.view.coastDownTitle') }}</h4>
                    <p class="opt-desc">{{ t('predictiveClimate.view.coastDownDesc') }}</p>
                  </div>
                </div>

                <div class="opt-inputs">
                  <div class="form-group">
                    <label for="coast-time">{{ t('predictiveClimate.view.departureSleepTime') }}</label>
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
                    {{ t('predictiveClimate.view.earlyShutdown') }}: <strong>{{ coastResult.coast_down_minutes }} min</strong>
                    ({{ t('predictiveClimate.view.stopsAt') }}: {{ new Date(coastResult.stop_time).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' }) }})
                  </div>
                  <p class="result-reason">
                    {{ coastResult.coast_down_minutes === 0
                      ? t('predictiveClimate.view.coastDownNoShutdownReason', {
                          limit: coastResult.comfort_limit_temp.toFixed(1)
                        })
                      : t('predictiveClimate.view.coastDownScheduledReason', {
                          minutes: coastResult.coast_down_minutes,
                          stop: new Date(coastResult.stop_time).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' }),
                          departure: new Date(coastResult.departure_time).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' }),
                          energy: coastResult.energy_saved_kwh.toFixed(2)
                        }) }}
                  </p>
                </div>
              </div>
            </div>
          </section>

          <!-- 3. Allenatore Cicli (Simulazione o Registrazione) -->
          <section class="profile-section training-box" :aria-label="t('predictiveClimate.view.thermalCycleSimulator')">
            <h3 class="section-heading">🧪 {{ t('predictiveClimate.view.modelTrainingTitle') }}</h3>
            <p class="section-subtext">
              {{ t('predictiveClimate.view.cycleTrainingDescription') }}
            </p>

            <div class="training-form-row">
              <div class="form-group">
                <label for="train-type">{{ t('predictiveClimate.view.cycleType') }}</label>
                <select id="train-type" v-model="trainCycleType" class="form-control form-control-sm">
                  <option value="heating">{{ t('predictiveClimate.view.activeHeating') }}</option>
                  <option value="cooling">{{ t('predictiveClimate.view.passiveCooling') }}</option>
                </select>
              </div>

              <div class="form-group">
                <label for="train-start">{{ t('predictiveClimate.view.startTemperature') }} (°C)</label>
                <input id="train-start" v-model.number="trainStartTemp" type="number" min="-20" max="50" step="0.1" class="form-control form-control-sm" />
              </div>

              <div class="form-group">
                <label for="train-end">{{ t('predictiveClimate.view.endTemperature') }} (°C)</label>
                <input id="train-end" v-model.number="trainEndTemp" type="number" min="-20" max="50" step="0.1" class="form-control form-control-sm" />
              </div>

              <div class="form-group">
                <label for="train-dur">{{ t('predictiveClimate.view.duration') }} ({{ t('predictiveClimate.view.minutes') }})</label>
                <input id="train-dur" v-model.number="trainDurationMin" type="number" min="15" max="720" step="5" class="form-control form-control-sm" />
              </div>

              <div class="form-group training-btn-group">
                <button
                  type="button"
                  class="btn btn-sm btn-primary"
                  :disabled="trainLoading || !trainingSampleValid"
                  @click="handleTrainCycle"
                >
                  {{ trainLoading ? t('common.loading') : t('predictiveClimate.view.recordCycle') }}
                </button>
              </div>
            </div>

            <p class="section-subtext">{{ t('predictiveClimate.view.trainingHelp') }}</p>
            <div v-if="trainSuccessMsg" class="train-alert-success" role="status">
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

.climate-feedback { padding: 0.75rem 1rem; border-radius: 0.5rem; margin: 0.75rem 0; background: var(--surface-secondary, #f3f4f6); }
.climate-feedback-error { color: var(--color-danger, #b42318); background: var(--surface-danger, #fef3f2); }
.evaluation-freshness { margin-top: 0.5rem; font-size: 0.8rem; color: var(--text-secondary, #667085); }

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

/* Responsive layout: prevent intrinsic content sizes from widening the page. */
.predictive-climate-container,
.predictive-climate-container * {
  box-sizing: border-box;
}

.predictive-climate-container {
  width: 100%;
  min-width: 0;
  padding-inline: clamp(0.75rem, 2.5vw, 1.5rem);
}

.title-section,
.kpi-content,
.zone-card-header > div,
.metric-box,
.context-item > div,
.zone-reason,
.reason-text,
.modal-header > div,
.form-group,
.form-row .col,
.profile-section,
.optimizer-card,
.opt-header > div {
  min-width: 0;
}

.view-header,
.zone-card-header,
.zone-card-footer,
.header-actions,
.footer-start-actions,
.footer-end-actions {
  flex-wrap: wrap;
}

.header-actions {
  max-width: 100%;
}

.header-actions .btn {
  white-space: normal;
}

.kpi-banner-grid {
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 250px), 1fr));
}

.zones-grid {
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 340px), 1fr));
  min-width: 0;
}

.zone-card {
  min-width: 0;
}

.zone-card-header {
  gap: 0.75rem;
}

.mode-badge,
.tag,
.context-val,
.context-sub,
.metric-label,
.metric-sub,
.reason-text,
.result-badge,
.result-reason,
.p-label,
.p-sub,
.section-subtext {
  overflow-wrap: anywhere;
  word-break: normal;
}

.zone-metrics-row {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  min-width: 0;
}

.zone-context-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  min-width: 0;
}

.zone-card-footer {
  gap: 0.75rem;
}

.footer-start-actions,
.footer-end-actions {
  min-width: 0;
}

.modal-backdrop {
  box-sizing: border-box;
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding:
    max(0.75rem, env(safe-area-inset-top))
    max(0.75rem, env(safe-area-inset-right))
    max(0.75rem, env(safe-area-inset-bottom))
    max(0.75rem, env(safe-area-inset-left));
  align-items: center;
}

.modal-card {
  box-sizing: border-box;
  width: min(100%, 620px);
  min-width: 0;
  max-width: 100%;
  max-height: calc(100vh - 1.5rem);
  max-height: calc(100dvh - 1.5rem);
  margin: auto;
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.modal-card-lg {
  width: min(100%, 820px);
  max-width: 100%;
}

.modal-header {
  gap: 0.75rem;
  min-width: 0;
}

.modal-header h2 {
  min-width: 0;
  overflow-wrap: anywhere;
}

.modal-form,
.profile-modal-content,
.modal-body {
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
}

.form-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
  min-width: 0;
}

.form-row .col {
  width: 100%;
}

.form-control {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
  text-overflow: ellipsis;
}

.modal-footer {
  flex-wrap: wrap;
  min-width: 0;
}

.modal-footer .btn {
  max-width: 100%;
  white-space: normal;
}

.profile-metrics-grid {
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 165px), 1fr));
}

.optimizers-grid {
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 260px), 1fr));
  min-width: 0;
}

.optimizer-card,
.profile-metric-card,
.training-box {
  min-width: 0;
}

.opt-inputs {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 170px), 1fr));
  min-width: 0;
}

.opt-inputs .form-group {
  min-width: 0;
}

.training-form-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 155px), 1fr));
  min-width: 0;
}

.training-form-row .form-group {
  min-width: 0;
}

@media (max-width: 760px) {
  .predictive-climate-container {
    padding: 0.85rem;
  }

  .view-header {
    align-items: stretch;
  }

  .title-section {
    width: 100%;
  }

  .view-title {
    font-size: clamp(1.35rem, 6vw, 1.75rem);
    overflow-wrap: anywhere;
  }

  .view-subtitle {
    line-height: 1.45;
  }

  .header-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    width: 100%;
  }

  .header-actions .btn {
    width: 100%;
  }

  .zones-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: 1rem;
  }

  .zone-card {
    padding: 1rem;
  }

  .zone-card-header {
    align-items: flex-start;
  }

  .zone-card-footer {
    align-items: stretch;
  }

  .footer-start-actions,
  .footer-end-actions {
    width: 100%;
  }

  .footer-start-actions .btn,
  .footer-end-actions .btn {
    flex: 1 1 auto;
  }

  .profile-modal-content {
    padding: 1rem;
    gap: 1rem;
  }

  .modal-header {
    padding: 1rem;
  }

  .modal-form {
    padding: 1rem;
  }

  .training-box {
    padding: 1rem;
  }
}

@media (max-width: 520px) {
  .header-actions {
    grid-template-columns: minmax(0, 1fr);
  }

  .kpi-banner-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: 0.75rem;
  }

  .kpi-card {
    padding: 1rem;
  }

  .kpi-value {
    font-size: 1.05rem;
  }

  .zone-card-header {
    flex-direction: column;
  }

  .zone-metrics-row {
    gap: 0.35rem;
    padding: 0.6rem 0.45rem;
  }

  .metric-label {
    font-size: 0.62rem;
  }

  .metric-temp,
  .metric-target {
    font-size: 1.05rem;
  }

  .calculated-box .metric-setpoint {
    font-size: 1.1rem;
  }

  .zone-context-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .zone-card-footer,
  .footer-start-actions,
  .footer-end-actions {
    flex-direction: column;
  }

  .footer-start-actions .btn,
  .footer-end-actions .btn {
    width: 100%;
  }

  .modal-backdrop {
    padding:
      max(0.4rem, env(safe-area-inset-top))
      max(0.4rem, env(safe-area-inset-right))
      max(0.4rem, env(safe-area-inset-bottom))
      max(0.4rem, env(safe-area-inset-left));
    align-items: flex-start;
  }

  .modal-card,
  .modal-card-lg {
    width: 100%;
    max-width: 100%;
    max-height: calc(100vh - 0.8rem);
    max-height: calc(100dvh - 0.8rem);
    border-radius: 12px;
  }

  .modal-header {
    align-items: flex-start;
    padding: 0.85rem;
  }

  .modal-header h2 {
    font-size: 1.1rem;
  }

  .modal-form,
  .profile-modal-content {
    padding: 0.85rem;
  }

  .form-row {
    grid-template-columns: minmax(0, 1fr);
    gap: 0.85rem;
  }

  .zone-card-footer {
    align-items: stretch;
  }

  .opt-inputs,
  .training-form-row {
    grid-template-columns: minmax(0, 1fr);
  }

  .modal-footer {
    justify-content: stretch;
  }

  .modal-footer .btn {
    flex: 1 1 100%;
    width: 100%;
  }

  .opt-result,
  .result-badge {
    min-width: 0;
    overflow-wrap: anywhere;
  }
}

@media (max-width: 360px) {
  .predictive-climate-container {
    padding: 0.6rem;
  }

  .zone-metrics-row {
    grid-template-columns: minmax(0, 1fr);
  }

  .zone-context-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .kpi-card {
    align-items: flex-start;
  }

  .footer-start-actions,
  .footer-end-actions {
    gap: 0.4rem;
  }
}

</style>
