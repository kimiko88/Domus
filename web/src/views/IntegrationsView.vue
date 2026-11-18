<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClient } from '../api/client'
import { useNotificationsStore } from '../stores/notifications'
import {
  Network,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  Wifi,
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
  Activity,
  QrCode,
  Key,
  ExternalLink
} from '@lucide/vue'
import { soundEffects } from '../utils/audioSynthesizer'
import { TUYA_THERMOSTAT_SETPOINTS, decodeTuyaThermostatSetpoint, encodeTuyaThermostatSetpoint } from '../utils/tuyaThermostatTemperature'
import type { NetworkDiscoveredHost, NetworkLeakAlert, NetworkGuardianLeaksResponse } from '../api/types'
import FirmwareFlasherModal from '../components/modals/FirmwareFlasherModal.vue'

const { t, locale } = useI18n()

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

// --- Official Tuya Cloud OpenAPI & Local Handover ---
interface TuyaCloudDevice {
  id: string
  name: string
  local_key?: string
  ip?: string
  product_name?: string
  category?: string
  online: boolean
  status: { code: string; value: unknown }[]
  local_configured?: boolean
  local_active?: boolean
  local_ip?: string
}

interface TuyaDeviceFunction {
  code: string
  name: string
  desc?: string
  type: string
  values: string
}

const tuyaCloudDevices = ref<TuyaCloudDevice[]>([])
const tuyaCloudLoading = ref(false)
const tuyaCloudError = ref('')
const tuyaCloudCommanding = ref<string | null>(null)
const tuyaSyncing = ref(false)
const tuyaDeviceFilter = ref<'all' | 'online' | 'offline' | 'selected' | 'unselected'>('all')
const tuyaSearchQuery = ref('')
const selectedTuyaDeviceIds = ref<string[]>([])
const tuyaSelectionInitialized = ref(false)
const tuyaDeviceFunctions = ref<Record<string, TuyaDeviceFunction[]>>({})
const tuyaFunctionPanels = ref<Record<string, boolean>>({})
const tuyaFunctionsLoading = ref<string | null>(null)
const tuyaFunctionInputs = reactive<Record<string, string>>({})
const tuyaFunctionBusy = ref<string | null>(null)
const tuyaTemperatureInputs = reactive<Record<string, string>>({})

const tuyaSelectedCount = computed(() =>
  tuyaCloudDevices.value.filter(device => selectedTuyaDeviceIds.value.includes(device.id)).length
)
const tuyaUnselectedCount = computed(() => tuyaCloudDevices.value.length - tuyaSelectedCount.value)
const tuyaOnlineCount = computed(() => tuyaCloudDevices.value.filter(device => device.online).length)
const tuyaOfflineCount = computed(() => tuyaCloudDevices.value.length - tuyaOnlineCount.value)

const filteredTuyaCloudDevices = computed(() => {
  const query = tuyaSearchQuery.value.trim().toLocaleLowerCase()
  return tuyaCloudDevices.value.filter(device => {
    const selected = selectedTuyaDeviceIds.value.includes(device.id)
    if (tuyaDeviceFilter.value === 'online' && !device.online) return false
    if (tuyaDeviceFilter.value === 'offline' && device.online) return false
    if (tuyaDeviceFilter.value === 'selected' && !selected) return false
    if (tuyaDeviceFilter.value === 'unselected' && selected) return false
    if (!query) return true
    return [
      device.name,
      device.id,
      device.product_name,
      device.category,
      device.local_ip
    ].some(value => String(value || '').toLocaleLowerCase().includes(query))
  })
})

function selectVisibleTuyaDevices() {
  const next = new Set(selectedTuyaDeviceIds.value)
  for (const device of filteredTuyaCloudDevices.value) next.add(device.id)
  selectedTuyaDeviceIds.value = [...next]
}

function deselectVisibleTuyaDevices() {
  const visible = new Set(filteredTuyaCloudDevices.value.map(device => device.id))
  selectedTuyaDeviceIds.value = selectedTuyaDeviceIds.value.filter(id => !visible.has(id))
}

function tuyaFunctionKey(deviceId: string, code: string) {
  return `${deviceId}:${code}`
}

function tuyaBooleanState(value: unknown): boolean | undefined {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') {
    if (value === 1) return true
    if (value === 0) return false
    return undefined
  }
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    if (['true', '1', 'on', 'enabled', 'active', 'yes'].includes(normalized)) return true
    if (['false', '0', 'off', 'disabled', 'inactive', 'no'].includes(normalized)) return false
  }
  return undefined
}

function tuyaFunctionValues(fn: TuyaDeviceFunction): Record<string, unknown> {
  try {
    const parsed: unknown = JSON.parse(fn.values || '{}')
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? parsed as Record<string, unknown>
      : {}
  } catch {
    return {}
  }
}

function isNumericTuyaFunction(fn: TuyaDeviceFunction) {
  return ['integer', 'number', 'float', 'double'].includes(fn.type.toLowerCase())
}

function tuyaFunctionScale(fn: TuyaDeviceFunction) {
  const scale = Number(tuyaFunctionValues(fn).scale ?? 0)
  return Number.isInteger(scale) && scale >= 0 && scale <= 4 ? scale : 0
}

function tuyaTemperatureFunction(deviceId: string) {
  return (tuyaDeviceFunctions.value[deviceId] || []).find(fn => {
    const code = fn.code.trim().toLowerCase()
    return ['temp_set', 'target_temp', 'target_temperature'].includes(code)
  })
}

async function loadTuyaTemperatureFunction(deviceId: string): Promise<TuyaDeviceFunction | undefined> {
  const cached = tuyaTemperatureFunction(deviceId)
  if (cached) return cached
  try {
    const result = await apiClient.get<{ functions: TuyaDeviceFunction[] }>(
      `/api/v1/integrations/tuya-cloud/devices/${encodeURIComponent(deviceId)}/functions`
    )
    tuyaDeviceFunctions.value[deviceId] = result.functions || []
    return tuyaTemperatureFunction(deviceId)
  } catch {
    return undefined
  }
}

function isTuyaClimateDevice(device: TuyaCloudDevice) {
  const category = (device.category || '').toLowerCase()
  const label = `${device.name} ${device.product_name || ''}`.toLowerCase()
  return category === 'wk' || category.includes('climate') || category.includes('thermostat') ||
    label.includes('thermostat') || label.includes('termostato')
}

function canControlLocalTuya(device: TuyaCloudDevice) {
  return Boolean(device.local_configured)
}

interface TuyaCloudConfig {
  configured: boolean
  client_id: string
  client_id_masked: string
  uid: string
  endpoint: string
  source: string
}

const tuyaConfig = ref<TuyaCloudConfig>({
  configured: false,
  client_id: '',
  client_id_masked: '',
  uid: '',
  endpoint: 'https://openapi.tuyaeu.com',
  source: 'none'
})

const tuyaConfigForm = reactive({
  client_id: '',
  client_secret: '',
  uid: '',
  endpoint: 'https://openapi.tuyaeu.com'
})

const isEditingTuyaConfig = ref(false)
const tuyaConfigSaving = ref(false)

// Maintained for backward compatibility with mock tests
interface TuyaOAuthStatus { connected: boolean; expires_at?: string }
const tuyaOAuthStatus = ref<TuyaOAuthStatus>({ connected: false })
const tuyaOAuthDevices = ref<TuyaCloudDevice[]>([])
const tuyaOAuthLoading = ref(false)

