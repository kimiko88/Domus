<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  Calendar,
  ChevronLeft,
  ChevronRight,
  Plus,
  Utensils,
  CheckCircle2,
  Users,
  Trash2,
  Clock,
  Sparkles
} from '@lucide/vue'
import type { MealPlan, MealType } from '../../api/types'

interface Props {
  mealPlans: MealPlan[]
  loading?: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'add-meal-plan', date?: string, mealType?: MealType): void
  (e: 'consume-meal', id: string): void
  (e: 'delete-meal', id: string): void
}>()

// Current week offset: 0 = current week, -1 = last week, +1 = next week
const weekOffset = ref(0)

// Compute current 7 days of the selected week (Monday to Sunday)
const currentWeekDays = computed(() => {
  const now = new Date()
  const dayOfWeek = now.getDay() // 0 = Sunday, 1 = Monday
  const distanceToMonday = (dayOfWeek + 6) % 7 // Monday = 0
  const monday = new Date(now)
  monday.setDate(now.getDate() - distanceToMonday + weekOffset.value * 7)
  monday.setHours(0, 0, 0, 0)

  const days: { dateStr: string; dayName: string; dayNum: number; isToday: boolean }[] = []
  const todayStr = getTodayString()

  const dayNames = ['Lunedì', 'Martedì', 'Mercoledì', 'Giovedì', 'Venerdì', 'Sabato', 'Domenica']

  for (let i = 0; i < 7; i++) {
    const d = new Date(monday)
    d.setDate(monday.getDate() + i)
    const dateStr = formatDateToISO(d)
    days.push({
      dateStr,
      dayName: dayNames[i],
      dayNum: d.getDate(),
      isToday: dateStr === todayStr
    })
  }
  return days
})

function getTodayString(): string {
  const d = new Date()
  return formatDateToISO(d)
}

function formatDateToISO(d: Date): string {
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const mealTypes: { key: MealType; label: string; icon: string }[] = [
  { key: 'breakfast', label: 'Colazione', icon: '☕' },
  { key: 'lunch', label: 'Pranzo', icon: '☀️' },
  { key: 'snack', label: 'Spuntino', icon: '🥪' },
  { key: 'dinner', label: 'Cena', icon: '🌙' },
  { key: 'dessert', label: 'Dessert', icon: '🍰' }
]

function getMealsForDayAndType(dateStr: string, type: MealType): MealPlan[] {
  return props.mealPlans.filter(
    (m) => m.plan_date === dateStr && m.meal_type === type
  )
}

function prevWeek() {
  weekOffset.value--
}
function nextWeek() {
  weekOffset.value++
}
function resetToToday() {
  weekOffset.value = 0
}

const weekHeaderTitle = computed(() => {
  if (currentWeekDays.value.length === 0) return ''
  const first = currentWeekDays.value[0]
  const last = currentWeekDays.value[6]
  return `${first.dayNum} - ${last.dayNum} ${getMonthName(first.dateStr)}`
})

function getMonthName(dateStr: string): string {
  const d = new Date(dateStr)
  return d.toLocaleDateString('it-IT', { month: 'long', year: 'numeric' })
}
</script>

<template>
  <div class="calendar-tab-content">
    <!-- Calendar Controls -->
    <div class="calendar-nav-bar glass-panel">
      <div class="nav-left">
        <button
          type="button"
          class="btn-nav"
          aria-label="Settimana precedente"
          @click="prevWeek"
        >
          <ChevronLeft :size="18" />
        </button>

        <h3 class="week-title">{{ weekHeaderTitle }}</h3>

        <button
          type="button"
          class="btn-nav"
          aria-label="Settimana successiva"
          @click="nextWeek"
        >
          <ChevronRight :size="18" />
        </button>

        <button
          v-if="weekOffset !== 0"
          type="button"
          class="btn-today"
          @click="resetToToday"
        >
          Oggi
        </button>
      </div>

      <button
        type="button"
        class="btn btn-primary"
        @click="emit('add-meal-plan')"
      >
        <Plus :size="18" />
        <span>Pianifica Pasto</span>
      </button>
    </div>

    <!-- 7-Days Schedule Grid -->
    <div class="days-schedule-container">
      <div
        v-for="day in currentWeekDays"
        :key="day.dateStr"
        :class="['day-column glass-panel', { 'is-today': day.isToday }]"
      >
        <!-- Day Header -->
        <div class="day-header">
          <div class="day-name">{{ day.dayName }}</div>
          <div class="day-number">{{ day.dayNum }}</div>
          <span v-if="day.isToday" class="today-indicator">Oggi</span>
        </div>

        <!-- Meal Slots in Day -->
        <div class="meal-slots">
          <div
            v-for="slot in mealTypes"
            :key="slot.key"
            class="meal-slot-group"
          >
            <div class="slot-header">
              <span class="slot-title">
                <span class="slot-emoji">{{ slot.icon }}</span>
                {{ slot.label }}
              </span>
              <button
                type="button"
                class="slot-add-btn"
                :title="`Aggiungi ${slot.label} per ${day.dayName}`"
                :aria-label="`Aggiungi ${slot.label}`"
                @click="emit('add-meal-plan', day.dateStr, slot.key)"
              >
                <Plus :size="14" />
              </button>
            </div>

            <!-- Planned Meals in this Slot -->
            <div
              v-for="plan in getMealsForDayAndType(day.dateStr, slot.key)"
              :key="plan.id"
              :class="['meal-card', { 'is-consumed': plan.status === 'consumed' }]"
            >
              <div class="meal-card-top">
                <span class="meal-title">
                  {{ plan.recipe?.title || plan.custom_title || 'Pasto Pianificato' }}
                </span>

                <button
                  type="button"
                  class="del-btn"
                  title="Elimina pasto"
                  aria-label="Elimina pasto"
                  @click="emit('delete-meal', plan.id)"
                >
                  <Trash2 :size="13" />
                </button>
              </div>

              <!-- Persons count & Notes -->
              <div class="meal-card-meta">
                <span class="commensali-tag" :title="`Ingredienti scalati per ${plan.persons_count} persone`">
                  <Users :size="12" />
                  {{ plan.persons_count }} pers.
                </span>

                <span v-if="plan.notes" class="meal-note-snippet" :title="plan.notes">
                  {{ plan.notes }}
                </span>
              </div>

              <!-- Action: Consume button or Consumed badge -->
              <div class="meal-action-row">
                <div v-if="plan.status === 'consumed'" class="consumed-badge">
                  <CheckCircle2 :size="14" class="text-success" />
                  <span>Consumato</span>
                </div>

                <button
                  v-else
                  type="button"
                  class="btn-consume-action"
                  title="Consuma pasto e scala automaticamente gli ingredienti dalla dispensa!"
                  @click="emit('consume-meal', plan.id)"
                >
                  <Utensils :size="14" />
                  <span>Consuma</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.calendar-tab-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 16px);
}

