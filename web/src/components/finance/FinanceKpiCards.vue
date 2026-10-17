<script setup lang="ts">
import { computed } from 'vue'
import {
  TrendingUp,
  TrendingDown,
  Wallet,
  PiggyBank,
  Calendar,
  Layers,
  AlertTriangle
} from '@lucide/vue'
import type { FinancialSummary } from '@/api/types'

const props = defineProps<{
  summary: FinancialSummary | null
  period: 'day' | 'week' | 'month' | 'year'
}>()

const periodLabels = {
  day: 'oggi',
  week: 'questa settimana',
  month: 'questo mese',
  year: 'quest\'anno'
}

const totalSpent = computed(() => props.summary?.total_spent ?? 0)
const previousTotal = computed(() => props.summary?.previous_period_total ?? 0)
const percentageChange = computed(() => props.summary?.percentage_change ?? 0)
const dailyAverage = computed(() => props.summary?.daily_average ?? 0)
const monthlyBudget = computed(() => props.summary?.monthly_budget ?? 0)
const remainingBudget = computed(() => props.summary?.remaining_budget ?? 0)

const budgetUsagePercent = computed(() => {
  if (!monthlyBudget.value || monthlyBudget.value <= 0) return 0
  return Math.min(Math.round((totalSpent.value / monthlyBudget.value) * 100), 100)
})

const isOverBudget = computed(() => {
  return monthlyBudget.value > 0 && totalSpent.value > monthlyBudget.value
})

const fixedVsVar = computed(() => {
  return (
    props.summary?.fixed_vs_variable ?? {
      fixed_amount: 0,
      variable_amount: 0,
      fixed_percentage: 0,
      variable_percentage: 0
    }
  )
})

function formatCurrency(val: number): string {
  return new Intl.NumberFormat('it-IT', {
    style: 'currency',
    currency: 'EUR',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(val)
}
</script>

<template>
  <div class="finance-kpis-grid" role="region" aria-label="Indicatori Finanziari Principali">
    <!-- 1. Totale Speso -->
    <div class="kpi-card glass-panel">
      <div class="kpi-header">
        <span class="kpi-title">Totale Speso</span>
        <div class="kpi-icon-badge primary">
          <Wallet :size="20" aria-hidden="true" />
        </div>
      </div>
      <div class="kpi-main-val">
        {{ formatCurrency(totalSpent) }}
      </div>
      <div class="kpi-footer">
        <span
          v-if="previousTotal > 0"
          :class="['kpi-trend-badge', percentageChange <= 0 ? 'trend-positive' : 'trend-negative']"
          :title="percentageChange <= 0 ? 'Spesa ridotta rispetto al periodo precedente' : 'Spesa aumentata rispetto al periodo precedente'"
        >
          <component :is="percentageChange <= 0 ? TrendingDown : TrendingUp" :size="14" aria-hidden="true" />
          <span>{{ Math.abs(percentageChange).toFixed(1) }}%</span>
        </span>
        <span class="kpi-subtitle">vs periodo precedente</span>
      </div>
    </div>

    <!-- 2. Budget Mensile & Residuo -->
    <div class="kpi-card glass-panel" :class="{ 'card-alert': isOverBudget }">
      <div class="kpi-header">
        <span class="kpi-title">Budget Mensile</span>
        <div class="kpi-icon-badge" :class="isOverBudget ? 'danger' : 'success'">
          <PiggyBank :size="20" aria-hidden="true" />
        </div>
      </div>
      <div class="kpi-main-val" :class="{ 'val-over-budget': isOverBudget }">
        {{ formatCurrency(remainingBudget) }}
      </div>
      <div class="kpi-footer budget-footer">
        <div class="budget-bar-track" role="progressbar" :aria-valuenow="budgetUsagePercent" aria-valuemin="0" aria-valuemax="100">
          <div
            class="budget-bar-fill"
            :class="{
              'fill-success': budgetUsagePercent < 75,
              'fill-warning': budgetUsagePercent >= 75 && budgetUsagePercent < 100,
              'fill-danger': isOverBudget
            }"
            :style="{ width: `${budgetUsagePercent}%` }"
          />
        </div>
        <div class="budget-details">
          <span>{{ budgetUsagePercent }}% speso</span>
          <span>Max: {{ formatCurrency(monthlyBudget) }}</span>
        </div>
      </div>
    </div>

    <!-- 3. Media Giornaliera / Settimanale -->
    <div class="kpi-card glass-panel">
      <div class="kpi-header">
        <span class="kpi-title">Media Giornaliera</span>
        <div class="kpi-icon-badge info">
          <Calendar :size="20" aria-hidden="true" />
        </div>
      </div>
      <div class="kpi-main-val">
        {{ formatCurrency(dailyAverage) }}
      </div>
      <div class="kpi-footer">
        <span class="kpi-subtitle">Ritmo di spesa medio calcolato per {{ periodLabels[period] }}</span>
      </div>
    </div>

    <!-- 4. Fisse vs Variabili -->
    <div class="kpi-card glass-panel">
      <div class="kpi-header">
        <span class="kpi-title">Fisse vs Variabili</span>
        <div class="kpi-icon-badge warning">
          <Layers :size="20" aria-hidden="true" />
        </div>
      </div>
      <div class="kpi-main-val split-val">
        <span class="fixed-num" :title="`Fisse: ${formatCurrency(fixedVsVar.fixed_amount)}`">
          {{ Math.round(fixedVsVar.fixed_percentage) }}% Fisse
        </span>
        <span class="split-separator">/</span>
        <span class="var-num" :title="`Variabili: ${formatCurrency(fixedVsVar.variable_amount)}`">
          {{ Math.round(fixedVsVar.variable_percentage) }}% Var
        </span>
      </div>
      <div class="kpi-footer">
        <div class="split-ratio-track" aria-hidden="true">
          <div class="split-fill-fixed" :style="{ width: `${fixedVsVar.fixed_percentage}%` }" />
          <div class="split-fill-var" :style="{ width: `${fixedVsVar.variable_percentage}%` }" />
        </div>
        <div class="split-labels">
          <span>Incomprimibili: {{ formatCurrency(fixedVsVar.fixed_amount) }}</span>
          <span>Flessibili: {{ formatCurrency(fixedVsVar.variable_amount) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.finance-kpis-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.kpi-card {
  padding: 1.25rem;
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  min-height: 140px;
  position: relative;
  transition: transform var(--transition-fast), border-color var(--transition-fast);
}

.kpi-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.18);
}

.card-alert {
  border-color: var(--color-danger);
  box-shadow: 0 0 15px var(--color-danger-glow);
}

.kpi-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.kpi-title {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-muted);
}

