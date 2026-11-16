<script setup lang="ts">
import { computed, ref, watch, toRef } from 'vue'
import type { Automation, Entity } from '../../api/types'
import { apiClient } from '../../api/client'
import BaseButton from '../ui/BaseButton.vue'
import { useModalA11y } from '../../composables/useModalA11y'
import { X, Zap, AlertCircle, CheckCircle, Code2, Sparkles, Plus, Trash2 } from '@lucide/vue'

interface Props {
  isOpen: boolean
  automation?: Automation | null
}

const props = withDefaults(defineProps<Props>(), {
  isOpen: false,
  automation: null
})

const emit = defineEmits<{
  (e: 'save', data: Partial<Automation>): void
  (e: 'close'): void
}>()

const modalContainerRef = ref<HTMLElement | null>(null)
useModalA11y({
  isOpen: toRef(props, 'isOpen'),
  modalRef: modalContainerRef,
  onClose: () => emit('close')
})

const name = ref('')
const description = ref('')
const executionMode = ref<'single' | 'restart' | 'queued' | 'parallel'>('single')
const cooldownSeconds = ref(0)
const dslJson = ref('')
const validationError = ref<string | null>(null)
const jsonValid = ref(true)

// Guided weekly thermostat program: at each configured time, read the latest
// temperature reported by a sensor and choose a setpoint for the thermostat.
interface ClimateScheduleSlot {
  id: string
  time: string
  days: number[]
  threshold: number
  coldSetpoint: number
  warmSetpoint: number
}

const climateWeekdays = [
  { value: 1, label: 'Lun' },
  { value: 2, label: 'Mar' },
  { value: 3, label: 'Mer' },
  { value: 4, label: 'Gio' },
  { value: 5, label: 'Ven' },
  { value: 6, label: 'Sab' },
  { value: 7, label: 'Dom' }
]

const climateBuilderOpen = ref(false)
const climateEntities = ref<Entity[]>([])
const climateEntitiesLoading = ref(false)
const climateEntitiesError = ref('')
const climateSensorKey = ref('')
const climateSensorAttribute = ref('')
const climateTargetKey = ref('')
const climateTimeSlots = ref<ClimateScheduleSlot[]>([
  { id: 'slot-1', time: '07:00', days: [1, 2, 3, 4, 5, 6, 7], threshold: 19, coldSetpoint: 21, warmSetpoint: 18 }
])
let nextClimateSlotNumber = 2

function entityKey(entity: Entity): string {
  if (entity.entity_key) return entity.entity_key
  return entity.id.startsWith(entity.domain + '.') ? entity.id : ''
}

const temperatureSources = computed(() => climateEntities.value.filter(entity => {
  const attributes = entity.attributes || {}
  const deviceClass = String(attributes.device_class || '').toLowerCase()
  const unit = String(attributes.unit_of_measurement || entity.unit_of_measurement || '').toLowerCase()
  const hasTemperatureUnit = ['°c', 'c', 'celsius'].includes(unit)
  const isFahrenheit = ['°f', 'f', 'fahrenheit'].includes(unit)
  const hasTemperatureAttribute =
    typeof attributes.current_temperature === 'number' ||
    typeof attributes.temperature === 'number'

  if (entity.domain === 'climate') return typeof attributes.current_temperature === 'number'
  return entity.domain === 'sensor' && !isFahrenheit &&
    (deviceClass === 'temperature' || hasTemperatureUnit || hasTemperatureAttribute)
}))
const climateTargets = computed(() => climateEntities.value.filter(entity => entity.domain === 'climate'))

async function toggleClimateBuilder() {
  climateBuilderOpen.value = !climateBuilderOpen.value
  if (!climateBuilderOpen.value || climateEntities.value.length || climateEntitiesLoading.value) return

  climateEntitiesLoading.value = true
  climateEntitiesError.value = ''
  try {
    climateEntities.value = await apiClient.get<Entity[]>('/api/v1/entities')
    if (!climateSensorKey.value) {
      const firstSource = temperatureSources.value[0]
      if (firstSource) climateSensorKey.value = entityKey(firstSource)
    }
    if (!climateTargetKey.value && climateTargets.value[0]) {
      climateTargetKey.value = entityKey(climateTargets.value[0])
    }
  } catch (err: any) {
    climateEntitiesError.value = err?.message || 'Impossibile caricare i dispositivi.'
  } finally {
    climateEntitiesLoading.value = false
  }
}

