<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  ChefHat,
  Plus,
  Clock,
  Users,
  CheckCircle2,
  AlertCircle,
  Calendar,
  Edit2,
  Trash2,
  Sparkles
} from '@lucide/vue'
import type { Recipe, RecipeCategory, CookableRecipeResult } from '../../api/types'

interface Props {
  recipes: Recipe[]
  cookableResults: CookableRecipeResult[]
  loading?: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'add-recipe'): void
  (e: 'edit-recipe', recipe: Recipe): void
  (e: 'delete-recipe', id: string): void
  (e: 'schedule-recipe', recipe: Recipe): void
}>()

const selectedCategory = ref<RecipeCategory | 'all'>('all')
const onlyCookable = ref(false)
const searchQuery = ref('')

const categories: { key: RecipeCategory | 'all'; label: string }[] = [
  { key: 'all', label: 'Tutte' },
  { key: 'first_course', label: '🍝 Primi' },
  { key: 'main_course', label: '🥩 Secondi' },
  { key: 'side_dish', label: '🥗 Contorni' },
  { key: 'single_dish', label: '🍲 Piatti Unici' },
  { key: 'breakfast', label: '☕ Colazioni' },
  { key: 'dessert', label: '🍰 Dolci' },
  { key: 'snack', label: '🥪 Spuntini' }
]

// Index cookable results by recipe ID
const cookableMap = computed(() => {
  const map = new Map<string, CookableRecipeResult>()
  for (const c of props.cookableResults) {
    if (c.recipe?.id) {
      map.set(c.recipe.id, c)
    }
  }
  return map
})

const filteredRecipes = computed(() => {
  return props.recipes.filter((rec) => {
    if (selectedCategory.value !== 'all' && rec.category !== selectedCategory.value) {
      return false
    }

    const cookable = cookableMap.value.get(rec.id)
    if (onlyCookable.value) {
      if (!cookable || !cookable.is_cookable) {
        return false
      }
    }

    if (searchQuery.value.trim()) {
      const q = searchQuery.value.toLowerCase()
      const titleMatch = rec.title.toLowerCase().includes(q)
      const ingMatch = rec.ingredients?.some((i) => i.name.toLowerCase().includes(q))
      return titleMatch || ingMatch
    }
    return true
  })
})

function getCategoryLabel(cat: RecipeCategory): string {
  switch (cat) {
    case 'first_course':
      return 'Primo Piatto'
    case 'main_course':
      return 'Secondo Piatto'
    case 'side_dish':
      return 'Contorno'
    case 'single_dish':
      return 'Piatto Unico'
    case 'breakfast':
      return 'Colazione'
    case 'dessert':
      return 'Dolce'
    case 'snack':
      return 'Spuntino'
    default:
      return 'Piatto'
  }
}

function parseRecipeSteps(instructions?: string): Array<{ step: number; text: string; time_minutes?: number }> | null {
  if (!instructions || !instructions.trim()) return null
  const trimmed = instructions.trim()
  if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
    try {
      const parsed = JSON.parse(trimmed)
      if (Array.isArray(parsed) && parsed.length > 0) {
        return parsed
      }
    } catch {}
  }
  return null
}
</script>

