<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAreasStore } from '../stores/areas'
import { useEntitiesStore } from '../stores/entities'
import { useDevicesStore } from '../stores/devices'
import { useAuthStore } from '../stores/auth'
import { useNotificationsStore } from '../stores/notifications'
import { apiClient } from '../api/client'
import type { Area, Device, Entity } from '../api/types'
import BaseButton from '../components/ui/BaseButton.vue'
import AreaEditorModal from '../components/modals/AreaEditorModal.vue'
import {
  Grid,
  Layers,
  Plus,
  Pencil,
  Trash2,
  Sofa,
  Utensils,
  Bed,
  Bath,
  Laptop,
  Trees,
  Sun,
  Car,
  Tv,
  Cpu,
  RefreshCw
} from '@lucide/vue'

const { t } = useI18n()

const areasStore = useAreasStore()
const entitiesStore = useEntitiesStore()
const devicesStore = useDevicesStore()
const authStore = useAuthStore()
const notificationsStore = useNotificationsStore()

const isEditorOpen = ref(false)
const editingArea = ref<Area | null>(null)
const managingArea = ref<Area | null>(null)
const isDeviceManagerOpen = ref(false)
const isSavingAreaDevices = ref(false)
const selectedDeviceIds = ref<string[]>([])
const isLoadingAssignableDevices = ref(false)

interface ExternalTuyaDevice {
  id: string
  name: string
  product_name?: string
  category?: string
  online?: boolean
  local_configured?: boolean
  local_active?: boolean
  local_ip?: string
  device_id?: string
  connected?: boolean
  ip?: string
}

const tuyaCloudDevices = ref<ExternalTuyaDevice[]>([])
const tuyaOAuthDevices = ref<ExternalTuyaDevice[]>([])
const tuyaLocalDevices = ref<ExternalTuyaDevice[]>([])

onMounted(async () => {
  isLoadingAssignableDevices.value = true
  try {
    const results = await Promise.allSettled([
      areasStore.fetchAreas(),
      entitiesStore.fetchEntities(),
      devicesStore.fetchDevices(),
      loadTuyaLocalDevices(),
      loadTuyaCloudDevices(),
      loadTuyaOAuthDevices()
    ])
    const failed = results.filter(result => result.status === 'rejected')
    if (failed.length > 0) {
      console.warn('AreasView data fetch warning:', failed.length, 'source(s) could not be loaded')
    }
  } finally {
    isLoadingAssignableDevices.value = false
  }
})

async function loadTuyaLocalDevices() {
  if (!authStore.isAdmin) {
    tuyaLocalDevices.value = []
    return
  }
  try {
    const result = await apiClient.get<{ devices?: ExternalTuyaDevice[] }>('/api/v1/integrations/tuya-local/devices')
    tuyaLocalDevices.value = Array.isArray(result?.devices) ? result.devices : []
  } catch {
    // The local adapter may be disabled on installations using cloud-only Tuya.
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
    const result = await apiClient.get<{ devices?: ExternalTuyaDevice[] }>('/api/v1/integrations/tuya-cloud/devices')
    tuyaCloudDevices.value = Array.isArray(result?.devices) ? result.devices : []
  } catch {
    // This legacy Tuya Cloud connection is optional; registered/local devices
    // must remain available even when the cloud API is not configured.
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
    const result = await apiClient.get<{ devices?: ExternalTuyaDevice[] }>('/api/v1/integrations/tuya-oauth/devices')
    tuyaOAuthDevices.value = Array.isArray(result?.devices) ? result.devices : []
  } catch {
    // OAuth devices are admin-only and optional. Do not block the room picker
    // when the account is disconnected or the provider is temporarily offline.
    tuyaOAuthDevices.value = []
  }
}

function getEntitiesForArea(areaId: string) {
  return entitiesStore.allEntities.filter(
    (e) => (e.attributes as any)?.area_id === areaId || (e as any).area_id === areaId
  )
}

