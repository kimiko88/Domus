<script setup lang="ts">
import { computed } from 'vue'
import type { AutomationCondition, Entity } from '../api/types'

const props = withDefaults(defineProps<{
  modelValue: AutomationCondition
  entities: Entity[]
  depth?: number
}>(), {
  depth: 0
})

const emit = defineEmits<{
  (event: 'update:modelValue', value: AutomationCondition): void
}>()

const logicalTypes = [
  { value: 'and', label: 'Tutte le condizioni (AND)' },
  { value: 'or', label: 'Almeno una condizione (OR)' },
  { value: 'not', label: 'Nessuna condizione (NOT)' }
]
const leafTypes = [
  { value: 'state', label: 'Stato entità' },
  { value: 'numeric_state', label: 'Soglia numerica' },
  { value: 'time', label: 'Fascia oraria' },
  { value: 'trigger', label: 'Trigger' }
]
const supportedTypes = new Set([...logicalTypes.map((item) => item.value), ...leafTypes.map((item) => item.value)])
const isLogical = computed(() => logicalTypes.some((item) => item.value === props.modelValue.condition))
const isSupported = computed(() => supportedTypes.has(props.modelValue.condition))
const children = computed(() => Array.isArray(props.modelValue.conditions) ? props.modelValue.conditions : [])
const canAddChild = computed(() => props.depth < 8 && (props.modelValue.condition !== 'not' || children.value.length === 0))

function emitValue(value: AutomationCondition) {
  emit('update:modelValue', value)
}

function updateField(field: string, value: unknown) {
  emitValue({ ...props.modelValue, [field]: value })
}

function createDefaultCondition(): AutomationCondition {
  return {
    condition: 'state',
    entity_key: props.entities[0]?.id || '',
    state: 'on'
  }
}

function removeDiscriminatorFields(condition: AutomationCondition): AutomationCondition {
  const next = { ...condition } as Record<string, unknown>
  for (const field of ['condition', 'conditions', 'entity_key', 'state', 'above', 'below', 'after', 'before', 'trigger_id']) {
    delete next[field]
  }
  return next as AutomationCondition
}

function changeType(type: string) {
  if (logicalTypes.some((item) => item.value === type)) {
    if (isLogical.value) {
      let nextChildren = [...children.value]
      if (type === 'not' && nextChildren.length > 1) {
        // Keep every existing predicate by grouping them under a single AND child.
        nextChildren = [{ condition: 'and', conditions: nextChildren }]
      } else if (nextChildren.length === 0) {
        nextChildren = [createDefaultCondition()]
      }
      emitValue({ ...props.modelValue, condition: type, conditions: nextChildren })
      return
    }

    // Wrapping a leaf in a logical group keeps the original leaf intact.
    const child = { ...props.modelValue }
    emitValue({ condition: type, conditions: [child] })
    return
  }

  if (!leafTypes.some((item) => item.value === type)) return
  const next = { ...removeDiscriminatorFields(props.modelValue), condition: type } as AutomationCondition
  if (type === 'state') {
    next.entity_key = props.modelValue.entity_key || props.entities[0]?.id || ''
    next.state = props.modelValue.state || 'on'
  } else if (type === 'numeric_state') {
    next.entity_key = props.modelValue.entity_key || props.entities.find((entity) => entity.domain === 'sensor')?.id || ''
    if (props.modelValue.above !== undefined) next.above = props.modelValue.above
    else next.above = 24
    if (props.modelValue.below !== undefined) next.below = props.modelValue.below
  } else if (type === 'time') {
    next.after = props.modelValue.after || '20:00:00'
    if (props.modelValue.before !== undefined) next.before = props.modelValue.before
  } else if (type === 'trigger') {
    next.trigger_id = props.modelValue.trigger_id || ''
  }
  emitValue(next)
}

function updateChild(index: number, condition: AutomationCondition) {
  const next = [...children.value]
  next[index] = condition
  updateField('conditions', next)
}

