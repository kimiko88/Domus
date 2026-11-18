<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  LayoutDashboard,
  Maximize2,
  Minimize2,
  Flame,
  Sun,
  Battery,
  Zap,
  Home,
  CloudSun,
  TrendingUp,
  Settings2,
  Sparkles,
  Tablet
} from '@lucide/vue'
import { apiClient } from '../api/client'

const { t, locale } = useI18n()

interface KioskConfig {
  enabled: boolean
  screensaverTimeoutSec: number
  wakeOnPresence: boolean
  wakeOnCamera?: boolean
  showClock: boolean
  showWeather: boolean
  pinCode?: string
}

interface DashboardWidget {
  id: string
  type: 'thermostat' | 'temperature' | 'energy' | 'entity' | 'text'
  title: string
  entity_key?: string
  col_span: number
  row_span: number
  config?: Record<string, any>
}

interface DashboardProfile {
  id: string
  title: string
  icon: string
  is_default: boolean
  kiosk_config_json: string
  layout_json?: string
}

const defaultDashboards: DashboardProfile[] = [
  {
    id: 'tablet_living_room',
    title: t('dashboard.view.defaultProfileLivingRoom'),
    icon: 'layout',
    is_default: true,
    kiosk_config_json: '{}',
    layout_json: JSON.stringify({ target_temp: 20.0, room: t('dashboard.view.roomLivingRoom') })
  },
  {
    id: 'tablet_kitchen',
    title: t('dashboard.view.defaultProfileKitchen'),
    icon: 'utensils',
    is_default: false,
    kiosk_config_json: '{}',
    layout_json: JSON.stringify({ target_temp: 19.5, room: t('dashboard.view.roomKitchen') })
  },
  {
    id: 'tablet_night',
    title: t('dashboard.view.defaultProfileNight'),
    icon: 'moon',
    is_default: false,
    kiosk_config_json: '{}',
    layout_json: JSON.stringify({ target_temp: 18.0, room: t('dashboard.view.roomBedroom') })
  }
]