function entityAreaId(entity: Entity): string {
  return String(entity.area_id || entity.attributes?.area_id || '')
}

function entityDeviceId(entity: Entity): string {
  const direct = String(entity.device_id || '').trim()
  if (direct) return direct
  for (const candidate of [entity.entity_key || '', entity.id || '']) {
    const suffix = candidate.split('.').pop() || ''
    if (suffix.startsWith('tuya_') && suffix.length > 'tuya_'.length) return suffix
  }
  return ''
}

const assignableDevices = computed<Device[]>(() => {
  const result = new Map<string, Device>()
  for (const item of devicesStore.devices) {
    result.set(item.id, { ...item })
  }

  // DevicesView also lists Tuya devices directly from Cloud/Smart Life APIs.
  // Include those same sources here even if the LocalTuya adapter has not yet
  // created a device/entity registry row (e.g. cloud-only devices without key).
  const externalTuyaDevices = [
    ...tuyaLocalDevices.value.map(item => ({
      ...item,
      id: item.device_id || item.id,
      online: item.connected === true || item.online === true,
      local_configured: true,
      source: 'tuya_local' as const
    })),
    ...tuyaCloudDevices.value.map(item => ({ ...item, source: 'tuya_cloud' as const })),
    ...tuyaOAuthDevices.value.map(item => ({ ...item, source: 'tuya_oauth' as const }))
  ].filter(item => Boolean(item.id))
  for (const item of externalTuyaDevices) {
    const id = `tuya_${item.id}`
    const existing = result.get(id) || [...result.values()].find(device =>
      device.external_id === item.id || device.id === item.id
    )
    if (existing) {
      existing.status = existing.status === 'online' || item.online ? 'online' : 'offline'
      if (!existing.name || existing.name === existing.id) existing.name = item.name
      if (!existing.model) existing.model = item.product_name || item.category
      continue
    }
    result.set(id, {
      id,
      external_id: item.id,
      name: item.name,
      integration_id: item.local_configured ? 'tuya_local' : item.source,
      manufacturer: 'Tuya',
      model: item.product_name || item.category,
      status: item.online ? 'online' : 'offline',
      created_at: '',
      updated_at: ''
    })
  }

  const entitiesByDevice = new Map<string, Entity[]>()
  for (const entity of entitiesStore.allEntities) {
    const deviceId = entityDeviceId(entity)
    if (!deviceId) continue
    const group = entitiesByDevice.get(deviceId) || []
    group.push(entity)
    entitiesByDevice.set(deviceId, group)
  }

  for (const [deviceId, linkedEntities] of entitiesByDevice) {
    const matching = [...result.values()].find(device =>
      device.id === deviceId ||
      device.external_id === deviceId ||
      device.id === `tuya_${deviceId}` ||
      (device.id.startsWith('tuya_') && device.id.slice(5) === deviceId)
    )
    if (matching) {
      if (!matching.area_id) {
        matching.area_id = linkedEntities.map(entityAreaId).find(Boolean) || undefined
      }
      continue
    }

    const attributes = linkedEntities.reduce<Record<string, any>>((all, entity) => ({
      ...all,
      ...(entity.attributes || {})
    }), {})
    const isTuya = deviceId.startsWith('tuya_')
    const status = linkedEntities.every(entity =>
      ['unavailable', 'unknown'].includes(String(entity.state || '').toLowerCase())
    ) ? 'offline' : 'online'
    result.set(deviceId, {
      id: deviceId,
      external_id: String(attributes.external_id || (isTuya ? deviceId.slice(5) : deviceId)),
      name: String(attributes.device_name || attributes.device_friendly_name ||
        attributes.friendly_name || linkedEntities[0]?.name || deviceId),
      integration_id: String(attributes.integration_id || attributes.integration ||
        attributes.platform || (isTuya ? 'tuya_local' : 'integrated')),
      manufacturer: attributes.manufacturer ? String(attributes.manufacturer) : undefined,
      model: attributes.model ? String(attributes.model) : undefined,
      area_id: linkedEntities.map(entityAreaId).find(Boolean) || undefined,
      status,
      created_at: '',
      updated_at: ''
    })
  }

  return [...result.values()].sort((a, b) => a.name.localeCompare(b.name))
})

