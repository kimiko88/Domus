<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ShoppingBag,
  Download,
  CheckCircle,
  PackageCheck,
  Search,
  Box,
  Sun,
  Gauge,
  X,
  ExternalLink,
  ShieldCheck,
  PlusCircle,
  Code,
  Globe,
  Clock,
  Check,
  AlertTriangle,
  Cpu,
  Radio,
  Share2,
  Activity,
  Volume2,
  Smartphone,
  TrendingDown,
  Camera
} from '@lucide/vue'
import { apiClient } from '../api/client'

const { t } = useI18n()

interface StorePackage {
  id: string
  package_id: string
  name: string
  type: 'wasm_plugin' | 'blueprint' | 'widget'
  tier?: 'verified' | 'community'
  review_status?: 'approved' | 'pending_review' | 'rejected'
  version: string
  author: string
  description: string
  repository_url?: string
  license?: string
  capabilities?: string[]
  installed: boolean
  installed_at?: string
  icon?: string
}

const activeTier = ref<string>('')
const activeType = ref<string>('')
const searchQuery = ref<string>('')
const selectedPackage = ref<StorePackage | null>(null)
const showSubmitModal = ref(false)

// Community Submission Form State
const submitForm = ref({
  name: '',
  package_id: '',
  type: 'wasm_plugin' as 'wasm_plugin' | 'blueprint' | 'widget',
  version: '1.0.0',
  author: '',
  repository_url: '',
  license: 'Apache-2.0',
  capabilities: 'wasi:serial-port, events:write',
  description: ''
})
const submitError = ref('')
const isSubmitting = ref(false)

const packages = ref<StorePackage[]>([])

const countByType = (type: string) => {
  if (!type) return packages.value.length
  return packages.value.filter(p => p.type === type).length
}

const countByTier = (tier: string) => {
  if (!tier) return packages.value.length
  if (tier === 'pending_review') {
    return packages.value.filter(p => p.review_status === 'pending_review').length
  }
  return packages.value.filter(p => (p.tier || 'verified') === tier && p.review_status !== 'pending_review').length
}

const filteredPackages = computed(() => {
  return packages.value.filter(p => {
    // Tier filter
    let matchesTier = true
    if (activeTier.value === 'pending_review') {
      matchesTier = p.review_status === 'pending_review'
    } else if (activeTier.value) {
      matchesTier = (p.tier || 'verified') === activeTier.value && p.review_status !== 'pending_review'
    }

    // Type filter
    const matchesType = !activeType.value || p.type === activeType.value

    // Search query
    const q = searchQuery.value.trim().toLowerCase()
    const matchesSearch = !q ||
      p.name.toLowerCase().includes(q) ||
      p.description.toLowerCase().includes(q) ||
      p.author.toLowerCase().includes(q) ||
      p.package_id.toLowerCase().includes(q) ||
      (p.license && p.license.toLowerCase().includes(q))

    return matchesTier && matchesType && matchesSearch
  })
})

async function toggleInstall(pkg: StorePackage, event?: Event) {
  if (event) {
    event.stopPropagation()
  }
  try {
    if (pkg.installed) {
      await apiClient.post(`/api/v1/store/packages/${pkg.id}/uninstall`, {})
      pkg.installed = false
      pkg.installed_at = undefined
    } else {
      await apiClient.post(`/api/v1/store/packages/${pkg.id}/install`, {})
      pkg.installed = true
      pkg.installed_at = new Date().toISOString()
    }
  } catch (err) {
    console.error('Failed to toggle install package:', err)
  }
}

async function approvePackage(pkg: StorePackage, event?: Event) {
  if (event) event.stopPropagation()
  try {
    await apiClient.post(`/api/v1/store/community/${pkg.id}/approve`, {})
    pkg.review_status = 'approved'
  } catch (err) {
    console.error('Failed to approve package:', err)
  }
}

async function rejectPackage(pkg: StorePackage, event?: Event) {
  if (event) event.stopPropagation()
  try {
    await apiClient.post(`/api/v1/store/community/${pkg.id}/reject`, { reason: 'Rifiutato dall\'amministratore' })
    pkg.review_status = 'rejected'
  } catch (err) {
    console.error('Failed to reject package:', err)
  }
}

