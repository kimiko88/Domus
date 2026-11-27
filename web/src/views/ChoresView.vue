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

const { t, locale } = useI18n()
const choresStore = useChoresStore()
const areasStore = useAreasStore()
const familyStore = useFamilyStore()
const notifStore = useNotificationsStore()

const currentTab = ref<'all' | 'pending' | 'overdue' | 'completed'>('all')
const viewMode = ref<'list' | 'calendar'>('list')
const showCreateModal = ref(false)
const editingChoreId = ref<string | null>(null)
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
  recurrence_type: 'none' as ChoreRecurrence,
  recurrence_interval: 1
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
  if (!userId) return t('chores.view.wholeFamily')
  const member = familyStore.members.find((m) => m.id === userId)
  return member ? member.username : t('chores.view.memberFallback')
}

function localDateKey(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function getRecurrenceInterval(rule?: string): number {
  if (!rule) return 1
  const match = rule.match(/(?:^|;)\s*INTERVAL\s*=\s*(\d+)/i)
  const legacyValue = /^\d+$/.test(rule.trim()) ? Number(rule.trim()) : undefined
  const interval = match ? Number(match[1]) : (legacyValue ?? 1)
  return Number.isSafeInteger(interval) && interval > 0 ? Math.min(interval, 999) : 1
}

function getRecurrenceLabel(chore: Chore): string {
  if (!['daily', 'weekly', 'monthly'].includes(chore.recurrence_type)) {
    return t(`chores.recurrences.${chore.recurrence_type}`)
  }
  const interval = getRecurrenceInterval(chore.recurrence_rule)
  const unitKey = chore.recurrence_type === 'daily'
    ? (interval === 1 ? 'dayUnit' : 'daysUnit')
    : chore.recurrence_type === 'weekly'
      ? (interval === 1 ? 'weekUnit' : 'weeksUnit')
      : (interval === 1 ? 'monthUnit' : 'monthsUnit')
  return t('chores.view.repeatEvery', { interval, unit: t(`chores.view.${unitKey}`) })
}

function formatDate(dateStr?: string) {
  if (!dateStr) return t('chores.view.noDueDate')
  try {
    const d = new Date(dateStr)
    return d.toLocaleDateString(locale.value, { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })
  } catch {
    return dateStr
  }
}

async function handleToggleComplete(chore: Chore) {
  try {
    if (chore.status === 'completed') {
      await choresStore.reopenChore(chore.id)
      notifStore.success(t('chores.view.reopenedTitle'), t('chores.view.reopenedMessage', { title: chore.title }))
    } else {
      await choresStore.completeChore(chore.id)
      notifStore.success(t('chores.view.completedTitle'), t('chores.view.completedMessage', { title: chore.title }))
    }
  } catch {
    notifStore.error(t('common.error'), t('chores.view.completeFailed'))
  }
}

async function handleDelete(id: string) {
  try {
    await choresStore.deleteChore(id)
    notifStore.success(t('chores.view.deletedTitle'), t('chores.view.deletedMessage'))
  } catch {
    notifStore.error(t('common.error'), t('chores.view.deleteFailed'))
  }
}

function resetChoreForm() {
  form.value = {
    title: '',
    description: '',
    area_id: '',
    assigned_user_id: '',
    priority: 'medium',
    due_date: '',
    recurrence_type: 'none',
    recurrence_interval: 1
  }
  editingChoreId.value = null
}

function openCreateForDate(dateStr?: string) {
  resetChoreForm()
  if (dateStr) {
    // Default to 09:00 of that date
    form.value.due_date = `${dateStr}T09:00`
  }
  showCreateModal.value = true
}

function toDateTimeLocal(dateStr?: string): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) return ''
  const localDate = new Date(date.getTime() - date.getTimezoneOffset() * 60000)
  return localDate.toISOString().slice(0, 16)
}

function openEditChore(chore: Chore) {
  editingChoreId.value = chore.id
  form.value = {
    title: chore.title,
    description: chore.description || '',
    area_id: chore.area_id || '',
    assigned_user_id: chore.assigned_user_id || '',
    priority: chore.priority,
    due_date: toDateTimeLocal(chore.due_date),
    recurrence_type: chore.recurrence_type || 'none',
    recurrence_interval: getRecurrenceInterval(chore.recurrence_rule)
  }
  showCreateModal.value = true
}