function getDevicesForArea(areaId: string): Device[] {
  return assignableDevices.value.filter(device => device.area_id === areaId)
}

function getAreaItemCount(areaId: string): number {
  const deviceCount = getDevicesForArea(areaId).length
  return deviceCount || getEntitiesForArea(areaId).length
}

function openDeviceManager(area: Area) {
  managingArea.value = area
  selectedDeviceIds.value = getDevicesForArea(area.id).map(device => device.id)
  isDeviceManagerOpen.value = true
}

function closeDeviceManager() {
  isDeviceManagerOpen.value = false
  managingArea.value = null
  selectedDeviceIds.value = []
}

async function refreshAssignableDevices() {
  if (isLoadingAssignableDevices.value || isSavingAreaDevices.value) return
  isLoadingAssignableDevices.value = true
  try {
    await Promise.all([
      devicesStore.fetchDevices(),
      entitiesStore.fetchEntities(),
      loadTuyaLocalDevices(),
      loadTuyaCloudDevices(),
      loadTuyaOAuthDevices()
    ])
  } catch {
    notificationsStore.error(
      t('devices.view.refreshErrorTitle'),
      t('devices.view.refreshErrorMessage')
    )
  } finally {
    isLoadingAssignableDevices.value = false
  }
}

function updateSelectedDevice(deviceId: string, event: Event) {
  const checked = (event.target as HTMLInputElement | null)?.checked === true
  selectedDeviceIds.value = checked
    ? [...new Set([...selectedDeviceIds.value, deviceId])]
    : selectedDeviceIds.value.filter(id => id !== deviceId)
}

async function saveAreaDevices() {
  const area = managingArea.value
  if (!area || isSavingAreaDevices.value) return

  const selected = new Set(selectedDeviceIds.value)
  const changes = assignableDevices.value.filter(device =>
    selected.has(device.id) ? device.area_id !== area.id : device.area_id === area.id
  )
  isSavingAreaDevices.value = true
  try {
    for (const device of changes) {
      const nextAreaId = selected.has(device.id) ? area.id : null
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
    }
    await Promise.all([devicesStore.fetchDevices(), entitiesStore.fetchEntities(), loadTuyaLocalDevices(), loadTuyaCloudDevices(), loadTuyaOAuthDevices()])
    notificationsStore.success(
      t('areas.devicesSaveSuccess'),
      t('areas.devicesSaveMessage', { count: changes.length, area: area.name })
    )
    closeDeviceManager()
  } catch (err: any) {
    await Promise.allSettled([devicesStore.fetchDevices(), entitiesStore.fetchEntities(), loadTuyaCloudDevices(), loadTuyaOAuthDevices()])
    notificationsStore.error(
      t('areas.devicesSaveError'),
      err?.message || t('areas.devicesSaveError')
    )
  } finally {
    isSavingAreaDevices.value = false
  }
}

function getAreaIcon(icon?: string) {
  switch (icon?.toLowerCase()) {
    case 'sofa':
    case 'living':
      return Sofa
    case 'utensils':
    case 'kitchen':
      return Utensils
    case 'bed':
    case 'bedroom':
      return Bed
    case 'bath':
    case 'bathroom':
      return Bath
    case 'laptop':
    case 'office':
      return Laptop
    case 'trees':
    case 'garden':
      return Trees
    case 'sun':
    case 'balcony':
    case 'terrace':
      return Sun
    case 'car':
    case 'garage':
      return Car
    case 'tv':
    case 'cinema':
      return Tv
    default:
      return Grid
  }
}

function openNewModal() {
  editingArea.value = null
  isEditorOpen.value = true
}

function openEditModal(area: Area) {
  editingArea.value = area
  isEditorOpen.value = true
}

