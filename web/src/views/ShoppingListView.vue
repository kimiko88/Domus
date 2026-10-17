<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
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
  User,
  Store,
  Building2,
  Filter,
  Check,
  PackageCheck
} from '@lucide/vue'
import { apiClient } from '../api/client'
import { usePantryStore } from '../stores/pantry'
import TransferShoppingModal, { type ShoppingItemForTransfer } from '../components/pantry/TransferShoppingModal.vue'

const { t } = useI18n()
const router = useRouter()
const pantryStore = usePantryStore()

interface ShoppingItem {
  id: string
  name: string
  quantity?: string
  category: 'alimentari' | 'frutta_verdura' | 'bevande' | 'igiene' | 'casa' | 'altro'
  supermarket?: string
  completed: boolean
  added_by: string
  completed_by?: string
  created_at: string
  completed_at?: string
}

const items = ref<ShoppingItem[]>([])

const filterStatus = ref<'all' | 'active' | 'completed'>('all')
const filterSupermarket = ref<string>('all')

const newItemName = ref('')
const newItemQty = ref('')
const newItemCategory = ref<'alimentari' | 'frutta_verdura' | 'bevande' | 'igiene' | 'casa' | 'altro'>('alimentari')
const newItemSupermarket = ref('')

const categories = [
  { key: 'alimentari', label: 'Alimentari', icon: Package },
  { key: 'frutta_verdura', label: 'Ortofrutta', icon: Apple },
  { key: 'bevande', label: 'Bevande', icon: Coffee },
  { key: 'igiene', label: 'Cura Persona', icon: Sparkles },
  { key: 'casa', label: 'Casa & Pulizie', icon: Home },
  { key: 'altro', label: 'Altro', icon: Layers }
]

const popularSupermarkets = [
  'Esselunga',
  'Coop',
  'Conad',
  'Lidl',
  'Carrefour',
  'Eurospin',
  'Ipercoop',
  'MD'
]

// Available supermarkets in existing list
const availableSupermarkets = computed(() => {
  const set = new Set<string>()
  for (const it of items.value) {
    if (it.supermarket && it.supermarket.trim()) {
      set.add(it.supermarket.trim())
    }
  }
  return Array.from(set).sort()
})

const filteredItems = computed(() => {
  return items.value.filter(it => {
    if (filterStatus.value === 'active' && it.completed) return false
    if (filterStatus.value === 'completed' && !it.completed) return false
    if (filterSupermarket.value !== 'all') {
      if ((it.supermarket || '').toLowerCase() !== filterSupermarket.value.toLowerCase()) {
        return false
      }
    }
    return true
  })
})

const pendingCount = computed(() => items.value.filter(i => !i.completed).length)
const completedCount = computed(() => items.value.filter(i => i.completed).length)

