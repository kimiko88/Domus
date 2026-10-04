<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useEntitiesStore } from '@/stores/entities'
import { useHistoryStore } from '@/stores/history'
import SensorHistoryChart from '@/components/charts/SensorHistoryChart.vue'

const { t } = useI18n()
const route = useRoute()
const entitiesStore = useEntitiesStore()
const historyStore = useHistoryStore()

type TimeRangeKey = '1h' | '24h' | '7d' | '30d'

interface RangeOption {
  key: TimeRangeKey
  labelKey: string
  durationHours: number
}

const timeRanges: RangeOption[] = [
  { key: '1h', labelKey: 'history.oneHour', durationHours: 1 },
  { key: '24h', labelKey: 'history.twentyFourHours', durationHours: 24 },
  { key: '7d', labelKey: 'history.sevenDays', durationHours: 24 * 7 },
  { key: '30d', labelKey: 'history.thirtyDays', durationHours: 24 * 30 }
]

const selectedRange = ref<TimeRangeKey>('24h')
const selectedEntityId = ref<string>('')

// Available entities (prioritizing sensors)
const availableEntities = computed(() => {
  return [...entitiesStore.allEntities].sort((a, b) => {
    if (a.domain === 'sensor' && b.domain !== 'sensor') return -1
    if (b.domain === 'sensor' && a.domain !== 'sensor') return 1
    return a.name.localeCompare(b.name)
  })
})

const currentEntity = computed(() => {
  return entitiesStore.byId(selectedEntityId.value)
})

const currentHistory = computed(() => {
  return historyStore.getHistory(selectedEntityId.value)
})

// Statistics computation
const statistics = computed(() => {
  const entries = currentHistory.value
  if (!entries || entries.length === 0) return null

  const numbers: number[] = []
  for (const entry of entries) {
    let num: number | null = null
    if (typeof entry.state === 'number') num = entry.state
    else if (typeof entry.state === 'string') {
      const parsed = parseFloat(entry.state)
      if (!isNaN(parsed)) num = parsed
    }
    if (num !== null) numbers.push(num)
  }

  if (numbers.length === 0) return null

  const min = Math.min(...numbers)
  const max = Math.max(...numbers)
  const sum = numbers.reduce((acc, curr) => acc + curr, 0)
  const avg = +(sum / numbers.length).toFixed(1)
  const current = numbers[numbers.length - 1]

  return { min, max, avg, current }
})

async function loadHistory() {
  if (!selectedEntityId.value) return

  const range = timeRanges.find(r => r.key === selectedRange.value)
  const hours = range ? range.durationHours : 24

  const to = new Date()
  const from = new Date(to.getTime() - hours * 3600 * 1000)

  try {
    await historyStore.fetchHistory(selectedEntityId.value, {
      from: from.toISOString(),
      to: to.toISOString(),
      limit: 300
    })
  } catch {
    // handled in store
  }
}

watch(selectedEntityId, () => {
  loadHistory()
})

watch(selectedRange, () => {
  loadHistory()
})

onMounted(async () => {
  if (entitiesStore.allEntities.length === 0) {
    try {
      await entitiesStore.fetchEntities()
    } catch {
      // ignore
    }
  }

  // Pre-select entity from query param or first available entity
  const queryId = route?.query?.entity_id as string | undefined
  if (queryId) {
    selectedEntityId.value = queryId
  } else if (availableEntities.value.length > 0) {
    selectedEntityId.value = availableEntities.value[0].id
  }
})
</script>

