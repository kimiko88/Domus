<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { History, Store, Pencil, Save, Plus, Trash2, X, FileText } from '@lucide/vue'
import { usePantryStore } from '../../stores/pantry'
import type { Receipt, ReceiptItem, PantryCategory, PantryUnit } from '../../api/types'

const { t } = useI18n()
const pantryStore = usePantryStore()
const isOpen = ref(true)
const isEditing = ref(false)
const isSaving = ref(false)
const loadError = ref('')
const saveError = ref('')
const draft = ref<Receipt | null>(null)

const receipts = computed(() => pantryStore.receiptHistory)
const categories: PantryCategory[] = ['produce', 'dairy', 'meat_fish', 'bakery', 'pantry', 'frozen', 'drinks', 'condiments', 'snacks', 'household', 'other']
const units: PantryUnit[] = ['pz', 'g', 'kg', 'ml', 'l', 'pack']

onMounted(async () => {
  try {
    await pantryStore.fetchReceiptHistory()
  } catch {
    loadError.value = t('shopping.receiptHistory.loadError')
  }
})

function editReceipt(receipt: Receipt) {
  draft.value = {
    ...receipt,
    items: receipt.items.map(item => ({ ...item }))
  }
  saveError.value = ''
  isEditing.value = true
}

function closeEditor() {
  if (isSaving.value) return
  isEditing.value = false
  draft.value = null
  saveError.value = ''
}

function addProduct() {
  if (!draft.value) return
  const item: ReceiptItem = {
    raw_line: '', name: t('shopping.receiptHistory.newProduct'), quantity: 1,
    unit: 'pz', unit_price: 0, total_price: 0, category: 'other'
  }
  draft.value.items.push(item)
}

function removeProduct(index: number) {
  if (!draft.value) return
  draft.value.items.splice(index, 1)
}

function normalizeDraftItem(item: ReceiptItem) {
  const quantity = Number(item.quantity)
  const total = Number(item.total_price)
  item.quantity = Number.isFinite(quantity) && quantity > 0 ? quantity : 0
  item.total_price = Number.isFinite(total) && total >= 0 ? total : 0
  item.unit_price = item.quantity > 0 ? Math.round((item.total_price / item.quantity) * 10000) / 10000 : 0
}

async function saveReceipt() {
  const receiptId = draft.value?.id
  if (!draft.value || !receiptId || isSaving.value) return
  const receipt = draft.value
  if (!receipt.store_name.trim() || !/^\d{4}-\d{2}-\d{2}$/.test(receipt.date)) {
    saveError.value = t('shopping.receiptHistory.validationError')
    return
  }
  const total = Number(receipt.total_amount)
  if (!Number.isFinite(total) || total < 0 || total > 1000000 || receipt.items.length > 500) {
    saveError.value = t('shopping.receiptHistory.validationError')
    return
  }
  for (const item of receipt.items) normalizeDraftItem(item)
  isSaving.value = true
  saveError.value = ''
  try {
    await pantryStore.updateReceipt(receiptId, {
      store_name: receipt.store_name.trim(),
      date: receipt.date,
      document_type: receipt.document_type === 'invoice' ? 'invoice' : 'receipt',
      total_amount: total,
      items: receipt.items
    })
    isEditing.value = false
    draft.value = null
  } catch (error) {
    console.error('Could not update saved receipt:', error)
    saveError.value = t('shopping.receiptHistory.saveError')
  } finally {
    isSaving.value = false
  }
}

function formatMoney(value: number) {
  return new Intl.NumberFormat(undefined, { style: 'currency', currency: 'EUR' }).format(value || 0)
}
</script>

