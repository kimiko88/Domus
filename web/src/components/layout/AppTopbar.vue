<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../../stores/auth'
import { useWsStore } from '../../stores/ws'
import { useTheme } from '../../composables/useTheme'
import { useNotificationsStore } from '../../stores/notifications'
import { getCurrentLanguage, setLanguage, supportedLanguages, type SupportedLanguage } from '../../locales'
import { Menu, LogOut, User as UserIcon, Sun, Moon, Sparkles, BookOpen, ChevronDown, Check, Bell, Video } from '@lucide/vue'
import VoiceAssistantModal from '../modals/VoiceAssistantModal.vue'
import ActionableNotificationsModal from '../modals/ActionableNotificationsModal.vue'
import IntercomCallModal from '../modals/IntercomCallModal.vue'

interface Props {
  title?: string
}

withDefaults(defineProps<Props>(), {
  title: 'Dashboard'
})

const emit = defineEmits<{
  (e: 'toggle-menu'): void
}>()

const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()
const wsStore = useWsStore()
const notifStore = useNotificationsStore()
const { isDark, toggleTheme } = useTheme()
const isAssistantOpen = ref(false)
const isNotifDrawerOpen = ref(false)
const isIntercomOpen = ref(false)
const isLangMenuOpen = ref(false)
const langDropdownRef = ref<HTMLElement | null>(null)

const currentUser = computed(() => authStore.user)
const isWsConnected = computed(() => wsStore.isConnected)
const currentLang = computed(() => getCurrentLanguage())
const currentLangMeta = computed(() => supportedLanguages.find(l => l.code === currentLang.value) || supportedLanguages[0])

function goToGuide() {
  router.push('/guide')
}

function chooseLanguage(lang: SupportedLanguage) {
  setLanguage(lang)
  isLangMenuOpen.value = false
}

function handleDocumentClick(e: MouseEvent) {
  if (langDropdownRef.value && !langDropdownRef.value.contains(e.target as Node)) {
    isLangMenuOpen.value = false
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    isLangMenuOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleDocumentClick)
  document.addEventListener('keydown', handleKeydown)
  notifStore.fetchActionable()
})

onUnmounted(() => {
  document.removeEventListener('click', handleDocumentClick)
  document.removeEventListener('keydown', handleKeydown)
})

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
      <!-- Domus Interactive Guide & Manual Button -->
      <button
        class="guide-toggle-btn"
        :title="t('nav.guide')"
        :aria-label="t('nav.guide')"
        @click="goToGuide"
      >
        <BookOpen :size="15" />
        <span class="guide-label">{{ t('nav.guide') }}</span>
      </button>

      <!-- Actionable Notifications Bell -->
      <button
        class="notif-toggle-btn"
        title="Notifiche Azionabili della Casa"
        aria-label="Apri Notifiche Azionabili"
        @click="isNotifDrawerOpen = true"
      >
        <Bell :size="16" />
        <span v-if="notifStore.unreadActionableCount > 0" class="topbar-badge">
          {{ notifStore.unreadActionableCount }}
        </span>
      </button>

      <!-- Domus AI Assistant Button -->
      <button
        class="assistant-toggle-btn"
        title="Apri Assistente AI Domus"
        aria-label="Apri Assistente AI Domus"
        @click="isAssistantOpen = true"
      >
        <Sparkles :size="16" />
        <span class="assistant-label">Assistant</span>
      </button>

      <!-- Video Intercom & Doorbell Button -->
      <button
        class="intercom-toggle-btn"
        title="Citofonia & Campanelli Video (WebRTC Live Intercom)"
        aria-label="Apri Video Citofono"
        @click="isIntercomOpen = true"
      >
        <Video :size="16" />
        <span class="intercom-label">Citofono</span>
      </button>

      <!-- Language Selector Dropdown -->
      <div ref="langDropdownRef" class="lang-dropdown-container">
        <button
          class="lang-dropdown-btn lang-quick-toggle"
          type="button"
          :title="t('settings.language')"
          :aria-label="t('settings.language')"
          aria-haspopup="menu"
          :aria-expanded="isLangMenuOpen"
          @click="isLangMenuOpen = !isLangMenuOpen"
        >
          <span class="lang-flag">{{ currentLangMeta.flag }}</span>
          <span class="lang-code">{{ currentLangMeta.code.toUpperCase() }}</span>
          <ChevronDown :size="13" class="lang-chevron" :class="{ 'rotate-180': isLangMenuOpen }" />
        </button>

        <!-- Dropdown Menu with all 5 languages -->
        <Transition name="fade-slide">
          <div
            v-if="isLangMenuOpen"
            class="lang-menu glass-panel"
            role="menu"
            aria-label="Seleziona Lingua"
          >
            <div class="lang-menu-header">
              <span>{{ t('settings.chooseLanguage') }}</span>
            </div>
            <button
              v-for="lang in supportedLanguages"
              :key="lang.code"
              class="lang-menu-item"
              :class="{ active: currentLang === lang.code }"
              role="menuitem"
              type="button"
              @click="chooseLanguage(lang.code)"
            >
              <span class="item-flag">{{ lang.flag }}</span>
              <span class="item-name">{{ lang.name }}</span>
              <span class="item-code">({{ lang.code.toUpperCase() }})</span>
              <Check v-if="currentLang === lang.code" :size="14" class="item-check" />
            </button>
          </div>
        </Transition>
      </div>

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

  <!-- Voice Assistant Modal -->
  <VoiceAssistantModal :is-open="isAssistantOpen" @close="isAssistantOpen = false" />

  <!-- Actionable Notifications Drawer Modal -->
  <ActionableNotificationsModal :is-open="isNotifDrawerOpen" @close="isNotifDrawerOpen = false" />

  <!-- Video Intercom & Doorbell Modal -->
  <IntercomCallModal :is-open="isIntercomOpen" @close="isIntercomOpen = false" />
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

