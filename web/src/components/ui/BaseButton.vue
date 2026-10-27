<script setup lang="ts">
interface Props {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost'
  size?: 'sm' | 'md' | 'lg'
  disabled?: boolean
  loading?: boolean
  type?: 'button' | 'submit' | 'reset'
}

withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  disabled: false,
  loading: false,
  type: 'button'
})

const emit = defineEmits<{
  (e: 'click', event: MouseEvent): void
}>()

function handleClick(event: MouseEvent) {
  emit('click', event)
}
</script>

<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    :aria-busy="loading ? 'true' : undefined"
    :class="[
      'btn',
      `btn-${variant}`,
      `btn-${size}`,
      { 'btn-loading': loading }
    ]"
    @click="handleClick"
  >
    <span v-if="loading" class="btn-spinner" aria-hidden="true" />
    <span v-if="loading" class="sr-only">Caricamento in corso...</span>
    <span :class="['btn-content', { 'opacity-0': loading }]">
      <slot />
    </span>
  </button>
</template>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  position: relative;
  font-weight: 500;
  border-radius: var(--radius-md);
  transition: all var(--transition-fast);
  cursor: pointer;
  white-space: nowrap;
  user-select: none;
  border: 1px solid transparent;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  pointer-events: none;
}

/* Sizes */
.btn-sm {
  padding: 6px 12px;
  font-size: 13px;
  gap: 6px;
  position: relative;
}

.btn-sm::after {
  content: '';
  position: absolute;
  inset: -6px 0;
  min-height: 44px;
}

.btn-md {
  padding: 10px 18px;
  font-size: 14px;
  gap: 8px;
}

.btn-lg {
  padding: 12px 24px;
  font-size: 16px;
  gap: 10px;
}

/* Variants */
.btn-primary {
  background: linear-gradient(135deg, var(--primary) 0%, #0284c7 100%);
  color: white;
  box-shadow: 0 2px 8px rgba(14, 165, 233, 0.35);
}

.btn-primary:hover:not(:disabled) {
  background: linear-gradient(135deg, var(--primary-hover) 0%, var(--primary) 100%);
  box-shadow: 0 4px 14px rgba(14, 165, 233, 0.5);
  transform: translateY(-1px);
}

.btn-secondary {
  background: var(--bg-surface);
  color: var(--text-main);
  border-color: var(--border-card);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--bg-surface-hover);
  border-color: rgba(255, 255, 255, 0.2);
  transform: translateY(-1px);
}

.btn-danger {
  background: linear-gradient(135deg, var(--color-danger) 0%, #e11d48 100%);
  color: white;
  box-shadow: 0 2px 8px rgba(244, 63, 94, 0.35);
}

.btn-danger:hover:not(:disabled) {
  box-shadow: 0 4px 14px rgba(244, 63, 94, 0.5);
  transform: translateY(-1px);
}

.btn-ghost {
  background: transparent;
  color: var(--text-muted);
}

.btn-ghost:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-main);
}

/* Spinner */
.btn-spinner {
  position: absolute;
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.2);
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.btn-content {
  display: inline-flex;
  align-items: center;
  gap: inherit;
}

.opacity-0 {
  opacity: 0;
}
</style>
