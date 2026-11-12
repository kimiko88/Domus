<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClient } from '../api/client'
import { useNotificationsStore } from '../stores/notifications'
import {
  Network,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  WifiOff,
  Zap,
  Radio,
  Search,
  Plus,
  ArrowRight,
  ShieldCheck,
  ShieldAlert,
  Cpu,
  Layers,
  Usb,
  Lock,
  Unlock,
  AlertTriangle,
  Globe,
  Activity
} from '@lucide/vue'
import { soundEffects } from '../utils/audioSynthesizer'
import type { NetworkDiscoveredHost, NetworkLeakAlert, NetworkGuardianLeaksResponse } from '../api/types'
import FirmwareFlasherModal from '../components/modals/FirmwareFlasherModal.vue'

const { t } = useI18n()

interface Integration {
  id: string
  name: string
  type: string
  enabled: boolean
  status: string
  last_error?: string | null
}

interface SuggestedEntity {
  domain: string
  name: string
  key_suffix: string
  unit_of_measurement?: string
  writable: boolean
  icon?: string
}

interface DiscoveredDevice {
  id: string
  brand: string
  name: string
  ip: string
  port: number
  mac?: string
  model: string
  firmware?: string
  status: string
  capabilities: string[]
  suggested_entities: SuggestedEntity[]
  discovered_at: string
}

interface BrandCatalogEntry {
  id: string
  name: string
  category: string
  protocol: string
  zero_cloud: boolean
  default_port: number
  description: string
  supported_models: string[]
  pairing_guide: string
  icon: string
}

const activeTab = ref<'active' | 'discovery' | 'catalog' | 'guardian' | 'cloud'>('active')
const integrations = ref<Integration[]>([])
const discoveredDevices = ref<DiscoveredDevice[]>([])
const catalog = ref<BrandCatalogEntry[]>([])
const loading = ref(false)
const scanning = ref(false)
const pairingId = ref<string | null>(null)
const reconnectingId = ref<string | null>(null)
const manualIp = ref('')
const brandFilter = ref('all')
const isFlasherOpen = ref(false)
const notificationsStore = useNotificationsStore()

// --- Network Scanner & IoT Guardian State ---
const networkHosts = ref<NetworkDiscoveredHost[]>([])
const networkLeaks = ref<NetworkLeakAlert[]>([])
const isolatedMacs = ref<string[]>([])
const probingIp = ref<string | null>(null)
const quarantiningMac = ref<string | null>(null)
const guardianLoading = ref(false)

// --- Official Tuya Cloud OpenAPI ---
interface TuyaCloudDevice {
  id: string
  name: string
  product_name?: string
  category?: string
  online: boolean
  status: { code: string; value: unknown }[]
}
const tuyaCloudDevices = ref<TuyaCloudDevice[]>([])
const tuyaCloudLoading = ref(false)
const tuyaCloudError = ref('')
const tuyaCloudCommanding = ref<string | null>(null)

interface TuyaOAuthStatus { connected: boolean; expires_at?: string }
const tuyaOAuthStatus = ref<TuyaOAuthStatus>({ connected: false })
const tuyaOAuthDevices = ref<TuyaCloudDevice[]>([])
const tuyaOAuthLoading = ref(false)
async function fetchTuyaOAuthStatus() {
  try {
    tuyaOAuthStatus.value = await apiClient.get<TuyaOAuthStatus>('/api/v1/integrations/tuya-oauth/status')
    if (tuyaOAuthStatus.value.connected) await fetchTuyaOAuthDevices()
  } catch {
    tuyaOAuthStatus.value = { connected: false }
    tuyaOAuthDevices.value = []
  }
}
async function fetchTuyaOAuthDevices() {
  tuyaOAuthLoading.value = true
  try {
    const result = await apiClient.get<{ devices: TuyaCloudDevice[] }>('/api/v1/integrations/tuya-oauth/devices')
    tuyaOAuthDevices.value = result.devices || []
  } catch (err: any) {
    notificationsStore.error('Smart Life OAuth', err?.message || 'Impossibile caricare i dispositivi autorizzati.')
  } finally { tuyaOAuthLoading.value = false }
}
async function connectTuyaAccount() {
  try {
    const result = await apiClient.post<{ authorization_url: string }>('/api/v1/integrations/tuya-oauth/authorize', {})
    if (!result.authorization_url || !result.authorization_url.startsWith('https://')) throw new Error('URL di autorizzazione Tuya non valido.')
    window.location.assign(result.authorization_url)
  } catch (err: any) {
    notificationsStore.error('Autorizzazione Smart Life', err?.message || 'Impossibile avviare OAuth.')
  }
}
async function disconnectTuyaAccount() {
  try {
    await apiClient.post('/api/v1/integrations/tuya-oauth/disconnect', {})
    tuyaOAuthStatus.value = { connected: false }
    tuyaOAuthDevices.value = []
    notificationsStore.success('Smart Life scollegato', 'I token OAuth cifrati sono stati eliminati da Domus.')
  } catch (err: any) {
    notificationsStore.error('Disconnessione Smart Life', err?.message || 'Impossibile scollegare l’account.')
  }
}

async function fetchTuyaCloudDevices() {
  tuyaCloudLoading.value = true
  tuyaCloudError.value = ''
  try {
    const result = await apiClient.get<{ devices: TuyaCloudDevice[] }>('/api/v1/integrations/tuya-cloud/devices')
    tuyaCloudDevices.value = result.devices || []
  } catch (err: any) {
    tuyaCloudError.value = err?.message || 'Cloud Tuya non configurato o non raggiungibile.'
  } finally {
    tuyaCloudLoading.value = false
  }
}

