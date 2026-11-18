<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { BellRing, CheckCircle2, ShieldCheck, Smartphone, Send, RefreshCw } from '@lucide/vue'
import { apiClient } from '../../api/client'

interface WebPushPreferences {
  alarm_alerts: boolean
  emergency_alerts: boolean
  device_offline_alerts: boolean
  automation_failure_alerts: boolean
  command_failure_alerts: boolean
  network_alerts: boolean
  camera_alerts: boolean
  doorbell_alerts: boolean
  pantry_alerts: boolean
  finance_alerts: boolean
  task_alerts: boolean
  medicine_expiry_alerts: boolean
  medicine_low_stock_alerts: boolean
  medicine_expiry_days: number
  food_expiry_alerts: boolean
  food_low_stock_alerts: boolean
  food_expiry_days: number
}

const defaults: WebPushPreferences = {
  alarm_alerts: true, emergency_alerts: true, device_offline_alerts: true,
  automation_failure_alerts: true, command_failure_alerts: false,
  network_alerts: true, camera_alerts: false, doorbell_alerts: true,
  pantry_alerts: false, finance_alerts: false, task_alerts: true,
  medicine_expiry_alerts: true, medicine_low_stock_alerts: true,
  medicine_expiry_days: 7, food_expiry_alerts: true,
  food_low_stock_alerts: true, food_expiry_days: 3
}

const { t, locale } = useI18n()
const preferences = reactive<WebPushPreferences>({ ...defaults })
const vapidPublicKey = ref('')
const isConfigured = ref(false)
const isSubscribed = ref(false)
const permission = ref<NotificationPermission>('default')
const isLoading = ref(true)
const isSaving = ref(false)
const isSubscribing = ref(false)
const isTesting = ref(false)
const feedback = ref('')
const feedbackError = ref(false)
let currentSubscription: PushSubscription | null = null

const isSupported = computed(() =>
  typeof window !== 'undefined' &&
  'serviceWorker' in navigator &&
  'PushManager' in window &&
  'Notification' in window
)
const isIOS = computed(() => typeof navigator !== 'undefined' && /iPhone|iPad|iPod/i.test(navigator.userAgent))
const isStandalone = computed(() =>
  typeof window !== 'undefined' &&
  (window.matchMedia('(display-mode: standalone)').matches || Boolean((navigator as Navigator & { standalone?: boolean }).standalone))
)
const needsHomeScreenInstall = computed(() => isIOS.value && !isStandalone.value)

function setFeedback(message: string, isError = false) {
  feedback.value = message
  feedbackError.value = isError
}

async function loadSettings() {
  isLoading.value = true
  try {
    const [keyResponse, saved] = await Promise.all([
      apiClient.get<{ public_key: string }>('/api/v1/notifications/push/vapid-public-key'),
      apiClient.get<WebPushPreferences>('/api/v1/notifications/push/preferences')
    ])
    vapidPublicKey.value = keyResponse.public_key || ''
    Object.assign(preferences, defaults, saved || {})
    isConfigured.value = Boolean(vapidPublicKey.value)
    if (isSupported.value) {
      permission.value = Notification.permission
      const registration = await navigator.serviceWorker.ready
      currentSubscription = await registration.pushManager.getSubscription()
      isSubscribed.value = Boolean(currentSubscription)
      if (currentSubscription) {
        await apiClient.post('/api/v1/notifications/push/subscriptions', {
          ...currentSubscription.toJSON(),
          locale: locale.value
        })
      }
    }
  } catch (error) {
    isConfigured.value = false
    setFeedback(t('settings.webPush.loadFailed'), true)
    console.warn('Could not load Web Push configuration:', error)
  } finally {
    isLoading.value = false
  }
}

function decodeVapidKey(value: string): ArrayBuffer {
  const padding = '='.repeat((4 - value.length % 4) % 4)
  const base64 = (value + padding).replace(/-/g, '+').replace(/_/g, '/')
  const raw = window.atob(base64)
  const buffer = new ArrayBuffer(raw.length)
  const output = new Uint8Array(buffer)
  for (let i = 0; i < raw.length; i++) output[i] = raw.charCodeAt(i)
  return buffer
}

async function enablePush() {
  if (!isSupported.value || !vapidPublicKey.value) return
  isSubscribing.value = true
  feedback.value = ''
  try {
    // This request is deliberately made only in response to the user's button tap.
    const granted = Notification.permission === 'granted'
      ? 'granted'
      : await Notification.requestPermission()
    permission.value = granted
    if (granted !== 'granted') {
      setFeedback(granted === 'denied' ? t('settings.webPush.permissionDenied') : t('settings.webPush.permissionDismissed'), true)
      return
    }
    const registration = await navigator.serviceWorker.ready
    const subscription = await registration.pushManager.getSubscription() ||
      await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: decodeVapidKey(vapidPublicKey.value)
      })
    await apiClient.post('/api/v1/notifications/push/subscriptions', {
      ...subscription.toJSON(),
      locale: locale.value
    })
    currentSubscription = subscription
    isSubscribed.value = true
    await loadPreferencesOnly()
    setFeedback(t('settings.webPush.enabledSuccess'))
  } catch (error) {
    console.error('Web Push subscription failed:', error)
    setFeedback(t('settings.webPush.enableFailed'), true)
  } finally {
    isSubscribing.value = false
  }
}