.calendar-nav-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3, 12px) var(--space-5, 20px);
  border-radius: var(--radius-lg, 16px);
  background: var(--surface-card, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  flex-wrap: wrap;
  gap: 12px;
}

.nav-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.btn-nav {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  color: var(--text-main, #f8fafc);
  cursor: pointer;
}
.btn-nav:hover {
  background: rgba(255, 255, 255, 0.12);
}

.week-title {
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
  margin: 0;
  text-transform: capitalize;
}

.btn-today {
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 0.78rem;
  background: rgba(59, 130, 246, 0.15);
  border: 1px solid rgba(59, 130, 246, 0.3);
  color: var(--primary, #3b82f6);
  cursor: pointer;
}

.days-schedule-container {
  display: grid;
  grid-template-columns: repeat(7, minmax(180px, 1fr));
  gap: var(--space-3, 12px);
  overflow-x: auto;
  padding-bottom: 8px;
}

.day-column {
  display: flex;
  flex-direction: column;
  background: var(--surface-card, rgba(255, 255, 255, 0.02));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.07));
  border-radius: var(--radius-lg, 16px);
  padding: 12px;
  min-height: 520px;
}

.day-column.is-today {
  border-color: rgba(59, 130, 246, 0.5);
  background: rgba(59, 130, 246, 0.03);
}

.day-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  margin-bottom: 12px;
}

.day-name {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
}

.day-number {
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--primary, #3b82f6);
}

.today-indicator {
  font-size: 0.7rem;
  background: var(--primary, #3b82f6);
  color: #fff;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 600;
}

.meal-slots {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex: 1;
}

.meal-slot-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.slot-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2px 0;
}

.slot-title {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--text-muted, #94a3b8);
  text-transform: uppercase;
  letter-spacing: 0.02em;
}

.slot-add-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.06);
  border: none;
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
}
.slot-add-btn:hover {
  background: rgba(255, 255, 255, 0.15);
  color: var(--text-main, #f8fafc);
}

.meal-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  transition: all 0.2s ease;
}

.meal-card:hover {
  background: rgba(255, 255, 255, 0.07);
  border-color: rgba(255, 255, 255, 0.2);
}

.meal-card.is-consumed {
  opacity: 0.7;
  border-color: rgba(16, 185, 129, 0.3);
  background: rgba(16, 185, 129, 0.05);
}

.meal-card-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 6px;
}

.meal-title {
  font-size: 0.86rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
  line-height: 1.3;
}

.del-btn {
  background: transparent;
  border: none;
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
  padding: 2px;
  border-radius: 4px;
}
.del-btn:hover {
  color: var(--danger, #ef4444);
}

.meal-card-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.commensali-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.72rem;
  padding: 2px 6px;
  border-radius: 4px;
  background: rgba(59, 130, 246, 0.15);
  color: var(--primary, #3b82f6);
  font-weight: 500;
}

.meal-note-snippet {
  font-size: 0.72rem;
  color: var(--text-muted, #94a3b8);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 120px;
}

.meal-action-row {
  margin-top: 4px;
}

.btn-consume-action {
  width: 100%;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: 6px;
  background: var(--primary, #3b82f6);
  color: #fff;
  border: none;
  font-size: 0.76rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-consume-action:hover {
  background: #2563eb;
}

.consumed-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.75rem;
  color: var(--success, #10b981);
  font-weight: 500;
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 20px;
  border-radius: var(--radius-md, 10px);
  font-size: 0.92rem;
  font-weight: 600;
  cursor: pointer;
  border: none;
}
.btn-primary {
  background: var(--primary, #3b82f6);
  color: #fff;
}
.btn-primary:hover {
  background: #2563eb;
}
</style>
