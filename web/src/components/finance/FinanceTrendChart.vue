<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import Chart from 'chart.js/auto'
import type { MonthlyTrend } from '@/api/types'

const props = defineProps<{
  trends: MonthlyTrend[]
}>()

const canvasRef = ref<HTMLCanvasElement | null>(null)
let chartInstance: Chart | null = null

function formatCurrency(val: number): string {
  return new Intl.NumberFormat('it-IT', {
    style: 'currency',
    currency: 'EUR',
    maximumFractionDigits: 0
  }).format(val)
}

function initChart() {
  if (!canvasRef.value) return

  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }

  if (!props.trends || props.trends.length === 0) return

  const labels = props.trends.map((t) => t.month_name)
  const utilitiesData = props.trends.map((t) => t.utilities_amount)
  const foodData = props.trends.map((t) => t.food_amount)
  const transportData = props.trends.map((t) => t.transport_amount)
  const otherData = props.trends.map(
    (t) => Math.max(0, t.total_amount - (t.utilities_amount + t.food_amount + t.transport_amount))
  )

  const ctx = canvasRef.value.getContext('2d')
  if (!ctx) return

  chartInstance = new Chart(ctx, {
    type: 'bar',
    data: {
      labels,
      datasets: [
        {
          label: '💡 Bollette & Utenze',
          data: utilitiesData,
          backgroundColor: '#0ea5e9',
          borderRadius: 4
        },
        {
          label: '🛒 Cibo & Spesa',
          data: foodData,
          backgroundColor: '#10b981',
          borderRadius: 4
        },
        {
          label: '⛽ Carburante & Auto',
          data: transportData,
          backgroundColor: '#f59e0b',
          borderRadius: 4
        },
        {
          label: '🏷️ Altro & Fisse',
          data: otherData,
          backgroundColor: '#64748b',
          borderRadius: 4
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        x: {
          stacked: true,
          grid: {
            color: 'rgba(255, 255, 255, 0.04)'
          },
          ticks: {
            color: '#94a3b8',
            font: {
              size: 11
            }
          }
        },
        y: {
          stacked: true,
          grid: {
            color: 'rgba(255, 255, 255, 0.05)'
          },
          ticks: {
            color: '#94a3b8',
            callback(val) {
              return `€${val}`
            }
          }
        }
      },
      plugins: {
        legend: {
          position: 'top',
          labels: {
            color: '#f8fafc',
            boxWidth: 12,
            boxHeight: 12,
            usePointStyle: true,
            font: {
              size: 11
            }
          }
        },
        tooltip: {
          backgroundColor: 'rgba(15, 23, 42, 0.95)',
          borderColor: 'rgba(255, 255, 255, 0.1)',
          borderWidth: 1,
          padding: 10,
          titleColor: '#f8fafc',
          bodyColor: '#94a3b8',
          callbacks: {
            footer(items) {
              let sum = 0
              items.forEach((item) => {
                sum += item.parsed.y as number
              })
              return `Totale Mese: ${formatCurrency(sum)}`
            }
          }
        }
      }
    }
  })
}

watch(
  () => props.trends,
  () => {
    initChart()
  },
  { deep: true }
)

onMounted(() => {
  initChart()
})

onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }
})
</script>

<template>
  <div class="trend-chart-card glass-panel" role="region" aria-label="Andamento Stagionale delle Spese negli Ultimi 12 Mesi">
    <div class="card-header">
      <div>
        <h3 class="card-title">Andamento Spese Ultimi 12 Mesi</h3>
        <span class="card-subtitle">Stagionalità bollette e consumi (inverno vs estate)</span>
      </div>
    </div>

    <div v-if="trends.length === 0" class="no-data-placeholder">
      <span>Nessun dato storico disponibile.</span>
    </div>

    <div v-else class="canvas-container">
      <canvas ref="canvasRef" aria-label="Istogramma andamento mensile 12 mesi" role="img" />
    </div>
  </div>
</template>

<style scoped>
.trend-chart-card {
  padding: 1.25rem;
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
}

.card-header {
  margin-bottom: 1rem;
}

.card-title {
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--text-main);
  margin: 0;
}

.card-subtitle {
  font-size: 0.8125rem;
  color: var(--text-dim);
}

.no-data-placeholder {
  padding: 3rem;
  text-align: center;
  color: var(--text-dim);
  font-size: 0.9rem;
}

.canvas-container {
  position: relative;
  width: 100%;
  height: 260px;
}
</style>
