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
} from '@lucide/vue'
import { apiClient } from '../api/client'
import type { MultiUtilitySummary, UtilityMeter } from '../api/types'

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
})

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
          <line x1="170" y1="95" x2="400" y2="200" stroke="#f59e0b" stroke-width="4" stroke-opacity="0.3" stroke-dasharray="6 4" />
          <line x1="170" y1="305" x2="400" y2="200" stroke="#3b82f6" stroke-width="4" stroke-opacity="0.3" stroke-dasharray="6 4" />
          <line x1="400" y1="200" x2="630" y2="95" stroke="#10b981" stroke-width="4" stroke-opacity="0.3" stroke-dasharray="6 4" />
          <line x1="400" y1="200" x2="630" y2="305" stroke="#06b6d4" stroke-width="4" stroke-opacity="0.3" stroke-dasharray="6 4" />
          <line x1="400" y1="200" x2="400" y2="340" stroke="#a855f7" stroke-width="4" stroke-opacity="0.3" stroke-dasharray="6 4" />

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

          <!-- Central Hub: Inverter Ibrido -->
          <g transform="translate(400, 200)">
            <circle r="46" fill="#1e293b" stroke="#64748b" stroke-width="3" />
            <circle r="38" fill="rgba(30, 41, 59, 0.8)" />
            <text y="-8" fill="#94a3b8" font-size="11" font-weight="700" text-anchor="middle">INVERTER</text>
            <text y="14" fill="#38bdf8" font-size="14" font-weight="800" text-anchor="middle">
              {{ flow.solar_watts + Math.abs(flow.battery_watts) }} W
            </text>
          </g>

          <!-- Node: Solar Fotovoltaico (Top-Left) -->
          <g transform="translate(100, 60)">
            <rect width="140" height="70" rx="12" fill="rgba(245, 158, 11, 0.15)" stroke="url(#solarGrad)" stroke-width="2" />
            <text x="70" y="28" fill="#f59e0b" font-weight="700" text-anchor="middle" font-size="13">☀️ SOLARE</text>
            <text x="70" y="52" fill="#ffffff" font-weight="800" text-anchor="middle" font-size="16">{{ flow.solar_watts }} W</text>
          </g>

          <!-- Node: Grid Rete Elettrica (Bottom-Left) -->
          <g transform="translate(100, 270)">
            <rect width="140" height="70" rx="12" fill="rgba(59, 130, 246, 0.15)" stroke="url(#gridGrad)" stroke-width="2" />
            <text x="70" y="28" fill="#3b82f6" font-weight="700" text-anchor="middle" font-size="13">🌐 RETE</text>
            <text x="70" y="52" fill="#ffffff" font-weight="800" text-anchor="middle" font-size="16">{{ Math.abs(flow.grid_watts) }} W</text>
          </g>

          <!-- Node: House Casa (Top-Right) -->
          <g transform="translate(560, 60)">
            <rect width="140" height="70" rx="12" fill="rgba(16, 185, 129, 0.15)" stroke="url(#houseGrad)" stroke-width="2" />
            <text x="70" y="28" fill="#10b981" font-weight="700" text-anchor="middle" font-size="13">🏠 CARICHI CASA</text>
            <text x="70" y="52" fill="#ffffff" font-weight="800" text-anchor="middle" font-size="16">{{ flow.house_consumption_watts }} W</text>
          </g>

          <!-- Node: Battery Accumulo BESS (Bottom-Right) -->
          <g transform="translate(560, 270)">
            <rect width="140" height="70" rx="12" fill="rgba(6, 182, 212, 0.15)" stroke="url(#batteryGrad)" stroke-width="2" />
            <text x="70" y="28" fill="#06b6d4" font-weight="700" text-anchor="middle" font-size="13">🔋 BATTERIA ({{ flow.battery_soc }}%)</text>
            <text x="70" y="52" fill="#ffffff" font-weight="800" text-anchor="middle" font-size="16">{{ Math.abs(flow.battery_watts) }} W</text>
          </g>

          <!-- Node: Wallbox EV (Bottom-Center) -->
          <g transform="translate(330, 310)">
            <rect width="140" height="58" rx="10" fill="rgba(168, 85, 247, 0.15)" stroke="url(#evGrad)" stroke-width="2" />
            <text x="70" y="24" fill="#c084fc" font-weight="700" text-anchor="middle" font-size="12">🚗 WALLBOX EV</text>
            <text x="70" y="44" fill="#ffffff" font-weight="800" text-anchor="middle" font-size="14">
              {{ flow.house_consumption_watts > 1500 ? '1800 W' : '0 W (Standby)' }}
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
  background: rgba(245, 158, 11, 0.15);
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
  background: rgba(17, 24, 39, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.08);
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
  color: #9ca3af;
}

.meta-val {
  font-size: 1.35rem;
  font-weight: 700;
  color: #ffffff;
}

/* Sankey Section */
.sankey-section {
  padding: 1.75rem;
  border-radius: 1.25rem;
  background: rgba(17, 24, 39, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.08);
  margin-bottom: 1.5rem;
}

.section-title {
  font-size: 1.15rem;
  font-weight: 700;
  color: #f3f4f6;
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
  background: rgba(15, 23, 42, 0.65);
  border: 1px solid rgba(255, 255, 255, 0.08);
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
  color: #94a3b8;
  text-transform: uppercase;
  font-weight: 600;
  letter-spacing: 0.05em;
}

.bess-stat .value {
  font-size: 1.15rem;
  font-weight: 800;
  color: #f8fafc;
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
  background: rgba(17, 24, 39, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.08);
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
  color: #f3f4f6;
  margin: 0;
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
  color: #ffffff;
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
  color: #9ca3af;
}

.tariff-row.current {
  background: rgba(255, 255, 255, 0.05);
  color: #f3f4f6;
  font-weight: 600;
}

.price-col {
  font-weight: 700;
  color: #f3f4f6;
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
  color: #9ca3af;
}

.u-val {
  font-size: 1.25rem;
  font-weight: 700;
  color: #ffffff;
}

.u-cost {
  font-size: 0.8rem;
  color: #10b981;
}

/* Dynamic Tariffs Section & 24h Bar Chart */
.dynamic-tariffs-section {
  padding: 1.5rem;
  border-radius: 1rem;
  margin-bottom: 1.5rem;
  background: rgba(17, 24, 39, 0.7);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.08);
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
  color: #9ca3af;
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
  background: rgba(31, 41, 55, 0.4);
  padding: 1rem 1.25rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.05);
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.pun-stat-lbl {
  font-size: 0.75rem;
  color: #9ca3af;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.pun-stat-val {
  font-size: 1.25rem;
  font-weight: 700;
  color: #ffffff;
}

.text-emerald {
  color: #10b981 !important;
}

.text-rose {
  color: #f43f5e !important;
}

/* 24-Hour Price Bar Chart */
.hourly-chart-container {
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.06);
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
  color: #e2e8f0;
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
  color: #f1f5f9;
  margin: 0;
}

.flexible-loads-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1rem;
}

.flexible-load-card {
  background: rgba(31, 41, 55, 0.5);
  border: 1px solid rgba(255, 255, 255, 0.07);
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
  color: #ffffff;
  margin: 0;
}

.load-key {
  font-size: 0.75rem;
  color: #94a3b8;
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
  background: rgba(15, 23, 42, 0.4);
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
  color: #f1f5f9;
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
</style>
