<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Chart from 'chart.js/auto'
import { useClimateHistoryStore, defaultClimateHistorySettings } from '@/stores/climateHistory'
import { useEntitiesStore } from '@/stores/entities'
import type { ClimateHistoryPoint, ClimateHistorySettings } from '@/api/types'

const { t, locale } = useI18n()
const store = useClimateHistoryStore()
const entitiesStore = useEntitiesStore()

type RangeKey = '24h' | '7d' | '30d' | 'custom'
type MetricKey = 'temperature' | 'humidity' | 'target_temperature'
const ranges: RangeKey[] = ['24h', '7d', '30d', 'custom']
const selectedRange = ref<RangeKey>('24h')
const selectedMetric = ref<MetricKey>('temperature')
const selectedArea = ref('')
const selectedEntities = ref<string[]>([])
const dateFrom = ref('')
const dateTo = ref('')
const canvas = ref<HTMLCanvasElement | null>(null)
const savingMessage = ref('')
const loadError = ref('')
const settingsForm = reactive<ClimateHistorySettings>({ ...defaultClimateHistorySettings })
let chart: Chart | null = null
let mounted = false

const availableAreas = computed(() => {
  const values = new Map<string, string>()
  store.points.forEach(point => {
    const id = point.area_id || 'unassigned'
    if (!values.has(id)) values.set(id, id === 'unassigned' ? t('climateHistory.unassigned') : id)
  })
  return [...values.entries()].map(([id, label]) => ({ id, label }))
})

const availableSeries = computed(() => {
  const map = new Map<string, string>()
  store.points.forEach(point => {
    if (!selectedArea.value || (point.area_id || 'unassigned') === selectedArea.value) {
      map.set(point.entity_id, point.entity_name || point.entity_key)
    }
  })
  return [...map.entries()].map(([id, name]) => ({ id, name }))
})

const filteredPoints = computed(() => store.points.filter(point => {
  const areaMatches = !selectedArea.value || (point.area_id || 'unassigned') === selectedArea.value
  const entityMatches = selectedEntities.value.length === 0 || selectedEntities.value.includes(point.entity_id)
  return areaMatches && entityMatches
}))

function valueFor(point: ClimateHistoryPoint): number | null {
  const value = selectedMetric.value === 'temperature'
    ? point.temperature_average
    : selectedMetric.value === 'humidity'
      ? point.humidity_average
      : point.target_temperature_average
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}

const chartData = computed(() => {
  const grouped = new Map<string, ClimateHistoryPoint[]>()
  filteredPoints.value.forEach(point => {
    const current = grouped.get(point.entity_id) || []
    current.push(point)
    grouped.set(point.entity_id, current)
  })
  const timestamps = [...new Set(filteredPoints.value.map(point => point.recorded_at))].sort()
  const labels = timestamps.map(value => new Date(value).toLocaleString(locale.value, {
    day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit'
  }))
  const palette = ['#0ea5e9', '#22c55e', '#f59e0b', '#a78bfa', '#f43f5e', '#14b8a6', '#e879f9', '#84cc16']
  const datasets = [...grouped.entries()].map(([id, series], index) => {
    const byTimestamp = new Map(series.map(point => [point.recorded_at, valueFor(point)]))
    const sample = series[series.length - 1]
    return {
      label: sample.entity_name || sample.entity_key || id,
      data: timestamps.map(timestamp => byTimestamp.get(timestamp) ?? null),
      borderColor: palette[index % palette.length],
      backgroundColor: palette[index % palette.length],
      borderWidth: 2,
      tension: 0.25,
      spanGaps: false,
      pointRadius: timestamps.length < 40 ? 2 : 0
    }
  })
  return { labels, datasets }
})

const metricUnit = computed(() => selectedMetric.value === 'humidity' ? '%' : '°C')
const visibleRooms = computed(() => store.summary?.rooms || [])
const hasClimateReadings = computed(() => store.points.length > 0)

