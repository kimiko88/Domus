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


  it('reuses the receipt idempotency key when a failed import is retried', async () => {
    const mockImportResult: ReceiptImportResult = {
      items_added: 1,
      items_updated: 0,
      expense_id: 'exp_retry',
      expense_amount: 2.5
    }
    const requestData = {
      store_name: 'Coop',
      date: '2026-11-26',
      items: [{
        name: 'Pasta',
        quantity: 1,
        unit: 'pz' as const,
        unit_price: 2.5,
        total_price: 2.5,
        category: 'pantry' as const
      }],
      total_amount: 2.5,
      document_type: 'receipt' as const,
      auto_add_to_finance: true,
      auto_update_pantry: true
    }

    vi.mocked(apiClient.post)
      .mockRejectedValueOnce(new Error('network timeout'))
      .mockResolvedValueOnce(mockImportResult)
    vi.mocked(apiClient.get).mockResolvedValue([])

    const store = usePantryStore()
    await expect(store.importReceipt(requestData)).rejects.toThrow('network timeout')
    await store.importReceipt(requestData)

    const firstHeaders = vi.mocked(apiClient.post).mock.calls[0][2]?.headers as Record<string, string>
    const secondHeaders = vi.mocked(apiClient.post).mock.calls[1][2]?.headers as Record<string, string>
    expect(firstHeaders['X-Idempotency-Key']).toBeTruthy()
    expect(secondHeaders['X-Idempotency-Key']).toBe(firstHeaders['X-Idempotency-Key'])
    expect(secondHeaders['Content-Type']).toBe('application/json')
  })

describe('Pantry purchase history', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const savedReceipt: Receipt = {
    id: 'receipt_1',
    store_name: 'Esselunga',
    date: '2026-11-26',
    document_type: 'receipt',
    items: [{ name: 'Pasta', quantity: 500, unit: 'g', unit_price: 0.002, total_price: 1, category: 'pantry' }],
    total_amount: 1,
    confidence: 0.9
  }

  it('loads saved purchase records', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce([savedReceipt])
    const store = usePantryStore()
    const result = await store.fetchReceiptHistory()
    expect(result).toEqual([savedReceipt])
    expect(store.receiptHistory).toEqual([savedReceipt])
    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/receipts?limit=50')
  })

  it('updates an existing purchase and replaces it in history', async () => {
    const updated = { ...savedReceipt, store_name: 'Coop', total_amount: 2 }
    vi.mocked(apiClient.put).mockResolvedValueOnce(updated)
    const store = usePantryStore()
    store.receiptHistory = [savedReceipt]
    const payload = { store_name: 'Coop', date: savedReceipt.date, document_type: 'receipt' as const, total_amount: 2, items: savedReceipt.items }
    const result = await store.updateReceipt('receipt_1', payload)
    expect(result.store_name).toBe('Coop')
    expect(store.receiptHistory[0].store_name).toBe('Coop')
    expect(apiClient.put).toHaveBeenCalledWith('/api/v1/receipts/receipt_1', payload)
  })
})

describe('Pantry item partial updates', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('merges a quantity-only edit with the existing stock metadata', async () => {
    const item: PantryItem = {
      id: 'pantry_quantity_test', name: 'Pasta', category: 'pantry', quantity: 3, unit: 'pz',
      min_threshold: 1, location: 'pantry', cost: 2.4, expiration_date: '2027-01-01',
      created_at: '2026-10-17T00:00:00Z', updated_at: '2026-10-17T00:00:00Z'
    }
    const updated = { ...item, quantity: 8 }
    vi.mocked(apiClient.put).mockResolvedValueOnce(updated)
    const store = usePantryStore()
    store.items = [item]
    await store.updatePantryItem(item.id, { quantity: 8 })
    expect(apiClient.put).toHaveBeenCalledWith(`/api/v1/pantry/items/${item.id}`, { ...item, quantity: 8 })
  })
})