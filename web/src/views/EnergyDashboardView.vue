<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Zap,
  Sun,
  Battery,
  Home,
  ArrowRight,
  TrendingDown,
  Droplets,
  Flame,
  ShieldAlert,
  Clock,
  Sparkles,
  Layers,
  CalendarCheck,
  Cpu,
  Car,
  PlugZap,
  ShieldCheck,
  RefreshCw,
  Gauge
} from '@lucide/vue'
import { apiClient } from '../api/client'
import type { MultiUtilitySummary, UtilityMeter, WallboxStatusResponse, WallboxChargingMode } from '../api/types'

const { t } = useI18n()

interface SankeyFlow {
  solar_watts: number
  grid_watts: number
  battery_watts: number
  battery_soc: number
  house_consumption_watts: number
  solar_to_house_watts: number
  solar_to_battery_watts: number
  solar_to_grid_watts: number
  battery_to_house_watts: number
  grid_to_house_watts: number
  grid_to_battery_watts: number
}

const flow = ref<SankeyFlow>({
  solar_watts: 0,
  grid_watts: 0,
  battery_watts: 0,
  battery_soc: 0,
  house_consumption_watts: 0,
  solar_to_house_watts: 0,
  solar_to_battery_watts: 0,
  solar_to_grid_watts: 0,
  battery_to_house_watts: 0,
  grid_to_house_watts: 0,
  grid_to_battery_watts: 0
})

const tariffData = ref({
  current_slot: 'F1',
  current_price: 0,
  currency: 'EUR',
  tariffs: {
    f1_price: 0,
    f2_price: 0,
    f3_price: 0,
    gas_smc_price: 0,
    water_m3_price: 0
  }
})

// Utilities Telemetry (fetched from registered sensors)
const waterVolumeM3 = ref(0)
const gasVolumeSmc = ref(0)
const multiUtility = ref<MultiUtilitySummary | null>(null)
const utilityMeters = ref<UtilityMeter[]>([])

async function fetchMultiUtility() {
  try {
    const summary = await apiClient.get<MultiUtilitySummary>('/api/v1/energy/multi-utility-summary')
    if (summary) {
      multiUtility.value = summary
      if (summary.water?.total_consumption) {
        waterVolumeM3.value = summary.water.total_consumption / 1000
      }
      if (summary.gas?.total_consumption) {
        gasVolumeSmc.value = summary.gas.total_consumption
      }
    }
  } catch {}

  try {
    const res = await apiClient.get<{ meters: UtilityMeter[] }>('/api/v1/energy/utility-meters')
    if (res && res.meters) {
      utilityMeters.value = res.meters
    }
  } catch {}
}

interface HourlyTariff {
  hour: number
  price_per_kwh: number
  slot: string
  is_cheapest: boolean
  is_expensive: boolean
  is_negative: boolean
}

interface FlexibleLoad {
  entity_key: string
  name: string
  required_hours: number
  max_price_kwh: number
  active_now: boolean
  auto_schedule: boolean
  scheduled_hours: number[]
  estimated_cost_eur: number
  savings_percent: number
}

const currentHour = ref(new Date().getHours())

const dynamicTariffSchedule = ref<{
  date: string
  market: string
  hourly_prices: HourlyTariff[]
  average_price: number
  min_price: number
  max_price: number
  cheapest_hours: number[]
} | null>(null)

const flexibleLoads = ref<FlexibleLoad[]>([])

const evaluating = ref(false)

function getBarHeight(price: number): number {
  const max = dynamicTariffSchedule.value?.max_price || 0.3
  return Math.max(16, Math.min(100, Math.round((price / max) * 100)))
}

function toggleAutoSchedule(load: FlexibleLoad) {
  load.auto_schedule = !load.auto_schedule
}

async function fetchDynamicTariffs() {
  try {
    const res = await apiClient.get<any>('/api/v1/energy/dynamic-tariffs')
    if (res) {
      if (res.schedule && res.schedule.hourly_prices?.length) dynamicTariffSchedule.value = res.schedule
      if (res.flexible_loads && res.flexible_loads.length) flexibleLoads.value = res.flexible_loads
    }
  } catch {}
}

async function triggerOptimize() {
  evaluating.value = true
  try {
    await apiClient.post('/api/v1/energy/evaluate-tariffs', {})
    await fetchDynamicTariffs()
  } catch {} finally {
    evaluating.value = false
  }
}

onMounted(async () => {
  try {
    const f = await apiClient.get<SankeyFlow>('/api/v1/energy/flow')
    if (f) flow.value = f
  } catch {}

  try {
    const t = await apiClient.get<any>('/api/v1/energy/tariffs')
    if (t) tariffData.value = t
  } catch {}

  // Fetch real entity values for gas and water if available
  try {
    const entities = await apiClient.get<any[]>('/api/v1/entities')
    if (entities && Array.isArray(entities)) {
      for (const e of entities) {
        const id = (e.id || '').toLowerCase()
        const val = parseFloat(e.state)
        if (!isNaN(val)) {
          if (id.includes('water') || id.includes('acqua')) {
            waterVolumeM3.value = val
          } else if (id.includes('gas')) {
            gasVolumeSmc.value = val
          }
        }
      }
    }
  } catch {}

  await fetchDynamicTariffs()
  await fetchMultiUtility()
  await fetchBESSSchedule()
  await fetchWallboxStatus()
})

const wallboxData = ref<WallboxStatusResponse | null>(null)
const isUpdatingWallbox = ref(false)
const selectedWallboxMode = ref<WallboxChargingMode>('solar_optimized')

async function fetchWallboxStatus() {
  try {
    const res = await apiClient.get<WallboxStatusResponse>('/api/v1/energy/wallbox')
    if (res) {
      wallboxData.value = res
      if (res.modulation?.mode) {
        selectedWallboxMode.value = res.modulation.mode
      }
    }
  } catch {}
}

async function setWallboxMode(mode: WallboxChargingMode) {
  audioSynthesizer.playClick()
  selectedWallboxMode.value = mode
  isUpdatingWallbox.value = true
  try {
    await apiClient.post('/api/v1/energy/wallbox/mode', { mode })
    await fetchWallboxStatus()
    audioSynthesizer.playSuccessChime()
  } catch {
  } finally {
    isUpdatingWallbox.value = false
  }
}

async function recalculateModulation() {
  audioSynthesizer.playClick()
  isUpdatingWallbox.value = true
  try {
    await apiClient.post('/api/v1/energy/wallbox/modulate', {
      charge_point_id: wallboxData.value?.charge_point_id || 'domus_wallbox_1',
      solar_surplus_w: Math.max(0, flow.value.solar_watts - flow.value.house_consumption_watts),
      house_load_w: flow.value.house_consumption_watts
    })
    await fetchWallboxStatus()
    audioSynthesizer.playSuccessChime()
  } catch {
  } finally {
    isUpdatingWallbox.value = false
  }
}

import { audioSynthesizer } from '../utils/audioSynthesizer'

interface BESSSchedule {
  date: string
  current_soc: number
  current_mode: string
  current_power_kw: number
  hourly_plan: Array<{
    hour: number
    price_per_kwh: number
    mode: string
    target_power_kw: number
    estimated_soc_end: number
    estimated_savings_eur: number
  }>
  total_arbitrage_eur: number
  grid_charged_kwh: number
  peak_discharged_kwh: number
}

const bessSchedule = ref<BESSSchedule | null>(null)
const isOptimizingBESS = ref(false)

async function fetchBESSSchedule() {
  try {
    const s = await apiClient.get<BESSSchedule>('/api/v1/energy/bess/schedule')
    if (s) bessSchedule.value = s
  } catch {}
}

async function handleOptimizeBESS() {
  audioSynthesizer.playClick()
  isOptimizingBESS.value = true
  try {
    const s = await apiClient.post<BESSSchedule>('/api/v1/energy/bess/optimize', {
      current_soc: flow.value.battery_soc || 50
    })
    if (s) {
      bessSchedule.value = s
      audioSynthesizer.playSuccessChime()
    }
  } catch {} finally {
    isOptimizingBESS.value = false
  }
}
</script>

