<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  Plus,
  Search,
  Minus,
  Edit2,
  Trash2,
  AlertTriangle,
  Clock,
  ShoppingCart,
  MapPin,
  Check
} from '@lucide/vue'
import type { PantryItem, PantryCategory, PantryLocation } from '../../api/types'
import { apiClient } from '../../api/client'

interface Props {
  items: PantryItem[]
  loading?: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'add-item'): void
  (e: 'edit-item', item: PantryItem): void
  (e: 'delete-item', id: string): void
  (e: 'adjust-qty', id: string, delta: number): void
  (e: 'toast', msg: string): void
}>()

const searchQuery = ref('')
const selectedCategory = ref<PantryCategory | 'all'>('all')
const selectedLocation = ref<PantryLocation | 'all'>('all')

const categories: { key: PantryCategory | 'all'; label: string; icon: string }[] = [
  { key: 'all', label: 'Tutte', icon: '🧺' },
  { key: 'produce', label: 'Ortofrutta', icon: '🍎' },
  { key: 'dairy', label: 'Latticini', icon: '🧀' },
  { key: 'meat_fish', label: 'Carne & Pesce', icon: '🥩' },
  { key: 'bakery', label: 'Forno', icon: '🥖' },
  { key: 'pantry', label: 'Dispensa', icon: '🍝' },
  { key: 'frozen', label: 'Surgelati', icon: '❄️' },
  { key: 'drinks', label: 'Bevande', icon: '🥤' },
  { key: 'condiments', label: 'Condimenti', icon: '🫒' },
  { key: 'snacks', label: 'Snack', icon: '🍪' },
  { key: 'household', label: 'Cura Casa', icon: '🧼' }
]

const locations: { key: PantryLocation | 'all'; label: string }[] = [
  { key: 'all', label: 'Tutti i Luoghi' },
  { key: 'fridge', label: 'Frigo' },
  { key: 'freezer', label: 'Freezer' },
  { key: 'pantry', label: 'Dispensa' },
  { key: 'cellar', label: 'Cantina' }
]

const filteredItems = computed(() => {
  return props.items.filter((item) => {
    if (selectedCategory.value !== 'all' && item.category !== selectedCategory.value) {
      return false
    }
    if (selectedLocation.value !== 'all' && item.location !== selectedLocation.value) {
      return false
    }
    if (searchQuery.value.trim()) {
      const q = searchQuery.value.toLowerCase()
      return item.name.toLowerCase().includes(q) || (item.barcode && item.barcode.includes(q))
    }
    return true
  })
})

function getCategoryBadge(cat: PantryCategory): { label: string; emoji: string } {
  switch (cat) {
    case 'produce':
      return { label: 'Ortofrutta', emoji: '🍎' }
    case 'dairy':
      return { label: 'Latticini', emoji: '🧀' }
    case 'meat_fish':
      return { label: 'Carne/Pesce', emoji: '🥩' }
    case 'bakery':
      return { label: 'Forno', emoji: '🥖' }
    case 'pantry':
      return { label: 'Dispensa', emoji: '🍝' }
    case 'frozen':
      return { label: 'Surgelati', emoji: '❄️' }
    case 'drinks':
      return { label: 'Bevande', emoji: '🥤' }
    case 'condiments':
      return { label: 'Condimenti', emoji: '🫒' }
    case 'snacks':
      return { label: 'Snack', emoji: '🍪' }
    case 'household':
      return { label: 'Casa', emoji: '🧼' }
    default:
      return { label: 'Altro', emoji: '📦' }
  }
}

function getLocationLabel(loc: PantryLocation): string {
  switch (loc) {
    case 'fridge':
      return 'Frigorifero'
    case 'freezer':
      return 'Freezer'
    case 'pantry':
      return 'Dispensa'
    case 'cellar':
      return 'Cantina'
    default:
      return 'Altro'
  }
}

function getExpiryStatus(expiryStr?: string): { isExpired: boolean; isSoon: boolean; text: string } {
  if (!expiryStr) return { isExpired: false, isSoon: false, text: '' }
  const expiry = new Date(expiryStr)
  if (isNaN(expiry.getTime())) return { isExpired: false, isSoon: false, text: '' }

  const today = new Date()
  today.setHours(0, 0, 0, 0)
  expiry.setHours(0, 0, 0, 0)

  const diffDays = Math.round((expiry.getTime() - today.getTime()) / (1000 * 60 * 60 * 24))

  if (diffDays < 0) {
    return { isExpired: true, isSoon: false, text: `Scaduto da ${Math.abs(diffDays)} gg` }
  } else if (diffDays === 0) {
    return { isExpired: false, isSoon: true, text: 'Scade oggi!' }
  } else if (diffDays <= 3) {
    return { isExpired: false, isSoon: true, text: `Scade tra ${diffDays} gg` }
  }
  return { isExpired: false, isSoon: false, text: `Scade il ${expiryStr}` }
}