watch(climateSensorKey, key => {
  const source = temperatureSources.value.find(entity => entityKey(entity) === key)
  climateSensorAttribute.value = source?.domain === 'climate' ? 'current_temperature' : ''
})

function addClimateTimeSlot() {
  const usedTimes = new Set(climateTimeSlots.value.map(slot => slot.time))
  const suggestedTime = ['12:00', '18:00', '22:00', '06:00', '09:00', '15:00']
    .find(time => !usedTimes.has(time)) || ''
  const lastSlot = climateTimeSlots.value[climateTimeSlots.value.length - 1]
  climateTimeSlots.value.push({
    id: `slot-${nextClimateSlotNumber++}`,
    time: suggestedTime,
    days: [...(lastSlot?.days ?? [1, 2, 3, 4, 5, 6, 7])],
    threshold: lastSlot?.threshold ?? 19,
    coldSetpoint: lastSlot?.coldSetpoint ?? 21,
    warmSetpoint: lastSlot?.warmSetpoint ?? 18
  })
}

function removeClimateTimeSlot(id: string) {
  if (climateTimeSlots.value.length <= 1) return
  climateTimeSlots.value = climateTimeSlots.value.filter(slot => slot.id !== id)
}

const orderedClimateTimeSlots = computed(() =>
  [...climateTimeSlots.value].sort((a, b) => a.time.localeCompare(b.time))
)

function buildClimateProgram() {
  validationError.value = ''
  const source = temperatureSources.value.find(entity => entityKey(entity) === climateSensorKey.value)
  const target = climateTargets.value.find(entity => entityKey(entity) === climateTargetKey.value)
  if (!source || !target || !entityKey(source) || !entityKey(target)) {
    validationError.value = 'Seleziona un sensore di temperatura e un termostato.'
    return
  }
  if (climateTimeSlots.value.length === 0) {
    validationError.value = 'Aggiungi almeno una fascia oraria.'
    return
  }

  const seenTimes = new Set<string>()
  for (const slot of climateTimeSlots.value) {
    if (!/^([01]\d|2[0-3]):[0-5]\d$/.test(slot.time)) {
      validationError.value = 'Inserisci orari validi nel formato HH:MM.'
      return
    }
    if (!slot.days.length || slot.days.some(day => !Number.isInteger(day) || day < 1 || day > 7)) {
      validationError.value = 'Seleziona almeno un giorno valido per ogni fascia.'
      return
    }
    for (const day of slot.days) {
      const dayTime = `${day}:${slot.time}`
      if (seenTimes.has(dayTime)) {
        validationError.value = `L'orario ${slot.time} è duplicato per almeno uno dei giorni selezionati.`
        return
      }
      seenTimes.add(dayTime)
    }
    const values = [slot.threshold, slot.coldSetpoint, slot.warmSetpoint]
    if (values.some(value => !Number.isFinite(value) || value < 5 || value > 35)) {
      validationError.value = 'Soglie e setpoint devono essere compresi tra 5 e 35 °C.'
      return
    }
  }

  const numericCondition: Record<string, any> = {
    condition: 'numeric_state',
    entity_key: entityKey(source)
  }
  if (climateSensorAttribute.value) numericCondition.attribute = climateSensorAttribute.value

  const serviceAction = (temperature: number) => ({
    type: 'service',
    service: 'set_temperature',
    target_entity_key: entityKey(target),
    params: { temperature: Math.round(temperature * 10) / 10 }
  })

  // The time-trigger ID selects the matching schedule row; its nested choose
  // then evaluates the latest measured temperature for that time of day.
  const slots = orderedClimateTimeSlots.value
  const dsl = {
    triggers: slots.map((slot, index) => ({
      id: `climate_slot_${index + 1}`,
      platform: 'time',
      at: slot.time + ':00',
      days: [...slot.days].sort((a, b) => a - b)
    })),
    // Missing or non-numeric sensor data skips the automation instead of
    // silently applying a fallback setpoint.
    conditions: [numericCondition],
    actions: [{
      type: 'choose',
      choose: slots.map((slot, index) => ({
        conditions: [{ condition: 'trigger', trigger_id: `climate_slot_${index + 1}` }],
        sequence: [{
          type: 'choose',
          choose: [{
            conditions: [{ ...numericCondition, below: slot.threshold }],
            sequence: [serviceAction(slot.coldSetpoint)]
          }],
          default: [serviceAction(slot.warmSetpoint)]
        }]
      }))
    }]
  }

  dslJson.value = JSON.stringify(dsl, null, 2)
  if (!name.value.trim()) name.value = 'Programma termostato ' + target.name
  if (!description.value.trim()) {
    const scheduleSummary = slots.map(slot => {
      const days = slot.days
        .map(day => climateWeekdays.find(item => item.value === day)?.label)
        .filter(Boolean)
        .join(', ')
      return `${days} ${slot.time}: sotto ${slot.threshold} °C → ${slot.coldSetpoint} °C, altrimenti ${slot.warmSetpoint} °C`
    }).join('; ')
    description.value = `Legge ${source.name} e regola ${target.name} secondo la programmazione settimanale: ${scheduleSummary}.`
  }
  jsonValid.value = true
  climateBuilderOpen.value = false
}

