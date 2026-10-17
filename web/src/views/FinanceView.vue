<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  Wallet,
  Plus,
  PiggyBank,
  Search,
  Filter,
  Trash2,
  Edit2,
  Calendar,
  AlertCircle,
  Tag,
  CheckCircle2,
  ShoppingBag,
  Zap,
  Fuel,
  Wrench,
  Shield
} from '@lucide/vue'
import { useFinanceStore } from '@/stores/finance'
import FinanceKpiCards from '@/components/finance/FinanceKpiCards.vue'
import FinanceCategoryChart from '@/components/finance/FinanceCategoryChart.vue'
import FinanceTrendChart from '@/components/finance/FinanceTrendChart.vue'
import FinanceSavingsInsights from '@/components/finance/FinanceSavingsInsights.vue'
import ExpenseModal from '@/components/finance/ExpenseModal.vue'
import BudgetModal from '@/components/finance/BudgetModal.vue'
import type { Expense, ExpenseCategory } from '@/api/types'

const financeStore = useFinanceStore()

const isExpenseModalOpen = ref(false)
const isBudgetModalOpen = ref(false)
const expenseToEdit = ref<Expense | null>(null)
const searchQuery = ref('')

const periods: { value: 'day' | 'week' | 'month' | 'year'; label: string }[] = [
  { value: 'day', label: 'Giorno' },
  { value: 'week', label: 'Settimana' },
  { value: 'month', label: 'Mese' },
  { value: 'year', label: 'Anno' }
]

const categoriesFilterList: { category: ExpenseCategory | 'all'; label: string; icon?: any }[] = [
  { category: 'all', label: 'Tutte le Categorie' },
  { category: 'food', label: 'Cibo & Spesa', icon: ShoppingBag },
  { category: 'utilities', label: 'Bollette & Utenze', icon: Zap },
  { category: 'transport', label: 'Carburante & Auto', icon: Fuel },
  { category: 'home_maintenance', label: 'Materiali & Casa', icon: Wrench },
  { category: 'insurance', label: 'Assicurazioni', icon: Shield },
  { category: 'other', label: 'Altro / Extra', icon: Tag }
]

const filteredExpensesList = computed(() => {
  let list = financeStore.filteredExpenses
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(
      (e) =>
        e.title.toLowerCase().includes(q) ||
        (e.notes && e.notes.toLowerCase().includes(q)) ||
        (e.payment_method && e.payment_method.toLowerCase().includes(q))
    )
  }
  return list
})

onMounted(async () => {
  await Promise.allSettled([
    financeStore.fetchSummary(),
    financeStore.fetchExpenses(),
    financeStore.fetchBudgets()
  ])
})

async function onPeriodChange(p: 'day' | 'week' | 'month' | 'year') {
  await financeStore.fetchSummary(p)
}

function openCreateExpense() {
  expenseToEdit.value = null
  isExpenseModalOpen.value = true
}

function openEditExpense(exp: Expense) {
  expenseToEdit.value = exp
  isExpenseModalOpen.value = true
}

async function handleSaveExpense(payload: Partial<Expense>) {
  if (expenseToEdit.value) {
    await financeStore.updateExpense(expenseToEdit.value.id, payload)
  } else {
    await financeStore.createExpense(payload)
  }
}

async function handleDeleteExpense(id: string) {
  if (window.confirm('Sei sicuro di voler eliminare questa spesa?')) {
    await financeStore.deleteExpense(id)
  }
}

async function handleSaveBudget(limit: number) {
  await financeStore.setBudget('total', limit)
}

function formatCurrency(val: number): string {
  return new Intl.NumberFormat('it-IT', {
    style: 'currency',
    currency: 'EUR',
    minimumFractionDigits: 2
  }).format(val)
}

function formatDate(iso: string): string {
  if (!iso) return ''
  return new Intl.DateTimeFormat('it-IT', {
    day: '2-digit',
    month: 'short',
    year: 'numeric'
  }).format(new Date(iso))
}

function getCategoryIcon(cat: ExpenseCategory) {
  switch (cat) {
    case 'food':
      return ShoppingBag
    case 'utilities':
      return Zap
    case 'transport':
      return Fuel
    case 'home_maintenance':
      return Wrench
    case 'insurance':
      return Shield
    default:
      return Tag
  }
}