async function fetchTuyaConfig() {
  try {
    const res = await apiClient.get<TuyaCloudConfig>('/api/v1/integrations/tuya-cloud/config')
    if (res) {
      tuyaConfig.value = res
      if (res.configured) {
        tuyaConfigForm.client_id = res.client_id
        tuyaConfigForm.uid = res.uid
        tuyaConfigForm.endpoint = res.endpoint || 'https://openapi.tuyaeu.com'
        await fetchTuyaCloudDevices()
      }
    }
  } catch (err: any) {
    console.warn('Failed to load Tuya config:', err)
  }
}

async function saveTuyaConfig() {
  tuyaConfigSaving.value = true
  try {
    const res = await apiClient.post<{ success: boolean; message: string }>('/api/v1/integrations/tuya-cloud/config', {
      client_id: tuyaConfigForm.client_id,
      client_secret: tuyaConfigForm.client_secret,
      uid: tuyaConfigForm.uid,
      endpoint: tuyaConfigForm.endpoint
    })
    notificationsStore.success(t('integrations.view.messages.tuyaConnectedTitle'), t('integrations.view.messages.tuyaConnectedDesc'))
    soundEffects.playSuccessChime()
    isEditingTuyaConfig.value = false
    await fetchTuyaConfig()
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.tuyaConfigErrorTitle'), t('integrations.view.messages.tuyaConfigErrorDesc'))
  } finally {
    tuyaConfigSaving.value = false
  }
}

async function disconnectTuyaConfig() {
  try {
    await apiClient.delete('/api/v1/integrations/tuya-cloud/config')
    notificationsStore.success(t('integrations.view.messages.integrationRemovedTitle'), t('integrations.view.messages.integrationRemovedDesc'))
    tuyaConfig.value = {
      configured: false,
      client_id: '',
      client_id_masked: '',
      uid: '',
      endpoint: 'https://openapi.tuyaeu.com',
      source: 'none'
    }
    tuyaConfigForm.client_secret = ''
    tuyaCloudDevices.value = []
    selectedTuyaDeviceIds.value = []
    tuyaSelectionInitialized.value = false
    tuyaSearchQuery.value = ''
    isEditingTuyaConfig.value = false
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.disconnectErrorTitle'), t('integrations.view.messages.disconnectErrorDesc'))
  }
}

async function fetchTuyaOAuthStatus() {
  await fetchTuyaConfig()
}
async function fetchTuyaOAuthDevices() {
  await fetchTuyaCloudDevices()
}
async function syncTuyaOAuthToLocal() {
  await syncTuyaCloudToLocal()
}
async function connectTuyaAccount() {
  isEditingTuyaConfig.value = true
}
async function disconnectTuyaAccount() {
  await disconnectTuyaConfig()
}

async function fetchTuyaCloudDevices(reconcileSelection = false) {
  tuyaCloudLoading.value = true
  tuyaCloudError.value = ''
  try {
    const result = await apiClient.get<{ devices: TuyaCloudDevice[] }>('/api/v1/integrations/tuya-cloud/devices')
    tuyaCloudDevices.value = result.devices || []
    // Keep unsaved checkbox changes while refreshing status. Reconcile from the
    // server on the first load and immediately after applying the selection.
    if (!tuyaSelectionInitialized.value || reconcileSelection) {
      selectedTuyaDeviceIds.value = tuyaCloudDevices.value
        .filter(device => device.local_configured)
        .map(device => device.id)
      tuyaSelectionInitialized.value = true
    } else {
      const knownIds = new Set(tuyaCloudDevices.value.map(device => device.id))
      selectedTuyaDeviceIds.value = selectedTuyaDeviceIds.value.filter(id => knownIds.has(id))
    }
    for (const device of tuyaCloudDevices.value) {
      if (!isTuyaClimateDevice(device)) continue
      const target = (device.status || []).find(point => /temp.*(set|target)|target.*temp/i.test(point.code))
      const raw = typeof target?.value === 'number' ? target.value : Number(target?.value)
      const temperatureFunction = await loadTuyaTemperatureFunction(device.id)
      const scale = temperatureFunction ? tuyaFunctionScale(temperatureFunction) : 1
      const normalizedRaw = Number.isFinite(raw) && target
        ? raw > 60 ? raw / Math.pow(10, scale) : raw
        : Number.NaN
      const thermostatDegrees = decodeTuyaThermostatSetpoint(normalizedRaw)

      if (thermostatDegrees !== undefined) {
        tuyaTemperatureInputs[device.id] = String(thermostatDegrees)
      } else if (tuyaTemperatureInputs[device.id] === undefined) {
        tuyaTemperatureInputs[device.id] = ''
      }
    }
  } catch (err: any) {
    tuyaCloudError.value = t('integrations.view.tuyaCloudUnavailable')
  } finally {
    tuyaCloudLoading.value = false
  }
}

async function syncTuyaCloudToLocal() {
  tuyaSyncing.value = true
  try {
    const res = await apiClient.post<{ success: boolean; synced_count: number; message: string }>(
      '/api/v1/integrations/tuya-cloud/sync-local',
      { device_ids: selectedTuyaDeviceIds.value }
    )
    notificationsStore.success(t('integrations.view.messages.localSelectionUpdatedTitle'), t('integrations.view.messages.localSelectionUpdatedDesc', { count: res.synced_count }))
    await fetchTuyaCloudDevices(true)
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.localSyncErrorTitle'), t('integrations.view.messages.localSyncErrorDesc'))
  } finally {
    tuyaSyncing.value = false
  }
}

async function toggleTuyaDeviceFunctions(device: TuyaCloudDevice) {
  if (tuyaFunctionPanels.value[device.id]) {
    tuyaFunctionPanels.value[device.id] = false
    return
  }
  tuyaFunctionPanels.value[device.id] = true
  if (tuyaDeviceFunctions.value[device.id]) return

  tuyaFunctionsLoading.value = device.id
  try {
    const result = await apiClient.get<{ functions: TuyaDeviceFunction[] }>(
      `/api/v1/integrations/tuya-cloud/devices/${encodeURIComponent(device.id)}/functions`
    )
    tuyaDeviceFunctions.value[device.id] = result.functions || []
    for (const fn of tuyaDeviceFunctions.value[device.id]) {
      const key = tuyaFunctionKey(device.id, fn.code)
      const status = (device.status || []).find(point => point.code === fn.code)
      const values = tuyaFunctionValues(fn)
      const current = status?.value
      if (current !== undefined) {
        const numeric = Number(current)
        tuyaFunctionInputs[key] = isNumericTuyaFunction(fn) && Number.isFinite(numeric) && /temp/i.test(fn.code)
          ? String(numeric / Math.pow(10, tuyaFunctionScale(fn)))
          : String(current)
      } else {
        tuyaFunctionInputs[key] = String(values.min ?? '')
      }
    }
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.tuyaCommandsUnavailableTitle'), t('integrations.view.messages.tuyaCommandsUnavailableDesc'))
    tuyaFunctionPanels.value[device.id] = false
  } finally {
    tuyaFunctionsLoading.value = null
  }
}