<template>
  <section class="receipt-history glass-panel">
    <button class="receipt-history-toggle" type="button" :aria-expanded="isOpen" @click="isOpen = !isOpen">
      <span class="receipt-history-title"><History :size="19" />
        <span><strong>{{ t('shopping.receiptHistory.title') }}</strong><small>{{ t('shopping.receiptHistory.subtitle') }}</small></span>
      </span>
      <span class="receipt-history-count">{{ receipts.length }}</span>
    </button>
    <div v-if="isOpen" class="receipt-history-body">
      <p v-if="loadError" class="receipt-history-error" role="alert">{{ loadError }}</p>
      <p v-else-if="receipts.length === 0" class="receipt-history-empty">{{ t('shopping.receiptHistory.empty') }}</p>
      <article v-for="receipt in receipts" :key="receipt.id" class="receipt-history-row">
        <div class="receipt-history-main">
          <div class="receipt-history-store"><Store :size="17" />
            <div><strong>{{ receipt.store_name }}</strong><small>{{ receipt.date }} · {{ receipt.document_type === 'invoice' ? t('shopping.receiptHistory.invoice') : t('shopping.receiptHistory.receipt') }}</small></div>
          </div>
          <div class="receipt-history-total">{{ formatMoney(receipt.total_amount) }}</div>
          <button class="receipt-history-edit" type="button" @click="editReceipt(receipt)"><Pencil :size="15" /> {{ t('shopping.receiptHistory.edit') }}</button>
        </div>
        <div class="receipt-history-products">
          <span v-for="(item, index) in receipt.items" :key="`${receipt.id}-${index}`">{{ item.name }} · {{ item.quantity }} {{ item.unit }} · {{ formatMoney(item.total_price) }}</span>
        </div>
      </article>
    </div>
  </section>

  <div v-if="isEditing && draft" class="receipt-edit-backdrop" role="dialog" aria-modal="true" :aria-label="t('shopping.receiptHistory.editTitle')" @click.self="closeEditor">
    <form class="receipt-edit-card glass-panel" @submit.prevent="saveReceipt">
      <header class="receipt-edit-header">
        <div><h2>{{ t('shopping.receiptHistory.editTitle') }}</h2><p>{{ t('shopping.receiptHistory.editSubtitle') }}</p></div>
        <button class="receipt-edit-close" type="button" :disabled="isSaving" :aria-label="t('shopping.receiptHistory.cancel')" @click="closeEditor"><X :size="20" /></button>
      </header>
      <div class="receipt-edit-fields">
        <label>{{ t('shopping.receiptHistory.store') }}<input v-model="draft.store_name" type="text" maxlength="150" required /></label>
        <label>{{ t('shopping.receiptHistory.date') }}<input v-model="draft.date" type="date" required /></label>
        <label>{{ t('shopping.receiptHistory.type') }}<select v-model="draft.document_type"><option value="receipt">{{ t('shopping.receiptHistory.receipt') }}</option><option value="invoice">{{ t('shopping.receiptHistory.invoice') }}</option></select></label>
        <label>{{ t('shopping.receiptHistory.total') }} (€)<input v-model.number="draft.total_amount" type="number" min="0" max="1000000" step="0.01" required /></label>
      </div>
      <div class="receipt-edit-products-heading"><strong>{{ t('shopping.receiptHistory.products') }} ({{ draft.items.length }})</strong><button type="button" class="receipt-edit-add" @click="addProduct"><Plus :size="15" /> {{ t('shopping.receiptHistory.addProduct') }}</button></div>
      <div class="receipt-edit-products">
        <div v-for="(item, index) in draft.items" :key="index" class="receipt-edit-product">
          <label class="receipt-edit-product-name">{{ t('shopping.receiptHistory.product') }}<input v-model="item.name" type="text" maxlength="200" required /></label>
          <label>{{ t('shopping.receiptHistory.quantity') }}<input v-model.number="item.quantity" type="number" min="0.001" step="any" required @change="normalizeDraftItem(item)" /></label>
          <label>{{ t('shopping.receiptHistory.unit') }}<select v-model="item.unit"><option v-for="unit in units" :key="unit" :value="unit">{{ unit }}</option></select></label>
          <label>{{ t('shopping.receiptHistory.price') }} (€)<input v-model.number="item.total_price" type="number" min="0" step="0.01" @change="normalizeDraftItem(item)" /></label>
          <label>{{ t('shopping.receiptHistory.category') }}<select v-model="item.category"><option v-for="category in categories" :key="category" :value="category">{{ t(`shopping.receiptHistory.categories.${category}`) }}</option></select></label>
          <button type="button" class="receipt-edit-remove" :aria-label="t('shopping.receiptHistory.removeProduct')" @click="removeProduct(index)"><Trash2 :size="16" /></button>
        </div>
      </div>
      <p class="receipt-edit-note"><FileText :size="16" /> {{ t('shopping.receiptHistory.inventoryNote') }}</p>
      <p v-if="saveError" class="receipt-history-error" role="alert">{{ saveError }}</p>
      <footer class="receipt-edit-actions">
        <button type="button" class="receipt-edit-cancel" :disabled="isSaving" @click="closeEditor">{{ t('shopping.receiptHistory.cancel') }}</button>
        <button type="submit" class="receipt-edit-save" :disabled="isSaving"><Save :size="16" /> {{ isSaving ? t('shopping.receiptHistory.saving') : t('shopping.receiptHistory.save') }}</button>
      </footer>
    </form>
  </div>
</template>