async function addItem() {
  if (!newItemName.value.trim()) return

  const payload = {
    name: newItemName.value.trim(),
    quantity: newItemQty.value.trim() || undefined,
    category: newItemCategory.value,
    supermarket: newItemSupermarket.value.trim() || undefined,
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
  // keep selected supermarket for easy batch adding
}

function selectSupermarketPreset(name: string) {
  if (newItemSupermarket.value === name) {
    newItemSupermarket.value = ''
  } else {
    newItemSupermarket.value = name
  }
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

const showTransferModal = ref(false)
const itemsToTransfer = ref<ShoppingItemForTransfer[]>([])
const transferFeedback = ref<{ message: string; type: 'success' | 'error' } | null>(null)

async function openTransferModal() {
  await pantryStore.fetchPantryItems().catch(() => {})
  const sourceItems = completedCount.value > 0
    ? items.value.filter(i => i.completed)
    : items.value

  if (sourceItems.length === 0) return

  itemsToTransfer.value = sourceItems.map(i => ({
    id: i.id,
    name: i.name,
    quantity: i.quantity,
    category: i.category,
    supermarket: i.supermarket
  }))
  showTransferModal.value = true
}

async function openTransferSingle(item: ShoppingItem) {
  await pantryStore.fetchPantryItems().catch(() => {})
  itemsToTransfer.value = [{
    id: item.id,
    name: item.name,
    quantity: item.quantity,
    category: item.category,
    supermarket: item.supermarket
  }]
  showTransferModal.value = true
}

async function handleTransferConfirm(payload: {
  transfers: Array<{
    shoppingId: string
    name: string
    quantity: number
    unit: any
    category: any
    location: any
    expirationDate?: string
    matchedItemId?: string
  }>
  removeTransferredFromShopping: boolean
}) {
  let added = 0
  let updated = 0

  for (const tr of payload.transfers) {
    try {
      if (tr.matchedItemId) {
        // Sums to existing item in pantry
        await pantryStore.adjustQuantity(tr.matchedItemId, tr.quantity)
        if (tr.expirationDate) {
          await pantryStore.updatePantryItem(tr.matchedItemId, {
            expiration_date: tr.expirationDate
          })
        }
        updated++
      } else {
        // Creates a new item in pantry
        await pantryStore.createPantryItem({
          name: tr.name,
          quantity: tr.quantity,
          unit: tr.unit,
          category: tr.category,
          location: tr.location,
          expiration_date: tr.expirationDate,
          min_threshold: tr.unit === 'g' || tr.unit === 'ml' ? 100 : 1
        })
        added++
      }

      if (payload.removeTransferredFromShopping) {
        await deleteItem(tr.shoppingId)
      }
    } catch (err) {
      console.error('Errore durante trasferimento in dispensa:', err)
    }
  }

  showTransferModal.value = false

  const parts: string[] = []
  if (added > 0) parts.push(`${added} nuov${added > 1 ? 'i' : 'o'}`)
  if (updated > 0) parts.push(`${updated} sommat${updated > 1 ? 'i' : 'o'}`)
  const summary = parts.length > 0 ? parts.join(', ') : 'Prodotti trasferiti'

  const msg = `Spesa registrata in Dispensa (${summary})!`
  transferFeedback.value = {
    message: msg,
    type: 'success'
  }
  setTimeout(() => {
    if (transferFeedback.value?.message === msg) {
      transferFeedback.value = null
    }
  }, 6000)
}

onMounted(async () => {
  try {
    const list = await apiClient.get<ShoppingItem[]>('/api/v1/shopping')
    items.value = list || []
  } catch {
    items.value = []
  }
  // Preload pantry items for instant matching
  pantryStore.fetchPantryItems().catch(() => {})
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
          <h1 class="page-title">{{ t('shopping.title') }}</h1>
          <p class="page-subtitle">{{ t('shopping.subtitle') }}</p>
        </div>
      </div>

      <div class="header-actions">
        <button
          v-if="items.length > 0"
          class="btn-transfer-pantry"
          @click="openTransferModal"
          :title="completedCount > 0 ? 'Sposta gli articoli acquistati nella dispensa' : 'Sposta articoli della spesa nella dispensa'"
        >
          <PackageCheck :size="16" />
          <span>{{ completedCount > 0 ? `Sposta Spesa in Dispensa (${completedCount})` : 'Sposta Spesa in Dispensa' }}</span>
        </button>

        <button
          v-if="completedCount > 0"
          class="btn-clear-completed"
          @click="clearCompleted"
        >
          <Trash2 :size="16" />
          <span>{{ t('shopping.clearCompleted') }} ({{ completedCount }})</span>
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
            :placeholder="t('shopping.placeholderName')"
            class="input-name"
            required
            aria-label="Nome Prodotto"
          />
          <input
            v-model="newItemQty"
            type="text"
            :placeholder="t('shopping.placeholderQty')"
            class="input-qty"
            aria-label="Quantità"
          />
          <div class="supermarket-input-wrap">
            <Store :size="16" class="store-input-icon" />
            <input
              v-model="newItemSupermarket"
              type="text"
              placeholder="Supermercato / Negozio"
              class="input-supermarket"
              list="supermarket-suggestions"
              aria-label="Supermercato o Negozio"
            />
            <datalist id="supermarket-suggestions">
              <option v-for="s in popularSupermarkets" :key="s" :value="s" />
            </datalist>
          </div>
          <button type="submit" class="btn-add" aria-label="Aggiungi alla lista">
            <Plus :size="18" />
            <span>{{ t('shopping.addBtn') }}</span>
          </button>
        </div>

        <!-- Supermarket Quick Chips -->
        <div class="supermarket-presets">
          <span class="preset-label">Supermercati rapidi:</span>
          <button
            v-for="s in popularSupermarkets"
            :key="s"
            type="button"
            class="store-chip"
            :class="{ active: newItemSupermarket === s }"
            @click="selectSupermarketPreset(s)"
          >
            <Building2 :size="12" />
            <span>{{ s }}</span>
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
      <!-- Status Filters -->
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
          {{ t('shopping.toBuy') }} ({{ pendingCount }})
        </button>
        <button
          :class="['pill-btn', { active: filterStatus === 'completed' }]"
          @click="filterStatus = 'completed'"
        >
          {{ t('shopping.completed') }} ({{ completedCount }})
        </button>
      </div>

      <!-- Supermarket Filter Dropdown / Chips -->
      <div v-if="availableSupermarkets.length > 0" class="supermarket-filter-bar">
        <span class="filter-icon-label">
          <Filter :size="14" />
          <span>Filtra per negozio:</span>
        </span>
        <button
          class="store-filter-chip"
          :class="{ active: filterSupermarket === 'all' }"
          @click="filterSupermarket = 'all'"
        >
          Tutti i negozi
        </button>
        <button
          v-for="s in availableSupermarkets"
          :key="s"
          class="store-filter-chip"
          :class="{ active: filterSupermarket.toLowerCase() === s.toLowerCase() }"
          @click="filterSupermarket = filterSupermarket.toLowerCase() === s.toLowerCase() ? 'all' : s"
        >
          <Building2 :size="12" />
          <span>{{ s }}</span>
        </button>
      </div>
    </div>

    <!-- Prompt Banner for Completed Items -->
    <div v-if="completedCount > 0" class="completed-prompt-banner glass-panel">
      <div class="prompt-left">
        <div class="prompt-icon-wrap">
          <PackageCheck :size="22" class="text-success" />
        </div>
        <div class="prompt-text">
          <h4 class="prompt-title">Hai completato {{ completedCount }} articol{{ completedCount > 1 ? 'i' : 'o' }} della spesa!</h4>
          <p class="prompt-sub">Spostali direttamente nella Dispensa: puoi sommare le quantità a quelle già presenti e impostare le scadenze dei cibi deperibili.</p>
        </div>
      </div>
      <button class="btn-prompt-transfer" @click="openTransferModal">
        <PackageCheck :size="16" />
        <span>Carica in Dispensa</span>
      </button>
    </div>

    <!-- Shopping Items List -->
    <div class="items-list">
      <div
        v-for="it in filteredItems"
        :key="it.id"
        :class="['item-card', 'glass-panel', { completed: it.completed }]"
      >
        <div class="item-left" @click="toggleItem(it)">
          <button class="checkbox-btn" :aria-label="`Segna ${it.name} come ${it.completed ? 'da comprare' : 'comprato'}`">
            <CheckCircle v-if="it.completed" :size="22" class="text-success check-icon" />
            <Circle v-else :size="22" class="text-muted uncheck-icon" />
          </button>

          <div class="item-info">
            <div class="name-qty-row">
              <span class="item-name">{{ it.name }}</span>
              <span v-if="it.quantity" class="item-qty">{{ it.quantity }}</span>

              <!-- Supermarket Tag -->
              <span
                v-if="it.supermarket"
                class="supermarket-tag"
                @click.stop="filterSupermarket = it.supermarket"
                :title="`Filtra per ${it.supermarket}`"
              >
                <Building2 :size="12" />
                <span>{{ it.supermarket }}</span>
              </span>
            </div>

            <div class="item-meta">
              <span class="cat-tag" :class="it.category">{{ it.category.replace('_', ' ') }}</span>
              <span class="added-tag">
                <User :size="12" />
                <span>{{ it.added_by }}</span>
              </span>
              <span v-if="it.completed && it.completed_by" class="completed-tag">
                {{ t('shopping.completedBy') }} {{ it.completed_by }}
              </span>
            </div>
          </div>
        </div>

        <div class="item-card-actions">
          <button
            class="btn-transfer-single"
            @click.stop="openTransferSingle(it)"
            title="Sposta questo articolo in Dispensa e imposta scadenza"
            aria-label="Sposta articolo in dispensa"
          >
            <PackageCheck :size="15" />
            <span class="btn-transfer-label">In Dispensa</span>
          </button>

          <button class="btn-delete" @click="deleteItem(it.id)" title="Rimuovi" aria-label="Rimuovi prodotto">
            <Trash2 :size="16" />
          </button>
        </div>
      </div>

      <!-- Empty State -->
      <div v-if="filteredItems.length === 0" class="empty-state glass-panel">
        <ShoppingCart :size="48" class="text-muted mb-2" />
        <h3>Nessun articolo nella lista</h3>
        <p class="text-dim">
          {{
            filterSupermarket !== 'all'
              ? `Nessun prodotto trovato per il supermercato "${filterSupermarket}".`
              : 'Tutti gli articoli sono stati acquistati o la lista è vuota.'
          }}
        </p>
      </div>
    </div>

    <!-- Transfer to Pantry Modal -->
    <TransferShoppingModal
      :is-open="showTransferModal"
      :shopping-items="itemsToTransfer"
      :existing-pantry-items="pantryStore.items"
      @close="showTransferModal = false"
      @confirm="handleTransferConfirm"
    />

    <!-- Toast Notification -->
    <Transition name="toast-slide">
      <div v-if="transferFeedback" class="transfer-feedback-toast glass-panel">
        <div class="toast-left">
          <PackageCheck :size="20" class="text-success" />
          <span>{{ transferFeedback.message }}</span>
        </div>
        <div class="toast-right">
          <button class="btn-goto-pantry" @click="router.push('/pantry')">
            Vai alla Dispensa &rarr;
          </button>
          <button class="btn-toast-close" @click="transferFeedback = null" aria-label="Chiudi notifica">
            &times;
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.shopping-view-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

.shopping-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-radius: 16px;
  background: var(--surface-card, rgba(15, 23, 42, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
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

.text-primary {
  color: var(--primary, #10b981);
}

.page-title {
  font-size: 1.4rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-main, #f8fafc);
}

.page-subtitle {
  font-size: 0.85rem;
  color: var(--text-muted, #94a3b8);
  margin: 4px 0 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.btn-transfer-pantry {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 8px;
  background: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.35);
  color: #10b981;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-transfer-pantry:hover {
  background: rgba(16, 185, 129, 0.25);
  border-color: #10b981;
  box-shadow: 0 0 12px rgba(16, 185, 129, 0.25);
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

.btn-clear-completed:hover {
  background: rgba(239, 68, 68, 0.25);
}

.add-bar-panel {
  padding: 20px;
  border-radius: 16px;
  background: var(--surface-card, rgba(15, 23, 42, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
}

.add-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.inputs-row {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.input-name {
  flex: 2;
  min-width: 200px;
  background: var(--bg-surface, rgba(255, 255, 255, 0.05));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.15));
  border-radius: 10px;
  padding: 12px 16px;
  color: var(--text-main);
  font-size: 0.95rem;
}

.input-qty {
  flex: 1;
  min-width: 100px;
  background: var(--bg-surface, rgba(255, 255, 255, 0.05));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.15));
  border-radius: 10px;
  padding: 12px 16px;
  color: var(--text-main);
  font-size: 0.95rem;
}

.supermarket-input-wrap {
  flex: 1.5;
  min-width: 200px;
  position: relative;
  display: flex;
  align-items: center;
}

.store-input-icon {
  position: absolute;
  left: 12px;
  color: var(--text-muted, #94a3b8);
  pointer-events: none;
}

.input-supermarket {
  width: 100%;
  background: var(--bg-surface, rgba(255, 255, 255, 0.05));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.15));
  border-radius: 10px;
  padding: 12px 16px 12px 38px;
  color: var(--text-main);
  font-size: 0.95rem;
}

.btn-add {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--primary, #3b82f6);
  border: none;
  border-radius: 10px;
  padding: 12px 24px;
  color: #fff;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
  white-space: nowrap;
}

.btn-add:hover {
  background: var(--primary-hover, #2563eb);
}

.supermarket-presets {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.preset-label {
  font-size: 0.75rem;
  color: #94a3b8;
  font-weight: 500;
}

.store-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.04);
  color: #cbd5e1;
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.2s;
}

.store-chip:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.2);
}

.store-chip.active {
  background: rgba(245, 158, 11, 0.2);
  border-color: #f59e0b;
  color: #fbbf24;
  font-weight: 600;
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
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  background: transparent;
  color: var(--text-muted, #94a3b8);
  font-size: 0.8rem;
  cursor: pointer;
  transition: all 0.2s;
}

.cat-pill:hover {
  background: rgba(255, 255, 255, 0.05);
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
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
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
  color: var(--text-muted, #94a3b8);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.pill-btn:hover {
  background: rgba(255, 255, 255, 0.08);
}

.pill-btn.active {
  background: var(--primary, #3b82f6);
  color: #fff;
}

.supermarket-filter-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.filter-icon-label {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.75rem;
  color: #94a3b8;
  margin-right: 4px;
}

.store-filter-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.04);
  color: #94a3b8;
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.2s;
}

.store-filter-chip:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}

.store-filter-chip.active {
  background: rgba(245, 158, 11, 0.2);
  border-color: #f59e0b;
  color: #fbbf24;
  font-weight: 600;
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
  background: var(--surface-card, rgba(15, 23, 42, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  transition: opacity 0.2s;
}

.item-card:hover {
  border-color: rgba(255, 255, 255, 0.2);
}

.item-card.completed {
  opacity: 0.6;
}

.item-card.completed .item-name {
  text-decoration: line-through;
  color: var(--text-muted, #94a3b8);
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

.text-success {
  color: #10b981;
}

.text-muted {
  color: #64748b;
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
  flex-wrap: wrap;
}

.item-name {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-main);
}

.item-qty {
  font-size: 0.85rem;
  color: #818cf8;
  background: rgba(99, 102, 241, 0.15);
  padding: 2px 8px;
  border-radius: 6px;
  font-weight: 600;
}

.supermarket-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.75rem;
  color: #fbbf24;
  background: rgba(245, 158, 11, 0.15);
  border: 1px solid rgba(245, 158, 11, 0.3);
  padding: 2px 8px;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.supermarket-tag:hover {
  background: rgba(245, 158, 11, 0.25);
  border-color: #f59e0b;
}

.item-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.75rem;
  color: var(--text-muted, #94a3b8);
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
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
  padding: 6px;
  border-radius: 6px;
  transition: color 0.2s;
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
  background: var(--surface-card, rgba(15, 23, 42, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
}

.text-dim {
  color: #94a3b8;
  font-size: 0.85rem;
}

.mb-2 {
  margin-bottom: 8px;
}

/* Prompt Banner for Completed Items */
.completed-prompt-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 14px 20px;
  border-radius: 12px;
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.12) 0%, rgba(15, 23, 42, 0.7) 100%);
  border: 1px solid rgba(16, 185, 129, 0.3);
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.2);
}

.prompt-left {
  display: flex;
  align-items: center;
  gap: 14px;
}

.prompt-icon-wrap {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: rgba(16, 185, 129, 0.18);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.prompt-title {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 600;
  color: #f8fafc;
}

.prompt-sub {
  margin: 2px 0 0;
  font-size: 0.8rem;
  color: #94a3b8;
}

.btn-prompt-transfer {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 8px;
  background: #10b981;
  border: none;
  color: #0f172a;
  font-size: 0.85rem;
  font-weight: 700;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s;
  min-height: 40px;
}

.btn-prompt-transfer:hover {
  background: #059669;
  box-shadow: 0 0 14px rgba(16, 185, 129, 0.4);
}

/* Item Card Actions */
.item-card-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-transfer-single {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border-radius: 6px;
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.25);
  color: #10b981;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  min-height: 32px;
}

.btn-transfer-single:hover {
  background: rgba(16, 185, 129, 0.2);
  border-color: #10b981;
  color: #34d399;
}

@media (max-width: 600px) {
  .btn-transfer-label {
    display: none;
  }
}

/* Toast Notification */
.transfer-feedback-toast {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: 1050;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 18px;
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.95);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(16, 185, 129, 0.4);
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.6), 0 0 15px rgba(16, 185, 129, 0.2);
  color: #f8fafc;
  max-width: 90vw;
}

.toast-left {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 0.88rem;
  font-weight: 500;
}

.toast-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.btn-goto-pantry {
  background: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.35);
  color: #10b981;
  padding: 5px 10px;
  border-radius: 6px;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
  white-space: nowrap;
}

.btn-goto-pantry:hover {
  background: rgba(16, 185, 129, 0.25);
}

.btn-toast-close {
  background: transparent;
  border: none;
  color: #94a3b8;
  font-size: 1.2rem;
  cursor: pointer;
  line-height: 1;
  padding: 0 4px;
}

.btn-toast-close:hover {
  color: #fff;
}

.toast-slide-enter-active,
.toast-slide-leave-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.toast-slide-enter-from,
.toast-slide-leave-to {
  opacity: 0;
  transform: translateY(20px) scale(0.95);
}
</style>