async function handleDelete(area: Area) {
  if (confirm(t('areas.confirmDelete', { name: area.name }))) {
    try {
      await areasStore.deleteArea(area.id)
      notificationsStore.success(t('areas.deleteSuccess'), `"${area.name}"`)
    } catch (err: any) {
      notificationsStore.error(t('common.error'), err?.message || t('areas.deleteError'))
    }
  }
}

async function handleSaveArea(payload: { name: string; icon: string; sort_order: number }) {
  try {
    if (editingArea.value) {
      await areasStore.updateArea(editingArea.value.id, payload)
      notificationsStore.success(t('areas.saveSuccess'), `"${payload.name}"`)
    } else {
      await areasStore.createArea(payload)
      notificationsStore.success(t('areas.saveSuccess'), `"${payload.name}"`)
    }
    isEditorOpen.value = false
    editingArea.value = null
  } catch (err: any) {
    notificationsStore.error(t('common.error'), err?.message || t('areas.saveError'))
  }
}
</script>

<template>
  <div class="areas-view">
    <div class="header-row">
      <div>
        <h2 class="section-title">{{ t('areas.title') }}</h2>
        <p class="section-subtitle">{{ t('areas.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <BaseButton variant="primary" @click="openNewModal">
          <Plus :size="16" />
          {{ t('areas.newArea') }}
        </BaseButton>
      </div>
    </div>

    <!-- Areas Grid -->
    <div v-if="areasStore.sortedAreas.length > 0" class="areas-grid">
      <div
        v-for="area in areasStore.sortedAreas"
        :key="area.id"
        class="area-card glass-card"
      >
        <div class="area-card-top">
          <div class="area-icon-box">
            <component :is="getAreaIcon(area.icon)" :size="24" />
          </div>
          <div class="area-info">
            <h3 class="area-name">{{ area.name }}</h3>
            <span class="area-count">
              {{ getAreaItemCount(area.id) }} {{ t('areas.devicesCount') }}
            </span>
          </div>

          <div class="card-actions">
            <button
              v-if="authStore.isAdmin"
              class="btn-card-action manage-devices-action"
              :title="t('areas.manageDevices')"
              :aria-label="t('areas.manageDevices')"
              @click="openDeviceManager(area)"
            >
              <Cpu :size="15" />
            </button>
            <button
              class="btn-card-action"
              :title="t('areas.editArea')"
              :aria-label="t('areas.editArea')"
              @click="openEditModal(area)"
            >
              <Pencil :size="15" />
            </button>
            <button
              class="btn-card-action is-danger"
              :title="t('areas.deleteArea')"
              :aria-label="t('areas.deleteArea')"
              @click="handleDelete(area)"
            >
              <Trash2 :size="15" />
            </button>
          </div>
        </div>

        <div v-if="getDevicesForArea(area.id).length > 0" class="area-entities-preview">
          <div
            v-for="device in getDevicesForArea(area.id).slice(0, 4)"
            :key="device.id"
            class="entity-pill"
            :title="`${device.name} · ${device.integration_id}`"
          >
            <span class="pill-dot" :class="{ 'is-on': device.status === 'online' }" />
            <span class="pill-name">{{ device.name }}</span>
          </div>
          <span v-if="getDevicesForArea(area.id).length > 4" class="more-count">
            +{{ getDevicesForArea(area.id).length - 4 }} {{ t('areas.moreEntities') }}
          </span>
        </div>
        <div v-else-if="getEntitiesForArea(area.id).length > 0" class="area-entities-preview">
          <div
            v-for="entity in getEntitiesForArea(area.id).slice(0, 4)"
            :key="entity.id"
            class="entity-pill"
          >
            <span class="pill-dot" :class="{ 'is-on': entity.state === 'on' }" />
            <span class="pill-name">{{ entity.name }}</span>
          </div>
          <span v-if="getEntitiesForArea(area.id).length > 4" class="more-count">
            +{{ getEntitiesForArea(area.id).length - 4 }} {{ t('areas.moreEntities') }}
          </span>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="empty-state glass-card">
      <Layers :size="42" class="empty-icon" />
      <h3 class="empty-title">{{ t('areas.emptyTitle') }}</h3>
      <p class="empty-desc">{{ t('areas.emptyDesc') }}</p>
      <BaseButton variant="primary" class="empty-btn" @click="openNewModal">
        <Plus :size="16" />
        {{ t('areas.createFirst') }}
      </BaseButton>
    </div>

    <!-- Editor Modal -->
    <AreaEditorModal
      :is-open="isEditorOpen"
      :area="editingArea"
      @close="isEditorOpen = false"
      @save="handleSaveArea"
    />

    <div
      v-if="isDeviceManagerOpen && managingArea"
      class="modal-backdrop area-device-modal-backdrop"
      @click.self="closeDeviceManager"
    >
      <section
        class="area-device-modal glass-card"
        role="dialog"
        aria-modal="true"
        :aria-label="`${t('areas.manageDevices')} — ${managingArea.name}`"
      >
        <header class="area-device-modal-header">
          <div class="area-device-modal-heading">
            <h3>{{ t('areas.manageDevices') }} · {{ managingArea.name }}</h3>
            <p>{{ t('areas.manageDevicesDescription') }}</p>
          </div>
          <div class="area-device-header-actions">
            <button
              class="area-device-refresh-btn"
              type="button"
              :disabled="isLoadingAssignableDevices || isSavingAreaDevices"
              @click="refreshAssignableDevices"
            >
              <RefreshCw :size="14" />
              {{ t('devices.view.refresh') }}
            </button>
            <button class="btn-card-action" type="button" :aria-label="t('devices.view.close')" @click="closeDeviceManager">
              ×
            </button>
          </div>
        </header>

        <p v-if="isLoadingAssignableDevices" class="area-device-empty" role="status">
          {{ t('common.loading') }}
        </p>
        <p v-else-if="assignableDevices.length === 0" class="area-device-empty">
          {{ t('areas.noAssignableDevices') }}
        </p>
        <div v-else class="area-device-list">
          <label v-for="device in assignableDevices" :key="device.id" class="area-device-option">
            <input
              type="checkbox"
              :checked="selectedDeviceIds.includes(device.id)"
              @change="updateSelectedDevice(device.id, $event)"
            />
            <span class="area-device-option-copy">
              <strong>{{ device.name }}</strong>
              <small>
                {{ device.integration_id }}
                ·
                {{ device.area_id ? (areasStore.byId(device.area_id)?.name || device.area_id) : t('areas.unassigned') }}
              </small>
            </span>
          </label>
        </div>

        <footer class="area-device-modal-footer">
          <button class="area-device-secondary-btn" type="button" :disabled="isSavingAreaDevices" @click="closeDeviceManager">
            {{ t('devices.view.close') }}
          </button>
          <button class="area-device-primary-btn" type="button" :disabled="isSavingAreaDevices || isLoadingAssignableDevices || assignableDevices.length === 0" @click="saveAreaDevices">
            {{ isSavingAreaDevices ? t('areas.savingDevices') : t('areas.saveDevices') }}
          </button>
        </footer>
      </section>
    </div>
  </div>
</template>

<style scoped>
.areas-view {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 16px;
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

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.areas-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
}

.area-card {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
  transition: all var(--transition-normal);
  position: relative;
}

.area-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
}

.area-card-top {
  display: flex;
  align-items: center;
  gap: 16px;
}

.area-icon-box {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.2) 0%, rgba(99, 102, 241, 0.15) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
  flex-shrink: 0;
}

.area-info {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.area-name {
  font-size: 17px;
  font-weight: 600;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.area-count {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 2px;
}

.card-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
}

.btn-card-action {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-surface-elevated, rgba(255, 255, 255, 0.05));
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-card-action:hover {
  background: rgba(14, 165, 233, 0.15);
  border-color: rgba(14, 165, 233, 0.4);
  color: var(--primary);
}

.btn-card-action.is-danger:hover {
  background: var(--color-danger-bg);
  border-color: var(--color-danger);
  color: var(--color-danger);
}

.area-entities-preview {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}

.entity-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  font-size: 12px;
  color: var(--text-main);
}

.pill-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: var(--text-dim);
}