<style scoped>
.receipt-history{padding:16px;margin-bottom:18px}
.receipt-history-toggle{display:flex;align-items:center;justify-content:space-between;width:100%;border:0;background:transparent;color:inherit;text-align:left;cursor:pointer;gap:12px}
.receipt-history-title{display:flex;align-items:center;gap:12px;min-width:0}
.receipt-history-title>span{display:grid;gap:3px;min-width:0}
.receipt-history-title small,.receipt-history-store small{color:var(--text-muted);font-size:.78rem}
.receipt-history-count{border-radius:999px;padding:4px 10px;background:var(--surface-2);font-variant-numeric:tabular-nums}
.receipt-history-body{display:grid;gap:10px;margin-top:14px}
.receipt-history-row{border:1px solid var(--border);border-radius:12px;padding:12px;min-width:0}
.receipt-history-main{display:flex;align-items:center;gap:12px;flex-wrap:wrap}
.receipt-history-store{display:flex;align-items:center;gap:9px;min-width:0;flex:1}
.receipt-history-store>div{display:grid;gap:3px;min-width:0}
.receipt-history-total{font-weight:700;font-variant-numeric:tabular-nums}
.receipt-history-edit,.receipt-edit-add{display:inline-flex;align-items:center;gap:6px;border:1px solid var(--border);border-radius:8px;background:var(--surface-2);color:inherit;padding:7px 10px;cursor:pointer}
.receipt-history-products{display:flex;flex-wrap:wrap;gap:6px;margin-top:9px}
.receipt-history-products span{font-size:.78rem;color:var(--text-muted);background:var(--surface-2);border-radius:6px;padding:4px 7px}
.receipt-history-empty,.receipt-history-error{padding:12px;border-radius:8px;background:var(--surface-2);color:var(--text-muted)}
.receipt-history-error{color:var(--danger,#dc2626)}
.receipt-edit-backdrop{position:fixed;inset:0;z-index:1200;background:rgba(0,0,0,.65);display:grid;place-items:center;padding:16px;overflow-y:auto}
.receipt-edit-card{width:min(980px,100%);max-height:min(92dvh,1000px);overflow:auto;padding:20px;display:grid;gap:16px}
.receipt-edit-header{display:flex;justify-content:space-between;gap:12px;align-items:flex-start}
.receipt-edit-header h2{margin:0}.receipt-edit-header p{margin:5px 0 0;color:var(--text-muted);font-size:.88rem}
.receipt-edit-close,.receipt-edit-remove{border:0;background:var(--surface-2);color:inherit;border-radius:8px;padding:8px;cursor:pointer}
.receipt-edit-fields{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px}
.receipt-edit-fields label,.receipt-edit-product label{display:grid;gap:5px;min-width:0;font-size:.8rem;color:var(--text-muted)}
.receipt-edit-fields input,.receipt-edit-fields select,.receipt-edit-product input,.receipt-edit-product select{width:100%;min-width:0;box-sizing:border-box;border:1px solid var(--border);border-radius:8px;padding:9px;background:var(--surface-1);color:inherit;font:inherit}
.receipt-edit-products-heading{display:flex;align-items:center;justify-content:space-between;gap:10px;flex-wrap:wrap}
.receipt-edit-products{display:grid;gap:8px}
.receipt-edit-product{display:grid;grid-template-columns:minmax(140px,2fr) minmax(85px,.8fr) minmax(70px,.7fr) minmax(90px,.9fr) minmax(120px,1.1fr) 36px;align-items:end;gap:8px;padding:10px;border:1px solid var(--border);border-radius:10px}
.receipt-edit-note{display:flex;gap:8px;align-items:flex-start;color:var(--text-muted);font-size:.82rem;margin:0}
.receipt-edit-actions{display:flex;justify-content:flex-end;gap:10px;flex-wrap:wrap}
.receipt-edit-cancel,.receipt-edit-save{display:inline-flex;align-items:center;justify-content:center;gap:7px;border:1px solid var(--border);border-radius:9px;padding:9px 14px;background:var(--surface-2);color:inherit;cursor:pointer}
.receipt-edit-save{background:var(--primary,#0ea5e9);color:white;border-color:transparent}
@media(max-width:760px){.receipt-edit-fields{grid-template-columns:repeat(2,minmax(0,1fr))}.receipt-edit-product{grid-template-columns:repeat(2,minmax(0,1fr))}.receipt-edit-product-name{grid-column:1/-1}.receipt-edit-remove{justify-self:end}}
@media(max-width:420px){.receipt-edit-card{padding:14px}.receipt-edit-fields{grid-template-columns:1fr}.receipt-history-main{align-items:flex-start}.receipt-history-edit{margin-left:auto}}
</style>