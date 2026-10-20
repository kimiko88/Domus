<template>
  <main class="zones-view" id="main-content" tabindex="-1">
    <header class="view-header">
      <div class="header-content">
        <h1>Zone Geografiche & Presenza</h1>
        <p class="subtitle">Geofencing spaziale, perimetri di rilevamento e tracciamento dispositivi familiari</p>
      </div>
      <div class="header-actions">
        <button
          class="btn btn-primary"
          @click="openCreateModal"
          aria-label="Crea nuova zona"
        >
          + Nuova Zona
        </button>
      </div>
    </header>

    <!-- Visual Map Canvas / Radar -->
    <section class="map-section" aria-label="Mappa visiva delle zone">
      <div class="map-card">
        <div class="map-header">
          <h2>Mappa Geofencing Interattiva</h2>
          <span class="active-count">{{ store.zones.length }} Zone Configurate</span>
        </div>
        <div class="svg-map-wrapper">
          <svg class="radar-svg" viewBox="0 0 600 400" preserveAspectRatio="xMidYMid meet" role="img" aria-label="Mappa delle zone">
            <!-- Grid lines -->
            <defs>
              <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
                <path d="M 40 0 L 0 0 0 40" fill="none" stroke="rgba(255,255,255,0.05)" stroke-width="1" />
              </pattern>
            </defs>
            <rect width="100%" height="100%" fill="url(#grid)" />

            <!-- Concentric range markers -->
            <circle cx="300" cy="200" r="160" fill="none" stroke="rgba(56, 189, 248, 0.15)" stroke-dasharray="4 4" />
            <circle cx="300" cy="200" r="100" fill="none" stroke="rgba(56, 189, 248, 0.2)" stroke-dasharray="4 4" />
            <circle cx="300" cy="200" r="50" fill="none" stroke="rgba(56, 189, 248, 0.25)" />

            <!-- Render Zones -->
            <g v-for="(zone, index) in store.zones" :key="zone.id">
              <circle
                :cx="getZoneX(index)"
                :cy="getZoneY(index)"
                :r="Math.max(20, Math.min(60, zone.radius / 10))"
                :class="['zone-circle', { active: selectedZone?.id === zone.id, passive: zone.passive }]"
                @click="selectZone(zone)"
                role="button"
                :aria-label="`Zona ${zone.name}, raggio ${zone.radius} metri`"
                tabindex="0"
                @keydown.enter="selectZone(zone)"
              />
              <text
                :x="getZoneX(index)"
                :y="getZoneY(index) + 5"
                text-anchor="middle"
                class="zone-label"
              >
                {{ zone.name }}
              </text>
            </g>
          </svg>
        </div>
      </div>
    </section>

    <!-- Zones Grid & Person Trackers -->
    <div class="content-grid">
      <!-- Zones List -->
      <section class="zones-list-section" aria-label="Lista delle zone">
        <h2>Zone Attive</h2>
        <div v-if="store.loading && store.zones.length === 0" class="loading-state">
          <p>Caricamento zone in corso...</p>
        </div>
        <div v-else-if="store.zones.length === 0" class="empty-state">
          <p>Nessuna zona creata. Crea "Casa", "Lavoro" o "Scuola" per iniziare.</p>
        </div>
        <div v-else class="zones-cards">
          <article
            v-for="zone in store.zones"
            :key="zone.id"
            class="zone-card"
            :class="{ selected: selectedZone?.id === zone.id }"
            @click="selectZone(zone)"
          >
            <div class="zone-card-top">
              <span class="zone-icon" aria-hidden="true">{{ zone.icon || '📍' }}</span>
              <div class="zone-info">
                <h3>{{ zone.name }}</h3>
                <span class="zone-coords">{{ zone.latitude.toFixed(4) }}, {{ zone.longitude.toFixed(4) }}</span>
              </div>
              <span v-if="zone.passive" class="tag-passive">Passiva</span>
            </div>
            <div class="zone-card-details">
              <span>Raggio: <strong>{{ zone.radius }} m</strong></span>
              <button
                class="btn-icon-danger"
                @click.stop="handleDelete(zone.id)"
                :aria-label="`Elimina zona ${zone.name}`"
              >
                🗑️
              </button>
            </div>
          </article>
        </div>
      </section>

      <!-- Tracked Persons (Family) -->
      <section class="persons-section" aria-label="Membri tracciati">
        <h2>Persone & Rilevamento</h2>
        <div v-if="store.persons.length === 0" class="empty-state">
          <p>Nessun membro con device tracker assegnato.</p>
        </div>
        <div v-else class="persons-list">
          <article v-for="person in store.persons" :key="person.id" class="person-card">
            <div class="person-avatar" aria-hidden="true">👤</div>
            <div class="person-meta">
              <h3>{{ person.name }}</h3>
              <span class="person-state" :class="person.state">{{ person.state }}</span>
            </div>
            <div class="person-trackers">
              <span
                v-for="tracker in person.device_trackers"
                :key="tracker"
                class="tracker-badge"
              >
                📱 {{ tracker }}
              </span>
            </div>
          </article>
        </div>
      </section>
    </div>

    <!-- Vacation Presence Simulator -->
    <section class="vacation-section" aria-label="Simulatore di Presenza Vacanza">
      <div class="vacation-header">
        <div class="vacation-title-box">
          <span class="vacation-icon-badge" aria-hidden="true">🏖️</span>
          <div>
            <h2 class="vacation-title">Simulatore di Presenza — Modalità Vacanza</h2>
            <p class="vacation-desc">
              Riproduce le abitudini quotidiane (luci, tapparelle, TV) con ±{{ vacationStatus.jitter_minutes }} min di jitter gaussiano
              per scoraggiare intrusioni e profilazioni esterne durante l'assenza.
            </p>
          </div>
        </div>

        <!-- Enable/Disable Toggle -->
        <div class="vacation-toggle-area">
          <span class="toggle-label">{{ vacationStatus.enabled ? 'Attiva' : 'Disattiva' }}</span>
          <button
            type="button"
            class="vacation-toggle"
            :class="{ enabled: vacationStatus.enabled }"
            role="switch"
            :aria-checked="vacationStatus.enabled"
            aria-label="Attiva o disattiva Modalità Vacanza"
            :disabled="isTogglingVacation"
            @click="toggleVacationMode"
          >
            <span class="vacation-toggle-thumb" />
          </button>
        </div>
      </div>

      <!-- Status Banner -->
      <div class="vacation-status-banner" :class="vacationStatus.enabled ? 'banner-active' : 'banner-inactive'">
        <span class="status-dot" />
        <span v-if="vacationStatus.enabled">
          🛡️ Simulazione attiva — le abitudini familiari vengono riprodotte automaticamente con perturbazione casuale.
        </span>
        <span v-else>
          Modalità Vacanza disattivata — abilita per attivare il replay intelligente delle abitudini in tua assenza.
        </span>
      </div>

      <!-- Simulated Actions List -->
      <div class="vacation-actions-wrap">
        <div class="va-header-row">
          <h3 class="va-title">Azioni Simulate Registrate</h3>
          <button
            type="button"
            class="va-add-btn"
            @click="openAddActionModal"
            aria-label="Aggiungi azione simulata"
          >
            + Aggiungi Azione
          </button>
        </div>

        <div v-if="simulatedActions.length === 0" class="va-empty">
          <span>Nessuna azione simulata. Aggiungi luci, tapparelle o TV che vuoi replicare durante la vacanza.</span>
        </div>
        <div v-else class="va-actions-grid">
          <div
            v-for="action in simulatedActions"
            :key="action.entity_id + action.typical_time"
            class="va-action-card"
          >
            <div class="va-action-icon">⏰</div>
            <div class="va-action-body">
              <span class="va-entity">{{ action.entity_id }}</span>
              <span class="va-cmd">{{ action.command }}</span>
            </div>
            <span class="va-time">{{ action.typical_time }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- Add Simulated Action Modal -->
    <div v-if="showActionModal" class="modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="action-modal-title" @click.self="showActionModal = false">
      <div class="modal-card">
        <h2 id="action-modal-title">Aggiungi Azione Simulata</h2>
        <form @submit.prevent="saveSimulatedAction" class="modal-form">
          <div class="form-group">
            <label for="va-entity">Entity ID (es. light.salotto)</label>
            <input id="va-entity" v-model="actionForm.entity_id" required placeholder="light.salotto" class="input" />
          </div>
          <div class="form-group">
            <label for="va-command">Comando (es. turn_on, turn_off)</label>
            <input id="va-command" v-model="actionForm.command" required placeholder="turn_on" class="input" />
          </div>
          <div class="form-group">
            <label for="va-time">Orario Tipico (HH:MM)</label>
            <input id="va-time" v-model="actionForm.typical_time" type="time" required class="input" />
          </div>
          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="showActionModal = false">Annulla</button>
            <button type="submit" class="btn btn-primary" :disabled="savingAction">Salva Azione</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Create/Edit Zone Modal -->
    <div v-if="showModal" class="modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="modal-title">
      <div class="modal-card">
        <h2 id="modal-title">Crea Nuova Zona Geografica</h2>
        <form @submit.prevent="saveZone" class="modal-form">
          <div class="form-group">
            <label for="zone-name">Nome Zona</label>
            <input id="zone-name" v-model="form.name" required placeholder="Es. Casa, Ufficio, Palestra" class="input" />
          </div>

          <div class="form-row">
            <div class="form-group">
              <label for="zone-lat">Latitudine</label>
              <input id="zone-lat" v-model.number="form.latitude" type="number" step="any" required class="input" />
            </div>
            <div class="form-group">
              <label for="zone-lon">Longitudine</label>
              <input id="zone-lon" v-model.number="form.longitude" type="number" step="any" required class="input" />
            </div>
          </div>

          <div class="form-group">
            <label for="zone-radius">Raggio di rilevamento: {{ form.radius }} metri</label>
            <input id="zone-radius" v-model.number="form.radius" type="range" min="50" max="2000" step="25" class="range" />
          </div>

          <div class="form-group checkbox-group">
            <label>
              <input type="checkbox" v-model="form.passive" />
              Zona Passiva (non altera lo stato di presenza principale)
            </label>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="closeModal">Annulla</button>
            <button type="submit" class="btn btn-primary" :disabled="saving">Salva Zona</button>
          </div>
        </form>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useZonesStore } from '../stores/zones'
