<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  LayoutDashboard,
  Maximize2,
  Minimize2,
  Plus,
  Flame,
  Sun,
  Battery,
  Zap,
  Home,
  Clock,
  CloudSun,
  TrendingUp,
  Settings2,
  Sparkles
} from '@lucide/vue'
import { apiClient } from '../api/client'

interface Widget {
  id: string
  type: 'thermostat' | 'sparkline' | 'sankey' | 'entity_card'
  title: string
  entityKey?: string
  value?: any
  colSpan: number
  rowSpan: number
}

interface KioskConfig {
  enabled: boolean
  screensaverTimeoutSec: number
  wakeOnPresence: boolean
  showClock: boolean
  showWeather: boolean
}

interface DashboardProfile {
  id: string
  title: string
  icon: string
  is_default: boolean
  kiosk_config_json: string
  layout_json: string
}

const dashboards = ref<DashboardProfile[]>([])
const currentDashboardId = ref('')
const isKioskMode = ref(false)
const showScreensaver = ref(false)
const inactivityTimer = ref<any>(null)
const currentTime = ref(new Date().toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' }))
const currentDate = ref(new Date().toLocaleDateString('it-IT', { weekday: 'long', day: 'numeric', month: 'long' }))

// Circular Thermostat State
const currentTemp = ref(20.0)
const targetTemp = ref(20.0)
const isHeating = ref(false)

function adjustTargetTemp(delta: number) {
  targetTemp.value = Math.round((targetTemp.value + delta) * 10) / 10
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
  if (isKioskMode.value) {
    inactivityTimer.value = setTimeout(() => {
      showScreensaver.value = true
    }, 60000)
  }
}

function toggleFullscreen() {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen().catch(() => {})
    isKioskMode.value = true
    resetInactivity()
  } else {
    document.exitFullscreen().catch(() => {})
    isKioskMode.value = false
    showScreensaver.value = false
    clearTimeout(inactivityTimer.value)
  }
}

let clockInterval: any = null