<template>
  <div class="recipe-tab-content">
    <!-- Toolbar -->
    <div class="toolbar-row">
      <div class="search-box">
        <input
          v-model="searchQuery"
          type="text"
          class="search-input"
          placeholder="Cerca ricette o ingredienti..."
          aria-label="Cerca ricette"
        />
      </div>

      <div class="toolbar-actions">
        <!-- Cookable Only Switch -->
        <label class="switch-control" title="Mostra solo le ricette che puoi preparare con gli alimenti attualmente in dispensa">
          <input
            v-model="onlyCookable"
            type="checkbox"
            role="switch"
            :aria-checked="onlyCookable"
          />
          <span class="switch-slider"></span>
          <span class="switch-label">
            <Sparkles :size="16" class="text-success" />
            Cucinabili Subito
          </span>
        </label>

        <button
          type="button"
          class="btn btn-primary"
          @click="emit('add-recipe')"
        >
          <Plus :size="18" />
          <span>Nuova Ricetta</span>
        </button>
      </div>
    </div>

    <!-- Category Filter Bar -->
    <div class="categories-row" role="tablist" aria-label="Categorie ricette">
      <button
        v-for="cat in categories"
        :key="cat.key"
        type="button"
        :class="['cat-btn', { active: selectedCategory === cat.key }]"
        role="tab"
        :aria-selected="selectedCategory === cat.key"
        @click="selectedCategory = cat.key"
      >
        {{ cat.label }}
      </button>
    </div>

    <!-- Empty State -->
    <div v-if="filteredRecipes.length === 0" class="empty-state glass-panel">
      <div class="empty-icon">🍳</div>
      <h3 class="empty-title">Nessuna ricetta trovata</h3>
      <p class="empty-desc">
        {{ onlyCookable ? 'Non ci sono ricette con tutti gli ingredienti presenti in casa al 100%.' : 'Aggiungi le tue ricette preferite per calcolare in automatico le dosi per i tuoi commensali!' }}
      </p>
      <button
        type="button"
        class="btn btn-primary"
        @click="emit('add-recipe')"
      >
        <Plus :size="18" />
        <span>Crea Nuova Ricetta</span>
      </button>
    </div>

    <!-- Recipe Cards Grid -->
    <div v-else class="recipes-grid">
      <div
        v-for="recipe in filteredRecipes"
        :key="recipe.id"
        class="recipe-card glass-panel"
      >
        <div class="card-header">
          <span class="recipe-cat-badge">{{ getCategoryLabel(recipe.category) }}</span>

          <div class="card-tools">
            <button
              type="button"
              class="icon-btn"
              title="Modifica ricetta"
              aria-label="Modifica ricetta"
              @click="emit('edit-recipe', recipe)"
            >
              <Edit2 :size="15" />
            </button>
            <button
              type="button"
              class="icon-btn delete-btn"
              title="Elimina ricetta"
              aria-label="Elimina ricetta"
              @click="emit('delete-recipe', recipe.id)"
            >
              <Trash2 :size="15" />
            </button>
          </div>
        </div>

        <h3 class="recipe-title">{{ recipe.title }}</h3>

        <!-- Meta: Portions and Times -->
        <div class="recipe-meta">
          <span class="meta-item" title="Porzioni calibrate">
            <Users :size="14" />
            {{ recipe.portions_base }} persone (base)
          </span>
          <span v-if="recipe.prep_time_minutes" class="meta-item">
            <Clock :size="14" />
            Prep: {{ recipe.prep_time_minutes }}m
          </span>
          <span v-if="recipe.cook_time_minutes" class="meta-item">
            <Clock :size="14" />
            Cottura: {{ recipe.cook_time_minutes }}m
          </span>
        </div>

        <!-- Cookable Status from Pantry -->
        <div
          v-if="cookableMap.get(recipe.id)"
          :class="[
            'cookable-banner',
            cookableMap.get(recipe.id)!.is_cookable ? 'cookable-ready' : 'cookable-missing'
          ]"
        >
          <div class="cookable-status-line">
            <CheckCircle2
              v-if="cookableMap.get(recipe.id)!.is_cookable"
              :size="16"
              class="text-success"
            />
            <AlertCircle
              v-else
              :size="16"
              class="text-warning"
            />
            <span class="cookable-text">
              {{
                cookableMap.get(recipe.id)!.is_cookable
                  ? 'Ingredienti al 100% in dispensa!'
                  : `Disponibile al ${Math.round(cookableMap.get(recipe.id)!.available_percent)}%`
              }}
            </span>
          </div>

          <div
            v-if="!cookableMap.get(recipe.id)!.is_cookable && cookableMap.get(recipe.id)!.missing_ingredients?.length"
            class="missing-list"
          >
            <span>Manca: </span>
            <span class="missing-items-names">
              {{ cookableMap.get(recipe.id)!.missing_ingredients!.join(', ') }}
            </span>
          </div>
        </div>

        <!-- Ingredients List -->
        <div class="recipe-ingredients-wrap">
          <span class="section-label">Ingredienti dosati:</span>
          <div class="ingredients-chips">
            <span
              v-for="(ing, idx) in recipe.ingredients"
              :key="idx"
              class="ingredient-chip"
            >
              <strong>{{ ing.quantity }} {{ ing.unit }}</strong> {{ ing.name }}
            </span>
          </div>
        </div>

        <!-- Preparation Instructions & Steps -->
        <div v-if="recipe.instructions" class="recipe-instructions-wrap">
          <div v-if="parseRecipeSteps(recipe.instructions)" class="steps-timeline">
            <div
              v-for="st in parseRecipeSteps(recipe.instructions)"
              :key="st.step"
              class="step-timeline-item"
            >
              <span class="step-num-pill">{{ st.step }}</span>
              <div class="step-content">
                <span class="step-text">{{ st.text }}</span>
                <span v-if="st.time_minutes" class="step-time-pill" title="Tempo passaggio">
                  <Clock :size="11" />
                  {{ st.time_minutes }}m
                </span>
              </div>
            </div>
          </div>
          <p v-else class="instructions-snippet">
            {{ recipe.instructions }}
          </p>
        </div>

        <!-- Actions -->
        <div class="recipe-actions">
          <button
            type="button"
            class="btn-schedule"
            @click="emit('schedule-recipe', recipe)"
          >
            <Calendar :size="16" />
            <span>Pianifica nel Calendario Pasti</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.recipe-tab-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 16px);
}