const dashboards = ref<DashboardProfile[]>([])
const currentDashboardId = ref('')
const isDashboardEditorOpen = ref(false)
const isCreatingDashboard = ref(false)
const dashboardError = ref('')
const dashboardSaving = ref(false)
const dashboardTitleDraft = ref('')
const widgetDrafts = ref<DashboardWidget[]>([])
const newWidgetType = ref<DashboardWidget['type']>('temperature')
const newWidgetTitle = ref('')
const newWidgetEntity = ref('')
const customWidgets = ref<DashboardWidget[]>([])
const availableEntities = ref<any[]>([])
function getWidgetEntityState(entityKey?: string) {
  if (!entityKey) return 'Seleziona un’entità'
  const entity = availableEntities.value.find(item => item.id === entityKey || item.entity_key === entityKey)
  return entity ? String(entity.state ?? '—') : 'Entità non disponibile'
}
const currentProfile = computed(() => dashboards.value.find(d => d.id === currentDashboardId.value))
const activeRoomName = ref(t('dashboard.view.roomLivingRoom'))
const isKioskMode = ref(false)
const showScreensaver = ref(false)
const inactivityTimer = ref<any>(null)
const currentTime = ref(new Date().toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' }))
const currentDate = ref(new Date().toLocaleDateString(locale.value, { weekday: 'long', day: 'numeric', month: 'long' }))

// Kiosk Configurator State
const isKioskConfigModalOpen = ref(false)
const isPinModalOpen = ref(false)
const enteredPin = ref('')
const pinError = ref(false)
const kioskConfig = ref<KioskConfig>({
  enabled: true,
  screensaverTimeoutSec: 60,
  wakeOnPresence: true,
  wakeOnCamera: false,
  showClock: true,
  showWeather: true,
  pinCode: ''
})

// Circular Thermostat State
const currentTemp = ref(20.0)
const targetTemp = ref(20.0)

function adjustTargetTemp(delta: number) {
  targetTemp.value = Math.round((targetTemp.value + delta) * 10) / 10
}

function readProfileWidgets(profile?: DashboardProfile): DashboardWidget[] {
  if (!profile?.layout_json) return []
  try {
    const parsed = JSON.parse(profile.layout_json)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((w: any) => w && typeof w.id === 'string' && typeof w.title === 'string').map((w: any) => ({
      id: w.id,
      type: ['thermostat', 'temperature', 'energy', 'entity', 'text'].includes(w.type) ? w.type : 'entity',
      title: w.title,
      entity_key: w.entity_key || '',
      col_span: Math.min(4, Math.max(1, Number(w.col_span) || 1)),
      row_span: Math.min(3, Math.max(1, Number(w.row_span) || 1)),
      config: w.config || {}
    }))
  } catch { return [] }
}

function openDashboardEditor(create = false) {
  const profile = currentProfile.value
  isCreatingDashboard.value = create
  dashboardError.value = ''
  dashboardTitleDraft.value = create ? '' : (profile?.title || '')
  widgetDrafts.value = create ? [] : readProfileWidgets(profile).map(w => ({ ...w }))
  newWidgetTitle.value = ''
  newWidgetEntity.value = ''
  isDashboardEditorOpen.value = true
}

function addDashboardWidget() {
  const labels: Record<string, string> = { thermostat: 'Termostato', temperature: 'Temperatura', energy: 'Energia', entity: 'Entità', text: 'Testo' }
  widgetDrafts.value.push({
    id: 'widget_' + Date.now().toString(36) + '_' + widgetDrafts.value.length,
    type: newWidgetType.value,
    title: newWidgetTitle.value.trim() || labels[newWidgetType.value],
    entity_key: newWidgetEntity.value.trim(),
    col_span: 1,
    row_span: 1
  })
  newWidgetTitle.value = ''
  newWidgetEntity.value = ''
}

function removeDashboardWidget(id: string) {
  widgetDrafts.value = widgetDrafts.value.filter(w => w.id !== id)
}

async function saveDashboardEditor() {
  const title = dashboardTitleDraft.value.trim()
  if (!title) { dashboardError.value = 'Inserisci il nome della dashboard.'; return }
  dashboardSaving.value = true
  dashboardError.value = ''
  try {
    const existing = currentProfile.value
    const profile: DashboardProfile = {
      id: isCreatingDashboard.value ? 'custom_' + Date.now().toString(36) : (existing?.id || ''),
      title,
      icon: existing?.icon || 'layout-dashboard',
      is_default: existing?.is_default || false,
      kiosk_config_json: existing?.kiosk_config_json || JSON.stringify(kioskConfig.value),
      layout_json: JSON.stringify(widgetDrafts.value)
    }
    if (isCreatingDashboard.value) {
      const created = await apiClient.post<DashboardProfile>('/api/v1/dashboards', profile)
      const saved = created || profile
      dashboards.value.push(saved)
      currentDashboardId.value = saved.id
      customWidgets.value = readProfileWidgets(saved)
    } else {
      await apiClient.put('/api/v1/dashboards/' + profile.id, profile)
      const index = dashboards.value.findIndex(d => d.id === profile.id)
      if (index >= 0) dashboards.value[index] = profile
      customWidgets.value = readProfileWidgets(profile)
    }
    activeRoomName.value = title
    localStorage.setItem('domus_active_tablet_dashboard', currentDashboardId.value)
    isDashboardEditorOpen.value = false
  } catch (err: any) {
    dashboardError.value = err?.message || 'Impossibile salvare la dashboard. Riprova.'
  } finally { dashboardSaving.value = false }
}

async function deleteCurrentDashboard() {
  const profile = currentProfile.value
  if (!profile || !profile.id.startsWith('custom_')) return
  if (!window.confirm('Eliminare la dashboard "' + profile.title + '"?')) return
  try {
    await apiClient.delete('/api/v1/dashboards/' + profile.id)
    dashboards.value = dashboards.value.filter(d => d.id !== profile.id)
    currentDashboardId.value = dashboards.value[0]?.id || ''
    switchDashboardProfile()
  } catch (err: any) {
    dashboardError.value = err?.message || 'Impossibile eliminare la dashboard.'
  }
}

function switchDashboardProfile() {
  const profile = dashboards.value.find(d => d.id === currentDashboardId.value)
  if (profile) {
    customWidgets.value = readProfileWidgets(profile)
    try {
      const cfg = JSON.parse(profile.layout_json || '{}')
      if (cfg.target_temp) {
        targetTemp.value = cfg.target_temp
      }
      activeRoomName.value = cfg.room || profile.title || t('dashboard.view.roomLivingRoom')
      customWidgets.value = readProfileWidgets(profile)
    } catch {}

    if (profile.kiosk_config_json) {
      try {
        const kCfg = JSON.parse(profile.kiosk_config_json)
        kioskConfig.value = { ...kioskConfig.value, ...kCfg }
      } catch {}
    }
    localStorage.setItem('domus_active_tablet_dashboard', profile.id)
  }
}

async function saveKioskConfig() {
  const profile = dashboards.value.find(d => d.id === currentDashboardId.value)
  if (profile) {
    profile.kiosk_config_json = JSON.stringify(kioskConfig.value)
    try {
      await apiClient.put(`/api/v1/dashboards/${profile.id}`, profile)
    } catch {}
  }
  localStorage.setItem(`kiosk_cfg_${currentDashboardId.value}`, JSON.stringify(kioskConfig.value))
  isKioskConfigModalOpen.value = false
  if (kioskConfig.value.wakeOnCamera) {
    startCameraMotion()
  } else {
    stopCameraMotion()
  }
  resetInactivity()
}

// Camera Motion Wake-up Detection for Wall Tablets
let cameraStream: MediaStream | null = null
let motionInterval: any = null
let lastFrameData: Uint8ClampedArray | null = null

async function startCameraMotion() {
  if (!kioskConfig.value.wakeOnCamera || typeof navigator === 'undefined' || !navigator.mediaDevices?.getUserMedia) return
  try {
    cameraStream = await navigator.mediaDevices.getUserMedia({ video: { width: 64, height: 64, facingMode: 'user' } })
    const video = document.createElement('video')
    video.srcObject = cameraStream
    video.autoplay = true
    await video.play().catch(() => {})

    const canvas = document.createElement('canvas')
    canvas.width = 32
    canvas.height = 32
    const ctx = canvas.getContext('2d')

    motionInterval = setInterval(() => {
      if (!ctx || !showScreensaver.value) return
      ctx.drawImage(video, 0, 0, 32, 32)
      const currentData = ctx.getImageData(0, 0, 32, 32).data
      if (lastFrameData) {
        let diff = 0
        for (let i = 0; i < currentData.length; i += 4) {
          diff += Math.abs(currentData[i] - lastFrameData[i])
        }
        if (diff > 8000) { // motion threshold
          resetInactivity()
        }
      }
      lastFrameData = new Uint8ClampedArray(currentData)
    }, 500)
  } catch {}
}

function stopCameraMotion() {
  if (cameraStream) {
    cameraStream.getTracks().forEach(t => t.stop())
    cameraStream = null
  }
  if (motionInterval) {
    clearInterval(motionInterval)
    motionInterval = null
  }
}

// Sparkline Telemetry History Data
const sparklineData = ref<number[]>([])
const sparklinePoints = computed(() => {
  const data = sparklineData.value
  if (!data || data.length < 2) return '0,20 140,20'
  const min = Math.min(...data)
  const max = Math.max(...data)
  const range = max - min || 1
  return data
    .map((val, idx) => {
      const x = (idx / (data.length - 1)) * 140
      const y = 40 - ((val - min) / range) * 35
      return `${x},${y}`
    })
    .join(' ')
})

// Sankey Real-Time Vectors
const solarWatts = ref(0)
const batteryWatts = ref(0)
const batterySoc = ref(0)
const gridWatts = ref(0)
const houseWatts = computed(() => Math.max(0, solarWatts.value + gridWatts.value + batteryWatts.value))

// Kiosk Activity & Wakeup
function resetInactivity() {
  if (showScreensaver.value) {
    showScreensaver.value = false
  }
  clearTimeout(inactivityTimer.value)
  if (isKioskMode.value && kioskConfig.value.screensaverTimeoutSec > 0) {
    inactivityTimer.value = setTimeout(() => {
      showScreensaver.value = true
    }, kioskConfig.value.screensaverTimeoutSec * 1000)
  }
}

function attemptExitKiosk() {
  if (kioskConfig.value.pinCode && kioskConfig.value.pinCode.trim().length > 0) {
    isPinModalOpen.value = true
    enteredPin.value = ''
    pinError.value = false
  } else {
    toggleFullscreen()
  }
}

function verifyPinAndExit() {
  if (enteredPin.value === kioskConfig.value.pinCode) {
    isPinModalOpen.value = false
    toggleFullscreen()
  } else {
    pinError.value = true
  }
}

function toggleFullscreen() {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen().catch(() => {})
    isKioskMode.value = true
    if (kioskConfig.value.wakeOnCamera) {
      startCameraMotion()
    }
    resetInactivity()
  } else {
    document.exitFullscreen().catch(() => {})
    isKioskMode.value = false
    showScreensaver.value = false
    clearTimeout(inactivityTimer.value)
    stopCameraMotion()
  }
}

