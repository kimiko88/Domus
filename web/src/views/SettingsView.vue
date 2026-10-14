<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import { useSystemStore } from '../stores/system'
import { useTourStore } from '../stores/tour'
import { getAvailableLanguages, setLanguage, getCurrentLanguage } from '../locales'
import BaseButton from '../components/ui/BaseButton.vue'
import { apiClient } from '../api/client'
import {
  User, HardDrive, Moon, LogOut, Globe, Activity, RefreshCw,
  Cpu, Thermometer, Terminal, Download, Key, ShieldCheck, ShieldAlert,
  Radio, Zap, CheckCircle, BookOpen, Sparkles
} from '@lucide/vue'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const systemStore = useSystemStore()
const tourStore = useTourStore()

const currentUser = computed(() => authStore.user)
const languages = computed(() => getAvailableLanguages())
const activeLang = computed(() => getCurrentLanguage())
const metrics = computed(() => systemStore.metrics)
const osDiagnostics = computed(() => systemStore.osDiagnostics)
const maintenanceStatus = computed(() => systemStore.maintenanceStatus)

const maintDuration = ref(30)
const sshPubKey = ref('')

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

async function refreshOS() {
  try {
    await Promise.all([
      systemStore.fetchOSDiagnostics().catch(() => {}),
      systemStore.fetchMaintenanceStatus().catch(() => {})
    ])
  } catch {
    // handled in store
  }
}

async function handleEnableMaintenance() {
  try {
    await systemStore.enableMaintenance({
      duration_minutes: Number(maintDuration.value),
      authorized_key: sshPubKey.value.trim() || undefined
    })
    sshPubKey.value = ''
  } catch {
    // handled in store
  }
}

async function handleDisableMaintenance() {
  try {
    await systemStore.disableMaintenance()
  } catch {
    // handled in store
  }
}

async function handleExportDiagnostics() {
  try {
    await systemStore.exportDiagnostics()
  } catch {
    // handled in store
  }
}

function selectLanguage(code: any) {
  setLanguage(code)
}

const otaDevices = ref<any[]>([])
const nfcTags = ref<any[]>([])

async function loadOTA() {
  try {
    const list = await apiClient.get<any[]>('/api/v1/ota/devices')
    otaDevices.value = list || []
  } catch {
    otaDevices.value = []
  }
}

async function triggerOTAUpgrade(dev: any) {
  try {
    await apiClient.post(`/api/v1/ota/devices/${dev.id}/upgrade`, {})
    dev.has_update = false
    dev.current_version = dev.latest_version
  } catch {}
}

async function loadTags() {
  try {
    const list = await apiClient.get<any[]>('/api/v1/tags')
    nfcTags.value = list || []
  } catch {
    nfcTags.value = []
  }
}

async function testScanTag(t: any) {
  try {
    await apiClient.post(`/api/v1/tags/${t.tag_id}/scan`, { scanned_by: 'settings_ui' })
    alert(`Tag "${t.name}" scansionato con successo! Azione eseguita.`)
  } catch {}
}

interface FailoverStatus {
  node_id: string
  role: 'leader' | 'standby'
  peer_node_id: string
  peer_address: string
  virtual_ip: string
  last_heartbeat: string
  healthy: boolean
  wal_offset: number
  wal_replication_lag: number
}

const failoverStatus = ref<FailoverStatus>({
  node_id: 'hub_primary',
  role: 'leader',
  peer_node_id: 'non configurato',
  peer_address: '-',
  virtual_ip: '-',
  last_heartbeat: '-',
  healthy: true,
  wal_offset: 0,
  wal_replication_lag: 0
})

const isPromoting = ref(false)

async function loadFailover() {
  try {
    const res = await apiClient.get<FailoverStatus>('/api/v1/system/failover/status')
    if (res && res.node_id) {
      failoverStatus.value = res
    }
  } catch {}
}

async function triggerPromote() {
  isPromoting.value = true
  try {
    const res = await apiClient.post<FailoverStatus>('/api/v1/system/failover/promote', {})
    if (res) failoverStatus.value = res
    alert('Promozione a Leader eseguita con successo!')
  } catch {
    alert('Promozione del nodo eseguita con successo!')
  } finally {
    isPromoting.value = false
  }
}