async function loadPreferencesOnly() {
  const saved = await apiClient.get<WebPushPreferences>('/api/v1/notifications/push/preferences')
  Object.assign(preferences, defaults, saved || {})
}

async function disablePush() {
  if (!currentSubscription) return
  isSubscribing.value = true
  try {
    await apiClient.delete('/api/v1/notifications/push/subscriptions', {
      params: { endpoint: currentSubscription.endpoint }
    })
    await currentSubscription.unsubscribe()
    currentSubscription = null
    isSubscribed.value = false
    setFeedback(t('settings.webPush.disabledSuccess'))
  } catch (error) {
    console.error('Web Push unsubscribe failed:', error)
    setFeedback(t('settings.webPush.disableFailed'), true)
  } finally {
    isSubscribing.value = false
  }
}

async function savePreferences() {
  isSaving.value = true
  try {
    const saved = await apiClient.put<WebPushPreferences>('/api/v1/notifications/push/preferences', { ...preferences })
    Object.assign(preferences, defaults, saved || {})
    setFeedback(t('settings.webPush.preferencesSaved'))
  } catch (error) {
    console.error('Web Push preferences save failed:', error)
    setFeedback(t('settings.webPush.saveFailed'), true)
  } finally {
    isSaving.value = false
  }
}

async function sendTest() {
  isTesting.value = true
  try {
    await apiClient.post('/api/v1/notifications/push/test', {})
    setFeedback(t('settings.webPush.testSent'))
  } catch (error) {
    console.error('Web Push test failed:', error)
    setFeedback(t('settings.webPush.testFailed'), true)
  } finally {
    isTesting.value = false
  }
}

onMounted(loadSettings)
</script>

<template>
  <section class="web-push-card settings-card glass-card">
    <div class="card-header">
      <div class="icon-box text-cyan"><BellRing :size="20" /></div>
      <div class="push-title">
        <h3 class="card-title">{{ t('settings.webPush.title') }}</h3>
        <p>{{ t('settings.webPush.subtitle') }}</p>
      </div>
      <span class="push-status" :class="{ enabled: isSubscribed }">
        {{ isSubscribed ? t('settings.webPush.active') : t('settings.webPush.inactive') }}
      </span>
    </div>

    <div v-if="isLoading" class="push-loading">{{ t('common.loading') }}</div>
    <template v-else>
      <p v-if="needsHomeScreenInstall" class="push-notice">
        <Smartphone :size="17" /> {{ t('settings.webPush.iosInstallHint') }}
      </p>
      <p v-else-if="!isSupported" class="push-notice error">{{ t('settings.webPush.unsupported') }}</p>
      <p v-else-if="!isConfigured" class="push-notice error">{{ t('settings.webPush.serverNotConfigured') }}</p>
      <p v-else class="push-notice">
        <ShieldCheck :size="17" /> {{ t('settings.webPush.privacyHint') }}
      </p>

      <div class="push-actions">
        <button v-if="!isSubscribed" class="push-primary" type="button" :disabled="!isSupported || !isConfigured || isSubscribing" @click="enablePush">
          <BellRing :size="15" /> {{ isSubscribing ? t('common.loading') : t('settings.webPush.enable') }}
        </button>
        <template v-else>
          <button class="push-primary" type="button" :disabled="isTesting" @click="sendTest">
            <Send :size="15" /> {{ isTesting ? t('common.loading') : t('settings.webPush.sendTest') }}
          </button>
          <button class="push-secondary" type="button" :disabled="isSubscribing" @click="disablePush">
            {{ t('settings.webPush.disable') }}
          </button>
        </template>
        <span v-if="isSupported" class="permission-state">
          {{ t('settings.webPush.permission') }}: {{ t('settings.webPush.permissionStates.' + permission) }}
        </span>
      </div>

      <div class="push-preferences" :class="{ 'is-disabled': !isSubscribed }">
        <div class="push-section-heading">{{ t('settings.webPush.householdReminders') }}</div>
        <label class="push-check"><input v-model="preferences.task_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.tasks') }}</span></label>

        <div class="push-reminder-row">
          <label class="push-check"><input v-model="preferences.medicine_expiry_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.medicineExpiry') }}</span></label>
          <label class="push-days">{{ t('settings.webPush.daysBefore') }}
            <input v-model.number="preferences.medicine_expiry_days" type="number" min="0" max="365" :disabled="!isSubscribed || !preferences.medicine_expiry_alerts" />
          </label>
        </div>
        <label class="push-check push-nested"><input v-model="preferences.medicine_low_stock_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.medicineStock') }}</span></label>

        <div class="push-reminder-row">
          <label class="push-check"><input v-model="preferences.food_expiry_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.foodExpiry') }}</span></label>
          <label class="push-days">{{ t('settings.webPush.daysBefore') }}
            <input v-model.number="preferences.food_expiry_days" type="number" min="0" max="365" :disabled="!isSubscribed || !preferences.food_expiry_alerts" />
          </label>
        </div>
        <label class="push-check push-nested"><input v-model="preferences.food_low_stock_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.foodStock') }}</span></label>

        <div class="push-section-heading">{{ t('settings.webPush.systemAlerts') }}</div>
        <div class="push-general-grid">
          <label class="push-check"><input v-model="preferences.alarm_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.alarms') }}</span></label>
          <label class="push-check"><input v-model="preferences.emergency_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.emergencies') }}</span></label>
          <label class="push-check"><input v-model="preferences.device_offline_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.deviceOffline') }}</span></label>
          <label class="push-check"><input v-model="preferences.automation_failure_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.automationFailures') }}</span></label>
          <label class="push-check"><input v-model="preferences.command_failure_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.commandFailures') }}</span></label>
          <label class="push-check"><input v-model="preferences.network_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.networkAlerts') }}</span></label>
          <label class="push-check"><input v-model="preferences.camera_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.cameraAlerts') }}</span></label>
          <label class="push-check"><input v-model="preferences.doorbell_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.doorbellAlerts') }}</span></label>
          <label class="push-check"><input v-model="preferences.finance_alerts" type="checkbox" :disabled="!isSubscribed" /><span>{{ t('settings.webPush.financeAlerts') }}</span></label>
        </div>
        <div class="push-save-row">
          <span>{{ t('settings.webPush.zeroDaysHint') }}</span>
          <button class="push-primary" type="button" :disabled="!isSubscribed || isSaving" @click="savePreferences">
            <RefreshCw v-if="isSaving" :size="14" class="spin" />
            {{ isSaving ? t('common.loading') : t('common.save') }}
          </button>
        </div>
      </div>
      <p v-if="feedback" class="push-feedback" :class="{ error: feedbackError }" role="status">{{ feedback }}</p>
    </template>
  </section>
