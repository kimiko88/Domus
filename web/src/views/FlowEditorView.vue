<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAutomationsStore } from '../stores/automations'
import { useEntitiesStore } from '../stores/entities'
import { useScenesStore } from '../stores/scenes'
import { useNotificationsStore } from '../stores/notifications'
import { soundEffects } from '../utils/audioSynthesizer'
import { apiClient } from '../api/client'
import type { Automation, AutomationTrigger, AutomationCondition, AutomationAction } from '../api/types'
import {
  ArrowLeft,
  Save,
  Play,
  RotateCcw,
  ZoomIn,
  ZoomOut,
  Maximize2,
  Trash2,
  Plus,
  Zap,
  Filter,
  Clock,
  Sliders,
  Send,
  Layers,
  Code,
  Copy,
  Check,
  X,
  AlertCircle,
  HelpCircle,
  Sparkles,
  Link,
  ChevronRight,
  Sun,
  Activity
} from '@lucide/vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const automationsStore = useAutomationsStore()
const entitiesStore = useEntitiesStore()
const scenesStore = useScenesStore()
const notificationsStore = useNotificationsStore()

// --- Flow Model Types ---
export type FlowNodeType = 'trigger' | 'condition' | 'action' | 'delay' | 'logic'

export interface FlowNode {
  id: string
  type: FlowNodeType
  subType: string // e.g. "state", "numeric_state", "time", "service", "scene", "delay", "and", "or"
  title: string
  subtitle: string
  x: number
  y: number
  data: Record<string, any>
}

export interface FlowWire {
  id: string
  fromNodeId: string
  toNodeId: string
}

// --- Editor Reactive State ---
const automationId = ref<string>((route.params.id as string) || '')
const automationName = ref('Nuova Automazione a Nodi')
const automationDesc = ref('')
const automationEnabled = ref(true)
const executionMode = ref<'single' | 'restart' | 'queued' | 'parallel'>('single')
const cooldownSeconds = ref(0)

const nodes = ref<FlowNode[]>([])
const wires = ref<FlowWire[]>([])
const selectedNodeId = ref<string | null>(null)
const isJsonModalOpen = ref(false)
const compiledJson = ref('')
const jsonCopied = ref(false)
const isTestingRun = ref(false)
const activeRunStepNodeId = ref<string | null>(null)

// --- Canvas Navigation (Pan & Zoom) ---
const zoom = ref(1)
const panX = ref(60)
const panY = ref(60)
const isPanning = ref(false)
const panStart = ref({ x: 0, y: 0 })

// --- Dragging Nodes ---
const draggingNodeId = ref<string | null>(null)
const dragOffset = ref({ x: 0, y: 0 })

// --- Wiring Interactive State ---
const isWiring = ref(false)
const wireStartNodeId = ref<string | null>(null)
const mousePos = ref({ x: 0, y: 0 })

// --- Available Node Palette Items ---
interface PaletteItem {
  type: FlowNodeType
  subType: string
  title: string
  desc: string
  icon: any
  category: 'Triggers' | 'Conditions' | 'Actions & Flow'
}

const paletteItems: PaletteItem[] = [
  {
    type: 'trigger',
    subType: 'state',
    title: 'Cambio Stato',
    desc: 'Commutazione entità (es. On/Off o sensore)',
    icon: Zap,
    category: 'Triggers'
  },
  {
    type: 'trigger',
    subType: 'numeric_state',
    title: 'Soglia Numerica',
    desc: 'Temperatura, potenza o luminosità > o <',
    icon: Activity,
    category: 'Triggers'
  },
  {
    type: 'trigger',
    subType: 'time',
    title: 'Orario Programmato',
    desc: 'Ora esatta (HH:MM:SS) o alba/tramonto',
    icon: Clock,
    category: 'Triggers'
  },
  {
    type: 'condition',
    subType: 'state',
    title: 'Stato Attuale',
    desc: 'Verifica se un dispositivo è attivo',
    icon: Filter,
    category: 'Conditions'
  },
  {
    type: 'condition',
    subType: 'time',
    title: 'Finestra Oraria',
    desc: 'Solo tra orari specifici (es. Notte)',
    icon: Sun,
    category: 'Conditions'
  },
  {
    type: 'condition',
    subType: 'logic',
    title: 'Logica AND / OR',
    desc: 'Combina condizioni multiple',
    icon: Layers,
    category: 'Conditions'
  },
  {
    type: 'action',
    subType: 'service',
    title: 'Comando Entità',
    desc: 'Accendi, spegni, apri, imposta clima',
    icon: Sliders,
    category: 'Actions & Flow'
  },
  {
    type: 'action',
    subType: 'scene',
    title: 'Attiva Scena',
    desc: 'Applica atmosfera luminosa o profilo',
    icon: Sparkles,
    category: 'Actions & Flow'
  },
  {
    type: 'delay',
    subType: 'delay',
    title: 'Pausa / Delay',
    desc: 'Attesa programmata prima del prossimo step',
    icon: Clock,
    category: 'Actions & Flow'
  },
  {
    type: 'action',
    subType: 'notify',
    title: 'Invia Notifica',
    desc: 'Alert push prioritario su smartphone',
    icon: Send,
    category: 'Actions & Flow'
  }
]

// --- Palette Drag & Drop into Canvas ---
function handleDragStartPalette(e: DragEvent, item: PaletteItem) {
  if (e.dataTransfer) {
    e.dataTransfer.setData('application/json', JSON.stringify(item))
    e.dataTransfer.effectAllowed = 'copy'
  }
}

function handleCanvasDrop(e: DragEvent) {
  e.preventDefault()
  if (!e.dataTransfer) return
  const raw = e.dataTransfer.getData('application/json')
  if (!raw) return
  try {
    const item = JSON.parse(raw) as PaletteItem
    const canvasRect = (e.currentTarget as HTMLElement).getBoundingClientRect()
    const dropX = (e.clientX - canvasRect.left - panX.value) / zoom.value
    const dropY = (e.clientY - canvasRect.top - panY.value) / zoom.value
    createNode(item.type, item.subType, item.title, Math.round(dropX), Math.round(dropY))
  } catch (err) {
    console.warn('Failed to parse dropped node item:', err)
  }
}