<template>
  <div class="history-view">
    <!-- Header -->
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t('history.title') }}</h1>
        <p class="page-subtitle">{{ t('history.subtitle') }}</p>
      </div>

      <!-- Controls: Entity select & Range pills -->
      <div class="header-controls">
        <select
          v-model="selectedEntityId"
          class="entity-picker"
          aria-label="Seleziona entità"
        >
          <option v-for="ent in availableEntities" :key="ent.id" :value="ent.id">
            [{{ ent.domain }}] {{ ent.name }}
          </option>
        </select>

        <div class="range-selector">
          <button
            v-for="range in timeRanges"
            :key="range.key"
            class="range-btn"
            :class="{ active: selectedRange === range.key }"
            @click="selectedRange = range.key"
          >
            {{ t(range.labelKey) }}
          </button>
        </div>
      </div>
    </header>

    <!-- Metrics Stat Cards -->
    <div v-if="statistics" class="stats-grid">
      <div class="stat-card">
        <span class="stat-label">{{ t('history.current') }}</span>
        <div class="stat-value text-accent">
          {{ statistics.current }}
          <span class="stat-unit">{{ currentEntity?.attributes?.unit_of_measurement }}</span>
        </div>
      </div>

      <div class="stat-card">
        <span class="stat-label">{{ t('history.min') }}</span>
        <div class="stat-value text-cyan">
          {{ statistics.min }}
          <span class="stat-unit">{{ currentEntity?.attributes?.unit_of_measurement }}</span>
        </div>
      </div>

      <div class="stat-card">
        <span class="stat-label">{{ t('history.max') }}</span>
        <div class="stat-value text-emerald">
          {{ statistics.max }}
          <span class="stat-unit">{{ currentEntity?.attributes?.unit_of_measurement }}</span>
        </div>
      </div>

      <div class="stat-card">
        <span class="stat-label">{{ t('history.avg') }}</span>
        <div class="stat-value text-muted">
          {{ statistics.avg }}
          <span class="stat-unit">{{ currentEntity?.attributes?.unit_of_measurement }}</span>
        </div>
      </div>
    </div>

    <!-- Main Chart Section -->
    <section class="chart-section">
      <SensorHistoryChart
        :history="currentHistory"
        :unit="currentEntity?.attributes?.unit_of_measurement"
        :label="currentEntity?.name"
        :loading="historyStore.loading"
      />
    </section>

    <!-- Raw Data Log Table -->
    <section v-if="currentHistory.length > 0" class="logs-section glass-panel">
      <div class="logs-header">
        <h2 class="section-title">{{ t('history.logs') }} ({{ currentHistory.length }})</h2>
      </div>

      <div class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th>{{ t('history.recordedAt') }}</th>
              <th>{{ t('history.value') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in currentHistory.slice().reverse().slice(0, 50)" :key="row.id || row.recorded_at">
              <td class="timestamp-col">{{ new Date(row.recorded_at).toLocaleString() }}</td>
              <td class="value-col">
                <code>{{ typeof row.state === 'object' ? JSON.stringify(row.state) : row.state }}</code>
                <span v-if="currentEntity?.attributes?.unit_of_measurement" class="unit-badge">
                  {{ currentEntity.attributes.unit_of_measurement }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<style scoped>
.history-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  padding: 1.5rem 0;
}

.page-header {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

@media (min-width: 768px) {
  .page-header {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }
}

.page-title {
  font-size: 1.6rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.page-subtitle {
  font-size: 0.9rem;
  color: var(--text-muted);
  margin: 0.25rem 0 0;
}

.header-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
}

.entity-picker {
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  padding: 0.5rem 1rem;
  border-radius: var(--radius-md, 10px);
  font-size: 0.9rem;
  outline: none;
  backdrop-filter: var(--blur-card);
}

.range-selector {
  display: flex;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md, 10px);
  padding: 2px;
}

.range-btn {
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: 0.85rem;
  font-weight: 500;
  padding: 0.4rem 0.75rem;
  border-radius: var(--radius-sm, 8px);
  cursor: pointer;
  transition: all 0.2s ease;
}

.range-btn:hover {
  color: var(--text-main);
}

.range-btn.active {
  background: rgba(6, 182, 212, 0.18);
  color: #0284c7;
  font-weight: 600;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
}

@media (min-width: 768px) {
  .stats-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

.stat-card {
  background: var(--bg-glass-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg, 14px);
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  box-shadow: var(--shadow-sm);
}

.stat-label {
  font-size: 0.8rem;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  display: flex;
  align-items: baseline;
  gap: 0.25rem;
  color: var(--text-main);
}

.stat-unit {
  font-size: 0.85rem;
  font-weight: 400;
  color: var(--text-dim);
}

.text-accent { color: #0284c7; }
.text-cyan { color: #0891b2; }
.text-emerald { color: #059669; }
.text-muted { color: var(--text-dim); }

.glass-panel {
  background: var(--bg-glass-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg, 16px);
  padding: 1.25rem;
  backdrop-filter: var(--blur-card);
  box-shadow: var(--shadow-sm);
}

.section-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-main);
  margin: 0;
}

.table-container {
  overflow-x: auto;
  margin-top: 1rem;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
  font-size: 0.875rem;
}

.data-table th {
  color: var(--text-muted);
  padding: 0.6rem 1rem;
  border-bottom: 1px solid var(--border-card);
  font-weight: 600;
}

.data-table td {
  padding: 0.6rem 1rem;
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-main);
}

.timestamp-col {
  color: var(--text-dim);
  font-size: 0.8rem;
  white-space: nowrap;
}

.value-col {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.unit-badge {
  color: var(--text-dim);
  font-size: 0.8rem;
}
</style>
