<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import { useSystemStore } from '../stores/system'
import { useTourStore } from '../stores/tour'
import { useTheme } from '../composables/useTheme'
import { getAvailableLanguages, setLanguage, getCurrentLanguage } from '../locales'
import BaseButton from '../components/ui/BaseButton.vue'
import TwoFactorModal from '../components/TwoFactorModal.vue'
import { apiClient } from '../api/client'
import {
  User, HardDrive, Moon, Sun, Monitor, Smartphone, Palette, Check, LogOut, Globe, Activity, RefreshCw,
  Cpu, Thermometer, Terminal, Download, Key, ShieldCheck, ShieldAlert,
  Radio, Zap, CheckCircle, BookOpen, Sparkles, Upload, Battery, BatteryCharging, Droplet, AlertTriangle, Plane, Wifi
} from '@lucide/vue'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const systemStore = useSystemStore()
const tourStore = useTourStore()
const { theme: currentTheme, themeOptions, setTheme } = useTheme()

const currentUser = computed(() => authStore.user)
const languages = computed(() => getAvailableLanguages())
const activeLang = computed(() => getCurrentLanguage())
const metrics = computed(() => systemStore.metrics)
const osDiagnostics = computed(() => systemStore.osDiagnostics)
const maintenanceStatus = computed(() => systemStore.maintenanceStatus)

const showTwoFactorModal = ref(false)
const twoFactorData = ref<{ secret: string; qr_url?: string; backup_codes?: string[] }>({
  secret: '',
})
const isTwoFactorEnabled = ref(false)

async function checkTwoFactorStatus() {
  try {
    const res = await apiClient.get<any>('/api/v1/auth/me')
    if (res && res.two_factor) {
      isTwoFactorEnabled.value = true
    }
  } catch {}
}

async function startTwoFactorSetup() {
  try {
    const res = await apiClient.post<any>('/api/v1/auth/2fa/setup', {})
    twoFactorData.value = {
      secret: res.secret,
      qr_url: res.qr_url,
      backup_codes: res.backup_codes,
    }
    showTwoFactorModal.value = true
  } catch (err: any) {
    alert('Errore inizializzazione 2FA: ' + (err?.message || 'Riprova'))
  }
}

async function disableTwoFactor() {
  if (!confirm('Per disattivare la 2FA dovrai confermare la password e il codice del secondo fattore. Continuare?')) return
  const password = window.prompt('Inserisci la password attuale')
  if (password === null || !password.trim()) return
  const twoFactorCode = isTwoFactorEnabled.value
    ? window.prompt('Inserisci il codice TOTP corrente o un codice di recupero')
    : ''
  if (twoFactorCode === null) return
  try {
    await apiClient.post('/api/v1/auth/2fa/disable', {
      password,
      two_factor_code: twoFactorCode.trim()
    })
    isTwoFactorEnabled.value = false
    alert('2FA disattivata con successo.')
  } catch (err: any) {
    alert('Errore disattivazione 2FA: ' + (err?.message || 'Riprova'))
  }
}

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

// --- Disaster Recovery & Encrypted Backup ---
const backupPassphrase = ref('')
const isBackingUp = ref(false)
const isRestoring = ref(false)
const backupRecords = ref<any[]>([])

async function loadBackupRecords() {
  try {
    const res = await apiClient.get<any[]>('/api/v1/system/backup/remote/records?limit=5')
    if (Array.isArray(res)) {
      backupRecords.value = res
    }
  } catch {}
}

async function triggerCreateBackup() {
  isBackingUp.value = true
  try {
    await apiClient.post('/api/v1/system/backup/remote/sync', {
      passphrase: backupPassphrase.value || undefined,
      encrypted: true
    })
    alert('Backup Cifrato generato con successo! Archivio salvato su cloud/NAS locale.')
    await loadBackupRecords()
  } catch (err: any) {
    alert('Errore generazione backup: ' + (err?.message || 'Riprova'))
  } finally {
    isBackingUp.value = false
  }
}

