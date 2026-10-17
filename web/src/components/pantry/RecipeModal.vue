<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import {
  X,
  Save,
  Plus,
  Trash2,
  AlertCircle,
  Clock,
  ArrowUp,
  ArrowDown,
  ListOrdered
} from '@lucide/vue'
import type { Recipe, RecipeCategory, PantryUnit } from '../../api/types'

interface Props {
  isOpen: boolean
  recipeToEdit?: Recipe | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', payload: Partial<Recipe>): void
}>()

export interface RecipeStep {
  id: string
  step: number
  text: string
  time_minutes: number
}

const title = ref('')
const category = ref<RecipeCategory>('first_course')
const portionsBase = ref<number>(2)
const prepTimeMinutes = ref<number>(10)
const cookTimeMinutes = ref<number>(15)
const ingredients = ref<Array<{ name: string; quantity: number; unit: PantryUnit; is_optional: boolean }>>([])
const steps = ref<RecipeStep[]>([])
const errorMessage = ref<string | null>(null)

const categories: { key: RecipeCategory; label: string }[] = [
  { key: 'first_course', label: '🍝 Primo Piatto' },
  { key: 'main_course', label: '🥩 Secondo Piatto' },
  { key: 'side_dish', label: '🥗 Contorno' },
  { key: 'single_dish', label: '🍲 Piatto Unico' },
  { key: 'breakfast', label: '☕ Colazione' },
  { key: 'dessert', label: '🍰 Dolce' },
  { key: 'snack', label: '🥪 Spuntino / Merenda' }
]

const units: { key: PantryUnit; label: string }[] = [
  { key: 'g', label: 'g (grammi)' },
  { key: 'kg', label: 'kg (chili)' },
  { key: 'pz', label: 'pz (pezzi)' },
  { key: 'ml', label: 'ml (millilitri)' },
  { key: 'l', label: 'L (litri)' },
  { key: 'pack', label: 'pack (confezione)' }
]

function parseInstructionsToSteps(raw: string): RecipeStep[] {
  if (!raw || !raw.trim()) {
    return [
      { id: 'step_1', step: 1, text: '', time_minutes: 5 }
    ]
  }
  const trimmed = raw.trim()
  if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
    try {
      const parsed = JSON.parse(trimmed)
      if (Array.isArray(parsed) && parsed.length > 0) {
        return parsed.map((s, idx) => ({
          id: 'step_' + (idx + 1) + '_' + Math.random().toString(36).substring(2, 7),
          step: idx + 1,
          text: s.text || s.instruction || '',
          time_minutes: Number(s.time_minutes || s.time || s.duration) || 0
        }))
      }
    } catch {}
  }

  // Fallback: parse lines
  const lines = trimmed.split('\n').filter((l) => l.trim().length > 0)
  if (lines.length > 1) {
    return lines.map((line, idx) => {
      let time = 0
      const timeMatch = line.match(/\((\d+)\s*(?:min|m)?\)/i)
      if (timeMatch) time = parseInt(timeMatch[1], 10)
      const cleanText = line
        .replace(/^\d+[\.\)\-]\s*/, '')
        .replace(/\(\d+\s*(?:min|m)?\)/i, '')
        .trim()
      return {
        id: 'step_' + (idx + 1) + '_' + Math.random().toString(36).substring(2, 7),
        step: idx + 1,
        text: cleanText,
        time_minutes: time
      }
    })
  }

  return [
    {
      id: 'step_1',
      step: 1,
      text: trimmed,
      time_minutes: 0
    }
  ]
}