async function executeTuyaFunction(device: TuyaCloudDevice, fn: TuyaDeviceFunction) {
  const key = tuyaFunctionKey(device.id, fn.code)
  tuyaFunctionBusy.value = key
  try {
    const current = (device.status || []).find(point => point.code === fn.code)?.value
    let value: unknown
    const type = fn.type.trim().toLowerCase()
    if (type === 'boolean') {
      const currentBoolean = tuyaBooleanState(current)
      if (currentBoolean === undefined) {
        notificationsStore.warning(
          t('integrations.view.messages.stateUnavailableTitle'),
          t('integrations.view.messages.stateUnavailableDesc', { name: fn.name || fn.code })
        )
        return
      }
      value = !currentBoolean
    } else if (isNumericTuyaFunction(fn)) {
      const parsed = Number(tuyaFunctionInputs[key])
      if (!Number.isFinite(parsed)) {
        notificationsStore.warning(t('integrations.view.messages.invalidValueTitle'), t('integrations.view.messages.invalidValueDesc'))
        return
      }
      value = /temp/i.test(fn.code)
        ? Math.round(parsed * Math.pow(10, tuyaFunctionScale(fn)))
        : parsed
    } else {
      value = tuyaFunctionInputs[key] ?? ''
    }

    const code = fn.code.toLowerCase()
    let sentLocally = false
    if (device.local_configured) {
      const localPath = `/api/v1/integrations/tuya-local/devices/${encodeURIComponent(device.id)}/command`
      if ((code.includes('switch') || code.includes('power')) && typeof value === 'boolean') {
        await apiClient.post(localPath, { command: value ? 'turn_on' : 'turn_off' })
        sentLocally = true
      } else if (['temp_set', 'target_temp', 'target_temperature'].includes(code) && typeof value === 'number') {
        await apiClient.post(localPath, {
          command: 'set_temperature',
          params: {
            temperature: value / Math.pow(10, tuyaFunctionScale(fn)),
            scale: tuyaFunctionScale(fn)
          }
        })
        sentLocally = true
      } else if (code === 'mode' || code === 'work_mode' || code.endsWith('_mode')) {
        if (typeof value === 'string' && ['auto', 'automatic', 'manual'].includes(value.toLowerCase())) {
          await apiClient.post(localPath, { command: 'set_mode', params: { mode: value } })
          sentLocally = true
        } else if (typeof value === 'number' && (value === 0 || value === 1)) {
          await apiClient.post(localPath, { command: 'set_dp', params: { dp_id: 4, value } })
          sentLocally = true
        }
      }
    }
    if (!sentLocally) {
      await apiClient.post(
        `/api/v1/integrations/tuya-cloud/devices/${encodeURIComponent(device.id)}/commands`,
        { commands: [{ code: fn.code, value }] }
      )
    }
    notificationsStore.success(t(sentLocally ? 'integrations.view.messages.commandSentLanTitle' : 'integrations.view.messages.commandSentTitle'), t('integrations.view.messages.commandSentDesc', { name: device.name, command: fn.name || fn.code }))
    await fetchTuyaCloudDevices()
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.tuyaCommandErrorTitle'), t('integrations.view.messages.communicationError'))
  } finally {
    tuyaFunctionBusy.value = null
  }
}

async function setTuyaClimateMode(device: TuyaCloudDevice, mode: 'auto' | 'manual') {
  if (!device.local_configured) return
  tuyaCloudCommanding.value = `${device.id}:dp4`
  try {
    const result = await apiClient.post<{
      acknowledged?: boolean
      confirmed?: boolean
      message?: string
    }>(
      `/api/v1/integrations/tuya-local/devices/${encodeURIComponent(device.id)}/command`,
      { command: 'set_dp', params: { dp_id: 4, value: mode === 'auto' ? 0 : 1 } }
    )
    notificationsStore.success(t('integrations.view.messages.thermostatModeUpdatedTitle'), t(mode === 'auto' ? 'integrations.view.messages.thermostatAuto' : 'integrations.view.messages.thermostatManual'))
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.thermostatCommandErrorTitle'), t('integrations.view.messages.lanCommunicationError'))
  } finally {
    tuyaCloudCommanding.value = null
  }
}

async function setTuyaTemperature(device: TuyaCloudDevice) {
  const temperature = Number(tuyaTemperatureInputs[device.id])
  const encodedTemperature = encodeTuyaThermostatSetpoint(temperature)
  if (encodedTemperature === undefined) {
    notificationsStore.warning(
      t('integrations.view.messages.unsupportedTemperatureTitle'),
      t('integrations.view.messages.unsupportedTemperatureDesc')
    )
    return
  }

  tuyaCloudCommanding.value = `${device.id}:temperature`
  try {
    if (device.local_configured) {
      const temperatureFunction = await loadTuyaTemperatureFunction(device.id)
      await apiClient.post<{
        acknowledged?: boolean
        confirmed?: boolean
        message?: string
      }>(
        `/api/v1/integrations/tuya-local/devices/${encodeURIComponent(device.id)}/command`,
        {
          command: 'set_temperature',
          params: {
            temperature: encodedTemperature,
            ...(temperatureFunction ? { scale: tuyaFunctionScale(temperatureFunction) } : {})
          }
        }
      )
    } else {
      const fn = tuyaTemperatureFunction(device.id)
      if (!fn) {
        notificationsStore.warning(t('integrations.view.messages.temperatureCommandMissingTitle'), t('integrations.view.messages.temperatureCommandMissingDesc'))
        return
      }
      if (!device.online) {
        notificationsStore.warning(t('integrations.view.messages.deviceOfflineTitle'), t('integrations.view.messages.deviceOfflineDesc'))
        return
      }
      await apiClient.post(
        `/api/v1/integrations/tuya-cloud/devices/${encodeURIComponent(device.id)}/commands`,
        {
          commands: [{
            code: fn.code,
            value: Math.round(encodedTemperature * Math.pow(10, tuyaFunctionScale(fn)))
          }]
        }
      )
    }
    notificationsStore.success(
      t('integrations.view.messages.setpointUpdatedTitle'),
      t('integrations.view.messages.setpointUpdatedDesc', { name: device.name, temperature: temperature.toLocaleString(locale.value), encoded: encodedTemperature.toLocaleString(locale.value) })
    )
    await fetchTuyaCloudDevices()
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.temperatureSetErrorTitle'), t('integrations.view.messages.communicationError'))
  } finally {
    tuyaCloudCommanding.value = null
  }
}

function tuyaTemperatureEncodedPreview(deviceId: string): string {
  const encoded = encodeTuyaThermostatSetpoint(Number(tuyaTemperatureInputs[deviceId]))
  return encoded === undefined ? '—' : encoded.toLocaleString(locale.value)
}

async function toggleTuyaStatus(device: TuyaCloudDevice, point: { code: string; value: unknown }) {
  if (typeof point.value !== 'boolean') return
  const key = `${device.id}:${point.code}`
  tuyaCloudCommanding.value = key
  try {
    if (device.local_configured) {
      await apiClient.post(`/api/v1/integrations/tuya-local/devices/${encodeURIComponent(device.id)}/command`, {
        command: point.value ? 'turn_off' : 'turn_on'
      })
      point.value = !point.value
      notificationsStore.success(t('integrations.view.messages.directLanCommandTitle'), t('integrations.view.messages.directLanCommandDesc', { name: device.name }))
    } else {
      await apiClient.post(`/api/v1/integrations/tuya-cloud/devices/${encodeURIComponent(device.id)}/commands`, {
        commands: [{ code: point.code, value: !point.value }]
      })
      point.value = !point.value
      notificationsStore.success(t('integrations.view.messages.commandSentTitle'), t('integrations.view.messages.commandSentDesc', { name: device.name, command: point.code }))
      await fetchTuyaCloudDevices()
    }
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.smartLifeCommandErrorTitle'), t('integrations.view.messages.communicationError'))
  } finally {
    tuyaCloudCommanding.value = null
  }
}

