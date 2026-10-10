<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTourStore } from '../../stores/tour'
import {
  Home,
  LayoutDashboard,
  Zap,
  Video,
  Workflow,
  Shield,
  Sparkles,
  Cpu,
  CheckCircle,
  X,
  ChevronRight,
  ChevronLeft
} from '@lucide/vue'

const { t } = useI18n()
const tourStore = useTourStore()

interface TourStepDef {
  icon: any
  titleKey: string
  taglineKey: string
  descKey: string
  routePath?: string
}

const steps: TourStepDef[] = [
  {
    icon: Home,
    titleKey: 'tour.step1Title',
    taglineKey: 'tour.step1Tagline',
    descKey: 'tour.step1Desc',
    routePath: '/'
  },
  {
    icon: LayoutDashboard,
    titleKey: 'tour.step2Title',
    taglineKey: 'tour.step2Tagline',
    descKey: 'tour.step2Desc',
    routePath: '/'
  },
  {
    icon: Zap,
    titleKey: 'tour.step3Title',
    taglineKey: 'tour.step3Tagline',
    descKey: 'tour.step3Desc',
    routePath: '/energy'
  },
  {
    icon: Video,
    titleKey: 'tour.step4Title',
    taglineKey: 'tour.step4Tagline',
    descKey: 'tour.step4Desc',
    routePath: '/cameras'
  },
  {
    icon: Workflow,
    titleKey: 'tour.step5Title',
    taglineKey: 'tour.step5Tagline',
    descKey: 'tour.step5Desc',
    routePath: '/automations'
  },
  {
    icon: Shield,
    titleKey: 'tour.step6Title',
    taglineKey: 'tour.step6Tagline',
    descKey: 'tour.step6Desc',
    routePath: '/'
  },
  {
    icon: Sparkles,
    titleKey: 'tour.step7Title',
    taglineKey: 'tour.step7Tagline',
    descKey: 'tour.step7Desc',
    routePath: '/media'
  },
  {
    icon: Cpu,
    titleKey: 'tour.step8Title',
    taglineKey: 'tour.step8Tagline',
    descKey: 'tour.step8Desc',
    routePath: '/integrations'
  },
  {
    icon: CheckCircle,
    titleKey: 'tour.step9Title',
    taglineKey: 'tour.step9Tagline',
    descKey: 'tour.step9Desc',
    routePath: '/guide'
  }
]

const currentStepIndex = computed(() => tourStore.currentStep)
const currentStepData = computed(() => steps[currentStepIndex.value] || steps[0])
const isLastStep = computed(() => currentStepIndex.value === steps.length - 1)
</script>

<template>
  <div v-if="tourStore.isOpen" class="tour-backdrop" role="dialog" aria-modal="true" aria-labelledby="tour-modal-title">
    <div class="tour-modal glass-card">
      <!-- Modal Header -->
      <div class="tour-header">
        <div class="tour-badge-box">
          <span class="tour-badge">{{ t('tour.stepIndicator', { current: currentStepIndex + 1, total: steps.length }) }}</span>
          <span v-if="currentStepData.routePath" class="route-pill">
            {{ currentStepData.routePath }}
          </span>
        </div>
        <button
          class="btn-close"
          :title="t('tour.skip')"
          :aria-label="t('tour.skip')"
          @click="tourStore.skipTour"
        >
          <X :size="18" />
        </button>
      </div>

      <!-- Tour Content Card -->
      <div class="tour-body">
        <div class="icon-hero-box">
          <component :is="currentStepData.icon" :size="36" class="step-icon" />
        </div>

        <h2 id="tour-modal-title" class="step-title">
          {{ t(currentStepData.titleKey) }}
        </h2>
        <p class="step-tagline">
          {{ t(currentStepData.taglineKey) }}
        </p>

        <div class="step-desc-box">
          <p class="step-desc">
            {{ t(currentStepData.descKey) }}
          </p>
        </div>

        <!-- Dot Indicators -->
        <div class="step-dots" role="tablist" aria-label="Passi del tour">
          <button
            v-for="(_, idx) in steps"
            :key="idx"
            class="dot-indicator"
            :class="{ active: idx === currentStepIndex }"
            :aria-label="`Vai al passo ${idx + 1}`"
            @click="tourStore.goToStep(idx)"
          />
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="tour-footer">
        <button class="btn-skip" @click="tourStore.skipTour">
          {{ t('tour.skip') }}
        </button>

        <div class="nav-buttons">
          <button
            class="btn-nav btn-prev"
            :disabled="currentStepIndex === 0"
            @click="tourStore.prevStep"
          >
            <ChevronLeft :size="16" />
            <span>{{ t('tour.prev') }}</span>
          </button>

          <button
            v-if="!isLastStep"
            class="btn-nav btn-next btn-primary"
            @click="tourStore.nextStep"
          >
            <span>{{ t('tour.next') }}</span>
            <ChevronRight :size="16" />
          </button>

          <button
            v-else
            class="btn-nav btn-finish btn-primary"
            @click="tourStore.completeTour"
          >
            <span>{{ t('tour.finish') }}</span>
            <CheckCircle :size="16" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tour-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(4, 7, 15, 0.75);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  animation: fadeIn 0.2s ease-out;
}