// --- Node Factory ---
function createNode(type: FlowNodeType, subType: string, title: string, x?: number, y?: number) {
  soundEffects.playClick()
  const id = `node_${type}_${Date.now().toString().slice(-6)}`
  const posX = x !== undefined ? x : 100 + nodes.value.length * 30
  const posY = y !== undefined ? y : 100 + (nodes.value.length % 5) * 50

  const defaultData: Record<string, any> = {}
  if (subType === 'state') {
    defaultData.entity_key = entitiesStore.allEntities[0]?.id || 'light.salotto'
    defaultData.to = 'on'
  } else if (subType === 'numeric_state') {
    defaultData.entity_key = 'sensor.temperatura_salotto'
    defaultData.above = 24
  } else if (subType === 'time') {
    defaultData.at = '22:30:00'
  } else if (subType === 'service') {
    defaultData.target_entity_key = entitiesStore.allEntities[0]?.id || 'light.salotto'
    defaultData.service = 'turn_on'
  } else if (subType === 'scene') {
    defaultData.scene_id = scenesStore.scenes[0]?.id || ''
  } else if (subType === 'delay') {
    defaultData.delay_seconds = 5
  } else if (subType === 'notify') {
    defaultData.message = 'Avviso da Domus Hub: evento rilevato.'
  }

  const newNode: FlowNode = {
    id,
    type,
    subType,
    title,
    subtitle: getSubtitleForNode(type, subType, defaultData),
    x: posX,
    y: posY,
    data: defaultData
  }

  nodes.value.push(newNode)
  selectedNodeId.value = id
}

function getSubtitleForNode(type: FlowNodeType, subType: string, data: Record<string, any>): string {
  if (type === 'trigger') {
    if (subType === 'state') return `${data.entity_key || 'entità'} -> ${data.to || 'qualsiasi'}`
    if (subType === 'numeric_state') return `${data.entity_key || 'sensore'} > ${data.above || 0}`
    if (subType === 'time') return `Alle ${data.at || '00:00:00'}`
  } else if (type === 'condition') {
    if (subType === 'state') return `Se ${data.entity_key || 'entità'} == ${data.state || 'on'}`
    if (subType === 'time') return `Tra ${data.after || '20:00'} e ${data.before || '06:00'}`
    if (subType === 'logic') return `Logica ${data.logic || 'AND'}`
  } else if (type === 'action') {
    if (subType === 'service') return `${data.service || 'turn_on'} -> ${data.target_entity_key || 'entità'}`
    if (subType === 'scene') return `Applica scena`
    if (subType === 'notify') return `Notifica smartphone`
  } else if (type === 'delay') {
    return `Attesa ${data.delay_seconds || 5}s`
  }
  return 'Configura proprietà'
}

// --- Wire Connections (Curves of Bézier) ---
function startWiring(nodeId: string, e: MouseEvent) {
  e.stopPropagation()
  soundEffects.playClick()
  isWiring.value = true
  wireStartNodeId.value = nodeId
  updateMousePos(e)
}

function endWiring(targetNodeId: string, e: MouseEvent) {
  e.stopPropagation()
  if (!isWiring.value || !wireStartNodeId.value) return

  if (wireStartNodeId.value !== targetNodeId) {
    // Avoid duplicate wires
    const exists = wires.value.some(
      (w) => w.fromNodeId === wireStartNodeId.value && w.toNodeId === targetNodeId
    )
    if (!exists) {
      wires.value.push({
        id: `wire_${wireStartNodeId.value}_${targetNodeId}`,
        fromNodeId: wireStartNodeId.value,
        toNodeId: targetNodeId
      })
      soundEffects.playSuccessChime()
    }
  }

  isWiring.value = false
  wireStartNodeId.value = null
}

function deleteWire(wireId: string) {
  soundEffects.playClick()
  wires.value = wires.value.filter((w) => w.id !== wireId)
}

function deleteNode(nodeId: string) {
  soundEffects.playClick()
  nodes.value = nodes.value.filter((n) => n.id !== nodeId)
  wires.value = wires.value.filter((w) => w.fromNodeId !== nodeId && w.toNodeId !== nodeId)
  if (selectedNodeId.value === nodeId) {
    selectedNodeId.value = null
  }
}

// Compute Wire Bézier Path
function getWirePath(wire: FlowWire): string {
  const from = nodes.value.find((n) => n.id === wire.fromNodeId)
  const to = nodes.value.find((n) => n.id === wire.toNodeId)
  if (!from || !to) return ''

  // Node dimensions: 230px width, ~80px height
  const x1 = from.x + 230
  const y1 = from.y + 45
  const x2 = to.x
  const y2 = to.y + 45

  const dx = Math.abs(x2 - x1) * 0.5
  return `M ${x1} ${y1} C ${x1 + Math.max(dx, 40)} ${y1}, ${x2 - Math.max(dx, 40)} ${y2}, ${x2} ${y2}`
}

function getDraftWirePath(): string {
  if (!isWiring.value || !wireStartNodeId.value) return ''
  const from = nodes.value.find((n) => n.id === wireStartNodeId.value)
  if (!from) return ''

  const x1 = from.x + 230
  const y1 = from.y + 45
  const x2 = (mousePos.value.x - panX.value) / zoom.value
  const y2 = (mousePos.value.y - panY.value) / zoom.value

  const dx = Math.abs(x2 - x1) * 0.5
  return `M ${x1} ${y1} C ${x1 + Math.max(dx, 40)} ${y1}, ${x2 - Math.max(dx, 40)} ${y2}, ${x2} ${y2}`
}