.guide-toggle-btn {
  background: rgba(14, 165, 233, 0.12);
  border: 1px solid rgba(56, 189, 248, 0.3);
  color: #38bdf8;
  padding: 5px 12px;
  border-radius: var(--radius-full);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 6px;
}

.guide-toggle-btn:hover {
  background: rgba(14, 165, 233, 0.25);
  border-color: #38bdf8;
  color: #f0f9ff;
  transform: translateY(-1px);
  box-shadow: 0 0 12px rgba(14, 165, 233, 0.3);
}

@media (max-width: 640px) {
  .guide-label {
    display: none;
  }
}

.notif-toggle-btn {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 9px;
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-color);
  color: var(--text-main);
  cursor: pointer;
  transition: all 0.2s ease;
}

.notif-toggle-btn:hover {
  background: var(--primary-light, rgba(14, 165, 233, 0.15));
  border-color: var(--primary);
  color: var(--primary);
}

.topbar-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  background: var(--color-danger, #f43f5e);
  color: #ffffff;
  font-size: 0.65rem;
  font-weight: 800;
  padding: 1px 5px;
  border-radius: 9px;
  box-shadow: 0 0 6px rgba(244, 63, 94, 0.5);
}

.assistant-toggle-btn {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.2), rgba(139, 92, 246, 0.2));
  border: 1px solid rgba(139, 92, 246, 0.4);
  color: #c084fc;
  padding: 5px 12px;
  border-radius: var(--radius-full);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 6px;
}

.assistant-toggle-btn:hover {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.35), rgba(139, 92, 246, 0.35));
  border-color: #a855f7;
  color: #e9d5ff;
  transform: translateY(-1px);
  box-shadow: 0 0 12px rgba(168, 85, 247, 0.3);
}

.intercom-toggle-btn {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.15), rgba(14, 165, 233, 0.15));
  border: 1px solid rgba(16, 185, 129, 0.35);
  color: #34d399;
  padding: 5px 12px;
  border-radius: var(--radius-full);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 6px;
}

.intercom-toggle-btn:hover {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.3), rgba(14, 165, 233, 0.3));
  border-color: #10b981;
  color: #a7f3d0;
  transform: translateY(-1px);
  box-shadow: 0 0 12px rgba(16, 185, 129, 0.3);
}

.lang-dropdown-container {
  position: relative;
}

.lang-dropdown-btn {
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  padding: 5px 10px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 5px;
}

.lang-dropdown-btn:hover {
  background: var(--bg-surface-hover);
  border-color: var(--primary);
  color: var(--primary);
}

.lang-chevron {
  transition: transform 0.2s ease;
  color: var(--text-muted);
}

.rotate-180 {
  transform: rotate(180deg);
}

.lang-menu {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  min-width: 175px;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: 6px;
  z-index: 1000;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.lang-menu-header {
  padding: 4px 8px 6px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-dim);
  border-bottom: 1px solid var(--border-subtle);
  margin-bottom: 4px;
}

.lang-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-main);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
  width: 100%;
  text-align: left;
}

.lang-menu-item:hover {
  background: var(--bg-surface-hover);
  color: var(--primary);
}

.lang-menu-item.active {
  background: var(--primary-light);
  color: var(--primary);
  font-weight: 600;
}

.item-flag {
  font-size: 16px;
}

.item-name {
  flex: 1;
}

.item-code {
  font-size: 11px;
  color: var(--text-muted);
}

.item-check {
  color: var(--primary);
  margin-left: auto;
}

.theme-toggle {
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-main);
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
  background: var(--bg-surface-hover);
  border-color: var(--primary);
  color: var(--primary);
  transform: rotate(12deg);
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

[data-theme="light"] .intercom-toggle-btn {
  background: rgba(16, 185, 129, 0.12);
  border-color: rgba(16, 185, 129, 0.4);
  color: #047857;
}

[data-theme="light"] .intercom-toggle-btn:hover {
  background: rgba(16, 185, 129, 0.2);
  color: #065f46;
}
</style>
