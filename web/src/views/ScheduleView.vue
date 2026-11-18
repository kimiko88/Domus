<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Calendar as CalendarIcon,
  Clock,
  Flame,
  Plus,
  Upload,
  CheckCircle,
  Thermometer,
  Power,
  Trash2,
  Copy,
  Save,
  Sun,
  Moon,
  Sparkles,
  X
} from '@lucide/vue'
import { apiClient } from '../api/client'
import { useEntitiesStore } from '../stores/entities'
import { useNotificationsStore } from '../stores/notifications'

const { t, locale } = useI18n()
const entitiesStore = useEntitiesStore()
const notifStore = useNotificationsStore()

export interface TimeSlot {
  start: string
  end: string
  value: number
}

export interface Schedule {
  id: string
  name: string
  entity_key: string
  enabled: boolean
  grid: Record<string, TimeSlot[]>
}

export interface CalendarEvent {
  id: string
  summary: string
  description?: string
  start_time: string
  end_time: string
  location?: string
  source: string
}

interface TuyaStatusPoint {
  code: string
  value: unknown
}

interface TuyaCloudDeviceItem {
  id: string
  name: string
  product_name?: string
  category?: string
  inferred_category?: string
  entity_key?: string
  online?: boolean
  local_configured?: boolean
  local_active?: boolean
  local_ip?: string
  status?: TuyaStatusPoint[]
}

interface TuyaLocalDeviceItem {
  device_id: string
  name: string
  ip?: string
  category?: string
  entity_key?: string
  connected?: boolean
}

export interface ThermostatOption {
  id: string
  entity_id?: string
  tuya_device_id?: string
  name: string
  source: 'entity' | 'tuya_local' | 'tuya_cloud'
  online?: boolean
  local_configured?: boolean
  current_temperature?: number
  target_temperature?: number
  hvac_mode?: 'heat' | 'auto' | 'off'
  temp_dp_code?: string
  temp_scale?: number
  mode_dp_code?: string
  switch_dp_code?: string
}

const activeTab = ref<'cronotermostato' | 'calendar'>('cronotermostato')

const daysOfWeek = computed(() => [
  { key: 'monday', label: t('schedules.view.days.monday') },
  { key: 'tuesday', label: t('schedules.view.days.tuesday') },
  { key: 'wednesday', label: t('schedules.view.days.wednesday') },
  { key: 'thursday', label: t('schedules.view.days.thursday') },
  { key: 'friday', label: t('schedules.view.days.friday') },
  { key: 'saturday', label: t('schedules.view.days.saturday') },
  { key: 'sunday', label: t('schedules.view.days.sunday') }
])

const schedules = ref<Schedule[]>([])
const selectedScheduleId = ref<string>('')
const calendarEvents = ref<CalendarEvent[]>([])
const icsUploadStatus = ref('')
const isSavingSchedule = ref(false)
const tuyaCloudDevices = ref<TuyaCloudDeviceItem[]>([])
const tuyaLocalDevices = ref<TuyaLocalDeviceItem[]>([])

const CLIMATE_CATEGORIES = new Set([
  'wk', 'wkf', 'bgl', 'qn', 'kt', 'wkcz', 'ktkzq', 'rdn', 'cn', 'rs', 'fskg',
  'climate', 'thermostat', 'heater', 'boiler', 'trv', 'hvac', 'ac'
])

const CLIMATE_DP_CODES = new Set([
  'temp_set', 'temp_current', 'upper_temp', 'lower_temp', 'target_temp', 'current_temp',
  'target_temperature', 'current_temperature', 'work_state', 'window_check', 'frost',
  'valve_state', 'hvac_mode', 'set_temp', 'va_temperature'
])

const CLIMATE_KEYWORDS = [
  'termostato', 'thermostat', 'cronotermostato', 'caldaia', 'boiler',
  'clima', 'condizionatore', 'valvola termostatica', 'trv', 'riscaldamento', 'heater'
]

function isClimateDeviceCandidate(
  name?: string,
  productName?: string,
  category?: string,
  inferredCategory?: string,
  entityKey?: string,
  status?: TuyaStatusPoint[]
): boolean {
  if (inferredCategory?.toLowerCase() === 'climate') return true
  if (entityKey?.toLowerCase().startsWith('climate.')) return true
  if (category && CLIMATE_CATEGORIES.has(category.toLowerCase().trim())) return true
  if (status && status.some((s) => CLIMATE_DP_CODES.has(s.code.toLowerCase().trim()))) return true
  const text = `${name || ''} ${productName || ''}`.toLowerCase()
  return CLIMATE_KEYWORDS.some((kw) => text.includes(kw))
}

function extractTuyaClimateTelemetry(status?: TuyaStatusPoint[]) {
  let current_temperature: number | undefined
  let target_temperature: number | undefined
  let hvac_mode: 'heat' | 'auto' | 'off' | undefined
  let temp_dp_code: string | undefined
  let temp_scale = 0
  let mode_dp_code: string | undefined
  let switch_dp_code: string | undefined

  for (const pt of status || []) {
    const code = pt.code.toLowerCase().trim()
    if (['switch', 'power', 'switch_1'].includes(code) && typeof pt.value === 'boolean') {
      switch_dp_code = pt.code
      if (!pt.value) hvac_mode = 'off'
      else if (!hvac_mode) hvac_mode = 'heat'
    } else if (['mode', 'work_mode', 'hvac_mode'].includes(code) && typeof pt.value === 'string') {
      mode_dp_code = pt.code
      const val = pt.value.toLowerCase()
      if (val === 'auto' || val === 'automatic' || val === 'program') hvac_mode = 'auto'
      else if (val === 'off') hvac_mode = 'off'
      else hvac_mode = 'heat'
    } else if (['temp_set', 'target_temp', 'target_temperature', 'upper_temp', 'set_temp'].includes(code)) {
      const num = Number(pt.value)
      if (!isNaN(num)) {
        temp_dp_code = pt.code
        if (num > 50) {
          temp_scale = 1
          target_temperature = num / 10.0
        } else {
          temp_scale = 0
          target_temperature = num
        }
      }
    } else if (['temp_current', 'current_temp', 'current_temperature', 'va_temperature', 'lower_temp'].includes(code)) {
      const num = Number(pt.value)
      if (!isNaN(num)) {
        current_temperature = num > 50 ? num / 10.0 : num
      }
    }
  }

  return {
    current_temperature,
    target_temperature,
    hvac_mode,
    temp_dp_code,
    temp_scale,
    mode_dp_code,
    switch_dp_code
  }
}