import { apiClient } from '../api/client'
import type { VacationModeStatus, VacationSimulatedAction } from '../api/types'
import type { SpatialZone } from '../api/types'

const store = useZonesStore()
const selectedZone = ref<SpatialZone | null>(null)
const showModal = ref(false)
const saving = ref(false)

const form = ref<Partial<SpatialZone>>({
  name: '',
  latitude: 41.9028,
  longitude: 12.4964,
  radius: 150,
  icon: '📍',
  passive: false,
})

// ─── Vacation Presence Simulator ───────────────────────────────────────────
const vacationStatus = ref<VacationModeStatus>({ enabled: false, jitter_minutes: 15 })
const simulatedActions = ref<VacationSimulatedAction[]>([])
const isTogglingVacation = ref(false)
const showActionModal = ref(false)
const savingAction = ref(false)
const actionForm = ref<VacationSimulatedAction>({ entity_id: '', command: 'turn_on', typical_time: '20:00' })

async function fetchVacationStatus() {
  try {
    const res = await apiClient.get<VacationModeStatus>('/api/v1/presence/vacation')
    if (res) vacationStatus.value = res
  } catch {}
}

async function toggleVacationMode() {
  isTogglingVacation.value = true
  try {
    const newState = !vacationStatus.value.enabled
    const res = await apiClient.post<VacationModeStatus>('/api/v1/presence/vacation', { enabled: newState })
    if (res) vacationStatus.value = { ...vacationStatus.value, enabled: res.enabled ?? newState }
  } catch {} finally {
    isTogglingVacation.value = false
  }
}

