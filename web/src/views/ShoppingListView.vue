<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  ShoppingCart,
  Plus,
  Trash2,
  CheckCircle,
  Circle,
  Apple,
  Coffee,
  Sparkles,
  Home,
  Package,
  Layers,
  User
} from '@lucide/vue'
import { apiClient } from '../api/client'

interface ShoppingItem {
  id: string
  name: string
  quantity?: string
  category: 'alimentari' | 'frutta_verdura' | 'bevande' | 'igiene' | 'casa' | 'altro'
  completed: boolean
  added_by: string
  completed_by?: string
  created_at: string
  completed_at?: string
}

const items = ref<ShoppingItem[]>([
  {
    id: 'shop_1',
    name: 'Latte Parzialmente Scremato',
    quantity: '2 bottiglie',
    category: 'bevande',
    completed: false,
    added_by: 'Marco',
    created_at: 'Oggi'
  },
  {
    id: 'shop_2',
    name: 'Pomodori San Marzano',
    quantity: '1 kg',
    category: 'frutta_verdura',
    completed: false,
    added_by: 'Chiara',
    created_at: 'Oggi'
  },
  {
    id: 'shop_3',
    name: 'Pastiglie Lavastoviglie',
    quantity: '1 pacco',
    category: 'casa',
    completed: true,
    added_by: 'Marco',
    completed_by: 'Chiara',
    created_at: 'Ieri',
    completed_at: 'Oggi'
  }
])

const filterStatus = ref<'all' | 'active' | 'completed'>('all')
const newItemName = ref('')
const newItemQty = ref('')
const newItemCategory = ref<'alimentari' | 'frutta_verdura' | 'bevande' | 'igiene' | 'casa' | 'altro'>('alimentari')

const categories = [
  { key: 'alimentari', label: 'Alimentari', icon: Package },
  { key: 'frutta_verdura', label: 'Ortofrutta', icon: Apple },
  { key: 'bevande', label: 'Bevande', icon: Coffee },
  { key: 'igiene', label: 'Cura Persona', icon: Sparkles },
  { key: 'casa', label: 'Casa & Pulizie', icon: Home },
  { key: 'altro', label: 'Altro', icon: Layers }
]

const filteredItems = computed(() => {
  return items.value.filter(it => {
    if (filterStatus.value === 'active' && it.completed) return false
    if (filterStatus.value === 'completed' && !it.completed) return false
    return true
  })
})

const pendingCount = computed(() => items.value.filter(i => !i.completed).length)
const completedCount = computed(() => items.value.filter(i => i.completed).length)

async function addItem() {
  if (!newItemName.value.trim()) return

  const payload = {
    name: newItemName.value.trim(),
    quantity: newItemQty.value.trim(),
    category: newItemCategory.value,
    added_by: 'Io'
  }

  try {
    const created = await apiClient.post<ShoppingItem>('/api/v1/shopping', payload)
    if (created) {
      items.value.unshift(created)
    }
  } catch {
    // Optimistic fallback
    items.value.unshift({
      id: 'shop_' + Date.now(),
      ...payload,
      completed: false,
      created_at: 'Adesso'
    })
  }

  newItemName.value = ''
  newItemQty.value = ''
}

async function toggleItem(it: ShoppingItem) {
  it.completed = !it.completed
  if (it.completed) {
    it.completed_by = 'Io'
  } else {
    it.completed_by = undefined
  }

  try {
    await apiClient.post(`/api/v1/shopping/${it.id}/toggle`, { completed_by: 'Io' })
  } catch {}
}

async function deleteItem(id: string) {
  items.value = items.value.filter(i => i.id !== id)
  try {
    await apiClient.delete(`/api/v1/shopping/${id}`)
  } catch {}
}

async function clearCompleted() {
  items.value = items.value.filter(i => !i.completed)
  try {
    await apiClient.post('/api/v1/shopping/clear-completed', {})
  } catch {}
}

onMounted(async () => {
  try {
    const list = await apiClient.get<ShoppingItem[]>('/api/v1/shopping')
    if (list && list.length > 0) items.value = list
  } catch {}
})
</script>