async function toggleTuyaCloudStatus(device: TuyaCloudDevice, point: { code: string; value: unknown }) {
  if (typeof point.value !== 'boolean') return
  const key = `${device.id}:${point.code}`
  tuyaCloudCommanding.value = key
  try {
    await apiClient.post(`/api/v1/integrations/tuya-cloud/devices/${encodeURIComponent(device.id)}/commands`, {
      commands: [{ code: point.code, value: !point.value }]
    })
    notificationsStore.success('Comando inviato', `${device.name}: ${point.code}`)
    await fetchTuyaCloudDevices()
  } catch (err: any) {
    notificationsStore.error('Comando Smart Life non riuscito', err?.message || 'Errore Tuya Cloud')
  } finally {
    tuyaCloudCommanding.value = null
  }
}



async function fetchIntegrations() {
  loading.value = true
  try {
    const data = await apiClient.get<Integration[]>('/api/v1/integrations')
    integrations.value = data || []
  } catch (err: any) {
    console.warn('Failed to load integrations:', err)
  } finally {
    loading.value = false
  }
}

async function fetchCatalog() {
  try {
    const data = await apiClient.get<BrandCatalogEntry[]>('/api/v1/integrations/catalog')
    catalog.value = data || []
  } catch (err: any) {
    console.warn('Failed to load brand catalog:', err)
  }
}

async function fetchDiscovered() {
  try {
    const data = await apiClient.get<DiscoveredDevice[]>('/api/v1/integrations/discovery/devices')
    discoveredDevices.value = data || []
  } catch (err: any) {
    console.warn('Failed to load discovered devices:', err)
  }
}

async function scanNetwork(target?: string) {
  scanning.value = true
  try {
    const res = await apiClient.post<{ status: string; devices: DiscoveredDevice[] }>(
      '/api/v1/integrations/discovery/scan',
      { target: target || manualIp.value }
    )
    if (res.devices && res.devices.length > 0) {
      notificationsStore.success(
        t('common.success'),
        `Trovati ${res.devices.length} dispositivi intelligenti sulla rete!`
      )
      manualIp.value = ''
    } else if (target || manualIp.value) {
      notificationsStore.warning('Nessun dispositivo', `Nessun dispositivo riconosciuto all'indirizzo indicato.`)
    }
    await fetchDiscovered()
  } catch (err: any) {
    notificationsStore.error('Errore scansione', err?.message || 'Impossibile completare la scansione.')
  } finally {
    scanning.value = false
  }
}

async function pairDevice(dev: DiscoveredDevice) {
  pairingId.value = dev.id
  try {
    await apiClient.post('/api/v1/integrations/discovery/pair', {
      brand: dev.brand,
      ip: dev.ip,
      port: dev.port,
      name: dev.name
    })
    notificationsStore.success(
      'Dispositivo Associato!',
      `${dev.name} è stato configurato con successo in Domus.`
    )
    await Promise.all([fetchIntegrations(), fetchDiscovered()])
  } catch (err: any) {
    notificationsStore.error('Errore Associazione', err?.message || 'Impossibile associare il dispositivo.')
  } finally {
    pairingId.value = null
  }
}

async function reconnect(integration: Integration) {
  reconnectingId.value = integration.id
  try {
    await apiClient.post(`/api/v1/integrations/${integration.id}/reconnect`)
    notificationsStore.info(
      t('integrations.reconnect'),
      t('integrations.reconnecting', { name: integration.name })
    )
    await fetchIntegrations()
  } catch (err: any) {
    notificationsStore.error(t('common.error'), err?.message || t('integrations.reconnectError'))
  } finally {
    reconnectingId.value = null
  }
}

const filteredCatalog = computed(() => {
  if (brandFilter.value === 'all') return catalog.value
  return catalog.value.filter((b) => b.id === brandFilter.value || b.category.toLowerCase().includes(brandFilter.value))
})

async function fetchGuardianData() {
  guardianLoading.value = true
  try {
    const [devices, leaksRes] = await Promise.all([
      apiClient.get<NetworkDiscoveredHost[]>('/api/v1/network/devices'),
      apiClient.get<NetworkGuardianLeaksResponse>('/api/v1/network/leaks')
    ])
    networkHosts.value = devices || []
    networkLeaks.value = leaksRes?.leaks || []
    isolatedMacs.value = leaksRes?.isolated_macs || []
  } catch (err: any) {
    console.warn('Failed to load network guardian telemetry:', err)
  } finally {
    guardianLoading.value = false
  }
}

async function triggerArpScan() {
  scanning.value = true
  soundEffects.playClick()
  try {
    const res = await apiClient.post<{ count: number; hosts: NetworkDiscoveredHost[] }>('/api/v1/network/scan')
    networkHosts.value = res.hosts || []
    notificationsStore.success('Scansione ARP Completata', `Rilevati ${res.count} host fisici sulla rete locale.`)
    soundEffects.playSuccessChime()
  } catch (err: any) {
    notificationsStore.error('Errore Scansione', err?.message || 'Impossibile completare la scansione di rete.')
  } finally {
    scanning.value = false
  }
}

async function probeHostPorts(ip: string) {
  probingIp.value = ip
  soundEffects.playClick()
  try {
    const res = await apiClient.post<{ ip: string; open_ports: number[]; host: NetworkDiscoveredHost }>('/api/v1/network/probe', { ip })
    notificationsStore.info('Port Probe Completato', `Host ${ip}: porte aperte [${res.open_ports.join(', ')}]`)
    await fetchGuardianData()
  } catch (err: any) {
    notificationsStore.error('Errore Probe', err?.message || 'Impossibile eseguire port probe.')
  } finally {
    probingIp.value = null
  }
}

async function quarantineDevice(mac: string, ip: string, reason?: string) {
  quarantiningMac.value = mac
  soundEffects.playClick()
  try {
    await apiClient.post('/api/v1/network/quarantine', {
      mac,
      ip,
      reason: reason || 'Dispositivo isolato manualmente da Network Guardian'
    })
    notificationsStore.warning('Dispositivo Confinato!', `MAC ${mac} isolato a livello di router/firewall.`)
    soundEffects.playSuccessChime()
    await fetchGuardianData()
  } catch (err: any) {
    notificationsStore.error('Errore Quarantena', err?.message || 'Impossibile isolare il dispositivo.')
  } finally {
    quarantiningMac.value = null
  }
}

