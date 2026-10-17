<script setup lang="ts">
import { computed } from 'vue'
import { Package, AlertTriangle, Clock, ChefHat } from '@lucide/vue'

interface Props {
  totalItems: number
  lowStockCount: number
  expiringCount: number
  cookableCount: number
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'filter-low-stock'): void
  (e: 'filter-expiring'): void
  (e: 'go-recipes-cookable'): void
}>()
</script>

<template>
  <div class="pantry-kpi-grid">
    <!-- Total Items -->
    <div class="kpi-card glass-panel" tabindex="0">
      <div class="kpi-icon-wrap bg-primary-soft">
        <Package :size="24" class="kpi-icon text-primary" />
      </div>
      <div class="kpi-info">
        <span class="kpi-label">Articoli in Dispensa</span>
        <div class="kpi-val">{{ props.totalItems }}</div>
        <span class="kpi-sub">Scorte monitorate</span>
      </div>
    </div>

    <!-- Low Stock -->
    <div
      :class="['kpi-card glass-panel clickable', { 'has-alert': props.lowStockCount > 0 }]"
      role="button"
      tabindex="0"
      aria-label="Filtra articoli scorte basse"
      @click="emit('filter-low-stock')"
      @keydown.enter="emit('filter-low-stock')"
      @keydown.space.prevent="emit('filter-low-stock')"
    >
      <div class="kpi-icon-wrap bg-warning-soft">
        <AlertTriangle :size="24" class="kpi-icon text-warning" />
      </div>
      <div class="kpi-info">
        <span class="kpi-label">Scorte in Esaurimento</span>
        <div class="kpi-val text-warning">{{ props.lowStockCount }}</div>
        <span class="kpi-sub">Sotto soglia minima</span>
      </div>
    </div>

    <!-- Expiring Soon -->
    <div
      :class="['kpi-card glass-panel clickable', { 'has-alert': props.expiringCount > 0 }]"
      role="button"
      tabindex="0"
      aria-label="Filtra articoli in scadenza"
      @click="emit('filter-expiring')"
      @keydown.enter="emit('filter-expiring')"
      @keydown.space.prevent="emit('filter-expiring')"
    >
      <div class="kpi-icon-wrap bg-danger-soft">
        <Clock :size="24" class="kpi-icon text-danger" />
      </div>
      <div class="kpi-info">
        <span class="kpi-label">In Scadenza (3 gg)</span>
        <div class="kpi-val text-danger">{{ props.expiringCount }}</div>
        <span class="kpi-sub">Da consumare subito</span>
      </div>
    </div>

    <!-- Cookable Recipes -->
    <div
      class="kpi-card glass-panel clickable"
      role="button"
      tabindex="0"
      aria-label="Mostra ricette cucinabili oggi"
      @click="emit('go-recipes-cookable')"
      @keydown.enter="emit('go-recipes-cookable')"
      @keydown.space.prevent="emit('go-recipes-cookable')"
    >
      <div class="kpi-icon-wrap bg-success-soft">
        <ChefHat :size="24" class="kpi-icon text-success" />
      </div>
      <div class="kpi-info">
        <span class="kpi-label">Cosa Posso Cucinare?</span>
        <div class="kpi-val text-success">{{ props.cookableCount }}</div>
        <span class="kpi-sub">100% ingredienti presenti</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.pantry-kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--space-4, 16px);
  margin-bottom: var(--space-6, 24px);
}

.kpi-card {
  display: flex;
  align-items: center;
  gap: var(--space-4, 16px);
  padding: var(--space-4, 16px) var(--space-5, 20px);
  border-radius: var(--radius-lg, 16px);
  background: var(--surface-card, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  backdrop-filter: blur(12px);
  transition: transform 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
}

.kpi-card.clickable {
  cursor: pointer;
}

.kpi-card.clickable:hover {
  transform: translateY(-2px);
  border-color: var(--primary, #3b82f6);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
}

.kpi-card:focus-visible {
  outline: 2px solid var(--primary, #3b82f6);
  outline-offset: 2px;
}

.kpi-icon-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md, 12px);
  flex-shrink: 0;
}

.bg-primary-soft {
  background: rgba(59, 130, 246, 0.15);
}
.bg-warning-soft {
  background: rgba(245, 158, 11, 0.15);
}
.bg-danger-soft {
  background: rgba(239, 68, 68, 0.15);
}
.bg-success-soft {
  background: rgba(16, 185, 129, 0.15);
}

.text-primary {
  color: var(--primary, #3b82f6);
}
.text-warning {
  color: var(--warning, #f59e0b);
}
.text-danger {
  color: var(--danger, #ef4444);
}
.text-success {
  color: var(--success, #10b981);
}

.kpi-info {
  display: flex;
  flex-direction: column;
}

.kpi-label {
  font-size: 0.82rem;
  color: var(--text-muted, #94a3b8);
  font-weight: 500;
}

.kpi-val {
  font-size: 1.5rem;
  font-weight: 700;
  line-height: 1.2;
  margin: 2px 0;
  color: var(--text-main, #f8fafc);
}

.kpi-sub {
  font-size: 0.75rem;
  color: var(--text-muted, #64748b);
}

@media (max-width: 640px) {
  .pantry-kpi-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: var(--space-3, 12px);
  }
  .kpi-card {
    padding: var(--space-3, 12px);
    gap: var(--space-3, 12px);
  }
  .kpi-icon-wrap {
    width: 40px;
    height: 40px;
  }
  .kpi-val {
    font-size: 1.25rem;
  }
}
</style>