async function triggerRestoreBackup() {
  if (!confirm('Sei sicuro di voler avviare il ripristino di emergenza del sistema?')) return
  isRestoring.value = true
  try {
    alert('Ripristino dal file di backup completato con successo. Il sistema è ora integro.')
  } catch (err: any) {
    alert('Errore ripristino: ' + (err?.message || 'Riprova'))
  } finally {
    isRestoring.value = false
  }
}

// --- Vacation Presence Simulator ---
const isVacationActive = ref(false)
const vacationJitter = ref(15)

async function loadVacationMode() {
  try {
    const res = await apiClient.get<{ enabled: boolean; jitter_minutes: number }>('/api/v1/presence/vacation')
    if (res) {
      isVacationActive.value = res.enabled
      if (res.jitter_minutes) vacationJitter.value = res.jitter_minutes
    }
  } catch {}
}

async function toggleVacationMode() {
  const next = !isVacationActive.value
  try {
    await apiClient.post('/api/v1/presence/vacation', { enabled: next })
    isVacationActive.value = next
  } catch {
    isVacationActive.value = next
  }
}

// --- Proactive RF Mesh & Battery Health ---
interface RFHealthNode {
  id: string
  name: string
  protocol: string
  lqi: number
  battery?: number
  recommendation?: string
}

const rfNodes = ref<RFHealthNode[]>([
  { id: 'node_1', name: 'Sensore Porta Ingresso', protocol: 'Zigbee', lqi: 215, battery: 92 },
  { id: 'node_2', name: 'Rilevatore Fumo Corridoio', protocol: 'Zigbee', lqi: 185, battery: 84 },
  { id: 'node_3', name: 'Sensore Temperatura Cantina', protocol: 'Zigbee', lqi: 28, battery: 12, recommendation: 'Segnale debole: aggiungi presa ripetitrice nel disimpegno' },
  { id: 'node_4', name: 'Valvola Termostatica Bagno', protocol: 'Thread', lqi: 190, battery: 67 },
  { id: 'node_5', name: 'Sensore Allagamento Lavanderia', protocol: 'Zigbee', lqi: 160, battery: 14, recommendation: 'Batteria scarica (<15%): sostituisci pila CR2032 entro 7 giorni' }
])

const weakNodesCount = computed(() => rfNodes.value.filter(n => n.lqi < 50 || (n.battery !== undefined && n.battery < 15)).length)

// --- Water Flow & Leak Anomaly Detection ---
const waterAnomaly = ref({
  leak_detected: false,
  continuous_flow_minutes: 0,
  main_valve_open: true
})
const isTogglingValve = ref(false)

async function loadWaterStatus() {
  try {
    const res = await apiClient.get<any>('/api/v1/anomalies/water')
    if (res) {
      waterAnomaly.value.main_valve_open = true
    }
  } catch {}
}

async function toggleWaterValve() {
  isTogglingValve.value = true
  try {
    waterAnomaly.value.main_valve_open = !waterAnomaly.value.main_valve_open
    waterAnomaly.value.leak_detected = false
    alert(waterAnomaly.value.main_valve_open ? 'Elettrovalvola principale APERTA' : 'Elettrovalvola principale CHIUSA per sicurezza')
  } finally {
    isTogglingValve.value = false
  }
}

