<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Network,
  Radio,
  Wifi,
  Activity,
  Battery,
  ShieldCheck,
  Info
} from '@lucide/vue'
import { apiClient } from '../api/client'

const { t } = useI18n()

interface MeshNode {
  id: string
  name: string
  protocol: string
  type: 'coordinator' | 'router' | 'end_device'
  lqi: number
  battery?: number
  parent_id?: string
}

interface MeshLink {
  source: string
  target: string
  lqi: number
  quality: 'excellent' | 'good' | 'weak' | 'critical'
}

interface MeshTopology {
  protocol: string
  nodes: MeshNode[]
  links: MeshLink[]
}

const currentProtocol = ref<'zigbee' | 'zwave' | 'thread'>('zigbee')

const topology = ref<MeshTopology>({
  protocol: 'zigbee',
  nodes: [],
  links: []
})

const selectedNode = ref<MeshNode | null>(null)

// Compute dynamic (x, y) coordinates for SVG rendering based on node hierarchy
const nodePositions = computed(() => {
  const map = new Map<string, { x: number; y: number }>()
  const nodes = topology.value.nodes || []
  if (nodes.length === 0) return map

  const coordinators = nodes.filter(n => n.type === 'coordinator')
  const routers = nodes.filter(n => n.type === 'router')
  const endDevices = nodes.filter(n => n.type !== 'coordinator' && n.type !== 'router')

  // Top level: Coordinator(s)
  coordinators.forEach((n, idx) => {
    const total = coordinators.length
    const x = 400 + (idx - (total - 1) / 2) * 140
    map.set(n.id, { x, y: 80 })
  })

  // Middle level: Routers
  routers.forEach((n, idx) => {
    const total = routers.length
    const step = 640 / (total + 1)
    const x = 80 + (idx + 1) * step
    map.set(n.id, { x, y: 220 })
  })

  // Bottom level: End devices
  endDevices.forEach((n, idx) => {
    const total = endDevices.length
    const step = 700 / (total + 1)
    const x = 50 + (idx + 1) * step
    map.set(n.id, { x, y: 360 })
  })

  // Any other nodes
  nodes.forEach((n, idx) => {
    if (!map.has(n.id)) {
      map.set(n.id, { x: 100 + ((idx * 100) % 600), y: 220 })
    }
  })

  return map
})

// Zigbee Direct Mesh Binding State
const bindings = ref<any[]>([])
const isBindingModalOpen = ref(false)
const sourceDeviceID = ref('')
const sourceDeviceName = ref('')
const targetDeviceID = ref('')
const targetDeviceName = ref('')
const selectedClusters = ref<string[]>(['genOnOff'])

async function loadBindings() {
  try {
    const list = await apiClient.get<any[]>('/api/v1/devices/bindings')
    bindings.value = list || []
  } catch {
    bindings.value = []
  }
}

async function handleCreateBinding() {
  if (!sourceDeviceID.value.trim() || !targetDeviceID.value.trim()) return
  try {
    const b = await apiClient.post<any>('/api/v1/devices/bindings', {
      source_device_id: sourceDeviceID.value.trim(),
      source_device_name: sourceDeviceName.value.trim() || sourceDeviceID.value.trim(),
      target_device_id: targetDeviceID.value.trim(),
      target_device_name: targetDeviceName.value.trim() || targetDeviceID.value.trim(),
      clusters: selectedClusters.value
    })
    if (b) bindings.value.push(b)
    isBindingModalOpen.value = false
    sourceDeviceID.value = ''
    sourceDeviceName.value = ''
    targetDeviceID.value = ''
    targetDeviceName.value = ''
  } catch {}
}

async function handleDeleteBinding(id: string) {
  try {
    await apiClient.delete(`/api/v1/devices/bindings/${id}`)
    bindings.value = bindings.value.filter(b => b.id !== id)
  } catch {}
}