async function toggleTuyaCloudStatus(device: TuyaCloudDevice, point: { code: string; value: unknown }) {
  await toggleTuyaStatus(device, point)
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
        t('integrations.view.messages.scanFoundDevicesDesc', { count: res.devices.length })
      )
      manualIp.value = ''
    } else if (target || manualIp.value) {
      notificationsStore.warning(t('integrations.view.messages.noDeviceTitle'), t('integrations.view.messages.noDeviceDesc'))
    }
    await fetchDiscovered()
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.scanErrorTitle'), t('integrations.view.messages.scanErrorDesc'))
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
      t('integrations.view.messages.devicePairedTitle'),
      t('integrations.view.messages.devicePairedDesc', { name: dev.name })
    )
    await Promise.all([fetchIntegrations(), fetchDiscovered()])
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.pairErrorTitle'), t('integrations.view.messages.pairErrorDesc'))
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
    notificationsStore.error(t('common.error'), t('integrations.reconnectError'))
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
    notificationsStore.success(t('integrations.view.messages.arpScanCompleteTitle'), t('integrations.view.messages.arpScanCompleteDesc', { count: res.count }))
    soundEffects.playSuccessChime()
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.networkScanErrorTitle'), t('integrations.view.messages.networkScanErrorDesc'))
  } finally {
    scanning.value = false
  }
}

async function probeHostPorts(ip: string) {
  probingIp.value = ip
  soundEffects.playClick()
  try {
    const res = await apiClient.post<{ ip: string; open_ports: number[]; host: NetworkDiscoveredHost }>('/api/v1/network/probe', { ip })
    notificationsStore.info(t('integrations.view.messages.portProbeCompleteTitle'), t('integrations.view.messages.portProbeCompleteDesc', { ip, ports: res.open_ports.join(', ') }))
    await fetchGuardianData()
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.portProbeErrorTitle'), t('integrations.view.messages.portProbeErrorDesc'))
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
      reason: reason || t('integrations.view.messages.manualIsolationReason')
    })
    notificationsStore.warning(t('integrations.view.messages.deviceQuarantinedTitle'), t('integrations.view.messages.deviceQuarantinedDesc', { mac }))
    soundEffects.playSuccessChime()
    await fetchGuardianData()
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.quarantineErrorTitle'), t('integrations.view.messages.quarantineErrorDesc'))
  } finally {
    quarantiningMac.value = null
  }
}

async function unblockDevice(mac: string, ip: string) {
  quarantiningMac.value = mac
  soundEffects.playClick()
  try {
    await apiClient.post('/api/v1/network/unblock', { mac, ip })
    notificationsStore.success(t('integrations.view.messages.deviceUnblockedTitle'), t('integrations.view.messages.deviceUnblockedDesc', { mac }))
    soundEffects.playSuccessChime()
    await fetchGuardianData()
  } catch (err: any) {
    notificationsStore.error(t('integrations.view.messages.unblockErrorTitle'), t('integrations.view.messages.unblockErrorDesc'))
  } finally {
    quarantiningMac.value = null
  }
}

