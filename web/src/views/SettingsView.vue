<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import { useSystemStore } from '../stores/system'
import { getAvailableLanguages, setLanguage, getCurrentLanguage } from '../locales'
import BaseButton from '../components/ui/BaseButton.vue'
import { User, HardDrive, Moon, LogOut, Globe, Activity, RefreshCw } from '@lucide/vue'

const { t } = useI18n()
const authStore = useAuthStore()
const systemStore = useSystemStore()

const currentUser = computed(() => authStore.user)
const languages = computed(() => getAvailableLanguages())
const activeLang = computed(() => getCurrentLanguage())
const metrics = computed(() => systemStore.metrics)

async function handleLogout() {
  await authStore.logout()
}

async function refreshMetrics() {
  try {
    await systemStore.fetchMetrics()
  } catch {
    // handled in store
  }
}

function selectLanguage(code: any) {
  setLanguage(code)
}

onMounted(() => {
  refreshMetrics()
})
</script>

<template>
  <div class="settings-view">
    <div class="header-row">
      <div>
        <h2 class="section-title">{{ t('settings.title') }}</h2>
        <p class="section-subtitle">{{ t('settings.subtitle') }}</p>
      </div>
    </div>

    <div class="settings-grid">
      <!-- Profile Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box">
            <User :size="20" />
          </div>
          <h3 class="card-title">{{ t('settings.userProfile') }}</h3>
        </div>

        <div v-if="currentUser" class="profile-details">
          <div class="detail-row">
            <span class="detail-label">{{ t('common.name') }}</span>
            <span class="detail-value font-mono">{{ currentUser.username }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Ruolo</span>
            <span class="role-badge">{{ currentUser.role }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">ID Utente</span>
            <span class="detail-value font-mono text-dim">{{ currentUser.id }}</span>
          </div>

          <div class="action-row">
            <BaseButton
              variant="danger"
              class="btn-logout-settings"
              @click="handleLogout"
            >
              <LogOut :size="16" />
              <span>{{ t('nav.logout') }}</span>
            </BaseButton>
          </div>
        </div>
      </div>

      <!-- Language Selector Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box text-cyan">
            <Globe :size="20" />
          </div>
          <h3 class="card-title">{{ t('settings.language') }}</h3>
        </div>

        <div class="profile-details">
          <p class="theme-desc">{{ t('settings.chooseLanguage') }}</p>

          <div class="language-selector">
            <button
              v-for="lang in languages"
              :key="lang.code"
              class="lang-btn"
              :class="{ active: activeLang === lang.code }"
              @click="selectLanguage(lang.code)"
            >
              <span class="lang-flag">{{ lang.flag }}</span>
              <span class="lang-name">{{ lang.name }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Live Operational Metrics Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box text-emerald">
            <Activity :size="20" />
          </div>
          <div class="title-with-refresh">
            <h3 class="card-title">{{ t('settings.systemMetrics') }}</h3>
            <button
              class="refresh-icon-btn"
              :title="t('common.refresh')"
              :disabled="systemStore.loading"
              @click="refreshMetrics"
            >
              <RefreshCw :size="15" :class="{ spinning: systemStore.loading }" />
            </button>
          </div>
        </div>

        <div v-if="metrics" class="profile-details">
          <div class="detail-row">
            <span class="detail-label">{{ t('settings.uptime') }}</span>
            <span class="detail-value font-mono text-cyan">{{ systemStore.formattedUptime }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">{{ t('dashboard.totalEntities') }}</span>
            <span class="detail-value font-mono">{{ metrics.entity_count }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">{{ t('dashboard.onlineDevices') }}</span>
            <span class="detail-value font-mono">{{ metrics.device_count }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">{{ t('settings.wsClients') }}</span>
            <span class="detail-value font-mono">{{ metrics.ws_clients }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">{{ t('settings.mqttBroker') }}</span>
            <span
              class="status-indicator-badge"
              :class="metrics.mqtt_connected ? 'badge-online' : 'badge-offline'"
            >
              {{ metrics.mqtt_connected ? 'Connesso' : 'Disconnesso' }}
            </span>
          </div>
        </div>
        <div v-else class="metrics-loading">
          <span>{{ t('common.loading') }}</span>
        </div>
      </div>

      <!-- System Architecture & Static Info Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box">
            <HardDrive :size="20" />
          </div>
          <h3 class="card-title">{{ t('settings.version') }}</h3>
        </div>

        <div class="profile-details">
          <div class="detail-row">
            <span class="detail-label">Versione Core</span>
            <span class="detail-value font-mono">v1.0.0 (Go modular monolith)</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Versione Web UI</span>
            <span class="detail-value font-mono">v0.8.0 (Vue 3 + Vite PWA)</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Database</span>
            <span class="detail-value font-mono">SQLite WAL mode</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Architettura target</span>
            <span class="detail-value font-mono">ARM64 / Raspberry Pi 4</span>
          </div>
        </div>
      </div>

      <!-- Appearance Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box">
            <Moon :size="20" />
          </div>
          <h3 class="card-title">Aspetto & Tema</h3>
        </div>

        <div class="profile-details">
          <div class="detail-row">
            <span class="detail-label">Tema attivo</span>
            <span class="detail-value">Dark Slate (Glassmorphism)</span>
          </div>
          <p class="theme-desc">
            Interfaccia ottimizzata per consumi ridotti su schermi OLED e contrasto visivo elevato in ambienti domestici.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-view {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-main);
  margin-bottom: 4px;
}

.section-subtitle {
  font-size: 14px;
  color: var(--text-muted);
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 20px;
}

.settings-card {
  padding: 24px;
  border-radius: var(--radius-xl);
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.title-with-refresh {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.refresh-icon-btn {
  background: transparent;
  border: none;
  color: #64748b;
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.refresh-icon-btn:hover {
  color: #f8fafc;
  background: rgba(255, 255, 255, 0.05);
}

.spinning {
  animation: spin 0.8s linear infinite;
}

.icon-box {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.05);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
}

.text-cyan { color: #06b6d4; }
.text-emerald { color: #10b981; }

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-main);
}

.profile-details {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.detail-label {
  font-size: 13px;
  color: var(--text-muted);
}

.detail-value {
  font-size: 13px;
  color: var(--text-main);
}

.font-mono {
  font-family: monospace;
}

.text-dim {
  color: #64748b;
  font-size: 11px;
}

.role-badge {
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: rgba(56, 189, 248, 0.15);
  color: var(--color-primary);
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
}

.status-indicator-badge {
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 500;
}

.badge-online {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.badge-offline {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.action-row {
  margin-top: 8px;
}

.btn-logout-settings {
  width: 100%;
}

.theme-desc {
  font-size: 13px;
  color: var(--text-muted);
  line-height: 1.5;
  margin: 0;
}

.language-selector {
  display: flex;
  gap: 12px;
  margin-top: 8px;
}

.lang-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 16px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-md);
  color: #94a3b8;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.lang-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #f8fafc;
}

.lang-btn.active {
  background: rgba(6, 182, 212, 0.2);
  border-color: #06b6d4;
  color: #06b6d4;
  font-weight: 600;
}

.lang-flag {
  font-size: 18px;
}

.metrics-loading {
  padding: 1rem;
  text-align: center;
  color: #64748b;
  font-size: 0.9rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