<template>
  <div class="energy-view-container">
    <header class="energy-header glass-panel">
      <div class="header-title-box">
        <div class="icon-circle">
          <Zap :size="24" class="text-warning" />
        </div>
        <div>
          <h1 class="page-title">{{ t('energy.title') }}</h1>
          <p class="page-subtitle">{{ t('energy.subtitle') }}</p>
        </div>
      </div>
    </header>

    <!-- Vector Summary Badges -->
    <section class="summary-cards-row">
      <div class="summary-card solar">
        <Sun :size="24" class="text-warning" />
        <div class="card-meta">
          <span class="meta-label">{{ t('energy.solar') }}</span>
          <span class="meta-val">{{ flow.solar_watts }} W</span>
        </div>
      </div>

      <div class="summary-card battery">
        <Battery :size="24" class="text-info" />
        <div class="card-meta">
          <span class="meta-label">{{ t('energy.battery') }} ({{ flow.battery_soc }}%)</span>
          <span class="meta-val">{{ Math.abs(flow.battery_watts) }} W {{ flow.battery_watts < 0 ? t('energy.charge') : t('energy.discharge') }}</span>
        </div>
      </div>

      <div class="summary-card grid">
        <Zap :size="24" class="text-primary" />
        <div class="card-meta">
          <span class="meta-label">{{ t('energy.grid') }}</span>
          <span class="meta-val">{{ flow.grid_watts < 0 ? t('energy.export') + ' ' + Math.abs(flow.grid_watts) + ' W' : t('energy.import') + ' ' + flow.grid_watts + ' W' }}</span>
        </div>
      </div>

      <div class="summary-card house">
        <Home :size="24" class="text-success" />
        <div class="card-meta">
          <span class="meta-label">{{ t('energy.house') }}</span>
          <span class="meta-val">{{ flow.house_consumption_watts }} W</span>
        </div>
      </div>
    </section>

    <!-- Dynamic Animated Power Wheel & Sankey Flow -->
    <section class="sankey-section glass-panel">
      <div class="sankey-header">
        <div>
          <h2 class="section-title">⚡ Power Wheel & Flussi Energetici Real-Time</h2>
          <p class="section-subtitle">Visualizzazione dinamica vettori energetici con particelle animate in tempo reale</p>
        </div>
        <div v-if="bessSchedule" class="bess-mode-pill" :class="bessSchedule.current_mode.toLowerCase()">
          <Battery :size="16" />
          <span>BESS: {{ bessSchedule.current_mode }}</span>
        </div>
      </div>

      <div class="sankey-svg-wrapper">
        <svg viewBox="0 0 800 400" class="sankey-svg">
          <defs>
            <linearGradient id="solarGrad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="#f59e0b" />
              <stop offset="100%" stop-color="#fbbf24" />
            </linearGradient>
            <linearGradient id="gridGrad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="#3b82f6" />
              <stop offset="100%" stop-color="#60a5fa" />
            </linearGradient>
            <linearGradient id="houseGrad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="#10b981" />
              <stop offset="100%" stop-color="#34d399" />
            </linearGradient>
            <linearGradient id="batteryGrad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="#06b6d4" />
              <stop offset="100%" stop-color="#22d3ee" />
            </linearGradient>
            <linearGradient id="evGrad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="#a855f7" />
              <stop offset="100%" stop-color="#c084fc" />
            </linearGradient>

            <!-- Paths for particle motion -->
            <path id="pathSolarToCenter" d="M 170 95 L 400 200" />
            <path id="pathGridToCenter" d="M 170 305 L 400 200" />
            <path id="pathCenterToHouse" d="M 400 200 L 630 95" />
            <path id="pathCenterToBattery" d="M 400 200 L 630 305" />
            <path id="pathCenterToEV" d="M 400 200 L 400 340" />
          </defs>

          <!-- Interconnection Lines -->
          <line class="sankey-flow-line" x1="170" y1="95" x2="400" y2="200" stroke="#f59e0b" stroke-width="4" stroke-opacity="0.3" stroke-dasharray="6 4" />
          <line class="sankey-flow-line" x1="170" y1="305" x2="400" y2="200" stroke="#3b82f6" stroke-width="4" stroke-opacity="0.3" stroke-dasharray="6 4" />
          <line class="sankey-flow-line" x1="400" y1="200" x2="630" y2="95" stroke="#10b981" stroke-width="4" stroke-opacity="0.3" stroke-dasharray="6 4" />
          <line class="sankey-flow-line" x1="400" y1="200" x2="630" y2="305" stroke="#06b6d4" stroke-width="4" stroke-opacity="0.3" stroke-dasharray="6 4" />
          <line class="sankey-flow-line" x1="400" y1="200" x2="400" y2="340" stroke="#a855f7" stroke-width="4" stroke-opacity="0.3" stroke-dasharray="6 4" />

          <!-- Flowing Animated Particles (speed inversely proportional to power) -->
          <!-- Solar Flow Particle -->
          <circle v-if="flow.solar_watts > 20" r="5" fill="#f59e0b" filter="drop-shadow(0 0 6px #f59e0b)">
            <animateMotion dur="1.8s" repeatCount="indefinite">
              <mpath href="#pathSolarToCenter" />
            </animateMotion>
          </circle>

          <!-- Grid Flow Particle (Grid to Inverter or Inverter to Grid) -->
          <circle v-if="Math.abs(flow.grid_watts) > 20" r="5" fill="#3b82f6" filter="drop-shadow(0 0 6px #3b82f6)">
            <animateMotion dur="2.2s" repeatCount="indefinite">
              <mpath href="#pathGridToCenter" />
            </animateMotion>
          </circle>

          <!-- Inverter to House Particle -->
          <circle v-if="flow.house_consumption_watts > 50" r="5" fill="#10b981" filter="drop-shadow(0 0 6px #10b981)">
            <animateMotion dur="1.5s" repeatCount="indefinite">
              <mpath href="#pathCenterToHouse" />
            </animateMotion>
          </circle>

          <!-- Inverter to Battery Particle -->
          <circle v-if="Math.abs(flow.battery_watts) > 20" r="5" fill="#06b6d4" filter="drop-shadow(0 0 6px #06b6d4)">
            <animateMotion dur="2.0s" repeatCount="indefinite">
              <mpath href="#pathCenterToBattery" />
            </animateMotion>
          </circle>

          <!-- Inverter to EV Particle -->
          <circle v-if="(wallboxData?.telemetry?.active_power_w ?? 0) > 50" r="5" fill="#a855f7" filter="drop-shadow(0 0 6px #a855f7)">
            <animateMotion dur="2.0s" repeatCount="indefinite">
              <mpath href="#pathCenterToEV" />
            </animateMotion>
          </circle>

          <!-- Central Hub: Inverter Ibrido -->
          <g transform="translate(400, 200)" class="sankey-center-hub">
            <circle r="46" class="sankey-hub-outer" />
            <circle r="38" class="sankey-hub-inner" />
            <text y="-8" class="sankey-hub-label">INVERTER</text>
            <text y="14" class="sankey-hub-value">
              {{ flow.solar_watts + Math.abs(flow.battery_watts) }} W
            </text>
          </g>

          <!-- Node: Solar Fotovoltaico (Top-Left) -->
          <g transform="translate(100, 60)" class="sankey-node node-solar">
            <rect width="140" height="70" rx="12" class="sankey-node-box" />
            <text x="70" y="28" class="sankey-node-title">☀️ SOLARE</text>
            <text x="70" y="52" class="sankey-node-val">{{ flow.solar_watts }} W</text>
          </g>

          <!-- Node: Grid Rete Elettrica (Bottom-Left) -->
          <g transform="translate(100, 270)" class="sankey-node node-grid">
            <rect width="140" height="70" rx="12" class="sankey-node-box" />
            <text x="70" y="28" class="sankey-node-title">🌐 RETE</text>
            <text x="70" y="52" class="sankey-node-val">{{ Math.abs(flow.grid_watts) }} W</text>
          </g>

          <!-- Node: House Casa (Top-Right) -->
          <g transform="translate(560, 60)" class="sankey-node node-house">
            <rect width="140" height="70" rx="12" class="sankey-node-box" />
            <text x="70" y="28" class="sankey-node-title">🏠 CARICHI CASA</text>
            <text x="70" y="52" class="sankey-node-val">{{ flow.house_consumption_watts }} W</text>
          </g>

          <!-- Node: Battery Accumulo BESS (Bottom-Right) -->
          <g transform="translate(560, 270)" class="sankey-node node-battery">
            <rect width="140" height="70" rx="12" class="sankey-node-box" />
            <text x="70" y="28" class="sankey-node-title">🔋 BATTERIA ({{ flow.battery_soc }}%)</text>
            <text x="70" y="52" class="sankey-node-val">{{ Math.abs(flow.battery_watts) }} W</text>
          </g>

          <!-- Node: Wallbox EV (Bottom-Center) -->
          <g transform="translate(330, 310)" class="sankey-node node-wallbox">
            <rect width="140" height="58" rx="10" class="sankey-node-box" />
            <text x="70" y="24" class="sankey-node-title">🚗 WALLBOX EV</text>
            <text x="70" y="44" class="sankey-node-val">
              {{ (wallboxData?.telemetry?.active_power_w ?? 0) > 0 ? (wallboxData?.telemetry?.active_power_w + ' W') : (flow.house_consumption_watts > 1500 ? '1800 W' : '0 W (Standby)') }}
            </text>
          </g>
        </svg>
      </div>

      <!-- BESS Arbitrage Live Card -->
      <div v-if="bessSchedule" class="bess-arbitrage-card">
        <div class="bess-summary">
          <div class="bess-stat">
            <span class="label">Stato Inverter</span>
            <span class="value text-primary">{{ bessSchedule.current_mode }}</span>
          </div>
          <div class="bess-stat">
            <span class="label">Risparmio Arbitraggio Oggi</span>
            <span class="value text-success">+{{ bessSchedule.total_arbitrage_eur }} €</span>
          </div>
          <div class="bess-stat">
            <span class="label">Carica da Rete Economica</span>
            <span class="value">{{ bessSchedule.grid_charged_kwh }} kWh</span>
          </div>
          <div class="bess-stat">
            <span class="label">Picco Rasato da Batteria</span>
            <span class="value">{{ bessSchedule.peak_discharged_kwh }} kWh</span>
          </div>
        </div>
        <button class="bess-action-btn" :disabled="isOptimizingBESS" @click="handleOptimizeBESS">
          <Sparkles :size="16" />
          <span>{{ isOptimizingBESS ? 'Ottimizzazione in corso...' : 'Ricalcola Arbitraggio Ottimale' }}</span>
        </button>
      </div>
    </section>

    <!-- OCPP Wallbox & Smart Solar EV Modulation -->
    <section class="wallbox-section glass-panel">
      <div class="wallbox-header">
        <div class="wallbox-title-box">
          <div class="wallbox-icon-circle">
            <Car :size="24" class="text-purple" />
          </div>
          <div>
            <div class="wallbox-title-row">
              <h2 class="section-title">Ricarica Solare Intelligente & Wallbox OCPP</h2>
              <span
                v-if="wallboxData?.telemetry"
                class="wb-status-badge"
                :class="wallboxData.telemetry.status.toLowerCase()"
              >
                <span class="pulse-dot"></span>
                {{ wallboxData.telemetry.status }}
              </span>
            </div>
            <p class="section-subtitle">
              Modulazione dinamica IEC 61851 su eccedenza fotovoltaica netta con protezione anti-blackout contatore
            </p>
          </div>
        </div>

        <button
          type="button"
          class="wb-recalc-btn"
          :disabled="isUpdatingWallbox"
          @click="recalculateModulation"
          title="Ricalcola modulazione fotovoltaica"
        >
          <RefreshCw :size="16" :class="{ 'spin-icon': isUpdatingWallbox }" />
          <span>{{ isUpdatingWallbox ? 'Aggiornamento...' : 'Ricalcola Modulazione' }}</span>
        </button>
      </div>

      <!-- Live Telemetry Badges -->
      <div class="wb-telemetry-grid">
        <div class="wb-stat-card">
          <div class="wb-stat-top">
            <span class="wb-stat-label">Potenza Attiva</span>
            <PlugZap :size="18" class="text-purple" />
          </div>
          <div class="wb-stat-val">
            {{ wallboxData?.telemetry?.active_power_w ?? 0 }} <span class="unit">W</span>
          </div>
          <div class="wb-stat-sub">
            Target: <strong>{{ wallboxData?.modulation?.target_amps ?? 16 }}A</strong> ({{ (((wallboxData?.modulation?.expected_power_w ?? 3680)) / 1000).toFixed(1) }} kW)
          </div>
        </div>

        <div class="wb-stat-card">
          <div class="wb-stat-top">
            <span class="wb-stat-label">Corrente Erogata</span>
            <Gauge :size="18" class="text-info" />
          </div>
          <div class="wb-stat-val">
            {{ wallboxData?.telemetry?.delivered_amps ?? 0 }} <span class="unit">A</span>
          </div>
          <div class="wb-stat-sub">
            Pilota PWM: {{ wallboxData?.telemetry?.offered_amps ?? 16 }}A erogabili
          </div>
        </div>

        <div class="wb-stat-card">
          <div class="wb-stat-top">
            <span class="wb-stat-label">Batteria Veicolo (SoC)</span>
            <Battery :size="18" class="text-success" />
          </div>
          <div class="wb-stat-val">
            {{ wallboxData?.telemetry?.vehicle_soc_pct ?? 68 }} <span class="unit">%</span>
          </div>
          <div class="wb-soc-track">
            <div
              class="wb-soc-fill"
              :style="{ width: (wallboxData?.telemetry?.vehicle_soc_pct ?? 68) + '%' }"
            ></div>
          </div>
        </div>

        <div class="wb-stat-card">
          <div class="wb-stat-top">
            <span class="wb-stat-label">Energia Sessione</span>
            <Zap :size="18" class="text-warning" />
          </div>
          <div class="wb-stat-val">
            {{ (((wallboxData?.telemetry?.energy_wh ?? 14200)) / 1000).toFixed(2) }} <span class="unit">kWh</span>
          </div>
          <div class="wb-stat-sub">
            Stazione: {{ wallboxData?.charge_point_id ?? 'domus_wallbox_1' }}
          </div>
        </div>
      </div>

      <!-- Charging Mode Selectors -->
      <div class="wb-modes-section">
        <h3 class="wb-modes-title">Modalità di Gestione Carica</h3>
        <div class="wb-modes-grid">
          <button
            type="button"
            class="wb-mode-card"
            :class="{ active: selectedWallboxMode === 'solar_only' }"
            @click="setWallboxMode('solar_only')"
          >
            <div class="mode-icon-box solar">
              <Sun :size="20" />
            </div>
            <div class="mode-info">
              <div class="mode-title-row">
                <span class="mode-name">Solo Solare (100% FV)</span>
                <span v-if="selectedWallboxMode === 'solar_only'" class="active-badge">Attivo</span>
              </div>
              <p class="mode-desc">Ricarica esclusivamente con eccedenza fotovoltaica netta (&ge; 6A / 1.4kW). Zero prelievi da rete.</p>
            </div>
          </button>

          <button
            type="button"
            class="wb-mode-card"
            :class="{ active: selectedWallboxMode === 'solar_optimized' }"
            @click="setWallboxMode('solar_optimized')"
          >
            <div class="mode-icon-box hybrid">
              <Sparkles :size="20" />
            </div>
            <div class="mode-info">
              <div class="mode-title-row">
                <span class="mode-name">Solare Ottimizzato (Ibrido)</span>
                <span v-if="selectedWallboxMode === 'solar_optimized'" class="active-badge">Attivo</span>
              </div>
              <p class="mode-desc">Garantisce 6A continui (1.4kW) e aumenta la corrente dinamicamente se c'è surplus solare.</p>
            </div>
          </button>

          <button
            type="button"
            class="wb-mode-card"
            :class="{ active: selectedWallboxMode === 'fast_charging' }"
            @click="setWallboxMode('fast_charging')"
          >
            <div class="mode-icon-box fast">
              <Zap :size="20" />
            </div>
            <div class="mode-info">
              <div class="mode-title-row">
                <span class="mode-name">Fast Charge (Massima Potenza)</span>
                <span v-if="selectedWallboxMode === 'fast_charging'" class="active-badge">Attivo</span>
              </div>
              <p class="mode-desc">Carica alla massima potenza contrattuale disponibile con protezione anti-blackout contatore.</p>
            </div>
          </button>

          <button
            type="button"
            class="wb-mode-card"
            :class="{ active: selectedWallboxMode === 'scheduled_night' }"
            @click="setWallboxMode('scheduled_night')"
          >
            <div class="mode-icon-box night">
              <Clock :size="20" />
            </div>
            <div class="mode-info">
              <div class="mode-title-row">
                <span class="mode-name">Notturna Economica</span>
                <span v-if="selectedWallboxMode === 'scheduled_night'" class="active-badge">Attivo</span>
              </div>
              <p class="mode-desc">Pianifica la carica nelle fasce orarie F3 / PUN minimo notturno per massimizzare il risparmio.</p>
            </div>
          </button>
        </div>
      </div>

      <!-- Modulation Decision Banner -->
      <div v-if="wallboxData?.modulation" class="wb-decision-banner">
        <ShieldCheck :size="20" class="text-success" />
        <div class="decision-content">
          <span class="decision-title">Decisione Algoritmo di Bilanciamento Dinamico:</span>
          <span class="decision-text">{{ wallboxData.modulation.reason }}</span>
        </div>
        <div class="decision-stats">
          <span class="stat-pill">Surplus FV: {{ Math.round(wallboxData.modulation.solar_surplus_w) }} W</span>
          <span class="stat-pill">Carichi Casa: {{ Math.round(wallboxData.modulation.home_grid_power_w) }} W</span>
        </div>
      </div>
    </section>

    <!-- Dynamic Tariffs (PUN / Nord Pool / Octopus) & Flexible Loads Scheduler -->
    <section v-if="dynamicTariffSchedule" class="dynamic-tariffs-section glass-panel">
      <div class="dynamic-header">
        <div class="title-cluster">
          <div class="dynamic-badge-icon">
            <TrendingDown :size="20" class="text-emerald" />
          </div>
          <div>
            <h2 class="section-title">{{ t('energy.dynamicTitle') }}</h2>
            <p class="section-sub">
              {{ t('energy.dynamicSubtitle') }}
            </p>
          </div>
        </div>

        <button
          type="button"
          class="btn-optimize-pun"
          :disabled="evaluating"
          @click="triggerOptimize"
        >
          <Sparkles :size="16" :class="{ 'spin-anim': evaluating }" />
          <span>{{ evaluating ? t('energy.optimizing') : t('energy.optimizeNow') }}</span>
        </button>
      </div>

      <!-- PUN Key Figures Bar -->
      <div class="pun-metrics-row">
        <div class="pun-stat-box">
          <span class="pun-stat-lbl">{{ t('energy.marketAndDate') }}</span>
          <span class="pun-stat-val">{{ dynamicTariffSchedule.market }} ({{ dynamicTariffSchedule.date }})</span>
        </div>
        <div class="pun-stat-box cheapest">
          <span class="pun-stat-lbl">{{ t('energy.minPrice') }}</span>
          <span class="pun-stat-val text-emerald">{{ dynamicTariffSchedule.min_price.toFixed(3) }} €/kWh</span>
        </div>
        <div class="pun-stat-box">
          <span class="pun-stat-lbl">{{ t('energy.avgPrice') }}</span>
          <span class="pun-stat-val">{{ dynamicTariffSchedule.average_price.toFixed(3) }} €/kWh</span>
        </div>
        <div class="pun-stat-box peak">
          <span class="pun-stat-lbl">{{ t('energy.maxPrice') }}</span>
          <span class="pun-stat-val text-rose">{{ dynamicTariffSchedule.max_price.toFixed(3) }} €/kWh</span>
        </div>
      </div>

      <!-- 24-Hour Price Bar Chart -->
      <div class="hourly-chart-container">
        <div class="chart-header-row">
          <span class="chart-title"><Clock :size="15" /> {{ t('energy.hourlyCurve') }}</span>
          <div class="chart-legend">
            <span class="legend-item"><span class="color-dot cheapest"></span> {{ t('energy.legendMin') }}</span>
            <span class="legend-item"><span class="color-dot regular"></span> {{ t('energy.legendNormal') }}</span>
            <span class="legend-item"><span class="color-dot peak"></span> {{ t('energy.legendPeak') }}</span>
            <span class="legend-item"><span class="color-dot now"></span> {{ t('energy.legendNow') }}</span>
          </div>
        </div>

        <div class="hourly-bars-grid">
          <div
            v-for="item in dynamicTariffSchedule.hourly_prices"
            :key="item.hour"
            class="hourly-bar-col"
            :class="{
              'is-current': item.hour === currentHour,
              'is-cheapest': item.is_cheapest,
              'is-expensive': item.is_expensive
            }"
          >
            <div class="bar-tooltip">
              <strong>{{ String(item.hour).padStart(2, '0') }}:00</strong>
              <span>{{ item.price_per_kwh.toFixed(3) }} €/kWh</span>
              <span class="tooltip-slot">{{ item.slot }}</span>
            </div>

            <div class="bar-fill-track">
              <div
                class="bar-fill"
                :style="{ height: getBarHeight(item.price_per_kwh) + '%' }"
              />
            </div>

            <span class="bar-hour-label">{{ String(item.hour).padStart(2, '0') }}</span>
          </div>
        </div>
      </div>

      <!-- Flexible Loads Scheduler Grid -->
      <div class="flexible-loads-section">
        <div class="sub-header">
          <Layers :size="18" class="text-primary" />
          <h3 class="sub-title">{{ t('energy.flexibleTitle') }}</h3>
        </div>

        <div class="flexible-loads-grid">
          <div
            v-for="load in flexibleLoads"
            :key="load.entity_key"
            class="flexible-load-card"
          >
            <div class="load-card-top">
              <div>
                <h4 class="load-name">{{ load.name }}</h4>
                <span class="load-key font-mono">{{ load.entity_key }}</span>
              </div>
              <button
                type="button"
                class="toggle-switch"
                :class="{ active: load.auto_schedule }"
                @click="toggleAutoSchedule(load)"
                :title="load.auto_schedule ? 'Disattiva schedulazione automatica' : 'Attiva schedulazione automatica'"
              >
                <span class="toggle-thumb" />
              </button>
            </div>

            <div class="load-meta-row">
              <div class="meta-item">
                <span class="lbl">{{ t('energy.optWindow') }}:</span>
                <span class="val scheduled-badge">
                  {{ load.scheduled_hours.map(h => String(h).padStart(2, '0') + ':00').join(', ') }}
                </span>
              </div>
              <div class="meta-item">
                <span class="lbl">{{ t('energy.duration') }}:</span>
                <span class="val">{{ load.required_hours }}h</span>
              </div>
              <div class="meta-item">
                <span class="lbl">{{ t('energy.maxPriceThreshold') }}:</span>
                <span class="val">&le; {{ load.max_price_kwh }} €/kWh</span>
              </div>
            </div>

            <div class="load-card-footer">
              <span class="cost-stat">
                {{ t('energy.estCost') }}: <strong>{{ load.estimated_cost_eur.toFixed(2) }} €</strong>
              </span>
              <span class="savings-tag">
                {{ t('energy.savings') }} -{{ load.savings_percent }}%
              </span>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Tariffs & Utilities Grid -->
    <div class="bottom-grid">
      <!-- Italian Dynamic Tariffs (PUN / F1, F2, F3) -->
      <section class="tariff-card glass-panel">
        <div class="card-header">
          <Clock :size="20" class="text-primary" />
          <h3 class="card-title">{{ t('energy.areraTitle') }}</h3>
        </div>

        <div class="tariff-status-row">
          <div class="slot-pill active">
            <span class="slot-name">{{ t('energy.currentSlot') }}: {{ tariffData.current_slot }}</span>
            <span class="slot-price">{{ tariffData.current_price }} €/kWh</span>
          </div>
        </div>

        <div class="tariff-table">
          <div :class="['tariff-row', { current: tariffData.current_slot === 'F1' }]">
            <span>F1 (Picco: Lun-Ven 08-19)</span>
            <span class="price-col">{{ tariffData.tariffs.f1_price }} €/kWh</span>
          </div>
          <div :class="['tariff-row', { current: tariffData.current_slot === 'F2' }]">
            <span>F2 (Intermedia: Lun-Ven 07-08, 19-23, Sab 07-23)</span>
            <span class="price-col">{{ tariffData.tariffs.f2_price }} €/kWh</span>
          </div>
          <div :class="['tariff-row', { current: tariffData.current_slot === 'F3' }]">
            <span>F3 (Fuori Picco: Notti, Domenica & Festivi)</span>
            <span class="price-col">{{ tariffData.tariffs.f3_price }} €/kWh</span>
          </div>
        </div>
      </section>

      <!-- Water, Gas & Multi-Utility Metering -->
      <section class="utilities-card glass-panel">
        <div class="card-header">
          <Droplets :size="20" class="text-info" />
          <h3 class="card-title">{{ t('energy.utilitiesTitle') }}</h3>
          <span v-if="multiUtility" class="cost-badge">Totale: {{ multiUtility.total_cost_eur.toFixed(2) }} €</span>
        </div>

        <div class="utility-items">
          <div class="utility-item">
            <div class="u-icon electricity">
              <Zap :size="22" class="text-warning" />
            </div>
            <div class="u-meta">
              <span class="u-title">Elettricità</span>
              <span class="u-val">{{ (multiUtility?.electricity.total_consumption ?? (flow.house_consumption_watts / 1000)).toFixed(1) }} kWh</span>
              <span class="u-cost">Costo: {{ (multiUtility?.electricity.cost_eur ?? 0).toFixed(2) }} €</span>
            </div>
          </div>

          <div class="utility-item">
            <div class="u-icon water">
              <Droplets :size="22" />
            </div>
            <div class="u-meta">
              <span class="u-title">{{ t('energy.waterMeter') }}</span>
              <span class="u-val">{{ (multiUtility?.water.total_consumption ?? (waterVolumeM3 * 1000)).toFixed(0) }} L ({{ waterVolumeM3.toFixed(2) }} m³)</span>
              <span class="u-cost">Costo: {{ (multiUtility?.water.cost_eur ?? (waterVolumeM3 * tariffData.tariffs.water_m3_price)).toFixed(2) }} €</span>
            </div>
          </div>

          <div class="utility-item">
            <div class="u-icon gas">
              <Flame :size="22" />
            </div>
            <div class="u-meta">
              <span class="u-title">{{ t('energy.gasMeter') }}</span>
              <span class="u-val">{{ (multiUtility?.gas.total_consumption ?? gasVolumeSmc).toFixed(2) }} Smc</span>
              <span class="u-cost">Costo: {{ (multiUtility?.gas.cost_eur ?? (gasVolumeSmc * tariffData.tariffs.gas_smc_price)).toFixed(2) }} €</span>
            </div>
          </div>
        </div>

        <div v-if="utilityMeters.length > 0" class="submeters-list">
          <div v-for="m in utilityMeters" :key="m.id" class="submeter-chip">
            <span class="chip-name">{{ m.name }}</span>
            <span class="chip-val">{{ m.current_reading }} {{ m.unit }}</span>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.energy-view-container {
  padding: 1.5rem;
  max-width: 1400px;
  margin: 0 auto;
}

