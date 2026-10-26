<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useEntitiesStore } from '../stores/entities'
import { useAreasStore } from '../stores/areas'
import { useNotificationsStore } from '../stores/notifications'
import LightCard from '../components/cards/LightCard.vue'
import SwitchCard from '../components/cards/SwitchCard.vue'
import SensorCard from '../components/cards/SensorCard.vue'
import ClimateCard from '../components/cards/ClimateCard.vue'
import AlarmControlPanelCard from '../components/cards/AlarmControlPanelCard.vue'
import {
  Lightbulb,
  Power,
  Radio,
  Thermometer,
  Sparkles,
  Shield,
  ShoppingBag,
  ArrowRight,
  Download,
  CheckCircle,
  ExternalLink,
  ShieldCheck,
  Globe,
  Search,
  X,
  Box,
  Cpu,
  Sun,
  Gauge,
  Share2,
  Activity,
  Volume2,
  Smartphone,
  TrendingDown,
  Camera
} from '@lucide/vue'
import { apiClient } from '../api/client'
import type { Entity, StorePackage } from '../api/types'

const { t } = useI18n()
const router = useRouter()
const entitiesStore = useEntitiesStore()
const areasStore = useAreasStore()
const notifStore = useNotificationsStore()

const activeDomainFilter = ref<string>('all')
const packages = ref<StorePackage[]>([])
const marketplaceSearchQuery = ref<string>('')
const marketplaceTypeFilter = ref<string>('')
const selectedPackage = ref<StorePackage | null>(null)
const installingPackageId = ref<string | null>(null)

async function fetchPackages() {
  try {
    const list = await apiClient.get<StorePackage[]>('/api/v1/store/packages')
    packages.value = list || []
  } catch {
    packages.value = []
  }
}

onMounted(async () => {
  try {
    await Promise.all([
      entitiesStore.fetchEntities(),
      areasStore.fetchAreas(),
      fetchPackages()
    ])
  } catch (err) {
    console.error('Error fetching dashboard initial data:', err)
  }
})

// Metrics
const activeLightsCount = computed(() => {
  return entitiesStore.lights.filter((l) => l.state === 'on').length
})

const activeSwitchesCount = computed(() => {
  return entitiesStore.switches.filter((s) => s.state === 'on').length
})

const totalSensorsCount = computed(() => {
  return entitiesStore.sensors.length + entitiesStore.binarySensors.length
})

const installedPackagesCount = computed(() => {
  return packages.value.filter((p) => p.installed).length
})

// Filtered Entities
const filteredEntities = computed(() => {
  const all = entitiesStore.allEntities
  if (activeDomainFilter.value === 'all') return all
  if (activeDomainFilter.value === 'light') return entitiesStore.lights
  if (activeDomainFilter.value === 'switch') return entitiesStore.switches
  if (activeDomainFilter.value === 'sensor') return [...entitiesStore.sensors, ...entitiesStore.binarySensors]
  if (activeDomainFilter.value === 'climate') return entitiesStore.byDomain('climate')
  return all
})

const filterOptions = computed(() => [
  { id: 'all', label: t('dashboard.filterAll'), icon: null },
  { id: 'alarm', label: 'Allarme', icon: Shield },
  { id: 'light', label: t('dashboard.filterLights'), icon: Lightbulb },
  { id: 'switch', label: t('dashboard.filterSwitches'), icon: Power },
  { id: 'sensor', label: t('dashboard.filterSensors'), icon: Radio },
  { id: 'climate', label: t('dashboard.filterClimate'), icon: Thermometer },
  { id: 'marketplace', label: t('dashboard.filterMarketplace', 'Marketplace'), icon: ShoppingBag }
])

// Showcase packages for overview (top 4 packages)
const showcasePackages = computed(() => {
  if (packages.value.length === 0) return []
  return packages.value.slice(0, 4)
})

// Filtered packages for dedicated marketplace tab
const filteredMarketplacePackages = computed(() => {
  return packages.value.filter((p) => {
    const matchesType = !marketplaceTypeFilter.value || p.type === marketplaceTypeFilter.value
    const q = marketplaceSearchQuery.value.trim().toLowerCase()
    const matchesQuery =
      !q ||
      p.name.toLowerCase().includes(q) ||
      p.description.toLowerCase().includes(q) ||
      p.author.toLowerCase().includes(q) ||
      p.package_id.toLowerCase().includes(q)
    return matchesType && matchesQuery
  })
})