function updateMousePos(e: MouseEvent) {
  const canvasEl = document.querySelector('.flow-canvas')
  if (canvasEl) {
    const rect = canvasEl.getBoundingClientRect()
    mousePos.value = {
      x: e.clientX - rect.left,
      y: e.clientY - rect.top
    }
  }
}

// --- Canvas Dragging & Panning ---
function handleCanvasMouseDown(e: MouseEvent) {
  if ((e.target as HTMLElement).closest('.flow-node') || (e.target as HTMLElement).closest('.node-port')) {
    return
  }
  isPanning.value = true
  panStart.value = {
    x: e.clientX - panX.value,
    y: e.clientY - panY.value
  }
  selectedNodeId.value = null
}

function handleNodeMouseDown(node: FlowNode, e: MouseEvent) {
  if ((e.target as HTMLElement).closest('.node-port')) return
  e.stopPropagation()
  selectedNodeId.value = node.id
  draggingNodeId.value = node.id
  dragOffset.value = {
    x: (e.clientX - panX.value) / zoom.value - node.x,
    y: (e.clientY - panY.value) / zoom.value - node.y
  }
}

function handleMouseMove(e: MouseEvent) {
  updateMousePos(e)

  if (isPanning.value) {
    panX.value = e.clientX - panStart.value.x
    panY.value = e.clientY - panStart.value.y
  } else if (draggingNodeId.value) {
    const node = nodes.value.find((n) => n.id === draggingNodeId.value)
    if (node) {
      node.x = Math.round((e.clientX - panX.value) / zoom.value - dragOffset.value.x)
      node.y = Math.round((e.clientY - panY.value) / zoom.value - dragOffset.value.y)
    }
  }
}

function handleMouseUp() {
  isPanning.value = false
  draggingNodeId.value = null
  if (isWiring.value) {
    isWiring.value = false
    wireStartNodeId.value = null
  }
}

function handleWheel(e: WheelEvent) {
  e.preventDefault()
  const factor = e.deltaY < 0 ? 1.08 : 0.92
  const newZoom = Math.min(Math.max(zoom.value * factor, 0.4), 2.0)
  zoom.value = parseFloat(newZoom.toFixed(2))
}

function resetZoom() {
  zoom.value = 1
  panX.value = 60
  panY.value = 60
}

// --- AST Compiler (Bidirectional: Graph <-> Automation JSON) ---
function compileGraphToAutomation(): Partial<Automation> {
  const triggers: AutomationTrigger[] = []
  const conditions: AutomationCondition[] = []
  const actions: AutomationAction[] = []

  for (const node of nodes.value) {
    if (node.type === 'trigger') {
      const trig: AutomationTrigger = {
        platform: node.subType
      }
      if (node.subType === 'state') {
        trig.entity_key = node.data.entity_key
        trig.to = node.data.to
      } else if (node.subType === 'numeric_state') {
        trig.entity_key = node.data.entity_key
        if (node.data.above !== undefined) trig.above = parseFloat(node.data.above)
        if (node.data.below !== undefined) trig.below = parseFloat(node.data.below)
      } else if (node.subType === 'time') {
        trig.at = node.data.at
      }
      triggers.push(trig)
    } else if (node.type === 'condition') {
      const cond: AutomationCondition = {
        condition: node.subType
      }
      if (node.subType === 'state') {
        cond.entity_key = node.data.entity_key
        cond.state = node.data.state
      } else if (node.subType === 'time') {
        cond.after = node.data.after
        cond.before = node.data.before
      }
      conditions.push(cond)
    } else if (node.type === 'action') {
      const act: AutomationAction = {
        type: node.subType === 'scene' ? 'scene' : 'service'
      }
      if (node.subType === 'service') {
        act.service = node.data.service || 'turn_on'
        act.target_entity_key = node.data.target_entity_key
      } else if (node.subType === 'scene') {
        act.service = 'apply_scene'
        act.params = { scene_id: node.data.scene_id }
      } else if (node.subType === 'notify') {
        act.type = 'event'
        act.event_type = 'notification.create'
        act.event_payload = { message: node.data.message }
      }
      actions.push(act)
    } else if (node.type === 'delay') {
      actions.push({
        type: 'delay',
        delay_ms: (parseInt(node.data.delay_seconds) || 5) * 1000
      })
    }
  }

  return {
    id: automationId.value || undefined,
    name: automationName.value.trim() || 'Automazione senza nome',
    description: automationDesc.value.trim() || undefined,
    enabled: automationEnabled.value,
    execution_mode: executionMode.value,
    cooldown_seconds: cooldownSeconds.value,
    triggers: triggers.length > 0 ? triggers : [{ platform: 'state', entity_key: 'light.salotto', to: 'on' }],
    conditions: conditions.length > 0 ? conditions : undefined,
    actions: actions.length > 0 ? actions : [{ type: 'service', service: 'turn_on', target_entity_key: 'light.salotto' }]
  }
}

