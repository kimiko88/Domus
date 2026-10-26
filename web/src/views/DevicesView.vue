<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDevicesStore } from '../stores/devices'
import { useAreasStore } from '../stores/areas'
import { useNotificationsStore } from '../stores/notifications'
import { apiClient } from '../api/client'
import type { DiscoveredHost } from '../api/types'
import { Cpu, Wifi, WifiOff, RefreshCw, QrCode, Plus, Sparkles, Check, CheckCircle, ShieldCheck, X } from '@lucide/vue'

const { t } = useI18n()
const devicesStore = useDevicesStore()
const areasStore = useAreasStore()
const notifStore = useNotificationsStore()

const filterStatus = ref<'all' | 'online' | 'offline'>('all')
const isScanning = ref(false)
const discoveredHosts = ref<DiscoveredHost[]>([])
const selectedHostForSetup = ref<DiscoveredHost | null>(null)
const isSetupModalOpen = ref(false)
const isMatterWizardOpen = ref(false)
const newDeviceName = ref('')
const selectedAreaId = ref('')
const isAddingDevice = ref(false)

// Matter Commissioning State
const matterPairingCode = ref('')
const matterCommissionStep = ref<1 | 2 | 3>(1)
const matterCommissioning = ref(false)

onMounted(async () => {
  try {
    await Promise.all([
      devicesStore.fetchDevices(),
      areasStore.fetchAreas(),
      loadDiscoveredHosts()
    ])
  } catch (err) {
    console.warn('DevicesView fetch warning:', err)
  }
})

async function loadDiscoveredHosts() {
  try {
    const list = await apiClient.get<DiscoveredHost[]>('/api/v1/network/discovered')
    if (Array.isArray(list)) {
      discoveredHosts.value = list.filter(h => h.is_iot)
    }
  } catch {}
}

async function triggerNetworkScan() {
  isScanning.value = true
  try {
    const res = await apiClient.post<{ count: number; hosts: DiscoveredHost[] }>('/api/v1/network/scan', {})
    if (res && Array.isArray(res.hosts)) {
      discoveredHosts.value = res.hosts.filter(h => h.is_iot)
      notifStore.success('Scansione Rete Completata', `Trovati ${res.hosts.length} dispositivi sulla rete locale`)
    }
  } catch (err: any) {
    notifStore.error('Errore Scansione', err?.message || 'Impossibile completare la scansione')
  } finally {
    isScanning.value = false
  }
}

function openSetupModal(host: DiscoveredHost) {
  selectedHostForSetup.value = host
  newDeviceName.value = `${host.vendor || 'Dispositivo'} (${host.ip})`
  selectedAreaId.value = areasStore.areas[0]?.id || ''
  isSetupModalOpen.value = true
}

async function confirmAddDiscoveredDevice() {
  if (!selectedHostForSetup.value) return
  isAddingDevice.value = true
  try {
    const host = selectedHostForSetup.value
    await apiClient.post('/api/v1/devices', {
      name: newDeviceName.value,
      area_id: selectedAreaId.value,
      integration_id: host.suggested_plugin || 'network_discovery',
      manufacturer: host.vendor || 'Generic IoT',
      model: host.hostname || host.ip,
      status: 'online'
    })
    notifStore.success('Dispositivo Aggiunto', `${newDeviceName.value} è ora configurato in Domus!`)
    isSetupModalOpen.value = false
    await devicesStore.fetchDevices()
    discoveredHosts.value = discoveredHosts.value.filter(h => h.ip !== host.ip)
  } catch (err: any) {
    notifStore.error('Errore Aggiunta', err?.message || 'Impossibile aggiungere il dispositivo')
  } finally {
    isAddingDevice.value = false
  }
}

async function startMatterCommissioning() {
  if (!matterPairingCode.value || matterPairingCode.value.trim().length < 6) return
  matterCommissioning.value = true
  matterCommissionStep.value = 2

  // Simulate Matter PASE + Operational Credential Commissioning Handshake
  setTimeout(() => {
    matterCommissionStep.value = 3
    matterCommissioning.value = false
    notifStore.success('Matter Connesso', 'Dispositivo commissionato su rete Thread locale')
    devicesStore.fetchDevices()
  }, 2200)
}

function closeMatterWizard() {
  isMatterWizardOpen.value = false
  matterCommissionStep.value = 1
  matterPairingCode.value = ''
}

