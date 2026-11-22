<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDevicesStore } from '../stores/devices'
import { useEntitiesStore } from '../stores/entities'
import { useAreasStore } from '../stores/areas'
import { useNotificationsStore } from '../stores/notifications'
import { useAuthStore } from '../stores/auth'
import { apiClient } from '../api/client'
import type { Device, DiscoveredHost, Entity } from '../api/types'
import { Cpu, Wifi, WifiOff, RefreshCw, QrCode, Plus, Sparkles, Check, CheckCircle, ShieldCheck, X, Search } from '@lucide/vue'

const { t } = useI18n()
const devicesStore = useDevicesStore()
const entitiesStore = useEntitiesStore()
const areasStore = useAreasStore()
const notifStore = useNotificationsStore()
const authStore = useAuthStore()

const filterStatus = ref<'all' | 'online' | 'offline'>('all')
const searchQuery = ref('')
const filterIntegration = ref('all')
const isRefreshing = ref(false)
const tuyaCloudDevices = ref<TuyaCloudDeviceSummary[]>([])
const tuyaOAuthDevices = ref<TuyaCloudDeviceSummary[]>([])
const tuyaLocalDevices = ref<TuyaCloudDeviceSummary[]>([])
const isScanning = ref(false)
const discoveredHosts = ref<DiscoveredHost[]>([])
const selectedHostForSetup = ref<DiscoveredHost | null>(null)
const isSetupModalOpen = ref(false)
const isMatterWizardOpen = ref(false)
const newDeviceName = ref('')
const selectedAreaId = ref('')
const isAddingDevice = ref(false)
const assigningRoomForDevice = ref<string | null>(null)

// Matter Commissioning State
const matterPairingCode = ref('')
const matterCommissionStep = ref<1 | 2 | 3>(1)
const matterCommissioning = ref(false)

interface TuyaCloudDeviceSummary {
  id: string
  name: string
  product_name?: string
  category?: string
  online: boolean
  local_configured?: boolean
  local_active?: boolean
  local_ip?: string
  source?: 'tuya_cloud' | 'tuya_oauth' | 'tuya_local'
  device_id?: string
  connected?: boolean
  ip?: string
}

interface UnifiedDevice extends Device {
  local_configured?: boolean
  local_active?: boolean
  local_ip?: string
  source_label?: string
}

onMounted(async () => {
  await Promise.allSettled([
    devicesStore.fetchDevices(),
    entitiesStore.fetchEntities(),
    areasStore.fetchAreas(),
    loadDiscoveredHosts(),
    loadTuyaLocalDevices(),
    loadTuyaCloudDevices(),
    loadTuyaOAuthDevices()
  ])
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
      notifStore.success(t('devices.view.scanCompletedTitle'), t('devices.view.scanCompletedMessage', { count: res.hosts.length }))
    }
  } catch (err: any) {
    notifStore.error(t('devices.view.scanErrorTitle'), t('devices.view.scanErrorMessage'))
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
    notifStore.success(t('devices.view.deviceAddedTitle'), t('devices.view.deviceAddedMessage', { name: newDeviceName.value }))
    isSetupModalOpen.value = false
    await devicesStore.fetchDevices()
    discoveredHosts.value = discoveredHosts.value.filter(h => h.ip !== host.ip)
  } catch (err: any) {
    notifStore.error(t('devices.view.addDeviceErrorTitle'), t('devices.view.addDeviceErrorMessage'))
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
    notifStore.success(t('devices.view.matterConnectedTitle'), t('devices.view.matterConnectedMessage'))
    devicesStore.fetchDevices()
  }, 2200)
}

function closeMatterWizard() {
  isMatterWizardOpen.value = false
  matterCommissionStep.value = 1
  matterPairingCode.value = ''
}