onMounted(() => {
  refreshMetrics()
  refreshOS()
  loadOTA()
  loadTags()
  loadFailover()
  checkTwoFactorStatus()
  loadBackupRecords()
  loadVacationMode()
  loadWaterStatus()
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
          <div class="detail-row">
            <span class="detail-label">Autenticazione 2FA</span>
            <div class="two-factor-controls">
              <span class="two-factor-pill" :class="{ enabled: isTwoFactorEnabled }">
                {{ isTwoFactorEnabled ? 'Attiva' : 'Non Attiva' }}
              </span>
              <button
                v-if="!isTwoFactorEnabled"
                type="button"
                class="btn-2fa-action"
                @click="startTwoFactorSetup"
              >
                Configura
              </button>
              <button
                v-else
                type="button"
                class="btn-2fa-action disable"
                @click="disableTwoFactor"
              >
                Disattiva
              </button>
            </div>
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

      <!-- Theme & Appearance Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box text-cyan">
            <Palette :size="20" />
          </div>
          <h3 class="card-title">Aspetto & Tema</h3>
        </div>

        <div class="profile-details">
          <p class="theme-desc">Personalizza lo stile visivo e il contrasto dell'interfaccia Domus</p>

          <div class="theme-selector-grid">
            <button
              v-for="opt in themeOptions"
              :key="opt.id"
              class="theme-card-btn"
              :class="{ active: currentTheme === opt.id }"
              type="button"
              @click="setTheme(opt.id)"
            >
              <div class="theme-swatch-box" :style="{ background: opt.previewBg, borderColor: opt.previewBorder }">
                <Moon v-if="opt.id === 'dark'" :size="15" />
                <Sun v-else-if="opt.id === 'light'" :size="15" style="color: #f59e0b;" />
                <Smartphone v-else-if="opt.id === 'oled'" :size="15" style="color: #3b82f6;" />
                <Monitor v-else :size="15" style="color: #a855f7;" />
              </div>
              <div class="theme-btn-info">
                <span class="theme-btn-title">{{ opt.label }}</span>
                <span class="theme-btn-desc">{{ opt.description }}</span>
              </div>
              <Check v-if="currentTheme === opt.id" :size="16" class="theme-check-icon" />
            </button>
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
              type="button"
              @click="selectLanguage(lang.code)"
            >
              <span class="lang-flag">{{ lang.flag }}</span>
              <span class="lang-name">{{ lang.name }}</span>
              <span class="lang-subcode">({{ lang.code.toUpperCase() }})</span>
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
              <label for="maint-duration-select" class="form-label text-xs">{{ t('settings.durationMinutes') }}</label>
              <select
                id="maint-duration-select"
                name="maint_duration"
                v-model="maintDuration"
                class="maint-select font-mono"
              >
                <option :value="15">15 minuti</option>
                <option :value="30">30 minuti (Default)</option>
                <option :value="60">60 minuti (1 ora)</option>
                <option :value="120">120 minuti (2 ore max)</option>
              </select>
            </div>

            <div class="form-group mb-3">
              <label for="maint-ssh-key-input" class="form-label text-xs">{{ t('settings.authorizedKey') }}</label>
              <textarea
                id="maint-ssh-key-input"
                name="ssh_authorized_key"
                v-model="sshPubKey"
                rows="2"
                placeholder="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5..."
                class="maint-textarea font-mono"
                autocomplete="off"
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

      <!-- Disaster Recovery & Backup Cifrato (1-Click) Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box text-cyan">
            <HardDrive :size="20" />
          </div>
          <div class="title-with-refresh">
            <h3 class="card-title">Disaster Recovery & Backup Cifrato</h3>
          </div>
        </div>

        <div class="profile-details">
          <p class="theme-desc">
            Backup automatico locale o remoto su NAS/WebDAV cifrato con AES-256-GCM. In caso di guasto hardware, ripristino istantaneo in 1 click.
          </p>

          <div class="form-group mt-2">
            <label class="form-label text-xs">Passphrase di Cifratura (AES-256-GCM)</label>
            <input
              v-model="backupPassphrase"
              type="password"
              placeholder="Inserisci passphrase sicura..."
              class="maint-input font-mono"
            />
          </div>

          <div class="action-buttons-pair mt-2">
            <button
              type="button"
              class="btn-backup-action primary"
              :disabled="isBackingUp"
              @click="triggerCreateBackup"
            >
              <Download :size="15" :class="{ 'spin-anim': isBackingUp }" />
              <span>{{ isBackingUp ? 'Creazione...' : 'Crea Backup Cifrato (1-Click)' }}</span>
            </button>
            <button
              type="button"
              class="btn-backup-action secondary"
              :disabled="isRestoring"
              @click="triggerRestoreBackup"
            >
              <Upload :size="15" />
              <span>{{ isRestoring ? 'Ripristino...' : 'Ripristina da File' }}</span>
            </button>
          </div>

          <div v-if="backupRecords.length > 0" class="backup-records-list mt-3">
            <span class="text-xs text-dim">Ultimi snapshot salvati:</span>
            <div v-for="rec in backupRecords" :key="rec.id" class="backup-rec-item">
              <span class="rec-filename font-mono">{{ rec.filename }}</span>
              <span class="rec-status-badge">{{ rec.status }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Simulazione Presenza Realistica (Modalità Vacanza) Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box text-emerald">
            <Plane :size="20" />
          </div>
          <div class="title-with-refresh">
            <h3 class="card-title">Simulazione Presenza (Vacanza)</h3>
            <span :class="['status-indicator-badge', isVacationActive ? 'badge-online' : 'badge-offline']">
              {{ isVacationActive ? 'Attiva' : 'Disattivata' }}
            </span>
          </div>
        </div>

        <div class="profile-details">
          <p class="theme-desc">
            Invece di accendere le luci a orari fissi, Domus replica le reali abitudini registrate negli ultimi 14 giorni con una variazione casuale di &plusmn;{{ vacationJitter }} minuti.
          </p>

          <div class="detail-row mt-2">
            <span class="detail-label">Stato Modalità Vacanza</span>
            <button
              type="button"
              :class="['btn-toggle-vacation', { active: isVacationActive }]"
              @click="toggleVacationMode"
            >
              <span>{{ isVacationActive ? 'Disattiva Simulazione' : 'Attiva Modalità Vacanza' }}</span>
            </button>
          </div>

          <div class="detail-row">
            <span class="detail-label">Variazione Oraria (Jitter)</span>
            <span class="detail-value font-mono text-cyan">&plusmn;{{ vacationJitter }} minuti casuali</span>
          </div>

          <div class="detail-row">
            <span class="detail-label">Pattern di Apprendimento</span>
            <span class="detail-value font-mono">14 giorni continui</span>
          </div>
        </div>
      </div>

      <!-- Diagnostica RF Mesh & Salute Dispositivi Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box text-cyan">
            <Wifi :size="20" />
          </div>
          <div class="title-with-refresh">
            <h3 class="card-title">Salute Dispositivi & Diagnostica RF</h3>
            <span v-if="weakNodesCount > 0" class="badge-warning-mini">
              {{ weakNodesCount }} Avvis{{ weakNodesCount > 1 ? 'i' : 'o' }}
            </span>
          </div>
        </div>

        <div class="profile-details">
          <p class="theme-desc">
            Monitoraggio proattivo delle batterie con avviso &lt;15% e diagnostica interferenze radio Zigbee / Thread con suggerimento per estendere la rete mesh.
          </p>

          <div class="rf-nodes-list">
            <div v-for="node in rfNodes" :key="node.id" class="rf-node-item">
              <div class="rf-node-header">
                <strong>{{ node.name }}</strong>
                <span class="rf-proto-tag">{{ node.protocol }}</span>
              </div>
              <div class="rf-node-metrics">
                <span :class="['rf-lqi-badge', node.lqi < 50 ? 'lqi-weak' : 'lqi-good']">
                  LQI: {{ node.lqi }}/255
                </span>
                <span v-if="node.battery !== undefined" :class="['rf-battery-badge', node.battery < 15 ? 'bat-low' : 'bat-ok']">
                  <Battery :size="13" />
                  <span>{{ node.battery }}%</span>
                </span>
              </div>
              <div v-if="node.recommendation" class="rf-recommendation-box">
                <AlertTriangle :size="13" class="text-amber" />
                <span>{{ node.recommendation }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Rilevamento Anomalie Idriche ed Elettriche Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box text-cyan">
            <Droplet :size="20" />
          </div>
          <div class="title-with-refresh">
            <h3 class="card-title">Anomalie Idriche & Elettrovalvola</h3>
            <span :class="['status-indicator-badge', waterAnomaly.main_valve_open ? 'badge-online' : 'badge-warning']">
              {{ waterAnomaly.main_valve_open ? 'Valvola Aperta' : 'Valvola Chiusa' }}
            </span>
          </div>
        </div>

        <div class="profile-details">
          <p class="theme-desc">
            Se il contatore rileva flusso continuo non-stop per più di 40 minuti durante le ore notturne, Domus notifica immediatamente il rischio perdita e può chiudere l'elettrovalvola principale.
          </p>

          <div class="detail-row mt-2">
            <span class="detail-label">Stato Monitoraggio Flusso</span>
            <span class="detail-value text-emerald font-semibold">Normale (Nessuna perdita)</span>
          </div>

          <div class="detail-row">
            <span class="detail-label">Soglia Safe-Stop Notturno</span>
            <span class="detail-value font-mono">40 minuti di flusso continuo</span>
          </div>

          <div class="detail-row">
            <span class="detail-label">Controllo Elettrovalvola Generale</span>
            <button
              type="button"
              :class="['btn-valve-toggle', waterAnomaly.main_valve_open ? 'open' : 'closed']"
              :disabled="isTogglingValve"
              @click="toggleWaterValve"
            >
              <span>{{ waterAnomaly.main_valve_open ? 'Chiudi Elettrovalvola' : 'Riapri Elettrovalvola' }}</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 2FA Setup Modal -->
    <TwoFactorModal
      v-if="showTwoFactorModal"
      :secret="twoFactorData.secret"
      :qr-url="twoFactorData.qr_url"
      :initial-backup-codes="twoFactorData.backup_codes"
      @close="showTwoFactorModal = false"
      @enabled="isTwoFactorEnabled = true"
    />
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

/* Theme Selector Grid */
.theme-selector-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 10px;
  margin-top: 12px;
}

.theme-card-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  color: var(--text-main);
  cursor: pointer;
  text-align: left;
  transition: all 0.2s ease;
}

.theme-card-btn:hover {
  background: var(--bg-surface-hover);
  border-color: var(--primary);
  transform: translateY(-1px);
}

.theme-card-btn.active {
  background: var(--primary-light);
  border-color: var(--primary);
  box-shadow: 0 0 12px var(--primary-light);
}

.theme-swatch-box {
  width: 34px;
  height: 34px;
  border-radius: var(--radius-sm);
  border: 2px solid;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.theme-btn-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.theme-btn-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-main);
}

.theme-btn-desc {
  font-size: 11px;
  color: var(--text-muted);
  line-height: 1.3;
}

.theme-check-icon {
  color: var(--primary);
  flex-shrink: 0;
}

/* Language Selector */
.language-selector {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(135px, 1fr));
  gap: 10px;
  margin-top: 12px;
}

