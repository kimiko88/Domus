<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  Pill,
  Plus,
  Search,
  Sparkles,
  AlertTriangle,
  AlertCircle,
  ThermometerSnowflake,
  Clock,
  Calendar,
  Trash2,
  Edit2,
  Check,
  X,
  Package,
  Layers,
  ShieldCheck,
  CheckCircle2,
  MinusCircle
} from '@lucide/vue'
import { useMedicineStore } from '../stores/medicine'
import type { Medicine, MedicineLocation, MedicineUnit } from '../api/types'

const medicineStore = useMedicineStore()

// --- MODAL STATE ---
const showAddModal = ref(false)
const showEditModal = ref(false)
const isQuickScanMode = ref(true)

// Datamatrix code input
const rawDatamatrixCode = ref('')
const parseError = ref<string | null>(null)

// Form fields
const editingId = ref<string | null>(null)
const formName = ref('')
const formActiveIngredient = ref('')
const formDosage = ref('')
const formForm = ref('')
const formAIC = ref('')
const formBarcode = ref('')
const formLotNumber = ref('')
const formExpirationDate = ref('')
const formPaoDays = ref(0)
const formIsOpened = ref(false)
const formOpenedAt = ref('')
const formLocation = ref<MedicineLocation>('armadietto_bagno')
const formRequiresColdChain = ref(false)
const formQuantity = ref(1)
const formUnit = ref<MedicineUnit>('compresse')
const formMinThreshold = ref(0)
const formPrescriptionRequired = ref(false)
const formNotes = ref('')

// AIFA Autocomplete state
const aifaSearchInput = ref('')
const showAifaSuggestions = ref(false)

const locations: { key: MedicineLocation; label: string }[] = [
  { key: 'armadietto_bagno', label: 'Armadietto Bagno' },
  { key: 'cassetto_notte', label: 'Cassetto Notte' },
  { key: 'borsa_viaggio', label: 'Borsa / Viaggio' },
  { key: 'frigo', label: 'Frigorifero (2-8°C)' },
  { key: 'altro', label: 'Altro' }
]

const units: { key: MedicineUnit; label: string }[] = [
  { key: 'compresse', label: 'Compresse' },
  { key: 'bustine', label: 'Bustine' },
  { key: 'gocce', label: 'Gocce' },
  { key: 'ml', label: 'ml' },
  { key: 'pz', label: 'Pezzi' },
  { key: 'confezione', label: 'Confezione' }
]

// Sample Datamatrix codes for quick demo/testing
const sampleCodes = [
  {
    label: 'Tachipirina 1000 mg (GS1)',
    code: '(01)08003423403501(17)280430(10)LOT789'
  },
  {
    label: 'Tobral Collirio Frigo (GS1)',
    code: '(01)08002564701201(17)271231(10)LOT332'
  },
  {
    label: 'Augmentin 1g (AIC diretto)',
    code: '026089073'
  }
]

function openAddModal() {
  resetForm()
  showAddModal.value = true
}

function resetForm() {
  editingId.value = null
  formName.value = ''
  formActiveIngredient.value = ''
  formDosage.value = ''
  formForm.value = ''
  formAIC.value = ''
  formBarcode.value = ''
  formLotNumber.value = ''
  formExpirationDate.value = ''
  formPaoDays.value = 0
  formIsOpened.value = false
  formOpenedAt.value = ''
  formLocation.value = 'armadietto_bagno'
  formRequiresColdChain.value = false
  formQuantity.value = 1
  formUnit.value = 'compresse'
  formMinThreshold.value = 0
  formPrescriptionRequired.value = false
  formNotes.value = ''
  rawDatamatrixCode.value = ''
  parseError.value = null
  aifaSearchInput.value = ''
  showAifaSuggestions.value = false
}

async function handleParseDatamatrix() {
  if (!rawDatamatrixCode.value.trim()) return
  parseError.value = null
  try {
    const res = await medicineStore.parseCode(rawDatamatrixCode.value.trim())
    if (res.expiration_date) formExpirationDate.value = res.expiration_date
    if (res.lot_number) formLotNumber.value = res.lot_number
    if (res.aic) {
      formAIC.value = res.aic
      // Lookup in AIFA catalog
      const aifaMatches = await medicineStore.lookupAIFA('', res.aic)
      if (aifaMatches && aifaMatches.length > 0) {
        applyAifaRecord(aifaMatches[0])
      }
    }
  } catch (err: any) {
    parseError.value = 'Codice non riconosciuto o non valido'
  }
}

function loadSampleCode(code: string) {
  rawDatamatrixCode.value = code
  handleParseDatamatrix()
}

async function onAifaInput() {
  if (aifaSearchInput.value.trim().length >= 2) {
    await medicineStore.lookupAIFA(aifaSearchInput.value.trim())
    showAifaSuggestions.value = true
  } else {
    showAifaSuggestions.value = false
  }
}

function applyAifaRecord(rec: any) {
  formName.value = rec.name
  formActiveIngredient.value = rec.active_ingredient
  formDosage.value = rec.dosage
  formForm.value = rec.form
  formAIC.value = rec.aic
  formUnit.value = rec.unit || 'compresse'
  if (rec.default_quantity) formQuantity.value = rec.default_quantity
  if (rec.requires_cold_chain) {
    formRequiresColdChain.value = true
    formLocation.value = 'frigo'
  }
  if (rec.default_pao_days) formPaoDays.value = rec.default_pao_days
  formPrescriptionRequired.value = rec.prescription_required || false
  showAifaSuggestions.value = false
  aifaSearchInput.value = rec.name
}

