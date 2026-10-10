<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  CheckSquare,
  Plus,
  Clock,
  AlertTriangle,
  CheckCircle2,
  Calendar,
  Repeat,
  Trash2,
  X,
  Filter,
  MapPin,
  Flame
} from '@lucide/vue'
import { useChoresStore } from '../stores/chores'
import { useAreasStore } from '../stores/areas'
import { useNotificationsStore } from '../stores/notifications'
import type { Chore, ChorePriority, ChoreRecurrence } from '../api/types'

const choresStore = useChoresStore()
const areasStore = useAreasStore()
const notifStore = useNotificationsStore()

const currentTab = ref<'all' | 'pending' | 'overdue' | 'completed'>('all')
const showCreateModal = ref(false)

const form = ref({
  title: '',
  description: '',
  area_id: '',
  priority: 'medium' as ChorePriority,
  due_date: '',
  recurrence_type: 'none' as ChoreRecurrence
})

onMounted(async () => {
  await Promise.all([
    choresStore.fetchChores(),
    areasStore.fetchAreas()
  ])
})

const filteredChores = computed(() => {
  switch (currentTab.value) {
    case 'pending':
      return choresStore.chores.filter((c) => c.status === 'pending')
    case 'overdue':
      return choresStore.overdueChores
    case 'completed':
      return choresStore.completedChores
    default:
      return choresStore.chores
  }
})

function getAreaName(areaId?: string) {
  if (!areaId) return null
  const area = areasStore.byId(areaId)
  return area ? area.name : null
}

function formatDate(dateStr?: string) {
  if (!dateStr) return 'Nessuna scadenza'
  try {
    const d = new Date(dateStr)
    return d.toLocaleDateString('it-IT', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })
  } catch {
    return dateStr
  }
}

async function handleComplete(chore: Chore) {
  try {
    await choresStore.completeChore(chore.id)
    notifStore.success('Faccenda Completata', `Bravo! "${chore.title}" segnata come completata.`)
  } catch {
    notifStore.error('Errore', 'Impossibile completare la faccenda')
  }
}

async function handleDelete(id: string) {
  try {
    await choresStore.deleteChore(id)
    notifStore.success('Eliminata', 'Faccenda rimossa')
  } catch {
    notifStore.error('Errore', 'Impossibile eliminare la faccenda')
  }
}

async function handleCreateChore() {
  if (!form.value.title.trim()) return

  let dueIso: string | undefined = undefined
  if (form.value.due_date) {
    dueIso = new Date(form.value.due_date).toISOString()
  }

  try {
    await choresStore.createChore({
      title: form.value.title.trim(),
      description: form.value.description.trim() || undefined,
      area_id: form.value.area_id || undefined,
      priority: form.value.priority,
      due_date: dueIso,
      recurrence_type: form.value.recurrence_type
    })
    showCreateModal.value = false
    form.value = {
      title: '',
      description: '',
      area_id: '',
      priority: 'medium',
      due_date: '',
      recurrence_type: 'none'
    }
    notifStore.success('Faccenda Aggiunta', 'Promemoria salvato con successo')
  } catch {
    notifStore.error('Errore', 'Impossibile creare la faccenda')
  }
}
</script>