<template>
  <div class="shopping-view-container">
    <header class="shopping-header glass-panel">
      <div class="header-title-box">
        <div class="icon-circle">
          <ShoppingCart :size="24" class="text-primary" />
        </div>
        <div>
          <h1 class="page-title">Lista della Spesa Condivisa</h1>
          <p class="page-subtitle">Sincronizzata in tempo reale con tutti i familiari e gli account della casa</p>
        </div>
      </div>

      <div class="header-actions">
        <button
          v-if="completedCount > 0"
          class="btn-clear-completed"
          @click="clearCompleted"
        >
          <Trash2 :size="16" />
          <span>Svuota Acquistati ({{ completedCount }})</span>
        </button>
      </div>
    </header>

    <!-- Quick Add Bar -->
    <div class="add-bar-panel glass-panel">
      <form class="add-form" @submit.prevent="addItem">
        <div class="inputs-row">
          <input
            v-model="newItemName"
            type="text"
            placeholder="Cosa serve comprare? (es. Pasta, Olio EVO, Caffè...)"
            class="input-name"
            required
          />
          <input
            v-model="newItemQty"
            type="text"
            placeholder="Quantità (es. 2 kg, 1 pacco)"
            class="input-qty"
          />
          <button type="submit" class="btn-add">
            <Plus :size="18" />
            <span>Aggiungi</span>
          </button>
        </div>

        <!-- Category Picker Pills -->
        <div class="category-pills">
          <button
            v-for="cat in categories"
            :key="cat.key"
            type="button"
            :class="['cat-pill', { active: newItemCategory === cat.key }]"
            @click="newItemCategory = cat.key as any"
          >
            <component :is="cat.icon" :size="14" />
            <span>{{ cat.label }}</span>
          </button>
        </div>
      </form>
    </div>

    <!-- Filters & Counter Bar -->
    <div class="filters-bar">
      <div class="filter-pills">
        <button
          :class="['pill-btn', { active: filterStatus === 'all' }]"
          @click="filterStatus = 'all'"
        >
          Tutti ({{ items.length }})
        </button>
        <button
          :class="['pill-btn', { active: filterStatus === 'active' }]"
          @click="filterStatus = 'active'"
        >
          Da Acquistare ({{ pendingCount }})
        </button>
        <button
          :class="['pill-btn', { active: filterStatus === 'completed' }]"
          @click="filterStatus = 'completed'"
        >
          Acquistati ({{ completedCount }})
        </button>
      </div>
    </div>

    <!-- Shopping Items List -->
    <div class="items-list">
      <div
        v-for="it in filteredItems"
        :key="it.id"
        :class="['item-card', 'glass-panel', { completed: it.completed }]"
      >
        <div class="item-left" @click="toggleItem(it)">
          <button class="checkbox-btn">
            <CheckCircle v-if="it.completed" :size="22" class="text-success check-icon" />
            <Circle v-else :size="22" class="text-muted uncheck-icon" />
          </button>

          <div class="item-info">
            <div class="name-qty-row">
              <span class="item-name">{{ it.name }}</span>
              <span v-if="it.quantity" class="item-qty">{{ it.quantity }}</span>
            </div>

            <div class="item-meta">
              <span class="cat-tag" :class="it.category">{{ it.category.replace('_', ' ') }}</span>
              <span class="added-tag">
                <User :size="12" />
                <span>{{ it.added_by }}</span>
              </span>
              <span v-if="it.completed && it.completed_by" class="completed-tag">
                Acquistato da {{ it.completed_by }}
              </span>
            </div>
          </div>
        </div>

        <button class="btn-delete" @click="deleteItem(it.id)" title="Rimuovi">
          <Trash2 :size="16" />
        </button>
      </div>

      <!-- Empty State -->
      <div v-if="filteredItems.length === 0" class="empty-state glass-panel">
        <ShoppingCart :size="48" class="text-muted mb-2" />
        <h3>Nessun articolo nella lista</h3>
        <p class="text-dim">Tutti gli articoli sono stati acquistati o la lista è vuota.</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.shopping-view-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.shopping-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-radius: 16px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
}

.header-title-box {
  display: flex;
  align-items: center;
  gap: 16px;
}

.icon-circle {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: rgba(16, 185, 129, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-title {
  font-size: 1.4rem;
  font-weight: 700;
  margin: 0;
}

.page-subtitle {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin: 4px 0 0;
}

.btn-clear-completed {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 8px;
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.add-bar-panel {
  padding: 20px;
  border-radius: 16px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
}

.add-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.inputs-row {
  display: flex;
  gap: 12px;
}

.input-name {
  flex: 2;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 12px 16px;
  color: #fff;
  font-size: 0.95rem;
}

.input-qty {
  flex: 1;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 12px 16px;
  color: #fff;
  font-size: 0.95rem;
}

.btn-add {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--primary);
  border: none;
  border-radius: 10px;
  padding: 12px 24px;
  color: #fff;
  font-weight: 600;
  cursor: pointer;
}

.category-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.cat-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 20px;
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-muted);
  font-size: 0.8rem;
  cursor: pointer;
  transition: all 0.2s;
}

.cat-pill.active {
  background: rgba(99, 102, 241, 0.2);
  border-color: #818cf8;
  color: #818cf8;
  font-weight: 600;
}

.filters-bar {
  display: flex;
  justify-content: space-between;
}

.filter-pills {
  display: flex;
  gap: 8px;
}

.pill-btn {
  padding: 6px 14px;
  border-radius: 8px;
  border: none;
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-muted);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
}

.pill-btn.active {
  background: var(--primary);
  color: #fff;
}

.items-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.item-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 18px;
  border-radius: 12px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  transition: opacity 0.2s;
}

.item-card.completed {
  opacity: 0.6;
}

.item-card.completed .item-name {
  text-decoration: line-through;
  color: var(--text-muted);
}

.item-left {
  display: flex;
  align-items: center;
  gap: 14px;
  flex: 1;
  cursor: pointer;
}

.checkbox-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 0;
  display: flex;
  align-items: center;
}

.item-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.name-qty-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.item-name {
  font-size: 1rem;
  font-weight: 600;
}

.item-qty {
  font-size: 0.85rem;
  color: #818cf8;
  background: rgba(99, 102, 241, 0.1);
  padding: 2px 8px;
  border-radius: 6px;
}

.item-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.cat-tag {
  text-transform: capitalize;
  background: rgba(255, 255, 255, 0.08);
  padding: 2px 6px;
  border-radius: 4px;
}

.added-tag {
  display: flex;
  align-items: center;
  gap: 4px;
}

.completed-tag {
  color: #10b981;
  font-weight: 600;
}

.btn-delete {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 6px;
  border-radius: 6px;
}

.btn-delete:hover {
  color: #ef4444;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px;
  border-radius: 16px;
  text-align: center;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
}
</style>