.energy-header {
  padding: 1.25rem 1.75rem;
  border-radius: 1rem;
  margin-bottom: 1.5rem;
  background: var(--surface-card);
  backdrop-filter: blur(16px);
  border: 1px solid var(--border-color);
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
  background: rgba(245, 158, 11, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.page-subtitle {
  font-size: 0.875rem;
  color: var(--text-muted);
  margin: 0.25rem 0 0;
}

/* Summary Cards */
.summary-cards-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 1.25rem;
  margin-bottom: 1.5rem;
}

.summary-card {
  padding: 1.25rem;
  border-radius: 1rem;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  gap: 1rem;
}

.card-meta {
  display: flex;
  flex-direction: column;
}

.meta-label {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.meta-val {
  font-size: 1.35rem;
  font-weight: 700;
  color: var(--text-main);
}

/* Sankey Section */
.sankey-section {
  padding: 1.75rem;
  border-radius: 1.25rem;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  margin-bottom: 1.5rem;
}

.section-title {
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--text-main);
  margin-bottom: 1rem;
}

.sankey-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
  margin-bottom: 1.25rem;
}

.bess-mode-pill {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 0.85rem;
  border-radius: 9999px;
  font-size: 0.85rem;
  font-weight: 700;
  background: rgba(6, 182, 212, 0.15);
  border: 1px solid rgba(6, 182, 212, 0.3);
  color: #22d3ee;
}