<template>
  <div class="chores-view">
    <!-- Header -->
    <header class="chores-header">
      <div>
        <div class="title-with-badge">
          <h1>Faccende Domestiche</h1>
          <span class="badge-pill">{{ choresStore.pendingChores.length }} Da fare</span>
        </div>
        <p class="subtitle">Gestisci compiti, pulizie e manutenzioni con promemoria e notifiche automatiche</p>
      </div>

      <button class="btn-primary" @click="showCreateModal = true">
        <Plus class="btn-icon" />
        Nuova Faccenda
      </button>
    </header>

    <!-- Stat Cards Overview -->
    <div class="stats-grid">
      <div class="stat-card urgent" @click="currentTab = 'overdue'">
        <div class="stat-icon-wrap urgent">
          <AlertTriangle class="stat-icon" />
        </div>
        <div class="stat-data">
          <span class="stat-value">{{ choresStore.overdueChores.length }}</span>
          <span class="stat-label">In Scadenza / Scadute</span>
        </div>
      </div>

      <div class="stat-card pending" @click="currentTab = 'pending'">
        <div class="stat-icon-wrap pending">
          <Clock class="stat-icon" />
        </div>
        <div class="stat-data">
          <span class="stat-value">{{ choresStore.pendingChores.length }}</span>
          <span class="stat-label">Compiti Attivi</span>
        </div>
      </div>

      <div class="stat-card completed" @click="currentTab = 'completed'">
        <div class="stat-icon-wrap completed">
          <CheckCircle2 class="stat-icon" />
        </div>
        <div class="stat-data">
          <span class="stat-value">{{ choresStore.completedChores.length }}</span>
          <span class="stat-label">Completate</span>
        </div>
      </div>
    </div>

    <!-- Filter Tabs -->
    <div class="filter-tabs-bar">
      <div class="tabs-list">
        <button
          class="tab-btn"
          :class="{ active: currentTab === 'all' }"
          @click="currentTab = 'all'"
        >
          Tutte ({{ choresStore.chores.length }})
        </button>
        <button
          class="tab-btn"
          :class="{ active: currentTab === 'pending' }"
          @click="currentTab = 'pending'"
        >
          Da Fare
        </button>
        <button
          class="tab-btn"
          :class="{ active: currentTab === 'overdue' }"
          @click="currentTab = 'overdue'"
        >
          Scadute ({{ choresStore.overdueChores.length }})
        </button>
        <button
          class="tab-btn"
          :class="{ active: currentTab === 'completed' }"
          @click="currentTab = 'completed'"
        >
          Completate ({{ choresStore.completedChores.length }})
        </button>
      </div>
    </div>

    <!-- Chores List -->
    <div class="chores-list">
      <div
        v-for="chore in filteredChores"
        :key="chore.id"
        class="chore-card"
        :class="{
          'is-completed': chore.status === 'completed',
          'is-overdue': chore.status === 'overdue'
        }"
      >
        <!-- Checkbox / Complete Button -->
        <button
          class="check-btn"
          :class="{ done: chore.status === 'completed' }"
          title="Segna come completato"
          @click="handleComplete(chore)"
        >
          <CheckCircle2 v-if="chore.status === 'completed'" class="check-icon done" />
          <div v-else class="check-circle" />
        </button>

        <!-- Main Info -->
        <div class="chore-main">
          <div class="chore-title-row">
            <h3 class="chore-title" :class="{ strike: chore.status === 'completed' }">
              {{ chore.title }}
            </h3>

            <!-- Priority Tag -->
            <span class="priority-badge" :class="chore.priority">
              <Flame v-if="chore.priority === 'urgent' || chore.priority === 'high'" class="priority-icon" />
              {{ chore.priority }}
            </span>
          </div>

          <p v-if="chore.description" class="chore-desc">{{ chore.description }}</p>

          <!-- Meta tags -->
          <div class="chore-meta">
            <!-- Area / Stanza -->
            <span v-if="getAreaName(chore.area_id)" class="meta-tag area">
              <MapPin class="meta-icon" />
              {{ getAreaName(chore.area_id) }}
            </span>

            <!-- Due Date -->
            <span v-if="chore.due_date" class="meta-tag due" :class="{ overdue: chore.status === 'overdue' }">
              <Calendar class="meta-icon" />
              {{ formatDate(chore.due_date) }}
            </span>

            <!-- Recurrence -->
            <span v-if="chore.recurrence_type !== 'none'" class="meta-tag recurrence">
              <Repeat class="meta-icon" />
              {{ chore.recurrence_type }}
            </span>
          </div>
        </div>

        <!-- Action Delete -->
        <button class="delete-btn" title="Elimina faccenda" @click="handleDelete(chore.id)">
          <Trash2 class="del-icon" />
        </button>
      </div>

      <!-- Empty State -->
      <div v-if="filteredChores.length === 0" class="empty-state">
        <CheckSquare class="empty-icon" />
        <h3>Nessuna faccenda in questa vista</h3>
        <p>Aggiungi una nuova faccenda domestica o seleziona un'altra categoria.</p>
        <button class="btn-primary" @click="showCreateModal = true">
          <Plus class="btn-icon" />
          Crea Faccenda
        </button>
      </div>
    </div>

    <!-- Create Chore Modal -->
    <div v-if="showCreateModal" class="modal-backdrop" @click="showCreateModal = false">
      <div class="modal-card" @click.stop>
        <div class="modal-header">
          <h2>Nuova Faccenda Domestica</h2>
          <button class="modal-close-btn" @click="showCreateModal = false">
            <X class="icon-close" />
          </button>
        </div>

        <form @submit.prevent="handleCreateChore" class="modal-body">
          <div class="form-group">
            <label>Titolo del Compito *</label>
            <input
              v-model="form.title"
              type="text"
              placeholder="es. Pulire filtro robot, annaffiare piante, lavare pavimenti"
              required
            />
          </div>

          <div class="form-group">
            <label>Descrizione / Dettagli</label>
            <textarea
              v-model="form.description"
              rows="2"
              placeholder="Istruzioni o dettagli per il compito"
            ></textarea>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label>Stanza / Area</label>
              <select v-model="form.area_id">
                <option value="">Tutta la casa / Esterno</option>
                <option v-for="area in areasStore.areas" :key="area.id" :value="area.id">
                  {{ area.name }}
                </option>
              </select>
            </div>

            <div class="form-group">
              <label>Priorità</label>
              <select v-model="form.priority">
                <option value="low">Bassa</option>
                <option value="medium">Media</option>
                <option value="high">Alta</option>
                <option value="urgent">Urgente</option>
              </select>
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label>Data & Ora Scadenza</label>
              <input v-model="form.due_date" type="datetime-local" />
            </div>

            <div class="form-group">
              <label>Ricorrenza</label>
              <select v-model="form.recurrence_type">
                <option value="none">Nessuna (Una tantum)</option>
                <option value="daily">Ogni Giorno</option>
                <option value="weekly">Ogni Settimana</option>
                <option value="monthly">Ogni Mese</option>
              </select>
            </div>
          </div>

          <div class="modal-footer">
            <button type="button" class="btn-cancel" @click="showCreateModal = false">Annulla</button>
            <button type="submit" class="btn-primary">Salva Faccenda</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chores-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  padding: 1.5rem;
  max-width: 1200px;
  margin: 0 auto;
  color: #f1f5f9;
}

