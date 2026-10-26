<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { X, Save, AlertCircle, AlertTriangle, Check } from '@lucide/vue'
import type { PantryItem, PantryCategory, PantryLocation, PantryUnit } from '../../api/types'
import { usePantryStore } from '../../stores/pantry'

interface Props {
  isOpen: boolean
  itemToEdit?: PantryItem | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', payload: Partial<PantryItem>): void
}>()

const pantryStore = usePantryStore()

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

function getCategoryLabel(catKey: PantryCategory): string {
  const c = categories.find((cat) => cat.key === catKey)
  return c ? c.label : catKey
}

// --- AUTOCOMPLETE & EXISTING PRODUCTS ---
interface ExistingProductSummary {
  name: string
  category: PantryCategory
  unit: PantryUnit
  location: PantryLocation
  min_threshold: number
  batchesCount: number
}

// Compute distinct existing products from the pantry store
const existingProducts = computed<ExistingProductSummary[]>(() => {
  const map = new Map<string, ExistingProductSummary>()
  for (const item of pantryStore.items) {
    const key = item.name.trim().toLowerCase()
    if (!key) continue
    const existing = map.get(key)
    if (!existing) {
      map.set(key, {
        name: item.name.trim(),
        category: item.category,
        unit: item.unit,
        location: item.location,
        min_threshold: item.min_threshold ?? 1,
        batchesCount: 1
      })
    } else {
      existing.batchesCount++
      // Keep established threshold
      if (item.min_threshold > 0 && existing.min_threshold === 0) {
        existing.min_threshold = item.min_threshold
      }
    }
  }
  return Array.from(map.values())
})

const showSuggestions = ref(false)
const highlightedIndex = ref(-1)

// Filter existing items matching user typing
const filteredSuggestions = computed(() => {
  if (props.itemToEdit) return []
  const q = name.value.trim().toLowerCase()
  if (!q) return []
  return existingProducts.value.filter((p) =>
    p.name.toLowerCase().includes(q)
  ).slice(0, 6)
})

// Match exact product name to retrieve established threshold
const matchedExistingProduct = computed<ExistingProductSummary | null>(() => {
  if (props.itemToEdit) return null
  const q = name.value.trim().toLowerCase()
  if (!q) return null
  return existingProducts.value.find((p) => p.name.toLowerCase() === q) || null
})

function selectSuggestion(item: ExistingProductSummary) {
  name.value = item.name
  category.value = item.category
  unit.value = item.unit
  location.value = item.location
  minThreshold.value = item.min_threshold
  showSuggestions.value = false
  highlightedIndex.value = -1
}

function onNameInput() {
  showSuggestions.value = true
  highlightedIndex.value = -1
}

function onNameFocus() {
  if (name.value.trim()) {
    showSuggestions.value = true
  }
}

function onNameBlur() {
  setTimeout(() => {
    showSuggestions.value = false
  }, 200)
}

function onNameKeydown(e: KeyboardEvent) {
  if (!showSuggestions.value || filteredSuggestions.value.length === 0) return
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    highlightedIndex.value = (highlightedIndex.value + 1) % filteredSuggestions.value.length
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    highlightedIndex.value =
      (highlightedIndex.value - 1 + filteredSuggestions.value.length) % filteredSuggestions.value.length
  } else if (e.key === 'Enter' && highlightedIndex.value >= 0) {
    e.preventDefault()
    selectSuggestion(filteredSuggestions.value[highlightedIndex.value])
  } else if (e.key === 'Escape') {
    showSuggestions.value = false
  }
}

// --- THRESHOLD CONFIRMATION MODAL STATE ---
const showThresholdConfirm = ref(false)
const pendingSavePayload = ref<Partial<PantryItem> | null>(null)
const thresholdDifference = ref<{
  productName: string
  existingThreshold: number
  newThreshold: number
  unit: PantryUnit
  batchesCount: number
} | null>(null)

