<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
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
  MapPin,
  Flame,
  Users,
  User,
  ChevronLeft,
  ChevronRight,
  ExternalLink,
  Copy,
  RefreshCw,
  List
} from '@lucide/vue'
import { useChoresStore } from '../stores/chores'
import { useAreasStore } from '../stores/areas'
import { useFamilyStore } from '../stores/family'
import { useNotificationsStore } from '../stores/notifications'
import type { Chore, ChorePriority, ChoreRecurrence } from '../api/types'

const { t } = useI18n()
const choresStore = useChoresStore()
const areasStore = useAreasStore()
const familyStore = useFamilyStore()
const notifStore = useNotificationsStore()

const currentTab = ref<'all' | 'pending' | 'overdue' | 'completed'>('all')
const viewMode = ref<'list' | 'calendar'>('list')
const showCreateModal = ref(false)
const showGCalModal = ref(false)

// Google Calendar sync form state
const gcalSyncUrl = ref('')
const isSyncingGCal = ref(false)
const copiedFeedUrl = ref(false)

// Calendar view date navigation
const calendarDate = ref(new Date())
const calendarViewMode = ref<'month' | 'week' | 'day'>('month')

const form = ref({
  title: '',
  description: '',
  area_id: '',
  assigned_user_id: '',
  priority: 'medium' as ChorePriority,
  due_date: '',
  recurrence_type: 'none' as ChoreRecurrence
})

