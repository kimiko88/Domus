<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  Zap,
  Trash2,
  ShieldAlert,
  ArrowRight,
  Sparkles,
  CheckCircle2,
  AlertTriangle,
  Info
} from '@lucide/vue'
import type { SavingsInsight } from '@/api/types'

const props = defineProps<{
  insights: SavingsInsight[]
}>()

const router = useRouter()

const totalPotentialSavings = computed(() => {
  return props.insights.reduce((acc, ins) => acc + (ins.potential_saving ?? 0), 0)
})

function formatCurrency(val: number): string {
  return new Intl.NumberFormat('it-IT', {
    style: 'currency',
    currency: 'EUR',
    minimumFractionDigits: 2
  }).format(val)
}

function getInsightIcon(type: string) {
  switch (type) {
    case 'energy_tariff':
      return Zap
    case 'food_waste':
      return Trash2
    case 'contract_expiry':
      return ShieldAlert
    default:
      return Sparkles
  }
}

function handleAction(url?: string) {
  if (url) {
    router.push(url)
  }
}
</script>

<template>
  <div class="savings-insights-section" role="region" aria-label="Suggerimenti Intelligenti di Risparmio">
    <div class="section-header">
      <div class="title-with-badge">
        <Sparkles :size="20" class="sparkle-icon" aria-hidden="true" />
        <h3 class="section-title">Suggerimenti Intelligenti di Risparmio</h3>
        <span class="domus-badge">La Chicca di Domus</span>
      </div>
      <div v-if="totalPotentialSavings > 0" class="total-savings-badge">
        <span>Risparmio potenziale stimato: <strong>{{ formatCurrency(totalPotentialSavings) }}</strong></span>
      </div>
    </div>

    <div v-if="insights.length === 0" class="no-insights-box glass-panel">
      <CheckCircle2 :size="24" class="no-insights-icon" aria-hidden="true" />
      <div>
        <p class="no-insights-title">Ottimo lavoro! Nessuna spesa anomala o spreco rilevato.</p>
        <p class="no-insights-sub">I consumi energetici, le scadenze e la gestione delle scorte sono perfettamente ottimizzati.</p>
      </div>
    </div>

    <div v-else class="insights-grid">
      <div
        v-for="ins in insights"
        :key="ins.id"
        class="insight-card glass-panel"
        :class="`severity-${ins.severity}`"
      >
        <div class="insight-header">
          <div class="insight-type-icon" :class="`icon-${ins.severity}`">
            <component :is="getInsightIcon(ins.type)" :size="18" aria-hidden="true" />
          </div>
          <div class="insight-meta">
            <h4 class="insight-title">{{ ins.title }}</h4>
            <span v-if="ins.potential_saving" class="saving-chip">
              Risparmio: +{{ formatCurrency(ins.potential_saving) }}
            </span>
          </div>
        </div>

        <p class="insight-message">{{ ins.message }}</p>

        <div v-if="ins.action_url" class="insight-actions">
          <button
            class="action-btn"
            :aria-label="`Vai all'azione per: ${ins.title}`"
            @click="handleAction(ins.action_url)"
          >
            <span>Risolvi & Ottimizza</span>
            <ArrowRight :size="15" aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.savings-insights-section {
  margin-bottom: 2rem;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.title-with-badge {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.sparkle-icon {
  color: var(--accent-light);
}

.section-title {
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--text-main);
  margin: 0;
}

.domus-badge {
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.2rem 0.6rem;
  border-radius: var(--radius-full);
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.2), rgba(168, 85, 247, 0.2));
  color: #38bdf8;
  border: 1px solid rgba(56, 189, 248, 0.3);
}

.total-savings-badge {
  font-size: 0.8125rem;
  color: var(--color-success);
  background: var(--color-success-bg);
  padding: 0.35rem 0.75rem;
  border-radius: var(--radius-full);
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.no-insights-box {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.25rem;
  border-radius: var(--radius-lg);
  border: 1px solid rgba(16, 185, 129, 0.2);
}

.no-insights-icon {
  color: var(--color-success);
  flex-shrink: 0;
}

.no-insights-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-main);
  margin: 0 0 0.25rem 0;
}

.no-insights-sub {
  font-size: 0.8125rem;
  color: var(--text-dim);
  margin: 0;
}

.insights-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 1rem;
}

.insight-card {
  padding: 1.25rem;
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  transition: transform var(--transition-fast), border-color var(--transition-fast);
}

.insight-card:hover {
  transform: translateY(-2px);
}

.severity-warning {
  border-color: rgba(245, 158, 11, 0.3);
}

.severity-alert {
  border-color: rgba(244, 63, 94, 0.35);
}

.severity-info {
  border-color: rgba(14, 165, 233, 0.3);
}

.insight-header {
  display: flex;
  align-items: flex-start;
  gap: 0.85rem;
  margin-bottom: 0.75rem;
}

.insight-type-icon {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.icon-warning {
  background: var(--color-warning-bg);
  color: var(--color-warning);
}

.icon-alert {
  background: var(--color-danger-bg);
  color: var(--color-danger);
}

.icon-info {
  background: var(--primary-light);
  color: var(--primary);
}

.insight-meta {
  flex: 1;
}

.insight-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-main);
  margin: 0 0 0.3rem 0;
}

.saving-chip {
  display: inline-block;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-success);
  background: var(--color-success-bg);
  padding: 0.15rem 0.5rem;
  border-radius: var(--radius-sm);
}

.insight-message {
  font-size: 0.85rem;
  color: var(--text-muted);
  line-height: 1.45;
  margin: 0 0 1rem 0;
  flex-grow: 1;
}

.insight-actions {
  display: flex;
  justify-content: flex-end;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.45rem 0.9rem;
  min-height: 38px;
  border-radius: var(--radius-md);
  font-size: 0.8125rem;
  font-weight: 600;
  background: var(--primary-light);
  color: var(--primary);
  border: 1px solid rgba(14, 165, 233, 0.25);
  cursor: pointer;
  transition: background var(--transition-fast), color var(--transition-fast);
}

.action-btn:hover {
  background: var(--primary);
  color: #fff;
}
</style>