let clockInterval: any = null

onMounted(async () => {
  clockInterval = setInterval(() => {
    currentTime.value = new Date().toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' })
    currentDate.value = new Date().toLocaleDateString(locale.value, { weekday: 'long', day: 'numeric', month: 'long' })
  }, 1000)

  window.addEventListener('mousemove', resetInactivity)
  window.addEventListener('touchstart', resetInactivity)
  window.addEventListener('keydown', resetInactivity)

  try {
    const list = await apiClient.get<DashboardProfile[]>('/api/v1/dashboards')
    if (list && list.length > 0) {
      dashboards.value = list
    } else {
      dashboards.value = defaultDashboards
    }
  } catch {
    dashboards.value = defaultDashboards
  }

  const saved = localStorage.getItem('domus_active_tablet_dashboard')
  if (saved && dashboards.value.some(d => d.id === saved)) {
    currentDashboardId.value = saved
  } else if (dashboards.value.length > 0) {
    currentDashboardId.value = dashboards.value[0].id
  }
  switchDashboardProfile()

  try {
    const f = await apiClient.get<any>('/api/v1/energy/flow')
    if (f) {
      solarWatts.value = f.solar_watts || 0
      batteryWatts.value = f.battery_watts || 0
      batterySoc.value = f.battery_soc || 0
      gridWatts.value = f.grid_watts || 0
    }
  } catch {}

  try {
    const entities = await apiClient.get<any[]>('/api/v1/entities')
    if (entities && Array.isArray(entities)) {
      availableEntities.value = entities
      const climate = entities.find(e => e.id.startsWith('climate.') || e.id.includes('temp'))
      if (climate) {
        const tVal = parseFloat(climate.state)
        if (!isNaN(tVal)) {
          currentTemp.value = tVal
          targetTemp.value = climate.attributes?.temperature || tVal
        }
      }
    }
  } catch {}
})

onUnmounted(() => {
  clearInterval(clockInterval)
  clearTimeout(inactivityTimer.value)
  window.removeEventListener('mousemove', resetInactivity)
  window.removeEventListener('touchstart', resetInactivity)
  window.removeEventListener('keydown', resetInactivity)
})
</script>