const defaultDsl = {
  triggers: [
    { platform: 'state', entity_key: 'binary_sensor.motion', to: 'on' }
  ],
  conditions: [],
  actions: [
    { type: 'service', service: 'turn_on', target_entity_key: 'light.living_room' }
  ]
}

watch(
  () => props.automation,
  (val) => {
    if (val) {
      name.value = val.name
      description.value = val.description || ''
      executionMode.value = val.execution_mode || 'single'
      cooldownSeconds.value = val.cooldown_seconds || 0
      dslJson.value = JSON.stringify(
        {
          triggers: val.triggers || [],
          conditions: val.conditions || [],
          actions: val.actions || []
        },
        null,
        2
      )
    } else {
      name.value = ''
      description.value = ''
      executionMode.value = 'single'
      cooldownSeconds.value = 0
      dslJson.value = JSON.stringify(defaultDsl, null, 2)
    }
    validationError.value = null
    jsonValid.value = true
  },
  { immediate: true }
)

function formatJson() {
  try {
    const parsed = JSON.parse(dslJson.value)
    dslJson.value = JSON.stringify(parsed, null, 2)
    jsonValid.value = true
    validationError.value = null
  } catch (err: any) {
    validationError.value = `Sintassi JSON non valida: ${err.message}`
    jsonValid.value = false
  }
}

function applyTemplate(type: string) {
  if (type === 'motion') {
    dslJson.value = JSON.stringify({
      triggers: [{ platform: 'state', entity_key: 'binary_sensor.hallway_motion', to: 'on' }],
      conditions: [{ condition: 'state', entity_key: 'light.hallway', state: 'off' }],
      actions: [{ type: 'service', service: 'turn_on', target_entity_key: 'light.hallway' }]
    }, null, 2)
  } else if (type === 'schedule') {
    dslJson.value = JSON.stringify({
      triggers: [{ platform: 'time', at: '23:30:00' }],
      conditions: [],
      actions: [{ type: 'service', service: 'turn_off', target_entity_key: 'light.all' }]
    }, null, 2)
  }
  validationError.value = null
  jsonValid.value = true
}

function handleSubmit() {
  validationError.value = null

  if (!name.value.trim()) {
    validationError.value = 'Il nome dell\'automazione è obbligatorio'
    return
  }

  let parsed: any
  try {
    parsed = JSON.parse(dslJson.value)
    jsonValid.value = true
  } catch (err: any) {
    validationError.value = `Sintassi JSON non valida: ${err.message}`
    jsonValid.value = false
    return
  }

  if (!parsed.triggers || !Array.isArray(parsed.triggers) || parsed.triggers.length === 0) {
    validationError.value = 'Almeno un trigger è richiesto nella regola'
    return
  }

  if (!parsed.actions || !Array.isArray(parsed.actions) || parsed.actions.length === 0) {
    validationError.value = 'Almeno un\'azione è richiesta nella regola'
    return
  }

  emit('save', {
    name: name.value.trim(),
    description: description.value.trim() || undefined,
    execution_mode: executionMode.value,
    cooldown_seconds: cooldownSeconds.value,
    triggers: parsed.triggers,
    conditions: parsed.conditions || [],
    actions: parsed.actions,
    enabled: props.automation ? props.automation.enabled : true
  })
}
</script>