// Connected Smart Thermostats merged from Entities Store, Tuya Local, and Tuya Cloud (Smart Life)
const mergedThermostats = computed<ThermostatOption[]>(() => {
  const byKey = new Map<string, ThermostatOption>()

  // 1. Entities from store
  for (const e of Object.values(entitiesStore.entities)) {
    const entityKey = e.entity_key || e.id
    const attrs = e.attributes || {}
    const isClimate =
      e.domain === 'climate' ||
      entityKey.startsWith('climate.') ||
      e.id.startsWith('climate.') ||
      attrs.target_temperature !== undefined ||
      attrs.current_temperature !== undefined ||
      attrs.hvac_mode !== undefined ||
      isClimateDeviceCandidate(e.name, undefined, e.domain, undefined, entityKey)

    if (!isClimate) continue

    let tuyaDevId: string | undefined
    if (e.device_id?.startsWith('tuya_')) {
      tuyaDevId = e.device_id.slice('tuya_'.length)
    } else if (entityKey.includes('.tuya_')) {
      tuyaDevId = entityKey.split('.tuya_')[1]
    }

    const canonicalId = entityKey.startsWith('climate.') || !tuyaDevId
      ? entityKey
      : `climate.tuya_${tuyaDevId}`

    const curTemp = attrs.current_temperature !== undefined
      ? Number(attrs.current_temperature)
      : attrs.temperature !== undefined
        ? Number(attrs.temperature)
        : !isNaN(parseFloat(e.state))
          ? parseFloat(e.state)
          : undefined

    const tgtTemp = attrs.target_temperature !== undefined
      ? Number(attrs.target_temperature)
      : attrs.temperature !== undefined
        ? Number(attrs.temperature)
        : undefined

    const mode: 'heat' | 'auto' | 'off' | undefined =
      e.state === 'heat' || e.state === 'auto' || e.state === 'off'
        ? e.state
        : attrs.hvac_mode === 'auto' || attrs.hvac_mode === 'off' || attrs.hvac_mode === 'heat'
          ? attrs.hvac_mode
          : undefined

    byKey.set(canonicalId, {
      id: canonicalId,
      entity_id: e.id,
      tuya_device_id: tuyaDevId,
      name: e.name,
      source: 'entity',
      online: true,
      current_temperature: curTemp,
      target_temperature: tgtTemp,
      hvac_mode: mode
    })
  }

  // 2. Tuya Local configured devices
  for (const d of tuyaLocalDevices.value) {
    const isClimate = isClimateDeviceCandidate(d.name, undefined, d.category, undefined, d.entity_key)
    if (!isClimate) continue

    const canonicalId = d.entity_key?.startsWith('climate.')
      ? d.entity_key
      : `climate.tuya_${d.device_id}`

    const existing = byKey.get(canonicalId)
    if (existing) {
      existing.tuya_device_id = d.device_id
      existing.local_configured = true
      if (!existing.name && d.name) existing.name = d.name
    } else {
      byKey.set(canonicalId, {
        id: canonicalId,
        tuya_device_id: d.device_id,
        name: d.name || `Termostato ${d.device_id}`,
        source: 'tuya_local',
        online: d.connected ?? true,
        local_configured: true
      })
    }
  }

  // 3. Tuya Cloud (Smart Life) devices
  for (const d of tuyaCloudDevices.value) {
    const isClimate = isClimateDeviceCandidate(
      d.name,
      d.product_name,
      d.category,
      d.inferred_category,
      d.entity_key,
      d.status
    )
    if (!isClimate) continue

    const canonicalId = d.entity_key?.startsWith('climate.')
      ? d.entity_key
      : `climate.tuya_${d.id}`

    const telemetry = extractTuyaClimateTelemetry(d.status)
    const existing = byKey.get(canonicalId)
    if (existing) {
      existing.tuya_device_id = d.id
      existing.online = d.online ?? existing.online
      existing.local_configured = d.local_configured ?? existing.local_configured
      if (telemetry.current_temperature !== undefined) existing.current_temperature = telemetry.current_temperature
      if (telemetry.target_temperature !== undefined) existing.target_temperature = telemetry.target_temperature
      if (telemetry.hvac_mode !== undefined) existing.hvac_mode = telemetry.hvac_mode
      existing.temp_dp_code = telemetry.temp_dp_code
      existing.temp_scale = telemetry.temp_scale
      existing.mode_dp_code = telemetry.mode_dp_code
      existing.switch_dp_code = telemetry.switch_dp_code
    } else {
      byKey.set(canonicalId, {
        id: canonicalId,
        tuya_device_id: d.id,
        name: d.name || d.product_name || `Termostato Smart Life (${d.id})`,
        source: 'tuya_cloud',
        online: d.online ?? true,
        local_configured: d.local_configured,
        ...telemetry
      })
    }
  }

  return Array.from(byKey.values())
})

// Additional Smart Life / Tuya devices (e.g. boiler relays or custom-named thermostats)
// so the user can always select any integrated Smart Life device if needed
const otherSmartLifeDevices = computed<ThermostatOption[]>(() => {
  const existingTuyaIds = new Set(
    mergedThermostats.value.map((t) => t.tuya_device_id).filter(Boolean)
  )
  const existingKeys = new Set(mergedThermostats.value.map((t) => t.id))
  const list: ThermostatOption[] = []

  for (const d of tuyaCloudDevices.value) {
    const key = d.entity_key || `climate.tuya_${d.id}`
    if (existingTuyaIds.has(d.id) || existingKeys.has(key)) continue
    const telemetry = extractTuyaClimateTelemetry(d.status)
    list.push({
      id: key,
      tuya_device_id: d.id,
      name: d.name || d.product_name || d.id,
      source: 'tuya_cloud',
      online: d.online ?? true,
      local_configured: d.local_configured,
      ...telemetry
    })
    existingTuyaIds.add(d.id)
    existingKeys.add(key)
  }

  for (const d of tuyaLocalDevices.value) {
    const key = d.entity_key || `climate.tuya_${d.device_id}`
    if (existingTuyaIds.has(d.device_id) || existingKeys.has(key)) continue
    list.push({
      id: key,
      tuya_device_id: d.device_id,
      name: d.name || d.device_id,
      source: 'tuya_local',
      online: d.connected ?? true,
      local_configured: true
    })
    existingTuyaIds.add(d.device_id)
    existingKeys.add(key)
  }

  return list
})

const allSelectableThermostats = computed<ThermostatOption[]>(() => [
  ...mergedThermostats.value,
  ...otherSmartLifeDevices.value
])

function findThermostatOption(idOrKey: string): ThermostatOption | undefined {
  return allSelectableThermostats.value.find(
    (t) => t.id === idOrKey || t.entity_id === idOrKey || (t.tuya_device_id && `climate.tuya_${t.tuya_device_id}` === idOrKey)
  )
}

// Active regulated thermostat (either from connectedThermostats or schedule target)
const activeThermostatId = ref<string>('')

// Live thermostat regulation state
const liveTargetTemp = ref<number>(20.5)
const liveHvacMode = ref('')
const activeThermostat = computed(() => {
  const option = findThermostatOption(activeThermostatId.value)
  return entitiesStore.entities[activeThermostatId.value] ||
    (option?.entity_id ? entitiesStore.entities[option.entity_id] : undefined)
})
const activeThermostatEntityKey = computed(() =>
  activeThermostat.value?.entity_key ||
  findThermostatOption(activeThermostatId.value)?.id ||
  activeThermostatId.value ||
  ''
)
const supportedHvacModes = computed<string[]>(() => {
  const modes = activeThermostat.value?.attributes?.hvac_modes
  if (Array.isArray(modes)) {
    return modes.filter((mode): mode is string => typeof mode === 'string')
  }

  const option = findThermostatOption(activeThermostatId.value)
  const fallback: string[] = []
  if (option?.mode_dp_code) fallback.push('heat', 'auto')
  if (option?.switch_dp_code) fallback.push('off')
  return fallback
})
const supportsSetpoint = computed(() => {
  const attrs = activeThermostat.value?.attributes
  const entityHasSetpoint = !!attrs && ['target_temperature', 'temperature', 'target_temp'].some(
    key => attrs[key] != null && Number.isFinite(Number(attrs[key]))
  )
  const option = findThermostatOption(activeThermostatId.value)
  const optionHasSetpoint = option?.target_temperature != null &&
    Number.isFinite(Number(option.target_temperature))
  return entityHasSetpoint || optionHasSetpoint
})
const setpointMin = computed(() => {
  const value = Number(activeThermostat.value?.attributes?.min_temp ?? 5)
  return Number.isFinite(value) ? value : 5
})
const setpointMax = computed(() => {
  const value = Number(activeThermostat.value?.attributes?.max_temp ?? 35)
  return Number.isFinite(value) ? value : 35
})
const manualOverrideDuration = ref(0)
const isAdjustingLive = ref(false)