function getPackageIcon(icon?: string, type?: string) {
  if (icon === 'cpu') return Cpu
  if (icon === 'radio') return Radio
  if (icon === 'sun') return Sun
  if (icon === 'share-2') return Share2
  if (icon === 'activity') return Activity
  if (icon === 'volume-2') return Volume2
  if (icon === 'smartphone') return Smartphone
  if (icon === 'trending-down') return TrendingDown
  if (icon === 'camera') return Camera
  if (type === 'wasm_plugin') return Box
  if (type === 'blueprint') return Sun
  return Gauge
}

async function toggleInstall(pkg: StorePackage, event?: Event) {
  if (event) {
    event.stopPropagation()
  }
  installingPackageId.value = pkg.id
  try {
    if (pkg.installed) {
      await apiClient.post(`/api/v1/store/packages/${pkg.id}/uninstall`, {})
      pkg.installed = false
      pkg.installed_at = undefined
      notifStore.success(pkg.name, 'Estensione disinstallata con successo')
    } else {
      await apiClient.post(`/api/v1/store/packages/${pkg.id}/install`, {})
      pkg.installed = true
      pkg.installed_at = new Date().toISOString()
      notifStore.success(pkg.name, 'Estensione installata e attiva in locale')
    }
  } catch (err) {
    console.error('Failed to toggle install package:', err)
    notifStore.error(pkg.name, 'Errore durante la modifica dello stato del pacchetto')
  } finally {
    installingPackageId.value = null
  }
}

function openDetails(pkg: StorePackage) {
  selectedPackage.value = pkg
}

function closeDetails() {
  selectedPackage.value = null
}

function navigateToStore() {
  if (router) {
    router.push('/store')
  } else {
    activeDomainFilter.value = 'marketplace'
  }
}
</script>

