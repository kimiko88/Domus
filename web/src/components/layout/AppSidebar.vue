<script setup lang="ts">
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
  ShieldCheck
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

const navItems = [
  { name: 'Dashboard', labelKey: 'nav.dashboard', path: '/', icon: LayoutDashboard },
  { name: 'Aree', labelKey: 'nav.areas', path: '/areas', icon: Grid },
  { name: 'Dispositivi', labelKey: 'nav.devices', path: '/devices', icon: Cpu },
  { name: 'Integrazioni', labelKey: 'nav.integrations', path: '/integrations', icon: Network },
  { name: 'Automazioni', labelKey: 'nav.automations', path: '/automations', icon: Zap },
  { name: 'Scene', labelKey: 'nav.scenes', path: '/scenes', icon: Clapperboard },
  { name: 'Storico', labelKey: 'nav.history', path: '/history', icon: Activity },
  { name: 'Audit', labelKey: 'nav.audit', path: '/audit', icon: ShieldCheck },
  { name: 'Impostazioni', labelKey: 'nav.settings', path: '/settings', icon: Settings }
]

function isItemActive(path: string): boolean {
  if (path === '/') return props.currentPath === '/'
  return props.currentPath?.startsWith(path) ?? false
}
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

      <!-- Navigation Links -->
      <nav class="sidebar-nav">
        <RouterLink
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          :class="['nav-item', { active: isItemActive(item.path) }]"
          @click="emit('close')"
        >
          <component :is="item.icon" :size="20" class="nav-icon" />
          <span class="nav-label">{{ item.labelKey ? t(item.labelKey) : item.name }}</span>
        </RouterLink>
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
  padding: 20px 14px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 10px 14px;
  border-radius: var(--radius-md);
  color: var(--text-muted);
  font-weight: 500;
  font-size: 14px;
  transition: all var(--transition-fast);
  border: 1px solid transparent;
}

.nav-item:hover {
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-main);
}

.nav-item.active {
  background: linear-gradient(90deg, rgba(14, 165, 233, 0.15) 0%, rgba(14, 165, 233, 0.03) 100%);
  color: var(--primary);
  border-color: rgba(14, 165, 233, 0.25);
  box-shadow: inset 3px 0 0 var(--primary);
}

.nav-icon {
  flex-shrink: 0;
}

.sidebar-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--border-subtle);
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
</style>
