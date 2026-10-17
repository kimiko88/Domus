<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { X, Save, AlertCircle } from '@lucide/vue'
import type { PantryItem, PantryCategory, PantryLocation, PantryUnit } from '../../api/types'

interface Props {
  isOpen: boolean
  itemToEdit?: PantryItem | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', payload: Partial<PantryItem>): void
}>()

const name = ref('')
const category = ref<PantryCategory>('pantry')
const quantity = ref<number>(1)
const unit = ref<PantryUnit>('pz')
const minThreshold = ref<number>(1)
const expirationDate = ref<string>('')
const location = ref<PantryLocation>('pantry')
const cost = ref<number>(0)
const errorMessage = ref<string | null>(null)

const modalRef = ref<HTMLElement | null>(null)

const categories: { key: PantryCategory; label: string }[] = [
  { key: 'produce', label: '🍎 Frutta & Verdura' },
  { key: 'dairy', label: '🧀 Latticini & Uova' },
  { key: 'meat_fish', label: '🥩 Carne & Pesce' },
  { key: 'bakery', label: '🥖 Pane & Forno' },
  { key: 'pantry', label: '🍝 Pasta, Riso & Farina' },
  { key: 'frozen', label: '❄️ Surgelati' },
  { key: 'drinks', label: '🥤 Bevande & Acqua' },
  { key: 'condiments', label: '🫒 Olio & Condimenti' },
  { key: 'snacks', label: '🍪 Snack & Dolci' },
  { key: 'household', label: '🧼 Cura Casa & Igiene' },
  { key: 'other', label: '📦 Altro' }
]

const locations: { key: PantryLocation; label: string }[] = [
  { key: 'fridge', label: 'Frigorifero' },
  { key: 'freezer', label: 'Freezer / Congelatore' },
  { key: 'pantry', label: 'Mobile Dispensa' },
  { key: 'cellar', label: 'Cantina / Garage' },
  { key: 'other', label: 'Altro Luogo' }
]

const units: { key: PantryUnit; label: string }[] = [
  { key: 'pz', label: 'Pezzi (pz)' },
  { key: 'g', label: 'Grammi (g)' },
  { key: 'kg', label: 'Chilogrammi (kg)' },
  { key: 'ml', label: 'Millilitri (ml)' },
  { key: 'l', label: 'Litri (L)' },
  { key: 'pack', label: 'Confezioni (pack)' }
]