async function unblockDevice(mac: string, ip: string) {
  quarantiningMac.value = mac
  soundEffects.playClick()
  try {
    await apiClient.post('/api/v1/network/unblock', { mac, ip })
    notificationsStore.success('Dispositivo Sbloccato', `MAC ${mac} rimosso dalla quarantena.`)
    soundEffects.playSuccessChime()
    await fetchGuardianData()
  } catch (err: any) {
    notificationsStore.error('Errore Sblocco', err?.message || 'Impossibile sbloccare il dispositivo.')
  } finally {
    quarantiningMac.value = null
  }
}

onMounted(() => {
  fetchIntegrations()
  fetchCatalog()
  fetchDiscovered()
  fetchGuardianData()
})
</script>


<template>
  <div class="integrations-view">
    <div class="header-row">
      <div>
        <h2 class="section-title">{{ t('integrations.title') }}</h2>
        <p class="section-subtitle">{{ t('integrations.subtitle') }}</p>
      </div>

      <div class="header-right-tools">
        <button class="btn-flasher-trigger glass-panel" @click="isFlasherOpen = true">
          <Usb :size="15" />
          <span>{{ t('brands.flasherBtn') }}</span>
        </button>

        <!-- Tab Buttons -->
        <div class="tab-switcher glass-panel">
        <button
          :class="['tab-btn', { active: activeTab === 'active' }]"
          @click="activeTab = 'active'"
        >
          <Network :size="16" />
          <span>{{ t('brands.activeTab') }}</span>
          <span class="counter-badge">{{ integrations.length }}</span>
        </button>

        <button
          :class="['tab-btn', { active: activeTab === 'discovery' }]"
          @click="activeTab = 'discovery'"
        >
          <Radio :size="16" />
          <span>{{ t('brands.discoveryTab') }}</span>
          <span v-if="discoveredDevices.length > 0" class="counter-badge pulse">{{ discoveredDevices.length }}</span>
        </button>

        <button
          :class="['tab-btn', { active: activeTab === 'catalog' }]"
          @click="activeTab = 'catalog'"
        >
          <Layers :size="16" />
          <span>{{ t('brands.catalogTab') }}</span>
        </button>

        <button
          :class="['tab-btn', { active: activeTab === 'guardian' }]"
          @click="activeTab = 'guardian'"
        >
          <ShieldAlert :size="16" />
          <span>LAN & IoT Guardian</span>
          <span v-if="networkLeaks.length > 0" class="counter-badge alert-pulse">{{ networkLeaks.length }}</span>
        </button>

        <button
          :class="['tab-btn', { active: activeTab === 'cloud' }]"
          @click="activeTab = 'cloud'; fetchTuyaOAuthStatus()"
        >
          <Globe :size="16" />
          <span>Smart Life Cloud</span>
          <span v-if="tuyaCloudDevices.length > 0" class="counter-badge">{{ tuyaCloudDevices.length }}</span>
        </button>
      </div>
    </div>
  </div>

    <!-- TAB 1: ACTIVE INTEGRATIONS -->
    <div v-if="activeTab === 'active'" class="tab-content">
      <div class="actions-bar">
        <span class="bar-info">Ponti hardware e protocolli attualmente collegati al sistema locale Domus.</span>
        <button class="btn-action glass-panel" :disabled="loading" @click="fetchIntegrations">
          <RefreshCw :size="15" :class="{ 'animate-spin': loading }" />
          <span>{{ t('integrations.refresh') }}</span>
        </button>
      </div>

      <div v-if="integrations.length > 0" class="integrations-grid">
        <div
          v-for="item in integrations"
          :key="item.id"
          :class="['integration-card glass-card', `status-${item.status}`]"
        >
          <div class="card-top">
            <div class="icon-box">
              <Network :size="24" />
            </div>
            <div class="info">
              <h3 class="name">{{ item.name }}</h3>
              <span class="type-tag">{{ item.type }}</span>
            </div>
            <div :class="['status-badge', `badge-${item.status}`]">
              <CheckCircle v-if="item.status === 'connected'" :size="14" />
              <WifiOff v-else-if="item.status === 'disconnected'" :size="14" />
              <AlertCircle v-else :size="14" />
              <span>{{ item.status }}</span>
            </div>
          </div>

          <div v-if="item.last_error" class="error-box">
            <AlertCircle :size="14" />
            <span>{{ item.last_error }}</span>
          </div>

          <div class="card-footer">
            <button
              class="btn-reconnect"
              :disabled="reconnectingId === item.id"
              @click="reconnect(item)"
            >
              <RefreshCw :size="14" :class="{ 'animate-spin': reconnectingId === item.id }" />
              <span>{{ t('integrations.reconnect') }}</span>
            </button>
          </div>
        </div>
      </div>

      <div v-else-if="!loading" class="empty-state glass-card">
        <Network :size="36" class="empty-icon" />
        <h3 class="empty-title">{{ t('integrations.emptyTitle') }}</h3>
        <p class="empty-desc">{{ t('integrations.emptyDesc') }}</p>
      </div>
    </div>

    <!-- TAB 2: LAN AUTO-DISCOVERY & 1-CLICK PAIRING -->
    <div v-else-if="activeTab === 'discovery'" class="tab-content">
      <div class="discovery-toolbar glass-card">
        <div class="discovery-intro">
          <div class="icon-chip"><Radio :size="20" /></div>
          <div>
            <h3 class="disc-title">Scansione Zero-Cloud sulla Rete Locale</h3>
            <p class="disc-sub">Rileva Shelly, Philips Hue, Sonoff LAN, Tasmota e WLED presenti sul Wi-Fi di casa.</p>
          </div>
        </div>

        <div class="discovery-inputs">
          <div class="manual-probe-box">
            <input
              v-model="manualIp"
              type="text"
              placeholder="Indirizzo IP (es. 192.168.1.150)"
              class="ip-input"
              @keyup.enter="scanNetwork()"
            />
            <button class="btn-probe" :disabled="scanning || !manualIp.trim()" @click="scanNetwork()">
              <Plus :size="15" />
              <span>Sonda IP</span>
            </button>
          </div>

          <button class="btn-scan" :disabled="scanning" @click="scanNetwork()">
            <RefreshCw :size="16" :class="{ 'animate-spin': scanning }" />
            <span>{{ scanning ? t('brands.scanning') : t('brands.scanLan') }}</span>
          </button>
        </div>
      </div>

      <!-- Discovered Devices List -->
      <div v-if="discoveredDevices.length > 0" class="discovered-grid">
        <div
          v-for="dev in discoveredDevices"
          :key="dev.id"
          class="discovered-card glass-card"
        >
          <div class="disc-card-header">
            <div class="disc-badge-brand">
              <Zap v-if="dev.brand === 'shelly'" :size="16" />
              <Cpu v-else :size="16" />
              <span>{{ dev.brand.toUpperCase() }}</span>
            </div>
            <span class="zero-cloud-tag">
              <ShieldCheck :size="13" />
              <span>100% Locale</span>
            </span>
          </div>

          <div class="disc-card-body">
            <h4 class="disc-name">{{ dev.name }}</h4>
            <div class="disc-meta">
              <span class="disc-ip">IP: {{ dev.ip }}:{{ dev.port }}</span>
              <span v-if="dev.model" class="disc-model">{{ dev.model }}</span>
              <span v-if="dev.firmware" class="disc-fw">FW: {{ dev.firmware }}</span>
            </div>

            <!-- Entities preview -->
            <div v-if="dev.suggested_entities && dev.suggested_entities.length > 0" class="entities-preview">
              <span class="preview-title">Entità create con 1-Click:</span>
              <div class="tags-cloud">
                <span
                  v-for="(ent, i) in dev.suggested_entities"
                  :key="i"
                  class="entity-chip"
                >
                  {{ ent.domain }}: {{ ent.name }}
                </span>
              </div>
            </div>
          </div>

          <div class="disc-card-footer">
            <button
              class="btn-pair"
              :disabled="pairingId === dev.id || dev.status === 'paired'"
              @click="pairDevice(dev)"
            >
              <CheckCircle v-if="dev.status === 'paired'" :size="16" />
              <RefreshCw v-else-if="pairingId === dev.id" :size="16" class="animate-spin" />
              <ArrowRight v-else :size="16" />
              <span>{{ dev.status === 'paired' ? t('brands.configured') : t('brands.pair') }}</span>
            </button>
          </div>
        </div>
      </div>

      <div v-else class="empty-state glass-card">
        <Radio :size="40" class="empty-icon" />
        <h3 class="empty-title">Nessun dispositivo in attesa di associazione</h3>
        <p class="empty-desc">
          Clicca su "Scansiona Rete" oppure inserisci l'indirizzo IP del tuo dispositivo Shelly, Sonoff o Tasmota per importarlo all'istante senza cloud.
        </p>
      </div>
    </div>

    <!-- TAB 3: BRAND CATALOG & ECOSYSTEM -->
    <div v-else-if="activeTab === 'catalog'" class="tab-content">
      <div class="catalog-grid">
        <div
          v-for="entry in filteredCatalog"
          :key="entry.id"
          class="brand-card glass-card"
        >
          <div class="brand-top">
            <div class="brand-icon-box">
              <Zap v-if="entry.id === 'shelly'" :size="24" />
              <Layers v-else-if="entry.id === 'matter'" :size="24" />
              <Cpu v-else :size="24" />
            </div>
            <div class="brand-titles">
              <h3 class="brand-title">{{ entry.name }}</h3>
              <span class="brand-category">{{ entry.category }}</span>
            </div>
            <div class="zero-cloud-pill">
              <ShieldCheck :size="13" />
              <span>Zero-Cloud</span>
            </div>
          </div>

          <p class="brand-desc">{{ entry.description }}</p>

          <div class="brand-specs">
            <div class="spec-row">
              <span class="spec-label">Protocollo:</span>
              <span class="spec-val">{{ entry.protocol }}</span>
            </div>
            <div class="spec-row">
              <span class="spec-label">Guida Rapida:</span>
              <span class="spec-val">{{ entry.pairing_guide }}</span>
            </div>
          </div>

          <div class="models-box">
            <span class="models-title">Modelli Supportati:</span>
            <div class="models-tags">
              <span
                v-for="(mod, mi) in entry.supported_models"
                :key="mi"
                class="model-tag"
              >
                {{ mod }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- TAB 4: LAN & IOT NETWORK GUARDIAN -->
    <div v-if="activeTab === 'guardian'" class="tab-content">
      <!-- Top Telemetry & Controls Banner -->
      <div class="guardian-banner glass-panel">
        <div class="guardian-meta">
          <div class="guardian-icon-box">
            <ShieldAlert :size="28" />
          </div>
          <div>
            <h3 class="guardian-title">IoT Network Guardian & Confinamento LAN</h3>
            <p class="guardian-subtitle">
              Monitoraggio attivo della LAN, analisi passiva ARP/DHCP e rilevamento istantaneo di tentativi di leak verso server cloud esterni non autorizzati.
            </p>
          </div>
        </div>

        <div class="guardian-stats-row">
          <div class="stat-pill glass-card">
            <Globe :size="16" class="stat-icon-blue" />
            <div class="stat-info">
              <span class="stat-value">{{ networkHosts.length }}</span>
              <span class="stat-label">Host LAN Rilevati</span>
            </div>
          </div>

          <div class="stat-pill glass-card">
            <Lock :size="16" class="stat-icon-amber" />
            <div class="stat-info">
              <span class="stat-value">{{ isolatedMacs.length }}</span>
              <span class="stat-label">In Quarantena</span>
            </div>
          </div>

          <div class="stat-pill glass-card">
            <AlertTriangle :size="16" class="stat-icon-red" />
            <div class="stat-info">
              <span class="stat-value">{{ networkLeaks.length }}</span>
              <span class="stat-label">Leak WAN Intercettati</span>
            </div>
          </div>

          <button
            class="btn-scan-arp glass-panel"
            :disabled="scanning || guardianLoading"
            @click="triggerArpScan"
          >
            <RefreshCw :size="16" :class="{ 'animate-spin': scanning }" />
            <span>{{ scanning ? 'Scansione in corso...' : 'Scansione ARP' }}</span>
          </button>
        </div>
      </div>

      <!-- Section: WAN Leak Alerts -->
      <div class="guardian-section">
        <div class="section-subhead">
          <div class="subhead-left">
            <AlertTriangle :size="18" class="text-warning" />
            <h4 class="subhead-title">Tentativi di Connessione Esterna (WAN Leaks)</h4>
          </div>
          <span class="subhead-badge" v-if="networkLeaks.length > 0">
            {{ networkLeaks.length }} anomalie registrate
          </span>
        </div>

        <div v-if="networkLeaks.length === 0" class="empty-guardian-state glass-card">
          <ShieldCheck :size="36" class="text-success" />
          <h4>Nessun Leak Rilevato</h4>
          <p>Tutti i dispositivi IoT monitorati sono conformi e operano rigorosamente offline all'interno della rete locale.</p>
        </div>

        <div v-else class="leaks-list">
          <div
            v-for="leak in networkLeaks"
            :key="leak.id"
            class="leak-card glass-card"
          >
            <div class="leak-left">
              <div class="leak-indicator"></div>
              <div class="leak-details">
                <div class="leak-header-row">
                  <span class="leak-device">{{ leak.device_name || 'Dispositivo IoT' }}</span>
                  <span class="leak-time">{{ new Date(leak.detected_at).toLocaleTimeString() }}</span>
                </div>
                <div class="leak-flow">
                  <code>{{ leak.source_ip }}</code>
                  <ArrowRight :size="14" class="flow-arrow" />
                  <code class="text-danger">{{ leak.destination_ip }}:{{ leak.port }} ({{ leak.protocol.toUpperCase() }})</code>
                </div>
                <p class="leak-reason">{{ leak.reason }}</p>
              </div>
            </div>

            <div class="leak-actions">
              <button
                v-if="leak.source_mac && isolatedMacs.includes(leak.source_mac.toLowerCase())"
                class="btn-quarantine active"
                :disabled="quarantiningMac === leak.source_mac"
                @click="unblockDevice(leak.source_mac, leak.source_ip)"
              >
                <Unlock :size="15" />
                <span>Sblocca</span>
              </button>
              <button
                v-else
                class="btn-quarantine"
                :disabled="quarantiningMac === leak.source_mac"
                @click="quarantineDevice(leak.source_mac || '', leak.source_ip, leak.reason)"
              >
                <Lock :size="15" />
                <span>Metti in Quarantena</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Section: Discovered Physical Hosts -->
      <div class="guardian-section">
        <div class="section-subhead">
          <div class="subhead-left">
            <Globe :size="18" class="text-primary" />
            <h4 class="subhead-title">Mappa Host Rilevati su LAN (OUI & Port Fingerprint)</h4>
          </div>
          <span class="subhead-badge">{{ networkHosts.length }} Host</span>
        </div>

        <div v-if="networkHosts.length === 0" class="empty-guardian-state glass-card">
          <RefreshCw :size="32" class="animate-spin text-muted" />
          <p>Nessun host registrato nella cache ARP. Clicca su "Scansione ARP" per rilevare i nodi.</p>
        </div>

        <div v-else class="hosts-grid">
          <div
            v-for="host in networkHosts"
            :key="host.mac || host.ip"
            :class="['host-card glass-card', { 'is-quarantined': host.mac && isolatedMacs.includes(host.mac.toLowerCase()) }]"
          >
            <div class="host-top">
              <div class="host-brand-icon">
                <Cpu :size="20" />
              </div>
              <div class="host-meta">
                <span class="host-vendor">{{ host.vendor || 'Generic LAN Host' }}</span>
                <span class="host-ip">{{ host.ip }}</span>
              </div>
              <div v-if="host.mac && isolatedMacs.includes(host.mac.toLowerCase())" class="badge-quarantined">
                <Lock :size="12" />
                <span>ISOLATO</span>
              </div>
              <div v-else-if="host.is_iot" class="badge-iot">
                <Zap :size="12" />
                <span>IoT</span>
              </div>
            </div>

            <div class="host-specs">
              <div class="spec-line">
                <span class="spec-key">MAC:</span>
                <code class="spec-code">{{ host.mac }}</code>
              </div>
              <div v-if="host.suggested_plugin" class="spec-line">
                <span class="spec-key">Plugin:</span>
                <span class="spec-val-plugin">{{ host.suggested_plugin }}</span>
              </div>
              <div class="spec-line">
                <span class="spec-key">Porte Aperte:</span>
                <span v-if="host.open_ports && host.open_ports.length > 0" class="ports-list">
                  <span v-for="p in host.open_ports" :key="p" class="port-tag">{{ p }}</span>
                </span>
                <span v-else class="text-muted-xs">Nessuna sonda recente</span>
              </div>
            </div>

            <div class="host-footer">
              <button
                class="btn-host-probe"
                :disabled="probingIp === host.ip"
                @click="probeHostPorts(host.ip)"
                title="Esegui TCP probe sulle porte IoT comuni (554, 1883, 6053, 502, 3671)"
              >
                <Activity :size="14" :class="{ 'animate-spin': probingIp === host.ip }" />
                <span>{{ probingIp === host.ip ? 'Probe...' : 'Probe Porte' }}</span>
              </button>

              <button
                v-if="host.mac && isolatedMacs.includes(host.mac.toLowerCase())"
                class="btn-host-quarantine unblock"
                :disabled="quarantiningMac === host.mac"
                @click="unblockDevice(host.mac, host.ip)"
              >
                <Unlock :size="14" />
                <span>Sblocca</span>
              </button>
              <button
                v-else
                class="btn-host-quarantine"
                :disabled="quarantiningMac === host.mac"
                @click="quarantineDevice(host.mac, host.ip, 'Isolamento manuale da mappa host')"
              >
                <Lock :size="14" />
                <span>Isola</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- TAB 5: OFFICIAL TUYA / SMART LIFE CLOUD -->
    <div v-if="activeTab === 'cloud'" class="tab-content">
      <div class="discovery-toolbar glass-card">
        <div class="discovery-intro">
          <div class="icon-chip"><Lock :size="20" /></div>
          <div>
            <h3 class="disc-title">Account Smart Life autorizzato</h3>
            <p class="disc-sub">Collega il tuo account tramite la pagina ufficiale Tuya. Domus non richiede né memorizza la password Smart Life.</p>
            <p class="disc-sub">{{ tuyaOAuthStatus.connected ? 'Account collegato: token cifrati sul server.' : 'Nessun account utente collegato.' }}</p>
          </div>
        </div>
        <div class="discovery-inputs">
          <button v-if="!tuyaOAuthStatus.connected" class="btn-scan" @click="connectTuyaAccount"><Lock :size="16" /><span>Collega account Smart Life</span></button>
          <template v-else>
            <button class="btn-action glass-panel" :disabled="tuyaOAuthLoading" @click="fetchTuyaOAuthDevices"><RefreshCw :size="15" :class="{ 'animate-spin': tuyaOAuthLoading }" /><span>Aggiorna account</span></button>
            <button class="btn-action glass-panel" @click="disconnectTuyaAccount"><Unlock :size="15" /><span>Disconnetti</span></button>
          </template>
        </div>
      </div>
      <div v-if="tuyaOAuthStatus.connected" class="hosts-grid">
        <div v-for="device in tuyaOAuthDevices" :key="device.id" class="host-card glass-card">
          <div class="host-top">
            <div class="host-brand-icon"><Zap :size="20" /></div>
            <div class="host-meta"><span class="host-vendor">{{ device.name }}</span><span class="host-ip">{{ device.product_name || device.category || device.id }}</span></div>
            <span :class="['badge-iot', { 'badge-quarantined': !device.online }]">{{ device.online ? 'Online' : 'Offline' }}</span>
          </div>
          <div class="host-specs"><div v-for="point in device.status || []" :key="point.code" class="spec-line"><span class="spec-key">{{ point.code }}</span><span class="spec-code">{{ String(point.value) }}</span></div></div>
        </div>
      </div>

      <div class="discovery-toolbar glass-card">
        <div class="discovery-intro">
          <div class="icon-chip"><Globe :size="20" /></div>
          <div>
            <h3 class="disc-title">Tuya / Smart Life Cloud</h3>
            <p class="disc-sub">Dispositivi collegati al progetto Tuya Cloud configurato sul server Domus. Le credenziali non vengono mai inviate al browser.</p>
          </div>
        </div>
        <button class="btn-scan" :disabled="tuyaCloudLoading" @click="fetchTuyaCloudDevices">
          <RefreshCw :size="16" :class="{ 'animate-spin': tuyaCloudLoading }" />
          <span>Aggiorna dispositivi</span>
        </button>
      </div>

      <div v-if="tuyaCloudError" class="empty-guardian-state glass-card">
        <AlertCircle :size="30" />
        <h4>Cloud Tuya non disponibile</h4>
        <p>{{ tuyaCloudError }}</p>
        <p>Configura TUYA_CLOUD_CLIENT_ID, TUYA_CLOUD_CLIENT_SECRET e, se necessario, TUYA_CLOUD_BASE_URL / TUYA_CLOUD_SCHEMA sul server. Consulta docs/integrations/tuya-cloud-and-pdf-scanning.md.</p>
      </div>

      <div v-else-if="tuyaCloudLoading && tuyaCloudDevices.length === 0" class="empty-guardian-state glass-card">
        <RefreshCw :size="28" class="animate-spin" />
        <p>Caricamento dispositivi Smart Life…</p>
      </div>

      <div v-else-if="tuyaCloudDevices.length === 0" class="empty-guardian-state glass-card">
        <Globe :size="32" />
        <h4>Nessun dispositivo visibile al progetto</h4>
        <p>Verifica regione, autorizzazioni API e collegamento dell’app Smart Life/OEM al progetto Tuya Cloud.</p>
      </div>

      <div v-else class="hosts-grid">
        <div v-for="device in tuyaCloudDevices" :key="device.id" class="host-card glass-card">
          <div class="host-top">
            <div class="host-brand-icon"><Zap :size="20" /></div>
            <div class="host-meta">
              <span class="host-vendor">{{ device.name }}</span>
              <span class="host-ip">{{ device.product_name || device.category || device.id }}</span>
            </div>
            <span :class="['badge-iot', { 'badge-quarantined': !device.online }]">{{ device.online ? 'Online' : 'Offline' }}</span>
          </div>
          <div class="host-specs">
            <div v-for="point in device.status || []" :key="point.code" class="spec-line">
              <span class="spec-key">{{ point.code }}</span>
              <span class="spec-code">{{ String(point.value) }}</span>
              <button
                v-if="typeof point.value === 'boolean'"
                class="btn-host-probe"
                :disabled="tuyaCloudCommanding === `${device.id}:${point.code}` || !device.online"
                @click="toggleTuyaCloudStatus(device, point)"
              >
                {{ point.value ? 'Spegni' : 'Accendi' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <FirmwareFlasherModal :is-open="isFlasherOpen" @close="isFlasherOpen = false" />
  </div>
</template>

<style scoped>
.integrations-view {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  flex-wrap: wrap;
}

.section-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.02em;
}

.section-subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 4px;
}

.header-right-tools {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.btn-flasher-trigger {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 8px 16px;
  border-radius: var(--radius-md);
  background: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.3);
  color: #10b981;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-flasher-trigger:hover {
  background: rgba(16, 185, 129, 0.22);
  transform: translateY(-1px);
}

.tab-switcher {
  display: inline-flex;
  padding: 4px;
  border-radius: var(--radius-lg);
  gap: 6px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-subtle);
}

