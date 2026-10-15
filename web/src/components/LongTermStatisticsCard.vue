<template>
  <div class="statistics-card" role="region" aria-label="Statistiche di lungo periodo">
    <div class="card-header">
      <div class="title-group">
        <h3>{{ title || 'Statistiche di Lungo Periodo' }}</h3>
        <span v-if="entityKey" class="entity-sub">{{ entityKey }}</span>
      </div>
      <div class="range-selector" role="group" aria-label="Seleziona intervallo">
        <button
          v-for="r in ranges"
          :key="r.id"
          class="range-btn"
          :class="{ active: selectedRange === r.id }"
          @click="changeRange(r.id)"
          :aria-pressed="selectedRange === r.id"
        >
          {{ r.label }}
        </button>
      </div>
    </div>

    <!-- Summary Metrics -->
    <div class="metrics-grid">
      <div class="metric-box">
        <span class="metric-label">Media</span>
        <span class="metric-val">{{ stats.mean !== undefined ? stats.mean.toFixed(1) : '-' }} <small>{{ unit }}</small></span>
      </div>
      <div class="metric-box">
        <span class="metric-label">Minimo</span>
        <span class="metric-val min">{{ stats.min !== undefined ? stats.min.toFixed(1) : '-' }} <small>{{ unit }}</small></span>
      </div>
      <div class="metric-box">
        <span class="metric-label">Massimo</span>
        <span class="metric-val max">{{ stats.max !== undefined ? stats.max.toFixed(1) : '-' }} <small>{{ unit }}</small></span>
      </div>
      <div class="metric-box">
        <span class="metric-label">Totale</span>
        <span class="metric-val">{{ stats.sum !== undefined ? stats.sum.toFixed(1) : '-' }} <small>{{ unit }}</small></span>
      </div>
    </div>

    <!-- Aggregate SVG Chart -->
    <div class="chart-container">
      <svg class="stats-svg" viewBox="0 0 500 160" preserveAspectRatio="none" role="img" aria-label="Grafico a barre delle statistiche">
        <g v-for="(item, index) in series" :key="index">
          <rect
            :x="index * (500 / series.length) + 4"
            :y="160 - getBarHeight(item.val)"
            :width="Math.max(6, (500 / series.length) - 8)"
            :height="getBarHeight(item.val)"
            rx="3"
            class="stat-bar"
          />
          <text
            :x="index * (500 / series.length) + (500 / series.length) / 2"
            y="152"
            text-anchor="middle"
            class="bar-label"
          >
            {{ item.label }}
          </text>
        </g>
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

const props = withDefaults(
  defineProps<{
    title?: string
    entityKey?: string
    unit?: string
    initialRange?: string
  }>(),
  {
    title: 'Statistiche Storiche',
    entityKey: '',
    unit: 'kWh',
    initialRange: 'month',
  }
)

const emit = defineEmits<{
  (e: 'range-change', range: string): void
}>()

const selectedRange = ref(props.initialRange)

const ranges = [
  { id: 'day', label: '24h' },
  { id: 'week', label: '7 Giorni' },
  { id: 'month', label: 'Mese' },
  { id: 'year', label: 'Anno' },
]

// Mocked/reactive statistics
const series = computed(() => {
  if (selectedRange.value === 'day') {
    return [
      { label: '00', val: 1.2 }, { label: '04', val: 0.8 }, { label: '08', val: 3.4 },
      { label: '12', val: 4.5 }, { label: '16', val: 3.8 }, { label: '20', val: 5.1 },
    ]
  }
  if (selectedRange.value === 'week') {
    return [
      { label: 'Lun', val: 14.2 }, { label: 'Mar', val: 16.5 }, { label: 'Mer', val: 15.0 },
      { label: 'Gio', val: 13.8 }, { label: 'Ven', val: 17.2 }, { label: 'Sab', val: 21.0 }, { label: 'Dom', val: 19.5 },
    ]
  }
  if (selectedRange.value === 'year') {
    return [
      { label: 'Gen', val: 320 }, { label: 'Feb', val: 280 }, { label: 'Mar', val: 240 },
      { label: 'Apr', val: 190 }, { label: 'Mag', val: 150 }, { label: 'Giu', val: 130 },
      { label: 'Lug', val: 140 }, { label: 'Ago', val: 160 }, { label: 'Set', val: 180 },
      { label: 'Ott', val: 220 }, { label: 'Nov', val: 290 }, { label: 'Dic', val: 340 },
    ]
  }
  // month
  return [
    { label: 'S1', val: 84 }, { label: 'S2', val: 92 }, { label: 'S3', val: 78 }, { label: 'S4', val: 88 },
  ]
})

const stats = computed(() => {
  const vals = series.value.map((s) => s.val)
  if (vals.length === 0) return { mean: 0, min: 0, max: 0, sum: 0 }
  const sum = vals.reduce((a, b) => a + b, 0)
  return {
    mean: sum / vals.length,
    min: Math.min(...vals),
    max: Math.max(...vals),
    sum,
  }
})

function getBarHeight(val: number): number {
  const max = stats.value.max || 1
  return Math.max(10, (val / max) * 120)
}

function changeRange(rangeId: string) {
  selectedRange.value = rangeId
  emit('range-change', rangeId)
}
</script>

<style scoped>
.statistics-card {
  background: var(--surface-card, rgba(30, 41, 59, 0.7));
  border: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.08));
  border-radius: var(--radius-lg, 12px);
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  backdrop-filter: blur(12px);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.title-group h3 {
  margin: 0;
  font-size: 1.15rem;
  color: var(--text-primary, #ffffff);
}

.entity-sub {
  font-size: 0.8rem;
  color: var(--text-secondary, #94a3b8);
  font-family: monospace;
}

.range-selector {
  display: flex;
  gap: 0.35rem;
  background: rgba(15, 23, 42, 0.6);
  padding: 0.25rem;
  border-radius: var(--radius-md, 8px);
}

.range-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary, #94a3b8);
  padding: 0.35rem 0.65rem;
  font-size: 0.8rem;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
  min-height: 36px;
}

.range-btn:hover {
  color: #ffffff;
}

.range-btn.active {
  background: var(--primary, #38bdf8);
  color: #0f172a;
  font-weight: 600;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.75rem;
}

@media (max-width: 600px) {
  .metrics-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

.metric-box {
  background: rgba(15, 23, 42, 0.5);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 8px;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
}

.metric-label {
  font-size: 0.75rem;
  color: var(--text-secondary, #94a3b8);
  text-transform: uppercase;
}

.metric-val {
  font-size: 1.25rem;
  font-weight: 700;
  color: #ffffff;
  margin-top: 0.25rem;
}

.metric-val small {
  font-size: 0.8rem;
  font-weight: 400;
  color: var(--text-secondary, #94a3b8);
}

.metric-val.min {
  color: #38bdf8;
}

.metric-val.max {
  color: #f87171;
}

.chart-container {
  height: 160px;
  width: 100%;
}

.stats-svg {
  width: 100%;
  height: 100%;
}

.stat-bar {
  fill: var(--primary, #38bdf8);
  opacity: 0.85;
  transition: height 0.3s ease, y 0.3s ease, opacity 0.2s ease;
}

.stat-bar:hover {
  opacity: 1;
  filter: drop-shadow(0 0 6px var(--primary, #38bdf8));
}

.bar-label {
  fill: #94a3b8;
  font-size: 10px;
}
</style>