.kpi-icon-badge {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
}

.kpi-icon-badge.primary {
  background: var(--primary-light);
  color: var(--primary);
}

.kpi-icon-badge.success {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.kpi-icon-badge.warning {
  background: var(--color-warning-bg);
  color: var(--color-warning);
}

.kpi-icon-badge.danger {
  background: var(--color-danger-bg);
  color: var(--color-danger);
}

.kpi-icon-badge.info {
  background: var(--color-info-bg);
  color: var(--color-info);
}

.kpi-main-val {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0.5rem 0;
  letter-spacing: -0.02em;
}

.val-over-budget {
  color: var(--color-danger);
}

.kpi-main-val.split-val {
  font-size: 1.15rem;
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.fixed-num {
  color: var(--color-warning);
}

.split-separator {
  color: var(--text-dim);
}

.var-num {
  color: var(--color-success);
}

.kpi-footer {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  color: var(--text-dim);
}

.kpi-trend-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.2rem 0.45rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  font-weight: 600;
}

.trend-positive {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.trend-negative {
  background: var(--color-danger-bg);
  color: var(--color-danger);
}

.budget-footer {
  flex-direction: column;
  align-items: stretch;
  gap: 0.35rem;
}

.budget-bar-track {
  width: 100%;
  height: 6px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.budget-bar-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.4s ease;
}

.fill-success {
  background: var(--color-success);
}

.fill-warning {
  background: var(--color-warning);
}

.fill-danger {
  background: var(--color-danger);
}

.budget-details {
  display: flex;
  justify-content: space-between;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.split-ratio-track {
  width: 100%;
  height: 6px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-full);
  display: flex;
  overflow: hidden;
  margin-bottom: 0.35rem;
}

.split-fill-fixed {
  background: var(--color-warning);
  height: 100%;
}

.split-fill-var {
  background: var(--color-success);
  height: 100%;
}

.split-labels {
  display: flex;
  justify-content: space-between;
  font-size: 0.75rem;
  color: var(--text-muted);
  width: 100%;
}
</style>
