<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { getActivePinia } from 'pinia'
import { useAuthStore } from '../../stores/auth'
import {
  LayoutDashboard,
  Grid,
  Cpu,
  Network,
  Zap,
  Clapperboard,
  Settings,
  Home,
  Activity,
  ShieldCheck,
  Users,
  Layers,
  CheckSquare,
  Tablet,
  Sun,
  Sliders,
  Video,
  Clock,
  Music,
  Radio,
  ShoppingBag,
  ShoppingCart,
  BookOpen,
  ScrollText,
  MapPin,
  Code2,
  Wallet,
  UtensilsCrossed,
  Pill,
  Thermometer,
  ChevronDown
} from '@lucide/vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface Props {
  isOpen?: boolean
  currentPath?: string
}

const props = withDefaults(defineProps<Props>(), {
  isOpen: false,
  currentPath: '/'
})

const emit = defineEmits<{
  (e: 'close'): void
}>()

export interface NavSubItem {
  name: string
  labelKey: string
  path: string
  icon: any
}

export interface NavMacroSection {
  id: string
  name: string
  labelKey: string
  icon: any
  items: NavSubItem[]
}

const macroSections: NavMacroSection[] = [
  {
    id: 'overview',
    name: 'Panoramica',
    labelKey: 'nav.sectionOverview',
    icon: LayoutDashboard,
    items: [
      { name: 'Dashboard', labelKey: 'nav.dashboard', path: '/', icon: LayoutDashboard },
      { name: 'Dashboard Tablet', labelKey: 'nav.customDashboards', path: '/custom-dashboards', icon: Tablet },
      { name: 'Mappa Piani', labelKey: 'nav.floorplan', path: '/floorplan', icon: Layers },
      { name: 'Telecamere', labelKey: 'nav.cameras', path: '/cameras', icon: Video }
    ]
  },
  {
    id: 'home',
    name: 'Casa & Famiglia',
    labelKey: 'nav.sectionHome',
    icon: Home,
    items: [
      { name: 'Dispensa & Ricette', labelKey: 'nav.pantry', path: '/pantry', icon: UtensilsCrossed },
      { name: 'Lista Spesa', labelKey: 'nav.shopping', path: '/shopping', icon: ShoppingCart },
      { name: 'Armadietto Farmaci', labelKey: 'nav.medicines', path: '/medicines', icon: Pill },
      { name: 'Faccende', labelKey: 'nav.chores', path: '/chores', icon: CheckSquare },
      { name: 'Bilancio Casa', labelKey: 'nav.finance', path: '/finance', icon: Wallet },
      { name: 'Famiglia', labelKey: 'nav.family', path: '/family', icon: Users }
    ]
  },
  {
    id: 'devices',
    name: 'Dispositivi & Spazi',
    labelKey: 'nav.sectionDevices',
    icon: Cpu,
    items: [
      { name: 'Aree', labelKey: 'nav.areas', path: '/areas', icon: Grid },
      { name: 'Dispositivi', labelKey: 'nav.devices', path: '/devices', icon: Cpu },
      { name: 'Integrazioni', labelKey: 'nav.integrations', path: '/integrations', icon: Network },
      { name: 'Helper Virtuali', labelKey: 'nav.helpers', path: '/helpers', icon: Sliders },
      { name: 'Entità Template', labelKey: 'nav.templates', path: '/templates', icon: Code2 },
      { name: 'Mappa Mesh', labelKey: 'nav.mesh', path: '/mesh', icon: Radio }
    ]
  },
  {
    id: 'automations',
    name: 'Automazioni & Media',
    labelKey: 'nav.sectionAutomations',
    icon: Zap,
    items: [
      { name: 'Automazioni', labelKey: 'nav.automations', path: '/automations', icon: Zap },
      { name: 'Scene', labelKey: 'nav.scenes', path: '/scenes', icon: Clapperboard },
      { name: 'Programmazione', labelKey: 'nav.schedules', path: '/schedules', icon: Clock },
      { name: 'Energia & Solare', labelKey: 'nav.energy', path: '/energy', icon: Sun },
      { name: 'Termostato Predittivo', labelKey: 'nav.predictiveClimate', path: '/climate/predictive', icon: Thermometer },
      { name: 'Audio Multi-room', labelKey: 'nav.media', path: '/media', icon: Music },
      { name: 'Zone & Presenza', labelKey: 'nav.zones', path: '/zones', icon: MapPin }
    ]
  },
  {
    id: 'system',
    name: 'Sistema & Sicurezza',
    labelKey: 'nav.sectionSystem',
    icon: ShieldCheck,
    items: [
      { name: 'Storico', labelKey: 'nav.history', path: '/history', icon: Activity },
      { name: 'Registro Eventi', labelKey: 'nav.logbook', path: '/logbook', icon: ScrollText },
      { name: 'Audit', labelKey: 'nav.audit', path: '/audit', icon: ShieldCheck },
      { name: 'Domus Store', labelKey: 'nav.store', path: '/store', icon: ShoppingBag },
      { name: 'Guida & Manuale', labelKey: 'nav.guide', path: '/guide', icon: BookOpen },
      { name: 'Impostazioni', labelKey: 'nav.settings', path: '/settings', icon: Settings }
    ]
  }
]

