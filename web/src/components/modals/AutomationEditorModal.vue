<script setup lang="ts">
import { ref, watch, toRef } from 'vue'
import type { Automation } from '../../api/types'
import BaseButton from '../ui/BaseButton.vue'
import { useModalA11y } from '../../composables/useModalA11y'
import { X, Zap, AlertCircle, CheckCircle, Code2, Sparkles } from '@lucide/vue'

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