.chores-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
}

.title-with-badge {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.title-with-badge h1 {
  font-size: 1.75rem;
  font-weight: 700;
  margin: 0;
  color: #fff;
}

.badge-pill {
  font-size: 0.75rem;
  padding: 0.2rem 0.6rem;
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
  border-radius: 9999px;
  font-weight: 600;
}

.subtitle {
  color: #94a3b8;
  font-size: 0.875rem;
  margin-top: 0.25rem;
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: #3b82f6;
  border: none;
  color: #fff;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary:hover {
  background: #2563eb;
}

.btn-icon {
  width: 1rem;
  height: 1rem;
}

/* Stats Overview */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 1rem;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 0.75rem;
  padding: 1.25rem;
  cursor: pointer;
  transition: transform 0.2s ease, background 0.2s;
}

.stat-card:hover {
  transform: translateY(-2px);
  background: rgba(30, 41, 59, 0.8);
}

.stat-icon-wrap {
  width: 3rem;
  height: 3rem;
  border-radius: 0.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-icon-wrap.urgent {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.stat-icon-wrap.pending {
  background: rgba(59, 130, 246, 0.15);
  color: #3b82f6;
}

.stat-icon-wrap.completed {
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
}

.stat-icon {
  width: 1.5rem;
  height: 1.5rem;
}

.stat-data {
  display: flex;
  flex-direction: column;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: #fff;
}

.stat-label {
  font-size: 0.8rem;
  color: #94a3b8;
}

/* Filter Tabs */
.filter-tabs-bar {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  padding-bottom: 0.5rem;
}

.tabs-list {
  display: flex;
  gap: 0.5rem;
}

.tab-btn {
  padding: 0.5rem 1rem;
  background: none;
  border: none;
  color: #94a3b8;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
  transition: all 0.2s;
}

.tab-btn:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.05);
}

