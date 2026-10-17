<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { X, Calendar, AlertCircle } from '@lucide/vue'
import type { MealPlan, MealType, Recipe } from '../../api/types'

interface Props {
  isOpen: boolean
  recipes: Recipe[]
  initialDate?: string
  initialMealType?: MealType
  initialRecipeId?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', payload: Partial<MealPlan>): void
}>()

const planDate = ref('')
const mealType = ref<MealType>('dinner')
const selectedRecipeId = ref<string>('')
const customTitle = ref('')
const personsCount = ref<number>(2)
const notes = ref('')
const errorMessage = ref<string | null>(null)

const mealTypes: { key: MealType; label: string }[] = [
  { key: 'breakfast', label: '☕ Colazione' },
  { key: 'lunch', label: '☀️ Pranzo' },
  { key: 'dinner', label: '🌙 Cena' },
  { key: 'snack', label: '🥪 Spuntino / Merenda' },
  { key: 'dessert', label: '🍰 Dessert' }
]

function getTodayString(): string {
  const d = new Date()
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

watch(
  () => props.isOpen,
  (val) => {
    if (val) {
      errorMessage.value = null
      planDate.value = props.initialDate || getTodayString()
      mealType.value = props.initialMealType || 'dinner'
      selectedRecipeId.value = props.initialRecipeId || ''
      customTitle.value = ''
      // If recipe is preselected, copy base portions
      if (selectedRecipeId.value) {
        const found = props.recipes.find((r) => r.id === selectedRecipeId.value)
        if (found) {
          personsCount.value = found.portions_base || 2
        } else {
          personsCount.value = 2
        }
      } else {
        personsCount.value = 2
      }
      notes.value = ''
    }
  },
  { immediate: true }
)

watch(selectedRecipeId, (newId) => {
  if (newId) {
    const found = props.recipes.find((r) => r.id === newId)
    if (found && found.portions_base) {
      personsCount.value = found.portions_base
    }
  }
})

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

function handleSubmit() {
  if (!planDate.value) {
    errorMessage.value = 'Seleziona la data del pasto'
    return
  }
  if (!selectedRecipeId.value && !customTitle.value.trim()) {
    errorMessage.value = 'Seleziona una ricetta dal ricettario oppure inserisci un titolo'
    return
  }
  if (personsCount.value < 1) {
    errorMessage.value = 'Il numero di commensali deve essere almeno 1'
    return
  }

  emit('save', {
    plan_date: planDate.value,
    meal_type: mealType.value,
    recipe_id: selectedRecipeId.value ? selectedRecipeId.value : undefined,
    custom_title: customTitle.value.trim() ? customTitle.value.trim() : undefined,
    persons_count: Number(personsCount.value),
    notes: notes.value.trim() ? notes.value.trim() : undefined
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
        aria-labelledby="meal-plan-title"
        @click.stop
      >
        <div class="modal-header">
          <h2 id="meal-plan-title" class="modal-title">
            Pianifica Pasto nel Calendario
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

        <form class="modal-body" @submit.prevent="handleSubmit">
          <div v-if="errorMessage" class="error-banner" role="alert">
            <AlertCircle :size="18" />
            <span>{{ errorMessage }}</span>
          </div>

          <!-- Date & Meal Type -->
          <div class="form-row">
            <div class="form-group">
              <label for="meal-date" class="form-label">Data Pasto *</label>
              <input
                id="meal-date"
                v-model="planDate"
                type="date"
                class="form-control"
                required
              />
            </div>

            <div class="form-group">
              <label for="meal-type" class="form-label">Slot Orario</label>
              <select id="meal-type" v-model="mealType" class="form-control">
                <option v-for="m in mealTypes" :key="m.key" :value="m.key">
                  {{ m.label }}
                </option>
              </select>
            </div>
          </div>

          <!-- Recipe Selection -->
          <div class="form-group">
            <label for="meal-recipe" class="form-label">Seleziona Ricetta dal Ricettario</label>
            <select id="meal-recipe" v-model="selectedRecipeId" class="form-control">
              <option value="">-- Nessuna (Titolo libero) --</option>
              <option v-for="r in recipes" :key="r.id" :value="r.id">
                {{ r.title }} (base: {{ r.portions_base }} pers.)
              </option>
            </select>
            <span class="help-text">
              Scegliendo una ricetta, le dosi degli ingredienti verranno scalate automaticamente in proporzione ai commensali quando consumerai il pasto!
            </span>
          </div>

          <!-- Custom Title (if not using recipe or override) -->
          <div v-if="!selectedRecipeId" class="form-group">
            <label for="custom-title" class="form-label">Oppure Titolo Piatto Libero *</label>
            <input
              id="custom-title"
              v-model="customTitle"
              type="text"
              class="form-control"
              placeholder="es. Pizza fatta in casa, Insalata veloce..."
            />
          </div>

          <!-- Persons Count -->
          <div class="form-group">
            <label for="persons-count" class="form-label">Numero Commensali (Persone) *</label>
            <div class="persons-selector">
              <input
                id="persons-count"
                v-model.number="personsCount"
                type="number"
                min="1"
                class="form-control count-input"
                required
              />
              <div class="quick-persons">
                <button
                  v-for="p in [1, 2, 4, 6]"
                  :key="p"
                  type="button"
                  :class="['pill-btn', { active: personsCount === p }]"
                  @click="personsCount = p"
                >
                  {{ p }} pers.
                </button>
              </div>
            </div>
          </div>

          <!-- Notes -->
          <div class="form-group">
            <label for="meal-notes" class="form-label">Note o Preferenze</label>
            <input
              id="meal-notes"
              v-model="notes"
              type="text"
              class="form-control"
              placeholder="es. Marco cena fuori, preparare porzione in meno..."
            />
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
              <Calendar :size="18" />
              <span>Pianifica Pasto</span>
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
  background: rgba(0, 0, 0, 0.65);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: var(--space-4, 16px);
}

.modal-dialog {
  width: 100%;
  max-width: 540px;
  background: var(--surface-card, #1e293b);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  border-radius: var(--radius-xl, 20px);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4, 16px) var(--space-6, 24px);
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
}

.modal-title {
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
  margin: 0;
}

.modal-body {
  padding: var(--space-6, 24px);
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 16px);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}

.form-row {
  display: flex;
  gap: var(--space-3, 12px);
}

.form-label {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text-muted, #94a3b8);
}

.help-text {
  font-size: 0.78rem;
  color: var(--primary, #3b82f6);
  line-height: 1.4;
  margin-top: 2px;
}

.form-control {
  height: 44px;
  padding: 0 14px;
  border-radius: var(--radius-md, 10px);
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  color: var(--text-main, #f8fafc);
  font-size: 0.95rem;
}
.form-control:focus {
  outline: none;
  border-color: var(--primary, #3b82f6);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.25);
}

select.form-control {
  color-scheme: dark;
}
select.form-control option {
  background-color: #1e293b !important;
  color: #f8fafc !important;
}

.persons-selector {
  display: flex;
  align-items: center;
  gap: 12px;
}

.count-input {
  width: 90px;
}

.quick-persons {
  display: flex;
  gap: 6px;
}

.pill-btn {
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 0.82rem;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
  transition: all 0.2s ease;
}
.pill-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-main, #f8fafc);
}
.pill-btn.active {
  background: var(--primary, #3b82f6);
  border-color: var(--primary, #3b82f6);
  color: #fff;
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
  gap: var(--space-3, 12px);
  margin-top: var(--space-2, 8px);
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

@media (max-width: 500px) {
  .form-row {
    flex-direction: column;
  }
}
</style>