watch(
  () => props.isOpen,
  (val) => {
    if (val) {
      errorMessage.value = null
      showSuggestions.value = false
      showThresholdConfirm.value = false
      pendingSavePayload.value = null
      thresholdDifference.value = null
      highlightedIndex.value = -1

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
    if (showThresholdConfirm.value) {
      cancelThresholdConfirm()
      return
    }
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

  const payload: Partial<PantryItem> = {
    name: trimmed,
    category: category.value,
    quantity: Number(quantity.value),
    unit: unit.value,
    min_threshold: Number(minThreshold.value),
    expiration_date: expirationDate.value ? expirationDate.value : undefined,
    location: location.value,
    cost: Number(cost.value) || 0
  }

  // If adding a food item whose name matches an existing product with a different alert threshold, confirm!
  if (!props.itemToEdit && matchedExistingProduct.value) {
    const existing = matchedExistingProduct.value
    const establishedThreshold = Number(existing.min_threshold)
    const proposedThreshold = Number(minThreshold.value)

    if (proposedThreshold !== establishedThreshold) {
      thresholdDifference.value = {
        productName: existing.name,
        existingThreshold: establishedThreshold,
        newThreshold: proposedThreshold,
        unit: existing.unit,
        batchesCount: existing.batchesCount
      }
      pendingSavePayload.value = payload
      showThresholdConfirm.value = true
      return
    }
  }

  executeSave(payload)
}

function confirmKeepExistingThreshold() {
  if (!pendingSavePayload.value || !thresholdDifference.value) return
  pendingSavePayload.value.min_threshold = thresholdDifference.value.existingThreshold
  minThreshold.value = thresholdDifference.value.existingThreshold
  showThresholdConfirm.value = false
  executeSave(pendingSavePayload.value)
}

function confirmUpdateThreshold() {
  if (!pendingSavePayload.value || !thresholdDifference.value) return
  const newThreshold = pendingSavePayload.value.min_threshold
  const productName = thresholdDifference.value.productName.toLowerCase()

  // Update other existing batches with the same product name so their thresholds stay consistent
  if (newThreshold !== undefined) {
    const otherBatches = pantryStore.items.filter(
      (it) => it.name.trim().toLowerCase() === productName && it.min_threshold !== newThreshold
    )
    for (const batch of otherBatches) {
      pantryStore.updatePantryItem(batch.id, { min_threshold: newThreshold }).catch(() => {})
    }
  }

  showThresholdConfirm.value = false
  executeSave(pendingSavePayload.value)
}

function cancelThresholdConfirm() {
  showThresholdConfirm.value = false
  pendingSavePayload.value = null
  thresholdDifference.value = null
}

function executeSave(payload: Partial<PantryItem>) {
  emit('save', payload)
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

          <!-- Name with Autocomplete -->
          <div class="form-group name-form-group">
            <div class="name-label-row">
              <label for="pantry-name" class="form-label">Nome Prodotto *</label>
              <span v-if="matchedExistingProduct && !props.itemToEdit" class="existing-indicator-pill">
                {{ matchedExistingProduct.batchesCount }} {{ matchedExistingProduct.batchesCount === 1 ? 'lotto già in dispensa' : 'lotti già in dispensa' }}
              </span>
            </div>

            <div class="autocomplete-wrap">
              <input
                id="pantry-name"
                name="pantry_name"
                v-model="name"
                type="text"
                class="form-control"
                placeholder="es. Pasta Spaghetti, Latte Intero, Mele..."
                autocomplete="off"
                required
                autofocus
                @input="onNameInput"
                @focus="onNameFocus"
                @blur="onNameBlur"
                @keydown="onNameKeydown"
                aria-autocomplete="list"
                :aria-expanded="showSuggestions && filteredSuggestions.length > 0"
              />

              <!-- Autocomplete suggestions dropdown -->
              <ul
                v-if="showSuggestions && filteredSuggestions.length > 0"
                class="autocomplete-dropdown"
                role="listbox"
              >
                <li
                  v-for="(sugg, index) in filteredSuggestions"
                  :key="sugg.name"
                  :class="['autocomplete-item', { 'is-highlighted': index === highlightedIndex }]"
                  role="option"
                  :aria-selected="index === highlightedIndex"
                  @mousedown.prevent="selectSuggestion(sugg)"
                >
                  <div class="sugg-left">
                    <span class="sugg-name">{{ sugg.name }}</span>
                    <span class="sugg-cat">{{ getCategoryLabel(sugg.category) }}</span>
                  </div>
                  <div class="sugg-right">
                    <span class="sugg-threshold">Soglia: {{ sugg.min_threshold }} {{ sugg.unit }}</span>
                    <span class="sugg-count">{{ sugg.batchesCount }} {{ sugg.batchesCount === 1 ? 'lotto' : 'lotti' }}</span>
                  </div>
                </li>
              </ul>
            </div>

            <!-- Hint banner when matching an existing item -->
            <div v-if="matchedExistingProduct && !props.itemToEdit" class="match-hint-banner">
              <span class="match-hint-icon">💡</span>
              <span class="match-hint-text">
                Alimento già presente in dispensa (<strong>{{ matchedExistingProduct.batchesCount }} {{ matchedExistingProduct.batchesCount === 1 ? 'lotto' : 'lotti' }}</strong> con scadenze diverse).
                Soglia avviso predefinita: <strong>{{ matchedExistingProduct.min_threshold }} {{ matchedExistingProduct.unit }}</strong>.
              </span>
            </div>
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
              <div class="threshold-label-row">
                <label for="pantry-threshold" class="form-label" title="Soglia Minima Avviso">Soglia Minima Avviso</label>
                <span
                  v-if="matchedExistingProduct && !props.itemToEdit && Number(minThreshold) !== Number(matchedExistingProduct.min_threshold)"
                  class="thresh-diff-badge"
                  title="Diversa dalla soglia esistente"
                >
                  ⚠️ Modificata (era {{ matchedExistingProduct.min_threshold }})
                </span>
              </div>
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

        <!-- Threshold Confirmation Modal Dialog -->
        <div
          v-if="showThresholdConfirm && thresholdDifference"
          class="threshold-confirm-overlay"
          role="alertdialog"
          aria-modal="true"
          aria-labelledby="thresh-confirm-title"
          @click.stop
        >
          <div class="threshold-confirm-card glass-panel" @click.stop>
            <div class="threshold-confirm-header">
              <div class="thresh-icon-box">
                <AlertTriangle :size="24" class="text-warning" />
              </div>
              <div class="thresh-header-text">
                <h3 id="thresh-confirm-title" class="thresh-title">Conferma Soglia di Avviso</h3>
                <p class="thresh-subtitle">Alimento già presente in dispensa</p>
              </div>
            </div>

            <div class="thresh-content">
              <p class="thresh-msg">
                Per l'alimento <strong>"{{ thresholdDifference.productName }}"</strong> è già impostata una soglia di avviso di
                <span class="badge-old-thresh">{{ thresholdDifference.existingThreshold }} {{ thresholdDifference.unit }}</span>
                ({{ thresholdDifference.batchesCount }} {{ thresholdDifference.batchesCount === 1 ? 'lotto già registrato' : 'lotti già registrati' }} con scadenze diverse).
              </p>
              <p class="thresh-question">
                Hai impostato una soglia di <span class="badge-new-thresh">{{ thresholdDifference.newThreshold }} {{ thresholdDifference.unit }}</span>.
                Sei sicuro di voler modificare la soglia di avviso per questo alimento o preferisci mantenere quella già configurata?
              </p>
            </div>

            <div class="thresh-actions">
              <button
                type="button"
                class="btn-thresh-keep"
                @click="confirmKeepExistingThreshold"
              >
                <Check :size="16" />
                <span class="btn-thresh-main">Mantieni Soglia Esistente ({{ thresholdDifference.existingThreshold }} {{ thresholdDifference.unit }})</span>
              </button>

              <button
                type="button"
                class="btn-thresh-modify"
                @click="confirmUpdateThreshold"
              >
                <span>Modifica Soglia a {{ thresholdDifference.newThreshold }} {{ thresholdDifference.unit }}</span>
              </button>

              <button
                type="button"
                class="btn-thresh-cancel"
                @click="cancelThresholdConfirm"
              >
                Annulla
              </button>
            </div>
          </div>
        </div>
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
  position: relative;
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

/* Autocomplete styling */
.name-label-row,
.threshold-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.existing-indicator-pill {
  font-size: 0.72rem;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(14, 165, 233, 0.15);
  color: #38bdf8;
  font-weight: 600;
}

.thresh-diff-badge {
  font-size: 0.72rem;
  color: #f59e0b;
  font-weight: 600;
}

.autocomplete-wrap {
  position: relative;
  width: 100%;
}

.autocomplete-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  background: var(--bg-surface-elevated, #1e293b);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.16));
  border-radius: var(--radius-md, 10px);
  max-height: 220px;
  overflow-y: auto;
  z-index: 60;
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.5);
  list-style: none;
  padding: 4px;
  margin: 4px 0 0;
}

.autocomplete-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s ease;
}

.autocomplete-item:hover,
.autocomplete-item.is-highlighted {
  background: rgba(14, 165, 233, 0.18);
}

.sugg-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sugg-name {
  font-weight: 600;
  color: var(--text-main, #f8fafc);
  font-size: 0.88rem;
}

.sugg-cat {
  font-size: 0.75rem;
  color: var(--text-muted, #94a3b8);
}

.sugg-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sugg-threshold {
  font-size: 0.72rem;
  font-weight: 600;
  padding: 2px 7px;
  border-radius: 4px;
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.sugg-count {
  font-size: 0.72rem;
  color: var(--text-dim, #64748b);
}

.match-hint-banner {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 12px;
  background: rgba(14, 165, 233, 0.08);
  border: 1px solid rgba(14, 165, 233, 0.2);
  border-radius: 8px;
  margin-top: 4px;
  font-size: 0.8rem;
  color: var(--text-main, #f8fafc);
  line-height: 1.4;
}

.match-hint-icon {
  font-size: 0.9rem;
}

/* Error Banner */
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

/* Threshold confirmation dialog overlay */
.threshold-confirm-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.78);
  backdrop-filter: blur(6px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  z-index: 100;
  border-radius: var(--radius-xl, 20px);
}

.threshold-confirm-card {
  width: 100%;
  max-width: 500px;
  background: #0f172a;
  border: 1.5px solid rgba(245, 158, 11, 0.4);
  border-radius: 14px;
  padding: 22px;
  box-shadow: 0 20px 45px rgba(0, 0, 0, 0.7);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.threshold-confirm-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.thresh-icon-box {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  background: rgba(245, 158, 11, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.thresh-title {
  font-size: 1.1rem;
  font-weight: 700;
  color: #f8fafc;
  margin: 0;
}

.thresh-subtitle {
  font-size: 0.8rem;
  color: #94a3b8;
  margin: 2px 0 0;
}

.thresh-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
  font-size: 0.88rem;
  line-height: 1.45;
  color: #e2e8f0;
}

.thresh-msg {
  margin: 0;
}

.thresh-question {
  margin: 0;
  color: #cbd5e1;
}

.badge-old-thresh {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 6px;
  background: rgba(14, 165, 233, 0.2);
  color: #38bdf8;
  font-weight: 700;
  border: 1px solid rgba(14, 165, 233, 0.4);
}

.badge-new-thresh {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 6px;
  background: rgba(245, 158, 11, 0.2);
  color: #fbbf24;
  font-weight: 700;
  border: 1px solid rgba(245, 158, 11, 0.4);
}

.thresh-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 4px;
}

.btn-thresh-keep {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: var(--primary, #0ea5e9);
  color: #ffffff;
  font-weight: 600;
  padding: 10px 16px;
  border-radius: 8px;
  border: none;
  cursor: pointer;
  font-size: 0.88rem;
  transition: background 0.2s;
}

.btn-thresh-keep:hover {
  filter: brightness(1.1);
}

.btn-thresh-modify {
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: #f8fafc;
  font-weight: 500;
  padding: 9px 16px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 0.85rem;
  transition: background 0.2s;
}

.btn-thresh-modify:hover {
  background: rgba(255, 255, 255, 0.14);
}

.btn-thresh-cancel {
  background: transparent;
  border: none;
  color: #94a3b8;
  font-size: 0.82rem;
  cursor: pointer;
  padding: 6px;
  text-align: center;
}

.btn-thresh-cancel:hover {
  color: #f8fafc;
}

/* ═══════════════════════════════════════════════════════════════════════════════
   LIGHT MODE OVERRIDES - High Contrast & WCAG 2.2 AA Compliance
   ═══════════════════════════════════════════════════════════════════════════════ */
:root[data-theme="light"] .modal-dialog,
[data-theme="light"] .modal-dialog {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  box-shadow: 0 20px 45px rgba(15, 23, 42, 0.16);
}

:root[data-theme="light"] .modal-title,
[data-theme="light"] .modal-title {
  color: #0f172a;
}

:root[data-theme="light"] .form-label,
[data-theme="light"] .form-label {
  color: #334155;
  font-weight: 600;
}

:root[data-theme="light"] .form-control,
[data-theme="light"] .form-control {
  background: #f8fafc;
  border: 1px solid #cbd5e1;
  color: #0f172a;
}

:root[data-theme="light"] .form-control:focus,
[data-theme="light"] .form-control:focus {
  background: #ffffff;
  border-color: #0284c7;
  box-shadow: 0 0 0 3px rgba(14, 165, 233, 0.18);
}

:root[data-theme="light"] .form-control::placeholder,
[data-theme="light"] .form-control::placeholder {
  color: #64748b;
  opacity: 0.9;
}

:root[data-theme="light"] select.form-control,
[data-theme="light"] select.form-control {
  color-scheme: light;
}

:root[data-theme="light"] select.form-control option,
[data-theme="light"] select.form-control option {
  background-color: #ffffff !important;
  color: #0f172a !important;
}

:root[data-theme="light"] .autocomplete-dropdown,
[data-theme="light"] .autocomplete-dropdown {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  box-shadow: 0 10px 25px rgba(15, 23, 42, 0.15);
}

:root[data-theme="light"] .autocomplete-item:hover,
[data-theme="light"] .autocomplete-item:hover,
:root[data-theme="light"] .autocomplete-item.is-highlighted,
[data-theme="light"] .autocomplete-item.is-highlighted {
  background: #f0f9ff;
}

:root[data-theme="light"] .sugg-name,
[data-theme="light"] .sugg-name {
  color: #0f172a;
}

:root[data-theme="light"] .sugg-cat,
[data-theme="light"] .sugg-cat {
  color: #475569;
}

:root[data-theme="light"] .sugg-threshold,
[data-theme="light"] .sugg-threshold {
  background: #dcfce7;
  color: #15803d;
}

:root[data-theme="light"] .sugg-count,
[data-theme="light"] .sugg-count {
  color: #64748b;
}

:root[data-theme="light"] .match-hint-banner,
[data-theme="light"] .match-hint-banner {
  background: #f0f9ff;
  border: 1px solid #bae6fd;
  color: #0369a1;
}

:root[data-theme="light"] .existing-indicator-pill,
[data-theme="light"] .existing-indicator-pill {
  background: #e0f2fe;
  color: #0284c7;
}

:root[data-theme="light"] .thresh-diff-badge,
[data-theme="light"] .thresh-diff-badge {
  color: #b45309;
}

:root[data-theme="light"] .chip-btn,
[data-theme="light"] .chip-btn {
  background: #f1f5f9;
  border: 1px solid #cbd5e1;
  color: #334155;
}

:root[data-theme="light"] .chip-btn:hover,
[data-theme="light"] .chip-btn:hover {
  background: #e2e8f0;
  color: #0f172a;
}

:root[data-theme="light"] .chip-btn.active,
[data-theme="light"] .chip-btn.active {
  background: #e0f2fe;
  border-color: #0284c7;
  color: #0284c7;
  font-weight: 600;
}

:root[data-theme="light"] .btn-secondary,
[data-theme="light"] .btn-secondary {
  background: #f1f5f9;
  border: 1px solid #cbd5e1;
  color: #334155;
}

:root[data-theme="light"] .btn-secondary:hover,
[data-theme="light"] .btn-secondary:hover {
  background: #e2e8f0;
  color: #0f172a;
}

:root[data-theme="light"] .btn-primary,
[data-theme="light"] .btn-primary {
  background: #0284c7;
  color: #ffffff;
}

:root[data-theme="light"] .btn-primary:hover,
[data-theme="light"] .btn-primary:hover {
  background: #0369a1;
}

:root[data-theme="light"] .icon-btn,
[data-theme="light"] .icon-btn {
  color: #475569;
}

:root[data-theme="light"] .icon-btn:hover,
[data-theme="light"] .icon-btn:hover {
  color: #0f172a;
  background: #f1f5f9;
}

/* Threshold Confirmation Modal in Light Mode */
:root[data-theme="light"] .threshold-confirm-card,
[data-theme="light"] .threshold-confirm-card {
  background: #ffffff;
  border: 1.5px solid #cbd5e1;
  box-shadow: 0 20px 45px rgba(15, 23, 42, 0.2);
}

:root[data-theme="light"] .thresh-title,
[data-theme="light"] .thresh-title {
  color: #0f172a;
}

:root[data-theme="light"] .thresh-subtitle,
[data-theme="light"] .thresh-subtitle {
  color: #475569;
}

:root[data-theme="light"] .thresh-msg,
[data-theme="light"] .thresh-msg {
  color: #1e293b;
}

:root[data-theme="light"] .thresh-question,
[data-theme="light"] .thresh-question {
  color: #334155;
}

:root[data-theme="light"] .badge-old-thresh,
[data-theme="light"] .badge-old-thresh {
  background: #e0f2fe;
  color: #0369a1;
  border-color: #7dd3fc;
}

:root[data-theme="light"] .badge-new-thresh,
[data-theme="light"] .badge-new-thresh {
  background: #fef3c7;
  color: #92400e;
  border-color: #fde68a;
}

:root[data-theme="light"] .btn-thresh-keep,
[data-theme="light"] .btn-thresh-keep {
  background: #0284c7;
  color: #ffffff;
}

:root[data-theme="light"] .btn-thresh-keep:hover,
[data-theme="light"] .btn-thresh-keep:hover {
  background: #0369a1;
}

:root[data-theme="light"] .btn-thresh-modify,
[data-theme="light"] .btn-thresh-modify {
  background: #f8fafc;
  border: 1px solid #cbd5e1;
  color: #0f172a;
}

:root[data-theme="light"] .btn-thresh-modify:hover,
[data-theme="light"] .btn-thresh-modify:hover {
  background: #f1f5f9;
}

:root[data-theme="light"] .btn-thresh-cancel,
[data-theme="light"] .btn-thresh-cancel {
  color: #475569;
}

:root[data-theme="light"] .btn-thresh-cancel:hover,
[data-theme="light"] .btn-thresh-cancel:hover {
  color: #0f172a;
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

