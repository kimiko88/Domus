<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useTourStore } from '../stores/tour'
import {
  BookOpen,
  Sparkles,
  Search,
  ChevronDown,
  ChevronUp,
  ArrowRight,
  Shield,
  Zap,
  Video,
  Cpu,
  Workflow,
  Users,
  Layers,
  Sun,
  LayoutDashboard,
  Radio,
  Sliders,
  CheckCircle,
  HelpCircle,
  Info,
  Volume2
} from '@lucide/vue'

const { t } = useI18n()
const router = useRouter()
const tourStore = useTourStore()

const searchQuery = ref('')
const activeCategory = ref<'all' | 'fundamentals' | 'energy' | 'security' | 'automations' | 'hardware'>('all')
const expandedSections = ref<Record<string, boolean>>({
  sec1: true,
  sec2: false,
  sec3: false,
  sec4: false,
  sec5: false,
  sec6: false,
  sec7: false,
  sec8: false,
  sec9: false,
  sec10: false,
  sec11: false,
  sec12: false,
  sec13: false
})

interface GuideTopic {
  id: string
  category: 'fundamentals' | 'energy' | 'security' | 'automations' | 'hardware'
  icon: any
  titleKey: string
  descKey: string
  routePath?: string
  routeLabelKey?: string
  highlightsKeys: string[]
}

const topics: GuideTopic[] = [
  {
    id: 'sec1',
    category: 'fundamentals',
    icon: Shield,
    titleKey: 'guide.sec1Title',
    descKey: 'guide.sec1Desc',
    highlightsKeys: [
      'guide.sec1H1',
      'guide.sec1H2',
      'guide.sec1H3'
    ]
  },
  {
    id: 'sec2',
    category: 'fundamentals',
    icon: LayoutDashboard,
    titleKey: 'guide.sec2Title',
    descKey: 'guide.sec2Desc',
    routePath: '/',
    routeLabelKey: 'nav.dashboard',
    highlightsKeys: [
      'guide.sec2H1',
      'guide.sec2H2',
      'guide.sec2H3'
    ]
  },
  {
    id: 'sec3',
    category: 'energy',
    icon: Sun,
    titleKey: 'guide.sec3Title',
    descKey: 'guide.sec3Desc',
    routePath: '/energy',
    routeLabelKey: 'nav.energy',
    highlightsKeys: [
      'guide.sec3H1',
      'guide.sec3H2',
      'guide.sec3H3'
    ]
  },
  {
    id: 'sec4',
    category: 'security',
    icon: Video,
    titleKey: 'guide.sec4Title',
    descKey: 'guide.sec4Desc',
    routePath: '/cameras',
    routeLabelKey: 'nav.cameras',
    highlightsKeys: [
      'guide.sec4H1',
      'guide.sec4H2',
      'guide.sec4H3'
    ]
  },
  {
    id: 'sec5',
    category: 'hardware',
    icon: Cpu,
    titleKey: 'guide.sec5Title',
    descKey: 'guide.sec5Desc',
    routePath: '/integrations',
    routeLabelKey: 'nav.integrations',
    highlightsKeys: [
      'guide.sec5H1',
      'guide.sec5H2',
      'guide.sec5H3'
    ]
  },
  {
    id: 'sec6',
    category: 'automations',
    icon: Workflow,
    titleKey: 'guide.sec6Title',
    descKey: 'guide.sec6Desc',
    routePath: '/automations',
    routeLabelKey: 'nav.automations',
    highlightsKeys: [
      'guide.sec6H1',
      'guide.sec6H2',
      'guide.sec6H3'
    ]
  },
  {
    id: 'sec7',
    category: 'security',
    icon: Shield,
    titleKey: 'guide.sec7Title',
    descKey: 'guide.sec7Desc',
    routePath: '/',
    routeLabelKey: 'guide.sec7RouteLabel',
    highlightsKeys: [
      'guide.sec7H1',
      'guide.sec7H2',
      'guide.sec7H3'
    ]
  },
  {
    id: 'sec8',
    category: 'hardware',
    icon: Radio,
    titleKey: 'guide.sec8Title',
    descKey: 'guide.sec8Desc',
    highlightsKeys: [
      'guide.sec8H1',
      'guide.sec8H2',
      'guide.sec8H3'
    ]
  },
  {
    id: 'sec9',
    category: 'hardware',
    icon: Sliders,
    titleKey: 'guide.sec9Title',
    descKey: 'guide.sec9Desc',
    routePath: '/integrations',
    routeLabelKey: 'nav.integrations',
    highlightsKeys: [
      'guide.sec9H1',
      'guide.sec9H2',
      'guide.sec9H3'
    ]
  },
  {
    id: 'sec10',
    category: 'fundamentals',
    icon: Layers,
    titleKey: 'guide.sec10Title',
    descKey: 'guide.sec10Desc',
    routePath: '/shopping',
    routeLabelKey: 'nav.shopping',
    highlightsKeys: [
      'guide.sec10H1',
      'guide.sec10H2',
      'guide.sec10H3'
    ]
  },
  {
    id: 'sec11',
    category: 'hardware',
    icon: Zap,
    titleKey: 'guide.sec11Title',
    descKey: 'guide.sec11Desc',
    routePath: '/settings',
    routeLabelKey: 'nav.settings',
    highlightsKeys: [
      'guide.sec11H1',
      'guide.sec11H2',
      'guide.sec11H3'
    ]
  },
  {
    id: 'sec12',
    category: 'fundamentals',
    icon: Users,
    titleKey: 'guide.sec12Title',
    descKey: 'guide.sec12Desc',
    routePath: '/family',
    routeLabelKey: 'nav.family',
    highlightsKeys: [
      'guide.sec12H1',
      'guide.sec12H2',
      'guide.sec12H3'
    ]
  },
  {
    id: 'sec13',
    category: 'hardware',
    icon: Volume2,
    titleKey: 'guide.sec13Title',
    descKey: 'guide.sec13Desc',
    routePath: '/media',
    routeLabelKey: 'nav.media',
    highlightsKeys: [
      'guide.sec13H1',
      'guide.sec13H2',
      'guide.sec13H3'
    ]
  }
]