function openAddActionModal() {
  actionForm.value = { entity_id: '', command: 'turn_on', typical_time: '20:00' }
  showActionModal.value = true
}

async function saveSimulatedAction() {
  if (!actionForm.value.entity_id || !actionForm.value.command) return
  savingAction.value = true
  try {
    const res = await apiClient.post<VacationSimulatedAction>('/api/v1/presence/vacation/actions', actionForm.value)
    if (res) simulatedActions.value.push(res)
    showActionModal.value = false
  } catch {} finally {
    savingAction.value = false
  }
}
// ────────────────────────────────────────────────────────────────────────────

onMounted(async () => {
  await store.fetchZones()
  await store.fetchPersons()
  await fetchVacationStatus()
  if (store.zones.length > 0) {
    selectedZone.value = store.zones[0]
  }
})

function selectZone(zone: SpatialZone) {
  selectedZone.value = zone
}

function openCreateModal() {
  form.value = {
    name: '',
    latitude: 41.9028,
    longitude: 12.4964,
    radius: 150,
    icon: '📍',
    passive: false,
  }
  showModal.value = true
}

function closeModal() {
  showModal.value = false
}

async function saveZone() {
  saving.value = true
  try {
    await store.createZone(form.value)
    closeModal()
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: string) {
  if (confirm('Sei sicuro di voler eliminare questa zona?')) {
    await store.deleteZone(id)
  }
}

function getZoneX(index: number): number {
  const positions = [300, 160, 440, 200, 400]
  return positions[index % positions.length]
}

function getZoneY(index: number): number {
  const positions = [200, 120, 280, 280, 120]
  return positions[index % positions.length]
}
</script>

<style scoped>
.zones-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-6, 1.5rem);
  padding: var(--space-6, 1.5rem);
  max-width: 1200px;
  margin: 0 auto;
}