.tab-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: var(--radius-md);
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.tab-btn:hover {
  color: var(--text-main);
  background: rgba(255, 255, 255, 0.05);
}

.tab-btn.active {
  color: #fff;
  background: var(--primary);
  box-shadow: 0 2px 10px rgba(14, 165, 233, 0.3);
}

.counter-badge {
  padding: 2px 7px;
  border-radius: var(--radius-full);
  font-size: 11px;
  background: rgba(255, 255, 255, 0.2);
}

.counter-badge.pulse {
  background: #10b981;
  color: white;
  animation: pulse-ring 2s infinite;
}

@keyframes pulse-ring {
  0% { transform: scale(1); }
  50% { transform: scale(1.1); }
  100% { transform: scale(1); }
}

.tab-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.actions-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.bar-info {
  font-size: 14px;
  color: var(--text-muted);
}

.btn-action {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: var(--radius-md);
  font-size: 13px;
  color: var(--text-main);
  cursor: pointer;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  transition: all var(--transition-fast);
}

.btn-action:hover:not(:disabled) {
  background: var(--bg-surface-hover);
}

.integrations-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
}

.integration-card {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
  transition: all var(--transition-normal);
}

.card-top {
  display: flex;
  align-items: center;
  gap: 14px;
}

.icon-box {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: rgba(14, 165, 233, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
  flex-shrink: 0;
}

.info {
  flex: 1;
  min-width: 0;
}

.name {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.type-tag {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-muted);
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 500;
  text-transform: capitalize;
}

.badge-connected {
  background: var(--color-success-bg);
  color: var(--color-success);
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.badge-disconnected {
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-dim);
  border: 1px solid var(--border-card);
}

.badge-error {
  background: var(--color-danger-bg);
  color: var(--color-danger);
  border: 1px solid rgba(244, 63, 94, 0.3);
}

.error-box {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  background: var(--color-danger-bg);
  color: var(--color-danger);
  font-size: 12px;
}

.card-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}