// Modal for adding a time slot
const showAddSlotModal = ref(false)
const targetSlotDay = ref('monday')
const newSlotForm = ref<TimeSlot>({
  start: '06:30',
  end: '08:30',
  value: 21.0
})

const selectedSchedule = computed(() => {
  return schedules.value.find((s) => s.id === selectedScheduleId.value) || null
})
const selectedScheduleThermostatAvailable = computed(() => {
  const target = selectedSchedule.value?.entity_key
  return !!target && !!findThermostatOption(target)
})

// Current room temperature from the best available source: merged Tuya
// telemetry first, then the persisted Domus entity.
const currentAmbientTemp = computed(() => {
  if (!activeThermostatId.value) return 20.8
  const option = findThermostatOption(activeThermostatId.value)
  if (option?.current_temperature != null && Number.isFinite(Number(option.current_temperature))) {
    return Number(option.current_temperature)
  }

  const entity = activeThermostat.value
  const value = entity?.attributes?.current_temperature ?? entity?.attributes?.temperature
  if (value != null && Number.isFinite(Number(value))) return Number(value)

  const parsed = Number.parseFloat(entity?.state || '')
  return Number.isFinite(parsed) ? parsed : 20.8
})


watchThermostatChange()

function watchThermostatChange() {
  const scheduleTarget = selectedSchedule.value?.entity_key
  if (scheduleTarget) {
    const scheduledOption = findThermostatOption(scheduleTarget)
    if (scheduledOption) {
      // Older schedules may have stored the entity UUID in the entity_key
      // column. Normalize it to the stable key so both selectors stay aligned.
      if (selectedSchedule.value && selectedSchedule.value.entity_key !== scheduledOption.id) {
        selectedSchedule.value.entity_key = scheduledOption.id
      }
      activeThermostatId.value = scheduledOption.entity_id || scheduledOption.id
      syncLiveThermostatValues()
      return
    }
  }

  const candidates = allSelectableThermostats.value
  if (candidates.length === 0) return

  const isPlaceholder =
    !activeThermostatId.value ||
    activeThermostatId.value === 'climate.termostato_principale' ||
    activeThermostatId.value === 'climate.termostato_salotto'

  if (isPlaceholder) {
    activeThermostatId.value = candidates[0].entity_id || candidates[0].id
    syncLiveThermostatValues()
  }

  if (
    selectedSchedule.value &&
    (!selectedSchedule.value.entity_key ||
      selectedSchedule.value.entity_key === 'climate.termostato_principale' ||
      selectedSchedule.value.entity_key === 'climate.termostato_salotto')
  ) {
    selectedSchedule.value.entity_key = activeThermostatEntityKey.value ||
      activeThermostatId.value
  }
}

function handleThermostatSelection() {
  syncLiveThermostatValues()
  if (selectedSchedule.value && activeThermostatId.value) {
    selectedSchedule.value.entity_key =
      activeThermostatEntityKey.value || activeThermostatId.value
  }
}

function syncThermostatFromSchedule() {
  const target = selectedSchedule.value?.entity_key
  if (!target) return

  const option = findThermostatOption(target)
  if (!option) return

  // Normalize legacy schedules that stored an entity UUID instead of its key.
  if (selectedSchedule.value && selectedSchedule.value.entity_key !== option.id) {
    selectedSchedule.value.entity_key = option.id
  }
  activeThermostatId.value = option.entity_id || option.id
  syncLiveThermostatValues()
}

watch(selectedScheduleId, syncThermostatFromSchedule)

function syncLiveThermostatValues() {
  if (!activeThermostatId.value) return

  const option = findThermostatOption(activeThermostatId.value)
  const entity = activeThermostat.value
  const target = option?.target_temperature ??
    entity?.attributes?.target_temperature ??
    entity?.attributes?.temperature ??
    entity?.attributes?.target_temp

  if (target != null && Number.isFinite(Number(target))) {
    liveTargetTemp.value = Number(target)
  }

  const currentMode = String(
    option?.hvac_mode ??
    entity?.attributes?.hvac_mode ??
    entity?.state ??
    ''
  )
  const supportedModes = supportedHvacModes.value
  if (supportedModes.length === 0 || supportedModes.includes(currentMode)) {
    liveHvacMode.value = currentMode
  } else {
    liveHvacMode.value = supportedModes[0] || ''
  }
}

function adjustLiveTemp(delta: number) {
  liveTargetTemp.value = Math.round((liveTargetTemp.value + delta) * 2) / 2
}

function setLivePreset(temp: number) {
  liveTargetTemp.value = Math.min(setpointMax.value, Math.max(setpointMin.value, temp))
}

async function applyLiveThermostatSettings() {
  if (!activeThermostatId.value) {
    notifStore.error(t('schedules.view.noThermostatTitle'), t('schedules.view.noThermostatMessage'))
    return
  }

  try {
    isAdjustingLive.value = true
    const option = findThermostatOption(activeThermostatId.value)
    const entity = activeThermostat.value
    const duration = Number(manualOverrideDuration.value)

    if (!Number.isInteger(duration) || duration < 0 || duration > 1440) {
      throw new Error(t('schedules.view.invalidOverrideDuration'))
    }
    if (!supportsSetpoint.value) {
      throw new Error(t('schedules.view.noProgrammableSetpoint'))
    }
    if (!Number.isFinite(liveTargetTemp.value) ||
      liveTargetTemp.value < setpointMin.value ||
      liveTargetTemp.value > setpointMax.value) {
      throw new Error(t('schedules.view.targetTemperatureUnsupported'))
    }

    if (entity) {
      // The backend owns the restore timer and dispatches through the Domus
      // entity bus, so the override survives navigation and page closure.
      await apiClient.post(
        '/api/v1/entities/' + encodeURIComponent(entity.id) + '/climate-profile',
        {
          profile: 'custom',
          temperature: liveTargetTemp.value,
          duration_minutes: duration
        }
      )

      if (liveHvacMode.value && supportedHvacModes.value.includes(liveHvacMode.value)) {
        await entitiesStore.callService(entity.domain || 'climate', 'set_hvac_mode', entity.id, {
          hvac_mode: liveHvacMode.value
        })
      }
    } else if (option?.tuya_device_id && option.source === 'tuya_cloud') {
      // A cloud-only device has no persisted Domus entity for the backend
      // restore scheduler. Permit immediate control, but never promise a timer.
      if (duration > 0) {
        throw new Error(t('schedules.view.timedRestoreRequiresEntity'))
      }

      const commands: Array<{ code: string; value: unknown }> = []
      if (liveHvacMode.value === 'off') {
        if (!option.switch_dp_code) {
          throw new Error(t('schedules.view.tuyaOffCommandUnavailable'))
        }
        commands.push({ code: option.switch_dp_code, value: false })
      } else {
        if (option.switch_dp_code) {
          commands.push({ code: option.switch_dp_code, value: true })
        }
        const temperatureCode = option.temp_dp_code || 'temp_set'
        const scaleFactor = option.temp_scale === 1 ? 10 : 1
        commands.push({
          code: temperatureCode,
          value: Math.round(liveTargetTemp.value * scaleFactor)
        })
        if (option.mode_dp_code && (liveHvacMode.value === 'auto' || liveHvacMode.value === 'heat')) {
          commands.push({
            code: option.mode_dp_code,
            value: liveHvacMode.value === 'auto' ? 'auto' : 'manual'
          })
        }
      }

      await apiClient.post(
        '/api/v1/integrations/tuya-cloud/devices/' +
          encodeURIComponent(option.tuya_device_id) + '/commands',
        { commands }
      )
    } else if (option?.tuya_device_id && option.source === 'tuya_local') {
      if (duration > 0) {
        throw new Error(t('schedules.view.timedRestoreRequiresEntity'))
      }
      await apiClient.post(
        '/api/v1/integrations/tuya-local/devices/' +
          encodeURIComponent(option.tuya_device_id) + '/command',
        {
          command: liveHvacMode.value === 'off' ? 'turn_off' : 'set_temperature',
          params: { temperature: liveTargetTemp.value, hvac_mode: liveHvacMode.value }
        }
      )
    } else {
      throw new Error(t('schedules.view.deviceNotControllable'))
    }

    if (option) {
      option.target_temperature = liveTargetTemp.value
      option.hvac_mode = liveHvacMode.value as ThermostatOption['hvac_mode']
    }

    notifStore.success(
      t('schedules.view.commandSentTitle'),
      duration === 0
        ? t('schedules.view.commandUntilNextChange', { temperature: liveTargetTemp.value.toFixed(1) })
        : t('schedules.view.commandForDuration', { temperature: liveTargetTemp.value.toFixed(1), duration })
    )
  } catch {
    notifStore.error(t('schedules.view.adjustmentErrorTitle'), t('schedules.view.adjustmentErrorMessage'))
  } finally {
    isAdjustingLive.value = false
  }
}