.bess-mode-pill.force_charge_grid {
  background: rgba(59, 130, 246, 0.2);
  border-color: #3b82f6;
  color: #60a5fa;
}

.bess-mode-pill.force_discharge {
  background: rgba(245, 158, 11, 0.2);
  border-color: #f59e0b;
  color: #fbbf24;
}

.bess-arbitrage-card {
  margin-top: 1.5rem;
  padding: 1.25rem;
  border-radius: 1rem;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.bess-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 1rem;
}

.bess-stat {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.bess-stat .label {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-transform: uppercase;
  font-weight: 600;
  letter-spacing: 0.05em;
}

.bess-stat .value {
  font-size: 1.15rem;
  font-weight: 800;
  color: var(--text-main);
}

.bess-action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.65rem 1.25rem;
  border-radius: 0.75rem;
  background: linear-gradient(135deg, #0ea5e9, #38bdf8);
  color: #0f172a;
  font-weight: 700;
  font-size: 0.875rem;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
  align-self: flex-start;
}

.bess-action-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(14, 165, 233, 0.35);
}

.bess-action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.sankey-svg-wrapper {
  width: 100%;
  display: flex;
  justify-content: center;
}

.sankey-svg {
  width: 100%;
  max-width: 900px;
  height: auto;
}

.sankey-flow-line {
  stroke-opacity: 0.35;
  transition: stroke-opacity 0.2s ease;
}