const addedToShopping = ref<Record<string, boolean>>({})

async function addToShoppingList(item: PantryItem) {
  try {
    const qty = `${Math.max(1, item.min_threshold * 2 - item.quantity)} ${item.unit}`
    await apiClient.post('/api/v1/shopping', {
      name: item.name,
      quantity: qty,
      category: 'alimentari',
      added_by: 'Dispensa Automatica'
    })
    addedToShopping.value[item.id] = true
    emit('toast', `Aggiunto "${item.name}" alla Lista della Spesa!`)
  } catch (err) {
    console.warn('Errore aggiunta lista spesa', err)
  }
}
</script>

<template>
  <div class="pantry-tab-content">
    <!-- Action and Search Bar -->
    <div class="toolbar-row">
      <div class="search-box">
        <Search :size="18" class="search-icon" />
        <input
          v-model="searchQuery"
          type="text"
          class="search-input"
          placeholder="Cerca alimenti per nome o barcode..."
          aria-label="Cerca alimenti"
        />
      </div>

      <div class="toolbar-actions">
        <select
          v-model="selectedLocation"
          class="location-select"
          aria-label="Filtra per posizione"
        >
          <option v-for="l in locations" :key="l.key" :value="l.key">
            {{ l.label }}
          </option>
        </select>

        <button
          type="button"
          class="btn btn-primary"
          @click="emit('add-item')"
        >
          <Plus :size="18" />
          <span>Nuovo Alimento</span>
        </button>
      </div>
    </div>

    <!-- Category Pills -->
    <div class="categories-carousel" role="tablist" aria-label="Categorie dispensa">
      <button
        v-for="cat in categories"
        :key="cat.key"
        type="button"
        :class="['cat-pill', { active: selectedCategory === cat.key }]"
        role="tab"
        :aria-selected="selectedCategory === cat.key"
        @click="selectedCategory = cat.key"
      >
        <span>{{ cat.icon }}</span>
        <span>{{ cat.label }}</span>
      </button>
    </div>

    <!-- Inventory Cards Grid -->
    <div v-if="filteredItems.length === 0" class="empty-state glass-panel">
      <div class="empty-icon">🧺</div>
      <h3 class="empty-title">Nessun alimento trovato</h3>
      <p class="empty-desc">
        Aggiungi i tuoi cibi manualmente o scansiona un nuovo scontrino nella sezione Scanner!
      </p>
      <button
        type="button"
        class="btn btn-primary"
        @click="emit('add-item')"
      >
        <Plus :size="18" />
        <span>Aggiungi Alimento</span>
      </button>
    </div>

    <div v-else class="items-grid">
      <div
        v-for="item in filteredItems"
        :key="item.id"
        :class="[
          'item-card glass-panel',
          {
            'is-low-stock': item.quantity <= item.min_threshold,
            'is-expired': getExpiryStatus(item.expiration_date).isExpired
          }
        ]"
      >
        <div class="card-top">
          <div class="cat-tag">
            <span class="cat-emoji">{{ getCategoryBadge(item.category).emoji }}</span>
            <span class="cat-name">{{ getCategoryBadge(item.category).label }}</span>
          </div>

          <div class="card-actions">
            <button
              type="button"
              class="icon-action-btn"
              title="Modifica alimento"
              aria-label="Modifica alimento"
              @click="emit('edit-item', item)"
            >
              <Edit2 :size="16" />
            </button>
            <button
              type="button"
              class="icon-action-btn delete-btn"
              title="Elimina alimento"
              aria-label="Elimina alimento"
              @click="emit('delete-item', item.id)"
            >
              <Trash2 :size="16" />
            </button>
          </div>
        </div>

        <h4 class="item-title">{{ item.name }}</h4>

        <div class="meta-row">
          <span class="location-tag">
            <MapPin :size="13" />
            {{ getLocationLabel(item.location) }}
          </span>

          <span
            v-if="item.expiration_date"
            :class="[
              'expiry-badge',
              {
                'expired-red': getExpiryStatus(item.expiration_date).isExpired,
                'soon-orange': getExpiryStatus(item.expiration_date).isSoon
              }
            ]"
          >
            <Clock :size="13" />
            {{ getExpiryStatus(item.expiration_date).text }}
          </span>
        </div>

        <!-- Quantity Adjuster -->
        <div class="quantity-control-wrap">
          <div class="quantity-adjuster">
            <button
              type="button"
              class="adjust-btn"
              title="Riduci quantità"
              aria-label="Riduci quantità"
              :disabled="item.quantity <= 0"
              @click="emit('adjust-qty', item.id, item.unit === 'g' || item.unit === 'ml' ? -50 : -1)"
            >
              <Minus :size="16" />
            </button>

            <div class="quantity-display">
              <span class="qty-num">{{ item.quantity }}</span>
              <span class="qty-unit">{{ item.unit }}</span>
            </div>

            <button
              type="button"
              class="adjust-btn"
              title="Aumenta quantità"
              aria-label="Aumenta quantità"
              @click="emit('adjust-qty', item.id, item.unit === 'g' || item.unit === 'ml' ? 50 : 1)"
            >
              <Plus :size="16" />
            </button>
          </div>

          <!-- Low stock alert & Add to Shopping -->
          <div v-if="item.quantity <= item.min_threshold" class="low-stock-bar">
            <span class="low-stock-text">
              <AlertTriangle :size="14" />
              Sotto soglia (min: {{ item.min_threshold }} {{ item.unit }})
            </span>
            <button
              type="button"
              :class="['shopping-quick-btn', { 'is-added': addedToShopping[item.id] }]"
              :disabled="addedToShopping[item.id]"
              @click="addToShoppingList(item)"
            >
              <Check v-if="addedToShopping[item.id]" :size="14" />
              <ShoppingCart v-else :size="14" />
              <span>{{ addedToShopping[item.id] ? 'In Lista' : 'Compra' }}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.pantry-tab-content {
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
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--surface-card, rgba(255, 255, 255, 0.04));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  border-radius: var(--radius-md, 12px);
  padding: 0 14px;
  height: 44px;
  flex: 1;
  min-width: 260px;
}