.toolbar-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4, 16px);
  flex-wrap: wrap;
}

.search-box {
  background: var(--surface-card, rgba(255, 255, 255, 0.04));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  border-radius: var(--radius-md, 12px);
  padding: 0 14px;
  height: 44px;
  flex: 1;
  min-width: 240px;
  display: flex;
  align-items: center;
}

.search-input {
  background: transparent;
  border: none;
  color: var(--text-main, #f8fafc);
  font-size: 0.92rem;
  width: 100%;
}
.search-input:focus {
  outline: none;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: var(--space-4, 16px);
}

.switch-control {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  user-select: none;
  font-size: 0.88rem;
  color: var(--text-main, #f8fafc);
}

.switch-control input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
}

.switch-slider {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
  background-color: rgba(255, 255, 255, 0.15);
  border-radius: 24px;
  transition: 0.2s;
}

.switch-slider::before {
  position: absolute;
  content: '';
  height: 18px;
  width: 18px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  border-radius: 50%;
  transition: 0.2s;
}

.switch-control input:checked + .switch-slider {
  background-color: var(--success, #10b981);
}

.switch-control input:checked + .switch-slider::before {
  transform: translateX(20px);
}

.switch-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
}

.categories-row {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 4px;
}

.cat-btn {
  padding: 6px 14px;
  border-radius: 20px;
  font-size: 0.85rem;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s;
}

.cat-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-main, #f8fafc);
}

