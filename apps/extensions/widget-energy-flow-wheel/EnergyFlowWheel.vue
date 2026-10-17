<script setup lang="ts">
import { computed } from 'vue'
import { Sun, BatteryCharging, Zap, Home } from '@lucide/vue'

interface Props {
  solarWatts?: number
  gridWatts?: number
  batteryWatts?: number
  batterySoc?: number
  homeWatts?: number
}

const props = withDefaults(defineProps<Props>(), {
  solarWatts: 4200,
  gridWatts: -150, // negative means export to grid
  batteryWatts: 1800, // positive means charging
  batterySoc: 78,
  homeWatts: 2250
})

const selfConsumptionPercent = computed(() => {
  if (props.solarWatts <= 0) return 0
  const consumed = Math.max(0, props.solarWatts - Math.max(0, -props.gridWatts))
  return Math.min(100, Math.round((consumed / props.solarWatts) * 100))
})
</script>

<template>
  <div class="energy-flow-widget glass-card">
    <div class="widget-header">
      <div class="title-row">
        <Zap :size="18" class="text-warning" />
        <h4 class="widget-title">Power Flow Wheel</h4>
      </div>
      <span class="badge-pill">Autoconsumo: {{ selfConsumptionPercent }}%</span>
    </div>

    <!-- Animated Circular SVG Flow Diagram -->
    <div class="flow-svg-wrapper">
      <svg viewBox="0 0 280 280" class="flow-wheel-svg" role="img" aria-label="Flusso Potenza">
        <!-- Center Home Hub Node -->
        <circle cx="140" cy="140" r="32" class="hub-circle" />

        <!-- Nodes: Top=Solar, Right=Grid, Bottom=Battery, Left=Home -->
        <!-- Solar Node (Top) -->
        <g class="node-group solar" transform="translate(140, 42)">
          <circle r="26" class="node-circle" />
        </g>
        <!-- Grid Node (Right) -->
        <g class="node-group grid" transform="translate(238, 140)">
          <circle r="26" class="node-circle" />
        </g>
        <!-- Battery Node (Bottom) -->
        <g class="node-group battery" transform="translate(140, 238)">
          <circle r="26" class="node-circle" />
        </g>
      </svg>

      <!-- Center Icon Overlay -->
      <div class="center-metrics">
        <Home :size="20" class="center-icon" />
        <span class="metric-val">{{ homeWatts }} W</span>
        <span class="metric-sub">Consumo Casa</span>
      </div>

      <!-- Solar Badge -->
      <div class="node-overlay top">
        <Sun :size="16" class="text-warning" />
        <span class="val">{{ solarWatts }} W</span>
      </div>

      <!-- Grid Badge -->
      <div class="node-overlay right">
        <Zap :size="16" :class="gridWatts >= 0 ? 'text-danger' : 'text-success'" />
        <span class="val">{{ Math.abs(gridWatts) }} W</span>
        <span class="dir-sub">{{ gridWatts >= 0 ? 'Prelievo' : 'Immissione' }}</span>
      </div>

      <!-- Battery Badge -->
      <div class="node-overlay bottom">
        <BatteryCharging :size="16" class="text-primary" />
        <span class="val">{{ batterySoc }}% ({{ batteryWatts }} W)</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.energy-flow-widget {
  padding: 18px;
  border-radius: 16px;
  background: var(--surface-card, rgba(30, 41, 59, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.widget-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.widget-title {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 700;
  color: #fff;
}

.badge-pill {
  font-size: 0.75rem;
  padding: 2px 8px;
  border-radius: 6px;
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  font-weight: 600;
}

.flow-svg-wrapper {
  position: relative;
  display: flex;
  justify-content: center;
  align-items: center;
  height: 260px;
}

.flow-wheel-svg {
  width: 100%;
  max-width: 260px;
  height: auto;
}

.hub-circle {
  fill: rgba(255, 255, 255, 0.05);
  stroke: var(--primary, #0ea5e9);
  stroke-width: 2;
}

.node-circle {
  fill: #1e293b;
  stroke: rgba(255, 255, 255, 0.2);
  stroke-width: 2;
}

.center-metrics {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.metric-val {
  font-size: 0.9rem;
  font-weight: 800;
  color: #fff;
}

.metric-sub {
  font-size: 0.65rem;
  color: var(--text-muted, #94a3b8);
}

.node-overlay {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  font-size: 0.72rem;
  font-weight: 700;
  color: #fff;
}

.node-overlay.top { top: 8px; }
.node-overlay.right { right: 8px; top: 110px; }
.node-overlay.bottom { bottom: 8px; }

.dir-sub {
  font-size: 0.62rem;
  font-weight: 500;
  color: var(--text-muted, #94a3b8);
}
</style>