watch(
  () => props.isOpen,
  (val) => {
    if (val) {
      errorMessage.value = null
      if (props.recipeToEdit) {
        title.value = props.recipeToEdit.title
        category.value = props.recipeToEdit.category
        portionsBase.value = props.recipeToEdit.portions_base || 2
        prepTimeMinutes.value = props.recipeToEdit.prep_time_minutes || 10
        cookTimeMinutes.value = props.recipeToEdit.cook_time_minutes || 15
        steps.value = parseInstructionsToSteps(props.recipeToEdit.instructions || '')
        ingredients.value = (props.recipeToEdit.ingredients || []).map((ing) => ({
          name: ing.name,
          quantity: ing.quantity,
          unit: ing.unit,
          is_optional: !!ing.is_optional
        }))
      } else {
        title.value = ''
        category.value = 'first_course'
        portionsBase.value = 2
        prepTimeMinutes.value = 10
        cookTimeMinutes.value = 15
        ingredients.value = [
          { name: '', quantity: 100, unit: 'g', is_optional: false }
        ]
        steps.value = [
          { id: 'step_1', step: 1, text: '', time_minutes: 5 },
          { id: 'step_2', step: 2, text: '', time_minutes: 10 }
        ]
      }
    }
  },
  { immediate: true }
)

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.isOpen) {
    emit('close')
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeyDown)
})

function addIngredient() {
  ingredients.value.push({
    name: '',
    quantity: 100,
    unit: 'g',
    is_optional: false
  })
}

function removeIngredient(index: number) {
  ingredients.value.splice(index, 1)
}

function addStep() {
  steps.value.push({
    id: 'step_' + (steps.value.length + 1) + '_' + Math.random().toString(36).substring(2, 7),
    step: steps.value.length + 1,
    text: '',
    time_minutes: 5
  })
}

function removeStep(index: number) {
  steps.value.splice(index, 1)
  steps.value.forEach((s, i) => {
    s.step = i + 1
  })
}

function moveStep(index: number, direction: -1 | 1) {
  const target = index + direction
  if (target < 0 || target >= steps.value.length) return
  const temp = steps.value[index]
  steps.value[index] = steps.value[target]
  steps.value[target] = temp
  steps.value.forEach((s, i) => {
    s.step = i + 1
  })
}

const totalStepsTime = computed(() => {
  return steps.value.reduce((acc, s) => acc + (Number(s.time_minutes) || 0), 0)
})

function applyStepsTotalToCookTime() {
  if (totalStepsTime.value > 0) {
    cookTimeMinutes.value = totalStepsTime.value
  }
}