async function submitMedicineForm() {
  const payload: Partial<Medicine> = {
    name: formName.value.trim(),
    active_ingredient: formActiveIngredient.value.trim(),
    dosage: formDosage.value.trim(),
    form: formForm.value.trim(),
    aic: formAIC.value.trim(),
    barcode: formBarcode.value.trim() || rawDatamatrixCode.value.trim(),
    lot_number: formLotNumber.value.trim(),
    expiration_date: formExpirationDate.value,
    pao_days: formPaoDays.value || 0,
    opened_at: formIsOpened.value ? (formOpenedAt.value || new Date().toISOString().slice(0, 10)) : undefined,
    location: formLocation.value,
    requires_cold_chain: formRequiresColdChain.value,
    quantity: formQuantity.value,
    unit: formUnit.value,
    min_threshold: formMinThreshold.value || 0,
    prescription_required: formPrescriptionRequired.value,
    notes: formNotes.value.trim()
  }

  if (editingId.value) {
    await medicineStore.updateMedicine(editingId.value, payload)
    showEditModal.value = false
  } else {
    await medicineStore.createMedicine(payload)
    showAddModal.value = false
  }
  resetForm()
}

function openEditModal(med: Medicine) {
  resetForm()
  editingId.value = med.id
  formName.value = med.name
  formActiveIngredient.value = med.active_ingredient || ''
  formDosage.value = med.dosage || ''
  formForm.value = med.form || ''
  formAIC.value = med.aic || ''
  formBarcode.value = med.barcode || ''
  formLotNumber.value = med.lot_number || ''
  formExpirationDate.value = med.expiration_date
  formPaoDays.value = med.pao_days
  formIsOpened.value = !!med.opened_at
  formOpenedAt.value = med.opened_at || ''
  formLocation.value = med.location
  formRequiresColdChain.value = med.requires_cold_chain
  formQuantity.value = med.quantity
  formUnit.value = med.unit
  formMinThreshold.value = med.min_threshold
  formPrescriptionRequired.value = med.prescription_required
  formNotes.value = med.notes || ''
  showEditModal.value = true
}

async function handleDelete(med: Medicine) {
  if (confirm(`Sei sicuro di voler rimuovere "${med.name}" dall'armadietto?`)) {
    await medicineStore.deleteMedicine(med.id)
  }
}

async function handleOpenBottle(med: Medicine) {
  const today = new Date().toISOString().slice(0, 10)
  await medicineStore.openMedicine(med.id, today)
}

async function handleConsume(med: Medicine) {
  await medicineStore.consumeDose(med.id, 1)
}

function getExpirationStatus(med: Medicine): { label: string; class: string; isUrgent: boolean } {
  const today = new Date().toISOString().slice(0, 10)
  const in7d = new Date()
  in7d.setDate(new Date().getDate() + 7)
  const in7dStr = in7d.toISOString().slice(0, 10)

  const in30d = new Date()
  in30d.setDate(new Date().getDate() + 30)
  const in30dStr = in30d.toISOString().slice(0, 10)

  const eff = med.effective_expiration_date
  if (eff < today) {
    return { label: `Scaduto (${eff})`, class: 'badge-expired', isUrgent: true }
  } else if (eff <= in7dStr) {
    return { label: `Scade tra pochi giorni (${eff})`, class: 'badge-expiring-7d', isUrgent: true }
  } else if (eff <= in30dStr) {
    return { label: `In scadenza (${eff})`, class: 'badge-expiring-30d', isUrgent: false }
  }
  return { label: `Scad. ${eff}`, class: 'badge-valid', isUrgent: false }
}

onMounted(async () => {
  await medicineStore.fetchMedicines()
  await medicineStore.fetchAlerts()
})
</script>