<template>
  <div class="dashboard-view">
    <!-- Quick Summary Cards -->
    <div class="summary-grid">
      <div class="summary-card glass-card">
        <div class="summary-icon icon-light">
          <Lightbulb :size="20" />
        </div>
        <div class="summary-info">
          <span class="summary-label">{{ t('dashboard.activeLights') }}</span>
          <span class="summary-value">{{ activeLightsCount }} {{ t('dashboard.activeCount') }}</span>
        </div>
      </div>

      <div class="summary-card glass-card">
        <div class="summary-icon icon-switch">
          <Power :size="20" />
        </div>
        <div class="summary-info">
          <span class="summary-label">{{ t('dashboard.activeSwitches') }}</span>
          <span class="summary-value">{{ activeSwitchesCount }} {{ t('dashboard.activeCount') }}</span>
        </div>
      </div>

      <div class="summary-card glass-card">
        <div class="summary-icon icon-sensor">
          <Radio :size="20" />
        </div>
        <div class="summary-info">
          <span class="summary-label">{{ t('dashboard.onlineSensors') }}</span>
          <span class="summary-value">{{ totalSensorsCount }} {{ t('dashboard.detectedCount') }}</span>
        </div>
      </div>

      <!-- Marketplace Summary Badge -->
      <div
        class="summary-card glass-card summary-card-clickable"
        tabindex="0"
        role="button"
        :aria-label="`${t('dashboard.installedExtensions')}: ${installedPackagesCount}`"
        @click="activeDomainFilter = 'marketplace'"
        @keydown.enter="activeDomainFilter = 'marketplace'"
      >
        <div class="summary-icon icon-store">
          <ShoppingBag :size="20" />
        </div>
        <div class="summary-info">
          <span class="summary-label">{{ t('dashboard.installedExtensions') }}</span>
          <span class="summary-value">{{ installedPackagesCount }} / {{ packages.length }}</span>
        </div>
      </div>
    </div>

    <!-- Filter Pills Row -->
    <div class="filters-row" role="tablist" aria-label="Filtri entità e sezioni della dashboard">
      <button
        v-for="pill in filterOptions"
        :key="pill.id"
        type="button"
        role="tab"
        :aria-selected="activeDomainFilter === pill.id"
        :class="['filter-pill', { active: activeDomainFilter === pill.id }]"
        @click="activeDomainFilter = pill.id"
      >
        <component :is="pill.icon" v-if="pill.icon" :size="16" class="pill-icon" />
        <span>{{ pill.label }}</span>
      </button>
    </div>

    <!-- Alarm Control Panel View -->
    <div v-if="activeDomainFilter === 'alarm'" class="alarm-view-container">
      <AlarmControlPanelCard />
    </div>

    <!-- Marketplace Dedicated View -->
    <div v-else-if="activeDomainFilter === 'marketplace'" class="marketplace-dashboard-view">
      <header class="marketplace-header glass-card">
        <div class="marketplace-header-left">
          <div class="icon-circle">
            <ShoppingBag :size="24" class="text-primary" />
          </div>
          <div>
            <h2 class="marketplace-title">{{ t('dashboard.marketplaceTitle', 'Marketplace & Estensioni') }}</h2>
            <p class="marketplace-subtitle">
              {{ t('dashboard.marketplaceSubtitle', 'Moduli WASI sandboxati, blueprint e widget verificati') }}
            </p>
          </div>
        </div>

        <button
          type="button"
          class="btn-open-store"
          @click="navigateToStore"
          aria-label="Apri catalogo completo nel Domus Store"
        >
          <span>{{ t('dashboard.exploreStore', 'Esplora Domus Store') }}</span>
          <ArrowRight :size="16" />
        </button>
      </header>

      <!-- Controls: Search and Type Pills -->
      <div class="marketplace-controls">
        <div class="search-input-wrapper">
          <Search :size="16" class="search-icon" aria-hidden="true" />
          <input
            id="dash-marketplace-search"
            v-model="marketplaceSearchQuery"
            type="search"
            :placeholder="t('dashboard.searchMarketplace', 'Cerca nel marketplace...')"
            class="dash-search-input"
            aria-label="Cerca estensioni marketplace"
          />
          <button
            v-if="marketplaceSearchQuery"
            type="button"
            class="clear-search-btn"
            aria-label="Cancella ricerca"
            @click="marketplaceSearchQuery = ''"
          >
            <X :size="14" />
          </button>
        </div>

        <div class="type-pills" role="tablist" aria-label="Filtro per tipologia di modulo">
          <button
            type="button"
            :class="['type-pill', { active: marketplaceTypeFilter === '' }]"
            @click="marketplaceTypeFilter = ''"
          >
            {{ t('dashboard.allTypes', 'Tutti i Tipi') }}
          </button>
          <button
            type="button"
            :class="['type-pill', { active: marketplaceTypeFilter === 'wasm_plugin' }]"
            @click="marketplaceTypeFilter = 'wasm_plugin'"
          >
            {{ t('dashboard.wasmPlugins', 'WASM Plugins') }}
          </button>
          <button
            type="button"
            :class="['type-pill', { active: marketplaceTypeFilter === 'blueprint' }]"
            @click="marketplaceTypeFilter = 'blueprint'"
          >
            {{ t('dashboard.blueprints', 'Blueprints') }}
          </button>
          <button
            type="button"
            :class="['type-pill', { active: marketplaceTypeFilter === 'widget' }]"
            @click="marketplaceTypeFilter = 'widget'"
          >
            {{ t('dashboard.widgets', 'Widget UI') }}
          </button>
        </div>
      </div>

      <!-- Marketplace Cards Grid -->
      <div v-if="filteredMarketplacePackages.length > 0" class="marketplace-grid">
        <article
          v-for="pkg in filteredMarketplacePackages"
          :key="pkg.id"
          class="marketplace-card glass-card"
          tabindex="0"
          role="button"
          :aria-label="`Dettagli per ${pkg.name}`"
          @click="openDetails(pkg)"
          @keydown.enter="openDetails(pkg)"
        >
          <div class="mp-card-top">
            <div class="mp-icon-box" :class="pkg.type">
              <component :is="getPackageIcon(pkg.icon, pkg.type)" :size="20" />
            </div>

            <div class="mp-badges">
              <span v-if="pkg.tier === 'community'" class="badge-community">
                <Globe :size="12" /> Community
              </span>
              <span v-else class="badge-verified">
                <ShieldCheck :size="12" /> Verified
              </span>
              <span class="mp-type-tag">{{ pkg.type.replace('_', ' ').toUpperCase() }}</span>
              <span class="mp-ver-tag">v{{ pkg.version }}</span>
            </div>
          </div>

          <div class="mp-card-body">
            <h3 class="mp-pkg-name">{{ pkg.name }}</h3>
            <span class="mp-pkg-author">Autore: {{ pkg.author }}</span>
            <p class="mp-pkg-desc">{{ pkg.description }}</p>

            <div v-if="pkg.capabilities && pkg.capabilities.length > 0" class="mp-caps-row">
              <span v-for="cap in pkg.capabilities.slice(0, 2)" :key="cap" class="mp-cap-tag">
                {{ cap }}
              </span>
              <span v-if="pkg.capabilities.length > 2" class="mp-cap-tag more">
                +{{ pkg.capabilities.length - 2 }}
              </span>
            </div>
          </div>

          <div class="mp-card-footer">
            <button
              type="button"
              :class="['btn-install-toggle', { installed: pkg.installed }]"
              :disabled="installingPackageId === pkg.id"
              :aria-label="pkg.installed ? `Disinstalla ${pkg.name}` : `Installa ${pkg.name}`"
              @click="toggleInstall(pkg, $event)"
            >
              <CheckCircle v-if="pkg.installed" :size="15" />
              <Download v-else :size="15" />
              <span>{{ pkg.installed ? `${t('store.installed', 'Installato')} (${t('store.uninstall', 'Disinstalla')})` : t('dashboard.oneClickInstall', 'Installa con 1 Click') }}</span>
            </button>
          </div>
        </article>
      </div>

      <!-- Empty Marketplace Results -->
      <div v-else class="empty-state glass-card">
        <Box :size="36" class="empty-icon" />
        <h3 class="empty-title">{{ t('dashboard.noMarketplaceResults', 'Nessuna estensione trovata') }}</h3>
        <p class="empty-desc">
          {{ t('dashboard.noMarketplaceResultsDesc', 'Prova a modificare i termini di ricerca o la categoria selezionata.') }}
        </p>
      </div>
    </div>

    <!-- Entity Grid -->
    <div v-else-if="filteredEntities.length > 0" class="entity-grid">
      <template v-for="entity in filteredEntities" :key="entity.id">
        <LightCard v-if="entity.domain === 'light'" :entity="entity" />
        <SwitchCard v-else-if="entity.domain === 'switch'" :entity="entity" />
        <SensorCard v-else-if="entity.domain === 'sensor' || entity.domain === 'binary_sensor'" :entity="entity" />
        <ClimateCard v-else-if="entity.domain === 'climate'" :entity="entity" />
        <SwitchCard v-else :entity="entity" />
      </template>
    </div>

    <!-- Empty State for Entities -->
    <div v-else class="empty-state glass-card">
      <Sparkles :size="36" class="empty-icon" />
      <h3 class="empty-title">{{ t('dashboard.noEntities') }}</h3>
      <p class="empty-desc">
        {{ t('dashboard.noEntitiesDesc') }}
      </p>
    </div>

    <!-- Marketplace Showcase Section on Overview (activeDomainFilter === 'all') -->
    <section
      v-if="activeDomainFilter === 'all' && showcasePackages.length > 0"
      class="dashboard-marketplace-showcase glass-card"
      aria-labelledby="marketplace-showcase-title"
    >
      <header class="showcase-header">
        <div class="showcase-header-left">
          <div class="icon-circle icon-circle-small">
            <ShoppingBag :size="20" class="text-primary" />
          </div>
          <div>
            <div class="showcase-title-row">
              <h2 id="marketplace-showcase-title" class="showcase-title">
                {{ t('dashboard.marketplaceTitle', 'Marketplace & Estensioni') }}
              </h2>
              <span class="showcase-pill-badge">
                {{ t('dashboard.showcaseBadge', 'In Evidenza') }}
              </span>
            </div>
            <p class="showcase-subtitle">
              {{ t('dashboard.marketplaceSubtitle', 'Moduli WASI sandboxati, blueprint e widget verificati') }}
            </p>
          </div>
        </div>

        <button
          type="button"
          class="btn-showcase-link"
          @click="navigateToStore"
          aria-label="Visualizza tutte le estensioni del catalogo Domus Store"
        >
          <span>{{ t('dashboard.exploreStore', 'Esplora Domus Store') }}</span>
          <ArrowRight :size="16" />
        </button>
      </header>

      <div class="showcase-grid">
        <article
          v-for="pkg in showcasePackages"
          :key="pkg.id"
          class="showcase-card glass-panel"
          tabindex="0"
          role="button"
          :aria-label="`Dettagli per ${pkg.name}`"
          @click="openDetails(pkg)"
          @keydown.enter="openDetails(pkg)"
        >
          <div class="sc-card-top">
            <div class="mp-icon-box" :class="pkg.type">
              <component :is="getPackageIcon(pkg.icon, pkg.type)" :size="18" />
            </div>
            <div class="sc-badges">
              <span v-if="pkg.tier === 'community'" class="badge-community">
                <Globe :size="11" /> Community
              </span>
              <span v-else class="badge-verified">
                <ShieldCheck :size="11" /> Verified
              </span>
              <span class="mp-ver-tag">v{{ pkg.version }}</span>
            </div>
          </div>

          <h3 class="sc-pkg-name">{{ pkg.name }}</h3>
          <p class="sc-pkg-desc">{{ pkg.description }}</p>

          <div class="sc-card-bottom">
            <span class="sc-author">{{ pkg.author }}</span>
            <button
              type="button"
              :class="['btn-install-mini', { installed: pkg.installed }]"
              :disabled="installingPackageId === pkg.id"
              :aria-label="pkg.installed ? `Disinstalla ${pkg.name}` : `Installa ${pkg.name}`"
              @click="toggleInstall(pkg, $event)"
            >
              <CheckCircle v-if="pkg.installed" :size="14" />
              <Download v-else :size="14" />
              <span>{{ pkg.installed ? t('store.installed', 'Installato') : t('store.install', 'Installa') }}</span>
            </button>
          </div>
        </article>
      </div>
    </section>

    <!-- Package Details Modal -->
    <div
      v-if="selectedPackage"
      class="modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="dashboard-pkg-modal-title"
      @click.self="closeDetails"
      @keydown.esc="closeDetails"
    >
      <div class="modal-card glass-card">
        <header class="modal-header">
          <div class="modal-title-box">
            <div class="mp-icon-box modal-icon" :class="selectedPackage.type">
              <component :is="getPackageIcon(selectedPackage.icon, selectedPackage.type)" :size="24" />
            </div>
            <div>
              <h2 id="dashboard-pkg-modal-title" class="modal-title">{{ selectedPackage.name }}</h2>
              <span class="modal-pkg-id">{{ selectedPackage.package_id }}</span>
            </div>
          </div>
          <button
            type="button"
            class="btn-close"
            aria-label="Chiudi dettagli modulo"
            @click="closeDetails"
          >
            <X :size="20" />
          </button>
        </header>

        <div class="modal-body">
          <div class="modal-badges-row">
            <span v-if="selectedPackage.tier === 'community'" class="badge-community">
              <Globe :size="13" /> Community Open Source
            </span>
            <span v-else class="badge-verified">
              <ShieldCheck :size="13" /> Domus Verified
            </span>
            <span class="mp-type-tag">{{ selectedPackage.type.replace('_', ' ').toUpperCase() }}</span>
            <span class="mp-ver-tag">v{{ selectedPackage.version }}</span>
          </div>

          <p class="modal-desc">{{ selectedPackage.description }}</p>

          <div class="modal-meta-grid">
            <div class="meta-item">
              <span class="meta-label">Autore:</span>
              <span class="meta-val">{{ selectedPackage.author }}</span>
            </div>
            <div v-if="selectedPackage.license" class="meta-item">
              <span class="meta-label">Licenza:</span>
              <span class="meta-val">{{ selectedPackage.license }}</span>
            </div>
            <div v-if="selectedPackage.installed_at" class="meta-item">
              <span class="meta-label">Data Installazione:</span>
              <span class="meta-val">{{ new Date(selectedPackage.installed_at).toLocaleDateString() }}</span>
            </div>
          </div>

          <div v-if="selectedPackage.capabilities && selectedPackage.capabilities.length > 0" class="modal-caps-box">
            <span class="caps-label">Permessi Sandbox WASI:</span>
            <div class="mp-caps-row">
              <span v-for="cap in selectedPackage.capabilities" :key="cap" class="mp-cap-tag">
                {{ cap }}
              </span>
            </div>
          </div>

          <div v-if="selectedPackage.repository_url" class="modal-repo-box">
            <a
              :href="selectedPackage.repository_url"
              target="_blank"
              rel="noopener noreferrer"
              class="source-link"
              :aria-label="`Visualizza codice sorgente open source di ${selectedPackage.name}`"
            >
              <span>Visualizza codice sorgente open source</span>
              <ExternalLink :size="14" />
            </a>
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
            :class="['btn-install-toggle modal-action-btn', { installed: selectedPackage.installed }]"
            :disabled="installingPackageId === selectedPackage.id"
            @click="toggleInstall(selectedPackage)"
          >
            <CheckCircle v-if="selectedPackage.installed" :size="16" />
            <Download v-else :size="16" />
            <span>{{ selectedPackage.installed ? 'Disinstalla Modulo' : 'Installa con 1 Click' }}</span>
          </button>
        </footer>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard-view {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

/* Summary Grid */
.summary-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
}

