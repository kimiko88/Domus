<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Entity } from '../../api/types'
import { useEntityCommand } from '../../composables/useEntityCommand'
import { Thermometer, Plus, Minus, Flame, Snowflake, RefreshCw, Power } from '@lucide/vue'
const { t } = useI18n()

interface Props {
  entity: Entity
}

const props = defineProps<Props>()
const { setTemperature, isExecuting } = useEntityCommand()

function numericTemperature(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) return parsed
  }
  return undefined
}

const currentTemp = computed(() => {
  const value = numericTemperature(
    props.entity.attributes?.current_temperature ?? props.entity.attributes?.current_temp
  )
  return value === undefined ? '--' : value.toFixed(1)
})

const targetTemp = computed(() => {
  const value = numericTemperature(
    props.entity.attributes?.target_temperature ?? props.entity.attributes?.temperature
  )
  return value ?? 20.0
})

const targetTemperatureDraft = ref(targetTemp.value.toFixed(1))
const targetTemperatureValid = computed(() => {
  const value = Number(targetTemperatureDraft.value)
  return Number.isFinite(value) && value >= 5 && value <= 35
})

watch(targetTemp, value => {
  targetTemperatureDraft.value = value.toFixed(1)
})

const unit = computed(() => props.entity.attributes?.unit_of_measurement || '°C')
const hvacMode = computed(() =>
  String(props.entity.attributes?.hvac_mode || props.entity.state || 'off').toLowerCase()
)

const modeIcon = computed(() => {
  switch (hvacMode.value) {
    case 'heat':
    case 'heating':
    case 'manual':
      return Flame
    case 'cool':
    case 'cooling':
      return Snowflake
    case 'auto':
    case 'automatic':
      return RefreshCw
    default:
      return Power
  }
})

async function adjustTemp(delta: number) {
  const newTemp = Math.round((targetTemp.value + delta) * 10) / 10
  targetTemperatureDraft.value = newTemp.toFixed(1)
  await setTemperature(props.entity, newTemp)
}

async function applyExactTemperature() {
  if (!targetTemperatureValid.value) return
  const value = Math.round(Number(targetTemperatureDraft.value) * 10) / 10
  await setTemperature(props.entity, value)
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
          <span class="current-label">Attuale: {{ currentTemp }}{{ unit }}</span>
        </div>
      </div>
    </div>

    <div class="card-body">
      <div class="target-section">
        <button
          class="btn-temp btn-temp-minus"
          :disabled="isExecuting"
          :aria-label="t('common.entityCard.decreaseTemperature')"
          @click="adjustTemp(-0.5)"
        >
          <Minus :size="18" />
        </button>

        <div class="target-display">
          <span class="target-label">Obiettivo</span>
          <div class="target-value-row">
            <span class="target-value">{{ targetTemp.toFixed(1) }}</span>
            <span class="target-unit">{{ unit }}</span>
          </div>
          <div class="exact-temperature-control">
            <input
              v-model="targetTemperatureDraft"
              type="number"
              min="5"
              max="35"
              step="0.1"
              inputmode="decimal"
              aria-label="Temperatura obiettivo esatta"
              :aria-invalid="!targetTemperatureValid"
              @keydown.enter.prevent="applyExactTemperature"
            />
            <button
              type="button"
              class="btn-temp-apply"
              :disabled="isExecuting || !targetTemperatureValid"
              aria-label="Applica temperatura obiettivo"
              @click="applyExactTemperature"
            >
              Imposta
            </button>
          </div>
        </div>

        <button
          class="btn-temp btn-temp-plus"
          :disabled="isExecuting"
          :aria-label="t('common.entityCard.increaseTemperature')"
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
  flex-direction: column;
  align-items: center;
  gap: 4px;
  flex: 1;
}

.target-label {
  color: var(--text-muted);
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.target-value-row {
  display: flex;
  align-items: baseline;
  gap: 3px;
}

.exact-temperature-control {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  max-width: 180px;
}

.exact-temperature-control input {
  min-width: 0;
  width: 86px;
  min-height: 44px;
  padding: 7px 8px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-main);
  font: 600 14px var(--font-mono);
  text-align: center;
}

.exact-temperature-control input[aria-invalid="true"] {
  border-color: var(--color-danger);
}

.btn-temp-apply {
  min-height: 44px;
  padding: 7px 9px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: rgba(14, 165, 233, 0.14);
  color: var(--primary);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.btn-temp-apply:disabled {
  opacity: 0.4;
  cursor: not-allowed;
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
  width: 44px;
  height: 44px;
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
