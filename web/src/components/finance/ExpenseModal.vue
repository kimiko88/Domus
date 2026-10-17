<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { X, Check, ShoppingBag, Zap, Fuel, Wrench, Shield, Tag } from '@lucide/vue'
import type { Expense, ExpenseCategory, ExpenseSubcategory } from '@/api/types'

const props = defineProps<{
  isOpen: boolean
  expenseToEdit?: Expense | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', payload: Partial<Expense>): void
}>()

const categoriesConfig: {
  category: ExpenseCategory
  label: string
  icon: any
  subcategories: { value: ExpenseSubcategory; label: string }[]
}[] = [
  {
    category: 'food',
    label: 'Cibo & Spesa',
    icon: ShoppingBag,
    subcategories: [
      { value: 'groceries', label: 'Supermercato' },
      { value: 'bakery', label: 'Panetteria' },
      { value: 'butcher', label: 'Macelleria' },
      { value: 'restaurant_takeaway', label: 'Ristorante / Asporto' }
    ]
  },
  {
    category: 'utilities',
    label: 'Bollette & Utenze',
    icon: Zap,
    subcategories: [
      { value: 'electricity', label: 'Elettricità' },
      { value: 'gas_heating', label: 'Gas / Riscaldamento' },
      { value: 'water', label: 'Acqua' },
      { value: 'waste_tari', label: 'Rifiuti (TARI)' },
      { value: 'telecom_fiber', label: 'Fibra / Telefono' }
    ]
  },
  {
    category: 'transport',
    label: 'Carburante & Auto',
    icon: Fuel,
    subcategories: [
      { value: 'fuel_petrol', label: 'Benzina' },
      { value: 'fuel_diesel', label: 'Diesel' },
      { value: 'ev_charging', label: 'Ricarica EV' },
      { value: 'toll_telepass', label: 'Telepass / Pedaggi' },
      { value: 'vehicle_maintenance', label: 'Manutenzione Auto' },
      { value: 'vehicle_tax_bollo', label: 'Bollo Auto' }
    ]
  },
  {
    category: 'home_maintenance',
    label: 'Materiali & Casa',
    icon: Wrench,
    subcategories: [
      { value: 'diy_hardware', label: 'Fai-da-te / Ferramenta' },
      { value: 'appliances', label: 'Elettrodomestici' },
      { value: 'furniture_decor', label: 'Arredamento' },
      { value: 'repairs', label: 'Riparazioni' }
    ]
  },
  {
    category: 'insurance',
    label: 'Assicurazioni',
    icon: Shield,
    subcategories: [
      { value: 'auto_insurance', label: 'Polizza Auto' },
      { value: 'home_insurance', label: 'Casa / Incendio' },
      { value: 'health_insurance', label: 'Sanitaria' },
      { value: 'life_insurance', label: 'Vita' }
    ]
  },
  {
    category: 'other',
    label: 'Altro / Extra',
    icon: Tag,
    subcategories: [
      { value: 'health_pharmacy', label: 'Salute / Farmacia' },
      { value: 'education', label: 'Istruzione' },
      { value: 'leisure_dining', label: 'Svago / Ristoranti' },
      { value: 'clothing', label: 'Abbigliamento' },
      { value: 'miscellaneous', label: 'Varie' }
    ]
  }
]

// Form State
const title = ref('')
const amount = ref<number | ''>('')
const category = ref<ExpenseCategory>('food')
const subcategory = ref<ExpenseSubcategory>('groceries')
const expenseDate = ref(new Date().toISOString().substring(0, 10))
const isFixed = ref(false)
const paymentMethod = ref('card')
const contractExpiryDate = ref('')
const energyKWh = ref<number | ''>('')
const gasSmc = ref<number | ''>('')
const foodWasteValue = ref<number | ''>('')
const notes = ref('')

const currentSubcategories = computed(() => {
  const cat = categoriesConfig.find((c) => c.category === category.value)
  return cat ? cat.subcategories : []
})