function addChild() {
  if (!canAddChild.value) return
  updateField('conditions', [...children.value, createDefaultCondition()])
}

function removeChild(index: number) {
  const next = children.value.filter((_, childIndex) => childIndex !== index)
  updateField('conditions', next)
}

function updateNumber(event: Event, field: 'above' | 'below') {
  const raw = (event.target as HTMLInputElement).value
  updateField(field, raw === '' ? undefined : Number(raw))
}

function updateText(event: Event, field: 'state' | 'trigger_id') {
  updateField(field, (event.target as HTMLInputElement).value)
}

function updateTime(event: Event, field: 'after' | 'before') {
  const raw = (event.target as HTMLInputElement).value
  updateField(field, raw || undefined)
}
</script>

<template>
  <section class="condition-tree-editor" :class="{ 'is-nested': depth > 0, 'is-unsupported': !isSupported }" data-test="condition-editor">
    <header class="condition-editor-header">
      <div class="condition-editor-heading">
        <span class="condition-depth-label">{{ isLogical ? 'Gruppo logico' : 'Condizione' }}</span>
        <strong>{{ modelValue.condition.toUpperCase() }}</strong>
      </div>
      <label class="condition-type-control">
        <span>Tipo</span>
        <select
          data-test="condition-type"
          :value="modelValue.condition"
          @change="changeType(($event.target as HTMLSelectElement).value)"
        >
          <optgroup v-if="isLogical" label="Operatori logici">
            <option v-for="option in logicalTypes" :key="option.value" :value="option.value">{{ option.label }}</option>
          </optgroup>
          <optgroup v-else label="Condizioni">
            <option v-for="option in leafTypes" :key="option.value" :value="option.value">{{ option.label }}</option>
            <option v-for="option in logicalTypes" :key="option.value" :value="option.value">{{ option.label }}</option>
          </optgroup>
        </select>
      </label>
    </header>

    <div v-if="isLogical" class="condition-children">
      <p class="condition-help">
        <template v-if="modelValue.condition === 'and'">Tutte le condizioni figlie devono essere vere.</template>
        <template v-else-if="modelValue.condition === 'or'">È sufficiente che una condizione figlia sia vera.</template>
        <template v-else>La condizione figlia deve essere falsa.</template>
      </p>

      <div v-for="(child, index) in children" :key="child.id || index" class="condition-child-row">
        <ConditionTreeEditor
          :model-value="child"
          :entities="entities"
          :depth="depth + 1"
          @update:model-value="updateChild(index, $event)"
        />
        <button
          v-if="modelValue.condition !== 'not' || children.length > 1"
          class="condition-remove-button"
          type="button"
          :aria-label="`Rimuovi condizione ${index + 1}`"
          :title="`Rimuovi condizione ${index + 1}`"
          :data-test="`remove-condition-${index}`"
          @click="removeChild(index)"
        >×</button>
      </div>

      <button
        class="condition-add-button"
        type="button"
        :disabled="!canAddChild"
        data-test="add-condition"
        @click="addChild"
      >＋ Aggiungi condizione</button>
      <p v-if="depth >= 8" class="condition-limit-note">Raggiunto il limite di annidamento supportato dal validatore.</p>
      <p v-else-if="modelValue.condition === 'not'" class="condition-limit-note">NOT accetta una sola condizione figlia.</p>
    </div>

    <div v-else-if="modelValue.condition === 'state'" class="condition-fields">
      <label>
        Entità
        <select
          data-test="condition-entity"
          :value="modelValue.entity_key || ''"
          @change="updateField('entity_key', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">Seleziona un'entità</option>
          <option v-for="entity in entities" :key="entity.id" :value="entity.id">
            {{ entity.name }} ({{ entity.id }})
          </option>
        </select>
      </label>
      <label>
        Stato richiesto
        <input data-test="condition-state" type="text" :value="modelValue.state || ''" placeholder="on, off, open..." @input="updateText($event, 'state')" />
      </label>
    </div>

    <div v-else-if="modelValue.condition === 'numeric_state'" class="condition-fields">
      <label>
        Sensore
        <select
          data-test="condition-entity"
          :value="modelValue.entity_key || ''"
          @change="updateField('entity_key', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">Seleziona un sensore</option>
          <option v-for="entity in entities.filter((item) => item.domain === 'sensor')" :key="entity.id" :value="entity.id">
            {{ entity.name }} ({{ entity.id }})
          </option>
        </select>
      </label>
      <div class="condition-number-row">
        <label>
          Maggiore di
          <input data-test="condition-above" type="number" step="any" :value="modelValue.above ?? ''" @input="updateNumber($event, 'above')" />
        </label>
        <label>
          Minore di
          <input data-test="condition-below" type="number" step="any" :value="modelValue.below ?? ''" @input="updateNumber($event, 'below')" />
        </label>
      </div>
      <p class="condition-help">Imposta almeno una delle due soglie.</p>
    </div>

    <div v-else-if="modelValue.condition === 'time'" class="condition-fields">
      <label>
        Dopo le
        <input data-test="condition-after" type="time" step="1" :value="modelValue.after || ''" @input="updateTime($event, 'after')" />
      </label>
      <label>
        Prima delle
        <input data-test="condition-before" type="time" step="1" :value="modelValue.before || ''" @input="updateTime($event, 'before')" />
      </label>
    </div>

    <div v-else-if="modelValue.condition === 'trigger'" class="condition-fields">
      <label>
        ID trigger
        <input data-test="condition-trigger-id" type="text" :value="modelValue.trigger_id || ''" placeholder="trigger_id" @input="updateText($event, 'trigger_id')" />
      </label>
    </div>

    <div v-else class="condition-unsupported-content">
      <p>Questo tipo di condizione non è modificabile con i controlli visuali. La configurazione originale viene mantenuta senza alterazioni.</p>
      <pre>{{ JSON.stringify(modelValue, null, 2) }}</pre>
    </div>
  </section>