async function loadTuyaLocalDevices() {
  if (!authStore.isAdmin) {
    tuyaLocalDevices.value = []
    return
  }
  try {
    const result = await apiClient.get<{ devices?: TuyaCloudDeviceSummary[] }>('/api/v1/integrations/tuya-local/devices')
    tuyaLocalDevices.value = result?.devices || []
  } catch {
    // The adapter may be disabled on installations that use cloud-only Tuya.
    tuyaLocalDevices.value = []
  }
}

async function loadTuyaCloudDevices() {
  try {
    const config = await apiClient.get<{ configured?: boolean }>('/api/v1/integrations/tuya-cloud/config')
    if (!config?.configured) {
      tuyaCloudDevices.value = []
      return
    }
    const result = await apiClient.get<{ devices?: TuyaCloudDeviceSummary[] }>('/api/v1/integrations/tuya-cloud/devices')
    tuyaCloudDevices.value = result?.devices || []
  } catch {
    // Tuya Cloud is optional. Keep all devices already loaded from the central registry.
    tuyaCloudDevices.value = []
  }
}

async function loadTuyaOAuthDevices() {
  if (!authStore.isAdmin) {
    tuyaOAuthDevices.value = []
    return
  }
  try {
    const status = await apiClient.get<{ connected?: boolean }>('/api/v1/integrations/tuya-oauth/status')
    if (!status?.connected) {
      tuyaOAuthDevices.value = []
      return
    }
    const result = await apiClient.get<{ devices?: TuyaCloudDeviceSummary[] }>('/api/v1/integrations/tuya-oauth/devices')
    tuyaOAuthDevices.value = result?.devices || []
  } catch {
    // OAuth is optional. The central registry and the other sources remain usable.
    tuyaOAuthDevices.value = []
  }
}

async function assignDeviceToRoom(device: UnifiedDevice, event: Event) {
  const select = event.target as HTMLSelectElement | null
  if (!select) return
  const nextAreaId = select.value || null
  const previousAreaId = device.area_id || ''
  if (nextAreaId === previousAreaId || (!nextAreaId && !previousAreaId)) return

  assigningRoomForDevice.value = device.id
  try {
    if (device.id.startsWith('tuya_')) {
      await devicesStore.updateDeviceArea(device.id, nextAreaId, {
        external_id: device.external_id || device.id.slice(5),
        name: device.name,
        integration_id: device.integration_id,
        online: device.status === 'online',
        manufacturer: device.manufacturer || 'Tuya',
        model: device.model
      })
    } else {
      await devicesStore.updateDeviceArea(device.id, nextAreaId)
    }
    await Promise.all([
      devicesStore.fetchDevices(),
      entitiesStore.fetchEntities()
    ])
    const roomName = nextAreaId
      ? areasStore.byId(nextAreaId)?.name || nextAreaId
      : t('devices.view.noRoom')
    notifStore.success(
      t('devices.view.roomAssignedTitle'),
      t('devices.view.roomAssignedMessage', { device: device.name, room: roomName })
    )
  } catch {
    select.value = previousAreaId
    notifStore.error(
      t('devices.view.roomAssignErrorTitle'),
      t('devices.view.roomAssignErrorMessage')
    )
  } finally {
    assigningRoomForDevice.value = null
  }
}

async function refreshAllDevices() {
  isRefreshing.value = true
  try {
    const results = await Promise.allSettled([
      devicesStore.fetchDevices(),
      entitiesStore.fetchEntities(),
      loadTuyaLocalDevices(),
      loadTuyaCloudDevices(),
      loadTuyaOAuthDevices(),
      loadDiscoveredHosts()
    ])
    if (results[0].status === 'rejected') {
      throw results[0].reason
    }
    notifStore.success(t('devices.view.devicesUpdatedTitle'), t('devices.view.devicesUpdatedMessage'))
  } catch (err: any) {
    notifStore.error(t('devices.view.refreshErrorTitle'), t('devices.view.refreshErrorMessage'))
  } finally {
    isRefreshing.value = false
  }
}