watch(
  () => props.isOpen,
  (val) => {
    if (val) {
      errorMessage.value = null
      if (props.itemToEdit) {
        name.value = props.itemToEdit.name
        category.value = props.itemToEdit.category
        quantity.value = props.itemToEdit.quantity
        unit.value = props.itemToEdit.unit
        minThreshold.value = props.itemToEdit.min_threshold
        expirationDate.value = props.itemToEdit.expiration_date || ''
        location.value = props.itemToEdit.location
        cost.value = props.itemToEdit.cost || 0
      } else {
        name.value = ''
        category.value = 'pantry'
        quantity.value = 1
        unit.value = 'pz'
        minThreshold.value = 1
        expirationDate.value = ''
        location.value = 'pantry'
        cost.value = 0
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

function setExpiryDays(days: number) {
  const d = new Date()
  d.setDate(d.getDate() + days)
  expirationDate.value = d.toISOString().split('T')[0]
}

function isDaysFromNow(days: number): boolean {
  if (!expirationDate.value) return false
  const target = new Date()
  target.setDate(target.getDate() + days)
  return expirationDate.value === target.toISOString().split('T')[0]
}

watch(category, (cat) => {
  if (!props.itemToEdit && !expirationDate.value) {
    if (cat === 'produce') setExpiryDays(5)
    else if (cat === 'dairy') setExpiryDays(7)
    else if (cat === 'meat_fish') setExpiryDays(3)
    else if (cat === 'bakery') setExpiryDays(4)
  }
})

function handleSubmit() {
  const trimmed = name.value.trim()
  if (!trimmed) {
    errorMessage.value = 'Il nome dell’alimento è obbligatorio'
    return
  }
  if (quantity.value < 0) {
    errorMessage.value = 'La quantità non può essere negativa'
    return
  }
  if (minThreshold.value < 0) {
    errorMessage.value = 'La soglia minima non può essere negativa'
    return
  }

  emit('save', {
    name: trimmed,
    category: category.value,
    quantity: Number(quantity.value),
    unit: unit.value,
    min_threshold: Number(minThreshold.value),
    expiration_date: expirationDate.value ? expirationDate.value : undefined,
    location: location.value,
    cost: Number(cost.value) || 0
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
        ref="modalRef"
        class="modal-dialog glass-panel"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="itemToEdit ? 'edit-title' : 'create-title'"
        @click.stop
      >
        <div class="modal-header">
          <h2 :id="itemToEdit ? 'edit-title' : 'create-title'" class="modal-title">
            {{ itemToEdit ? 'Modifica Alimento in Dispensa' : 'Nuovo Alimento in Dispensa' }}
          </h2>
          <button
            type="button"
            class="icon-btn"
            aria-label="Chiudi finestra"
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

          <!-- Name -->
          <div class="form-group">
            <label for="pantry-name" class="form-label">Nome Prodotto *</label>
            <input
              id="pantry-name"
              name="pantry_name"
              v-model="name"
              type="text"
              class="form-control"
              placeholder="es. Pasta Spaghetti, Latte Intero, Mele..."
              required
              autofocus
            />
          </div>

          <!-- Category & Location -->
          <div class="form-row">
            <div class="form-group">
              <label for="pantry-category" class="form-label">Categoria</label>
              <select id="pantry-category" name="pantry_category" v-model="category" class="form-control">
                <option v-for="c in categories" :key="c.key" :value="c.key">
                  {{ c.label }}
                </option>
              </select>
            </div>

            <div class="form-group">
              <label for="pantry-location" class="form-label">Posizione / Stoccaggio</label>
              <select id="pantry-location" name="pantry_location" v-model="location" class="form-control">
                <option v-for="l in locations" :key="l.key" :value="l.key">
                  {{ l.label }}
                </option>
              </select>
            </div>
          </div>

          <!-- Quantity, Unit & Threshold -->
          <div class="form-row form-row-3">
            <div class="form-group">
              <label for="pantry-qty" class="form-label" title="Quantità Attuale">Quantità Attuale *</label>
              <input
                id="pantry-qty"
                name="pantry_quantity"
                v-model.number="quantity"
                type="number"
                step="any"
                min="0"
                class="form-control"
                required
              />
            </div>

            <div class="form-group">
              <label for="pantry-unit" class="form-label" title="Unità">Unità</label>
              <select id="pantry-unit" name="pantry_unit" v-model="unit" class="form-control">
                <option v-for="u in units" :key="u.key" :value="u.key">
                  {{ u.label }}
                </option>
              </select>
            </div>

            <div class="form-group">
              <label for="pantry-threshold" class="form-label" title="Soglia Minima Avviso">Soglia Minima Avviso</label>
              <input
                id="pantry-threshold"
                name="pantry_threshold"
                v-model.number="minThreshold"
                type="number"
                step="any"
                min="0"
                class="form-control"
                title="Sotto questa soglia l'alimento viene segnalato come 'Scorta bassa'"
              />
            </div>
          </div>

          <!-- Expiration Date & Cost -->
          <div class="form-row form-row-expiry-cost">
            <div class="form-group">
              <div class="expiry-label-row">
                <label for="pantry-expiry" class="form-label" title="Data di Scadenza (Opzionale)">Data di Scadenza (Opzionale)</label>
                <button
                  v-if="expirationDate"
                  type="button"
                  class="text-btn-clear"
                  @click="expirationDate = ''"
                >
                  Rimuovi scadenza
                </button>
              </div>
              <input
                id="pantry-expiry"
                name="pantry_expiration_date"
                v-model="expirationDate"
                type="date"
                class="form-control"
              />
              <div class="expiry-chips-row">
                <span class="expiry-chips-label">Scadenza rapida dall'acquisto:</span>
                <div class="chips-group">
                  <button type="button" class="chip-btn" :class="{ active: isDaysFromNow(3) }" @click="setExpiryDays(3)">+3gg (Carne/Pesce)</button>
                  <button type="button" class="chip-btn" :class="{ active: isDaysFromNow(5) }" @click="setExpiryDays(5)">+5gg (Ortofrutta)</button>
                  <button type="button" class="chip-btn" :class="{ active: isDaysFromNow(7) }" @click="setExpiryDays(7)">+7gg (Freschi)</button>
                  <button type="button" class="chip-btn" :class="{ active: isDaysFromNow(14) }" @click="setExpiryDays(14)">+14gg</button>
                  <button type="button" class="chip-btn" :class="{ active: isDaysFromNow(30) }" @click="setExpiryDays(30)">+1 mese</button>
                  <button type="button" class="chip-btn" :class="{ active: isDaysFromNow(180) }" @click="setExpiryDays(180)">+6 mesi</button>
                </div>
              </div>
            </div>

            <div class="form-group">
              <label for="pantry-cost" class="form-label" title="Costo Acquisto (€)">Costo Acquisto (€)</label>
              <input
                id="pantry-cost"
                name="pantry_cost"
                v-model.number="cost"
                type="number"
                step="0.01"
                min="0"
                class="form-control"
                placeholder="0.00"
              />
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
              <span>{{ itemToEdit ? 'Salva Modifiche' : 'Aggiungi Alimento' }}</span>
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
  width: 95vw;
  max-width: 660px;
  background: var(--surface-card, #1e293b);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  border-radius: var(--radius-xl, 20px);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);
  overflow: hidden;
  box-sizing: border-box;
  animation: modalScale 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes modalScale {
  from {
    transform: scale(0.96);
    opacity: 0;
  }
  to {
    transform: scale(1);
    opacity: 1;
  }
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
  width: 100%;
  box-sizing: border-box;
  overflow-x: hidden;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
  min-width: 0;
  box-sizing: border-box;
}

.form-row {
  display: flex;
  gap: var(--space-3, 12px);
  width: 100%;
  box-sizing: border-box;
  min-width: 0;
}

.form-row-3 {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-3, 12px);
  width: 100%;
  box-sizing: border-box;
  min-width: 0;
}

.form-row-expiry-cost {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(0, 1fr);
  gap: var(--space-3, 12px);
  width: 100%;
  box-sizing: border-box;
  min-width: 0;
}

.form-label {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text-muted, #94a3b8);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
  line-height: 1.25;
}

.form-control {
  width: 100%;
  min-width: 0;
  max-width: 100%;
  box-sizing: border-box;
  height: 44px;
  padding: 0 14px;
  border-radius: var(--radius-md, 10px);
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  color: var(--text-main, #f8fafc);
  font-size: 0.95rem;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.form-control:focus {
  outline: none;
  border-color: var(--primary, #3b82f6);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.25);
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
  margin-top: var(--space-3, 12px);
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

.btn-secondary:hover {
  background: rgba(255, 255, 255, 0.1);
}

.btn-primary {
  background: var(--primary, #3b82f6);
  color: #fff;
}

.btn-primary:hover {
  background: #2563eb;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.35);
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
  transition: background 0.2s, color 0.2s;
}

.icon-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-main, #f8fafc);
}

select.form-control {
  color-scheme: dark;
}

select.form-control option {
  background-color: #1e293b !important;
  color: #f8fafc !important;
}

.expiry-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.text-btn-clear {
  background: transparent;
  border: none;
  font-size: 0.75rem;
  color: var(--danger, #ef4444);
  cursor: pointer;
  padding: 0;
}
.text-btn-clear:hover {
  text-decoration: underline;
}

.expiry-chips-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 4px;
}

.expiry-chips-label {
  font-size: 0.72rem;
  color: var(--text-muted, #94a3b8);
}

.chips-group {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.chip-btn {
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 0.72rem;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: var(--text-main, #f8fafc);
  cursor: pointer;
  transition: all 0.15s ease;
}
.chip-btn:hover {
  background: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.2);
}
.chip-btn.active {
  background: rgba(14, 165, 233, 0.25);
  border-color: var(--primary, #0ea5e9);
  color: #38bdf8;
  font-weight: 600;
}

@media (max-width: 600px) {
  .form-row,
  .form-row-3,
  .form-row-expiry-cost {
    display: flex;
    flex-direction: column;
  }
}
</style>
