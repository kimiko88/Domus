import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import FinanceSavingsInsights from './FinanceSavingsInsights.vue'
import type { SavingsInsight } from '@/api/types'

const mockPush = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockPush
  })
}))

describe('FinanceSavingsInsights.vue', () => {
  const mockInsights: SavingsInsight[] = [
    {
      id: 'ins_1',
      type: 'energy_tariff',
      title: 'Bolletta Elettrica Elevata (0.50 €/kWh)',
      message: 'Costo unitario superiore alla media di mercato',
      potential_saving: 45.0,
      severity: 'warning',
      action_url: '/energy'
    },
    {
      id: 'ins_2',
      type: 'food_waste',
      title: 'Spreco Alimentare Evitabile: 18.50 €',
      message: 'Usa la dispensa per ridurre gli sprechi',
      potential_saving: 18.5,
      severity: 'alert',
      action_url: '/shopping'
    },
    {
      id: 'ins_3',
      type: 'contract_expiry',
      title: 'Scadenza Contratto: Polizza Casa (tra 18 giorni)',
      message: 'Rinegozia la tariffa prima del tacito rinnovo',
      severity: 'info',
      action_url: '/finance'
    }
  ]

  it('renders all three smart savings insights and displays total potential savings', () => {
    const wrapper = mount(FinanceSavingsInsights, {
      props: {
        insights: mockInsights
      }
    })

    expect(wrapper.text()).toContain('Suggerimenti Intelligenti di Risparmio')
    expect(wrapper.text()).toContain('La Chicca di Domus')
    expect(wrapper.text()).toContain('Bolletta Elettrica Elevata')
    expect(wrapper.text()).toContain('Spreco Alimentare Evitabile')
    expect(wrapper.text()).toContain('Scadenza Contratto: Polizza Casa')
    // Total savings: 45 + 18.5 = 63.50 €
    expect(wrapper.text()).toContain('63,50')
  })

  it('navigates to action_url when action button is clicked', async () => {
    const wrapper = mount(FinanceSavingsInsights, {
      props: {
        insights: mockInsights
      }
    })

    const buttons = wrapper.findAll('.action-btn')
    expect(buttons.length).toBe(3)

    await buttons[0].trigger('click')
    expect(mockPush).toHaveBeenCalledWith('/energy')

    await buttons[1].trigger('click')
    expect(mockPush).toHaveBeenCalledWith('/shopping')
  })

  it('displays empty state when no insights are present', () => {
    const wrapper = mount(FinanceSavingsInsights, {
      props: {
        insights: []
      }
    })

    expect(wrapper.text()).toContain('Ottimo lavoro! Nessuna spesa anomala o spreco rilevato.')
    expect(wrapper.find('.no-insights-box').exists()).toBe(true)
  })
})