.view-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

h1 {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--text-primary, #ffffff);
  margin: 0;
}

.subtitle {
  color: var(--text-secondary, #94a3b8);
  margin-top: 0.25rem;
  font-size: 0.95rem;
}

.map-card {
  background: var(--surface-card, rgba(30, 41, 59, 0.7));
  border: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.08));
  border-radius: var(--radius-lg, 12px);
  padding: 1.25rem;
  backdrop-filter: blur(12px);
}

.map-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.map-header h2 {
  font-size: 1.15rem;
  color: var(--text-primary, #ffffff);
  margin: 0;
}

.active-count {
  font-size: 0.85rem;
  color: var(--primary, #38bdf8);
  background: rgba(56, 189, 248, 0.1);
  padding: 0.2rem 0.6rem;
  border-radius: var(--radius-full, 9999px);
}

.svg-map-wrapper {
  width: 100%;
  height: 280px;
  background: #090d16;
  border-radius: var(--radius-md, 8px);
  overflow: hidden;
  display: flex;
  justify-content: center;
  align-items: center;
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.radar-svg {
  width: 100%;
  height: 100%;
}

.zone-circle {
  fill: rgba(56, 189, 248, 0.25);
  stroke: #38bdf8;
  stroke-width: 2;
  cursor: pointer;
  transition: all 0.2s ease;
}

.zone-circle:hover,
.zone-circle.active {
  fill: rgba(56, 189, 248, 0.5);
  stroke-width: 3;
  filter: drop-shadow(0 0 8px rgba(56, 189, 248, 0.8));
}

.zone-circle.passive {
  fill: rgba(148, 163, 184, 0.2);
  stroke: #94a3b8;
}

.zone-label {
  fill: #ffffff;
  font-size: 12px;
  font-weight: 600;
  pointer-events: none;
}

.content-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
}

@media (max-width: 768px) {
  .content-grid {
    grid-template-columns: 1fr;
  }
}

.zones-cards,
.persons-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.zone-card,
.person-card {
  background: var(--surface-card, rgba(30, 41, 59, 0.6));
  border: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.08));
  border-radius: var(--radius-md, 10px);
  padding: 1rem;
  backdrop-filter: blur(8px);
  cursor: pointer;
  transition: all 0.15s ease;
}

