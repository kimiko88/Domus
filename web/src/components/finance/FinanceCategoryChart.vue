<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import Chart from 'chart.js/auto'
import type { CategoryBreakdown, ExpenseCategory } from '@/api/types'

const props = defineProps<{
  categories: CategoryBreakdown[]
  selectedCategory: ExpenseCategory | 'all'
}>()

const emit = defineEmits<{
  (e: 'select-category', cat: ExpenseCategory | 'all'): void
}>()

const canvasRef = ref<HTMLCanvasElement | null>(null)
let chartInstance: Chart | null = null

const categoryColors: Record<string, string> = {
  food: '#10b981',
  utilities: '#0ea5e9',
  transport: '#f59e0b',
  home_maintenance: '#8b5cf6',
  insurance: '#f43f5e',
  other: '#64748b'
}

const totalSum = computed(() => {
  return props.categories.reduce((acc, c) => acc + c.total_amount, 0)
})

function formatCurrency(val: number): string {
  return new Intl.NumberFormat('it-IT', {
    style: 'currency',
    currency: 'EUR',
    minimumFractionDigits: 2
  }).format(val)
}

function initChart() {
  if (!canvasRef.value) return

  // Destroy previous instance
  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }

  if (props.categories.length === 0) return

  const labels = props.categories.map((c) => c.name)
  const data = props.categories.map((c) => c.total_amount)
  const bgColors = props.categories.map((c) => categoryColors[c.category] || '#64748b')

  const ctx = canvasRef.value.getContext('2d')
  if (!ctx) return

  chartInstance = new Chart(ctx, {
    type: 'doughnut',
    data: {
      labels,
      datasets: [
        {
          data,
          backgroundColor: bgColors,
          borderWidth: 2,
          borderColor: '#1e293b',
          hoverOffset: 6
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      cutout: '70%',
      plugins: {
        legend: {
          display: false
        },
        tooltip: {
          backgroundColor: 'rgba(15, 23, 42, 0.9)',
          borderColor: 'rgba(255, 255, 255, 0.1)',
          borderWidth: 1,
          padding: 10,
          titleColor: '#f8fafc',
          bodyColor: '#94a3b8',
          callbacks: {
            label(ctx) {
              const val = ctx.parsed as number
              const pct = totalSum.value > 0 ? ((val / totalSum.value) * 100).toFixed(1) : '0'
              return ` ${formatCurrency(val)} (${pct}%)`
            }
          }
        }
      }
    }
  })
}

watch(
  () => props.categories,
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
  <div class="category-breakdown-card glass-panel" role="region" aria-label="Ripartizione Spese per Categoria">
    <div class="card-header">
      <h3 class="card-title">Ripartizione Categorie (Ciambella)</h3>
      <button
        v-if="selectedCategory !== 'all'"
        class="reset-filter-btn"
        @click="emit('select-category', 'all')"
      >
        Azzera Filtro
      </button>
    </div>

    <div v-if="categories.length === 0" class="no-data-placeholder">
      <span>Nessuna spesa registrata per questo periodo.</span>
    </div>

    <div v-else class="chart-content">
      <div class="canvas-wrapper">
        <canvas ref="canvasRef" aria-label="Grafico a ciambella spese per categoria" role="img" />
        <div class="canvas-center-stat" aria-hidden="true">
          <span class="center-label">Totale</span>
          <span class="center-val">{{ formatCurrency(totalSum) }}</span>
        </div>
      </div>

      <div class="category-legend" role="list">
        <button
          v-for="cat in categories"
          :key="cat.category"
          class="legend-item"
          :class="{ 'is-selected': selectedCategory === cat.category }"
          :aria-pressed="selectedCategory === cat.category"
          @click="emit('select-category', selectedCategory === cat.category ? 'all' : cat.category)"
        >
          <div class="legend-left">
            <span
              class="legend-color-dot"
              :style="{ backgroundColor: categoryColors[cat.category] || '#64748b' }"
            />
            <span class="legend-name">{{ cat.name }}</span>
          </div>
          <div class="legend-right">
            <span class="legend-amount">{{ formatCurrency(cat.total_amount) }}</span>
            <span class="legend-percent">{{ cat.percentage.toFixed(1) }}%</span>
          </div>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.category-breakdown-card {
  padding: 1.25rem;
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.card-title {
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--text-main);
  margin: 0;
}

.reset-filter-btn {
  font-size: 0.75rem;
  color: var(--primary);
  background: var(--primary-light);
  border: none;
  padding: 0.25rem 0.6rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: opacity var(--transition-fast);
}

.reset-filter-btn:hover {
  opacity: 0.85;
}

.no-data-placeholder {
  padding: 2.5rem;
  text-align: center;
  color: var(--text-dim);
  font-size: 0.9rem;
}

.chart-content {
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 1.5rem;
  align-items: center;
}

@media (max-width: 640px) {
  .chart-content {
    grid-template-columns: 1fr;
  }
}

.canvas-wrapper {
  position: relative;
  width: 200px;
  height: 200px;
  margin: 0 auto;
}

.canvas-center-stat {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  pointer-events: none;
}

.center-label {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.center-val {
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-main);
}

.category-legend {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.legend-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 0.75rem;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  color: var(--text-main);
  cursor: pointer;
  transition: background var(--transition-fast), border-color var(--transition-fast);
  text-align: left;
}

.legend-item:hover {
  background: rgba(255, 255, 255, 0.07);
}

.legend-item.is-selected {
  background: var(--primary-light);
  border-color: var(--primary);
}

.legend-left {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.legend-color-dot {
  width: 10px;
  height: 10px;
  border-radius: var(--radius-full);
}

.legend-name {
  font-size: 0.85rem;
  font-weight: 500;
}

.legend-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.legend-amount {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-main);
}

.legend-percent {
  font-size: 0.75rem;
  color: var(--text-muted);
  min-width: 42px;
  text-align: right;
}
</style>