async function loadTopology(proto: 'zigbee' | 'zwave' | 'thread') {
  currentProtocol.value = proto
  selectedNode.value = null
  try {
    const data = await apiClient.get<MeshTopology>(`/api/v1/mesh/topology?protocol=${proto}`)
    topology.value = data || { protocol: proto, nodes: [], links: [] }
  } catch {
    topology.value = { protocol: proto, nodes: [], links: [] }
  }
}

onMounted(() => {
  loadTopology('zigbee')
  loadBindings()
})
</script>

<template>
  <div class="mesh-view-container">
    <header class="mesh-header glass-panel">
      <div class="header-title-box">
        <div class="icon-circle">
          <Network :size="24" class="text-primary" />
        </div>
        <div>
          <h1 class="page-title">{{ t('mesh.title') }}</h1>
          <p class="page-subtitle">{{ t('mesh.subtitle') }}</p>
        </div>
      </div>

      <div class="protocol-pills">
        <button
          :class="['proto-btn', { active: currentProtocol === 'zigbee' }]"
          @click="loadTopology('zigbee')"
        >
          Zigbee 3.0
        </button>
        <button
          :class="['proto-btn', { active: currentProtocol === 'zwave' }]"
          @click="loadTopology('zwave')"
        >
          Z-Wave JS
        </button>
        <button
          :class="['proto-btn', { active: currentProtocol === 'thread' }]"
          @click="loadTopology('thread')"
        >
          Matter / Thread
        </button>
      </div>
    </header>

    <!-- Visual Graph Card -->
    <div class="mesh-canvas-card glass-panel">
      <div class="canvas-legend-bar">
        <div class="legend-item"><span class="dot coordinator" /> Coordinator (Gateway)</div>
        <div class="legend-item"><span class="dot router" /> Router / Ripetitore (Alimentato)</div>
        <div class="legend-item"><span class="dot end-device" /> End-Device (A batteria)</div>
        <div class="legend-separator" />
        <div class="legend-item"><span class="link-line excellent" /> LQI &gt; 180 (Ottimo)</div>
        <div class="legend-item"><span class="link-line good" /> LQI 100-180 (Buono)</div>
        <div class="legend-item"><span class="link-line weak" /> LQI &lt; 100 (Debole)</div>
      </div>

      <!-- Empty State when no nodes exist -->
      <div v-if="topology.nodes.length === 0" class="svg-graph-container" style="display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 380px; gap: 12px;">
        <Network :size="48" style="opacity: 0.4; color: var(--text-muted);" />
        <p style="font-weight: 600; font-size: 1.05rem; margin: 0;">Nessun dispositivo {{ currentProtocol.toUpperCase() }} rilevato nella rete mesh</p>
        <p style="font-size: 0.875rem; color: var(--text-muted); margin: 0;">I dispositivi configurati per questo protocollo verranno visualizzati qui con le loro metriche LQI reali.</p>
      </div>

      <!-- Graph Diagram SVG -->
      <div v-else class="svg-graph-container">
        <svg viewBox="0 0 800 450" class="mesh-svg">
          <!-- Dynamic Links -->
          <g v-for="(link, i) in topology.links" :key="'link-' + i">
            <line
              v-if="nodePositions.get(link.source) && nodePositions.get(link.target)"
              :x1="nodePositions.get(link.source)!.x"
              :y1="nodePositions.get(link.source)!.y"
              :x2="nodePositions.get(link.target)!.x"
              :y2="nodePositions.get(link.target)!.y"
              :stroke="link.quality === 'excellent' ? '#10b981' : link.quality === 'good' ? '#3b82f6' : '#f59e0b'"
              :stroke-width="link.quality === 'excellent' ? 3 : 2"
              :stroke-dasharray="link.quality === 'weak' ? '4' : '0'"
            />
            <text
              v-if="nodePositions.get(link.source) && nodePositions.get(link.target) && link.lqi"
              :x="(nodePositions.get(link.source)!.x + nodePositions.get(link.target)!.x) / 2"
              :y="(nodePositions.get(link.source)!.y + nodePositions.get(link.target)!.y) / 2 - 6"
              :fill="link.quality === 'excellent' ? '#10b981' : link.quality === 'good' ? '#3b82f6' : '#f59e0b'"
              font-size="11"
              font-weight="bold"
              text-anchor="middle"
            >
              LQI {{ link.lqi }}
            </text>
          </g>

          <!-- Dynamic Nodes -->
          <g
            v-for="node in topology.nodes"
            :key="node.id"
            class="node-group"
            :class="node.type"
            @click="selectedNode = node"
            :transform="`translate(${nodePositions.get(node.id)?.x || 400}, ${nodePositions.get(node.id)?.y || 200})`"
            style="cursor: pointer;"
          >
            <circle
              cx="0"
              cy="0"
              :r="node.type === 'coordinator' ? 30 : node.type === 'router' ? 24 : 20"
              :fill="node.type === 'coordinator' ? '#6366f1' : node.type === 'router' ? '#10b981' : '#f59e0b'"
              :filter="node.type === 'coordinator' ? 'drop-shadow(0 0 10px rgba(99,102,241,0.6))' : 'none'"
            />
            <text x="0" y="4" text-anchor="middle" fill="#fff" font-size="10" font-weight="bold">
              {{ node.type === 'coordinator' ? 'GW' : node.battery ? `${node.battery}%` : node.type === 'router' ? 'RTR' : 'END' }}
            </text>
            <text x="0" y="32" text-anchor="middle" fill="var(--text-primary, #fff)" font-size="11" font-weight="500">
              {{ node.name.length > 15 ? node.name.slice(0, 13) + '...' : node.name }}
            </text>
          </g>
        </svg>
      </div>

      <!-- Node Inspector Drawer (if selected) -->
      <div v-if="selectedNode" class="node-inspector">
        <div class="inspector-header">
          <h3>{{ selectedNode.name }}</h3>
          <button class="close-btn" @click="selectedNode = null">✕</button>
        </div>
        <div class="inspector-props">
          <div class="prop-row">
            <span>Tipo Nodo:</span>
            <strong>{{ selectedNode.type.toUpperCase() }}</strong>
          </div>
          <div class="prop-row">
            <span>Link Quality (LQI):</span>
            <strong :class="selectedNode.lqi > 150 ? 'text-success' : 'text-warning'">{{ selectedNode.lqi }} / 255</strong>
          </div>
          <div v-if="selectedNode.battery" class="prop-row">
            <span>Batteria Residua:</span>
            <strong>{{ selectedNode.battery }}%</strong>
          </div>
        </div>
      </div>
    </div>

    <!-- Zigbee Direct Binding Section -->
    <div class="mesh-binding-section glass-panel">
      <div class="binding-header">
        <div class="header-title-box">
          <div class="icon-circle">
            <Radio :size="24" class="text-primary" />
          </div>
          <div>
            <h2 class="binding-title">Zigbee Direct Binding (Associazione Diretta Senza Hub)</h2>
            <p class="binding-desc">Associa direttamente un telecomando wireless o pulsante a una lampadina/gruppo. Continua a funzionare istantaneamente anche se il server Domus è spento!</p>
          </div>
        </div>
        <button class="proto-btn active" @click="isBindingModalOpen = true">
          + Nuovo Legame Diretto
        </button>
      </div>

      <div v-if="bindings.length === 0" class="binding-empty">
        <Radio :size="32" class="empty-icon" />
        <p>Nessun legame diretto attivo. Crea una connessione diretta hardware tra un interruttore e un dispositivo attuatore.</p>
      </div>

      <div v-else class="bindings-grid">
        <div v-for="b in bindings" :key="b.id" class="binding-card">
          <div class="binding-card-header">
            <span class="source-tag">Sorgente: {{ b.source_device_name }}</span>
            <span class="arrow-tag">➜</span>
            <span class="target-tag">Destinazione: {{ b.target_device_name }}</span>
          </div>
          <div class="binding-card-body">
            <span>Cluster: <code>{{ b.clusters?.join(', ') || 'genOnOff' }}</code></span>
            <button class="unbind-btn" title="Rimuovi legame" @click="handleDeleteBinding(b.id)">
              Elimina
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Binding Modal -->
    <div v-if="isBindingModalOpen" class="modal-backdrop" @click.self="isBindingModalOpen = false">
      <div class="modal-card glass-panel">
        <div class="modal-header">
          <h3>Nuovo Binding Zigbee Diretto</h3>
          <button class="close-btn" @click="isBindingModalOpen = false">✕</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>Dispositivo Sorgente (Telecomando/Interruttore)</label>
            <input v-model="sourceDeviceName" type="text" placeholder="Nome (es. Telecomando Letto)" class="form-input" />
            <input v-model="sourceDeviceID" type="text" placeholder="IEEE / ID Dispositivo" class="form-input" style="margin-top: 6px;" />
          </div>

          <div class="form-group">
            <label>Dispositivo Destinazione (Lampadina/Relè/Gruppo)</label>
            <input v-model="targetDeviceName" type="text" placeholder="Nome (es. Luce Comodino)" class="form-input" />
            <input v-model="targetDeviceID" type="text" placeholder="IEEE / ID Dispositivo" class="form-input" style="margin-top: 6px;" />
          </div>

          <div class="form-group">
            <label>Cluster Funzionale</label>
            <select v-model="selectedClusters[0]" class="form-input">
              <option value="genOnOff">Accensione/Spegnimento (genOnOff)</option>
              <option value="genLevelCtrl">Dimming / Luminosità (genLevelCtrl)</option>
              <option value="lightingColorCtrl">Colore / Temperatura (lightingColorCtrl)</option>
            </select>
          </div>
        </div>
        <div class="modal-footer">
          <button class="cancel-btn" @click="isBindingModalOpen = false">Annulla</button>
          <button class="submit-btn" @click="handleCreateBinding">Crea Legame Diretto</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mesh-view-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.mesh-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-radius: 16px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
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
  background: rgba(16, 185, 129, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-title {
  font-size: 1.4rem;
  font-weight: 700;
  margin: 0;
}

.page-subtitle {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin: 4px 0 0;
}

.protocol-pills {
  display: flex;
  gap: 8px;
  background: rgba(255, 255, 255, 0.05);
  padding: 4px;
  border-radius: 10px;
}

.proto-btn {
  padding: 8px 16px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: var(--text-muted);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
}

.proto-btn.active {
  background: var(--primary);
  color: #fff;
}

.mesh-canvas-card {
  padding: 24px;
  border-radius: 16px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  position: relative;
}

.canvas-legend-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 16px;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.dot.coordinator { background: #6366f1; }
.dot.router { background: #10b981; }
.dot.end-device { background: #f59e0b; }

.link-line {
  width: 24px;
  height: 3px;
  border-radius: 2px;
}

.link-line.excellent { background: #10b981; }
.link-line.good { background: #3b82f6; }
.link-line.weak { background: #f59e0b; }

.legend-separator {
  width: 1px;
  height: 16px;
  background: var(--border-color);
}

.svg-graph-container {
  width: 100%;
  display: flex;
  justify-content: center;
}

.mesh-svg {
  width: 100%;
  max-width: 900px;
  height: auto;
}

.node-group {
  cursor: pointer;
  transition: transform 0.2s;
}

.node-group:hover circle {
  stroke: #fff;
  stroke-width: 2px;
}

.node-inspector {
  position: absolute;
  bottom: 24px;
  right: 24px;
  width: 280px;
  background: rgba(18, 18, 28, 0.95);
  backdrop-filter: blur(12px);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px;
}

.inspector-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.inspector-header h3 {
  font-size: 0.95rem;
  margin: 0;
}

.close-btn {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
}

.inspector-props {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 0.85rem;
}

.prop-row {
  display: flex;
  justify-content: space-between;
}
</style>