async function handleCommunitySubmit() {
  submitError.value = ''
  if (!submitForm.value.name.trim() || !submitForm.value.package_id.trim()) {
    submitError.value = 'Nome e ID pacchetto sono obbligatori.'
    return
  }
  const repo = submitForm.value.repository_url.trim()
  if (!repo || (!repo.startsWith('http://') && !repo.startsWith('https://'))) {
    submitError.value = 'È richiesto un URL valido di un repository open source pubblico (es. GitHub o GitLab).'
    return
  }
  if (!submitForm.value.license.trim()) {
    submitError.value = 'È richiesta una licenza open source valida (es. Apache-2.0, MIT, GPL-3.0).'
    return
  }

  isSubmitting.value = true
  try {
    const caps = submitForm.value.capabilities
      .split(',')
      .map(c => c.trim())
      .filter(c => c.length > 0)

    const created = await apiClient.post<StorePackage>('/api/v1/store/community/submit', {
      name: submitForm.value.name.trim(),
      package_id: submitForm.value.package_id.trim(),
      type: submitForm.value.type,
      version: submitForm.value.version.trim() || '1.0.0',
      author: submitForm.value.author.trim() || 'Community Developer',
      description: submitForm.value.description.trim(),
      repository_url: repo,
      license: submitForm.value.license.trim(),
      capabilities: caps,
      icon: submitForm.value.type === 'wasm_plugin' ? 'box' : 'sun'
    })

    if (created) {
      packages.value.unshift(created)
      showSubmitModal.value = false
      submitForm.value = {
        name: '',
        package_id: '',
        type: 'wasm_plugin',
        version: '1.0.0',
        author: '',
        repository_url: '',
        license: 'Apache-2.0',
        capabilities: 'wasi:serial-port, events:write',
        description: ''
      }
    }
  } catch (err: any) {
    submitError.value = err?.message || 'Errore durante la pubblicazione del pacchetto.'
  } finally {
    isSubmitting.value = false
  }
}

function openDetails(pkg: StorePackage) {
  selectedPackage.value = pkg
}

function closeDetails() {
  selectedPackage.value = null
}

onMounted(async () => {
  try {
    const list = await apiClient.get<StorePackage[]>('/api/v1/store/packages')
    packages.value = list || []
  } catch {
    packages.value = []
  }
})
</script>