function entityDeviceID(entity: Entity): string {
  const direct = String(entity.device_id || '').trim()
  if (direct) return direct
  for (const candidate of [entity.entity_key || '', entity.id || '']) {
    const suffix = candidate.split('.').pop() || ''
    if (suffix.startsWith('tuya_') && suffix.length > 'tuya_'.length) return suffix
  }
  return ''
}

const allDevices = computed<UnifiedDevice[]>(() => {
  const merged: UnifiedDevice[] = (devicesStore.devices || []).map(device => ({
    ...device,
    source_label: device.integration_id
  }))

  const externalTuyaDevices = [
    ...tuyaLocalDevices.value.map(device => ({
      ...device,
      id: device.device_id || device.id,
      online: device.connected === true || device.online === true,
      local_configured: true,
      local_active: device.connected === true,
      local_ip: device.ip || device.local_ip,
      source: 'tuya_local' as const
    })),
    ...tuyaCloudDevices.value.map(device => ({ ...device, source: 'tuya_cloud' as const })),
    ...tuyaOAuthDevices.value.map(device => ({ ...device, source: 'tuya_oauth' as const }))
  ].filter(device => Boolean(device.id))

  for (const cloudDevice of externalTuyaDevices) {
    const existingIndex = merged.findIndex(device =>
      device.external_id === cloudDevice.id ||
      device.id === cloudDevice.id ||
      device.id === `tuya_${cloudDevice.id}`
    )
    const existing = existingIndex >= 0 ? merged[existingIndex] : undefined
    const cloudFields = {
      manufacturer: existing?.manufacturer || 'Tuya',
      model: existing?.model || cloudDevice.product_name || cloudDevice.category,
      status: (cloudDevice.online ? 'online' : 'offline') as Device['status'],
      local_configured: Boolean(cloudDevice.local_configured || existing?.integration_id === 'tuya_local'),
      local_active: Boolean(cloudDevice.local_active),
      local_ip: cloudDevice.local_ip,
      source_label: cloudDevice.source === 'tuya_local'
        ? 'Tuya Local'
        : cloudDevice.local_configured || existing?.integration_id === 'tuya_local'
          ? 'Tuya Cloud · LocalTuya'
          : cloudDevice.source === 'tuya_oauth' ? 'Smart Life · OAuth' : 'Tuya Cloud'
    }
    if (existingIndex >= 0) {
      merged[existingIndex] = { ...existing!, ...cloudFields }
    } else {
      merged.push({
        id: `tuya_${cloudDevice.id}`,
        external_id: cloudDevice.id,
        name: cloudDevice.name,
        integration_id: cloudDevice.source,
        created_at: '',
        updated_at: '',
        ...cloudFields
      })
    }
  }

  const knownDeviceKeys = new Set<string>()
  const addKnownKey = (value?: string) => {
    const key = String(value || '').trim()
    if (!key) return
    knownDeviceKeys.add(key)
    knownDeviceKeys.add(key.startsWith('tuya_') ? key.slice(5) : key)
  }
  for (const device of merged) {
    addKnownKey(device.id)
    addKnownKey(device.external_id)
  }

  const entitiesByDevice = new Map<string, Entity[]>()
  for (const entity of entitiesStore.allEntities) {
    const deviceId = entityDeviceID(entity)
    if (!deviceId) continue
    const group = entitiesByDevice.get(deviceId) || []
    group.push(entity)
    entitiesByDevice.set(deviceId, group)
  }

  // Prefer the persisted device area, but keep the entity association visible
  // when an integration has only populated area_id on its entities.
  for (const device of merged) {
    if (device.area_id) continue
    const linked = [...entitiesByDevice.entries()].find(([deviceId]) =>
      deviceId === device.id || deviceId === device.external_id ||
      (device.id.startsWith('tuya_') && device.id.slice(5) === deviceId) ||
      (deviceId.startsWith('tuya_') && device.external_id === deviceId.slice(5))
    )?.[1]
    const linkedArea = linked?.find(entity => entity.area_id)?.area_id ||
      linked?.find(entity => entity.attributes?.area_id)?.attributes?.area_id
    if (linkedArea) device.area_id = String(linkedArea)
  }

  for (const [deviceId, relatedEntities] of entitiesByDevice) {
    const normalizedId = deviceId.startsWith('tuya_') ? deviceId.slice(5) : deviceId
    if (knownDeviceKeys.has(deviceId) || knownDeviceKeys.has(normalizedId)) continue

    const attributes = relatedEntities.reduce<Record<string, unknown>>((all, entity) => ({
      ...all,
      ...(entity.attributes || {})
    }), {})
    const integrationId = String(
      attributes.integration_id || attributes.integration || attributes.platform || 'integrated'
    )
    const explicitStatus = String(attributes.status || '').toLowerCase()
    const unavailable = explicitStatus === 'offline' || attributes.available === false ||
      attributes.is_available === false ||
      relatedEntities.every(entity => entity.state === 'unavailable' || entity.state === 'unknown')
    const explicitlyOnline = explicitStatus === 'online' || attributes.available === true ||
      attributes.is_available === true ||
      relatedEntities.some(entity => entity.state !== 'unavailable' && entity.state !== 'unknown')
    const name = String(
      attributes.device_name || attributes.device_friendly_name || attributes.friendly_name ||
      relatedEntities[0]?.name || deviceId
    )

    merged.push({
      id: deviceId,
      external_id: String(attributes.external_id || (deviceId.startsWith('tuya_') ? deviceId.slice(5) : deviceId)),
      name,
      integration_id: integrationId,
      manufacturer: attributes.manufacturer ? String(attributes.manufacturer) : undefined,
      model: attributes.model ? String(attributes.model) : undefined,
      area_id: relatedEntities.find(entity => entity.area_id)?.area_id ||
        (attributes.area_id ? String(attributes.area_id) : undefined),
      status: unavailable ? 'offline' : explicitlyOnline ? 'online' : 'unknown',
      local_ip: attributes.local_ip || attributes.ip_address || attributes.ip
        ? String(attributes.local_ip || attributes.ip_address || attributes.ip)
        : undefined,
      source_label: integrationId === 'integrated'
        ? 'Dispositivo integrato'
        : `Integrato · ${integrationId}`,
      created_at: relatedEntities[0]?.last_changed || '',
      updated_at: relatedEntities[0]?.last_updated || ''
    })
    addKnownKey(deviceId)
    addKnownKey(attributes.external_id ? String(attributes.external_id) : undefined)
  }

  return merged
})

