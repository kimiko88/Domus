<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Calendar as CalendarIcon,
  Clock,
  Flame,
  Plus,
  Upload,
  CheckCircle,
  FileText
} from '@lucide/vue'
import { apiClient } from '../api/client'

const { t } = useI18n()

interface TimeSlot {
  start: string
  end: string
  value: number
}

interface Schedule {
  id: string
  name: string
  entity_key: string
  enabled: boolean
  grid: Record<string, TimeSlot[]>
}

interface CalendarEvent {
  id: string
  summary: string
  description?: string
  start_time: string
  end_time: string
  location?: string
  source: string
}

const activeTab = ref<'schedules' | 'calendar'>('schedules')

const daysOfWeek = [
  { key: 'monday', label: 'Lunedì' },
  { key: 'tuesday', label: 'Martedì' },
  { key: 'wednesday', label: 'Mercoledì' },
  { key: 'thursday', label: 'Giovedì' },
  { key: 'friday', label: 'Venerdì' },
  { key: 'saturday', label: 'Sabato' },
  { key: 'sunday', label: 'Domenica' }
]

const schedules = ref<Schedule[]>([])
const calendarEvents = ref<CalendarEvent[]>([])
const selectedSchedule = ref<Schedule | null>(null)
const icsUploadStatus = ref('')

async function handleICalUpload(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return

  const text = await file.text()
  try {
    await apiClient.post('/api/v1/calendar/import', text)
    icsUploadStatus.value = 'Calendario importato con successo!'
    setTimeout(() => icsUploadStatus.value = '', 4000)
    const e = await apiClient.get<CalendarEvent[]>('/api/v1/calendar/events')
    calendarEvents.value = e || []
  } catch {
    icsUploadStatus.value = 'Errore durante l\'importazione iCal'
  }
}

onMounted(async () => {
  try {
    const s = await apiClient.get<Schedule[]>('/api/v1/schedules')
    schedules.value = s || []
    if (schedules.value.length > 0) {
      selectedSchedule.value = schedules.value[0]
    } else {
      selectedSchedule.value = null
    }
  } catch {
    schedules.value = []
    selectedSchedule.value = null
  }

  try {
    const e = await apiClient.get<CalendarEvent[]>('/api/v1/calendar/events')
    calendarEvents.value = e || []
  } catch {
    calendarEvents.value = []
  }
})
</script>

