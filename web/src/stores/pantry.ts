import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '../api/client'
import type {
  PantryItem,
  PantryCategory,
  PantryLocation,
  Recipe,
  RecipeCategory,
  CookableRecipeResult,
  MealPlan,
  Receipt,
  ReceiptImportResult
} from '../api/types'

export const usePantryStore = defineStore('pantry', () => {
  const items = ref<PantryItem[]>([])
  const expiringItems = ref<PantryItem[]>([])
  const lowStockItems = ref<PantryItem[]>([])
  const recipes = ref<Recipe[]>([])
  const cookableRecipes = ref<CookableRecipeResult[]>([])
  const mealPlans = ref<MealPlan[]>([])
  const parsedReceipt = ref<Receipt | null>(null)

  const selectedCategory = ref<PantryCategory | 'all'>('all')
  const selectedLocation = ref<PantryLocation | 'all'>('all')
  const selectedRecipeCategory = ref<RecipeCategory | 'all'>('all')

  const loading = ref(false)
  const error = ref<string | null>(null)

  // --- COMPUTED ---

  const totalItemsCount = computed(() => items.value.length)
  const lowStockCount = computed(() => lowStockItems.value.length)
  const expiringCount = computed(() => expiringItems.value.length)
  const cookableCount = computed(() => cookableRecipes.value.filter((c) => c.is_cookable).length)

  const filteredItems = computed(() => {
    return items.value.filter((it) => {
      if (selectedCategory.value !== 'all' && it.category !== selectedCategory.value) {
        return false
      }
      if (selectedLocation.value !== 'all' && it.location !== selectedLocation.value) {
        return false
      }
      return true
    })
  })

  const filteredRecipes = computed(() => {
    if (selectedRecipeCategory.value === 'all') {
      return recipes.value
    }
    return recipes.value.filter((r) => r.category === selectedRecipeCategory.value)
  })

  const mealPlansByDate = computed(() => {
    const map = new Map<string, MealPlan[]>()
    for (const plan of mealPlans.value) {
      const arr = map.get(plan.plan_date) ?? []
      arr.push(plan)
      map.set(plan.plan_date, arr)
    }
    return map
  })

  // --- PANTRY ACTIONS ---

  async function fetchPantryItems(category?: string, location?: string): Promise<PantryItem[]> {
    try {
      loading.value = true
      error.value = null
      const params: Record<string, string> = {}
      if (category && category !== 'all') params.category = category
      if (location && location !== 'all') params.location = location

      const data = await apiClient.get<PantryItem[]>('/api/v1/pantry/items', params)
      items.value = data ?? []
      return items.value
    } catch (err: any) {
      error.value = err?.message || 'Errore nel caricamento della dispensa'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchExpiringItems(days: number = 3): Promise<PantryItem[]> {
    try {
      const data = await apiClient.get<PantryItem[]>('/api/v1/pantry/expiring', { days })
      expiringItems.value = data ?? []
      return expiringItems.value
    } catch (err: any) {
      console.warn('Errore nel caricamento alimenti in scadenza', err)
      return []
    }
  }

  async function fetchLowStockItems(): Promise<PantryItem[]> {
    try {
      const data = await apiClient.get<PantryItem[]>('/api/v1/pantry/low-stock')
      lowStockItems.value = data ?? []
      return lowStockItems.value
    } catch (err: any) {
      console.warn('Errore nel caricamento scorte basse', err)
      return []
    }
  }

  async function createPantryItem(item: Partial<PantryItem>): Promise<PantryItem> {
    try {
      loading.value = true
      error.value = null
      const created = await apiClient.post<PantryItem>('/api/v1/pantry/items', item)
      items.value.unshift(created)
      if (created.quantity <= created.min_threshold) {
        lowStockItems.value.unshift(created)
      }
      return created
    } catch (err: any) {
      error.value = err?.message || 'Errore nella creazione articolo dispensa'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updatePantryItem(id: string, item: Partial<PantryItem>): Promise<PantryItem> {
    try {
      loading.value = true
      error.value = null
      const updated = await apiClient.put<PantryItem>(`/api/v1/pantry/items/${id}`, item)
      const idx = items.value.findIndex((i) => i.id === id)
      if (idx !== -1) items.value[idx] = updated

      // Update low stock list
      const lowIdx = lowStockItems.value.findIndex((i) => i.id === id)
      if (updated.quantity <= updated.min_threshold) {
        if (lowIdx === -1) lowStockItems.value.push(updated)
        else lowStockItems.value[lowIdx] = updated
      } else if (lowIdx !== -1) {
        lowStockItems.value.splice(lowIdx, 1)
      }

      return updated
    } catch (err: any) {
      error.value = err?.message || "Errore nell'aggiornamento articolo dispensa"
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deletePantryItem(id: string): Promise<void> {
    try {
      loading.value = true
      error.value = null
      await apiClient.delete(`/api/v1/pantry/items/${id}`)
      items.value = items.value.filter((i) => i.id !== id)
      lowStockItems.value = lowStockItems.value.filter((i) => i.id !== id)
      expiringItems.value = expiringItems.value.filter((i) => i.id !== id)
    } catch (err: any) {
      error.value = err?.message || "Errore nell'eliminazione articolo dispensa"
      throw err
    } finally {
      loading.value = false
    }
  }

  async function adjustQuantity(id: string, delta: number): Promise<PantryItem> {
    try {
      const updated = await apiClient.post<PantryItem>(`/api/v1/pantry/items/${id}/adjust`, { delta })
      const idx = items.value.findIndex((i) => i.id === id)
      if (idx !== -1) items.value[idx] = updated

      const lowIdx = lowStockItems.value.findIndex((i) => i.id === id)
      if (updated.quantity <= updated.min_threshold) {
        if (lowIdx === -1) lowStockItems.value.push(updated)
        else lowStockItems.value[lowIdx] = updated
      } else if (lowIdx !== -1) {
        lowStockItems.value.splice(lowIdx, 1)
      }

      return updated
    } catch (err: any) {
      error.value = err?.message || 'Errore nella modifica quantità'
      throw err
    }
  }

  // --- RECIPE ACTIONS ---

  async function fetchRecipes(category?: string): Promise<Recipe[]> {
    try {
      loading.value = true
      error.value = null
      const params: Record<string, string> = {}
      if (category && category !== 'all') params.category = category

      const data = await apiClient.get<Recipe[]>('/api/v1/recipes', params)
      recipes.value = data ?? []
      return recipes.value
    } catch (err: any) {
      error.value = err?.message || 'Errore nel caricamento del ricettario'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchCookableRecipes(): Promise<CookableRecipeResult[]> {
    try {
      const data = await apiClient.get<CookableRecipeResult[]>('/api/v1/recipes/cookable')
      cookableRecipes.value = data ?? []
      return cookableRecipes.value
    } catch (err: any) {
      console.warn('Errore nel calcolo ricette cucinabili', err)
      return []
    }
  }

  async function createRecipe(recipe: Partial<Recipe>): Promise<Recipe> {
    try {
      loading.value = true
      error.value = null
      const created = await apiClient.post<Recipe>('/api/v1/recipes', recipe)
      recipes.value.unshift(created)
      await fetchCookableRecipes()
      return created
    } catch (err: any) {
      error.value = err?.message || 'Errore nella creazione ricetta'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateRecipe(id: string, recipe: Partial<Recipe>): Promise<Recipe> {
    try {
      loading.value = true
      error.value = null
      const updated = await apiClient.put<Recipe>(`/api/v1/recipes/${id}`, recipe)
      const idx = recipes.value.findIndex((r) => r.id === id)
      if (idx !== -1) recipes.value[idx] = updated
      await fetchCookableRecipes()
      return updated
    } catch (err: any) {
      error.value = err?.message || "Errore nell'aggiornamento ricetta"
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteRecipe(id: string): Promise<void> {
    try {
      loading.value = true
      error.value = null
      await apiClient.delete(`/api/v1/recipes/${id}`)
      recipes.value = recipes.value.filter((r) => r.id !== id)
      cookableRecipes.value = cookableRecipes.value.filter((c) => c.recipe.id !== id)
    } catch (err: any) {
      error.value = err?.message || "Errore nell'eliminazione ricetta"
      throw err
    } finally {
      loading.value = false
    }
  }

  // --- MEAL PLANNING ACTIONS ---

  async function fetchMealPlans(startDate?: string, endDate?: string): Promise<MealPlan[]> {
    try {
      loading.value = true
      error.value = null
      const params: Record<string, string> = {}
      if (startDate) params.start_date = startDate
      if (endDate) params.end_date = endDate

      const data = await apiClient.get<MealPlan[]>('/api/v1/meals', params)
      mealPlans.value = data ?? []
      return mealPlans.value
    } catch (err: any) {
      error.value = err?.message || 'Errore nel caricamento del piano pasti'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createMealPlan(plan: Partial<MealPlan>): Promise<MealPlan> {
    try {
      loading.value = true
      error.value = null
      const created = await apiClient.post<MealPlan>('/api/v1/meals', plan)
      mealPlans.value.push(created)
      return created
    } catch (err: any) {
      error.value = err?.message || 'Errore nella pianificazione pasto'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateMealPlan(id: string, plan: Partial<MealPlan>): Promise<MealPlan> {
    try {
      loading.value = true
      error.value = null
      const updated = await apiClient.put<MealPlan>(`/api/v1/meals/${id}`, plan)
      const idx = mealPlans.value.findIndex((m) => m.id === id)
      if (idx !== -1) mealPlans.value[idx] = updated
      return updated
    } catch (err: any) {
      error.value = err?.message || "Errore nell'aggiornamento pasto"
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteMealPlan(id: string): Promise<void> {
    try {
      loading.value = true
      error.value = null
      await apiClient.delete(`/api/v1/meals/${id}`)
      mealPlans.value = mealPlans.value.filter((m) => m.id !== id)
    } catch (err: any) {
      error.value = err?.message || "Errore nell'eliminazione pasto"
      throw err
    } finally {
      loading.value = false
    }
  }

  async function consumeMealPlan(id: string): Promise<MealPlan> {
    try {
      loading.value = true
      error.value = null
      const consumed = await apiClient.post<MealPlan>(`/api/v1/meals/${id}/consume`)
      const idx = mealPlans.value.findIndex((m) => m.id === id)
      if (idx !== -1) mealPlans.value[idx] = consumed

      // Re-fetch pantry & low stock since quantities were proportionally decremented
      await Promise.all([fetchPantryItems(), fetchLowStockItems(), fetchCookableRecipes()])
      return consumed
    } catch (err: any) {
      error.value = err?.message || 'Errore nel consumo pasto'
      throw err
    } finally {
      loading.value = false
    }
  }

  // --- RECEIPT OCR & SMART IMPORT ---

  async function parseReceipt(rawText: string): Promise<Receipt> {
    try {
      loading.value = true
      error.value = null
      const result = await apiClient.post<Receipt>('/api/v1/receipts/parse', { raw_text: rawText })
      parsedReceipt.value = result
      return result
    } catch (err: any) {
      error.value = err?.message || "Errore nell'analisi dello scontrino"
      throw err
    } finally {
      loading.value = false
    }
  }

  async function importReceipt(data: {
    store_name: string
    date: string
    items: Receipt['items']
    total_amount: number
    document_type: 'receipt' | 'invoice'
    auto_add_to_finance: boolean
    auto_update_pantry: boolean
  }): Promise<ReceiptImportResult> {
    try {
      loading.value = true
      error.value = null
      const res = await apiClient.post<ReceiptImportResult>('/api/v1/receipts/import', data)
      parsedReceipt.value = null
      // Re-fetch pantry and low stock
      await Promise.all([fetchPantryItems(), fetchLowStockItems(), fetchCookableRecipes()])
      return res
    } catch (err: any) {
      error.value = err?.message || "Errore nell'importazione dello scontrino"
      throw err
    } finally {
      loading.value = false
    }
  }

  function clearParsedReceipt() {
    parsedReceipt.value = null
  }

  return {
    items,
    expiringItems,
    lowStockItems,
    recipes,
    cookableRecipes,
    mealPlans,
    parsedReceipt,
    selectedCategory,
    selectedLocation,
    selectedRecipeCategory,
    loading,
    error,
    totalItemsCount,
    lowStockCount,
    expiringCount,
    cookableCount,
    filteredItems,
    filteredRecipes,
    mealPlansByDate,
    fetchPantryItems,
    fetchExpiringItems,
    fetchLowStockItems,
    createPantryItem,
    updatePantryItem,
    deletePantryItem,
    adjustQuantity,
    fetchRecipes,
    fetchCookableRecipes,
    createRecipe,
    updateRecipe,
    deleteRecipe,
    fetchMealPlans,
    createMealPlan,
    updateMealPlan,
    deleteMealPlan,
    consumeMealPlan,
    parseReceipt,
    importReceipt,
    clearParsedReceipt
  }
})