<template>
  <div class="medicines-view">
    <!-- Header -->
    <header class="medicines-header glass-panel">
      <div class="header-left">
        <div class="header-icon-box">
          <Pill :size="24" class="text-primary" />
        </div>
        <div>
          <h1 class="page-title">Armadietto Farmaci & Scadenze</h1>
          <p class="page-subtitle">
            Inventario medicinali, calcolo PAO post-apertura e allarmi catena del freddo.
          </p>
        </div>
      </div>

      <div class="header-actions">
        <button class="btn-primary-add" @click="openAddModal">
          <Plus :size="16" />
          <span>Inserimento Rapido</span>
        </button>
      </div>
    </header>

    <!-- KPI Summary Cards -->
    <div class="kpi-grid">
      <div class="kpi-card glass-panel" @click="medicineStore.selectedStatus = 'all'">
        <div class="kpi-top">
          <span class="kpi-label">Totale Farmaci</span>
          <Package :size="18" class="text-muted" />
        </div>
        <div class="kpi-val">{{ medicineStore.totalCount }}</div>
      </div>

      <div
        class="kpi-card glass-panel"
        :class="{ 'border-warning': medicineStore.expiringCount > 0 }"
        @click="medicineStore.selectedStatus = 'expiring_30d'"
      >
        <div class="kpi-top">
          <span class="kpi-label">In Scadenza (&lt; 30 gg)</span>
          <Clock :size="18" class="text-warning" />
        </div>
        <div class="kpi-val text-warning">{{ medicineStore.expiringCount }}</div>
      </div>

      <div
        class="kpi-card glass-panel"
        :class="{ 'border-danger': medicineStore.expiredCount > 0 }"
        @click="medicineStore.selectedStatus = 'expired'"
      >
        <div class="kpi-top">
          <span class="kpi-label">Scaduti</span>
          <AlertCircle :size="18" class="text-danger" />
        </div>
        <div class="kpi-val text-danger">{{ medicineStore.expiredCount }}</div>
      </div>

      <div class="kpi-card glass-panel" @click="medicineStore.selectedStatus = 'cold_chain'">
        <div class="kpi-top">
          <span class="kpi-label">Frigorifero (2-8°C)</span>
          <ThermometerSnowflake :size="18" class="text-cold" />
        </div>
        <div class="kpi-val text-cold">{{ medicineStore.coldChainCount }}</div>
      </div>

      <div class="kpi-card glass-panel" @click="medicineStore.selectedStatus = 'low_stock'">
        <div class="kpi-top">
          <span class="kpi-label">Scorte Basse</span>
          <MinusCircle :size="18" class="text-amber" />
        </div>
        <div class="kpi-val text-amber">{{ medicineStore.lowStockCount }}</div>
      </div>
    </div>

    <!-- Active Alerts Banner if any -->
    <div v-if="medicineStore.alerts.length > 0" class="alerts-banner glass-panel">
      <div class="alerts-banner-icon">
        <AlertTriangle :size="20" class="text-danger" />
      </div>
      <div class="alerts-banner-content">
        <h4 class="alerts-banner-title">Avvisi Armadietto Farmaci ({{ medicineStore.alerts.length }})</h4>
        <ul class="alerts-list">
          <li v-for="(al, i) in medicineStore.alerts.slice(0, 3)" :key="i" class="alert-item">
            {{ al.message }}
          </li>
        </ul>
      </div>
    </div>

    <!-- Filters & Search Toolbar -->
    <div class="toolbar-panel glass-panel">
      <div class="search-input-wrap">
        <Search :size="16" class="search-icon" />
        <input
          v-model="medicineStore.searchQuery"
          type="text"
          placeholder="Cerca farmaco per nome, principio attivo o codice AIC..."
          class="search-input"
          aria-label="Cerca farmaci"
        />
        <button
          v-if="medicineStore.searchQuery"
          class="btn-clear-search"
          @click="medicineStore.searchQuery = ''"
          aria-label="Pulisci ricerca"
        >
          &times;
        </button>
      </div>

      <div class="filters-row">
        <div class="filter-group">
          <span class="filter-label">Posizione:</span>
          <select v-model="medicineStore.selectedLocation" class="select-filter" aria-label="Filtro Posizione">
            <option value="all">Tutte le posizioni</option>
            <option v-for="loc in locations" :key="loc.key" :value="loc.key">{{ loc.label }}</option>
          </select>
        </div>

        <div class="filter-group">
          <span class="filter-label">Stato:</span>
          <select v-model="medicineStore.selectedStatus" class="select-filter" aria-label="Filtro Stato">
            <option value="all">Tutti i farmaci</option>
            <option value="expiring_30d">In scadenza (&lt; 30 gg)</option>
            <option value="expiring_7d">In scadenza urgente (&lt; 7 gg)</option>
            <option value="expired">Solo scaduti</option>
            <option value="cold_chain">Solo Frigorifero (2-8°C)</option>
            <option value="low_stock">Scorte basse</option>
          </select>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="medicineStore.filteredMedicines.length === 0" class="empty-medicines glass-panel">
      <Pill :size="48" class="text-muted" />
      <h3>Nessun medicinale trovato</h3>
      <p>Aggiungi farmaci al tuo inventario tramite la scansione Datamatrix o la ricerca AIFA.</p>
      <button class="btn-primary-add" @click="openAddModal">
        <Plus :size="16" />
        <span>Inserisci Primo Farmaco</span>
      </button>
    </div>

    <!-- Medicines Grid -->
    <div v-else class="medicines-grid">
      <article
        v-for="med in medicineStore.filteredMedicines"
        :key="med.id"
        class="med-card glass-panel"
      >
        <header class="med-card-header">
          <div class="med-card-title-box">
            <h3 class="med-name">{{ med.name }}</h3>
            <span v-if="med.active_ingredient" class="med-active-ingredient">
              {{ med.active_ingredient }}
            </span>
          </div>

          <div class="med-badges">
            <span :class="['badge-status', getExpirationStatus(med).class]">
              {{ getExpirationStatus(med).label }}
            </span>
            <span v-if="med.requires_cold_chain" class="badge-cold" title="Conservare in frigo a 2-8°C">
              <ThermometerSnowflake :size="12" /> Frigo 2-8°C
            </span>
          </div>
        </header>

        <div class="med-card-body">
          <div class="info-row">
            <span class="info-label">Posizione:</span>
            <span class="info-val font-medium">
              {{ locations.find(l => l.key === med.location)?.label || med.location }}
            </span>
          </div>

          <div class="info-row">
            <span class="info-label">Quantità:</span>
            <span class="info-val">
              <strong>{{ med.quantity }}</strong> {{ med.unit }}
              <span v-if="med.min_threshold > 0 && med.quantity <= med.min_threshold" class="text-amber text-xs font-semibold">
                (Scorta minima: {{ med.min_threshold }})
              </span>
            </span>
          </div>

          <!-- PAO Section -->
          <div v-if="med.pao_days > 0" class="pao-card-section">
            <div v-if="med.opened_at" class="pao-opened">
              <span class="text-xs text-muted">Aperto il {{ med.opened_at }} (PAO: {{ med.pao_days }} gg)</span>
              <span class="pao-effective">Scadenza effettiva: <strong>{{ med.effective_expiration_date }}</strong></span>
            </div>
            <div v-else class="pao-unopened">
              <span class="text-xs text-muted">Sigillato (validità {{ med.pao_days }} giorni dall'apertura)</span>
              <button class="btn-open-bottle" @click="handleOpenBottle(med)" title="Segna come aperto oggi">
                <CheckCircle2 :size="14" />
                <span>Apri Oggi</span>
              </button>
            </div>
          </div>

          <!-- AIC & Lot Info -->
          <div v-if="med.aic || med.lot_number" class="code-details">
            <span v-if="med.aic">AIC: {{ med.aic }}</span>
            <span v-if="med.lot_number">Lotto: {{ med.lot_number }}</span>
          </div>
        </div>

        <footer class="med-card-footer">
          <div class="footer-quick-actions">
            <button
              class="btn-consume-dose"
              @click="handleConsume(med)"
              :disabled="med.quantity <= 0"
              title="Consuma 1 dose"
            >
              <MinusCircle :size="14" />
              <span>-1 dose</span>
            </button>
          </div>

          <div class="footer-manage-actions">
            <button class="btn-icon" @click="openEditModal(med)" aria-label="Modifica farmaco">
              <Edit2 :size="15" />
            </button>
            <button class="btn-icon text-danger" @click="handleDelete(med)" aria-label="Rimuovi farmaco">
              <Trash2 :size="15" />
            </button>
          </div>
        </footer>
      </article>
    </div>

    <!-- MODAL: Inserimento Rapido / Modifica -->
    <div
      v-if="showAddModal || showEditModal"
      class="modal-backdrop"
      role="dialog"
      aria-modal="true"
      @click.self="showAddModal = false; showEditModal = false"
    >
      <div class="modal-card glass-panel">
        <header class="modal-header">
          <div class="modal-title-box">
            <Pill :size="20" class="text-primary" />
            <h2 class="modal-title">{{ editingId ? 'Modifica Farmaco' : 'Inserimento Rapido Farmaco' }}</h2>
          </div>
          <button class="btn-close-modal" @click="showAddModal = false; showEditModal = false">
            <X :size="20" />
          </button>
        </header>

        <div class="modal-body">
          <!-- Quick Scan section only for new additions -->
          <div v-if="!editingId" class="quick-scan-box glass-panel">
            <div class="quick-scan-title">
              <Sparkles :size="16" class="text-primary" />
              <span>Scansione Datamatrix GS1 o Codice AIC</span>
            </div>

            <div class="quick-scan-input-row">
              <input
                v-model="rawDatamatrixCode"
                type="text"
                class="input-code"
                placeholder="Inquadra o incolla codice (es: (01)...(17)280430 o AIC 9 cifre)"
                @keyup.enter="handleParseDatamatrix"
              />
              <button class="btn-parse-code" type="button" @click="handleParseDatamatrix">
                Decodifica
              </button>
            </div>

            <!-- Sample buttons -->
            <div class="sample-codes-row">
              <span class="sample-label">Esempi pronti:</span>
              <button
                v-for="s in sampleCodes"
                :key="s.label"
                type="button"
                class="chip-sample"
                @click="loadSampleCode(s.code)"
              >
                {{ s.label }}
              </button>
            </div>
            <p v-if="parseError" class="parse-error-text">{{ parseError }}</p>
          </div>

          <!-- AIFA Search Autocomplete -->
          <div v-if="!editingId" class="aifa-search-box">
            <label class="form-label">Ricerca Rapida Catalogo Farmaci AIFA:</label>
            <div class="aifa-input-wrap">
              <input
                v-model="aifaSearchInput"
                type="text"
                class="form-input"
                placeholder="Digita es: Tachipirina, Brufen, Oki, Tobral..."
                @input="onAifaInput"
              />
              <div v-if="showAifaSuggestions && medicineStore.aifaSearchResults.length > 0" class="aifa-suggestions-dropdown">
                <div
                  v-for="res in medicineStore.aifaSearchResults"
                  :key="res.aic"
                  class="aifa-suggestion-item"
                  @click="applyAifaRecord(res)"
                >
                  <div class="sugg-name">{{ res.name }}</div>
                  <div class="sugg-sub">
                    <span>{{ res.active_ingredient }}</span> &bull;
                    <span>AIC: {{ res.aic }}</span>
                    <span v-if="res.requires_cold_chain" class="text-cold"> &bull; Frigo 2-8°C</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Main form fields -->
          <form class="med-form" @submit.prevent="submitMedicineForm">
            <div class="form-grid">
              <div class="form-group span-2">
                <label class="form-label">Nome Farmaco *</label>
                <input v-model="formName" type="text" class="form-input" required placeholder="Es. Tachipirina 1000 mg" />
              </div>

              <div class="form-group">
                <label class="form-label">Principio Attivo</label>
                <input v-model="formActiveIngredient" type="text" class="form-input" placeholder="Es. Paracetamolo" />
              </div>

              <div class="form-group">
                <label class="form-label">Codice AIC (9 cifre)</label>
                <input v-model="formAIC" type="text" class="form-input" placeholder="Es. 034234035" />
              </div>

              <div class="form-group">
                <label class="form-label">Data di Scadenza Confezione *</label>
                <input v-model="formExpirationDate" type="date" class="form-input" required />
              </div>

              <div class="form-group">
                <label class="form-label">Numero di Lotto</label>
                <input v-model="formLotNumber" type="text" class="form-input" placeholder="Es. LOT982" />
              </div>

              <div class="form-group">
                <label class="form-label">Quantità Residua</label>
                <input v-model.number="formQuantity" type="number" step="0.5" class="form-input" required />
              </div>

              <div class="form-group">
                <label class="form-label">Unità</label>
                <select v-model="formUnit" class="form-input">
                  <option v-for="u in units" :key="u.key" :value="u.key">{{ u.label }}</option>
                </select>
              </div>

              <div class="form-group">
                <label class="form-label">Posizione Conservazione</label>
                <select v-model="formLocation" class="form-input">
                  <option v-for="loc in locations" :key="loc.key" :value="loc.key">{{ loc.label }}</option>
                </select>
              </div>

              <div class="form-group">
                <label class="form-label">Scorta Minima di Allarme</label>
                <input v-model.number="formMinThreshold" type="number" step="1" class="form-input" placeholder="Es. 2" />
              </div>
            </div>

            <!-- PAO Config Section -->
            <div class="pao-config-box">
              <h4 class="pao-title">Period After Opening (PAO)</h4>
              <p class="pao-desc">Per colliri, sciroppi o pomate che scadono N giorni dopo la prima apertura.</p>
              <div class="form-grid">
                <div class="form-group">
                  <label class="form-label">Giorni Validità dall'Apertura (PAO)</label>
                  <input v-model.number="formPaoDays" type="number" class="form-input" placeholder="0 = non applicabile (es. 28)" />
                </div>
                <div class="form-group flex-col justify-end">
                  <label class="checkbox-label">
                    <input type="checkbox" v-model="formIsOpened" />
                    <span>Confezione già aperta</span>
                  </label>
                  <input
                    v-if="formIsOpened"
                    v-model="formOpenedAt"
                    type="date"
                    class="form-input mt-2"
                    placeholder="Data apertura"
                  />
                </div>
              </div>
            </div>

            <!-- Flags -->
            <div class="flags-row">
              <label class="checkbox-label">
                <input type="checkbox" v-model="formRequiresColdChain" />
                <span class="flex items-center gap-1">
                  <ThermometerSnowflake :size="14" class="text-cold" />
                  Richiede conservazione in Frigorifero (2-8°C)
                </span>
              </label>

              <label class="checkbox-label">
                <input type="checkbox" v-model="formPrescriptionRequired" />
                <span>Richiede Ricetta Medica</span>
              </label>
            </div>

            <div class="modal-footer">
              <button type="button" class="btn-cancel" @click="showAddModal = false; showEditModal = false">
                Annulla
              </button>
              <button type="submit" class="btn-save">
                <Check :size="16" />
                <span>{{ editingId ? 'Salva Modifiche' : 'Registra Farmaco' }}</span>
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.medicines-view {
  display: flex;
  flex-direction: column;
  gap: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

.medicines-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-radius: 16px;
  background: var(--surface-card, rgba(15, 23, 42, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.header-icon-box {
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
  color: var(--text-main, #f8fafc);
}

.page-subtitle {
  font-size: 0.88rem;
  color: var(--text-muted, #94a3b8);
  margin: 4px 0 0;
}

.btn-primary-add {
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
  transition: transform 0.2s, background 0.2s;
}

.btn-primary-add:hover {
  transform: translateY(-1px);
  filter: brightness(1.1);
}

/* KPI Grid */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 14px;
}

.kpi-card {
  padding: 16px;
  border-radius: 12px;
  cursor: pointer;
  transition: transform 0.2s, border-color 0.2s;
  background: var(--surface-card, rgba(15, 23, 42, 0.6));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
}

.kpi-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
}

.kpi-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.kpi-label {
  font-size: 0.82rem;
  color: var(--text-muted, #94a3b8);
  font-weight: 500;
}

.kpi-val {
  font-size: 1.6rem;
  font-weight: 700;
  color: var(--text-main, #f8fafc);
}

.text-warning { color: #f59e0b; }
.text-danger { color: #ef4444; }
.text-cold { color: #38bdf8; }
.text-amber { color: #f97316; }
.border-warning { border-color: rgba(245, 158, 11, 0.4) !important; }
.border-danger { border-color: rgba(239, 68, 68, 0.4) !important; }

/* Alerts banner */
.alerts-banner {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 16px 20px;
  border-radius: 12px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.alerts-banner-title {
  font-size: 0.95rem;
  font-weight: 700;
  color: #f87171;
  margin: 0 0 6px;
}

.alerts-list {
  margin: 0;
  padding-left: 18px;
  color: #fca5a5;
  font-size: 0.85rem;
}

/* Toolbar */
.toolbar-panel {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 14px 20px;
  border-radius: 12px;
  background: var(--surface-card, rgba(15, 23, 42, 0.6));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
}

.search-input-wrap {
  position: relative;
  flex: 1;
  min-width: 260px;
}

.search-icon {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-dim, #64748b);
}

.search-input {
  width: 100%;
  padding: 10px 36px 10px 38px;
  border-radius: 8px;
  background: var(--bg-surface, rgba(0, 0, 0, 0.25));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  color: var(--text-main, #f8fafc);
  font-size: 0.88rem;
}

.btn-clear-search {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  background: transparent;
  border: none;
  color: var(--text-muted, #94a3b8);
  font-size: 1.1rem;
  cursor: pointer;
}

.filters-row {
  display: flex;
  align-items: center;
  gap: 14px;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-label {
  font-size: 0.82rem;
  color: var(--text-muted, #94a3b8);
}

.select-filter {
  padding: 8px 12px;
  border-radius: 8px;
  background: var(--bg-surface, rgba(0, 0, 0, 0.25));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  color: var(--text-main, #f8fafc);
  font-size: 0.85rem;
}

/* Empty State */
.empty-medicines {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 48px 24px;
  border-radius: 16px;
  gap: 12px;
  background: var(--surface-card, rgba(15, 23, 42, 0.6));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
}

.empty-medicines h3 {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-main, #f8fafc);
  margin: 0;
}

.empty-medicines p {
  color: var(--text-muted, #94a3b8);
  margin: 0;
  max-width: 480px;
  line-height: 1.5;
}

/* Medicines Grid */
.medicines-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.med-card {
  padding: 18px;
  border-radius: 14px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  background: var(--surface-card, rgba(15, 23, 42, 0.6));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  transition: transform 0.2s, border-color 0.2s;
}

.med-card:hover {
  border-color: rgba(255, 255, 255, 0.2);
}

.med-card-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.med-name {
  font-size: 1.05rem;
  font-weight: 700;
  color: var(--text-main, #f8fafc);
  margin: 0;
}

.med-active-ingredient {
  font-size: 0.82rem;
  color: var(--text-muted, #94a3b8);
}

.med-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.badge-status {
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
}

.badge-valid { background: rgba(16, 185, 129, 0.15); color: #10b981; }
.badge-expiring-30d { background: rgba(245, 158, 11, 0.15); color: #f59e0b; }
.badge-expiring-7d { background: rgba(239, 68, 68, 0.2); color: #f87171; border: 1px solid rgba(239, 68, 68, 0.4); }
.badge-expired { background: rgba(239, 68, 68, 0.3); color: #ef4444; font-weight: 700; }

.badge-cold {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: 6px;
  background: rgba(56, 189, 248, 0.15);
  color: #38bdf8;
  font-size: 0.75rem;
  font-weight: 600;
}

.med-card-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 0.85rem;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.info-label { color: var(--text-muted, #94a3b8); }
.info-val { color: var(--text-main, #f1f5f9); }

/* PAO Card section */
.pao-card-section {
  background: var(--bg-card, rgba(0, 0, 0, 0.2));
  border-radius: 8px;
  padding: 10px;
  margin-top: 4px;
}

.pao-opened {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.pao-effective {
  font-size: 0.82rem;
  color: var(--text-main, #f1f5f9);
}

.pao-unopened {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.btn-open-bottle {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  background: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.3);
  color: #10b981;
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
}

.code-details {
  display: flex;
  justify-content: space-between;
  font-size: 0.75rem;
  color: var(--text-dim, #64748b);
  margin-top: 4px;
}

/* Card footer */
.med-card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 10px;
  border-top: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.08));
}

.btn-consume-dose {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  background: var(--bg-surface, rgba(255, 255, 255, 0.06));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  color: var(--text-main, #f8fafc);
  padding: 5px 10px;
  border-radius: 6px;
  font-size: 0.8rem;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-consume-dose:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.12);
}

.btn-consume-dose:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.footer-manage-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-icon {
  background: transparent;
  border: none;
  color: var(--text-muted, #94a3b8);
  padding: 6px;
  border-radius: 6px;
  cursor: pointer;
}

.btn-icon:hover { color: var(--text-main, #fff); }

/* MODALS */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 20px;
}

.modal-card {
  width: 100%;
  max-width: 650px;
  max-height: 85vh;
  overflow-y: auto;
  border-radius: 16px;
  background: var(--surface-card, #0f172a);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.15));
  padding: 24px;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.modal-title-box {
  display: flex;
  align-items: center;
  gap: 10px;
}

.modal-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-main, #f8fafc);
  margin: 0;
}

.quick-scan-box {
  padding: 16px;
  border-radius: 12px;
  background: rgba(16, 185, 129, 0.05);
  border: 1px solid rgba(16, 185, 129, 0.2);
  margin-bottom: 18px;
}

.quick-scan-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.9rem;
  font-weight: 600;
  color: #10b981;
  margin-bottom: 10px;
}

.quick-scan-input-row {
  display: flex;
  gap: 10px;
}

.input-code {
  flex: 1;
  padding: 8px 12px;
  border-radius: 8px;
  background: var(--bg-surface, rgba(0, 0, 0, 0.3));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.15));
  color: var(--text-main, #f8fafc);
  font-size: 0.85rem;
  font-family: monospace;
}

.btn-parse-code {
  background: rgba(16, 185, 129, 0.2);
  border: 1px solid rgba(16, 185, 129, 0.4);
  color: #10b981;
  font-weight: 600;
  padding: 8px 16px;
  border-radius: 8px;
  cursor: pointer;
}

.sample-codes-row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 10px;
}

.sample-label {
  font-size: 0.78rem;
  color: var(--text-muted, #94a3b8);
}

.parse-error-text {
  color: #f87171;
  font-size: 0.8rem;
  margin: 8px 0 0;
}

/* AIFA Search box */
.aifa-search-box {
  margin-bottom: 18px;
}

.aifa-input-wrap {
  position: relative;
}

.aifa-suggestions-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  background: var(--bg-surface-elevated, #1e293b);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.15));
  border-radius: 8px;
  max-height: 200px;
  overflow-y: auto;
  z-index: 50;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
}

.aifa-suggestion-item {
  padding: 10px 14px;
  cursor: pointer;
  border-bottom: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.05));
}

.aifa-suggestion-item:hover {
  background: rgba(16, 185, 129, 0.15);
}

.sugg-name {
  font-weight: 600;
  color: var(--text-main, #f8fafc);
  font-size: 0.88rem;
}

.sugg-sub {
  font-size: 0.78rem;
  color: var(--text-muted, #94a3b8);
  margin-top: 2px;
}

/* Form */
.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.span-2 {
  grid-column: span 2;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 0.82rem;
  color: var(--text-muted, #cbd5e1);
  font-weight: 500;
}

.form-input {
  width: 100%;
  padding: 9px 12px;
  border-radius: 8px;
  background: var(--bg-surface, rgba(0, 0, 0, 0.25));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  color: var(--text-main, #f8fafc);
  font-size: 0.88rem;
}

.pao-config-box {
  background: var(--bg-card, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.08));
  border-radius: 10px;
  padding: 14px;
  margin: 16px 0;
}

.pao-title {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-main, #e2e8f0);
  margin: 0 0 4px;
}

.pao-desc {
  font-size: 0.78rem;
  color: var(--text-muted, #94a3b8);
  margin: 0 0 10px;
}

.flags-row {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 14px 0 20px;
}

.checkbox-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  color: var(--text-main, #e2e8f0);
  cursor: pointer;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
}

.btn-cancel {
  background: transparent;
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.2));
  color: var(--text-muted, #cbd5e1);
  padding: 10px 18px;
  border-radius: 8px;
  font-weight: 500;
  cursor: pointer;
}

.btn-save {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: var(--primary, #10b981);
  color: #000;
  font-weight: 600;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  cursor: pointer;
}

/* ═══════════════════════════════════════════════════════════════════════════════
   LIGHT MODE OVERRIDES - High Contrast & WCAG 2.2 AA Compliance
   ═══════════════════════════════════════════════════════════════════════════════ */
:root[data-theme="light"] .medicines-header,
[data-theme="light"] .medicines-header {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.05);
}

:root[data-theme="light"] .page-title,
[data-theme="light"] .page-title {
  color: #0f172a;
}

:root[data-theme="light"] .page-subtitle,
[data-theme="light"] .page-subtitle {
  color: #334155;
}

:root[data-theme="light"] .header-icon-box,
[data-theme="light"] .header-icon-box {
  background: rgba(16, 185, 129, 0.12);
  color: #047857;
}

:root[data-theme="light"] .btn-primary-add,
[data-theme="light"] .btn-primary-add {
  background: #0284c7;
  color: #ffffff;
  font-weight: 600;
  box-shadow: 0 2px 6px rgba(2, 132, 199, 0.3);
}

:root[data-theme="light"] .btn-primary-add:hover,
[data-theme="light"] .btn-primary-add:hover {
  background: #0369a1;
}

/* KPI Cards in Light Mode */
:root[data-theme="light"] .kpi-card,
[data-theme="light"] .kpi-card {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  box-shadow: 0 2px 6px rgba(15, 23, 42, 0.04);
}

:root[data-theme="light"] .kpi-card:hover,
[data-theme="light"] .kpi-card:hover {
  border-color: #94a3b8;
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.08);
}

:root[data-theme="light"] .kpi-label,
[data-theme="light"] .kpi-label {
  color: #475569;
  font-weight: 600;
}

:root[data-theme="light"] .kpi-val,
[data-theme="light"] .kpi-val {
  color: #0f172a;
}

:root[data-theme="light"] .text-warning,
[data-theme="light"] .text-warning {
  color: #b45309;
}

:root[data-theme="light"] .text-danger,
[data-theme="light"] .text-danger {
  color: #b91c1c;
}

:root[data-theme="light"] .text-cold,
[data-theme="light"] .text-cold {
  color: #0284c7;
}

:root[data-theme="light"] .text-amber,
[data-theme="light"] .text-amber {
  color: #c2410c;
}

:root[data-theme="light"] .border-warning,
[data-theme="light"] .border-warning {
  border-color: rgba(180, 83, 9, 0.5) !important;
}

:root[data-theme="light"] .border-danger,
[data-theme="light"] .border-danger {
  border-color: rgba(185, 28, 28, 0.5) !important;
}

/* Toolbar & Filters in Light Mode */
:root[data-theme="light"] .toolbar-panel,
[data-theme="light"] .toolbar-panel {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  box-shadow: 0 2px 6px rgba(15, 23, 42, 0.04);
}

:root[data-theme="light"] .search-input,
[data-theme="light"] .search-input {
  background: #f8fafc;
  border: 1px solid #cbd5e1;
  color: #0f172a;
}

:root[data-theme="light"] .search-input:focus,
[data-theme="light"] .search-input:focus {
  background: #ffffff;
  border-color: #0284c7;
  box-shadow: 0 0 0 3px rgba(14, 165, 233, 0.15);
}

:root[data-theme="light"] .search-input::placeholder,
[data-theme="light"] .search-input::placeholder {
  color: #64748b;
  opacity: 0.9;
}

:root[data-theme="light"] .search-icon,
[data-theme="light"] .search-icon {
  color: #64748b;
}

:root[data-theme="light"] .btn-clear-search,
[data-theme="light"] .btn-clear-search {
  color: #64748b;
}

:root[data-theme="light"] .filter-label,
[data-theme="light"] .filter-label {
  color: #334155;
  font-weight: 600;
}

:root[data-theme="light"] .select-filter,
[data-theme="light"] .select-filter {
  background: #f8fafc;
  border: 1px solid #cbd5e1;
  color: #0f172a;
  font-weight: 500;
}

:root[data-theme="light"] .select-filter:focus,
[data-theme="light"] .select-filter:focus {
  background: #ffffff;
  border-color: #0284c7;
}

:root[data-theme="light"] .select-filter option,
[data-theme="light"] .select-filter option {
  background: #ffffff;
  color: #0f172a;
}

/* Empty State in Light Mode */
:root[data-theme="light"] .empty-medicines,
[data-theme="light"] .empty-medicines {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  box-shadow: 0 2px 6px rgba(15, 23, 42, 0.04);
}

:root[data-theme="light"] .empty-medicines h3,
[data-theme="light"] .empty-medicines h3 {
  color: #0f172a;
}

:root[data-theme="light"] .empty-medicines p,
[data-theme="light"] .empty-medicines p {
  color: #334155;
}

/* Medicines Cards in Light Mode */
:root[data-theme="light"] .med-card,
[data-theme="light"] .med-card {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.05);
}

:root[data-theme="light"] .med-card:hover,
[data-theme="light"] .med-card:hover {
  border-color: #94a3b8;
  box-shadow: 0 6px 16px rgba(15, 23, 42, 0.08);
}

:root[data-theme="light"] .med-name,
[data-theme="light"] .med-name {
  color: #0f172a;
}

:root[data-theme="light"] .med-active-ingredient,
[data-theme="light"] .med-active-ingredient {
  color: #475569;
  font-weight: 500;
}

:root[data-theme="light"] .badge-valid,
[data-theme="light"] .badge-valid {
  background: #dcfce7;
  color: #15803d;
}

:root[data-theme="light"] .badge-expiring-30d,
[data-theme="light"] .badge-expiring-30d {
  background: #fef3c7;
  color: #b45309;
}

:root[data-theme="light"] .badge-expiring-7d,
[data-theme="light"] .badge-expiring-7d {
  background: #fee2e2;
  color: #b91c1c;
  border-color: #fca5a5;
}

:root[data-theme="light"] .badge-expired,
[data-theme="light"] .badge-expired {
  background: #fee2e2;
  color: #991b1b;
}

:root[data-theme="light"] .badge-cold,
[data-theme="light"] .badge-cold {
  background: #e0f2fe;
  color: #0369a1;
}

:root[data-theme="light"] .info-label,
[data-theme="light"] .info-label {
  color: #475569;
}

:root[data-theme="light"] .info-val,
[data-theme="light"] .info-val {
  color: #0f172a;
  font-weight: 600;
}

:root[data-theme="light"] .pao-card-section,
[data-theme="light"] .pao-card-section {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
}

:root[data-theme="light"] .pao-effective,
[data-theme="light"] .pao-effective {
  color: #0f172a;
}

:root[data-theme="light"] .btn-open-bottle,
[data-theme="light"] .btn-open-bottle {
  background: #dcfce7;
  border-color: #86efac;
  color: #15803d;
}

:root[data-theme="light"] .code-details,
[data-theme="light"] .code-details {
  color: #475569;
}

:root[data-theme="light"] .med-card-footer,
[data-theme="light"] .med-card-footer {
  border-top: 1px solid #e2e8f0;
}

:root[data-theme="light"] .btn-consume-dose,
[data-theme="light"] .btn-consume-dose {
  background: #f1f5f9;
  border: 1px solid #cbd5e1;
  color: #0f172a;
  font-weight: 600;
}

:root[data-theme="light"] .btn-consume-dose:hover:not(:disabled),
[data-theme="light"] .btn-consume-dose:hover:not(:disabled) {
  background: #e2e8f0;
  color: #0f172a;
}

:root[data-theme="light"] .btn-icon,
[data-theme="light"] .btn-icon {
  color: #475569;
}

:root[data-theme="light"] .btn-icon:hover,
[data-theme="light"] .btn-icon:hover {
  color: #0f172a;
  background: #f1f5f9;
}

/* Alerts Banner in Light Mode */
:root[data-theme="light"] .alerts-banner,
[data-theme="light"] .alerts-banner {
  background: #fef2f2;
  border: 1px solid #fecaca;
}

:root[data-theme="light"] .alerts-banner-title,
[data-theme="light"] .alerts-banner-title {
  color: #991b1b;
}

:root[data-theme="light"] .alerts-list,
[data-theme="light"] .alerts-list {
  color: #b91c1c;
}

/* Modal in Light Mode */
:root[data-theme="light"] .modal-card,
[data-theme="light"] .modal-card {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  box-shadow: 0 20px 45px rgba(15, 23, 42, 0.16);
}

:root[data-theme="light"] .modal-title,
[data-theme="light"] .modal-title {
  color: #0f172a;
}

:root[data-theme="light"] .quick-scan-box,
[data-theme="light"] .quick-scan-box {
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
}

:root[data-theme="light"] .quick-scan-title,
[data-theme="light"] .quick-scan-title {
  color: #15803d;
}

:root[data-theme="light"] .input-code,
[data-theme="light"] .input-code {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  color: #0f172a;
}

:root[data-theme="light"] .input-code:focus,
[data-theme="light"] .input-code:focus {
  border-color: #059669;
}

:root[data-theme="light"] .sample-label,
[data-theme="light"] .sample-label {
  color: #475569;
}

:root[data-theme="light"] .aifa-suggestions-dropdown,
[data-theme="light"] .aifa-suggestions-dropdown {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  box-shadow: 0 10px 25px rgba(15, 23, 42, 0.15);
}

:root[data-theme="light"] .aifa-suggestion-item,
[data-theme="light"] .aifa-suggestion-item {
  border-bottom: 1px solid #f1f5f9;
}

:root[data-theme="light"] .aifa-suggestion-item:hover,
[data-theme="light"] .aifa-suggestion-item:hover {
  background: #f0fdf4;
}

:root[data-theme="light"] .sugg-name,
[data-theme="light"] .sugg-name {
  color: #0f172a;
}

:root[data-theme="light"] .sugg-sub,
[data-theme="light"] .sugg-sub {
  color: #475569;
}

:root[data-theme="light"] .form-label,
[data-theme="light"] .form-label {
  color: #1e293b;
  font-weight: 600;
}

:root[data-theme="light"] .form-input,
[data-theme="light"] .form-input {
  background: #f8fafc;
  border: 1px solid #cbd5e1;
  color: #0f172a;
}

:root[data-theme="light"] .form-input:focus,
[data-theme="light"] .form-input:focus {
  background: #ffffff;
  border-color: #0284c7;
  box-shadow: 0 0 0 3px rgba(14, 165, 233, 0.15);
}

:root[data-theme="light"] .pao-config-box,
[data-theme="light"] .pao-config-box {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
}

:root[data-theme="light"] .pao-title,
[data-theme="light"] .pao-title {
  color: #0f172a;
}

:root[data-theme="light"] .pao-desc,
[data-theme="light"] .pao-desc {
  color: #475569;
}

:root[data-theme="light"] .checkbox-label,
[data-theme="light"] .checkbox-label {
  color: #1e293b;
}

:root[data-theme="light"] .btn-cancel,
[data-theme="light"] .btn-cancel {
  border: 1px solid #cbd5e1;
  color: #334155;
}

:root[data-theme="light"] .btn-cancel:hover,
[data-theme="light"] .btn-cancel:hover {
  background: #f1f5f9;
  color: #0f172a;
}

:root[data-theme="light"] .btn-save,
[data-theme="light"] .btn-save {
  background: #0284c7;
  color: #ffffff;
}

:root[data-theme="light"] .btn-save:hover,
[data-theme="light"] .btn-save:hover {
  background: #0369a1;
}
</style>