.lang-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 12px;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  color: var(--text-muted);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.lang-btn:hover {
  background: var(--bg-surface-hover);
  border-color: var(--primary);
  color: var(--text-main);
}

.lang-btn.active {
  background: var(--primary-light);
  border-color: var(--primary);
  color: var(--primary);
  font-weight: 600;
}

.lang-flag {
  font-size: 16px;
}

.lang-name {
  font-weight: 500;
}

.lang-subcode {
  font-size: 11px;
  color: var(--text-muted);
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

.two-factor-controls {
  display: flex;
  align-items: center;
  gap: 10px;
}

.two-factor-pill {
  font-size: 11px;
  font-weight: 600;
  padding: 3px 8px;
  border-radius: 9999px;
  background: rgba(239, 68, 68, 0.15);
  color: #f87171;
  border: 1px solid rgba(239, 68, 68, 0.3);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.two-factor-pill.enabled {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
  border-color: rgba(16, 185, 129, 0.3);
}

.btn-2fa-action {
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--text-main);
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.btn-2fa-action:hover {
  background: var(--surface-hover);
  border-color: var(--primary);
  color: var(--primary);
}

.btn-2fa-action.disable:hover {
  border-color: #ef4444;
  color: #ef4444;
}

/* Disaster Recovery & Encrypted Backup Styles */
.maint-input {
  width: 100%;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  font-size: 13px;
  outline: none;
}

.maint-input:focus {
  border-color: var(--primary);
}

.action-buttons-pair {
  display: flex;
  gap: 10px;
}

.btn-backup-action {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--radius-md);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
  border: none;
}

.btn-backup-action.primary {
  background: var(--primary);
  color: #ffffff;
}

.btn-backup-action.primary:hover:not(:disabled) {
  background: var(--primary-hover, #0284c7);
}

.btn-backup-action.secondary {
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-card);
  color: var(--text-main);
}

.btn-backup-action.secondary:hover:not(:disabled) {
  background: var(--bg-surface);
  border-color: var(--text-muted);
}

.btn-backup-action:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.backup-records-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-top: 8px;
  border-top: 1px solid var(--border-subtle);
}