watch(
  category,
  (newCat) => {
    const cfg = categoriesConfig.find((c) => c.category === newCat)
    if (cfg && cfg.subcategories.length > 0) {
      // Default to first subcategory if current not in list
      if (!cfg.subcategories.some((s) => s.value === subcategory.value)) {
        subcategory.value = cfg.subcategories[0].value
      }
    }
    // Auto toggle fixed for typical fixed expenses
    if (newCat === 'insurance') {
      isFixed.value = true
    }
  }
)

watch(
  () => props.expenseToEdit,
  (exp) => {
    if (exp) {
      title.value = exp.title
      amount.value = exp.amount
      category.value = exp.category
      subcategory.value = exp.subcategory || 'groceries'
      expenseDate.value = exp.expense_date ? exp.expense_date.substring(0, 10) : new Date().toISOString().substring(0, 10)
      isFixed.value = exp.is_fixed
      paymentMethod.value = exp.payment_method || 'card'
      contractExpiryDate.value = exp.contract_expiry_date ? exp.contract_expiry_date.substring(0, 10) : ''
      energyKWh.value = exp.energy_kwh ?? ''
      gasSmc.value = exp.gas_smc ?? ''
      foodWasteValue.value = exp.food_waste_value ?? ''
      notes.value = exp.notes || ''
    } else {
      resetForm()
    }
  },
  { immediate: true }
)

function resetForm() {
  title.value = ''
  amount.value = ''
  category.value = 'food'
  subcategory.value = 'groceries'
  expenseDate.value = new Date().toISOString().substring(0, 10)
  isFixed.value = false
  paymentMethod.value = 'card'
  contractExpiryDate.value = ''
  energyKWh.value = ''
  gasSmc.value = ''
  foodWasteValue.value = ''
  notes.value = ''
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.isOpen) {
    emit('close')
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
})

function submitForm() {
  if (!title.value.trim() || !amount.value || Number(amount.value) <= 0) {
    return
  }

  const payload: Partial<Expense> = {
    title: title.value.trim(),
    amount: Number(amount.value),
    currency: 'EUR',
    category: category.value,
    subcategory: subcategory.value,
    expense_date: new Date(expenseDate.value).toISOString(),
    is_fixed: isFixed.value,
    payment_method: paymentMethod.value,
    notes: notes.value.trim() || undefined
  }

  if (contractExpiryDate.value) {
    payload.contract_expiry_date = new Date(contractExpiryDate.value).toISOString()
  }
  if (energyKWh.value && Number(energyKWh.value) > 0) {
    payload.energy_kwh = Number(energyKWh.value)
  }
  if (gasSmc.value && Number(gasSmc.value) > 0) {
    payload.gas_smc = Number(gasSmc.value)
  }
  if (foodWasteValue.value && Number(foodWasteValue.value) > 0) {
    payload.food_waste_value = Number(foodWasteValue.value)
  }

  emit('save', payload)
  emit('close')
}
</script>

