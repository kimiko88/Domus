<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Entity } from '../../api/types'
import { useEntityCommand } from '../../composables/useEntityCommand'
import { Lightbulb } from '@lucide/vue'

interface Props {
  entity: Entity
}

const props = defineProps<Props>()
const { toggle, setBrightness, isExecuting } = useEntityCommand()

const isOn = computed(() => props.entity.state === 'on')

// Brightness from 0 to 255 mapped to 0-100%
const brightnessPercent = computed(() => {
  if (typeof props.entity.attributes?.brightness === 'number') {
    return Math.round((props.entity.attributes.brightness / 255) * 100)
  }
  return isOn.value ? 100 : 0
})

const sliderValue = ref(brightnessPercent.value)

watch(
  brightnessPercent,
  (val) => {
    sliderValue.value = val
  },
  { immediate: true }
)

async function handleToggle() {
  await toggle(props.entity)
}

async function handleBrightnessChange() {
  const scaled = Math.round((sliderValue.value / 100) * 255)
  await setBrightness(props.entity, scaled)
}
</script>

<template>
  <div :class="['light-card glass-card', { 'is-on': isOn }]">
    <div class="card-top">
      <div class="icon-wrapper" :class="{ 'glow-icon': isOn }">
        <Lightbulb :size="24" />
      </div>
      <div class="info">
        <h4 class="entity-name" :title="entity.name">{{ entity.name }}</h4>
        <div class="entity-status">
          <span class="status-badge">{{ isOn ? 'On' : 'Off' }}</span>
          <span v-if="isOn" class="brightness-text">{{ brightnessPercent }}%</span>
        </div>
      </div>
      <button
        class="toggle-btn"
        :class="{ active: isOn }"
        :disabled="isExecuting"
        :aria-label="isOn ? 'Turn off' : 'Turn on'"
        @click="handleToggle"
      >
        <span class="toggle-indicator" />
      </button>
    </div>

    <!-- Brightness Slider -->
    <div v-if="isOn" class="slider-row">
      <input
        v-model.number="sliderValue"
        type="range"
        min="1"
        max="100"
        class="brightness-slider"
        aria-label="Brightness"
        @change="handleBrightnessChange"
      />
    </div>
  </div>
</template>

<style scoped>
.light-card {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  transition: all var(--transition-normal);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
}

.light-card.is-on {
  border-color: rgba(251, 191, 36, 0.4);
  box-shadow: 0 4px 20px rgba(251, 191, 36, 0.15);
  background: linear-gradient(145deg, rgba(251, 191, 36, 0.06) 0%, rgba(30, 41, 59, 0.7) 100%);
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
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-dim);
  transition: all var(--transition-normal);
  flex-shrink: 0;
}

.glow-icon {
  background: rgba(251, 191, 36, 0.18);
  color: #fbbf24;
  box-shadow: 0 0 16px rgba(251, 191, 36, 0.4);
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

.entity-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 2px;
  font-size: 13px;
  color: var(--text-muted);
}

.status-badge {
  text-transform: capitalize;
}

.brightness-text {
  font-family: var(--font-mono);
  font-size: 12px;
  color: #fbbf24;
}

/* Toggle Switch Button */
.toggle-btn {
  width: 48px;
  height: 28px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.1);
  position: relative;
  cursor: pointer;
  border: 1px solid var(--border-card);
  transition: all var(--transition-fast);
  flex-shrink: 0;
  padding: 2px;
}

.toggle-indicator {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: white;
  transition: all var(--transition-fast);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.4);
}

.toggle-btn.active {
  background: linear-gradient(135deg, #fbbf24 0%, #f59e0b 100%);
  border-color: #f59e0b;
}

.toggle-btn.active .toggle-indicator {
  transform: translateX(20px);
}

/* Brightness Slider */
.slider-row {
  display: flex;
  align-items: center;
  padding-top: 4px;
}

.brightness-slider {
  width: 100%;
  height: 6px;
  -webkit-appearance: none;
  appearance: none;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 999px;
  outline: none;
  cursor: pointer;
}

.brightness-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: #fbbf24;
  box-shadow: 0 0 8px rgba(251, 191, 36, 0.6);
  cursor: pointer;
  transition: transform var(--transition-fast);
}

.brightness-slider::-webkit-slider-thumb:hover {
  transform: scale(1.2);
}
</style>