function rangeQuery() {
  const to = selectedRange.value === 'custom' && dateTo.value
    ? new Date(dateTo.value + 'T23:59:59').toISOString()
    : new Date().toISOString()
  const from = selectedRange.value === 'custom' && dateFrom.value
    ? new Date(dateFrom.value + 'T00:00:00').toISOString()
    : new Date(Date.now() - (selectedRange.value === '7d' ? 7 : selectedRange.value === '30d' ? 30 : 1) * 86400000).toISOString()
  return { from, to, area_id: selectedArea.value || undefined, limit: 10000 }
}

async function loadDashboard() {
  loadError.value = ''
  try {
    await store.refresh(rangeQuery())
    if (selectedEntities.value.length === 0) {
      selectedEntities.value = availableSeries.value.map(series => series.id)
    } else {
      const available = new Set(availableSeries.value.map(series => series.id))
      selectedEntities.value = selectedEntities.value.filter(id => available.has(id))
    }
  } catch {
    loadError.value = store.error || t('climateHistory.errors.load')
  }
}

function renderChart() {
  if (!canvas.value) return
  chart?.destroy()
  chart = new Chart(canvas.value, {
    type: 'line',
    data: chartData.value,
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        legend: { display: true, position: 'bottom', labels: { usePointStyle: true, boxWidth: 8 } },
        tooltip: {
          callbacks: {
            label: context => String(context.dataset.label) + ': ' + String(context.parsed.y ?? '—') + ' ' + metricUnit.value
          }
        }
      },
      scales: {
        x: { grid: { display: false }, ticks: { maxTicksLimit: 10, maxRotation: 0 } },
        y: { title: { display: true, text: metricUnit.value }, grace: '5%' }
      }
    }
  })
}

async function saveSettings() {
  savingMessage.value = ''
  try {
    const saved = await store.saveSettings({ ...settingsForm })
    Object.assign(settingsForm, saved)
    savingMessage.value = t('climateHistory.settings.saved')
    await loadDashboard()
  } catch {
    savingMessage.value = store.error || t('climateHistory.errors.save')
  }
}

async function exportCSV() {
  try {
    await store.exportCSV(rangeQuery())
  } catch {
    loadError.value = t('climateHistory.errors.export')
  }
}

function formatNumber(value?: number, digits = 1) {
  if (value === undefined || value === null || !Number.isFinite(value)) return '—'
  return new Intl.NumberFormat(locale.value, { maximumFractionDigits: digits, minimumFractionDigits: digits }).format(value)
}
function roomName(areaId: string) {
  return availableAreas.value.find(area => area.id === areaId)?.label || areaId
}
function insightLabel(code: string) {
  return t('climateHistory.insights.' + code)
}

watch(chartData, async () => {
  await nextTick()
  if (mounted) renderChart()
}, { deep: true })
watch([selectedRange, selectedArea, dateFrom, dateTo], () => {
  if (mounted) loadDashboard()
})
watch(selectedMetric, async () => {
  await nextTick()
  if (mounted) renderChart()
})

onMounted(async () => {
  try {
    await Promise.all([store.fetchSettings(), entitiesStore.fetchEntities()])
    Object.assign(settingsForm, store.settings)
  } catch {
    loadError.value = store.error || t('climateHistory.errors.load')
  }
  await loadDashboard()
  mounted = true
  await nextTick()
  renderChart()
})
onBeforeUnmount(() => {
  mounted = false
  chart?.destroy()
  chart = null
})
</script>