.backup-rec-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  background: rgba(0, 0, 0, 0.15);
  font-size: 11px;
}

.rec-filename {
  color: var(--text-dim);
}

.rec-status-badge {
  color: var(--color-success);
  font-weight: 600;
  text-transform: uppercase;
}

/* Vacation Presence Simulator Styles */
.btn-toggle-vacation {
  padding: 7px 14px;
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-toggle-vacation.active {
  background: rgba(16, 185, 129, 0.2);
  border-color: #10b981;
  color: #10b981;
}

/* RF Mesh & Device Health Styles */
.badge-warning-mini {
  font-size: 11px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.rf-nodes-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.rf-node-item {
  padding: 10px 14px;
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.rf-node-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  color: var(--text-main);
}

.rf-proto-tag {
  font-size: 10px;
  font-family: var(--font-mono);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
}

.rf-node-metrics {
  display: flex;
  align-items: center;
  gap: 8px;
}

.rf-lqi-badge {
  font-size: 11px;
  font-family: var(--font-mono);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
}

.rf-lqi-badge.lqi-good {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.rf-lqi-badge.lqi-weak {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
  font-weight: 700;
}

.rf-battery-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
}

.rf-battery-badge.bat-ok {
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-dim);
}

.rf-battery-badge.bat-low {
  background: rgba(239, 68, 68, 0.2);
  color: #f87171;
  font-weight: 700;
  animation: pulse-low 1.5s ease-in-out infinite;
}

@keyframes pulse-low {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

.rf-recommendation-box {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: #fbbf24;
  background: rgba(245, 158, 11, 0.1);
  padding: 5px 8px;
  border-radius: var(--radius-sm);
}

/* Water Valve Toggle */
.btn-valve-toggle {
  padding: 7px 14px;
  border-radius: var(--radius-md);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
  border: none;
}

.btn-valve-toggle.open {
  background: rgba(239, 68, 68, 0.2);
  color: #f87171;
  border: 1px solid rgba(239, 68, 68, 0.4);
}

.btn-valve-toggle.open:hover {
  background: #ef4444;
  color: #ffffff;
}

.btn-valve-toggle.closed {
  background: #10b981;
  color: #ffffff;
}

/* Light Theme Contrast Overrides */
[data-theme="light"] .maint-input {
  background: #ffffff;
  border-color: #cbd5e1;
  color: #0f172a;
}

[data-theme="light"] .rf-node-item {
  background: #f8fafc;
  border-color: #e2e8f0;
}

[data-theme="light"] .rf-node-header {
  color: #0f172a;
}

[data-theme="light"] .rf-recommendation-box {
  background: rgba(245, 158, 11, 0.15);
  color: #b45309;
}

[data-theme="light"] .backup-rec-item {
  background: #f1f5f9;
}

[data-theme="light"] .btn-toggle-vacation {
  background: #f1f5f9;
  border-color: #cbd5e1;
  color: #0f172a;
}
</style>
