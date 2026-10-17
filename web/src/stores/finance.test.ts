import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useFinanceStore } from './finance'
import { apiClient } from '../api/client'
import type { FinancialSummary, Expense, BudgetTarget } from '../api/types'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

describe('Finance Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const mockSummary: FinancialSummary = {
    period: 'month',
    start_date: '2026-10-01T00:00:00Z',
    end_date: '2026-10-31T23:59:59Z',
    total_spent: 850.5,
    previous_period_total: 920.0,
    percentage_change: -7.55,
    daily_average: 27.43,
    monthly_budget: 1500.0,
    remaining_budget: 649.5,
    categories: [
      {
        category: 'food',
        name: 'Cibo & Alimentari',
        total_amount: 350.5,
        percentage: 41.2,
        count: 8
      },
      {
        category: 'utilities',
        name: 'Bollette & Utenze',
        total_amount: 250.0,
        percentage: 29.4,
        count: 2
      },
      {
        category: 'transport',
        name: 'Carburante & Mobilità',
        total_amount: 150.0,
        percentage: 17.6,
        count: 3
      },
      {
        category: 'home_maintenance',
        name: 'Materiali & Manutenzione Casa',
        total_amount: 100.0,
        percentage: 11.8,
        count: 1
      }
    ],
    monthly_trends: [
      {
        year_month: '2026-10',
        month_name: 'Ottobre',
        total_amount: 850.5,
        utilities_amount: 250.0,
        food_amount: 350.5,
        transport_amount: 150.0
      }
    ],
    fixed_vs_variable: {
      fixed_amount: 250.0,
      variable_amount: 600.5,
      fixed_percentage: 29.4,
      variable_percentage: 70.6
    },
    insights: [
      {
        id: 'ins_1',
        type: 'energy_tariff',
        title: 'Bolletta Elettrica Elevata',
        message: 'Costo unitario elevato rispetto alla media',
        potential_saving: 35.0,
        severity: 'warning',
        action_url: '/energy'
      },
      {
        id: 'ins_2',
        type: 'food_waste',
        title: 'Spreco Alimentare Evitabile',
        message: 'Valore cibo scaduto',
        potential_saving: 18.5,
        severity: 'alert',
        action_url: '/shopping'
      }
    ]
  }

  const mockExpenses: Expense[] = [
    {
      id: 'exp-1',
      title: 'Spesa Conad',
      amount: 65.4,
      currency: 'EUR',
      category: 'food',
      subcategory: 'groceries',
      expense_date: '2026-10-15T12:00:00Z',
      is_fixed: false,
      created_at: '',
      updated_at: ''
    },
    {
      id: 'exp-2',
      title: 'Bolletta Enel',
      amount: 180.0,
      currency: 'EUR',
      category: 'utilities',
      subcategory: 'electricity',
      expense_date: '2026-10-10T10:00:00Z',
      is_fixed: true,
      energy_kwh: 360,
      created_at: '',
      updated_at: ''
    }
  ]

  it('fetches summary and calculates KPI computed properties', async () => {
    const store = useFinanceStore()
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockSummary)

    await store.fetchSummary('month')
    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/finance/summary', { period: 'month' })
    expect(store.totalSpent).toBe(850.5)
    expect(store.monthlyBudget).toBe(1500.0)
    expect(store.remainingBudget).toBe(649.5)
    expect(store.percentageChange).toBe(-7.55)
    expect(store.budgetUsagePercent).toBe(57) // 850.5 / 1500 = 56.7% -> 57%
    expect(store.isOverBudget).toBe(false)
    expect(store.categoryBreakdown.length).toBe(4)
    expect(store.savingsInsights.length).toBe(2)
  })

  it('filters expenses by category', async () => {
    const store = useFinanceStore()
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockExpenses)

    await store.fetchExpenses()
    expect(store.expenses.length).toBe(2)
    expect(store.filteredExpenses.length).toBe(2)

    store.setCategory('food')
    expect(store.filteredExpenses.length).toBe(1)
    expect(store.filteredExpenses[0].id).toBe('exp-1')

    store.setCategory('all')
    expect(store.filteredExpenses.length).toBe(2)
  })

  it('creates an expense and prepends it to expenses list', async () => {
    const store = useFinanceStore()
    const newExp: Expense = {
      id: 'exp-new',
      title: 'Pieno Benzina',
      amount: 80.0,
      currency: 'EUR',
      category: 'transport',
      subcategory: 'fuel_petrol',
      expense_date: '2026-10-17T09:00:00Z',
      is_fixed: false,
      created_at: '',
      updated_at: ''
    }
    vi.mocked(apiClient.post).mockResolvedValueOnce(newExp)
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockSummary)

    const result = await store.createExpense(newExp)
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/finance/expenses', newExp)
    expect(store.expenses[0].id).toBe('exp-new')
    expect(result.title).toBe('Pieno Benzina')
  })

  it('deletes an expense by ID', async () => {
    const store = useFinanceStore()
    store.expenses = [...mockExpenses]
    vi.mocked(apiClient.delete).mockResolvedValueOnce(undefined)
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockSummary)

    await store.deleteExpense('exp-1')
    expect(apiClient.delete).toHaveBeenCalledWith('/api/v1/finance/expenses/exp-1')
    expect(store.expenses.length).toBe(1)
    expect(store.expenses[0].id).toBe('exp-2')
  })

  it('sets and retrieves monthly budget targets', async () => {
    const store = useFinanceStore()
    const mockBudget: BudgetTarget = {
      id: 'b-1',
      category: 'total',
      monthly_limit: 2000.0,
      month_year: 'global',
      created_at: '',
      updated_at: ''
    }
    vi.mocked(apiClient.post).mockResolvedValueOnce(mockBudget)
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockSummary)

    const res = await store.setBudget('total', 2000.0)
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/finance/budgets', {
      category: 'total',
      monthly_limit: 2000.0,
      month_year: 'global'
    })
    expect(res.monthly_limit).toBe(2000.0)
    expect(store.budgets.length).toBe(1)
  })
})