.btn-reconnect {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  font-size: 13px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-reconnect:hover:not(:disabled) {
  background: var(--bg-surface-hover);
  border-color: rgba(255, 255, 255, 0.2);
}

/* DISCOVERY TAB STYLES */
.discovery-toolbar {
  padding: 20px;
  border-radius: var(--radius-xl);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  flex-wrap: wrap;
  border: 1px solid var(--border-subtle);
}

.discovery-intro {
  display: flex;
  align-items: center;
  gap: 16px;
}

.icon-chip {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  display: flex;
  align-items: center;
  justify-content: center;
}

.disc-title {
  font-size: 17px;
  font-weight: 600;
  color: var(--text-main);
}

.disc-sub {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 2px;
}

.discovery-inputs {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.manual-probe-box {
  display: flex;
  align-items: center;
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.ip-input {
  background: transparent;
  border: none;
  padding: 8px 12px;
  font-size: 13px;
  color: var(--text-main);
  outline: none;
  width: 200px;
}

.btn-probe {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.08);
  border: none;
  border-left: 1px solid var(--border-subtle);
  color: var(--text-main);
  font-size: 13px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-probe:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.15);
}

.btn-scan {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 9px 18px;
  border-radius: var(--radius-md);
  background: var(--primary);
  color: #fff;
  border: none;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  box-shadow: 0 2px 10px rgba(14, 165, 233, 0.3);
  transition: all var(--transition-fast);
}

.btn-scan:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.discovered-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 20px;
}

