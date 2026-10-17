<script setup lang="ts">
import { ref, computed, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { X, Play, Plus, Zap, Filter, Clock, ArrowRight, Check, Trash2, Layers, Sliders } from '@lucide/vue'
import { audioSynthesizer } from '../../utils/audioSynthesizer'

const { t } = useI18n()

const props = defineProps<{
  isOpen: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', automationData: any): void
}>()

interface FlowNode {
  id: string
  type: 'trigger' | 'condition' | 'delay' | 'action'
  title: string
  subtitle: string
  x: number
  y: number
  entityKey?: string
  service?: string
}

interface FlowConnection {
  from: string
  to: string
}

const nodes = ref<FlowNode[]>([
  {
    id: 'n_trig_1',
    type: 'trigger',
    title: 'Rilevato Movimento',
    subtitle: 'binary_sensor.salotto_motion == true',
    x: 60,
    y: 120,
    entityKey: 'binary_sensor.salotto_motion'
  },
  {
    id: 'n_cond_1',
    type: 'condition',
    title: 'Luminosità Bassa',
    subtitle: 'sensor.salotto_lux < 30 lx',
    x: 320,
    y: 120,
    entityKey: 'sensor.salotto_lux'
  },
  {
    id: 'n_del_1',
    type: 'delay',
    title: 'Attendi 2s',
    subtitle: 'Pausa soft fade-in',
    x: 580,
    y: 120
  },
  {
    id: 'n_act_1',
    type: 'action',
    title: 'Accendi Luce Calda',
    subtitle: 'light.salotto_piantana -> turn_on',
    x: 820,
    y: 120,
    entityKey: 'light.salotto_piantana',
    service: 'turn_on'
  }
])

const connections = ref<FlowConnection[]>([
  { from: 'n_trig_1', to: 'n_cond_1' },
  { from: 'n_cond_1', to: 'n_del_1' },
  { from: 'n_del_1', to: 'n_act_1' }
])

const isSimulating = ref(false)
const activeSimIndex = ref(-1)

// Selection & Inspector State
const selectedNodeId = ref<string | null>('n_cond_1')
const selectedNode = computed(() => nodes.value.find((n) => n.id === selectedNodeId.value) || null)
const activePanel = ref<'inspector' | 'dryrun'>('inspector')

// Dragging State
const isDraggingNode = ref(false)
const draggingNodeId = ref<string | null>(null)
let dragStartPointer = { x: 0, y: 0 }
let dragStartNodePos = { x: 0, y: 0 }

function startDrag(e: PointerEvent, node: FlowNode) {
  // Prevent drag if clicked on a button or form element
  if ((e.target as HTMLElement).closest('button, input, select, a')) return

  selectedNodeId.value = node.id
  activePanel.value = 'inspector'
  isDraggingNode.value = true
  draggingNodeId.value = node.id
  dragStartPointer = { x: e.clientX, y: e.clientY }
  dragStartNodePos = { x: node.x, y: node.y }

  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', onPointerUp)
}

function onPointerMove(e: PointerEvent) {
  if (!isDraggingNode.value || !draggingNodeId.value) return
  const node = nodes.value.find((n) => n.id === draggingNodeId.value)
  if (!node) return

  const dx = e.clientX - dragStartPointer.x
  const dy = e.clientY - dragStartPointer.y

  node.x = Math.max(20, Math.min(2200, dragStartNodePos.x + dx))
  node.y = Math.max(20, Math.min(1200, dragStartNodePos.y + dy))
}

function onPointerUp() {
  isDraggingNode.value = false
  draggingNodeId.value = null
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
}

onBeforeUnmount(() => {
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
})

// Bézier curve calculation between nodes
function calculateBezier(from: FlowNode, to: FlowNode): string {
  const x1 = from.x + 200
  const y1 = from.y + 44
  const x2 = to.x
  const y2 = to.y + 44
  const dx = Math.max(50, Math.abs(x2 - x1) * 0.5)
  return `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`
}

// Node Presets
interface NodePreset {
  id: string
  label: string
  title: string
  subtitle: string
  entityKey?: string
  service?: string
}

const CONDITION_PRESETS: NodePreset[] = [
  { id: 'custom', label: '✏️ Condizione Personalizzata', title: 'Condizione Personalizzata', subtitle: 'regola_personalizzata == true' },
  { id: 'lux_low', label: '☀️ Luminosità Sensore Bassa (< 30 lx)', title: 'Luminosità Bassa', subtitle: 'sensor.salotto_lux < 30 lx', entityKey: 'sensor.salotto_lux' },
  { id: 'lux_high', label: '☀️ Luminosità Sensore Alta (>= 50 lx)', title: 'Luminosità Alta', subtitle: 'sensor.salotto_lux >= 50 lx', entityKey: 'sensor.salotto_lux' },
  { id: 'presence_home', label: '🏠 Presenza Utente (A casa)', title: 'Presenza Utente', subtitle: 'person.utente == home', entityKey: 'person.utente' },
  { id: 'presence_away', label: '🚶 Nessuno a Casa (Casa Vuota)', title: 'Casa Vuota', subtitle: 'group.all_persons == not_home', entityKey: 'group.all_persons' },
  { id: 'night_time', label: '🌙 Fascia Oraria Notturna (22:00 - 06:00)', title: 'Fascia Notturna', subtitle: 'time >= 22:00 || time <= 06:00', entityKey: 'time' },
  { id: 'day_time', label: '☀️ Fascia Oraria Diurna (06:00 - 22:00)', title: 'Fascia Diurna', subtitle: 'time >= 06:00 && time <= 22:00', entityKey: 'time' },
  { id: 'temp_high', label: '🌡️ Temperatura Alta (> 24°C)', title: 'Temperatura Alta', subtitle: 'sensor.temperatura > 24°C', entityKey: 'sensor.temperatura' },
  { id: 'temp_low', label: '❄️ Temperatura Bassa (< 18°C)', title: 'Temperatura Bassa', subtitle: 'sensor.temperatura < 18°C', entityKey: 'sensor.temperatura' },
  { id: 'window_open', label: '🚪 Finestra o Porta Aperta', title: 'Finestra Aperta', subtitle: 'binary_sensor.finestra == on', entityKey: 'binary_sensor.finestra' },
  { id: 'battery_low', label: '🪫 Batteria Dispositivo Bassa (< 20%)', title: 'Batteria Bassa', subtitle: 'sensor.battery_level < 20%', entityKey: 'sensor.battery_level' }
]

const TRIGGER_PRESETS: NodePreset[] = [
  { id: 'custom', label: '✏️ Trigger Personalizzato', title: 'Trigger Personalizzato', subtitle: 'evento_scatenante == true' },
  { id: 'motion', label: '🚶 Rilevato Movimento (PIR)', title: 'Rilevato Movimento', subtitle: 'binary_sensor.salotto_motion == true', entityKey: 'binary_sensor.salotto_motion' },
  { id: 'time', label: '⏰ Orario Fisso (22:30)', title: 'Trigger Orario', subtitle: 'Alle 22:30 ogni giorno', entityKey: 'time' },
  { id: 'sunset', label: '🌅 Tramonto Solare', title: 'Tramonto Solare', subtitle: 'sun.sun == below_horizon', entityKey: 'sun.sun' },
  { id: 'switch', label: '🔘 Pressione Pulsante / Click', title: 'Pulsante Premuto', subtitle: 'event.button_click == single', entityKey: 'event.button_click' },
  { id: 'webhook', label: '🌐 Webhook Inbound Ricevuto', title: 'Webhook Ricevuto', subtitle: 'webhook.inbound == trigger', entityKey: 'webhook.inbound' },
  { id: 'state_change', label: '⚡ Cambio Stato Entità', title: 'Cambio Stato Sensore', subtitle: 'binary_sensor.porta == open', entityKey: 'binary_sensor.porta' }
]

const ACTION_PRESETS: NodePreset[] = [
  { id: 'custom', label: '✏️ Azione Personalizzata', title: 'Azione Personalizzata', subtitle: 'servizio -> esegui' },
  { id: 'turn_on', label: '💡 Accendi Luce', title: 'Accendi Luce Calda', subtitle: 'light.salotto_piantana -> turn_on', entityKey: 'light.salotto_piantana', service: 'turn_on' },
  { id: 'turn_off', label: '🌑 Spegni Luce', title: 'Spegni Luce', subtitle: 'light.salotto_piantana -> turn_off', entityKey: 'light.salotto_piantana', service: 'turn_off' },
  { id: 'dim', label: '🔆 Imposta Luminosità (50%)', title: 'Dimmer 50%', subtitle: 'light.salotto_piantana -> brightness: 128', entityKey: 'light.salotto_piantana', service: 'turn_on' },
  { id: 'scene', label: '🎬 Attiva Scena Cinema', title: 'Attiva Scena Cinema', subtitle: 'scene.cinema -> apply', entityKey: 'scene.cinema', service: 'apply' },
  { id: 'notify', label: '📱 Notifica Push Smartphone', title: 'Notifica Smartphone', subtitle: 'notify.mobile_app -> Allerta Casa', entityKey: 'notify.mobile_app', service: 'notify' },
  { id: 'plug', label: '🔌 Attiva Presa Smart', title: 'Attiva Presa Smart', subtitle: 'switch.presa_tv -> turn_on', entityKey: 'switch.presa_tv', service: 'turn_on' },
  { id: 'vacuum', label: '🤖 Avvia Pulizia Robot', title: 'Avvia Aspirapolvere', subtitle: 'vacuum.robot -> start', entityKey: 'vacuum.robot', service: 'start' }
]

const DELAY_PRESETS = [
  { seconds: 2, label: '2s' },
  { seconds: 5, label: '5s' },
  { seconds: 10, label: '10s' },
  { seconds: 30, label: '30s' },
  { seconds: 60, label: '60s' }
]

function applyPreset(presetId: string) {
  if (!selectedNode.value) return
  let list: NodePreset[] = []
  if (selectedNode.value.type === 'condition') list = CONDITION_PRESETS
  else if (selectedNode.value.type === 'trigger') list = TRIGGER_PRESETS
  else if (selectedNode.value.type === 'action') list = ACTION_PRESETS

  const found = list.find((p) => p.id === presetId)
  if (found && found.id !== 'custom') {
    selectedNode.value.title = found.title
    selectedNode.value.subtitle = found.subtitle
    if (found.entityKey) selectedNode.value.entityKey = found.entityKey
    if (found.service) selectedNode.value.service = found.service
    audioSynthesizer.playClick()
    if (activePanel.value === 'dryrun') executeDryRun()
  }
}

function setDelaySeconds(sec: number) {
  if (!selectedNode.value || selectedNode.value.type !== 'delay') return
  selectedNode.value.title = `Ritardo ${sec}s`
  selectedNode.value.subtitle = `Attesa di ${sec} secondi prima del prossimo step`
  audioSynthesizer.playClick()
}

// Dry-Run Simulator State
const dryRunInputs = ref({
  motion: true,
  lux: 15,
  time: '22:30',
  presence: true
})

interface StepEval {
  status: 'passed' | 'failed' | 'pending'
  detail: string
}
const nodeEvals = ref<Record<string, StepEval>>({})
const dryRunLogs = ref<string[]>([])

function toggleDryRun() {
  audioSynthesizer.playClick()
  activePanel.value = activePanel.value === 'dryrun' ? 'inspector' : 'dryrun'
  if (activePanel.value === 'dryrun') {
    executeDryRun()
  }
}

function toggleInspector() {
  audioSynthesizer.playClick()
  activePanel.value = activePanel.value === 'inspector' ? 'dryrun' : 'inspector'
}

function executeDryRun() {
  audioSynthesizer.playClick()
  nodeEvals.value = {}
  dryRunLogs.value = []

  // 1. Evaluate Trigger
  const trigNode = nodes.value.find((n) => n.type === 'trigger')
  if (trigNode) {
    if (dryRunInputs.value.motion) {
      nodeEvals.value[trigNode.id] = { status: 'passed', detail: 'Trigger scatenato: Movimento rilevato (true)' }
      dryRunLogs.value.push(`[1/3] Trigger [${trigNode.title}]: VERIFICATO (movimento = true)`)
    } else {
      nodeEvals.value[trigNode.id] = { status: 'failed', detail: 'Nessun movimento (false)' }
      dryRunLogs.value.push(`[1/3] Trigger [${trigNode.title}]: NON ATTIVATO`)
      return
    }
  }

  // 2. Evaluate Condition(s) dynamically
  const condNodes = nodes.value.filter((n) => n.type === 'condition')
  for (const condNode of condNodes) {
    const text = (condNode.title + ' ' + condNode.subtitle).toLowerCase()
    let passed = false
    let reason = ''

    if (text.includes('lux') || text.includes('luminosità')) {
      if (text.includes('>=') || text.includes('alta') || text.includes('> 50')) {
        passed = dryRunInputs.value.lux >= 50
        reason = passed ? `${dryRunInputs.value.lux} lx >= 50 lx` : `${dryRunInputs.value.lux} lx < 50 lx (minimo 50 lx)`
      } else {
        passed = dryRunInputs.value.lux < 30
        reason = passed ? `${dryRunInputs.value.lux} lx < 30 lx` : `${dryRunInputs.value.lux} lx >= 30 lx (massimo 30 lx)`
      }
    } else if (text.includes('casa') || text.includes('presenza') || text.includes('persona')) {
      if (text.includes('vuota') || text.includes('nessuno') || text.includes('not_home')) {
        passed = !dryRunInputs.value.presence
        reason = passed ? 'Nessuno presente a casa' : 'Rilevata presenza (condizione richiede casa vuota)'
      } else {
        passed = dryRunInputs.value.presence
        reason = passed ? 'Utente presente a casa' : 'Nessuno presente (condizione richiede presenza)'
      }
    } else if (text.includes('notturna') || text.includes('22:00')) {
      passed = dryRunInputs.value.time >= '22:00' || dryRunInputs.value.time <= '06:00'
      reason = passed ? `Orario ${dryRunInputs.value.time} in fascia notturna` : `Orario ${dryRunInputs.value.time} fuori fascia notturna`
    } else {
      passed = true
      reason = 'Condizione verificata con successo'
    }

    if (passed) {
      nodeEvals.value[condNode.id] = { status: 'passed', detail: `Condizione soddisfatta: ${reason}` }
      dryRunLogs.value.push(`[2/3] Condizione [${condNode.title}]: PASSATA (${reason})`)
    } else {
      nodeEvals.value[condNode.id] = { status: 'failed', detail: `Condizione fallita: ${reason}` }
      dryRunLogs.value.push(`[2/3] Condizione [${condNode.title}]: BLOCCATA (${reason})`)
      return
    }
  }

  // 3. Delays & Actions
  const actNodes = nodes.value.filter((n) => n.type === 'action' || n.type === 'delay')
  for (const act of actNodes) {
    nodeEvals.value[act.id] = { status: 'passed', detail: 'Azione pianificata ed eseguita' }
    dryRunLogs.value.push(`[3/3] Esecuzione: ${act.title} -> SUCCESSO`)
  }

  audioSynthesizer.playSuccessChime()
}

function getNodeById(id: string) {
  return nodes.value.find((n) => n.id === id)
}

function simulateFlow() {
  if (isSimulating.value) return
  isSimulating.value = true
  activeSimIndex.value = 0

  const interval = setInterval(() => {
    activeSimIndex.value++
    if (activeSimIndex.value >= nodes.value.length) {
      clearInterval(interval)
      setTimeout(() => {
        isSimulating.value = false
        activeSimIndex.value = -1
      }, 600)
    }
  }, 700)
}

function addNode(type: 'trigger' | 'condition' | 'delay' | 'action') {
  const id = `n_${type}_${Date.now()}`
  const lastNode = nodes.value[nodes.value.length - 1]
  const newX = lastNode ? lastNode.x + 240 : 100
  const newY = lastNode ? lastNode.y : 120

  let title = 'Nuovo Blocco'
  let subtitle = ''
  let entityKey: string | undefined = undefined
  let service: string | undefined = undefined

  if (type === 'trigger') {
    title = 'Rilevato Movimento'
    subtitle = 'binary_sensor.salotto_motion == true'
    entityKey = 'binary_sensor.salotto_motion'
  } else if (type === 'condition') {
    title = 'Luminosità Bassa'
    subtitle = 'sensor.salotto_lux < 30 lx'
    entityKey = 'sensor.salotto_lux'
  } else if (type === 'delay') {
    title = 'Ritardo 5s'
    subtitle = 'Attesa di 5 secondi prima del prossimo blocco'
  } else {
    title = 'Accendi Luce Calda'
    subtitle = 'light.salotto_piantana -> turn_on'
    entityKey = 'light.salotto_piantana'
    service = 'turn_on'
  }

  const newNode: FlowNode = { id, type, title, subtitle, x: newX, y: newY, entityKey, service }
  nodes.value.push(newNode)

  if (lastNode) {
    connections.value.push({ from: lastNode.id, to: id })
  }

  // Automatically select the new node and open the Inspector to customize the condition/parameters right away!
  selectedNodeId.value = id
  activePanel.value = 'inspector'
  audioSynthesizer.playClick()
}

function removeNode(id: string) {
  nodes.value = nodes.value.filter((n) => n.id !== id)
  connections.value = connections.value.filter((c) => c.from !== id && c.to !== id)
  if (selectedNodeId.value === id) {
    selectedNodeId.value = nodes.value[0]?.id || null
  }
}

function handleSave() {
  emit('save', {
    name: 'Automazione da Flowchart Visuale',
    nodes: nodes.value,
    connections: connections.value
  })
  emit('close')
}
</script>

<template>
  <div v-if="isOpen" class="modal-backdrop">
    <div class="flowchart-modal glass-card">
      <div class="modal-header">
        <div class="header-titles">
          <div class="icon-chip"><Layers :size="20" /></div>
          <div>
            <h3 class="modal-title">{{ t('flowchart.modalTitle') }}</h3>
            <p class="modal-sub">{{ t('flowchart.desc') }}</p>
          </div>
        </div>

        <div class="header-actions">
          <button
            class="btn-panel-tab"
            :class="{ active: activePanel === 'inspector' && selectedNode }"
            @click="toggleInspector"
          >
            <Sliders :size="15" />
            <span>⚙️ Configura Nodo</span>
          </button>
          <button
            class="btn-dryrun"
            :class="{ active: activePanel === 'dryrun' }"
            @click="toggleDryRun"
          >
            <span>🧪 Collaudo Dry-Run</span>
          </button>
          <button class="btn-simulate" :disabled="isSimulating" @click="simulateFlow">
            <Play :size="15" :class="{ 'animate-spin': isSimulating }" />
            <span>{{ isSimulating ? t('flowchart.btnPause') : t('flowchart.btnSimulate') }}</span>
          </button>
          <button class="btn-close" aria-label="Chiudi finestra" @click="emit('close')">
            <X :size="18" />
          </button>
        </div>
      </div>

      <!-- Toolbar Add Nodes -->
      <div class="flow-toolbar">
        <span class="tool-label">Aggiungi Blocco:</span>
        <button class="btn-tool type-trig" @click="addNode('trigger')">
          <Zap :size="14" />
          <span>{{ t('flowchart.addTrigger') }}</span>
        </button>
        <button class="btn-tool type-cond" @click="addNode('condition')">
          <Filter :size="14" />
          <span>{{ t('flowchart.addCondition') }}</span>
        </button>
        <button class="btn-tool type-del" @click="addNode('delay')">
          <Clock :size="14" />
          <span>+ Ritardo</span>
        </button>
        <button class="btn-tool type-act" @click="addNode('action')">
          <ArrowRight :size="14" />
          <span>{{ t('flowchart.addAction') }}</span>
        </button>
      </div>

      <!-- Node Graph Canvas Viewport -->
      <div class="canvas-viewport" @click.self="selectedNodeId = null">
        <div class="canvas-grid">
          <!-- Connection Lines SVG -->
          <svg class="connections-svg">
            <g v-for="(conn, ci) in connections" :key="ci">
              <path
                v-if="getNodeById(conn.from) && getNodeById(conn.to)"
                :d="calculateBezier(getNodeById(conn.from)!, getNodeById(conn.to)!)"
                :class="['wire-path', { active: isSimulating && activeSimIndex === ci }]"
              />
            </g>
          </svg>

          <!-- Nodes Cards -->
          <div
            v-for="(n, ni) in nodes"
            :key="n.id"
            :class="[
              'flow-node',
              `node-${n.type}`,
              {
                'sim-active': isSimulating && activeSimIndex === ni,
                'eval-passed': nodeEvals[n.id]?.status === 'passed',
                'eval-failed': nodeEvals[n.id]?.status === 'failed',
                'is-selected': selectedNodeId === n.id,
                'is-dragging': isDraggingNode && draggingNodeId === n.id
              }
            ]"
            :style="{ left: `${n.x}px`, top: `${n.y}px` }"
            @pointerdown="(e) => startDrag(e, n)"
            @click="selectedNodeId = n.id; activePanel = 'inspector'"
          >
            <!-- In Port Indicator -->
            <div v-if="n.type !== 'trigger'" class="node-port port-in" title="Ingresso"></div>

            <div class="node-header">
              <div class="node-badge-wrap">
                <span class="node-badge">{{ n.type.toUpperCase() }}</span>
                <span class="drag-hint" title="Trascina con il mouse">⋮⋮</span>
              </div>
              <span v-if="nodeEvals[n.id]" class="eval-pill" :class="nodeEvals[n.id].status">
                {{ nodeEvals[n.id].status === 'passed' ? 'PASSATO' : 'BLOCCATO' }}
              </span>
              <button
                v-if="nodes.length > 1"
                class="btn-node-del"
                aria-label="Elimina blocco"
                @click.stop="removeNode(n.id)"
              >
                <Trash2 :size="12" />
              </button>
            </div>

            <h4 class="node-title">{{ n.title }}</h4>
            <p class="node-sub">{{ n.subtitle }}</p>
            <p v-if="nodeEvals[n.id]" class="node-eval-msg" :class="nodeEvals[n.id].status">
              {{ nodeEvals[n.id].detail }}
            </p>

            <!-- Out Port Indicator -->
            <div class="node-port port-out" title="Uscita"></div>
          </div>
        </div>

        <!-- Sidebar Panel 1: Node Inspector (Configuratore di Blocco) -->
        <aside v-if="selectedNode && activePanel === 'inspector'" class="sidebar-panel inspector-panel">
          <div class="panel-header">
            <div class="panel-title-box">
              <span class="panel-badge" :class="`badge-${selectedNode.type}`">
                {{ selectedNode.type.toUpperCase() }}
              </span>
              <h4 class="panel-title">Configura Nodo</h4>
            </div>
            <button class="btn-panel-close" aria-label="Chiudi ispettore" @click="selectedNodeId = null">
              <X :size="16" />
            </button>
          </div>

          <div class="panel-body">
            <!-- Preset Selector for Condition -->
            <div v-if="selectedNode.type === 'condition'" class="form-group">
              <label for="preset-condition" class="panel-label">⚡ Seleziona Condizione Predefinita</label>
              <select
                id="preset-condition"
                name="preset_condition"
                class="panel-select"
                @change="(e) => applyPreset((e.target as HTMLSelectElement).value)"
              >
                <option value="" disabled selected>-- Scegli Condizione --</option>
                <option v-for="p in CONDITION_PRESETS" :key="p.id" :value="p.id">
                  {{ p.label }}
                </option>
              </select>
            </div>

            <!-- Preset Selector for Trigger -->
            <div v-else-if="selectedNode.type === 'trigger'" class="form-group">
              <label for="preset-trigger" class="panel-label">⚡ Seleziona Trigger Predefinito</label>
              <select
                id="preset-trigger"
                name="preset_trigger"
                class="panel-select"
                @change="(e) => applyPreset((e.target as HTMLSelectElement).value)"
              >
                <option value="" disabled selected>-- Scegli Trigger --</option>
                <option v-for="p in TRIGGER_PRESETS" :key="p.id" :value="p.id">
                  {{ p.label }}
                </option>
              </select>
            </div>

            <!-- Preset Selector for Action -->
            <div v-else-if="selectedNode.type === 'action'" class="form-group">
              <label for="preset-action" class="panel-label">⚡ Seleziona Azione Predefinita</label>
              <select
                id="preset-action"
                name="preset_action"
                class="panel-select"
                @change="(e) => applyPreset((e.target as HTMLSelectElement).value)"
              >
                <option value="" disabled selected>-- Scegli Azione --</option>
                <option v-for="p in ACTION_PRESETS" :key="p.id" :value="p.id">
                  {{ p.label }}
                </option>
              </select>
            </div>

            <!-- Delay Duration Pills -->
            <div v-else-if="selectedNode.type === 'delay'" class="form-group">
              <label class="panel-label">⏱️ Durata Ritardo</label>
              <div class="delay-pills">
                <button
                  v-for="d in DELAY_PRESETS"
                  :key="d.seconds"
                  type="button"
                  class="delay-pill"
                  :class="{ active: selectedNode.title.includes(`${d.seconds}s`) }"
                  @click="setDelaySeconds(d.seconds)"
                >
                  {{ d.label }}
                </button>
              </div>
            </div>

            <!-- Editable Fields -->
            <div class="panel-divider"></div>

            <div class="form-group">
              <label for="node-edit-title" class="panel-label">Nome / Titolo Blocco</label>
              <input
                id="node-edit-title"
                name="node_title"
                v-model="selectedNode.title"
                type="text"
                class="panel-input"
                placeholder="es. Luminosità Bassa"
              />
            </div>

            <div class="form-group">
              <label for="node-edit-subtitle" class="panel-label">
                {{ selectedNode.type === 'condition' ? 'Regola o Soglia' : selectedNode.type === 'delay' ? 'Descrizione Attesa' : 'Descrizione / Target' }}
              </label>
              <input
                id="node-edit-subtitle"
                name="node_subtitle"
                v-model="selectedNode.subtitle"
                type="text"
                class="panel-input"
                placeholder="es. sensor.lux < 30 lx"
              />
            </div>

            <div v-if="selectedNode.type !== 'delay'" class="form-group">
              <label for="node-edit-entity" class="panel-label">Entità di Sistema (opzionale)</label>
              <input
                id="node-edit-entity"
                name="node_entity"
                v-model="selectedNode.entityKey"
                type="text"
                class="panel-input"
                placeholder="es. sensor.salotto_lux"
              />
            </div>

            <div v-if="selectedNode.type === 'action'" class="form-group">
              <label for="node-edit-service" class="panel-label">Servizio / Comando (opzionale)</label>
              <input
                id="node-edit-service"
                name="node_service"
                v-model="selectedNode.service"
                type="text"
                class="panel-input"
                placeholder="es. turn_on, turn_off"
              />
            </div>

            <div class="panel-actions">
              <button
                v-if="nodes.length > 1"
                type="button"
                class="btn-delete-block"
                @click="removeNode(selectedNode.id)"
              >
                <Trash2 :size="14" />
                <span>Elimina questo blocco</span>
              </button>
            </div>
          </div>
        </aside>

        <!-- Sidebar Panel 2: Dry-Run Simulator Panel -->
        <aside v-if="activePanel === 'dryrun'" class="sidebar-panel dryrun-panel">
          <div class="panel-header">
            <div class="panel-title-box">
              <h4 class="panel-title">🧪 Collaudo Dry-Run</h4>
            </div>
            <button class="btn-panel-close" aria-label="Chiudi simulatore" @click="activePanel = 'inspector'">
              <X :size="16" />
            </button>
          </div>

          <div class="panel-body">
            <p class="dryrun-desc">Inietta valori simulati per collaudare in tempo reale le condizioni prima del salvataggio:</p>

            <div class="form-group">
              <label class="checkbox-row">
                <input type="checkbox" v-model="dryRunInputs.motion" @change="executeDryRun" />
                <span>Rilevato Movimento (Sensore PIR)</span>
              </label>
            </div>

            <div class="form-group">
              <label class="checkbox-row">
                <input type="checkbox" v-model="dryRunInputs.presence" @change="executeDryRun" />
                <span>Presenza Utente a Casa</span>
              </label>
            </div>

            <div class="form-group">
              <label for="dryrun-lux" class="panel-label">
                Luminosità Sensore: <strong>{{ dryRunInputs.lux }} lx</strong>
              </label>
              <input
                id="dryrun-lux"
                name="dryrun_lux"
                type="range"
                min="0"
                max="100"
                v-model.number="dryRunInputs.lux"
                class="panel-range"
                @input="executeDryRun"
              />
            </div>

            <div class="form-group">
              <label for="dryrun-time" class="panel-label">Orario Simulato</label>
              <input
                id="dryrun-time"
                name="dryrun_time"
                type="time"
                v-model="dryRunInputs.time"
                class="panel-input"
                @change="executeDryRun"
              />
            </div>

            <div class="dryrun-trace-box">
              <h5 class="trace-title">Trace Esecuzione:</h5>
              <div v-for="(log, li) in dryRunLogs" :key="li" class="trace-line">
                {{ log }}
              </div>
            </div>
          </div>
        </aside>
      </div>

      <!-- Footer -->
      <div class="modal-footer">
        <span class="footer-tip">Trascina o seleziona i blocchi per configurare le regole native Domus.</span>
        <div class="footer-btns">
          <button class="btn-cancel" @click="emit('close')">Annulla</button>
          <button class="btn-save" @click="handleSave">
            <Check :size="16" />
            <span>{{ t('flowchart.btnSave') }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 24px;
}