const filteredTopics = computed(() => {
  return topics.filter(tItem => {
    // Category match
    if (activeCategory.value !== 'all' && tItem.category !== activeCategory.value) {
      return false
    }

    // Search query match
    if (searchQuery.value.trim()) {
      const q = searchQuery.value.toLowerCase()
      const title = t(tItem.titleKey).toLowerCase()
      const desc = t(tItem.descKey).toLowerCase()
      return title.includes(q) || desc.includes(q)
    }

    return true
  })
})

function toggleSection(id: string) {
  expandedSections.value[id] = !expandedSections.value[id]
}

function startTour() {
  tourStore.startTour()
}

function navigateTo(path: string) {
  router.push(path)
}
</script>

<template>
  <div class="guide-view">
    <!-- Top Hero Banner -->
    <header class="guide-hero glass-panel">
      <div class="hero-content">
        <div class="hero-badge">
          <BookOpen :size="16" />
          <span>{{ t('guide.featuresCount') }}</span>
        </div>
        <h1 class="hero-title">{{ t('guide.title') }}</h1>
        <p class="hero-subtitle">{{ t('guide.subtitle') }}</p>
      </div>

      <div class="hero-actions">
        <button class="btn-tour-hero" @click="startTour">
          <Sparkles :size="18" />
          <span>{{ t('guide.startTour') }}</span>
        </button>
      </div>
    </header>

    <!-- Search & Filter Controls -->
    <section class="controls-bar glass-panel">
      <div class="search-box">
        <Search :size="18" class="search-icon" />
        <label for="guide-search-input" class="sr-only">{{ t('guide.searchPlaceholder') }}</label>
        <input
          id="guide-search-input"
          name="guide_search"
          v-model="searchQuery"
          type="search"
          :placeholder="t('guide.searchPlaceholder')"
          class="search-input"
          aria-label="Cerca nella guida"
          autocomplete="off"
        />
      </div>

      <div class="filter-tabs" role="tablist">
        <button
          class="tab-btn"
          :class="{ active: activeCategory === 'all' }"
          @click="activeCategory = 'all'"
        >
          {{ t('guide.tabAll') }}
        </button>
        <button
          class="tab-btn"
          :class="{ active: activeCategory === 'fundamentals' }"
          @click="activeCategory = 'fundamentals'"
        >
          {{ t('guide.tabFundamentals') }}
        </button>
        <button
          class="tab-btn"
          :class="{ active: activeCategory === 'energy' }"
          @click="activeCategory = 'energy'"
        >
          {{ t('guide.tabEnergy') }}
        </button>
        <button
          class="tab-btn"
          :class="{ active: activeCategory === 'security' }"
          @click="activeCategory = 'security'"
        >
          {{ t('guide.tabSecurity') }}
        </button>
        <button
          class="tab-btn"
          :class="{ active: activeCategory === 'automations' }"
          @click="activeCategory = 'automations'"
        >
          {{ t('guide.tabAutomations') }}
        </button>
        <button
          class="tab-btn"
          :class="{ active: activeCategory === 'hardware' }"
          @click="activeCategory = 'hardware'"
        >
          {{ t('guide.tabHardware') }}
        </button>
      </div>
    </section>

    <!-- Topics Grid -->
    <main class="topics-container">
      <div v-if="filteredTopics.length === 0" class="empty-state glass-card">
        <HelpCircle :size="40" class="empty-icon" />
        <h3>{{ t('guide.noResults') }}</h3>
        <p>"{{ searchQuery }}"</p>
      </div>

      <div
        v-for="topic in filteredTopics"
        :key="topic.id"
        class="topic-card glass-card"
        :class="{ expanded: expandedSections[topic.id] }"
      >
        <div class="topic-header" @click="toggleSection(topic.id)">
          <div class="topic-icon-box">
            <component :is="topic.icon" :size="22" />
          </div>

          <div class="topic-meta">
            <h3 class="topic-title">{{ t(topic.titleKey) }}</h3>
            <p class="topic-summary">{{ t(topic.descKey) }}</p>
          </div>

          <button
            class="btn-expand"
            :aria-expanded="expandedSections[topic.id]"
            :aria-label="expandedSections[topic.id] ? 'Comprimi' : 'Espandi'"
          >
            <ChevronUp v-if="expandedSections[topic.id]" :size="18" />
            <ChevronDown v-else :size="18" />
          </button>
        </div>

        <!-- Collapsible Details -->
        <div v-if="expandedSections[topic.id]" class="topic-details">
          <ul class="highlights-list">
            <li v-for="(hk, i) in topic.highlightsKeys" :key="i">
              <CheckCircle :size="16" class="check-icon" />
              <span>{{ t(hk) }}</span>
            </li>
          </ul>

          <div v-if="topic.routePath" class="topic-actions">
            <button class="btn-navigate" @click="navigateTo(topic.routePath)">
              <span>{{ topic.routeLabelKey ? t(topic.routeLabelKey) : topic.routePath }}</span>
              <ArrowRight :size="15" />
            </button>
          </div>
        </div>
      </div>
    </main>

    <!-- Bottom Quick Tips Box -->
    <footer class="quick-tips glass-panel">
      <div class="tips-icon">
        <Info :size="24" />
      </div>
      <div class="tips-body">
        <h4>{{ t('guide.quickTipsTitle') }}</h4>
        <p>{{ t('guide.quickTipsDesc') }}</p>
      </div>
      <div class="tips-cta">
        <button class="btn-restart-tour" @click="tourStore.resetTour">
          <Sparkles :size="16" />
          <span>{{ t('tour.startTour') }}</span>
        </button>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.guide-view {
  display: flex;
  flex-direction: column;
  gap: 20px;
  max-width: 1200px;
  margin: 0 auto;
  padding-bottom: 40px;
}