.discovered-card {
  padding: 22px;
  border-radius: var(--radius-xl);
  display: flex;
  flex-direction: column;
  gap: 16px;
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
  transition: all var(--transition-normal);
}

.disc-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.disc-badge-brand {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
  font-size: 12px;
  font-weight: 700;
}

.zero-cloud-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: #10b981;
  background: rgba(16, 185, 129, 0.12);
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  font-weight: 500;
}

.disc-card-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.disc-name {
  font-size: 17px;
  font-weight: 600;
  color: var(--text-main);
}

.disc-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  color: var(--text-muted);
}

.disc-ip {
  font-family: var(--font-mono);
  color: var(--text-main);
}

.entities-preview {
  margin-top: 6px;
  padding-top: 10px;
  border-top: 1px dashed var(--border-subtle);
}

.preview-title {
  font-size: 12px;
  color: var(--text-muted);
  display: block;
  margin-bottom: 6px;
}

.tags-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.entity-chip {
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-subtle);
  font-size: 11px;
  color: var(--text-dim);
}

.disc-card-footer {
  margin-top: auto;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}

.btn-pair {
  width: 100%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px;
  border-radius: var(--radius-md);
  background: #10b981;
  color: #fff;
  border: none;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-pair:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

/* CATALOG TAB STYLES */
.catalog-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  gap: 20px;
}

