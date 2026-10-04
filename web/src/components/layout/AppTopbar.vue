<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from '../../stores/auth'
import { useWsStore } from '../../stores/ws'
import { useTheme } from '../../composables/useTheme'
import { getCurrentLanguage, setLanguage } from '../../locales'
import { Menu, LogOut, User as UserIcon, Sun, Moon } from '@lucide/vue'

interface Props {
  title?: string
}

withDefaults(defineProps<Props>(), {
  title: 'Dashboard'
})

const emit = defineEmits<{
  (e: 'toggle-menu'): void
}>()

const authStore = useAuthStore()
const wsStore = useWsStore()
const { isDark, toggleTheme } = useTheme()

const currentUser = computed(() => authStore.user)
const isWsConnected = computed(() => wsStore.isConnected)
const currentLang = computed(() => getCurrentLanguage())

function toggleLanguage() {
  const next = currentLang.value === 'it' ? 'en' : 'it'
  setLanguage(next)
}

async function handleLogout() {
  await authStore.logout()
}
</script>

<template>
  <header class="app-topbar glass-panel">
    <div class="topbar-left">
      <button
        class="btn-menu-toggle"
        aria-label="Toggle Navigation Menu"
        @click="emit('toggle-menu')"
      >
        <Menu :size="20" />
      </button>
      <h1 class="page-title">{{ title }}</h1>
    </div>

    <div class="topbar-right">
      <!-- Language Quick Switcher -->
      <button
        class="lang-quick-toggle"
        :title="currentLang === 'it' ? 'Passa a Inglese' : 'Switch to Italian'"
        @click="toggleLanguage"
      >
        <span class="lang-flag-pill">{{ currentLang === 'it' ? '🇮🇹 IT' : '🇬🇧 EN' }}</span>
      </button>

      <!-- Theme Toggle -->
      <button
        class="theme-toggle"
        :title="isDark ? 'Passa al tema chiaro' : 'Switch to dark mode'"
        :aria-label="isDark ? 'Attiva tema chiaro' : 'Attiva tema scuro'"
        @click="toggleTheme"
      >
        <Sun v-if="isDark" :size="16" />
        <Moon v-else :size="16" />
      </button>

      <!-- WS Live Status Badge -->
      <div
        :class="['ws-status', { 'is-connected': isWsConnected, 'is-disconnected': !isWsConnected }]"
        :title="isWsConnected ? 'WebSocket Connected' : 'WebSocket Disconnected'"
      >
        <span class="status-indicator" />
        <span class="status-text">{{ isWsConnected ? 'Live' : 'Offline' }}</span>
      </div>

      <!-- User Info & Logout -->
      <div v-if="currentUser" class="user-profile">
        <div class="avatar-box">
          <UserIcon :size="16" />
        </div>
        <div class="user-meta">
          <span class="username">{{ currentUser.username }}</span>
          <span class="role-badge">{{ currentUser.role }}</span>
        </div>
        <button
          class="btn-logout"
          title="Logout"
          aria-label="Logout"
          @click="handleLogout"
        >
          <LogOut :size="16" />
        </button>
      </div>
    </div>
  </header>
</template>

<style scoped>
.app-topbar {
  height: var(--topbar-height);
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-card);
  position: sticky;
  top: 0;
  z-index: 900;
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.btn-menu-toggle {
  display: none;
  background: transparent;
  color: var(--text-main);
  padding: 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  align-items: center;
  justify-content: center;
}

@media (max-width: 768px) {
  .btn-menu-toggle {
    display: flex;
  }
}

.page-title {
  font-size: 20px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--text-main);
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 14px;
}

.lang-quick-toggle {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #cbd5e1;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
}

.lang-quick-toggle:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #f8fafc;
  border-color: rgba(255, 255, 255, 0.2);
}

.lang-flag-pill {
  display: flex;
  align-items: center;
  gap: 4px;
}

.theme-toggle {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #cbd5e1;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.theme-toggle:hover {
  background: rgba(255, 255, 255, 0.12);
  color: #f8fafc;
  border-color: rgba(255, 255, 255, 0.22);
  transform: rotate(12deg);
}

:root[data-theme="light"] .theme-toggle {
  background: rgba(15, 23, 42, 0.06);
  border-color: rgba(15, 23, 42, 0.12);
  color: #475569;
}

:root[data-theme="light"] .theme-toggle:hover {
  background: rgba(15, 23, 42, 0.1);
  color: #0f172a;
}

:root[data-theme="light"] .lang-quick-toggle {
  background: rgba(15, 23, 42, 0.06);
  border-color: rgba(15, 23, 42, 0.12);
  color: #475569;
}

:root[data-theme="light"] .lang-quick-toggle:hover {
  background: rgba(15, 23, 42, 0.1);
  color: #0f172a;
}

/* WebSocket Connection Status */
.ws-status {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 500;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-subtle);
}

.status-indicator {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: var(--text-dim);
}

.ws-status.is-connected {
  background: var(--color-success-bg);
  border-color: rgba(16, 185, 129, 0.3);
  color: var(--color-success);
}

.ws-status.is-connected .status-indicator {
  background-color: var(--color-success);
  box-shadow: 0 0 8px var(--color-success-glow);
}

.ws-status.is-disconnected {
  background: var(--color-danger-bg);
  border-color: rgba(244, 63, 94, 0.3);
  color: var(--color-danger);
}

.ws-status.is-disconnected .status-indicator {
  background-color: var(--color-danger);
}

/* User Profile */
.user-profile {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-left: 12px;
  border-left: 1px solid var(--border-subtle);
}

.avatar-box {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
}

.user-meta {
  display: flex;
  flex-direction: column;
}

@media (max-width: 640px) {
  .user-meta {
    display: none;
  }
}

.username {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-main);
  line-height: 1.2;
}

.role-badge {
  font-size: 10px;
  font-weight: 500;
  text-transform: uppercase;
  color: var(--text-dim);
  letter-spacing: 0.04em;
}

.btn-logout {
  background: transparent;
  color: var(--text-dim);
  padding: 8px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
}

.btn-logout:hover {
  color: var(--color-danger);
  background: rgba(244, 63, 94, 0.1);
}
</style>
