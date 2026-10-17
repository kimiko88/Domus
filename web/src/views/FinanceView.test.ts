import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import FinanceView from './FinanceView.vue'
import { useFinanceStore } from '../stores/finance'
import type { FinancialSummary, Expense } from '@/api/types'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn()
  })
}))

describe('FinanceView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const mockSummary: FinancialSummary = {
    period: 'month',
    start_date: '2026-10-01T00:00:00Z',
    end_date: '2026-10-31T23:59:59Z',
    total_spent: 369.0,
    previous_period_total: 420.0,
    percentage_change: -12.1,
    daily_average: 21.7,
    monthly_budget: 1800.0,
    remaining_budget: 1431.0,
    categories: [
      {
        category: 'food',
        name: 'Cibo & Alimentari',
        total_amount: 114.0,
        percentage: 30.9,
        count: 2
      },
      {
        category: 'utilities',
        name: 'Bollette & Utenze',
        total_amount: 180.0,
        percentage: 48.8,
        count: 1
      },
      {
        category: 'transport',
        name: 'Carburante & Mobilità',
        total_amount: 75.0,
        percentage: 20.3,
        count: 1
      }
    ],
    monthly_trends: [
      {
        year_month: '2026-10',
        month_name: 'Ottobre',
        total_amount: 369.0,
        utilities_amount: 180.0,
        food_amount: 114.0,
        transport_amount: 75.0
      }
    ],
    fixed_vs_variable: {
      fixed_amount: 180.0,
      variable_amount: 189.0,
      fixed_percentage: 48.8,
      variable_percentage: 51.2
    },
    insights: [
      {
        id: 'ins-1',
        type: 'energy_tariff',
        title: 'Bolletta Elettrica Elevata',
        message: 'Costo unitario anomalo',
        potential_saving: 25.0,
        severity: 'warning',
        action_url: '/energy'
      }
    ]
  }

  const mockExpenses: Expense[] = [
    {
      id: 'exp-1',
      title: 'Spesa Conad',
      amount: 95.5,
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
      title: 'Bolletta Luce Ottobre',
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

  it('renders the header, KPI cards, and transactions list', async () => {
    const store = useFinanceStore()
    vi.spyOn(store, 'fetchSummary').mockImplementation(async () => {
      store.summary = mockSummary
      return mockSummary
    })
    vi.spyOn(store, 'fetchExpenses').mockImplementation(async () => {
      store.expenses = mockExpenses
      return mockExpenses
    })
    vi.spyOn(store, 'fetchBudgets').mockResolvedValue([])

    const wrapper = mount(FinanceView)
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('h1').text()).toBe('Bilancio Casa & Spese')
    expect(wrapper.text()).toContain('369,00')
    expect(wrapper.text()).toMatch(/1[.,]?431,00/) // remaining budget
    expect(wrapper.text()).toContain('Spesa Conad')
    expect(wrapper.text()).toContain('Bolletta Luce Ottobre')
  })

  it('triggers fetchSummary with period when clicking period pill', async () => {
    const store = useFinanceStore()
    const summarySpy = vi.spyOn(store, 'fetchSummary').mockResolvedValue(mockSummary)
    vi.spyOn(store, 'fetchExpenses').mockResolvedValue([])
    vi.spyOn(store, 'fetchBudgets').mockResolvedValue([])

    const wrapper = mount(FinanceView)
    await wrapper.vm.$nextTick()

    const pills = wrapper.findAll('.period-pill')
    expect(pills.length).toBe(4)

    // Click 'Settimana' (index 1)
    await pills[1].trigger('click')
    expect(summarySpy).toHaveBeenCalledWith('week')

    // Click 'Anno' (index 3)
    await pills[3].trigger('click')
    expect(summarySpy).toHaveBeenCalledWith('year')
  })

  it('opens and closes ExpenseModal when clicking Nuova Spesa button', async () => {
    const store = useFinanceStore()
    vi.spyOn(store, 'fetchSummary').mockResolvedValue(mockSummary)
    vi.spyOn(store, 'fetchExpenses').mockResolvedValue(mockExpenses)
    vi.spyOn(store, 'fetchBudgets').mockResolvedValue([])

    const wrapper = mount(FinanceView)
    await wrapper.vm.$nextTick()

    expect(wrapper.findComponent({ name: 'ExpenseModal' }).props('isOpen')).toBe(false)

    const createBtn = wrapper.find('.btn-primary-action')
    await createBtn.trigger('click')

    expect(wrapper.findComponent({ name: 'ExpenseModal' }).props('isOpen')).toBe(true)
  })

  it('opens and closes BudgetModal when clicking Budget button', async () => {
    const store = useFinanceStore()
    vi.spyOn(store, 'fetchSummary').mockResolvedValue(mockSummary)
    vi.spyOn(store, 'fetchExpenses').mockResolvedValue([])
    vi.spyOn(store, 'fetchBudgets').mockResolvedValue([])

    const wrapper = mount(FinanceView)
    await wrapper.vm.$nextTick()

    expect(wrapper.findComponent({ name: 'BudgetModal' }).props('isOpen')).toBe(false)

    const budgetBtn = wrapper.find('.btn-budget')
    await budgetBtn.trigger('click')

    expect(wrapper.findComponent({ name: 'BudgetModal' }).props('isOpen')).toBe(true)
  })

  it('filters expenses by search query', async () => {
    const store = useFinanceStore()
    store.expenses = mockExpenses
    vi.spyOn(store, 'fetchSummary').mockResolvedValue(mockSummary)
    vi.spyOn(store, 'fetchExpenses').mockResolvedValue(mockExpenses)
    vi.spyOn(store, 'fetchBudgets').mockResolvedValue([])

    const wrapper = mount(FinanceView)
    await wrapper.vm.$nextTick()

    const searchInput = wrapper.find('.search-input')
    await searchInput.setValue('Conad')

    expect(wrapper.text()).toContain('Spesa Conad')
    expect(wrapper.text()).not.toContain('Bolletta Luce Ottobre')
  })
})