onMounted(() => {
  fetchIntegrations()
  fetchCatalog()
  fetchDiscovered()
  fetchGuardianData()
  fetchTuyaConfig()
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
          <span>{{ t('integrations.view.lanGuardianTitle') }}</span>
          <span v-if="networkLeaks.length > 0" class="counter-badge alert-pulse">{{ networkLeaks.length }}</span>
        </button>

        <button
          :class="['tab-btn', { active: activeTab === 'cloud' }]"
          @click="activeTab = 'cloud'; fetchTuyaConfig()"
        >
          <Globe :size="16" />
          <span>{{ t('integrations.view.smartLifeCloud') }}</span>
          <span v-if="tuyaCloudDevices.length > 0" class="counter-badge">{{ tuyaCloudDevices.length }}</span>
        </button>
      </div>
    </div>
  </div>

    <!-- TAB 1: ACTIVE INTEGRATIONS -->
    <div v-if="activeTab === 'active'" class="tab-content">
      <div class="actions-bar">
        <span class="bar-info">{{ t('integrations.view.bridgeInfo') }}</span>
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
            <h3 class="disc-title">{{ t('integrations.view.localScanTitle') }}</h3>
            <p class="disc-sub">{{ t('integrations.view.localScanDesc') }}</p>
          </div>
        </div>

        <div class="discovery-inputs">
          <div class="manual-probe-box">
            <input
              v-model="manualIp"
              type="text"
              :placeholder="t('integrations.view.ipAddressPlaceholder')"
              class="ip-input"
              @keyup.enter="scanNetwork()"
            />
            <button class="btn-probe" :disabled="scanning || !manualIp.trim()" @click="scanNetwork()">
              <Plus :size="15" />
              <span>{{ t('integrations.view.ipProbe') }}</span>
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
              <span>{{ t('integrations.view.localOnly') }}</span>
            </span>
          </div>

          <div class="disc-card-body">
            <h4 class="disc-name">{{ dev.name }}</h4>
            <div class="disc-meta">
              <span class="disc-ip">{{ t('integrations.view.ipLabel') }}: {{ dev.ip }}:{{ dev.port }}</span>
              <span v-if="dev.model" class="disc-model">{{ dev.model }}</span>
              <span v-if="dev.firmware" class="disc-fw">{{ t('integrations.view.firmwareLabel') }}: {{ dev.firmware }}</span>
            </div>

            <!-- Entities preview -->
            <div v-if="dev.suggested_entities && dev.suggested_entities.length > 0" class="entities-preview">
              <span class="preview-title">{{ t('integrations.view.oneClickEntities') }}</span>
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
        <h3 class="empty-title">{{ t('integrations.view.pendingPairingEmpty') }}</h3>
        <p class="empty-desc">
          {{ t('integrations.view.pendingPairingInstructions') }}
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
              <span>{{ t('integrations.view.zeroCloud') }}</span>
            </div>
          </div>

          <p class="brand-desc">{{ entry.description }}</p>

          <div class="brand-specs">
            <div class="spec-row">
              <span class="spec-label">{{ t('integrations.view.protocolLabel') }}:</span>
              <span class="spec-val">{{ entry.protocol }}</span>
            </div>
            <div class="spec-row">
              <span class="spec-label">{{ t('integrations.view.quickGuide') }}:</span>
              <span class="spec-val">{{ entry.pairing_guide }}</span>
            </div>
          </div>

          <div class="models-box">
            <span class="models-title">{{ t('integrations.view.supportedModels') }}:</span>
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
            <h3 class="guardian-title">{{ t('integrations.view.networkGuardianTitle') }}</h3>
            <p class="guardian-subtitle">
              {{ t('integrations.view.networkGuardianDescription') }}
            </p>
          </div>
        </div>

        <div class="guardian-stats-row">
          <div class="stat-pill glass-card">
            <Globe :size="16" class="stat-icon-blue" />
            <div class="stat-info">
              <span class="stat-value">{{ networkHosts.length }}</span>
              <span class="stat-label">{{ t('integrations.view.detectedLanHosts') }}</span>
            </div>
          </div>

          <div class="stat-pill glass-card">
            <Lock :size="16" class="stat-icon-amber" />
            <div class="stat-info">
              <span class="stat-value">{{ isolatedMacs.length }}</span>
              <span class="stat-label">{{ t('integrations.view.quarantinedDevices') }}</span>
            </div>
          </div>

          <div class="stat-pill glass-card">
            <AlertTriangle :size="16" class="stat-icon-red" />
            <div class="stat-info">
              <span class="stat-value">{{ networkLeaks.length }}</span>
              <span class="stat-label">{{ t('integrations.view.wanLeaksCaught') }}</span>
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
            <h4 class="subhead-title">{{ t('integrations.view.externalAttemptsTitle') }}</h4>
          </div>
          <span class="subhead-badge" v-if="networkLeaks.length > 0">
            {{ networkLeaks.length }} {{ t('integrations.view.anomaliesRegistered') }}
          </span>
        </div>

        <div v-if="networkLeaks.length === 0" class="empty-guardian-state glass-card">
          <ShieldCheck :size="36" class="text-success" />
          <h4>{{ t('integrations.view.noLeaksTitle') }}</h4>
          <p>{{ t('integrations.view.noLeaksDesc') }}</p>
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
                <span>{{ t('integrations.view.unlock') }}</span>
              </button>
              <button
                v-else
                class="btn-quarantine"
                :disabled="quarantiningMac === leak.source_mac"
                @click="quarantineDevice(leak.source_mac || '', leak.source_ip, leak.reason)"
              >
                <Lock :size="15" />
                <span>{{ t('integrations.view.quarantine') }}</span>
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
            <h4 class="subhead-title">{{ t('integrations.view.hostMapTitle') }}</h4>
          </div>
          <span class="subhead-badge">{{ networkHosts.length }} {{ t('integrations.view.host') }}</span>
        </div>

        <div v-if="networkHosts.length === 0" class="empty-guardian-state glass-card">
          <RefreshCw :size="32" class="animate-spin text-muted" />
          <p>{{ t('integrations.view.arpCacheEmpty') }}</p>
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
                <span class="host-vendor">{{ host.vendor || t('integrations.view.genericLanHost') }}</span>
                <span class="host-ip">{{ host.ip }}</span>
              </div>
              <div v-if="host.mac && isolatedMacs.includes(host.mac.toLowerCase())" class="badge-quarantined">
                <Lock :size="12" />
                <span>{{ t('integrations.view.isolated') }}</span>
              </div>
              <div v-else-if="host.is_iot" class="badge-iot">
                <Zap :size="12" />
                <span>{{ t('integrations.view.iot') }}</span>
              </div>
            </div>

            <div class="host-specs">
              <div class="spec-line">
                <span class="spec-key">{{ t('integrations.view.mac') }}:</span>
                <code class="spec-code">{{ host.mac }}</code>
              </div>
              <div v-if="host.suggested_plugin" class="spec-line">
                <span class="spec-key">{{ t('integrations.view.plugin') }}:</span>
                <span class="spec-val-plugin">{{ host.suggested_plugin }}</span>
              </div>
              <div class="spec-line">
                <span class="spec-key">{{ t('integrations.view.openPorts') }}:</span>
                <span v-if="host.open_ports && host.open_ports.length > 0" class="ports-list">
                  <span v-for="p in host.open_ports" :key="p" class="port-tag">{{ p }}</span>
                </span>
                <span v-else class="text-muted-xs">{{ t('integrations.view.noRecentProbe') }}</span>
              </div>
            </div>

            <div class="host-footer">
              <button
                class="btn-host-probe"
                :disabled="probingIp === host.ip"
                @click="probeHostPorts(host.ip)"
                :title="t('integrations.view.runTcpProbe')"
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
                <span>{{ t('integrations.view.unlockButton') }}</span>
              </button>
              <button
                v-else
                class="btn-host-quarantine"
                :disabled="quarantiningMac === host.mac"
                @click="quarantineDevice(host.mac, host.ip, 'Isolamento manuale da mappa host')"
              >
                <Lock :size="14" />
                <span>{{ t('integrations.view.isolate') }}</span>
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
          <div class="icon-chip"><Zap :size="20" /></div>
          <div>
            <h3 class="disc-title">{{ t('integrations.view.tuyaSetupTitle') }}</h3>
            <p class="disc-sub">{{ t('integrations.view.tuyaSetupDesc') }}</p>
          </div>
        </div>
        <div class="discovery-inputs">
          <button class="btn-scan" :disabled="tuyaCloudLoading || !tuyaConfig.configured" @click="() => fetchTuyaCloudDevices()">
            <RefreshCw :size="15" :class="{ 'animate-spin': tuyaCloudLoading }" />
            <span>{{ t('integrations.view.refreshDevices') }}</span>
          </button>
          <button class="btn-action glass-panel" :disabled="tuyaSyncing || tuyaCloudLoading || !tuyaConfig.configured" @click="syncTuyaCloudToLocal">
            <Wifi :size="15" />
            <span>{{ tuyaSyncing ? 'Aggiornamento LocalTuya…' : `Applica selezione LAN (${selectedTuyaDeviceIds.length})` }}</span>
          </button>
          <button v-if="tuyaConfig.configured" class="btn-action glass-panel" @click="isEditingTuyaConfig = !isEditingTuyaConfig">
            <Key :size="15" />
            <span>{{ isEditingTuyaConfig ? 'Chiudi Config' : 'Modifica Config' }}</span>
          </button>
          <button v-if="tuyaConfig.configured" class="btn-action glass-panel" @click="disconnectTuyaConfig">
            <Unlock :size="15" />
            <span>{{ t('integrations.view.disconnect') }}</span>
          </button>
        </div>
      </div>

      <!-- STATUS BANNER WHEN CONFIGURED -->
      <div v-if="tuyaConfig.configured && !isEditingTuyaConfig" class="tuya-status-banner glass-card">
        <div class="tuya-status-left">
          <div class="badge-direct-mode"><CheckCircle :size="15" /> <span>{{ t('integrations.view.serverToServer') }}</span></div>
          <div class="tuya-meta-chips">
            <span class="meta-chip"><strong>{{ t('integrations.view.clientId') }}</strong> {{ tuyaConfig.client_id_masked || tuyaConfig.client_id }}</span>
            <span class="meta-chip"><strong>{{ t('integrations.view.smartLifeUidLabel') }}</strong> {{ tuyaConfig.uid }}</span>
            <span class="meta-chip"><strong>{{ t('integrations.view.endpointLabel') }}</strong> {{ tuyaConfig.endpoint }}</span>
          </div>
        </div>
      </div>

      <!-- SETUP GUIDE & CONFIG FORM (WHEN NOT CONFIGURED OR EDITING) -->
      <div v-if="!tuyaConfig.configured || isEditingTuyaConfig" class="tuya-config-panel glass-card">
        <div class="tuya-guide-box">
          <h4 class="guide-title"><QrCode :size="18" /> {{ t('integrations.view.tuyaConnectTitle') }}</h4>
          <ol class="guide-steps">
            <li>
              Entra nel tuo progetto su 
              <a href="https://iot.tuya.com" target="_blank" rel="noopener noreferrer" class="link-external">
                iot.tuya.com <ExternalLink :size="12" />
              </a>
              (Cloud → Development → apri il progetto Domus).
            </li>
            <li>{{ t('integrations.view.dataCenterInstruction') }}</li>
            <li>{{ t('integrations.view.devicesAccountInstruction') }}</li>
            <li>{{ t('integrations.view.qrAccountInstruction') }}</li>
            <li>
              {{ t('integrations.view.smartLifeScanInstruction') }}
            </li>
            <li>{{ t('integrations.view.copyCredentialsInstruction') }}</li>
          </ol>
        </div>

        <form class="tuya-form-grid" @submit.prevent="saveTuyaConfig">
          <div class="form-group">
            <label for="tuya-client-id">{{ t('integrations.view.accessId') }} *</label>
            <input
              id="tuya-client-id"
              v-model="tuyaConfigForm.client_id"
              type="text"
              class="form-control glass-panel"
              :placeholder="t('integrations.view.accessIdPlaceholder')"
              required
            />
          </div>

          <div class="form-group">
            <label for="tuya-client-secret">{{ t('integrations.view.accessSecret') }} *</label>
            <input
              id="tuya-client-secret"
              v-model="tuyaConfigForm.client_secret"
              type="password"
              class="form-control glass-panel"
              :placeholder="t('integrations.view.accessSecretPlaceholder')"
              required
            />
          </div>

          <div class="form-group">
            <label for="tuya-uid">{{ t('integrations.view.smartLifeUid') }} *</label>
            <input
              id="tuya-uid"
              v-model="tuyaConfigForm.uid"
              type="text"
              class="form-control glass-panel"
              :placeholder="t('integrations.view.uidPlaceholder')"
              required
            />
          </div>

          <div class="form-group">
            <label for="tuya-endpoint">{{ t('integrations.view.apiEndpoint') }}</label>
            <select
              id="tuya-endpoint"
              v-model="tuyaConfigForm.endpoint"
              class="form-control glass-panel"
            >
              <option value="https://openapi.tuyaeu.com">{{ t('integrations.view.centralEurope') }}</option>
              <option value="https://openapi-weaz.tuyaeu.com">{{ t('integrations.view.westernEurope') }}</option>
              <option value="https://openapi.tuyaus.com">{{ t('integrations.view.westernAmerica') }}</option>
              <option value="https://openapi-ueaz.tuyaus.com">{{ t('integrations.view.easternAmerica') }}</option>
              <option value="https://openapi.tuyacn.com">{{ t('integrations.view.chinaDataCenter') }}</option>
              <option value="https://openapi.tuyain.com">{{ t('integrations.view.indiaDataCenter') }}</option>
            </select>
          </div>

          <div class="form-actions">
            <button type="submit" class="btn-scan" :disabled="tuyaConfigSaving">
              <CheckCircle :size="16" />
              <span>{{ tuyaConfigSaving ? 'Connessione e verifica…' : 'Salva e Collega Smart Life' }}</span>
            </button>
            <button
              v-if="tuyaConfig.configured"
              type="button"
              class="btn-action glass-panel"
              @click="isEditingTuyaConfig = false"
            >
              Annulla
            </button>
          </div>
        </form>
      </div>

      <div v-if="tuyaCloudError && tuyaConfig.configured" class="empty-guardian-state glass-card">
        <AlertCircle :size="30" />
        <h4>{{ t('integrations.view.tuyaCloudUnavailable') }}</h4>
        <p>{{ tuyaCloudError }}</p>
        <p>{{ t('integrations.view.tuyaCredentialsHint') }}</p>
      </div>

      <div v-else-if="tuyaCloudLoading && tuyaCloudDevices.length === 0" class="empty-guardian-state glass-card">
        <RefreshCw :size="28" class="animate-spin" />
        <p>{{ t('integrations.view.loadingSmartLifeDevices') }}</p>
      </div>

      <div v-else-if="tuyaConfig.configured && tuyaCloudDevices.length === 0" class="empty-guardian-state glass-card">
        <Globe :size="32" />
        <h4>{{ t('integrations.view.noDevicesForUid') }}</h4>
        <p>{{ t('integrations.view.verifyUidInstruction') }}</p>
      </div>

      <div v-if="tuyaCloudDevices.length > 0" class="tuya-device-toolbar glass-card">
        <div class="tuya-device-toolbar-summary">
          <div>
            <strong>{{ t('integrations.view.smartLifeDevices') }}</strong>
            <span class="tuya-device-count">{{ tuyaCloudDevices.length }} totali · {{ tuyaOnlineCount }} online · {{ tuyaOfflineCount }} offline</span>
          </div>
          <span class="tuya-device-count">{{ t('integrations.view.selectedForLocalTuya', { count: tuyaSelectedCount }) }}</span>
        </div>
        <div class="tuya-device-filter-row">
          <label class="tuya-device-search">
            <Search :size="15" />
            <input id="tuya-device-search" v-model="tuyaSearchQuery" type="search" :placeholder="t('integrations.view.searchDevicesPlaceholder')" />
          </label>
          <div class="tuya-device-filters" role="group" :aria-label="t('integrations.view.filterTuyaDevices')">
            <button type="button" data-filter="all" :aria-pressed="tuyaDeviceFilter === 'all'" :class="{ active: tuyaDeviceFilter === 'all' }" @click="tuyaDeviceFilter = 'all'">{{ t('integrations.view.filterAll', { count: tuyaCloudDevices.length }) }}</button>
            <button type="button" data-filter="online" :aria-pressed="tuyaDeviceFilter === 'online'" :class="{ active: tuyaDeviceFilter === 'online' }" @click="tuyaDeviceFilter = 'online'">{{ t('integrations.view.filterOnline', { count: tuyaOnlineCount }) }}</button>
            <button type="button" data-filter="offline" :aria-pressed="tuyaDeviceFilter === 'offline'" :class="{ active: tuyaDeviceFilter === 'offline' }" @click="tuyaDeviceFilter = 'offline'">{{ t('integrations.view.filterOffline', { count: tuyaOfflineCount }) }}</button>
            <button type="button" data-filter="selected" :aria-pressed="tuyaDeviceFilter === 'selected'" :class="{ active: tuyaDeviceFilter === 'selected' }" @click="tuyaDeviceFilter = 'selected'">{{ t('integrations.view.filterSelected', { count: tuyaSelectedCount }) }}</button>
            <button type="button" data-filter="unselected" :aria-pressed="tuyaDeviceFilter === 'unselected'" :class="{ active: tuyaDeviceFilter === 'unselected' }" @click="tuyaDeviceFilter = 'unselected'">{{ t('integrations.view.filterUnselected', { count: tuyaUnselectedCount }) }}</button>
          </div>
        </div>
        <div class="tuya-device-bulk-actions">
          <span>{{ t('integrations.view.devicesShown', { count: filteredTuyaCloudDevices.length }) }}</span>
          <div>
            <button type="button" @click="selectVisibleTuyaDevices" :disabled="filteredTuyaCloudDevices.length === 0">{{ t('integrations.view.selectVisible') }}</button>
            <button type="button" @click="deselectVisibleTuyaDevices" :disabled="!filteredTuyaCloudDevices.some(device => selectedTuyaDeviceIds.includes(device.id))">{{ t('integrations.view.deselectVisible') }}</button>
          </div>
        </div>
      </div>

      <div v-if="filteredTuyaCloudDevices.length > 0" class="hosts-grid">
        <div v-for="device in filteredTuyaCloudDevices" :key="device.id" :class="['host-card glass-card', { 'tuya-device-offline': !device.online }]">
          <div class="tuya-device-selection">
            <label>
              <input v-model="selectedTuyaDeviceIds" type="checkbox" :value="device.id" />
              <span>{{ t('integrations.view.useInDomus') }}</span>
            </label>
            <span v-if="device.local_configured" class="tuya-selected-note">{{ t('integrations.view.enabled') }}</span>
          </div>
          <div class="host-top">
            <div class="host-brand-icon"><Zap :size="20" /></div>
            <div class="host-meta">
              <span class="host-vendor">{{ device.name }}</span>
              <span class="host-ip">{{ device.product_name || device.category || device.id }}</span>
              <span v-if="device.local_ip" class="spec-lan-ip">LAN: {{ device.local_ip }}</span>
            </div>
            <div style="display: flex; gap: 6px; align-items: center;">
              <span v-if="device.local_configured" class="badge-lan" :title="t('integrations.view.localLanControl')"><Wifi :size="12" /> {{ t('integrations.view.lanLocal') }}</span>
              <span :class="['badge-iot', { 'badge-quarantined': !device.online }]">{{ device.online ? t('integrations.view.deviceOnline') : t('integrations.view.deviceOffline') }}</span>
            </div>
          </div>
          <div class="host-specs">
            <div v-for="point in device.status || []" :key="point.code" class="spec-line">
              <span class="spec-key">{{ point.code }}</span>
              <span class="spec-code">{{ String(point.value) }}</span>
              <button
                v-if="typeof point.value === 'boolean' && (point.code.toLowerCase().includes('switch') || point.code.toLowerCase().includes('power'))"
                class="btn-host-probe"
                :disabled="tuyaCloudCommanding === `${device.id}:${point.code}` || (!device.online && !device.local_active)"
                @click="toggleTuyaStatus(device, point)"
              >
                {{ point.value ? t('integrations.view.turnOff') : t('integrations.view.turnOn') }}
              </button>
            </div>
          </div>

          <div v-if="isTuyaClimateDevice(device)" class="tuya-climate-controls">
            <div class="tuya-temperature-control">
              <label :for="`tuya-temp-${device.id}`">{{ t('integrations.view.targetTemperature') }}</label>
              <select :id="`tuya-temp-${device.id}`" v-model="tuyaTemperatureInputs[device.id]">
                <option value="" disabled>{{ t('integrations.view.chooseTemperature') }}</option>
                <option v-for="mapping in TUYA_THERMOSTAT_SETPOINTS" :key="mapping.degrees" :value="String(mapping.degrees)">
                  {{ mapping.degrees.toLocaleString(locale.value) }} °C · {{ t('integrations.view.encodedValue') }} {{ mapping.encoded.toLocaleString(locale.value) }}
                </option>
              </select>
              <div class="tuya-temperature-preview">
                {{ t('integrations.view.valueToSend') }}: <strong>{{ tuyaTemperatureEncodedPreview(device.id) }}</strong>
              </div>
              <button class="btn-host-probe" :disabled="tuyaCloudCommanding === `${device.id}:temperature` || !encodeTuyaThermostatSetpoint(Number(tuyaTemperatureInputs[device.id])) || (!device.local_configured && (!device.online || !tuyaTemperatureFunction(device.id)))" @click="setTuyaTemperature(device)">{{ t('integrations.view.setTemperature') }}</button>
            </div>
            <div class="tuya-temperature-map" :aria-label="t('integrations.view.setpointMapAria')">
              <div v-for="mapping in TUYA_THERMOSTAT_SETPOINTS" :key="mapping.encoded" class="tuya-temperature-map-row">
                <code>{{ mapping.encoded.toLocaleString(locale.value) }}</code>
                <span>→</span>
                <strong>{{ mapping.degrees.toLocaleString(locale.value) }} °C</strong>
              </div>
            </div>
            <div v-if="canControlLocalTuya(device)" class="tuya-mode-control">
              <span>{{ t('integrations.view.localTuyaMode') }}</span>
              <button class="btn-host-probe" :disabled="tuyaCloudCommanding === `${device.id}:dp4`" @click="setTuyaClimateMode(device, 'auto')">{{ t('integrations.view.automaticDp4') }}</button>
              <button class="btn-host-probe" :disabled="tuyaCloudCommanding === `${device.id}:dp4`" @click="setTuyaClimateMode(device, 'manual')">{{ t('integrations.view.manualDp4') }}</button>
            </div>
          </div>

          <div class="tuya-functions-toggle">
            <button type="button" class="btn-action glass-panel" :disabled="tuyaFunctionsLoading === device.id" @click="toggleTuyaDeviceFunctions(device)">
              <RefreshCw v-if="tuyaFunctionsLoading === device.id" :size="14" class="animate-spin" />
              <Cpu v-else :size="14" />
              <span>{{ tuyaFunctionPanels[device.id] ? t('integrations.view.hideCommands') : t('integrations.view.availableApiCommands') }}</span>
            </button>
          </div>
          <div v-if="tuyaFunctionPanels[device.id]" class="tuya-functions-panel">
            <div v-if="(tuyaDeviceFunctions[device.id] || []).length === 0 && tuyaFunctionsLoading !== device.id" class="tuya-no-functions">
              {{ t('integrations.view.noWritableFunctions') }}
            </div>
            <div v-for="fn in tuyaDeviceFunctions[device.id] || []" :key="fn.code" class="tuya-function-row">
              <div class="tuya-function-meta">
                <strong>{{ fn.name || fn.code }}</strong>
                <span>{{ fn.code }} · {{ fn.type }}</span>
                <small>{{ fn.desc || fn.values || 'Parametri non specificati' }}</small>
              </div>
              <div class="tuya-function-input">
                <input
                  v-if="isNumericTuyaFunction(fn)"
                  v-model="tuyaFunctionInputs[tuyaFunctionKey(device.id, fn.code)]"
                  type="number"
                  :aria-label="t('integrations.view.valueFor', { name: fn.name || fn.code })"
                />
                <input
                  v-else-if="fn.type.toLowerCase() !== 'boolean'"
                  v-model="tuyaFunctionInputs[tuyaFunctionKey(device.id, fn.code)]"
                  type="text"
                  :aria-label="t('integrations.view.valueFor', { name: fn.name || fn.code })"
                />
                <button type="button" class="btn-host-probe" :disabled="tuyaFunctionBusy === tuyaFunctionKey(device.id, fn.code) || (!device.online && !device.local_active)" @click="executeTuyaFunction(device, fn)">
                  {{ tuyaFunctionBusy === tuyaFunctionKey(device.id, fn.code) ? 'Invio…' : fn.type.toLowerCase() === 'boolean' ? 'Inverti' : 'Invia' }}
                </button>
              </div>
              <code class="tuya-function-values">{{ fn.values }}</code>
            </div>
          </div>
        </div>
      </div>
      <div v-else-if="tuyaCloudDevices.length > 0" class="empty-guardian-state glass-card">
        <WifiOff :size="28" />
        <h4>{{ t('integrations.view.noFilterMatches') }}</h4>
        <p>{{ t('integrations.view.adjustSearchFilters') }}</p>
        <button type="button" class="btn-action glass-panel" @click="tuyaDeviceFilter = 'all'; tuyaSearchQuery = ''">{{ t('integrations.view.resetFilters') }}</button>
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

.badge-lan {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: var(--radius-full);
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  font-size: 11px;
  font-weight: 600;
}

.spec-lan-ip {
  font-size: 11px;
  color: #10b981;
  font-family: var(--font-mono, monospace);
  margin-top: 2px;
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

/* SMART LIFE DIRECT SERVER-TO-SERVER STYLES */
.tuya-status-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-radius: var(--radius-lg);
  border: 1px solid rgba(16, 185, 129, 0.25);
  background: rgba(16, 185, 129, 0.05);
}

.badge-direct-mode {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--success);
}

.tuya-meta-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 6px;
}

