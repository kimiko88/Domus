<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  Package,
  ChefHat,
  Calendar,
  Sparkles,
  AlertTriangle
} from '@lucide/vue'
import { usePantryStore } from '../stores/pantry'
import PantryKpiCards from '../components/pantry/PantryKpiCards.vue'
import PantryInventoryTab from '../components/pantry/PantryInventoryTab.vue'
import PantryRecipeTab from '../components/pantry/PantryRecipeTab.vue'
import PantryMealCalendarTab from '../components/pantry/PantryMealCalendarTab.vue'
import PantryItemModal from '../components/pantry/PantryItemModal.vue'
import RecipeModal from '../components/pantry/RecipeModal.vue'
import MealPlanModal from '../components/pantry/MealPlanModal.vue'
import type { PantryItem, Recipe, MealPlan, MealType } from '../api/types'

const pantryStore = usePantryStore()

type TabKey = 'inventory' | 'recipes' | 'calendar'
const activeTab = ref<TabKey>('inventory')

// Modals
const isItemModalOpen = ref(false)
const itemToEdit = ref<PantryItem | null>(null)

const isRecipeModalOpen = ref(false)
const recipeToEdit = ref<Recipe | null>(null)

const isMealModalOpen = ref(false)
const initialMealDate = ref<string | undefined>()
const initialMealType = ref<MealType | undefined>()
const initialRecipeId = ref<string | undefined>()

// Toast
const toastMessage = ref<string | null>(null)
let toastTimeout: ReturnType<typeof setTimeout> | undefined

function showToast(msg: string) {
  toastMessage.value = msg
  if (toastTimeout) clearTimeout(toastTimeout)
  toastTimeout = setTimeout(() => {
    toastMessage.value = null
  }, 4000)
}

onMounted(async () => {
  await Promise.allSettled([
    pantryStore.fetchPantryItems(),
    pantryStore.fetchExpiringItems(),
    pantryStore.fetchLowStockItems(),
    pantryStore.fetchRecipes(),
    pantryStore.fetchCookableRecipes(),
    pantryStore.fetchMealPlans()
  ])
})

// Tab navigation helpers from KPI cards
function onFilterLowStock() {
  activeTab.value = 'inventory'
}
function onFilterExpiring() {
  activeTab.value = 'inventory'
}
function onGoRecipesCookable() {
  activeTab.value = 'recipes'
}

// Pantry Item CRUD
function openAddItem() {
  itemToEdit.value = null
  isItemModalOpen.value = true
}

function openEditItem(item: PantryItem) {
  itemToEdit.value = item
  isItemModalOpen.value = true
}

async function handleSaveItem(payload: Partial<PantryItem>) {
  if (itemToEdit.value) {
    await pantryStore.updatePantryItem(itemToEdit.value.id, payload)
    showToast('Alimento aggiornato con successo!')
  } else {
    await pantryStore.createPantryItem(payload)
    showToast('Alimento aggiunto alla dispensa!')
  }
}

async function handleDeleteItem(id: string) {
  if (window.confirm('Vuoi rimuovere questo articolo dalla dispensa?')) {
    await pantryStore.deletePantryItem(id)
    showToast('Alimento rimosso.')
  }
}

async function handleAdjustQty(id: string, delta: number) {
  try {
    const updated = await pantryStore.adjustQuantity(id, delta)
    if (updated.quantity <= updated.min_threshold) {
      showToast(`Scorta bassa per "${updated.name}" (${updated.quantity} ${updated.unit})!`)
    }
  } catch (err: any) {
    showToast(err.message || 'Errore nella modifica scorte')
  }
}

// Recipe CRUD & Scheduling
function openAddRecipe() {
  recipeToEdit.value = null
  isRecipeModalOpen.value = true
}

function openEditRecipe(recipe: Recipe) {
  recipeToEdit.value = recipe
  isRecipeModalOpen.value = true
}

async function handleSaveRecipe(payload: Partial<Recipe>) {
  if (recipeToEdit.value) {
    await pantryStore.updateRecipe(recipeToEdit.value.id, payload)
    showToast('Ricetta aggiornata!')
  } else {
    await pantryStore.createRecipe(payload)
    showToast('Nuova ricetta salvata nel ricettario!')
  }
}