function getCategoryBadgeColor(cat: ExpenseCategory): string {
  switch (cat) {
    case 'food':
      return 'badge-food'
    case 'utilities':
      return 'badge-utilities'
    case 'transport':
      return 'badge-transport'
    case 'home_maintenance':
      return 'badge-maintenance'
    case 'insurance':
      return 'badge-insurance'
    default:
      return 'badge-other'
  }
}
</script>

<template>
  <div class="finance-view" role="main" aria-label="Bilancio Casa & Gestione Spese">
    <!-- Top Action Bar -->
    <header class="finance-header">
      <div class="header-left">
        <div class="title-row">
          <div class="header-icon-box">
            <Wallet :size="24" class="header-icon" aria-hidden="true" />
          </div>
          <div>
            <h1 class="page-title">Bilancio Casa & Spese</h1>
            <p class="page-subtitle">Controllo delle uscite familiari, analisi consumi e risparmio intelligente</p>
          </div>
        </div>
      </div>

      <div class="header-right">
        <!-- Period Selector -->
        <div class="period-selector-pills" role="radiogroup" aria-label="Seleziona Periodo">
          <button
            v-for="p in periods"
            :key="p.value"
            type="button"
            class="period-pill"
            :class="{ 'is-active': financeStore.selectedPeriod === p.value }"
            :aria-checked="financeStore.selectedPeriod === p.value"
            role="radio"
            @click="onPeriodChange(p.value)"
          >
            {{ p.label }}
          </button>
        </div>

        <!-- Action Buttons -->
        <div class="action-buttons">
          <button
            class="btn-budget"
            aria-label="Imposta Budget Mensile"
            @click="isBudgetModalOpen = true"
          >
            <PiggyBank :size="16" aria-hidden="true" />
            <span>Budget</span>
          </button>
          <button
            class="btn-primary-action"
            aria-label="Registra Nuova Spesa"
            @click="openCreateExpense"
          >
            <Plus :size="18" aria-hidden="true" />
            <span>Nuova Spesa</span>
          </button>
        </div>
      </div>
    </header>

    <!-- KPI Summary Cards -->
    <FinanceKpiCards
      :summary="financeStore.summary"
      :period="financeStore.selectedPeriod"
    />

    <!-- Smart Savings Insights (La Chicca di Domus) -->
    <FinanceSavingsInsights
      :insights="financeStore.savingsInsights"
    />

    <!-- Interactive Charts Grid -->
    <section class="charts-section-grid">
      <!-- 1. Donut Chart (Breakdown Categorie) -->
      <FinanceCategoryChart
        :categories="financeStore.categoryBreakdown"
        :selected-category="financeStore.selectedCategory"
        @select-category="financeStore.setCategory"
      />

      <!-- 2. 12-Month Seasonality Trend (Istogramma) -->
      <FinanceTrendChart
        :trends="financeStore.monthlyTrends"
      />
    </section>

    <!-- Recent Expenses & Transactions Section -->
    <section class="transactions-section glass-panel" role="region" aria-label="Lista Transazioni e Spese">
      <div class="section-top">
        <div>
          <h2 class="section-title">Transazioni & Spese Registrate</h2>
          <span class="section-count">{{ filteredExpensesList.length }} movimenti trovati</span>
        </div>

        <div class="filters-row">
          <!-- Category Filter Dropdown -->
          <div class="category-filter-box">
            <select
              id="category-filter-select"
              v-model="financeStore.selectedCategory"
              class="category-select"
              aria-label="Filtra per Categoria"
            >
              <option
                v-for="cat in categoriesFilterList"
                :key="cat.category"
                :value="cat.category"
              >
                {{ cat.label }}
              </option>
            </select>
          </div>

          <!-- Search input -->
          <div class="search-box">
            <Search :size="16" class="search-icon" aria-hidden="true" />
            <input
              v-model="searchQuery"
              type="text"
              class="search-input"
              placeholder="Cerca spesa o supermercato..."
              aria-label="Cerca transazione"
            />
          </div>
        </div>
      </div>

      <!-- Expenses Table / List -->
      <div v-if="filteredExpensesList.length === 0" class="no-expenses">
        <AlertCircle :size="32" class="no-exp-icon" aria-hidden="true" />
        <p class="no-exp-title">Nessuna spesa trovata</p>
        <p class="no-exp-sub">Registra la tua prima spesa o modifica i filtri di ricerca per visualizzare le transazioni.</p>
        <button class="btn-create-first" @click="openCreateExpense">
          <Plus :size="16" aria-hidden="true" />
          <span>Registra Spesa Ora</span>
        </button>
      </div>

      <div v-else class="table-container">
        <table class="expenses-table">
          <thead>
            <tr>
              <th scope="col">Data</th>
              <th scope="col">Descrizione</th>
              <th scope="col">Categoria</th>
              <th scope="col">Tipo</th>
              <th scope="col" class="text-right">Importo</th>
              <th scope="col" class="text-center">Azioni</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="exp in filteredExpensesList" :key="exp.id" class="table-row">
              <td class="date-cell">
                <Calendar :size="14" class="date-icon" aria-hidden="true" />
                <span>{{ formatDate(exp.expense_date) }}</span>
              </td>
              <td class="title-cell">
                <span class="exp-title-text">{{ exp.title }}</span>
                <span v-if="exp.notes" class="exp-notes-text">{{ exp.notes }}</span>
                <span v-if="exp.energy_kwh" class="exp-kwh-badge">
                  {{ exp.energy_kwh }} kWh ({{ (exp.amount / exp.energy_kwh).toFixed(2) }} €/kWh)
                </span>
                <span v-if="exp.contract_expiry_date" class="exp-expiry-badge">
                  Scadenza: {{ formatDate(exp.contract_expiry_date) }}
                </span>
              </td>
              <td>
                <span class="category-badge" :class="getCategoryBadgeColor(exp.category)">
                  <component :is="getCategoryIcon(exp.category)" :size="13" aria-hidden="true" />
                  <span>{{ exp.category }}</span>
                </span>
              </td>
              <td>
                <span :class="['cost-pill', exp.is_fixed ? 'fixed-pill' : 'variable-pill']">
                  {{ exp.is_fixed ? 'Fisso' : 'Variabile' }}
                </span>
              </td>
              <td class="amount-cell text-right">
                {{ formatCurrency(exp.amount) }}
              </td>
              <td class="actions-cell text-center">
                <div class="row-actions">
                  <button
                    class="action-icon-btn edit-btn"
                    :aria-label="`Modifica spesa: ${exp.title}`"
                    @click="openEditExpense(exp)"
                  >
                    <Edit2 :size="16" aria-hidden="true" />
                  </button>
                  <button
                    class="action-icon-btn delete-btn"
                    :aria-label="`Elimina spesa: ${exp.title}`"
                    @click="handleDeleteExpense(exp.id)"
                  >
                    <Trash2 :size="16" aria-hidden="true" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Modals -->
    <ExpenseModal
      :is-open="isExpenseModalOpen"
      :expense-to-edit="expenseToEdit"
      @close="isExpenseModalOpen = false"
      @save="handleSaveExpense"
    />

    <BudgetModal
      :is-open="isBudgetModalOpen"
      :current-budget="financeStore.monthlyBudget"
      @close="isBudgetModalOpen = false"
      @save="handleSaveBudget"
    />
  </div>