const authStore = getActivePinia() ? useAuthStore() : null

const visibleMacroSections = computed(() => {
  if (!authStore) return macroSections
  return macroSections
    .map(sec => ({
      ...sec,
      items: sec.items.filter(item => authStore.canAccessRoute(item.path))
    }))
    .filter(sec => sec.items.length > 0)
})

function isItemActive(path: string): boolean {
  if (path === '/') return props.currentPath === '/'
  return props.currentPath?.startsWith(path) ?? false
}

function isSectionActive(section: NavMacroSection): boolean {
  return section.items.some((i) => isItemActive(i.path))
}

const STORAGE_KEY = 'domus_sidebar_macro_sections'

function loadSavedOpenSections(): Record<string, boolean> {
  try {
    const raw = typeof window !== 'undefined' ? localStorage.getItem(STORAGE_KEY) : null
    if (raw) return JSON.parse(raw)
  } catch {}
  return {
    overview: true,
    home: false,
    devices: false,
    automations: false,
    system: false
  }
}

const openSections = ref<Record<string, boolean>>(loadSavedOpenSections())

function toggleSection(sectionId: string) {
  openSections.value[sectionId] = !openSections.value[sectionId]
  try {
    if (typeof window !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(openSections.value))
    }
  } catch {}
}

// Automatically ensure the section containing the active route is open
watch(
  () => props.currentPath,
  () => {
    for (const sec of macroSections) {
      if (isSectionActive(sec)) {
        openSections.value[sec.id] = true
      }
    }
  },
  { immediate: true }
)

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.isOpen) {
    emit('close')
  }
}

watch(
  () => props.isOpen,
  (open) => {
    if (typeof window === 'undefined') return
    if (open) {
      document.addEventListener('keydown', handleKeydown)
    } else {
      document.removeEventListener('keydown', handleKeydown)
    }
  },
  { immediate: true }
)

onUnmounted(() => {
  if (typeof window !== 'undefined') {
    document.removeEventListener('keydown', handleKeydown)
  }
})
</script>

<template>
  <div>
    <!-- Mobile backdrop -->
    <div
      v-if="isOpen"
      class="sidebar-backdrop"
      aria-hidden="true"
      @click="emit('close')"
    />

    <aside :class="['app-sidebar glass-panel', { 'is-open': isOpen }]">
      <!-- Logo Branding -->
      <div class="sidebar-header">
        <div class="logo-box">
          <Home :size="20" class="logo-icon" />
        </div>
        <div class="brand-text">
          <span class="brand-title">Domus</span>
          <span class="brand-subtitle">Smart Hub</span>
        </div>
      </div>

      <!-- Navigation Links Grouped into Macrosections -->
      <nav class="sidebar-nav">
        <div
          v-for="section in visibleMacroSections"
          :key="section.id"
          class="macro-section"
          :class="{ 'is-expanded': openSections[section.id], 'has-active': isSectionActive(section) }"
        >
          <!-- Macro-section Header Accordion Button -->
          <button
            :id="`btn-section-${section.id}`"
            type="button"
            class="macro-section-header"
            :aria-expanded="openSections[section.id]"
            :aria-controls="`section-${section.id}`"
            @click="toggleSection(section.id)"
          >
            <div class="macro-header-left">
              <component :is="section.icon" :size="18" class="macro-icon" />
              <span class="macro-title">{{ section.labelKey ? t(section.labelKey) : section.name }}</span>
            </div>

            <div class="macro-header-right">
              <span
                v-if="!openSections[section.id] && isSectionActive(section)"
                class="macro-active-dot"
                title="Sezione con elemento attivo"
              />
              <span class="macro-count">{{ section.items.length }}</span>
              <ChevronDown
                :size="15"
                class="macro-chevron"
                :class="{ 'rotate-180': openSections[section.id] }"
              />
            </div>
          </button>

          <!-- Sub-sections List -->
          <div
            v-show="openSections[section.id]"
            :id="`section-${section.id}`"
            class="nav-sub-list"
            role="region"
            :aria-labelledby="`btn-section-${section.id}`"
          >
            <RouterLink
              v-for="item in section.items"
              :key="item.path"
              :to="item.path"
              :class="['nav-item', 'sub-nav-item', { active: isItemActive(item.path) }]"
              :aria-current="isItemActive(item.path) ? 'page' : undefined"
              @click="emit('close')"
            >
              <component :is="item.icon" :size="17" class="nav-icon" />
              <span class="nav-label">{{ item.labelKey ? t(item.labelKey) : item.name }}</span>
            </RouterLink>
          </div>
        </div>
      </nav>

      <!-- Sidebar Footer -->
      <div class="sidebar-footer">
        <div class="hub-status">
          <span class="status-dot pulse" />
          <span class="status-text">Hub Locale v0.8</span>
        </div>
      </div>
    </aside>
  </div>