onMounted(async () => {
  await Promise.allSettled([
    choresStore.fetchChores(),
    areasStore.fetchAreas(),
    familyStore.fetchMembers()
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

function getAssigneeName(userId?: string): string {
  if (!userId) return 'Tutta la famiglia'
  const member = familyStore.members.find((m) => m.id === userId)
  return member ? member.username : 'Membro'
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

function openCreateForDate(dateStr?: string) {
  if (dateStr) {
    // Default to 09:00 of that date
    form.value.due_date = `${dateStr}T09:00`
  } else {
    form.value.due_date = ''
  }
  showCreateModal.value = true
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
      assigned_user_id: form.value.assigned_user_id || undefined,
      priority: form.value.priority,
      due_date: dueIso,
      recurrence_type: form.value.recurrence_type
    })
    showCreateModal.value = false
    const assignedText = form.value.assigned_user_id
      ? `notifica inviata a ${getAssigneeName(form.value.assigned_user_id)}`
      : 'notifica inviata a tutta la famiglia'
    notifStore.success('Faccenda Programmata', `Promemoria salvato con successo (${assignedText})`)

    form.value = {
      title: '',
      description: '',
      area_id: '',
      assigned_user_id: '',
      priority: 'medium',
      due_date: '',
      recurrence_type: 'none'
    }
  } catch {
    notifStore.error('Errore', 'Impossibile creare la faccenda')
  }
}

// Google Calendar Feed Copy
async function copySubscriptionLink() {
  const url = choresStore.getICalSubscriptionUrl()
  try {
    await navigator.clipboard.writeText(url)
    copiedFeedUrl.value = true
    notifStore.success('Link Copiato', 'URL iCal copiato negli appunti. Incollalo in Google Calendar.')
    setTimeout(() => {
      copiedFeedUrl.value = false
    }, 3000)
  } catch {
    notifStore.error('Errore', 'Impossibile copiare negli appunti')
  }
}

// Google Calendar Reverse Sync
async function handleSyncGoogleCalendar() {
  if (!gcalSyncUrl.value.trim()) return
  try {
    isSyncingGCal.value = true
    const res = await choresStore.syncGoogleCalendar(gcalSyncUrl.value.trim())
    notifStore.success('Sincronizzazione Riuscita', res.message || `${res.synced_count} eventi sincronizzati`)
    gcalSyncUrl.value = ''
    showGCalModal.value = false
  } catch (err: any) {
    notifStore.error('Errore Sincronizzazione', err?.message || 'Impossibile sincronizzare da Google Calendar')
  } finally {
    isSyncingGCal.value = false
  }
}

// Calendar Month Grid Computed
interface CalendarDay {
  date: Date
  dateString: string
  dayNumber: number
  isCurrentMonth: boolean
  isToday: boolean
  chores: Chore[]
}

const calendarDays = computed<CalendarDay[]>(() => {
  const year = calendarDate.value.getFullYear()
  const month = calendarDate.value.getMonth()

  // First day of current month
  const firstDay = new Date(year, month, 1)
  // Day of week: 0=Sun, 1=Mon, ..., 6=Sat. European style starts on Monday (1)
  let startDayOfWeek = firstDay.getDay()
  if (startDayOfWeek === 0) startDayOfWeek = 7 // Sunday -> 7

  const days: CalendarDay[] = []
  const today = new Date()
  const todayStr = today.toISOString().split('T')[0]

  // Leading days from previous month
  const prevMonthLastDay = new Date(year, month, 0).getDate()
  for (let i = startDayOfWeek - 1; i > 0; i--) {
    const d = new Date(year, month - 1, prevMonthLastDay - i + 1)
    const dStr = d.toISOString().split('T')[0]
    days.push({
      date: d,
      dateString: dStr,
      dayNumber: d.getDate(),
      isCurrentMonth: false,
      isToday: dStr === todayStr,
      chores: getChoresForDate(dStr)
    })
  }

  // Days of current month
  const lastDay = new Date(year, month + 1, 0).getDate()
  for (let i = 1; i <= lastDay; i++) {
    const d = new Date(year, month, i)
    const dStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(i).padStart(2, '0')}`
    days.push({
      date: d,
      dateString: dStr,
      dayNumber: i,
      isCurrentMonth: true,
      isToday: dStr === todayStr,
      chores: getChoresForDate(dStr)
    })
  }

  // Trailing days to fill 35 or 42 grid cells
  const remainingCells = (7 - (days.length % 7)) % 7
  for (let i = 1; i <= remainingCells; i++) {
    const d = new Date(year, month + 1, i)
    const dStr = d.toISOString().split('T')[0]
    days.push({
      date: d,
      dateString: dStr,
      dayNumber: i,
      isCurrentMonth: false,
      isToday: dStr === todayStr,
      chores: getChoresForDate(dStr)
    })
  }

  return days
})

function getChoresForDate(dateStr: string): Chore[] {
  return choresStore.chores.filter((c) => {
    if (!c.due_date) return false
    return c.due_date.startsWith(dateStr)
  })
}

function getStartOfWeek(date: Date): Date {
  const d = new Date(date)
  const day = d.getDay()
  const diff = d.getDate() - day + (day === 0 ? -6 : 1)
  d.setDate(diff)
  d.setHours(0, 0, 0, 0)
  return d
}

function formatTime(dateStr?: string): string {
  if (!dateStr) return ''
  try {
    const d = new Date(dateStr)
    return d.toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' })
  } catch {
    return ''
  }
}

interface WeekDay {
  date: Date
  dateString: string
  dayNumber: number
  dayName: string
  fullDayName: string
  isToday: boolean
  chores: Chore[]
}

const weekDays = computed<WeekDay[]>(() => {
  const start = getStartOfWeek(calendarDate.value)
  const days: WeekDay[] = []
  const todayStr = new Date().toISOString().split('T')[0]
  for (let i = 0; i < 7; i++) {
    const d = new Date(start)
    d.setDate(start.getDate() + i)
    const dStr = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
    days.push({
      date: d,
      dateString: dStr,
      dayNumber: d.getDate(),
      dayName: d.toLocaleDateString('it-IT', { weekday: 'short' }),
      fullDayName: d.toLocaleDateString('it-IT', { weekday: 'long' }),
      isToday: dStr === todayStr,
      chores: getChoresForDate(dStr)
    })
  }
  return days
})

const selectedDayData = computed(() => {
  const d = calendarDate.value
  const dStr = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  const todayStr = new Date().toISOString().split('T')[0]
  return {
    date: d,
    dateString: dStr,
    dayNumber: d.getDate(),
    dayName: d.toLocaleDateString('it-IT', { weekday: 'long' }),
    monthName: d.toLocaleDateString('it-IT', { month: 'long', year: 'numeric' }),
    isToday: dStr === todayStr,
    chores: getChoresForDate(dStr)
  }
})

const currentPeriodLabel = computed(() => {
  const d = calendarDate.value
  if (calendarViewMode.value === 'month') {
    return d.toLocaleDateString('it-IT', { month: 'long', year: 'numeric' })
  } else if (calendarViewMode.value === 'week') {
    const start = getStartOfWeek(d)
    const end = new Date(start)
    end.setDate(start.getDate() + 6)
    if (start.getMonth() === end.getMonth()) {
      return `${start.getDate()} - ${end.getDate()} ${start.toLocaleDateString('it-IT', { month: 'short', year: 'numeric' })}`
    } else {
      return `${start.getDate()} ${start.toLocaleDateString('it-IT', { month: 'short' })} - ${end.getDate()} ${end.toLocaleDateString('it-IT', { month: 'short', year: 'numeric' })}`
    }
  } else {
    // Day view
    return d.toLocaleDateString('it-IT', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
  }
})

const currentMonthLabel = computed(() => currentPeriodLabel.value)

function prevPeriod() {
  const d = new Date(calendarDate.value)
  if (calendarViewMode.value === 'month') {
    d.setMonth(d.getMonth() - 1)
  } else if (calendarViewMode.value === 'week') {
    d.setDate(d.getDate() - 7)
  } else {
    d.setDate(d.getDate() - 1)
  }
  calendarDate.value = d
}

function nextPeriod() {
  const d = new Date(calendarDate.value)
  if (calendarViewMode.value === 'month') {
    d.setMonth(d.getMonth() + 1)
  } else if (calendarViewMode.value === 'week') {
    d.setDate(d.getDate() + 7)
  } else {
    d.setDate(d.getDate() + 1)
  }
  calendarDate.value = d
}

function prevMonth() {
  prevPeriod()
}

function nextMonth() {
  nextPeriod()
}

function goToToday() {
  calendarDate.value = new Date()
}

function selectDayView(dateStr: string) {
  calendarDate.value = new Date(dateStr)
  calendarViewMode.value = 'day'
}
</script>

<template>
  <div class="chores-view">
    <!-- Header -->
    <header class="chores-header">
      <div>
        <div class="title-with-badge">
          <h1>{{ t('chores.title') }}</h1>
          <span class="badge-pill">{{ choresStore.pendingChores.length }} Da fare</span>
        </div>
        <p class="subtitle">{{ t('chores.subtitle') }}</p>
      </div>

      <div class="header-action-group">
        <!-- View Mode Switcher -->
        <div class="view-mode-toggle" role="tablist" aria-label="Visualizzazione Faccende">
          <button
            class="toggle-btn"
            :class="{ active: viewMode === 'list' }"
            @click="viewMode = 'list'"
            aria-label="Vista Elenco"
          >
            <List class="btn-icon" />
            <span>Elenco</span>
          </button>
          <button
            class="toggle-btn"
            :class="{ active: viewMode === 'calendar' }"
            @click="viewMode = 'calendar'"
            aria-label="Vista Calendario"
          >
            <Calendar class="btn-icon" />
            <span>Calendario</span>
          </button>
        </div>

        <!-- Google Calendar Connect Button -->
        <button class="btn-secondary" @click="showGCalModal = true" aria-label="Collega Google Calendar">
          <ExternalLink class="btn-icon" />
          <span>Google Calendar</span>
        </button>

        <!-- New Chore Button -->
        <button class="btn-primary" @click="openCreateForDate()" aria-label="Nuova Faccenda">
          <Plus class="btn-icon" />
          <span>{{ t('chores.newChore') }}</span>
        </button>
      </div>
    </header>

    <!-- Stat Cards Overview -->
    <div class="stats-grid">
      <div class="stat-card urgent" @click="currentTab = 'overdue'; viewMode = 'list'">
        <div class="stat-icon-wrap urgent">
          <AlertTriangle class="stat-icon" />
        </div>
        <div class="stat-data">
          <span class="stat-value">{{ choresStore.overdueChores.length }}</span>
          <span class="stat-label">In Scadenza / Scadute</span>
        </div>
      </div>

      <div class="stat-card pending" @click="currentTab = 'pending'; viewMode = 'list'">
        <div class="stat-icon-wrap pending">
          <Clock class="stat-icon" />
        </div>
        <div class="stat-data">
          <span class="stat-value">{{ choresStore.pendingChores.length }}</span>
          <span class="stat-label">Compiti Attivi</span>
        </div>
      </div>

      <div class="stat-card completed" @click="currentTab = 'completed'; viewMode = 'list'">
        <div class="stat-icon-wrap completed">
          <CheckCircle2 class="stat-icon" />
        </div>
        <div class="stat-data">
          <span class="stat-value">{{ choresStore.completedChores.length }}</span>
          <span class="stat-label">{{ t('chores.completed') }}</span>
        </div>
      </div>
    </div>

    <!-- LIST VIEW -->
    <div v-if="viewMode === 'list'" class="list-view-section">
      <!-- Filter Tabs -->
      <div class="filter-tabs-bar">
        <div class="tabs-list">
          <button
            class="tab-btn"
            :class="{ active: currentTab === 'all' }"
            @click="currentTab = 'all'"
          >
            {{ t('chores.all') }} ({{ choresStore.chores.length }})
          </button>
          <button
            class="tab-btn"
            :class="{ active: currentTab === 'pending' }"
            @click="currentTab = 'pending'"
          >
            {{ t('chores.pending') }}
          </button>
          <button
            class="tab-btn"
            :class="{ active: currentTab === 'overdue' }"
            @click="currentTab = 'overdue'"
          >
            {{ t('chores.overdue') }} ({{ choresStore.overdueChores.length }})
          </button>
          <button
            class="tab-btn"
            :class="{ active: currentTab === 'completed' }"
            @click="currentTab = 'completed'"
          >
            {{ t('chores.completed') }} ({{ choresStore.completedChores.length }})
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
            :aria-label="`Segna ${chore.title} come completata`"
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

              <!-- Assignee Pill -->
              <span class="assignee-pill" :class="{ 'family-all': !chore.assigned_user_id }">
                <Users v-if="!chore.assigned_user_id" class="meta-icon" />
                <User v-else class="meta-icon" />
                <span>{{ getAssigneeName(chore.assigned_user_id) }}</span>
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
          <button class="delete-btn" title="Elimina faccenda" aria-label="Elimina faccenda" @click="handleDelete(chore.id)">
            <Trash2 class="del-icon" />
          </button>
        </div>

        <!-- Empty State -->
        <div v-if="filteredChores.length === 0" class="empty-state">
          <CheckSquare class="empty-icon" />
          <h3>Nessuna faccenda in questa vista</h3>
          <p>Aggiungi una nuova faccenda domestica o seleziona un'altra categoria.</p>
          <button class="btn-primary" @click="openCreateForDate()">
            <Plus class="btn-icon" />
            Crea Faccenda
          </button>
        </div>
      </div>
    </div>

    <!-- CALENDAR VIEW -->
    <div v-else class="calendar-view-section">
      <!-- Calendar Navigation Bar -->
      <div class="calendar-nav-bar">
        <div class="cal-nav-left">
          <button class="nav-arrow-btn" @click="prevPeriod" :title="calendarViewMode === 'day' ? 'Giorno precedente' : calendarViewMode === 'week' ? 'Settimana precedente' : 'Mese precedente'" aria-label="Periodo precedente">
            <ChevronLeft class="btn-icon" />
          </button>
          <h2 class="month-heading">{{ currentPeriodLabel }}</h2>
          <button class="nav-arrow-btn" @click="nextPeriod" :title="calendarViewMode === 'day' ? 'Giorno successivo' : calendarViewMode === 'week' ? 'Settimana successiva' : 'Mese successivo'" aria-label="Periodo successivo">
            <ChevronRight class="btn-icon" />
          </button>
          <button class="btn-today" @click="goToToday">Oggi</button>
        </div>

        <!-- View Mode Switcher: Giorno / Settimana / Mese -->
        <div class="cal-view-selector" role="tablist" aria-label="Modalità Vista Calendario">
          <button
            type="button"
            class="cal-view-btn"
            :class="{ active: calendarViewMode === 'day' }"
            @click="calendarViewMode = 'day'"
            role="tab"
            :aria-selected="calendarViewMode === 'day'"
          >
            Giorno
          </button>
          <button
            type="button"
            class="cal-view-btn"
            :class="{ active: calendarViewMode === 'week' }"
            @click="calendarViewMode = 'week'"
            role="tab"
            :aria-selected="calendarViewMode === 'week'"
          >
            Settimana
          </button>
          <button
            type="button"
            class="cal-view-btn"
            :class="{ active: calendarViewMode === 'month' }"
            @click="calendarViewMode = 'month'"
            role="tab"
            :aria-selected="calendarViewMode === 'month'"
          >
            Mese
          </button>
        </div>

        <div class="cal-legend">
          <span class="legend-item"><span class="dot urgent"></span> Urgente</span>
          <span class="legend-item"><span class="dot high"></span> Alta</span>
          <span class="legend-item"><span class="dot medium"></span> Media</span>
          <span class="legend-item"><span class="dot low"></span> Bassa</span>
        </div>
      </div>

      <!-- MONTH VIEW -->
      <div v-if="calendarViewMode === 'month'" class="calendar-grid-container">
        <!-- Day Names Row -->
        <div class="calendar-weekdays">
          <span>Lun</span>
          <span>Mar</span>
          <span>Mer</span>
          <span>Gio</span>
          <span>Ven</span>
          <span>Sab</span>
          <span>Dom</span>
        </div>

        <!-- Day Cells -->
        <div class="calendar-cells-grid">
          <div
            v-for="(day, idx) in calendarDays"
            :key="idx"
            class="cal-day-cell"
            :class="{
              'is-current-month': day.isCurrentMonth,
              'is-today': day.isToday
            }"
            @click="openCreateForDate(day.dateString)"
          >
            <div class="cell-header">
              <span
                class="cell-day-num"
                :class="{ 'today-num': day.isToday }"
                @click.stop="selectDayView(day.dateString)"
                title="Apri vista giorno"
              >
                {{ day.dayNumber }}
              </span>
              <button
                class="cell-add-btn"
                title="Aggiungi faccenda a questo giorno"
                @click.stop="openCreateForDate(day.dateString)"
              >
                <Plus :size="12" />
              </button>
            </div>

            <!-- Chore Badges for Day -->
            <div class="cell-chores-list">
              <div
                v-for="chore in day.chores"
                :key="chore.id"
                class="cal-chore-badge"
                :class="[
                  chore.priority,
                  { 'is-completed': chore.status === 'completed' }
                ]"
                @click.stop="handleComplete(chore)"
                :title="`${chore.title} (${getAssigneeName(chore.assigned_user_id)}) - Clicca per completare`"
              >
                <span class="priority-dot" :class="chore.priority"></span>
                <span class="badge-title" :class="{ strike: chore.status === 'completed' }">{{ chore.title }}</span>
                <span class="badge-assignee" :class="{ 'family-all': !chore.assigned_user_id }">
                  {{ chore.assigned_user_id ? getAssigneeName(chore.assigned_user_id).slice(0, 3) : 'Fam' }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- WEEK VIEW -->
      <div v-else-if="calendarViewMode === 'week'" class="week-grid-container">
        <div
          v-for="day in weekDays"
          :key="day.dateString"
          class="week-day-col"
          :class="{ 'is-today': day.isToday }"
        >
          <div class="week-col-header" @click="selectDayView(day.dateString)" title="Clicca per aprire la vista giorno">
            <div class="col-header-text">
              <span class="week-day-name">{{ day.dayName }}</span>
              <span class="week-day-num" :class="{ 'today-num': day.isToday }">{{ day.dayNumber }}</span>
            </div>
            <button
              class="col-add-btn"
              title="Aggiungi faccenda"
              @click.stop="openCreateForDate(day.dateString)"
              aria-label="Aggiungi faccenda a questo giorno"
            >
              <Plus :size="13" />
            </button>
          </div>

          <div class="week-col-chores">
            <div
              v-for="chore in day.chores"
              :key="chore.id"
              class="week-chore-card"
              :class="[chore.priority, { 'is-completed': chore.status === 'completed' }]"
              @click="handleComplete(chore)"
              :title="`${chore.title} - Clicca per completare`"
            >
              <div class="chore-card-top">
                <span class="priority-dot" :class="chore.priority"></span>
                <span class="card-time" v-if="chore.due_date && chore.due_date.includes('T')">
                  {{ formatTime(chore.due_date) }}
                </span>
                <span v-else class="card-time text-dim">Tutto il dì</span>
              </div>
              <span class="card-title" :class="{ strike: chore.status === 'completed' }">
                {{ chore.title }}
              </span>
              <div class="card-footer">
                <span v-if="getAreaName(chore.area_id)" class="area-chip">
                  {{ getAreaName(chore.area_id) }}
                </span>
                <span class="assignee-chip" :class="{ 'family-all': !chore.assigned_user_id }">
                  {{ chore.assigned_user_id ? getAssigneeName(chore.assigned_user_id) : 'Famiglia' }}
                </span>
              </div>
            </div>

            <button
              v-if="day.chores.length === 0"
              class="col-empty-btn"
              @click="openCreateForDate(day.dateString)"
            >
              <Plus :size="12" />
              <span>Aggiungi</span>
            </button>
          </div>
        </div>
      </div>

      <!-- DAY VIEW -->
      <div v-else-if="calendarViewMode === 'day'" class="day-view-container">
        <div class="day-view-hero">
          <div class="day-hero-info">
            <div class="day-hero-badge" v-if="selectedDayData.isToday">Oggi</div>
            <h3 class="day-hero-title">{{ selectedDayData.dayName }}, {{ selectedDayData.dayNumber }} {{ selectedDayData.monthName }}</h3>
            <p class="day-hero-sub">
              {{ selectedDayData.chores.length }} {{ selectedDayData.chores.length === 1 ? 'faccenda programmata' : 'faccende programmate' }} •
              <span class="text-success">{{ selectedDayData.chores.filter(c => c.status === 'completed').length }} completate</span>
            </p>
          </div>

          <button class="btn-primary" @click="openCreateForDate(selectedDayData.dateString)">
            <Plus :size="16" />
            <span>Nuova Faccenda</span>
          </button>
        </div>

        <div v-if="selectedDayData.chores.length === 0" class="day-empty-box">
          <Calendar :size="42" class="empty-icon" />
          <h4>Nessuna faccenda per questa data</h4>
          <p>Tutti i compiti per questa giornata sono stati completati o non ce ne sono di programmati.</p>
          <button class="btn-secondary" @click="openCreateForDate(selectedDayData.dateString)">
            <Plus :size="14" />
            <span>Aggiungi promemoria per questa data</span>
          </button>
        </div>

        <div v-else class="day-chores-list">
          <div
            v-for="chore in selectedDayData.chores"
            :key="chore.id"
            class="day-chore-card"
            :class="[chore.priority, { 'is-completed': chore.status === 'completed' }]"
          >
            <div class="day-chore-main" @click="handleComplete(chore)">
              <button
                type="button"
                class="checkbox-btn"
                :aria-label="chore.status === 'completed' ? 'Segna come da fare' : 'Segna come completata'"
              >
                <CheckCircle2 v-if="chore.status === 'completed'" class="check-icon done" />
                <div v-else class="check-circle-empty"></div>
              </button>

              <div class="chore-info-block">
                <div class="title-and-pills">
                  <span class="day-chore-title" :class="{ strike: chore.status === 'completed' }">
                    {{ chore.title }}
                  </span>
                  <span class="priority-pill" :class="chore.priority">
                    {{ t(`chores.priorities.${chore.priority}`) }}
                  </span>
                  <span v-if="chore.recurrence_type && chore.recurrence_type !== 'none'" class="recurrence-pill">
                    <Repeat :size="12" />
                    <span>{{ t(`chores.recurrences.${chore.recurrence_type}`) }}</span>
                  </span>
                </div>

                <p v-if="chore.description" class="chore-desc-text">{{ chore.description }}</p>

                <div class="chore-meta-row">
                  <span v-if="chore.due_date && chore.due_date.includes('T')" class="meta-tag">
                    <Clock :size="13" />
                    <span>{{ formatTime(chore.due_date) }}</span>
                  </span>
                  <span v-if="getAreaName(chore.area_id)" class="meta-tag">
                    <MapPin :size="13" />
                    <span>{{ getAreaName(chore.area_id) }}</span>
                  </span>
                  <span class="meta-tag assignee" :class="{ 'family-all': !chore.assigned_user_id }">
                    <User :size="13" />
                    <span>{{ chore.assigned_user_id ? getAssigneeName(chore.assigned_user_id) : 'Tutta la famiglia' }}</span>
                  </span>
                </div>
              </div>
            </div>

            <div class="day-chore-actions">
              <button
                type="button"
                class="btn-delete-chore"
                @click="handleDelete(chore.id)"
                title="Elimina faccenda"
                aria-label="Elimina faccenda"
              >
                <Trash2 :size="16" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Create Chore Modal -->
    <div v-if="showCreateModal" class="modal-backdrop" @click="showCreateModal = false">
      <div class="modal-card" @click.stop role="dialog" aria-modal="true" aria-labelledby="modal-title">
        <div class="modal-header">
          <h2 id="modal-title">Nuova Faccenda Domestica</h2>
          <button class="modal-close-btn" @click="showCreateModal = false" aria-label="Chiudi finestra">
            <X class="icon-close" />
          </button>
        </div>

        <form @submit.prevent="handleCreateChore" class="modal-body">
          <div class="form-group">
            <label for="chore-title">Titolo del Compito *</label>
            <input
              id="chore-title"
              v-model="form.title"
              type="text"
              placeholder="es. Pulire filtro robot, annaffiare piante, lavare pavimenti"
              required
            />
          </div>

          <div class="form-group">
            <label for="chore-desc">Descrizione / Dettagli</label>
            <textarea
              id="chore-desc"
              v-model="form.description"
              rows="2"
              placeholder="Istruzioni o dettagli per il compito"
            ></textarea>
          </div>

          <!-- Assignment & Notification Rule -->
          <div class="form-group">
            <label for="chore-assignee">Assegna Persona & Notifica *</label>
            <select id="chore-assignee" v-model="form.assigned_user_id">
              <option value="">👨‍👩‍👧‍👦 Tutta la famiglia (Notifica broadcast a tutti i membri)</option>
              <option v-for="member in familyStore.members" :key="member.id" :value="member.id">
                👤 {{ member.username }} ({{ member.role }})
              </option>
            </select>
            <span class="field-hint">
              {{
                form.assigned_user_id
                  ? `La notifica push e il promemoria di scadenza saranno inviati a ${getAssigneeName(form.assigned_user_id)}.`
                  : 'Nessun membro specifico: la notifica di scadenza e promemoria arriverà a tutta la famiglia.'
              }}
            </span>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label for="chore-area">Stanza / Area</label>
              <select id="chore-area" v-model="form.area_id">
                <option value="">Tutta la casa / Esterno</option>
                <option v-for="area in areasStore.areas" :key="area.id" :value="area.id">
                  {{ area.name }}
                </option>
              </select>
            </div>

            <div class="form-group">
              <label for="chore-priority">Priorità</label>
              <select id="chore-priority" v-model="form.priority">
                <option value="low">Bassa</option>
                <option value="medium">Media</option>
                <option value="high">Alta</option>
                <option value="urgent">Urgente</option>
              </select>
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label for="chore-due">Data & Ora Scadenza</label>
              <input id="chore-due" v-model="form.due_date" type="datetime-local" />
            </div>

            <div class="form-group">
              <label for="chore-recurrence">Ricorrenza</label>
              <select id="chore-recurrence" v-model="form.recurrence_type">
                <option value="none">Nessuna (Una tantum)</option>
                <option value="daily">Ogni Giorno</option>
                <option value="weekly">Ogni Settimana</option>
                <option value="monthly">Ogni Mese</option>
              </select>
            </div>
          </div>

          <div class="modal-footer">
            <button type="button" class="btn-cancel" @click="showCreateModal = false">Annulla</button>
            <button type="submit" class="btn-primary">Salva & Programma</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Google Calendar Integration Modal -->
    <div v-if="showGCalModal" class="modal-backdrop" @click="showGCalModal = false">
      <div class="modal-card gcal-modal" @click.stop role="dialog" aria-modal="true" aria-labelledby="gcal-title">
        <div class="modal-header">
          <div class="gcal-title-box">
            <Calendar class="gcal-icon" />
            <h2 id="gcal-title">Collega Google Calendar</h2>
          </div>
          <button class="modal-close-btn" @click="showGCalModal = false" aria-label="Chiudi finestra">
            <X class="icon-close" />
          </button>
        </div>

        <div class="modal-body">
          <!-- Option A: Subscribe from Google Calendar (Export) -->
          <div class="gcal-section">
            <div class="section-badge">OPZIONE 1: SOTTOSCRIVI DA GOOGLE CALENDAR</div>
            <h3>Sincronizza le faccende Domus nel tuo Google Calendar</h3>
            <p class="gcal-desc">
              Aggiungi questo feed iCal standard al tuo account Google, Apple Calendar o Outlook per visualizzare sempre tutte le faccende domestiche aggiornate:
            </p>

            <div class="feed-copy-box">
              <input
                type="text"
                readonly
                :value="choresStore.getICalSubscriptionUrl()"
                class="feed-input"
              />
              <button class="btn-copy" @click="copySubscriptionLink">
                <component :is="copiedFeedUrl ? CheckCircle2 : Copy" class="btn-icon" />
                <span>{{ copiedFeedUrl ? 'Copiato!' : 'Copia Link' }}</span>
              </button>
            </div>

            <div class="gcal-steps">
              <div class="step-num">1</div>
              <div class="step-text">Apri Google Calendar su browser o smartphone.</div>
              <div class="step-num">2</div>
              <div class="step-text">Nella barra laterale sinistra, accanto ad <strong>"Altri calendari"</strong>, fai clic su <strong>+ &gt; Da URL</strong>.</div>
              <div class="step-num">3</div>
              <div class="step-text">Incolla l'indirizzo copiato e conferma con <strong>"Aggiungi calendario"</strong>.</div>
            </div>
          </div>

          <div class="divider"></div>

          <!-- Option B: Sync / Import from Google Calendar (Import) -->
          <div class="gcal-section">
            <div class="section-badge">OPZIONE 2: IMPORTA DA GOOGLE CALENDAR</div>
            <h3>Importa eventi / compiti da Google Calendar</h3>
            <p class="gcal-desc">
              Incolla l'indirizzo iCal segreto o pubblico del tuo Google Calendar per importare i tuoi impegni direttamente come faccende domestiche in Domus:
            </p>

            <form @submit.prevent="handleSyncGoogleCalendar" class="sync-form">
              <div class="form-group">
                <label for="gcal-url">Indirizzo iCal di Google Calendar (.ics)</label>
                <input
                  id="gcal-url"
                  v-model="gcalSyncUrl"
                  type="url"
                  placeholder="https://calendar.google.com/calendar/ical/.../basic.ics"
                  required
                />
              </div>

              <button type="submit" class="btn-primary" :disabled="isSyncingGCal">
                <RefreshCw class="btn-icon" :class="{ 'spin-icon': isSyncingGCal }" />
                <span>{{ isSyncingGCal ? 'Sincronizzazione in corso...' : 'Sincronizza Ora' }}</span>
              </button>
            </form>
          </div>
        </div>
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
  max-width: 1400px;
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
  color: var(--text-main);
}

.badge-pill {
  font-size: 0.75rem;
  padding: 0.2rem 0.6rem;
  background: rgba(59, 130, 246, 0.2);
  color: var(--primary);
  border-radius: 9999px;
  font-weight: 600;
}

.subtitle {
  color: var(--text-muted);
  font-size: 0.875rem;
  margin-top: 0.25rem;
}

.header-action-group {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.view-mode-toggle {
  display: flex;
  background: var(--bg-surface, #1e293b);
  border: 1px solid var(--border-card, rgba(255, 255, 255, 0.12));
  border-radius: 0.5rem;
  padding: 0.2rem;
}

.toggle-btn {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.4rem 0.85rem;
  border: none;
  background: transparent;
  color: var(--text-muted, #94a3b8);
  font-size: 0.8rem;
  font-weight: 600;
  border-radius: 0.35rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.toggle-btn.active {
  background: var(--primary, #3b82f6);
  color: #fff;
}

.btn-secondary {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #cbd5e1;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-secondary:hover {
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: var(--primary, #3b82f6);
  border: none;
  color: #fff;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary:hover {
  background: var(--primary-hover, #2563eb);
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
  background: var(--bg-glass-card, rgba(15, 23, 42, 0.7));
  border: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.1));
  border-radius: 0.75rem;
  padding: 1.25rem;
  cursor: pointer;
  transition: transform 0.2s ease, background 0.2s, border-color 0.2s;
}

.stat-card:hover {
  transform: translateY(-2px);
  background: var(--bg-card-hover, rgba(30, 41, 59, 0.8));
  border-color: var(--border-card, rgba(255, 255, 255, 0.2));
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
  color: var(--text-main, #f8fafc);
}

.stat-label {
  font-size: 0.8rem;
  color: var(--text-muted, #94a3b8);
}

/* Filter Tabs */
.filter-tabs-bar {
  border-bottom: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.1));
  padding-bottom: 0.5rem;
  margin-bottom: 1rem;
}

.tabs-list {
  display: flex;
  gap: 0.5rem;
}

.tab-btn {
  padding: 0.5rem 1rem;
  background: none;
  border: none;
  color: var(--text-muted, #94a3b8);
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
  transition: all 0.2s;
}

.tab-btn:hover {
  color: var(--text-main, #f8fafc);
  background: var(--bg-surface-hover, rgba(255, 255, 255, 0.05));
}

.tab-btn.active {
  color: var(--primary, #3b82f6);
  background: var(--primary-light, rgba(59, 130, 246, 0.15));
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
  background: var(--bg-glass-card, rgba(15, 23, 42, 0.7));
  border: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.1));
  border-radius: 0.75rem;
  padding: 1rem 1.25rem;
  transition: all 0.2s ease;
}

.chore-card:hover {
  background: var(--bg-card-hover, rgba(30, 41, 59, 0.8));
  border-color: var(--border-card, rgba(255, 255, 255, 0.2));
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
  border: 2px solid var(--border-card, rgba(255, 255, 255, 0.3));
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
  flex-wrap: wrap;
}

.chore-title {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
}

.chore-title.strike {
  text-decoration: line-through;
  color: var(--text-dim, #64748b);
}

.chore-desc {
  margin: 0;
  font-size: 0.8rem;
  color: var(--text-muted, #94a3b8);
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

.assignee-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.75rem;
  padding: 0.15rem 0.6rem;
  border-radius: 12px;
  background: rgba(147, 51, 234, 0.15);
  border: 1px solid rgba(147, 51, 234, 0.3);
  color: #c084fc;
  font-weight: 600;
}

.assignee-pill.family-all {
  background: rgba(59, 130, 246, 0.15);
  border-color: rgba(59, 130, 246, 0.3);
  color: #60a5fa;
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
  background: var(--bg-surface-hover);
  border: 1px dashed var(--border-color);
  border-radius: 0.75rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
}

.empty-icon {
  width: 3rem;
  height: 3rem;
  color: var(--text-muted);
}

.empty-state h3 {
  margin: 0;
  color: var(--text-main);
}

.empty-state p {
  color: var(--text-muted);
  font-size: 0.875rem;
  margin: 0;
}

/* CALENDAR VIEW STYLING */
.calendar-view-section {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.calendar-nav-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.85rem 1.25rem;
  background: var(--bg-glass-card, var(--bg-card));
  border: 1px solid var(--border-card);
  border-radius: var(--radius-lg, 1rem);
  box-shadow: var(--shadow-sm);
  backdrop-filter: var(--blur-card, blur(12px));
  flex-wrap: wrap;
  gap: 1rem;
}

.cal-nav-left {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

/* Arrow buttons: clearly visible in Light, Dark, and OLED themes */
.nav-arrow-btn {
  background: var(--bg-surface-hover, rgba(255, 255, 255, 0.08));
  border: 1px solid var(--border-card, rgba(255, 255, 255, 0.14));
  color: var(--text-main, #f8fafc);
  border-radius: 0.5rem;
  width: 2.25rem;
  height: 2.25rem;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast, 0.15s ease);
}

.nav-arrow-btn:hover {
  background: var(--primary-light, rgba(14, 165, 233, 0.15));
  border-color: var(--primary, #0ea5e9);
  color: var(--primary, #0ea5e9);
  transform: translateY(-1px);
}

.nav-arrow-btn:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

.nav-arrow-btn .btn-icon {
  width: 1.2rem;
  height: 1.2rem;
  color: currentColor;
}

.month-heading {
  margin: 0 0.4rem;
  font-size: 1.25rem;
  font-weight: 700;
  text-transform: capitalize;
  color: var(--text-main);
  min-width: 180px;
}

.btn-today {
  padding: 0.4rem 0.85rem;
  background: var(--primary-light, rgba(14, 165, 233, 0.15));
  border: 1px solid var(--primary, #0ea5e9);
  color: var(--primary, #0ea5e9);
  border-radius: 0.5rem;
  font-size: 0.825rem;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast, 0.15s ease);
}

.btn-today:hover {
  background: var(--primary, #0ea5e9);
  color: #fff;
}

.btn-today:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

/* View Selector Pill Tabs: Giorno / Settimana / Mese */
.cal-view-selector {
  display: flex;
  background: var(--bg-surface, #1e293b);
  border: 1px solid var(--border-card);
  border-radius: 0.55rem;
  padding: 0.25rem;
  gap: 0.2rem;
}

.cal-view-btn {
  padding: 0.4rem 0.85rem;
  border: none;
  background: transparent;
  color: var(--text-muted, #94a3b8);
  font-size: 0.825rem;
  font-weight: 600;
  border-radius: 0.4rem;
  cursor: pointer;
  transition: all var(--transition-fast, 0.15s ease);
}

.cal-view-btn:hover {
  color: var(--text-main);
  background: var(--bg-surface-hover);
}

.cal-view-btn.active {
  background: var(--primary, #0ea5e9);
  color: #fff;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
}

.cal-view-btn:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

.cal-legend {
  display: flex;
  align-items: center;
  gap: 1rem;
  font-size: 0.78rem;
  color: var(--text-muted);
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.dot.urgent { background: #ef4444; }
.dot.high { background: #fb923c; }
.dot.medium { background: #3b82f6; }
.dot.low { background: #94a3b8; }

/* MONTH VIEW */
.calendar-grid-container {
  display: flex;
  flex-direction: column;
  background: var(--bg-glass-card, var(--bg-card));
  border: 1px solid var(--border-card);
  border-radius: var(--radius-lg, 1rem);
  box-shadow: var(--shadow-sm);
  overflow: hidden;
}

.calendar-weekdays {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  background: var(--bg-surface-hover);
  border-bottom: 1px solid var(--border-card);
  text-align: center;
  font-weight: 700;
  font-size: 0.82rem;
  color: var(--text-muted);
  padding: 0.7rem 0;
}

.calendar-cells-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  grid-auto-rows: minmax(115px, auto);
}

.cal-day-cell {
  border-right: 1px solid var(--border-subtle);
  border-bottom: 1px solid var(--border-subtle);
  padding: 0.55rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  background: var(--bg-card);
  cursor: pointer;
  transition: background var(--transition-fast, 0.15s ease);
}

.cal-day-cell:nth-child(7n) {
  border-right: none;
}

.cal-day-cell:hover {
  background: var(--bg-surface-hover);
}

.cal-day-cell:not(.is-current-month) {
  opacity: 0.42;
  background: var(--bg-secondary);
}

.cal-day-cell.is-today {
  background: var(--primary-light);
}

.cell-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.cell-day-num {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-main);
  width: 1.6rem;
  height: 1.6rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  cursor: pointer;
  transition: all 0.15s ease;
}

.cell-day-num:hover {
  background: var(--bg-surface-hover);
  color: var(--primary);
}

.cell-day-num.today-num {
  background: var(--primary, #0ea5e9);
  color: #fff;
  font-weight: 700;
}

.cell-add-btn {
  opacity: 0;
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-subtle);
  color: var(--text-main);
  border-radius: 4px;
  width: 1.35rem;
  height: 1.35rem;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: opacity 0.2s ease, background 0.2s ease;
}

.cal-day-cell:hover .cell-add-btn {
  opacity: 1;
}

.cell-add-btn:hover {
  background: var(--primary);
  color: #fff;
  border-color: var(--primary);
}

.cell-chores-list {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  overflow-y: auto;
  max-height: 90px;
}

.cal-chore-badge {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.45rem;
  border-radius: 5px;
  font-size: 0.72rem;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  cursor: pointer;
  transition: all 0.15s ease;
}

.cal-chore-badge:hover {
  transform: translateX(2px);
  border-color: var(--primary);
}

.cal-chore-badge.is-completed {
  opacity: 0.55;
}

.priority-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.priority-dot.urgent { background: #ef4444; }
.priority-dot.high { background: #fb923c; }
.priority-dot.medium { background: #3b82f6; }
.priority-dot.low { background: #94a3b8; }

.badge-title {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text-main);
  font-weight: 500;
}

.badge-title.strike {
  text-decoration: line-through;
}

.badge-assignee {
  font-size: 0.65rem;
  font-weight: 700;
  padding: 0.05rem 0.3rem;
  border-radius: 4px;
  background: rgba(147, 51, 234, 0.15);
  color: #a855f7;
}

.badge-assignee.family-all {
  background: var(--primary-light);
  color: var(--primary);
}

/* WEEK VIEW */
.week-grid-container {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 0.75rem;
  overflow-x: auto;
  padding-bottom: 0.5rem;
}

@media (max-width: 900px) {
  .week-grid-container {
    grid-template-columns: repeat(7, minmax(130px, 1fr));
  }
}

.week-day-col {
  display: flex;
  flex-direction: column;
  background: var(--bg-glass-card, var(--bg-card));
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md, 0.75rem);
  padding: 0.75rem;
  gap: 0.6rem;
  min-height: 480px;
  box-shadow: var(--shadow-sm);
  transition: all 0.2s ease;
}

.week-day-col.is-today {
  border-color: var(--primary);
  background: var(--bg-card);
  box-shadow: 0 0 16px var(--primary-light);
}

.week-col-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 0.6rem;
  border-bottom: 1px solid var(--border-subtle);
  cursor: pointer;
}

.col-header-text {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.week-day-name {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.week-day-num {
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--text-main);
}

.week-day-num.today-num {
  color: var(--primary);
}

.col-add-btn {
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  width: 1.6rem;
  height: 1.6rem;
  border-radius: 0.35rem;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
}

.col-add-btn:hover {
  background: var(--primary);
  color: #fff;
  border-color: var(--primary);
}

.week-col-chores {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  flex: 1;
}

.week-chore-card {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.55rem;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.week-chore-card:hover {
  transform: translateY(-2px);
  border-color: var(--primary);
  box-shadow: var(--shadow-sm);
}

.week-chore-card.urgent {
  border-left: 3px solid #ef4444;
}

.week-chore-card.high {
  border-left: 3px solid #fb923c;
}

.week-chore-card.medium {
  border-left: 3px solid #3b82f6;
}

.week-chore-card.low {
  border-left: 3px solid #94a3b8;
}

.week-chore-card.is-completed {
  opacity: 0.55;
}

.chore-card-top {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.card-time {
  font-size: 0.72rem;
  font-weight: 600;
  color: var(--text-muted);
}

.card-time.text-dim {
  color: var(--text-dim);
}

.card-title {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--text-main);
  line-height: 1.3;
}

.card-title.strike {
  text-decoration: line-through;
  color: var(--text-muted);
}

.card-footer {
  display: flex;
  flex-wrap: wrap;
  gap: 0.3rem;
  margin-top: 0.2rem;
}

.area-chip {
  font-size: 0.65rem;
  padding: 0.1rem 0.35rem;
  border-radius: 4px;
  background: var(--bg-surface-hover);
  color: var(--text-muted);
  border: 1px solid var(--border-subtle);
}

.assignee-chip {
  font-size: 0.65rem;
  font-weight: 600;
  padding: 0.1rem 0.35rem;
  border-radius: 4px;
  background: rgba(147, 51, 234, 0.15);
  color: #a855f7;
}

.assignee-chip.family-all {
  background: var(--primary-light);
  color: var(--primary);
}

.col-empty-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.3rem;
  padding: 0.6rem;
  border: 1px dashed var(--border-card);
  background: transparent;
  color: var(--text-dim);
  border-radius: 0.45rem;
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  margin-top: auto;
}

.col-empty-btn:hover {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--primary-light);
}

/* DAY VIEW */
.day-view-container {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.day-view-hero {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.25rem 1.5rem;
  background: var(--bg-glass-card, var(--bg-card));
  border: 1px solid var(--border-card);
  border-radius: var(--radius-lg, 1rem);
  box-shadow: var(--shadow-sm);
  flex-wrap: wrap;
  gap: 1rem;
}

.day-hero-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.day-hero-badge {
  display: inline-flex;
  align-self: flex-start;
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  padding: 0.15rem 0.5rem;
  border-radius: 9999px;
  background: var(--primary-light);
  color: var(--primary);
  letter-spacing: 0.05em;
}

.day-hero-title {
  margin: 0;
  font-size: 1.45rem;
  font-weight: 700;
  color: var(--text-main);
  text-transform: capitalize;
}

.day-hero-sub {
  margin: 0;
  font-size: 0.85rem;
  color: var(--text-muted);
}

.text-success {
  color: var(--color-success, #10b981);
  font-weight: 600;
}

.day-empty-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3.5rem 2rem;
  background: var(--bg-glass-card, var(--bg-card));
  border: 1px solid var(--border-card);
  border-radius: var(--radius-lg, 1rem);
  text-align: center;
  gap: 0.75rem;
}

.day-empty-box .empty-icon {
  color: var(--text-dim);
  opacity: 0.6;
}

.day-empty-box h4 {
  margin: 0;
  font-size: 1.15rem;
  color: var(--text-main);
}

.day-empty-box p {
  margin: 0;
  font-size: 0.875rem;
  color: var(--text-muted);
  max-width: 420px;
}

.day-chores-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.day-chore-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  background: var(--bg-glass-card, var(--bg-card));
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md, 0.75rem);
  box-shadow: var(--shadow-sm);
  transition: all 0.2s ease;
}

.day-chore-card:hover {
  transform: translateY(-2px);
  border-color: var(--border-focus, var(--primary));
}

.day-chore-card.urgent {
  border-left: 4px solid #ef4444;
}

.day-chore-card.high {
  border-left: 4px solid #fb923c;
}

.day-chore-card.medium {
  border-left: 4px solid #3b82f6;
}

.day-chore-card.low {
  border-left: 4px solid #94a3b8;
}

.day-chore-card.is-completed {
  opacity: 0.6;
}

.day-chore-main {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  flex: 1;
  cursor: pointer;
}

.checkbox-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
  margin-top: 0.2rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.check-circle-empty {
  width: 1.35rem;
  height: 1.35rem;
  border-radius: 50%;
  border: 2px solid var(--text-muted);
  transition: all 0.2s ease;
}

.checkbox-btn:hover .check-circle-empty {
  border-color: var(--primary);
  background: var(--primary-light);
}

.chore-info-block {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  flex: 1;
}

.title-and-pills {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.day-chore-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-main);
}

.day-chore-title.strike {
  text-decoration: line-through;
  color: var(--text-muted);
}

.priority-pill {
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: capitalize;
  padding: 0.15rem 0.5rem;
  border-radius: 9999px;
}

.priority-pill.urgent {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.priority-pill.high {
  background: rgba(251, 146, 60, 0.15);
  color: #fb923c;
}

.priority-pill.medium {
  background: rgba(59, 130, 246, 0.15);
  color: #3b82f6;
}

.priority-pill.low {
  background: rgba(148, 163, 184, 0.15);
  color: #94a3b8;
}

.recurrence-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.72rem;
  color: var(--text-dim);
  background: var(--bg-surface-hover);
  padding: 0.15rem 0.5rem;
  border-radius: 9999px;
  border: 1px solid var(--border-subtle);
}

.chore-desc-text {
  margin: 0;
  font-size: 0.85rem;
  color: var(--text-muted);
  line-height: 1.4;
}

.chore-meta-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
  margin-top: 0.25rem;
}

.day-chore-actions {
  display: flex;
  align-items: center;
  margin-left: 0.75rem;
}

.btn-delete-chore {
  background: transparent;
  border: none;
  color: var(--text-dim);
  width: 2.2rem;
  height: 2.2rem;
  border-radius: 0.4rem;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-delete-chore:hover {
  background: rgba(244, 63, 94, 0.15);
  color: var(--color-danger, #f43f5e);
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
  padding: 1rem;
}

.modal-card {
  background: #0f172a;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 0.75rem;
  width: 100%;
  max-width: 540px;
  padding: 1.5rem;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.7);
  max-height: 90vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.modal-header h2 {
  font-size: 1.25rem;
  margin: 0;
  color: #f8fafc;
}

.modal-close-btn {
  background: none;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0.25rem;
}

.modal-close-btn:hover {
  color: #fff;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-bottom: 1rem;
}

.field-hint {
  font-size: 0.75rem;
  color: #94a3b8;
  margin-top: 0.2rem;
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

.btn-cancel:hover {
  color: #fff;
}

/* Google Calendar Modal Specifics */
.gcal-modal {
  max-width: 620px;
}

.gcal-title-box {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.gcal-icon {
  width: 1.5rem;
  height: 1.5rem;
  color: #3b82f6;
}

.gcal-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.section-badge {
  font-size: 0.65rem;
  font-weight: 700;
  color: #38bdf8;
  letter-spacing: 0.05em;
}

.gcal-section h3 {
  margin: 0;
  font-size: 1rem;
  color: #f1f5f9;
}

.gcal-desc {
  margin: 0;
  font-size: 0.825rem;
  color: #94a3b8;
  line-height: 1.4;
}

.feed-copy-box {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.feed-input {
  flex: 1;
  padding: 0.5rem 0.75rem;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 0.4rem;
  color: #e2e8f0;
  font-family: monospace;
  font-size: 0.8rem;
}

.btn-copy {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.5rem 1rem;
  background: rgba(59, 130, 246, 0.2);
  border: 1px solid rgba(59, 130, 246, 0.4);
  color: #60a5fa;
  border-radius: 0.4rem;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s;
}

.btn-copy:hover {
  background: rgba(59, 130, 246, 0.3);
}

.gcal-steps {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 0.4rem 0.6rem;
  margin-top: 0.75rem;
  background: rgba(255, 255, 255, 0.03);
  padding: 0.75rem;
  border-radius: 0.5rem;
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.step-num {
  width: 1.25rem;
  height: 1.25rem;
  border-radius: 50%;
  background: rgba(59, 130, 246, 0.25);
  color: #60a5fa;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.7rem;
  font-weight: 700;
}

.step-text {
  font-size: 0.78rem;
  color: #cbd5e1;
  line-height: 1.35;
}

.divider {
  height: 1px;
  background: rgba(255, 255, 255, 0.1);
  margin: 1.25rem 0;
}

.sync-form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-top: 0.5rem;
}

.spin-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