</template>

<style scoped>
.finance-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  padding: 1.5rem;
  max-width: 1400px;
  margin: 0 auto;
  width: 100%;
}

@media (max-width: 768px) {
  .finance-view {
    padding: 1rem;
  }
}

.finance-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 1rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-subtle);
}

.header-left {
  display: flex;
  align-items: center;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 0.85rem;
}

.header-icon-box {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  background: var(--primary-light);
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-title {
  font-size: 1.6rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
  letter-spacing: -0.02em;
}

.page-subtitle {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin: 0.2rem 0 0 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  flex-wrap: wrap;
}

.period-selector-pills {
  display: flex;
  align-items: center;
  background: rgba(255, 255, 255, 0.04);
  padding: 0.25rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
}

.period-pill {
  padding: 0.4rem 0.85rem;
  min-height: 44px;
  border: none;
  background: transparent;
  color: var(--text-muted);
  font-size: 0.8125rem;
  font-weight: 500;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.period-pill:hover {
  color: var(--text-main);
}

.period-pill.is-active {
  background: var(--primary);
  color: #fff;
  font-weight: 600;
  box-shadow: 0 2px 8px var(--primary-glow);
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-budget {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.55rem 0.95rem;
  min-height: 44px;
  border-radius: var(--radius-md);
  background: var(--color-success-bg);
  color: var(--color-success);
  border: 1px solid rgba(16, 185, 129, 0.3);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-budget:hover {
  background: var(--color-success);
  color: #fff;
}

.btn-primary-action {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.55rem 1.15rem;
  min-height: 44px;
  border-radius: var(--radius-md);
  background: var(--primary);
  color: #fff;
  border: none;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 4px 12px var(--primary-glow);
  transition: all var(--transition-fast);
}

.btn-primary-action:hover {
  background: var(--primary-hover);
  transform: translateY(-1px);
}

.charts-section-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(380px, 1fr));
  gap: 1.25rem;
  margin-bottom: 1.5rem;
}

@media (max-width: 640px) {
  .charts-section-grid {
    grid-template-columns: 1fr;
  }
}

.transactions-section {
  padding: 1.5rem;
  border-radius: var(--radius-xl);
}

.section-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 1rem;
  margin-bottom: 1.25rem;
}

.section-title {
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.section-count {
  font-size: 0.8125rem;
  color: var(--text-dim);
}

.filters-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.category-select {
  padding: 0.5rem 0.85rem;
  min-height: 44px;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  color: var(--text-main);
  font-size: 0.85rem;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0 0.85rem;
  min-height: 44px;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}

.search-icon {
  color: var(--text-dim);
}

.search-input {
  background: none;
  border: none;
  color: var(--text-main);
  font-size: 0.85rem;
  min-width: 180px;
}

.search-input:focus {
  outline: none;
}

.no-expenses {
  padding: 3rem 1.5rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.no-exp-icon {
  color: var(--text-dim);
}

.no-exp-title {
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--text-main);
  margin: 0;
}

.no-exp-sub {
  font-size: 0.85rem;
  color: var(--text-muted);
  max-width: 400px;
  margin: 0 0 1rem 0;
}

.btn-create-first {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.6rem 1.25rem;
  min-height: 44px;
  border-radius: var(--radius-md);
  background: var(--primary);
  color: #fff;
  border: none;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
}

.table-container {
  overflow-x: auto;
}

.expenses-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}

.expenses-table th {
  padding: 0.75rem 1rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-dim);
  text-transform: uppercase;
  border-bottom: 1px solid var(--border-subtle);
}