<template>
  <div v-if="isOpen" class="modal-backdrop" @click="emit('close')">
    <div
      ref="modalContainerRef"
      class="modal-card glass-panel animate-enter"
      role="dialog"
      aria-modal="true"
      aria-labelledby="automation-modal-title"
      @click.stop
    >
      <div class="modal-header">
        <div class="header-title">
          <Zap :size="20" class="title-icon" />
          <h3 id="automation-modal-title">{{ automation ? 'Modifica Automazione' : 'Nuova Automazione' }}</h3>
        </div>
        <button class="btn-close" aria-label="Chiudi finestra automazione" @click="emit('close')">
          <X :size="18" />
        </button>
      </div>

      <div v-if="validationError" class="error-alert">
        <AlertCircle :size="16" class="error-icon" />
        <span>{{ validationError }}</span>
      </div>

      <form class="modal-form" @submit.prevent="handleSubmit">
        <div class="form-row">
          <div class="form-group flex-2">
            <label for="auto-name">Nome Automazione</label>
            <input
              id="auto-name"
              v-model="name"
              name="auto-name"
              type="text"
              placeholder="es. Accensione Luce Notte"
              required
            />
          </div>

          <div class="form-group flex-1">
            <label for="exec-mode">Modalità Esecuzione</label>
            <select id="exec-mode" v-model="executionMode">
              <option value="single">Single (ignora se in esecuzione)</option>
              <option value="restart">Restart (riavvia esecuzione)</option>
              <option value="queued">Queued (accoda richieste)</option>
              <option value="parallel">Parallel (esegui in parallelo)</option>
            </select>
          </div>
        </div>

        <div class="form-row">
          <div class="form-group flex-2">
            <label for="auto-desc">Descrizione (opzionale)</label>
            <input
              id="auto-desc"
              v-model="description"
              type="text"
              placeholder="Dettagli sullo scopo della regola..."
            />
          </div>

          <div class="form-group flex-1">
            <label for="cooldown">Cooldown (secondi)</label>
            <input
              id="cooldown"
              v-model.number="cooldownSeconds"
              type="number"
              min="0"
              placeholder="0"
            />
          </div>
        </div>

        <!-- DSL Editor Section -->
        <div class="dsl-section">
          <div class="dsl-header">
            <div class="dsl-label">
              <Code2 :size="16" />
              <span>Regola DSL JSON (Triggers, Condizioni, Azioni)</span>
            </div>

            <div class="dsl-actions">
              <button
                type="button"
                class="btn-template climate-template"
                data-testid="climate-program-toggle"
                :aria-expanded="climateBuilderOpen"
                @click="toggleClimateBuilder"
              >
                <Sparkles :size="12" />
                <span>Programma termostato</span>
              </button>
              <button type="button" class="btn-template" @click="applyTemplate('motion')">
                <Sparkles :size="12" />
                <span>Motion Preset</span>
              </button>
              <button type="button" class="btn-template" @click="applyTemplate('schedule')">
                <Sparkles :size="12" />
                <span>Schedule Preset</span>
              </button>
              <button type="button" class="btn-format" @click="formatJson">
                Formatta
              </button>
            </div>
          </div>

          <section v-if="climateBuilderOpen" class="climate-builder" aria-label="Programma termostato basato sul sensore">
            <div class="climate-builder-heading">
              <div>
                <strong>Programma giornaliero basato sulla temperatura</strong>
                <p>All'orario scelto legge il sensore e imposta il termostato in uno dei due modi.</p>
              </div>
            </div>

            <p v-if="climateEntitiesLoading" class="builder-hint">Caricamento dispositivi...</p>
            <p v-else-if="climateEntitiesError" class="builder-error">{{ climateEntitiesError }}</p>
            <p v-else-if="temperatureSources.length === 0 || climateTargets.length === 0" class="builder-hint">
              Servono almeno un sensore di temperatura e un'entità Climate. Verifica che siano già configurati in Domus.
            </p>

            <template v-else>
              <div class="form-row">
                <div class="form-group flex-1">
                  <label for="climate-source">Sensore da leggere</label>
                  <select id="climate-source" name="climate-sensor" v-model="climateSensorKey">
                    <option v-for="entity in temperatureSources" :key="entity.id" :value="entityKey(entity)">
                      {{ entity.name }} · {{ entity.entity_key || entity.id }}
                    </option>
                  </select>
                  <small v-if="climateSensorAttribute" class="builder-hint">Lettura della temperatura ambiente corrente.</small>
                </div>
                <div class="form-group flex-1">
                  <label for="climate-target">Termostato da regolare</label>
                  <select id="climate-target" name="climate-thermostat" v-model="climateTargetKey">
                    <option v-for="entity in climateTargets" :key="entity.id" :value="entityKey(entity)">
                      {{ entity.name }} · {{ entity.entity_key || entity.id }}
                    </option>
                  </select>
                </div>
              </div>

              <div class="climate-slots-heading">
                <div>
                  <strong>Programmazione settimanale</strong>
                  <p class="builder-hint">Scegli i giorni e l'orario. A ogni esecuzione Domus legge la temperatura aggiornata e regola il termostato.</p>
                </div>
                <button type="button" class="btn-template" data-testid="add-climate-slot" @click="addClimateTimeSlot">
                  <Plus :size="13" />
                  <span>Aggiungi fascia</span>
                </button>
              </div>

              <div v-for="(slot, index) in climateTimeSlots" :key="slot.id" class="climate-slot">
                <div class="climate-slot-heading">
                  <strong>Fascia {{ index + 1 }}</strong>
                  <button
                    v-if="climateTimeSlots.length > 1"
                    type="button"
                    class="climate-slot-remove"
                    :aria-label="`Rimuovi fascia ${index + 1}`"
                    :data-testid="`remove-climate-slot-${slot.id}`"
                    @click="removeClimateTimeSlot(slot.id)"
                  >
                    <Trash2 :size="14" />
                    <span>Rimuovi</span>
                  </button>
                </div>

                <div class="form-row">
                  <div class="form-group flex-1">
                    <label :for="index === 0 ? 'climate-time' : `climate-time-${slot.id}`">Orario</label>
                    <input
                      :id="index === 0 ? 'climate-time' : `climate-time-${slot.id}`"
                      name="climate-time"
                      v-model="slot.time"
                      type="time"
                    />
                  </div>
                  <div class="form-group flex-1">
                    <label :for="index === 0 ? 'climate-threshold' : `climate-threshold-${slot.id}`">Soglia del sensore (°C)</label>
                    <input
                      :id="index === 0 ? 'climate-threshold' : `climate-threshold-${slot.id}`"
                      v-model.number="slot.threshold"
                      type="number"
                      min="5"
                      max="35"
                      step="0.1"
                    />
                  </div>
                </div>

                <div class="climate-weekdays" role="group" :aria-label="`Giorni attivi fascia ${index + 1}`">
                  <label v-for="day in climateWeekdays" :key="day.value" class="climate-weekday">
                    <input v-model="slot.days" type="checkbox" :value="day.value" />
                    <span>{{ day.label }}</span>
                  </label>
                </div>

                <div class="form-row">
                  <div class="form-group flex-1">
                    <label :for="index === 0 ? 'climate-cold-setpoint' : `climate-cold-setpoint-${slot.id}`">Se è sotto soglia (°C)</label>
                    <input
                      :id="index === 0 ? 'climate-cold-setpoint' : `climate-cold-setpoint-${slot.id}`"
                      v-model.number="slot.coldSetpoint"
                      type="number"
                      min="5"
                      max="35"
                      step="0.1"
                    />
                  </div>
                  <div class="form-group flex-1">
                    <label :for="index === 0 ? 'climate-warm-setpoint' : `climate-warm-setpoint-${slot.id}`">Altrimenti imposta (°C)</label>
                    <input
                      :id="index === 0 ? 'climate-warm-setpoint' : `climate-warm-setpoint-${slot.id}`"
                      v-model.number="slot.warmSetpoint"
                      type="number"
                      min="5"
                      max="35"
                      step="0.1"
                    />
                  </div>
                </div>
              </div>

              <div class="climate-rule-preview">
                <p v-for="slot in orderedClimateTimeSlots" :key="slot.id">
                  {{ slot.days.map(day => climateWeekdays.find(item => item.value === day)?.label).join(', ') }} alle <strong>{{ slot.time }}</strong>, se la temperatura è sotto <strong>{{ slot.threshold }} °C</strong>
                  imposta <strong>{{ slot.coldSetpoint }} °C</strong>, altrimenti <strong>{{ slot.warmSetpoint }} °C</strong>.
                </p>
              </div>
              <div class="builder-actions">
                <button type="button" class="btn-template" data-testid="generate-climate-program" @click="buildClimateProgram">
                  Genera regola DSL
                </button>
              </div>
            </template>
          </section>

          <textarea
            v-model="dslJson"
            name="dsl-json"
            rows="11"
            class="dsl-textarea font-mono"
            placeholder="JSON DSL..."
            required
          />
        </div>

        <div class="modal-footer">
          <BaseButton type="button" variant="ghost" @click="emit('close')">
            Annulla
          </BaseButton>
          <BaseButton type="submit" variant="primary">
            Salva Automazione
          </BaseButton>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: var(--bg-overlay);
  backdrop-filter: blur(8px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.modal-card {
  width: 100%;
  max-width: 680px;
  max-height: 90vh;
  overflow-y: auto;
  border-radius: var(--radius-xl);
  padding: 28px;
  border: 1px solid var(--border-card);
  box-shadow: var(--shadow-lg);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.title-icon {
  color: #f59e0b;
}

h3 {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-main);
}

.btn-close {
  background: transparent;
  color: var(--text-dim);
  padding: 4px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color var(--transition-fast);
}

.btn-close:hover {
  color: var(--text-main);
}

.error-alert {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--radius-md);
  background: var(--color-danger-bg);
  color: var(--color-danger);
  font-size: 13px;
  margin-bottom: 16px;
}

