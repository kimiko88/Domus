<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { X, Play, Plus, Zap, Filter, Clock, ArrowRight, Check, Trash2, Layers } from '@lucide/vue'

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

import { audioSynthesizer } from '../../utils/audioSynthesizer'

// Dry-Run Simulator State
const isDryRunOpen = ref(false)
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
  isDryRunOpen.value = !isDryRunOpen.value
  if (isDryRunOpen.value) {
    executeDryRun()
  }
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

  // 2. Evaluate Condition
  const condNode = nodes.value.find((n) => n.type === 'condition')
  if (condNode) {
    if (dryRunInputs.value.lux < 30) {
      nodeEvals.value[condNode.id] = { status: 'passed', detail: `Condizione soddisfatta: ${dryRunInputs.value.lux} lx < 30 lx` }
      dryRunLogs.value.push(`[2/3] Condizione [${condNode.title}]: PASSATA (${dryRunInputs.value.lux} lx < 30 lx)`)
    } else {
      nodeEvals.value[condNode.id] = { status: 'failed', detail: `Condizione fallita: ${dryRunInputs.value.lux} lx >= 30 lx` }
      dryRunLogs.value.push(`[2/3] Condizione [${condNode.title}]: BLOCCATA (${dryRunInputs.value.lux} lx >= 30 lx)`)
      return
    }
  }

  // 3. Actions
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
  if (type === 'trigger') {
    title = 'Trigger Orario'
    subtitle = 'Alle 22:30 ogni giorno'
  } else if (type === 'condition') {
    title = 'Presenza Utente'
    subtitle = 'Qualcuno è a casa'
  } else if (type === 'delay') {
    title = 'Ritardo 5s'
    subtitle = 'Attesa timer'
  } else {
    title = 'Notifica Smartphone'
    subtitle = 'Invia allerta push'
  }

  const newNode: FlowNode = { id, type, title, subtitle, x: newX, y: newY }
  nodes.value.push(newNode)

  if (lastNode) {
    connections.value.push({ from: lastNode.id, to: id })
  }
}

function removeNode(id: string) {
  nodes.value = nodes.value.filter((n) => n.id !== id)
  connections.value = connections.value.filter((c) => c.from !== id && c.to !== id)
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
          <button class="btn-dryrun" :class="{ active: isDryRunOpen }" @click="toggleDryRun">
            <span>🧪 Collaudo Dry-Run</span>
          </button>
          <button class="btn-simulate" :disabled="isSimulating" @click="simulateFlow">
            <Play :size="15" :class="{ 'animate-spin': isSimulating }" />
            <span>{{ isSimulating ? t('flowchart.btnPause') : t('flowchart.btnSimulate') }}</span>
          </button>
          <button class="btn-close" @click="emit('close')">
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

      <!-- Node Graph Canvas -->
      <div class="canvas-viewport">
        <div class="canvas-grid">
          <!-- Connection Lines SVG -->
          <svg class="connections-svg">
            <g v-for="(conn, ci) in connections" :key="ci">
              <path
                v-if="getNodeById(conn.from) && getNodeById(conn.to)"
                :d="`M ${getNodeById(conn.from)!.x + 180} ${getNodeById(conn.from)!.y + 40} C ${getNodeById(conn.from)!.x + 230} ${getNodeById(conn.from)!.y + 40}, ${getNodeById(conn.to)!.x - 50} ${getNodeById(conn.to)!.y + 40}, ${getNodeById(conn.to)!.x} ${getNodeById(conn.to)!.y + 40}`"
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
                'eval-failed': nodeEvals[n.id]?.status === 'failed'
              }
            ]"
            :style="{ left: `${n.x}px`, top: `${n.y}px` }"
          >
            <div class="node-header">
              <span class="node-badge">{{ n.type.toUpperCase() }}</span>
              <span v-if="nodeEvals[n.id]" class="eval-pill" :class="nodeEvals[n.id].status">
                {{ nodeEvals[n.id].status === 'passed' ? 'PASSATO' : 'BLOCCATO' }}
              </span>
              <button v-if="nodes.length > 1" class="btn-node-del" @click="removeNode(n.id)">
                <Trash2 :size="12" />
              </button>
            </div>
            <h4 class="node-title">{{ n.title }}</h4>
            <p class="node-sub">{{ n.subtitle }}</p>
            <p v-if="nodeEvals[n.id]" class="node-eval-msg" :class="nodeEvals[n.id].status">
              {{ nodeEvals[n.id].detail }}
            </p>
          </div>
        </div>

        <!-- Dry-Run Inspector Panel -->
        <div v-if="isDryRunOpen" class="dryrun-panel">
          <div class="dryrun-header">
            <h4>🧪 Simulatore di Collaudo</h4>
            <span class="dryrun-sub">Inietta valori fittizi per testare le condizioni prima del salvataggio</span>
          </div>

          <div class="dryrun-form">
            <label class="dryrun-field">
              <span>Movimento (Trigger)</span>
              <input type="checkbox" v-model="dryRunInputs.motion" @change="executeDryRun" />
            </label>

            <label class="dryrun-field">
              <span>Luminosità Sensore: {{ dryRunInputs.lux }} lx</span>
              <input type="range" min="0" max="100" v-model.number="dryRunInputs.lux" @input="executeDryRun" />
            </label>

            <label class="dryrun-field">
              <span>Orario Simulato</span>
              <input type="time" v-model="dryRunInputs.time" @change="executeDryRun" />
            </label>
          </div>

          <div class="dryrun-trace">
            <h5>Trace Esecuzione:</h5>
            <div v-for="(log, li) in dryRunLogs" :key="li" class="trace-log-line">
              {{ log }}
            </div>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="modal-footer">
        <span class="footer-tip">Trascina o aggiungi blocchi; la sequenza viene convertita automaticamente in regole native Domus.</span>
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
  padding: 20px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-subtle);
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
}