.guide-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 28px 32px;
  border-radius: 20px;
  background: var(--bg-glass-card);
  border: 1px solid var(--border-card);
  box-shadow: var(--shadow-md);
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.78rem;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--primary);
  background: var(--primary-light);
  border: 1px solid rgba(14, 165, 233, 0.3);
  padding: 4px 12px;
  border-radius: 999px;
  margin-bottom: 10px;
}

.hero-title {
  font-size: 1.85rem;
  font-weight: 800;
  color: var(--text-main);
  margin: 0 0 6px 0;
  letter-spacing: -0.02em;
}

.hero-subtitle {
  font-size: 0.98rem;
  color: var(--text-muted);
  margin: 0;
  max-width: 600px;
}

.btn-tour-hero {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: linear-gradient(135deg, #0ea5e9, #0284c7);
  color: #ffffff;
  border: 1px solid rgba(56, 189, 248, 0.4);
  padding: 12px 22px;
  border-radius: 12px;
  font-size: 0.95rem;
  font-weight: 700;
  cursor: pointer;
  box-shadow: 0 6px 20px rgba(14, 165, 233, 0.35);
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  white-space: nowrap;
}

.btn-tour-hero:hover {
  background: linear-gradient(135deg, #38bdf8, #0ea5e9);
  box-shadow: 0 8px 24px rgba(14, 165, 233, 0.5);
  transform: translateY(-2px);
}

.controls-bar {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 18px 24px;
  border-radius: 16px;
  background: var(--bg-glass-card);
  border: 1px solid var(--border-subtle);
}

.search-box {
  position: relative;
  width: 100%;
}

.search-icon {
  position: absolute;
  left: 14px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-dim);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 12px 16px 12px 42px;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  border-radius: 10px;
  color: var(--text-main);
  font-size: 0.92rem;
  outline: none;
  transition: border-color 0.15s ease;
}

.search-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-light);
}