async function handleDeleteRecipe(id: string) {
  if (window.confirm('Vuoi eliminare questa ricetta?')) {
    await pantryStore.deleteRecipe(id)
    showToast('Ricetta eliminata.')
  }
}

function handleScheduleRecipe(recipe: Recipe) {
  initialRecipeId.value = recipe.id
  initialMealDate.value = new Date().toISOString().slice(0, 10)
  initialMealType.value = 'dinner'
  isMealModalOpen.value = true
}

// Meal Planning
function openAddMealPlan(date?: string, mealType?: MealType) {
  initialMealDate.value = date
  initialMealType.value = mealType
  initialRecipeId.value = undefined
  isMealModalOpen.value = true
}

async function handleSaveMealPlan(payload: Partial<MealPlan>) {
  await pantryStore.createMealPlan(payload)
  showToast('Pasto pianificato nel calendario!')
}

async function handleConsumeMeal(id: string) {
  try {
    await pantryStore.consumeMealPlan(id)
    showToast('🍽️ Pasto consumato! Gli ingredienti sono stati scalati in proporzione ai commensali.')
  } catch (err: any) {
    showToast(err.message || 'Errore durante il consumo pasto')
  }
}

async function handleDeleteMeal(id: string) {
  if (window.confirm('Vuoi rimuovere questo pasto dal calendario?')) {
    await pantryStore.deleteMealPlan(id)
    showToast('Pasto rimosso dal calendario.')
  }
}


</script>

<template>
  <main id="main-content" class="pantry-view-container">
    <!-- Header -->
    <header class="pantry-header">
      <div>
        <h1 class="page-title">Dispensa & Ricettario Intelligente</h1>
        <p class="page-subtitle">
          Monitora scorte alimentari, crea ricette, pianifica i pasti con scalatura automatica per commensali e leggi scontrini OCR.
        </p>
      </div>
    </header>

    <!-- Global Toast Alert -->
    <div
      v-if="toastMessage"
      class="toast-notification glass-panel"
      role="status"
      aria-live="polite"
    >
      <Sparkles :size="18" class="text-primary" />
      <span>{{ toastMessage }}</span>
    </div>

    <!-- Top KPI Cards -->
    <PantryKpiCards
      :total-items="pantryStore.totalItemsCount"
      :low-stock-count="pantryStore.lowStockCount"
      :expiring-count="pantryStore.expiringCount"
      :cookable-count="pantryStore.cookableCount"
      @filter-low-stock="onFilterLowStock"
      @filter-expiring="onFilterExpiring"
      @go-recipes-cookable="onGoRecipesCookable"
    />

    <!-- Main Navigation Tabs -->
    <div class="main-tabs-bar" role="tablist" aria-label="Sezioni dispensa">
      <button
        type="button"
        :class="['tab-nav-btn', { active: activeTab === 'inventory' }]"
        role="tab"
        :aria-selected="activeTab === 'inventory'"
        @click="activeTab = 'inventory'"
      >
        <Package :size="18" />
        <span>Dispensa & Scorte</span>
        <span class="badge-count">{{ pantryStore.totalItemsCount }}</span>
      </button>

      <button
        type="button"
        :class="['tab-nav-btn', { active: activeTab === 'recipes' }]"
        role="tab"
        :aria-selected="activeTab === 'recipes'"
        @click="activeTab = 'recipes'"
      >
        <ChefHat :size="18" />
        <span>Ricettario</span>
        <span v-if="pantryStore.cookableCount > 0" class="badge-cookable">
          {{ pantryStore.cookableCount }} cucinabili
        </span>
      </button>

      <button
        type="button"
        :class="['tab-nav-btn', { active: activeTab === 'calendar' }]"
        role="tab"
        :aria-selected="activeTab === 'calendar'"
        @click="activeTab = 'calendar'"
      >
        <Calendar :size="18" />
        <span>Calendario Pasti</span>
      </button>

    </div>

    <!-- Active Tab Component -->
    <section class="tab-body">
      <!-- 1. Inventory -->
      <PantryInventoryTab
        v-if="activeTab === 'inventory'"
        :items="pantryStore.items"
        :loading="pantryStore.loading"
        @add-item="openAddItem"
        @edit-item="openEditItem"
        @delete-item="handleDeleteItem"
        @adjust-qty="handleAdjustQty"
        @toast="showToast"
      />

      <!-- 2. Recipes -->
      <PantryRecipeTab
        v-else-if="activeTab === 'recipes'"
        :recipes="pantryStore.recipes"
        :cookable-results="pantryStore.cookableRecipes"
        :loading="pantryStore.loading"
        @add-recipe="openAddRecipe"
        @edit-recipe="openEditRecipe"
        @delete-recipe="handleDeleteRecipe"
        @schedule-recipe="handleScheduleRecipe"
      />

      <!-- 3. Meal Calendar -->
      <PantryMealCalendarTab
        v-else-if="activeTab === 'calendar'"
        :meal-plans="pantryStore.mealPlans"
        :loading="pantryStore.loading"
        @add-meal-plan="openAddMealPlan"
        @consume-meal="handleConsumeMeal"
        @delete-meal="handleDeleteMeal"
      />

    </section>

    <!-- Modals -->
    <PantryItemModal
      :is-open="isItemModalOpen"
      :item-to-edit="itemToEdit"
      @close="isItemModalOpen = false"
      @save="handleSaveItem"
    />

    <RecipeModal
      :is-open="isRecipeModalOpen"
      :recipe-to-edit="recipeToEdit"
      @close="isRecipeModalOpen = false"
      @save="handleSaveRecipe"
    />

    <MealPlanModal
      :is-open="isMealModalOpen"
      :recipes="pantryStore.recipes"
      :initial-date="initialMealDate"
      :initial-meal-type="initialMealType"
      :initial-recipe-id="initialRecipeId"
      @close="isMealModalOpen = false"
      @save="handleSaveMealPlan"
    />
  </main>