<template>
  <main class="climate-history-view">
    <header class="page-header">
      <div>
        <h1>{{ t('climateHistory.title') }}</h1>
        <p>{{ t('climateHistory.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <button class="secondary-btn" :disabled="store.loading" @click="loadDashboard">{{ t('climateHistory.refresh') }}</button>
        <button class="secondary-btn" :disabled="store.loading" @click="exportCSV">{{ t('climateHistory.exportCsv') }}</button>
      </div>
    </header>

    <p v-if="loadError" class="feedback error" role="alert">{{ loadError }}</p>

    <section class="toolbar panel" :aria-label="t('climateHistory.filters')">
      <div class="range-control">
        <button v-for="range in ranges" :key="range" class="range-btn"
          :class="{ active: selectedRange === range }" @click="selectedRange = range">
          {{ t('climateHistory.ranges.' + range) }}
        </button>
      </div>
      <label>
        <span>{{ t('climateHistory.room') }}</span>
        <select v-model="selectedArea">
          <option value="">{{ t('climateHistory.allRooms') }}</option>
          <option v-for="area in availableAreas" :key="area.id" :value="area.id">{{ area.label }}</option>
        </select>
      </label>
      <label>
        <span>{{ t('climateHistory.metric') }}</span>
        <select v-model="selectedMetric">
          <option value="temperature">{{ t('climateHistory.metrics.temperature') }}</option>
          <option value="humidity">{{ t('climateHistory.metrics.humidity') }}</option>
          <option value="target_temperature">{{ t('climateHistory.metrics.target') }}</option>
        </select>
      </label>
      <label v-if="selectedRange === 'custom'">
        <span>{{ t('climateHistory.from') }}</span>
        <input v-model="dateFrom" type="date">
      </label>
      <label v-if="selectedRange === 'custom'">
        <span>{{ t('climateHistory.to') }}</span>
        <input v-model="dateTo" type="date">
      </label>
    </section>

    <section class="panel series-panel" v-if="availableSeries.length">
      <label class="series-label" for="climate-series">{{ t('climateHistory.selectSeries') }}</label>
      <select id="climate-series" v-model="selectedEntities" multiple>
        <option v-for="series in availableSeries" :key="series.id" :value="series.id">{{ series.name }}</option>
      </select>
      <small>{{ t('climateHistory.selectSeriesHint') }}</small>
    </section>

    <section class="panel chart-panel">
      <div class="section-heading">
        <div>
          <h2>{{ t('climateHistory.trendTitle') }}</h2>
          <p>{{ t('climateHistory.chartDescription') }}</p>
        </div>
        <span class="sample-count">{{ t('climateHistory.samples', { count: filteredPoints.length }) }}</span>
      </div>
      <div v-if="store.loading" class="empty-state" role="status">{{ t('climateHistory.loading') }}</div>
      <div v-else-if="!hasClimateReadings" class="empty-state">{{ t('climateHistory.noData') }}</div>
      <div v-else class="chart-wrap">
        <canvas ref="canvas" role="img" :aria-label="t('climateHistory.chartAria')"></canvas>
      </div>
    </section>

    <section v-if="settingsForm.room_comparison_enabled || settingsForm.room_map_enabled" class="room-grid">
      <article v-for="room in visibleRooms" :key="room.area_id" class="panel room-card">
        <div class="room-heading">
          <h3>{{ roomName(room.area_id) }}</h3>
          <span>{{ room.sample_count }} {{ t('climateHistory.samplesShort') }}</span>
        </div>
        <div class="room-metrics">
          <div>
            <span>{{ t('climateHistory.temperature') }}</span>
            <strong>{{ formatNumber(room.current_temperature) }} °C</strong>
            <small>{{ t('climateHistory.average') }} {{ formatNumber(room.average_temperature) }} °C</small>
          </div>
          <div>
            <span>{{ t('climateHistory.humidity') }}</span>
            <strong>{{ formatNumber(room.current_humidity) }}%</strong>
            <small>{{ t('climateHistory.average') }} {{ formatNumber(room.average_humidity) }}%</small>
          </div>
        </div>
        <div v-if="settingsForm.thermal_analysis_enabled && room.heating_rate_c_per_hour !== undefined" class="thermal-info">
          <span>{{ t('climateHistory.thermalRate') }}: {{ formatNumber(room.heating_rate_c_per_hour) }} °C/h</span>
          <span v-if="settingsForm.forecast_enabled && room.forecast_temperature_1h !== undefined">
            {{ t('climateHistory.forecast') }}: {{ formatNumber(room.forecast_temperature_1h) }} °C
          </span>
        </div>
        <div class="minmax">{{ t('climateHistory.minMax') }}: {{ formatNumber(room.minimum_temperature) }} / {{ formatNumber(room.maximum_temperature) }} °C</div>
      </article>
    </section>

    <section v-if="store.summary?.insights?.length" class="panel insights-panel">
      <div class="section-heading">
        <div>
          <h2>{{ t('climateHistory.insightsTitle') }}</h2>
          <p>{{ t('climateHistory.insightsDescription') }}</p>
        </div>
      </div>
      <ul>
        <li v-for="(insight, index) in store.summary.insights" :key="insight.code + '-' + insight.area_id + '-' + index">
          <span class="insight-dot" aria-hidden="true"></span>
          <span>{{ insightLabel(insight.code) }} <strong>{{ roomName(insight.area_id || 'unassigned') }}</strong></span>
          <span class="insight-value" v-if="insight.value !== undefined">{{ formatNumber(insight.value) }} {{ insight.metric === 'humidity' ? '%' : '°C' }}</span>
        </li>
      </ul>
    </section>

    <details class="panel settings-panel">
      <summary>{{ t('climateHistory.settings.title') }}</summary>
      <p class="settings-description">{{ t('climateHistory.settings.description') }}</p>
      <div class="settings-grid">
        <label class="switch-row"><input v-model="settingsForm.enabled" type="checkbox"><span>{{ t('climateHistory.settings.enabled') }}</span></label>
        <label>
          <span>{{ t('climateHistory.settings.interval') }}</span>
          <select v-model.number="settingsForm.interval_minutes">
            <option v-for="value in [1,2,5,10,15,30,60]" :key="value" :value="value">{{ value }} {{ t('climateHistory.settings.minutes') }}</option>
          </select>
        </label>
        <label class="switch-row"><input v-model="settingsForm.record_temperature" type="checkbox"><span>{{ t('climateHistory.settings.recordTemperature') }}</span></label>
        <label class="switch-row"><input v-model="settingsForm.record_humidity" type="checkbox"><span>{{ t('climateHistory.settings.recordHumidity') }}</span></label>
        <label class="switch-row"><input v-model="settingsForm.record_thermostat" type="checkbox"><span>{{ t('climateHistory.settings.recordThermostat') }}</span></label>
        <label class="switch-row"><input v-model="settingsForm.smart_sampling" type="checkbox"><span>{{ t('climateHistory.settings.smartSampling') }}</span></label>
        <label><span>{{ t('climateHistory.settings.temperatureThreshold') }}</span><input v-model.number="settingsForm.temperature_change_threshold" type="number" min="0.1" max="10" step="0.1"></label>
        <label><span>{{ t('climateHistory.settings.humidityThreshold') }}</span><input v-model.number="settingsForm.humidity_change_threshold" type="number" min="0.1" max="30" step="0.5"></label>
        <label><span>{{ t('climateHistory.settings.rawRetention') }}</span><input v-model.number="settingsForm.raw_retention_days" type="number" min="1" max="30"></label>
        <label><span>{{ t('climateHistory.settings.hourlyRetention') }}</span><input v-model.number="settingsForm.hourly_retention_days" type="number" min="30" max="365"></label>
      </div>

      <h3>{{ t('climateHistory.settings.analysisTitle') }}</h3>
      <div class="settings-grid toggles-grid">
        <label class="switch-row"><input v-model="settingsForm.temperature_alerts_enabled" type="checkbox"><span>{{ t('climateHistory.settings.temperatureAlerts') }}</span></label>
        <label class="switch-row"><input v-model="settingsForm.humidity_alerts_enabled" type="checkbox"><span>{{ t('climateHistory.settings.humidityAlerts') }}</span></label>
        <label class="switch-row"><input v-model="settingsForm.comfort_analysis_enabled" type="checkbox"><span>{{ t('climateHistory.settings.comfortAnalysis') }}</span></label>
        <label class="switch-row"><input v-model="settingsForm.room_comparison_enabled" type="checkbox"><span>{{ t('climateHistory.settings.roomComparison') }}</span></label>
        <label class="switch-row"><input v-model="settingsForm.thermal_analysis_enabled" type="checkbox"><span>{{ t('climateHistory.settings.thermalAnalysis') }}</span></label>
        <label class="switch-row"><input v-model="settingsForm.anomaly_detection_enabled" type="checkbox"><span>{{ t('climateHistory.settings.anomalyDetection') }}</span></label>
        <label class="switch-row"><input v-model="settingsForm.forecast_enabled" type="checkbox"><span>{{ t('climateHistory.settings.forecast') }}</span></label>
        <label class="switch-row"><input v-model="settingsForm.room_map_enabled" type="checkbox"><span>{{ t('climateHistory.settings.roomMap') }}</span></label>
      </div>
      <h3>{{ t('climateHistory.settings.thresholdsTitle') }}</h3>
      <div class="settings-grid">
        <label><span>{{ t('climateHistory.settings.lowTemperature') }}</span><input v-model.number="settingsForm.low_temperature_limit" type="number" min="-10" max="50" step="0.5"></label>
        <label><span>{{ t('climateHistory.settings.highTemperature') }}</span><input v-model.number="settingsForm.high_temperature_limit" type="number" min="-10" max="50" step="0.5"></label>
        <label><span>{{ t('climateHistory.settings.lowHumidity') }}</span><input v-model.number="settingsForm.low_humidity_limit" type="number" min="0" max="100"></label>
        <label><span>{{ t('climateHistory.settings.highHumidity') }}</span><input v-model.number="settingsForm.high_humidity_limit" type="number" min="0" max="100"></label>
      </div>
      <div class="settings-footer">
        <span v-if="savingMessage" class="feedback" role="status">{{ savingMessage }}</span>
        <button class="primary-btn" :disabled="store.saving" @click="saveSettings">
          {{ store.saving ? t('climateHistory.settings.saving') : t('climateHistory.settings.save') }}
        </button>
      </div>
    </details>
  </main>
</template>

<style scoped>
.climate-history-view { display:flex; flex-direction:column; gap:1.1rem; padding:1.25rem 0; min-width:0; }
.page-header,.section-heading,.room-heading,.settings-footer { display:flex; justify-content:space-between; align-items:center; gap:1rem; }
.page-header { flex-wrap:wrap; }
h1,h2,h3,p { margin:0; }
h1 { font-size:1.65rem; font-weight:750; }
h2 { font-size:1.08rem; font-weight:700; }
h3 { font-size:1rem; font-weight:650; }
.page-header p,.section-heading p,.settings-description { color:var(--text-muted, #94a3b8); margin-top:.3rem; font-size:.9rem; }
.header-actions { display:flex; flex-wrap:wrap; gap:.5rem; }
.panel { min-width:0; padding:1rem; border:1px solid var(--border-color, rgba(148,163,184,.2)); border-radius:1rem; background:var(--surface-card, rgba(15,23,42,.35)); }
.toolbar { display:flex; gap:.8rem; flex-wrap:wrap; align-items:end; }
.toolbar label,.settings-grid label { display:flex; flex-direction:column; gap:.35rem; min-width:9rem; flex:1; }
.toolbar label span,.settings-grid label>span { color:var(--text-muted,#94a3b8); font-size:.78rem; }
select,input[type="date"],input[type="number"] { width:100%; min-height:2.45rem; color:var(--text-primary,#e2e8f0); background:var(--surface-input,#111827); border:1px solid var(--border-color,rgba(148,163,184,.3)); border-radius:.55rem; padding:.45rem .6rem; }
select[multiple] { min-height:6rem; }
.range-control { display:flex; flex-wrap:wrap; gap:.25rem; padding:.2rem; background:rgba(148,163,184,.1); border-radius:.65rem; }
.range-btn,.secondary-btn,.primary-btn { border:1px solid var(--border-color,rgba(148,163,184,.25)); border-radius:.6rem; padding:.55rem .8rem; color:var(--text-primary,#e2e8f0); background:transparent; cursor:pointer; }
.range-btn { border:0; font-size:.83rem; }
.range-btn.active { background:var(--accent-color,#0ea5e9); color:white; }
.secondary-btn:hover,.range-btn:hover { background:rgba(148,163,184,.15); }
.primary-btn { background:var(--accent-color,#0ea5e9); border-color:transparent; color:white; }
button:disabled { opacity:.55; cursor:not-allowed; }
.series-panel { display:grid; grid-template-columns:minmax(0,1fr); gap:.45rem; }
.series-label { font-size:.85rem; font-weight:600; }
.series-panel small { color:var(--text-muted,#94a3b8); }
.chart-panel { min-height:19rem; }
.sample-count { white-space:nowrap; color:var(--text-muted,#94a3b8); font-size:.8rem; }
.chart-wrap { height:18rem; position:relative; margin-top:1rem; }
.empty-state { display:grid; place-items:center; min-height:15rem; color:var(--text-muted,#94a3b8); text-align:center; }
.room-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(min(100%,16rem),1fr)); gap:.8rem; }
.room-card { display:flex; flex-direction:column; gap:.9rem; }
.room-heading span,.minmax { color:var(--text-muted,#94a3b8); font-size:.75rem; }
.room-metrics { display:grid; grid-template-columns:1fr 1fr; gap:.6rem; }
.room-metrics>div { display:flex; flex-direction:column; gap:.25rem; }
.room-metrics span,.room-metrics small { color:var(--text-muted,#94a3b8); font-size:.78rem; }
.room-metrics strong { font-size:1.35rem; }
.thermal-info { display:flex; flex-direction:column; gap:.25rem; font-size:.78rem; color:var(--text-muted,#94a3b8); }
.insights-panel ul { list-style:none; display:flex; flex-direction:column; gap:.65rem; padding:0; margin:1rem 0 0; }
.insights-panel li { display:flex; align-items:center; gap:.55rem; font-size:.88rem; }
.insight-dot { width:.5rem; height:.5rem; border-radius:50%; background:#f59e0b; flex:none; }
.insight-value { margin-left:auto; font-variant-numeric:tabular-nums; white-space:nowrap; }
.settings-panel summary { cursor:pointer; font-weight:700; }
.settings-description { margin:.6rem 0 1rem; }
.settings-panel h3 { margin:1.2rem 0 .7rem; }
.settings-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(min(100%,13rem),1fr)); gap:.75rem; }
.switch-row { display:flex!important; flex-direction:row!important; align-items:center; gap:.6rem!important; min-height:2.3rem; }
.switch-row input { width:1.1rem; height:1.1rem; accent-color:var(--accent-color,#0ea5e9); }
.switch-row span { color:var(--text-primary,#e2e8f0)!important; }
.settings-footer { margin-top:1.2rem; flex-wrap:wrap; justify-content:flex-end; }
.feedback { font-size:.85rem; color:var(--text-muted,#94a3b8); }
.feedback.error { color:#f87171; }
@media (max-width:640px) { .page-header { align-items:flex-start; } .header-actions { width:100%; } .header-actions button { flex:1; } .toolbar { align-items:stretch; } .range-control { width:100%; } .range-btn { flex:1; min-width:3rem; } .section-heading { align-items:flex-start; flex-direction:column; } .chart-wrap { height:15rem; } }
</style>
