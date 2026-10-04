<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import Chart from 'chart.js/auto'
import type { StateHistory } from '@/api/types'

const props = withDefaults(
  defineProps<{
    history: StateHistory[]
    unit?: string
    color?: string
    label?: string
    loading?: boolean
  }>(),
  {
    unit: '',
    color: '#06b6d4',
    label: '',
    loading: false
  }
)

const { t } = useI18n()
const canvasRef = ref<HTMLCanvasElement | null>(null)
let chartInstance: Chart | null = null

function parseNumericValue(val: any): number | null {
  if (typeof val === 'number') return isNaN(val) ? null : val
  if (typeof val === 'boolean') return val ? 1 : 0
  if (typeof val === 'string') {
    const trimmed = val.trim().toLowerCase()
    if (trimmed === 'on' || trimmed === 'true') return 1
    if (trimmed === 'off' || trimmed === 'false') return 0
    const parsedNum = parseFloat(trimmed)
    if (!isNaN(parsedNum)) return parsedNum

    try {
      const obj = JSON.parse(val)
      if (typeof obj === 'number') return obj
      if (typeof obj === 'object' && obj !== null) {
        for (const k of ['value', 'temp', 'temperature', 'humidity', 'power', 'battery']) {
          if (typeof obj[k] === 'number') return obj[k]
        }
      }
    } catch {
      // not json
    }
  }
  return null
}

const processedPoints = computed(() => {
  if (!props.history || props.history.length === 0) return []

  return props.history
    .map(entry => {
      const val = parseNumericValue(entry.state)
      const date = new Date(entry.recorded_at)
      return {
        timestamp: date,
        timeLabel: date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        fullDateLabel: date.toLocaleString(),
        value: val
      }
    })
    .filter((pt): pt is { timestamp: Date; timeLabel: string; fullDateLabel: string; value: number } => pt.value !== null)
})

function renderChart() {
  if (!canvasRef.value) return
  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }

  const points = processedPoints.value
  if (points.length === 0) return

  const ctx = canvasRef.value.getContext('2d')
  if (!ctx) return

  const gradient = ctx.createLinearGradient(0, 0, 0, 240)
  gradient.addColorStop(0, 'rgba(6, 182, 212, 0.35)')
  gradient.addColorStop(1, 'rgba(6, 182, 212, 0.0)')

  const labels = points.map(p => p.timeLabel)
  const data = points.map(p => p.value)

  chartInstance = new Chart(ctx, {
    type: 'line',
    data: {
      labels,
      datasets: [
        {
          label: props.label || t('history.value'),
          data,
          borderColor: props.color || '#06b6d4',
          borderWidth: 2.5,
          backgroundColor: gradient,
          fill: true,
          tension: 0.35,
          pointRadius: points.length < 30 ? 3 : 0,
          pointHoverRadius: 6,
          pointBackgroundColor: props.color || '#06b6d4'
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: {
        intersect: false,
        mode: 'index'
      },
      plugins: {
        legend: {
          display: false
        },
        tooltip: {
          backgroundColor: 'rgba(15, 23, 42, 0.9)',
          borderColor: 'rgba(255, 255, 255, 0.1)',
          borderWidth: 1,
          padding: 10,
          titleColor: '#94a3b8',
          bodyColor: '#f8fafc',
          callbacks: {
            title: items => {
              if (items.length > 0) {
                const idx = items[0].dataIndex
                return points[idx]?.fullDateLabel || items[0].label
              }
              return ''
            },
            label: context => {
              const val = context.parsed.y
              return ` ${val} ${props.unit}`
            }
          }
        }
      },
      scales: {
        x: {
          grid: {
            color: 'rgba(148, 163, 184, 0.12)'
          },
          ticks: {
            color: '#64748b',
            maxRotation: 0,
            autoSkip: true,
            maxTicksLimit: 8
          }
        },
        y: {
          grid: {
            color: 'rgba(148, 163, 184, 0.12)'
          },
          ticks: {
            color: '#64748b',
            callback: value => `${value} ${props.unit}`
          }
        }
      }
    }
  })
}

watch(
  () => props.history,
  () => {
    setTimeout(renderChart, 0)
  },
  { deep: true }
)

onMounted(() => {
  renderChart()
})

onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }
})

defineExpose({
  processedPoints
})
</script>

<template>
  <div class="sensor-history-chart">
    <div v-if="loading" class="chart-loading">
      <div class="spinner"></div>
      <span>{{ t('common.loading') }}</span>
    </div>

    <div v-else-if="processedPoints.length === 0" class="empty-state">
      <svg class="empty-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M13 10V3L4 14h7v7l9-11h-7z" />
      </svg>
      <p class="empty-text">{{ t('history.noData') }}</p>
    </div>

    <div v-else class="canvas-wrapper">
      <canvas ref="canvasRef"></canvas>
    </div>
  </div>
</template>

<style scoped>
.sensor-history-chart {
  position: relative;
  width: 100%;
  height: 280px;
  background: var(--bg-glass-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg, 16px);
  padding: 1rem;
  backdrop-filter: var(--blur-glass);
  box-shadow: var(--shadow-sm);
}

.canvas-wrapper {
  position: relative;
  width: 100%;
  height: 100%;
}

.chart-loading,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 0.75rem;
  color: var(--text-dim);
}

.spinner {
  width: 28px;
  height: 28px;
  border: 3px solid rgba(148, 163, 184, 0.2);
  border-top-color: #06b6d4;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.empty-icon {
  width: 36px;
  height: 36px;
  opacity: 0.6;
}

.empty-text {
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--text-muted);
  margin: 0;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