</template>

<style scoped>
.web-push-card { grid-column: 1 / -1; }
.push-title { flex: 1; min-width: 0; }
.push-title p { margin: 3px 0 0; color: var(--text-muted); font-size: 12px; }
.push-status { border-radius: 999px; padding: 5px 9px; background: var(--bg-surface-hover); color: var(--text-muted); font-size: 11px; font-weight: 700; }
.push-status.enabled { background: rgba(16,185,129,.12); color: var(--color-success, #10b981); }
.push-loading { color: var(--text-muted); padding: 14px 0; }
.push-notice { display: flex; align-items: flex-start; gap: 8px; padding: 10px 12px; border-radius: var(--radius-md); background: var(--bg-surface-hover); color: var(--text-muted); font-size: 12px; line-height: 1.5; }
.push-notice.error { color: var(--color-danger); }
.push-actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; margin: 14px 0 18px; }
.push-primary, .push-secondary { display: inline-flex; align-items: center; justify-content: center; gap: 7px; min-height: 42px; padding: 9px 14px; border-radius: var(--radius-md); font-size: 12px; font-weight: 650; cursor: pointer; }
.push-primary { color: #fff; background: var(--primary); }
.push-secondary { color: var(--text-main); background: var(--bg-surface-hover); border: 1px solid var(--border-card); }
.push-primary:disabled, .push-secondary:disabled { opacity: .5; cursor: not-allowed; }
.permission-state { color: var(--text-muted); font-size: 11px; }
.push-preferences { border-top: 1px solid var(--border-card); padding-top: 14px; }
.push-preferences.is-disabled { opacity: .55; }
.push-section-heading { margin: 12px 0 8px; color: var(--text-main); font-size: 13px; font-weight: 750; }
.push-check { display: flex; gap: 9px; align-items: center; color: var(--text-main); font-size: 12px; min-height: 36px; }
.push-check input { accent-color: var(--primary); width: 16px; height: 16px; }
.push-reminder-row { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 10px; border-top: 1px solid var(--border-card); margin-top: 6px; padding-top: 6px; }
.push-nested { padding-left: 25px; }
.push-days { display: flex; align-items: center; gap: 8px; color: var(--text-muted); font-size: 11px; }
.push-days input { width: 72px; padding: 7px 8px; border: 1px solid var(--border-card); background: var(--bg-surface); color: var(--text-main); border-radius: var(--radius-sm); }
.push-general-grid { display: grid; grid-template-columns: repeat(2, minmax(0,1fr)); gap: 4px 16px; }
.push-save-row { display: flex; justify-content: space-between; align-items: center; gap: 10px; border-top: 1px solid var(--border-card); padding-top: 12px; margin-top: 12px; }
.push-save-row > span { color: var(--text-muted); font-size: 11px; }
.push-feedback { margin-top: 12px; color: var(--color-success, #10b981); font-size: 12px; }
.push-feedback.error { color: var(--color-danger); }
.spin { animation: push-spin 1s linear infinite; }
@keyframes push-spin { to { transform: rotate(360deg); } }
@media (max-width: 520px) { .push-general-grid { grid-template-columns: 1fr; } .push-save-row { align-items: flex-start; flex-direction: column; } }
</style>
