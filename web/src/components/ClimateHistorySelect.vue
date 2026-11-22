<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

export interface ClimateHistorySelectOption {
  value: string | number
  label: string
}

const props = defineProps<{
  id: string
  label: string
  modelValue: string | number
  options: ClimateHistorySelectOption[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

const root = ref<HTMLElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)
const open = ref(false)
const highlightedIndex = ref(0)
const listboxId = computed(() => `${props.id}-options`)
const activeOptionId = computed(() => `${listboxId.value}-${highlightedIndex.value}`)
const selectedOption = computed(() =>
  props.options.find(option => option.value === props.modelValue) ?? props.options[0]
)

watch(() => props.modelValue, value => {
  const index = props.options.findIndex(option => option.value === value)
  if (index >= 0) highlightedIndex.value = index
})

function showMenu() {
  if (!props.options.length) return
  const selectedIndex = props.options.findIndex(option => option.value === props.modelValue)
  highlightedIndex.value = selectedIndex >= 0 ? selectedIndex : 0
  open.value = true
}

function closeMenu() {
  open.value = false
}

function chooseOption(option: ClimateHistorySelectOption) {
  emit('update:modelValue', option.value)
  closeMenu()
  nextTick(() => trigger.value?.focus())
}

function moveHighlight(delta: number) {
  if (!props.options.length) return
  const count = props.options.length
  highlightedIndex.value = (highlightedIndex.value + delta + count) % count
}

function onTriggerKeydown(event: KeyboardEvent) {
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      if (!open.value) showMenu()
      else moveHighlight(1)
      break
    case 'ArrowUp':
      event.preventDefault()
      if (!open.value) showMenu()
      else moveHighlight(-1)
      break
    case 'Home':
      if (open.value) {
        event.preventDefault()
        highlightedIndex.value = 0
      }
      break
    case 'End':
      if (open.value) {
        event.preventDefault()
        highlightedIndex.value = Math.max(0, props.options.length - 1)
      }
      break
    case 'Enter':
    case ' ':
      event.preventDefault()
      if (!open.value) showMenu()
      else if (props.options[highlightedIndex.value]) chooseOption(props.options[highlightedIndex.value])
      break
    case 'Escape':
      if (open.value) {
        event.preventDefault()
        closeMenu()
      }
      break
    case 'Tab':
      closeMenu()
      break
  }
}

function onDocumentPointerDown(event: PointerEvent) {
  if (root.value && !root.value.contains(event.target as Node)) closeMenu()
}

onMounted(() => document.addEventListener('pointerdown', onDocumentPointerDown))
onBeforeUnmount(() => document.removeEventListener('pointerdown', onDocumentPointerDown))
</script>

<template>
  <div ref="root" class="climate-history-select">
    <button
      :id="id"
      ref="trigger"
      type="button"
      class="climate-history-select__trigger"
      role="combobox"
      aria-haspopup="listbox"
      :aria-label="label"
      :aria-expanded="open"
      :aria-controls="listboxId"
      :aria-activedescendant="open ? activeOptionId : undefined"
      @click="open ? closeMenu() : showMenu()"
      @keydown="onTriggerKeydown"
    >
      <span class="climate-history-select__value">{{ selectedOption?.label ?? '' }}</span>
      <svg class="climate-history-select__chevron" :class="{ 'is-open': open }" viewBox="0 0 16 16" aria-hidden="true">
        <path d="m3.5 6 4.5 4 4.5-4" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    </button>

    <div
      v-if="open"
      :id="listboxId"
      class="climate-history-select__menu"
      role="listbox"
      :aria-label="label"
    >
      <div
        v-for="(option, index) in options"
        :id="`${listboxId}-${index}`"
        :key="String(option.value)"
        class="climate-history-select__option"
        :class="{
          'is-highlighted': highlightedIndex === index,
          'is-selected': modelValue === option.value
        }"
        role="option"
        :aria-selected="modelValue === option.value"
        @mouseenter="highlightedIndex = index"
        @mousedown.prevent
        @click="chooseOption(option)"
      >
        <span>{{ option.label }}</span>
        <svg v-if="modelValue === option.value" viewBox="0 0 16 16" aria-hidden="true">
          <path d="m3 8 3.2 3.2L13 4.5" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </div>
    </div>
  </div>
</template>

<style scoped>
.climate-history-select {
  position: relative;
  width: 100%;
  min-width: 0;
  color: var(--text-main, #f8fafc);
  color-scheme: inherit;
}
.climate-history-select__trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: .65rem;
  width: 100%;
  min-height: 2.45rem;
  padding: .45rem .65rem;
  border: 1px solid var(--border-card, rgba(148,163,184,.3));
  border-radius: .55rem;
  background: var(--bg-surface, #1e293b);
  color: var(--text-main, #f8fafc);
  text-align: left;
  cursor: pointer;
}
.climate-history-select__trigger:focus-visible {
  outline: 2px solid var(--primary, #0ea5e9);
  outline-offset: 2px;
}
.climate-history-select__value {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.climate-history-select__chevron {
  width: 1rem;
  height: 1rem;
  flex: none;
  transition: transform .15s ease;
}
.climate-history-select__chevron.is-open { transform: rotate(180deg); }
.climate-history-select__menu {
  position: absolute;
  z-index: 1200;
  top: calc(100% + .3rem);
  left: 0;
  right: 0;
  max-height: min(18rem, 55vh);
  overflow-y: auto;
  padding: .3rem;
  border: 1px solid var(--border-card, rgba(148,163,184,.3));
  border-radius: .65rem;
  background: var(--bg-surface, #1e293b);
  color: var(--text-main, #f8fafc);
  box-shadow: var(--shadow-lg, 0 12px 25px rgba(0,0,0,.25));
}
.climate-history-select__option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: .6rem;
  min-height: 2.35rem;
  padding: .45rem .6rem;
  border-radius: .4rem;
  color: var(--text-main, #f8fafc);
  cursor: pointer;
  overflow-wrap: anywhere;
}
.climate-history-select__option svg {
  width: 1rem;
  height: 1rem;
  flex: none;
  color: var(--primary, #0ea5e9);
}
.climate-history-select__option.is-highlighted,
.climate-history-select__option:hover {
  background: var(--bg-surface-hover, rgba(148,163,184,.16));
}
.climate-history-select__option.is-selected { font-weight: 650; }
:global(:root[data-theme="light"]) .climate-history-select__trigger,
:global(:root[data-theme="light"]) .climate-history-select__menu {
  background: #fff;
  color: #0f172a;
}
:global(:root[data-theme="light"]) .climate-history-select__option {
  color: #0f172a;
}
:global(:root[data-theme="light"]) .climate-history-select__option.is-highlighted,
:global(:root[data-theme="light"]) .climate-history-select__option:hover {
  background: #eaf2fb;
}
@media (prefers-reduced-motion: reduce) {
  .climate-history-select__chevron { transition: none; }
}
</style>