</template>

<style scoped>
.condition-tree-editor {
  min-width: 0;
  padding: 12px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.condition-tree-editor.is-nested {
  background: var(--bg-surface-hover);
}
.condition-tree-editor.is-unsupported {
  border-style: dashed;
}
.condition-editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.condition-editor-heading {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.condition-depth-label,
.condition-type-control > span {
  color: var(--text-muted);
  font-size: 11px;
}
.condition-editor-heading strong {
  color: var(--text-main);
  font-size: 12px;
  letter-spacing: .04em;
}
.condition-type-control,
.condition-fields label {
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
  color: var(--text-muted);
  font-size: 11px;
}
.condition-type-control select,
.condition-fields select,
.condition-fields input {
  width: 100%;
  min-width: 0;
  min-height: 36px;
  padding: 7px 9px;
  color: var(--text-main);
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  font: inherit;
  font-size: 12px;
}
.condition-type-control select {
  max-width: 250px;
}
.condition-children,
.condition-fields {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}
.condition-child-row {
  display: flex;
  align-items: flex-start;
  gap: 7px;
}
.condition-child-row > .condition-tree-editor {
  flex: 1;
}
.condition-remove-button,
.condition-add-button {
  min-height: 34px;
  padding: 6px 10px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-main);
  background: var(--bg-surface);
  cursor: pointer;
}
.condition-remove-button {
  flex: 0 0 34px;
  font-size: 18px;
}
.condition-add-button {
  align-self: flex-start;
  font-size: 12px;
  font-weight: 600;
}
.condition-add-button:disabled {
  opacity: .45;
  cursor: not-allowed;
}
.condition-number-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 9px;
}
.condition-help,
.condition-limit-note,
.condition-unsupported-content p {
  margin: 0;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.45;
}
.condition-unsupported-content pre {
  max-height: 220px;
  overflow: auto;
  padding: 9px;
  background: var(--bg-surface-hover);
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  font-size: 10px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
@media (max-width: 480px) {
  .condition-number-row {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