function loadAutomationIntoGraph(auto: Automation) {
  automationId.value = auto.id
  automationName.value = auto.name
  automationDesc.value = auto.description || ''
  automationEnabled.value = auto.enabled
  executionMode.value = auto.execution_mode || 'single'
  cooldownSeconds.value = auto.cooldown_seconds || 0

  const newNodes: FlowNode[] = []
  const newWires: FlowWire[] = []

  let col1X = 60
  let col2X = 360
  let col3X = 660

  // 1. Triggers
  let lastTrigId: string | null = null
  auto.triggers.forEach((trig, idx) => {
    const id = `trig_${idx}`
    lastTrigId = id
    newNodes.push({
      id,
      type: 'trigger',
      subType: trig.platform,
      title: trig.platform === 'numeric_state' ? 'Soglia Numerica' : trig.platform === 'time' ? 'Orario' : 'Cambio Stato',
      subtitle: trig.entity_key ? `${trig.entity_key} -> ${trig.to || 'attivo'}` : trig.at || 'Trigger',
      x: col1X,
      y: 80 + idx * 130,
      data: {
        entity_key: trig.entity_key,
        to: trig.to,
        above: trig.above,
        below: trig.below,
        at: trig.at
      }
    })
  })

  // 2. Conditions
  let lastCondId: string | null = null
  if (auto.conditions && auto.conditions.length > 0) {
    auto.conditions.forEach((cond, idx) => {
      const id = `cond_${idx}`
      lastCondId = id
      newNodes.push({
        id,
        type: 'condition',
        subType: cond.condition,
        title: cond.condition === 'time' ? 'Fascia Oraria' : 'Filtro Stato',
        subtitle: cond.entity_key ? `${cond.entity_key} == ${cond.state || 'on'}` : 'Condizione',
        x: col2X,
        y: 80 + idx * 130,
        data: {
          entity_key: cond.entity_key,
          state: cond.state,
          after: cond.after,
          before: cond.before
        }
      })
      if (lastTrigId) {
        newWires.push({
          id: `w_trig_cond_${idx}`,
          fromNodeId: lastTrigId,
          toNodeId: id
        })
      }
    })
  }

  // 3. Actions
  let prevActionId: string | null = lastCondId || lastTrigId
  auto.actions.forEach((act, idx) => {
    const id = `act_${idx}`
    const isDelay = act.type === 'delay'
    newNodes.push({
      id,
      type: isDelay ? 'delay' : 'action',
      subType: isDelay ? 'delay' : (act.type === 'scene' ? 'scene' : 'service'),
      title: isDelay ? 'Pausa / Delay' : (act.type === 'scene' ? 'Applica Scena' : 'Comando'),
      subtitle: isDelay ? `Attesa ${(act.delay_ms || 1000) / 1000}s` : `${act.service || 'azione'} -> ${act.target_entity_key || ''}`,
      x: col3X,
      y: 80 + idx * 130,
      data: {
        target_entity_key: act.target_entity_key,
        service: act.service,
        delay_seconds: (act.delay_ms || 1000) / 1000,
        scene_id: act.params?.scene_id
      }
    })

    if (prevActionId) {
      newWires.push({
        id: `w_seq_${idx}`,
        fromNodeId: prevActionId,
        toNodeId: id
      })
    }
    prevActionId = id
  })

  nodes.value = newNodes
  wires.value = newWires
}

// --- Action Handlers: Save, Test Run, Export ---
async function handleSave() {
  const autoData = compileGraphToAutomation()
  try {
    if (automationId.value) {
      await automationsStore.updateAutomation(automationId.value, autoData)
      notificationsStore.success('Automazione Aggiornata', `"${autoData.name}" salvata con successo.`)
    } else {
      const created = await automationsStore.createAutomation(autoData)
      if (created) automationId.value = created.id
      notificationsStore.success('Automazione Creata', `"${autoData.name}" è ora attiva sull'Hub.`)
    }
    soundEffects.playSuccessChime()
  } catch (err: any) {
    notificationsStore.error('Errore Salvataggio', err?.message || 'Impossibile salvare il flusso.')
  }
}

async function handleTestRun() {
  isTestingRun.value = true
  soundEffects.playClick()
  notificationsStore.info('Test Run Avviato', 'Esecuzione sequenziale dei nodi del flusso...')

  // Step-by-step visual animation through nodes
  for (let i = 0; i < nodes.value.length; i++) {
    activeRunStepNodeId.value = nodes.value[i].id
    soundEffects.playClick()
    await new Promise((r) => setTimeout(r, 600))
  }

  activeRunStepNodeId.value = null
  isTestingRun.value = false
  soundEffects.playSuccessChime()
  notificationsStore.success('Test Run Concluso', 'Tutti i rami del flusso hanno restituito ACK positivo.')
}

function handleOpenJson() {
  soundEffects.playClick()
  const payload = compileGraphToAutomation()
  compiledJson.value = JSON.stringify(payload, null, 2)
  isJsonModalOpen.value = true
}

async function handleCopyJson() {
  try {
    await navigator.clipboard.writeText(compiledJson.value)
    jsonCopied.value = true
    soundEffects.playSuccessChime()
    setTimeout(() => {
      jsonCopied.value = false
    }, 2000)
  } catch {}
}

const selectedNode = computed(() => {
  return nodes.value.find((n) => n.id === selectedNodeId.value) || null
})

// Update subtitle reactively when inspector modifies data
function handleInspectorDataChange() {
  if (selectedNode.value) {
    selectedNode.value.subtitle = getSubtitleForNode(
      selectedNode.value.type,
      selectedNode.value.subType,
      selectedNode.value.data
    )
  }
}

onMounted(async () => {
  if (!automationId.value) {
    // Default initial starter template flow
    createNode('trigger', 'state', 'Rilevato Movimento', 80, 140)
    createNode('condition', 'time', 'Fascia Notturna', 360, 140)
    createNode('action', 'service', 'Accendi Luce Calda', 640, 140)

    wires.value = [
      { id: 'w1', fromNodeId: nodes.value[0].id, toNodeId: nodes.value[1].id },
      { id: 'w2', fromNodeId: nodes.value[1].id, toNodeId: nodes.value[2].id }
    ]
  }

  await Promise.all([
    entitiesStore.fetchEntities().catch(() => {}),
    scenesStore.fetchScenes().catch(() => {}),
    automationsStore.fetchAutomations().catch(() => {})
  ])

  if (automationId.value) {
    const existing = automationsStore.automations.find((a) => a.id === automationId.value)
    if (existing) {
      loadAutomationIntoGraph(existing)
      return
    }
  }
})
</script>