.sankey-hub-outer {
  fill: #0f172a;
  stroke: #0ea5e9;
  stroke-width: 2.5;
  transition: all 0.2s ease;
}

.sankey-hub-inner {
  fill: #1e293b;
  stroke: rgba(14, 165, 233, 0.4);
  stroke-width: 1.5;
  transition: all 0.2s ease;
}

.sankey-hub-label {
  fill: #94a3b8;
  font-size: 11px;
  font-weight: 700;
  text-anchor: middle;
  letter-spacing: 0.05em;
}

.sankey-hub-value {
  fill: #38bdf8;
  font-size: 13px;
  font-weight: 800;
  text-anchor: middle;
  letter-spacing: 0.02em;
}

.sankey-node-box {
  stroke-width: 1.5;
  transition: all 0.2s ease;
}

.node-solar .sankey-node-box {
  fill: rgba(245, 158, 11, 0.12);
  stroke: #f59e0b;
}

.node-grid .sankey-node-box {
  fill: rgba(59, 130, 246, 0.12);
  stroke: #3b82f6;
}

.node-house .sankey-node-box {
  fill: rgba(16, 185, 129, 0.12);
  stroke: #10b981;
}

.node-battery .sankey-node-box {
  fill: rgba(6, 182, 212, 0.12);
  stroke: #06b6d4;
}

.node-wallbox .sankey-node-box {
  fill: rgba(168, 85, 247, 0.12);
  stroke: #a855f7;
}

.sankey-node-title {
  text-anchor: middle;
  font-size: 11px;
  font-weight: 700;
}

