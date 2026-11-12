<script setup lang="ts">
import { ref } from 'vue'
import {
  FileText,
  Sparkles,
  ShoppingBag,
  CheckCircle,
  Plus,
  Trash2,
  Calendar,
  DollarSign,
  AlertCircle,
  PackageCheck
} from '@lucide/vue'
import type { Receipt, ReceiptItem, PantryCategory, PantryUnit } from '../../api/types'

interface Props {
  parsedReceipt: Receipt | null
  loading?: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'parse-receipt', rawText: string): void
  (e: 'parse-pdf', file: File): void
  (e: 'parse-image', file: File): void
  (e: 'import-receipt', data: {
    store_name: string
    date: string
    items: ReceiptItem[]
    total_amount: number
    document_type: 'receipt' | 'invoice'
    auto_add_to_finance: boolean
    auto_update_pantry: boolean
  }): void
  (e: 'clear-parsed'): void
}>()

const rawReceiptText = ref('')
const documentType = ref<'receipt' | 'invoice'>('receipt')
const ocrStatus = ref('')
const maxImageBytes = 10 * 1024 * 1024
const maxPDFBytes = 3 * 1024 * 1024
const autoAddToFinance = ref(true)
const autoUpdatePantry = ref(true)

const categories: { key: PantryCategory; label: string }[] = [
  { key: 'produce', label: '🍎 Ortofrutta' },
  { key: 'dairy', label: '🧀 Latticini' },
  { key: 'meat_fish', label: '🥩 Carne/Pesce' },
  { key: 'bakery', label: '🥖 Forno' },
  { key: 'pantry', label: '🍝 Dispensa' },
  { key: 'frozen', label: '❄️ Surgelati' },
  { key: 'drinks', label: '🥤 Bevande' },
  { key: 'condiments', label: '🫒 Condimenti' },
  { key: 'snacks', label: '🍪 Snack' },
  { key: 'household', label: '🧼 Casa' },
  { key: 'other', label: '📦 Altro' }
]

const sampleReceiptEsselunga = `ESSELUNGA S.P.A.
VIA PISACANE 12 - MILANO
DATA 18/10/2026 ORA 19:15
PASTA PENNE RIGATE 500G 1.30
RISO CARNAROLI 1KG 2.79
LATTE PARZ SCREM 1L 1.49
MELE GOLDEN KG 1.500 2.99
MACINATO MANZO 450G 4.90
ACQUA NATURALE 6X1.5L 2.10
SUBTOTALE EURO 15.57
TOTALE COMPLESSIVO EURO 15.57
PAGAMENTO POS 15.57`

const sampleReceiptCoop = `COOP LOMBARDIA
DATA 17/10/2026
PANE INTEGRALE 500G 1.80
YOGURT GRECO BIANCO 2PZ 2.20
PASSATA DI POMODORO 700G 0.99
BANANE BIO KG 1.200 2.40
TOTALE EURO 7.39`

function loadSample(sample: string) {
  rawReceiptText.value = sample
}

async function handleImageSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return

  if (file.type === 'application/pdf' || file.name.toLowerCase().endsWith('.pdf')) {
    if (file.size > maxPDFBytes) {
      ocrStatus.value = 'PDF troppo grande: il limite è 3 MB.'
      return
    }
    ocrStatus.value = 'Invio del PDF al server Domus per estrarre il testo (massimo 20 pagine)…'
    emit('parse-pdf', file)
    return
  }
  if (!file.type.startsWith('image/')) {
    ocrStatus.value = 'Seleziona un’immagine JPG/PNG oppure un PDF.'
    return
  }
  if (file.size > maxImageBytes) {
    ocrStatus.value = 'Immagine troppo grande: il limite è 10 MB.'
    return
  }

  const detectorWindow = window as unknown as {
    TextDetector?: new () => { detect: (image: ImageBitmap) => Promise<Array<{ rawValue?: string }>> }
  }
  if (!detectorWindow.TextDetector || typeof createImageBitmap !== 'function') {
    ocrStatus.value = 'OCR locale non supportato: invio della fotografia al server Domus…'
    emit('parse-image', file)
    return
  }

  let bitmap: ImageBitmap | undefined
  try {
    ocrStatus.value = 'Lettura locale dell’immagine in corso…'
    bitmap = await createImageBitmap(file)
    const detector = new detectorWindow.TextDetector()
    const blocks = await detector.detect(bitmap)
    const extracted = blocks.map((block) => block.rawValue?.trim()).filter(Boolean).join('\n')
    if (!extracted) {
      ocrStatus.value = 'Nessun testo riconosciuto localmente: provo con l’OCR del server Domus…'
      emit('parse-image', file)
      return
    }
    rawReceiptText.value = extracted
    ocrStatus.value = 'Testo estratto sul dispositivo. Controllalo e avvia l’analisi.'
  } catch {
    ocrStatus.value = 'OCR locale non riuscito: invio della fotografia al server Domus…'
    emit('parse-image', file)
  } finally {
    bitmap?.close()
  }
}