.summary-card {
  padding: 16px 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  border: 1px solid var(--border-card);
}

.summary-card-clickable {
  cursor: pointer;
  transition: all var(--transition-fast);
}

.summary-card-clickable:hover {
  background: var(--bg-surface-hover);
  border-color: var(--primary);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.summary-card-clickable:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

.summary-icon {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.icon-light {
  background: rgba(251, 191, 36, 0.15);
  color: #fbbf24;
}

.icon-switch {
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
}

.icon-sensor {
  background: rgba(16, 185, 129, 0.15);
  color: var(--color-success);
}

.icon-store {
  background: rgba(168, 85, 247, 0.15);
  color: #c084fc;
}

.summary-info {
  display: flex;
  flex-direction: column;
}

.summary-label {
  font-size: 13px;
  color: var(--text-muted);
}

.summary-value {
  font-size: 17px;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.01em;
}

/* Filter Pills */
.filters-row {
  display: flex;
  align-items: center;
  gap: 10px;
  overflow-x: auto;
  padding-bottom: 4px;
  scrollbar-width: none;
}

.filters-row::-webkit-scrollbar {
  display: none;
}

.filter-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: var(--radius-full);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-muted);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: all var(--transition-fast);
}

.filter-pill:hover {
  background: var(--bg-surface-hover);
  color: var(--text-main);
}