.cat-btn.active {
  background: var(--primary, #3b82f6);
  border-color: var(--primary, #3b82f6);
  color: #fff;
  font-weight: 500;
}

.recipes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: var(--space-4, 16px);
}

.recipe-card {
  display: flex;
  flex-direction: column;
  padding: var(--space-5, 20px);
  border-radius: var(--radius-lg, 16px);
  background: var(--surface-card, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  transition: transform 0.2s ease, border-color 0.2s ease;
}

.recipe-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.recipe-cat-badge {
  font-size: 0.75rem;
  padding: 3px 8px;
  border-radius: 6px;
  background: rgba(59, 130, 246, 0.15);
  color: var(--primary, #3b82f6);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.card-tools {
  display: flex;
  align-items: center;
  gap: 4px;
}

.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 6px;
  background: transparent;
  border: none;
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
}
.icon-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-main, #f8fafc);
}
.delete-btn:hover {
  color: var(--danger, #ef4444);
}

.recipe-title {
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
  margin: 0 0 8px 0;
}

.recipe-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.8rem;
  color: var(--text-muted, #94a3b8);
}

.cookable-banner {
  padding: 8px 12px;
  border-radius: 8px;
  margin-bottom: 14px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.cookable-ready {
  background: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.cookable-missing {
  background: rgba(245, 158, 11, 0.1);
  border: 1px solid rgba(245, 158, 11, 0.25);
}

.cookable-status-line {
  display: flex;
  align-items: center;
  gap: 6px;
}

.cookable-text {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
}

.missing-list {
  font-size: 0.75rem;
  color: var(--warning, #f59e0b);
}

.missing-items-names {
  font-weight: 500;
}

.recipe-ingredients-wrap {
  margin-bottom: 12px;
}

.section-label {
  display: block;
  font-size: 0.78rem;
  color: var(--text-muted, #94a3b8);
  margin-bottom: 6px;
}

.ingredients-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.ingredient-chip {
  padding: 3px 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.05);
  font-size: 0.78rem;
  color: var(--text-main, #f8fafc);
}

.recipe-instructions-wrap {
  margin: 10px 0;
}

.steps-timeline {
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 10px;
  padding: 8px 10px;
  border: 1px solid rgba(255, 255, 255, 0.05);
  margin-bottom: 12px;
}

.step-timeline-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.step-num-pill {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: rgba(59, 130, 246, 0.25);
  border: 1px solid rgba(59, 130, 246, 0.4);
  color: var(--primary, #3b82f6);
  font-size: 0.7rem;
  font-weight: 700;
  flex-shrink: 0;
  margin-top: 2px;
}

.step-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex: 1;
}

.step-text {
  font-size: 0.82rem;
  color: var(--text-main, #f8fafc);
  line-height: 1.35;
}

.step-time-pill {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 0.72rem;
  font-weight: 600;
  color: var(--primary, #38bdf8);
  background: rgba(14, 165, 233, 0.12);
  border: 1px solid rgba(14, 165, 233, 0.25);
  padding: 2px 6px;
  border-radius: 4px;
  white-space: nowrap;
  flex-shrink: 0;
}

.instructions-snippet {
  font-size: 0.82rem;
  color: var(--text-muted, #94a3b8);
  line-height: 1.4;
  margin: 0 0 16px 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.recipe-actions {
  margin-top: auto;
  display: flex;
}

.btn-schedule {
  width: 100%;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border-radius: var(--radius-md, 10px);
  background: rgba(59, 130, 246, 0.12);
  border: 1px solid rgba(59, 130, 246, 0.3);
  color: var(--primary, #3b82f6);
  font-size: 0.88rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-schedule:hover {
  background: var(--primary, #3b82f6);
  color: #fff;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  text-align: center;
  border-radius: var(--radius-xl, 20px);
}

.empty-icon {
  font-size: 3rem;
  margin-bottom: 12px;
}

.empty-title {
  font-size: 1.25rem;
  font-weight: 600;
  margin: 0 0 8px 0;
  color: var(--text-main, #f8fafc);
}

.empty-desc {
  font-size: 0.9rem;
  color: var(--text-muted, #94a3b8);
  max-width: 440px;
  margin: 0 0 20px 0;
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 20px;
  border-radius: var(--radius-md, 10px);
  font-size: 0.92rem;
  font-weight: 600;
  cursor: pointer;
  border: none;
}
.btn-primary {
  background: var(--primary, #3b82f6);
  color: #fff;
}
.btn-primary:hover {
  background: #2563eb;
}
</style>