<template>
  <div :class="['custom-dash-container', { 'kiosk-active': isKioskMode }]">
    <!-- Header Controls -->
    <header v-if="!isKioskMode" class="dash-header glass-panel">
      <div class="dash-title-group">
        <LayoutDashboard :size="24" class="text-primary" />
        <div>
          <h1 class="dash-title">{{ t('dashboard.view.customTitle') }}</h1>
          <p class="dash-subtitle">{{ t('dashboard.view.customSubtitle') }}</p>
        </div>
      </div>

      <div class="dash-actions">
        <!-- Profile Selector -->
        <select
          id="profile-select"
          name="profile_select"
          v-model="currentDashboardId"
          class="profile-select"
          :aria-label="t('dashboard.view.selectTabletProfile')"
          @change="switchDashboardProfile"
        >
          <option v-for="d in dashboards" :key="d.id" :value="d.id">
            {{ d.title }}
          </option>
        </select>

        <button class="kiosk-config-btn" @click="openDashboardEditor(false)" title="Personalizza dashboard">
          <Settings2 :size="18" /><span>Personalizza</span>
        </button>
        <button class="kiosk-config-btn" @click="openDashboardEditor(true)" title="Crea una nuova dashboard">+ Nuova</button>
        <button v-if="currentProfile && currentProfile.id.startsWith('custom_')" class="kiosk-config-btn dashboard-delete-btn" @click="deleteCurrentDashboard">Elimina</button>

        <button class="kiosk-config-btn" @click="isKioskConfigModalOpen = true" :title="t('dashboard.view.configureKiosk')">
          <Settings2 :size="18" />
          <span>{{ t('dashboard.view.configureKioskButton') }}</span>
        </button>

        <button class="kiosk-toggle-btn" @click="toggleFullscreen" :title="t('dashboard.view.startWallTablet')">
          <Maximize2 :size="18" />
          <span>{{ t('dashboard.view.kioskMode') }}</span>
        </button>
      </div>
    </header>

    <!-- Floating Exit Kiosk Button (only visible in kiosk) -->
    <button v-if="isKioskMode && !showScreensaver" class="exit-kiosk-fab" @click="attemptExitKiosk" :title="t('dashboard.view.exitKiosk')">
      <Minimize2 :size="20" />
    </button>

    <!-- Widget Grid -->
    <main v-if="customWidgets.length === 0" class="widget-grid">
      <!-- 1. Circular Climate / Thermostat Widget (ColSpan 2) -->
      <section class="widget-card thermostat-card glass-panel">
        <div class="card-header">
          <div class="header-left">
            <Flame :size="20" class="text-warning animate-pulse" />
            <span class="card-title">{{ t('dashboard.view.climateThermostat') }} {{ activeRoomName }}</span>
          </div>
          <span class="badge heating-badge">{{ t('dashboard.view.heatingActive') }}</span>
        </div>

        <div class="dial-container">
          <!-- Circular Dial SVG -->
          <svg class="thermostat-dial" viewBox="0 0 200 200">
            <!-- Background Arc -->
            <circle
              cx="100"
              cy="100"
              r="75"
              fill="none"
              stroke="rgba(255, 255, 255, 0.08)"
              stroke-width="12"
              stroke-dasharray="353"
              stroke-dashoffset="118"
              stroke-linecap="round"
            />
            <!-- Active Heating Arc -->
            <circle
              cx="100"
              cy="100"
              r="75"
              fill="none"
              stroke="url(#heatGradient)"
              stroke-width="12"
              stroke-dasharray="353"
              :stroke-dashoffset="353 - (targetTemp - 15) * 18"
              stroke-linecap="round"
              class="dial-progress"
            />
            <defs>
              <linearGradient id="heatGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" stop-color="#f59e0b" />
                <stop offset="100%" stop-color="#ef4444" />
              </linearGradient>
            </defs>
          </svg>

          <!-- Inside Dial Values -->
          <div class="dial-content">
            <span class="dial-current-label">{{ t('dashboard.view.current') }}</span>
            <span class="dial-current-val">{{ currentTemp.toFixed(1) }}°C</span>
            <div class="dial-target-row">
              <span class="dial-target-label">{{ t('dashboard.view.target') }}:</span>
              <span class="dial-target-val">{{ targetTemp.toFixed(1) }}°C</span>
            </div>
          </div>
        </div>

        <!-- Controls -->
        <div class="temp-control-buttons">
          <button class="temp-btn" @click="adjustTargetTemp(-0.5)">- 0.5°</button>
          <button class="temp-btn primary" @click="adjustTargetTemp(0.5)">+ 0.5°</button>
        </div>
      </section>

      <!-- 2. Sparkline Telemetry History Card -->
      <section class="widget-card sparkline-card glass-panel">
        <div class="card-header">
          <div class="header-left">
            <TrendingUp :size="20" class="text-info" />
            <span class="card-title">{{ t('dashboard.view.livingRoomTemperature') }}</span>
          </div>
          <span class="spark-val">{{ currentTemp }}°C</span>
        </div>

        <div class="sparkline-wrapper">
          <svg viewBox="0 0 140 40" class="sparkline-svg">
            <defs>
              <linearGradient id="sparkGradient" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" stop-color="#06b6d4" stop-opacity="0.4" />
                <stop offset="100%" stop-color="#06b6d4" stop-opacity="0.0" />
              </linearGradient>
            </defs>
            <polyline
              fill="none"
              stroke="#06b6d4"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
              :points="sparklinePoints"
            />
          </svg>
        </div>

        <div class="spark-footer">
          <span>{{ t('dashboard.view.minimumTemperature', { value: '19.2°C' }) }}</span>
          <span class="text-success">{{ t('dashboard.view.risingTemperature', { value: '+0.4°C/h' }) }}</span>
          <span>{{ t('dashboard.view.maximumTemperature', { value: '21.4°C' }) }}</span>
        </div>
      </section>

      <!-- 3. Sankey Energy Overview Card -->
      <section class="widget-card sankey-mini-card glass-panel">
        <div class="card-header">
          <div class="header-left">
            <Zap :size="20" class="text-warning" />
            <span class="card-title">{{ t('dashboard.view.instantEnergyFlow') }}</span>
          </div>
          <span class="text-success badge">{{ t('dashboard.view.selfConsumption') }}</span>
        </div>

        <div class="vectors-row">
          <div class="vector-node solar">
            <Sun :size="24" class="text-warning animate-spin-slow" />
            <span class="v-val">{{ solarWatts }} W</span>
            <span class="v-label">{{ t('dashboard.view.solar') }}</span>
          </div>

          <div class="vector-node battery">
            <Battery :size="24" class="text-info" />
            <span class="v-val">{{ Math.abs(batteryWatts) }} W</span>
            <span class="v-label">{{ t('dashboard.view.batteryLabel', { value: batterySoc }) }}</span>
          </div>

          <div class="vector-node grid">
            <Zap :size="24" class="text-primary" />
            <span class="v-val">{{ Math.abs(gridWatts) }} W</span>
            <span class="v-label">{{ t('dashboard.view.feedIn') }}</span>
          </div>

          <div class="vector-node house">
            <Home :size="24" class="text-success" />
            <span class="v-val">{{ houseWatts }} W</span>
            <span class="v-label">{{ t('dashboard.view.home') }}</span>
          </div>
        </div>
      </section>
    </main>


    <section v-if="customWidgets.length" class="custom-widget-grid" aria-label="Widget personalizzati">
      <article v-for="widget in customWidgets" :key="widget.id" class="widget-card custom-widget-card glass-panel"
        :style="{ '--widget-col-span': widget.col_span, '--widget-row-span': widget.row_span }">
        <header class="card-header">
          <div class="header-left"><LayoutDashboard :size="20" class="text-primary" /><span class="card-title">{{ widget.title }}</span></div>
          <span class="badge">{{ widget.type }}</span>
        </header>
        <div class="custom-widget-content">
          <template v-if="widget.type === 'thermostat'"><strong class="custom-widget-value">{{ targetTemp.toFixed(1) }}°C</strong><span>Setpoint ambiente</span></template>
          <template v-else-if="widget.type === 'temperature'"><strong class="custom-widget-value">{{ currentTemp.toFixed(1) }}°C</strong><span>Temperatura rilevata</span></template>
          <template v-else-if="widget.type === 'energy'"><strong class="custom-widget-value">{{ houseWatts }} W</strong><span>Potenza istantanea casa</span></template>
          <template v-else-if="widget.type === 'entity'"><strong class="custom-widget-value">{{ getWidgetEntityState(widget.entity_key) }}</strong><span>Entità Domus configurata</span></template>
          <template v-else><p>{{ widget.config?.text || widget.entity_key || 'Widget di testo personalizzato' }}</p></template>
        </div>
      </article>
    </section>

    <div v-if="isDashboardEditorOpen" class="modal-backdrop dashboard-editor-backdrop" @click.self="isDashboardEditorOpen = false">
      <section class="dashboard-editor glass-panel" role="dialog" aria-modal="true" aria-labelledby="dashboard-editor-title">
        <header class="modal-header">
          <div><h3 id="dashboard-editor-title" class="modal-title">{{ isCreatingDashboard ? 'Crea dashboard personalizzata' : 'Personalizza dashboard' }}</h3><p class="field-hint">Scegli nome, widget e dimensioni della griglia.</p></div>
          <button class="btn-close-modal" @click="isDashboardEditorOpen = false" aria-label="Chiudi">✕</button>
        </header>
        <div class="modal-body dashboard-editor-body">
          <div class="config-field"><label for="dashboard-title">Nome dashboard</label><input id="dashboard-title" v-model="dashboardTitleDraft" class="form-input" maxlength="80" placeholder="Es. Dashboard piano terra" /></div>
          <div class="widget-builder">
            <h4>Aggiungi widget</h4>
            <div class="widget-builder-fields">
              <select v-model="newWidgetType" class="form-select" aria-label="Tipo widget">
                <option value="thermostat">Termostato</option><option value="temperature">Temperatura</option><option value="energy">Energia</option><option value="entity">Entità</option><option value="text">Testo / Nota</option>
              </select>
              <input v-model="newWidgetTitle" class="form-input" placeholder="Titolo widget (facoltativo)" aria-label="Titolo widget" />
              <input v-model="newWidgetEntity" class="form-input" placeholder="Entity ID (facoltativo)" aria-label="Entity ID" />
              <button class="btn-save" type="button" @click="addDashboardWidget">Aggiungi</button>
            </div>
          </div>
          <div class="widget-draft-list">
            <h4>Widget della dashboard ({{ widgetDrafts.length }})</h4>
            <p v-if="widgetDrafts.length === 0" class="field-hint">Nessun widget aggiunto. Puoi creare una dashboard vuota e configurarla in seguito.</p>
            <article v-for="widget in widgetDrafts" :key="widget.id" class="widget-draft-row">
              <div class="widget-draft-main"><strong>{{ widget.title }}</strong><small>{{ widget.type }} <span v-if="widget.entity_key">· {{ widget.entity_key }}</span></small></div>
              <label class="widget-span-control">Colonne<select v-model.number="widget.col_span" class="form-select"><option :value="1">1</option><option :value="2">2</option><option :value="3">3</option><option :value="4">4</option></select></label>
              <label class="widget-span-control">Righe<select v-model.number="widget.row_span" class="form-select"><option :value="1">1</option><option :value="2">2</option><option :value="3">3</option></select></label>
              <button type="button" class="btn-cancel" @click="removeDashboardWidget(widget.id)" :aria-label="'Rimuovi ' + widget.title">Rimuovi</button>
            </article>
          </div>
          <p v-if="dashboardError" class="dashboard-editor-error" role="alert">{{ dashboardError }}</p>
        </div>
        <footer class="modal-footer"><button class="btn-cancel" type="button" @click="isDashboardEditorOpen = false">Annulla</button><button class="btn-save" type="button" :disabled="dashboardSaving" @click="saveDashboardEditor">{{ dashboardSaving ? 'Salvataggio…' : 'Salva dashboard' }}</button></footer>
      </section>
    </div>

    <!-- 4. Screensaver Overlay for Wall Tablet Kiosk Mode -->
    <transition name="fade">
      <div v-if="showScreensaver" class="screensaver-overlay" @click="resetInactivity">
        <div class="screensaver-content">
          <div v-if="kioskConfig.showClock" class="clock-display">{{ currentTime }}</div>
          <div v-if="kioskConfig.showClock" class="date-display">{{ currentDate }}</div>

          <div v-if="kioskConfig.showWeather" class="screensaver-weather">
            <CloudSun :size="36" class="text-warning" />
            <div class="weather-meta">
              <span class="temp">{{ t('dashboard.view.sunnyWeather', { temperature: '22°C' }) }}</span>
              <span class="desc">{{ t('dashboard.view.airQuality', { quality: t('dashboard.view.excellent'), value: '12 µg/m³' }) }}</span>
            </div>
          </div>

          <div class="screensaver-hint">
            <Sparkles :size="16" />
            <span>{{ t('dashboard.view.activateDomusHint') }}</span>
          </div>
        </div>
      </div>
    </transition>

    <!-- Kiosk Configuration Modal Dialog -->
    <div v-if="isKioskConfigModalOpen" class="modal-backdrop" @click="isKioskConfigModalOpen = false">
      <div class="kiosk-modal glass-panel" @click.stop>
        <div class="modal-header">
          <div class="header-left">
            <Tablet :size="22" class="text-primary" />
            <h3 class="modal-title">{{ t('dashboard.view.kioskSettingsTitle') }}</h3>
          </div>
          <button class="btn-close-modal" @click="isKioskConfigModalOpen = false">✕</button>
        </div>

        <div class="modal-body">
          <div class="config-field">
            <label>{{ t('dashboard.view.screensaverTimeout') }}</label>
            <select v-model.number="kioskConfig.screensaverTimeoutSec" class="form-select">
              <option :value="15">{{ t('dashboard.view.secondsQuickTest') }}</option>
              <option :value="30">{{ t('dashboard.view.seconds', { value: 30 }) }}</option>
              <option :value="60">{{ t('dashboard.view.minuteRecommended') }}</option>
              <option :value="120">{{ t('dashboard.view.minutes', { value: 2 }) }}</option>
              <option :value="300">{{ t('dashboard.view.minutes', { value: 5 }) }}</option>
              <option :value="0">{{ t('dashboard.view.neverAlwaysOn') }}</option>
            </select>
            <p class="field-hint">{{ t('dashboard.view.screensaverTimeoutHint') }}</p>
          </div>

          <div class="config-toggle-row">
            <div>
              <span class="toggle-label">{{ t('dashboard.view.wakeOnPresence') }}</span>
              <p class="toggle-desc">{{ t('dashboard.view.wakeOnPresenceDesc') }}</p>
            </div>
            <input type="checkbox" v-model="kioskConfig.wakeOnCamera" class="toggle-checkbox" />
          </div>

          <div class="config-toggle-row">
            <div>
              <span class="toggle-label">{{ t('dashboard.view.showClockDate') }}</span>
              <p class="toggle-desc">{{ t('dashboard.view.showClockDateDesc') }}</p>
            </div>
            <input type="checkbox" v-model="kioskConfig.showClock" class="toggle-checkbox" />
          </div>

          <div class="config-toggle-row">
            <div>
              <span class="toggle-label">{{ t('dashboard.view.showWeatherAirQuality') }}</span>
              <p class="toggle-desc">{{ t('dashboard.view.showWeatherAirQualityDesc') }}</p>
            </div>
            <input type="checkbox" v-model="kioskConfig.showWeather" class="toggle-checkbox" />
          </div>

          <div class="config-field">
            <label>{{ t('dashboard.view.exitPin') }}</label>
            <input
              type="password"
              maxlength="6"
              v-model="kioskConfig.pinCode"
              :placeholder="t('dashboard.view.exitPinPlaceholder')"
              class="form-input"
            />
            <p class="field-hint">{{ t('dashboard.view.exitPinHint') }}</p>
          </div>
        </div>

        <div class="modal-footer">
          <button class="btn-cancel" @click="isKioskConfigModalOpen = false">{{ t('common.cancel') }}</button>
          <button class="btn-save" @click="saveKioskConfig">{{ t('dashboard.view.saveConfiguration') }}</button>
        </div>
      </div>
    </div>

    <!-- PIN Unlock Modal for Exiting Kiosk Mode -->
    <div v-if="isPinModalOpen" class="modal-backdrop" @click="isPinModalOpen = false">
      <div class="pin-modal glass-panel" @click.stop>
        <h3 class="pin-title">{{ t('dashboard.view.unlockKioskTitle') }}</h3>
        <p class="pin-desc">{{ t('dashboard.view.unlockKioskDesc') }}</p>
        <input
          type="password"
          maxlength="6"
          v-model="enteredPin"
          placeholder="••••"
          class="pin-input"
          @keyup.enter="verifyPinAndExit"
          autofocus
        />
        <p v-if="pinError" class="pin-error">{{ t('dashboard.view.invalidPin') }}</p>
        <div class="pin-actions">
          <button class="btn-cancel" @click="isPinModalOpen = false">{{ t('common.cancel') }}</button>
          <button class="btn-save" @click="verifyPinAndExit">{{ t('dashboard.view.unlock') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.custom-dash-container {
  padding: 1.5rem;
  max-width: 1400px;
  margin: 0 auto;
  position: relative;
  min-height: 85vh;
}

.custom-dash-container.kiosk-active {
  padding: 2rem;
  max-width: 100vw;
  min-height: 100vh;
  background: radial-gradient(circle at top right, #111827, #030712);
}

.dash-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.25rem 1.75rem;
  border-radius: 1rem;
  margin-bottom: 2rem;
  background: var(--surface-card, rgba(17, 24, 39, 0.7));
  backdrop-filter: blur(16px);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
}

.dash-title-group {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.dash-title {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.dash-subtitle {
  font-size: 0.875rem;
  color: var(--text-muted);
  margin: 0.25rem 0 0;
}

.dash-actions {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.profile-select {
  background: var(--bg-surface);
  border: 1px solid var(--border-color);
  color: var(--text-main);
  padding: 0.6rem 1rem;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s;
}

.profile-select:focus {
  border-color: var(--primary);
  outline: none;
}

.kiosk-toggle-btn {
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
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

.kiosk-toggle-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.35);
}

/* Exit Kiosk Floating Action Button */
.exit-kiosk-fab {
  position: fixed;
  top: 1.5rem;
  right: 1.5rem;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  color: var(--text-muted);
  border-radius: 50%;
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  z-index: 50;
  transition: opacity 0.2s, background 0.2s;
}

.exit-kiosk-fab:hover {
  color: var(--text-main);
  background: var(--bg-surface-hover);
}

/* Widget Grid */
.widget-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
  gap: 1.5rem;
}

.widget-card {
  border-radius: 1.25rem;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.card-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-main);
}

.badge {
  font-size: 0.75rem;
  padding: 0.25rem 0.6rem;
  border-radius: 9999px;
  font-weight: 600;
}

.heating-badge {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

/* Thermostat Dial */
.dial-container {
  position: relative;
  width: 200px;
  height: 200px;
  margin: 1rem auto;
}

.thermostat-dial {
  transform: rotate(-90deg);
  width: 100%;
  height: 100%;
}

.dial-progress {
  transition: stroke-dashoffset 0.3s ease;
}

.dial-content {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.dial-current-label {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-transform: uppercase;
}

.dial-current-val {
  font-size: 2.25rem;
  font-weight: 800;
  color: var(--text-main);
  letter-spacing: -0.05em;
}

.dial-target-row {
  display: flex;
  gap: 0.25rem;
  font-size: 0.875rem;
  color: #f59e0b;
  font-weight: 600;
}

.temp-control-buttons {
  display: flex;
  gap: 1rem;
  margin-top: 1rem;
}

.temp-btn {
  flex: 1;
  padding: 0.75rem;
  border-radius: 0.75rem;
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-color);
  color: var(--text-main);
  font-weight: 700;
  font-size: 1.1rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.temp-btn:hover {
  background: var(--bg-surface);
}

.temp-btn.primary {
  background: linear-gradient(135deg, #f59e0b, #ef4444);
}

/* Sparkline Card */
.spark-val {
  font-size: 1.25rem;
  font-weight: 700;
  color: #06b6d4;
}

.sparkline-wrapper {
  margin: 1.5rem 0;
  height: 60px;
}

.sparkline-svg {
  width: 100%;
  height: 100%;
}

.spark-footer {
  display: flex;
  justify-content: space-between;
  font-size: 0.8rem;
  color: #9ca3af;
}

/* Sankey Mini Card */
.vectors-row {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
  margin-top: 1rem;
}

.vector-node {
  background: var(--bg-surface-hover);
  padding: 1rem;
  border-radius: 0.75rem;
  border: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
}

.v-val {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--text-main);
}

.v-label {
  font-size: 0.75rem;
  color: var(--text-muted);
}

/* Screensaver Overlay */
.screensaver-overlay {
  position: fixed;
  inset: 0;
  background: #030712;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.screensaver-content {
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1.5rem;
}

.clock-display {
  font-size: 6.5rem;
  font-weight: 800;
  letter-spacing: -0.05em;
  color: #f9fafb;
  text-shadow: 0 0 30px rgba(255, 255, 255, 0.2);
}

.date-display {
  font-size: 1.75rem;
  color: #9ca3af;
  text-transform: capitalize;
}

.screensaver-weather {
  display: flex;
  align-items: center;
  gap: 1rem;
  background: rgba(31, 41, 55, 0.6);
  padding: 1rem 2rem;
  border-radius: 1.25rem;
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.weather-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.weather-meta .temp {
  font-size: 1.1rem;
  font-weight: 600;
  color: #ffffff;
}

.weather-meta .desc {
  font-size: 0.85rem;
  color: #9ca3af;
}

.screensaver-hint {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #6b7280;
  font-size: 0.875rem;
  margin-top: 2rem;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.5s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.kiosk-config-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.6rem 1rem;
  border-radius: 0.75rem;
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-color);
  color: var(--text-main);
  font-weight: 600;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.kiosk-config-btn:hover {
  background: var(--primary-light, rgba(14, 165, 233, 0.15));
  border-color: var(--primary);
  color: var(--primary);
}

.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  backdrop-filter: blur(8px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}

.kiosk-modal {
  width: 100%;
  max-width: 500px;
  background: var(--bg-surface);
  border: 1.5px solid var(--border-color);
  border-radius: 1.25rem;
  overflow: hidden;
  box-shadow: 0 20px 45px rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border-color);
}

.modal-header .header-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.modal-title {
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.btn-close-modal {
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: 1.2rem;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
}

.modal-body {
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  max-height: 70vh;
  overflow-y: auto;
}

.config-field {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.config-field label {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-main);
}

.field-hint {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin: 0.2rem 0 0;
}

.form-select,
.form-input {
  width: 100%;
  padding: 0.65rem 0.85rem;
  border-radius: 0.6rem;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  color: var(--text-main);
  font-size: 0.9rem;
}

.config-toggle-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--border-color);
}

.toggle-label {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-main);
}

.toggle-desc {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin: 0.2rem 0 0;
}

.toggle-checkbox {
  width: 20px;
  height: 20px;
  cursor: pointer;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--border-color);
  background: var(--bg-surface-hover);
}

.btn-cancel {
  padding: 0.55rem 1.15rem;
  border-radius: 0.6rem;
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-muted);
  font-weight: 600;
  cursor: pointer;
}