.filter-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tab-btn {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  padding: 6px 14px;
  border-radius: 8px;
  font-size: 0.84rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.tab-btn:hover {
  background: var(--bg-surface-hover);
  color: var(--text-main);
}

.tab-btn.active {
  background: var(--primary-light);
  border-color: var(--primary);
  color: var(--primary);
}

.topics-container {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.topic-card {
  border-radius: 16px;
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
  overflow: hidden;
  transition: all 0.2s ease;
}

.topic-card:hover {
  border-color: var(--border-card);
  background: var(--bg-card-hover);
}

.topic-card.expanded {
  border-color: var(--border-card);
  box-shadow: var(--shadow-md);
}

.topic-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 18px 22px;
  cursor: pointer;
  user-select: none;
}

.topic-icon-box {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: var(--primary-light);
  border: 1px solid rgba(14, 165, 233, 0.25);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
  flex-shrink: 0;
}

.topic-meta {
  flex: 1;
}

.topic-title {
  font-size: 1.08rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0 0 4px 0;
}

.topic-summary {
  font-size: 0.88rem;
  color: var(--text-muted);
  margin: 0;
  line-height: 1.45;
}

.btn-expand {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 6px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.btn-expand:hover {
  color: var(--text-main);
  background: var(--bg-surface-hover);
}

.topic-details {
  padding: 0 22px 20px 82px;
  border-top: 1px solid var(--border-subtle);
  padding-top: 16px;
  animation: fadeIn 0.15s ease-out;
}

.highlights-list {
  list-style: none;
  padding: 0;
  margin: 0 0 16px 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.highlights-list li {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  font-size: 0.88rem;
  color: var(--text-muted);
  line-height: 1.5;
}

.check-icon {
  color: var(--primary);
  flex-shrink: 0;
  margin-top: 3px;
}

.topic-actions {
  display: flex;
  justify-content: flex-end;
}

.btn-navigate {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--primary-light);
  border: 1px solid rgba(14, 165, 233, 0.25);
  color: var(--primary);
  padding: 6px 14px;
  border-radius: 8px;
  font-size: 0.84rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-navigate:hover {
  background: var(--primary);
  color: #ffffff;
  border-color: var(--primary);
}

.empty-state {
  text-align: center;
  padding: 48px 24px;
  border-radius: 16px;
  color: var(--text-muted);
}

.empty-icon {
  margin-bottom: 12px;
  color: var(--text-dim);
}

.quick-tips {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 20px 28px;
  border-radius: 16px;
  background: var(--bg-glass-card);
  border: 1px solid var(--border-subtle);
}

.tips-icon {
  color: var(--primary);
  flex-shrink: 0;
}

.tips-body {
  flex: 1;
}

.tips-body h4 {
  font-size: 0.98rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0 0 4px 0;
}

.tips-body p {
  font-size: 0.86rem;
  color: var(--text-muted);
  margin: 0;
}

.btn-restart-tour {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  padding: 10px 16px;
  border-radius: 10px;
  font-size: 0.86rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-restart-tour:hover {
  background: var(--bg-surface-hover);
  border-color: var(--primary);
  color: var(--primary);
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(-4px); }
  to { opacity: 1; transform: translateY(0); }
}

@media (max-width: 768px) {
  .guide-hero {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }
  .btn-tour-hero {
    width: 100%;
    justify-content: center;
  }
  .topic-details {
    padding-left: 22px;
  }
  .quick-tips {
    flex-direction: column;
    align-items: flex-start;
  }
  .btn-restart-tour {
    width: 100%;
    justify-content: center;
  }
}
</style>