<template>
  <div class="flow-editor-page">
    <!-- Top Header Bar -->
    <header class="editor-header glass-panel">
      <div class="header-left">
        <button
          class="btn-back glass-panel"
          @click="router.push('/automations')"
          title="Torna alle Automazioni"
          aria-label="Torna alle Automazioni"
        >
          <ArrowLeft :size="18" />
        </button>

        <div class="title-group">
          <input
            v-model="automationName"
            type="text"
            class="title-input"
            placeholder="Nome automazione..."
          />
          <div class="badge-status-row">
            <label class="toggle-enabled">
              <input type="checkbox" v-model="automationEnabled" />
              <span class="slider"></span>
              <span class="toggle-label">{{ automationEnabled ? 'Attiva' : 'Disattivata' }}</span>
            </label>
            <span class="badge-mode">Modalità: {{ executionMode }}</span>
          </div>
        </div>
      </div>

      <div class="header-right">
        <button
          class="btn-tool glass-panel"
          @click="handleTestRun"
          :disabled="isTestingRun"
          title="Simula l'esecuzione del flusso"
        >
          <Play :size="16" :class="{ 'animate-pulse': isTestingRun }" />
          <span>{{ isTestingRun ? 'In Esecuzione...' : 'Test Run' }}</span>
        </button>

        <button
          class="btn-tool glass-panel"
          @click="handleOpenJson"
          title="Visualizza o esporta il JSON AST"
        >
          <Code :size="16" />
          <span>AST JSON</span>
        </button>

        <button class="btn-save" @click="handleSave" title="Salva automazione su Domus Hub">
          <Save :size="16" />
          <span>Salva Flusso</span>
        </button>
      </div>
    </header>

    <!-- Main Workspace: Palette Sidebar + Canvas + Inspector -->
    <div class="workspace-layout">
      <!-- Left: Node Library Palette -->
      <aside class="node-palette glass-panel">
        <div class="palette-header">
          <h4 class="palette-title">Libreria Nodi</h4>
          <span class="palette-desc">Trascina sul canvas o clicca (+) per inserire.</span>
        </div>

        <div class="palette-categories">
          <div
            v-for="cat in ['Triggers', 'Conditions', 'Actions & Flow'] as const"
            :key="cat"
            class="palette-group"
          >
            <h5 class="category-title">{{ cat }}</h5>
            <div class="palette-items">
              <div
                v-for="item in paletteItems.filter((i) => i.category === cat)"
                :key="item.subType + item.type"
                class="palette-item glass-card"
                draggable="true"
                @dragstart="handleDragStartPalette($event, item)"
                @click="createNode(item.type, item.subType, item.title)"
              >
                <div :class="['palette-icon', `type-${item.type}`]">
                  <component :is="item.icon" :size="16" />
                </div>
                <div class="palette-text">
                  <span class="item-name">{{ item.title }}</span>
                  <span class="item-desc">{{ item.desc }}</span>
                </div>
                <button
                  class="btn-add-mini"
                  @click.stop="createNode(item.type, item.subType, item.title)"
                  aria-label="Aggiungi nodo"
                >
                  <Plus :size="14" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </aside>

      <!-- Center: Infinite Canvas & SVG Wires -->
      <main
        class="flow-canvas"
        @mousedown="handleCanvasMouseDown"
        @mousemove="handleMouseMove"
        @mouseup="handleMouseUp"
        @wheel="handleWheel"
        @dragover.prevent
        @drop="handleCanvasDrop"
      >
        <!-- Canvas Toolbar Overlay (Zoom & Controls) -->
        <div class="canvas-floating-controls glass-panel">
          <button class="btn-ctl" @click="zoom = Math.min(zoom + 0.15, 2)" title="Ingrandisci">
            <ZoomIn :size="16" />
          </button>
          <span class="zoom-level">{{ Math.round(zoom * 100) }}%</span>
          <button class="btn-ctl" @click="zoom = Math.max(zoom - 0.15, 0.4)" title="Riduci">
            <ZoomOut :size="16" />
          </button>
          <button class="btn-ctl" @click="resetZoom" title="Centra vista (100%)">
            <RotateCcw :size="16" />
          </button>
        </div>

        <!-- Transform Layer (Zoom & Pan) -->
        <div
          class="canvas-transform-layer"
          :style="{
            transform: `translate(${panX}px, ${panY}px) scale(${zoom})`,
            transformOrigin: '0 0'
          }"
        >
          <!-- SVG Connecting Wires (Bézier Curves) -->
          <svg class="wires-layer">
            <!-- Active Existing Wires -->
            <g v-for="wire in wires" :key="wire.id" class="wire-group">
              <path
                :d="getWirePath(wire)"
                class="wire-path-hitbox"
                @click.stop="deleteWire(wire.id)"
                title="Clicca per rimuovere il collegamento"
              />
              <path
                :d="getWirePath(wire)"
                :class="['wire-path', { 'is-active-step': activeRunStepNodeId === wire.toNodeId }]"
              />
            </g>

            <!-- Draft Wire in Progress -->
            <path
              v-if="isWiring"
              :d="getDraftWirePath()"
              class="wire-path-draft"
            />
          </svg>

          <!-- Flow Nodes (HTML Cards) -->
          <div
            v-for="node in nodes"
            :key="node.id"
            :class="[
              'flow-node glass-card',
              `node-type-${node.type}`,
              {
                'is-selected': selectedNodeId === node.id,
                'is-active-step': activeRunStepNodeId === node.id
              }
            ]"
            :style="{
              transform: `translate(${node.x}px, ${node.y}px)`
            }"
            @mousedown="handleNodeMouseDown(node, $event)"
          >
            <!-- Node Input Port (Left) -->
            <div
              v-if="node.type !== 'trigger'"
              class="node-port port-input"
              @mouseup="endWiring(node.id, $event)"
              title="Collega qui il segnale di ingresso"
            >
              <div class="port-dot"></div>
            </div>

            <!-- Node Header -->
            <div class="node-header">
              <div :class="['node-badge-icon', `bg-${node.type}`]">
                <Zap v-if="node.type === 'trigger'" :size="14" />
                <Filter v-else-if="node.type === 'condition'" :size="14" />
                <Clock v-else-if="node.type === 'delay'" :size="14" />
                <Sliders v-else :size="14" />
              </div>
              <div class="node-titles">
                <span class="node-category">{{ node.type.toUpperCase() }}</span>
                <h4 class="node-title">{{ node.title }}</h4>
              </div>
              <button
                class="btn-delete-node"
                @click.stop="deleteNode(node.id)"
                title="Elimina nodo"
                aria-label="Elimina nodo"
              >
                <Trash2 :size="13" />
              </button>
            </div>

            <!-- Node Subtitle / Summary -->
            <div class="node-body">
              <span class="node-subtitle">{{ node.subtitle }}</span>
            </div>

            <!-- Node Output Port (Right) -->
            <div
              class="node-port port-output"
              @mousedown="startWiring(node.id, $event)"
              title="Trascina verso un altro nodo per collegare"
            >
              <div class="port-dot"></div>
            </div>
          </div>
        </div>
      </main>

      <!-- Right: Node Properties Inspector Drawer -->
      <aside v-if="selectedNode" class="node-inspector glass-panel">
        <div class="inspector-header">
          <div class="inspector-titles">
            <span class="inspector-pill">{{ selectedNode.type }}</span>
            <h4 class="inspector-title">{{ selectedNode.title }}</h4>
          </div>
          <button class="btn-close-inspector" @click="selectedNodeId = null">
            <X :size="16" />
          </button>
        </div>

        <div class="inspector-content">
          <div class="prop-group">
            <label class="prop-label">Titolo Visuale</label>
            <input
              v-model="selectedNode.title"
              type="text"
              class="prop-input"
              @input="handleInspectorDataChange"
            />
          </div>

          <!-- Dynamic Form based on SubType -->
          <div v-if="selectedNode.subType === 'state'" class="prop-group">
            <label class="prop-label">Entità Target</label>
            <select
              v-model="selectedNode.data.entity_key"
              class="prop-select"
              @change="handleInspectorDataChange"
            >
              <option
                v-for="ent in entitiesStore.allEntities"
                :key="ent.id"
                :value="ent.id"
              >
                {{ ent.name }} ({{ ent.id }})
              </option>
            </select>

            <label class="prop-label mt-12">Stato Previsto (To / State)</label>
            <input
              v-model="selectedNode.data.to"
              type="text"
              class="prop-input"
              placeholder="es. on, off, open, closed"
              @input="handleInspectorDataChange"
            />
          </div>

          <div v-else-if="selectedNode.subType === 'numeric_state'" class="prop-group">
            <label class="prop-label">Sensore Numerico</label>
            <select
              v-model="selectedNode.data.entity_key"
              class="prop-select"
              @change="handleInspectorDataChange"
            >
              <option
                v-for="ent in entitiesStore.allEntities"
                :key="ent.id"
                :value="ent.id"
              >
                {{ ent.name }} ({{ ent.id }})
              </option>
            </select>

            <div class="prop-row mt-12">
              <div class="prop-col">
                <label class="prop-label">Soglia Superiore (&gt;)</label>
                <input
                  v-model.number="selectedNode.data.above"
                  type="number"
                  class="prop-input"
                  @input="handleInspectorDataChange"
                />
              </div>
              <div class="prop-col">
                <label class="prop-label">Soglia Inferiore (&lt;)</label>
                <input
                  v-model.number="selectedNode.data.below"
                  type="number"
                  class="prop-input"
                  @input="handleInspectorDataChange"
                />
              </div>
            </div>
          </div>

          <div v-else-if="selectedNode.subType === 'time'" class="prop-group">
            <label class="prop-label">Orario Esatto (HH:MM:SS)</label>
            <input
              v-model="selectedNode.data.at"
              type="time"
              step="1"
              class="prop-input"
              @input="handleInspectorDataChange"
            />
          </div>

          <div v-else-if="selectedNode.subType === 'service'" class="prop-group">
            <label class="prop-label">Entità da Controllare</label>
            <select
              v-model="selectedNode.data.target_entity_key"
              class="prop-select"
              @change="handleInspectorDataChange"
            >
              <option
                v-for="ent in entitiesStore.allEntities"
                :key="ent.id"
                :value="ent.id"
              >
                {{ ent.name }} ({{ ent.id }})
              </option>
            </select>

            <label class="prop-label mt-12">Servizio / Azione</label>
            <select
              v-model="selectedNode.data.service"
              class="prop-select"
              @change="handleInspectorDataChange"
            >
              <option value="turn_on">Accendi (turn_on)</option>
              <option value="turn_off">Spegni (turn_off)</option>
              <option value="toggle">Commuta (toggle)</option>
              <option value="close_cover">Chiudi Tapparella</option>
              <option value="open_cover">Apri Tapparella</option>
              <option value="set_temperature">Imposta Clima</option>
            </select>
          </div>

          <div v-else-if="selectedNode.subType === 'delay'" class="prop-group">
            <label class="prop-label">Secondi di Attesa (Pausa)</label>
            <input
              v-model.number="selectedNode.data.delay_seconds"
              type="number"
              min="1"
              max="3600"
              class="prop-input"
              @input="handleInspectorDataChange"
            />
          </div>

          <div v-else-if="selectedNode.subType === 'notify'" class="prop-group">
            <label class="prop-label">Messaggio Notifica</label>
            <textarea
              v-model="selectedNode.data.message"
              class="prop-textarea"
              rows="3"
              @input="handleInspectorDataChange"
            ></textarea>
          </div>

          <div class="inspector-footer">
            <button class="btn-remove-selected" @click="deleteNode(selectedNode.id)">
              <Trash2 :size="15" />
              <span>Elimina Nodo</span>
            </button>
          </div>
        </div>
      </aside>
    </div>

    <!-- Modal: View AST JSON -->
    <div v-if="isJsonModalOpen" class="modal-backdrop" @click="isJsonModalOpen = false">
      <div class="modal-container glass-card" @click.stop>
        <div class="modal-header">
          <div class="modal-title-row">
            <Code :size="20" class="text-primary" />
            <h3>AST JSON Compilato</h3>
          </div>
          <button class="btn-modal-close" @click="isJsonModalOpen = false">
            <X :size="18" />
          </button>
        </div>

        <div class="modal-body">
          <pre class="json-code-box"><code>{{ compiledJson }}</code></pre>
        </div>

        <div class="modal-actions">
          <button class="btn-copy-json glass-panel" @click="handleCopyJson">
            <Check v-if="jsonCopied" :size="16" class="text-success" />
            <Copy v-else :size="16" />
            <span>{{ jsonCopied ? 'Copiato!' : 'Copia negli Appunti' }}</span>
          </button>
          <button class="btn-done" @click="isJsonModalOpen = false">Chiudi</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.flow-editor-page {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 80px);
  overflow: hidden;
  position: relative;
}

