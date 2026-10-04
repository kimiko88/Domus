<script setup lang="ts">
import { computed } from 'vue'
import type { Entity } from '../../api/types'
import { useEntityCommand } from '../../composables/useEntityCommand'
import { Thermometer, Plus, Minus, Flame, Snowflake, RefreshCw, Power } from '@lucide/vue'

interface Props {
  entity: Entity
}

const props = defineProps<Props>()
const { setTemperature, isExecuting } = useEntityCommand()

const currentTemp = computed(() => {
  const t = props.entity.attributes?.current_temperature ?? props.entity.attributes?.current_temp
  return typeof t === 'number' ? t.toFixed(1) : '--'
})

const targetTemp = computed(() => {
  const t = props.entity.attributes?.temperature ?? props.entity.attributes?.target_temperature
  return typeof t === 'number' ? t : 20.0
})

const unit = computed(() => props.entity.attributes?.unit_of_measurement || '°C')
const hvacMode = computed(() => props.entity.state || 'off')

const modeIcon = computed(() => {
  switch (hvacMode.value) {
    case 'heat':
      return Flame
    case 'cool':
      return Snowflake
    case 'auto':
      return RefreshCw
    default:
      return Power
  }
})

async function adjustTemp(delta: number) {
  const newTemp = Math.round((targetTemp.value + delta) * 10) / 10
  await setTemperature(props.entity, newTemp)
}
</script>

<template>
  <div :class="['climate-card glass-card', `mode-${hvacMode}`]">
    <div class="card-top">
      <div class="icon-wrapper">
        <component :is="modeIcon" :size="22" />
      </div>
      <div class="info">
        <h4 class="entity-name" :title="entity.name">{{ entity.name }}</h4>
        <div class="mode-badge">
          <span class="mode-text">{{ hvacMode }}</span>
          <span class="current-label">Current: {{ currentTemp }}{{ unit }}</span>
        </div>
      </div>
    </div>

    <div class="card-body">
      <div class="target-section">
        <button
          class="btn-temp btn-temp-minus"
          :disabled="isExecuting"
          aria-label="Decrease temperature"
          @click="adjustTemp(-0.5)"
        >
          <Minus :size="18" />
        </button>

        <div class="target-display">
          <span class="target-value">{{ targetTemp.toFixed(1) }}</span>
          <span class="target-unit">{{ unit }}</span>
        </div>

        <button
          class="btn-temp btn-temp-plus"
          :disabled="isExecuting"
          aria-label="Increase temperature"
          @click="adjustTemp(0.5)"
        >
          <Plus :size="18" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.climate-card {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
  transition: all var(--transition-normal);
}

.climate-card.mode-heat {
  border-color: rgba(249, 115, 22, 0.4);
  box-shadow: 0 4px 20px rgba(249, 115, 22, 0.12);
}

.climate-card.mode-cool {
  border-color: rgba(14, 165, 233, 0.4);
  box-shadow: 0 4px 20px rgba(14, 165, 233, 0.12);
}

.card-top {
  display: flex;
  align-items: center;
  gap: 14px;
}

.icon-wrapper {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(249, 115, 22, 0.15);
  color: #f97316;
  flex-shrink: 0;
}

.mode-cool .icon-wrapper {
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
}

.info {
  flex: 1;
  min-width: 0;
}

.entity-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mode-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 2px;
  font-size: 13px;
  color: var(--text-muted);
}

.mode-text {
  text-transform: capitalize;
  font-weight: 500;
}

.current-label {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-dim);
}

/* Setpoint Control Row */
.target-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: rgba(0, 0, 0, 0.2);
  padding: 8px 12px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
}

.target-display {
  display: flex;
  align-items: baseline;
  gap: 3px;
}

.target-value {
  font-size: 26px;
  font-weight: 700;
  font-family: var(--font-mono);
  color: var(--text-main);
}

.target-unit {
  font-size: 14px;
  color: var(--text-muted);
}

.btn-temp {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-main);
  cursor: pointer;
  border: 1px solid var(--border-subtle);
  transition: all var(--transition-fast);
}

.btn-temp:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.16);
  border-color: rgba(255, 255, 255, 0.3);
  transform: scale(1.05);
}

.btn-temp:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
</style>
