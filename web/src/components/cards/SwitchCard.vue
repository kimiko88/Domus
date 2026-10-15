<script setup lang="ts">
import { computed } from 'vue'
import type { Entity } from '../../api/types'
import { useEntityCommand } from '../../composables/useEntityCommand'
import { Power, Zap } from '@lucide/vue'

interface Props {
  entity: Entity
}

const props = defineProps<Props>()
const { toggle, isExecuting } = useEntityCommand()

const isOn = computed(() => props.entity.state === 'on')

const power = computed(() => {
  if (typeof props.entity.attributes?.power === 'number') {
    return `${props.entity.attributes.power} W`
  }
  return null
})

async function handleToggle() {
  await toggle(props.entity)
}
</script>

<template>
  <div :class="['switch-card glass-card', { 'is-on': isOn }]">
    <div class="card-top">
      <div class="icon-wrapper" :class="{ 'glow-icon': isOn }">
        <Power :size="22" />
      </div>
      <div class="info">
        <h4 class="entity-name" :title="entity.name">{{ entity.name }}</h4>
        <div class="entity-status">
          <span class="status-badge">{{ isOn ? 'On' : 'Off' }}</span>
          <span v-if="power" class="power-badge">
            <Zap :size="12" class="zap-icon" />
            {{ power }}
          </span>
        </div>
      </div>
      <button
        class="toggle-btn"
        :class="{ active: isOn }"
        :disabled="isExecuting"
        role="switch"
        :aria-checked="isOn"
        :aria-label="isOn ? `Disattiva ${entity.name}` : `Attiva ${entity.name}`"
        @click="handleToggle"
      >
        <span class="toggle-indicator" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.switch-card {
  padding: 16px;
  display: flex;
  flex-direction: column;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
  transition: all var(--transition-normal);
}

.switch-card.is-on {
  border-color: rgba(14, 165, 233, 0.4);
  box-shadow: 0 4px 20px rgba(14, 165, 233, 0.15);
  background: linear-gradient(145deg, rgba(14, 165, 233, 0.07) 0%, rgba(30, 41, 59, 0.7) 100%);
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
  background: rgba(14, 165, 233, 0.18);
  color: var(--primary);
  box-shadow: 0 0 16px rgba(14, 165, 233, 0.4);
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

.power-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-family: var(--font-mono);
  font-size: 11px;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  background: rgba(14, 165, 233, 0.12);
  color: var(--primary);
}

.zap-icon {
  color: var(--primary);
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
  background: linear-gradient(135deg, var(--primary) 0%, #0284c7 100%);
  border-color: var(--primary);
}

.toggle-btn.active .toggle-indicator {
  transform: translateX(20px);
}
</style>