async function handleSaveChore() {
  if (!form.value.title.trim()) return

  let dueIso: string | undefined
  if (form.value.due_date) {
    const parsedDate = new Date(form.value.due_date)
    if (Number.isNaN(parsedDate.getTime())) return
    dueIso = parsedDate.toISOString()
  }

  const payload = {
    title: form.value.title.trim(),
    description: form.value.description.trim() || undefined,
    area_id: form.value.area_id || undefined,
    assigned_user_id: form.value.assigned_user_id || undefined,
    priority: form.value.priority,
    due_date: editingChoreId.value ? (dueIso ?? '') : dueIso,
    recurrence_type: form.value.recurrence_type,
    recurrence_rule: form.value.recurrence_type === 'none'
      ? undefined
      : `INTERVAL=${Math.max(1, Math.min(999, Math.trunc(Number(form.value.recurrence_interval) || 1)))}`
  }

  try {
    if (editingChoreId.value) {
      await choresStore.updateChore(editingChoreId.value, payload)
      notifStore.success(t('chores.view.updatedTitle'), t('chores.view.updatedMessage', { title: payload.title }))
    } else {
      await choresStore.createChore(payload)
      const assignedText = form.value.assigned_user_id
        ? t('chores.view.createdAssigned', { name: getAssigneeName(form.value.assigned_user_id) })
        : t('chores.view.createdFamily')
      notifStore.success(t('chores.view.createdTitle'), assignedText)
    }
    showCreateModal.value = false
    resetChoreForm()
  } catch {
    notifStore.error(t('common.error'), t(editingChoreId.value ? 'chores.view.updateFailed' : 'chores.view.createFailed'))
  }
}

// Google Calendar Feed Copy
async function copySubscriptionLink() {
  const url = choresStore.getICalSubscriptionUrl()
  try {
    await navigator.clipboard.writeText(url)
    copiedFeedUrl.value = true
    notifStore.success(t('chores.view.copiedTitle'), t('chores.view.copiedMessage'))
    setTimeout(() => {
      copiedFeedUrl.value = false
    }, 3000)
  } catch {
    notifStore.error(t('common.error'), t('chores.view.copyFailed'))
  }
}