.flowchart-modal {
  width: 100%;
  max-width: 1100px;
  height: 85vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-2xl);
  overflow: hidden;
  box-shadow: 0 25px 60px rgba(0, 0, 0, 0.5);
}

.modal-header {
  padding: 18px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-subtle);
  background: rgba(0, 0, 0, 0.1);
}

.header-titles {
  display: flex;
  align-items: center;
  gap: 14px;
}

.icon-chip {
  width: 42px;
  height: 42px;
  border-radius: var(--radius-md);
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.modal-sub {
  font-size: 13px;
  color: var(--text-muted);
  margin: 2px 0 0 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.btn-panel-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-muted);
  border: 1px solid var(--border-subtle);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-panel-tab:hover,
.btn-panel-tab.active {
  background: rgba(14, 165, 233, 0.2);
  color: var(--primary);
  border-color: var(--primary);
}

.btn-dryrun {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-muted);
  border: 1px solid var(--border-subtle);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-dryrun:hover,
.btn-dryrun.active {
  background: rgba(16, 185, 129, 0.2);
  color: #10b981;
  border-color: #10b981;
  box-shadow: 0 0 12px rgba(16, 185, 129, 0.3);
}

.btn-simulate {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: var(--radius-md);
  background: #10b981;
  color: #fff;
  border: none;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-simulate:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.btn-close {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 6px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
}

.btn-close:hover {
  color: var(--text-main);
  background: rgba(255, 255, 255, 0.08);
}

.flow-toolbar {
  padding: 10px 24px;
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(0, 0, 0, 0.2);
  border-bottom: 1px solid var(--border-subtle);
}

.tool-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  margin-right: 4px;
}

.btn-tool {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid transparent;
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-main);
  transition: all var(--transition-fast);
}