.filter-pill.active {
  background: var(--primary);
  color: #ffffff;
  border-color: var(--primary);
  box-shadow: 0 0 12px var(--primary-glow);
}

.filter-pill:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

.pill-icon {
  flex-shrink: 0;
}

/* Entity Grid */
.entity-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

/* Empty State */
.empty-state {
  padding: 48px 24px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  max-width: 500px;
  margin: 30px auto;
  border-radius: var(--radius-xl);
  background: var(--bg-card);
  border: 1px solid var(--border-card);
}

.empty-icon {
  color: var(--primary);
  margin-bottom: 16px;
  opacity: 0.8;
}

.empty-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-main);
}

.empty-desc {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 8px;
  line-height: 1.5;
}

/* Dedicated Marketplace Dashboard View */
.marketplace-dashboard-view {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.marketplace-header {
  padding: 24px;
  border-radius: var(--radius-xl);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  background: var(--bg-card);
  border: 1px solid var(--border-card);
  flex-wrap: wrap;
}

.marketplace-header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.icon-circle {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  background: var(--primary-light);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.icon-circle-small {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
}

.text-primary {
  color: var(--primary);
}

.marketplace-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
  letter-spacing: -0.01em;
}

.marketplace-subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin: 4px 0 0 0;
}

.btn-open-store,
.btn-showcase-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border-radius: var(--radius-md);
  background: var(--primary);
  color: #ffffff;
  font-size: 14px;
  font-weight: 600;
  border: none;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-open-store:hover,