.meta-chip {
  font-size: 12px;
  color: var(--text-dim);
  background: rgba(255, 255, 255, 0.04);
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
}

.meta-chip strong {
  color: var(--text-main);
  margin-right: 4px;
}

.tuya-config-panel {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 24px;
  border-radius: var(--radius-xl);
}

.tuya-guide-box {
  background: rgba(14, 165, 233, 0.06);
  border: 1px solid rgba(14, 165, 233, 0.2);
  border-radius: var(--radius-lg);
  padding: 18px 20px;
}

.guide-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--primary);
  margin-bottom: 12px;
}

.guide-steps {
  margin: 0;
  padding-left: 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;
  color: var(--text-main);
  line-height: 1.5;
}

.guide-steps strong {
  color: #fff;
}

.guide-steps code {
  background: rgba(255, 255, 255, 0.1);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-family: monospace;
  font-size: 12px;
}

.link-external {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--primary);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.tuya-form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
  align-items: end;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
}

.form-control {
  padding: 10px 14px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  background: rgba(0, 0, 0, 0.25);
  color: var(--text-main);
  font-size: 13px;
  outline: none;
  transition: border-color var(--transition-fast);
}

.form-control:focus {
  border-color: var(--primary);
}

.form-actions {
  grid-column: 1 / -1;
  display: flex;
  gap: 12px;
  margin-top: 8px;
}