</template>

<style scoped>
.pantry-view-container {
  width: 100%;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-5, 20px);
  padding: var(--space-6, 24px);
  max-width: 1400px;
  margin: 0 auto;
  width: 100%;
}

.pantry-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.page-title {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--text-main, #f8fafc);
  margin: 0 0 6px 0;
  letter-spacing: -0.02em;
}

.page-subtitle {
  font-size: 0.95rem;
  color: var(--text-muted, #94a3b8);
  margin: 0;
  line-height: 1.5;
}

.toast-notification {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 12px 18px;
  border-radius: var(--radius-md, 12px);
  background: rgba(30, 41, 59, 0.9);
  border: 1px solid var(--primary, #3b82f6);
  color: var(--text-main, #f8fafc);
  font-size: 0.9rem;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3);
  animation: slideInDown 0.25s ease;
}

@keyframes slideInDown {
  from {
    transform: translateY(-8px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.main-tabs-bar {
  display: flex;
  gap: 10px;
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  padding-bottom: 2px;
  overflow-x: auto;
  scrollbar-width: thin;
}

.tab-nav-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 18px;
  border-radius: 10px 10px 0 0;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--text-muted, #94a3b8);
  font-size: 0.92rem;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s ease;
}

.tab-nav-btn:hover {
  color: var(--text-main, #f8fafc);
  background: rgba(255, 255, 255, 0.03);
}

.tab-nav-btn.active {
  color: var(--primary, #3b82f6);
  font-weight: 600;
  border-bottom-color: var(--primary, #3b82f6);
  background: rgba(59, 130, 246, 0.06);
}

.badge-count {
  font-size: 0.75rem;
  padding: 2px 6px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-main, #f8fafc);
}

.badge-cookable {
  font-size: 0.72rem;
  padding: 2px 6px;
  border-radius: 10px;
  background: rgba(16, 185, 129, 0.2);
  color: var(--success, #10b981);
  font-weight: 600;
}

.tab-body {
  margin-top: var(--space-2, 8px);
}

@media (max-width: 768px) {
  .pantry-view-container {
    padding: var(--space-4, 16px);
  }
  .page-title {
    font-size: 1.4rem;
    overflow-wrap: anywhere;
  }

  .main-tabs-bar {
    max-width: 100%;
    overscroll-behavior-x: contain;
    -webkit-overflow-scrolling: touch;
  }

  .tab-nav-btn {
    min-height: 44px;
    flex-shrink: 0;
  }

  .tab-body,
  .pantry-header,
  .pantry-header > div {
    min-width: 0;
  }
}
</style>