function handleParse() {
  if (!rawReceiptText.value.trim()) return
  emit('parse-receipt', rawReceiptText.value.trim())
}

function removeReceiptItem(index: number) {
  if (!props.parsedReceipt) return
  props.parsedReceipt.items.splice(index, 1)
  // Recalculate total
  const sum = props.parsedReceipt.items.reduce((acc, curr) => acc + (curr.total_price || 0), 0)
  props.parsedReceipt.total_amount = Math.round(sum * 100) / 100
}

function handleConfirmImport() {
  if (!props.parsedReceipt) return
  emit('import-receipt', {
    store_name: props.parsedReceipt.store_name,
    date: props.parsedReceipt.date || new Date().toISOString().slice(0, 10),
    items: props.parsedReceipt.items,
    total_amount: props.parsedReceipt.total_amount,
    document_type: documentType.value,
    auto_add_to_finance: autoAddToFinance.value,
    auto_update_pantry: documentType.value === 'receipt' && autoUpdatePantry.value
  })
}
</script>

<template>
  <div class="scanner-tab-content">
    <!-- Step 1: Input Raw Text / Paste Area -->
    <div v-if="!parsedReceipt" class="scanner-input-card glass-panel">
      <div class="document-type-control">
        <label for="document-type">Tipo documento</label>
        <select id="document-type" v-model="documentType" class="table-select">
          <option value="receipt">Scontrino</option>
          <option value="invoice">Fattura</option>
        </select>
      </div>

      <div class="card-intro">
        <div class="intro-icon bg-primary-soft">
          <FileText :size="24" class="text-primary" />
        </div>
        <div>
          <h3 class="intro-title">Lettura intelligente di scontrini e fatture</h3>
          <p class="intro-desc">
            Carica una foto oppure incolla il testo di uno scontrino o di una fattura. Verifica sempre i dati estratti prima dell’importazione.
          </p>
        </div>
      </div>

      <div class="image-scan-controls">
        <label class="btn btn-secondary" for="receipt-image-input">Scatta o carica immagine / PDF</label>
        <input
          id="receipt-image-input"
          class="visually-hidden"
          type="file"
          accept="image/*,.pdf,application/pdf"
          capture="environment"
          aria-label="Scatta o seleziona un'immagine di scontrino o fattura"
          @change="handleImageSelected"
        />
        <p v-if="ocrStatus" class="ocr-status" role="status" aria-live="polite">{{ ocrStatus }}</p>
        <p class="ocr-privacy-note">Le immagini vengono lette localmente quando possibile; sui browser non compatibili vengono inviate al server Domus per l’OCR. PDF e immagini vengono elaborati temporaneamente e non salvati.</p>
      </div>

      <div class="quick-samples">
        <span class="samples-label">Oppure prova un esempio reale:</span>
        <button
          type="button"
          class="sample-btn"
          @click="loadSample(sampleReceiptEsselunga)"
        >
          🛒 Scontrino Esselunga
        </button>
        <button
          type="button"
          class="sample-btn"
          @click="loadSample(sampleReceiptCoop)"
        >
          🏪 Scontrino Coop
        </button>
      </div>

      <div class="textarea-wrap">
        <textarea
          v-model="rawReceiptText"
          rows="9"
          class="receipt-textarea"
          placeholder="Incolla qui il testo dello scontrino... (es. PASTA 500G 1.29€, MELE 1.5KG 2.99€, TOTALE 15.57€)"
          aria-label="Testo scontrino"
        ></textarea>
      </div>

      <div class="input-actions">
        <button
          type="button"
          class="btn btn-primary"
          :disabled="!rawReceiptText.trim() || loading"
          @click="handleParse"
        >
          <Sparkles :size="18" />
          <span>{{ loading ? 'Analisi in corso...' : 'Analizza documento' }}</span>
        </button>
      </div>
    </div>

    <!-- Step 2: Parsed Receipt Preview & Confirm -->
    <div v-else class="scanner-preview-card glass-panel">
      <div class="preview-header">
        <div class="header-left">
          <div class="store-badge">
            <ShoppingBag :size="20" class="text-primary" />
            <input
              v-model="parsedReceipt.store_name"
              class="store-name-input"
              title="Nome supermercato"
            />
          </div>

          <div class="date-badge">
            <Calendar :size="16" class="text-muted" />
            <input
              v-model="parsedReceipt.date"
              type="date"
              class="date-input"
              title="Data spesa"
            />
          </div>
        </div>

        <div class="header-right">
          <div class="total-badge">
            <span class="total-label">Totale Spesa:</span>
            <span class="total-amount">{{ parsedReceipt.total_amount.toFixed(2) }} €</span>
          </div>

          <button
            type="button"
            class="btn-ghost"
            title="Annulla e ricarica"
            @click="emit('clear-parsed')"
          >
            Rianalizza
          </button>
        </div>
      </div>

      <!-- Items Table -->
      <div class="items-table-wrap">
        <table class="receipt-table" role="table">
          <thead>
            <tr>
              <th>Prodotto Riconosciuto</th>
              <th>Quantità</th>
              <th>Unità</th>
              <th>Prezzo (€)</th>
              <th>Categoria Assegnata</th>
              <th aria-label="Azioni"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(it, idx) in parsedReceipt.items" :key="idx">
              <td>
                <input v-model="it.name" class="table-input item-name" />
              </td>
              <td class="qty-col">
                <input
                  v-model.number="it.quantity"
                  type="number"
                  step="any"
                  min="0"
                  class="table-input item-qty"
                />
              </td>
              <td class="unit-col">
                <select v-model="it.unit" class="table-select">
                  <option value="pz">pz</option>
                  <option value="g">g</option>
                  <option value="kg">kg</option>
                  <option value="ml">ml</option>
                  <option value="l">L</option>
                  <option value="pack">pack</option>
                </select>
              </td>
              <td class="price-col">
                <input
                  v-model.number="it.total_price"
                  type="number"
                  step="0.01"
                  min="0"
                  class="table-input item-price"
                />
              </td>
              <td>
                <select v-model="it.category" class="table-select cat-select">
                  <option v-for="c in categories" :key="c.key" :value="c.key">
                    {{ c.label }}
                  </option>
                </select>
              </td>
              <td class="action-col">
                <button
                  type="button"
                  class="table-del-btn"
                  title="Rimuovi prodotto"
                  aria-label="Rimuovi prodotto"
                  @click="removeReceiptItem(idx)"
                >
                  <Trash2 :size="15" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Auto Integration Switches -->
      <div class="integration-switches-box">
        <label class="switch-row" :class="{ 'switch-disabled': documentType === 'invoice' }">
          <input v-model="autoUpdatePantry" type="checkbox" :disabled="documentType === 'invoice'" />
          <div class="switch-info">
            <span class="switch-title">📦 Incrementa automaticamente le scorte in Dispensa</span>
            <span class="switch-desc">{{ documentType === 'invoice' ? 'Non disponibile per le fatture.' : 'I prodotti riconosciuti vengono aggiunti o incrementati nel magazzino di casa.' }}</span>
          </div>
        </label>

        <label class="switch-row">
          <input v-model="autoAddToFinance" type="checkbox" />
          <div class="switch-info">
            <span class="switch-title">💰 Registra automaticamente la spesa in Home Finance</span>
            <span class="switch-desc">Crea un record di uscita nella categoria "Cibo & Alimentari" con il totale dello scontrino.</span>
          </div>
        </label>
      </div>

      <!-- Confirm Action -->
      <div class="confirm-bar">
        <button
          type="button"
          class="btn btn-secondary"
          @click="emit('clear-parsed')"
        >
          Annulla
        </button>

        <button
          type="button"
          class="btn btn-primary btn-large"
          :disabled="parsedReceipt.items.length === 0 || loading"
          @click="handleConfirmImport"
        >
          <PackageCheck :size="20" />
          <span>{{ loading ? 'Importazione in corso...' : 'Conferma e Importa in Casa' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.document-type-control { display: flex; align-items: center; gap: 10px; }
.document-type-control label { color: var(--text-muted, #94a3b8); font-size: 0.88rem; }
.switch-disabled { opacity: 0.65; }
.image-scan-controls {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}
.visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
.ocr-status, .ocr-privacy-note {
  margin: 0;
  color: var(--text-muted, #94a3b8);
  font-size: 0.82rem;
}
.ocr-status { color: var(--text-main, #f8fafc); }

.scanner-tab-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 16px);
}

.scanner-input-card,
.scanner-preview-card {
  padding: var(--space-5, 20px);
  border-radius: var(--radius-xl, 20px);
  background: var(--surface-card, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 16px);
}

.card-intro {
  display: flex;
  align-items: center;
  gap: var(--space-4, 16px);
}

.intro-icon {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.bg-primary-soft {
  background: rgba(59, 130, 246, 0.15);
}
.text-primary {
  color: var(--primary, #3b82f6);
}

.intro-title {
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
  margin: 0 0 4px 0;
}

.intro-desc {
  font-size: 0.88rem;
  color: var(--text-muted, #94a3b8);
  margin: 0;
  line-height: 1.4;
}

.quick-samples {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.samples-label {
  font-size: 0.8rem;
  color: var(--text-muted, #94a3b8);
}

.sample-btn {
  padding: 4px 10px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  color: var(--text-main, #f8fafc);
  font-size: 0.78rem;
  cursor: pointer;
  transition: all 0.2s;
}
.sample-btn:hover {
  background: rgba(255, 255, 255, 0.12);
}

.textarea-wrap {
  position: relative;
}

.receipt-textarea {
  width: 100%;
  border-radius: var(--radius-md, 12px);
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  padding: 14px;
  color: var(--text-main, #f8fafc);
  font-family: monospace;
  font-size: 0.9rem;
  line-height: 1.5;
  resize: vertical;
}
.receipt-textarea:focus {
  outline: none;
  border-color: var(--primary, #3b82f6);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.25);
}

.input-actions {
  display: flex;
  justify-content: flex-end;
}

.preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  flex-wrap: wrap;
  gap: 12px;
}

.header-left,
.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.store-badge,
.date-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
}

.store-name-input {
  background: transparent;
  border: none;
  color: var(--text-main, #f8fafc);
  font-size: 1rem;
  font-weight: 600;
  width: 140px;
}

.date-input {
  background: transparent;
  border: none;
  color: var(--text-main, #f8fafc);
  font-size: 0.9rem;
}

.total-badge {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 6px 14px;
  border-radius: 8px;
  background: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.total-label {
  font-size: 0.85rem;
  color: var(--text-muted, #94a3b8);
}

.total-amount {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--success, #10b981);
}

.items-table-wrap {
  overflow-x: auto;
}

.receipt-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}

.receipt-table th {
  padding: 10px 12px;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-muted, #94a3b8);
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
}

.receipt-table td {
  padding: 8px 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}

.table-input {
  height: 36px;
  padding: 0 10px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  color: var(--text-main, #f8fafc);
  font-size: 0.88rem;
  width: 100%;
}

.item-name {
  min-width: 180px;
}
.item-qty {
  width: 80px;
}
.item-price {
  width: 90px;
}

.table-select {
  height: 36px;
  padding: 0 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  color: var(--text-main, #f8fafc);
  font-size: 0.85rem;
  color-scheme: dark;
}

.table-select option {
  background-color: #1e293b !important;
  color: #f8fafc !important;
}

.table-del-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  background: transparent;
  border: none;
  color: var(--danger, #ef4444);
  cursor: pointer;
  border-radius: 6px;
}
.table-del-btn:hover {
  background: rgba(239, 68, 68, 0.15);
}

.integration-switches-box {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
}

.switch-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  cursor: pointer;
}

.switch-row input {
  margin-top: 3px;
  width: 18px;
  height: 18px;
}

.switch-info {
  display: flex;
  flex-direction: column;
}

.switch-title {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
}

.switch-desc {
  font-size: 0.78rem;
  color: var(--text-muted, #94a3b8);
}

.confirm-bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 8px;
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

.btn-large {
  padding: 0 28px;
  font-size: 1rem;
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-main, #f8fafc);
}
.btn-primary {
  background: var(--primary, #3b82f6);
  color: #fff;
}
.btn-primary:hover {
  background: #2563eb;
}
.btn-ghost {
  background: transparent;
  border: none;
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
  font-size: 0.85rem;
}

@media (max-width: 640px) {
  .scanner-input-card,
  .scanner-preview-card { padding: 12px; border-radius: 14px; }
  .card-intro { align-items: flex-start; gap: 10px; }
  .intro-icon { width: 42px; height: 42px; flex: 0 0 42px; }
  .intro-title { font-size: 1rem; }
  .image-scan-controls,
  .image-scan-controls .btn { width: 100%; }
  .image-scan-controls .btn { justify-content: center; min-height: 44px; text-align: center; }
  .input-actions,
  .input-actions .btn { width: 100%; }
  .input-actions .btn { justify-content: center; min-height: 44px; }
  .preview-header,
  .header-left,
  .header-right { width: 100%; align-items: stretch; flex-direction: column; }
  .store-badge,
  .date-badge,
  .total-badge { width: 100%; min-width: 0; }
  .store-name-input,
  .date-input { min-width: 0; max-width: 100%; }
  .items-table-wrap { max-width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .receipt-table { min-width: 700px; }
  .integration-switches-box { padding: 10px; }
  .confirm-bar { flex-direction: column-reverse; align-items: stretch; }
  .confirm-bar .btn { width: 100%; justify-content: center; min-height: 44px; }
  .modal-footer-actions { flex-direction: column-reverse; }
  .modal-footer-actions > button { width: 100%; min-height: 44px; }
}

@media (max-width: 420px) {
  .scanner-input-card,
  .scanner-preview-card { padding: 10px; }
  .quick-samples { align-items: stretch; }
  .sample-btn { min-height: 40px; }
  .receipt-textarea { font-size: 16px; }
  .preview-header { gap: 8px; }
}
</style>