/* Mobile and narrow-screen layout: keep every integration tab reachable. */
.integrations-view,
.tab-content {
  width: 100%;
  min-width: 0;
}

.header-row { align-items: flex-start; }
.header-right-tools { min-width: 0; max-width: 100%; }

.tab-switcher {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
  flex-wrap: nowrap;
  justify-content: flex-start;
  -webkit-overflow-scrolling: touch;
  overscroll-behavior-x: contain;
  scrollbar-width: thin;
  touch-action: pan-x;
}

.tab-btn {
  flex: 0 0 auto;
  min-height: 44px;
  white-space: nowrap;
}

.actions-bar { gap: 12px; min-width: 0; }
.bar-info { min-width: 0; }

.integration-card,
.discovered-card,
.brand-card,
.host-card,
.discovery-toolbar,
.guardian-banner,
.leak-card {
  min-width: 0;
  max-width: 100%;
}

@media (max-width: 768px) {
  .header-right-tools {
    width: 100%;
    flex: 1 1 100%;
    flex-direction: column;
    align-items: stretch;
  }
  .btn-flasher-trigger { align-self: flex-start; }
  .tab-switcher { width: 100%; }
  .tab-btn { padding-inline: 12px; }
  .actions-bar { flex-wrap: wrap; align-items: flex-start; }

  .integrations-grid,
  .discovered-grid,
  .catalog-grid,
  .hosts-grid {
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  }

  .discovery-inputs { width: 100%; min-width: 0; }
  .manual-probe-box { min-width: 0; max-width: 100%; }
  .ip-input { min-width: 0; flex: 1 1 auto; }
  .leak-left { min-width: 0; }
}