.error-icon {
  flex-shrink: 0;
}

.modal-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-row {
  display: flex;
  gap: 14px;
}

@media (max-width: 600px) {
  .form-row {
    flex-direction: column;
  }
}

.flex-1 {
  flex: 1;
}

.flex-2 {
  flex: 2;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
}

input, select {
  width: 100%;
  padding: 10px 14px;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  color: var(--text-main);
  font-size: 14px;
}

input:focus, select:focus {
  border-color: var(--primary);
  outline: none;
}

.dsl-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.dsl-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}

.dsl-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
}

.dsl-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-template, .btn-format {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  font-size: 11px;
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-card);
  color: var(--text-dim);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-template:hover, .btn-format:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-main);
}

.climate-builder {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px;
  border: 1px solid var(--border-card);
  border-radius: var(--radius-lg);
  background: var(--bg-surface);
}

.climate-builder-heading strong {
  color: var(--text-main);
  font-size: 14px;
}

.climate-builder-heading p,
.builder-hint {
  margin: 5px 0 0;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.45;
}

.builder-error {
  color: var(--color-danger);
  font-size: 12px;
}

.climate-slots-heading,
.climate-weekdays {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  margin: 0.25rem 0 0.9rem;
}

.climate-weekday {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  min-height: 36px;
  padding: 0.35rem 0.55rem;
  border: 1px solid var(--border-color, #d1d5db);
  border-radius: 0.5rem;
  font-size: 0.8rem;
  cursor: pointer;
}

.climate-weekday input {
  accent-color: var(--color-primary, #2563eb);
}

.climate-slot-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.climate-slots-heading > div {
  min-width: 0;
}

.climate-slot {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.025);
}

.climate-slot-heading strong,
.climate-slots-heading strong {
  color: var(--text-main);
  font-size: 13px;
}

.climate-slot-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  min-height: 36px;
  padding: 6px 9px;
  border: 1px solid var(--border-card);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
}

.climate-slot-remove:hover {
  color: var(--color-danger);
  border-color: var(--color-danger);
}

.climate-rule-preview {
  margin: 0;
  padding: 10px 12px;
  border-radius: var(--radius-md);
  background: rgba(14, 165, 233, 0.08);
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.climate-rule-preview strong {
  color: var(--text-main);
}

.builder-actions {
  display: flex;
  justify-content: flex-end;
}

.climate-template {
  border-color: rgba(14, 165, 233, 0.35);
  color: var(--primary);
}

.form-group small {
  font-size: 11px;
}

.dsl-textarea {
  width: 100%;
  padding: 12px 14px;
  background: #060913;
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  color: #38bdf8;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.5;
  resize: vertical;
}

.dsl-textarea:focus {
  border-color: var(--primary);
  outline: none;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 8px;
  padding-top: 16px;
  border-top: 1px solid var(--border-subtle);
}
</style>