</template>

<style scoped>
.sidebar-backdrop {
  position: fixed;
  inset: 0;
  background: var(--bg-overlay);
  backdrop-filter: blur(4px);
  z-index: 990;
}

.app-sidebar {
  position: fixed;
  top: 0;
  bottom: 0;
  left: 0;
  width: var(--sidebar-width);
  display: flex;
  flex-direction: column;
  z-index: 1000;
  border-right: 1px solid var(--border-card);
  transition: transform var(--transition-normal);
}

@media (max-width: 768px) {
  .app-sidebar {
    transform: translateX(-100%);
  }

  .app-sidebar.is-open {
    transform: translateX(0);
  }
}

.sidebar-header {
  height: var(--topbar-height);
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 20px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.logo-box {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, var(--primary) 0%, #0284c7 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  box-shadow: 0 0 16px var(--primary-glow);
}

.brand-text {
  display: flex;
  flex-direction: column;
}

.brand-title {
  font-size: 18px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--text-main);
  line-height: 1.2;
}

.brand-subtitle {
  font-size: 11px;
  font-weight: 500;
  color: var(--primary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.sidebar-nav {
  flex: 1;
  padding: 16px 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  overflow-y: auto;
}

/* Custom sleek scrollbar for sidebar nav */
.sidebar-nav::-webkit-scrollbar {
  width: 4px;
}
.sidebar-nav::-webkit-scrollbar-track {
  background: transparent;
}
.sidebar-nav::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
}
.sidebar-nav::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.2);
}

/* Macro Section */
.macro-section {
  display: flex;
  flex-direction: column;
  margin-bottom: 2px;
}

.macro-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 9px 12px;
  min-height: 40px;
  border-radius: var(--radius-md);
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  cursor: pointer;
  transition: all var(--transition-fast);
  user-select: none;
}

.macro-section-header:hover {
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-main);
}

.macro-section-header:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

.macro-section.has-active .macro-section-header {
  color: var(--text-main);
}

.macro-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.macro-icon {
  color: var(--primary);
  flex-shrink: 0;
  opacity: 0.8;
  transition: opacity var(--transition-fast);
}

.macro-section:hover .macro-icon,
.macro-section.has-active .macro-icon {
  opacity: 1;
}

.macro-title {
  line-height: 1.2;
}

.macro-header-right {
  display: flex;
  align-items: center;
  gap: 6px;
}

.macro-active-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--primary);
  box-shadow: 0 0 8px var(--primary-glow);
}

.macro-count {
  font-size: 11px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-dim);
}

.macro-chevron {
  color: var(--text-muted);
  transition: transform var(--transition-fast);
}

.macro-chevron.rotate-180 {
  transform: rotate(180deg);
}

/* Sub-sections tree styling */
.nav-sub-list {
  display: flex;
  flex-direction: column;
  gap: 3px;
  margin-left: 12px;
  padding-left: 10px;
  border-left: 1px solid rgba(255, 255, 255, 0.08);
  margin-top: 3px;
  margin-bottom: 6px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  color: var(--text-muted);
  font-weight: 500;
  font-size: 13.5px;
  transition: all var(--transition-fast);
  border: 1px solid transparent;
  text-decoration: none;
  min-height: 38px;
}

.nav-item:hover {
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-main);
}

.nav-item:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

.nav-item.active {
  background: linear-gradient(90deg, rgba(14, 165, 233, 0.15) 0%, rgba(14, 165, 233, 0.03) 100%);
  color: var(--primary);
  border-color: rgba(14, 165, 233, 0.25);
  box-shadow: inset 3px 0 0 var(--primary);
  font-weight: 600;
}

.nav-icon {
  flex-shrink: 0;
}

.sidebar-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.hub-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-dim);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: var(--color-success);
  box-shadow: 0 0 8px var(--color-success-glow);
}

.status-dot.pulse {
  animation: pulse-subtle 2s infinite;
}

@media (prefers-reduced-motion: reduce) {
  .macro-chevron,
  .status-dot.pulse {
    animation: none;
    transition: none;
  }
}
</style>