.pill-dot.is-on {
  background-color: #fbbf24;
  box-shadow: 0 0 6px rgba(251, 191, 36, 0.6);
}

.pill-name {
  max-width: 120px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.more-count {
  font-size: 12px;
  color: var(--text-dim);
  font-family: var(--font-mono);
}

.empty-state {
  padding: 48px 24px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  max-width: 520px;
  margin: 40px auto;
  border-radius: var(--radius-xl);
}

.empty-icon {
  color: var(--primary);
  margin-bottom: 16px;
  opacity: 0.85;
}

.empty-title {
  font-size: 19px;
  font-weight: 700;
  color: var(--text-main);
}

.empty-desc {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 8px;
  margin-bottom: 24px;
  line-height: 1.6;
}

.empty-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.manage-devices-action { color: var(--primary); }

.area-device-modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 1200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: rgba(2, 6, 23, .68);
  backdrop-filter: blur(5px);
}

.area-device-modal {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  width: min(100%, 560px);
  max-height: min(82vh, 720px);
  overflow: hidden;
  padding: 1.25rem;
  border: 1px solid var(--border-card);
  border-radius: var(--radius-lg);
  background: var(--bg-surface, #111827);
  box-shadow: var(--shadow-xl, 0 24px 70px rgba(0,0,0,.35));
}

.area-device-modal-header,
.area-device-modal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: .75rem;
}