.brand-card {
  padding: 24px;
  border-radius: var(--radius-xl);
  display: flex;
  flex-direction: column;
  gap: 16px;
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
}

.brand-top {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.brand-icon-box {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  background: rgba(14, 165, 233, 0.12);
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.brand-titles {
  flex: 1;
}

.brand-title {
  font-size: 17px;
  font-weight: 700;
  color: var(--text-main);
}

.brand-category {
  font-size: 12px;
  color: var(--text-muted);
}

.zero-cloud-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border-radius: var(--radius-full);
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  font-size: 11px;
  font-weight: 600;
}

.brand-desc {
  font-size: 13px;
  color: var(--text-dim);
  line-height: 1.5;
}

.brand-specs {
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: rgba(0, 0, 0, 0.2);
  padding: 12px;
  border-radius: var(--radius-md);
}

.spec-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.spec-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
}

.spec-val {
  font-size: 12px;
  color: var(--text-main);
}

.models-box {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.models-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
}

.models-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.model-tag {
  font-size: 11px;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-subtle);
  color: var(--text-dim);
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

/* ========================================================
   NETWORK GUARDIAN & LAN CONFINEMENT STYLES
   ======================================================== */
.guardian-banner {
  padding: 24px;
  border-radius: var(--radius-xl);
  display: flex;
  flex-direction: column;
  gap: 20px;
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.05), rgba(14, 165, 233, 0.05));
  border: 1px solid var(--border-subtle);
}

