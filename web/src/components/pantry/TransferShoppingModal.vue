<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import {
  X,
  PackageCheck,
  Calendar,
  Clock,
  AlertCircle,
  PlusCircle,
  RefreshCw,
  Sparkles,
  Check
} from '@lucide/vue'
import type { PantryItem, PantryCategory, PantryLocation, PantryUnit } from '../../api/types'

export interface ShoppingItemForTransfer {
  id: string
  name: string
  quantity?: string
  category: string
  supermarket?: string
}

interface Props {
  isOpen: boolean
  shoppingItems: ShoppingItemForTransfer[]
  existingPantryItems: PantryItem[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'confirm', payload: {
    transfers: Array<{
      shoppingId: string
      name: string
      quantity: number
      unit: PantryUnit
      category: PantryCategory
      location: PantryLocation
      expirationDate?: string
      matchedItemId?: string
    }>
    removeTransferredFromShopping: boolean
  }): void
}>()

interface TransferDraft {
  shoppingId: string
  name: string
  quantity: number
  unit: PantryUnit
  category: PantryCategory
  location: PantryLocation
  matchedItem: PantryItem | null
  expiryMode: 'default' | 'custom' | 'none'
  defaultDays: number
  customDate: string
  selected: boolean
}

const drafts = ref<TransferDraft[]>([])
const removeAfterTransfer = ref(true)
const isSubmitting = ref(false)

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
  { key: 'pantry', label: 'Dispensa' },
  { key: 'fridge', label: 'Frigorifero' },
  { key: 'freezer', label: 'Freezer' },
  { key: 'cellar', label: 'Cantina' },
  { key: 'other', label: 'Altro' }
]

const units: PantryUnit[] = ['pz', 'kg', 'g', 'l', 'ml', 'pack']

function parseQtyAndUnit(str?: string): { quantity: number; unit: PantryUnit } {
  if (!str || !str.trim()) return { quantity: 1, unit: 'pz' }
  const clean = str.trim().toLowerCase().replace(',', '.')
  const match = clean.match(/^([\d\.]+)\s*([a-z]+)?$/)
  if (match) {
    const qty = parseFloat(match[1]) || 1
    const rawUnit = match[2] || ''
    if (rawUnit === 'kg' || rawUnit === 'chili') return { quantity: qty, unit: 'kg' }
    if (rawUnit === 'g' || rawUnit === 'gr' || rawUnit === 'grammi') return { quantity: qty, unit: 'g' }
    if (rawUnit === 'l' || rawUnit === 'litri' || rawUnit === 'lt') return { quantity: qty, unit: 'l' }
    if (rawUnit === 'ml') return { quantity: qty, unit: 'ml' }
    if (rawUnit === 'pack' || rawUnit === 'pacco' || rawUnit === 'pacchi' || rawUnit === 'conf') return { quantity: qty, unit: 'pack' }
    return { quantity: qty, unit: 'pz' }
  }
  return { quantity: 1, unit: 'pz' }
}

function mapShoppingCategoryToPantry(cat: string): PantryCategory {
  switch (cat.toLowerCase()) {
    case 'frutta_verdura':
    case 'ortofrutta':
      return 'produce'
    case 'latticini':
    case 'freschi':
      return 'dairy'
    case 'bevande':
      return 'drinks'
    case 'igiene':
    case 'casa':
      return 'household'
    case 'alimentari':
      return 'pantry'
    default:
      return 'other'
  }
}

function getDefaultShelfLifeDays(cat: PantryCategory): number {
  switch (cat) {
    case 'produce':
      return 5 // Ortofrutta: 5 giorni
    case 'dairy':
      return 7 // Latticini & Freschi: 7 giorni
    case 'meat_fish':
      return 3 // Carne e Pesce: 3 giorni
    case 'bakery':
      return 4 // Pane: 4 giorni
    case 'drinks':
      return 30
    case 'pantry':
      return 180 // Secco: 6 mesi
    default:
      return 14
  }
}

