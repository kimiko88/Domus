<script setup lang="ts">
import { computed } from 'vue'
import type { Entity } from '../../api/types'
import {
  Thermometer,
  Droplets,
  Gauge,
  Activity,
  Battery,
  ShieldAlert,
  Radio
} from '@lucide/vue'

interface Props {
  entity: Entity
}

const props = defineProps<Props>()

const isBinary = computed(() => props.entity.domain === 'binary_sensor')
const isTriggered = computed(() => isBinary.value && props.entity.state === 'on')

const deviceClass = computed(() => props.entity.attributes?.device_class || '')

const iconComponent = computed(() => {
  const dc = deviceClass.value.toLowerCase()
  if (dc === 'temperature' || dc.includes('temp')) return Thermometer
  if (dc === 'humidity' || dc.includes('moisture')) return Droplets
  if (dc === 'pressure') return Gauge
  if (dc === 'motion' || dc === 'occupancy') return Activity
  if (dc === 'battery') return Battery
  if (dc === 'smoke' || dc === 'gas' || dc === 'problem') return ShieldAlert
  return Radio
})

const displayValue = computed(() => {
  if (isBinary.value) {
    if (deviceClass.value === 'motion') {
      return isTriggered.value ? 'Motion' : 'Clear'
    }
    if (deviceClass.value === 'door' || deviceClass.value === 'window') {
      return isTriggered.value ? 'Open' : 'Closed'
    }
    return isTriggered.value ? 'Active' : 'Idle'
  }
  return props.entity.state
})

const unit = computed(() => props.entity.attributes?.unit_of_measurement || '')
const battery = computed(() => {
  if (typeof props.entity.attributes?.battery === 'number') {
    return props.entity.attributes.battery
  }
  return null
})
</script>

<template>
  <div :class="['sensor-card glass-card', { 'is-triggered': isTriggered }]">
    <div class="card-top">
      <div class="icon-wrapper" :class="{ 'triggered-icon': isTriggered }">
        <component :is="iconComponent" :size="22" />
      </div>
      <div class="info">
        <h4 class="entity-name" :title="entity.name">{{ entity.name }}</h4>
        <div v-if="battery !== null" class="battery-badge">
          <Battery :size="12" />
          <span>{{ battery }}%</span>
        </div>
      </div>
    </div>

    <div class="sensor-reading">
      <span class="value">{{ displayValue }}</span>
      <span v-if="unit" class="unit">{{ unit }}</span>
    </div>
  </div>
</template>

<style scoped>
.sensor-card {
  padding: 16px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
  transition: all var(--transition-normal);
  min-height: 110px;
}

.sensor-card.is-triggered {
  border-color: rgba(244, 63, 94, 0.4);
  box-shadow: 0 4px 20px rgba(244, 63, 94, 0.15);
  background: linear-gradient(145deg, rgba(244, 63, 94, 0.08) 0%, rgba(30, 41, 59, 0.7) 100%);
}

.card-top {
  display: flex;
  align-items: center;
  gap: 12px;
}

.icon-wrapper {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-dim);
  transition: all var(--transition-normal);
  flex-shrink: 0;
}

.triggered-icon {
  background: rgba(244, 63, 94, 0.18);
  color: var(--color-danger);
  box-shadow: 0 0 16px rgba(244, 63, 94, 0.4);
}

.info {
  flex: 1;
  min-width: 0;
}

.entity-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.battery-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-dim);
  margin-top: 2px;
}

.sensor-reading {
  display: flex;
  align-items: baseline;
  gap: 4px;
  margin-top: 10px;
}

.value {
  font-size: 22px;
  font-weight: 700;
  font-family: var(--font-mono);
  color: var(--text-main);
  letter-spacing: -0.02em;
}

.unit {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-muted);
}
</style>