.tour-modal {
  width: 100%;
  max-width: 520px;
  background: linear-gradient(135deg, rgba(20, 27, 45, 0.95), rgba(11, 15, 25, 0.98));
  border: 1px solid rgba(255, 255, 255, 0.12);
  box-shadow: 0 24px 48px -12px rgba(0, 0, 0, 0.7), 0 0 0 1px rgba(14, 165, 233, 0.2);
  border-radius: 20px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  animation: slideUp 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}

.tour-header {
  padding: 18px 24px 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.tour-badge-box {
  display: flex;
  align-items: center;
  gap: 10px;
}

.tour-badge {
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: #38bdf8;
  background: rgba(14, 165, 233, 0.12);
  border: 1px solid rgba(14, 165, 233, 0.25);
  padding: 4px 10px;
  border-radius: 999px;
}

.route-pill {
  font-size: 0.72rem;
  font-family: monospace;
  color: #94a3b8;
  background: rgba(255, 255, 255, 0.06);
  padding: 3px 8px;
  border-radius: 6px;
}

.btn-close {
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 6px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.btn-close:hover {
  color: #f1f5f9;
  background: rgba(255, 255, 255, 0.08);
}

.tour-body {
  padding: 24px 28px 20px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.icon-hero-box {
  width: 72px;
  height: 72px;
  border-radius: 20px;
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.2), rgba(2, 132, 199, 0.08));
  border: 1px solid rgba(56, 189, 248, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #38bdf8;
  margin-bottom: 20px;
  box-shadow: 0 8px 24px -4px rgba(14, 165, 233, 0.25);
}

.step-title {
  font-size: 1.35rem;
  font-weight: 700;
  color: #f8fafc;
  margin: 0 0 6px 0;
  letter-spacing: -0.01em;
}

.step-tagline {
  font-size: 0.92rem;
  font-weight: 500;
  color: #38bdf8;
  margin: 0 0 16px 0;
}

.step-desc-box {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  padding: 14px 18px;
  margin-bottom: 20px;
  max-width: 440px;
}

.step-desc {
  font-size: 0.88rem;
  line-height: 1.55;
  color: #cbd5e1;
  margin: 0;
}

.step-dots {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
}

.dot-indicator {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.2);
  border: none;
  cursor: pointer;
  padding: 0;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.dot-indicator.active {
  width: 24px;
  background: #38bdf8;
  box-shadow: 0 0 10px rgba(56, 189, 248, 0.6);
}

.dot-indicator:hover:not(.active) {
  background: rgba(255, 255, 255, 0.4);
}

.tour-footer {
  padding: 16px 24px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(0, 0, 0, 0.15);
}

.btn-skip {
  background: transparent;
  border: none;
  color: #94a3b8;
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  padding: 6px 12px;
  border-radius: 8px;
  transition: color 0.15s ease;
}

.btn-skip:hover {
  color: #e2e8f0;
}

.nav-buttons {
  display: flex;
  align-items: center;
  gap: 10px;
}

.btn-nav {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 10px;
  font-size: 0.86rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
  border: 1px solid transparent;
}

.btn-prev {
  background: rgba(255, 255, 255, 0.06);
  color: #cbd5e1;
  border-color: rgba(255, 255, 255, 0.08);
}

.btn-prev:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.1);
  color: #f8fafc;
}

.btn-prev:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.btn-primary {
  background: linear-gradient(135deg, #0ea5e9, #0284c7);
  color: #ffffff;
  border: 1px solid rgba(56, 189, 248, 0.4);
  box-shadow: 0 4px 14px rgba(14, 165, 233, 0.35);
}

.btn-primary:hover {
  background: linear-gradient(135deg, #38bdf8, #0ea5e9);
  box-shadow: 0 6px 20px rgba(14, 165, 233, 0.5);
  transform: translateY(-1px);
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideUp {
  from { opacity: 0; transform: translateY(16px) scale(0.97); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}
</style>