function mapCategoryToLocation(cat: PantryCategory): PantryLocation {
  if (cat === 'produce' || cat === 'dairy' || cat === 'meat_fish') {
    return 'fridge'
  }
  if (cat === 'frozen') {
    return 'freezer'
  }
  return 'pantry'
}

function calculateDate(days: number): string {
  const d = new Date()
  d.setDate(d.getDate() + days)
  return d.toISOString().split('T')[0]
}

function findMatchingPantryItem(name: string, items: PantryItem[]): PantryItem | null {
  const target = name.trim().toLowerCase()
  if (!target) return null
  return (
    items.find((p) => {
      const pName = p.name.trim().toLowerCase()
      return pName === target || pName.includes(target) || target.includes(pName)
    }) || null
  )
}

watch(
  () => props.isOpen,
  (val) => {
    if (val) {
      isSubmitting.value = false
      drafts.value = props.shoppingItems.map((item) => {
        const parsed = parseQtyAndUnit(item.quantity)
        const matched = findMatchingPantryItem(item.name, props.existingPantryItems)
        const cat = matched ? matched.category : mapShoppingCategoryToPantry(item.category)
        const loc = matched ? matched.location : mapCategoryToLocation(cat)
        const defDays = getDefaultShelfLifeDays(cat)

        return {
          shoppingId: item.id,
          name: item.name,
          quantity: parsed.quantity,
          unit: matched ? matched.unit : parsed.unit,
          category: cat,
          location: loc,
          matchedItem: matched,
          expiryMode: cat === 'household' ? 'none' : 'default',
          defaultDays: defDays,
          customDate: calculateDate(defDays),
          selected: true
        }
      })
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

function getComputedExpiry(draft: TransferDraft): string | undefined {
  if (draft.expiryMode === 'none') return undefined
  if (draft.expiryMode === 'custom') return draft.customDate || undefined
  return calculateDate(draft.defaultDays)
}

function setGlobalDays(days: number) {
  for (const d of drafts.value) {
    if (d.category !== 'household') {
      d.expiryMode = 'default'
      d.defaultDays = days
      d.customDate = calculateDate(days)
    }
  }
}

const selectedCount = computed(() => drafts.value.filter((d) => d.selected).length)

function handleConfirm() {
  const selectedDrafts = drafts.value.filter((d) => d.selected)
  if (selectedDrafts.length === 0) return

  isSubmitting.value = true

  const transfers = selectedDrafts.map((d) => ({
    shoppingId: d.shoppingId,
    name: d.name.trim(),
    quantity: Number(d.quantity) || 1,
    unit: d.unit,
    category: d.category,
    location: d.location,
    expirationDate: getComputedExpiry(d),
    matchedItemId: d.matchedItem ? d.matchedItem.id : undefined
  }))

  emit('confirm', {
    transfers,
    removeTransferredFromShopping: removeAfterTransfer.value
  })
}
</script>

<template>
  <Teleport to="body">
    <div v-if="isOpen" class="modal-backdrop" @click="emit('close')">
      <div
        class="modal-dialog glass-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="transfer-pantry-title"
        @click.stop
      >
        <!-- Header -->
        <div class="modal-header">
          <div class="header-title-box">
            <PackageCheck :size="24" class="text-success" />
            <div>
              <h2 id="transfer-pantry-title" class="modal-title">
                Fatta Spesa! Carica in Dispensa
              </h2>
              <p class="modal-subtitle">
                Seleziona i prodotti comprati per incrementare o inserire le scorte in dispensa
              </p>
            </div>
          </div>
          <button
            type="button"
            class="icon-btn"
            aria-label="Chiudi finestra"
            @click="emit('close')"
          >
            <X :size="20" />
          </button>
        </div>

        <!-- Global Batch Shortcuts -->
        <div class="global-shortcuts-bar">
          <div class="shortcuts-intro">
            <Sparkles :size="16" class="text-primary" />
            <span>Imposta scadenza rapida per tutti i deperibili:</span>
          </div>
          <div class="quick-days-pills">
            <button type="button" class="pill-btn" @click="setGlobalDays(3)">
              +3gg (Carne/Pesce)
            </button>
            <button type="button" class="pill-btn" @click="setGlobalDays(5)">
              +5gg (Ortofrutta)
            </button>
            <button type="button" class="pill-btn" @click="setGlobalDays(7)">
              +7gg (Latticini)
            </button>
            <button type="button" class="pill-btn" @click="setGlobalDays(14)">
              +14gg
            </button>
            <button type="button" class="pill-btn" @click="setGlobalDays(30)">
              +1 mese
            </button>
          </div>
        </div>

        <!-- Items Table / List -->
        <div class="modal-body scrollable">
          <div v-if="drafts.length === 0" class="empty-state">
            Nessun articolo selezionato dalla lista della spesa.
          </div>

          <div v-else class="items-transfer-list">
            <div
              v-for="d in drafts"
              :key="d.shoppingId"
              class="transfer-card"
              :class="{ 'is-matched': !!d.matchedItem, 'is-selected': d.selected }"
            >
              <!-- Checkbox & Product Name -->
              <div class="transfer-card-header">
                <label class="select-item-label">
                  <input v-model="d.selected" type="checkbox" />
                  <span class="product-name">{{ d.name }}</span>
                </label>

                <!-- Status Badge: Existing (Summed) vs New -->
                <div v-if="d.matchedItem" class="badge-matched" title="Alimento già presente in dispensa: la quantità verrà sommata">
                  <RefreshCw :size="13" />
                  <span>Si somma a "{{ d.matchedItem.name }}"</span>
                </div>
                <div v-else class="badge-new" title="Nuovo alimento: verrà aggiunto all'inventario della dispensa">
                  <PlusCircle :size="13" />
                  <span>Nuovo in Dispensa</span>
                </div>
              </div>

              <!-- Match calculation detail -->
              <div v-if="d.matchedItem" class="match-calc-note">
                Giacenza attuale: <strong>{{ d.matchedItem.quantity }} {{ d.matchedItem.unit }}</strong>
                + Spesa: <strong>{{ d.quantity }} {{ d.unit }}</strong>
                = Nuovo totale: <strong class="text-success">{{ (Number(d.matchedItem.quantity) + Number(d.quantity)).toFixed(1).replace('.0', '') }} {{ d.unit }}</strong>
              </div>

              <!-- Quantity, Unit, Category and Storage Row -->
              <div class="transfer-row-controls">
                <div class="control-col qty-col">
                  <label class="ctrl-label">Quantità</label>
                  <div class="qty-unit-wrap">
                    <input
                      v-model.number="d.quantity"
                      type="number"
                      step="any"
                      min="0.1"
                      class="form-control ctrl-input"
                    />
                    <select v-model="d.unit" class="form-control ctrl-select">
                      <option v-for="u in units" :key="u" :value="u">{{ u }}</option>
                    </select>
                  </div>
                </div>

                <div class="control-col">
                  <label class="ctrl-label">Categoria</label>
                  <select v-model="d.category" class="form-control ctrl-select">
                    <option v-for="c in categories" :key="c.key" :value="c.key">
                      {{ c.label }}
                    </option>
                  </select>
                </div>

                <div class="control-col">
                  <label class="ctrl-label">Posizione</label>
                  <select v-model="d.location" class="form-control ctrl-select">
                    <option v-for="l in locations" :key="l.key" :value="l.key">
                      {{ l.label }}
                    </option>
                  </select>
                </div>
              </div>

              <!-- Expiry date selector & shelf life -->
              <div class="expiry-section">
                <div class="expiry-header">
                  <Clock :size="14" class="text-primary" />
                  <span class="expiry-title">Scadenza:</span>

                  <div class="expiry-mode-radios">
                    <label class="radio-label">
                      <input v-model="d.expiryMode" type="radio" value="default" />
                      <span>Predefinita (+{{ d.defaultDays }}gg)</span>
                    </label>
                    <label class="radio-label">
                      <input v-model="d.expiryMode" type="radio" value="custom" />
                      <span>Data specifica</span>
                    </label>
                    <label class="radio-label">
                      <input v-model="d.expiryMode" type="radio" value="none" />
                      <span>Nessuna</span>
                    </label>
                  </div>
                </div>

                <!-- Custom Date Input or Quick Presets -->
                <div v-if="d.expiryMode === 'default'" class="expiry-details">
                  <span class="calc-date-badge">
                    Scadenza: <strong>{{ calculateDate(d.defaultDays) }}</strong>
                  </span>
                  <div class="chip-days-row">
                    <button
                      type="button"
                      class="chip-day-btn"
                      :class="{ active: d.defaultDays === 3 }"
                      @click="d.defaultDays = 3"
                    >
                      +3gg
                    </button>
                    <button
                      type="button"
                      class="chip-day-btn"
                      :class="{ active: d.defaultDays === 5 }"
                      @click="d.defaultDays = 5"
                    >
                      +5gg
                    </button>
                    <button
                      type="button"
                      class="chip-day-btn"
                      :class="{ active: d.defaultDays === 7 }"
                      @click="d.defaultDays = 7"
                    >
                      +7gg
                    </button>
                    <button
                      type="button"
                      class="chip-day-btn"
                      :class="{ active: d.defaultDays === 14 }"
                      @click="d.defaultDays = 14"
                    >
                      +14gg
                    </button>
                    <button
                      type="button"
                      class="chip-day-btn"
                      :class="{ active: d.defaultDays === 30 }"
                      @click="d.defaultDays = 30"
                    >
                      +30gg
                    </button>
                    <button
                      type="button"
                      class="chip-day-btn"
                      :class="{ active: d.defaultDays === 180 }"
                      @click="d.defaultDays = 180"
                    >
                      +6 mesi
                    </button>
                  </div>
                </div>

                <div v-else-if="d.expiryMode === 'custom'" class="expiry-details">
                  <input
                    v-model="d.customDate"
                    type="date"
                    class="form-control date-input"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="modal-footer">
          <label class="remove-after-checkbox">
            <input v-model="removeAfterTransfer" type="checkbox" />
            <span>Rimuovi gli articoli trasferiti dalla lista della spesa</span>
          </label>

          <div class="footer-buttons">
            <button type="button" class="btn btn-secondary" @click="emit('close')">
              Annulla
            </button>
            <button
              type="button"
              class="btn btn-primary"
              :disabled="selectedCount === 0 || isSubmitting"
              @click="handleConfirm"
            >
              <PackageCheck :size="18" />
              <span>Conferma e Trasferisci ({{ selectedCount }})</span>
            </button>
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
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 16px;
}

.modal-dialog {
  width: 95vw;
  max-width: 780px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  background: var(--surface-card, #1e293b);
  border: 1.5px solid var(--border-color, rgba(255, 255, 255, 0.15));
  border-radius: var(--radius-xl, 20px);
  box-shadow: 0 25px 60px rgba(0, 0, 0, 0.6);
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

.header-title-box {
  display: flex;
  align-items: center;
  gap: 12px;
}

.modal-title {
  font-size: 1.2rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
  margin: 0;
}

.modal-subtitle {
  font-size: 0.82rem;
  color: var(--text-muted, #94a3b8);
  margin: 2px 0 0 0;
}

.global-shortcuts-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 24px;
  background: rgba(14, 165, 233, 0.08);
  border-bottom: 1px solid rgba(14, 165, 233, 0.2);
  flex-wrap: wrap;
  flex-shrink: 0;
}

.shortcuts-intro {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.82rem;
  color: var(--primary, #38bdf8);
  font-weight: 500;
}

.quick-days-pills {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.pill-btn {
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 0.75rem;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: var(--text-main, #f8fafc);
  cursor: pointer;
  transition: all 0.15s;
}
.pill-btn:hover {
  background: var(--primary, #0ea5e9);
  color: #fff;
  border-color: var(--primary, #0ea5e9);
}

.modal-body.scrollable {
  padding: 16px 24px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-y: auto;
  overflow-x: hidden;
}

.empty-state {
  text-align: center;
  padding: 40px 16px;
  color: var(--text-muted, #94a3b8);
}

.items-transfer-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.transfer-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-lg, 14px);
  transition: border-color 0.2s;
}
.transfer-card.is-matched {
  border-color: rgba(16, 185, 129, 0.35);
  background: rgba(16, 185, 129, 0.03);
}

.transfer-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.select-item-label {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  user-select: none;
}
.select-item-label input[type="checkbox"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
}

.product-name {
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
}

.badge-matched {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 3px 8px;
  border-radius: 6px;
  background: rgba(16, 185, 129, 0.15);
  color: var(--color-success, #10b981);
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.badge-new {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 3px 8px;
  border-radius: 6px;
  background: rgba(59, 130, 246, 0.15);
  color: var(--primary, #3b82f6);
  border: 1px solid rgba(59, 130, 246, 0.3);
}

.match-calc-note {
  font-size: 0.8rem;
  color: var(--text-muted, #94a3b8);
  background: rgba(0, 0, 0, 0.2);
  padding: 6px 10px;
  border-radius: 6px;
  border-left: 3px solid var(--color-success, #10b981);
}

.transfer-row-controls {
  display: grid;
  grid-template-columns: 180px 1fr 1fr;
  gap: 10px;
}

.control-col {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ctrl-label {
  font-size: 0.75rem;
  color: var(--text-muted, #94a3b8);
  font-weight: 500;
}

.qty-unit-wrap {
  display: grid;
  grid-template-columns: 1fr 80px;
  gap: 6px;
}

.form-control {
  height: 38px;
  padding: 0 10px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: var(--text-main, #f8fafc);
  font-size: 0.88rem;
  min-width: 0;
  width: 100%;
  box-sizing: border-box;
}

/* Dropdown readability in dark mode */
select.form-control,
.ctrl-select {
  color-scheme: dark;
}

select.form-control option,
.ctrl-select option {
  background-color: #1e293b !important;
  color: #f8fafc !important;
}

.expiry-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
  background: rgba(0, 0, 0, 0.15);
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.expiry-header {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.expiry-title {
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
}

.expiry-mode-radios {
  display: flex;
  align-items: center;
  gap: 12px;
}

.radio-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.75rem;
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
}

.expiry-details {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  padding-top: 4px;
}

.calc-date-badge {
  font-size: 0.78rem;
  color: var(--primary, #38bdf8);
  background: rgba(14, 165, 233, 0.12);
  padding: 3px 8px;
  border-radius: 6px;
  border: 1px solid rgba(14, 165, 233, 0.25);
}

.chip-days-row {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.chip-day-btn {
  padding: 2px 7px;
  border-radius: 6px;
  font-size: 0.72rem;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
}
.chip-day-btn:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.1);
}
.chip-day-btn.active {
  background: rgba(14, 165, 233, 0.25);
  border-color: var(--primary, #0ea5e9);
  color: #38bdf8;
  font-weight: 600;
}

.date-input {
  max-width: 180px;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  border-top: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  gap: 16px;
  flex-wrap: wrap;
}

.remove-after-checkbox {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.82rem;
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
}
.remove-after-checkbox input {
  width: 16px;
  height: 16px;
}

.footer-buttons {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-left: auto;
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 42px;
  padding: 0 18px;
  border-radius: var(--radius-md, 10px);
  font-size: 0.88rem;
  font-weight: 600;
  cursor: pointer;
  border: none;
  transition: all 0.2s;
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-main, #f8fafc);
}
.btn-primary {
  background: var(--color-success, #10b981);
  color: #000;
  font-weight: 700;
}
.btn-primary:hover:not(:disabled) {
  background: #059669;
  color: #fff;
}
.btn-primary:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  background: transparent;
  border: none;
  color: var(--text-muted, #94a3b8);
  border-radius: 50%;
  cursor: pointer;
}

@media (max-width: 640px) {
  .transfer-row-controls {
    grid-template-columns: 1fr;
  }
  .global-shortcuts-bar {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