.search-icon {
  color: var(--text-muted, #94a3b8);
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
  gap: var(--space-3, 12px);
}

.location-select {
  height: 44px;
  padding: 0 14px;
  background: var(--surface-card, rgba(255, 255, 255, 0.04));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  border-radius: var(--radius-md, 12px);
  color: var(--text-main, #f8fafc);
  font-size: 0.88rem;
  color-scheme: dark;
}

.location-select option {
  background-color: #1e293b !important;
  color: #f8fafc !important;
}

.categories-carousel {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 4px;
  scrollbar-width: thin;
}

.cat-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 38px;
  padding: 0 14px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  color: var(--text-muted, #94a3b8);
  font-size: 0.85rem;
  white-space: nowrap;
  cursor: pointer;
  transition: all 0.2s ease;
}

.cat-pill:hover {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-main, #f8fafc);
}

.cat-pill.active {
  background: var(--primary, #3b82f6);
  border-color: var(--primary, #3b82f6);
  color: #fff;
  font-weight: 500;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.items-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--space-4, 16px);
}

.item-card {
  display: flex;
  flex-direction: column;
  padding: var(--space-4, 16px);
  border-radius: var(--radius-lg, 16px);
  background: var(--surface-card, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  position: relative;
  transition: transform 0.2s ease, border-color 0.2s ease;
}

.item-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
}

.item-card.is-low-stock {
  border-color: rgba(245, 158, 11, 0.4);
}

.item-card.is-expired {
  border-color: rgba(239, 68, 68, 0.5);
}

.card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.cat-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 6px;
  font-size: 0.78rem;
  color: var(--text-muted, #94a3b8);
}

.card-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.icon-action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  background: transparent;
  border: none;
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
  transition: all 0.2s;
}

.icon-action-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-main, #f8fafc);
}

.delete-btn:hover {
  color: var(--danger, #ef4444);
}

.item-title {
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
  margin: 0 0 8px 0;
}

.meta-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.location-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.78rem;
  color: var(--text-muted, #94a3b8);
}

.expiry-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.75rem;
  padding: 2px 8px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-muted, #94a3b8);
}

.expired-red {
  background: rgba(239, 68, 68, 0.15);
  color: var(--danger, #ef4444);
  font-weight: 600;
}

.soon-orange {
  background: rgba(245, 158, 11, 0.15);
  color: var(--warning, #f59e0b);
  font-weight: 500;
}

.quantity-control-wrap {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.quantity-adjuster {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: rgba(0, 0, 0, 0.25);
  border-radius: var(--radius-md, 10px);
  padding: 4px;
}

.adjust-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.08);
  border: none;
  color: var(--text-main, #f8fafc);
  cursor: pointer;
  transition: background 0.2s;
}

.adjust-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.18);
}

.adjust-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.quantity-display {
  display: flex;
  align-items: baseline;
  gap: 4px;
}

.qty-num {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-main, #f8fafc);
}

.qty-unit {
  font-size: 0.82rem;
  color: var(--text-muted, #94a3b8);
}

.low-stock-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  background: rgba(245, 158, 11, 0.12);
  border-radius: 8px;
}

.low-stock-text {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.76rem;
  color: var(--warning, #f59e0b);
  font-weight: 500;
}

.shopping-quick-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border-radius: 6px;
  background: var(--warning, #f59e0b);
  color: #000;
  font-size: 0.75rem;
  font-weight: 600;
  border: none;
  cursor: pointer;
  transition: transform 0.15s ease;
}

.shopping-quick-btn:hover:not(:disabled) {
  transform: scale(1.04);
}

.shopping-quick-btn.is-added {
  background: rgba(16, 185, 129, 0.2);
  color: var(--success, #10b981);
  cursor: default;
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