// Slot Management
function openAddSlot(dayKey: string) {
  targetSlotDay.value = dayKey
  newSlotForm.value = {
    start: '07:00',
    end: '09:00',
    value: Math.min(setpointMax.value, Math.max(setpointMin.value, liveTargetTemp.value))
  }
  showAddSlotModal.value = true
}

function saveSlot() {
  if (!selectedSchedule.value) return
  const sched = selectedSchedule.value
  const toMinutes = (time: string) => {
    const match = /^(\d{2}):(\d{2})$/.exec(time)
    return match ? Number(match[1]) * 60 + Number(match[2]) : null
  }
  const start = toMinutes(newSlotForm.value.start)
  const end = toMinutes(newSlotForm.value.end)
  if (start === null || end === null || start >= end) {
    notifStore.error(t('schedules.view.invalidTimeTitle'), t('schedules.view.invalidTimeMessage'))
    return
  }
  if (!Number.isFinite(newSlotForm.value.value) || newSlotForm.value.value < setpointMin.value || newSlotForm.value.value > setpointMax.value) {
    notifStore.error(t('schedules.view.invalidSetpointTitle'), t('schedules.view.invalidSetpointMessage', { min: setpointMin.value, max: setpointMax.value }))
    return
  }
  const existingSlots = sched.grid[targetSlotDay.value] || []
  const overlaps = existingSlots.some(slot => {
    const slotStart = toMinutes(slot.start)
    const slotEnd = toMinutes(slot.end)
    return slotStart !== null && slotEnd !== null && start < slotEnd && slotStart < end
  })
  if (overlaps) {
    notifStore.error(t('schedules.view.overlappingSlotsTitle'), t('schedules.view.overlappingSlotsMessage'))
    return
  }
  if (!sched.grid[targetSlotDay.value]) {
    sched.grid[targetSlotDay.value] = []
  }

  sched.grid[targetSlotDay.value].push({
    start: newSlotForm.value.start,
    end: newSlotForm.value.end,
    value: Number(newSlotForm.value.value)
  })

  // Sort slots by start time
  sched.grid[targetSlotDay.value].sort((a, b) => a.start.localeCompare(b.start))

  showAddSlotModal.value = false
  notifStore.success(t('schedules.view.timeSlotAddedTitle'), t('schedules.view.timeSlotAddedMessage', { start: newSlotForm.value.start, end: newSlotForm.value.end, temperature: newSlotForm.value.value, day: t(`schedules.view.days.${targetSlotDay.value}`) }))
}

function removeSlot(dayKey: string, index: number) {
  if (!selectedSchedule.value) return
  const sched = selectedSchedule.value
  if (sched.grid[dayKey]) {
    sched.grid[dayKey].splice(index, 1)
  }
}

// Preset Profiles for Schedule
function applyWinterWorkingProfile() {
  if (!selectedSchedule.value) return
  const sched = selectedSchedule.value

  const workdaySlots: TimeSlot[] = [
    { start: '06:00', end: '08:30', value: 21.0 }, // Sveglia comfort
    { start: '08:30', end: '17:30', value: 18.5 }, // Lavoro eco
    { start: '17:30', end: '22:30', value: 21.5 }, // Rientro serale
    { start: '22:30', end: '06:00', value: 17.5 }  // Notte
  ]

  const weekendSlots: TimeSlot[] = [
    { start: '07:30', end: '23:00', value: 21.0 }, // Giorno comfort continuo
    { start: '23:00', end: '07:30', value: 17.5 }  // Notte
  ]

  sched.grid = {
    monday: JSON.parse(JSON.stringify(workdaySlots)),
    tuesday: JSON.parse(JSON.stringify(workdaySlots)),
    wednesday: JSON.parse(JSON.stringify(workdaySlots)),
    thursday: JSON.parse(JSON.stringify(workdaySlots)),
    friday: JSON.parse(JSON.stringify(workdaySlots)),
    saturday: JSON.parse(JSON.stringify(weekendSlots)),
    sunday: JSON.parse(JSON.stringify(weekendSlots))
  }

  notifStore.success(t('schedules.view.profileAppliedTitle'), t('schedules.view.profileAppliedMessage'))
}

function copyMondayToWorkdays() {
  if (!selectedSchedule.value) return
  const sched = selectedSchedule.value
  const mondaySlots = sched.grid['monday'] || []

  sched.grid['tuesday'] = JSON.parse(JSON.stringify(mondaySlots))
  sched.grid['wednesday'] = JSON.parse(JSON.stringify(mondaySlots))
  sched.grid['thursday'] = JSON.parse(JSON.stringify(mondaySlots))
  sched.grid['friday'] = JSON.parse(JSON.stringify(mondaySlots))

  notifStore.success(t('schedules.view.copiedTitle'), t('schedules.view.copyMondayToWorkdaysMessage'))
}

function copySaturdayToSunday() {
  if (!selectedSchedule.value) return
  const sched = selectedSchedule.value
  const satSlots = sched.grid['saturday'] || []

  sched.grid['sunday'] = JSON.parse(JSON.stringify(satSlots))
  notifStore.success(t('schedules.view.copiedTitle'), t('schedules.view.copySaturdayToSundayMessage'))
}

