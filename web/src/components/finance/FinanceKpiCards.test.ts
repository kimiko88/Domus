import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FinanceKpiCards from './FinanceKpiCards.vue'
import type { FinancialSummary } from '@/api/types'

describe('FinanceKpiCards.vue', () => {
  const mockSummary: FinancialSummary = {
    period: 'month',
    start_date: '2026-10-01T00:00:00Z',
    end_date: '2026-10-31T23:59:59Z',
    total_spent: 450.0,
    previous_period_total: 500.0,
    percentage_change: -10.0,
    daily_average: 15.0,
    monthly_budget: 1000.0,
    remaining_budget: 550.0,
    categories: [],
    monthly_trends: [],
    fixed_vs_variable: {
      fixed_amount: 180.0,
      variable_amount: 270.0,
      fixed_percentage: 40.0,
      variable_percentage: 60.0
    },
    insights: []
  }

  it('renders total spent and negative percentage change badge (spending decreased)', () => {
    const wrapper = mount(FinanceKpiCards, {
      props: {
        summary: mockSummary,
        period: 'month'
      }
    })

    expect(wrapper.text()).toContain('Totale Speso')
    expect(wrapper.text()).toContain('450,00')
    expect(wrapper.find('.trend-positive').exists()).toBe(true)
    expect(wrapper.text()).toContain('10.0%')
  })

  it('renders monthly budget progress bar and remaining budget', () => {
    const wrapper = mount(FinanceKpiCards, {
      props: {
        summary: mockSummary,
        period: 'month'
      }
    })

    expect(wrapper.text()).toContain('Budget Mensile')
    expect(wrapper.text()).toContain('550,00') // remaining
    expect(wrapper.text()).toContain('45% speso') // 450 / 1000
    const progressBar = wrapper.find('[role="progressbar"]')
    expect(progressBar.attributes('aria-valuenow')).toBe('45')
  })

  it('highlights over-budget state when spending exceeds limit', () => {
    const overBudgetSummary: FinancialSummary = {
      ...mockSummary,
      total_spent: 1200.0,
      remaining_budget: -200.0,
      monthly_budget: 1000.0
    }

    const wrapper = mount(FinanceKpiCards, {
      props: {
        summary: overBudgetSummary,
        period: 'month'
      }
    })

    expect(wrapper.find('.card-alert').exists()).toBe(true)
    expect(wrapper.find('.val-over-budget').exists()).toBe(true)
  })

  it('renders fixed vs variable ratio breakdown', () => {
    const wrapper = mount(FinanceKpiCards, {
      props: {
        summary: mockSummary,
        period: 'month'
      }
    })

    expect(wrapper.text()).toContain('40% Fisse')
    expect(wrapper.text()).toContain('60% Var')
    expect(wrapper.text()).toContain('180,00')
    expect(wrapper.text()).toContain('270,00')
  })
})
