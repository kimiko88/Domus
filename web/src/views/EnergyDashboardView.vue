<script setup lang="ts">
import { ref, onMounted } from 'vue'
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
  Sparkles
} from '@lucide/vue'
import { apiClient } from '../api/client'

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
  solar_watts: 4200,
  grid_watts: -950, // exporting
  battery_watts: -1800, // charging
  battery_soc: 82,
  house_consumption_watts: 1450,
  solar_to_house_watts: 1450,
  solar_to_battery_watts: 1800,
  solar_to_grid_watts: 950,
  battery_to_house_watts: 0,
  grid_to_house_watts: 0,
  grid_to_battery_watts: 0
})

const tariffData = ref({
  current_slot: 'F1',
  current_price: 0.28,
  currency: 'EUR',
  tariffs: {
    f1_price: 0.28,
    f2_price: 0.24,
    f3_price: 0.19,
    gas_smc_price: 0.95,
    water_m3_price: 1.80
  }
})

// Utilities Telemetry
const waterVolumeM3 = ref(14.85)
const gasVolumeSmc = ref(122.40)

onMounted(async () => {
  try {
    const f = await apiClient.get<SankeyFlow>('/api/v1/energy/flow')
    if (f) flow.value = f
  } catch {}

  try {
    const t = await apiClient.get<any>('/api/v1/energy/tariffs')
    if (t) tariffData.value = t
  } catch {}
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
          <h1 class="page-title">Monitoraggio Energetico & Flusso Solare</h1>
          <p class="page-subtitle">Vettori energetici in tempo reale, fasce orarie ARERA e telemetria utenze</p>
        </div>
      </div>
    </header>

    <!-- Vector Summary Badges -->
    <section class="summary-cards-row">
      <div class="summary-card solar">
        <Sun :size="24" class="text-warning" />
        <div class="card-meta">
          <span class="meta-label">Produzione Solare</span>
          <span class="meta-val">{{ flow.solar_watts }} W</span>
        </div>
      </div>

      <div class="summary-card battery">
        <Battery :size="24" class="text-info" />
        <div class="card-meta">
          <span class="meta-label">Batteria ({{ flow.battery_soc }}%)</span>
          <span class="meta-val">{{ Math.abs(flow.battery_watts) }} W {{ flow.battery_watts < 0 ? 'Carica' : 'Scarica' }}</span>
        </div>
      </div>

      <div class="summary-card grid">
        <Zap :size="24" class="text-primary" />
        <div class="card-meta">
          <span class="meta-label">Rete Elettrica</span>
          <span class="meta-val">{{ flow.grid_watts < 0 ? 'Immissione ' + Math.abs(flow.grid_watts) + ' W' : 'Prelievo ' + flow.grid_watts + ' W' }}</span>
        </div>
      </div>

      <div class="summary-card house">
        <Home :size="24" class="text-success" />
        <div class="card-meta">
          <span class="meta-label">Consumo Casa</span>
          <span class="meta-val">{{ flow.house_consumption_watts }} W</span>
        </div>
      </div>
    </section>

    <!-- Sankey Flow Diagram SVG -->
    <section class="sankey-section glass-panel">
      <h2 class="section-title">Diagramma di Sankey Vettori Energetici</h2>
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

    <!-- Tariffs & Utilities Grid -->
    <div class="bottom-grid">
      <!-- Italian Dynamic Tariffs (PUN / F1, F2, F3) -->
      <section class="tariff-card glass-panel">
        <div class="card-header">
          <Clock :size="20" class="text-primary" />
          <h3 class="card-title">Tariffe Elettriche Fasce PUN (ARERA)</h3>
        </div>

        <div class="tariff-status-row">
          <div class="slot-pill active">
            <span class="slot-name">Fascia Attuale: {{ tariffData.current_slot }}</span>
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

      <!-- Water and Gas Metering -->
      <section class="utilities-card glass-panel">
        <div class="card-header">
          <Droplets :size="20" class="text-info" />
          <h3 class="card-title">Monitoraggio Acqua e Gas</h3>
        </div>

        <div class="utility-items">
          <div class="utility-item">
            <div class="u-icon water">
              <Droplets :size="22" />
            </div>
            <div class="u-meta">
              <span class="u-title">Contatore Acqua Sanitaria</span>
              <span class="u-val">{{ waterVolumeM3.toFixed(2) }} m³</span>
              <span class="u-cost">Costo stimato: {{ (waterVolumeM3 * tariffData.tariffs.water_m3_price).toFixed(2) }} €</span>
            </div>
          </div>

          <div class="utility-item">
            <div class="u-icon gas">
              <Flame :size="22" />
            </div>
            <div class="u-meta">
              <span class="u-title">Contatore Gas Metano</span>
              <span class="u-val">{{ gasVolumeSmc.toFixed(2) }} Smc</span>
              <span class="u-cost">Costo stimato: {{ (gasVolumeSmc * tariffData.tariffs.gas_smc_price).toFixed(2) }} €</span>
            </div>
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
</style>