function handleSubmit() {
  const trimmedTitle = title.value.trim()
  if (!trimmedTitle) {
    errorMessage.value = 'Il titolo della ricetta è obbligatorio'
    return
  }
  if (portionsBase.value < 1) {
    errorMessage.value = 'Le porzioni base devono essere almeno 1'
    return
  }
  const validIngredients = ingredients.value
    .filter((i) => i.name.trim().length > 0)
    .map((i) => ({
      name: i.name.trim(),
      quantity: Number(i.quantity) || 0,
      unit: i.unit,
      is_optional: i.is_optional
    }))

  if (validIngredients.length === 0) {
    errorMessage.value = 'Aggiungi almeno un ingrediente alla ricetta'
    return
  }

  const validSteps = steps.value
    .filter((s) => s.text.trim().length > 0)
    .map((s, idx) => ({
      step: idx + 1,
      text: s.text.trim(),
      time_minutes: Number(s.time_minutes) || 0
    }))

  const serializedInstructions = validSteps.length > 0
    ? JSON.stringify(validSteps)
    : ''

  emit('save', {
    title: trimmedTitle,
    category: category.value,
    portions_base: Number(portionsBase.value),
    prep_time_minutes: Number(prepTimeMinutes.value) || 0,
    cook_time_minutes: Number(cookTimeMinutes.value) || 0,
    instructions: serializedInstructions,
    ingredients: validIngredients
  })
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="isOpen"
      class="modal-backdrop"
      @click="emit('close')"
    >
      <div
        class="modal-dialog glass-panel"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="recipeToEdit ? 'edit-recipe-title' : 'create-recipe-title'"
        @click.stop
      >
        <div class="modal-header">
          <h2 :id="recipeToEdit ? 'edit-recipe-title' : 'create-recipe-title'" class="modal-title">
            {{ recipeToEdit ? 'Modifica Ricetta' : 'Nuova Ricetta nel Ricettario' }}
          </h2>
          <button
            type="button"
            class="icon-btn"
            aria-label="Chiudi"
            @click="emit('close')"
          >
            <X :size="20" />
          </button>
        </div>

        <form class="modal-body scrollable" @submit.prevent="handleSubmit">
          <div v-if="errorMessage" class="error-banner" role="alert">
            <AlertCircle :size="18" />
            <span>{{ errorMessage }}</span>
          </div>

          <!-- Title -->
          <div class="form-group">
            <label for="recipe-title" class="form-label">Titolo Ricetta *</label>
            <input
              id="recipe-title"
              v-model="title"
              type="text"
              class="form-control"
              placeholder="es. Risotto ai Funghi Porcini, Carbonara, Salmone al Forno..."
              required
              autofocus
            />
          </div>

          <!-- Category & Portions Base -->
          <div class="form-row">
            <div class="form-group">
              <label for="recipe-cat" class="form-label">Portata / Categoria</label>
              <select id="recipe-cat" v-model="category" class="form-control select-dark">
                <option v-for="c in categories" :key="c.key" :value="c.key">
                  {{ c.label }}
                </option>
              </select>
            </div>

            <div class="form-group">
              <label for="recipe-portions" class="form-label">Porzioni Base (Persone) *</label>
              <input
                id="recipe-portions"
                v-model.number="portionsBase"
                type="number"
                min="1"
                class="form-control"
                required
                title="Dosi calibrate per questo numero di commensali (verranno scalate in automatico nel calendario)"
              />
            </div>
          </div>

          <!-- Prep & Cook Time -->
          <div class="form-row">
            <div class="form-group">
              <label for="recipe-prep" class="form-label">Preparazione (min)</label>
              <input
                id="recipe-prep"
                v-model.number="prepTimeMinutes"
                type="number"
                min="0"
                class="form-control"
              />
            </div>

            <div class="form-group">
              <label for="recipe-cook" class="form-label">Cottura (min)</label>
              <input
                id="recipe-cook"
                v-model.number="cookTimeMinutes"
                type="number"
                min="0"
                class="form-control"
              />
            </div>
          </div>

          <!-- Ingredients List -->
          <div class="ingredients-section">
            <div class="section-header">
              <span class="form-label section-title">Ingredienti (per {{ portionsBase }} persone)</span>
              <button
                type="button"
                class="btn-sm btn-ghost"
                @click="addIngredient"
              >
                <Plus :size="16" />
                <span>Aggiungi Ingrediente</span>
              </button>
            </div>

            <div class="ingredients-list">
              <div
                v-for="(ing, idx) in ingredients"
                :key="idx"
                class="ingredient-row"
              >
                <input
                  v-model="ing.name"
                  type="text"
                  class="form-control ing-name"
                  placeholder="Nome ingrediente (es. Spaghetti)"
                  required
                />

                <input
                  v-model.number="ing.quantity"
                  type="number"
                  step="any"
                  min="0"
                  class="form-control ing-qty"
                  placeholder="Dose"
                  required
                />

                <select v-model="ing.unit" class="form-control ing-unit select-dark">
                  <option v-for="u in units" :key="u.key" :value="u.key">
                    {{ u.key }}
                  </option>
                </select>

                <label class="optional-label" title="Opzionale">
                  <input v-model="ing.is_optional" type="checkbox" />
                  <span>Opz.</span>
                </label>

                <button
                  type="button"
                  class="icon-btn-danger"
                  aria-label="Rimuovi ingrediente"
                  @click="removeIngredient(idx)"
                >
                  <Trash2 :size="16" />
                </button>
              </div>
            </div>
          </div>

          <!-- Recipe Steps & Timers Section -->
          <div class="steps-section">
            <div class="section-header">
              <div class="steps-title-group">
                <ListOrdered :size="18" class="text-primary" />
                <span class="form-label section-title">Passaggi & Tempi di Preparazione</span>
                <span v-if="totalStepsTime > 0" class="steps-total-badge">
                  ⏱️ Somma passaggi: {{ totalStepsTime }} min
                </span>
              </div>
              <div class="steps-header-actions">
                <button
                  v-if="totalStepsTime > 0 && totalStepsTime !== cookTimeMinutes"
                  type="button"
                  class="btn-sm btn-outline-timer"
                  title="Imposta il tempo cottura totale pari alla somma dei passaggi"
                  @click="applyStepsTotalToCookTime"
                >
                  <Clock :size="14" />
                  <span>Usa come tempo cottura</span>
                </button>
                <button
                  type="button"
                  class="btn-sm btn-ghost"
                  @click="addStep"
                >
                  <Plus :size="16" />
                  <span>Aggiungi Step</span>
                </button>
              </div>
            </div>

            <div v-if="steps.length === 0" class="steps-empty">
              Nessun passaggio definito. Clicca su "+ Aggiungi Step" per descrivere la preparazione passo dopo passo.
            </div>

            <div v-else class="steps-list">
              <div
                v-for="(st, idx) in steps"
                :key="st.id"
                class="step-card"
              >
                <div class="step-card-header">
                  <div class="step-badge">
                    <span>Step {{ idx + 1 }}</span>
                  </div>

                  <div class="step-timer-field">
                    <Clock :size="14" class="text-muted" />
                    <label :for="'step-time-' + idx" class="sr-only">Tempo Step {{ idx + 1 }} in minuti</label>
                    <input
                      :id="'step-time-' + idx"
                      v-model.number="st.time_minutes"
                      type="number"
                      min="0"
                      class="step-time-input"
                      placeholder="0"
                      title="Durata timer passaggio in minuti (0 se istantaneo)"
                    />
                    <span class="step-time-unit">min</span>
                  </div>

                  <div class="step-actions">
                    <button
                      type="button"
                      class="step-reorder-btn"
                      :disabled="idx === 0"
                      title="Sposta su"
                      aria-label="Sposta su"
                      @click="moveStep(idx, -1)"
                    >
                      <ArrowUp :size="14" />
                    </button>
                    <button
                      type="button"
                      class="step-reorder-btn"
                      :disabled="idx === steps.length - 1"
                      title="Sposta giù"
                      aria-label="Sposta giù"
                      @click="moveStep(idx, 1)"
                    >
                      <ArrowDown :size="14" />
                    </button>
                    <button
                      type="button"
                      class="step-delete-btn"
                      title="Elimina passaggio"
                      aria-label="Elimina passaggio"
                      @click="removeStep(idx)"
                    >
                      <Trash2 :size="14" />
                    </button>
                  </div>
                </div>

                <div class="step-card-body">
                  <textarea
                    v-model="st.text"
                    rows="2"
                    class="form-control step-textarea"
                    :placeholder="'Descrivi cosa fare nel passaggio ' + (idx + 1) + ' (es. Soffriggere cipolla, sfumare col vino, infornare...)'"
                  ></textarea>
                </div>
              </div>
            </div>
          </div>

          <div class="modal-footer">
            <button
              type="button"
              class="btn btn-secondary"
              @click="emit('close')"
            >
              Annulla
            </button>
            <button
              type="submit"
              class="btn btn-primary"
            >
              <Save :size="18" />
              <span>{{ recipeToEdit ? 'Salva Ricetta' : 'Crea Ricetta' }}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: var(--space-4, 16px);
  overflow-y: auto;
}

