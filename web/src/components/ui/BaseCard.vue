<script setup lang="ts">
interface Props {
  active?: boolean
  clickable?: boolean
}

withDefaults(defineProps<Props>(), {
  active: false,
  clickable: false
})

const emit = defineEmits<{
  (e: 'click', event: MouseEvent): void
}>()

function handleClick(event: MouseEvent) {
  emit('click', event)
}
</script>

<template>
  <div
    :class="[
      'base-card glass-card',
      { 'is-active': active, 'is-clickable': clickable }
    ]"
    @click="handleClick"
  >
    <div v-if="$slots.header" class="card-header">
      <slot name="header" />
    </div>

    <div class="card-body">
      <slot />
    </div>

    <div v-if="$slots.footer" class="card-footer">
      <slot name="footer" />
    </div>
  </div>
</template>

<style scoped>
.base-card {
  position: relative;
  display: flex;
  flex-direction: column;
  border-radius: var(--radius-lg);
  padding: 18px;
  overflow: hidden;
  transition: all var(--transition-normal);
}

.is-clickable {
  cursor: pointer;
}

.is-clickable:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
}

.is-active {
  border-color: var(--primary);
  box-shadow: 0 0 20px var(--primary-glow);
}

.card-header {
  margin-bottom: 12px;
}

.card-body {
  flex: 1;
}

.card-footer {
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}
</style>