.btn-showcase-link:hover {
  background: var(--primary-hover);
  box-shadow: 0 0 16px var(--primary-glow);
  transform: translateX(2px);
}

.btn-open-store:focus-visible,
.btn-showcase-link:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

/* Marketplace Controls */
.marketplace-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.search-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 260px;
  max-width: 420px;
}

.search-icon {
  position: absolute;
  left: 14px;
  color: var(--text-muted);
  pointer-events: none;
}

.dash-search-input {
  width: 100%;
  padding: 10px 36px 10px 40px;
  border-radius: var(--radius-full);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  font-size: 14px;
  outline: none;
  transition: all var(--transition-fast);
}

.dash-search-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-light);
}

.clear-search-btn {
  position: absolute;
  right: 12px;
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.type-pills {
  display: flex;
  gap: 8px;
  overflow-x: auto;
}

.type-pill {
  padding: 8px 14px;
  border-radius: var(--radius-full);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-muted);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: all var(--transition-fast);
}

.type-pill:hover {
  background: var(--bg-surface-hover);
  color: var(--text-main);
}

.type-pill.active {
  background: var(--primary-light);
  color: var(--primary);
  border-color: var(--primary);
  font-weight: 600;
}

.type-pill:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

/* Marketplace Cards Grid */
.marketplace-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(290px, 1fr));
  gap: 16px;
}

