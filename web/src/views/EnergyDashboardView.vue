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
})
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

    <!-- Sankey Flow Diagram SVG -->
    <section class="sankey-section glass-panel">
      <h2 class="section-title">{{ t('energy.sankeyTitle') }}</h2>
      <div class="sankey-svg-wrapper">
        <svg viewBox="0 0 800 360" class="sankey-svg">
          <defs>
            <!-- Solar to House Gradient -->
            <linearGradient id="solarToHouse" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stop-color="#f59e0b" />
              <stop offset="100%" stop-color="#10b981" />
            </linearGradient>

            <!-- Solar to Battery Gradient -->
            <linearGradient id="solarToBat" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stop-color="#f59e0b" />
              <stop offset="100%" stop-color="#06b6d4" />
            </linearGradient>

            <!-- Solar to Grid Gradient -->
            <linearGradient id="solarToGrid" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stop-color="#f59e0b" />
              <stop offset="100%" stop-color="#3b82f6" />
            </linearGradient>
          </defs>

          <!-- Source: Solar -->
          <g transform="translate(60, 60)">
            <rect width="140" height="70" rx="10" fill="rgba(245, 158, 11, 0.15)" stroke="#f59e0b" stroke-width="2" />
            <text x="70" y="32" fill="#f59e0b" font-weight="700" text-anchor="middle" font-size="14">FOTOVOLTAICO</text>
            <text x="70" y="52" fill="#ffffff" font-weight="800" text-anchor="middle" font-size="16">{{ flow.solar_watts }} W</text>
          </g>

          <!-- Source/Sink: Grid -->
          <g transform="translate(60, 220)">
            <rect width="140" height="70" rx="10" fill="rgba(59, 130, 246, 0.15)" stroke="#3b82f6" stroke-width="2" />
            <text x="70" y="32" fill="#3b82f6" font-weight="700" text-anchor="middle" font-size="14">RETE ELETTRICA</text>
            <text x="70" y="52" fill="#ffffff" font-weight="800" text-anchor="middle" font-size="16">{{ Math.abs(flow.grid_watts) }} W</text>
          </g>

          <!-- Destination: House -->
          <g transform="translate(600, 60)">
            <rect width="140" height="70" rx="10" fill="rgba(16, 185, 129, 0.15)" stroke="#10b981" stroke-width="2" />
            <text x="70" y="32" fill="#10b981" font-weight="700" text-anchor="middle" font-size="14">CASA</text>
            <text x="70" y="52" fill="#ffffff" font-weight="800" text-anchor="middle" font-size="16">{{ flow.house_consumption_watts }} W</text>
          </g>

          <!-- Storage: Battery -->
          <g transform="translate(600, 220)">
            <rect width="140" height="70" rx="10" fill="rgba(6, 182, 212, 0.15)" stroke="#06b6d4" stroke-width="2" />
            <text x="70" y="32" fill="#06b6d4" font-weight="700" text-anchor="middle" font-size="14">BATTERIA ({{ flow.battery_soc }}%)</text>
            <text x="70" y="52" fill="#ffffff" font-weight="800" text-anchor="middle" font-size="16">{{ Math.abs(flow.battery_watts) }} W</text>
          </g>

          <!-- Flows / Paths -->
          <!-- Solar -> House -->
          <path
            d="M 200 95 C 400 95, 400 95, 600 95"
            fill="none"
            stroke="url(#solarToHouse)"
            stroke-width="16"
            stroke-linecap="round"
            opacity="0.85"
          />

          <!-- Solar -> Battery -->
          <path
            d="M 200 95 C 380 95, 420 255, 600 255"
            fill="none"
            stroke="url(#solarToBat)"
            stroke-width="14"
            stroke-linecap="round"
            opacity="0.8"
          />

          <!-- Solar -> Grid -->
          <path
            d="M 200 95 C 260 120, 260 230, 200 255"
            fill="none"
            stroke="url(#solarToGrid)"
            stroke-width="10"
            stroke-linecap="round"
            opacity="0.75"
          />
        </svg>
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