.guardian-meta {
  display: flex;
  align-items: center;
  gap: 16px;
}

.guardian-icon-box {
  width: 52px;
  height: 52px;
  border-radius: var(--radius-lg);
  background: rgba(239, 68, 68, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.guardian-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.01em;
}

.guardian-subtitle {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 4px;
  line-height: 1.4;
}

.guardian-stats-row {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-wrap: wrap;
}

.stat-pill {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  border-radius: var(--radius-lg);
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  flex: 1;
  min-width: 160px;
}

.stat-icon-blue { color: #0ea5e9; }
.stat-icon-amber { color: #f59e0b; }
.stat-icon-red { color: #ef4444; }

.stat-info {
  display: flex;
  flex-direction: column;
}

.stat-value {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-main);
  font-family: var(--font-mono);
}

.stat-label {
  font-size: 11px;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.btn-scan-arp {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border-radius: var(--radius-lg);
  background: var(--primary);
  color: white;
  border: none;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-scan-arp:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.guardian-section {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.section-subhead {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.subhead-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.subhead-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-main);
}

.subhead-badge {
  font-size: 12px;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-muted);
}

.empty-guardian-state {
  padding: 36px 20px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  border-radius: var(--radius-lg);
  border: 1px dashed var(--border-card);
  color: var(--text-muted);
}

.empty-guardian-state h4 {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-main);
}

.empty-guardian-state p {
  font-size: 13px;
  max-width: 480px;
}

.leaks-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.leak-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-radius: var(--radius-lg);
  background: rgba(239, 68, 68, 0.04);
  border: 1px solid rgba(239, 68, 68, 0.25);
  gap: 16px;
  flex-wrap: wrap;
}

.leak-left {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  flex: 1;
  min-width: 260px;
}

.leak-indicator {
  width: 10px;
  height: 10px;
  border-radius: var(--radius-full);
  background: #ef4444;
  margin-top: 6px;
  box-shadow: 0 0 10px rgba(239, 68, 68, 0.6);
  animation: pulse-ring 2s infinite;
}

.leak-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.leak-header-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.leak-device {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-main);
}

.leak-time {
  font-size: 12px;
  color: var(--text-muted);
}

.leak-flow {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.flow-arrow {
  color: var(--text-muted);
}

.leak-reason {
  font-size: 12px;
  color: var(--text-muted);
}

.btn-quarantine {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 14px;
  border-radius: var(--radius-md);
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-quarantine:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.25);
}

.btn-quarantine.active {
  background: rgba(16, 185, 129, 0.15);
  border-color: rgba(16, 185, 129, 0.3);
  color: #10b981;
}

.hosts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.host-card {
  padding: 16px;
  border-radius: var(--radius-lg);
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  gap: 14px;
  transition: all var(--transition-normal);
}

.host-card.is-quarantined {
  border-color: rgba(245, 158, 11, 0.4);
  background: rgba(245, 158, 11, 0.04);
}

.host-top {
  display: flex;
  align-items: center;
  gap: 12px;
}

.host-brand-icon {
  width: 38px;
  height: 38px;
  border-radius: var(--radius-md);
  background: rgba(14, 165, 233, 0.1);
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.host-meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.host-vendor {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.host-ip {
  font-size: 12px;
  color: var(--text-muted);
  font-family: var(--font-mono);
}

.badge-quarantined {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: var(--radius-full);
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  font-size: 11px;
  font-weight: 700;
}

.badge-iot {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: var(--radius-full);
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
  font-size: 11px;
  font-weight: 600;
}

.host-specs {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 12px;
}

.spec-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.spec-key {
  color: var(--text-muted);
}

.spec-code {
  font-family: var(--font-mono);
  color: var(--text-main);
}

.spec-val-plugin {
  font-family: var(--font-mono);
  color: #10b981;
  background: rgba(16, 185, 129, 0.1);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}

.ports-list {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.port-tag {
  padding: 1px 5px;
  border-radius: var(--radius-sm);
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
  font-family: var(--font-mono);
  font-size: 11px;
}

.text-muted-xs {
  font-size: 11px;
  color: var(--text-muted);
}

.host-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: auto;
  padding-top: 10px;
  border-top: 1px solid var(--border-subtle);
}

.btn-host-probe,
.btn-host-quarantine {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  padding: 6px 10px;
  border-radius: var(--radius-md);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-host-probe {
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-card);
  color: var(--text-main);
}

.btn-host-probe:hover:not(:disabled) {
  background: var(--bg-surface-active);
}

.btn-host-quarantine {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.25);
  color: #ef4444;
}

.btn-host-quarantine:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.2);
}

.btn-host-quarantine.unblock {
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.25);
  color: #10b981;
}

.alert-pulse {
  background: #ef4444 !important;
  color: white;
  animation: pulse-ring 1.8s infinite;
}
</style>

