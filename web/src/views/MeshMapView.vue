<script setup lang="ts">
import { ref, onMounted } from 'vue'
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
  nodes: [
    {
      id: 'coord_1',
      name: 'Coordinator Zigbee USB',
      protocol: 'zigbee',
      type: 'coordinator',
      lqi: 255
    },
    {
      id: 'router_1',
      name: 'Presa Smart Salotto (Router)',
      protocol: 'zigbee',
      type: 'router',
      lqi: 215,
      parent_id: 'coord_1'
    },
    {
      id: 'router_2',
      name: 'Relè Incasso Cucina (Router)',
      protocol: 'zigbee',
      type: 'router',
      lqi: 160,
      parent_id: 'coord_1'
    },
    {
      id: 'sensor_1',
      name: 'Sensore Balcone (End-device)',
      protocol: 'zigbee',
      type: 'end_device',
      lqi: 75,
      battery: 85,
      parent_id: 'router_1'
    },
    {
      id: 'sensor_2',
      name: 'Sensore Camera (End-device)',
      protocol: 'zigbee',
      type: 'end_device',
      lqi: 190,
      battery: 92,
      parent_id: 'router_2'
    }
  ],
  links: [
    { source: 'coord_1', target: 'router_1', lqi: 215, quality: 'excellent' },
    { source: 'coord_1', target: 'router_2', lqi: 160, quality: 'good' },
    { source: 'router_1', target: 'router_2', lqi: 110, quality: 'good' },
    { source: 'router_1', target: 'sensor_1', lqi: 75, quality: 'weak' },
    { source: 'router_2', target: 'sensor_2', lqi: 190, quality: 'excellent' }
  ]
})

const selectedNode = ref<MeshNode | null>(null)

async function loadTopology(proto: 'zigbee' | 'zwave' | 'thread') {
  currentProtocol.value = proto
  try {
    const data = await apiClient.get<MeshTopology>(`/api/v1/mesh/topology?protocol=${proto}`)
    if (data && data.nodes && data.nodes.length > 0) {
      topology.value = data
    }
  } catch {}
}

onMounted(() => {
  loadTopology('zigbee')
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
          <h1 class="page-title">Diagnostica Radio & Mappa Mesh</h1>
          <p class="page-subtitle">Topologia della rete, qualità di segnale LQI e salti di routing per Zigbee, Z-Wave e Thread</p>
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

      <!-- Graph Diagram SVG -->
      <div class="svg-graph-container">
        <svg viewBox="0 0 800 450" class="mesh-svg">
          <!-- Links -->
          <line x1="400" y1="80" x2="220" y2="220" stroke="#10b981" stroke-width="3" stroke-dasharray="4" />
          <line x1="400" y1="80" x2="580" y2="220" stroke="#3b82f6" stroke-width="2.5" />
          <line x1="220" y1="220" x2="580" y2="220" stroke="#3b82f6" stroke-width="2" stroke-opacity="0.5" />
          <line x1="220" y1="220" x2="140" y2="360" stroke="#f59e0b" stroke-width="2" />
          <line x1="580" y1="220" x2="660" y2="360" stroke="#10b981" stroke-width="3" />

          <!-- Link LQI Badges -->
          <text x="300" y="140" fill="#10b981" font-size="12" font-weight="bold">LQI 215</text>
          <text x="500" y="140" fill="#3b82f6" font-size="12" font-weight="bold">LQI 160</text>
          <text x="170" y="295" fill="#f59e0b" font-size="12" font-weight="bold">LQI 75</text>
          <text x="630" y="295" fill="#10b981" font-size="12" font-weight="bold">LQI 190</text>

          <!-- Nodes -->
          <!-- 1. Coordinator -->
          <g class="node-group coordinator" @click="selectedNode = topology.nodes[0]">
            <circle cx="400" cy="80" r="32" fill="#6366f1" filter="drop-shadow(0 0 12px rgba(99,102,241,0.6))" />
            <text x="400" y="85" text-anchor="middle" fill="#fff" font-size="12" font-weight="bold">GW USB</text>
            <text x="400" y="125" text-anchor="middle" fill="#fff" font-size="11">Coordinator</text>
          </g>

          <!-- 2. Routers -->
          <g class="node-group router" @click="selectedNode = topology.nodes[1]">
            <circle cx="220" cy="220" r="26" fill="#10b981" />
            <text x="220" y="225" text-anchor="middle" fill="#fff" font-size="10" font-weight="bold">ROUTER</text>
            <text x="220" y="260" text-anchor="middle" fill="#fff" font-size="11">Salotto</text>
          </g>

          <g class="node-group router" @click="selectedNode = topology.nodes[2]">
            <circle cx="580" cy="220" r="26" fill="#10b981" />
            <text x="580" y="225" text-anchor="middle" fill="#fff" font-size="10" font-weight="bold">ROUTER</text>
            <text x="580" y="260" text-anchor="middle" fill="#fff" font-size="11">Cucina</text>
          </g>

          <!-- 3. End Devices -->
          <g class="node-group end-device" @click="selectedNode = topology.nodes[3]">
            <circle cx="140" cy="360" r="22" fill="#f59e0b" />
            <text x="140" y="365" text-anchor="middle" fill="#fff" font-size="10" font-weight="bold">85%</text>
            <text x="140" y="398" text-anchor="middle" fill="#fff" font-size="11">Balcone</text>
          </g>

          <g class="node-group end-device" @click="selectedNode = topology.nodes[4]">
            <circle cx="660" cy="360" r="22" fill="#10b981" />
            <text x="660" y="365" text-anchor="middle" fill="#fff" font-size="10" font-weight="bold">92%</text>
            <text x="660" y="398" text-anchor="middle" fill="#fff" font-size="11">Camera</text>
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
