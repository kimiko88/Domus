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
  Gauge
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

const packages = ref<StorePackage[]>([])

const filteredPackages = computed(() => {
  return packages.value.filter(p => !activeType.value || p.type === activeType.value)
})

async function toggleInstall(pkg: StorePackage) {
  try {
    if (pkg.installed) {
      await apiClient.post(`/api/v1/store/packages/${pkg.id}/uninstall`, {})
      pkg.installed = false
    } else {
      await apiClient.post(`/api/v1/store/packages/${pkg.id}/install`, {})
      pkg.installed = true
    }
  } catch {}
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
      <div class="header-title-box">
        <div class="icon-circle">
          <ShoppingBag :size="24" class="text-primary" />
        </div>
        <div>
          <h1 class="page-title">{{ t('store.title') }}</h1>
          <p class="page-subtitle">{{ t('store.subtitle') }}</p>
        </div>
      </div>

      <!-- Type Filter Pills -->
      <div class="filter-pills">
        <button :class="['pill-btn', { active: activeType === '' }]" @click="activeType = ''">Tutti</button>
        <button :class="['pill-btn', { active: activeType === 'wasm_plugin' }]" @click="activeType = 'wasm_plugin'">WASM Plugins</button>
        <button :class="['pill-btn', { active: activeType === 'blueprint' }]" @click="activeType = 'blueprint'">Blueprints</button>
        <button :class="['pill-btn', { active: activeType === 'widget' }]" @click="activeType = 'widget'">Widget UI</button>
      </div>
    </header>

    <!-- Empty State -->
    <div v-if="filteredPackages.length === 0" class="empty-state glass-panel">
      <PackageCheck :size="48" class="empty-icon text-muted" />
      <p class="empty-title">{{ t('common.no_data', 'Nessun pacchetto disponibile') }}</p>
      <p class="empty-desc">I pacchetti o blueprint caricati nel sistema appariranno qui.</p>
    </div>

    <!-- Packages Grid -->
    <div v-else class="packages-grid">
      <div
        v-for="pkg in filteredPackages"
        :key="pkg.id"
        class="package-card glass-panel"
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
            :class="['install-btn', { installed: pkg.installed }]"
            @click="toggleInstall(pkg)"
          >
            <CheckCircle v-if="pkg.installed" :size="16" />
            <Download v-else :size="16" />
            <span>{{ pkg.installed ? `${t('store.installed')} (${t('store.uninstall')})` : `${t('store.install')} con 1 Click` }}</span>
          </button>
        </div>
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
  background: rgba(99, 102, 241, 0.15);
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

.filter-pills {
  display: flex;
  gap: 8px;
  background: rgba(255, 255, 255, 0.05);
  padding: 4px;
  border-radius: 10px;
}

.pill-btn {
  padding: 8px 14px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: var(--text-muted);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
}

.pill-btn.active {
  background: var(--primary);
  color: #fff;
}

.packages-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 20px;
}

.package-card {
  padding: 22px;
  border-radius: 16px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 16px;
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
  background: rgba(255, 255, 255, 0.08);
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
}

.pkg-author {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.pkg-desc {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin: 8px 0 0;
  line-height: 1.4;
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
  background: var(--primary);
  color: #fff;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.2s;
}

.install-btn.installed {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  border: 1px solid rgba(16, 185, 129, 0.3);
}
</style>