.modal-sub {
  font-size: 13px;
  color: var(--text-muted);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
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
}

.btn-close:hover {
  color: var(--text-main);
  background: rgba(255, 255, 255, 0.05);
}

.flow-toolbar {
  padding: 12px 24px;
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
}

.type-trig { color: #10b981; border-color: rgba(16, 185, 129, 0.3); }
.type-cond { color: #f59e0b; border-color: rgba(245, 158, 11, 0.3); }
.type-del  { color: #0ea5e9; border-color: rgba(14, 165, 233, 0.3); }
.type-act  { color: #a855f7; border-color: rgba(168, 85, 247, 0.3); }

.canvas-viewport {
  flex: 1;
  position: relative;
  overflow: auto;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.04) 1px, transparent 1px);
  background-size: 24px 24px;
}

.canvas-grid {
  position: relative;
  min-width: 1200px;
  min-height: 500px;
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
  stroke: rgba(255, 255, 255, 0.2);
  stroke-width: 3;
  stroke-dasharray: 6 6;
  transition: all 0.3s ease;
}

.wire-path.active {
  stroke: #10b981;
  stroke-width: 5;
  stroke-dasharray: none;
  filter: drop-shadow(0 0 8px #10b981);
}

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
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.flow-node.sim-active {
  transform: scale(1.08);
  box-shadow: 0 0 30px rgba(16, 185, 129, 0.6);
  border-color: #10b981;
}

.node-trigger { border-left: 4px solid #10b981; }
.node-condition { border-left: 4px solid #f59e0b; }
.node-delay { border-left: 4px solid #0ea5e9; }
.node-action { border-left: 4px solid #a855f7; }

.node-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.node-badge {
  font-size: 10px;
  font-weight: 700;
  color: var(--text-muted);
  letter-spacing: 0.05em;
}

.btn-node-del {
  background: transparent;
  border: none;
  color: var(--text-dim);
  cursor: pointer;
  padding: 2px;
}

.btn-node-del:hover {
  color: var(--color-danger);
}

.node-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-main);
}

.node-sub {
  font-size: 11px;
  color: var(--text-muted);
  line-height: 1.4;
  word-break: break-all;
}

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
}

.btn-dryrun {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-main);
  border: 1px solid var(--border-subtle);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-dryrun:hover {
  background: rgba(255, 255, 255, 0.15);
}

.btn-dryrun.active {
  background: rgba(14, 165, 233, 0.2);
  color: var(--primary);
  border-color: var(--primary);
  box-shadow: 0 0 14px rgba(14, 165, 233, 0.35);
}

.flow-node.eval-passed {
  border-color: #10b981 !important;
  box-shadow: 0 0 24px rgba(16, 185, 129, 0.45) !important;
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.15) 0%, var(--bg-glass-card) 100%) !important;
}

.flow-node.eval-failed {
  border-color: #ef4444 !important;
  box-shadow: 0 0 24px rgba(239, 68, 68, 0.45) !important;
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.15) 0%, var(--bg-glass-card) 100%) !important;
}