.tab-btn.active {
  color: #3b82f6;
  background: rgba(59, 130, 246, 0.15);
  font-weight: 600;
}

/* Chores List */
.chores-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.chore-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 0.75rem;
  padding: 1rem 1.25rem;
  transition: all 0.2s ease;
}

.chore-card:hover {
  background: rgba(30, 41, 59, 0.7);
  border-color: rgba(255, 255, 255, 0.15);
}

.chore-card.is-overdue {
  border-left: 4px solid #ef4444;
}

.chore-card.is-completed {
  opacity: 0.6;
}

.check-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.check-circle {
  width: 1.5rem;
  height: 1.5rem;
  border-radius: 50%;
  border: 2px solid #64748b;
  transition: all 0.2s;
}

.check-circle:hover {
  border-color: #22c55e;
  transform: scale(1.1);
}

.check-icon.done {
  width: 1.5rem;
  height: 1.5rem;
  color: #22c55e;
}

.chore-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.chore-title-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.chore-title {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: #f8fafc;
}

.chore-title.strike {
  text-decoration: line-through;
  color: #64748b;
}

.chore-desc {
  margin: 0;
  font-size: 0.8rem;
  color: #94a3b8;
}

.priority-badge {
  font-size: 0.65rem;
  padding: 0.15rem 0.5rem;
  border-radius: 9999px;
  font-weight: 600;
  text-transform: uppercase;
  display: flex;
  align-items: center;
  gap: 0.2rem;
}

.priority-badge.low {
  background: rgba(148, 163, 184, 0.2);
  color: #94a3b8;
}

.priority-badge.medium {
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
}

.priority-badge.high {
  background: rgba(249, 115, 22, 0.2);
  color: #fb923c;
}

.priority-badge.urgent {
  background: rgba(239, 68, 68, 0.2);
  color: #f87171;
}

.priority-icon {
  width: 0.75rem;
  height: 0.75rem;
}

.chore-meta {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 0.25rem;
  flex-wrap: wrap;
}

.meta-tag {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.75rem;
  color: #94a3b8;
}

.meta-tag.due.overdue {
  color: #ef4444;
  font-weight: 600;
}

.meta-icon {
  width: 0.85rem;
  height: 0.85rem;
}

.delete-btn {
  background: none;
  border: none;
  color: #64748b;
  cursor: pointer;
  padding: 0.5rem;
  border-radius: 0.375rem;
  transition: all 0.2s;
}

.delete-btn:hover {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.1);
}

.del-icon {
  width: 1.1rem;
  height: 1.1rem;
}

.empty-state {
  text-align: center;
  padding: 3rem 1rem;
  background: rgba(15, 23, 42, 0.4);
  border: 1px dashed rgba(255, 255, 255, 0.1);
  border-radius: 0.75rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
}

.empty-icon {
  width: 3rem;
  height: 3rem;
  color: #475569;
}

.empty-state h3 {
  margin: 0;
  color: #cbd5e1;
}

.empty-state p {
  color: #64748b;
  font-size: 0.875rem;
  margin: 0;
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
}

.modal-card {
  background: #0f172a;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 0.75rem;
  width: 100%;
  max-width: 520px;
  padding: 1.5rem;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.7);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
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

.form-group input,
.form-group textarea,
.form-group select {
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
</style>