@media (max-width: 480px) {
  .header-row { gap: 12px; }
  .header-right-tools { gap: 8px; }
  .tab-switcher { border-radius: var(--radius-md); }
  .tab-btn { gap: 6px; padding: 9px 11px; font-size: 12px; }
  .tab-btn svg { flex: 0 0 auto; }

  .actions-bar { align-items: stretch; }
  .bar-info { flex: 1 1 100%; }
  .btn-action { width: 100%; justify-content: center; min-height: 44px; }

  .integration-card,
  .discovered-card,
  .brand-card,
  .host-card { padding: 14px; }

  .card-top { flex-wrap: wrap; }
  .status-badge { margin-left: auto; }

  .discovery-toolbar,
  .guardian-banner,
  .leak-card { padding: 14px; }

  .discovery-intro { align-items: flex-start; min-width: 0; }
  .discovery-inputs { flex-direction: column; align-items: stretch; }
  .manual-probe-box { width: 100%; }
  .btn-scan { width: 100%; min-height: 44px; justify-content: center; }

  .discovered-grid,
  .catalog-grid,
  .hosts-grid,
  .integrations-grid { grid-template-columns: minmax(0, 1fr); }

  .disc-card-header,
  .leak-header-row,
  .spec-line { flex-wrap: wrap; }

  .leak-left { flex: 1 1 100%; }
  .btn-quarantine { width: 100%; min-height: 44px; justify-content: center; }

  .host-footer { flex-direction: column; }
  .btn-host-probe,
  .btn-host-quarantine { width: 100%; min-height: 44px; }
}


/* Tuya device selection, discovery filters and thermostat controls */
.tuya-device-toolbar {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px 18px;
  margin-bottom: 14px;
}
.tuya-device-toolbar-summary,
.tuya-device-filter-row,
.tuya-device-bulk-actions,
.tuya-device-bulk-actions > div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}
.tuya-device-search {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: min(100%, 250px);
  flex: 1;
  padding: 8px 11px;
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  color: var(--text-muted);
}
.tuya-device-search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text-main);
  font: inherit;
}
.tuya-device-bulk-actions {
  padding-top: 10px;
  border-top: 1px solid var(--border-subtle);
  color: var(--text-muted);
  font-size: 12px;
}
.tuya-device-bulk-actions button {
  padding: 6px 10px;
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  color: var(--text-main);
  font: inherit;
  cursor: pointer;
}
.tuya-device-bulk-actions button:disabled {
  opacity: .45;
  cursor: not-allowed;
}
.tuya-device-count {
  display: block;
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 12px;
}
.tuya-device-filters {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.tuya-device-filters button {
  min-height: 40px;
  padding: 7px 12px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: rgba(255,255,255,.035);
  color: var(--text-muted);
  cursor: pointer;
}
.tuya-device-filters button.active {
  border-color: var(--primary);
  color: var(--text-main);
  background: rgba(14,165,233,.12);
}
.tuya-device-selection {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 12px;
  color: var(--text-main);
  font-size: 12px;
}
.tuya-device-selection label {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  cursor: pointer;
}
.tuya-device-selection input { width: 18px; height: 18px; accent-color: var(--primary); }
.tuya-selected-note { color: var(--success); font-size: 11px; font-weight: 700; }
.tuya-device-offline { opacity: .82; }
.tuya-climate-controls, .tuya-functions-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 14px;
  padding: 14px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: rgba(255,255,255,.025);
}
.tuya-temperature-control, .tuya-mode-control, .tuya-function-input {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.tuya-temperature-control label, .tuya-mode-control > span {
  flex: 1 1 100%;
  color: var(--text-muted);
  font-size: 12px;
}
.tuya-temperature-control input, .tuya-temperature-control select, .tuya-function-input input {
  min-height: 40px;
  width: 100%;
  max-width: 240px;
  padding: 7px 10px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: rgba(0,0,0,.2);
  color: var(--text-main);
}
.tuya-temperature-preview {
  flex: 1 1 100%;
  color: var(--text-muted);
  font-size: 12px;
}
.tuya-temperature-preview strong {
  color: var(--text-main);
  font-family: var(--font-mono);
}
.tuya-temperature-map {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(110px, 1fr));
  gap: 6px;
}
.tuya-temperature-map-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  padding: 7px 9px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: rgba(255,255,255,.025);
  font-size: 12px;
}
.tuya-temperature-map-row code {
  color: var(--primary);
  font-family: var(--font-mono);
}
.tuya-temperature-map-row strong {
  color: var(--text-main);
}
.tuya-functions-toggle { margin-top: 12px; }
.tuya-function-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  padding: 10px 0;
  border-bottom: 1px solid var(--border-subtle);
}
.tuya-function-row:last-child { border-bottom: 0; }
.tuya-function-meta { display: flex; flex-direction: column; gap: 3px; min-width: 0; }
.tuya-function-meta strong { font-size: 13px; color: var(--text-main); }
.tuya-function-meta span, .tuya-function-meta small, .tuya-function-values { font-size: 11px; color: var(--text-muted); overflow-wrap: anywhere; }
.tuya-function-values { grid-column: 1 / -1; white-space: pre-wrap; }
.tuya-no-functions { color: var(--text-muted); font-size: 12px; }
@media (max-width: 620px) {
  .tuya-device-toolbar-summary,
  .tuya-device-filter-row,
  .tuya-device-bulk-actions {
    align-items: stretch;
    flex-direction: column;
  }
  .tuya-device-search {
    width: 100%;
    min-width: 0;
  }
  .tuya-device-filters button {
    flex: 1 1 calc(50% - 6px);
  }
  .tuya-device-bulk-actions > div {
    width: 100%;
  }
  .tuya-device-bulk-actions button {
    flex: 1;
  }
  .tuya-function-row { grid-template-columns: 1fr; }
  .tuya-function-input input { flex: 1; width: min(100%, 180px); }
}

</style>