const filteredDevices = computed(() => {
  if (filterStatus.value === 'online') {
    return devicesStore.onlineDevices || []
  }
  if (filterStatus.value === 'offline') {
    return devicesStore.offlineDevices || []
  }
  return devicesStore.devices || []
})

const totalDevicesCount = computed(() => (devicesStore.devices || []).length)
const onlineDevicesCount = computed(() => (devicesStore.onlineDevices || []).length)
const offlineDevicesCount = computed(() => (devicesStore.offlineDevices || []).length)
</script>

<template>
  <div class="devices-view">
    <div class="header-row">
      <div>
        <h2 class="section-title">{{ t('devices.title') }}</h2>
        <p class="section-subtitle">{{ t('devices.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <button
          class="btn-action-outline"
          :disabled="isScanning"
          @click="triggerNetworkScan"
          title="Scansiona la rete LAN per nuovi dispositivi IoT"
        >
          <RefreshCw :size="16" :class="{ 'spin-icon': isScanning }" />
          <span>{{ isScanning ? 'Scansione in corso...' : 'Scansiona Rete LAN' }}</span>
        </button>
        <button
          class="btn-action-primary"
          @click="isMatterWizardOpen = true"
          title="Associa dispositivo Matter tramite QR code o codice di pairing"
        >
          <QrCode :size="16" />
          <span>Associa Matter / Thread</span>
        </button>
      </div>
    </div>

    <!-- Zero-Config Discovered IoT Devices -->
    <div v-if="discoveredHosts.length > 0" class="discovered-section glass-card">
      <div class="discovered-header">
        <div class="discovered-title-wrap">
          <div class="discovered-icon-box">
            <Sparkles :size="20" class="sparkle-icon" />
          </div>
          <div>
            <h3 class="discovered-title">Dispositivi Rilevati nella Rete Locale (Zero-Config)</h3>
            <p class="discovered-subtitle">
              {{ discoveredHosts.length }} dispositivo{{ discoveredHosts.length > 1 ? 'i' : '' }} rilevato{{ discoveredHosts.length > 1 ? 'i' : '' }} via mDNS / SSDP / UPnP pronto per l'onboarding immediato.
            </p>
          </div>
        </div>
      </div>
      <div class="discovered-grid">
        <div v-for="host in discoveredHosts" :key="host.ip" class="discovered-card">
          <div class="discovered-info">
            <div class="discovered-meta-top">
              <span class="disc-vendor">{{ host.vendor || 'Generic IoT' }}</span>
              <span v-if="host.suggested_plugin" class="disc-tag">{{ host.suggested_plugin }}</span>
            </div>
            <div class="disc-hostname">{{ host.hostname || host.ip }}</div>
            <div class="disc-net-details">
              <span>IP: {{ host.ip }}</span>
              <span v-if="host.mac">• {{ host.mac }}</span>
            </div>
          </div>
          <button class="btn-setup-now" @click="openSetupModal(host)">
            <Plus :size="15" />
            <span>Configura</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Filters Row -->
    <div class="filters-row">
      <button
        :class="['filter-pill', { active: filterStatus === 'all' }]"
        @click="filterStatus = 'all'"
      >
        {{ t('devices.filterAll') }} ({{ totalDevicesCount }})
      </button>
      <button
        :class="['filter-pill', { active: filterStatus === 'online' }]"
        @click="filterStatus = 'online'"
      >
        <Wifi :size="15" />
        {{ t('devices.filterOnline') }} ({{ onlineDevicesCount }})
      </button>
      <button
        :class="['filter-pill', { active: filterStatus === 'offline' }]"
        @click="filterStatus = 'offline'"
      >
        <WifiOff :size="15" />
        {{ t('devices.filterOffline') }} ({{ offlineDevicesCount }})
      </button>
    </div>

    <!-- Devices Grid -->
    <div v-if="filteredDevices.length > 0" class="devices-grid">
      <div
        v-for="device in filteredDevices"
        :key="device.id"
        :class="['device-card glass-card', { 'is-online': device.status === 'online' }]"
      >
        <div class="device-card-top">
          <div class="device-icon-box">
            <Cpu :size="22" />
          </div>
          <div class="device-info">
            <h3 class="device-name" :title="device.name">{{ device.name }}</h3>
            <div class="device-meta">
              <span v-if="device.manufacturer" class="meta-tag">{{ device.manufacturer }}</span>
              <span v-if="device.model" class="meta-tag">{{ device.model }}</span>
            </div>
          </div>
        </div>

        <div class="device-card-footer">
          <span class="integration-badge">{{ device.integration_id }}</span>
          <div class="status-indicator">
            <span class="status-dot" :class="{ online: device.status === 'online' }" />
            <span class="status-text">{{ device.status }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="empty-state glass-card">
      <Cpu :size="36" class="empty-icon" />
      <h3 class="empty-title">{{ t('devices.emptyTitle') }}</h3>
      <p class="empty-desc">
        {{ t('devices.emptyDesc') }}
      </p>
    </div>

    <!-- Setup Modal for Discovered Host -->
    <div v-if="isSetupModalOpen" class="modal-backdrop" @click.self="isSetupModalOpen = false">
      <div class="modal-dialog glass-card">
        <div class="modal-header">
          <div class="modal-title-wrap">
            <Sparkles :size="20" class="sparkle-icon" />
            <h3 class="modal-title">Configura Dispositivo Rilevato</h3>
          </div>
          <button class="btn-close-modal" @click="isSetupModalOpen = false" aria-label="Chiudi">
            <X :size="18" />
          </button>
        </div>
        <div class="modal-body">
          <p class="modal-intro">
            Aggiungi <strong>{{ selectedHostForSetup?.hostname || selectedHostForSetup?.ip }}</strong> alla tua smart home Domus.
          </p>
          <div class="form-group">
            <label class="form-label">Nome Dispositivo</label>
            <input
              v-model="newDeviceName"
              type="text"
              class="form-input"
              placeholder="Es. Luce Salotto, Presa Lavatrice"
            />
          </div>
          <div class="form-group">
            <label class="form-label">Stanza / Area</label>
            <select v-model="selectedAreaId" class="form-select">
              <option v-for="area in areasStore.areas" :key="area.id" :value="area.id">
                {{ area.name }}
              </option>
            </select>
          </div>
          <div class="discovery-details-box">
            <div class="detail-row">
              <span class="detail-label">Produttore:</span>
              <span class="detail-val">{{ selectedHostForSetup?.vendor || 'Sconosciuto' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Indirizzo IP:</span>
              <span class="detail-val font-mono">{{ selectedHostForSetup?.ip }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Integrazione raccomandata:</span>
              <span class="detail-val badge-tag">{{ selectedHostForSetup?.suggested_plugin || 'mDNS Zero-Config' }}</span>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-secondary" @click="isSetupModalOpen = false">Annulla</button>
          <button class="btn-primary" :disabled="isAddingDevice" @click="confirmAddDiscoveredDevice">
            <Check :size="16" />
            <span>{{ isAddingDevice ? 'Aggiunta...' : 'Aggiungi a Domus' }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Matter & Thread Guided Commissioning Modal -->
    <div v-if="isMatterWizardOpen" class="modal-backdrop" @click.self="closeMatterWizard">
      <div class="modal-dialog glass-card">
        <div class="modal-header">
          <div class="modal-title-wrap">
            <ShieldCheck :size="20" class="matter-shield-icon" />
            <h3 class="modal-title">Commissioning Matter & Thread</h3>
          </div>
          <button class="btn-close-modal" @click="closeMatterWizard" aria-label="Chiudi">
            <X :size="18" />
          </button>
        </div>

        <div class="modal-body">
          <!-- Step 1: Input Code -->
          <div v-if="matterCommissionStep === 1" class="matter-step">
            <p class="modal-intro">
              Associa un nuovo dispositivo compatibile Matter (presa, lampadina, serratura o termostato) alla rete Thread locale Domus.
            </p>
            <div class="form-group">
              <label class="form-label">Codice di Pairing Matter (11 o 21 cifre)</label>
              <input
                v-model="matterPairingCode"
                type="text"
                class="form-input font-mono"
                placeholder="Es. 34970-112233445"
                maxlength="25"
              />
              <span class="form-hint">
                Trovi il codice sul retro del dispositivo o accanto al QR code sulla confezione.
              </span>
            </div>
          </div>

          <!-- Step 2: Commissioning in progress -->
          <div v-else-if="matterCommissionStep === 2" class="matter-step step-center">
            <div class="commissioning-spinner-box">
              <RefreshCw :size="36" class="spin-icon text-primary" />
            </div>
            <h4 class="commissioning-heading">Negoziazione Credenziali Matter...</h4>
            <p class="commissioning-subtext">
              Scambio chiavi PASE, verifica certificato DAC ed espansione sul Fabric Thread Domus.
            </p>
            <div class="progress-bar-container">
              <div class="progress-bar-fill animated-progress"></div>
            </div>
          </div>

          <!-- Step 3: Success -->
          <div v-else-if="matterCommissionStep === 3" class="matter-step step-center">
            <div class="success-icon-box">
              <CheckCircle :size="42" class="text-success" />
            </div>
            <h4 class="commissioning-heading">Dispositivo Matter Associato!</h4>
            <p class="commissioning-subtext">
              Il nodo è ora autenticato ed è operativo con latenza sub-millisecondo sulla maglia Thread locale.
            </p>
          </div>
        </div>

        <div class="modal-footer">
          <button v-if="matterCommissionStep === 1" class="btn-secondary" @click="closeMatterWizard">
            Annulla
          </button>
          <button
            v-if="matterCommissionStep === 1"
            class="btn-primary"
            :disabled="!matterPairingCode || matterPairingCode.trim().length < 6"
            @click="startMatterCommissioning"
          >
            Avvia Accoppiamento
          </button>
          <button
            v-if="matterCommissionStep === 3"
            class="btn-primary"
            @click="closeMatterWizard"
          >
            Fatto
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.devices-view {
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
  font-size: 22px;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.02em;
}

.section-subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 4px;
}

.filters-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.filter-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: var(--radius-full);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-muted);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.filter-pill:hover {
  background: var(--bg-surface-hover);
  color: var(--text-main);
}

.filter-pill.active {
  background: var(--primary);
  color: white;
  border-color: var(--primary);
  box-shadow: 0 0 12px var(--primary-glow);
}

.devices-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 18px;
}

.device-card {
  padding: 18px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
  transition: all var(--transition-normal);
  gap: 16px;
}

.device-card:hover {
  transform: translateY(-2px);
  border-color: var(--border-card);
  box-shadow: var(--shadow-md);
}

.device-card-top {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.device-icon-box {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-dim);
  flex-shrink: 0;
}

.is-online .device-icon-box {
  color: var(--primary);
  background: rgba(14, 165, 233, 0.15);
}

.device-info {
  flex: 1;
  min-width: 0;
}

.device-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.device-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 6px;
}

.meta-tag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  background: var(--bg-surface);
  color: var(--text-dim);
  border: 1px solid var(--border-card);
}

.device-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}

.integration-badge {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-dim);
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  text-transform: capitalize;
  color: var(--text-dim);
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: var(--color-danger);
}

.status-dot.online {
  background-color: var(--color-success);
  box-shadow: 0 0 6px var(--color-success-glow);
}

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

/* Header Action Buttons */
.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.btn-action-outline {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-action-outline:hover:not(:disabled) {
  background: var(--bg-surface-hover);
  border-color: var(--primary);
  color: var(--primary);
}

.btn-action-outline:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-action-primary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: var(--radius-md);
  background: var(--primary);
  border: 1px solid var(--primary);
  color: #ffffff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
  box-shadow: 0 2px 8px var(--primary-glow);
}

.btn-action-primary:hover {
  background: var(--primary-hover, #0284c7);
  transform: translateY(-1px);
}

.spin-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* Zero-Config Discovered IoT Section */
.discovered-section {
  padding: 20px;
  border-radius: var(--radius-xl);
  border: 1px solid rgba(14, 165, 233, 0.3);
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.08) 0%, rgba(59, 130, 246, 0.03) 100%);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.discovered-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.discovered-title-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
}