.area-device-modal-heading {
  min-width: 0;
}

.area-device-modal-header h3 {
  color: var(--text-main);
  font-size: 1.05rem;
  font-weight: 700;
}

.area-device-header-actions {
  display: flex;
  align-items: center;
  gap: .45rem;
  flex: none;
}

.area-device-refresh-btn {
  display: inline-flex;
  align-items: center;
  gap: .4rem;
  min-height: 36px;
  padding: .4rem .65rem;
  color: var(--text-main);
  background: var(--bg-surface-elevated, rgba(148,163,184,.08));
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  font: inherit;
  font-size: .8rem;
  font-weight: 600;
  cursor: pointer;
}

.area-device-refresh-btn:disabled {
  opacity: .55;
  cursor: not-allowed;
}

.area-device-modal-header p,
.area-device-empty {
  color: var(--text-muted);
  font-size: .85rem;
  margin-top: .3rem;
}

.area-device-list {
  display: grid;
  gap: .55rem;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: .1rem .15rem .1rem 0;
}

.area-device-option {
  display: flex;
  align-items: center;
  gap: .75rem;
  min-width: 0;
  padding: .75rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--bg-surface-elevated, rgba(148,163,184,.06));
  cursor: pointer;
}

.area-device-option input {
  flex: none;
  width: 1rem;
  height: 1rem;
  accent-color: var(--primary);
}

.area-device-option-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: .2rem;
}

.area-device-option-copy strong {
  color: var(--text-main);
  font-size: .9rem;
  overflow-wrap: anywhere;
}

.area-device-option-copy small {
  color: var(--text-muted);
  font-size: .76rem;
  overflow-wrap: anywhere;
}

.area-device-modal-footer { justify-content: flex-end; }

.area-device-modal-footer button {
  min-height: 40px;
  padding: .55rem .9rem;
  border-radius: var(--radius-md);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}

.area-device-secondary-btn {
  color: var(--text-main);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
}

.area-device-primary-btn {
  color: white;
  background: var(--primary);
  border: 1px solid var(--primary);
}

.area-device-modal-footer button:disabled {
  opacity: .55;
  cursor: not-allowed;
}

@media (max-width: 480px) {
  .area-device-modal { padding: 1rem; }
  .areas-grid { grid-template-columns: minmax(0, 1fr); }
}
</style>
