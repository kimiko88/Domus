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
  PackageCheck,
  Receipt,
  X
} from '@lucide/vue'
import { apiClient } from '../api/client'
import { usePantryStore } from '../stores/pantry'
import TransferShoppingModal, { type ShoppingItemForTransfer } from '../components/pantry/TransferShoppingModal.vue'
import PantryReceiptScannerTab from '../components/pantry/PantryReceiptScannerTab.vue'
import type { Receipt as ReceiptData } from '../api/types'

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

async function toggleItem(it: ShoppingItem): Promise<boolean> {
  const previousCompleted = it.completed
  const previousCompletedBy = it.completed_by
  it.completed = !previousCompleted
  it.completed_by = it.completed ? 'Io' : undefined

  try {
    await apiClient.post(`/api/v1/shopping/${it.id}/toggle`, { completed_by: 'Io' })
    return true
  } catch {
    it.completed = previousCompleted
    it.completed_by = previousCompletedBy
    return false
  }
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

// --- RECEIPT SCANNER ---
const showReceiptModal = ref(false)
const isParsingReceipt = ref(false)
const selectedReceiptMatches = ref<Record<string, boolean>>({})

function openReceiptModal() {
  selectedReceiptMatches.value = {}
  pantryStore.clearParsedReceipt()
  showReceiptModal.value = true
}

function closeReceiptModal() {
  showReceiptModal.value = false
  selectedReceiptMatches.value = {}
  pantryStore.clearParsedReceipt()
}

function selectReceiptMatches(receipt: { items: Array<{ name: string }> }) {
  const matches: Record<string, boolean> = {}
  for (const receiptItem of receipt.items) {
    const receiptName = receiptItem.name.trim().toLowerCase()
    if (!receiptName) continue
    const match = items.value.find((item) => {
      const itemName = item.name.trim().toLowerCase()
      return !item.completed && (receiptName.includes(itemName) || itemName.includes(receiptName))
    })
    if (match) matches[match.id] = true
  }
  selectedReceiptMatches.value = matches
}

async function handleParseShoppingReceipt(rawText: string) {
  if (!rawText.trim()) return
  isParsingReceipt.value = true
  try {
    const receipt = await pantryStore.parseReceipt(rawText.trim())
    selectReceiptMatches(receipt)
  } catch (err) {
    console.error('Errore parsing scontrino:', err)
    transferFeedback.value = { message: 'Impossibile analizzare lo scontrino. Riprova.', type: 'error' }
  } finally {
    isParsingReceipt.value = false
  }
}

async function handleParseShoppingReceiptPDF(file: File) {
  isParsingReceipt.value = true
  try {
    const receipt = await pantryStore.parseReceiptPDF(file)
    selectReceiptMatches(receipt)
  } catch (err) {
    console.error('Errore lettura PDF:', err)
    transferFeedback.value = { message: 'Impossibile leggere il PDF. Verifica il file e riprova.', type: 'error' }
  } finally {
    isParsingReceipt.value = false
  }
}

const matchedShoppingItems = computed(() => {
  if (!pantryStore.parsedReceipt) return []
  const matches: { shoppingItem: ShoppingItem; receiptName: string; price: number }[] = []
  for (const receiptItem of pantryStore.parsedReceipt.items) {
    const receiptName = receiptItem.name.toLowerCase()
    const match = items.value.find((item) => {
      const itemName = item.name.toLowerCase()
      return !item.completed && (receiptName.includes(itemName) || itemName.includes(receiptName))
    })
    if (match) matches.push({ shoppingItem: match, receiptName: receiptItem.name, price: receiptItem.total_price })
  }
  return matches
})

async function handleImportShoppingReceipt(data: {
  store_name: string
  date: string
  items: ReceiptData['items']
  total_amount: number
  document_type: 'receipt' | 'invoice'
  auto_add_to_finance: boolean
  auto_update_pantry: boolean
}) {
  // Persist the receipt first; never mark items purchased if import failed.
  if (data.auto_add_to_finance || data.auto_update_pantry) {
    try {
      await pantryStore.importReceipt({
        ...data,
        date: data.date || new Date().toISOString().slice(0, 10)
      })
    } catch (err) {
      console.error('Errore import scontrino:', err)
      transferFeedback.value = { message: 'Importazione non riuscita: gli articoli della lista non sono stati modificati.', type: 'error' }
      return
    }
  }

  let count = 0
  let failedCount = 0
  if (data.document_type === 'receipt') {
    for (const [id, selected] of Object.entries(selectedReceiptMatches.value)) {
      if (!selected) continue
      const item = items.value.find((candidate) => candidate.id === id)
      if (item && !item.completed) {
        if (await toggleItem(item)) count++
        else failedCount++
      }
    }
  }

  closeReceiptModal()
  const message = failedCount > 0
    ? `Importazione completata; ${failedCount} articol${failedCount === 1 ? 'o' : 'i'} non sono stati aggiornati nella lista.`
    : data.document_type === 'invoice'
      ? (data.auto_add_to_finance ? 'Fattura registrata nel bilancio familiare.' : 'Fattura analizzata.')
      : `Scontrino applicato: ${count} articol${count === 1 ? 'o' : 'i'} segnati come acquistati.`
  transferFeedback.value = { message, type: failedCount > 0 ? 'error' : 'success' }
  setTimeout(() => {
    if (transferFeedback.value?.message === message) transferFeedback.value = null
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
          class="btn-scan-receipt"
          @click="openReceiptModal"
          title="Scansiona scontrino della spesa"
        >
          <Receipt :size="16" />
          <span>Scansiona Scontrino</span>
        </button>

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

    <!-- Receipt Scanner Modal -->
    <div
      v-if="showReceiptModal"
      class="receipt-modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="receipt-modal-title"
      @click.self="closeReceiptModal"
    >
      <div class="receipt-modal-card glass-panel" @click.stop>
        <header class="receipt-modal-header">
          <div class="modal-header-info">
            <div class="icon-circle-receipt">
              <Receipt :size="22" class="text-primary" />
            </div>
            <div>
              <h2 id="receipt-modal-title" class="receipt-modal-title">Scansiona Scontrino Spesa</h2>
              <p class="receipt-modal-sub">
                Scatta una foto, carica un’immagine o un PDF oppure incolla il testo. Verifica i prodotti prima di confermare.
              </p>
            </div>
          </div>
          <button class="btn-close-modal" type="button" @click="closeReceiptModal" aria-label="Chiudi scanner">
            <X :size="20" />
          </button>
        </header>

        <PantryReceiptScannerTab
          :parsed-receipt="pantryStore.parsedReceipt"
          :loading="isParsingReceipt || pantryStore.loading"
          @parse-receipt="handleParseShoppingReceipt"
          @parse-pdf="handleParseShoppingReceiptPDF"
          @import-receipt="handleImportShoppingReceipt"
          @clear-parsed="pantryStore.clearParsedReceipt"
        />

        <div v-if="pantryStore.parsedReceipt" class="shopping-receipt-matches">
          <h3>Confronto con la lista della spesa</h3>
          <p v-if="matchedShoppingItems.length === 0" class="text-dim">
            Nessuna corrispondenza automatica: puoi comunque verificare lo scontrino e confermare l’importazione.
          </p>
          <div v-else class="matched-checkboxes">
            <label v-for="match in matchedShoppingItems" :key="match.shoppingItem.id" class="matched-label">
              <input v-model="selectedReceiptMatches[match.shoppingItem.id]" type="checkbox" />
              <span>
                Segna come acquistato: <strong>{{ match.shoppingItem.name }}</strong>
                <small>(riconosciuto come “{{ match.receiptName }}”)</small>
              </span>
            </label>
          </div>
        </div>
      </div>
    </div>

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

.btn-scan-receipt {
  display: flex;
  align-items: center;
  gap: 8px;
  background: rgba(14, 165, 233, 0.15);
  color: #38bdf8;
  border: 1px solid rgba(14, 165, 233, 0.35);
  padding: 8px 14px;
  border-radius: 8px;
  font-size: 0.88rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-scan-receipt:hover {
  background: rgba(14, 165, 233, 0.25);
  border-color: rgba(14, 165, 233, 0.5);
  transform: translateY(-1px);
}

.receipt-modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 20px;
}

.receipt-modal-card {
  width: 100%;
  max-width: 680px;
  max-height: 85vh;
  overflow-y: auto;
  border-radius: 16px;
  background: var(--surface-card, #0f172a);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.receipt-modal-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.modal-header-info {
  display: flex;
  align-items: center;
  gap: 14px;
}

.icon-circle-receipt {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: rgba(14, 165, 233, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
}

.receipt-modal-title {
  font-size: 1.25rem;
  font-weight: 700;
  margin: 0;
  color: #f8fafc;
}

.receipt-modal-sub {
  font-size: 0.85rem;
  color: #94a3b8;
  margin: 4px 0 0;
}

.btn-close-modal {
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  transition: color 0.2s;
}

.btn-close-modal:hover {
  color: #fff;
}

.receipt-modal-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.samples-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.samples-label {
  font-size: 0.8rem;
  color: #94a3b8;
}

.chip-sample {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #cbd5e1;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 0.78rem;
  cursor: pointer;
  transition: background 0.2s;
}

.chip-sample:hover {
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
}

.chip-clear {
  background: transparent;
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #f87171;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 0.78rem;
  cursor: pointer;
}

.receipt-textarea {
  width: 100%;
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  border-radius: 10px;
  padding: 12px;
  color: #f8fafc;
  font-family: monospace;
  font-size: 0.85rem;
  resize: vertical;
}

.btn-run-parse {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: var(--primary, #10b981);
  color: #000;
  font-weight: 600;
  border: none;
  padding: 10px 18px;
  border-radius: 8px;
  cursor: pointer;
  transition: opacity 0.2s;
}

.btn-run-parse:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.parsed-results-panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  padding: 16px;
}

.parsed-meta-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 10px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.meta-store-name {
  font-weight: 700;
  color: #f8fafc;
  margin-right: 12px;
}

.meta-date {
  font-size: 0.85rem;
  color: #94a3b8;
}

.meta-total {
  font-size: 0.95rem;
  color: var(--primary, #10b981);
}

.matches-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: #e2e8f0;
  margin: 0 0 10px;
}

.parsed-items-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 180px;
  overflow-y: auto;
}

.parsed-item-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: rgba(0, 0, 0, 0.2);
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 0.85rem;
}

.parsed-item-name {
  font-weight: 500;
  color: #f8fafc;
  margin-right: 8px;
}

.parsed-item-qty {
  font-size: 0.78rem;
  color: #94a3b8;
}

.parsed-item-price {
  font-weight: 600;
  color: #cbd5e1;
}

.matched-box {
  margin-top: 12px;
  background: rgba(16, 185, 129, 0.08);
  border: 1px solid rgba(16, 185, 129, 0.25);
  border-radius: 8px;
  padding: 12px;
}

.matched-box-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.88rem;
  font-weight: 600;
  color: #10b981;
  margin: 0 0 8px;
}

.matched-checkboxes {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.matched-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  color: #f1f5f9;
  cursor: pointer;
}

.import-options-row {
  margin-top: 6px;
}

.checkbox-option {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  color: #94a3b8;
  cursor: pointer;
}

.modal-footer-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 10px;
}

.btn-cancel-modal {
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: #cbd5e1;
  padding: 8px 14px;
  border-radius: 8px;
  font-weight: 500;
  cursor: pointer;
}

.btn-apply-receipt {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--primary, #10b981);
  color: #000;
  font-weight: 600;
  border: none;
  padding: 8px 16px;
  border-radius: 8px;
  cursor: pointer;
}

.shopping-receipt-matches {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.025);
}

.shopping-receipt-matches h3 {
  font-size: 0.95rem;
  margin: 0;
}

.shopping-receipt-matches .matched-label small {
  display: block;
  color: var(--text-muted, #94a3b8);
  margin-top: 2px;
}

@media (max-width: 768px) {
  .shopping-header {
    align-items: flex-start;
    flex-direction: column;
    gap: 16px;
    padding: 16px;
  }

  .header-title-box {
    min-width: 0;
    width: 100%;
    align-items: flex-start;
  }

  .header-title-box > div:last-child {
    min-width: 0;
  }

  .page-title,
  .page-subtitle {
    overflow-wrap: anywhere;
  }

  .header-actions {
    width: 100%;
    align-items: stretch;
  }

  .header-actions > button {
    flex: 1 1 180px;
    justify-content: center;
    min-height: 44px;
  }

  .add-bar-panel { padding: 14px; }

  .inputs-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 0.65fr);
  }

  .inputs-row > *,
  .input-name,
  .input-qty,
  .input-supermarket {
    min-width: 0;
    width: 100%;
  }

  .supermarket-input-wrap {
    min-width: 0;
    grid-column: 1 / -1;
  }

  .btn-add { min-height: 44px; justify-content: center; }

  .filters-bar {
    align-items: stretch;
    flex-direction: column;
  }

  .filter-pills {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    width: 100%;
  }

  .pill-btn {
    min-width: 0;
    padding: 8px 6px;
    overflow-wrap: anywhere;
  }

  .supermarket-filter-bar { min-width: 0; }

  .completed-prompt-banner {
    align-items: stretch;
    flex-direction: column;
    padding: 14px;
  }

  .prompt-left { align-items: flex-start; }
  .btn-prompt-transfer { width: 100%; justify-content: center; min-height: 44px; }

  .item-card { align-items: flex-start; gap: 12px; padding: 12px; }
  .item-left { min-width: 0; align-items: flex-start; gap: 10px; }
  .item-info { min-width: 0; }
  .item-name { overflow-wrap: anywhere; }
  .item-card-actions { flex-shrink: 0; }

  .receipt-modal-backdrop {
    align-items: flex-start;
    padding: max(8px, env(safe-area-inset-top)) 8px max(8px, env(safe-area-inset-bottom));
    overflow-y: auto;
  }

  .receipt-modal-card {
    width: 100%;
    max-width: none;
    max-height: none;
    min-height: 0;
    padding: 14px;
    gap: 14px;
    margin: auto 0;
  }

  .receipt-modal-header,
  .modal-header-info { min-width: 0; }

  .modal-header-info { align-items: flex-start; gap: 10px; }
  .receipt-modal-title { font-size: 1.05rem; overflow-wrap: anywhere; }
  .receipt-modal-sub { line-height: 1.4; }

  .transfer-feedback-toast {
    left: 8px;
    right: 8px;
    bottom: max(8px, env(safe-area-inset-bottom));
    max-width: none;
    flex-wrap: wrap;
    gap: 10px;
    padding: 12px;
  }

  .toast-left { min-width: 0; flex: 1 1 100%; }
  .toast-right { width: 100%; justify-content: flex-end; }
}

@media (max-width: 420px) {
  .shopping-view-container { gap: 14px; }
  .icon-circle { width: 40px; height: 40px; flex: 0 0 40px; }
  .page-title { font-size: 1.15rem; line-height: 1.25; }
  .page-subtitle { font-size: 0.8rem; }
  .header-actions > button { flex-basis: 100%; }
  .filter-pills { grid-template-columns: 1fr; }
  .item-card { flex-wrap: wrap; }
  .item-left { flex: 1 1 100%; }
  .item-card-actions { width: 100%; justify-content: flex-end; }
  .empty-state { padding: 28px 16px; }
}

</style>