async function saveCurrentSchedule() {
  if (!selectedSchedule.value) return
  const schedule = selectedSchedule.value
  const targetEntityKey = schedule.entity_key ||
    activeThermostatEntityKey.value ||
    'climate.termostato_principale'

  schedule.entity_key = targetEntityKey

  try {
    isSavingSchedule.value = true
    await apiClient.put('/api/v1/schedules/' + encodeURIComponent(schedule.id), {
      name: schedule.name,
      entity_key: targetEntityKey,
      enabled: schedule.enabled,
      grid: schedule.grid
    })
    notifStore.success(
      t('schedules.view.scheduleSavedTitle'),
      t('schedules.view.scheduleSavedMessage', { name: schedule.name })
    )
  } catch {
    notifStore.error(t('schedules.view.scheduleSaveErrorTitle'), t('schedules.view.scheduleSaveErrorMessage'))
  } finally {
    isSavingSchedule.value = false
  }
}

async function createNewSchedule() {
  const newSchedName = t('schedules.view.newScheduleName', { number: schedules.value.length + 1 })
  const targetEntity = activeThermostatEntityKey.value || 'climate.termostato_principale'

  try {
    const created = await apiClient.post<Schedule>('/api/v1/schedules', {
      name: newSchedName,
      entity_key: targetEntity,
      enabled: true,
      grid: {
        monday: [{ start: '06:30', end: '08:30', value: 21.0 }, { start: '17:30', end: '22:30', value: 21.0 }],
        tuesday: [{ start: '06:30', end: '08:30', value: 21.0 }, { start: '17:30', end: '22:30', value: 21.0 }],
        wednesday: [{ start: '06:30', end: '08:30', value: 21.0 }, { start: '17:30', end: '22:30', value: 21.0 }],
        thursday: [{ start: '06:30', end: '08:30', value: 21.0 }, { start: '17:30', end: '22:30', value: 21.0 }],
        friday: [{ start: '06:30', end: '08:30', value: 21.0 }, { start: '17:30', end: '22:30', value: 21.0 }],
        saturday: [{ start: '08:00', end: '23:00', value: 21.0 }],
        sunday: [{ start: '08:00', end: '23:00', value: 21.0 }]
      }
    })

    if (created) {
      schedules.value.push(created)
      selectedScheduleId.value = created.id
      notifStore.success(t('schedules.view.newThermostatTitle'), t('schedules.view.newThermostatMessage', { name: created.name }))
    }
  } catch {
    // Fallback if offline
    const fallbackSched: Schedule = {
      id: 'sched_' + Date.now(),
      name: newSchedName,
      entity_key: targetEntity,
      enabled: true,
      grid: {
        monday: [{ start: '06:30', end: '08:30', value: 21.0 }, { start: '17:30', end: '22:30', value: 21.0 }]
      }
    }
    schedules.value.push(fallbackSched)
    selectedScheduleId.value = fallbackSched.id
  }
}

async function fetchTuyaThermostats() {
  const [localRes, cloudRes] = await Promise.allSettled([
    apiClient.get<{ devices: TuyaLocalDeviceItem[] }>('/api/v1/integrations/tuya-local/devices'),
    apiClient.get<{ devices: TuyaCloudDeviceItem[] }>('/api/v1/integrations/tuya-cloud/devices')
  ])

  if (localRes.status === 'fulfilled' && localRes.value?.devices) {
    tuyaLocalDevices.value = localRes.value.devices
  }
  if (cloudRes.status === 'fulfilled' && cloudRes.value?.devices) {
    tuyaCloudDevices.value = cloudRes.value.devices
  }
}

// iCal Upload Handler
async function handleICalUpload(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return

  const text = await file.text()
  try {
    await apiClient.post('/api/v1/calendar/import', text)
    icsUploadStatus.value = t('schedules.view.calendarImportSuccess')
    setTimeout(() => (icsUploadStatus.value = ''), 4000)
    const e = await apiClient.get<CalendarEvent[]>('/api/v1/calendar/events')
    calendarEvents.value = e || []
  } catch {
    icsUploadStatus.value = t('schedules.view.calendarImportError')
  }
}

onMounted(async () => {
  await Promise.allSettled([
    entitiesStore.fetchEntities(),
    fetchTuyaThermostats()
  ])

  watchThermostatChange()

  try {
    const s = await apiClient.get<Schedule[]>('/api/v1/schedules')
    schedules.value = s || []
    if (schedules.value.length > 0) {
      selectedScheduleId.value = schedules.value[0].id
    }
  } catch {
    schedules.value = []
  }

  // If no schedules exist, generate an initial default one so the user has immediate visual controls
  if (schedules.value.length === 0) {
    await createNewSchedule()
  }

  watchThermostatChange()

  try {
    const e = await apiClient.get<CalendarEvent[]>('/api/v1/calendar/events')
    calendarEvents.value = e || []
  } catch {
    calendarEvents.value = []
  }
})
</script>