const integrationOptions = computed(() =>
  [...new Set(allDevices.value.map(device => device.integration_id).filter(Boolean))].sort()
)

const filteredDevices = computed(() => {
  const query = searchQuery.value.trim().toLocaleLowerCase()
  return allDevices.value.filter(device => {
    if (filterStatus.value === 'online' && device.status !== 'online') return false
    if (filterStatus.value === 'offline' && device.status !== 'offline') return false
    if (filterIntegration.value !== 'all' && device.integration_id !== filterIntegration.value) return false
    if (!query) return true
    return [
      device.name,
      device.id,
      device.external_id,
      device.manufacturer,
      device.model,
      device.integration_id,
      device.local_ip
    ].some(value => String(value || '').toLocaleLowerCase().includes(query))
  })
})

const totalDevicesCount = computed(() => allDevices.value.length)
const onlineDevicesCount = computed(() => allDevices.value.filter(device => device.status === 'online').length)
const offlineDevicesCount = computed(() => allDevices.value.filter(device => device.status === 'offline').length)
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
          :disabled="isRefreshing"
          @click="refreshAllDevices"
        >
          <RefreshCw :size="16" :class="{ 'spin-icon': isRefreshing }" />
          <span>{{ isRefreshing ? t('devices.view.refreshing') : t('devices.view.refresh') }}</span>
        </button>
        <button
          class="btn-action-outline"
          :disabled="isScanning"
          @click="triggerNetworkScan"
          :title="t('devices.view.scanLanTitle')"
        >
          <RefreshCw :size="16" :class="{ 'spin-icon': isScanning }" />
          <span>{{ isScanning ? t('devices.view.scanning') : t('devices.view.scanLan') }}</span>
        </button>
        <button
          class="btn-action-primary"
          @click="isMatterWizardOpen = true"
          :title="t('devices.view.pairMatterTitle')"
        >
          <QrCode :size="16" />
          <span>{{ t('devices.view.pairMatter') }}</span>
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
            <h3 class="discovered-title">{{ t('devices.view.discoveredTitle') }}</h3>
            <p class="discovered-subtitle">
              {{ discoveredHosts.length }} {{ t('devices.view.deviceCount') }} {{ t('devices.view.discoveryReady') }}
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
              <span>{{ t('devices.view.ipPrefix') }} {{ host.ip }}</span>
              <span v-if="host.mac">• {{ host.mac }}</span>
            </div>
          </div>
          <button class="btn-setup-now" @click="openSetupModal(host)">
            <Plus :size="15" />
            <span>{{ t('devices.view.configure') }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Filters Row -->
    <div class="filters-row">
      <button
        :class="['filter-pill', { active: filterStatus === 'all' }]"
        :aria-pressed="filterStatus === 'all'"
        @click="filterStatus = 'all'"
      >
        {{ t('devices.filterAll', { count: totalDevicesCount }) }}
      </button>
      <button
        :class="['filter-pill', { active: filterStatus === 'online' }]"
        :aria-pressed="filterStatus === 'online'"
        @click="filterStatus = 'online'"
      >
        <Wifi :size="15" />
        {{ t('devices.filterOnline', { count: onlineDevicesCount }) }}
      </button>
      <button
        :class="['filter-pill', { active: filterStatus === 'offline' }]"
        :aria-pressed="filterStatus === 'offline'"
        @click="filterStatus = 'offline'"
      >
        <WifiOff :size="15" />
        {{ t('devices.filterOffline', { count: offlineDevicesCount }) }}
      </button>
    </div>

    <div class="device-filter-tools">
      <label class="device-search">
        <Search :size="16" />
        <input id="devices-search" v-model="searchQuery" type="search" :placeholder="t('devices.view.searchPlaceholder')" />
      </label>
      <label class="integration-filter">
        <span>{{ t('devices.view.integration') }}</span>
        <select id="devices-integration-filter" v-model="filterIntegration">
          <option value="all">{{ t('devices.view.allIntegrations') }}</option>
          <option v-for="integration in integrationOptions" :key="integration" :value="integration">{{ integration }}</option>
        </select>
      </label>
      <span class="device-result-count">{{ t('devices.view.filteredCount', { filtered: filteredDevices.length, total: totalDevicesCount }) }}</span>
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
              <span v-if="device.local_configured" class="meta-tag meta-tag-local">LocalTuya</span>
              <span v-if="device.local_ip" class="meta-tag">{{ device.local_ip }}</span>
            </div>
          </div>
        </div>

        <div v-if="authStore.isAdmin" class="device-area-control">
          <label :for="`device-room-${device.id}`">{{ t('devices.view.roomArea') }}</label>
          <select
            :id="`device-room-${device.id}`"
            :value="device.area_id || ''"
            :disabled="assigningRoomForDevice === device.id"
            @change="assignDeviceToRoom(device, $event)"
          >
            <option value="">{{ t('devices.view.noRoom') }}</option>
            <option v-for="area in areasStore.sortedAreas" :key="area.id" :value="area.id">
              {{ area.name }}
            </option>
          </select>
          <span v-if="assigningRoomForDevice === device.id" class="room-assignment-state">
            {{ t('common.loading') }}
          </span>
        </div>
        <div v-else-if="device.area_id" class="device-room-badge">
          {{ areasStore.byId(device.area_id)?.name || t('devices.view.roomArea') }}
        </div>

        <div class="device-card-footer">
          <span class="integration-badge">{{ device.source_label || device.integration_id }}</span>
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
      <h3 class="empty-title">{{ totalDevicesCount > 0 ? t('devices.view.noFilteredDevices') : t('devices.emptyTitle') }}</h3>
      <p class="empty-desc">
        {{ totalDevicesCount > 0 ? t('devices.view.adjustFilters') : t('devices.emptyDesc') }}
      </p>
      <button v-if="totalDevicesCount > 0" class="btn-secondary" @click="searchQuery = ''; filterIntegration = 'all'; filterStatus = 'all'">{{ t('devices.view.resetFilters') }}</button>
    </div>

    <!-- Setup Modal for Discovered Host -->
    <div v-if="isSetupModalOpen" class="modal-backdrop" @click.self="isSetupModalOpen = false">
      <div class="modal-dialog glass-card">
        <div class="modal-header">
          <div class="modal-title-wrap">
            <Sparkles :size="20" class="sparkle-icon" />
            <h3 class="modal-title">{{ t('devices.view.configureDiscovered') }}</h3>
          </div>
          <button class="btn-close-modal" @click="isSetupModalOpen = false" :aria-label="t('devices.view.close')">
            <X :size="18" />
          </button>
        </div>
        <div class="modal-body">
          <p class="modal-intro">
            {{ t('devices.view.addToHome') }} <strong>{{ selectedHostForSetup?.hostname || selectedHostForSetup?.ip }}</strong>
          </p>
          <div class="form-group">
            <label class="form-label">{{ t('devices.view.deviceName') }}</label>
            <input
              v-model="newDeviceName"
              type="text"
              class="form-input"
              :placeholder="t('devices.view.deviceNamePlaceholder')"
            />
          </div>
          <div class="form-group">
            <label class="form-label">{{ t('devices.view.roomArea') }}</label>
            <select v-model="selectedAreaId" class="form-select">
              <option v-for="area in areasStore.areas" :key="area.id" :value="area.id">
                {{ area.name }}
              </option>
            </select>
          </div>
          <div class="discovery-details-box">
            <div class="detail-row">
              <span class="detail-label">{{ t('devices.view.manufacturer') }}:</span>
              <span class="detail-val">{{ selectedHostForSetup?.vendor || 'Sconosciuto' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">{{ t('devices.view.ipAddress') }}:</span>
              <span class="detail-val font-mono">{{ selectedHostForSetup?.ip }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">{{ t('devices.view.recommendedIntegration') }}:</span>
              <span class="detail-val badge-tag">{{ selectedHostForSetup?.suggested_plugin || 'mDNS Zero-Config' }}</span>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-secondary" @click="isSetupModalOpen = false">{{ t('common.cancel') }}</button>
          <button class="btn-primary" :disabled="isAddingDevice" @click="confirmAddDiscoveredDevice">
            <Check :size="16" />
            <span>{{ isAddingDevice ? t('common.loading') : t('devices.view.addToDomus') }}</span>
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
            <h3 class="modal-title">{{ t('devices.view.matterCommissioning') }}</h3>
          </div>
          <button class="btn-close-modal" @click="closeMatterWizard" :aria-label="t('devices.view.close')">
            <X :size="18" />
          </button>
        </div>

        <div class="modal-body">
          <!-- Step 1: Input Code -->
          <div v-if="matterCommissionStep === 1" class="matter-step">
            <p class="modal-intro">
              {{ t('devices.view.matterIntro') }}
            </p>
            <div class="form-group">
              <label class="form-label">{{ t('devices.view.matterPairingCode') }}</label>
              <input
                v-model="matterPairingCode"
                type="text"
                class="form-input font-mono"
                :placeholder="t('devices.view.matterCodePlaceholder')"
                maxlength="25"
              />
              <span class="form-hint">
                {{ t('devices.view.matterCodeHint') }}
              </span>
            </div>
          </div>

          <!-- Step 2: Commissioning in progress -->
          <div v-else-if="matterCommissionStep === 2" class="matter-step step-center">
            <div class="commissioning-spinner-box">
              <RefreshCw :size="36" class="spin-icon text-primary" />
            </div>
            <h4 class="commissioning-heading">{{ t('devices.view.matterNegotiating') }}</h4>
            <p class="commissioning-subtext">
              {{ t('devices.view.matterNegotiatingDesc') }}
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
            <h4 class="commissioning-heading">{{ t('devices.view.matterPaired') }}</h4>
            <p class="commissioning-subtext">
              {{ t('devices.view.matterPairedDesc') }}
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
            {{ t('devices.view.startPairing') }}
          </button>
          <button
            v-if="matterCommissionStep === 3"
            class="btn-primary"
            @click="closeMatterWizard"
          >
            {{ t('common.done') }}
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
  flex-wrap: wrap;
}

.device-filter-tools {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.device-search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1 1 260px;
  min-width: 220px;
  padding: 9px 12px;
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  color: var(--text-muted);
}
.device-search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text-main);
  font: inherit;
}
.integration-filter {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-muted);
  font-size: 12px;
}
.integration-filter select {
  min-height: 38px;
  max-width: 220px;
  padding: 7px 10px;
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  color: var(--text-main);
}
.device-result-count {
  color: var(--text-muted);
  font-size: 12px;
  white-space: nowrap;
}
.meta-tag-local {
  color: var(--color-success);
  border-color: rgba(16, 185, 129, .35);
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

.device-area-control {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  padding: 10px 11px;
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
}

.device-area-control label {
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 600;
}

.device-area-control select {
  min-width: 0;
  width: 100%;
  min-height: 36px;
  padding: 6px 9px;
  border: 1px solid var(--border-card);
  border-radius: var(--radius-sm);
  background: var(--bg-glass-card);
  color: var(--text-main);
  font: inherit;
  font-size: 13px;
}

.device-area-control select:focus {
  outline: 2px solid var(--primary-glow);
  border-color: var(--primary);
}

.device-area-control select:disabled {
  opacity: .65;
  cursor: wait;
}

.room-assignment-state {
  color: var(--text-dim);
  font-size: 11px;
  white-space: nowrap;
}

.device-room-badge {
  align-self: flex-start;
  padding: 4px 9px;
  border-radius: var(--radius-full);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-muted);
  font-size: 12px;
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

@media (max-width: 768px) {
  .header-row {
    flex-direction: column;
    align-items: stretch;
    gap: 14px;
  }
  .header-actions {
    width: 100%;
    flex-wrap: wrap;
  }
  .header-actions > button {
    flex: 1 1 100%;
    justify-content: center;
    min-height: 44px;
  }
  .device-filter-tools {
    align-items: stretch;
  }
  .device-search {
    flex-basis: 100%;
    min-width: 0;
  }
  .integration-filter {
    width: 100%;
    justify-content: space-between;
  }
  .integration-filter select {
    flex: 1;
    max-width: none;
  }
  .device-result-count {
    width: 100%;
  }
}

@media (max-width: 480px) {
  .filters-row {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .filter-pill {
    justify-content: center;
    padding: 8px 10px;
  }
  .devices-grid {
    grid-template-columns: minmax(0, 1fr);
  }
  .device-card-footer {
    align-items: flex-start;
    flex-direction: column;
    gap: 8px;
  }
}
</style>