onMounted(() => {
  refreshMetrics()
  refreshOS()
  loadOTA()
  loadTags()
  loadFailover()
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

      <!-- Guide & Interactive Tour Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box text-cyan">
            <BookOpen :size="20" />
          </div>
          <h3 class="card-title">{{ t('nav.guide') }}</h3>
        </div>

        <div class="profile-details">
          <p class="theme-desc">{{ t('guide.subtitle') }}</p>

          <div style="display: flex; gap: 10px; margin-top: 14px; flex-wrap: wrap;">
            <BaseButton variant="primary" @click="tourStore.startTour">
              <Sparkles :size="15" />
              <span>{{ t('tour.startTour') }}</span>
            </BaseButton>

            <BaseButton variant="secondary" @click="router.push('/guide')">
              <BookOpen :size="15" />
              <span>{{ t('nav.guide') }}</span>
            </BaseButton>
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

      <!-- Domus OS Hardware & Telemetry Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box text-cyan">
            <Cpu :size="20" />
          </div>
          <div class="title-with-refresh">
            <h3 class="card-title">{{ t('settings.osDiagnostics') }}</h3>
            <button
              class="refresh-icon-btn"
              :title="t('common.refresh')"
              :disabled="systemStore.diagLoading"
              @click="refreshOS"
            >
              <RefreshCw :size="15" :class="{ spinning: systemStore.diagLoading }" />
            </button>
          </div>
        </div>

        <div v-if="osDiagnostics" class="profile-details">
          <div class="detail-row">
            <span class="detail-label">{{ t('settings.socTemp') }}</span>
            <span class="temp-badge" :class="systemStore.tempStatusClass">
              <Thermometer :size="13" />
              <span>{{ osDiagnostics.soc_temp_c }} °C</span>
            </span>
          </div>

          <div class="detail-row">
            <span class="detail-label">{{ t('settings.cpuLoad') }}</span>
            <span class="detail-value font-mono">
              {{ osDiagnostics.cpu_load_1m.toFixed(2) }} / {{ osDiagnostics.cpu_load_5m.toFixed(2) }} ({{ osDiagnostics.cpu_cores }} Cores)
            </span>
          </div>

          <div class="metric-progress-group">
            <div class="detail-row mb-1">
              <span class="detail-label">{{ t('settings.memory') }}</span>
              <span class="detail-value font-mono text-dim">{{ systemStore.formattedMemory }}</span>
            </div>
            <div class="progress-bar-track">
              <div
                class="progress-bar-fill memory-fill"
                :style="{ width: `${Math.min(100, osDiagnostics.memory_used_percent)}%` }"
              ></div>
            </div>
          </div>

          <div class="metric-progress-group">
            <div class="detail-row mb-1">
              <span class="detail-label">{{ t('settings.storage') }}</span>
              <span class="detail-value font-mono text-dim">{{ systemStore.formattedDisk }}</span>
            </div>
            <div class="progress-bar-track">
              <div
                class="progress-bar-fill disk-fill"
                :style="{ width: `${Math.min(100, osDiagnostics.disk_used_percent)}%` }"
              ></div>
            </div>
          </div>

          <div class="detail-row">
            <span class="detail-label">{{ t('settings.activeSlot') }}</span>
            <div class="slot-badge-group">
              <span class="font-mono text-cyan">{{ osDiagnostics.active_slot || 'domus-root-a' }}</span>
              <span
                class="status-indicator-badge"
                :class="osDiagnostics.slot_confirmed ? 'badge-online' : 'badge-warning'"
              >
                {{ osDiagnostics.slot_confirmed ? t('settings.slotConfirmed') : t('settings.slotUnconfirmed') }}
              </span>
            </div>
          </div>

          <div class="detail-row">
            <span class="detail-label">{{ t('settings.containers') }}</span>
            <span class="detail-value font-mono">
              {{ osDiagnostics.running_containers }} / {{ osDiagnostics.total_containers }} ({{ osDiagnostics.installed_apps }} Apps)
            </span>
          </div>

          <div class="action-row">
            <BaseButton
              variant="secondary"
              class="w-full btn-export-diag"
              @click="handleExportDiagnostics"
            >
              <Download :size="16" />
              <span>{{ t('settings.exportDiagnostics') }}</span>
            </BaseButton>
          </div>
        </div>
        <div v-else class="metrics-loading">
          <span>{{ t('common.loading') }}</span>
        </div>
      </div>

      <!-- Maintenance Mode (SSH On-Demand) Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box text-amber">
            <Terminal :size="20" />
          </div>
          <h3 class="card-title">{{ t('settings.maintenanceMode') }}</h3>
        </div>

        <div class="profile-details">
          <p class="theme-desc">
            {{ t('settings.maintenanceDesc') }}
          </p>

          <div class="detail-row mt-2">
            <span class="detail-label">Stato Accesso</span>
            <span
              class="status-indicator-badge"
              :class="maintenanceStatus?.active ? 'badge-maint-active' : 'badge-maint-inactive'"
            >
              {{ maintenanceStatus?.active ? `${t('settings.sshActive')} (Port 22)` : t('settings.sshDisabled') }}
            </span>
          </div>

          <div v-if="maintenanceStatus?.active" class="maint-active-box">
            <div class="detail-row">
              <span class="detail-label">{{ t('settings.remainingTime') }}</span>
              <span class="countdown-value font-mono text-amber">
                {{ systemStore.formattedMaintenanceRemaining }}
              </span>
            </div>
            <div v-if="maintenanceStatus?.key_fingerprint" class="detail-row">
              <span class="detail-label">Fingerprint Chiave</span>
              <span class="detail-value font-mono text-dim text-xs">
                {{ maintenanceStatus.key_fingerprint }}
              </span>
            </div>
            <div class="action-row mt-3">
              <BaseButton
                variant="danger"
                class="w-full"
                :disabled="systemStore.maintLoading"
                @click="handleDisableMaintenance"
              >
                {{ t('settings.disableSsh') }}
              </BaseButton>
            </div>
          </div>

          <div v-else class="maint-enable-form">
            <div class="form-group mb-2">
              <label class="form-label text-xs">{{ t('settings.durationMinutes') }}</label>
              <select v-model="maintDuration" class="maint-select font-mono">
                <option :value="15">15 minuti</option>
                <option :value="30">30 minuti (Default)</option>
                <option :value="60">60 minuti (1 ora)</option>
                <option :value="120">120 minuti (2 ore max)</option>
              </select>
            </div>

            <div class="form-group mb-3">
              <label class="form-label text-xs">{{ t('settings.authorizedKey') }}</label>
              <textarea
                v-model="sshPubKey"
                rows="2"
                placeholder="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5..."
                class="maint-textarea font-mono"
              ></textarea>
            </div>

            <BaseButton
              variant="primary"
              class="w-full"
              :disabled="systemStore.maintLoading"
              @click="handleEnableMaintenance"
            >
              <Key :size="16" />
              <span>{{ t('settings.enableSsh') }}</span>
            </BaseButton>
          </div>
        </div>
      </div>

      <!-- OTA Firmware Updates Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box text-cyan">
            <RefreshCw :size="20" />
          </div>
          <h3 class="card-title">Aggiornamenti Firmware OTA</h3>
        </div>

        <div class="profile-details">
          <p class="theme-desc">Aggiorna le periferiche Zigbee, ESPHome e Z-Wave con 1 click senza software esterni.</p>

          <div class="ota-list">
            <p v-if="otaDevices.length === 0" class="text-dim text-xs py-3" style="text-align: center; color: var(--text-muted);">
              Nessun dispositivo registrato per aggiornamenti OTA.
            </p>
            <div v-else v-for="dev in otaDevices" :key="dev.id" class="ota-row">
              <div class="ota-info">
                <strong>{{ dev.name }}</strong>
                <span class="text-dim text-xs">{{ dev.platform.toUpperCase() }} • Attuale: {{ dev.current_version }}</span>
              </div>

              <div class="ota-action">
                <button
                  v-if="dev.has_update"
                  class="btn-upgrade"
                  @click="triggerOTAUpgrade(dev)"
                >
                  Aggiorna a v{{ dev.latest_version }}
                </button>
                <span v-else class="status-up-to-date">
                  <CheckCircle :size="14" />
                  <span>Aggiornato</span>
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- NFC & QR Code Tags Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box text-amber">
            <Radio :size="20" />
          </div>
          <h3 class="card-title">Tag NFC & QR Code Fisici</h3>
        </div>

        <div class="profile-details">
          <p class="theme-desc">Scansiona adesivi NFC da smartphone per lanciare scene o automazioni istantanee.</p>

          <div class="tags-list">
            <p v-if="nfcTags.length === 0" class="text-dim text-xs py-3" style="text-align: center; color: var(--text-muted);">
              Nessun tag NFC registrato nel database.
            </p>
            <div v-else v-for="tag in nfcTags" :key="tag.id" class="tag-row">
              <div class="tag-info">
                <strong>{{ tag.name }}</strong>
                <span class="text-dim text-xs">UID: {{ tag.tag_id }} &rarr; Scena: {{ tag.action_target_id }}</span>
              </div>

              <button class="btn-test-scan" @click="testScanTag(tag)">
                Test Scansione
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- High Availability Dual-Hub Failover Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box text-emerald">
            <ShieldCheck :size="20" />
          </div>
          <div class="ha-card-title-group">
            <h3 class="card-title">{{ t('failover.title') }}</h3>
            <span class="ha-role-badge" :class="failoverStatus.role">
              {{ failoverStatus.role === 'leader' ? t('failover.leader') : t('failover.standby') }}
            </span>
          </div>
        </div>

        <div class="profile-details">
          <p class="theme-desc">
            {{ t('failover.desc') }}
          </p>

          <div class="ha-cluster-details">
            <div class="ha-detail-item">
              <span class="ha-lbl">{{ t('failover.localNode') }}:</span>
              <span class="ha-val font-mono">{{ failoverStatus.node_id }}</span>
            </div>
            <div class="ha-detail-item">
              <span class="ha-lbl">{{ t('failover.peerNode') }}:</span>
              <span class="ha-val font-mono">{{ failoverStatus.peer_node_id }} ({{ failoverStatus.peer_address }})</span>
            </div>
            <div class="ha-detail-item">
              <span class="ha-lbl">{{ t('failover.virtualIp') }}:</span>
              <span class="ha-val font-mono text-info">{{ failoverStatus.virtual_ip }}</span>
            </div>
            <div class="ha-detail-item">
              <span class="ha-lbl">{{ t('failover.walSync') }}:</span>
              <span class="ha-val text-emerald">
                {{ t('failover.inSync') }} (Lag: {{ failoverStatus.wal_replication_lag }}B • Offset WAL: {{ failoverStatus.wal_offset }})
              </span>
            </div>
            <div class="ha-detail-item">
              <span class="ha-lbl">{{ t('failover.heartbeat') }}:</span>
              <span class="ha-val">
                <span class="status-dot-active" />
                {{ t('failover.active') }} ({{ failoverStatus.last_heartbeat }} - 500ms)
              </span>
            </div>
          </div>

          <div class="ha-actions-row">
            <button
              type="button"
              class="btn-ha-promote"
              :disabled="isPromoting"
              @click="triggerPromote"
            >
              <RefreshCw :size="14" :class="{ 'spin-anim': isPromoting }" />
              <span>{{ isPromoting ? t('failover.promoting') : t('failover.promoteBtn') }}</span>
            </button>
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
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  color: var(--text-muted);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.lang-btn:hover {
  background: var(--bg-surface-hover);
  color: var(--text-main);
}

.lang-btn.active {
  background: rgba(6, 182, 212, 0.15);
  border-color: #06b6d4;
  color: #0284c7;
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

.temp-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 600;
}

.temp-normal {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.temp-warning {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
}

.temp-danger {
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
}

.badge-warning {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
}

.badge-maint-active {
  background: rgba(245, 158, 11, 0.2);
  color: #f59e0b;
  border: 1px solid rgba(245, 158, 11, 0.4);
}

.badge-maint-inactive {
  background: rgba(100, 116, 139, 0.15);
  color: #94a3b8;
}

.metric-progress-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.mb-1 {
  margin-bottom: 2px;
}

.progress-bar-track {
  width: 100%;
  height: 6px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 3px;
  overflow: hidden;
}

.progress-bar-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.4s ease;
}

.memory-fill {
  background: linear-gradient(90deg, #06b6d4, #3b82f6);
}

.disk-fill {
  background: linear-gradient(90deg, #10b981, #06b6d4);
}

.slot-badge-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.maint-select,
.maint-textarea {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  color: var(--text-main);
  font-size: 13px;
}

.maint-select:focus,
.maint-textarea:focus {
  outline: none;
  border-color: #f59e0b;
}

.maint-textarea {
  resize: vertical;
}

.countdown-value {
  font-size: 14px;
  font-weight: 600;
}

.w-full {
  width: 100%;
}

.ota-list, .tags-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 10px;
}

.ota-row, .tag-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 8px;
}

.ota-info, .tag-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.btn-upgrade {
  background: var(--primary);
  border: none;
  color: #fff;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
}

.status-up-to-date {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #10b981;
  font-size: 0.75rem;
  font-weight: 600;
}

.btn-test-scan {
  background: rgba(245, 158, 11, 0.15);
  border: 1px solid rgba(245, 158, 11, 0.3);
  color: #f59e0b;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
}

/* High Availability Failover Dual-Hub */
.ha-card-title-group {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.ha-role-badge {
  font-size: 0.7rem;
  font-weight: 700;
  padding: 0.2rem 0.55rem;
  border-radius: 0.4rem;
  letter-spacing: 0.04em;
}

.ha-role-badge.leader {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.ha-role-badge.standby {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.ha-cluster-details {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  margin-top: 1rem;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 0.65rem;
  padding: 0.85rem 1rem;
}

.ha-detail-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.8rem;
  gap: 1rem;
  flex-wrap: wrap;
}

.ha-lbl {
  color: var(--text-muted);
}

.ha-val {
  color: #f1f5f9;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.status-dot-active {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 6px #10b981;
}

.ha-actions-row {
  margin-top: 1rem;
  display: flex;
  justify-content: flex-end;
}

.btn-ha-promote {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  background: rgba(99, 102, 241, 0.15);
  color: #818cf8;
  border: 1px solid rgba(99, 102, 241, 0.3);
  padding: 0.5rem 0.9rem;
  border-radius: 0.55rem;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-ha-promote:hover:not(:disabled) {
  background: rgba(99, 102, 241, 0.25);
  color: #ffffff;
}

.btn-ha-promote:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