.expenses-table td {
  padding: 1rem;
  font-size: 0.875rem;
  color: var(--text-main);
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}

.table-row:hover {
  background: rgba(255, 255, 255, 0.02);
}

.date-cell {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--text-muted);
  font-size: 0.8125rem;
}

.date-icon {
  color: var(--text-dim);
}

.title-cell {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.exp-title-text {
  font-weight: 600;
  color: var(--text-main);
}

.exp-notes-text {
  font-size: 0.75rem;
  color: var(--text-dim);
}

.exp-kwh-badge {
  display: inline-block;
  font-size: 0.7rem;
  color: var(--primary);
  background: var(--primary-light);
  padding: 0.1rem 0.4rem;
  border-radius: var(--radius-sm);
  width: fit-content;
}

.exp-expiry-badge {
  display: inline-block;
  font-size: 0.7rem;
  color: var(--color-warning);
  background: var(--color-warning-bg);
  padding: 0.1rem 0.4rem;
  border-radius: var(--radius-sm);
  width: fit-content;
}

.category-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.6rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: capitalize;
}

.badge-food {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.badge-utilities {
  background: rgba(14, 165, 233, 0.15);
  color: #0ea5e9;
}

.badge-transport {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
}

.badge-maintenance {
  background: rgba(139, 92, 246, 0.15);
  color: #8b5cf6;
}

.badge-insurance {
  background: rgba(244, 63, 94, 0.15);
  color: #f43f5e;
}

.badge-other {
  background: rgba(100, 116, 139, 0.2);
  color: #94a3b8;
}

.cost-pill {
  font-size: 0.75rem;
  padding: 0.2rem 0.5rem;
  border-radius: var(--radius-full);
}

.fixed-pill {
  background: var(--color-warning-bg);
  color: var(--color-warning);
}

.variable-pill {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.amount-cell {
  font-weight: 700;
  color: var(--text-main);
  font-size: 0.95rem;
}

.text-right {
  text-align: right;
}

.text-center {
  text-align: center;
}

.row-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
}

.action-icon-btn {
  background: none;
  border: none;
  padding: 0.4rem;
  min-height: 44px;
  min-width: 44px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--text-dim);
  transition: all var(--transition-fast);
}

.edit-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-main);
}

.delete-btn:hover {
  background: var(--color-danger-bg);
  color: var(--color-danger);
}
</style>
