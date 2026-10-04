<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useEntitiesStore } from '../stores/entities'
import { useAreasStore } from '../stores/areas'
import LightCard from '../components/cards/LightCard.vue'
import SwitchCard from '../components/cards/SwitchCard.vue'
import SensorCard from '../components/cards/SensorCard.vue'
import ClimateCard from '../components/cards/ClimateCard.vue'
import { Lightbulb, Power, Radio, Thermometer, Sparkles } from '@lucide/vue'
import type { Entity } from '../api/types'

const entitiesStore = useEntitiesStore()
const areasStore = useAreasStore()

const activeDomainFilter = ref<string>('all')

onMounted(async () => {
  try {
    await Promise.all([
      entitiesStore.fetchEntities(),
      areasStore.fetchAreas()
    ])
  } catch (err) {
    console.error('Error fetching dashboard initial data:', err)
  }
})

// Metrics
const activeLightsCount = computed(() => {
  return entitiesStore.lights.filter((l) => l.state === 'on').length
})

const activeSwitchesCount = computed(() => {
  return entitiesStore.switches.filter((s) => s.state === 'on').length
})

const totalSensorsCount = computed(() => {
  return entitiesStore.sensors.length + entitiesStore.binarySensors.length
})

// Filtered Entities
const filteredEntities = computed(() => {
  const all = entitiesStore.allEntities
  if (activeDomainFilter.value === 'all') return all
  if (activeDomainFilter.value === 'light') return entitiesStore.lights
  if (activeDomainFilter.value === 'switch') return entitiesStore.switches
  if (activeDomainFilter.value === 'sensor') return [...entitiesStore.sensors, ...entitiesStore.binarySensors]
  if (activeDomainFilter.value === 'climate') return entitiesStore.byDomain('climate')
  return all
})

const filterOptions = [
  { id: 'all', label: 'Tutte' },
  { id: 'light', label: 'Luci', icon: Lightbulb },
  { id: 'switch', label: 'Interruttori', icon: Power },
  { id: 'sensor', label: 'Sensori', icon: Radio },
  { id: 'climate', label: 'Clima', icon: Thermometer }
]
</script>

<template>
  <div class="dashboard-view">
    <!-- Quick Summary Cards -->
    <div class="summary-grid">
      <div class="summary-card glass-card">
        <div class="summary-icon icon-light">
          <Lightbulb :size="20" />
        </div>
        <div class="summary-info">
          <span class="summary-label">Luci Accese</span>
          <span class="summary-value">{{ activeLightsCount }} Attive</span>
        </div>
      </div>

      <div class="summary-card glass-card">
        <div class="summary-icon icon-switch">
          <Power :size="20" />
        </div>
        <div class="summary-info">
          <span class="summary-label">Prese Attive</span>
          <span class="summary-value">{{ activeSwitchesCount }} Attive</span>
        </div>
      </div>

      <div class="summary-card glass-card">
        <div class="summary-icon icon-sensor">
          <Radio :size="20" />
        </div>
        <div class="summary-info">
          <span class="summary-label">Sensori Online</span>
          <span class="summary-value">{{ totalSensorsCount }} Rilevati</span>
        </div>
      </div>
    </div>

    <!-- Filter Pills Row -->
    <div class="filters-row">
      <button
        v-for="pill in filterOptions"
        :key="pill.id"
        :class="['filter-pill', { active: activeDomainFilter === pill.id }]"
        @click="activeDomainFilter = pill.id"
      >
        <component :is="pill.icon" v-if="pill.icon" :size="16" class="pill-icon" />
        <span>{{ pill.label }}</span>
      </button>
    </div>

    <!-- Entity Grid -->
    <div v-if="filteredEntities.length > 0" class="entity-grid">
      <template v-for="entity in filteredEntities" :key="entity.id">
        <LightCard v-if="entity.domain === 'light'" :entity="entity" />
        <SwitchCard v-else-if="entity.domain === 'switch'" :entity="entity" />
        <SensorCard v-else-if="entity.domain === 'sensor' || entity.domain === 'binary_sensor'" :entity="entity" />
        <ClimateCard v-else-if="entity.domain === 'climate'" :entity="entity" />
        <SwitchCard v-else :entity="entity" />
      </template>
    </div>

    <!-- Empty State -->
    <div v-else class="empty-state glass-card">
      <Sparkles :size="36" class="empty-icon" />
      <h3 class="empty-title">Nessuna entità trovata</h3>
      <p class="empty-desc">
        Non ci sono dispositivi configurati per questo filtro. Collega un'integrazione o aggiungi dispositivi per vederli apparire qui.
      </p>
    </div>
  </div>
</template>

<style scoped>
.dashboard-view {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

/* Summary Grid */
.summary-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
}

.summary-card {
  padding: 16px 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  border-radius: var(--radius-lg);
}

.summary-icon {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.icon-light {
  background: rgba(251, 191, 36, 0.15);
  color: #fbbf24;
}

.icon-switch {
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
}

.icon-sensor {
  background: rgba(16, 185, 129, 0.15);
  color: var(--color-success);
}

.summary-info {
  display: flex;
  flex-direction: column;
}

.summary-label {
  font-size: 13px;
  color: var(--text-muted);
}

.summary-value {
  font-size: 17px;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.01em;
}

/* Filter Pills */
.filters-row {
  display: flex;
  align-items: center;
  gap: 10px;
  overflow-x: auto;
  padding-bottom: 4px;
}

.filter-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: var(--radius-full);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-muted);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: all var(--transition-fast);
}

.filter-pill:hover {
  background: var(--bg-surface-hover);
  color: var(--text-main);
}

.filter-pill.active {
  background: var(--primary);
  color: white;
  border-color: var(--primary);
  box-shadow: 0 0 12px var(--primary-glow);
}

.pill-icon {
  flex-shrink: 0;
}

/* Entity Grid */
.entity-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

/* Empty State */
.empty-state {
  padding: 48px 24px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  max-width: 500px;
  margin: 30px auto;
  border-radius: var(--radius-xl);
}

.empty-icon {
  color: var(--primary);
  margin-bottom: 16px;
  opacity: 0.8;
}

.empty-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-main);
}

.empty-desc {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 8px;
  line-height: 1.5;
}
</style>
