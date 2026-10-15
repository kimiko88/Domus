<template>
  <main class="logbook-view" id="main-content" tabindex="-1">
    <header class="view-header">
      <div class="header-content">
        <h1>Registro Eventi (Logbook)</h1>
        <p class="subtitle">Timeline contestuale dell'attività della casa intelligente</p>
      </div>
      <div class="header-actions">
        <button
          class="btn btn-secondary"
          @click="refresh"
          :disabled="store.loading"
          aria-label="Aggiorna registro eventi"
        >
          <span v-if="store.loading">Caricamento...</span>
          <span v-else>Aggiorna</span>
        </button>
      </div>
    </header>

    <!-- Filters Bar -->
    <section class="filters-bar" aria-label="Filtri registro eventi">
      <div class="search-box">
        <input
          v-model="searchQuery"
          type="search"
          placeholder="Cerca evento, entità o utente..."
          aria-label="Cerca nel logbook"
          class="input-search"
        />
      </div>

      <div class="domain-chips" role="group" aria-label="Filtro dominio">
        <button
          v-for="d in domains"
          :key="d.id"
          class="chip"
          :class="{ active: selectedDomain === d.id }"
          @click="selectDomain(d.id)"
          :aria-pressed="selectedDomain === d.id"
        >
          {{ d.label }}
        </button>
      </div>
    </section>

    <!-- Timeline Content -->
    <section class="timeline-container" aria-live="polite">
      <div v-if="store.loading && filteredEntries.length === 0" class="loading-state">
        <div class="spinner" aria-hidden="true"></div>
        <p>Caricamento eventi in corso...</p>
      </div>

      <div v-else-if="filteredEntries.length === 0" class="empty-state">
        <div class="empty-icon" aria-hidden="true">📋</div>
        <h2>Nessun evento registrato</h2>
        <p>Gli eventi appariranno qui non appena le entità o le automazioni cambieranno stato.</p>
      </div>

      <ol v-else class="timeline-list" role="list">
        <li
          v-for="entry in filteredEntries"
          :key="entry.id"
          class="timeline-item"
          role="listitem"
        >
          <div class="timeline-time">
            <time :datetime="entry.timestamp">{{ formatTime(entry.timestamp) }}</time>
            <span class="timeline-date">{{ formatDate(entry.timestamp) }}</span>
          </div>

          <div class="timeline-marker" :class="entry.domain" aria-hidden="true"></div>

          <article class="timeline-card">
            <div class="card-header">
              <span class="entity-badge" :class="entry.domain">{{ entry.domain }}</span>
              <strong class="entity-name">{{ entry.entity_name || entry.entity_id }}</strong>
              <span class="state-pill">{{ entry.state }}</span>
            </div>

            <p class="event-message">{{ entry.message }}</p>

            <footer class="card-footer">
              <span class="context-attribution" :class="entry.context_type">
                <span class="context-icon" aria-hidden="true">{{ contextIcon(entry.context_type) }}</span>
                <span>{{ formatContext(entry) }}</span>
              </span>
            </footer>
          </article>
        </li>
      </ol>
    </section>
  </main>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useLogbookStore } from '../stores/logbook'
import type { LogbookEntry } from '../api/types'

const store = useLogbookStore()
const searchQuery = ref('')
const selectedDomain = ref('all')

const domains = [
  { id: 'all', label: 'Tutti' },
  { id: 'light', label: 'Luci' },
  { id: 'switch', label: 'Prese' },
  { id: 'climate', label: 'Clima' },
  { id: 'alarm', label: 'Allarme' },
  { id: 'person', label: 'Persone' },
  { id: 'automation', label: 'Automazioni' },
]

onMounted(async () => {
  await store.fetchEntries({ limit: 100 })
})

async function refresh() {
  await store.fetchEntries({
    domain: selectedDomain.value !== 'all' ? selectedDomain.value : undefined,
    limit: 100,
  })
}

function selectDomain(domain: string) {
  selectedDomain.value = domain
  store.fetchEntries({
    domain: domain !== 'all' ? domain : undefined,
    limit: 100,
  })
}

const filteredEntries = computed(() => {
  return store.entries.filter((entry) => {
    if (selectedDomain.value !== 'all' && entry.domain !== selectedDomain.value) {
      return false
    }
    if (!searchQuery.value) return true
    const q = searchQuery.value.toLowerCase()
    return (
      entry.entity_name?.toLowerCase().includes(q) ||
      entry.entity_id?.toLowerCase().includes(q) ||
      entry.message?.toLowerCase().includes(q) ||
      entry.context_name?.toLowerCase().includes(q)
    )
  })
})

function formatTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function formatDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleDateString([], { day: '2-digit', month: 'short' })
}

function contextIcon(type: string): string {
  switch (type) {
    case 'user':
      return '👤'
    case 'automation':
      return '⚡'
    case 'script':
      return '📜'
    default:
      return '⚙️'
  }
}