/* ================= HEADER ================= */
.editor-header {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  border-bottom: 1px solid var(--border-subtle);
  background: rgba(15, 23, 42, 0.75);
  backdrop-filter: blur(16px);
  z-index: 20;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.btn-back {
  width: 38px;
  height: 38px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-subtle);
  background: var(--bg-surface);
  color: var(--text-main);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-back:hover {
  background: var(--bg-surface-hover);
}

.title-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.title-input {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-main);
  background: transparent;
  border: none;
  border-bottom: 1px dashed transparent;
  outline: none;
  padding: 2px 0;
  transition: border var(--transition-fast);
}

.title-input:focus {
  border-bottom-color: var(--primary);
}

.badge-status-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.toggle-enabled {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}

.toggle-enabled input {
  display: none;
}

.slider {
  width: 28px;
  height: 16px;
  border-radius: var(--radius-full);
  background: rgba(255, 255, 255, 0.15);
  position: relative;
  transition: background var(--transition-fast);
}

.slider::after {
  content: '';
  position: absolute;
  top: 2px;
  left: 2px;
  width: 12px;
  height: 12px;
  border-radius: var(--radius-full);
  background: white;
  transition: transform var(--transition-fast);
}

.toggle-enabled input:checked + .slider {
  background: #10b981;
}

