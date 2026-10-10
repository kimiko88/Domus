<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
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

const packages = ref<StorePackage[]>([
  {
    id: 'pkg_aqara_wasm',
    package_id: 'community.aqara.cube',
    name: 'Aqara Magic Cube Controller',
    type: 'wasm_plugin',
    version: '1.2.0',
    author: 'DomusCommunity',
    description: 'Plugin WASM in sandbox sicura per interpretare rotazioni, tap su tavolo e scuotimento del cubo Aqara Zigbee',
    installed: true,
    icon: 'box'
  },
  {
    id: 'pkg_bp_circadian',
    package_id: 'blueprints.lighting.circadian',
    name: 'Illuminazione Circadiana Dinamica',
    type: 'blueprint',
    version: '2.0.1',
    author: 'SmartHomeItaly',
    description: 'Regola temperatura colore (Kelvin) e luminosità dei faretti in base all\'elevazione solare reale astronomica',
    installed: false,
    icon: 'sun'
  },
  {
    id: 'pkg_widget_gauges',
    package_id: 'widgets.energy.flow_circle',
    name: 'Widget Ghiera Consumi Circolare',
    type: 'widget',
    version: '1.0.4',
    author: 'DomusLabs',
    description: 'Tessera interattiva con misuratore tachimetrico rotante per i carichi elettrici istantanei e distacco carichi',
    installed: false,
    icon: 'gauge'
  }
])

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
    if (list && list.length > 0) packages.value = list
  } catch {}
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
          <h1 class="page-title">Domus Store & Marketplace Estensioni</h1>
          <p class="page-subtitle">Installa plugin WASM sandboxed, Blueprint di automazioni e Widget per dashboard con 1 click</p>
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

    <!-- Packages Grid -->
    <div class="packages-grid">
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
            <span>{{ pkg.installed ? 'Installato (Rimuovi)' : 'Installa con 1 Click' }}</span>
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