<template>
  <div class="store-view-container">
    <header class="store-header glass-panel">
      <div class="header-main-row">
        <div class="header-title-box">
          <div class="icon-circle">
            <ShoppingBag :size="24" class="text-primary" />
          </div>
          <div>
            <h1 class="page-title">{{ t('store.title', 'Domus Store & Marketplace Estensioni') }}</h1>
            <p class="page-subtitle">{{ t('store.subtitle', 'Estendi Domus con plugin WASI ad alta efficienza e blueprint open source verificati') }}</p>
          </div>
        </div>

        <div class="header-actions">
          <!-- Proponi Plugin Community Button -->
          <button
            type="button"
            class="btn-propose-community"
            @click="showSubmitModal = true"
            aria-label="Proponi un plugin open source della community"
          >
            <PlusCircle :size="18" />
            <span>Proponi Plugin Community</span>
          </button>
        </div>
      </div>

      <!-- Search Bar -->
      <div class="search-wrapper">
        <Search :size="16" class="search-icon" aria-hidden="true" />
        <input
          id="store-search"
          name="store_search"
          v-model="searchQuery"
          type="search"
          placeholder="Cerca per nome, autore, licenza o package ID..."
          class="store-search-input"
          autocomplete="off"
          aria-label="Cerca nel marketplace"
        />
        <button
          v-if="searchQuery"
          type="button"
          class="clear-search-btn"
          @click="searchQuery = ''"
          aria-label="Cancella ricerca"
        >
          ✕
        </button>
      </div>

      <!-- Dual Tier & Type Filter System -->
      <div class="filters-container">
        <!-- Tier / Source Filters -->
        <div class="filter-group">
          <span class="filter-group-label">Origine & Verifica:</span>
          <div class="filter-pills" role="tablist" aria-label="Filtro categoria verifica">
            <button
              type="button"
              :class="['pill-btn', { active: activeTier === '' }]"
              @click="activeTier = ''"
            >
              Tutti <span class="pill-badge">{{ packages.length }}</span>
            </button>
            <button
              type="button"
              :class="['pill-btn tier-verified', { active: activeTier === 'verified' }]"
              @click="activeTier = 'verified'"
            >
              <ShieldCheck :size="14" />
              Domus Verified <span class="pill-badge">{{ countByTier('verified') }}</span>
            </button>
            <button
              type="button"
              :class="['pill-btn tier-community', { active: activeTier === 'community' }]"
              @click="activeTier = 'community'"
            >
              <Globe :size="14" />
              Community Open Source <span class="pill-badge">{{ countByTier('community') }}</span>
            </button>
            <button
              v-if="countByTier('pending_review') > 0"
              type="button"
              :class="['pill-btn tier-pending', { active: activeTier === 'pending_review' }]"
              @click="activeTier = 'pending_review'"
            >
              <Clock :size="14" />
              In Revisione <span class="pill-badge pending-badge">{{ countByTier('pending_review') }}</span>
            </button>
          </div>
        </div>

        <!-- Extension Type Filters -->
        <div class="filter-group">
          <span class="filter-group-label">Tipo Modulo:</span>
          <div class="filter-pills" role="tablist" aria-label="Filtro tipo estensioni">
            <button
              type="button"
              :class="['pill-btn', { active: activeType === '' }]"
              @click="activeType = ''"
            >
              Tutti i Tipi
            </button>
            <button
              type="button"
              :class="['pill-btn', { active: activeType === 'wasm_plugin' }]"
              @click="activeType = 'wasm_plugin'"
            >
              WASM Plugins <span class="pill-badge">{{ countByType('wasm_plugin') }}</span>
            </button>
            <button
              type="button"
              :class="['pill-btn', { active: activeType === 'blueprint' }]"
              @click="activeType = 'blueprint'"
            >
              Blueprints <span class="pill-badge">{{ countByType('blueprint') }}</span>
            </button>
            <button
              type="button"
              :class="['pill-btn', { active: activeType === 'widget' }]"
              @click="activeType = 'widget'"
            >
              Widget UI <span class="pill-badge">{{ countByType('widget') }}</span>
            </button>
          </div>
        </div>
      </div>
    </header>

    <!-- Empty State -->
    <div v-if="filteredPackages.length === 0" class="empty-state glass-panel">
      <PackageCheck :size="48" class="empty-icon text-muted" />
      <p class="empty-title">{{ searchQuery ? 'Nessun risultato per la ricerca' : 'Nessun pacchetto disponibile' }}</p>
      <p class="empty-desc">
        {{ searchQuery ? 'Prova con parole chiave differenti come "sunspec", "wmbus", "radar", o "modem".' : 'I pacchetti del catalogo e proposti dalla community appariranno qui.' }}
      </p>
    </div>

    <!-- Packages Grid -->
    <div v-else class="packages-grid">
      <article
        v-for="pkg in filteredPackages"
        :key="pkg.id"
        class="package-card glass-panel"
        tabindex="0"
        role="button"
        :aria-label="`Dettagli per ${pkg.name}`"
        @click="openDetails(pkg)"
        @keydown.enter="openDetails(pkg)"
      >
        <div class="card-top">
          <div class="pkg-icon-box" :class="pkg.type">
            <Cpu v-if="pkg.icon === 'cpu'" :size="22" />
            <Radio v-else-if="pkg.icon === 'radio'" :size="22" />
            <Sun v-else-if="pkg.icon === 'sun'" :size="22" />
            <Share2 v-else-if="pkg.icon === 'share-2'" :size="22" />
            <Activity v-else-if="pkg.icon === 'activity'" :size="22" />
            <Volume2 v-else-if="pkg.icon === 'volume-2'" :size="22" />
            <Smartphone v-else-if="pkg.icon === 'smartphone'" :size="22" />
            <TrendingDown v-else-if="pkg.icon === 'trending-down'" :size="22" />
            <Camera v-else-if="pkg.icon === 'camera'" :size="22" />
            <Box v-else-if="pkg.type === 'wasm_plugin'" :size="22" />
            <Sun v-else-if="pkg.type === 'blueprint'" :size="22" />
            <Gauge v-else :size="22" />
          </div>

          <div class="pkg-badges">
            <!-- Review Status / Tier Badge -->
            <span v-if="pkg.review_status === 'pending_review'" class="badge-pending">
              <Clock :size="12" /> In Revisione
            </span>
            <span v-else-if="pkg.review_status === 'rejected'" class="badge-rejected">
              Rifiutato
            </span>
            <span v-else-if="pkg.tier === 'community'" class="badge-community">
              <Globe :size="12" /> Community
            </span>
            <span v-else class="badge-verified">
              <ShieldCheck :size="12" /> Verified
            </span>

            <span class="type-tag">{{ pkg.type.replace('_', ' ').toUpperCase() }}</span>
            <span class="ver-tag">v{{ pkg.version }}</span>
          </div>
        </div>

        <div class="card-content">
          <h3 class="pkg-title">{{ pkg.name }}</h3>
          <div class="pkg-meta-row">
            <span class="pkg-author">Autore: {{ pkg.author }}</span>
            <span v-if="pkg.license" class="pkg-license-tag">{{ pkg.license }}</span>
          </div>
          <p class="pkg-desc">{{ pkg.description }}</p>

          <!-- Capabilities badges -->
          <div v-if="pkg.capabilities && pkg.capabilities.length > 0" class="caps-tags-row">
            <span v-for="cap in pkg.capabilities.slice(0, 3)" :key="cap" class="cap-tag">
              {{ cap }}
            </span>
            <span v-if="pkg.capabilities.length > 3" class="cap-tag more-caps">
              +{{ pkg.capabilities.length - 3 }}
            </span>
          </div>

          <!-- Open Source Code Link -->
          <div v-if="pkg.repository_url" class="source-link-box">
            <a
              :href="pkg.repository_url"
              target="_blank"
              rel="noopener noreferrer"
              class="source-link"
              @click.stop
              :aria-label="`Visualizza codice sorgente di ${pkg.name}`"
            >
              <Code :size="14" />
              <span>Codice Sorgente Open Source</span>
              <ExternalLink :size="12" />
            </a>
          </div>
        </div>

        <div class="card-bottom">
          <!-- Pending review approval controls -->
          <div v-if="pkg.review_status === 'pending_review'" class="review-actions-row">
            <button
              type="button"
              class="btn-approve"
              @click="approvePackage(pkg, $event)"
              aria-label="Approva e pubblica pacchetto"
            >
              <Check :size="14" /> Approva
            </button>
            <button
              type="button"
              class="btn-reject"
              @click="rejectPackage(pkg, $event)"
              aria-label="Rifiuta pacchetto"
            >
              <X :size="14" /> Rifiuta
            </button>
          </div>

          <!-- Normal install button -->
          <button
            v-else
            type="button"
            :class="['install-btn', { installed: pkg.installed }]"
            @click="toggleInstall(pkg, $event)"
          >
            <CheckCircle v-if="pkg.installed" :size="16" />
            <Download v-else :size="16" />
            <span>{{ pkg.installed ? `${t('store.installed', 'Installato')} (${t('store.uninstall', 'Disinstalla')})` : `${t('store.install', 'Installa')} con 1 Click` }}</span>
          </button>
        </div>
      </article>
    </div>

    <!-- Modal: Proponi Plugin Community (Open Source) -->
    <div
      v-if="showSubmitModal"
      class="modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="submit-modal-title"
      @click.self="showSubmitModal = false"
      @keydown.esc="showSubmitModal = false"
    >
      <div class="modal-card glass-panel modal-lg">
        <header class="modal-header">
          <div class="modal-title-box">
            <div class="pkg-icon-box wasm_plugin">
              <PlusCircle :size="24" />
            </div>
            <div>
              <h2 id="submit-modal-title" class="modal-title">Proponi un Plugin Open Source</h2>
              <span class="pkg-id-code">Invia un modulo WASI o blueprint per la revisione della community</span>
            </div>
          </div>
          <button
            type="button"
            class="btn-close"
            aria-label="Chiudi finestra"
            @click="showSubmitModal = false"
          >
            <X :size="20" />
          </button>
        </header>

        <form @submit.prevent="handleCommunitySubmit" class="modal-body form-body">
          <div class="open-source-alert">
            <ShieldCheck :size="20" class="text-primary" />
            <div>
              <strong>Requisito Codice Sorgente Aperto:</strong> Tutti i plugin della community devono avere il codice sorgente pubblicato e consultabile pubblicamente con licenza libera (es. GitHub o GitLab) per consentire l'audit di sicurezza sandbox prima dell'approvazione.
            </div>
          </div>

          <div v-if="submitError" class="error-banner">
            <AlertTriangle :size="16" />
            <span>{{ submitError }}</span>
          </div>

          <div class="form-grid">
            <div class="form-field">
              <label for="pkg-name">Nome del Modulo *</label>
              <input
                id="pkg-name"
                v-model="submitForm.name"
                type="text"
                placeholder="es. M-Bus Smart Water Reader"
                required
                class="form-input"
              />
            </div>

            <div class="form-field">
              <label for="pkg-id">Package ID (univoco) *</label>
              <input
                id="pkg-id"
                v-model="submitForm.package_id"
                type="text"
                placeholder="es. domus.community.mbus-water"
                required
                class="form-input font-mono"
              />
            </div>

            <div class="form-field">
              <label for="pkg-type">Tipo di Estensione *</label>
              <select id="pkg-type" v-model="submitForm.type" class="form-input">
                <option value="wasm_plugin">WASM / WASI Plugin (Sandboxed)</option>
                <option value="blueprint">Blueprint Automazione</option>
                <option value="widget">Widget Dashboard UI</option>
              </select>
            </div>

            <div class="form-field">
              <label for="pkg-author">Autore / Community Team</label>
              <input
                id="pkg-author"
                v-model="submitForm.author"
                type="text"
                placeholder="es. GreenEnergy Labs"
                class="form-input"
              />
            </div>

            <div class="form-field full-width">
              <label for="pkg-repo">URL Repository Open Source Pubblico (GitHub/GitLab) *</label>
              <input
                id="pkg-repo"
                v-model="submitForm.repository_url"
                type="url"
                placeholder="https://github.com/tuo-team/domus-wasi-plugin"
                required
                class="form-input font-mono"
              />
            </div>

            <div class="form-field">
              <label for="pkg-license">Licenza Open Source *</label>
              <select id="pkg-license" v-model="submitForm.license" class="form-input">
                <option value="Apache-2.0">Apache 2.0</option>
                <option value="MIT">MIT</option>
                <option value="GPL-3.0">GPL v3</option>
                <option value="BSD-3-Clause">BSD 3-Clause</option>
                <option value="MPL-2.0">Mozilla Public License 2.0</option>
              </select>
            </div>

            <div class="form-field">
              <label for="pkg-caps">WASI Capabilities richieste</label>
              <input
                id="pkg-caps"
                v-model="submitForm.capabilities"
                type="text"
                placeholder="wasi:serial-port, events:write"
                class="form-input font-mono"
              />
            </div>

            <div class="form-field full-width">
              <label for="pkg-desc">Descrizione e Funzionalità</label>
              <textarea
                id="pkg-desc"
                v-model="submitForm.description"
                rows="3"
                placeholder="Descrivi cosa fa il modulo, l'hardware supportato e i casi d'uso..."
                class="form-input"
              ></textarea>
            </div>
          </div>

          <footer class="modal-footer">
            <button
              type="button"
              class="btn-secondary"
              @click="showSubmitModal = false"
            >
              Annulla
            </button>
            <button
              type="submit"
              class="btn-primary-action"
              :disabled="isSubmitting"
            >
              <CheckCircle v-if="!isSubmitting" :size="16" />
              <span>{{ isSubmitting ? 'Pubblicazione in corso...' : 'Invia per Revisione' }}</span>
            </button>
          </footer>
        </form>
      </div>
    </div>

    <!-- Detail & Manifest Inspection Modal -->
    <div
      v-if="selectedPackage"
      class="modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="package-modal-title"
      @click.self="closeDetails"
      @keydown.esc="closeDetails"
    >
      <div class="modal-card glass-panel">
        <header class="modal-header">
          <div class="modal-title-box">
            <div class="pkg-icon-box modal-icon" :class="selectedPackage.type">
              <Box v-if="selectedPackage.type === 'wasm_plugin'" :size="24" />
              <Sun v-else-if="selectedPackage.type === 'blueprint'" :size="24" />
              <Gauge v-else :size="24" />
            </div>
            <div>
              <h2 id="package-modal-title" class="modal-title">{{ selectedPackage.name }}</h2>
              <span class="pkg-id-code font-mono">{{ selectedPackage.package_id }}</span>
            </div>
          </div>
          <button
            type="button"
            class="btn-close"
            aria-label="Chiudi finestra"
            @click="closeDetails"
          >
            <X :size="20" />
          </button>
        </header>

        <div class="modal-body">
          <div class="info-row">
            <span class="info-label">Tipo Pacchetto:</span>
            <span class="type-pill" :class="selectedPackage.type">{{ selectedPackage.type.toUpperCase() }}</span>
          </div>

          <div class="info-row">
            <span class="info-label">Verifica & Livello:</span>
            <span v-if="selectedPackage.review_status === 'pending_review'" class="badge-pending">
              In Attesa di Revisione
            </span>
            <span v-else-if="selectedPackage.tier === 'community'" class="badge-community">
              Community Open Source
            </span>
            <span v-else class="badge-verified">
              Domus Verified
            </span>
          </div>

          <div class="info-row">
            <span class="info-label">Versione:</span>
            <span class="font-mono">v{{ selectedPackage.version }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Autore / Publisher:</span>
            <span>{{ selectedPackage.author }}</span>
          </div>
          <div v-if="selectedPackage.license" class="info-row">
            <span class="info-label">Licenza Software:</span>
            <span class="pkg-license-tag">{{ selectedPackage.license }}</span>
          </div>

          <!-- Open Source Repository Link in Modal -->
          <div v-if="selectedPackage.repository_url" class="info-row">
            <span class="info-label">Codice Sorgente:</span>
            <a
              :href="selectedPackage.repository_url"
              target="_blank"
              rel="noopener noreferrer"
              class="source-link"
            >
              <Code :size="14" />
              <span>Visualizza Repository Pubblico</span>
              <ExternalLink :size="12" />
            </a>
          </div>

          <div class="info-row">
            <span class="info-label">Stato Installazione:</span>
            <span :class="selectedPackage.installed ? 'text-success' : 'text-muted'">
              {{ selectedPackage.installed ? 'Installata e Attiva nel Sistema' : 'Non Installata' }}
            </span>
          </div>

          <!-- Capabilities -->
          <div v-if="selectedPackage.capabilities && selectedPackage.capabilities.length > 0" class="desc-section">
            <h4 class="section-title">Permessi Sandbox (WASI Capabilities)</h4>
            <div class="caps-tags-row">
              <span v-for="cap in selectedPackage.capabilities" :key="cap" class="cap-tag">
                {{ cap }}
              </span>
            </div>
          </div>

          <div class="desc-section">
            <h4 class="section-title">Descrizione & Funzionalità</h4>
            <p class="section-text">{{ selectedPackage.description }}</p>
          </div>

          <div class="security-box">
            <ShieldCheck :size="18" class="text-success" />
            <div class="security-text">
              <strong>Sicurezza Domus Verified:</strong> Esecuzione in sandbox WebAssembly (WASI) con memoria lineare isolata, permessi rigorosamente controllati e zero chiamate cloud non autorizzate.
            </div>
          </div>
        </div>

        <footer class="modal-footer">
          <button
            type="button"
            class="btn-secondary"
            @click="closeDetails"
          >
            Chiudi
          </button>
          <button
            v-if="selectedPackage.review_status !== 'pending_review'"
            type="button"
            :class="['install-btn modal-install-btn', { installed: selectedPackage.installed }]"
            @click="toggleInstall(selectedPackage)"
          >
            <CheckCircle v-if="selectedPackage.installed" :size="16" />
            <Download v-else :size="16" />
            <span>{{ selectedPackage.installed ? 'Disinstalla' : 'Installa Subito con 1 Click' }}</span>
          </button>
        </footer>
      </div>
    </div>
  </div>
</template>

<style scoped>
.store-view-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.store-header {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  border-radius: 16px;
  background: var(--surface-card, rgba(19, 28, 46, 0.6));
  backdrop-filter: blur(12px);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
}

.header-main-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}