function formatContext(entry: LogbookEntry): string {
  if (entry.context_type === 'user') {
    return entry.context_name ? `Azionato da ${entry.context_name}` : 'Azionato da Utente'
  }
  if (entry.context_type === 'automation') {
    return entry.context_name ? `Innescato da "${entry.context_name}"` : 'Innescato da Automazione'
  }
  if (entry.context_type === 'script') {
    return entry.context_name ? `Eseguito da script "${entry.context_name}"` : 'Eseguito da Script'
  }
  return 'Evento di Sistema'
}
</script>

<style scoped>
.logbook-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-6, 1.5rem);
  padding: var(--space-6, 1.5rem);
  max-width: 1000px;
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

.filters-bar {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  background: var(--surface-card, rgba(30, 41, 59, 0.7));
  backdrop-filter: blur(12px);
  border: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.08));
  padding: 1rem;
  border-radius: var(--radius-lg, 12px);
}

.input-search {
  width: 100%;
  padding: 0.625rem 1rem;
  border-radius: var(--radius-md, 8px);
  border: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.12));
  background: var(--surface-elevated, rgba(15, 23, 42, 0.6));
  color: var(--text-primary, #ffffff);
  font-size: 0.95rem;
  outline-offset: 2px;
}

.input-search:focus-visible {
  outline: 2px solid var(--primary, #38bdf8);
}

.domain-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.chip {
  padding: 0.35rem 0.85rem;
  border-radius: var(--radius-full, 9999px);
  background: var(--surface-elevated, rgba(255, 255, 255, 0.05));
  border: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.1));
  color: var(--text-secondary, #cbd5e1);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s ease;
  min-height: 36px;
}

.chip:hover {
  background: var(--surface-hover, rgba(255, 255, 255, 0.1));
}

.chip.active {
  background: var(--primary, #38bdf8);
  color: #0f172a;
  border-color: var(--primary, #38bdf8);
  font-weight: 600;
}

.timeline-container {
  position: relative;
}

.timeline-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.timeline-item {
  display: grid;
  grid-template-columns: 90px 24px 1fr;
  align-items: flex-start;
  gap: 1rem;
}

.timeline-time {
  text-align: right;
  font-size: 0.85rem;
  color: var(--text-secondary, #94a3b8);
  display: flex;
  flex-direction: column;
  padding-top: 0.5rem;
}

.timeline-date {
  font-size: 0.75rem;
  opacity: 0.75;
}

.timeline-marker {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  margin-top: 0.65rem;
  background: var(--primary, #38bdf8);
  box-shadow: 0 0 8px var(--primary, #38bdf8);
  border: 2px solid #0f172a;
}

.timeline-card {
  background: var(--surface-card, rgba(30, 41, 59, 0.6));
  border: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.08));
  border-radius: var(--radius-md, 10px);
  padding: 1rem;
  backdrop-filter: blur(8px);
  transition: transform 0.15s ease, border-color 0.15s ease;
}

.timeline-card:hover {
  border-color: var(--border-hover, rgba(255, 255, 255, 0.18));
}

.card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.4rem;
}

.entity-badge {
  text-transform: uppercase;
  font-size: 0.7rem;
  font-weight: 700;
  padding: 0.15rem 0.4rem;
  border-radius: 4px;
  background: rgba(56, 189, 248, 0.15);
  color: #38bdf8;
}

.entity-name {
  font-size: 1rem;
  color: var(--text-primary, #ffffff);
}

.state-pill {
  margin-left: auto;
  font-size: 0.8rem;
  padding: 0.15rem 0.5rem;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-secondary, #cbd5e1);
}

.event-message {
  margin: 0.25rem 0 0.5rem 0;
  color: var(--text-secondary, #cbd5e1);
  font-size: 0.95rem;
}

.card-footer {
  font-size: 0.8rem;
  color: var(--text-muted, #64748b);
}

.context-attribution {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.empty-state,
.loading-state {
  text-align: center;
  padding: 3rem 1rem;
  color: var(--text-secondary, #94a3b8);
}

.empty-icon {
  font-size: 2.5rem;
  margin-bottom: 0.5rem;
}

.btn {
  padding: 0.5rem 1rem;
  border-radius: var(--radius-md, 8px);
  cursor: pointer;
  font-weight: 500;
  min-height: 44px;
}

.btn-secondary {
  background: var(--surface-elevated, rgba(255, 255, 255, 0.08));
  color: var(--text-primary, #ffffff);
  border: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.12));
}

@media (max-width: 640px) {
  .timeline-item {
    grid-template-columns: 1fr;
    gap: 0.25rem;
  }
  .timeline-time {
    text-align: left;
    flex-direction: row;
    gap: 0.5rem;
  }
  .timeline-marker {
    display: none;
  }
}
</style>