.node-sim-status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  font-weight: 700;
  padding: 3px 6px;
  border-radius: var(--radius-xs, 4px);
  margin-top: 4px;
  width: fit-content;
}

.node-sim-status.passed {
  background: rgba(16, 185, 129, 0.2);
  color: #10b981;
  border: 1px solid rgba(16, 185, 129, 0.4);
}

.node-sim-status.failed {
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.4);
}

/* Dryrun Side Panel */
.dryrun-panel {
  width: 320px;
  background: var(--bg-surface);
  border-left: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  z-index: 10;
  box-shadow: -4px 0 20px rgba(0, 0, 0, 0.25);
}

.dryrun-panel-header {
  padding: 16px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-subtle);
  background: rgba(0, 0, 0, 0.15);
}

.dryrun-panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 700;
  color: var(--text-main);
}

.btn-close-dryrun {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 4px;
  border-radius: var(--radius-sm);
}

.btn-close-dryrun:hover {
  color: var(--text-main);
}

.dryrun-body {
  padding: 16px;
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.dryrun-section-title {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  margin-bottom: 8px;
}

.mock-inputs-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.mock-input-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.mock-input-item label {
  font-size: 12px;
  color: var(--text-main);
  font-weight: 500;
}

.mock-input-item input[type="text"],
.mock-input-item input[type="number"],
.mock-input-item input[type="time"] {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-main);
  padding: 6px 10px;
  font-size: 13px;
  outline: none;
}

.mock-input-item input:focus {
  border-color: var(--primary);
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
  accent-color: var(--primary);
  width: 16px;
  height: 16px;
}

.eval-summary-card {
  padding: 12px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  gap: 12px;
  border: 1px solid var(--border-subtle);
}

.eval-summary-card.passed {
  background: rgba(16, 185, 129, 0.12);
  border-color: rgba(16, 185, 129, 0.35);
}

.eval-summary-card.failed {
  background: rgba(239, 68, 68, 0.12);
  border-color: rgba(239, 68, 68, 0.35);
}

.eval-badge {
  padding: 4px 8px;
  border-radius: var(--radius-xs, 4px);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.05em;
}

.eval-summary-card.passed .eval-badge {
  background: #10b981;
  color: #fff;
}

.eval-summary-card.failed .eval-badge {
  background: #ef4444;
  color: #fff;
}

.eval-summary-text {
  font-size: 12px;
  color: var(--text-main);
  line-height: 1.4;
}

.dryrun-actions {
  display: flex;
  gap: 8px;
}

.btn-eval-now {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 12px;
  background: var(--primary);
  color: #fff;
  border: none;
  border-radius: var(--radius-md);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.btn-reset-mock {
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  border-radius: var(--radius-md);
  font-size: 12px;
  cursor: pointer;
}

.btn-reset-mock:hover {
  color: var(--text-main);
}

.dryrun-log-box {
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 10px;
  max-height: 160px;
  overflow-y: auto;
  font-family: monospace;
  font-size: 11px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.log-entry {
  color: var(--text-muted);
  word-break: break-all;
}

.log-entry.log-success {
  color: #10b981;
}

.log-entry.log-error {
  color: #ef4444;
}
</style>