.header-title-box {
  display: flex;
  align-items: center;
  gap: 14px;
}

.icon-circle {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: rgba(14, 165, 233, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-title {
  font-size: 1.4rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-main, #ffffff);
}

.page-subtitle {
  font-size: 0.88rem;
  color: var(--text-muted, #94a3b8);
  margin: 4px 0 0;
}

.btn-propose-community {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.25), rgba(14, 165, 233, 0.25));
  border: 1px solid rgba(99, 102, 241, 0.5);
  border-radius: 10px;
  color: #c7d2fe;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.2s;
  min-height: 44px;
}

.btn-propose-community:hover {
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.4), rgba(14, 165, 233, 0.4));
  border-color: #818cf8;
  color: #fff;
}

.search-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 14px;
  color: var(--text-muted, #94a3b8);
  pointer-events: none;
}

.store-search-input {
  width: 100%;
  padding: 12px 38px;
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  border-radius: 10px;
  color: var(--text-main, #fff);
  font-size: 0.95rem;
  outline: none;
  transition: border-color 0.2s;
}

.store-search-input:focus {
  border-color: var(--primary, #0ea5e9);
}

.clear-search-btn {
  position: absolute;
  right: 12px;
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: 14px;
  cursor: pointer;
  padding: 4px 8px;
}

.filters-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-top: 6px;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.filter-group-label {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-muted, #94a3b8);
  min-width: 120px;
}

.filter-pills {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.pill-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: var(--text-muted, #94a3b8);
  font-size: 0.82rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  min-height: 38px;
}

.pill-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-main, #fff);
}

.pill-btn.active {
  background: var(--primary, #0ea5e9);
  border-color: var(--primary, #0ea5e9);
  color: #fff;
}

.pill-btn.tier-verified.active {
  background: #059669;
  border-color: #10b981;
}

.pill-btn.tier-community.active {
  background: #6366f1;
  border-color: #818cf8;
}

.pill-btn.tier-pending.active {
  background: #d97706;
  border-color: #f59e0b;
}

.pill-badge {
  font-size: 0.72rem;
  padding: 2px 6px;
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.25);
}

.pending-badge {
  background: #f59e0b;
  color: #000;
  font-weight: 700;
}

/* Packages Grid */
.packages-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(330px, 1fr));
  gap: 20px;
}

.package-card {
  border-radius: 16px;
  background: var(--surface-card, rgba(19, 28, 46, 0.6));
  backdrop-filter: blur(12px);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  padding: 20px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 16px;
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}

.package-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3);
}