<template>
  <div v-if="isOpen" class="modal-backdrop" @click="emit('close')" aria-hidden="true">
    <div
      class="modal-dialog glass-panel"
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-expense-title"
      @click.stop
    >
      <div class="modal-header">
        <h3 id="modal-expense-title" class="modal-title">
          {{ expenseToEdit ? 'Modifica Spesa' : 'Registra Nuova Spesa' }}
        </h3>
        <button
          class="close-btn"
          aria-label="Chiudi finestra"
          @click="emit('close')"
        >
          <X :size="20" aria-hidden="true" />
        </button>
      </div>

      <form class="modal-form" @submit.prevent="submitForm">
        <!-- Category selector pills -->
        <div class="form-group">
          <label class="form-label">Categoria</label>
          <div class="category-pills" role="radiogroup" aria-label="Seleziona Categoria">
            <button
              v-for="cat in categoriesConfig"
              :key="cat.category"
              type="button"
              class="cat-pill-btn"
              :class="{ 'is-selected': category === cat.category }"
              :aria-checked="category === cat.category"
              role="radio"
              @click="category = cat.category"
            >
              <component :is="cat.icon" :size="16" aria-hidden="true" />
              <span>{{ cat.label }}</span>
            </button>
          </div>
        </div>

        <!-- Subcategory selector -->
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="exp-subcategory" class="form-label">Sottocategoria</label>
            <select id="exp-subcategory" v-model="subcategory" class="form-input form-select">
              <option
                v-for="sub in currentSubcategories"
                :key="sub.value"
                :value="sub.value"
              >
                {{ sub.label }}
              </option>
            </select>
          </div>

          <div class="form-group flex-1">
            <label for="exp-payment" class="form-label">Metodo di Pagamento</label>
            <select id="exp-payment" v-model="paymentMethod" class="form-input form-select">
              <option value="card">Carta di Credito / Bancomat</option>
              <option value="cash">Contanti</option>
              <option value="bank_transfer">Bonifico Bancario</option>
              <option value="direct_debit">RID / Addebito Diretto</option>
            </select>
          </div>
        </div>

        <!-- Title & Amount -->
        <div class="form-row">
          <div class="form-group flex-2">
            <label for="exp-title" class="form-label">Descrizione / Titolo *</label>
            <input
              id="exp-title"
              v-model="title"
              type="text"
              required
              class="form-input"
              placeholder="es. Spesa settimanale, Bolletta Enel..."
            />
          </div>

          <div class="form-group flex-1">
            <label for="exp-amount" class="form-label">Importo (€) *</label>
            <input
              id="exp-amount"
              v-model="amount"
              type="number"
              step="0.01"
              min="0.01"
              required
              class="form-input amount-input"
              placeholder="0,00"
            />
          </div>
        </div>

        <!-- Date & Fixed switch -->
        <div class="form-row align-center">
          <div class="form-group flex-1">
            <label for="exp-date" class="form-label">Data Spesa</label>
            <input id="exp-date" v-model="expenseDate" type="date" class="form-input" />
          </div>

          <div class="form-group flex-1 switch-group">
            <label class="form-label">Tipo di Costo</label>
            <button
              type="button"
              class="cost-type-toggle"
              :class="{ 'is-fixed': isFixed }"
              role="switch"
              :aria-checked="isFixed"
              @click="isFixed = !isFixed"
            >
              <span class="toggle-pill">{{ isFixed ? 'Costo Fisso' : 'Costo Variabile' }}</span>
              <span class="toggle-hint">
                {{ isFixed ? 'Incomprimibile (mutuo, assicurazioni, canoni)' : 'Flessibile (spesa, consumi, svago)' }}
              </span>
            </button>
          </div>
        </div>

        <!-- Contextual: Energy kWh for Utilities -->
        <div v-if="category === 'utilities'" class="form-row contextual-box">
          <div class="form-group flex-1">
            <label for="exp-kwh" class="form-label">Energia Fatturata (kWh) — Correlazione Domus</label>
            <input
              id="exp-kwh"
              v-model="energyKWh"
              type="number"
              step="1"
              min="0"
              class="form-input"
              placeholder="es. 360 kWh per calcolo tariffa"
            />
          </div>
          <div class="form-group flex-1">
            <label for="exp-smc" class="form-label">Gas Consumato (Smc)</label>
            <input
              id="exp-smc"
              v-model="gasSmc"
              type="number"
              step="1"
              min="0"
              class="form-input"
              placeholder="es. 85 Smc"
            />
          </div>
        </div>

        <!-- Contextual: Insurance or Contracts Expiry -->
        <div v-if="category === 'insurance' || isFixed" class="form-group contextual-box">
          <label for="exp-contract-expiry" class="form-label">Scadenza Contratto (Promemoria 30gg prima)</label>
          <input
            id="exp-contract-expiry"
            v-model="contractExpiryDate"
            type="date"
            class="form-input"
          />
          <span class="input-tip">Domus ti avviserà 30 giorni prima per consentirti di rinegoziare la tariffa.</span>
        </div>

        <!-- Contextual: Food waste tracking -->
        <div v-if="category === 'food'" class="form-group contextual-box">
          <label for="exp-food-waste" class="form-label">Di cui Cibo Scaduto / Buttato (€)</label>
          <input
            id="exp-food-waste"
            v-model="foodWasteValue"
            type="number"
            step="0.01"
            min="0"
            class="form-input"
            placeholder="0,00 € (calcolo spreco alimentare monetizzato)"
          />
        </div>

        <!-- Notes -->
        <div class="form-group">
          <label for="exp-notes" class="form-label">Note Aggiuntive</label>
          <textarea
            id="exp-notes"
            v-model="notes"
            rows="2"
            class="form-input form-textarea"
            placeholder="Dettagli sullo scontrino o acquisto..."
          />
        </div>

        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="emit('close')">
            Annulla
          </button>
          <button type="submit" class="btn btn-primary">
            <Check :size="16" aria-hidden="true" />
            <span>{{ expenseToEdit ? 'Aggiorna Spesa' : 'Salva Spesa' }}</span>
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: var(--bg-overlay);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-dialog {
  width: 100%;
  max-width: 600px;
  max-height: 90vh;
  overflow-y: auto;
  border-radius: var(--radius-xl);
  padding: 1.5rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-card);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.5);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-title {
  font-size: 1.2rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 0.4rem;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 44px;
  min-width: 44px;
  transition: color var(--transition-fast);
}