.node-solar .sankey-node-title { fill: #fbbf24; }
.node-grid .sankey-node-title { fill: #60a5fa; }
.node-house .sankey-node-title { fill: #34d399; }
.node-battery .sankey-node-title { fill: #22d3ee; }
.node-wallbox .sankey-node-title { fill: #c084fc; }

.sankey-node-val {
  text-anchor: middle;
  font-size: 14px;
  font-weight: 800;
  fill: #ffffff;
}

.node-wallbox .sankey-node-val {
  font-size: 12.5px;
}

/* Bottom Grid */
.bottom-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 1.5rem;
}

.tariff-card,
.utilities-card {
  padding: 1.5rem;
  border-radius: 1rem;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1.25rem;
}

.card-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-main);
  margin: 0;
}

.cost-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.25rem 0.65rem;
  border-radius: 9999px;
  font-size: 0.8rem;
  font-weight: 700;
  background: rgba(14, 165, 233, 0.15);
  color: #38bdf8;
  border: 1px solid rgba(14, 165, 233, 0.3);
  margin-left: auto;
}

.slot-pill {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.2), rgba(37, 99, 235, 0.1));
  border: 1px solid #3b82f6;
  padding: 0.75rem 1.25rem;
  border-radius: 0.75rem;
  font-weight: 700;
  color: var(--text-main);
  margin-bottom: 1rem;
}

.tariff-table {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.tariff-row {
  display: flex;
  justify-content: space-between;
  padding: 0.6rem 0.8rem;
  border-radius: 0.5rem;
  font-size: 0.85rem;
  color: var(--text-muted);
}

.tariff-row.current {
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-main);
  font-weight: 600;
}

.price-col {
  font-weight: 700;
  color: var(--text-main);
}

/* Utilities */
.utility-items {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.utility-item {
  display: flex;
  align-items: center;
  gap: 1rem;
  background: rgba(31, 41, 55, 0.4);
  padding: 1rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.u-icon {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.u-icon.water {
  background: rgba(6, 182, 212, 0.15);
  color: #06b6d4;
}

.u-icon.gas {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
}

.u-meta {
  display: flex;
  flex-direction: column;
}

.u-title {
  font-size: 0.85rem;
  color: var(--text-muted);
}

.u-val {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-main);
}

.u-cost {
  font-size: 0.8rem;
  color: #10b981;
}

.submeters-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-color);
}

.submeter-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.35rem 0.75rem;
  border-radius: 0.5rem;
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-color);
  font-size: 0.8rem;
}

.chip-name {
  color: var(--text-muted);
}

.chip-val {
  font-weight: 700;
  color: var(--text-main);
  font-family: var(--font-mono, monospace);
}

/* Dynamic Tariffs Section & 24h Bar Chart */
.dynamic-tariffs-section {
  padding: 1.5rem;
  border-radius: 1rem;
  margin-bottom: 1.5rem;
  background: var(--surface-card);
  backdrop-filter: blur(16px);
  border: 1px solid var(--border-color);
}

.dynamic-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1.5rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.title-cluster {
  display: flex;
  gap: 1rem;
  align-items: flex-start;
}

.dynamic-badge-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: rgba(16, 185, 129, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
}

.section-sub {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin: 0.25rem 0 0;
  max-width: 680px;
}

.btn-optimize-pun {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.6rem 1.2rem;
  border-radius: 0.75rem;
  font-size: 0.875rem;
  font-weight: 600;
  color: #ffffff;
  background: linear-gradient(135deg, #10b981, #059669);
  border: none;
  cursor: pointer;
  box-shadow: 0 4px 14px rgba(16, 185, 129, 0.25);
  transition: all 0.2s ease;
}

.btn-optimize-pun:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(16, 185, 129, 0.35);
}

.btn-optimize-pun:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}

.spin-anim {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.pun-metrics-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.pun-stat-box {
  background: var(--bg-surface-hover);
  padding: 1rem 1.25rem;
  border-radius: 0.75rem;
  border: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.pun-stat-lbl {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.pun-stat-val {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-main);
}

.text-emerald {
  color: #10b981 !important;
}

.text-rose {
  color: #f43f5e !important;
}

/* 24-Hour Price Bar Chart */
.hourly-chart-container {
  background: var(--bg-surface);
  border: 1px solid var(--border-color);
  border-radius: 0.85rem;
  padding: 1.25rem;
  margin-bottom: 1.75rem;
}

.chart-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.chart-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-main);
}

.chart-legend {
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.75rem;
  color: #94a3b8;
}

.color-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.color-dot.cheapest {
  background: #10b981;
}

.color-dot.regular {
  background: #6366f1;
}

.color-dot.peak {
  background: #f43f5e;
}

.color-dot.now {
  background: #38bdf8;
  box-shadow: 0 0 6px #38bdf8;
}

.hourly-bars-grid {
  display: grid;
  grid-template-columns: repeat(24, 1fr);
  gap: 6px;
  align-items: flex-end;
  height: 160px;
  padding-bottom: 24px;
  position: relative;
}

.hourly-bar-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
  justify-content: flex-end;
  position: relative;
  cursor: pointer;
}

.bar-fill-track {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: flex-end;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 4px;
  overflow: hidden;
}

.bar-fill {
  width: 100%;
  border-radius: 4px 4px 0 0;
  background: linear-gradient(180deg, #6366f1, #4338ca);
  transition: height 0.4s ease;
}

.hourly-bar-col.is-cheapest .bar-fill {
  background: linear-gradient(180deg, #10b981, #059669);
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.3);
}

.hourly-bar-col.is-expensive .bar-fill {
  background: linear-gradient(180deg, #f43f5e, #be123c);
}

.hourly-bar-col.is-current {
  outline: 2px solid #38bdf8;
  outline-offset: 2px;
  border-radius: 6px;
}

.bar-hour-label {
  position: absolute;
  bottom: -22px;
  font-size: 0.7rem;
  color: #64748b;
  font-family: monospace;
}

.hourly-bar-col.is-current .bar-hour-label {
  color: #38bdf8;
  font-weight: 700;
}

.bar-tooltip {
  display: none;
  position: absolute;
  bottom: 110%;
  left: 50%;
  transform: translateX(-50%);
  background: #0f172a;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 6px;
  padding: 6px 8px;
  font-size: 0.7rem;
  white-space: nowrap;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  z-index: 10;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.5);
  pointer-events: none;
}

.hourly-bar-col:hover .bar-tooltip {
  display: flex;
}

.tooltip-slot {
  font-size: 0.65rem;
  color: #94a3b8;
}

/* Flexible Loads Grid */
.flexible-loads-section {
  margin-top: 1.5rem;
}

.sub-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.sub-title {
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--text-main);
  margin: 0;
}

.flexible-loads-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1rem;
}

.flexible-load-card {
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  border-radius: 0.85rem;
  padding: 1.1rem;
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  transition: border-color 0.2s ease;
}

.flexible-load-card:hover {
  border-color: rgba(16, 185, 129, 0.3);
}

.load-card-top {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.load-name {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-main);
  margin: 0;
}

.load-key {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.toggle-switch {
  width: 42px;
  height: 24px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.15);
  border: none;
  cursor: pointer;
  position: relative;
  padding: 2px;
  transition: background 0.25s ease;
}

.toggle-switch.active {
  background: #10b981;
}

.toggle-thumb {
  display: block;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #ffffff;
  transition: transform 0.25s ease;
}

.toggle-switch.active .toggle-thumb {
  transform: translateX(18px);
}

.load-meta-row {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: 0.8rem;
  background: var(--bg-surface-hover);
  padding: 0.65rem 0.8rem;
  border-radius: 0.5rem;
}

.meta-item {
  display: flex;
  justify-content: space-between;
}

.meta-item .lbl {
  color: #94a3b8;
}

.meta-item .val {
  color: var(--text-main);
  font-weight: 500;
}

.scheduled-badge {
  color: #10b981 !important;
  font-weight: 700 !important;
}

.load-card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.8rem;
  padding-top: 0.25rem;
}

.cost-stat {
  color: #94a3b8;
}

.savings-tag {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  font-weight: 700;
  padding: 0.2rem 0.5rem;
  border-radius: 0.4rem;
  font-size: 0.75rem;
  border: 1px solid rgba(16, 185, 129, 0.25);
}