.package-card:focus-visible {
  outline: 2px solid var(--primary, #0ea5e9);
  outline-offset: 2px;
}

.card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.pkg-icon-box {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.pkg-icon-box.wasm_plugin { background: rgba(99, 102, 241, 0.2); color: #818cf8; }
.pkg-icon-box.blueprint { background: rgba(245, 158, 11, 0.2); color: #f59e0b; }
.pkg-icon-box.widget { background: rgba(16, 185, 129, 0.2); color: #10b981; }

.pkg-badges {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.badge-verified {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.7rem;
  font-weight: 700;
  padding: 2px 7px;
  border-radius: 6px;
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.badge-community {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.7rem;
  font-weight: 700;
  padding: 2px 7px;
  border-radius: 6px;
  background: rgba(99, 102, 241, 0.15);
  color: #a5b4fc;
  border: 1px solid rgba(99, 102, 241, 0.3);
}

.badge-pending {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.7rem;
  font-weight: 700;
  padding: 2px 7px;
  border-radius: 6px;
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.badge-rejected {
  font-size: 0.7rem;
  font-weight: 700;
  padding: 2px 7px;
  border-radius: 6px;
  background: rgba(239, 68, 68, 0.15);
  color: #f87171;
}

.type-tag {
  font-size: 0.7rem;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-muted);
}

.ver-tag {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.card-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.pkg-title {
  font-size: 1.05rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-main, #fff);
  line-height: 1.35;
}

.pkg-meta-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pkg-author {
  font-size: 0.78rem;
  color: var(--text-muted, #94a3b8);
}

.pkg-license-tag {
  font-size: 0.7rem;
  padding: 1px 6px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.07);
  color: var(--text-muted);
  font-family: monospace;
}

.pkg-desc {
  font-size: 0.85rem;
  color: var(--text-muted, #94a3b8);
  margin: 6px 0 0;
  line-height: 1.45;
}

.caps-tags-row {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 8px;
}

.cap-tag {
  font-size: 0.7rem;
  font-family: monospace;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(14, 165, 233, 0.1);
  color: #38bdf8;
  border: 1px solid rgba(14, 165, 233, 0.2);
}

.cap-tag.more-caps {
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-muted);
  border-color: transparent;
}

.source-link-box {
  margin-top: 8px;
}

.source-link {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 0.78rem;
  color: #818cf8;
  text-decoration: none;
  font-weight: 600;
  transition: color 0.15s;
}

.source-link:hover {
  color: #c7d2fe;
  text-decoration: underline;
}

.card-bottom {
  margin-top: 8px;
}

.install-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px;
  border-radius: 10px;
  border: none;
  background: var(--primary, #0ea5e9);
  color: #fff;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.2s;
  min-height: 44px;
}

.install-btn:hover {
  filter: brightness(1.1);
}

.install-btn.installed {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.install-btn.installed:hover {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
  border-color: rgba(239, 68, 68, 0.3);
}

.review-actions-row {
  display: flex;
  gap: 8px;
}

.btn-approve {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 9px;
  border-radius: 8px;
  border: none;
  background: #059669;
  color: #fff;
  font-weight: 600;
  font-size: 0.85rem;
  cursor: pointer;
  min-height: 44px;
}

.btn-reject {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 9px;
  border-radius: 8px;
  border: 1px solid rgba(239, 68, 68, 0.4);
  background: rgba(239, 68, 68, 0.15);
  color: #f87171;
  font-weight: 600;
  font-size: 0.85rem;
  cursor: pointer;
  min-height: 44px;
}

/* Modals */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(6px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 16px;
}

.modal-card {
  width: 100%;
  max-width: 580px;
  border-radius: 20px;
  background: var(--surface-card, #131c2e);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.15));
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
  overflow: hidden;
  color: var(--text-main, #fff);
  max-height: 90vh;
  display: flex;
  flex-direction: column;
}

.modal-card.modal-lg {
  max-width: 680px;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
}

.modal-title-box {
  display: flex;
  align-items: center;
  gap: 14px;
}

.modal-icon {
  width: 48px;
  height: 48px;
}

.modal-title {
  margin: 0;
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--text-main);
}

.pkg-id-code {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.btn-close {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 6px;
  border-radius: 8px;
  transition: all 0.15s;
}

.btn-close:hover {
  color: var(--text-main);
  background: rgba(255, 255, 255, 0.1);
}

.modal-body {
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow-y: auto;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.85rem;
}

.info-label {
  color: var(--text-muted);
}

.type-pill {
  font-size: 0.7rem;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.1);
}

.type-pill.wasm_plugin { background: rgba(99, 102, 241, 0.25); color: #a5b4fc; }
.type-pill.blueprint { background: rgba(245, 158, 11, 0.25); color: #fcd34d; }
.type-pill.widget { background: rgba(16, 185, 129, 0.25); color: #6ee7b7; }

.desc-section {
  margin-top: 6px;
}

.section-title {
  margin: 0 0 6px 0;
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--text-main);
}

.section-text {
  margin: 0;
  font-size: 0.85rem;
  line-height: 1.5;
  color: var(--text-muted);
}

.security-box {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px;
  border-radius: 10px;
  background: rgba(16, 185, 129, 0.08);
  border: 1px solid rgba(16, 185, 129, 0.2);
  margin-top: 6px;
}

.security-text {
  font-size: 0.78rem;
  color: #10b981;
  line-height: 1.4;
}

.open-source-alert {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px;
  border-radius: 12px;
  background: rgba(99, 102, 241, 0.1);
  border: 1px solid rgba(99, 102, 241, 0.3);
  font-size: 0.82rem;
  line-height: 1.45;
  color: #c7d2fe;
}

.error-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: 8px;
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.4);
  color: #f87171;
  font-size: 0.85rem;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-field.full-width {
  grid-column: 1 / -1;
}

.form-field label {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-muted);
}

.form-input {
  padding: 10px 14px;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  border-radius: 8px;
  color: #fff;
  font-size: 0.88rem;
  outline: none;
}

.form-input:focus {
  border-color: var(--primary, #0ea5e9);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  border-top: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  background: rgba(0, 0, 0, 0.1);
}

.btn-secondary {
  padding: 9px 18px;
  border-radius: 10px;
  background: var(--bg-surface, rgba(255, 255, 255, 0.05));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  color: var(--text-main, #fff);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  min-height: 44px;
}

.btn-primary-action {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 20px;
  border-radius: 10px;
  border: none;
  background: var(--primary, #0ea5e9);
  color: #fff;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  min-height: 44px;
}

.btn-primary-action:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.modal-install-btn {
  width: auto;
  min-width: 170px;
  padding: 9px 20px;
}
</style>