.marketplace-card {
  padding: 20px;
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  border: 1px solid var(--border-card);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 16px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.marketplace-card:hover {
  border-color: var(--border-focus);
  transform: translateY(-3px);
  box-shadow: var(--shadow-md);
}

.marketplace-card:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

.mp-card-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.mp-icon-box {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
}

.mp-icon-box.wasm_plugin {
  background: rgba(168, 85, 247, 0.15);
  color: #c084fc;
}

.mp-icon-box.blueprint {
  background: rgba(251, 191, 36, 0.15);
  color: #fbbf24;
}

.mp-icon-box.widget {
  background: rgba(16, 185, 129, 0.15);
  color: var(--color-success);
}

.modal-icon {
  width: 48px;
  height: 48px;
}

.mp-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
  justify-content: flex-end;
}

.badge-verified {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  padding: 3px 8px;
  border-radius: var(--radius-full);
  background: rgba(16, 185, 129, 0.15);
  color: var(--color-success);
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.badge-community {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  padding: 3px 8px;
  border-radius: var(--radius-full);
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
  border: 1px solid rgba(14, 165, 233, 0.3);
}

.mp-type-tag {
  font-size: 10px;
  font-weight: 700;
  padding: 3px 6px;
  border-radius: var(--radius-sm);
  background: var(--bg-surface);
  color: var(--text-muted);
  border: 1px solid var(--border-subtle);
  letter-spacing: 0.04em;
}

.mp-ver-tag {
  font-size: 11px;
  font-weight: 600;
  font-family: var(--font-mono);
  color: var(--text-dim);
}

.mp-card-body {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.mp-pkg-name {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
  line-height: 1.3;
}

.mp-pkg-author {
  font-size: 12px;
  color: var(--text-dim);
}

.mp-pkg-desc {
  font-size: 13px;
  color: var(--text-muted);
  line-height: 1.5;
  margin: 6px 0 0 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.mp-caps-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 6px;
}

.mp-cap-tag {
  font-size: 11px;
  font-family: var(--font-mono);
  padding: 2px 7px;
  border-radius: var(--radius-sm);
  background: var(--bg-surface);
  color: var(--text-muted);
  border: 1px solid var(--border-subtle);
}

.mp-cap-tag.more {
  color: var(--primary);
}

.mp-card-footer {
  margin-top: auto;
  padding-top: 10px;
  border-top: 1px solid var(--border-subtle);
}

.btn-install-toggle {
  width: 100%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--radius-md);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid var(--border-card);
  background: var(--primary-light);
  color: var(--primary);
  transition: all var(--transition-fast);
}

.btn-install-toggle:hover:not(:disabled) {
  background: var(--primary);
  color: #ffffff;
  border-color: var(--primary);
  box-shadow: 0 0 12px var(--primary-glow);
}

.btn-install-toggle.installed {
  background: var(--color-success-bg);
  color: var(--color-success);
  border-color: rgba(16, 185, 129, 0.3);
}

.btn-install-toggle.installed:hover:not(:disabled) {
  background: var(--color-danger-bg);
  color: var(--color-danger);
  border-color: rgba(244, 63, 94, 0.4);
}

.btn-install-toggle:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

.btn-install-toggle:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Marketplace Showcase Section on Overview */
.dashboard-marketplace-showcase {
  padding: 24px;
  border-radius: var(--radius-xl);
  background: var(--bg-card);
  border: 1px solid var(--border-card);
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.showcase-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.showcase-header-left {
  display: flex;
  align-items: center;
  gap: 14px;
}

.showcase-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.showcase-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.showcase-pill-badge {
  font-size: 11px;
  font-weight: 600;
  padding: 3px 8px;
  border-radius: var(--radius-full);
  background: rgba(168, 85, 247, 0.15);
  color: #c084fc;
  border: 1px solid rgba(168, 85, 247, 0.3);
}

.showcase-subtitle {
  font-size: 13px;
  color: var(--text-muted);
  margin: 3px 0 0 0;
}

.showcase-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}

.showcase-card {
  padding: 16px;
  border-radius: var(--radius-lg);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  display: flex;
  flex-direction: column;
  gap: 10px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.showcase-card:hover {
  border-color: var(--primary);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.showcase-card:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

.sc-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.sc-badges {
  display: flex;
  align-items: center;
  gap: 6px;
}

.sc-pkg-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-main);
  margin: 0;
  line-height: 1.3;
}

.sc-pkg-desc {
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.45;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.sc-card-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: auto;
  padding-top: 8px;
  border-top: 1px solid var(--border-subtle);
}

.sc-author {
  font-size: 11px;
  color: var(--text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.btn-install-mini {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 600;
  border: 1px solid var(--border-card);
  background: var(--primary-light);
  color: var(--primary);
  cursor: pointer;
  flex-shrink: 0;
  transition: all var(--transition-fast);
}

.btn-install-mini:hover:not(:disabled) {
  background: var(--primary);
  color: #ffffff;
}

.btn-install-mini.installed {
  background: var(--color-success-bg);
  color: var(--color-success);
  border-color: rgba(16, 185, 129, 0.3);
}

.btn-install-mini:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

/* Modal Details */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: var(--bg-overlay);
  backdrop-filter: var(--blur-glass);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  z-index: 1000;
}

.modal-card {
  width: 100%;
  max-width: 580px;
  border-radius: var(--radius-xl);
  background: var(--bg-card);
  border: var(--modal-border, 1.5px solid var(--border-card));
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  padding: 20px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-title-box {
  display: flex;
  align-items: center;
  gap: 14px;
}

.modal-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.modal-pkg-id {
  font-size: 12px;
  color: var(--text-muted);
  font-family: var(--font-mono);
}

.btn-close {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 6px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
}

.btn-close:hover {
  background: var(--bg-surface-hover);
  color: var(--text-main);
}

.btn-close:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

.modal-body {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 18px;
  overflow-y: auto;
  max-height: 70vh;
}

.modal-badges-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.modal-desc {
  font-size: 14px;
  color: var(--text-main);
  line-height: 1.6;
  margin: 0;
}

.modal-meta-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
  padding: 14px;
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
}

.meta-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.meta-label {
  font-size: 11px;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.meta-val {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-main);
}

.modal-caps-box {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.caps-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
}

.modal-repo-box {
  margin-top: 4px;
}

.source-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--primary);
  font-size: 13px;
  font-weight: 500;
  text-decoration: none;
}

.source-link:hover {
  text-decoration: underline;
}

.modal-footer {
  padding: 16px 24px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  border-top: 1px solid var(--border-subtle);
  background: var(--bg-surface);
}

.btn-secondary {
  padding: 10px 18px;
  border-radius: var(--radius-md);
  background: transparent;
  border: 1px solid var(--border-card);
  color: var(--text-main);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-secondary:hover {
  background: var(--bg-surface-hover);
}

.modal-action-btn {
  width: auto;
  min-width: 160px;
}

/* Reduced motion preference */
@media (prefers-reduced-motion: reduce) {
  .summary-card-clickable,
  .marketplace-card,
  .showcase-card,
  .btn-open-store,
  .btn-showcase-link,
  .btn-install-toggle,
  .filter-pill {
    transition: none !important;
    transform: none !important;
  }
}
</style>