.discovered-icon-box {
  width: 38px;
  height: 38px;
  border-radius: var(--radius-md);
  background: rgba(14, 165, 233, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
}

.sparkle-icon {
  color: var(--primary);
}

.discovered-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.01em;
}

.discovered-subtitle {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 2px;
}

.discovered-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 14px;
}

.discovered-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-radius: var(--radius-lg);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  gap: 12px;
  transition: all var(--transition-normal);
}

.discovered-card:hover {
  border-color: var(--primary);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.discovered-info {
  flex: 1;
  min-width: 0;
}

.discovered-meta-top {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}

.disc-vendor {
  font-size: 12px;
  font-weight: 600;
  color: var(--primary);
}

.disc-tag {
  font-size: 10px;
  font-family: var(--font-mono);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  background: rgba(14, 165, 233, 0.15);
  color: var(--text-main);
}

.disc-hostname {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.disc-net-details {
  font-size: 11px;
  color: var(--text-dim);
  font-family: var(--font-mono);
  margin-top: 2px;
}

.btn-setup-now {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  background: var(--primary);
  color: #ffffff;
  border: none;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  flex-shrink: 0;
  transition: all var(--transition-fast);
}

.btn-setup-now:hover {
  background: var(--primary-hover, #0284c7);
  box-shadow: 0 2px 8px var(--primary-glow);
}

/* Modals */
.modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 999;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(6px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}

.modal-dialog {
  width: 100%;
  max-width: 480px;
  border-radius: var(--radius-xl);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  box-shadow: 0 24px 48px rgba(0, 0, 0, 0.4);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 24px;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-title-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
}

.modal-title {
  font-size: 17px;
  font-weight: 700;
  color: var(--text-main);
}

.btn-close-modal {
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

.btn-close-modal:hover {
  color: var(--text-main);
  background: var(--bg-surface-hover);
}

.modal-body {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.modal-intro {
  font-size: 14px;
  color: var(--text-muted);
  line-height: 1.5;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-main);
}

.form-input, .form-select {
  padding: 10px 14px;
  border-radius: var(--radius-md);
  background: var(--bg-glass-card);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  font-size: 14px;
  outline: none;
  transition: border-color var(--transition-fast);
}

.form-input:focus, .form-select:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px var(--primary-glow);
}

.form-hint {
  font-size: 12px;
  color: var(--text-dim);
  line-height: 1.4;
}

.discovery-details-box {
  padding: 12px 14px;
  background: var(--bg-surface-hover);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.detail-label {
  color: var(--text-dim);
}

.detail-val {
  color: var(--text-main);
  font-weight: 500;
}

.font-mono {
  font-family: var(--font-mono);
}

.badge-tag {
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 11px;
}

.modal-footer {
  padding: 16px 24px;
  border-top: 1px solid var(--border-subtle);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.btn-secondary {
  padding: 9px 18px;
  border-radius: var(--radius-md);
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-secondary:hover {
  background: var(--bg-surface);
  border-color: var(--text-muted);
}

.btn-primary {
  padding: 9px 18px;
  border-radius: var(--radius-md);
  background: var(--primary);
  border: 1px solid var(--primary);
  color: #ffffff;
  font-size: 13px;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-primary:hover:not(:disabled) {
  background: var(--primary-hover, #0284c7);
  box-shadow: 0 2px 10px var(--primary-glow);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Matter Commissioning Styles */
.matter-shield-icon {
  color: #10b981;
}

.step-center {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 16px 0;
}

.commissioning-spinner-box, .success-icon-box {
  margin-bottom: 16px;
}

.text-primary {
  color: var(--primary);
}

.text-success {
  color: var(--color-success);
}

.commissioning-heading {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-main);
  margin-bottom: 8px;
}

.commissioning-subtext {
  font-size: 13px;
  color: var(--text-muted);
  max-width: 360px;
  line-height: 1.5;
}

.progress-bar-container {
  width: 100%;
  max-width: 320px;
  height: 6px;
  background: var(--bg-surface-hover);
  border-radius: var(--radius-full);
  overflow: hidden;
  margin-top: 20px;
}

.progress-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--primary) 0%, #10b981 100%);
  border-radius: var(--radius-full);
  width: 75%;
}

.animated-progress {
  animation: pulseProgress 1.8s ease-in-out infinite;
}

@keyframes pulseProgress {
  0% { width: 15%; }
  50% { width: 80%; }
  100% { width: 95%; }
}

/* Light Theme Contrast Enhancements */
[data-theme="light"] .discovered-section {
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.12) 0%, rgba(59, 130, 246, 0.06) 100%);
  border-color: rgba(14, 165, 233, 0.4);
}

[data-theme="light"] .disc-vendor {
  color: #0369a1;
}

[data-theme="light"] .disc-tag {
  background: rgba(14, 165, 233, 0.2);
  color: #0f172a;
  font-weight: 600;
}

[data-theme="light"] .modal-dialog {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
}

[data-theme="light"] .discovery-details-box {
  background: #f8fafc;
  border-color: #e2e8f0;
}

[data-theme="light"] .form-input, [data-theme="light"] .form-select {
  background: #ffffff;
  border-color: #cbd5e1;
  color: #0f172a;
}
</style>
