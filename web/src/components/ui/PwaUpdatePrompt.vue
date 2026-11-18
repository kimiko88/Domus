<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { registerSW } from 'virtual:pwa-register'
import { CheckCircle, RefreshCw, X } from '@lucide/vue'

const { t } = useI18n()
const updateAvailable = ref(false)
const offlineReady = ref(false)
const isUpdating = ref(false)

let updateServiceWorker: ((reloadPage?: boolean) => Promise<void>) | undefined
let updateCheckInterval: ReturnType<typeof setInterval> | undefined
let visibilityHandler: (() => void) | undefined

onMounted(() => {
  updateServiceWorker = registerSW({
    immediate: true,
    onNeedRefresh() {
      updateAvailable.value = true
      offlineReady.value = false
    },
    onOfflineReady() {
      if (!updateAvailable.value) offlineReady.value = true
    },
    onRegisteredSW(_serviceWorkerUrl, registration) {
      if (!registration) return

      const checkForUpdate = () => {
        if (navigator.onLine) {
          void registration.update().catch(() => undefined)
        }
      }

      updateCheckInterval = setInterval(checkForUpdate, 60 * 60 * 1000)
      visibilityHandler = () => {
        if (document.visibilityState === 'visible') checkForUpdate()
      }
      document.addEventListener('visibilitychange', visibilityHandler)
    },
    onRegisterError(error) {
      console.warn('Domus service worker registration failed:', error)
    }
  })
})

onUnmounted(() => {
  if (updateCheckInterval) clearInterval(updateCheckInterval)
  if (visibilityHandler) document.removeEventListener('visibilitychange', visibilityHandler)
})

async function applyUpdate() {
  if (!updateServiceWorker) return
  isUpdating.value = true
  try {
    await updateServiceWorker(true)
  } catch (error) {
    isUpdating.value = false
    console.warn('Domus service worker update failed:', error)
  }
}

function dismissPrompt() {
  updateAvailable.value = false
  offlineReady.value = false
}
</script>

<template>
  <Transition name="pwa-update">
    <section
      v-if="updateAvailable || offlineReady"
      class="pwa-update-prompt glass-panel"
      :class="{ 'is-update': updateAvailable }"
      role="status"
      aria-live="polite"
    >
      <div class="pwa-update-icon">
        <RefreshCw v-if="updateAvailable" :size="20" />
        <CheckCircle v-else :size="20" />
      </div>
      <div class="pwa-update-copy">
        <strong>{{ updateAvailable ? t('common.pwa.updateAvailableTitle') : t('common.pwa.offlineReadyTitle') }}</strong>
        <p>{{ updateAvailable ? t('common.pwa.updateAvailableDescription') : t('common.pwa.offlineReadyDescription') }}</p>
      </div>
      <div class="pwa-update-actions">
        <button v-if="updateAvailable" type="button" class="pwa-update-primary" :disabled="isUpdating" @click="applyUpdate">
          <RefreshCw v-if="isUpdating" :size="14" class="is-spinning" />
          {{ isUpdating ? t('common.loading') : t('common.pwa.updateNow') }}
        </button>
        <button type="button" class="pwa-update-dismiss" @click="dismissPrompt">
          {{ updateAvailable ? t('common.pwa.later') : t('common.pwa.dismiss') }}
        </button>
      </div>
      <button
        v-if="updateAvailable"
        type="button"
        class="pwa-update-close"
        :aria-label="t('common.pwa.dismiss')"
        @click="dismissPrompt"
      >
        <X :size="16" />
      </button>
    </section>
  </Transition>
</template>

<style scoped>
.pwa-update-prompt {
  position: fixed;
  z-index: 12000;
  right: max(16px, env(safe-area-inset-right, 0px));
  bottom: calc(16px + env(safe-area-inset-bottom, 0px));
  display: flex;
  align-items: flex-start;
  gap: 12px;
  width: min(520px, calc(100vw - 32px - env(safe-area-inset-left, 0px) - env(safe-area-inset-right, 0px)));
  padding: 16px;
  border: 1px solid var(--border-card);
  border-radius: var(--radius-lg);
  background: var(--bg-surface-elevated);
  box-shadow: var(--shadow-lg);
  color: var(--text-main);
}

.pwa-update-icon {
  display: flex;
  flex: 0 0 36px;
  width: 36px;
  height: 36px;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  background: rgba(14, 165, 233, 0.14);
  color: var(--primary);
}

.pwa-update-copy {
  flex: 1;
  min-width: 0;
  padding-right: 18px;
}

.pwa-update-copy strong {
  display: block;
  font-size: 14px;
  line-height: 1.35;
}

.pwa-update-copy p {
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.45;
}

.pwa-update-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 10px;
}

.pwa-update-primary,
.pwa-update-dismiss {
  min-height: 44px;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.pwa-update-primary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--primary);
  color: #fff;
}

.pwa-update-primary:disabled {
  opacity: .65;
  cursor: wait;
}

.pwa-update-dismiss {
  background: var(--bg-surface-hover);
  color: var(--text-main);
}

.pwa-update-close {
  position: absolute;
  top: 6px;
  right: 6px;
  display: flex;
  width: 36px;
  height: 36px;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
}

.is-spinning {
  animation: pwa-spin .8s linear infinite;
}

.pwa-update-enter-active,
.pwa-update-leave-active {
  transition: opacity .2s ease, transform .2s ease;
}

.pwa-update-enter-from,
.pwa-update-leave-to {
  opacity: 0;
  transform: translateY(12px);
}

@keyframes pwa-spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 560px) {
  .pwa-update-prompt {
    left: max(12px, env(safe-area-inset-left, 0px));
    right: max(12px, env(safe-area-inset-right, 0px));
    bottom: calc(12px + env(safe-area-inset-bottom, 0px));
    width: auto;
    flex-wrap: wrap;
  }

  .pwa-update-actions {
    width: 100%;
    padding-left: 48px;
  }

  .pwa-update-primary,
  .pwa-update-dismiss {
    flex: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .pwa-update-enter-active,
  .pwa-update-leave-active,
  .is-spinning {
    animation: none;
    transition: none;
  }
}
</style>
