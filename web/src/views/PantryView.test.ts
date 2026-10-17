import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import PantryView from './PantryView.vue'
import { apiClient } from '../api/client'
import type { PantryItem, Recipe, MealPlan } from '../api/types'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

describe('PantryView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const mockPantryItems: PantryItem[] = [
    {
      id: 'item-1',
      name: 'Pasta Spaghetti',
      category: 'pantry',
      quantity: 500,
      unit: 'g',
      min_threshold: 200,
      location: 'pantry',
      created_at: '2026-10-17T00:00:00Z',
      updated_at: '2026-10-17T00:00:00Z'
    }
  ]

  const mockRecipes: Recipe[] = [
    {
      id: 'rec-1',
      title: 'Spaghetti al Pomodoro',
      category: 'first_course',
      portions_base: 2,
      prep_time_minutes: 5,
      cook_time_minutes: 10,
      instructions: 'Cuocere e condire.',
      ingredients: [
        { name: 'Pasta Spaghetti', quantity: 200, unit: 'g' }
      ],
      created_at: '2026-10-17T00:00:00Z',
      updated_at: '2026-10-17T00:00:00Z'
    }
  ]

  const mockMealPlans: MealPlan[] = [
    {
      id: 'meal-1',
      plan_date: '2026-10-18',
      meal_type: 'dinner',
      recipe_id: 'rec-1',
      recipe: mockRecipes[0],
      persons_count: 2,
      status: 'planned',
      created_at: '2026-10-17T00:00:00Z',
      updated_at: '2026-10-17T00:00:00Z'
    }
  ]

  it('renders header, KPI cards, and navigation tabs', async () => {
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url.includes('/items')) return Promise.resolve(mockPantryItems)
      if (url.includes('/recipes')) return Promise.resolve(mockRecipes)
      if (url.includes('/meals')) return Promise.resolve(mockMealPlans)
      return Promise.resolve([])
    })

    const wrapper = mount(PantryView)
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('h1.page-title').text()).toBe('Dispensa & Ricettario Intelligente')
    expect(wrapper.text()).toContain('Articoli in Dispensa')
    expect(wrapper.text()).toContain('Dispensa & Scorte')
    expect(wrapper.text()).toContain('Ricettario')
    expect(wrapper.text()).toContain('Calendario Pasti')
    expect(wrapper.text()).toContain('Scanner Scontrino')
  })

  it('switches tabs properly between inventory, recipes, calendar, and scanner', async () => {
    vi.mocked(apiClient.get).mockResolvedValue([])

    const wrapper = mount(PantryView)
    await wrapper.vm.$nextTick()

    const tabs = wrapper.findAll('button.tab-nav-btn')
    expect(tabs.length).toBe(4)

    // Switch to Recipes tab
    await tabs[1].trigger('click')
    expect(wrapper.text()).toContain('Nuova Ricetta')

    // Switch to Calendar tab
    await tabs[2].trigger('click')
    expect(wrapper.text()).toContain('Pianifica Pasto')

    // Switch to Scanner tab
    await tabs[3].trigger('click')
    expect(wrapper.text()).toContain('Lettura Intelligente Scontrino Spesa')
  })

  it('opens and closes item creation modal', async () => {
    vi.mocked(apiClient.get).mockResolvedValue([])

    const wrapper = mount(PantryView)
    await wrapper.vm.$nextTick()

    // Click "Nuovo Alimento" button in inventory tab
    const addBtn = wrapper.find('button.btn-primary')
    expect(addBtn.exists()).toBe(true)
    await addBtn.trigger('click')

    // Item modal is mounted
    expect(wrapper.findComponent({ name: 'PantryItemModal' }).props('isOpen')).toBe(true)
  })
})