// Google Calendar Reverse Sync
async function handleSyncGoogleCalendar() {
  if (!gcalSyncUrl.value.trim()) return
  try {
    isSyncingGCal.value = true
    const res = await choresStore.syncGoogleCalendar(gcalSyncUrl.value.trim())
    notifStore.success(t('chores.view.syncSuccessTitle'), t('chores.view.syncSuccessMessage', { count: res.synced_count }))
    gcalSyncUrl.value = ''
    showGCalModal.value = false
  } catch (err: any) {
    notifStore.error(t('chores.view.syncErrorTitle'), t('chores.view.syncFailed'))
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
  const todayStr = localDateKey(today)

  // Leading days from previous month
  const prevMonthLastDay = new Date(year, month, 0).getDate()
  for (let i = startDayOfWeek - 1; i > 0; i--) {
    const d = new Date(year, month - 1, prevMonthLastDay - i + 1)
    const dStr = localDateKey(d)
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
  const targetCellCount = days.length <= 35 ? 35 : 42
  const remainingCells = targetCellCount - days.length
  for (let i = 1; i <= remainingCells; i++) {
    const d = new Date(year, month + 1, i)
    const dStr = localDateKey(d)
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
  return choresStore.chores.filter((chore) => {
    if (!chore.due_date) return false
    const dueDate = new Date(chore.due_date)
    return !Number.isNaN(dueDate.getTime()) && localDateKey(dueDate) === dateStr
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
    return d.toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' })
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
  const todayStr = localDateKey(new Date())
  for (let i = 0; i < 7; i++) {
    const d = new Date(start)
    d.setDate(start.getDate() + i)
    const dStr = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
    days.push({
      date: d,
      dateString: dStr,
      dayNumber: d.getDate(),
      dayName: d.toLocaleDateString(locale.value, { weekday: 'short' }),
      fullDayName: d.toLocaleDateString(locale.value, { weekday: 'long' }),
      isToday: dStr === todayStr,
      chores: getChoresForDate(dStr)
    })
  }
  return days
})

const selectedDayData = computed(() => {
  const d = calendarDate.value
  const dStr = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  const todayStr = localDateKey(new Date())
  return {
    date: d,
    dateString: dStr,
    dayNumber: d.getDate(),
    dayName: d.toLocaleDateString(locale.value, { weekday: 'long' }),
    monthName: d.toLocaleDateString(locale.value, { month: 'long', year: 'numeric' }),
    isToday: dStr === todayStr,
    chores: getChoresForDate(dStr)
  }
})

const currentPeriodLabel = computed(() => {
  const d = calendarDate.value
  if (calendarViewMode.value === 'month') {
    return d.toLocaleDateString(locale.value, { month: 'long', year: 'numeric' })
  } else if (calendarViewMode.value === 'week') {
    const start = getStartOfWeek(d)
    const end = new Date(start)
    end.setDate(start.getDate() + 6)
    if (start.getMonth() === end.getMonth()) {
      return `${start.getDate()} - ${end.getDate()} ${start.toLocaleDateString(locale.value, { month: 'short', year: 'numeric' })}`
    } else {
      return `${start.getDate()} ${start.toLocaleDateString(locale.value, { month: 'short' })} - ${end.getDate()} ${end.toLocaleDateString(locale.value, { month: 'short', year: 'numeric' })}`
    }
  } else {
    // Day view
    return d.toLocaleDateString(locale.value, { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
  }
})

const currentMonthLabel = computed(() => currentPeriodLabel.value)

function prevPeriod() {
  const d = new Date(calendarDate.value)
  if (calendarViewMode.value === 'month') {
    d.setDate(1)
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
    d.setDate(1)
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
  const [year, month, day] = dateStr.split('-').map(Number)
  calendarDate.value = new Date(year, month - 1, day)
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
          <span class="badge-pill">{{ choresStore.pendingChores.length }} {{ t('chores.view.toDo') }}</span>
        </div>
        <p class="subtitle">{{ t('chores.subtitle') }}</p>
      </div>

      <div class="header-action-group">
        <!-- View Mode Switcher -->
        <div class="view-mode-toggle" role="tablist" :aria-label="t('chores.view.viewModes')">
          <button
            class="toggle-btn"
            :class="{ active: viewMode === 'list' }"
            @click="viewMode = 'list'"
            :aria-label="t('chores.view.listView')"
          >
            <List class="btn-icon" />
            <span>{{ t('chores.view.listView') }}</span>
          </button>
          <button
            class="toggle-btn"
            :class="{ active: viewMode === 'calendar' }"
            @click="viewMode = 'calendar'"
            :aria-label="t('chores.view.calendarView')"
          >
            <Calendar class="btn-icon" />
            <span>{{ t('chores.view.calendarView') }}</span>
          </button>
        </div>

        <!-- Google Calendar Connect Button -->
        <button class="btn-secondary" @click="showGCalModal = true" :aria-label="t('chores.view.connectGoogleCalendar')">
          <ExternalLink class="btn-icon" />
          <span>Google Calendar</span>
        </button>

        <!-- New Chore Button -->
        <button class="btn-primary" @click="openCreateForDate()" :aria-label="t('chores.view.newChore')">
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
          <span class="stat-label">{{ t('chores.view.dueOrOverdue') }}</span>
        </div>
      </div>

      <div class="stat-card pending" @click="currentTab = 'pending'; viewMode = 'list'">
        <div class="stat-icon-wrap pending">
          <Clock class="stat-icon" />
        </div>
        <div class="stat-data">
          <span class="stat-value">{{ choresStore.pendingChores.length }}</span>
          <span class="stat-label">{{ t('chores.view.activeTasks') }}</span>
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
            :aria-label="chore.status === 'completed' ? t('chores.view.markPending') : t('chores.view.markDone')"
            @click="handleToggleComplete(chore)"
          >
            <CheckCircle2 v-if="chore.status === 'completed'" class="check-icon done" />
            <div v-else class="check-circle" />
          </button>

          <!-- Main Info -->
          <div class="chore-main" role="button" tabindex="0" @click="openEditChore(chore)" @keydown.enter="openEditChore(chore)">
            <div class="chore-title-row">
              <h3 class="chore-title" :class="{ strike: chore.status === 'completed' }">
                {{ chore.title }}
              </h3>

              <!-- Priority Tag -->
              <span class="priority-badge" :class="chore.priority">
                <Flame v-if="chore.priority === 'urgent' || chore.priority === 'high'" class="priority-icon" />
                {{ t(`chores.priorities.${chore.priority}`) }}
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
                {{ getRecurrenceLabel(chore) }}
              </span>
            </div>
          </div>

          <!-- Action Delete -->
          <button class="delete-btn"  :title="t('chores.view.deleteChore')" :aria-label="t('chores.view.deleteChore')" @click="handleDelete(chore.id)">
            <Trash2 class="del-icon" />
          </button>
        </div>

        <!-- Empty State -->
        <div v-if="filteredChores.length === 0" class="empty-state">
          <CheckSquare class="empty-icon" />
          <h3>{{ t('chores.view.noChoresInView') }}</h3>
          <p>{{ t('chores.view.emptyHint') }}</p>
          <button class="btn-primary" @click="openCreateForDate()">
            <Plus class="btn-icon" />
            {{ t('chores.view.createChore') }}
          </button>
        </div>
      </div>
    </div>

    <!-- CALENDAR VIEW -->
    <div v-else class="calendar-view-section">
      <!-- Calendar Navigation Bar -->
      <div class="calendar-nav-bar">
        <div class="cal-nav-left">
          <button class="nav-arrow-btn" @click="prevPeriod" :title="calendarViewMode === 'day' ? t('chores.view.previousDay') : calendarViewMode === 'week' ? t('chores.view.previousWeek') : t('chores.view.previousMonth')" :aria-label="t('chores.view.previousPeriod')">
            <ChevronLeft class="btn-icon" />
          </button>
          <h2 class="month-heading">{{ currentPeriodLabel }}</h2>
          <button class="nav-arrow-btn" @click="nextPeriod" :title="calendarViewMode === 'day' ? t('chores.view.nextDay') : calendarViewMode === 'week' ? t('chores.view.nextWeek') : t('chores.view.nextMonth')" :aria-label="t('chores.view.nextPeriod')">
            <ChevronRight class="btn-icon" />
          </button>
          <button class="btn-today" @click="goToToday">{{ t('chores.view.today') }}</button>
        </div>

        <!-- View Mode Switcher: Giorno / Settimana / Mese -->
        <div class="cal-view-selector" role="tablist" :aria-label="t('chores.view.calendarModes')">
          <button
            type="button"
            class="cal-view-btn"
            :class="{ active: calendarViewMode === 'day' }"
            @click="calendarViewMode = 'day'"
            role="tab"
            :aria-selected="calendarViewMode === 'day'"
          >
            {{ t('chores.view.day') }}
          </button>
          <button
            type="button"
            class="cal-view-btn"
            :class="{ active: calendarViewMode === 'week' }"
            @click="calendarViewMode = 'week'"
            role="tab"
            :aria-selected="calendarViewMode === 'week'"
          >
            {{ t('chores.view.week') }}
          </button>
          <button
            type="button"
            class="cal-view-btn"
            :class="{ active: calendarViewMode === 'month' }"
            @click="calendarViewMode = 'month'"
            role="tab"
            :aria-selected="calendarViewMode === 'month'"
          >
            {{ t('chores.view.month') }}
          </button>
        </div>

        <div class="cal-legend">
          <span class="legend-item"><span class="dot urgent"></span> {{ t('chores.view.urgent') }}</span>
          <span class="legend-item"><span class="dot high"></span> {{ t('chores.view.high') }}</span>
          <span class="legend-item"><span class="dot medium"></span> {{ t('chores.view.medium') }}</span>
          <span class="legend-item"><span class="dot low"></span> {{ t('chores.view.low') }}</span>
        </div>
      </div>

      <!-- MONTH VIEW -->
      <div v-if="calendarViewMode === 'month'" class="calendar-grid-container">
        <!-- Day Names Row -->
        <div class="calendar-weekdays">
          <span>{{ t('chores.view.mon') }}</span>
          <span>{{ t('chores.view.tue') }}</span>
          <span>{{ t('chores.view.wed') }}</span>
          <span>{{ t('chores.view.thu') }}</span>
          <span>{{ t('chores.view.fri') }}</span>
          <span>{{ t('chores.view.sat') }}</span>
          <span>{{ t('chores.view.sun') }}</span>
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
            @click="selectDayView(day.dateString)"
            :title="t('chores.view.openDay')"
          >
            <div class="cell-header">
              <span
                class="cell-day-num"
                :class="{ 'today-num': day.isToday }"
                @click.stop="selectDayView(day.dateString)"
                :title="t('chores.view.openDay')"
              >
                {{ day.dayNumber }}
              </span>
              <button
                class="cell-add-btn"
                :title="t('chores.view.addChoreToDay')"
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
                @click.stop="openEditChore(chore)"
                :title="t('chores.view.editTaskTitle', { title: chore.title })"
              >
                <button
                  type="button"
                  class="cal-chore-check"
                  @click.stop="handleToggleComplete(chore)"
                  :aria-label="chore.status === 'completed' ? t('chores.view.markPending') : t('chores.view.markDone')"
                >
                  <CheckCircle2 v-if="chore.status === 'completed'" class="check-icon done" />
                  <span v-else class="check-circle-empty"></span>
                </button>
                <span class="priority-dot" :class="chore.priority"></span>
                <span class="badge-title" :class="{ strike: chore.status === 'completed' }">{{ chore.title }}</span>
                <span class="badge-assignee" :class="{ 'family-all': !chore.assigned_user_id }">
                  {{ chore.assigned_user_id ? getAssigneeName(chore.assigned_user_id).slice(0, 3) : t('chores.view.familyShort') }}
                </span>
              </div>
            </div>
            <div
              v-if="day.chores.length"
              class="cell-mobile-summary"
              :aria-label="day.chores.map((chore) => chore.title).join(', ')"
            >
              <span
                v-for="chore in day.chores.slice(0, 3)"
                :key="chore.id"
                class="priority-dot"
                :class="chore.priority"
              ></span>
              <span class="mobile-chore-count">{{ day.chores.length }}</span>
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
          <div class="week-col-header" @click="selectDayView(day.dateString)"  :title="t('chores.view.openDayTitle')">
            <div class="col-header-text">
              <span class="week-day-name">{{ day.dayName }}</span>
              <span class="week-day-num" :class="{ 'today-num': day.isToday }">{{ day.dayNumber }}</span>
            </div>
            <button
              class="col-add-btn"
               :title="t('chores.view.addChoreTitle')"
              @click.stop="openCreateForDate(day.dateString)"
              :aria-label="t('chores.view.addChoreForDay')"
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
              @click="openEditChore(chore)"
              :title="t('chores.view.editTaskTitle', { title: chore.title })"
            >
              <div class="chore-card-top">
                <button
                  type="button"
                  class="calendar-check-btn"
                  @click.stop="handleToggleComplete(chore)"
                  :aria-label="chore.status === 'completed' ? t('chores.view.markPending') : t('chores.view.markDone')"
                >
                  <CheckCircle2 v-if="chore.status === 'completed'" class="check-icon done" />
                  <span v-else class="check-circle-empty"></span>
                </button>
                <span class="priority-dot" :class="chore.priority"></span>
                <span class="card-time" v-if="chore.due_date && chore.due_date.includes('T')">
                  {{ formatTime(chore.due_date) }}
                </span>
                <span v-else class="card-time text-dim">{{ t('chores.view.allDay') }}</span>
              </div>
              <span class="card-title" :class="{ strike: chore.status === 'completed' }">
                {{ chore.title }}
              </span>
              <div class="card-footer">
                <span v-if="getAreaName(chore.area_id)" class="area-chip">
                  {{ getAreaName(chore.area_id) }}
                </span>
                <span class="assignee-chip" :class="{ 'family-all': !chore.assigned_user_id }">
                  {{ chore.assigned_user_id ? getAssigneeName(chore.assigned_user_id) : t('chores.view.familyShort') }}
                </span>
              </div>
            </div>

            <button
              v-if="day.chores.length === 0"
              class="col-empty-btn"
              @click="openCreateForDate(day.dateString)"
            >
              <Plus :size="12" />
              <span>{{ t('chores.view.add') }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- DAY VIEW -->
      <div v-else-if="calendarViewMode === 'day'" class="day-view-container">
        <div class="day-view-hero">
          <div class="day-hero-info">
            <div class="day-hero-badge" v-if="selectedDayData.isToday">{{ t('chores.view.today') }}</div>
            <h3 class="day-hero-title">{{ selectedDayData.dayName }}, {{ selectedDayData.dayNumber }} {{ selectedDayData.monthName }}</h3>
            <p class="day-hero-sub">
              {{ selectedDayData.chores.length }} {{ selectedDayData.chores.length === 1 ? t('chores.view.dayTasksSingular') : t('chores.view.dayTasksPlural') }} •
              <span class="text-success">{{ selectedDayData.chores.filter(c => c.status === 'completed').length }} {{ t('chores.view.completedCount') }}</span>
            </p>
          </div>

          <button class="btn-primary" @click="openCreateForDate(selectedDayData.dateString)">
            <Plus :size="16" />
            <span>{{ t('chores.view.newChore') }}</span>
          </button>
        </div>

        <div v-if="selectedDayData.chores.length === 0" class="day-empty-box">
          <Calendar :size="42" class="empty-icon" />
          <h4>{{ t('chores.view.noChoresForDate') }}</h4>
          <p>{{ t('chores.view.noChoresForDateHint') }}</p>
          <button class="btn-secondary" @click="openCreateForDate(selectedDayData.dateString)">
            <Plus :size="14" />
            <span>{{ t('chores.view.addReminderForDate') }}</span>
          </button>
        </div>

        <div v-else class="day-chores-list">
          <div
            v-for="chore in selectedDayData.chores"
            :key="chore.id"
            class="day-chore-card"
            :class="[chore.priority, { 'is-completed': chore.status === 'completed' }]"
          >
            <div class="day-chore-main" @click="openEditChore(chore)">
              <button
                type="button"
                class="checkbox-btn"
                @click.stop="handleToggleComplete(chore)"
                :aria-label="chore.status === 'completed' ? t('chores.view.markPending') : t('chores.view.markDone')"
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
                    <span>{{ getRecurrenceLabel(chore) }}</span>
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
                    <span>{{ chore.assigned_user_id ? getAssigneeName(chore.assigned_user_id) : t('chores.view.wholeFamily') }}</span>
                  </span>
                </div>
              </div>
            </div>

            <div class="day-chore-actions">
              <button
                type="button"
                class="btn-delete-chore"
                @click="handleDelete(chore.id)"
                 :title="t('chores.view.deleteChoreAria')"
                 :aria-label="t('chores.view.deleteChoreAria')"
              >
                <Trash2 :size="16" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Create / Edit Chore Modal -->
    <div v-if="showCreateModal" class="modal-backdrop" @click="showCreateModal = false">
      <div class="modal-card" @click.stop role="dialog" aria-modal="true" aria-labelledby="modal-title">
        <div class="modal-header">
          <h2 id="modal-title">{{ t(editingChoreId ? 'chores.view.editHouseholdChore' : 'chores.view.newHouseholdChore') }}</h2>
          <button class="modal-close-btn" @click="showCreateModal = false"  :aria-label="t('chores.view.closeWindow')">
            <X class="icon-close" />
          </button>
        </div>

        <form @submit.prevent="handleSaveChore" class="modal-body">
          <div class="form-group">
            <label for="chore-title">{{ t('chores.view.taskTitle') }} *</label>
            <input
              id="chore-title"
              v-model="form.title"
              type="text"
              :placeholder="t('chores.view.titlePlaceholder')"
              required
            />
          </div>

          <div class="form-group">
            <label for="chore-desc">{{ t('chores.view.descriptionDetails') }}</label>
            <textarea
              id="chore-desc"
              v-model="form.description"
              rows="2"
              :placeholder="t('chores.view.descriptionPlaceholder')"
            ></textarea>
          </div>

          <!-- Assignment & Notification Rule -->
          <div class="form-group">
            <label for="chore-assignee">{{ t('chores.view.assignAndNotify') }} *</label>
            <select id="chore-assignee" v-model="form.assigned_user_id">
              <option value="">{{ t('chores.view.wholeFamily') }}</option>
              <option v-for="member in familyStore.members" :key="member.id" :value="member.id">
                👤 {{ member.username }} ({{ member.role }})
              </option>
            </select>
            <span class="field-hint">
              {{
                form.assigned_user_id
                  ? t('chores.view.notifyAssigned', { name: getAssigneeName(form.assigned_user_id) })
                  : t('chores.view.notifyFamily')
              }}
            </span>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label for="chore-area">{{ t('chores.view.roomArea') }}</label>
              <select id="chore-area" v-model="form.area_id">
                <option value="">{{ t('chores.view.wholeHome') }}</option>
                <option v-for="area in areasStore.areas" :key="area.id" :value="area.id">
                  {{ area.name }}
                </option>
              </select>
            </div>

            <div class="form-group">
              <label for="chore-priority">{{ t('chores.view.priority') }}</label>
              <select id="chore-priority" v-model="form.priority">
                <option value="low">{{ t('chores.view.optionLow') }}</option>
                <option value="medium">{{ t('chores.view.optionMedium') }}</option>
                <option value="high">{{ t('chores.view.optionHigh') }}</option>
                <option value="urgent">{{ t('chores.view.optionUrgent') }}</option>
              </select>
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label for="chore-due">{{ t('chores.view.dueDateTime') }}</label>
              <input id="chore-due" v-model="form.due_date" type="datetime-local" />
            </div>

            <div class="form-group">
              <label for="chore-recurrence">{{ t('chores.view.recurrence') }}</label>
              <select id="chore-recurrence" v-model="form.recurrence_type">
                <option value="none">{{ t('chores.view.noneOneTime') }}</option>
                <option value="daily">{{ t('chores.view.daily') }}</option>
                <option value="weekly">{{ t('chores.view.weekly') }}</option>
                <option value="monthly">{{ t('chores.view.monthly') }}</option>
              </select>
            </div>
          </div>

          <div v-if="form.recurrence_type !== 'none'" class="form-group recurrence-interval-group">
            <label for="chore-recurrence-interval">{{ t('chores.view.recurrenceInterval') }}</label>
            <div class="recurrence-interval-control">
              <span>{{ t('chores.view.every') }}</span>
              <input
                id="chore-recurrence-interval"
                v-model.number="form.recurrence_interval"
                type="number"
                min="1"
                max="999"
                step="1"
                inputmode="numeric"
                required
                :aria-label="t('chores.view.recurrenceInterval')"
              />
              <span>
                {{
                  form.recurrence_type === 'daily'
                    ? t(form.recurrence_interval === 1 ? 'chores.view.dayUnit' : 'chores.view.daysUnit')
                    : form.recurrence_type === 'weekly'
                      ? t(form.recurrence_interval === 1 ? 'chores.view.weekUnit' : 'chores.view.weeksUnit')
                      : t(form.recurrence_interval === 1 ? 'chores.view.monthUnit' : 'chores.view.monthsUnit')
                }}
              </span>
            </div>
            <span class="field-hint">{{ t('chores.view.recurrenceIntervalHint') }}</span>
          </div>

          <div class="modal-footer">
            <button type="button" class="btn-cancel" @click="showCreateModal = false">{{ t('chores.view.cancel') }}</button>
            <button type="submit" class="btn-primary">{{ t(editingChoreId ? 'chores.view.saveChanges' : 'chores.view.saveAndSchedule') }}</button>
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
            <h2 id="gcal-title">{{ t('chores.view.connectGoogleCalendar') }}</h2>
          </div>
          <button class="modal-close-btn" @click="showGCalModal = false" :aria-label="t('chores.view.closeWindow')">
            <X class="icon-close" />
          </button>
        </div>

        <div class="modal-body">
          <!-- Option A: Subscribe from Google Calendar (Export) -->
          <div class="gcal-section">
            <div class="section-badge">{{ t('chores.view.optionOne') }}</div>
            <h3>{{ t('chores.view.syncChoresToGoogleCalendar') }}</h3>
            <p class="gcal-desc">
              {{ t('chores.view.calendarDesc') }}
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
                <span>{{ copiedFeedUrl ? t('chores.view.copyDone') : t('chores.view.copyLink') }}</span>
              </button>
            </div>

            <div class="gcal-steps">
              <div class="step-num">1</div>
              <div class="step-text">{{ t('chores.view.googleStep1') }}</div>
              <div class="step-num">2</div>
              <div class="step-text">{{ t('chores.view.step2') }}</div>
              <div class="step-num">3</div>
              <div class="step-text">{{ t('chores.view.googleStep3') }}</div>
            </div>
          </div>

          <div class="divider"></div>

          <!-- Option B: Sync / Import from Google Calendar (Import) -->
          <div class="gcal-section">
            <div class="section-badge">{{ t('chores.view.optionTwo') }}</div>
            <h3>{{ t('chores.view.importFromGoogleCalendar') }}</h3>
            <p class="gcal-desc">
              {{ t('chores.view.importDesc') }}
            </p>

            <form @submit.prevent="handleSyncGoogleCalendar" class="sync-form">
              <div class="form-group">
                <label for="gcal-url">{{ t('chores.view.googleIcalAddress') }}</label>
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
                <span>{{ isSyncingGCal ? t('chores.view.syncing') : t('chores.view.syncNow') }}</span>
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

/* Responsive layout: avoid page-level overflow and keep the calendar usable on touch screens. */
.chores-view,
.calendar-view-section,
.list-view-section,
.calendar-grid-container,
.week-grid-container,
.day-view-container {
  min-width: 0;
  width: 100%;
  max-width: 100%;
}
.calendar-weekdays,
.calendar-cells-grid { grid-template-columns: repeat(7, minmax(0, 1fr)); }
.cal-day-cell,
.cell-chores-list,
.badge-title { min-width: 0; }
.cell-mobile-summary { display: none; }
.recurrence-interval-control {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  min-width: 0;
  color: #cbd5e1;
  font-size: 0.875rem;
}
.recurrence-interval-control input {
  width: 5.25rem;
  min-width: 4.5rem;
  text-align: center;
  font-variant-numeric: tabular-nums;
}
.recurrence-interval-control span:last-child { overflow-wrap: anywhere; }
.recurrence-interval-group {
  padding: 0.85rem;
  border: 1px solid rgba(59, 130, 246, 0.25);
  border-radius: 0.6rem;
  background: rgba(59, 130, 246, 0.06);
}
@media (max-width: 900px) {
  .chores-view { padding: 1rem; gap: 1rem; }
  .calendar-nav-bar { align-items: flex-start; }
  .cal-legend { flex-wrap: wrap; row-gap: 0.45rem; }
  .calendar-cells-grid { grid-auto-rows: minmax(92px, auto); }
  .cal-day-cell { padding: 0.4rem; }
  .cell-chores-list { max-height: 72px; }
  .cal-chore-badge { gap: 0.25rem; padding: 0.2rem 0.3rem; }
  .badge-assignee { display: none; }
  .week-grid-container {
    width: 100%; max-width: 100%; overflow-x: auto;
    overscroll-behavior-x: contain; scroll-snap-type: x proximity;
    -webkit-overflow-scrolling: touch;
  }
  .week-day-col { scroll-snap-align: start; }
}
@media (max-width: 640px) {
  .chores-view { padding: 0.75rem; gap: 0.85rem; }
  .chores-header { align-items: stretch; flex-direction: column; }
  .chores-header > div:first-child { min-width: 0; }
  .title-with-badge { align-items: flex-start; flex-wrap: wrap; gap: 0.45rem 0.65rem; }
  .title-with-badge h1 { font-size: 1.45rem; }
  .subtitle { margin-bottom: 0; }
  .header-action-group {
    display: grid; grid-template-columns: repeat(2, minmax(0, 1fr));
    width: 100%; gap: 0.5rem;
  }
  .view-mode-toggle { grid-column: 1 / -1; width: 100%; }
  .toggle-btn { flex: 1; justify-content: center; min-width: 0; padding: 0.55rem 0.4rem; }
  .header-action-group > .btn-secondary,
  .header-action-group > .btn-primary {
    justify-content: center; min-width: 0; padding: 0.6rem 0.45rem; font-size: 0.78rem;
  }
  .stats-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.45rem; }
  .stat-card { flex-direction: column; align-items: flex-start; gap: 0.45rem; padding: 0.7rem; min-width: 0; }
  .stat-icon-wrap { width: 2rem; height: 2rem; }
  .stat-icon { width: 1.1rem; height: 1.1rem; }
  .stat-value { font-size: 1.2rem; }
  .stat-label { font-size: 0.68rem; line-height: 1.2; overflow-wrap: anywhere; }
  .filter-tabs-bar { overflow: hidden; }
  .tabs-list {
    overflow-x: auto; padding-bottom: 0.35rem; scrollbar-width: thin;
    -webkit-overflow-scrolling: touch;
  }
  .tab-btn { flex: 0 0 auto; white-space: nowrap; padding: 0.5rem 0.7rem; font-size: 0.8rem; }
  .chore-card { align-items: flex-start; gap: 0.65rem; padding: 0.85rem; }
  .chore-main { min-width: 0; }
  .chore-title-row { gap: 0.45rem; }
  .assignee-pill { max-width: 100%; }
  .assignee-pill span { overflow-wrap: anywhere; }
  .calendar-nav-bar { flex-direction: column; align-items: stretch; gap: 0.75rem; padding: 0.75rem; }
  .cal-nav-left {
    display: grid; grid-template-columns: auto minmax(0, 1fr) auto auto;
    align-items: center; gap: 0.35rem; width: 100%;
  }
  .month-heading {
    min-width: 0; margin: 0; padding: 0 0.15rem; text-align: center;
    font-size: 0.95rem; overflow-wrap: anywhere;
  }
  .nav-arrow-btn { width: 2rem; height: 2rem; }
  .btn-today { padding: 0.4rem 0.55rem; font-size: 0.75rem; }
  .cal-view-selector { display: flex; width: 100%; }
  .cal-view-btn { flex: 1; min-width: 0; padding: 0.55rem 0.35rem; font-size: 0.78rem; }
  .cal-legend {
    display: grid; grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.4rem; font-size: 0.68rem;
  }
  .legend-item { min-width: 0; gap: 0.25rem; white-space: nowrap; }
  .calendar-grid-container { border-radius: 0.65rem; }
  .calendar-weekdays { padding: 0.5rem 0; font-size: 0.68rem; }
  .calendar-cells-grid { grid-auto-rows: minmax(76px, auto); }
  .cal-day-cell { min-height: 76px; padding: 0.28rem; gap: 0.15rem; overflow: hidden; }
  .cell-day-num { width: 1.45rem; height: 1.45rem; font-size: 0.75rem; }
  .cell-add-btn { opacity: 1; width: 1.15rem; height: 1.15rem; flex: 0 0 auto; }
  .cell-chores-list { display: none; }
  .cell-mobile-summary {
    display: flex; align-items: center; justify-content: center;
    gap: 0.2rem; min-height: 1.15rem; overflow: hidden;
  }
  .cell-mobile-summary .priority-dot { width: 0.38rem; height: 0.38rem; }
  .mobile-chore-count { font-size: 0.62rem; font-weight: 700; color: var(--text-main); line-height: 1; }
  .week-grid-container {
    grid-template-columns: repeat(7, minmax(145px, 1fr));
    gap: 0.55rem; padding-bottom: 0.75rem;
  }
  .week-day-col { min-height: 360px; padding: 0.65rem; scroll-snap-align: start; }
  .week-col-chores { min-width: 0; }
  .day-view-hero { align-items: flex-start; padding: 1rem; }
  .day-hero-title { font-size: 1.15rem; overflow-wrap: anywhere; }
  .day-view-hero > .btn-primary { width: 100%; justify-content: center; }
  .day-chore-card { align-items: flex-start; padding: 0.85rem; gap: 0.5rem; }
  .day-chore-main { min-width: 0; gap: 0.65rem; }
  .chore-info-block { min-width: 0; }
  .day-chore-actions { margin-left: 0; }
  .modal-backdrop {
    align-items: flex-start; justify-content: center; overflow-y: auto;
    padding: max(0.65rem, env(safe-area-inset-top)) 0.65rem max(0.65rem, env(safe-area-inset-bottom));
  }
  .modal-card {
    width: 100%; max-width: 100%; max-height: calc(100dvh - 1.3rem);
    margin: auto 0; padding: 1rem; border-radius: 0.7rem;
  }
  .modal-header h2 { min-width: 0; font-size: 1.05rem; overflow-wrap: anywhere; }
  .form-row { grid-template-columns: minmax(0, 1fr); gap: 0; }
  .form-group input, .form-group textarea, .form-group select { min-width: 0; width: 100%; }
  .modal-footer { flex-direction: column-reverse; gap: 0.5rem; }
  .modal-footer > button { width: 100%; justify-content: center; }
  .feed-copy-box { flex-direction: column; min-width: 0; }
  .feed-input, .btn-copy { min-width: 0; width: 100%; }
  .gcal-modal { max-height: calc(100dvh - 1.3rem); }
}
@media (max-width: 380px) {
  .chores-view { padding: 0.55rem; }
  .calendar-weekdays { font-size: 0.62rem; }
  .calendar-cells-grid { grid-auto-rows: minmax(66px, auto); }
  .cal-day-cell { min-height: 66px; padding: 0.2rem; }
  .cell-day-num { width: 1.3rem; height: 1.3rem; }
  .cal-legend { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .header-action-group > .btn-secondary, .header-action-group > .btn-primary { font-size: 0.72rem; }
}
/* Editing and check controls remain distinct on both pointer and touch layouts. */
.chore-title-edit {
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  padding: 0;
}
.chore-title-edit:hover,
.day-chore-main:hover .day-chore-title {
  color: var(--primary);
}
.chore-main[role="button"] { cursor: pointer; }
.chore-main[role="button"]:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 3px;
  border-radius: 0.35rem;
}
.check-btn,
.checkbox-btn,
.cal-chore-check,
.calendar-check-btn {
  flex: 0 0 auto;
  min-width: 2rem;
  min-height: 2rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  cursor: pointer;
  touch-action: manipulation;
}
.cal-chore-check,
.calendar-check-btn {
  padding: 0;
  border: 1px solid var(--border-subtle);
  background: transparent;
  color: var(--text-muted);
}
.cal-chore-check:hover,
.calendar-check-btn:hover {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--primary-light);
}
.cal-chore-check .check-circle-empty,
.calendar-check-btn .check-circle-empty {
  width: 0.75rem;
  height: 0.75rem;
}
.cal-chore-badge { cursor: pointer; }
.week-chore-card { cursor: pointer; }
</style>