.close-btn:hover {
  color: var(--text-main);
}

.modal-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.form-row {
  display: flex;
  gap: 0.75rem;
}

@media (max-width: 520px) {
  .form-row {
    flex-direction: column;
  }
}

.flex-1 {
  flex: 1;
}

.flex-2 {
  flex: 2;
}

.align-center {
  align-items: center;
}

.form-label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--text-muted);
}

.category-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.cat-pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.45rem 0.75rem;
  min-height: 44px;
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.cat-pill-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-main);
}

.cat-pill-btn.is-selected {
  background: var(--primary-light);
  border-color: var(--primary);
  color: var(--primary);
  font-weight: 600;
}

.form-input {
  width: 100%;
  padding: 0.65rem 0.85rem;
  min-height: 44px;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  color: var(--text-main);
  font-size: 0.9rem;
  transition: border-color var(--transition-fast);
}

.form-input:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 2px var(--primary-light);
}

.amount-input {
  font-weight: 700;
  color: var(--color-success);
}

.form-textarea {
  min-height: 60px;
  resize: vertical;
}

.switch-group {
  justify-content: flex-end;
}

.cost-type-toggle {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  padding: 0.5rem 0.75rem;
  min-height: 44px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  cursor: pointer;
  text-align: left;
  transition: all var(--transition-fast);
}

.cost-type-toggle.is-fixed {
  background: var(--color-warning-bg);
  border-color: var(--color-warning);
}

.toggle-pill {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-main);
}

.cost-type-toggle.is-fixed .toggle-pill {
  color: var(--color-warning);
}

.toggle-hint {
  font-size: 0.7rem;
  color: var(--text-dim);
}

.contextual-box {
  padding: 0.75rem;
  background: rgba(14, 165, 233, 0.05);
  border: 1px dashed rgba(14, 165, 233, 0.25);
  border-radius: var(--radius-md);
}

.input-tip {
  font-size: 0.75rem;
  color: var(--text-dim);
  margin-top: 0.2rem;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-subtle);
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.65rem 1.25rem;
  min-height: 44px;
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  border: none;
  transition: opacity var(--transition-fast);
}

.btn-primary {
  background: var(--primary);
  color: #fff;
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-muted);
}

.btn-secondary:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-main);
}
</style>