.zone-card:hover,
.zone-card.selected {
  border-color: var(--primary, #38bdf8);
}

.zone-card-top {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.zone-icon {
  font-size: 1.5rem;
}

.zone-info h3 {
  margin: 0;
  font-size: 1rem;
  color: var(--text-primary, #ffffff);
}

.zone-coords {
  font-size: 0.8rem;
  color: var(--text-secondary, #94a3b8);
}

.tag-passive {
  margin-left: auto;
  font-size: 0.75rem;
  padding: 0.15rem 0.4rem;
  border-radius: 4px;
  background: rgba(148, 163, 184, 0.15);
  color: #94a3b8;
}

.zone-card-details {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 0.75rem;
  padding-top: 0.5rem;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
  font-size: 0.85rem;
  color: var(--text-secondary, #cbd5e1);
}

.btn-icon-danger {
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.btn-icon-danger:hover {
  background: rgba(239, 68, 68, 0.15);
}

.person-card {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.person-avatar {
  font-size: 1.75rem;
}

.person-meta h3 {
  margin: 0;
  font-size: 1rem;
  color: var(--text-primary, #ffffff);
}

.person-state {
  display: inline-block;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.1rem 0.4rem;
  border-radius: 4px;
  background: rgba(34, 197, 94, 0.15);
  color: #4ade80;
}

.person-trackers {
  margin-left: auto;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.tracker-badge {
  font-size: 0.75rem;
  background: rgba(255, 255, 255, 0.06);
  padding: 0.15rem 0.5rem;
  border-radius: 4px;
  color: var(--text-secondary, #cbd5e1);
}

.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-card {
  background: #1e293b;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  padding: 1.5rem;
  width: 90%;
  max-width: 480px;
}

.modal-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-top: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.input,
.range {
  padding: 0.5rem;
  border-radius: 6px;
  border: 1px solid var(--border-color);
  background: var(--bg-surface);
  color: var(--text-main);
}

.checkbox-group {
  flex-direction: row;
  align-items: center;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1rem;
}

.btn {
  padding: 0.5rem 1rem;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 600;
  min-height: 44px;
}

.btn-primary {
  background: var(--primary, #38bdf8);
  color: #0f172a;
  border: none;
}

.btn-secondary {
  background: var(--bg-surface-hover);
  color: var(--text-main);
  border: 1px solid var(--border-color);
}

/* ─── Vacation Presence Simulator ─── */
.vacation-section {
  background: var(--surface-card, rgba(30, 41, 59, 0.7));
  border: 1px solid var(--border-color);
  border-radius: 1rem;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  backdrop-filter: blur(12px);
}

.vacation-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.vacation-title-box {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  flex: 1;
}

.vacation-icon-badge {
  font-size: 2rem;
  flex-shrink: 0;
  margin-top: 0.1rem;
}

.vacation-title {
  font-size: 1.05rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0 0 0.25rem;
}

.vacation-desc {
  font-size: 0.8rem;
  color: var(--text-muted);
  margin: 0;
  line-height: 1.45;
}

.vacation-toggle-area {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-shrink: 0;
}

.toggle-label {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-muted);
  min-width: 56px;
  text-align: right;
}

.vacation-toggle {
  width: 52px;
  height: 28px;
  border-radius: 14px;
  background: rgba(148, 163, 184, 0.25);
  border: 1px solid rgba(148, 163, 184, 0.3);
  position: relative;
  cursor: pointer;
  transition: background 0.25s ease, border-color 0.25s ease;
  padding: 0;
}

.vacation-toggle.enabled {
  background: rgba(16, 185, 129, 0.4);
  border-color: #10b981;
}

.vacation-toggle:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.vacation-toggle-thumb {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #ffffff;
  transition: transform 0.25s ease;
  display: block;
}

.vacation-toggle.enabled .vacation-toggle-thumb {
  transform: translateX(24px);
}

/* Status Banner */
.vacation-status-banner {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-radius: 0.6rem;
  font-size: 0.85rem;
}

.banner-active {
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.25);
  color: #a7f3d0;
}

.banner-inactive {
  background: rgba(148, 163, 184, 0.08);
  border: 1px solid rgba(148, 163, 184, 0.15);
  color: var(--text-muted);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
  flex-shrink: 0;
  opacity: 0.8;
}

.banner-active .status-dot {
  box-shadow: 0 0 6px #10b981;
  animation: pulseDot 2s infinite;
}

@keyframes pulseDot {
  0%, 100% { opacity: 0.5; }
  50% { opacity: 1; }
}

/* Actions */
.vacation-actions-wrap {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.va-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.va-title {
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.va-add-btn {
  font-size: 0.8rem;
  font-weight: 700;
  padding: 0.35rem 0.75rem;
  border-radius: 0.5rem;
  background: rgba(56, 189, 248, 0.12);
  color: #38bdf8;
  border: 1px solid rgba(56, 189, 248, 0.25);
  cursor: pointer;
  transition: background 0.2s;
  min-height: 32px;
}

.va-add-btn:hover {
  background: rgba(56, 189, 248, 0.22);
}

.va-empty {
  font-size: 0.8rem;
  color: var(--text-muted);
  padding: 1rem;
  background: rgba(148, 163, 184, 0.05);
  border-radius: 0.5rem;
  border: 1px dashed rgba(148, 163, 184, 0.15);
  text-align: center;
}

.va-actions-grid {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.va-action-card {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.7rem 1rem;
  border-radius: 0.6rem;
  background: rgba(15, 23, 42, 0.5);
  border: 1px solid var(--border-color);
}

.va-action-icon {
  font-size: 1.1rem;
  flex-shrink: 0;
}

.va-action-body {
  display: flex;
  flex-direction: column;
  flex: 1;
}

.va-entity {
  font-size: 0.82rem;
  font-weight: 700;
  color: var(--text-main);
  font-family: var(--font-mono, monospace);
}

.va-cmd {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.va-time {
  font-size: 0.8rem;
  font-weight: 700;
  color: #38bdf8;
  font-family: var(--font-mono, monospace);
  background: rgba(56, 189, 248, 0.1);
  padding: 0.15rem 0.45rem;
  border-radius: 0.35rem;
}
</style>