.modal-dialog {
  width: 95vw;
  max-width: 760px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  background: var(--surface-card, #1e293b);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  border-radius: var(--radius-xl, 20px);
  box-shadow: 0 25px 50px rgba(0, 0, 0, 0.5);
  overflow: hidden;
  box-sizing: border-box;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  flex-shrink: 0;
}

.modal-title {
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
  margin: 0;
}

.modal-body.scrollable {
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow-y: auto;
  overflow-x: hidden;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.form-label {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text-muted, #94a3b8);
}

.form-control {
  height: 44px;
  padding: 0 14px;
  border-radius: var(--radius-md, 10px);
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.14));
  color: var(--text-main, #f8fafc);
  font-size: 0.95rem;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
  min-width: 0;
  width: 100%;
  box-sizing: border-box;
}

.form-control:focus {
  outline: none;
  border-color: var(--primary, #3b82f6);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.25);
}

/* Ensure native select and option readability in dark mode */
select.form-control,
.select-dark {
  color-scheme: dark;
}

select.form-control option,
.select-dark option {
  background-color: #1e293b !important;
  color: #f8fafc !important;
}

.ingredients-section,
.steps-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  background: rgba(255, 255, 255, 0.02);
  border: 1px dashed var(--border-color, rgba(255, 255, 255, 0.14));
  border-radius: var(--radius-lg, 14px);
  box-sizing: border-box;
  width: 100%;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}

.section-title {
  font-weight: 600;
  color: var(--text-main, #f8fafc);
}

.ingredients-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
}

.ingredient-row {
  display: grid;
  grid-template-columns: 1fr 100px 90px auto 38px;
  align-items: center;
  gap: 8px;
  width: 100%;
  box-sizing: border-box;
}

.ingredient-row input,
.ingredient-row select {
  min-width: 0;
}

.optional-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.8rem;
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
  white-space: nowrap;
  user-select: none;
}

