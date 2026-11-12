import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePantryStore } from './pantry'
import { apiClient } from '../api/client'
import type {
  PantryItem,
  Recipe,
  MealPlan,
  CookableRecipeResult,
  Receipt,
  ReceiptImportResult
} from '../api/types'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

describe('Pantry Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const mockItem: PantryItem = {
    id: 'pantry_1',
    name: 'Pasta Penne',
    category: 'pantry',
    quantity: 3,
    unit: 'pz',
    min_threshold: 1,
    location: 'pantry',
    created_at: '2026-10-17T00:00:00Z',
    updated_at: '2026-10-17T00:00:00Z'
  }

  const mockRecipe: Recipe = {
    id: 'rec_1',
    title: 'Spaghetti al Pomodoro',
    category: 'first_course',
    portions_base: 2,
    prep_time_minutes: 5,
    cook_time_minutes: 10,
    instructions: 'Cuocere e condire.',
    ingredients: [
      { name: 'Spaghetti', quantity: 200, unit: 'g' }
    ],
    created_at: '2026-10-17T00:00:00Z',
    updated_at: '2026-10-17T00:00:00Z'
  }

  const mockMealPlan: MealPlan = {
    id: 'meal_1',
    plan_date: '2026-10-18',
    meal_type: 'lunch',
    recipe_id: 'rec_1',
    recipe: mockRecipe,
    persons_count: 2,
    status: 'planned',
    created_at: '2026-10-17T00:00:00Z',
    updated_at: '2026-10-17T00:00:00Z'
  }

  it('fetches pantry items and computes counts', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce([mockItem])
    const store = usePantryStore()

    await store.fetchPantryItems()
    expect(store.items).toHaveLength(1)
    expect(store.totalItemsCount).toBe(1)
    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/pantry/items', {})
  })

  it('creates and deletes pantry items', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce(mockItem)
    vi.mocked(apiClient.delete).mockResolvedValueOnce({})
    const store = usePantryStore()

    const created = await store.createPantryItem({ name: 'Pasta Penne' })
    expect(created.id).toBe('pantry_1')
    expect(store.items).toHaveLength(1)

    await store.deletePantryItem('pantry_1')
    expect(store.items).toHaveLength(0)
  })

  it('fetches cookable recipes and computes count', async () => {
    const cookableResult: CookableRecipeResult = {
      recipe: mockRecipe,
      is_cookable: true,
      available_percent: 100
    }
    vi.mocked(apiClient.get).mockResolvedValueOnce([cookableResult])
    const store = usePantryStore()

    await store.fetchCookableRecipes()
    expect(store.cookableRecipes).toHaveLength(1)
    expect(store.cookableCount).toBe(1)
  })

  it('consumes meal plan and refreshes inventory', async () => {
    const consumedMeal: MealPlan = { ...mockMealPlan, status: 'consumed' }
    vi.mocked(apiClient.post).mockResolvedValueOnce(consumedMeal)
    vi.mocked(apiClient.get).mockResolvedValue([])

    const store = usePantryStore()
    store.mealPlans = [mockMealPlan]

    const result = await store.consumeMealPlan('meal_1')
    expect(result.status).toBe('consumed')
    expect(store.mealPlans[0].status).toBe('consumed')
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/meals/meal_1/consume')
  })

  it('parses receipt and imports into pantry and finance', async () => {
    const mockReceipt: Receipt = {
      store_name: 'Esselunga',
      date: '2026-10-18',
      total_amount: 15.5,
      items: [
        {
          name: 'Latte',
          quantity: 1,
          unit: 'l',
          unit_price: 1.5,
          total_price: 1.5,
          category: 'dairy'
        }
      ],
      confidence: 0.95
    }
    const mockImportResult: ReceiptImportResult = {
      items_added: 1,
      items_updated: 0,
      expense_id: 'exp_123',
      expense_amount: 15.5
    }

    vi.mocked(apiClient.post)
      .mockResolvedValueOnce(mockReceipt)
      .mockResolvedValueOnce(mockImportResult)
    vi.mocked(apiClient.get).mockResolvedValue([])

    const store = usePantryStore()
    const parsed = await store.parseReceipt('ESSELUNGA...')
    expect(parsed.store_name).toBe('Esselunga')
    expect(store.parsedReceipt).toEqual(mockReceipt)

    const imported = await store.importReceipt({
      store_name: parsed.store_name,
      date: parsed.date,
      items: parsed.items,
      total_amount: parsed.total_amount,
      document_type: 'receipt',
      auto_add_to_finance: true,
      auto_update_pantry: true
    })
    expect(imported.items_added).toBe(1)
    expect(imported.expense_id).toBe('exp_123')
    expect(store.parsedReceipt).toBeNull()
  })
})