/* Wallbox OCPP & EV Smart Charging */
.wallbox-section {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  padding: 1.5rem;
  border-radius: 1.25rem;
  background: var(--surface-card);
  backdrop-filter: blur(16px);
  border: 1px solid var(--border-color);
  margin-bottom: 1.5rem;
}

.wallbox-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
}

.wallbox-title-box {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.wallbox-icon-circle {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: rgba(168, 85, 247, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.wallbox-title-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.text-purple {
  color: #c084fc;
}

.wb-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.25rem 0.65rem;
  border-radius: 2rem;
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  background: rgba(148, 163, 184, 0.15);
  color: #94a3b8;
  border: 1px solid rgba(148, 163, 184, 0.25);
}

.wb-status-badge.charging {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  border-color: rgba(16, 185, 129, 0.3);
}

.wb-status-badge.suspendedevse,
.wb-status-badge.suspendedev {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  border-color: rgba(245, 158, 11, 0.3);
}

.pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: currentColor;
  box-shadow: 0 0 6px currentColor;
  animation: pulseWB 2s infinite ease-in-out;
}

@keyframes pulseWB {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 1; }
}

.wb-recalc-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.65rem 1.15rem;
  border-radius: 0.75rem;
  font-weight: 600;
  font-size: 0.85rem;
  background: rgba(168, 85, 247, 0.15);
  color: #c084fc;
  border: 1px solid rgba(168, 85, 247, 0.3);
  cursor: pointer;
  transition: all 0.2s ease;
}

.wb-recalc-btn:hover:not(:disabled) {
  background: rgba(168, 85, 247, 0.25);
  transform: translateY(-1px);
}

.wb-recalc-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.spin-icon {
  animation: spinWB 1s linear infinite;
}

@keyframes spinWB {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* Telemetry Grid */
.wb-telemetry-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
}

.wb-stat-card {
  padding: 1rem;
  border-radius: 0.85rem;
  background: rgba(15, 23, 42, 0.5);
  border: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.wb-stat-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.wb-stat-label {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.wb-stat-val {
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--text-main);
}

.wb-stat-val .unit {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-muted);
}