.btn-save {
  padding: 0.55rem 1.25rem;
  border-radius: 0.6rem;
  background: var(--primary);
  border: none;
  color: #ffffff;
  font-weight: 700;
  cursor: pointer;
}

/* PIN Modal */
.pin-modal {
  width: 100%;
  max-width: 360px;
  background: var(--bg-surface);
  border: 1.5px solid var(--border-color);
  border-radius: 1.25rem;
  padding: 1.75rem;
  text-align: center;
  box-shadow: 0 20px 45px rgba(0, 0, 0, 0.5);
}

.pin-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0 0 0.5rem;
}

.pin-desc {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin: 0 0 1.25rem;
}

.pin-input {
  width: 160px;
  font-size: 1.5rem;
  letter-spacing: 0.5rem;
  text-align: center;
  padding: 0.5rem;
  border-radius: 0.6rem;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  color: var(--text-main);
  margin-bottom: 0.75rem;
}

.pin-error {
  font-size: 0.8rem;
  color: var(--color-danger, #f43f5e);
  margin: 0 0 0.75rem;
}

.pin-actions {
  display: flex;
  justify-content: center;
  gap: 0.75rem;
  margin-top: 1rem;
}

/* Light Mode Overrides */
:root[data-theme="light"] .kiosk-modal,
:root[data-theme="light"] .pin-modal {
  background: #ffffff;
  border-color: #e2e8f0;
  box-shadow: 0 20px 45px rgba(15, 23, 42, 0.15);
}

:root[data-theme="light"] .modal-title,
:root[data-theme="light"] .pin-title,
:root[data-theme="light"] .config-field label,
:root[data-theme="light"] .toggle-label {
  color: #0f172a;
}

:root[data-theme="light"] .field-hint,
:root[data-theme="light"] .toggle-desc,
:root[data-theme="light"] .pin-desc {
  color: #475569;
}

:root[data-theme="light"] .form-select,
:root[data-theme="light"] .form-input,
:root[data-theme="light"] .pin-input {
  background: #f8fafc;
  border-color: #cbd5e1;
  color: #0f172a;
}

/* Responsive dashboard composition and editor */
.custom-dash-container, .custom-dash-container * { box-sizing: border-box; }
.custom-dash-container { width: 100%; min-width: 0; overflow-x: clip; padding-inline: clamp(.75rem, 2.5vw, 1.5rem); }
.dash-header, .dash-title-group, .dash-actions, .card-header, .header-left { min-width: 0; }
.dash-header { flex-wrap: wrap; gap: 1rem; }
.dash-actions { flex-wrap: wrap; justify-content: flex-end; }
.profile-select { min-width: 0; max-width: 100%; }
.widget-grid { grid-template-columns: repeat(auto-fit, minmax(min(100%, 300px), 1fr)); min-width: 0; }
.widget-card { min-width: 0; overflow-wrap: anywhere; }
.custom-widget-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 1.5rem; margin-top: 1.5rem; }
.custom-widget-card { min-width: 0;  }
.custom-widget-content { min-width: 0; display: flex; flex-direction: column; gap: .5rem; align-items: flex-start; color: var(--text-muted); }
.custom-widget-value { font-size: clamp(1.35rem, 4vw, 2rem); color: var(--text-main); overflow-wrap: anywhere; }
.dashboard-editor-backdrop { overflow-y: auto; padding: max(.5rem, env(safe-area-inset-top)) max(.5rem, env(safe-area-inset-right)) max(.5rem, env(safe-area-inset-bottom)) max(.5rem, env(safe-area-inset-left)); }
.dashboard-editor { width: min(100%, 760px); max-width: 100%; max-height: calc(100vh - 1rem); max-height: calc(100dvh - 1rem); min-width: 0; margin: auto; display: flex; flex-direction: column; overflow: hidden; border-radius: 1rem; }
.dashboard-editor-body { min-height: 0; max-height: none; overflow-y: auto; }
.widget-builder, .widget-draft-list { display: flex; flex-direction: column; gap: .75rem; min-width: 0; }
.widget-builder h4, .widget-draft-list h4 { margin: 0; color: var(--text-main); }
.widget-builder-fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: .65rem; }
.widget-builder-fields > * { min-width: 0; }
.widget-draft-row { display: grid; grid-template-columns: minmax(0, 1fr) auto auto auto; gap: .65rem; align-items: center; padding: .75rem; border: 1px solid var(--border-color); border-radius: .75rem; }
.widget-draft-main { display: flex; flex-direction: column; min-width: 0; overflow-wrap: anywhere; color: var(--text-main); }
.widget-draft-main small { color: var(--text-muted); }
.widget-span-control { display: flex; flex-direction: column; gap: .25rem; font-size: .75rem; color: var(--text-muted); }
.widget-span-control .form-select { min-width: 4rem; }
.dashboard-editor-error { color: var(--color-danger, #ef4444); margin: 0; }
.dashboard-delete-btn { color: var(--color-danger, #ef4444); }
@media (max-width: 900px) { .custom-widget-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } .custom-widget-card { grid-column: span min(var(--widget-col-span, 1), 2) !important; } }
@media (max-width: 640px) {
  .custom-dash-container { padding: .75rem; } .custom-dash-container.kiosk-active { padding: .75rem; }
  .dash-header { align-items: stretch; padding: 1rem; margin-bottom: 1rem; }
  .dash-title-group { align-items: flex-start; } .dash-title { font-size: 1.2rem; overflow-wrap: anywhere; }
  .dash-actions { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); width: 100%; gap: .5rem; }
  .dash-actions > * { width: 100%; min-width: 0; justify-content: center; } .profile-select { grid-column: 1 / -1; }
  .widget-grid, .custom-widget-grid { grid-template-columns: minmax(0, 1fr); gap: .85rem; }
  .custom-widget-card { grid-column: 1 !important; grid-row: auto !important; } .widget-card { padding: 1rem; }
  .card-header { align-items: flex-start; flex-wrap: wrap; gap: .5rem; } .header-left { align-items: flex-start; }
  .card-title { overflow-wrap: anywhere; } .spark-footer { flex-wrap: wrap; gap: .35rem .75rem; }
  .vectors-row { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .modal-backdrop { align-items: flex-start; }
  .kiosk-modal, .pin-modal, .dashboard-editor { width: 100%; max-width: 100%; max-height: calc(100dvh - 1rem); }
  .modal-body { max-height: none; min-height: 0; } .widget-builder-fields { grid-template-columns: minmax(0, 1fr); }
  .widget-draft-row { grid-template-columns: minmax(0, 1fr) minmax(4.5rem, auto) minmax(4.5rem, auto); }
  .widget-draft-row .btn-cancel { grid-column: 1 / -1; }
  .modal-footer, .pin-actions { flex-wrap: wrap; } .modal-footer > *, .pin-actions > * { flex: 1 1 100%; }
  .clock-display { font-size: clamp(3rem, 18vw, 6.5rem); } .date-display { font-size: clamp(1rem, 5vw, 1.75rem); }
}
@media (max-width: 360px) {
  .dash-actions { grid-template-columns: minmax(0, 1fr); }
  .widget-draft-row { grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); }
  .widget-draft-main, .widget-draft-row .btn-cancel { grid-column: 1 / -1; }
  .temp-control-buttons { gap: .5rem; } .temp-btn { font-size: .95rem; }
}

</style>