.btn-tool:hover {
  background: rgba(255, 255, 255, 0.12);
  transform: translateY(-1px);
}

.type-trig { color: #10b981; border-color: rgba(16, 185, 129, 0.3); }
.type-cond { color: #f59e0b; border-color: rgba(245, 158, 11, 0.3); }
.type-del  { color: #0ea5e9; border-color: rgba(14, 165, 233, 0.3); }
.type-act  { color: #a855f7; border-color: rgba(168, 85, 247, 0.3); }

/* Canvas Area */
.canvas-viewport {
  flex: 1;
  position: relative;
  overflow: auto;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.04) 1px, transparent 1px);
  background-size: 24px 24px;
}

.canvas-grid {
  position: relative;
  min-width: 1400px;
  min-height: 600px;
  padding: 40px;
}

.connections-svg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.wire-path {
  fill: none;
  stroke: rgba(255, 255, 255, 0.25);
  stroke-width: 3;
  stroke-dasharray: 6 6;
  transition: stroke 0.3s ease, stroke-width 0.3s ease;
}

.wire-path.active {
  stroke: #10b981;
  stroke-width: 5;
  stroke-dasharray: none;
  filter: drop-shadow(0 0 8px #10b981);
}

/* Nodes */
.flow-node {
  position: absolute;
  width: 200px;
  padding: 14px;
  border-radius: var(--radius-lg);
  background: var(--bg-glass-card);
  border: 1px solid var(--border-subtle);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
  display: flex;
  flex-direction: column;
  gap: 6px;
  cursor: grab;
  user-select: none;
  touch-action: none;
  transition: box-shadow 0.2s ease, border-color 0.2s ease, transform 0.2s ease;
}

.flow-node.is-dragging {
  cursor: grabbing !important;
  transition: none !important;
  z-index: 50 !important;
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.5), 0 0 0 2px var(--primary);
  transform: scale(1.02);
}

.flow-node.is-selected {
  border-color: var(--primary) !important;
  box-shadow: 0 0 0 2px rgba(14, 165, 233, 0.45), 0 10px 28px rgba(0, 0, 0, 0.4) !important;
}

.flow-node.sim-active {
  transform: scale(1.06);
  box-shadow: 0 0 30px rgba(16, 185, 129, 0.6);
  border-color: #10b981;
}

.node-trigger { border-left: 4px solid #10b981; }
.node-condition { border-left: 4px solid #f59e0b; }
.node-delay { border-left: 4px solid #0ea5e9; }
.node-action { border-left: 4px solid #a855f7; }

/* Connection Port Indicators */
.node-port {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #1e293b;
  border: 2px solid var(--text-muted);
  z-index: 2;
  transition: all 0.2s ease;
}

.port-in {
  left: -6px;
}

.port-out {
  right: -6px;
  background: var(--primary);
  border-color: #fff;
}

.flow-node:hover .node-port {
  transform: translateY(-50%) scale(1.25);
  box-shadow: 0 0 8px var(--primary);
}

.node-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.node-badge-wrap {
  display: flex;
  align-items: center;
  gap: 6px;
}

.node-badge {
  font-size: 10px;
  font-weight: 700;
  color: var(--text-muted);
  letter-spacing: 0.05em;
}

.drag-hint {
  font-size: 11px;
  color: var(--text-dim);
  cursor: grab;
  opacity: 0.6;
}

.flow-node:hover .drag-hint {
  opacity: 1;
}

.btn-node-del {
  background: transparent;
  border: none;
  color: var(--text-dim);
  cursor: pointer;
  padding: 2px;
  border-radius: 4px;
}

.btn-node-del:hover {
  color: var(--color-danger);
  background: rgba(239, 68, 68, 0.15);
}

.node-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.node-sub {
  font-size: 11px;
  color: var(--text-muted);
  line-height: 1.4;
  word-break: break-all;
  margin: 0;
}

.eval-pill {
  font-size: 9px;
  font-weight: 800;
  padding: 2px 5px;
  border-radius: 4px;
  letter-spacing: 0.04em;
}

.eval-pill.passed {
  background: rgba(16, 185, 129, 0.2);
  color: #10b981;
  border: 1px solid rgba(16, 185, 129, 0.4);
}

.eval-pill.failed {
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.4);
}

.node-eval-msg {
  font-size: 10px;
  padding: 4px 6px;
  border-radius: 4px;
  margin-top: 4px;
}

.node-eval-msg.passed {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.node-eval-msg.failed {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.flow-node.eval-passed {
  border-color: #10b981 !important;
  box-shadow: 0 0 24px rgba(16, 185, 129, 0.45) !important;
}

.flow-node.eval-failed {
  border-color: #ef4444 !important;
  box-shadow: 0 0 24px rgba(239, 68, 68, 0.45) !important;
}

/* Sidebar Panels (Inspector & Dry-Run) */
.sidebar-panel {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  width: 330px;
  background: rgba(15, 23, 42, 0.95);
  backdrop-filter: blur(16px);
  border-left: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  z-index: 20;
  box-shadow: -6px 0 24px rgba(0, 0, 0, 0.45);
}

.panel-header {
  padding: 16px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-subtle);
  background: rgba(0, 0, 0, 0.2);
}

.panel-title-box {
  display: flex;
  align-items: center;
  gap: 10px;
}

.panel-badge {
  font-size: 10px;
  font-weight: 800;
  padding: 2px 7px;
  border-radius: 4px;
  letter-spacing: 0.05em;
}

.badge-trigger   { background: rgba(16, 185, 129, 0.2); color: #10b981; border: 1px solid rgba(16, 185, 129, 0.3); }
.badge-condition { background: rgba(245, 158, 11, 0.2);  color: #f59e0b; border: 1px solid rgba(245, 158, 11, 0.3); }
.badge-delay     { background: rgba(14, 165, 233, 0.2);  color: #0ea5e9; border: 1px solid rgba(14, 165, 233, 0.3); }
.badge-action    { background: rgba(168, 85, 247, 0.2);  color: #a855f7; border: 1px solid rgba(168, 85, 247, 0.3); }

.panel-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.btn-panel-close {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-panel-close:hover {
  color: var(--text-main);
  background: rgba(255, 255, 255, 0.08);
}

.panel-body {
  padding: 16px 20px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.panel-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
}

.panel-select,
.panel-input {
  width: 100%;
  box-sizing: border-box;
  padding: 8px 12px;
  background: rgba(0, 0, 0, 0.35);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  color: var(--text-main);
  font-size: 13px;
  outline: none;
  transition: border-color 0.2s ease;
}

.panel-select option {
  background: #1e293b;
  color: #fff;
}

.panel-select:focus,
.panel-input:focus {
  border-color: var(--primary);
}

.delay-pills {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.delay-pill {
  padding: 6px 12px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.delay-pill:hover,
.delay-pill.active {
  background: rgba(14, 165, 233, 0.25);
  border-color: var(--primary);
  color: #fff;
}

.panel-divider {
  height: 1px;
  background: var(--border-subtle);
  margin: 4px 0;
}

.panel-actions {
  margin-top: 8px;
}

.btn-delete-block {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 8px 12px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
  border-radius: var(--radius-md);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-delete-block:hover {
  background: rgba(239, 68, 68, 0.2);
  border-color: #ef4444;
}

.checkbox-row {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--text-main);
  cursor: pointer;
}

.checkbox-row input[type="checkbox"] {
  accent-color: var(--primary);
  width: 16px;
  height: 16px;
}

.dryrun-desc {
  font-size: 12px;
  color: var(--text-muted);
  margin: 0;
  line-height: 1.4;
}

.panel-range {
  width: 100%;
  accent-color: var(--primary);
}

.dryrun-trace-box {
  background: rgba(0, 0, 0, 0.4);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 10px;
  max-height: 150px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.trace-title {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--text-muted);
  margin: 0 0 4px 0;
}

.trace-line {
  font-family: monospace;
  font-size: 11px;
  color: #10b981;
  word-break: break-all;
}

/* Footer */
.modal-footer {
  padding: 16px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-top: 1px solid var(--border-subtle);
  background: var(--bg-surface);
}

.footer-tip {
  font-size: 12px;
  color: var(--text-muted);
}

.footer-btns {
  display: flex;
  align-items: center;
  gap: 12px;
}

.btn-cancel {
  padding: 8px 16px;
  border-radius: var(--radius-md);
  background: transparent;
  border: 1px solid var(--border-subtle);
  color: var(--text-main);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-cancel:hover {
  background: rgba(255, 255, 255, 0.05);
}

.btn-save {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 18px;
  border-radius: var(--radius-md);
  background: var(--primary);
  color: #fff;
  border: none;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-save:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}
</style>