.wb-stat-sub {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.wb-soc-track {
  width: 100%;
  height: 6px;
  border-radius: 3px;
  background: rgba(148, 163, 184, 0.2);
  overflow: hidden;
  margin-top: 0.25rem;
}

.wb-soc-fill {
  height: 100%;
  background: linear-gradient(90deg, #10b981, #34d399);
  border-radius: 3px;
  transition: width 0.3s ease;
}

/* Modes Grid */
.wb-modes-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.wb-modes-title {
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.wb-modes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 1rem;
}

.wb-mode-card {
  display: flex;
  align-items: flex-start;
  gap: 0.85rem;
  padding: 1rem;
  border-radius: 0.85rem;
  background: rgba(15, 23, 42, 0.5);
  border: 1px solid var(--border-color);
  cursor: pointer;
  text-align: left;
  transition: all 0.2s ease;
  color: inherit;
}

.wb-mode-card:hover {
  background: rgba(30, 41, 59, 0.6);
  border-color: rgba(168, 85, 247, 0.4);
}

.wb-mode-card.active {
  background: rgba(168, 85, 247, 0.12);
  border-color: #a855f7;
  box-shadow: 0 0 16px rgba(168, 85, 247, 0.2);
}

.mode-icon-box {
  width: 38px;
  height: 38px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.mode-icon-box.solar {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
}

.mode-icon-box.hybrid {
  background: rgba(168, 85, 247, 0.15);
  color: #c084fc;
}

.mode-icon-box.fast {
  background: rgba(59, 130, 246, 0.15);
  color: #3b82f6;
}

.mode-icon-box.night {
  background: rgba(6, 182, 212, 0.15);
  color: #06b6d4;
}

.mode-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  flex: 1;
}

.mode-title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.mode-name {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--text-main);
}

.active-badge {
  font-size: 0.7rem;
  font-weight: 800;
  color: #c084fc;
  background: rgba(168, 85, 247, 0.2);
  padding: 0.15rem 0.4rem;
  border-radius: 0.3rem;
}

.mode-desc {
  font-size: 0.75rem;
  color: var(--text-muted);
  line-height: 1.35;
  margin: 0;
}

/* Decision Banner */
.wb-decision-banner {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.85rem 1.15rem;
  border-radius: 0.75rem;
  background: rgba(16, 185, 129, 0.08);
  border: 1px solid rgba(16, 185, 129, 0.2);
  flex-wrap: wrap;
}

.decision-content {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  flex: 1;
}

.decision-title {
  font-size: 0.75rem;
  font-weight: 700;
  color: #10b981;
  text-transform: uppercase;
}

.decision-text {
  font-size: 0.85rem;
  color: var(--text-main);
}

.decision-stats {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.stat-pill {
  font-size: 0.75rem;
  background: rgba(15, 23, 42, 0.6);
  padding: 0.25rem 0.55rem;
  border-radius: 0.4rem;
  border: 1px solid var(--border-color);
  color: var(--text-muted);
  font-family: var(--font-mono, monospace);
}

/* ═══════════════════════════════════════════════════════════════════════════
   LIGHT THEME CONTRAST & ACCESSIBILITY OVERRIDES (WCAG 2.2 AA / AAA)
   ═══════════════════════════════════════════════════════════════════════════ */

:root[data-theme="light"] .energy-header,
:root[data-theme="light"] .summary-card,
:root[data-theme="light"] .sankey-section,
:root[data-theme="light"] .tariff-card,
:root[data-theme="light"] .utilities-card,
:root[data-theme="light"] .dynamic-tariffs-section,
:root[data-theme="light"] .wallbox-section {
  background: #ffffff;
  border-color: #e2e8f0;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.05);
}

:root[data-theme="light"] .icon-circle {
  background: #fef3c7;
  color: #d97706;
}

:root[data-theme="light"] .meta-label,
:root[data-theme="light"] .page-subtitle,
:root[data-theme="light"] .section-subtitle,
:root[data-theme="light"] .section-sub,
:root[data-theme="light"] .load-key {
  color: #475569;
}

/* Sankey Diagram in Light Mode */
:root[data-theme="light"] .sankey-flow-line {
  stroke-opacity: 0.65;
}

:root[data-theme="light"] .sankey-hub-outer {
  fill: #f8fafc;
  stroke: #0284c7;
  stroke-width: 2.5;
}

:root[data-theme="light"] .sankey-hub-inner {
  fill: #ffffff;
  stroke: rgba(2, 132, 199, 0.35);
  stroke-width: 1.5;
}

:root[data-theme="light"] .sankey-hub-label {
  fill: #475569;
}

:root[data-theme="light"] .sankey-hub-value {
  fill: #0369a1;
}

:root[data-theme="light"] .node-solar .sankey-node-box {
  fill: #fffbeb;
  stroke: #d97706;
  stroke-width: 2;
}

:root[data-theme="light"] .node-grid .sankey-node-box {
  fill: #eff6ff;
  stroke: #2563eb;
  stroke-width: 2;
}

:root[data-theme="light"] .node-house .sankey-node-box {
  fill: #f0fdf4;
  stroke: #059669;
  stroke-width: 2;
}

:root[data-theme="light"] .node-battery .sankey-node-box {
  fill: #ecfeff;
  stroke: #0891b2;
  stroke-width: 2;
}

:root[data-theme="light"] .node-wallbox .sankey-node-box {
  fill: #faf5ff;
  stroke: #7e22ce;
  stroke-width: 2;
}

:root[data-theme="light"] .node-solar .sankey-node-title { fill: #b45309; }
:root[data-theme="light"] .node-grid .sankey-node-title { fill: #1d4ed8; }
:root[data-theme="light"] .node-house .sankey-node-title { fill: #047857; }
:root[data-theme="light"] .node-battery .sankey-node-title { fill: #0e7490; }
:root[data-theme="light"] .node-wallbox .sankey-node-title { fill: #6b21a8; }

:root[data-theme="light"] .sankey-node-val {
  fill: #0f172a;
}

/* BESS Section in Light Mode */
:root[data-theme="light"] .bess-mode-pill {
  background: #ecfeff;
  border-color: #67e8f9;
  color: #0e7490;
}

:root[data-theme="light"] .bess-mode-pill.force_charge_grid {
  background: #eff6ff;
  border-color: #93c5fd;
  color: #1d4ed8;
}

:root[data-theme="light"] .bess-mode-pill.force_discharge {
  background: #fffbeb;
  border-color: #fde68a;
  color: #b45309;
}

:root[data-theme="light"] .bess-arbitrage-card {
  background: #f8fafc;
  border-color: #e2e8f0;
}

:root[data-theme="light"] .bess-stat .label {
  color: #475569;
}

:root[data-theme="light"] .bess-stat .value {
  color: #0f172a;
}

:root[data-theme="light"] .bess-stat .value.text-primary {
  color: #0284c7;
}

:root[data-theme="light"] .bess-stat .value.text-success {
  color: #047857;
}

:root[data-theme="light"] .bess-action-btn {
  background: linear-gradient(135deg, #0284c7, #0ea5e9);
  color: #ffffff;
  font-weight: 700;
}

/* Tariffs in Light Mode */
:root[data-theme="light"] .slot-pill {
  background: #eff6ff;
  border-color: #3b82f6;
  color: #1e40af;
}

:root[data-theme="light"] .slot-pill .slot-price {
  color: #1e3a8a;
  font-weight: 800;
}

:root[data-theme="light"] .tariff-row {
  color: #475569;
}

:root[data-theme="light"] .tariff-row.current {
  background: #f1f5f9;
  color: #0f172a;
  font-weight: 700;
}

:root[data-theme="light"] .price-col {
  color: #0f172a;
}

/* Utilities in Light Mode */
:root[data-theme="light"] .utility-item {
  background: #f8fafc;
  border-color: #e2e8f0;
}

:root[data-theme="light"] .u-title {
  color: #475569;
}

:root[data-theme="light"] .u-val {
  color: #0f172a;
}

:root[data-theme="light"] .u-cost {
  color: #047857;
  font-weight: 600;
}

:root[data-theme="light"] .cost-badge {
  background: #e0f2fe;
  color: #0369a1;
  border-color: #7dd3fc;
}

:root[data-theme="light"] .submeter-chip {
  background: #f1f5f9;
  border-color: #cbd5e1;
}

:root[data-theme="light"] .chip-name {
  color: #475569;
}

:root[data-theme="light"] .chip-val {
  color: #0f172a;
}

/* Wallbox in Light Mode */
:root[data-theme="light"] .text-purple {
  color: #7e22ce !important;
}

:root[data-theme="light"] .wallbox-icon-circle {
  background: #f3e8ff;
  color: #7e22ce;
}

:root[data-theme="light"] .wb-status-badge {
  background: #f1f5f9;
  color: #334155;
  border-color: #cbd5e1;
}

:root[data-theme="light"] .wb-status-badge.charging {
  background: #ecfdf5;
  color: #047857;
  border-color: #a7f3d0;
}

:root[data-theme="light"] .wb-status-badge.suspendedevse,
:root[data-theme="light"] .wb-status-badge.suspendedev {
  background: #fffbeb;
  color: #b45309;
  border-color: #fde68a;
}

:root[data-theme="light"] .wb-recalc-btn {
  background: #f3e8ff;
  color: #6b21a8;
  border-color: #d8b4fe;
}

:root[data-theme="light"] .wb-recalc-btn:hover:not(:disabled) {
  background: #e9d5ff;
  color: #581c87;
}

:root[data-theme="light"] .wb-stat-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
}

:root[data-theme="light"] .wb-stat-label,
:root[data-theme="light"] .wb-stat-sub {
  color: #475569;
}

:root[data-theme="light"] .wb-stat-val {
  color: #0f172a;
}

:root[data-theme="light"] .wb-stat-sub strong {
  color: #0f172a;
}

:root[data-theme="light"] .wb-soc-track {
  background: #e2e8f0;
}

:root[data-theme="light"] .wb-mode-card {
  background: #ffffff;
  border: 1.5px solid #e2e8f0;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.05);
}

:root[data-theme="light"] .wb-mode-card:hover {
  background: #f8fafc;
  border-color: #c084fc;
}

:root[data-theme="light"] .wb-mode-card.active {
  background: #faf5ff;
  border-color: #7e22ce;
  box-shadow: 0 0 12px rgba(126, 34, 206, 0.15);
}

:root[data-theme="light"] .mode-icon-box.solar {
  background: #fef3c7;
  color: #b45309;
}

:root[data-theme="light"] .mode-icon-box.hybrid {
  background: #f3e8ff;
  color: #7e22ce;
}

:root[data-theme="light"] .mode-icon-box.fast {
  background: #eff6ff;
  color: #1d4ed8;
}

:root[data-theme="light"] .mode-icon-box.night {
  background: #ecfeff;
  color: #0e7490;
}

:root[data-theme="light"] .mode-name {
  color: #0f172a;
}

:root[data-theme="light"] .mode-desc {
  color: #475569;
}

:root[data-theme="light"] .active-badge {
  color: #581c87;
  background: #f3e8ff;
  border: 1px solid #d8b4fe;
}

:root[data-theme="light"] .wb-decision-banner {
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
}

:root[data-theme="light"] .decision-title {
  color: #047857;
}

:root[data-theme="light"] .decision-text {
  color: #064e3b;
  font-weight: 500;
}

:root[data-theme="light"] .stat-pill {
  background: #ffffff;
  border: 1px solid #a7f3d0;
  color: #065f46;
  font-weight: 600;
}

/* PUN & Dynamic Tariffs in Light Mode */
:root[data-theme="light"] .dynamic-badge-icon {
  background: #ecfdf5;
  color: #059669;
}

:root[data-theme="light"] .pun-stat-box {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
}

:root[data-theme="light"] .pun-stat-lbl {
  color: #475569;
  font-weight: 700;
}

:root[data-theme="light"] .pun-stat-val {
  color: #0f172a;
}

:root[data-theme="light"] .text-emerald {
  color: #047857 !important;
}

:root[data-theme="light"] .text-rose {
  color: #be123c !important;
}

:root[data-theme="light"] .hourly-chart-container {
  background: #f8fafc;
  border-color: #e2e8f0;
}

:root[data-theme="light"] .chart-title {
  color: #0f172a;
}

:root[data-theme="light"] .legend-item {
  color: #475569;
  font-weight: 600;
}

:root[data-theme="light"] .bar-fill-track {
  background: rgba(15, 23, 42, 0.06);
}

:root[data-theme="light"] .bar-hour-label {
  color: #475569;
  font-weight: 600;
}

:root[data-theme="light"] .hourly-bar-col.is-current .bar-hour-label {
  color: #0284c7;
  font-weight: 800;
}

:root[data-theme="light"] .bar-tooltip {
  background: #0f172a;
  color: #ffffff;
}

:root[data-theme="light"] .bar-tooltip .tooltip-slot {
  color: #cbd5e1;
}

:root[data-theme="light"] .sub-title {
  color: #0f172a;
}

:root[data-theme="light"] .flexible-load-card {
  background: #ffffff;
  border: 1.5px solid #e2e8f0;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.05);
}

:root[data-theme="light"] .flexible-load-card:hover {
  border-color: #10b981;
}

:root[data-theme="light"] .load-name {
  color: #0f172a;
}

:root[data-theme="light"] .toggle-switch {
  background: #cbd5e1;
}

:root[data-theme="light"] .toggle-switch.active {
  background: #059669;
}

:root[data-theme="light"] .load-meta-row {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
}

:root[data-theme="light"] .meta-item .lbl {
  color: #475569;
}

:root[data-theme="light"] .meta-item .val {
  color: #0f172a;
}

:root[data-theme="light"] .scheduled-badge {
  color: #047857 !important;
}

:root[data-theme="light"] .cost-stat {
  color: #334155;
}

:root[data-theme="light"] .cost-stat strong {
  color: #0f172a;
}

:root[data-theme="light"] .savings-tag {
  background: #ecfdf5;
  color: #047857;
  border: 1px solid #a7f3d0;
}
</style>
