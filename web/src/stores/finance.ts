import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '../api/client'
import type {
  Expense,
  ExpenseCategory,
  BudgetTarget,
  FinancialSummary,
  SavingsInsight,
  CategoryBreakdown,
  MonthlyTrend,
  FixedVsVariable
} from '../api/types'

export const useFinanceStore = defineStore('finance', () => {
  const summary = ref<FinancialSummary | null>(null)
  const expenses = ref<Expense[]>([])
  const budgets = ref<BudgetTarget[]>([])
  const selectedPeriod = ref<'day' | 'week' | 'month' | 'year'>('month')
  const selectedCategory = ref<ExpenseCategory | 'all'>('all')
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Computed
  const totalSpent = computed(() => summary.value?.total_spent ?? 0)
  const monthlyBudget = computed(() => summary.value?.monthly_budget ?? 0)
  const remainingBudget = computed(() => summary.value?.remaining_budget ?? 0)
  const percentageChange = computed(() => summary.value?.percentage_change ?? 0)
  const dailyAverage = computed(() => summary.value?.daily_average ?? 0)

  const budgetUsagePercent = computed(() => {
    if (!monthlyBudget.value || monthlyBudget.value <= 0) return 0
    return Math.min(Math.round((totalSpent.value / monthlyBudget.value) * 100), 100)
  })

  const isOverBudget = computed(() => {
    return monthlyBudget.value > 0 && totalSpent.value > monthlyBudget.value
  })

  const categoryBreakdown = computed<CategoryBreakdown[]>(() => {
    return summary.value?.categories ?? []
  })

  const monthlyTrends = computed<MonthlyTrend[]>(() => {
    return summary.value?.monthly_trends ?? []
  })

  const fixedVsVariable = computed<FixedVsVariable>(() => {
    return (
      summary.value?.fixed_vs_variable ?? {
        fixed_amount: 0,
        variable_amount: 0,
        fixed_percentage: 0,
        variable_percentage: 0
      }
    )
  })

  const savingsInsights = computed<SavingsInsight[]>(() => {
    return summary.value?.insights ?? []
  })

  const filteredExpenses = computed(() => {
    if (selectedCategory.value === 'all') {
      return expenses.value
    }
    return expenses.value.filter((e) => e.category === selectedCategory.value)
  })

  // Actions
  async function fetchSummary(
    period: 'day' | 'week' | 'month' | 'year' = selectedPeriod.value,
    refDate?: string
  ): Promise<FinancialSummary | null> {
    try {
      loading.value = true
      error.value = null
      selectedPeriod.value = period
      const params: Record<string, string> = { period }
      if (refDate) {
        params.ref_date = refDate
      }
      const data = await apiClient.get<FinancialSummary>('/api/v1/finance/summary', params)
      summary.value = data
      return data
    } catch (err: any) {
      error.value = err?.message || 'Impossibile caricare il riepilogo finanziario'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchExpenses(filter?: {
    category?: string
    subcategory?: string
    start_date?: string
    end_date?: string
    is_fixed?: boolean
    limit?: number
    offset?: number
  }): Promise<Expense[]> {
    try {
      loading.value = true
      error.value = null
      const raw = await apiClient.get<Expense[] | null>('/api/v1/finance/expenses', filter as any)
      const data = raw ?? []
      expenses.value = data
      return data
    } catch (err: any) {
      error.value = err?.message || 'Impossibile caricare le spese'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createExpense(payload: Partial<Expense>): Promise<Expense> {
    try {
      loading.value = true
      error.value = null
      const created = await apiClient.post<Expense>('/api/v1/finance/expenses', payload)
      expenses.value.unshift(created)
      // refresh summary asynchronously
      fetchSummary().catch(() => {})
      return created
    } catch (err: any) {
      error.value = err?.message || 'Impossibile registrare la spesa'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateExpense(id: string, payload: Partial<Expense>): Promise<Expense> {
    try {
      loading.value = true
      error.value = null
      const updated = await apiClient.put<Expense>(`/api/v1/finance/expenses/${id}`, payload)
      const idx = expenses.value.findIndex((e) => e.id === id)
      if (idx !== -1) {
        expenses.value[idx] = updated
      }
      fetchSummary().catch(() => {})
      return updated
    } catch (err: any) {
      error.value = err?.message || 'Impossibile aggiornare la spesa'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteExpense(id: string): Promise<void> {
    try {
      loading.value = true
      error.value = null
      await apiClient.delete(`/api/v1/finance/expenses/${id}`)
      expenses.value = expenses.value.filter((e) => e.id !== id)
      fetchSummary().catch(() => {})
    } catch (err: any) {
      error.value = err?.message || 'Impossibile eliminare la spesa'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchBudgets(monthYear?: string): Promise<BudgetTarget[]> {
    try {
      loading.value = true
      error.value = null
      const params = monthYear ? { month_year: monthYear } : undefined
      const raw = await apiClient.get<BudgetTarget[] | null>('/api/v1/finance/budgets', params)
      const data = raw ?? []
      budgets.value = data
      return data
    } catch (err: any) {
      error.value = err?.message || 'Impossibile caricare i budget'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function setBudget(
    category: string,
    monthlyLimit: number,
    monthYear: string = 'global'
  ): Promise<BudgetTarget> {
    try {
      loading.value = true
      error.value = null
      const res = await apiClient.post<BudgetTarget>('/api/v1/finance/budgets', {
        category,
        monthly_limit: monthlyLimit,
        month_year: monthYear
      })
      const idx = budgets.value.findIndex((b) => b.category === category && b.month_year === monthYear)
      if (idx !== -1) {
        budgets.value[idx] = res
      } else {
        budgets.value.push(res)
      }
      fetchSummary().catch(() => {})
      return res
    } catch (err: any) {
      error.value = err?.message || 'Impossibile impostare il budget'
      throw err
    } finally {
      loading.value = false
    }
  }

  function setCategory(cat: ExpenseCategory | 'all') {
    selectedCategory.value = cat
  }

  return {
    summary,
    expenses,
    budgets,
    selectedPeriod,
    selectedCategory,
    loading,
    error,
    // Computed
    totalSpent,
    monthlyBudget,
    remainingBudget,
    percentageChange,
    dailyAverage,
    budgetUsagePercent,
    isOverBudget,
    categoryBreakdown,
    monthlyTrends,
    fixedVsVariable,
    savingsInsights,
    filteredExpenses,
    // Actions
    fetchSummary,
    fetchExpenses,
    createExpense,
    updateExpense,
    deleteExpense,
    fetchBudgets,
    setBudget,
    setCategory
  }
})