.icon-btn-danger {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 8px;
  background: transparent;
  border: none;
  color: var(--danger, #ef4444);
  cursor: pointer;
  transition: background 0.2s;
  flex-shrink: 0;
}
.icon-btn-danger:hover {
  background: rgba(239, 68, 68, 0.15);
}

/* Steps Section Styling */
.steps-title-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.steps-total-badge {
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--primary, #38bdf8);
  background: rgba(14, 165, 233, 0.15);
  padding: 3px 8px;
  border-radius: 6px;
}

.steps-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-outline-timer {
  color: var(--primary, #38bdf8);
  border-color: rgba(14, 165, 233, 0.3);
  background: rgba(14, 165, 233, 0.08);
}
.btn-outline-timer:hover {
  background: rgba(14, 165, 233, 0.18);
}

.steps-empty {
  padding: 16px;
  text-align: center;
  font-size: 0.85rem;
  color: var(--text-muted, #94a3b8);
  border: 1px dashed rgba(255, 255, 255, 0.08);
  border-radius: 8px;
}

.steps-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.step-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
}

.step-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.step-badge {
  font-size: 0.82rem;
  font-weight: 700;
  color: #fff;
  background: rgba(59, 130, 246, 0.3);
  border: 1px solid rgba(59, 130, 246, 0.5);
  padding: 3px 10px;
  border-radius: 6px;
}

.step-timer-field {
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(0, 0, 0, 0.25);
  padding: 2px 8px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.step-time-input {
  width: 50px;
  height: 28px;
  background: transparent;
  border: none;
  color: #fff;
  font-weight: 600;
  text-align: center;
  font-size: 0.9rem;
}
.step-time-input:focus {
  outline: none;
}

.step-time-unit {
  font-size: 0.75rem;
  color: var(--text-muted, #94a3b8);
}

.step-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.step-reorder-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
}
.step-reorder-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}
.step-reorder-btn:not(:disabled):hover {
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
}

.step-delete-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.2);
  color: var(--danger, #ef4444);
  cursor: pointer;
}
.step-delete-btn:hover {
  background: rgba(239, 68, 68, 0.25);
}

.step-textarea {
  min-height: 60px;
  height: auto;
  padding: 10px;
  font-size: 0.9rem;
  line-height: 1.4;
  resize: vertical;
}

.btn-sm {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  font-size: 0.82rem;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.15));
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-main, #f8fafc);
  transition: all 0.2s;
}
.btn-sm:hover {
  background: rgba(255, 255, 255, 0.1);
}

.error-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid var(--danger, #ef4444);
  border-radius: var(--radius-md, 8px);
  color: var(--danger, #ef4444);
  font-size: 0.85rem;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 8px;
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 20px;
  border-radius: var(--radius-md, 10px);
  font-size: 0.92rem;
  font-weight: 600;
  cursor: pointer;
  border: none;
  transition: all 0.2s ease;
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-main, #f8fafc);
}
.btn-primary {
  background: var(--primary, #3b82f6);
  color: #fff;
}
.btn-primary:hover {
  background: #2563eb;
}

.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background: transparent;
  border: none;
  color: var(--text-muted, #94a3b8);
  border-radius: 50%;
  cursor: pointer;
}

@media (max-width: 640px) {
  .form-row {
    grid-template-columns: 1fr;
    gap: 10px;
  }
  .ingredient-row {
    grid-template-columns: 1fr 80px 75px auto 34px;
    gap: 6px;
  }
  .section-header {
    flex-direction: column;
    align-items: flex-start;
  }
  .steps-header-actions {
    width: 100%;
    justify-content: space-between;
  }
}
</style>
