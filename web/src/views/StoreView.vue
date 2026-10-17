<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ShoppingBag,
  Sparkles,
  Download,
  Trash2,
  CheckCircle,
  PackageCheck,
  Search,
  Box,
  Sun,
  Gauge,
  Info,
  X,
  ExternalLink,
  ShieldCheck,
  Layers
} from '@lucide/vue'
import { apiClient } from '../api/client'

const { t } = useI18n()

interface StorePackage {
  id: string
  package_id: string
  name: string
  type: 'wasm_plugin' | 'blueprint' | 'widget'
  version: string
  author: string
  description: string
  installed: boolean
  installed_at?: string
  icon?: string
}

const activeType = ref<string>('')
const searchQuery = ref<string>('')
const selectedPackage = ref<StorePackage | null>(null)

const packages = ref<StorePackage[]>([])

const countByType = (type: string) => {
  if (!type) return packages.value.length
  return packages.value.filter(p => p.type === type).length
}

const filteredPackages = computed(() => {
  return packages.value.filter(p => {
    const matchesType = !activeType.value || p.type === activeType.value
    const q = searchQuery.value.trim().toLowerCase()
    const matchesSearch = !q ||
      p.name.toLowerCase().includes(q) ||
      p.description.toLowerCase().includes(q) ||
      p.author.toLowerCase().includes(q) ||
      p.package_id.toLowerCase().includes(q)
    return matchesType && matchesSearch
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
      <div class="header-left">
        <div class="header-title-box">
          <div class="icon-circle">
            <ShoppingBag :size="24" class="text-primary" />
          </div>
          <div>
            <h1 class="page-title">{{ t('store.title') }}</h1>
            <p class="page-subtitle">{{ t('store.subtitle') }}</p>
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
            placeholder="Cerca per nome, autore o package ID..."
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
      </div>

      <!-- Type Filter Pills -->
      <div class="filter-pills" role="tablist" aria-label="Filtro tipo estensioni">
        <button
          type="button"
          :class="['pill-btn', { active: activeType === '' }]"
          @click="activeType = ''"
        >
          Tutti <span class="pill-badge">{{ packages.length }}</span>
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
    </header>

    <!-- Empty State -->
    <div v-if="filteredPackages.length === 0" class="empty-state glass-panel">
      <PackageCheck :size="48" class="empty-icon text-muted" />
      <p class="empty-title">{{ searchQuery ? 'Nessun risultato per la ricerca' : t('common.no_data', 'Nessun pacchetto disponibile') }}</p>
      <p class="empty-desc">
        {{ searchQuery ? 'Prova con parole chiave differenti come "bess", "circadiano", o "clima".' : 'I pacchetti o blueprint caricati nel sistema appariranno qui.' }}
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
            <Box v-if="pkg.type === 'wasm_plugin'" :size="22" />
            <Sun v-else-if="pkg.type === 'blueprint'" :size="22" />
            <Gauge v-else :size="22" />
          </div>

          <div class="pkg-badges">
            <span class="type-tag">{{ pkg.type.replace('_', ' ').toUpperCase() }}</span>
            <span class="ver-tag">v{{ pkg.version }}</span>
          </div>
        </div>

        <div class="card-content">
          <h3 class="pkg-title">{{ pkg.name }}</h3>
          <span class="pkg-author">Autore: {{ pkg.author }}</span>
          <p class="pkg-desc">{{ pkg.description }}</p>
        </div>

        <div class="card-bottom">
          <button
            type="button"
            :class="['install-btn', { installed: pkg.installed }]"
            @click="toggleInstall(pkg, $event)"
          >
            <CheckCircle v-if="pkg.installed" :size="16" />
            <Download v-else :size="16" />
            <span>{{ pkg.installed ? `${t('store.installed')} (${t('store.uninstall')})` : `${t('store.install')} con 1 Click` }}</span>
          </button>
        </div>
      </article>
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
            <span class="info-label">Versione:</span>
            <span class="font-mono">v{{ selectedPackage.version }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Autore / Publisher:</span>
            <span>{{ selectedPackage.author }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Stato Installazione:</span>
            <span :class="selectedPackage.installed ? 'text-success' : 'text-muted'">
              {{ selectedPackage.installed ? 'Installata e Attiva nel Sistema' : 'Non Installata' }}
            </span>
          </div>

          <div class="desc-section">
            <h4 class="section-title">Descrizione & Funzionalità</h4>
            <p class="section-text">{{ selectedPackage.description }}</p>
          </div>

          <div class="security-box">
            <ShieldCheck :size="18" class="text-success" />
            <div class="security-text">
              <strong>Sicurezza Domus Verified:</strong> Esecuzione in sandbox WASM con memoria protetta, isolamento locale e zero chiamate cloud non autorizzate.
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
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 20px 24px;
  border-radius: 16px;
  background: var(--surface-card, rgba(30, 41, 59, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
}

.header-left {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 24px;
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
  background: rgba(14, 165, 233, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-title {
  font-size: 1.4rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-main);
}

.page-subtitle {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin: 4px 0 0;
}

.search-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  min-width: 260px;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--text-muted);
  pointer-events: none;
}

.store-search-input {
  width: 100%;
  padding: 8px 32px 8px 36px;
  border-radius: 10px;
  background: var(--bg-surface);
  border: 1px solid var(--border-color);
  color: var(--text-main);
  font-size: 0.85rem;
  transition: all 0.2s ease;
}

.store-search-input:focus {
  outline: none;
  border-color: var(--primary);
  background: var(--bg-surface-hover);
  box-shadow: 0 0 0 2px rgba(14, 165, 233, 0.25);
}

.clear-search-btn {
  position: absolute;
  right: 10px;
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: 0.8rem;
  cursor: pointer;
}

.filter-pills {
  display: flex;
  gap: 8px;
  background: var(--bg-surface);
  border: 1px solid var(--border-color);
  padding: 4px;
  border-radius: 10px;
}

.pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: var(--text-muted);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.pill-btn.active {
  background: var(--primary);
  color: #fff;
}

.pill-badge {
  font-size: 0.7rem;
  padding: 1px 6px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.15);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  border-radius: 16px;
  text-align: center;
  background: var(--surface-card, rgba(30, 41, 59, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
}

.empty-icon {
  margin-bottom: 16px;
}

.empty-title {
  font-size: 1.1rem;
  font-weight: 700;
  margin: 0 0 6px 0;
  color: var(--text-main);
}

.empty-desc {
  font-size: 0.85rem;
  color: var(--text-muted);
  max-width: 420px;
  margin: 0;
}

.packages-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 20px;
}

.package-card {
  padding: 22px;
  border-radius: 16px;
  background: var(--surface-card, rgba(30, 41, 59, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 16px;
  cursor: pointer;
  transition: transform 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
}

.package-card:hover {
  transform: translateY(-2px);
  border-color: rgba(14, 165, 233, 0.4);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
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
  width: 42px;
  height: 42px;
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
  gap: 6px;
}

.type-tag {
  font-size: 0.7rem;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 6px;
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
}

.ver-tag {
  font-size: 0.7rem;
  color: var(--text-muted);
  padding: 3px 6px;
}

.card-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.pkg-title {
  font-size: 1.05rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-main);
}

.pkg-author {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.pkg-desc {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin: 8px 0 0;
  line-height: 1.45;
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

/* Modal Detail Inspection */
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
  color: var(--text-main);
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
  background: var(--bg-surface);
  border: 1px solid var(--border-color);
  color: var(--text-main);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-secondary:hover {
  background: var(--bg-surface-hover);
}

.modal-install-btn {
  width: auto;
  min-width: 170px;
  padding: 9px 20px;
}
</style>