.toggle-enabled input:checked + .slider::after {
  transform: translateX(12px);
}

.toggle-label {
  font-size: 11px;
  color: var(--text-muted);
}

.badge-mode {
  font-size: 11px;
  color: var(--text-muted);
  font-family: var(--font-mono);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.btn-tool {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 8px 14px;
  border-radius: var(--radius-md);
  font-size: 13px;
  font-weight: 500;
  color: var(--text-main);
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-tool:hover:not(:disabled) {
  background: var(--bg-surface-hover);
}

.btn-save {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 8px 18px;
  border-radius: var(--radius-md);
  font-size: 13px;
  font-weight: 600;
  color: white;
  background: var(--primary);
  border: none;
  cursor: pointer;
  box-shadow: 0 2px 10px rgba(14, 165, 233, 0.35);
  transition: all var(--transition-fast);
}

.btn-save:hover {
  opacity: 0.95;
  transform: translateY(-1px);
}

/* ================= WORKSPACE LAYOUT ================= */
.workspace-layout {
  display: flex;
  flex: 1;
  position: relative;
  overflow: hidden;
}

/* ================= PALETTE SIDEBAR ================= */
.node-palette {
  width: 280px;
  border-right: 1px solid var(--border-subtle);
  background: rgba(15, 23, 42, 0.6);
  backdrop-filter: blur(12px);
  display: flex;
  flex-direction: column;
  z-index: 10;
  overflow-y: auto;
}

.palette-header {
  padding: 18px 16px 12px;
  border-bottom: 1px solid var(--border-subtle);
}

.palette-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-main);
}

.palette-desc {
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 2px;
  display: block;
}

.palette-categories {
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.category-title {
  font-size: 11px;
  font-weight: 700;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 8px;
}

.palette-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.palette-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  cursor: grab;
  transition: all var(--transition-fast);
  user-select: none;
}

.palette-item:hover {
  border-color: var(--primary);
  background: var(--bg-surface-hover);
  transform: translateX(2px);
}