onMounted(async () => {
  clockInterval = setInterval(() => {
    currentTime.value = new Date().toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' })
  }, 1000)

  window.addEventListener('mousemove', resetInactivity)
  window.addEventListener('touchstart', resetInactivity)
  window.addEventListener('keydown', resetInactivity)

  try {
    const list = await apiClient.get<DashboardProfile[]>('/api/v1/dashboards')
    if (list && list.length > 0) {
      dashboards.value = list
      currentDashboardId.value = list[0].id
    }
  } catch {}

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
      const climate = entities.find(e => e.id.startsWith('climate.') || e.id.includes('temp'))
      if (climate) {
        const tVal = parseFloat(climate.state)
        if (!isNaN(tVal)) {
          currentTemp.value = tVal
          targetTemp.value = climate.attributes?.temperature || tVal
          isHeating.value = climate.attributes?.hvac_action === 'heating'
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
          <h1 class="dash-title">Dashboard Personalizzata</h1>
          <p class="dash-subtitle">Widget interattivi a griglia ottimizzati per wall tablet e kiosk mode</p>
        </div>
      </div>

      <div class="dash-actions">
        <!-- Profile Selector -->
        <select v-model="currentDashboardId" class="profile-select">
          <option v-for="d in dashboards" :key="d.id" :value="d.id">
            {{ d.title }}
          </option>
        </select>

        <button class="kiosk-toggle-btn" @click="toggleFullscreen" title="Avvia Modalità Tablet a Parete">
          <Maximize2 :size="18" />
          <span>Kiosk Mode</span>
        </button>
      </div>
    </header>

    <!-- Floating Exit Kiosk Button (only visible in kiosk) -->
    <button v-if="isKioskMode && !showScreensaver" class="exit-kiosk-fab" @click="toggleFullscreen">
      <Minimize2 :size="20" />
    </button>

    <!-- Widget Grid -->
    <main class="widget-grid">
      <!-- 1. Circular Climate / Thermostat Widget (ColSpan 2) -->
      <section class="widget-card thermostat-card glass-panel">
        <div class="card-header">
          <div class="header-left">
            <Flame :size="20" class="text-warning animate-pulse" />
            <span class="card-title">Clima & Termostato Salotto</span>
          </div>
          <span class="badge heating-badge">Riscaldamento Attivo</span>
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
            <span class="dial-current-label">Attuale</span>
            <span class="dial-current-val">{{ currentTemp.toFixed(1) }}°C</span>
            <div class="dial-target-row">
              <span class="dial-target-label">Target:</span>
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
            <span class="card-title">Temperatura Salotto (Trend 6h)</span>
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
          <span>Min: 19.2°C</span>
          <span class="text-success">In aumento (+0.4°C/h)</span>
          <span>Max: 21.4°C</span>
        </div>
      </section>

      <!-- 3. Sankey Energy Overview Card -->
      <section class="widget-card sankey-mini-card glass-panel">
        <div class="card-header">
          <div class="header-left">
            <Zap :size="20" class="text-warning" />
            <span class="card-title">Flusso Energetico Istantaneo</span>
          </div>
          <span class="text-success badge">Autoconsumo 100%</span>
        </div>

        <div class="vectors-row">
          <div class="vector-node solar">
            <Sun :size="24" class="text-warning animate-spin-slow" />
            <span class="v-val">{{ solarWatts }} W</span>
            <span class="v-label">Solare</span>
          </div>

          <div class="vector-node battery">
            <Battery :size="24" class="text-info" />
            <span class="v-val">{{ Math.abs(batteryWatts) }} W</span>
            <span class="v-label">Batteria ({{ batterySoc }}%)</span>
          </div>

          <div class="vector-node grid">
            <Zap :size="24" class="text-primary" />
            <span class="v-val">{{ Math.abs(gridWatts) }} W</span>
            <span class="v-label">Immissione</span>
          </div>

          <div class="vector-node house">
            <Home :size="24" class="text-success" />
            <span class="v-val">{{ houseWatts }} W</span>
            <span class="v-label">Casa</span>
          </div>
        </div>
      </section>
    </main>

    <!-- 4. Screensaver Overlay for Wall Tablet Kiosk Mode -->
    <transition name="fade">
      <div v-if="showScreensaver" class="screensaver-overlay" @click="resetInactivity">
        <div class="screensaver-content">
          <div class="clock-display">{{ currentTime }}</div>
          <div class="date-display">{{ currentDate }}</div>

          <div class="screensaver-weather">
            <CloudSun :size="36" class="text-warning" />
            <div class="weather-meta">
              <span class="temp">22°C Soleggiato</span>
              <span class="desc">Qualità Aria: Ottima (12 µg/m³)</span>
            </div>
          </div>

          <div class="screensaver-hint">
            <Sparkles :size="16" />
            <span>Tocca lo schermo o avvicinati per attivare Domus</span>
          </div>
        </div>
      </div>
    </transition>
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
  background: rgba(17, 24, 39, 0.7);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.dash-title-group {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.dash-title {
  font-size: 1.5rem;
  font-weight: 700;
  color: #f9fafb;
  margin: 0;
}

.dash-subtitle {
  font-size: 0.875rem;
  color: #9ca3af;
  margin: 0.25rem 0 0;
}

.dash-actions {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.profile-select {
  background: rgba(31, 41, 55, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #f3f4f6;
  padding: 0.6rem 1rem;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  cursor: pointer;
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
  background: rgba(31, 41, 55, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: #9ca3af;
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
  color: white;
  background: rgba(55, 65, 81, 0.9);
}

/* Widget Grid */
.widget-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
  gap: 1.5rem;
}

.widget-card {
  border-radius: 1.25rem;
  background: rgba(17, 24, 39, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.08);
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
  color: #f3f4f6;
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
  color: #9ca3af;
  text-transform: uppercase;
}

.dial-current-val {
  font-size: 2.25rem;
  font-weight: 800;
  color: #ffffff;
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
  background: rgba(31, 41, 55, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: white;
  font-weight: 700;
  font-size: 1.1rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.temp-btn:hover {
  background: rgba(55, 65, 81, 0.9);
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
  background: rgba(31, 41, 55, 0.4);
  padding: 1rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.05);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
}

.v-val {
  font-size: 1.1rem;
  font-weight: 700;
  color: #ffffff;
}

.v-label {
  font-size: 0.75rem;
  color: #9ca3af;
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
</style>