<template>
  <div class="schedule-view-container">
    <header class="schedule-header glass-panel">
      <div class="header-title-box">
        <div class="icon-circle">
          <Flame :size="26" class="text-flame" />
        </div>
        <div>
          <h1 class="page-title">{{ t('schedules.title') }}</h1>
          <p class="page-subtitle">{{ t('schedules.subtitle') }}</p>
        </div>
      </div>

      <div class="tab-selector" role="tablist" :aria-label="t('schedules.view.tabSelector')">
        <button
          :class="['tab-btn', { active: activeTab === 'cronotermostato' }]"
          @click="activeTab = 'cronotermostato'"
          :aria-label="t('schedules.view.weeklyTabAria')"
        >
          <Flame :size="16" />
          <span>{{ t('schedules.view.weeklyTab') }}</span>
        </button>
        <button
          :class="['tab-btn', { active: activeTab === 'calendar' }]"
          @click="activeTab = 'calendar'"
          :aria-label="t('schedules.view.calendarTabAria')"
        >
          <CalendarIcon :size="16" />
          <span>{{ t('schedules.view.calendarEventsTab') }}</span>
        </button>
      </div>
    </header>

    <!-- Tab 1: Cronotermostato Live & Weekly Matrix -->
    <div v-if="activeTab === 'cronotermostato'" class="cronotermostato-layout">
      <!-- SECTION 1: LIVE SMART THERMOSTAT REGULATOR WIDGET -->
      <section class="live-thermostat-card glass-panel" :aria-label="t('schedules.view.liveThermostatAria')">
        <div class="thermostat-top-row">
          <div class="thermostat-title-box">
            <Thermometer :size="20" class="text-primary" />
            <h3>{{ t('schedules.view.liveThermostatTitle') }}</h3>
          </div>

          <!-- Selector of Connected Thermostats -->
          <div class="thermostat-selector-wrap">
            <label for="thermostat-select">{{ t('schedules.view.deviceLabel') }}</label>
            <select
              id="thermostat-select"
              v-model="activeThermostatId"
              class="thermostat-select"
              @change="handleThermostatSelection"
            >
              <option
                v-for="thermostat in allSelectableThermostats"
                :key="thermostat.id"
                :value="thermostat.entity_id || thermostat.id"
              >
                🌡️ {{ thermostat.name }} ({{ thermostat.id }})
              </option>
              <option v-if="allSelectableThermostats.length === 0" value="climate.termostato_salotto">
                🌡️ {{ t('schedules.view.simulatedThermostat') }}
              </option>
            </select>
          </div>
        </div>

        <div class="thermostat-dial-grid">
          <!-- Current Reading -->
          <div class="temp-display-block current">
            <span class="temp-sublabel">{{ t('schedules.view.actualTemperature') }}</span>
            <div class="temp-value-huge">
              <span class="number">{{ currentAmbientTemp.toFixed(1) }}</span>
              <span class="unit">°C</span>
            </div>
            <span class="status-indicator">
              <span class="dot-online"></span> {{ t('schedules.view.continuousMonitoring') }}
            </span>
          </div>

          <!-- Interactive Setpoint Control -->
          <div v-if="supportsSetpoint" class="temp-display-block target">
            <span class="temp-sublabel">{{ t('schedules.view.desiredSetpoint') }}</span>

            <div class="setpoint-adjust-row">
              <button
                v-if="liveTargetTemp - 0.5 >= setpointMin"
                class="btn-temp-circle"
                @click="adjustLiveTemp(-0.5)"
                :title="t('schedules.view.decreaseHalfDegree')"
                :aria-label="t('schedules.view.decreaseTemperature')"
              >
                -
              </button>
              <div class="temp-value-huge text-target">
                <span class="number">{{ liveTargetTemp.toFixed(1) }}</span>
                <span class="unit">°C</span>
              </div>
              <button
                v-if="liveTargetTemp + 0.5 <= setpointMax"
                class="btn-temp-circle"
                @click="adjustLiveTemp(0.5)"
                :title="t('schedules.view.increaseHalfDegree')"
                :aria-label="t('schedules.view.increaseTemperature')"
              >
                +
              </button>
            </div>

            <!-- Quick Presets -->
            <div class="quick-temp-pills">
              <button
                class="temp-pill"
                :class="{ active: liveTargetTemp === 18.0 }"
                @click="setLivePreset(18.0)"
              >
                <Moon :size="12" />
                <span>{{ t('schedules.view.ecoPreset') }}</span>
              </button>
              <button
                class="temp-pill"
                :class="{ active: liveTargetTemp === 20.0 }"
                @click="setLivePreset(20.0)"
              >
                <Sun :size="12" />
                <span>{{ t('schedules.view.normalPreset') }}</span>
              </button>
              <button
                class="temp-pill"
                :class="{ active: liveTargetTemp === 21.5 }"
                @click="setLivePreset(21.5)"
              >
                <Flame :size="12" />
                <span>{{ t('schedules.view.comfortPreset') }}</span>
              </button>
              <button
                class="temp-pill"
                :class="{ active: liveTargetTemp === 23.0 }"
                @click="setLivePreset(23.0)"
              >
                <Sparkles :size="12" />
                <span>{{ t('schedules.view.boostPreset') }}</span>
              </button>
            </div>
          </div>

          <!-- HVAC Mode & Execution Column -->
          <div class="temp-display-block actions-col">
            <span class="temp-sublabel">{{ t('schedules.view.operatingMode') }}</span>

            <div v-if="supportedHvacModes.length" class="hvac-mode-buttons">
              <button
                v-for="mode in supportedHvacModes"
                :key="mode"
                class="mode-btn"
                :class="{ active: liveHvacMode === mode }"
                @click="liveHvacMode = mode"
              >
                <Clock v-if="mode === 'auto'" :size="14" />
                <Flame v-else-if="mode === 'heat' || mode === 'manual'" :size="14" />
                <Power v-else-if="mode === 'off'" :size="14" />
                <span>{{ mode === 'auto' ? t('schedules.view.automatic') : mode === 'heat' ? t('schedules.view.heating') : mode === 'off' ? t('schedules.view.off') : mode }}</span>
              </button>
            </div>

            <div class="manual-override-duration">
              <label for="manual-climate-override-duration">{{ t('schedules.view.manualOverrideDuration') }}</label>
              <select id="manual-climate-override-duration" data-testid="manual-climate-override-duration" v-model.number="manualOverrideDuration">
                <option :value="0">{{ t('schedules.view.untilNextChange') }}</option>
                <option :value="30">{{ t('schedules.view.minutes30') }}</option>
                <option :value="60">{{ t('schedules.view.hour1') }}</option>
                <option :value="120">{{ t('schedules.view.hours2') }}</option>
                <option :value="240">{{ t('schedules.view.hours4') }}</option>
                <option :value="480">{{ t('schedules.view.hours8') }}</option>
              </select>
              <small>{{ t('schedules.view.manualOverrideExpiryHint') }}</small>
            </div>
            <button
              class="btn-apply-thermostat"
              :disabled="isAdjustingLive"
              @click="applyLiveThermostatSettings"
            >
              <Save :size="16" />
              <span>{{ isAdjustingLive ? t('schedules.view.sendingCommand') : t('schedules.view.adjustThermostatNow') }}</span>
            </button>
          </div>
        </div>
      </section>

      <!-- SECTION 2: WEEKLY SCHEDULE MATRIX (7 GIORNI X 24H) -->
      <section v-if="selectedSchedule" class="planner-card glass-panel" :aria-label="t('schedules.view.weeklyProgrammer')">
        <div class="planner-top">
          <div class="schedule-title-block">
            <h2 class="schedule-title">{{ selectedSchedule.name }}</h2>
            <span class="schedule-target">{{ t('schedules.view.controlledEntity') }} {{ selectedSchedule.entity_key }}</span>
            <small v-if="activeThermostatEntityKey === selectedSchedule.entity_key && !supportsSetpoint">{{ t('schedules.view.noProgrammableSetpoint') }}</small>
          </div>

          <div class="schedule-selector-row">
            <div class="form-group inline">
              <label for="sched-picker">{{ t('schedules.view.profile') }}</label>
              <select id="sched-picker" v-model="selectedScheduleId" class="sched-select">
                <option v-for="s in schedules" :key="s.id" :value="s.id">
                  {{ s.name }} ({{ s.entity_key }})
                </option>
              </select>
            </div>

            <div v-if="selectedSchedule" class="form-group inline">
              <label for="schedule-thermostat-select">{{ t('schedules.view.deviceLabel') }}</label>
              <select
                id="schedule-thermostat-select"
                v-model="selectedSchedule.entity_key"
                class="sched-select"
                @change="syncThermostatFromSchedule"
              >
                <option
                  v-for="thermostat in allSelectableThermostats"
                  :key="thermostat.id"
                  :value="thermostat.id"
                >
                  {{ thermostat.name }} ({{ thermostat.id }})
                </option>
                <option
                  v-if="!selectedScheduleThermostatAvailable"
                  :value="selectedSchedule?.entity_key || ''"
                >
                  {{ selectedSchedule?.entity_key }}
                </option>
              </select>
            </div>

            <button class="btn-tool-secondary" @click="createNewSchedule">
              <Plus :size="14" />
              <span>{{ t('schedules.view.newProfile') }}</span>
            </button>
          </div>

          <div class="planner-top-actions">
            <!-- Preset Profile Buttons -->
            <button class="btn-profile-preset" @click="applyWinterWorkingProfile" :title="t('schedules.view.winterWorkProfile')">
              <Sparkles :size="14" />
              <span>{{ t('schedules.view.winterWorkProfile') }}</span>
            </button>
            <router-link
              class="btn-profile-preset"
              :to="{ path: '/automations', query: { climateProgram: '1', target: selectedSchedule.entity_key } }"
              data-testid="sensor-automation-link"
            >
              <Sparkles :size="14" />
              <span>{{ t('schedules.view.externalSensorAutomation') }}</span>
            </router-link>
            <button class="btn-profile-preset" @click="copyMondayToWorkdays" :title="t('schedules.view.copyMondayToFriday')">
              <Copy :size="14" />
              <span>{{ t('schedules.view.copyMondayToFriday') }}</span>
            </button>
            <button class="btn-profile-preset" @click="copySaturdayToSunday" :title="t('schedules.view.copySaturdayToSunday')">
              <Copy :size="14" />
              <span>{{ t('schedules.view.copySaturdayToSunday') }}</span>
            </button>

            <!-- Save Schedule Button -->
            <button class="btn-primary" :disabled="isSavingSchedule" @click="saveCurrentSchedule">
              <Save :size="16" />
              <span>{{ isSavingSchedule ? t('schedules.view.saving') : t('schedules.view.saveSchedule') }}</span>
            </button>
          </div>
        </div>

        <!-- 7-Day Matrix Grid -->
        <div class="weekly-matrix">
          <div v-for="day in daysOfWeek" :key="day.key" class="day-row">
            <div class="day-col-left">
              <span class="day-label">{{ day.label }}</span>
              <button class="btn-add-slot" @click="openAddSlot(day.key)" :title="t('schedules.view.addTimeSlot')">
                <Plus :size="12" />
                <span>{{ t('schedules.view.timeSlot') }}</span>
              </button>
            </div>

            <div class="slots-track">
              <div
                v-for="(slot, idx) in (selectedSchedule.grid[day.key] || [])"
                :key="idx"
                class="slot-block"
              >
                <div class="slot-info">
                  <span class="slot-time">{{ slot.start }} - {{ slot.end }}</span>
                  <span class="slot-temp">{{ Number(slot.value).toFixed(1) }}°C</span>
                </div>
                <button
                  class="btn-del-slot"
                  @click="removeSlot(day.key, idx)"
                   :title="t('schedules.view.deleteTimeSlot')"
                  :aria-label="t('schedules.view.deleteTimeSlotAria')"
                >
                  <X :size="12" />
                </button>
              </div>

              <div
                v-if="!selectedSchedule.grid[day.key] || selectedSchedule.grid[day.key].length === 0"
                class="no-slots"
              >
                {{ t('schedules.view.emptyTimeSlots') }}
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- Tab 2: Calendar & iCal Sync -->
    <div v-else class="calendar-layout">
      <div class="calendar-card glass-panel">
        <div class="cal-top-bar">
          <h2 class="section-title">{{ t('schedules.view.syncedEventsTitle') }}</h2>
          <label class="import-btn">
            <Upload :size="16" />
            <span>{{ t('schedules.view.importIcs') }}</span>
            <input type="file" accept=".ics" class="hidden-input" @change="handleICalUpload" />
          </label>
        </div>

        <p v-if="icsUploadStatus" class="status-msg text-success">{{ icsUploadStatus }}</p>

        <p v-if="calendarEvents.length === 0" class="empty-state">
          {{ t('schedules.view.noSyncedEvents') }}
        </p>

        <div v-else class="calendar-events-list">
          <div v-for="evt in calendarEvents" :key="evt.id" class="cal-event-card">
            <div class="cal-time-col">
              <span class="cal-date">{{ new Date(evt.start_time).toLocaleDateString(locale) }}</span>
              <span class="cal-hours">{{
                new Date(evt.start_time).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })
              }}</span>
            </div>

            <div class="cal-info-col">
              <h3 class="evt-summary">{{ evt.summary }}</h3>
              <p class="evt-desc">{{ evt.description }}</p>
              <span v-if="evt.location" class="evt-loc">{{ t('schedules.view.room') }} {{ evt.location }}</span>
            </div>

            <span class="source-tag" :class="evt.source">{{ t(`schedules.view.eventSources.${evt.source}`) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal for Adding a Time Slot -->
    <div v-if="showAddSlotModal" class="modal-backdrop" @click="showAddSlotModal = false">
      <div class="modal-card" @click.stop role="dialog" aria-modal="true" aria-labelledby="modal-slot-title">
        <div class="modal-header">
          <h2 id="modal-slot-title">{{ t('schedules.view.newTimeSlotTitle') }}</h2>
          <button class="modal-close-btn" @click="showAddSlotModal = false" :aria-label="t('schedules.view.closeWindow')">
            <X class="icon-close" />
          </button>
        </div>

        <form @submit.prevent="saveSlot" class="modal-body">
          <div class="form-row">
            <div class="form-group">
              <label for="slot-start">{{ t('schedules.view.startTime') }}</label>
              <input id="slot-start" v-model="newSlotForm.start" type="time" required />
            </div>
            <div class="form-group">
              <label for="slot-end">{{ t('schedules.view.endTime') }}</label>
              <input id="slot-end" v-model="newSlotForm.end" type="time" required />
            </div>
          </div>

          <div class="form-group">
            <label for="slot-temp">{{ t('schedules.view.targetTemperature') }}</label>
            <input
              id="slot-temp"
              v-model.number="newSlotForm.value"
              type="number"
              step="0.1"
              :min="setpointMin"
              :max="setpointMax"
              required
            />
          </div>

          <div class="modal-footer">
            <button type="button" class="btn-cancel" @click="showAddSlotModal = false">{{ t('schedules.view.cancel') }}</button>
            <button type="submit" class="btn-primary">{{ t('schedules.view.addSlot') }}</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.schedule-view-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.schedule-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-radius: 16px;
  background: var(--surface-card, rgba(15, 23, 42, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  flex-wrap: wrap;
  gap: 16px;
}

.header-title-box {
  display: flex;
  align-items: center;
  gap: 16px;
}

.icon-circle {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: rgba(245, 158, 11, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
}

.text-flame {
  color: #f59e0b;
}

.page-title {
  font-size: 1.4rem;
  font-weight: 700;
  margin: 0;
  color: var(--text-main, #f8fafc);
}

.page-subtitle {
  font-size: 0.85rem;
  color: var(--text-muted, #94a3b8);
  margin: 4px 0 0;
}

.tab-selector {
  display: flex;
  gap: 8px;
  background: rgba(255, 255, 255, 0.05);
  padding: 4px;
  border-radius: 10px;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: var(--text-muted, #94a3b8);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.tab-btn.active {
  background: var(--primary, #3b82f6);
  color: #fff;
}

/* LIVE SMART THERMOSTAT CARD */
.cronotermostato-layout {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.live-thermostat-card {
  padding: 24px;
  border-radius: 16px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  box-shadow: var(--shadow-md);
}

.thermostat-top-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  flex-wrap: wrap;
  gap: 12px;
}

.thermostat-title-box {
  display: flex;
  align-items: center;
  gap: 8px;
}

.thermostat-title-box h3 {
  margin: 0;
  font-size: 1.1rem;
  color: var(--text-main);
}

.thermostat-selector-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  color: var(--text-muted);
}

.thermostat-select {
  padding: 6px 12px;
  background: var(--bg-surface);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-main);
  font-size: 0.85rem;
}

.thermostat-dial-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 20px;
}

.temp-display-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
  border-radius: 12px;
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-color);
}

.temp-sublabel {
  font-size: 0.8rem;
  color: var(--text-muted);
  font-weight: 600;
  margin-bottom: 12px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.temp-value-huge {
  display: flex;
  align-items: baseline;
  gap: 4px;
}

.temp-value-huge .number {
  font-size: 3rem;
  font-weight: 800;
  color: var(--text-main);
  line-height: 1;
}

.temp-value-huge .unit {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-muted);
}

.temp-value-huge.text-target .number {
  color: #f59e0b;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  color: #10b981;
  margin-top: 12px;
}

.dot-online {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 8px #10b981;
}

.setpoint-adjust-row {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 12px;
}

.btn-temp-circle {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: 2px solid var(--border-color);
  background: var(--bg-surface-hover);
  color: var(--text-main);
  font-size: 1.5rem;
  font-weight: 700;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.btn-temp-circle:hover {
  background: rgba(245, 158, 11, 0.3);
  border-color: #f59e0b;
  transform: scale(1.08);
}

.quick-temp-pills {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: center;
}

.temp-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.05);
  color: #cbd5e1;
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.2s;
}

.temp-pill:hover {
  background: rgba(255, 255, 255, 0.1);
}

.temp-pill.active {
  background: rgba(245, 158, 11, 0.25);
  border-color: #f59e0b;
  color: #fbbf24;
  font-weight: 600;
}

.actions-col {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.hvac-mode-buttons {
  display: flex;
  gap: 6px;
  width: 100%;
}

.mode-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 8px 10px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.05);
  color: #cbd5e1;
  font-size: 0.78rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.mode-btn:hover {
  background: rgba(255, 255, 255, 0.1);
}

.mode-btn.active {
  background: rgba(245, 158, 11, 0.3);
  border-color: #f59e0b;
  color: #fbbf24;
}

.mode-btn.off.active {
  background: rgba(239, 68, 68, 0.2);
  border-color: #ef4444;
  color: #f87171;
}

.manual-override-duration {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin: 10px 0;
  font-size: 12px;
}
.manual-override-duration select {
  min-height: 38px;
  padding: 6px 9px;
  border: 1px solid var(--border-card);
  border-radius: var(--radius-sm);
  background: var(--bg-surface);
  color: var(--text-main);
}
.manual-override-duration small {
  color: var(--text-muted);
  line-height: 1.4;
}

.btn-apply-thermostat {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px;
  border-radius: 10px;
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  border: none;
  color: #000;
  font-weight: 700;
  font-size: 0.9rem;
  cursor: pointer;
  transition: filter 0.2s, transform 0.15s;
}

.btn-apply-thermostat:hover {
  filter: brightness(1.1);
  transform: translateY(-1px);
}

/* SECTION 2: WEEKLY SCHEDULE MATRIX */
.planner-card,
.calendar-card {
  padding: 24px;
  border-radius: 16px;
  background: var(--surface-card, rgba(15, 23, 42, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
}

.planner-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  flex-wrap: wrap;
  gap: 16px;
}

.schedule-selector-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.form-group.inline {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
}

.form-group.inline label {
  font-size: 0.85rem;
  color: var(--text-muted);
}

.sched-select {
  padding: 6px 12px;
  background: var(--bg-surface);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-main);
  font-size: 0.85rem;
}

.btn-tool-secondary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-color);
  color: var(--text-muted);
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-tool-secondary:hover {
  background: var(--bg-surface);
  color: var(--text-main);
}

.planner-top-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.btn-profile-preset {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  background: rgba(59, 130, 246, 0.15);
  border: 1px solid rgba(59, 130, 246, 0.3);
  color: #60a5fa;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-profile-preset:hover {
  background: rgba(59, 130, 246, 0.25);
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--primary, #3b82f6);
  border: none;
  color: #fff;
  border-radius: 8px;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary:hover {
  background: var(--primary-hover, #2563eb);
}

.weekly-matrix {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.day-row {
  display: grid;
  grid-template-columns: 140px 1fr;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 10px;
}

.day-col-left {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.day-label {
  font-size: 0.9rem;
  font-weight: 700;
  color: #f1f5f9;
}

.btn-add-slot {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #94a3b8;
  font-size: 0.7rem;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-add-slot:hover {
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
  border-color: #3b82f6;
}

.slots-track {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.slot-block {
  display: flex;
  align-items: center;
  gap: 8px;
  background: rgba(245, 158, 11, 0.15);
  border: 1px solid rgba(245, 158, 11, 0.3);
  padding: 6px 10px;
  border-radius: 8px;
}

.slot-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.slot-time {
  font-size: 0.8rem;
  color: #cbd5e1;
  font-weight: 500;
}

.slot-temp {
  font-size: 0.85rem;
  font-weight: 700;
  color: #fbbf24;
}

.btn-del-slot {
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}

.btn-del-slot:hover {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.15);
}

.no-slots {
  font-size: 0.8rem;
  color: #64748b;
  font-style: italic;
}

/* Tab 2 Calendar Styling */
.cal-top-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-title {
  font-size: 1.1rem;
  font-weight: 600;
  margin: 0;
  color: var(--text-main);
}

.import-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 8px;
  background: rgba(99, 102, 241, 0.15);
  color: #818cf8;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
}

.hidden-input {
  display: none;
}

.calendar-events-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.cal-event-card {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 14px 18px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 12px;
}

.cal-time-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 90px;
}

.cal-date {
  font-size: 0.8rem;
  color: var(--text-muted, #94a3b8);
}

.cal-hours {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--primary, #3b82f6);
}

.cal-info-col {
  flex: 1;
}

.evt-summary {
  font-size: 0.95rem;
  font-weight: 600;
  margin: 0;
}

.evt-desc {
  font-size: 0.8rem;
  color: var(--text-muted, #94a3b8);
  margin: 4px 0;
}

.evt-loc {
  font-size: 0.75rem;
  color: #f59e0b;
}

.source-tag {
  font-size: 0.7rem;
  font-weight: 700;
  padding: 4px 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.1);
}

.empty-state {
  padding: 32px;
  text-align: center;
  color: var(--text-muted, #94a3b8);
}

/* Modals */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
  padding: 1rem;
}

.modal-card {
  background: #0f172a;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 0.75rem;
  width: 100%;
  max-width: 440px;
  padding: 1.5rem;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.7);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.modal-header h2 {
  font-size: 1.25rem;
  margin: 0;
  color: #f8fafc;
}

.modal-close-btn {
  background: none;
  border: none;
  color: #94a3b8;
  cursor: pointer;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-bottom: 1rem;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.form-group label {
  font-size: 0.8rem;
  font-weight: 500;
  color: #cbd5e1;
}

.form-group input {
  padding: 0.6rem;
  background: #1e293b;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 0.375rem;
  color: #fff;
  font-size: 0.875rem;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1.5rem;
}

.btn-cancel {
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #94a3b8;
  border-radius: 0.375rem;
  cursor: pointer;
}

/* Keep schedule tabs reachable on narrow screens and prevent flex overflow. */
.schedule-view-container {
  width: 100%;
  min-width: 0;
}

.tab-selector {
  max-width: 100%;
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
  flex-wrap: nowrap;
  -webkit-overflow-scrolling: touch;
  overscroll-behavior-x: contain;
  scrollbar-width: thin;
  touch-action: pan-x;
}

.tab-selector .tab-btn {
  flex: 0 0 auto;
  min-height: 44px;
  white-space: nowrap;
}

@media (max-width: 640px) {
  .schedule-header {
    align-items: flex-start;
    padding: 16px;
  }

  .header-title-box {
    min-width: 0;
    align-items: flex-start;
  }

  .tab-selector {
    width: 100%;
  }

  .live-thermostat-card {
    padding: 16px;
  }

  .thermostat-dial-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

</style>