.palette-icon {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.palette-icon.type-trigger { background: rgba(168, 85, 247, 0.15); color: #a855f7; }
.palette-icon.type-condition { background: rgba(14, 165, 233, 0.15); color: #0ea5e9; }
.palette-icon.type-action { background: rgba(16, 185, 129, 0.15); color: #10b981; }
.palette-icon.type-delay { background: rgba(245, 158, 11, 0.15); color: #f59e0b; }

.palette-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.item-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-main);
}

.item-desc {
  font-size: 11px;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.btn-add-mini {
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-add-mini:hover {
  background: var(--primary);
  color: white;
  border-color: var(--primary);
}

/* ================= INFINITE CANVAS ================= */
.flow-canvas {
  flex: 1;
  position: relative;
  overflow: hidden;
  background-color: #0b0f19;
  background-image: radial-gradient(rgba(255, 255, 255, 0.08) 1px, transparent 1px);
  background-size: 24px 24px;
  cursor: grab;
}

.flow-canvas:active {
  cursor: grabbing;
}

.canvas-floating-controls {
  position: absolute;
  bottom: 24px;
  left: 24px;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px;
  border-radius: var(--radius-lg);
  background: rgba(15, 23, 42, 0.85);
  border: 1px solid var(--border-subtle);
  backdrop-filter: blur(12px);
  z-index: 15;
}

.btn-ctl {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  background: transparent;
  border: none;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-ctl:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-main);
}

.zoom-level {
  font-size: 12px;
  font-family: var(--font-mono);
  color: var(--text-muted);
  padding: 0 6px;
}

.canvas-transform-layer {
  position: absolute;
  top: 0;
  left: 0;
  width: 10000px;
  height: 10000px;
}

/* ================= SVG WIRES ================= */
.wires-layer {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 2;
}

.wire-group {
  pointer-events: all;
}

.wire-path-hitbox {
  fill: none;
  stroke: transparent;
  stroke-width: 18px;
  cursor: pointer;
}

.wire-path {
  fill: none;
  stroke: rgba(14, 165, 233, 0.45);
  stroke-width: 3px;
  stroke-linecap: round;
  transition: stroke 0.2s, stroke-width 0.2s;
}

.wire-group:hover .wire-path {
  stroke: #ef4444;
  stroke-width: 4px;
}

.wire-path.is-active-step {
  stroke: #10b981;
  stroke-width: 5px;
  filter: drop-shadow(0 0 8px rgba(16, 185, 129, 0.8));
}

.wire-path-draft {
  fill: none;
  stroke: #0ea5e9;
  stroke-width: 3px;
  stroke-dasharray: 6 6;
  animation: dash 1s linear infinite;
}

@keyframes dash {
  to { stroke-dashoffset: -12; }
}

/* ================= FLOW NODES ================= */
.flow-node {
  position: absolute;
  width: 230px;
  padding: 14px;
  border-radius: var(--radius-lg);
  background: rgba(30, 41, 59, 0.85);
  border: 1px solid var(--border-subtle);
  backdrop-filter: blur(14px);
  cursor: move;
  user-select: none;
  z-index: 5;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
}

.flow-node:hover {
  border-color: rgba(255, 255, 255, 0.25);
}

.flow-node.is-selected {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px rgba(14, 165, 233, 0.4), 0 8px 24px rgba(0, 0, 0, 0.5);
}

.flow-node.is-active-step {
  border-color: #10b981;
  box-shadow: 0 0 20px rgba(16, 185, 129, 0.6);
}

.node-header {
  display: flex;
  align-items: center;
  gap: 10px;
}

.node-badge-icon {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.bg-trigger { background: rgba(168, 85, 247, 0.2); color: #a855f7; }
.bg-condition { background: rgba(14, 165, 233, 0.2); color: #0ea5e9; }
.bg-action { background: rgba(16, 185, 129, 0.2); color: #10b981; }
.bg-delay { background: rgba(245, 158, 11, 0.2); color: #f59e0b; }

.node-titles {
  flex: 1;
  min-width: 0;
}

.node-category {
  font-size: 10px;
  font-weight: 700;
  color: var(--text-dim);
  letter-spacing: 0.05em;
  display: block;
}

.node-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.btn-delete-node {
  width: 22px;
  height: 22px;
  border-radius: var(--radius-sm);
  background: transparent;
  border: none;
  color: var(--text-dim);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: color var(--transition-fast);
}

.btn-delete-node:hover {
  color: #ef4444;
}

.node-body {
  margin-top: 8px;
}

.node-subtitle {
  font-size: 11px;
  color: var(--text-muted);
  display: block;
  font-family: var(--font-mono);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Connection Ports */
.node-port {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: crosshair;
  z-index: 8;
}

.port-input {
  left: -8px;
}

.port-output {
  right: -8px;
}

.port-dot {
  width: 10px;
  height: 10px;
  border-radius: var(--radius-full);
  background: var(--primary);
  border: 2px solid #0f172a;
  transition: transform var(--transition-fast);
}

.node-port:hover .port-dot {
  transform: scale(1.4);
  background: #fff;
}

/* ================= INSPECTOR DRAWER ================= */
.node-inspector {
  width: 320px;
  border-left: 1px solid var(--border-subtle);
  background: rgba(15, 23, 42, 0.7);
  backdrop-filter: blur(14px);
  display: flex;
  flex-direction: column;
  z-index: 10;
  overflow-y: auto;
}

.inspector-header {
  padding: 18px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-subtle);
}

.inspector-pill {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--primary);
  letter-spacing: 0.05em;
}

.inspector-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-main);
  margin-top: 2px;
}

.btn-close-inspector {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-md);
  background: transparent;
  border: none;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.btn-close-inspector:hover {
  color: var(--text-main);
}

.inspector-content {
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.prop-group {
  display: flex;
  flex-direction: column;
}

.prop-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  margin-bottom: 6px;
}

.mt-12 {
  margin-top: 12px;
}

.prop-input,
.prop-select,
.prop-textarea {
  width: 100%;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  color: var(--text-main);
  font-size: 13px;
  outline: none;
  transition: border var(--transition-fast);
}

.prop-input:focus,
.prop-select:focus,
.prop-textarea:focus {
  border-color: var(--primary);
}

.prop-row {
  display: flex;
  gap: 10px;
}

.prop-col {
  flex: 1;
}

.inspector-footer {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--border-subtle);
}

.btn-remove-selected {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px;
  border-radius: var(--radius-md);
  background: rgba(239, 68, 68, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.25);
  color: #ef4444;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-remove-selected:hover {
  background: rgba(239, 68, 68, 0.22);
}

/* ================= MODAL JSON ================= */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal-container {
  width: 90%;
  max-width: 640px;
  background: #0f172a;
  border-radius: var(--radius-xl);
  border: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.6);
}

.modal-header {
  padding: 18px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.modal-title-row h3 {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-main);
}

.btn-modal-close {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
}

.modal-body {
  padding: 20px;
  max-height: 480px;
  overflow-y: auto;
}

.json-code-box {
  background: #020617;
  padding: 16px;
  border-radius: var(--radius-md);
  font-family: var(--font-mono);
  font-size: 12px;
  color: #38bdf8;
  overflow-x: auto;
}

.modal-actions {
  padding: 16px 24px;
  border-top: 1px solid var(--border-subtle);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
}

.btn-copy-json {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  color: var(--text-main);
  font-size: 13px;
  cursor: pointer;
}

.btn-done {
  padding: 8px 20px;
  border-radius: var(--radius-md);
  background: var(--primary);
  color: white;
  border: none;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
}
</style>