<template>
  <div class="schedule-view-container">
    <header class="schedule-header glass-panel">
      <div class="header-title-box">
        <div class="icon-circle">
          <Clock :size="24" class="text-primary" />
        </div>
        <div>
          <h1 class="page-title">{{ t('schedules.title') }}</h1>
          <p class="page-subtitle">{{ t('schedules.subtitle') }}</p>
        </div>
      </div>

      <div class="tab-selector">
        <button
          :class="['tab-btn', { active: activeTab === 'schedules' }]"
          @click="activeTab = 'schedules'"
        >
          <Flame :size="16" />
          <span>Cronotermostato</span>
        </button>
        <button
          :class="['tab-btn', { active: activeTab === 'calendar' }]"
          @click="activeTab = 'calendar'"
        >
          <CalendarIcon :size="16" />
          <span>Calendario & iCal</span>
        </button>
      </div>
    </header>

    <!-- Tab 1: Visual Cronotermostato Planner -->
    <div v-if="activeTab === 'schedules'" class="planner-layout">
      <div v-if="selectedSchedule" class="planner-card glass-panel">
        <div class="planner-top">
          <div>
            <h2 class="schedule-title">{{ selectedSchedule.name }}</h2>
            <span class="schedule-target">Entità controllata: {{ selectedSchedule.entity_key }}</span>
          </div>
          <span class="active-badge">ATTIVO</span>
        </div>

        <!-- 7-Day Matrix -->
        <div class="weekly-matrix">
          <div v-for="day in daysOfWeek" :key="day.key" class="day-row">
            <span class="day-label">{{ day.label }}</span>
            <div class="slots-track">
              <div
                v-for="(slot, idx) in (selectedSchedule.grid[day.key] || [])"
                :key="idx"
                class="slot-block"
              >
                <span class="slot-time">{{ slot.start }} - {{ slot.end }}</span>
                <span class="slot-temp">{{ slot.value.toFixed(1) }}°C</span>
              </div>
              <div v-if="!selectedSchedule.grid[day.key] || selectedSchedule.grid[day.key].length === 0" class="no-slots">
                Nessuna fascia (temperatura di mantenimento 18.0°C)
              </div>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="planner-card glass-panel">
        <p class="empty-state" style="padding: 32px; text-align: center; color: var(--text-muted);">
          Nessuna programmazione oraria configurata nel database.
        </p>
      </div>
    </div>

    <!-- Tab 2: Calendar & iCal Sync -->
    <div v-else-if="activeTab === 'calendar'" class="calendar-layout">
      <div class="calendar-card glass-panel">
        <div class="cal-top-bar">
          <h2 class="section-title">Eventi Sincronizzati & Trigger Automazioni</h2>
          <label class="import-btn">
            <Upload :size="16" />
            <span>Importa .ics / CalDAV</span>
            <input type="file" accept=".ics" class="hidden-input" @change="handleICalUpload" />
          </label>
        </div>

        <p v-if="icsUploadStatus" class="status-msg text-success">{{ icsUploadStatus }}</p>

        <p v-if="calendarEvents.length === 0" class="empty-state" style="padding: 32px; text-align: center; color: var(--text-muted);">
          Nessun evento sincronizzato nel database.
        </p>

        <div v-else class="calendar-events-list">
          <div v-for="evt in calendarEvents" :key="evt.id" class="cal-event-card">
            <div class="cal-time-col">
              <span class="cal-date">{{ new Date(evt.start_time).toLocaleDateString('it-IT') }}</span>
              <span class="cal-hours">{{ new Date(evt.start_time).toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' }) }}</span>
            </div>

            <div class="cal-info-col">
              <h3 class="evt-summary">{{ evt.summary }}</h3>
              <p class="evt-desc">{{ evt.description }}</p>
              <span class="evt-loc">Stanza: {{ evt.location }}</span>
            </div>

            <span class="source-tag" :class="evt.source">{{ evt.source.toUpperCase() }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.schedule-view-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.schedule-header {
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
  background: rgba(245, 158, 11, 0.15);
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

.tab-selector {
  display: flex;
  gap: 8px;
  background: rgba(255, 255, 255, 0.05);
  padding: 4px;
  border-radius: 10px;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: var(--text-muted);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.tab-btn.active {
  background: var(--primary);
  color: #fff;
}

.planner-card, .calendar-card {
  padding: 24px;
  border-radius: 16px;
  background: var(--surface-card);
  border: 1px solid var(--border-color);
}

.planner-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.schedule-title {
  font-size: 1.2rem;
  font-weight: 700;
  margin: 0;
}

.schedule-target {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.active-badge {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 700;
}

.weekly-matrix {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.day-row {
  display: grid;
  grid-template-columns: 120px 1fr;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.02);
  border-radius: 10px;
}

.day-label {
  font-size: 0.9rem;
  font-weight: 600;
}

.slots-track {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.slot-block {
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  padding: 6px 12px;
  border-radius: 8px;
}

.slot-time {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.slot-temp {
  font-size: 0.85rem;
  font-weight: 700;
  color: #ef4444;
}

.no-slots {
  font-size: 0.8rem;
  color: var(--text-muted);
  font-style: italic;
}

.cal-top-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-title {
  font-size: 1.1rem;
  font-weight: 600;
  margin: 0;
}

.import-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 8px;
  background: rgba(99, 102, 241, 0.15);
  color: #818cf8;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
}

.hidden-input {
  display: none;
}

.calendar-events-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.cal-event-card {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 14px 18px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 12px;
}

.cal-time-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 90px;
}

.cal-date {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.cal-hours {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--primary);
}

.cal-info-col {
  flex: 1;
}

.evt-summary {
  font-size: 0.95rem;
  font-weight: 600;
  margin: 0;
}

.evt-desc {
  font-size: 0.8rem;
  color: var(--text-muted);
  margin: 4px 0;
}

.evt-loc {
  font-size: 0.75rem;
  color: #f59e0b;
}

.source-tag {
  font-size: 0.7rem;
  font-weight: 700;
  padding: 4px 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.1);
}
</style>
